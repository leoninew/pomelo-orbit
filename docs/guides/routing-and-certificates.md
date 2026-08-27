# 路由与证书
最后修改时间: 2026-08-26 23:01:37

Doc role: living guide

Orbit 通过 Gateway 的 Traefik `providers.rest` 管理自定义 Route，并在变更时发送完整 HTTP/TCP snapshot。Component `endpoint.mode=gateway` 仍是独立的 Docker label 路由，默认 entrypoint/TLS 由 GatewayConfig 控制。

HTTP Route 可使用手工 PEM、mkcert 或 Let's Encrypt。HTTP-01 需要 Gateway `http` 或 `http-dns` profile；DNS-01 需要 `dns` 或 `http-dns` profile 和 Gateway 中保存的 Cloudflare token。DNS-01 仍要求可注册的真实域名。

配置 DNS-01：

1. 在 Cloudflare 创建仅包含目标 zone `Zone:Read`、`DNS:Edit` 的 API token。
2. 在 Gateway 编辑页选择 `dns` 或 `http-dns`，填写 email 与 token。
3. 保存并重新部署 Gateway。
4. 在 Route Let's Encrypt 对话框选择 DNS-01。

TCP Route 先要求选中 Gateway Version 已声明对应 TCP endpoint 和 host port，再创建 Route。GatewayConfig 不提供 listener 或端口编辑。
