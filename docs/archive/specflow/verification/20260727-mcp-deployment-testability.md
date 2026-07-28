# MCP 直接部署测试可用性验证
最后修改时间: 2026-07-27 20:38:35

Review status: Draft

## Basis

- Requirement: `docs/requirement/20260727-mcp-deployment-testability.md` (`Accepted`)
- Spec: `docs/spec/20260727-mcp-deployment-testability.md` (`Accepted`)
- Plan: `docs/plan/20260727-mcp-deployment-testability.md` (`Accepted`)
- Flow mode: `standard` / 标准模式。

本轮验证当前暂存的 MCP、Go 和直接部署前端合同改动，以及为通过格式检查产生的一处未暂存格式调整；工作区另有未暂存 Web 改动，不纳入本功能结论。未读取、修改或删除仓库 `data/`，未执行 Docker、deploy、stop、restart、delete、wait 或 verify 生命周期动作。

## Requirement alignment

1. `.codex/config.toml` 声明了名为 `pomelo_orbit` 的本地 stdio MCP Server，使用相对 `cwd = "."` 与 `uv --directory mcp run pomelo-orbit-mcp`，不包含秘密、Docker 参数或 data-root 设置。
2. `codex mcp get pomelo_orbit` 显示 Server 已启用，transport 为 stdio，命令、工作目录和超时与 manifest 一致。
3. 新的隔离 Codex 子会话直接调用一次只读 `pomelo_orbit/orbit_list_projects`，调用成功并返回 1 个项目；未使用临时 stdio 客户端包装代码。
4. MCP 提供 `orbit_list_application_services`，通过允许字段投影返回 Service 身份、实例和状态摘要，不透传 `runtime_config`。
5. 已建立 MCP session 后的 validation、Orbit API、runtime 和内部异常统一映射为 `CallToolResult.isError=true`，安全文本和 `structuredContent.error` 使用同一错误契约；验证与稳定性检查不再把系统异常伪装为 `inconclusive`。
6. 事务 begin/commit/panic 路径使用现有 `{code, error, requestId}` HTTP 错误契约；commit 失败会丢弃缓冲的成功响应。
7. 新增业务 Application 的 Gateway 运行前置校验：未配置 Gateway，或已配置 Gateway 没有 `running` Service 时，deploy/restart 会在创建 Deployment、调度任务和执行 Docker 前返回带 Gateway code、ID、状态和操作建议的 validation error。

## Spec and plan alignment

| Expected area | Actual result |
| --- | --- |
| Project MCP registration | `.codex/config.toml`、MCP README 与直接操作指南已新增；直接只读 MCP 调用成功。 |
| Service 查询和错误契约 | Service 允许字段投影、Server `call_tool` 错误边界、stdio 协议测试和 verification 异常传播均已实现并通过测试。 |
| Go write error contract | 事务响应缓冲、begin/commit/panic 处理与 SQL mock 测试均已实现并通过测试。 |
| Direct Compose deployment capability | MCP 新增结构化 Version component/expose 输入；Go 支持条件式 `depends_on`，并在首次 deploy 时按 `instance_key` 创建或复用 Service。 |
| Web deployment contract | Application API 类型和全部 deploy 调用点改用 `instance_key`；无需先查询或创建 Service，前端 typecheck 通过。 |
| Gateway readiness before Docker | deploy/restart 在持久化或调度前检查 Gateway 是否实际运行；已覆盖未部署和运行中的 Gateway 两种情况。 |

## Actual diff summary

| Area | Changed files |
| --- | --- |
| MCP registration and guide | `.codex/config.toml`、`mcp/README.md`、`docs/guides/mcp-direct-operations.md` |
| MCP input, query and error handling | `mcp/src/pomelo_orbit_mcp/{version_specs,mcp_errors,server,orbit_client,verification}.py`、`tools/orbit.py` 及对应测试 |
| Compose, deployment and Gateway preflight | Application/Deployment/Gateway usecase、DTO、proto 和测试 |
| Web deployment callers | Application API、Application generated type、Version/Gateway/Service 详情和 Service 列表的 deploy 调用点 |
| Go transaction error handling | `internal/infrastructure/database/tx/request.go` 及测试，`go.mod`/`go.sum` |
| Compatibility caller | `scripts/ragflow_initialize.py` 将 deploy 参数由 `service_id` 改为 `instance_key` |

## Acceptance checklist

- [x] Codex 能显示并启动 `pomelo_orbit` stdio MCP Server。
- [x] 新 Codex 子会话可直接发现并调用 MCP 工具，无需临时 Python stdio wrapper。
- [x] Service 列表只返回允许字段，调用方可取得 `orbit_stop` 所需的 `service_id`。
- [x] MCP 测试覆盖错误类别、错误脱敏与领域结论边界。
- [x] Go 测试覆盖 Gateway 未运行时 deploy 在持久化/调度前被拒绝，以及 Gateway 运行时放行。
- [x] Go 事务测试覆盖 begin、commit、rollback 和 panic 的统一错误响应。
- [x] 前端 deploy 调用与 ApplicationDeployReq 均使用 `instance_key`，`vue-tsc --noEmit` 通过。
- [x] 本轮没有把 deploy、stop、delete 或“清理”组合为隐式流程，也没有操作 `data/`。
- [ ] 项目 manifest 在没有同名全局 MCP 配置的全新 Codex 环境中独立加载，尚未获得有效结论。

## Test results

| Command or check | Result |
| --- | --- |
| `make -C mcp check` | Passed: lock、Ruff lint/format、mypy；pytest 35 passed、1 deselected（显式 Docker 测试）。 |
| `go test ./...` | Passed. |
| `go test -count=1 ./internal/application/gateway/usecase ./internal/application/deployment/usecase ./internal/infrastructure/database/tx` | Passed; 强制实际执行关键包。 |
| `go vet ./cmd/... ./internal/... ./sql` | Passed. |
| `task test` | Passed: Web 43 passed，Go test passed。 |
| `yarn --cwd web typecheck` | Passed: `vue-tsc --noEmit`。 |
| `git diff --cached --check` | Passed. |
| `codex mcp get pomelo_orbit` | Passed: enabled stdio Server，配置与 manifest 一致。 |
| Fresh `codex exec` + one `orbit_list_projects` call | Passed: MCP 调用成功，返回 1 个项目。 |

## Scope deviations and risks

1. 最初接受的 Requirement/Spec/Plan 明确排除脚本；实际暂存 diff 修改了 `scripts/ragflow_initialize.py`，用于适配 deploy 的 `instance_key` 合同。直接 MCP 测试未调用该脚本，但这仍是文档范围外改动。提交前应由用户决定保留该兼容性更新，或拆分/回退该文件。
2. 为解决“Traefik 未部署直到 Docker 命令才失败”的实际问题，改动新增了 Gateway readiness、首次 deploy 创建 Service、结构化 Version 输入和条件 `depends_on`。这些改动与目标直接相关，但不在最初 Plan 的文件清单中，应作为同一功能的范围扩展审查。
3. 直接 MCP 调用证明 Server 能被当前 Codex 环境使用，但当前环境还存在同名全局注册；`--ignore-user-config` 会同时移除当前 provider 所需配置，因此未能隔离验证项目 manifest 的独立发现。
4. 本轮未重复 RAGFlow Docker 集成部署。Docker 集成测试保持显式且默认跳过，符合“不隐式执行生命周期动作”的约束。

## Incomplete items

1. 在不依赖同名全局 MCP 配置、且保留有效 Codex provider 配置的受信任会话中，单独验证 `.codex/config.toml` 的发现行为。
2. 确认或拆分 `scripts/ragflow_initialize.py` 的兼容性改动，并在选择后更新 Requirement/Spec/Plan 的范围记录。

## Conclusion

自动化测试、静态检查和一次真实只读 MCP 调用均通过；Gateway 未运行时提前失败的关键行为已由无缓存单元测试覆盖。由于项目 manifest 的独立发现尚未隔离验证，且存在超出已接受范围的脚本兼容性改动，本验证保持 `Draft`，等待上述两项范围与验收决策后再标记为 `Accepted`。
