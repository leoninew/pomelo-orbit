# CD 产品模型
最后修改时间: 2026-09-04

Doc role: living product model

## 核心对象

| 对象 | 职责 |
| --- | --- |
| Project | 成员权限、资源隔离和部署环境切换边界；类似轻量租户 |
| Environment | Project 唯一的 SSH 部署宿主，保存平台、SSH 目标、受管私钥绑定和主机密钥指纹 |
| Application | 通用应用元数据，属于一个 Project |
| Version / Component | 可编辑、可 fork 的 Compose 拓扑声明 |
| Service | Version 的运行实例和运行时覆盖 |
| Deployment | 异步部署任务及不可变执行输入 |
| GatewayConfig | Gateway Application 的业务属性和 ACME profile 选择 |
| Route | 自定义 HTTP/TCP 入口和受管 target |

`Project 1:1 Environment 1:1 Gateway`。切换当前 Project 就是切换部署环境和默认 Gateway。关联采用逻辑外键：Environment 的 `project_id`、`gateway_application_id` 和 SSH credential binding 均不使用数据库物理外键。

Environment 与已绑定的 Gateway 不提供删除能力。Project code 同时是 Environment code，环境级 Docker 网络由它稳定派生为 `orbit-<environment-code>-traefik`。

## Environment

Environment 只能是下列运行目标：

- Linux OpenSSH，宿主具备 Docker Engine 与 Docker Compose。
- Windows native OpenSSH，宿主通过 WSL2 Docker Desktop 提供 Docker Engine 与 Docker Compose。

不支持本机 Docker 回退、macOS、其他 Windows Docker 形态或任意 SSH command 执行。部署私钥是 Project Environment 管理的内部凭据，不能通过凭据 API、MCP 或日志读取。连接必须严格匹配保存的 `SHA256:` SSH host-key fingerprint；不支持密码认证、交互式 shell、PTY 或端口转发。

镜像 registry、登录方式和多 registry 配置是宿主机责任，不属于 Orbit Project 或 Environment 配置。

## Gateway

Gateway 是一个绑定到 Project Environment 的普通 Application。每个 Environment 仅允许一个 Gateway 和一个停止态 `default` Service。创建时生成 `base`、`http`、`dns`、`http-dns` 四个普通 Version；它们与普通 Version/Component 一样可以查看、编辑、fork 和部署。

GatewayConfig 保存控制面、base domain、Component label 默认策略、ACME profile、email 和 DNS token。它不保存镜像、mount、endpoint、TCP listener 或 resolver 布局。profile 选择对应的绑定 Version；空 profile 使用 `base` Version。

| 能力 | 来源 |
| --- | --- |
| Traefik 端口、socket、cert/acme mount、resolver YAML | Gateway Version / Component |
| Component Docker label 默认 entrypoint/TLS | GatewayConfig |
| HTTP-01 / DNS-01 可用性 | GatewayConfig profile，DNS 另需 Gateway token |
| TCP Route listener | 选中 Gateway Version endpoint |

DNS token 是 Gateway 属性，直接返回、快照并作为 `CF_DNS_API_TOKEN` 写入 DNS profile 的 Compose environment。本期不提供全局 token、secret workspace、Compose secret 或脱敏接口。
