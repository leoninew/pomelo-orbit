# 异步操作状态机收敛验证
最后修改时间: 2026-08-08 16:21:04

Review status: Draft

## Requirement Alignment

- Deployment、Pipeline Run 与 Pipeline Stage Run 只使用 `waiting_to_run`、`running`、`ran_to_completion`、`faulted`、`canceled` 五态；终态条件写入禁止覆盖已取消记录。
- Pipeline Run Retry 创建新 Run 并保留 `retry_of`，不把 Retry 作为状态转换，也不以来源 Run 状态作为前置条件。
- 未实际运行的下游 Stage 保持 `waiting_to_run`；取消只终结当前 `running` Stage。
- `Service.deploying` 已从运行态与操作守卫中移除，活跃 Deployment 成为并发与界面守卫依据。
- `worker.max_attempts` 在 task 创建时冻结到任务记录；开发默认值为 `1`，失败转换读取记录中的 `attempts/max_attempts`。

## Spec Alignment

- SQLite 与 MySQL `000031_async_operation_state_machine` 均保留 `background_task.max_attempts`；本任务不额外占用迁移版本。
- SQLite 为活跃 Deployment、活跃 Pipeline Run 和 `(pipeline_run_id, stage_id)` 建立唯一约束；MySQL 使用等价生成列唯一键。
- Deploy/Restart 保持以 Compose 命令零退出码认定成功，Healthcheck 不进入 Deployment 主状态判定；Pipeline timeout 仍写为 `faulted`。
- Cancel API 立即写入 `canceled`，执行器按 `worker.poll_interval` 观察该持久化信号并取消外部 Context。

## Plan Alignment

- 暂存区包含 70 个状态机专属文件：迁移、条件 SQL、sqlc/Proto 输出、Deployment/Pipeline Run/Task 用例、Service/Gateway 活跃守卫、前端状态显示、活文档和离线历史脚本。
- 阶段模板库、服务编码、共享 Schema/模型和版本组件端口默认值由并行工作保留在未暂存区，不计入本次状态机暂存范围。
- 离线脚本不被服务调用或启动流程执行；测试只使用临时数据库。

## Acceptance Checklist

- [x] 活文档定义状态集合、允许转换、终态、取消、Retry 与并发规则。
- [x] Deployment、Pipeline Run 与 Stage Run 状态写入均使用期望源状态条件。
- [x] Pipeline Run Retry 创建新记录并保留来源 Run。
- [x] 持久化任务预算控制 `pending`/`failed` 转换；测试覆盖预算为 `2` 时的重投和终结。
- [x] timeout 作为 `faulted` 原因，默认预算为 `1` 时 task 首次失败终结。
- [x] Compose 零退出码仍是 Deploy/Restart 成功条件，Healthcheck 不改变主状态。
- [x] 取消不可被后续成功或失败写入覆盖；未运行 Stage 保持 `waiting_to_run`。
- [x] Service 活跃 Deployment 独立于 Service 运行态暴露给 API 与 UI。
- [x] 开发库迁移状态 clean，既有 task 全部保有有效冻结预算。

## Test Results

| 检查 | 结果 |
|---|---|
| `task sqlc` | 通过 |
| `task proto` | 通过 |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `python -m unittest scripts/test_reconcile_legacy_service_deploying.py` | 通过，3 tests |
| `git diff --cached --check` | 通过 |
| `go run ./cmd/migrate -status` | `version=32 state=clean` |

## Development Database

- `schema_migrations`: `version=32`, `dirty=0`。
- `background_task`: 19 条记录，`min(max_attempts)=1`、`max(max_attempts)=1`、无小于 1 的预算。
- 只查询结构与聚合值，未读取 task payload、环境变量或其他敏感运行数据。

## Scope And Risks

- 未配置 `BACKEND_GO_E2E_CONFIG`，因此没有对真实 MySQL 实例执行 E2E 迁移；MySQL DDL 已由源码审查，仍应在可用实例上补跑。
- 未执行真实 Docker Compose 或容器生命周期命令；Deployment/Pipeline 取消、超时和竞争行为由 fake runner 与 Go 测试覆盖。
- 当前工作区含并行任务。状态机暂存区刻意未包含共享 Schema/生成模型和阶段模板库的未暂存改动；提交前应按任务边界再次复核暂存区，避免将它作为包含所有并行变更的完整工作区快照。

## Conclusion

状态机功能、迁移链、持久化任务预算、离线脚本与前端类型检查均通过当前验证。结论为代码级通过，保留真实 MySQL 与 Docker 运行时验证作为环境依赖的后续检查。
