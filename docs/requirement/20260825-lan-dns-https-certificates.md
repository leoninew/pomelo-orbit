# 局域网 DNS-01 HTTPS 与 Gateway 配置收敛需求
最后修改时间: 2026-08-26 23:01:37

Review status: Accepted

流程模式: 严格 / strict

## Background

Traefik Gateway 已由普通 Application、Version、Component、Service 与 Deployment 部署。基础 Gateway Version 已声明 Docker socket、静态 `traefik.yml`、`web:80`、`websecure:443`、REST API `http:8080`、证书/ACME 工作目录；它支持 HTTP、HTTPS 与手工 PEM/mkcert，不要求 ACME email。

本期 DNS-01 不应把上述部署拓扑搬到 GatewayConfig，再由 worker 临时创建 mount、endpoint、listener 或完整静态配置。Gateway 的职责是管理其业务属性、选择 ACME profile，并在部署时向已声明的 Traefik Component 注入少量 Gateway 值。

## Goal

1. 保持基础 Gateway Version 的 HTTP/HTTPS、手工证书和固定 TCP entrypoint 能力。GatewayConfig 不保存 platform listener，也不生成 Component mount、endpoint 或 resolver 结构。
2. 每个 Gateway 创建一个基础 Version，以及 `http`、`dns`、`http-dns` 三个 ACME profile Version。profile Version 固化对应 resolver 结构；启用 ACME 时三选一，未启用时使用基础 Version，不引入名为 `none` 的 profile。
3. GatewayConfig 保存 ACME profile、ACME email 和 Cloudflare DNS API token。DNS token 不再从 `POMELO_ORBIT_CF_DNS_API_TOKEN` 或 Settings 读取。
4. Gateway 部署先依据保存的 profile 选择并持久化 Service 的普通 Version，再走既有 Deployment 管线。worker 仅对已声明 Traefik Component 写入 Gateway email，并在 DNS profile 直接注入 `CF_DNS_API_TOKEN`；不将这些值回写 Version。
5. TCP 端口继续由 Gateway Version 的静态 Traefik entrypoint 与 Component endpoint 声明；未使用时不创建 TCP Route。本期不提供 Gateway listener rows 或部署期动态端口。
6. 保留 Component Docker label、派生本地 Host、自定义 HTTP/TCP Route、手工 PEM、mkcert 与现有通用 Version/Service 操作。

## Non-goal

- 不将 HTTP-01、DNS-01 或 DNS token 应用于普通 Component Docker labels。
- 不创建通用 Version controlled-file 模板/变量机制，也不通过 Service overlay 替换 `traefik.yml` 内容。
- 不保留 `cert.letsencrypt.*`、`POMELO_ORBIT_CF_DNS_API_TOKEN`、secret workspace、Compose secret、token 脱敏或 provider availability 的兼容路径。
- 不在 GatewayConfig 中保存动态 TCP listener、镜像、Component mount、endpoint、Traefik static YAML 或 Version 的完整副本。
- 不新增 Gateway 专用 Application/Version 生命周期，或阻止通用 Version/Component/Service 编辑、fork、选择和部署。

## User Scenarios

1. 管理员创建 Gateway，系统创建基础 Version、三个 ACME profile Version 和默认绑定基础 Version 的停止态 Service。基础 Gateway 可立即用于 HTTP、HTTPS 手工证书和已声明 TCP 端口，无需 email。
2. 管理员在 Gateway 编辑页选择 `http`、`dns` 或 `http-dns`，填写 email；选择含 DNS 的 profile 时填写 Cloudflare token。保存不部署。
3. 管理员部署 Gateway。系统将 Service 切换到 profile 对应的普通 Version，快照 Gateway 配置；worker 保留 Version 拓扑，只把 email 写入该 profile 的 resolver，并把 DNS token 作为 Traefik Component environment 注入。
4. 管理员为 Route 启用 Let's Encrypt。HTTP-01 只能在 `http`/`http-dns` profile 下选择；DNS-01 只能在 `dns`/`http-dns` 且 token 已填写时选择。
5. 管理员需要新的 TCP 端口时，按既有 Version/Component 工作流声明 Traefik entrypoint 与 host endpoint，部署该 Version，再创建 TCP Route；GatewayConfig 不出现 listener 编辑器。

## Acceptance

- [ ] 基础 Version 保留 80/443/8080、socket、static config、cert/acme mount 与 HTTP/HTTPS 手工证书能力，不要求 ACME email。
- [ ] 每个 Gateway 有精确绑定的 `http`、`dns`、`http-dns` profile Version；profile 结构仅存在于 Version/Component。
- [ ] GatewayConfig 仅保存控制面、域名/label 默认策略、profile 选择、email 和 DNS token；不保存 listener 或 resolver 布局。
- [ ] 部署时先将 Gateway Service 指向选定普通 Version；worker 不追加/删除 Component mount、endpoint、listener 或 resolver。
- [ ] Gateway 部署期仅修改已声明 Traefik static config 中的 ACME email，并在 DNS profile 的 Compose environment 直接传递 `CF_DNS_API_TOKEN`。
- [ ] 移除 `POMELO_ORBIT_CF_DNS_API_TOKEN`、Cloudflare Settings 配置、secret workspace、Compose secret、token redaction 和 DNS credential availability API。
- [ ] Route challenge 可用性由 Gateway 的 profile 和 DNS token 定义；HTTP/DNS resolver 名保持 `letsencrypt` / `letsencrypt-dns`。
- [ ] TCP listener 仅来自选中 Gateway Version；本期 Gateway API/UI/schema 无 listener 行。

## Decisions

- 基础 HTTP/HTTPS 与 ACME profile 是不同概念。基础 Version 不是 `none` profile；它是 ACME 未启用时的正常 Gateway 版本。
- GatewayConfig 的 `acme_profile` 为空时使用 Service 当前基础 Version；非空时为 `http`、`dns`、`http-dns` 三选一，并映射到已创建的 profile Version。
- email 和 DNS token 是 Gateway 属性，不是 Version template 参数、Service runtime config 或 Orbit 全局设置。它们仅作为 Gateway deployment snapshot 的输入。
- Gateway deployment 的额外处理是窄范围的：修改已存在的 resolver email 标量，或为 DNS profile 设置已知环境变量；它不构成动态 Component topology projection。
- DNS token 直接保存、返回并写入 Gateway Compose environment；本期不增加额外 secret/redaction 机制。

## Risks

- 修改 email、token 或 profile 后需重新部署 Gateway；已运行的 Traefik 不会自动变化。
- DNS token 将存在 Gateway API、deployment snapshot、rendered Compose 和容器环境中，这是本期明确接受的可见性边界。
- 固定 TCP 端口在不同 profile Version 间需要由管理员保持一致；Route 最终以选中 Version 的实际声明为准。

## User Review Notes

- 2026-08-26：确认 HTTP/HTTPS 基础 Gateway 不需要 email；不创建 `none` ACME profile。
- 2026-08-26：确认不实现通用 controlled-file 变量能力；email 由 Gateway deployment 专项处理。
- 2026-08-26：确认 DNS token 不再使用 `POMELO_ORBIT_CF_DNS_API_TOKEN`，改由 Traefik Gateway 自行管理。
- 2026-08-26：确认 TCP entrypoint/端口为 Version 现有能力，不纳入 Gateway 动态 listener 配置。
