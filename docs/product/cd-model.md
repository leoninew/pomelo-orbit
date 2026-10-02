# Project 环境与 CD 产品模型
最后修改时间: 2026-10-02 10:42:03

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

Environment 保存的 `workspace_root` 是该 Project 的工作区根目录；CD 使用 `<workspace_root>/deployment/<service-code>`，Gateway certificates、独立 Route 文件及发布记录也位于对应 Service 目录下。当前 CI 执行器支持通过最新 Probe 的 `local` Environment，以及受管密钥和 host-key pinning 就绪的 Windows/Linux SSH Environment；分别在本机或远端 `<workspace_root>/pipeline` 中执行。SSH 登录用户还须能写入目标的 `pipeline` 目录，且目标 Docker daemon 能挂载相应宿主路径；Environment Probe 成功不保证已存在的 `pipeline` 子目录可写。SSH 目标绝不能作为控制面 Docker 的本地挂载源，也不在 SSH 失败时回退本地。值可以是平台绝对路径或 `~` / `~/...`，配置、界面和库存都原样保存，只在使用时展开 `~`。控制面仅保留 `workspace.root` 作为无 Project 的启动配置基准；`logging.deployment_root` 仅用于控制面部署日志。Environment 与已绑定的 Gateway 不提供删除能力。Project code 同时是 Environment code。Gateway 和声明加入 Traefik 的 Service 共享部署宿主上的 Docker bridge network `traefik`；网络名不由 Environment code 派生。Gateway Component 名称固定为 `traefik`，初始 pull policy 固定 `missing`，二者都不是 GatewayConfig 字段。

## Environment

Environment 的 target type 是显式联合：

- `local`：在 Orbit 控制面宿主机的 Docker daemon 上执行，工作目录为该 Environment 保存的 `workspace_root`；`~` / `~/...` 在使用时展开为控制面进程用户主目录。它不保存 SSH host、用户、私钥、host key 或 SSH 初始化认证。
- `ssh`：Linux OpenSSH + Docker Engine/Compose，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。完整 SSH target 可以先保存为未初始化状态；用户显式生成初始化命令时才创建或复用其 `environment_credential` binding。命令执行并通过首次 Probe 后才记录 host-key fingerprint。工作目录同样由该 Environment 保存的 `workspace_root` 提供；`~` / `~/...` 在使用时展开为远端登录用户主目录。

一个 Docker target 只能绑定一个活跃 Project。`local` target 全局独占；`ssh` target 按精确的 host + port 独占。已废弃 Project 保留其历史 Environment，但不再占用 target；Gateway 使用固定端口和共享 `traefik` 网络，保存 Environment 时若发现其他活跃 Project 已绑定同一 target 会直接拒绝。

`ssh` 到 `127.0.0.1` 仍是 SSH target，必须使用密钥认证和 host-key pinning，不会转换为 `local`。保存、编辑、状态读取和 Probe 不创建、轮换或替换部署密钥；未生成初始化命令时 Probe 返回可恢复诊断。部署私钥只属于 SSH Environment，存在 `environment_credential`，不能通过仓库凭据 API、MCP 或日志读取。Probe、部署和 CI 不使用交互式 shell 或 PTY，也不支持密码认证、端口转发或操作者个人私钥。环境页终端是独立能力：当前 Project 成员可在最新 Probe 成功后，以该 Environment 保存的 SSH 用户和受管密钥打开目标宿主机的交互式 PTY shell；终端不进入应用容器或 Windows 的 WSL2 发行版。

终端抽屉关闭后保留当前环境页内的终端实例、输出与 SSH 会话，重新打开继续同一会话；离开环境页、切换 Project、目标或凭据变更、主动断开以及服务端会话限制会终止连接。断线后仅手动重连。

镜像 registry、登录方式和多 registry 配置是宿主机责任，不属于 Orbit Project 或 Environment 配置。

## Gateway

Gateway 是一个绑定到 Project Environment 的普通 Application。每个 Environment 仅允许一个 Gateway 和一个受管 Service。Application name/code 是 Project 内唯一的产品身份 `Traefik` / `traefik`，不把 Project code 拼进名称；受管 Service 的既有 code 为 Project 内唯一的 `traefik-default`，但该字面量只是一项稳定编码，不表示默认实例。工作目录和 Compose project 都使用 Service code。普通 Application 可以拥有多条 Service；每条 Service 以 Project 内唯一 code 区分。创建时生成 `base`、`http`、`dns`、`http-dns` 四个普通 Version；它们与普通 Version/Component 一样可以查看、编辑、fork 和部署。

GatewayConfig 保存控制面 REST URL/readiness、`internal_domain`、可选的 `external_domain`、Component label 默认策略、ACME profile、email 和 DNS token。`internal_domain` 沿用原有域名机制，供 gateway mode 的 Docker labels 和 Gateway 派生地址使用；`external_domain` 只为创建、编辑自定义 Route 时拼接域名提供后缀，不管理 DNS 记录，也不改变已有 Route。未配置外网域名时，Route 创建表单的域名控件留空。它不保存 Component 名称、镜像、mount、endpoint、TCP listener 或 resolver 布局。profile 选择对应的绑定 Version；空 profile 使用 `base` Version。

自定义 Route 的编码使用最多 32 字符的小写 DNS 单标签（以字母开头，仅含 `a-z`、`0-9`、`-`，末尾为字母或数字）。外网域名后缀支持多级子域名，各级标签遵循 ASCII 主机名规则，末级不能纯数字。配置外网域名后，在创建或编辑表单修改编码会将 `编码.external_domain` 填入域名控件；已手工填写的其他域名不会被覆盖。Route 仍持久化完整域名，修改 Gateway 的外网域名不会批量更新 Route。

| 能力 | 来源 |
| --- | --- |
| Traefik 镜像、端口、socket、其他 provider 与 resolver YAML | Gateway Version / Component |
| Route File provider、dynamic/certs/acme 受管目录挂载 | Gateway 有效部署计划统一覆盖，不写回 Version |
| Component Docker label 默认 entrypoint/TLS | GatewayConfig |
| HTTP-01 / DNS-01 可用性 | GatewayConfig profile，DNS 另需 Gateway token |
| TCP Route listener | 选中 Gateway Version endpoint |

DNS token 是 Gateway 属性，直接返回、快照并作为 `CF_DNS_API_TOKEN` 写入 DNS profile 的 Compose environment。本期不提供全局 token、secret workspace、Compose secret 或脱敏接口。

Gateway 部署会写入当前受管 File provider 配置并自动重建容器，迁移脚本初始化和已部署 REST 的受管实例同样适用。首次切换通过线下传达重新部署及显式同步一次的步骤，不增加 UI 或转换旧发布快照；部署不会从业务 Route 自动发布草稿。
