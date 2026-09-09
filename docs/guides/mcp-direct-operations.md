# MCP 直接操作
最后修改时间: 2026-09-09 17:04:54

## 命名与启动约定

Codex 注册名为 `pomelo-orbit-mcp`，本地 stdio 入口为 `go run ./cmd/server mcp`，工具在客户端中显示为 `mcp__pomelo-orbit-mcp__orbit_*`。它直接构造 Orbit Go delivery MCP Core，工具调用不会回环到 Orbit HTTP API，也不提供远程 `/mcp` 端点。先在已登录的 Orbit Web 控制台“系统管理 / 访问令牌”创建一个命名 PAT，并在创建窗口中复制一次。`initialize` 和 `tools/list` 不读取或验证凭据；每次实际 `tools/call` 都使用 `POMELO_ORBIT_MCP__ACCESS_TOKEN` 调用同一 Auth Service 查找 PAT 摘要、校验未撤销/未过期及用户 enabled 状态，并把 session 固定到首次通过校验的用户。父进程通过项目 `.codex/config.toml` 的 `env_vars` 将变量传给 stdio 子进程；不要将 token 写入配置文件、浏览器 localStorage 或用户配置目录。PAT 默认不过期，也可创建为有限有效期；撤销、到期或替换后，更新父进程环境并重启 MCP session。stdio 仍需运行在可访问同一 Orbit 数据库、签名配置、Docker 和 workspace 的可信环境。不要使用已废弃的注册名、浏览器授权页面或 Python MCP 命令。

`pomelo-orbit-mcp` 只执行用户明确要求的独立动作。调用结果为 `isError=true` 时，该调用失败；不要自动执行依赖它的后续动作。

对于带持久卷的状态服务，MCP Server instructions 和环境变量工具共同约束：Version 中由 Service 决定的环境值必须使用精确 `${KEY}` 占位，具体值只通过 `orbit_update_service_env` 保存；新 Service 的口令、令牌和密钥一次安全随机生成后跨重部署保持稳定且不在报告中暴露。数据库镜像的 bootstrap 环境变量只在空数据卷生效，因此修改 Service 值后重部署不会轮换既有数据库凭据；必须取得用户对原地轮换或重置卷的明确授权。

| 用户明确要求 | 调用工具 | 不隐含的动作 |
| --- | --- | --- |
| 查看项目部署 Environment | `orbit_get_project_environment` | 编辑、Probe、部署或 Gateway provision |
| 编辑项目部署 Environment | `orbit_update_project_environment` | Probe、Gateway provision、部署或同步 Route |
| Probe 项目部署 Environment | `orbit_probe_project_environment` | Gateway provision、部署或同步 Route |
| 供应或复用 Traefik Gateway | `orbit_provision_gateway` | 修改已有 Gateway 配置、覆盖已有 Service runtime configuration、删除失败资源 |
| 查看 Application 的 Service | `orbit_list_application_services` | deploy、stop、delete |
| 查看或编辑自定义 Route | `orbit_list_routes`、`orbit_get_route`、`orbit_update_route` | enable、disable、删除或同步 |
| 创建自定义 Route | `orbit_create_route` | enable、deploy Gateway、删除或同步 |
| 启用自定义 Route | `orbit_enable_route` | deploy Gateway、停用、删除或同步 |
| 停用自定义 Route | `orbit_disable_route` | 删除、同步或重新部署 Gateway |
| 部署 | `orbit_deploy` | wait、verify、stop、delete |
| 等待 Deployment | `orbit_wait_deployment` | verify、stop、delete |
| 验证 Deployment | `verify_deployment` | stop、delete |
| 停止 Service | `orbit_list_application_services` 后选择 `orbit_stop` 的 `service_id` | 删除 volume、删除 Application |
| 删除受管 volume | `orbit_stop(remove_volumes=true)` | 删除 Application |
| 删除 Application | `orbit_delete_application` | 删除目录，除非显式传入 `remove_dir=true` |

`orbit_wait_deployment` 的 deployment 终态和 `timed_out`，以及 `verify_deployment` 的 `failed`、`drift`、`inconclusive`，都是正常领域结果。Orbit API、Docker、runtime target、输入校验和内部失败则统一返回结构化 MCP error result。

`orbit_create_gateway` 创建完整 Gateway 资源组：Application、GatewayConfig、初始可编辑 Version/Traefik Component 和默认停止态 Service。`orbit_provision_gateway` 是幂等资源准备工具：它按 `project_id` 与 `code=traefik` 查找 Gateway，零个时创建、恰好一个时复用、多个时返回冲突；指定实例不存在时按通用 Service 创建语义新增停止态 binding。它不会发布 Version、部署、等待或检查 Docker 网络。需要运行 Gateway 时，随后显式调用 `orbit_deploy(service_id)`，并按需调用 `orbit_wait_deployment`。

每个 Project 只有一个 deployment Environment，target type 为 `local` 或 `ssh`。`orbit_get_project_environment`、`orbit_update_project_environment` 与 `orbit_probe_project_environment` 始终以 `project_id` 作为授权和目标 scope，不接受 `environment_id`。local 输出仅包含控制面工作目录；SSH 输入/输出使用 nested `ssh` object 和主机指纹。MCP 不接受或输出部署私钥、bootstrap 密码、bootstrap 私钥或初始化命令。SSH 第一次探测成功后记下主机密钥指纹。`ssh` 到 `127.0.0.1` 仍按 SSH 执行。编辑 target 后必须显式 Probe 成功，才能 provision Gateway 或创建新的部署。

通过 `orbit_create_version` 或 `orbit_create_version_component` 创建 Component 时，必须显式提交 `pull_policy` 和 `restart_policy`；策略分别只能是 `missing`、`always`、`never` 和 `no`、`on-failure`、`always`、`unless-stopped`。

Route 工具使用 Route 表单的持久化字段。HTTP Route 的 `path_prefix` 可省略并默认 `/`，且必须提供受管 HTTP target（`service_id`、`component_name`、`endpoint_protocol`、`endpoint_container_port`）或高级 `target_url` 之一；两种 target 互斥。TCP Route 必须提供受管 TCP target 与 `listen_port`，不能使用 `path_prefix` 或 `target_url`。启用 TCP Route 前，目标 Gateway Version 必须已经声明并部署对应的 `tcp<listen_port>` entrypoint 与宿主机端口。

`runtime_doctor` 只接受一个已受管运行时目标：传 `application_id` 与可选 `instance_key`，或传 `gateway_application_id` 与可选 `gateway_instance_key`。运行时工具从 Application 的 Project 解析唯一 Environment；它们不接受 `environment_id`，并在输出中返回派生的 `project_id`。

`runtime_compose_ps` 和 `verify_deployment` 默认返回摘要。需要原始 Compose、inspect 或完整 evidence 时，明确传入 `detail=true`；也可以使用已有的 scoped logs、container inspect、network inspect 和 compose config 工具。MCP Server 是 stdio 进程，修改工具后需要重启 MCP client session，并通过 Server instructions 中的 source/schema 指纹确认新的工具表已生效。

Component 写操作已细分为 `orbit_update_version_component_basic`、`runtime`、`endpoints`、`env`、`mounts`、`dependencies` 和 `advanced`。`orbit_update_version_component_mounts` 接收顶层 `mounts` 集合：除 `named_volume` 外，`directory`、`file` 和 `controlled_file` 的 `source` 必须是绝对路径或显式以 `./` 开头的相对路径，例如 `/etc/app/app.env` 或 `./config/app.env`；裸的 `config/app.env` 会被 Compose 解释为卷名，因此不允许。`named_volume` 只使用裸卷名；`controlled_file` 还要求 `source_is_host_path=false`，`content` 可为空，`mode` 必须是四位 Unix 八进制值，且内容上限为 256 KiB。端点使用 `internal`、`local`、`host` 或 `gateway` mode：`gateway` 仅用于 HTTP 派生域名；公开 TCP 使用自定义 Route 的 `domain:listen_port` 到项目内 Service Component 的 `internal` TCP Endpoint。`gateway` 不替代 `local`/`host` 的直接端口映射；每个 TCP 监听端口只能有一条启用 Route。旧的 `orbit_update_version_component_connectivity` 不再注册，且 Component JSON 不再接受 `networks`；调用方必须使用新 MCP session 读取工具 schema 后再写入。

Service Component 的实例级运行配置通过 `orbit_update_service_component_overlay` 与其他稀疏覆盖一起完整替换。`entrypoint`、`command`、`pull_policy` 和 `restart_policy` 省略时会清除对应的 Service 覆盖并按 Version 声明生效；空的 `entrypoint` 或 `command` 字符串用于明确选择空 argv。调用方必须保留未修改的 env、mounts、resources 和 endpoints；挂载覆盖中的 `source_is_host_path` 会原样传递。
