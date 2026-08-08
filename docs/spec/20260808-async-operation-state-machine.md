# 异步操作状态机收敛规格
最后修改时间: 2026-08-08 13:43:45

Review status: Accepted

## Requirement basis

本规格落实已接受的 [异步操作状态机收敛需求](../requirement/20260808-async-operation-state-machine.md)。范围包含 Deployment、Pipeline Run、Pipeline Stage Run、`background_task` 的调度边界，以及 Service 运行态与 Deployment 操作态的分离。

不为 Deployment 设计 Retry 或 `retry_of`。现有 `PipelineRun.retry_of` 保留，且 Retry 始终创建新的 Pipeline Run。

## Overview

用户可见的异步操作只使用以下五个主状态：

```text
waiting_to_run -> running -> ran_to_completion
                         -> faulted
                         -> canceled
waiting_to_run -> canceled
```

`ran_to_completion`、`faulted`、`canceled` 都是不可逆终态。运行中取消直接写入 `canceled`；该终态同时是 worker 停止外部执行的持久化信号。它表示取消已被接受，不承诺 Docker 或 Stage 容器已经退出或回滚。

所有状态转换都由条件更新完成。仅按 ID 更新状态的 SQL 不再合法；调用方必须处理“预期源状态不匹配”的结果，而不能覆盖较新的终态。

## State Ownership

| 对象 | 记录创建者 | `waiting_to_run -> running` | 终态写入者 | 取消控制 |
|---|---|---|---|---|
| Deployment | HTTP/MCP 命令用例 | Deployment worker | Deployment worker；Cancel API 可从非终态直接写 `canceled` | Cancel API 写状态；worker 停止执行 |
| Pipeline Run | Trigger/Retry 用例 | Pipeline worker | Pipeline worker；Cancel API 可从非终态直接写 `canceled` | Cancel API 写状态；worker 停止执行 |
| Pipeline Stage Run | 创建 Pipeline Run 时按 Snapshot 一次性物化 | Pipeline executor | Pipeline executor | Pipeline Run 的取消控制，不提供 Stage API |
| `background_task` | dispatcher 或内部 task API | worker lease/claim | worker | 不作为用户可见的取消对象 |

Stage Run 在 Pipeline Run 创建的同一短事务内，为 Snapshot 中每个 Stage 建立一条 `waiting_to_run` 记录。执行器只将准备实际执行的 Stage 条件迁移至 `running`。上游失败或 Pipeline Run 取消都不得修改未开始 Stage 的 `waiting_to_run` 状态。

## Allowed Transitions

| 源状态 | 目标状态 | 发起者 | 前提 |
|---|---|---|---|
| 新记录 | `waiting_to_run` | 创建用例 | 记录、Snapshot、调度任务已持久化 |
| `waiting_to_run` | `running` | worker/executor | 条件更新成功；尚未被排队取消 |
| `waiting_to_run` | `canceled` | Cancel API | 条件更新成功；不启动外部副作用 |
| `running` | `ran_to_completion` | worker/executor | 外部工作成功，且其条件更新先于 Cancel API 提交 |
| `running` | `faulted` | worker/executor | 外部工作失败或执行时限超时，且其条件更新先于 Cancel API 提交 |
| `running` | `canceled` | Cancel API | 条件更新成功；worker 观察到终态后停止外部命令/容器 |

worker 领取后必须首先条件迁移业务记录至 `running`，再做会失败的解析、渲染、工作目录或外部执行。这样所有被领取后的失败都符合 `running -> faulted`，不会产生 `waiting_to_run -> faulted` 的隐式回边。

未能取得 `waiting_to_run -> running` 的 worker（记录已经被取消、已运行或已终结）必须无副作用返回成功，让其 `background_task` 正常完成。

## Deployment Success Criterion

当前 Deploy/Restart worker 只执行 `docker compose up -d --remove-orphans --pull <policy>`。该命令返回 `0` 后，worker 立即将 Service 写为 `running`、Deployment 写为 `ran_to_completion`；命令没有 `--wait`，也没有在该路径中读取容器 `Health.Status`。

这是已确认的成功规则。Version Component 的 Healthcheck 会正确渲染进 Compose，`depends_on.condition=service_healthy` 也会影响 Compose 内的依赖启动顺序，但两者不构成 Deployment 主状态的成功条件。

另有独立的 MCP `VerifyDeployment`：它会读取 Compose runtime、`docker inspect` 与重启计数，并以默认 60 秒窗口观察稳定性。该验证不在 worker 成功路径中自动执行，且当前只把 `unhealthy`、非 `running` 与重启次数增长判为失败；`starting` 或未配置 Healthcheck 不会阻止其稳定结论。

## Cancellation Protocol

`POST /api/deployment/:deployment_id/cancel` 与 `POST /api/pipeline-run/:run_id/cancel` 保留原路径。它们执行一次条件更新：`WHERE status IN ('waiting_to_run', 'running')`，立即将记录写为 `canceled`、写入完成时间和固定取消说明，并返回 `200`。未命中时读取当前状态并返回验证错误，不能重复取消或覆盖终态。

`canceled` 本身是跨进程的取消信号，不增加 `cancel_requested_at`、`cancel_message` 或第六个状态。worker 在开始外部工作前和工作期间，按既有 `worker.poll_interval` 轮询关联业务记录的状态；读到 `canceled` 后取消子 Context。`CommandRunner` 与 `ContainerRunner` 必须响应 Context，终止其子进程/容器并等待退出。

所有成功、失败和 Stage 终态写入均使用 `WHERE status = 'running'`。因此 Cancel API 已写入 `canceled` 后，worker 只能清理文件、日志和运行态观察，不能把业务操作覆盖回 `ran_to_completion` 或 `faulted`。

取消 Pipeline Run 时，Cancel API 同时将当前 `running` 的 Stage Run 条件迁移为 `canceled`；worker 据此停止对应容器。尚未进入 `running` 的 Stage 保持 `waiting_to_run`，执行器删除现有 `cancelRemaining` 写入 `canceled` 的行为。

Deployment 在取消后仍须观察 Docker 的实际运行态，并据此更新 Service 的 `running`、`stopped` 或 `faulted`。`canceled` 描述取消已接受，不承诺 Docker 没有产生部分副作用；特别是 `Stop(remove_volumes=true)` 取消后不执行自动回滚。取消后发生的停止异常写入操作日志，不改变已终结的 Deployment 状态。

### Timeout

超时不是用户取消。现有 `pipeline_run.execution_timeout` 触发 `context.DeadlineExceeded` 时，尚处于 `running` 的 Pipeline Stage Run 与 Pipeline Run 都迁移为 `faulted`，其 `error_message` 明确包含 timeout 与配置时限。关联 background task 的失败结果仍只按其冻结预算决定；开发默认预算为 `1`，因此首次失败即进入 `failed`。

Deployment 当前没有独立的执行超时配置；因此不会凭 worker lease 过期把 Deployment 判为超时。若后续为 Deployment 引入执行时限，`context.DeadlineExceeded` 必须按同一规则写为 `faulted` 并记录 timeout 原因。worker lease 只用于任务领取恢复，不是业务执行 deadline。

Cancel 与 timeout 并发时，以最先成功的条件终态更新为准：Cancel API 先写入则保持 `canceled`；timeout 的条件更新先写入则保持 `faulted`。

## Data and Repository Design

### New migration

新增 SQLite 与 MySQL 迁移，禁止修改既有迁移。迁移内容包括：

- 为 `pipeline_stage_run` 建立 `(pipeline_run_id, stage_id)` 唯一约束，保证预物化和执行过程不会重复创建 Stage Run；迁移前先校验历史重复数据，发现重复时停止并报告，不静默删除审计记录；
- 为 Service 的活跃 Deployment 及 Repository 的活跃 Pipeline Run 添加数据库级唯一保障。SQLite 使用部分唯一索引；MySQL 使用等价的生成列唯一键。活跃集合固定为 `waiting_to_run`、`running`。
- 保留 `background_task.max_attempts`。`000031` 重建 SQLite task 表时必须原样保留该列及其数据，MySQL 不得删除该列；新任务在入队时将 `worker.max_attempts` 冻结写入该列。`attempts` 保留为领取审计计数。

`retry_of` 不变，不向 Deployment 表增加对应字段。

### Conditional repository methods

Repository 不再提供无条件的 `Mark*Running`、`Complete*`、`Cancel*` 状态覆写。替换为意图明确、返回受影响行数的方法：

- `BeginDeployment` / `BeginPipelineRun` / `BeginPipelineStageRun`：`WHERE status = 'waiting_to_run'`；
- `CancelDeployment` / `CancelPipelineRun`：`WHERE status IN ('waiting_to_run', 'running')`；Pipeline Run 取消同时条件更新所有 `running` Stage Run；
- `Complete*`：`WHERE status = 'running'`；worker 只能写 `ran_to_completion` 或 `faulted`，`canceled` 只由 Cancel API 写入；
- `ListActiveDeploymentsByService` 与 `RepositoryHasActivePipelineRun`：活跃状态同时包含 `waiting_to_run` 和 `running`。

创建 Deployment 或 Pipeline Run 时，活跃唯一约束失败必须翻译为业务冲突错误。所有创建、Stage Run 预物化和任务入队使用短数据库事务；Docker、容器、文件和网络工作始终在事务外执行。

## Queue Retry Boundary

`worker.max_attempts` 只在 task 创建时读取，并持久化为 `background_task.max_attempts`。`FailTask` 仅根据当前记录的 `attempts >= max_attempts` 条件转换：未达到预算时 `running -> pending`，达到预算时 `running -> failed`。worker 不读取配置决定既有任务的重试，因此配置变更只影响后续新任务。

开发配置的默认预算固定为 `1`，所以常规业务任务第一次失败仍直接终结。`WorkerConfig.MaxAttempts`、`worker.max_attempts`、Settings 元数据、`.env.example` 和 task 模型保留；任务 HTTP 响应展示冻结后的 `max_attempts`，创建请求不允许覆盖该配置。Pipeline Run Retry 保持为独立的显式用户命令：它新建 Run、Stage Run 和 task，不会重用失败任务。

## Service and Gateway Rules

Service 状态只保留运行态 `running`、`stopped`、`faulted`；删除 `deploying` 常量和所有读写路径。创建 Deploy、Restart、Stop 时不再预先修改 Service。worker 仅在外部执行结果或取消后完成运行态观察时更新它。

对同一 Service 的 Deploy、Restart、Stop 以活跃 Deployment 为并发守卫，不再以 `Service.deploying` 判断。Gateway 的“单活跃”约束改为：已有运行中的 Gateway Service，或另一 Gateway Service 存在活跃 Deployment 时，拒绝冲突操作。解析实际可用 Gateway 配置仍只依赖 `Service.status=running`，不会把排队/运行中的部署误认为已可路由。

现有历史 `Service.status=deploying` 的清点和修复在主业务完成后由独立离线脚本执行，不进入业务用例、启动流程或兼容逻辑。脚本根据人工确认的状态映射处理历史记录；业务代码不承担历史数据迁移或回写。

## Pipeline Run Retry

Retry 不检查来源 Pipeline Run 的状态：`waiting_to_run`、`running`、`ran_to_completion`、`faulted`、`canceled` 都可作为来源。它创建新的 Run、Stage Run 集合和 `background_task`，新 Run 的 `retry_of` 指向原 Run；来源 Run 不被修改。

Retry 复用原 Run 的触发 ref 和变量覆盖值，但按当前 Pipeline 创建路径取得 Snapshot；因此 `retry_of` 是谱系关联，不是不可变重放承诺。Repository 活跃约束同样适用于 Retry，避免同一 Repository 同时有多个 `waiting_to_run` 或 `running` Run。该冲突约束独立于来源 Run 状态：来源 Run 为活跃记录时，Retry 可因新 Run 无法取得活跃槽位而被拒绝。

## Interface Contract

- HTTP 路由不新增别名；Cancel 无论原状态是排队还是运行中，成功均返回 `200` 与 `canceled` 资源。
- Deployment、Pipeline Run 的 `canceled` 在取消操作接受后立即展示；Web 禁用重复取消和冲突操作。
- Pipeline Stage Run 继续只展示五态，不增加 `skipped`、取消控制字段或独立取消命令。
- task HTTP/Proto 响应展示持久化 `max_attempts`；失败结果遵循记录的冻结预算。

## Affected Components

- `internal/common/constant/status.go`：收敛 Service 与 work status 常量。
- `sql/migration/{sqlite,mysql}/`、`sql/query/{deployment,pipeline_run,task,gateway}/` 与 sqlc 生成结果：持久化字段、条件迁移和唯一约束。
- `internal/repository/impl/sqlc/{deployment,pipeline_run,task,gateway,service}/`：条件写入、冲突翻译、活跃查询。
- `internal/application/{deployment,pipeline_run,gateway}/usecase`：取消编排、执行 Context、Stage 预物化、Service runtime reconciliation。
- `internal/queue/{task,dispatch,worker}/`：在创建时冻结 task 重试预算，并实现按 `worker.poll_interval` 的取消状态观察及无副作用 no-op handler 行为。
- `proto/`、`internal/api/http/`、`web/`：展示 task 的冻结重试预算，统一 Cancel `200` 和直接取消状态展示。

## Technical Questions

暂无需要用户确认的未决事项。

## Risks and Alternatives

- 运行中取消直接终结业务记录，无法保证外部副作用已经停止，特别是 Docker `down --volumes`；worker 必须继续执行 Context 取消和 Service 实际运行态核对，但不回滚。
- 数据库唯一约束是防止多 worker/lease 过期并发的最终保障；只做用例层查询会保留竞态。
- 不采用将未执行下游 Stage 写为 `canceled` 或 `skipped`：前者混淆主动取消，后者扩大已确认的状态枚举。
- 不保留任何 task 自动重投：失败任务必须终结；用户可见重试仅通过 Pipeline Run Retry 创建新记录。

## User Review Notes

- 2026-08-08：Requirement 已接受；取消直接写 `canceled`，worker 复用 `worker.poll_interval` 观察并停止外部执行。
- 2026-08-08：Deployment Retry 不在范围内；`retry_of` 仅属于 Pipeline Run。
- 2026-08-08：开发默认业务 task 预算为 `1`。`max_attempts` 从配置在入队时冻结并持久化，worker 不在执行时读取配置。历史 `Service.deploying` 离线处理，不写入业务代码。
- 2026-08-08：确认当前 Deployment 成功只依据 `docker compose up -d` 退出码；Healthcheck 仅渲染和供独立验证读取，尚未纳入主状态成功判定。
- 2026-08-08：用户确认 Compose 零退出码即为 Deploy/Restart 成功；Pipeline Run Retry 不检查来源 Run 状态。
- 2026-08-08：用户要求主业务完成后再以独立离线脚本处理历史 `Service.deploying` 数据。
- 2026-08-08：用户要求开始 Plan，Spec 视为接受。
