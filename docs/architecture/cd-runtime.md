# CD 运行时与 Gateway
最后修改时间: 2026-10-03 10:10:00

Doc role: living architecture

## 运行时模型

Project 的共用 Environment 是 CD 的部署运行边界：当前初始化向导完成后的 Project 有一个 Environment 和一个默认 Gateway；新建 Project 不预置这两项。Application、Version、Component、Service 与 Deployment 仍是 Compose 拓扑的唯一来源；Gateway 是关联 `GatewayConfig` 的普通 Application。

```text
Project
  -> Environment (local | ssh)
  -> GatewayConfig / Gateway Service
  -> Deployment snapshot
  -> Version/Component effective plan
  -> target runtime
     local -> Environment.workspace_root/{pipeline,deployment} + control-plane Docker daemon
     ssh   -> Environment.workspace_root/deployment + docker compose over SSH
```

组合根按持久化的 `target_type` 注入 local 与 SSH runtime dispatcher；不会根据 hostname、空 SSH 字段或执行失败猜测另一种 runtime。两类 runtime 都从 Environment 保存的 `workspace_root` materialize Service workspace，保存值可以是绝对路径或 `~` / `~/...`；local 使用时把 `~` 展开为控制面进程用户主目录并交给 Docker daemon，SSH 在远端把 `~` 展开为登录用户主目录后执行。SSH 支持 Linux OpenSSH + Docker，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers，并维持私钥与 host-key pinning。`ssh` 到 loopback 也走 SSH runtime。控制面部署执行日志使用独立的 `logging.deployment_root`，不属于 Environment workspace。

Environment target 是 Project 的独占部署边界：local target 全局唯一，SSH target 按精确 host + port 唯一。这样固定 Gateway 端口和共享 `traefik` 网络不会被多个 Project 同时占用。

Environment Probe、Compose deploy/restart/stop、运行时查询、容器日志、Route 文件/证书发布和 Traefik API 查询通过同一个 target runtime 执行。Service 目录统一为 `<workspace_root>/deployment/<service-code>`；Pipeline 在控制面本地执行时使用同一根下的 `<workspace_root>/pipeline`，不把 SSH 远端路径作为本地 Docker bind source。local Probe 只验证控制面 Docker/Compose；SSH Probe 使用 Environment binding 的受管私钥验证认证、pinned host key 和目标 Docker prerequisites。

Deployment 将 `environment_id`、Environment `target_type`、target revision 与可选 Gateway Application identity 保存为正式不可变列。SSH deployment 额外保存 SSH Credential identity/revision；local deployment 的 SSH snapshot 为空。`options_json` 只保存命令选项和 Gateway 配置快照；worker 在执行前以正式列核对当前 Environment，目标变更后的排队任务直接失败，不能落到其他 Project 或新目标。

Environment Probe 在 HTTP 请求事务外执行 target I/O。Probe 完成后使用带 `target_revision` 条件的单条更新写入结果，旧配置上的探测不会覆盖新配置状态；单表更新本身是原子操作，不额外引入应用层事务编排。

SSH 初始化命令是普通 HTTP 写事务内的无远端 I/O 动作：它以提交的完整 SSH target 更新 Environment，并只在 binding 缺失或失效时创建 `environment_credential`。有效 binding 会原样复用。保存、编辑和 Probe 均不隐式生成密钥；未初始化 SSH target 的 Probe 仅写入“先生成并执行初始化命令”的失败诊断，不启动 SSH runner。

环境页的 SSH 终端在独立 WebSocket 会话中复用当前 Environment 的受管密钥、SSH 用户及 pinned host key，向目标 Linux/Windows 宿主机申请 PTY 并启动默认 shell。Bearer 认证的短请求签发一次性、短时效票据；WebSocket 通过子协议传票据，服务器仅协商固定协议名，不把票据放入 URL 或常规 HTTP 正文日志。会话在请求事务外运行，周期性复核 Project 成员及目标修订，且在输入前复核；有连接数、空闲和最长时长限制。Probe、部署与 CI 仍运行预定的非交互命令，不共享终端输入输出。

## GatewayConfig 与部署

Gateway 创建四个可编辑的普通 Version：`base`、`http`、`dns`、`http-dns`。每个 Version 固化 Traefik component、Docker socket、File provider watch、dynamic/certs 只读目录挂载、ACME 读写目录挂载、TCP 80/443 与 local HTTP 8080；profile Version 还固化对应 resolver YAML。`gateway_acme_profile_version` 记录四个角色到 Version 的 binding。

Gateway Compose 创建或复用部署宿主上的 Docker bridge network `traefik`。普通 Service 在声明加入 Traefik 网络时以 external 方式接入该共享网络；缺少 Gateway 网络配置时渲染失败。网络名固定为 `traefik`，不由 Environment code 派生。

GatewayConfig 保存 Traefik API URL/readiness、`internal_domain`、可选的 `external_domain`、Component ingress 默认策略，以及 `acme_profile`、`acme_email`、`dns_api_token`。API 地址用于就绪检查、运行时查询和发布后加载核验；本地容器中的 Orbit 使用 `rest_api_url`，本机 Orbit 与 SSH 目标使用 `rest_api_host_url`，SSH 查询在目标宿主机执行。Gateway 详情的 API 配置和基本信息分区展示及编辑。gateway mode 的 Docker labels 和 Gateway 派生地址继续使用 `internal_domain`；`external_domain` 仅由自定义 Route 表单读取以生成域名控件的候选值，不参与部署渲染、DNS 管理或 Traefik 发布。Traefik Component 名称固定为 `traefik`，从绑定 Version 解析，不存在 GatewayConfig 中。空 profile 选择 `base`；其余 profile 为 `http`、`dns`、`http-dns`。

创建 Gateway deployment 前，默认 Service 切换到保存 profile 所绑定的普通 Version，并持久化该 Version ID。worker 只执行该 ID，不能因之后的 profile 更新重新选择 Version。

API 计划/hash、worker 与 Compose 渲染共用 Gateway enrichment，在有效计划中处理：

- profile Version 的已声明 `/etc/traefik/traefik.yml` 中写入已有 resolver 的 `acme.email`。
- DNS profile 为 Traefik component 的有效 environment 设置 `CF_DNS_API_TOKEN`。
- 静态 YAML 移除 REST provider，覆盖 File provider 为 `/etc/traefik/dynamic`、`watch=true`，保留其他 provider、entrypoint 和 resolver；静态受控文件始终覆盖现有内容。
- 按 target 覆盖或补齐 dynamic/certs 只读与 acme 读写目录挂载，沿用 local/SSH/DooD 的路径与文件机制。

受管部分只改变有效计划，不写回 Version/Service；其他拓扑仍由 Version 声明。Gateway 部署和 Orbit 发起的 restart 自动 force-recreate，确保静态受控文件替换和挂载变更生效。迁移脚本初始化或已部署 REST 的受管 Gateway 使用同一流程，不修改已执行迁移、不转换旧路由快照、不新增 UI。首次切换后由用户显式同步路由，操作通过线下传达，切换期间可能暂不可用。email 与 token 在 Gateway deployment snapshot 和 rendered Compose 中可见。

## Route 发布

自定义 HTTP/TCP Route 通过 Traefik File provider 持久化发布，每条 Route 使用 `gateway/dynamic/route-<code>.yaml`，router/service 名称也由编码派生。手工证书仍为 `gateway/certs/route-<id>/<sha256-revision>/{cert.pem,key.pem}`；发布记录与旧文件位于未挂载的 `.orbit/route-publication/<id>/`。发布适配器按记录中的编码跟踪候选与旧路径，改名须撤下旧文件并在一次观测中确认新资源与旧资源缺席；失败及中断恢复共用文件转换逻辑，拒绝占用的编码与无归属的目标文件，已确认撤销的编码可复用。文件提交复用 WorkspaceFile/SyncFiles 和 target writer，不对整个证书目录 prune。SSH 写远端路径；local DooD 写 logical path，挂载经既有 resolver 转为 daemon 可见 physical path。

普通创建、编辑、删除与证书接口先保存业务配置；列表/详情启停为前端草稿。preview 明确 `selected` 或 `project` 范围，显示发布/撤销/跳过清单，返回冻结 ID 顺序、整批及每项独立的 `business_hash` 和 `publication_hash`；后者覆盖选定记录、实际 YAML/PEM fingerprint 与 Gateway/Environment 依赖。预览不查询运行时 router/service 差异，也不提交 `traefik_hash`。Project 范围取业务记录与已知发布记录的并集，已删除 ID 仍可撤销；未知文件及其他 provider 保留。禁用且从未发布的 Route 跳过文件操作。

Web 按冻结顺序在前端循环，每次只确认一个 ID，提交其原预览修订和对应草稿；每条响应立即更新状态及已保存草稿，失败继续，成功记录保留。不同项的独立发布不会使下一项过期，同一项或共享依赖变化仍须重新预览。界面只显示待处理/处理中/完成/失败和失败原因提示；不增加异步后台任务或页面/弹窗关闭后的续跑管理。API/MCP 保留显式批量能力。

预览规则直接返回 `protocol`、`match`、`target`；HTTP/HTTPS 由入口配置决定，与上游 URL 的协议分开。同步清单的协议列展示入口协议及证书方式，规则列展示匹配条件和目标，预览、处理中及完成后沿用同一表格。

确认在短事务内复核并保存单条启停草稿，再在事务外提交该 Route 的文件并等待 `@file` 资源匹配。响应 HTTP 200 携带 `route_sync_completed`/`route_sync_incomplete` 及逐项保存、文件、配置、证书、恢复/清理状态与原因编码；执行前的鉴权、范围或过期错误沿用通用 HTTP 错误契约。每个确认请求使用集中配置 `route.sync_timeout`，默认总预算 30 秒，包含失败恢复；单次 API 请求、重载、匹配、恢复阶段上限分别为 3、5、10、10 秒。匹配在重载后独立计时，发布预留最多 10 秒恢复时间，恢复不得延长原请求截止时间；API/MCP 批量请求内未执行项明确报告，Web 的多条请求不共享整批预算。业务保存不因外部失败回滚。中断后先核对 pending 记录、活动/备份 YAML 和 PEM，确认或恢复该操作后才接受新的发布，不以业务草稿重建。

Windows 文件提交后的加载收敛在 Traefik 发布适配器：SSH 使用声明的目标平台，local 使用现有 resolver 返回的 daemon mount source 区分 Windows/Linux 与 DooD。发布、撤销、失败恢复和 pending 恢复共用必要时 SIGHUP 重载再核对配置的路径；容器定位复用 Gateway Service 内的 Compose 查询，信号通过目标 Runtime 执行，不重启 Gateway，不改变其他 Route 文件。重载与配置匹配共用 8 秒预算，失败以 route_sync_reload_failed 进入既有逐项报告和恢复机制。Linux 继续依赖 watcher，远程 SSH Linux 实测留待后续；应用层和文件 writer 不加入平台分支或新接口。

配置匹配与实际 TLS 证书验证独立。router/service API 不能证明新 PEM 生效，当前 HTTPS 正常同步报告 `unverified`，不强制握手或因此自动回滚配置。HTTP-01 仅在 `http`/`http-dns` 可用；DNS-01 仅在 `dns`/`http-dns` 且 Gateway token 非空时可用。同域名候选必须与其他 Route 的已发布证书配置一致，不携带其他 Route 尚未同步的证书编辑。

API 与 worker 通过同进程、按 Project/Gateway 的协调器串行化发布和部署。部署前检查已发布上游、网络、entrypoint/resolver 引用；Gateway deploy/restart 后只核对持久化文件与加载结果，不重发当前数据库配置。Traefik 直接重启自行加载持久化文件，Orbit 离线时也能恢复。已确认 YAML 缺失不阻止 Gateway staging 创建空目录，部署后仍报告缺失；重新预览并确认 Route 同步可重建选中文件。文件不存在时保留空恢复基线，不恢复过期 backup；外部修改、损坏的证书或 pending 仍报告问题。新建 Gateway 可先部署空目录，再显式同步 Route。

TCP Route 的 entrypoint/host port 由选中 Gateway Version 的 Component endpoint 声明。Route 不创建 Gateway listener，也不改变 Gateway Version。

## Web 日志流

CI/CD 日志统一通过 Fetch + SSE 读取，所有入口沿用 Bearer 与 Project 成员授权。CD 操作文件保持位于控制面 `logging.deployment_root`，读取历史不要求远端在线；部署详情容器、Service Component、Gateway 与 Route 日志通过明确的 local/SSH target runtime 执行持续读取。Route 展示关联 Gateway 的输出。

操作文件入口为 `/api/deployment/:deployment_id/log/stream`，部署容器入口为 `/api/deployment/:deployment_id/container-log/stream`，持续运行的容器入口为 `/api/application/:app_id/log/stream`，后者要求显式 service_id，可指定 Component 名称。所有路由均为 GET，并要求 project_id，可携带续读 cursor。旧 Web 日志快照路由已移除；MCP 的 DeploymentLog/RuntimeComposeLogs 按需读取用途独立保留。

容器日志在操作进行中订阅；容器或 Compose 配置尚未出现时保持 waiting。以非 PTY 的 `docker compose logs --follow --timestamps --no-color` 传递 stdout，命令 stderr 单独保留最多 32 KiB 诊断。两秒复核成员、资源、目标/凭据修订与 Docker 容器 ID；容器重建时刷新来源，取消旧读取命令后继续。部署结束不会结束容器日志；stop 操作继续不展示容器日志。

文件日志采用精确字节游标；容器日志按时间戳重叠两秒补读，前端采用有界记录次数去重。来源变化和不能保证补齐的恢复显示缺口；Docker 删除的历史无法恢复。客户端断开仅取消读取命令/session，不停止部署任务或容器。SSE 心跳 15 秒，慢连接单次写入超时 10 秒；新增日志 SSE 路由关闭响应正文捕获，保留请求元信息。
