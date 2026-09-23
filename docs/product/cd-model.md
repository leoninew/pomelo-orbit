# Project 环境与 CD 产品模型
最后修改时间: 2026-09-23 15:36:02

Doc role: living product model

## 核心对象

| 对象 | 职责 |
| --- | --- |
| Project | 成员权限、资源隔离和 CI/CD 共用环境切换边界；类似轻量租户 |
| Environment | Project 唯一的 CI/CD 共用执行目标；类型为控制面本机或 SSH 宿主（当前 CI 支持 local、Windows SSH 与 Linux SSH） |
| Application | 通用应用元数据，属于一个 Project |
| Version / Component | 可编辑、可 fork 的 Compose 拓扑声明 |
| Service | Version 的独立运行绑定和运行时覆盖，由 Project 内唯一的服务编码标识 |
| Deployment | 异步部署任务及不可变执行输入 |
| GatewayConfig | Gateway Application 的业务属性和 ACME profile 选择 |
| Route | 自定义 HTTP/TCP 入口和受管 target |

`Project 1:1 Environment 1:1 Gateway` 是就绪后的运行时关系，不是 Project 创建时的预置数据。空库 identity seed 和新创建的 Project 都只有 Project 与 membership；Environment 与 Gateway 只能由 Web Project Initialization Wizard 写入。切换当前 Project 就是切换该 Project 已保存的共用环境及关联的 CD Gateway。关联采用逻辑外键：Environment 的 `project_id`、`gateway_application_id`，以及仅 SSH Environment 对 `environment_credential` 的 binding 均不使用数据库物理外键。

Environment 保存的 `workspace_root` 是该 Project 的工作区根目录；CD 使用 `<workspace_root>/deployment/<service-code>`，Gateway certificates 与 Route snapshot 也位于对应 Service 目录下。当前 CI 执行器支持通过最新 Probe 的 `local` Environment，以及受管密钥和 host-key pinning 就绪的 Windows/Linux SSH Environment；分别在本机或远端 `<workspace_root>/pipeline` 中执行。SSH 登录用户还须能写入目标的 `pipeline` 目录，且目标 Docker daemon 能挂载相应宿主路径；Environment Probe 成功不保证已存在的 `pipeline` 子目录可写。SSH 目标绝不能作为控制面 Docker 的本地挂载源，也不在 SSH 失败时回退本地。值可以是平台绝对路径或 `~` / `~/...`，配置、界面和库存都原样保存，只在使用时展开 `~`。控制面仅保留 `workspace.root` 作为无 Project 的启动配置基准；`logging.deployment_root` 仅用于控制面部署日志。Environment 与已绑定的 Gateway 不提供删除能力。Project code 同时是 Environment code。Gateway 和声明加入 Traefik 的 Service 共享部署宿主上的 Docker bridge network `traefik`；网络名不由 Environment code 派生。Gateway Component 名称固定为 `traefik`，初始 pull policy 固定 `missing`，二者都不是 GatewayConfig 字段。

## Environment

Environment 的 target type 是显式联合：

- `local`：在 Orbit 控制面宿主机的 Docker daemon 上执行，工作目录为该 Environment 保存的 `workspace_root`；`~` / `~/...` 在使用时展开为控制面进程用户主目录。它不保存 SSH host、用户、私钥、host key 或 SSH 初始化认证。
- `ssh`：Linux OpenSSH + Docker Engine/Compose，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。完整 SSH target 可以先保存为未初始化状态；用户显式生成初始化命令时才创建或复用其 `environment_credential` binding。命令执行并通过首次 Probe 后才记录 host-key fingerprint。工作目录同样由该 Environment 保存的 `workspace_root` 提供；`~` / `~/...` 在使用时展开为远端登录用户主目录。

一个 Docker target 只能绑定一个活跃 Project。`local` target 全局独占；`ssh` target 按精确的 host + port 独占。已废弃 Project 保留其历史 Environment，但不再占用 target；Gateway 使用固定端口和共享 `traefik` 网络，保存 Environment 时若发现其他活跃 Project 已绑定同一 target 会直接拒绝。

`ssh` 到 `127.0.0.1` 仍是 SSH target，必须使用密钥认证和 host-key pinning，不会转换为 `local`。保存、编辑、状态读取和 Probe 不创建、轮换或替换部署密钥；未生成初始化命令时 Probe 返回可恢复诊断。部署私钥只属于 SSH Environment，存在 `environment_credential`，不能通过仓库凭据 API、MCP 或日志读取。SSH 连接不支持密码认证、交互式 shell、PTY 或端口转发，也不接收操作者个人私钥。

镜像 registry、登录方式和多 registry 配置是宿主机责任，不属于 Orbit Project 或 Environment 配置。

## Gateway

Gateway 是一个绑定到 Project Environment 的普通 Application。每个 Environment 仅允许一个 Gateway 和一个受管 Service。Application name/code 是 Project 内唯一的产品身份 `Traefik` / `traefik`，不把 Project code 拼进名称；受管 Service 的既有 code 为 Project 内唯一的 `traefik-default`，但该字面量只是一项稳定编码，不表示默认实例。工作目录和 Compose project 都使用 Service code。普通 Application 可以拥有多条 Service；每条 Service 以 Project 内唯一 code 区分。创建时生成 `base`、`http`、`dns`、`http-dns` 四个普通 Version；它们与普通 Version/Component 一样可以查看、编辑、fork 和部署。

GatewayConfig 保存控制面 REST URL/readiness、base domain、Component label 默认策略、ACME profile、email 和 DNS token。它不保存 Component 名称、镜像、mount、endpoint、TCP listener 或 resolver 布局。profile 选择对应的绑定 Version；空 profile 使用 `base` Version。

| 能力 | 来源 |
| --- | --- |
| Traefik 端口、socket、cert/acme mount、resolver YAML | Gateway Version / Component |
| Component Docker label 默认 entrypoint/TLS | GatewayConfig |
| HTTP-01 / DNS-01 可用性 | GatewayConfig profile，DNS 另需 Gateway token |
| TCP Route listener | 选中 Gateway Version endpoint |

DNS token 是 Gateway 属性，直接返回、快照并作为 `CF_DNS_API_TOKEN` 写入 DNS profile 的 Compose environment。本期不提供全局 token、secret workspace、Compose secret 或脱敏接口。
