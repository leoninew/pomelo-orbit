# MCP 直接操作

`pomelo_orbit` 只执行用户明确要求的独立动作。调用结果为 `isError=true` 时，该调用失败；不要自动执行依赖它的后续动作。

| 用户明确要求 | 调用工具 | 不隐含的动作 |
| --- | --- | --- |
| 查看 Application 的 Service | `orbit_list_application_services` | deploy、stop、delete |
| 部署 | `orbit_deploy` | wait、verify、stop、delete |
| 等待 Deployment | `orbit_wait_deployment` | verify、stop、delete |
| 验证 Deployment | `verify_deployment` | stop、delete |
| 停止 Service | `orbit_list_application_services` 后选择 `orbit_stop` 的 `service_id` | 删除 volume、删除 Application |
| 删除受管 volume | `orbit_stop(remove_volumes=true)` | 删除 Application |
| 删除 Application | `orbit_delete_application` | 删除目录，除非显式传入 `remove_dir=true` |

`orbit_wait_deployment` 的 deployment 终态和 `timed_out`，以及 `verify_deployment` 的 `failed`、`drift`、`inconclusive`，都是正常领域结果。Orbit API、Docker、runtime target、输入校验和内部失败则统一返回结构化 MCP error result。
