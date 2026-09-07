# 持续集成 MCP 集成评估验证记录
最后修改时间: 2026-08-04 17:21:58

Review status: Accepted

流程模式: 标准 / standard

## 需求对齐

- 对照已接受的 [持续集成 MCP 集成评估](../requirement/20260804-ci-mcp-integration-evaluation.md) 及其 2026-08-04 的实施决策，本次交付实现独立、只读的 `pomelo-pipeline-mcp`，不扩展 `pomelo-orbit-mcp`。
- MCP client 仅发起固定的受鉴权 `GET` 请求；没有触发、取消、重试、部署或其他写操作。
- Server 未注册进 `.codex/config.toml`，不会自动加入现有 stdio 会话，也不会把 CI 成功自动衔接为 Orbit 部署。
- 响应投影排除阶段/模板脚本、变量 default/value、运行错误文本和制品物理路径；阶段日志会进行常见凭据遮蔽，并通过受限 `max_bytes` 与 `offset` 支持续读。

## 计划对齐

- 按 [持续集成 MCP 集成评估计划](../plan/20260804-ci-mcp-integration-evaluation.md) 的首期范围，已实现阶段、模板、快照、运行、制品和阶段日志的查询工具。
- `pipeline_resolve_template_variables` 未纳入首期：现有 HTTP 契约为 `POST`，不满足本轮“固定 GET 查询”的最小权限边界。
- HTTP client 需要外部提供 `POMELO_PIPELINE_URL` 和短期 `POMELO_PIPELINE_JWT`；不接受用户名/密码，也不写入 token cache。
- 项目本地检查入口已通过 Makefile 和 README 明确为 Ruff、mypy 与 pytest。

## 实际 Diff 摘要

本记录只核对已暂存的 `mcp/` 范围，不包含工作区内并行的前端或其他改动。

| 预期区域 | 实际结果 |
| --- | --- |
| `mcp/README.md` | 将 pipeline MCP 标记为已实现的独立只读查询 Server。 |
| `mcp/src/pipeline-mcp/` 项目元数据 | 新增启动配置示例、Makefile，补充运行依赖、开发检查配置和锁文件。 |
| `pomelo_pipeline_mcp` 源码 | 新增 settings、HTTP client、错误映射、响应投影、Server 与只读工具注册。 |
| `pomelo_pipeline_mcp/tools/pipeline.py` | 注册 11 个 `pipeline_*` 查询工具，限制分页大小、日志 offset 和日志字节窗口。 |
| `mcp/src/pipeline-mcp/tests/` | 新增 client、投影与脱敏、工具 schema、配置读取测试。 |
| Codex 注册、Go CI API、`pomelo-orbit-mcp` | 未改动。 |

已暂存 MCP diff 共 20 个文件，新增 2,124 行、删除 6 行。实现包含：

- `pipeline_list_stages`、`pipeline_get_stage`
- `pipeline_list_templates`、`pipeline_get_template`、`pipeline_get_snapshot`
- `pipeline_list_runs`、`pipeline_list_repository_runs`、`pipeline_get_run`
- `pipeline_list_artifacts`、`pipeline_list_run_artifacts`
- `pipeline_get_stage_log`

## 验收清单

- [x] 独立 CI MCP 与 `pomelo-orbit-mcp` 保持职责和注册边界；没有 CI 到 CD 的自动调用。
- [x] 查询工具覆盖阶段、模板、快照、运行、制品与增量阶段日志。
- [x] client 使用固定 GET 路由和 Bearer JWT，不直连数据库、日志文件、Docker 或任务队列。
- [x] 工具 schema 要求资源范围标识，并限制分页大小、日志 offset 和单次日志读取窗口。
- [x] 变量、脚本、运行错误和制品路径不会进入投影；日志的凭据遮蔽与 UTF-8 字节截断有单元测试。
- [x] 未实现任何写工具、Codex 自动注册或 CI->CD 自动串联。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `uv --directory mcp/src/pipeline-mcp run ruff format --check .` | 通过，15 个文件已格式化。 |
| `uv --directory mcp/src/pipeline-mcp run ruff check .` | 通过。 |
| `uv --directory mcp/src/pipeline-mcp run mypy` | 通过，8 个源码文件无类型问题。 |
| `uv --directory mcp/src/pipeline-mcp run pytest` | 通过，15/15 用例通过，耗时 0.46 秒。 |
| `git diff --cached --check` | 通过。 |

## 范围偏差

无。本次实现遵循需求文档中已记录的“首期独立只读 CI MCP”实施决策；未实现的变量预览和全部写入能力仍留在后续需求范围。

## 风险与未完成项

- 未使用真实短期用户 JWT 连接实际 CI HTTP 服务，因此尚未进行项目成员授权、服务端错误契约和真实日志/制品响应的端到端验证。
- 单元测试覆盖 client 的固定 GET 请求、错误响应脱敏、投影脱敏、日志窗口与工具 schema；真实 API 契约变更仍需要在隔离环境中以新 stdio 会话复核。
- Server 尚未注册到 Codex 配置。注册前仍需审阅配置身份、工具 schema 和本地运行边界；注册后需要重启 stdio MCP 会话。

## 结论

静态检查、类型检查和 15 个单元测试全部通过。首期 `pomelo-pipeline-mcp` 的只读范围、最小权限边界和响应脱敏实现与已接受的需求和计划一致。真实 CI 服务的身份授权与端到端契约验证尚未执行，不影响本地实现验证结论，但应作为注册或启用前的后续验收项。
