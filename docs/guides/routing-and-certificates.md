# 路由与证书
最后修改时间: 2026-09-28

Doc role: living guide

Orbit 通过 Gateway 的 Traefik `providers.rest` 管理自定义 Route。Route 创建、编辑、删除先写入业务数据；列表页和详情页的启停先保留在前端草稿中。详情页的证书弹窗只保存 HTTP/HTTPS、证书方式及验证方式，不触发路由发布。同步入口比较业务 Route 与全部 REST router/service：自定义 Route 使用域名、路径、目标地址、协议和 TCP 监听端口，未受管 REST 配置以可读规则和上游展示；Traefik API 不提供可靠的证书内容对比。确认同步时保存启停草稿，再通过一次完整 PUT 覆盖 `providers.rest` snapshot。发布失败时配置可能已保存，页面保留同步提示，可重新预览并重试。Gateway 部署完成后 worker 也会重新发布已保存配置的完整 snapshot。Component `endpoint.mode=gateway` 仍是独立的 Docker label 路由，默认 entrypoint/TLS 由 GatewayConfig 控制。

前端 `/routes` 以自定义 Route 为主列表，提供创建、编辑、批量启停草稿和同步入口；Route 详情页提供单条启停草稿、独立证书配置和同步。已启用 Route 修改证书后先保存，页面提示待同步；若路由规则未变化，同步弹窗说明本次发布的是证书配置。未启用 Route 的证书配置同样直接保存，不提示待同步，启用并同步后才发布。未启用 Route 的创建、编辑和删除也不单独提示待同步。`/route/traefik` 单独展示 Traefik 当前路由，可从主列表进入并返回。Traefik 页面展示运行时路由，不提供 Dashboard 跳转。

同步预览将已保存的证书状态纳入过期校验；显式同步始终发布完整快照。

发布因远端工作区文件更新被拒绝时，API 返回 `route_sync_publish_permission_denied`，弹窗显示权限错误和请求 ID；其他发布失败返回 `route_sync_publish_failed`。底层错误保留在服务日志中，可按请求 ID 定位。

HTTP Route 可使用手工 PEM、mkcert 或 Let's Encrypt。HTTP-01 需要 Gateway `http` 或 `http-dns` profile；DNS-01 需要 `dns` 或 `http-dns` profile 和 Gateway 中保存的 Cloudflare token。DNS-01 仍要求可注册的真实域名。

配置 DNS-01：

1. 在 Cloudflare 创建仅包含目标 zone `Zone:Read`、`DNS:Edit` 的 API token。
2. 在 Gateway 编辑页选择 `dns` 或 `http-dns`，填写 email 与 token。
3. 保存并重新部署 Gateway。
4. 在 Route Let's Encrypt 对话框选择 DNS-01。

TCP Route 先要求选中 Gateway Version 已声明对应 TCP endpoint 和 host port，再创建 Route。GatewayConfig 不提供 listener 或端口编辑。
