# MCP 直接操作

`pomelo_orbit` 只执行用户明确要求的独立动作。调用结果为 `isError=true` 时，该调用失败；不要自动执行依赖它的后续动作。

| 用户明确要求 | 调用工具 | 不隐含的动作 |
| --- | --- | --- |
| 供应或复用 Traefik Gateway | `orbit_provision_gateway` | 修改已有 Gateway 配置、覆盖已有 Service runtime configuration、删除失败资源 |
| 查看 Application 的 Service | `orbit_list_application_services` | deploy、stop、delete |
| 部署 | `orbit_deploy` | wait、verify、stop、delete |
| 等待 Deployment | `orbit_wait_deployment` | verify、stop、delete |
| 验证 Deployment | `verify_deployment` | stop、delete |
| 停止 Service | `orbit_list_application_services` 后选择 `orbit_stop` 的 `service_id` | 删除 volume、删除 Application |
| 删除受管 volume | `orbit_stop(remove_volumes=true)` | 删除 Application |
| 删除 Application | `orbit_delete_application` | 删除目录，除非显式传入 `remove_dir=true` |

`orbit_wait_deployment` 的 deployment 终态和 `timed_out`，以及 `verify_deployment` 的 `failed`、`drift`、`inconclusive`，都是正常领域结果。Orbit API、Docker、runtime target、输入校验和内部失败则统一返回结构化 MCP error result。

`orbit_create_gateway` 只创建低层 Gateway 配置；`orbit_provision_gateway` 是明确的高层生命周期工作流。后者会按 `project_id` 与 `code=traefik` 查找 Gateway：零个时创建，恰好一个时复用，多个时返回冲突。它随后准备同一 `instance_key` 的 Service、发布 Version、部署、等待，并检查固定的 `traefik` bridge 网络。部署失败或超时时不会继续网络检查，也不会自动删除已经创建的资源。

没有受管 Application target 时，使用 `runtime_doctor(network_name="traefik")` 进行只读预检。该参数只接受 `traefik`，不能与 Application 或 Gateway target 组合；`runtime_network_inspect` 仍只允许读取从受管 Compose target 派生的网络。

`runtime_compose_ps` 和 `verify_deployment` 默认返回摘要。需要原始 Compose、inspect 或完整 evidence 时，明确传入 `detail=true`；也可以使用已有的 scoped logs、container inspect、network inspect 和 compose config 工具。MCP Server 是 stdio 进程，修改工具后需要重启 MCP client session，并通过 Server instructions 中的 source/schema 指纹确认新的工具表已生效。

Component 写操作已细分为 `orbit_update_version_component_basic`、`runtime`、`ports`、`env`、`mounts`、`dependencies` 和 `advanced`。旧的 `orbit_update_version_component_connectivity` 不再注册，且 Component JSON 不再接受 `networks`；调用方必须使用新 MCP session 读取工具 schema 后再写入。
