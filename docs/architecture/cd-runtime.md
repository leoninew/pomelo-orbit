# CD 运行时与 Gateway
最后修改时间: 2026-09-11 14:48:17

Doc role: living architecture

## 运行时模型

Project 是部署运行边界：一个 Project 有一个 Environment，Environment 有一个默认 Gateway。Application、Version、Component、Service 与 Deployment 仍是 Compose 拓扑的唯一来源；Gateway 是关联 `GatewayConfig` 的普通 Application。

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

Environment Probe、Compose deploy/restart/stop、运行时查询、容器日志、证书同步和 Traefik REST 通过同一个 target runtime 执行。Service 目录统一为 `<workspace_root>/deployment/<service-code>`；Pipeline 在控制面本地执行时使用同一根下的 `<workspace_root>/pipeline`，不把 SSH 远端路径作为本地 Docker bind source。local Probe 只验证控制面 Docker/Compose；SSH Probe 验证认证、pinned host key 和目标 Docker prerequisites。

Deployment 将 `environment_id`、Environment `target_type`、target revision 与可选 Gateway Application identity 保存为正式不可变列。SSH deployment 额外保存 SSH Credential identity/revision；local deployment 的 SSH snapshot 为空。`options_json` 只保存命令选项和 Gateway 配置快照；worker 在执行前以正式列核对当前 Environment，目标变更后的排队任务直接失败，不能落到其他 Project 或新目标。

Environment Probe 在 HTTP 请求事务外执行 target I/O。Probe 完成后使用带 `target_revision` 条件的单条更新写入结果，旧配置上的探测不会覆盖新配置状态；单表更新本身是原子操作，不额外引入应用层事务编排。

## GatewayConfig 与部署

Gateway 创建四个可编辑的普通 Version：`base`、`http`、`dns`、`http-dns`。每个 Version 固化 Traefik component、Docker socket、证书/ACME directory mount、TCP 80/443 与 local HTTP 8080；profile Version 还固化对应 resolver YAML。`gateway_acme_profile_version` 记录四个角色到 Version 的 binding。

Gateway Compose 创建或复用部署宿主上的 Docker bridge network `traefik`。普通 Service 在声明加入 Traefik 网络时以 external 方式接入该共享网络；缺少 Gateway 网络配置时渲染失败。网络名固定为 `traefik`，不由 Environment code 派生。

GatewayConfig 保存 REST URL/readiness、base domain、Component ingress 默认策略，以及 `acme_profile`、`acme_email`、`dns_api_token`。Traefik Component 名称固定为 `traefik`，从绑定 Version 解析，不存在 GatewayConfig 中。空 profile 选择 `base`；其余 profile 为 `http`、`dns`、`http-dns`。

创建 Gateway deployment 前，默认 Service 切换到保存 profile 所绑定的普通 Version，并持久化该 Version ID。worker 只执行该 ID，不能因之后的 profile 更新重新选择 Version。

worker 对有效计划仅作窄范围处理：

- profile Version 的已声明 `/etc/traefik/traefik.yml` 中写入已有 resolver 的 `acme.email`。
- DNS profile 为 Traefik component 的有效 environment 设置 `CF_DNS_API_TOKEN`。

它不新增、删除或重写 Version/Service 的拓扑。email 与 token 在 Gateway deployment snapshot 和 rendered Compose 中可见，这是本期明确的产品边界。

## Route 发布

自定义 HTTP/TCP Route 继续通过 Traefik `providers.rest` 全量发布。Route 业务变更先写入业务数据；列表页和详情页启停均为前端草稿。同步入口预览业务 Route 与全部 REST router/service 的差异：自定义 Route 按域名、路径、目标地址、协议和 TCP 监听端口对比，未受管 REST 配置以可读规则和上游展示；不读取或比较证书、证书解析器及其他 Traefik 内部配置。

Route snapshot 与端口冲突检查始终限定在当前 Project。确认同步或 Gateway deploy/restart 成功后，通过当前 Environment target runtime 调用该 Gateway 的 Traefik REST API 发布完整快照。HTTP-01 仅在 `http`/`http-dns` 可用；DNS-01 仅在 `dns`/`http-dns` 且 Gateway token 非空时可用。手工 PEM、mkcert 与 ACME account data 使用 Version 已声明的 cert/acme mount。

TCP Route 的 entrypoint/host port 由选中 Gateway Version 的 Component endpoint 声明。Route 不创建 Gateway listener，也不改变 Gateway Version。
