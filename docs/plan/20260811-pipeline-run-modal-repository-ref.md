# 流水线运行时变量解析与运行弹窗实施计划
最后修改时间: 2026-08-12 15:00:03

Review status: Accepted

Mode: strict

## Basis

- Requirement: [流水线运行时变量解析与运行弹窗](../requirement/20260811-pipeline-run-modal-repository-ref.md)
- Spec: [流水线运行时变量解析与运行弹窗规格](../spec/20260811-pipeline-run-modal-repository-ref.md)

本计划以一次前后端同版本发布为前提，彻底移除 `trigger_ref`，不提供兼容字段、别名或运行时回退路径。`PipelineSnapshot.variables_snapshot` 继续保存完整可快照化声明历史，但不作为运行时变量解析来源。

## Implementation Steps

1. **调整 Schema、迁移与 SQLC 输入**
   - 新建 `000031` SQLite/MySQL 成对结构迁移及对应 down migration：将
     `pipeline_run.trigger_ref` 重命名为 `repository_ref`。不修改 `000020`、`000022`
     或其他已执行迁移，也不修改 `pipeline_snapshot.variables_snapshot`。
   - 更新 `sql/schema/schema.sql` 至 version 31 终态和
     `sql/query/pipeline_run/pipeline_run.sql`。`sql/query/pipeline/pipeline.sql` 保持
     Snapshot 变量声明快照的读取和写入。
   - 修改 SQLC Repository 映射和 Repository 测试建表/插入样例；执行 `task sqlc`，
     只接受与 Query/Schema 改动对应的生成物变化。
   - 更新 `internal/bootstrap/app_test.go`，并使 SQLite 迁移测试覆盖 migration 31；
     MySQL e2e 的版本断言更新为 31，但仍仅在配置 `BACKEND_GO_E2E_CONFIG` 时运行。

2. **收敛 Model、Snapshot 与 Proto 契约**
   - 将 `model.PipelineRun.TriggerRef` 改为 `RepositoryRef`；保留
     `model.PipelineSnapshot.VariablesSnapshot` 及其 DTO、Proto、Repository 和 HTTP
     mapper，以提供创建 Snapshot 时完整的 Repository/Pipeline/Stage 声明历史及 runtime/system 元数据。
   - 在 `proto/orbit/v1/pipeline_run/pipeline_run.proto` 中移除请求/预览响应的
     `trigger_ref`，将 `variables` 和 `variable_declarations` 设为 tag `1`，并将 Run 响应 tag `10`
     改名为 `repository_ref`。
   - 更新 Pipeline Run DTO 与 HTTP handler/mapper，仅在 `Variables` map 中传递运行表单；运行列表、运行详情页改读 `repository_ref`。同步需要改名的 i18n key/文案，确保不再以“触发分支”描述该字段。
   - 执行 `task proto`，以生成的 Go/TypeScript 类型驱动剩余编译错误的收敛。

3. **重构变量声明与值解析规则**
   - 在 `internal/application/pipeline/rule/pipelinevariable/runtime.go` 用明确的纯函数分离：
     Snapshot 声明快照生成、运行时阶段变量提取、Repository/Pipeline/Stage 声明去重合并、
     `repository_ref` runtime 声明、system 声明、完整非 system 表单校验、最终 Run 快照序列化与反序列化。
   - 运行时输出只使用 `repository`、`pipeline`、`pipeline_stage`、`runtime`、`system`；
     `repository_custom`/`pipeline_custom` 仅作为配置存储的输入标签。
   - 修复 `ResolvePipelineVariables`：先保留全部 Pipeline 声明，再用阶段提取补充未声明变量和 Liquid `default`；不得因为阶段确实提取到变量而丢弃未引用的 Pipeline 变量。
   - 按“表单 -> Repository value/default -> Pipeline value/default -> Stage default”解析可配置值；
     `repository_ref` Preview 缺省为 `default_branch`、Trigger 必填且原样保留。系统键、未知键、空必填值返回 validation error。
   - 扩展 `runtime_test.go`，覆盖阶段 `default`、无阶段变量、未引用 Pipeline 变量、三层同名覆盖、完整声明唯一性、`repository_ref` 可覆盖、system/未知键拒绝，以及完整 Run 快照包含 system 值。

4. **切换运行变量解析与 Run 生命周期**
   - `snapshot_create.go` 扩展为从当时的 Repository、Application Pipeline 与阶段定义生成并序列化完整可快照化 `VariablesSnapshot`，同时包含 runtime/system 声明元数据；用户表单值与每次生成的 system 值不得写入可复用 Snapshot。
   - 将运行路径从 `CompleteSnapshotVariableDeclarations`（或任何读取
     `snapshot.VariablesSnapshot` 的 helper）切换为实时解析 helper；Snapshot 变量声明在运行路径中不得参与默认值或取值决策。
   - Preview 使用当前 Application Pipeline 阶段定义及当前 Pipeline/Repository 配置解析，且不创建 Snapshot。
   - Trigger 与 Retry 在取得/复用 Snapshot 后，从其 `StagesSnapshot` 提取阶段声明，再读当前 Pipeline/Repository 配置和表单，原子持久化完整 `PipelineRun.variables_snapshot` 与一致的 `repository_ref`。
   - Retry 从原 Run 快照提取全部非 `system` value（含 `repository_ref`）作为新表单；不复用原变量快照，也不回写原 Run。
   - 删除 `pipelineRunRef` 及本地目录的 `ResolveRevision` 调用。保留 `Executor` 的
     `DockerHostPath` 源码挂载能力，且不引入 `repository_sha`。
   - 为 `pipeline_run/usecase` 新增针对 Preview、Trigger、Retry 和执行变量加载的测试桩，验证当前配置重新解析、Snapshot 声明历史保持创建时值、Run 快照不可变、ref 原样保留、快照 ref 一致性和执行不读取可变变量配置。

5. **让执行器只使用 Run 变量快照**
   - 在 `execution.go` 反序列化 `PipelineRun.variables_snapshot` 为环境变量 map，验证
     `repository_ref` 与 Run 字段一致，然后渲染 `StagesSnapshot`。
   - 移除执行时对 `BuildRuntimeVariables`、Repository/Pipeline 声明和 Snapshot 变量列的运行时调用；
     仅保留 Repository 的凭据、URL 与本地目录挂载等执行连接信息读取。
   - 对损坏的 Run 变量快照或 ref 不一致，将 Run 标记为 faulted 并保留可诊断错误；不尝试从当前配置补齐。

6. **重构运行弹窗及 Run 历史展示**
   - 在 `web/src/views/pipeline/PipelineDetail.vue` 以 Preview 的变量声明作为唯一的
     `runForm.variables` 数据源，删除独立 `trigger_ref` 状态、输入、watcher、Preview 参数和 diff-only 提交逻辑。
   - 初始化将所有非 `system` 有效值写入表单；每次编辑提交完整非 system form 进行 debounce Preview；
     system 字段只读展示且不进入请求。过期 Preview 响应不得覆盖最新表单。
   - 使用字段错误 map 实现 `repository_ref` 与其他缺失值的错误边框、`aria-invalid`、紧邻错误文本；编辑字段仅清除自己的错误。保留网络/解析失败作为表单级反馈，关闭弹窗时清理错误与计时器。
   - 更新 `PipelineRunPage.vue`、`PipelineRunDetail.vue`、相关 i18n 和任何其他类型消费者，统一显示 `repository_ref`。
   - 如现有测试工具适配 Vue SFC，补充弹窗表单函数或组件测试，重点覆盖完整请求、system 排除、单字段错误清理与过期 Preview 防护；否则以 typecheck、lint 和手动场景清单验收。

7. **收尾检查与文档对齐**
   - 全仓搜索确认业务代码、Proto、SQL、前端和活文档不再保留 `trigger_ref`。保留的
     `PipelineSnapshot.variables_snapshot` 只能出现在 Snapshot 创建、持久化、历史 API 和
     追溯展示路径，不能出现在 Preview、Trigger、Retry 或 Execute 的运行变量解析路径。
   - 最后更新 `docs/guides/ci-pipeline-design.md` 与
     `docs/guides/ci-pipeline-vars-design.md` 的时间戳和实现细节，保持与实际代码一致。
   - 不改动当前工作树中与本功能无关的 SQL Query/SQLC/Vue 修改；生成命令若扩大已有用户改动，按实际 diff 审核并保留其内容。

## Files To Change

| 区域 | 主要路径 |
|---|---|
| 迁移与 Schema | `sql/migration/{sqlite,mysql}/000031_*`、`sql/schema/schema.sql` |
| SQL/SQLC | `sql/query/pipeline_run/pipeline_run.sql`、`internal/gen/sqlc/pipeline_run/*`、对应 Repository 与测试 |
| Proto | `proto/orbit/v1/pipeline_run/pipeline_run.proto`、`internal/gen/proto/**`、`web/src/gen/proto/**` |
| Model/Pipeline | `internal/model/pipeline_run.go`、`internal/application/pipeline/{usecase,rule}/**` |
| Pipeline Run | `internal/application/pipeline_run/{dto,usecase}/**`、`internal/api/http/handler/pipeline_run/**`、`internal/repository/impl/sqlc/pipeline_run/**` |
| 前端 | `web/src/views/pipeline/PipelineDetail.vue`、`web/src/views/pipeline_run/{PipelineRunPage,PipelineRunDetail}.vue`、`web/src/i18n/locales/{zh-CN,en-US}.ts` |
| 文档和迁移测试 | `docs/guides/ci-pipeline-*.md`、`internal/{bootstrap,infrastructure/database,test/e2e}/*_test.go` |

## Verification Plan

1. 生成与静态一致性：执行 `task proto`、`task sqlc`，检查生成物仅反映本次契约/Schema 改动。
2. Go 定向测试：先运行 `go test ./internal/application/pipeline/... ./internal/application/pipeline_run/... ./internal/repository/impl/sqlc/pipeline_run/... ./internal/infrastructure/database/... ./internal/bootstrap/...`，再运行项目约定的 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。
3. 前端：执行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`；新增前端测试时执行 `yarn --cwd web test`。
4. 迁移：SQLite 测试从空库升级到 version 31，并验证 `pipeline_snapshot.variables_snapshot` 保持可读、`pipeline_run` 使用 `repository_ref`。配置外部 MySQL e2e 环境时验证同一迁移终态。
5. 行为回归：按 Spec 测试矩阵验证声明合并、`IMAGE_TAG` 默认值、完整表单、系统/未知键拒绝、Snapshot 复用的重新解析、Retry、Run 快照执行和本地目录 ref 不解析 SHA。
6. UI 手动验收：打开运行弹窗、编辑 `repository_ref` 和普通变量、观察 debounce Preview、提交/空提交、关闭重开及 Run 列表/详情显示；不启动或管理开发服务器。

## Assumptions And Risks

- 迁移编号采用当前最高版本后的 `000031`；SQLite 使用该版本支持的列删除/重命名语法，MySQL 使用等价 DDL。若项目的 migrate driver 对 SQLite `DROP COLUMN` 受限，则在实施时用可验证的表重建迁移，不改已执行迁移。
- `PipelineSnapshot` 保存结构版本和创建时可获得的完整声明版本，Run 保存变量执行版本；Snapshot API 继续展示声明历史，但其数据不是运行时变量权威来源。可复用 Snapshot 不保存用户表单值或每次生成的 system 值。
- Preview 读取当前阶段定义而 Trigger 读取 Snapshot 阶段定义。在 Pipeline 在 Preview 和 Trigger 之间变更时，Trigger 的返回/Run 快照是权威结果；弹窗应依赖 Trigger 的服务端校验而非假定 Preview 永远一致。
- 当前工作树已有 SQL、生成代码及 Vue 的用户改动。实施时不能回退或覆盖这些改动；每次生成后必须识别本任务实际影响。

## Rollback

本任务不支持通过兼容字段在线回滚。代码部署失败时应回退到与数据库 migration 31 相匹配的修复版本，而不是恢复 `trigger_ref`。迁移前的数据库备份是生产回退前提。

## User Review Notes

- 用户已确认移除 `trigger_ref`、采用完整非 `system` 提交表单，以及 `repository_ref` 原样保留。
- 用户已确认不解析/新增 `repository_sha`，Git commit 继续通过 `command/git_object_id` 制品追溯。
- 用户已确认 Snapshot 保留完整声明快照用于追溯，但复用 Snapshot 不复用变量；每次 Run 重新解析，Retry 从原 Run 非系统表单重新创建 Run。
