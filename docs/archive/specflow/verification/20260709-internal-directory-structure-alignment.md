# Verification: internal 后两级目录结构对齐与架构收敛

最后修改时间: 2026-07-12

## Verification target

本次验证完成 light / 轻量模式下的以下已实施 issue：

1. **Issue 1**：`internal/common/civariable` 迁出 `common`；
2. **Issue 2**：`internal/application/<domain>` 重整为 domain-first 的职责子目录；
3. **Issue 3**：`internal/repository` 的持久化 port 与 SQLC/SQLX implementation 对齐；
4. **Issue 4**：worker runtime 与 handler dispatch 收拢至 `internal/queue/worker`；
5. **Issue 5**：Traefik 与 Turnstile external adapter 收拢至 `internal/infrastructure/external/<provider>`；
6. **Issue 6**：HTTP handler 内的纯字段转换收敛至 domain-local mapper；
7. **Issue 7**：SQLC 生成目录与固定工具版本对齐至 `internal/gen/sqlc` / v1.30.0；
8. **Issue 8**：CI/CD 异步执行编排从 `internal/workflow/activity` 收敛到 application usecase。

对应 requirement：`docs/requirement/20260709-internal-directory-structure-alignment.md`。

## Requirement alignment

### Issue 1-7

Issue 1-7 的验证结论保持不变：CI variable 规则位于 application CI rule；application、repository、queue worker、external provider、HTTP mapper 和 SQLC generated code 均已按各自 issue 的最终边界归位，且未保留旧路径兼容层。

### Issue 8

| Requirement item | Actual result |
| --- | --- |
| workflow/activity 不形成第二套业务流程 | `ExecutePipelineRun`、CI stage DAG / log / artifact / credential execution，以及 CD deploy/restart/stop / rendering / materialization / status orchestration 均已归入 application CI/CD usecase。 |
| queue handler 仅作为入站适配器 | CI/CD handler 继续仅解析、校验 task payload 并调用 application execution service；未加入 repository、状态流转、渲染或 runner 逻辑。 |
| application execution contract 完整 | CI/CD port 新增 execution-only log store capability，组合已有 read 与 writer 能力；HTTP query constructor 仍只依赖 `LogReader`，不泄漏 local execution-log concrete type。 |
| bootstrap 作为组合根 | `NewTaskRouter` 构造 SQLX execution repository、local log store 和既有 application runners，再注入 `cisvc.NewExecutionService` 与 `cdsvc.NewExecutionService`。 |
| 删除旧职责路径 | 完整 `internal/workflow/activity/{ci,cd}` 与无内容的 `internal/workflow` tree 已删除；全仓 active Go 源未保留 `workflow/activity` import、wrapper、alias、forwarding constructor 或旧 DTO。 |
| 不创建空结构 | 未创建 workflow runtime、definition、trigger、engine、scheduler 或 registry 空目录；当前 queue worker 是实际运行机制，pipeline/deployment definition 继续由 application/model/repository 表达。 |
| 既有行为保持 | task type、payload key、handler 错误文案、queue lease/retry/concurrency、CI Docker 参数、CD Compose 参数、workspace/log 路径、CI DAG/cancel/credential 语义与 CD 三种 operation 的状态补偿均保持。 |

Issue 8 未修改 API route、request/response contract、task payload、migration、数据库 schema 或前端行为。

## Actual diff summary

- CI 的 PipelineRun execution、stage DAG 分层并发、stage/run 状态、log、artifact、credential rewrite 与 cancellation guard 迁至 `internal/application/ci/usecase/{execution.go,executor.go}`。
- CD 的 deploy/restart/stop execution、workspace materialization、init script normalization、deployment renderer 与 route label 注入迁至 `internal/application/cd/usecase/deployment_execution*.go`。
- CI execution input 归入 `application/ci/dto`；CI queue handler 删除对 workflow DTO 的依赖。
- CI/CD application port 分别声明 execution-only log store；worker execution service 取得 writer/read capability，HTTP 查询侧继续依赖 read-only capability。
- worker bootstrap 改为注入 application CI/CD execution services 与 infrastructure Docker/Shell runner adapters。
- Docker/Shell runner 与 local workspace filesystem mechanics 已移至 infrastructure；application 仅通过 workspace、streamed command 与 command-query ports 调用，CD status/log 不再 direct `os/exec`。
- SQLC/SQLX singleton lookup 在 adapter 边界将 `sql.ErrNoRows` 映射为 `repository.ErrNotFound`，application 已移除 database-driver sentinel 依赖。
- CI/CD command usecase 通过 typed dispatch port 发起业务动作；queue dispatch adapter 独占 task type 和 JSON payload 映射，application 不再依赖 `queue/task.Task` 或 generic payload contract。
- 已删除完整 `internal/workflow` source tree，未保留 compatibility layer。

## Expected vs actual changed files

| Expected | Actual |
| --- | --- |
| CI/CD execution 归入 application | 已完成：CI 位于 `application/ci/usecase`，CD 位于 `application/cd/usecase`。 |
| handler 仅负责 payload adapter | 已完成：CI handler 依赖 application CI DTO；CD handler 继续只依赖 execution interface。 |
| bootstrap 更新最终 composition | 已完成：`internal/bootstrap/worker.go` 构造 application execution services 并保留四个 task registration。 |
| workflow 旧路径与兼容层删除 | 已完成：`internal/workflow` 不存在，active Go source 无旧 import。 |
| 行为和 task contract 不变 | 已完成：现有 execution、handler、bootstrap 与全量后端测试通过。 |
| C1 execution adapter inversion | 已完成：Docker/Shell runner 与 local workspace 由 infrastructure 实现并经 bootstrap 注入；CI port 使用 application mount/workspace DTO，CD status/log 通过 command-query port 获取输出。 |
| C2 repository absence semantics | 已完成：SQL adapter 将 no-row 映射至 `repository.ErrNotFound`，application 不再 import `database/sql`。 |
| C3 typed task dispatch | 已完成：CI/CD application port 使用 typed dispatch input，queue dispatch adapter 保持既有 task type 与 JSON payload。 |

## Acceptance checklist

- [x] CI `ExecutePipelineRun` 及 stage execution orchestration 位于 application CI usecase。
- [x] CD deploy/restart/stop orchestration 和 deployment-time rendering 位于 application CD usecase。
- [x] queue handler 只进行 payload decoding、required-field validation 和 application execution dispatch。
- [x] CI queue handler 不再 import workflow-local DTO；task payload key 与错误文案未变化。
- [x] application execution constructor 使用最小 execution log port；HTTP 查询侧仍使用 read-only port。
- [x] bootstrap 使用 application execution services 和 infrastructure runner/workspace adapters。
- [x] application CI/CD 不依赖 local workspace implementation、`os/exec`、`database/sql` 或 `queue/task` implementation；application ports 不暴露 concrete workspace type、SQL sentinel 或 generic queue payload。
- [x] CI stage success/failure/cancel、credential non-leak、variables/mounts、CD operation states、Compose arguments、Liquid physical root、route labels、init script 和 status/log query 现有测试均随最终职责路径保留。
- [x] queue dispatch adapter 保持四个 task type、原有 JSON payload key/value、worker handler validation 与 lease/retry/concurrency 生命周期。
- [x] 完整 `internal/workflow` tree 与旧 application runner implementation 已删除，无 wrapper、alias、forwarding package 或新旧路径并存。
- [x] 未创建空 workflow directory。
- [x] 未修改 API route、request/response contract、task payload、migration、queue lifecycle 或前端行为。

## Test results

| Command | Result |
| --- | --- |
| `go test ./internal/repository/... ./internal/application/ci/usecase ./internal/application/cd/usecase ./internal/infrastructure/... ./internal/queue/... ./internal/bootstrap ./internal/api/http/handler/authz` | 通过。 |
| `go fmt ./cmd/... ./internal/...` | 通过。 |
| `./bin/golangci-lint fmt ./cmd/... ./internal/...` | 通过。 |
| `./bin/golangci-lint run ./cmd/... ./internal/...` | 通过，`0 issues.` |
| `go vet ./cmd/... ./internal/...` | 通过。 |
| `go test ./cmd/... ./internal/...` | 通过。 |
| `git diff --check` | 通过。 |
| active Go source workflow/activity import audit | 通过，未发现匹配。 |

未启动、停止或重启开发服务器。

## Completion

- [x] Issue 8：workflow execution 已收敛至 application，workflow tree 已删除。
- [x] C1：concrete Docker/Shell runner 与 local workspace adapter 已收敛至 infrastructure 并由 bootstrap 注入；CD status/log 已通过 application command-query port 调用。
- [x] C2：`sql.ErrNoRows` 已在 SQL repository adapter 边界归一为 `repository.ErrNotFound`；application usecase 不再依赖 database-driver sentinel。
- [x] C3：CI/CD application command 已通过 typed dispatch port 请求异步业务动作；queue adapter 独占 queue task contract。

## Conclusion

**Issue 1 至 Issue 8 及后续 C1-C3 均已完成并验证通过。** CI/CD execution 由 application usecase 统一编排，queue worker handler 保持入站 task adapter；workspace、process execution、repository absence 与 task dispatch 均在正确的 infrastructure/repository/queue boundary 实现。