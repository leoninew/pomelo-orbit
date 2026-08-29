# CD 运行时与 Gateway
最后修改时间: 2026-08-29 14:31:44

Doc role: living architecture

## 运行时模型

Application、Version、Component、Service 与 Deployment 是 Compose 拓扑的唯一来源。Gateway 是关联 `GatewayConfig` 的普通 Application；GatewayConfig 不保存或投影 Component、mount、endpoint、TCP listener 或 Traefik resolver 结构。

```text
GatewayConfig profile
  -> bound ordinary Version
  -> Service + Deployment snapshot
  -> Version/Component effective plan
  -> narrow Gateway enrichment
  -> docker compose
```

Gateway 创建四个可编辑的普通 Version：`base`、`http`、`dns`、`http-dns`。每个 Version 固化 Traefik component、Docker socket、证书/ACME directory mount、TCP 80/443 与 local HTTP 8080；profile Version 还固化对应 resolver YAML。`gateway_acme_profile_version` 记录四个角色到 Version 的 binding。

## GatewayConfig 与部署

GatewayConfig 保存 Traefik component 名、REST URL/readiness、base domain、Component ingress 默认策略，以及 `acme_profile`、`acme_email`、`dns_api_token`。空 profile 选择 `base`；其余 profile 为 `http`、`dns`、`http-dns`。

创建 Gateway deployment 前，默认 Service 切换到保存 profile 所绑定的普通 Version，并持久化该 Version ID。worker 只执行该 ID，不能因之后的 profile 更新重新选择 Version。

worker 对有效计划仅作窄范围处理：

- profile Version 的已声明 `/etc/traefik/traefik.yml` 中写入已有 resolver 的 `acme.email`。
- DNS profile 为 Traefik component 的有效 environment 设置 `CF_DNS_API_TOKEN`。

它不新增、删除或重写 Version/Service 的拓扑。email 与 token 在 Gateway deployment snapshot 和 rendered Compose 中可见，这是本期明确的产品边界。

## Route 发布

自定义 HTTP/TCP Route 继续通过 Traefik `providers.rest` 全量发布。Route 业务变更先写入业务数据；列表页和详情页启停均为前端草稿。同步入口预览业务 Route 与全部 REST router/service 的差异：自定义 Route 按域名、路径、目标地址、协议和 TCP 监听端口对比，未受管 REST 配置以可读规则和上游展示；不读取或比较证书、证书解析器及其他 Traefik 内部配置。确认后覆盖 REST provider 快照。Gateway deploy/restart 成功后也会自动重新发布完整快照。HTTP-01 仅在 `http`/`http-dns` 可用；DNS-01 仅在 `dns`/`http-dns` 且 Gateway token 非空时可用。手工 PEM、mkcert 与 ACME account data 使用 Version 已声明的 cert/acme mount。

TCP Route 的 entrypoint/host port 由选中 Gateway Version 的 Component endpoint 声明。Route 不创建 Gateway listener，也不改变 Gateway Version。
