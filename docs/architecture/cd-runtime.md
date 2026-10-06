# CD 运行时与 Gateway
最后修改时间: 2026-10-06

Doc role: living architecture

## 运行时模型

Project 的共用 Environment 是 CD 的部署运行边界：当前初始化向导完成后的 Project 有一个 Environment 和一个默认 Gateway；新建 Project 不预置这两项。Application、Version、Component、Service 与 Deployment 仍是 Compose 拓扑的唯一来源；Gateway 是 `kind=gateway` 且关联 `GatewayConfig` 的 Application。

```text
Project
  -> Environment (local | ssh)
  -> GatewayConfig / Gateway Service
  -> Deployment snapshot
  -> Version/Component effective plan
  -> target runtime
     local -> explicit Service directory + control-plane Docker daemon
     ssh   -> explicit remote Service directory + docker compose over SSH
```

组合根按持久化的 `target_type` 注入 local 与 SSH runtime dispatcher；不会根据 hostname、空 SSH 字段或执行失败猜测另一种 runtime。两类 runtime 使用 Service 确认的目录 materialize workspace，首次候选来自 Environment 的 `workspace_root`。保存值可以是绝对路径或 `~` / `~/...`；local 使用时把 `~` 展开为控制面进程用户主目录并交给 Docker daemon，SSH 在远端把 `~` 展开为登录用户主目录后执行。SSH 支持 Linux OpenSSH + Docker，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers，并维持私钥与 host-key pinning。`ssh` 到 loopback 也走 SSH runtime。控制面部署执行日志使用独立的 `logging.deployment_root`，不属于 Environment workspace。

Environment target 是 Project 的独占部署边界：local target 全局唯一，SSH target 按精确 host + port 唯一。这样固定 Gateway 端口和共享 `traefik` 网络不会被多个 Project 同时占用。

Environment Probe、Compose deploy/restart/stop、运行时查询、容器日志、Route 文件/证书发布和 Traefik API 查询通过同一个 target runtime 执行。Service 首次目录默认为 `<workspace_root>/deployment/<service-code>`，部署和依赖服务目录的操作接收显式服务编码与目录；Web 容器日志流在 Environment 工作区根目录按 Compose project 读取。Pipeline 在控制面本地执行时使用同一根下的 `<workspace_root>/pipeline`，不把 SSH 远端路径作为本地 Docker bind source。local Probe 只验证控制面 Docker/Compose；SSH Probe 使用 Environment binding 的受管私钥验证认证、pinned host key 和目标 Docker prerequisites。

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

每个预览/确认请求持有显式发布 session：RuntimeSessions 在目标 runtime 边界提供请求内 SSH/SFTP 复用，local session 不持有连接。绑定目标/凭据修订及工作区；阶段取消或 SSH/SFTP 失效淘汰连接，独立恢复可重新连接，退出时关闭。先按范围读取 Route 业务记录和 code，再逐条读取对应 state.json、实际 YAML/PEM 及必要 previous。selected 不列发布目录、不读取范围外记录；Project 范围枚举业务记录与已发布 ID 后使用同一单条流程，保留已删除 Route 的撤销能力。每项发布前只重读当前记录与文件，检查预览后变化；session 不保存全局发布元数据，不跨请求缓存。

加载匹配只查候选及撤下资源涉及的协议，HTTP/TCP 各为 router、service 两个 API；协议切换核对新协议与旧协议资源缺席。文件/证书和当前 target revision 一致的 confirmed 项只核验加载结果，已匹配则报告 `file_commit=unchanged`，不重复写入/重载/收尾；加载不符才重载并匹配。pending 恢复所得基线等于期望时复用其运行核验。合法复用的旧编码不再作为 absent 资源检查。Web saved 更新本地草稿与启停，synced 清理成功状态，finished 在整轮结束时统一刷新一次；详情仅重读 Route。

Publication.TargetRevision 是该工作区记录写入时的目标来源，允许读取历史 revision。Environment 切换与 Probe 不迁移或改写发布记录/路由文件；用户在新目标显式预览并确认后重新发布，写入当前 revision。历史 revision 不使用无变化捷径。本次请求内目标变化、预览/确认间依赖变化仍使预览过期，Project/Gateway 归属及文件、证书检查保持有效。

预览规则直接返回 `protocol`、`match`、`target`；HTTP/HTTPS 由入口配置决定，与上游 URL 的协议分开。同步清单的协议列展示入口协议及证书方式，规则列展示匹配条件和目标，预览、处理中及完成后沿用同一表格。

确认在短事务内复核并保存单条启停草稿，再在事务外提交该 Route 的文件并等待 `@file` 资源匹配。响应 HTTP 200 携带 `route_sync_completed`/`route_sync_incomplete` 及逐项保存、文件、配置、证书、恢复/清理状态与原因编码；执行前的鉴权、范围或过期错误沿用通用 HTTP 错误契约。同步不设请求级总预算，也不从正常发布中预扣恢复时间；Web 预览和确认取消 Axios 总超时。集中 `route` 配置分别限制 Gateway 锁等待、状态准备、文件处理、单次 API、重载、匹配和恢复，当前为 30、10、5、10、5、5、20 秒。状态准备包含范围内逐条发布记录读取与 Gateway 验证；仅 Project 预览枚举发布记录目录，每次发布前检查重新获得独立状态预算。文件提交和确认后记录/清理分别获得文件处理预算；匹配在重载后独立计时。API/MCP 批量请求的单项阶段超时不截断后续项；调用方取消则跳过尚未开始的项。需要恢复时使用脱离调用方取消、单独限时的 context，恢复内部重载/匹配受其剩余时间约束，未完成则保留恢复材料。业务保存不因外部失败回滚。中断后先核对 pending 记录、活动/备份 YAML 和 PEM，确认或恢复该操作后才接受新的发布，不以业务草稿重建。

Windows 文件提交后的加载收敛在 Traefik 发布适配器：SSH 使用声明的目标平台，local 使用现有 resolver 返回的 daemon mount source 区分 Windows/Linux 与 DooD。发布、撤销、失败恢复和 pending 恢复共用必要时 SIGHUP 重载再核对配置的路径；容器定位复用 Gateway Service 内的 Compose 查询，信号通过目标 Runtime 执行，不重启 Gateway，不改变其他 Route 文件。重载与配置匹配分别使用集中配置的 5 秒独立阶段预算，单次 API 上限为 10 秒，在匹配阶段内受其剩余时间约束。日志统一记录阶段起止、上限、实际耗时和状态，发布记录读取另记每条耗时；SSH/SFTP 错误同时保留阶段 context 的取消或超时原因。重载失败以 `route_sync_reload_failed` 报告；API 查询始终失败或超时导致无法确认配置时，以 `route_sync_configuration_unavailable` 报告；已成功观察 API 但配置不符时，以 `route_sync_configuration_mismatch` 报告；其他发布阶段超时使用 `route_sync_publish_timeout`。失败恢复状态独立返回并在 Web 原因提示中说明；原始 cause 只进入日志。Linux 继续依赖 watcher，远程 SSH Linux 实测留待后续；平台重载逻辑仅位于 Traefik 适配器。

配置匹配与实际 TLS 证书验证独立。router/service API 不能证明新 PEM 生效，当前 HTTPS 正常同步报告 `unverified`，不强制握手或因此自动回滚配置。HTTP-01 仅在 `http`/`http-dns` 可用；DNS-01 仅在 `dns`/`http-dns` 且 Gateway token 非空时可用。同步校验当前 Route 的证书与私钥、受管证书版本及恢复材料，不读取其他 Route 的证书或发布状态。同域名多 Route 的证书配置不再由同步中的全局记录扫描核验。

Route 同步阶段日志使用 `project_code`、`route_code` 识别业务对象，名称上下文随发布、恢复、重载和 Traefik API 请求传播。已删除业务记录的读取先记录发布记录路径，成功解码后补充发布时的 Route code。

自定义 Route 发布通过按 Project/Gateway 的协调器串行化，与 Service deploy/restart 独立。部署不读取或检查自定义 Route 的业务数据、发布记录、YAML/PEM 或上游引用，不获取 Route 发布锁，也不发布或恢复 Route；pending、发布失败、文件缺失或证书损坏不会阻止部署。普通 Service 不等待 Gateway 就绪；Gateway deploy/restart 仍准备受管 provider/mount、重建容器，并仅按部署快照中的 API 地址与就绪超时检查 Traefik 自身可用性。Traefik 直接重启自行加载持久化文件，Orbit 离线时也能恢复。Gateway staging 可创建空 dynamic 目录且不自动补齐 Route；重新预览并确认 Route 同步才能重建选中文件。文件不存在时保留空恢复基线，不恢复过期 backup；外部修改、损坏的证书或 pending 由显式 Route 同步处理。调整 Service endpoint、网络或 Gateway entrypoint/resolver 后，对应 Route 是否仍有效也由后续显式同步检查。

TCP Route 的 entrypoint/host port 由选中 Gateway Version 的 Component endpoint 声明。Route 不创建 Gateway listener，也不改变 Gateway Version。

## Web 日志流

CI/CD 日志统一通过 Fetch + SSE 读取，所有入口沿用 Bearer 与 Project 成员授权。CD 操作文件保持位于控制面 `logging.deployment_root`，读取历史不要求远端在线；部署详情容器、Service Component、Gateway 与 Route 日志通过明确的 local/SSH target runtime 执行持续读取。Route 展示关联 Gateway 的输出。

操作文件入口为 `/api/deployment/:deployment_id/log/stream`，部署容器入口为 `/api/deployment/:deployment_id/container-log/stream`，持续运行的容器入口为 `/api/application/:app_id/log/stream`，后者要求显式 service_id，可指定 Component 名称。所有路由均为 GET，并要求 project_id，可携带续读 cursor。旧 Web 日志快照路由已移除；MCP 的 DeploymentLog/RuntimeComposeLogs 按需读取用途独立保留。DeploymentLog 读取控制面操作日志，RuntimeComposeLogs 仍在 Service 运行目录执行带 `-f docker-compose.yml` 的快照命令；Web 日志流的目录解耦不改变这两个工具的前置条件。

Web 容器日志流可在操作进行中订阅；按 Service code 的 Compose project 标签和可选 Component 标签查询目标 Docker，容器尚未出现时保持 waiting。读取在 Environment 工作区根目录执行，使用显式 `docker compose -p <service-code> logs --follow --timestamps --no-color`，不加载 Compose 文件，也不依赖 Service 状态或已确认的运行目录，因此可读取目标环境上手动启动的同一 Compose project。以非 PTY 传递 stdout，命令 stderr 单独保留最多 32 KiB 诊断。两秒复核成员、资源、目标/凭据修订与 Docker 容器 ID；容器重建时刷新来源，取消旧读取命令后继续。部署结束不会结束容器日志；stop 操作继续不展示容器日志。

文件日志采用精确字节游标；容器日志按时间戳重叠两秒补读，前端采用有界记录次数去重。来源变化和不能保证补齐的恢复显示缺口；Docker 删除的历史无法恢复。客户端断开仅取消读取命令/session，不停止部署任务或容器。SSE 心跳 15 秒，慢连接单次写入超时 10 秒；新增日志 SSE 路由关闭响应正文捕获，保留请求元信息。

## 部署目录与执行边界

部署请求在现有写事务中保存确认目录与 Deployment 的 `working_directory` 快照。Worker 在目标修订校验后展开目标主目录，检查目录归属、写入权限和挂载映射，准备 Compose 与受控文件；准备成功且即将执行 Compose 时用短数据库更新绑定运行目录。绑定前失败保留旧目录，命令执行中失败或取消保留新运行操作目录。状态更新不得覆盖目录绑定。部署不移动、复制或删除旧数据。

local 在 Orbit 可见路径写文件，DooD 将每个具体相对 bind source 映射到 Docker daemon 可见路径，按最长挂载前缀处理嵌套挂载；未映射路径在命令前失败。SSH 使用远端路径与 SFTP，Linux 沿用 shell 引号，Windows 沿用非交互 PowerShell 和 `Set-Location -LiteralPath`，不通过本地 DooD resolver。

Gateway 路由、证书、ACME 及发布记录沿用运行目录；文件接口限制在显式服务范围。目录与目标修订属于 publication preview 依赖，目录变更使旧预览失效，不自动发布业务 Route。

目录归属以当前目标修订下各 Service 的确认目录、运行目录及实际运行容器的 bind mount 为依据，拒绝其他服务的目录和运行挂载占用。通过检查后按本次确认目录准备文件，既有 Compose 按有效计划替换；运行目录为空不阻止部署。local 与 SSH 文件范围同时检查解析后的路径，防止符号链接越界；部署期间复用已有 target session。
