# CD 运行时与 Gateway
最后修改时间: 2026-09-30 14:23:58

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

Environment Probe、Compose deploy/restart/stop、运行时查询、容器日志、证书同步和 Traefik REST 通过同一个 target runtime 执行。Service 目录统一为 `<workspace_root>/deployment/<service-code>`；Pipeline 在控制面本地执行时使用同一根下的 `<workspace_root>/pipeline`，不把 SSH 远端路径作为本地 Docker bind source。local Probe 只验证控制面 Docker/Compose；SSH Probe 使用 Environment binding 的受管私钥验证认证、pinned host key 和目标 Docker prerequisites。

Deployment 将 `environment_id`、Environment `target_type`、target revision 与可选 Gateway Application identity 保存为正式不可变列。SSH deployment 额外保存 SSH Credential identity/revision；local deployment 的 SSH snapshot 为空。`options_json` 只保存命令选项和 Gateway 配置快照；worker 在执行前以正式列核对当前 Environment，目标变更后的排队任务直接失败，不能落到其他 Project 或新目标。

Environment Probe 在 HTTP 请求事务外执行 target I/O。Probe 完成后使用带 `target_revision` 条件的单条更新写入结果，旧配置上的探测不会覆盖新配置状态；单表更新本身是原子操作，不额外引入应用层事务编排。

SSH 初始化命令是普通 HTTP 写事务内的无远端 I/O 动作：它以提交的完整 SSH target 更新 Environment，并只在 binding 缺失或失效时创建 `environment_credential`。有效 binding 会原样复用。保存、编辑和 Probe 均不隐式生成密钥；未初始化 SSH target 的 Probe 仅写入“先生成并执行初始化命令”的失败诊断，不启动 SSH runner。

环境页的 SSH 终端在独立 WebSocket 会话中复用当前 Environment 的受管密钥、SSH 用户及 pinned host key，向目标 Linux/Windows 宿主机申请 PTY 并启动默认 shell。Bearer 认证的短请求签发一次性、短时效票据；WebSocket 通过子协议传票据，服务器仅协商固定协议名，不把票据放入 URL 或常规 HTTP 正文日志。会话在请求事务外运行，周期性复核 Project 成员及目标修订，且在输入前复核；有连接数、空闲和最长时长限制。Probe、部署与 CI 仍运行预定的非交互命令，不共享终端输入输出。

## GatewayConfig 与部署

Gateway 创建四个可编辑的普通 Version：`base`、`http`、`dns`、`http-dns`。每个 Version 固化 Traefik component、Docker socket、证书/ACME directory mount、TCP 80/443 与 local HTTP 8080；profile Version 还固化对应 resolver YAML。`gateway_acme_profile_version` 记录四个角色到 Version 的 binding。

Gateway Compose 创建或复用部署宿主上的 Docker bridge network `traefik`。普通 Service 在声明加入 Traefik 网络时以 external 方式接入该共享网络；缺少 Gateway 网络配置时渲染失败。网络名固定为 `traefik`，不由 Environment code 派生。

GatewayConfig 保存 REST URL/readiness、`internal_domain`、可选的 `external_domain`、Component ingress 默认策略，以及 `acme_profile`、`acme_email`、`dns_api_token`。gateway mode 的 Docker labels 和 Gateway 派生地址继续使用 `internal_domain`；`external_domain` 仅由自定义 Route 表单读取以生成域名控件的候选值，不参与部署渲染、DNS 管理或 Traefik 发布。Traefik Component 名称固定为 `traefik`，从绑定 Version 解析，不存在 GatewayConfig 中。空 profile 选择 `base`；其余 profile 为 `http`、`dns`、`http-dns`。

创建 Gateway deployment 前，默认 Service 切换到保存 profile 所绑定的普通 Version，并持久化该 Version ID。worker 只执行该 ID，不能因之后的 profile 更新重新选择 Version。

worker 对有效计划仅作窄范围处理：

- profile Version 的已声明 `/etc/traefik/traefik.yml` 中写入已有 resolver 的 `acme.email`。
- DNS profile 为 Traefik component 的有效 environment 设置 `CF_DNS_API_TOKEN`。

它不新增、删除或重写 Version/Service 的拓扑。email 与 token 在 Gateway deployment snapshot 和 rendered Compose 中可见，这是本期明确的产品边界。

## Route 发布

自定义 HTTP/TCP Route 继续通过 Traefik `providers.rest` 全量发布。普通 Route 创建、编辑、删除先写入业务数据；列表页和详情页启停均为前端草稿。详情页的 HTTP/HTTPS 与证书方式通过证书接口直接保存，不触发发布。同步预览和确认共用期望状态校验；前者比较业务 Route 与全部 REST router/service，自定义 Route 按域名、路径、目标地址、协议和 TCP 监听端口对比，未受管 REST 配置以可读规则和上游展示。已保存的证书配置参与预览过期校验；Traefik API 不提供可靠的证书内容对比，故不把证书差异伪装成运行时路由差异。

Route snapshot 与端口冲突检查始终限定在当前 Project。确认同步或 Gateway deploy/restart 成功后，通过当前 Environment target runtime 调用该 Gateway 的 Traefik REST API 发布完整快照。HTTP-01 仅在 `http`/`http-dns` 可用；DNS-01 仅在 `dns`/`http-dns` 且 Gateway token 非空时可用。手工 PEM、mkcert 与 ACME account data 使用 Version 已声明的 cert/acme mount。

同步确认先在数据库事务内保存启停草稿，再执行外部 Traefik 发布。若发布失败，业务配置已保存，API 返回 `route_sync_publish_failed`，前端重新预览并保留同步提示供重试；不承诺跨数据库和 Traefik 的原子提交。证书接口只保存证书配置；已启用 Route 的证书更新在下一次显式同步时随完整快照发布，未启用 Route 的证书配置在启用并同步后发布。

TCP Route 的 entrypoint/host port 由选中 Gateway Version 的 Component endpoint 声明。Route 不创建 Gateway listener，也不改变 Gateway Version。

## Web 日志流

CI/CD 日志统一通过 Fetch + SSE 读取，所有入口沿用 Bearer 与 Project 成员授权。CD 操作文件保持位于控制面 `logging.deployment_root`，读取历史不要求远端在线；部署详情容器、Service Component、Gateway 与 Route 日志通过明确的 local/SSH target runtime 执行持续读取。Route 展示关联 Gateway 的输出。

操作文件入口为 `/api/deployment/:deployment_id/log/stream`，部署容器入口为 `/api/deployment/:deployment_id/container-log/stream`，持续运行的容器入口为 `/api/application/:app_id/log/stream`，后者要求显式 service_id，可指定 Component 名称。所有路由均为 GET，并要求 project_id，可携带续读 cursor。旧 Web 日志快照路由已移除；MCP 的 DeploymentLog/RuntimeComposeLogs 按需读取用途独立保留。

容器日志在操作进行中订阅；容器或 Compose 配置尚未出现时保持 waiting。以非 PTY 的 `docker compose logs --follow --timestamps --no-color` 传递 stdout，命令 stderr 单独保留最多 32 KiB 诊断。两秒复核成员、资源、目标/凭据修订与 Docker 容器 ID；容器重建时刷新来源，取消旧读取命令后继续。部署结束不会结束容器日志；stop 操作继续不展示容器日志。

文件日志采用精确字节游标；容器日志按时间戳重叠两秒补读，前端采用有界记录次数去重。来源变化和不能保证补齐的恢复显示缺口；Docker 删除的历史无法恢复。客户端断开仅取消读取命令/session，不停止部署任务或容器。SSE 心跳 15 秒，慢连接单次写入超时 10 秒；新增日志 SSE 路由关闭响应正文捕获，保留请求元信息。
