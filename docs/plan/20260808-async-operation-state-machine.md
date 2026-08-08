# 异步操作状态机收敛计划
最后修改时间: 2026-08-08 13:43:45

Review status: Accepted

## Basis

- Requirement：[异步操作状态机收敛需求](../requirement/20260808-async-operation-state-machine.md)（Accepted）
- Spec：[异步操作状态机收敛规格](../spec/20260808-async-operation-state-machine.md)（Accepted）

交付分为两个顺序阶段：先完成并验证主业务状态机；随后交付并单独执行历史数据脚本。历史数据不进入业务用例、启动流程或兼容逻辑。

## Implementation Steps

1. 建立新持久化边界与生成代码。
   - 新增 `sql/migration/sqlite/000031_async_operation_state_machine.{up,down}.sql` 和 MySQL 对应文件；不修改现有迁移。编号顺延以避免与并行的 `000030_pipeline_stage_template_library` 冲突。
   - `000031` 重建 SQLite `background_task` 时原样保留 `max_attempts`，MySQL 不删除该列；现有 task 继续保留自身预算。新 task 在入队时写入 `worker.max_attempts`，为 `pipeline_stage_run(pipeline_run_id, stage_id)`、同一 Service 的活跃 Deployment、同一 Repository 的活跃 Pipeline Run 增加数据库级唯一保障。活跃集合固定为 `waiting_to_run`、`running`。
   - 更新 `sql/schema/schema.sql`、`sql/query/task/task.sql`、`sql/query/deployment/deployment.sql`、`sql/query/pipeline_run/pipeline_run.sql`、`sql/query/gateway/gateway.sql`，将所有业务状态迁移改为带源状态条件且返回受影响行数。
   - 运行 `task sqlc`，提交相关 `internal/gen/sqlc/**`；Repository 层将零行更新翻译为可区分的状态冲突或 no-op。

2. 恢复 background task 的持久化重试预算。
   - 修改 `internal/queue/task/{model,service}.go`、`internal/repository/impl/sqlc/task/repository.go` 与 worker 测试：创建时将 `worker.max_attempts` 写入 task；失败转换只按任务记录的 `attempts/max_attempts` 决定 `running -> pending|failed`。
   - 恢复 `internal/config/config.go`、`configs/config.yaml`、`.env.example`、`internal/bootstrap/http.go`、`internal/application/settings/usecase/service.go` 及测试中的 `worker.max_attempts`，开发值固定为 `1`。
   - 在 `proto/orbit/v1/task/task.proto`、HTTP task mapper/测试及 task DTO 展示持久化 `max_attempts`；创建请求不接收覆盖值。运行 `task proto` 重新生成 Go/TypeScript DTO。

3. 实现 Deployment 状态转换、取消和运行态分离。
   - 更新 `internal/application/deployment/{port,usecase}`、`internal/repository/{deployment,service,gateway}.go` 和 sqlc Repository。Deployment worker 在任何可能失败的准备工作前通过条件写入 `waiting_to_run -> running`；成功仍以 `docker compose up -d` 零退出码为准，失败或部署 timeout 写 `faulted`。
   - Cancel API 改为单次条件迁移 `waiting_to_run|running -> canceled`。worker 在执行前及执行期间按 `worker.poll_interval` 读取 Deployment 状态；读取到 `canceled` 后取消传给 Docker runner 的 Context，并且所有后续成功/失败写入只能命中 `running`，不可覆盖取消终态。
   - 去除 `ServiceStatusDeploying` 的常量和所有写入；Deploy/Restart/Stop 创建时不改 Service。worker 在执行完成或取消清理后依据 Docker runtime 观察写入 `running`、`stopped` 或 `faulted`。
   - 用活跃 Deployment 查询和数据库唯一约束取代 Service 状态并发守卫；同时改造 Gateway 的单活跃检查与配置解析，确保只有实际 `running` 的 Gateway 可供路由。

4. 实现 Pipeline Run、Stage Run 和 Retry 的状态转换。
   - 更新 `internal/application/pipeline_run/{port,usecase}`、`internal/repository/pipeline_run.go`、对应 sqlc Repository 和查询。在创建 Trigger/Retry Run 的短事务内按 Snapshot 预物化全部 Stage Run 为 `waiting_to_run`，并建立唯一约束保护。
   - 移除 `RetryPipelineRun` 对来源状态的校验；保留 `retry_of` 父关联。创建新 Run 时只应用 Repository 活跃并发守卫，不改写来源 Run。
   - Pipeline worker 先条件开始 Run；Stage executor 只在真正开始容器前条件迁移单个 Stage 为 `running`。删除 `cancelRemaining` 及其下游 `canceled` 写入。
   - Cancel Run 时在同一短事务中将 Run 和所有 `running` Stage Run 写为 `canceled`。执行器轮询 Run/Stage 状态并取消对应 ContainerRunner Context；未运行 Stage 保持 `waiting_to_run`。
   - 保持 `pipeline_run.execution_timeout`：超时将仍处于 `running` 的 Stage Run 与 Run 条件迁移为 `faulted`，`error_message` 包含 timeout 与时限。worker lease 不参与该业务判定。

5. 统一 HTTP、MCP、Proto、Web 与活文档。
   - 保持既有单数 Cancel/Retry 路由，取消成功统一返回 `200` 和已终结资源；更新 handler/usecase/repository 测试覆盖排队、运行、已终结及竞态结果。
   - 更新 Deployment、Pipeline Run 详情与列表的前端操作守卫和文案：活跃记录阻止冲突操作，`canceled` 后禁用重复取消；Stage 未运行时展示 `waiting_to_run`。
   - 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md` 以移除 `Service.deploying` 和自动重试的过时说明；Healthcheck 只描述 Compose 与独立验证用途，不写成主部署成功条件。

6. 完成主业务测试与检查。
   - Repository SQL 测试覆盖每个允许和拒绝转换、条件更新零行、活跃唯一冲突和 task 冻结预算后的重投/失败终结。
   - Deployment 用例/worker 测试覆盖：Compose 零退出码成功、错误/timeout `faulted`、排队取消 no-op、运行取消不可被成功/失败覆盖、Service 运行态观察、Gateway 活跃冲突。
   - Pipeline 用例/执行器测试覆盖：所有 Stage 预物化、上游失败的下游保持 `waiting_to_run`、运行 Stage 的取消、timeout `faulted`、任意来源状态 Retry、来源 Run 不变及活跃 Run 冲突。
   - 运行 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`；若改动前端，运行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。

7. 在主业务通过后交付独立离线历史数据脚本。
   - 新增 `scripts/reconcile_legacy_service_deploying.py` 与说明文档。脚本不被 Go 服务调用，也不由启动流程触发。
   - 脚本接受显式数据库路径和人工审核过的状态映射文件；默认 dry-run 仅输出将变更的 `service_id`、旧状态和目标运行态。仅在显式 `--apply` 时更新仍为 `deploying` 的记录，拒绝其他源状态，输出可审计报告。
   - 交付脚本单元测试/fixture。执行前先备份目标数据库；脚本完成后查询确认没有遗留 `Service.deploying`。该步骤只处理历史数据，不改变主业务状态机的测试或启动行为。

## Files to Change

- 数据库与生成：`sql/migration/{sqlite,mysql}/000031_async_operation_state_machine.*.sql`、`sql/schema/schema.sql`、`sql/query/{task,deployment,pipeline_run,gateway}/*.sql`、`internal/gen/sqlc/{task,deployment,pipeline_run,gateway}/**`。
- 队列与配置：`internal/queue/{task,worker,dispatch}/**`、`internal/repository/impl/sqlc/task/**`、`internal/config/**`、`internal/bootstrap/http.go`、`internal/application/settings/**`、`configs/config.yaml`、`.env.example`。
- Deployment/Service/Gateway：`internal/application/{deployment,gateway}/**`、`internal/repository/{deployment,service,gateway}.go`、`internal/repository/impl/sqlc/{deployment,service,gateway}/**`、`internal/common/constant/status.go`。
- Pipeline：`internal/application/pipeline_run/**`、`internal/repository/pipeline_run.go`、`internal/repository/impl/sqlc/pipeline_run/**`。
- Transport/UI：`proto/orbit/v1/task/task.proto`、`internal/api/http/handler/{task,deployment,pipeline_run}/**`、`internal/gen/proto/**`、`web/src/gen/proto/**`、相关 Deployment/Pipeline Run 视图与 i18n。
- 文档和离线数据：`docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`、`scripts/reconcile_legacy_service_deploying.py`、其说明与测试。

## Verification Plan

- 从空 SQLite 数据库完整执行迁移链，检查新 schema、唯一约束，以及 task 具备 `max_attempts` 字段。
- 使用开发数据库的副本验证旧 `background_task` 数据在重建后保留，主业务不触碰旧 `Service.deploying`；在独立副本上验证离线脚本的 dry-run、`--apply` 和非 `deploying` 保护。
- 使用 fake Runner/ContainerRunner 验证取消、超时、Docker 返回成功/失败及状态条件竞争；不运行实际 Docker 生命周期命令。
- 执行第 6 步中的 Go、Proto、sqlc、前端检查；检查更新后的活文档不再将 Healthcheck 或 `Service.deploying` 误述为业务操作态。

## Blockers

- 主业务实现、迁移、生成代码和测试没有外部阻塞。
- 离线脚本的实际 `service_id -> 运行态` 映射必须由数据所有者审核后提供；该映射不阻塞主业务实现和验证，但阻塞历史数据脚本的 `--apply`。

## Assumptions and Risks

- `canceled` 是立即可见的终态，同时也是跨进程停止信号；外部 Docker/Container 操作可能在短暂窗口内继续，无法保证回滚。
- `000031` 是本任务唯一的迁移版本，重建 task 表时必须保留每条记录既有的 `max_attempts`；预算不能通过配置回溯改变。
- 活跃唯一约束需要 SQLite 与 MySQL 分别实现等价语义，并在实际迁移前用两种引擎验证。
- 离线脚本只执行人工审核映射，不猜测 Docker 运行态；映射错误由脚本输入者负责，dry-run 与数据库备份是必要保护。

## Rollback

- 主业务代码的状态问题通过前向修复处理；不重新引入 `Service.deploying`，并由 `000031` 保留 task 持久化预算。
- 数据库迁移及离线脚本执行前必须备份。若离线脚本映射错误，从备份恢复数据后修正映射并重跑。
- 不自动执行任何 Docker 反向操作或删除数据卷。

## User Review Notes

- 2026-08-08：用户要求先完成主业务，后以独立离线脚本处理历史数据；脚本不写入业务路径。
- 2026-08-08：用户要求开始 Implementation，Plan 视为接受。
- 2026-08-08：用户更正 `max_attempts` 应由配置在任务创建时冻结到数据库，worker 只能读取任务记录。
- 2026-08-08：用户要求将预算变更收回 `000031`，本开发任务不额外占用迁移版本。
