# 可复用流水线阶段规格
最后修改时间: 2026-08-08 13:39:00

Review status: Accepted

流程模式: 严格 / strict

## Requirement basis

- 需求文档：[可复用流水线阶段](../requirement/20260808-pipeline-stage-template-library.md)，已接受。
- 当前实现中，`Pipeline` 已通过 `kind=template|application` 区分模板和应用流水线；`PipelineStage` 仍是带有 `pipeline_id`、DAG、排序及制品 JSON 的 Pipeline 私有节点。
- 本规格取代该私有节点模型。代码领域名仍为 `PipelineStage`，不引入独立的 `PipelineStageTemplate` 领域实体；可复用资源由 `PipelineStage.kind=template` 表示。

## Overview

阶段定义、模板流水线编排和应用流水线执行配置必须分开保存。共享阶段只保存可复用的执行定义；任何 DAG、排序、应用制品或来源 Version 策略都不能回写到它。

```text
PipelineStage(kind=template, project scoped, versioned)
  name / image / script / description
                 |
                 | import: copy definition + record source snapshot
                 v
PipelineStageReference (owned by Template Pipeline)
  local name / description / image / script / depends_on / sort_order
  source stage id / name / description / applied version
                 |
                 | instantiate: materialize reference snapshot and remap DAG identifiers
                 v
PipelineStage(kind=application, owned by Application Pipeline)
  local name / description / image / script / depends_on / sort_order
  source stage id / name / description / applied version / artifacts
                 |
                 v
PipelineSnapshot -> PipelineRun -> Artifact / generated Version
```

`PipelineStageReference` 是模板流水线内的编排节点，不是新的共享阶段实体。它保存引入时的完整执行定义快照，因此同一模板阶段可被同一流水线多次引入，并且以后模板阶段的更新或删除不会改写已有节点。创建新的 Application Pipeline 时，服务端只读取该 Template Pipeline 的引用快照并物化新的应用阶段，不重新读取阶段库。

## Design decisions

### 领域边界

1. `PipelineStage.kind` 仅允许 `template` 与 `application`。
2. `template` 阶段归属一个 Project，不归属 Pipeline；名称在同一 Project 的模板阶段中唯一，拥有单调递增的 `version`。
3. `application` 阶段归属一条 `kind=application` 的 Pipeline，是可执行的私有节点。它必须保存非空的来源模板阶段 ID、名称及已应用版本，但不对来源模板阶段建立物理外键。
4. Template Pipeline 不拥有 `PipelineStage(kind=application)`。它使用 `PipelineStageReference` 保存模板阶段的冻结定义、私有节点名称、依赖和排序。
5. 应用阶段的 `artifacts` 是唯一可持久化制品配置的位置。模板阶段和模板流水线引用均没有制品字段；`component_name` 只能出现在应用阶段的 `docker_image` 制品中。
6. 来源 Version 策略仍是 Application Pipeline 的配置，沿用现有“有组件映射时才需要 `latest|fixed`”及唯一上游 `git_object_id` 规则。它不属于任意 `PipelineStage(kind=template)`，也不迁入 `PipelineStageReference`。
7. Pipeline 节点名称是私有名称。引入时默认等于模板阶段名称；同一个来源可多次引入，Pipeline 内名称唯一性仍由服务端校验，用户自行使它们可辨识。

### 冻结、更新与删除

1. 创建模板阶段的 `version=1`；名称、镜像、脚本或说明发生实际改变时递增一次版本。无变化更新不递增版本。
2. 从阶段库引入时，服务端读取当前模板阶段，并将其 ID、名称、版本、镜像、脚本和说明复制到目标节点；节点说明同时作为可编辑的私有说明初始化。客户端只能提交模板阶段 ID 与目标节点的私有编排字段，不能提交或伪造模板定义。
3. 模板流水线已有引用不会自动跟随模板阶段更新。实例化新的 Application Pipeline 时，服务端使用引用中冻结的来源 ID、名称、版本、镜像、脚本和说明快照物化应用阶段，并保留引用的私有名称、私有说明、DAG 与排序；不重新读取阶段库。直接在应用流水线引入时才使用当时的最新模板阶段版本。
4. 应用阶段、模板流水线引用和历史 Snapshot 都不在运行时读取模板阶段。模板阶段更新、删除或缺失不能改变已有应用流水线、Snapshot、Run、Artifact 或已生成 Version。
5. 删除模板阶段只删除其 `kind=template` 行。既有引用和应用阶段因保留来源快照继续可查看和使用；Template Pipeline 仍可从引用快照创建新的 Application Pipeline，已物化的应用阶段继续执行。既有引用不再显示“更新至模板”操作。删除不级联删除 Pipeline、应用阶段、Snapshot、Run 或 Artifact。
6. 模板阶段来源仍存在且其当前版本大于节点的已应用版本时，节点详情响应标识有可用更新。用户必须显式预览并确认更新；普通“保存”不能隐式更新来源定义。
7. 确认更新会以当前模板阶段定义覆盖节点的镜像和脚本，并把来源名称、来源说明快照和已应用版本更新为当前值；保留节点私有名称、私有说明、DAG 依赖、排序、制品、组件映射及 Application Pipeline 的 Version 策略。预览仍展示模板说明的差异，明确它更新的是来源快照，不会覆盖用户编辑的节点说明。

### 数据处置边界

本次切换不在 HTTP handler、usecase、repository、worker 或运行时读取路径中兼容无来源的旧 `pipeline_stage` 数据。新模型要求每一条应用阶段均有非空来源快照。旧数据在主业务完成后由独立离线脚本处理；该脚本不属于本功能的业务实现，也不改变新模型的运行时分支。

部署前，运维或开发环境必须通过离线数据迁移或重建处置旧 Pipeline 配置；该处置不属于本功能的业务实现。新 DDL migration 只建立新模型，不尝试猜测旧阶段的模板来源，也不以空来源、默认模板或旧 API 降级。现有 Snapshot、Run、StageRun 和 Artifact 已保存运行时快照/展示字段，不依赖 `pipeline_stage` 的物理外键，必须保留。

## Persistence model

新 migration 编号以落地时迁移链的下一个编号为准；当前链末为 `000029`，预期新增 `000030_pipeline_stage_template_library` 的 SQLite/MySQL up/down 文件，并同步 `sql/schema/schema.sql` 与 SQLC 输入。不得修改 `000020_pipeline` 或其他已执行 migration。

### `pipeline_stage`

重建为同一领域表，不保留旧的 `depends_on` 与 `sort_order` 列。

| 字段 | `template` | `application` | 说明 |
|---|---|---|---|
| `id` | 必填 | 必填 | ULID；应用阶段 ID 是应用 Pipeline DAG 的本地节点 ID。 |
| `project_id` | 必填 | 必填 | 通过 Pipeline/Project 校验，不能跨项目引用。 |
| `kind` | `template` | `application` | 由 Pipeline 业务服务校验。 |
| `pipeline_id` | `NULL` | 必填 | 仅应用阶段；物理外键指向 Pipeline，删除应用 Pipeline 时级联删除。 |
| `name` / `image` / `script` / `description` | 必填 | 必填 | 可复用执行定义或其私有物化副本。 |
| `version` | 正整数 | `NULL` | 仅模板阶段自身版本。 |
| `source_template_stage_id` | `NULL` | 必填 | 逻辑引用，不设物理外键。 |
| `source_template_stage_name` | `NULL` | 必填 | 引入/更新时的来源名称快照。 |
| `source_template_stage_version` | `NULL` | 正整数 | 引入/更新时的来源版本快照。 |
| `source_template_stage_description` | `NULL` | 必填 | 引入/更新时的模板说明快照；节点 `description` 是可编辑的私有说明。 |
| `artifacts` | `NULL` | 非空 JSON 数组 | 仅应用阶段可保存，空数组表示尚未声明制品。 |
| `created_at` / `updated_at` | 必填 | 必填 | 现有时间语义。 |

模板/应用阶段的字段形态由 Pipeline 业务服务校验：模板阶段不得携带 Pipeline、来源、制品、依赖或排序；应用阶段必须具备 Pipeline、非空来源快照、JSON 数组制品与依赖以及非负排序，且 `kind` 只能为 `template|application`。数据库只保留外键、非空列、唯一性和索引等结构完整性；模板阶段同一 Project 的名称唯一、应用阶段 `(pipeline_id, name)` 唯一。普通索引只保留 `(project_id, kind, name)`；应用阶段的唯一键可覆盖按 Pipeline 的读取与名称校验。

### `pipeline_stage_reference`

这是 Template Pipeline 的私有编排表，不等同于 `PipelineStage`。

| 字段 | 说明 |
|---|---|
| `id` | ULID，也是 Template Pipeline DAG 的本地节点 ID。 |
| `pipeline_id` | 必填，物理外键指向 `kind=template` 的 Pipeline，删除 Pipeline 时级联删除。kind 一致性由 usecase 校验。 |
| `source_template_stage_id` / `source_template_stage_name` / `source_template_stage_version` / `source_template_stage_description` | 必填的逻辑来源快照；不设到模板阶段的物理外键。 |
| `name` | 必填，Pipeline 私有节点名称；同一 Pipeline 唯一。 |
| `image` / `script` | 引入时冻结的模板执行定义；显式更新来源版本时替换。 |
| `description` | Pipeline 私有说明，引入时以模板说明初始化；节点编辑和模板更新均不得隐式丢失它。 |
| `depends_on` | 非空 JSON 数组，只可引用同一 Template Pipeline 的引用 ID。 |
| `sort_order` | 非负整数；仅是同层展示/稳定排序，不替代依赖边。 |
| `created_at` / `updated_at` | 现有时间语义。 |

不额外建立普通索引。`(pipeline_id, name)` 唯一键可覆盖按 Pipeline 读取引用；阶段数量有限，排序在查询中完成。不建立到 `pipeline_stage` 的外键，使模板阶段删除后冻结引用仍然有效。

### Snapshot 与运行时记录

`PipelineSnapshot.stages_snapshot` 继续保存已解析的应用阶段 `StageDefinition`，无需为模板阶段增加运行时查询。可选地在 `StageDefinition` 中追加来源模板阶段 ID、名称和已应用版本，作为审计展示字段；执行器仍仅使用 ID、名称、镜像、脚本、依赖、排序和制品定义。该追加字段不改变已有 Snapshot 的不可变性。

不修改 `pipeline_run`、`pipeline_stage_run`、`artifact`、`pipeline_run_version_binding` 的历史语义或外键。旧 Pipeline 配置的离线处置不得删除这些运行记录。

## Application services and transactions

`PipelineStore` 扩展为两个明确的聚合读写面：Project 级模板阶段库，以及按 Pipeline kind 加载/保存的编排节点。不要让调用方在同一个 `PipelineStages` 方法中混淆模板引用和应用阶段。

### 模板阶段库

服务提供 `ListPipelineStageTemplates`、`PipelineStageTemplateForUser`、`CreatePipelineStageTemplate`、`UpdatePipelineStageTemplate` 和 `DeletePipelineStageTemplate`。所有入口先校验 Project 存在与成员资格，再校验资源 Project 归属。创建和更新只接受名称、镜像、脚本、说明；任何制品、依赖、排序、Pipeline ID、应用绑定或来源 Version 字段均返回 validation error。

### 引入与编辑 Pipeline 节点

1. 引入目标为 Template Pipeline 时，事务内锁定/读取同 Project 的模板阶段，创建一条 `PipelineStageReference`，应用用户提交的私有名称、说明、依赖和排序，执行 DAG 校验，Pipeline `version + 1`，并一次提交。
2. 引入目标为 Application Pipeline 时，事务内读取模板阶段，创建一条 `PipelineStage(kind=application)`，将来源 ID、名称、版本、说明快照和执行定义写为非空快照，初始化 `artifacts=[]`，写入私有名称、说明、依赖/排序，执行 DAG、应用/组件映射及 Version 策略校验，Pipeline `version + 1`，并一次提交。
3. Template Pipeline 节点更新只允许私有名称、说明、依赖和排序；其镜像和脚本只能通过“更新至模板”动作替换。Application Pipeline 节点更新允许私有名称、说明、依赖、排序及制品，镜像和脚本可作为本地调整保存。
4. 任一节点删除都会先从同一 Pipeline 的节点集移除它，并拒绝仍被其他节点 `depends_on` 引用的删除请求。通过校验后删除节点且 Pipeline `version + 1`，同一事务提交。
5. 任何改变 Pipeline 配置的引入、编辑、删除、来源更新和应用流水线物化都在同一个 repository transaction 内完成 Pipeline 版本递增及完整校验。无实际字段改变不递增版本。

### 从模板流水线实例化

仅 Template Pipeline 可实例化。事务中读取其 `PipelineStageReference` 列表，为每个引用生成新的 `PipelineStage(kind=application)` ID，复制引用快照中的镜像、脚本、来源名称、来源说明和版本，保留引用的私有名称与私有说明，初始化空制品数组，并将 `depends_on` 从引用 ID 映射到新应用阶段 ID。新 Application Pipeline 和全部阶段必须在同一事务创建；任一引用来源快照缺失、DAG 无效或复制失败均整体回滚。来源模板阶段物理删除不阻断该过程。

### 显式更新至模板

预览与应用都必须重新按 Pipeline ID 和节点 ID 校验成员资格、节点归属、来源模板 ID、来源版本和 Project。预览返回当前节点的来源快照、当前模板阶段及名称/镜像/脚本/说明的字段差异。

应用请求带 `expected_source_template_stage_version` 与 `target_template_stage_version`，用作乐观并发检查。服务拒绝以下情况：来源模板不存在、模板不属于 Pipeline Project、目标版本不大于已应用版本、模板当前版本与目标版本不一致，或节点在预览之后已变更来源。成功时以当前模板阶段定义更新来源快照、镜像和脚本，保留私有名称、说明及其他私有字段，重新执行 DAG/组件映射校验，Pipeline `version + 1`，同一事务提交。前端收到冲突后重新加载详情和预览，不自动重试。

## HTTP and Proto contract

API 继续使用单数资源路由。当前 `/api/pipeline/:pipeline_id/stage` 的“直接提交执行定义创建阶段”契约移除；同一路径改为仅从模板阶段引入，前端与 Proto 同轮切换，不保留旧请求别名。

### 阶段库

| 方法与路径 | 请求/响应 | 语义 |
|---|---|---|
| `GET /api/pipeline-stage` | `project_id`、`search`、分页 -> `PipelineStagePaginatedResp` | 只返回当前 Project 的 `kind=template` 阶段。 |
| `POST /api/pipeline-stage?project_id=` | `PipelineStageCreateReq` -> `PipelineStageResp` | 创建模板阶段。 |
| `GET /api/pipeline-stage/:stage_id` | -> `PipelineStageResp` | 返回模板阶段详情。 |
| `PUT /api/pipeline-stage/:stage_id` | `PipelineStageUpdateReq` -> `PipelineStageResp` | 更新定义，发生实际变化时版本递增。 |
| `DELETE /api/pipeline-stage/:stage_id` | `204 No Content` | 删除模板阶段，不级联现有编排或运行数据。 |

`PipelineStageResp` 用于阶段库资源，字段包含 `id`、`project_id`、`kind`、`name`、`image`、`script`、`description`、`version`、`created_at`、`updated_at`；它不包含制品、依赖、排序或 Application 配置。

### Pipeline 编排节点

`PipelineResp.stages` 更名为 `repeated PipelineStageNodeResp stage_nodes`，避免把“阶段模板资源”和“Pipeline 内节点”混为同一 DTO。旧字段不保留兼容别名。

`PipelineStageNodeResp` 必须包含本地节点 ID、节点类型（`template_reference|application`）、私有名称、私有说明、镜像、脚本、依赖、排序、来源模板阶段 ID/名称/说明/已应用版本、可选 `latest_template_stage_version` 以及应用节点的制品。模板引用绝不返回制品字段内容；应用节点的来源字段必须非空。

| 方法与路径 | 请求/响应 | 语义 |
|---|---|---|
| `POST /api/pipeline/:pipeline_id/stage` | `PipelineStageImportReq` -> `PipelineResp` | 仅接受 `source_template_stage_id`、私有名称、依赖和排序；按目标 Pipeline 类型创建引用或应用阶段。 |
| `PUT /api/pipeline/:pipeline_id/stage/:stage_id` | `PipelineStageNodeUpdateReq` -> `PipelineResp` | 更新同一 Pipeline 的私有节点字段；字段白名单由 Pipeline 类型决定。 |
| `DELETE /api/pipeline/:pipeline_id/stage/:stage_id` | -> `PipelineResp` | 删除节点，并拒绝破坏其他节点依赖的操作。 |
| `GET /api/pipeline/:pipeline_id/stage/:stage_id/template-update-preview` | -> `PipelineStageTemplateUpdatePreviewResp` | 返回可用性、来源和目标版本及字段差异；无更新时成功返回 `available=false`。 |
| `POST /api/pipeline/:pipeline_id/stage/:stage_id/template-update` | `PipelineStageTemplateUpdateReq` -> `PipelineResp` | 使用预览期望版本显式应用模板更新。 |

`PipelineStageImportReq` 不含镜像、脚本、制品或组件映射字段，但允许可选的私有名称、说明、依赖和排序。`PipelineStageNodeUpdateReq` 对模板引用不提供制品、组件映射、镜像、脚本或来源策略字段；对应用节点的 `artifacts` 使用现有 `ArtifactConfigListReq`，继续由后端校验 collector、组件唯一性和 Pipeline 的 Version 策略。所有 Proto、Go 生成物和 TypeScript 生成物在同一改动生成并替换。

## Frontend behavior

### 导航与阶段管理

CI 二级导航新增“阶段”入口，路由为 `/pipeline-stage`，使用当前激活 Project 作为范围。新增 `PipelineStagePage.vue` 采用既有列表页组合：`app-toolbar-simple`、`SearchControl`、`app-surface`、`app-data-table`、`ListPagination`、`AppDialog` 和 `AppDialogActions`。

列表展示名称、执行镜像、版本、说明、更新时间和操作，支持搜索、分页、创建、编辑和删除。创建/编辑表单只有名称、执行镜像、脚本、说明；必填错误显示在对应控件下方。删除使用确认对话框。任何制品、依赖、排序、组件、应用或来源 Version 控件均不出现在此页。

### Pipeline 编辑

`PipelineDetail.vue` 的“添加阶段”替换为“引入阶段”。对 Template Pipeline 和 Application Pipeline 都打开同一个阶段库选择流程，使用 `ComboboxSelect` 搜索并选择本 Project 的模板阶段；选择后只编辑私有名称、说明、依赖和排序。Template Pipeline 的节点编辑器不显示制品区域。

Application Pipeline 的节点编辑器保留现有制品及组件映射配置，且只有其 `docker_image` 制品可显示组件映射和 Pipeline 的来源 Version 策略。若删除制品、切换 collector 或取消组件映射，应清除已不适用的本地字段错误；字段错误不能只通过 Toast 提示。

节点来源仍存在且版本有更新时，阶段编辑器标题右上角按钮组显示警告色 `更新至模板 vN`。该样式作为共享 warning button variant 提供，不能在页面内临时拼色。点击后打开差异确认 `AppDialog`，逐项显示旧/新来源名称、镜像、脚本和模板说明，并明确标注私有说明会保留；确认主操作文字为“更新并保存”。取消、关闭或普通保存均不更新模板来源。来源已删除或无较高版本时不显示按钮。

### 交互与刷新

所有成功的模板更新、引入、编辑和删除都以接口返回的完整 Pipeline 替换本地详情，并刷新变量预览。模板阶段库自身更新不会推送或自动刷新正在编辑的 Pipeline；重新打开/刷新 Pipeline 时由 `latest_template_stage_version` 决定是否展示更新操作。列表与详情的时间显示继续使用 `web/src/utils/time.ts`。

## Affected components

| 区域 | 预期改动 |
|---|---|
| `sql/migration/{sqlite,mysql}`、`sql/schema/schema.sql` | 新 DDL migration、`pipeline_stage` 重建、`pipeline_stage_reference`、结构完整性与索引。 |
| `sql/query/pipeline/pipeline.sql`、SQLC 生成物 | 区分阶段库、模板引用、应用阶段的查询与事务。 |
| `internal/model`、`internal/repository`、`internal/infrastructure` | 新 kind/source/reference 模型与分离的 store API；删除旧私有阶段单模型接口。 |
| `internal/application/pipeline` | 模板阶段 CRUD、引入、物化、显式更新、DAG/组件映射校验及事务测试。 |
| `proto/orbit/v1/pipeline`、`internal/gen/proto`、`web/src/gen/proto` | 阶段库 DTO、节点 DTO、引入/更新/预览契约及生成代码。 |
| `internal/api/http/{handler,routes}/pipeline` | 阶段库单数路由和新的 Pipeline 节点动作路由。 |
| `web/src/api/pipeline`、`web/src/views/pipeline`、`web/src/router/index.ts`、`web/src/navigation.ts`、i18n | 阶段库页、导航、节点编辑器、模板选择与差异确认。 |
| `docs/product`、`docs/guides/ci-pipeline-design.md` | 回写当前 Pipeline/Stage 领域边界、模板更新与运维数据处置说明。 |

## Risks and alternatives

### Risks

1. `pipeline_stage` 现有定义的 `pipeline_id NOT NULL` 与 DAG/排序列无法表达模板阶段，migration 必须重建表结构；旧数据后续由独立脚本处理，不能在运行时代码中吸收旧行。
2. 若模板阶段仍用物理外键被引用，删除会阻塞或级联破坏冻结节点；来源关系必须是逻辑快照。
3. 若阶段库更新时错误地更新 Pipeline 版本或 Snapshot，会改变后续 Run 的可预期性；只有显式节点更新才递增对应 Pipeline 版本。
4. Template Pipeline 引用 ID 与 Application Stage ID 不同。实例化必须只使用引用快照并完整映射依赖，否则会错误读取可变模板定义或得到指向不存在节点的 DAG；该转换需要定向单元测试。
5. 模板说明与 Pipeline 私有说明同时存在。差异弹窗必须清楚展示前者会更新为新来源快照、后者会保留，避免用户误把普通保存理解为同步。

### Rejected alternatives

1. 将全局模板阶段直接挂为多条 Pipeline 的共享 DAG 节点：拒绝。依赖、排序和私有名称无法隔离，同一模板阶段重复引入也没有稳定节点 ID。
2. 在应用流水线运行时读取最新模板阶段：拒绝。会破坏 Pipeline 版本、Snapshot 和历史执行的冻结语义。
3. 将制品及 Component 映射放入模板阶段：拒绝。它们依赖具体 Application、Version 与 Pipeline，无法项目内通用。
4. 模板阶段更新自动同步所有节点：拒绝。会静默覆盖已验证的流水线配置，并难以辨别哪些 Run 使用何种定义。
5. 为旧无来源阶段提供空来源或“手工阶段”兼容类型：拒绝。它会削弱强制来源约束并长期保留双模型。

## User review notes

1. 界面名称保持“阶段”，阶段管理只管理 `PipelineStage(kind=template)`。
2. 同一模板阶段允许被同一 Pipeline 多次引入，由用户自行命名区分。
3. 应用阶段必须有模板阶段来源；不在业务逻辑中迁移、兼容或适配既有无来源数据。
4. 模板阶段本身没有制品；只有 Application Pipeline 的应用阶段可以配置制品。
5. 模板有新版本时，阶段编辑器右上角须提供警告色、显式的“更新至模板 vN”操作，确认文案为“更新并保存”。
6. 用户确认 Template Pipeline 实例化使用关联引用的快照数据；模板阶段删除后，逻辑来源字段和快照仍可支持物化。
7. 用户确认旧数据在主业务完成后由独立脚本处理，业务运行时不迁移、不兼容。
8. 用户要求阶段类型字段形态不以数据库 `CHECK` 约束实现，统一由 Pipeline 业务服务验证。
