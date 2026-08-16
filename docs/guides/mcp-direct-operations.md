# MCP 直接操作
最后修改时间: 2026-08-16 22:49:05

## 命名与启动约定

Codex 注册名为 `pomelo_delivery`，本地 stdio 入口为 `go run ./cmd/server mcp`，工具在客户端中显示为 `mcp__pomelo_delivery__orbit_*`。它直接构造绑定当前用户的 Go delivery MCP Core，工具调用不会回环到 Orbit HTTP API。Server 初始化和工具发现不会读取或验证本地凭据；首次实际 `tools/call` 时，缓存凭据缺失或失效才会打开配置的 Orbit 浏览器登录页。浏览器以短时一次性授权码回调本机 loopback，stdio 进程交换并将 bearer credential 保存到用户配置目录。不要复制浏览器 localStorage token，也不要使用已移除的 `pomelo_orbit` 注册名或 Python MCP 命令。

`pomelo_delivery` 只执行用户明确要求的独立动作。调用结果为 `isError=true` 时，该调用失败；不要自动执行依赖它的后续动作。

| 用户明确要求 | 调用工具 | 不隐含的动作 |
| --- | --- | --- |
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

Route 工具使用 Route 表单的持久化字段。HTTP Route 的 `path_prefix` 可省略并默认 `/`，且必须提供受管 HTTP target（`service_id`、`component_name`、`endpoint_protocol`、`endpoint_container_port`）或高级 `target_url` 之一；两种 target 互斥。TCP Route 必须提供受管 TCP target 与 `listen_port`，不能使用 `path_prefix` 或 `target_url`。启用 TCP Route 前，目标 Gateway Version 必须已经声明并部署对应的 `tcp<listen_port>` entrypoint 与宿主机端口。

没有受管 Application target 时，使用 `runtime_doctor(network_name="traefik")` 进行只读预检。该参数只接受 `traefik`，不能与 Application 或 Gateway target 组合；`runtime_network_inspect` 仍只允许读取从受管 Compose target 派生的网络。

`runtime_compose_ps` 和 `verify_deployment` 默认返回摘要。需要原始 Compose、inspect 或完整 evidence 时，明确传入 `detail=true`；也可以使用已有的 scoped logs、container inspect、network inspect 和 compose config 工具。MCP Server 是 stdio 进程，修改工具后需要重启 MCP client session，并通过 Server instructions 中的 source/schema 指纹确认新的工具表已生效。

Component 写操作已细分为 `orbit_update_version_component_basic`、`runtime`、`endpoints`、`env`、`mounts`、`dependencies` 和 `advanced`。`orbit_update_version_component_mounts` 接收顶层 `mounts` 集合：`controlled_file` 的 `source` 是相对的受管路径，`source_is_host_path=false`，`content` 可为空，`mode` 必须是四位 Unix 八进制值，且内容上限为 256 KiB。端点使用 `internal`、`local`、`host` 或 `gateway` mode：`gateway` 仅用于 HTTP 派生域名；公开 TCP 使用自定义 Route 的 `domain:listen_port` 到项目内 Service Component 的 `internal` TCP Endpoint。`gateway` 不替代 `local`/`host` 的直接端口映射；每个 TCP 监听端口只能有一条启用 Route。旧的 `orbit_update_version_component_connectivity` 不再注册，且 Component JSON 不再接受 `networks`；调用方必须使用新 MCP session 读取工具 schema 后再写入。

Service Component 的实例级运行配置通过 `orbit_update_service_component_overlay` 与其他稀疏覆盖一起完整替换。`entrypoint`、`command`、`pull_policy` 和 `restart_policy` 省略时会清除对应的 Service 覆盖并按 Version 声明生效；空的 `entrypoint` 或 `command` 字符串用于明确选择空 argv。调用方必须保留未修改的 env、mounts、resources 和 endpoints；挂载覆盖中的 `source_is_host_path` 会原样传递。
