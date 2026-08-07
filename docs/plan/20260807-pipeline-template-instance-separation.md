# 流水线模板与应用流水线分离计划
最后修改时间: 2026-08-07 18:55:18

Review status: Accepted

流程模式: 严格 / strict

## Input and boundary

- 依据已接受的 [Requirement](../requirement/20260807-pipeline-template-instance-separation.md) 和 [Spec](../spec/20260807-pipeline-template-instance-separation.md)。
- 实现目标是以 `Pipeline(kind=template|application)` 取代现有 Template、全局 Stage 和 Template Snapshot 运行模型。
- 业务代码、HTTP/Proto、Web 与测试只支持新模型；不保留旧路由、字段、DTO、SQL 查询或运行时分支。
- 开发数据库的就地结构和数据切换由独立脚本或 CI 作业实现，不嵌入业务运行时。用户已授权在本计划中补充独立 SQLite 切换脚本；它不修改已执行迁移文件或 `schema_migrations`，也不为业务代码增加兼容路径。
- Webhook 尚未正式投入使用，本计划删除旧的 Template Webhook 业务入口和前端，不设计新的 PipelineWebhook。
- 不启动、停止或重启开发服务器。

## Implementation status

1. 步骤 1 至 6 已完成：新 Pipeline 聚合、Application Pipeline 实例化、Snapshot/Run、HTTP/Proto、SQLC 和前端工作流均已切换到新模型。
2. 步骤 7 的活文档已完成首次同步；本次更新继续将 RSPV 与实现、离线迁移和变量预览行为对齐。
3. 数据库基线改为在 version 30 直接创建最终 Pipeline 模型，并通过空库迁移测试验证。
4. 尚未完成的交付项是浏览器人工验收；完整 Go 测试目前仅因迁移版本断言仍期望 `30` 而失败，用户已明确将该单元测试修复留给后续会话。

## Implementation steps

1. 定义新 Pipeline 聚合的模型、DTO、Repository 端口和 SQL 查询。
   - 将 `internal/model/pipeline.go` 从 `PipelineTemplate`、全局 `PipelineStage` 与关联行重构为 `Pipeline`、自有 `PipelineStage` 和 `PipelineSnapshot`。
   - 在 `Pipeline` 中实现 `kind`、来源 Template/Application/Repository/固定 Version 的逻辑引用及名称/标签快照、来源 Version 策略和 version；在 Artifact 声明中保存可选 `component_name`，Stage 不保存 Application/Component/fork 绑定。
   - 更新 `internal/model/pipeline_run.go`，以 Pipeline 身份替换 Template 身份，同时保留 `snapshot_id`、单条 Run 级 Version 逻辑绑定和 Artifact 历史字段；在 Version Component 保存 Artifact 的展示快照。
   - 替换 `internal/repository/{pipeline,pipeline_run,repository}.go` 中的 Template/全局 Stage/Webhook 能力；调整 `sql/query/pipeline/*.sql`、`sql/query/pipeline_run/*.sql` 和生成的 sqlc 适配器，使所有查询以 Pipeline 所有权和 Pipeline ID 过滤。
   - 按 Spec 的引用评估表建立 SQL 约束：Stage/Run/Snapshot/Artifact 的聚合内所有权保留物理外键，Template/Application/Repository/Version/Pipeline 历史指向改为逻辑外键，不以引用计数阻断物理删除或要求 `SET NULL`。

2. 重建 Pipeline 配置用例，按 kind 分离允许的操作。
   - 在 `internal/application/pipeline/usecase/` 以 Pipeline CRUD 取代 Template/Stage 分离 CRUD：Template 可以创建、编辑自有 Stage/DAG/变量；Application Pipeline 只能通过 Template instantiate 创建。
   - 实现 instantiate 事务：验证同 Project 的 Template 和必填 Repository；仅在提供 Application 时验证其 Project 归属。保存 Template ID/版本，深复制变量和 Stage，并使用旧到新 Stage ID 映射重写 `depends_on`。
   - 将 Application、Repository 和来源 Template 作为 Application Pipeline 不可变身份；业务更新接口拒绝更换它们。
   - 在 Stage 保存与 Pipeline 更新时统一递增所属 Pipeline version；拒绝 Template Artifact 的 `component_name`，校验 Application Pipeline 的来源 Version 策略、全 Pipeline Component 唯一性和每个绑定镜像 Artifact 的唯一上游 source commit。
   - 删除旧 `pipeline_stage_build_version_binding` 的删除阻断语义；fixed Version 为逻辑引用，缺失时只在后续 Run 返回明确错误。

3. 将 Snapshot 业务完整迁移到 Application Pipeline。
   - 重写 `internal/application/pipeline/usecase/snapshot_create.go` 与 `snapshot.go`：仅 `kind=application` 可以 `GetOrCreatePipelineSnapshot`；Template 调用被拒绝。
   - 保持当前 “Pipeline version 相等复用 Snapshot，否则物化新的 `stages_snapshot`、`variables_snapshot`” 规则，并把 Application、Repository、来源 Template ID/版本、来源 Version 策略、Artifact Component 映射及其展示快照写入 Snapshot。
   - 修改 Snapshot 详情、Repository 查询和权限检查，使其从 Snapshot 的 Application Pipeline 归属而不是旧 Template 归属判断。
   - 保证 worker 与 Run 执行只读取 Snapshot 载荷，不从可变 Pipeline 或 Stage 记录回读执行输入。

4. 收敛所有 PipelineRun 创建与执行路径。
   - 将手动触发改为 `TriggerPipeline`：根据 Pipeline 固定 Repository 解析 ref；存在 Component-bound Artifact 时按 Snapshot 的 Pipeline 策略解析一个来源 Version，并原子写入 Run 与唯一的 `PipelineRunVersionBinding`。
   - Retry 通过同一内部 Run 创建服务，并按新 Run 创建时刻重新解析 `latest`；保留旧 Run 的 Snapshot 与绑定历史。
   - 提供无副作用的 `variable-preview` 应用服务，复用 Run 创建的变量声明、Repository 覆盖和用户覆盖解析；详情用默认输入加载结果，运行对话框按当前 ref/覆盖重算，预览不得创建 Snapshot、Run、任务或 Version binding。
   - 删除 Repository Template Webhook 的 CRUD、公开接收和执行路径，不增加新的 Webhook 领域模型或依赖注入。
   - 更新 `internal/application/pipeline_run/usecase/{pipeline_run,build_version_binding,execution,executor}.go`：Stage 成功时先归档所有 Artifact；全部 Stage 成功后，在一个事务中 fork 一次 Version、更新所有目标 Component 并回写唯一 Run 级 Version 绑定。

5. 更换 HTTP、Proto、路由和依赖注入契约。
   - 以 `proto/orbit/v1/pipeline/{pipeline,pipeline_stage,snapshot}.proto` 取代旧 `template.proto` 契约；移除 Stage 级 Application ID/Component/fork 字段和 Template Trigger 字段，在 Artifact 声明中增加 Component 映射。
   - 将 `pipeline_run.proto`、`artifact.proto` 的 Template 字段替换为 Pipeline 字段，触发请求只保留 `trigger_ref` 和 `variables`。
   - 更新 `internal/api/http/handler/{pipeline,pipeline_run,repository}/`，删除 Repository trigger 与 Template Webhook handler，新增 Pipeline instantiate 与 trigger handler。
   - 注册 `POST /api/pipeline/{pipeline_id}/variable-preview`，并使其仅返回变量解析结果。
   - 在 `internal/api/http/routes/` 移除 `/api/pipeline/template/*`、`/api/pipeline/stage/*`、`/api/repository/:id/trigger` 和 Repository Webhook 路由，注册 Spec 中定义的 Pipeline 单数路由。
   - 调整 `internal/bootstrap/http.go` 的 Store 与服务组合，删除 Webhook 依赖，组合新的 Pipeline 与 PipelineRun 服务。
   - 运行 SQLC 与 Go/TypeScript Buf 生成，只保留新生成的 Go、TypeScript 与 sqlc 代码。

6. 重做前端的信息架构与运行路径。
   - 以 `web/src/views/pipeline/` 的 Pipeline 列表与详情取代独立的 Stage 列表、Template 列表和 Template 详情；按 kind 筛选 Template 与 Application Pipeline。
   - Template 详情只编辑蓝图、Stage/DAG 和变量，并提供“创建应用流水线”表单；该表单必须选择 Repository，并在其下方可选选择 Application。
   - Application Pipeline 详情显示不可变来源 Template/Repository 和可选 Application，管理私有 Stage、变量；只有绑定 Application 时才提供 Pipeline 来源 Version 策略与镜像制品 Component 映射，并提供运行和历史入口。
   - 详情变量声明表加载默认解析值；运行对话框复用同一预览函数，在首屏或重新计算时显示加载状态，预览失败时禁用确认。
   - 删除 `web/src/views/repository/components/TriggerModal.vue` 与 `WebhookList.vue` 的 Repository/Template 交互；本轮不提供替代 Webhook 管理界面。
   - 更新 `web/src/api/pipeline/*`、`web/src/api/pipeline_run/*`、`web/src/api/repository/*`、`web/src/router/index.ts`、导航菜单与 i18n，移除 Template ID 字段并展示 Pipeline ID/名称/版本与 Snapshot 追溯。
   - 更新 PipelineRun、Artifact、Snapshot、Repository、Application Version 的详情页，让其从 Pipeline/Run binding 显示来源 Template、Application、Repository、生成 Version、Component、镜像 SHA 和 source commit。

7. 更新活文档并清理旧模型表述。
   - 更新 `docs/product/overview.md`、`docs/guides/ci-pipeline-design.md`、`docs/guides/ci-pipeline-vars-design.md`，说明 Template 版本、Application Pipeline 版本/Snapshot、固定 Repository、可选 Application 与唯一触发入口。
   - 更新 `docs/INDEX.md` 中 CI 入口名称；删除或重写仍声明全局可复用 Stage、Template 直接触发、Repository Template Webhook 的活文档内容。
   - 保留本任务的 Requirement、Spec、Plan；仅在验收完成后创建 Verification。

8. 将最终 Pipeline 模型并入 SQLite/MySQL 的 version 20-23 基础 DDL 和 version 30 种子。
   - 空库迁移至 version 30 后直接存在 Pipeline、私有 Stage、Snapshot、Run、Artifact、Run Version binding 和 VersionComponent 最终字段。
   - 种子 Template 使用私有 Stage ID 作为 DAG 依赖，不创建旧 Template、全局 Stage、Stage binding 或 Webhook 表。

## Files and ownership

| 区域 | 主要路径 | 预期责任 |
| --- | --- | --- |
| 领域与持久化 | `internal/model/`, `internal/repository/`, `sql/query/pipeline/`, `sql/query/pipeline_run/` | 新 Pipeline 聚合、Snapshot/Run/Artifact 身份与查询 |
| Pipeline 用例 | `internal/application/pipeline/` | Template 编辑、Application Pipeline instantiate、Artifact Component 映射、Snapshot |
| 执行用例 | `internal/application/pipeline_run/` | 统一创建 Run、Retry、执行、Artifact 与 Version fork |
| Repository 用例 | `internal/application/repository/` | 移除旧 Trigger/Webhook 职责，保留 Repository 源码访问 |
| 传输与组装 | `proto/orbit/v1/pipeline*`, `internal/api/http/`, `internal/bootstrap/http.go` | 破坏性 API 契约、路由和依赖注入 |
| 前端 | `web/src/views/pipeline/`, `web/src/views/pipeline_run/`, `web/src/api/`, `web/src/router/` | Template/Application Pipeline 工作流、运行与历史展示 |
| 文档 | `docs/product/`, `docs/guides/`, `docs/INDEX.md` | 新术语和操作路径 |
| 数据库基线 | `sql/migration/{sqlite,mysql}/000020..000030_*` | 空库直接创建最终 Pipeline 模型和种子 |

## Verification plan

1. 模型和 Repository 测试：覆盖 kind 约束、Project 归属、Template 不可运行/无 Snapshot、Application Pipeline 身份不可改、Template 实例化的深复制与 DAG ID 重映射，以及逻辑/物理外键评估表的删除行为和展示冗余字段。
2. Snapshot 测试：覆盖 Template 拒绝创建 Snapshot、Application Pipeline 同版本复用、版本变化创建新 Snapshot，以及 Run 始终使用 Snapshot 而非当前 Pipeline 配置。
3. Run 和 Artifact 测试：覆盖手动触发与 Retry 共用绑定冻结；`latest`/`fixed`；多个跨 Stage 的 Component-bound Artifact；镜像 Artifact/source commit；全 Run 单次 fork、全部 Component 原子更新与 Run 级历史。
4. HTTP/Proto 测试：覆盖新 Pipeline 路由和请求响应，确认旧 Template/Repository trigger 路由、字段和 handler 不再注册。
5. 前端测试：更新 router/API 单测与相关 Vitest 断言；手工检查 Template 创建 Application Pipeline（Application 可选）、运行只输入 ref/变量、详情默认变量值与运行时重算、Artifact Component 映射、Snapshot/Artifact 追溯、删除弹窗错误反馈与 Webhook 入口移除。
6. 生成与静态检查：运行 `./bin/sqlc generate`、`./bin/buf generate`、`./bin/buf generate . --template web/buf.gen.yaml --output web`、`go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。完整 Go 测试的迁移版本断言跟随当前 schema 版本更新，不能以业务兼容分支掩盖。
7. 离线迁移：在当前开发 SQLite 的独立副本执行预检和完整切换，确认历史删除计数、备份和迁移后 `PRAGMA integrity_check` / `foreign_key_check`；原开发库不得写入。

## Blockers and assumptions

- 数据库就地迁移通过独立脚本完成；环境切换时必须按逻辑/物理外键评估表完成一致性检查，业务代码不处理旧表或未转换数据。
- 旧共享 Stage/Application 绑定或 Template Snapshot 无法确定对应 Application Pipeline 时，独立脚本不得推断默认映射；它将旧运行历史作为显式删除项，保留 Template 供用户重新实例化。
- 一个 Application Pipeline 可以跨多个 Stage 声明多个 Component-bound 镜像 Artifact，但每个 Component 只能被一个声明绑定；一个成功 Run 最多 fork 一个 Version。
- 由于新旧 API 与 schema 不兼容，切换后的回退需要通过独立数据脚本恢复先前数据库状态并部署旧版本，不能通过新业务代码同时支持两种模型。

## User review notes

1. 用户明确要求从架构与产品目标选择正确模型，不以最小改动或保守兼容为准。
2. 用户明确要求业务代码不做向后兼容。
3. 用户明确要求数据库的开发环境就地变更由独立脚本或 CI 作业处理，不写进业务实现。
4. 用户确认 Template 保留版本但不再有 Snapshot；Application Pipeline 沿用 Snapshot、Run 与 Artifact 数据链。
5. 用户确认 Pipeline 允许物理删除，不增加状态；跨生命周期引用使用带展示冗余字段的逻辑外键，只有聚合内组成关系使用物理外键。
6. 用户确认 Component 映射属于 `docker_image` 制品声明，一个成功 Run 最多 fork 一个 Application Version；Webhook 暂不纳入本轮模型。
7. 空库迁移链直接建立最终模型；不提供开发数据库的就地切换脚本，也不扩展业务兼容范围。
