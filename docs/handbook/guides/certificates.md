# HTTPS 证书

组件入口使用网关默认 TLS 策略；自定义 HTTP 路由单独选择证书方式。

## 网关 ACME 能力

| Profile | 能力 | 配置 |
| --- | --- | --- |
| 未启用 | 无 Let's Encrypt resolver | 无 |
| `http` | HTTP-01 | ACME 邮箱 |
| `dns` | Cloudflare DNS-01 | 邮箱和 DNS API Token |
| `http-dns` | 两种挑战 | 邮箱和 DNS API Token |

组件默认 TLS 模式 `letsencrypt` 需要 `http` 或 `http-dns`。仅启用 `dns` 不满足该模式。更改能力后重新部署网关，组件入口策略更改后重新部署相关服务。

## 自定义 HTTP 路由

在路由详情点击“配置证书”：

| 方式 | 前提与结果 |
| --- | --- |
| HTTP | 使用 `web`，不启用该路由的 HTTPS |
| HTTPS · PEM | 上传包含证书和私钥的 PEM；同步检查配对，域名、有效期和信任需核对 |
| HTTPS · mkcert | 后端具备 mkcert 和 CA 材料；客户端需自行信任 CA |
| Let's Encrypt · HTTP-01 | 公开注册域名，网关有 HTTP 能力，挑战入口可达 |
| Let's Encrypt · DNS-01 | 公开注册域名，网关有 DNS 能力，Cloudflare 权限与传播正常 |

HTTPS 路由使用 `websecure`。保存后按[自定义路由](./routes.md)预览并确认同步，再查看 Traefik 的签发日志和客户端访问结果。

配置 ACME 只声明申请方式，证书的签发与续期由实际挑战结果决定。mkcert 适合已配置客户端信任的环境，CA 私钥应单独保护。

## Token 与私钥

切换到不含 DNS 的网关 profile 会清空 Token，同时影响[DNS 管理](./dns.md)。PEM 私钥、mkcert CA 材料及 Cloudflare Token 都按秘密保管，不放入公开配置或截图。
