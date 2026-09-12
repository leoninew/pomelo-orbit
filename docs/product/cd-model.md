# CD 产品模型
最后修改时间: 2026-09-11 14:48:17

Doc role: living product model

## 核心对象

| 对象 | 职责 |
| --- | --- |
| Project | 成员权限、资源隔离和部署环境切换边界；类似轻量租户 |
| Environment | Project 唯一的部署目标；类型为控制面本机或 SSH 宿主 |
| Application | 通用应用元数据，属于一个 Project |
| Version / Component | 可编辑、可 fork 的 Compose 拓扑声明 |
| Service | Version 的运行实例和运行时覆盖 |
| Deployment | 异步部署任务及不可变执行输入 |
| GatewayConfig | Gateway Application 的业务属性和 ACME profile 选择 |
| Route | 自定义 HTTP/TCP 入口和受管 target |

`Project 1:1 Environment 1:1 Gateway` 是就绪后的运行时关系，不是 Project 创建时的预置数据。空库 identity seed 和新创建的 Project 都只有 Project 与 membership；Environment 与 Gateway 只能由 Web Project Initialization Wizard 写入。切换当前 Project 就是切换该 Project 已保存的部署环境和默认 Gateway。关联采用逻辑外键：Environment 的 `project_id`、`gateway_application_id`，以及仅 SSH Environment 的 credential binding 均不使用数据库物理外键。

Environment 保存的 `workspace_root` 是 Service Compose、Gateway certificates 与 Route snapshot 的部署目标目录；值可以是平台绝对路径或 `~` / `~/...`，配置、界面和库存都原样保存，只在使用时展开 `~`。控制面 `workspace.pipeline` 与 `logging.deployment_root` 不属于 Environment。Environment 与已绑定的 Gateway 不提供删除能力。Project code 同时是 Environment code。Gateway 和声明加入 Traefik 的 Service 共享部署宿主上的 Docker bridge network `traefik`；网络名不由 Environment code 派生。Gateway Component 名称固定为 `traefik`，初始 pull policy 固定 `missing`，二者都不是 GatewayConfig 字段。

## Environment

Environment 的 target type 是显式联合：

- `local`：在 Orbit 控制面宿主机的 Docker daemon 上执行，工作目录为该 Environment 保存的 `workspace_root`；`~` / `~/...` 在使用时展开为控制面进程用户主目录。它不保存 SSH host、用户、私钥、host key 或 SSH 初始化认证。
- `ssh`：Linux OpenSSH + Docker Engine/Compose，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。它保存平台、SSH target、受管私钥 binding 与 host-key fingerprint，工作目录同样由该 Environment 保存的 `workspace_root` 提供；`~` / `~/...` 在使用时展开为远端登录用户主目录。

`ssh` 到 `127.0.0.1` 仍是 SSH target，必须使用密钥认证和 host-key pinning，不会转换为 `local`。不支持 macOS、其他 Windows Docker 形态或任意 SSH command 执行。部署私钥只属于 SSH Environment，不能通过凭据 API、MCP 或日志读取。SSH 连接不支持密码认证、交互式 shell、PTY 或端口转发。

镜像 registry、登录方式和多 registry 配置是宿主机责任，不属于 Orbit Project 或 Environment 配置。

## Gateway

Gateway 是一个绑定到 Project Environment 的普通 Application。每个 Environment 仅允许一个 Gateway 和一个停止态 `default` Service。Application name/code 是产品身份 `Traefik` / `traefik`，不把 Project code 拼进名称；default Service code 为 `traefik-default`。工作目录和 Compose project 都使用 Service code。创建时生成 `base`、`http`、`dns`、`http-dns` 四个普通 Version；它们与普通 Version/Component 一样可以查看、编辑、fork 和部署。

GatewayConfig 保存控制面 REST URL/readiness、base domain、Component label 默认策略、ACME profile、email 和 DNS token。它不保存 Component 名称、镜像、mount、endpoint、TCP listener 或 resolver 布局。profile 选择对应的绑定 Version；空 profile 使用 `base` Version。

| 能力 | 来源 |
| --- | --- |
| Traefik 端口、socket、cert/acme mount、resolver YAML | Gateway Version / Component |
| Component Docker label 默认 entrypoint/TLS | GatewayConfig |
| HTTP-01 / DNS-01 可用性 | GatewayConfig profile，DNS 另需 Gateway token |
| TCP Route listener | 选中 Gateway Version endpoint |

DNS token 是 Gateway 属性，直接返回、快照并作为 `CF_DNS_API_TOKEN` 写入 DNS profile 的 Compose environment。本期不提供全局 token、secret workspace、Compose secret 或脱敏接口。
