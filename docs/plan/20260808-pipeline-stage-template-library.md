# 可复用流水线阶段实施计划
最后修改时间: 2026-08-08 13:39:00

Review status: Accepted

流程模式: 严格 / strict

## Inputs

- [Requirement](../requirement/20260808-pipeline-stage-template-library.md)：`Accepted`
- [Spec](../spec/20260808-pipeline-stage-template-library.md)：`Accepted`
- 当前实现基线：`Pipeline(kind=template|application)` 已存在；`pipeline_stage` 仍强制归属 Pipeline，并在同一行保存 DAG、排序和制品 JSON。

## Implementation steps

1. 建立新阶段持久化模型，并更新空库 schema 基线。
   - 新增下一编号的 SQLite/MySQL migration（当前迁移文件末尾为 `000029`，预期为 `000030_pipeline_stage_template_library`），绝不改写现有 migration。
   - 按 Spec 在两个方言中重建 `pipeline_stage`：添加 `project_id`、`kind`、模板 `version`、应用阶段的来源 ID/名称/版本/说明快照和可空的 `pipeline_id`；删除模板阶段不能拥有的 DAG/排序列。`kind=template` 不持久化制品，`kind=application` 必须有非空来源快照和制品 JSON 数组。
   - 创建 `pipeline_stage_reference`，仅承载 Template Pipeline 的私有节点名称/说明、来源快照、镜像/脚本、依赖和排序；其 `pipeline_id` 对 Pipeline 使用聚合内物理外键，来源模板阶段使用逻辑引用而非物理外键。
   - 增加 Project 内模板名称唯一、应用阶段 Pipeline 内名称唯一，以及用于项目级模板名称查找的 `(project_id, kind, name)` 索引；阶段数量有限，按 Pipeline 的排序与引用来源反查不建立普通索引。数据库只保留外键、非空和唯一性等结构完整性，不以 `CHECK` 表达模板/应用字段互斥性。
   - 更新 `sql/schema/schema.sql` 以镜像新 migration 链。调整 `internal/infrastructure/database/migration_test.go`，从空数据库验证两张阶段表、关键列、结构完整性、`foreign_key_check` 与新版迁移链。
   - 主业务完成后另行执行独立脚本处理旧 Pipeline 配置；该脚本不进入业务 usecase、handler、repository 或运行时兼容分支。migration 和业务代码不得猜测旧 Stage 来源；历史 Snapshot/Run/Artifact 表不删改。

2. 重构领域模型、Repository port 和 SQLC 查询，使模板阶段库、模板引用和应用阶段有清晰边界。
   - 在 `internal/model/pipeline.go` 增加 `PipelineStageKindTemplate`、`PipelineStageKindApplication`，将 `PipelineStage` 改为可表达项目级模板和应用私有阶段；新增 `PipelineStageReference` 与 Pipeline 节点/来源快照模型。必要时扩展 `StageDefinition`，只增加用于审计展示的来源字段，不改变执行器输入语义。
   - 更新 `internal/application/pipeline/dto/`：区分阶段库资源、Pipeline 阶段节点、引入请求、节点更新请求与模板更新预览/确认请求。移除“客户端提交完整 Stage 定义以创建 Pipeline 节点”的 DTO。
   - 拆分 `internal/repository/pipeline.go` 的端口：Project 级模板阶段 CRUD、Template Pipeline 引用读写、Application Pipeline 阶段读写，以及保留的 Snapshot 操作。接口名必须表达资源种类，不能继续让单个 `PipelineStages` 在两种节点模型间含混。
   - 改造 `sql/query/pipeline/pipeline.sql`、`internal/repository/impl/sqlc/pipeline/repository.go` 和对应生成物：支持模板阶段列表搜索/分页、同 Project 归属读取、引用列表、应用阶段列表、整条 Pipeline 配置事务和节点来源更新。Repository 只做持久化映射，不承载成员校验或模板选择规则。
   - 生成 SQLC 代码后清理所有旧字段和查询调用点；Snapshot 创建路径只加载 Application Pipeline 的阶段，变量解析分别接收应用阶段或 Template 引用可解析的脚本视图。

3. 实现模板阶段库及 Pipeline 节点的应用服务，收敛校验和原子写入。
   - 在 `internal/application/pipeline/usecase/` 新增模板阶段的列表、详情、创建、更新和删除。所有入口先校验 Project 成员资格；创建/更新仅接受名称、镜像、脚本、说明，名称在同 Project 唯一；实际变更时模板版本递增。
   - 在模型、DTO 绑定和 usecase 三层拒绝模板阶段的制品、依赖、排序、Pipeline、Application、Component 和 Version 策略字段；usecase 同时集中校验应用阶段的来源快照、制品/依赖 JSON 数组、排序和 `kind`，以替代数据库字段形态 `CHECK`。删除模板阶段不得级联既有引用、应用阶段或运行时历史。
   - 用“引入阶段”替换现有 `CreatePipelineStage`：服务端按目标 Pipeline kind 读取同 Project 的模板阶段。Template Pipeline 创建 `PipelineStageReference`；Application Pipeline 创建带完整来源快照的 `PipelineStage(kind=application)`，并初始化空制品数组。客户端不能覆盖来源定义。
   - 改写节点编辑/删除逻辑：Template 引用只可编辑私有名称、说明、依赖和排序；Application 节点可再编辑制品、镜像和脚本。删除仍被依赖的节点返回 validation error；两种节点的 DAG 均只可引用同 Pipeline 的本地 ID。
   - 保留现有 Application Pipeline 的制品 collector、组件唯一性、`git_object_id` 上游和来源 Version 策略校验。它们仅针对应用阶段及 Application Pipeline；模板引用与模板阶段均不进入这些校验输入。
   - 所有引入、节点编辑/删除、模板版本应用与 Application Pipeline 物化在一个 repository transaction 中完成完整 DAG/组件校验与所属 Pipeline 版本递增；无实际变化时不递增。

4. 重写 Template Pipeline 实例化、Snapshot 和模板更新路径。
   - 修改 `InstantiatePipeline`：加载 Template Pipeline 的 `PipelineStageReference`，只使用引用中冻结的来源 ID/名称/版本/说明、镜像和脚本创建新的应用阶段，不重新读取阶段库；保留引用的私有名称、私有说明、依赖和排序。
   - 以引用 ID 到应用阶段 ID 的完整映射重写 `depends_on`；初始化每条应用阶段为 `artifacts=[]`。任何引用来源快照缺失、DAG 无效或存储失败都回滚整次实例化。模板删除后，既有 Application Pipeline 和含该引用的 Template Pipeline 仍可继续使用快照实例化。
   - 更新 `snapshot_create.go` 与相关 Snapshot 读取：只读取应用阶段，拒绝无来源的应用阶段，并把新增来源审计字段写入 Snapshot（若步骤 2 扩展了 `StageDefinition`）。PipelineSnapshot、PipelineRun、Artifact 与 Version fork 的既有执行链不得回读可变模板阶段。
   - 实现模板更新预览和应用：预览返回来源模板的当前版本以及名称、镜像、脚本、模板说明差异；应用请求带已应用版本和目标版本作乐观并发检查。成功时更新节点镜像、脚本及来源快照，保留私有名称、私有说明、DAG、排序、制品、组件映射和 Pipeline Version 策略。
   - 补充并更新 `internal/application/pipeline/usecase/*_test.go`：模板 CRUD/版本、跨项目拒绝、重复引入、Template/Application 节点字段白名单、DAG、实例化只使用引用快照与 ID 重映射、来源删除后仍可实例化、显式更新并发冲突、Snapshot 不回读模板、组件映射不回归。

5. 变更 Proto、HTTP handler、路由和生成物，完成破坏性契约切换。
   - 在 `proto/orbit/v1/pipeline/` 定义阶段库资源响应/请求、`PipelineStageNodeResp`、`PipelineStageImportReq`、节点更新、模板更新预览和确认消息；`PipelineResp` 以 `stage_nodes` 替换旧 `stages`，不保留字段别名。
   - 修改 `internal/api/http/handler/pipeline/` 的 request binding 和 mapper，按 Project/Pipeline/节点归属调用新服务，确保模板资源 API 不可注入应用节点字段。
   - 在 `internal/api/http/routes/pipeline.go` 注册单数资源路由：`/api/pipeline-stage` 的 CRUD，保留 `/api/pipeline/:pipeline_id/stage` 作为仅引入模板阶段的动作，添加模板更新预览与确认路径；移除旧“直接提交执行定义创建阶段”的契约。
   - 同步更新 `internal/bootstrap/http.go` 的 service/repository 组合与编译时接口实现断言。执行仓库 Taskfile 定义的 `task proto`，提交仅由新契约生成的 Go/TypeScript 文件。
   - 增加 handler/route 测试，覆盖模板阶段项目成员校验、跨项目模板 ID 拒绝、模板制品字段拒绝、导入/节点更新、预览/确认冲突、删除不影响历史以及旧 Stage 创建请求不再可用。

6. 实现阶段库页面，并改造 Pipeline 编辑器。
   - 新增 `web/src/views/pipeline/PipelineStagePage.vue` 与 `web/src/api/pipeline/pipeline_stage.ts`，管理当前 Project 的 `kind=template` 阶段。列表使用 `SearchControl`、`app-toolbar-simple`、`app-data-table`、`ListPagination`、`AppDialog` 和 `AppDialogActions`；表单只显示名称、镜像、脚本、说明，并提供字段级错误。
   - 在 `web/src/router/index.ts` 注册 `/pipeline-stage`，在 `web/src/navigation.ts` 的 CI 二级导航添加“阶段”，并补齐 `web/src/i18n/locales/{zh-CN,en-US}.ts`。不增加细粒度 RBAC，因为当前 CI 页面只做登录和项目成员校验。
   - 改造 `web/src/api/pipeline/pipeline.ts` 与 `web/src/views/pipeline/PipelineDetail.vue` 使用新的 `stage_nodes` 和引入 API。将“添加阶段”替换为“引入阶段”，通过 `ComboboxSelect` 选择本 Project 模板阶段；引入弹窗/编辑器只提交私有字段，不复制模板定义。
   - 对 Template Pipeline 节点隐藏制品、组件映射和 Version 策略控件；对 Application Pipeline 节点保留既有制品编辑及字段清理逻辑。同步调整 `StageDAGView` 和所有读取 `PipelineResp.stages` 的页面/测试。
   - 在阶段编辑器右上角的按钮组增加共享 warning button variant，并按 `latest_template_stage_version` 显示 `更新至模板 vN`。使用 `AppDialog` 展示旧/新来源名称、镜像、脚本和模板说明差异，显式说明私有说明会保留；确认按钮为“更新并保存”。普通保存、取消和关闭不得触发更新。
   - 所有成功写操作以接口返回的完整 Pipeline 刷新本地详情与变量预览；时间继续使用 `web/src/utils/time.ts`。为阶段库、模板引入、应用制品可见性和模板更新差异补充/更新 Vitest 覆盖。

7. 回写活文档并执行全量验证。
   - 更新 `docs/product/cd-model.md` 和 `docs/guides/ci-pipeline-design.md`，说明 Stage kind、项目级模板库、`PipelineStageReference`、应用阶段来源快照、制品边界、模板删除后仍可从引用快照实例化及显式更新语义；更新 `docs/INDEX.md` 中受影响的 CI 导航说明（如需要）。
   - 生成与格式化顺序：先运行 `task sqlc` 与 `task proto`，再运行 `go fmt ./cmd/... ./internal/...`，然后运行定向 Go 测试和 `go vet ./cmd/... ./internal/...` / `go test ./cmd/... ./internal/...`。
   - 前端改动完成后必须运行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`；存在相关测试时再运行 `yarn --cwd web test`。不主动启动或重启开发服务器。
   - 实现完成后只报告实际 diff、检查结果、未运行项与风险，停留在 Implementation；用户明确要求后才创建 Verification 文档。

## Files and ownership

| 区域 | 主要路径 | 责任 |
|---|---|---|
| DDL 与 schema | `sql/migration/{sqlite,mysql}/000030_*`、`sql/schema/schema.sql` | 新阶段模型、结构完整性、索引及空库基线。 |
| SQLC 与 repository | `sql/query/pipeline/pipeline.sql`、`internal/gen/sqlc/pipeline/`、`internal/repository/{pipeline.go,impl/sqlc/pipeline/}` | 模板阶段、引用、应用阶段与事务持久化。 |
| 领域与服务 | `internal/model/pipeline.go`、`internal/application/pipeline/{dto,usecase,rule}/` | 类型、来源快照、DAG、实例化、模板更新和 Snapshot。 |
| HTTP 与契约 | `proto/orbit/v1/pipeline/`、`internal/gen/proto/`、`internal/api/http/{handler,routes}/pipeline/`、`internal/bootstrap/http.go` | 破坏性 API/Proto、映射、路由和组装。 |
| 前端 | `web/src/api/pipeline/`、`web/src/views/pipeline/`、`web/src/router/index.ts`、`web/src/navigation.ts`、`web/src/i18n/locales/`、`web/src/gen/proto/` | 阶段管理、Pipeline 引入、类型隔离、差异确认与导航。 |
| 测试与文档 | `internal/infrastructure/database/migration_test.go`、Pipeline usecase/handler/repository 测试、`docs/product/`、`docs/guides/ci-pipeline-design.md` | 迁移/事务/行为回归与活文档同步。 |

## Verification plan

1. Migration tests：从空 SQLite 数据库迁移到最新版本，断言引用表、索引、`PRAGMA integrity_check` 和 `foreign_key_check`；阶段字段形态不由 migration `CHECK` 拒绝，不将旧阶段转换留给应用代码。
2. Repository/usecase tests：覆盖模板阶段的项目隔离与版本递增、模板字段拒绝、重复引入、Pipeline 名称与 DAG 约束、应用阶段来源/JSON 数组/排序必填校验、Template 删除不级联、实例化只读取引用快照、依赖 ID 重映射、来源模板删除后仍可实例化、显式更新冲突和 Snapshot 冻结。
3. HTTP/Proto tests：覆盖新 `/api/pipeline-stage` CRUD、成员校验、导入、节点更新/删除、更新预览/确认和旧创建契约移除；生成的 Go/TypeScript 类型必须与 handler/前端调用一致。
4. Frontend tests and review：覆盖阶段库筛选/表单错误、模板与应用编辑器字段差异、引入模板、警告色更新操作和差异确认；人工检查小屏下表格/弹窗不溢出及模板阶段不显示制品配置。
5. Toolchain：运行 `task sqlc`、`task proto`、`go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，并在变更相关测试存在时运行 `yarn --cwd web test`。

## Blockers, assumptions and rollback

- 旧 Pipeline 配置在主业务完成后由独立脚本处理；该处理不进入本次业务实现。新业务运行时若遇到无来源应用阶段，不以空字段、默认模板或旧 API 工作。
- 当前 `sql/schema/schema.sql` 的注释/基线与迁移目录版本不一致；步骤 1 必须以实际迁移目录和空库迁移结果为准校准，不能回改历史文件。
- 模板阶段物理删除是允许的。它不会影响已物化应用阶段、历史数据或包含该来源快照的 Template Pipeline 后续实例化；引用和应用阶段通过来源 ID/名称/版本的逻辑布局保留审计展示。
- 本轮不增加新的 RBAC permission、跨项目阶段库、自动同步、运行时模板回读或数据兼容层。
- 回滚是恢复迁移前的数据库备份并部署旧应用版本；不得通过新代码同时支持两种阶段模型。迁移落地前必须先确认离线恢复材料可用。

## User review notes

1. 用户要求统一领域名为 `PipelineStage(kind=template|application)`，阶段管理只管理模板类型。
2. 模板阶段制品必须为空，制品、组件映射和来源 Version 策略只在应用流水线语境中生效。
3. 每个应用阶段强制保存模板来源；不在业务逻辑中迁移或兼容既有无来源数据。
4. 模板更新是带差异预览的显式操作，按钮位于阶段编辑器右上角并使用警告色；普通保存不触发同步。
5. 用户确认 Template Pipeline 实例化只使用关联阶段引用快照，不重新读取阶段库；模板删除后仍支持该快照物化。
6. 用户确认旧数据在主业务完成后由独立脚本处理，业务代码不迁移、不兼容。
7. 用户要求阶段字段形态校验位于业务服务，不使用数据库 `CHECK` 约束。
