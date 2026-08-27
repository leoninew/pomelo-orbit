# 证书管理
最后修改时间: 2026-08-26 23:01:37

Doc role: living guide

自定义 HTTP Route 支持 `manual`、`mkcert` 与 `letsencrypt`。TCP Route 不属于证书管理范围。

手工 PEM 和 mkcert 文件由 Route 同步到 Gateway runtime service 的 `gateway/certs` 目录；Gateway Version 的 Traefik component 已声明该目录挂载到 `/etc/traefik/certs`。

Let's Encrypt resolver 由 Gateway profile 对应的普通 Version 固化：

- `http`: `letsencrypt`，HTTP-01。
- `dns`: `letsencrypt-dns`，Cloudflare DNS-01。
- `http-dns`: 两个 resolver。

选择 profile 后填写 ACME email；DNS profile 还填写 Cloudflare token。DNS profile 在创建 TXT 后固定等待 60 秒，再执行传播检查并通知 ACME，避免跨网络 DNS 未收敛时过早验证。保存并部署 Gateway 后，worker 只补写 resolver email，并把 token 作为 `CF_DNS_API_TOKEN` 传给 Traefik。token 需要实际 zone 的 `Zone:Read` 与 `DNS:Edit` 权限。

Route 保存不表示证书已签发。排查时确认 Gateway 已重新部署、Route profile capability、Cloudflare TXT propagation 和 Traefik ACME 日志。
