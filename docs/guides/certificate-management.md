# 证书管理
最后修改时间: 2026-10-02 13:15:41

Doc role: living guide

自定义 HTTP Route 支持 `manual`、`mkcert` 与 `letsencrypt`。TCP Route 不属于证书管理范围。

手工 PEM 和 mkcert 文件由单条 Route 同步到 Gateway Service 的 `gateway/certs/route-<route-id>/<sha256-revision>/{cert.pem,key.pem}`；Gateway 有效部署计划统一将该目录只读挂载到 `/etc/traefik/certs`。对应 `gateway/dynamic/route-<route-code>.yaml` 通过 File provider 的 `tls.certificates` 引用该版本；动态文件使用编码，证书目录使用稳定 ID，改名不迁移证书。新版本先完整写入，再替换 YAML；已存在的相同版本验证内容后复用，内容被修改时报告冲突。清理仅处理该 Route 的旧版本，保留当前及必要的上一版本，不改写其他 Route 的证书或权限。

证书配置保存与发布分离，启用后显式同步才发布。配置 API 匹配和实际证书验证独立报告；当前每次同步不进行 TLS 握手，HTTPS 证书结果为未验证。TLS 使用 Gateway 共享 SNI store，同域不同路径可共用一致证书，但不能按路径选择不同证书；与其他已发布 Route 的证书方式、证书或 ACME challenge 冲突时拒绝该项。

Let's Encrypt resolver 由 Gateway profile 对应的普通 Version 固化：

- `http`: `letsencrypt`，HTTP-01。
- `dns`: `letsencrypt-dns`，Cloudflare DNS-01。
- `http-dns`: 两个 resolver。

选择 profile 后填写 ACME email；DNS profile 还填写 Cloudflare token。DNS profile 在创建 TXT 后固定等待 60 秒，再执行传播检查并通知 ACME，避免跨网络 DNS 未收敛时过早验证。保存并部署 Gateway 后，worker 只补写 resolver email，并把 token 作为 `CF_DNS_API_TOKEN` 传给 Traefik。token 需要实际 zone 的 `Zone:Read` 与 `DNS:Edit` 权限。

ACME 数据由 Traefik 写入读写挂载的 `gateway/acme`，不参与 Route 的证书版本清理。Route 保存不表示证书已签发。排查时确认 Gateway 已重新部署、Route profile capability、Cloudflare TXT propagation 和 Traefik ACME 日志。
