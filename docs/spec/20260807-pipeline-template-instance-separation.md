# 流水线模板与应用流水线分离规格
最后修改时间: 2026-08-07 18:55:18

Review status: Accepted

流程模式: 严格 / strict

## Requirement basis

- 依据已接受的 [流水线模板与应用流水线分离 Requirement](../requirement/20260807-pipeline-template-instance-separation.md)。
- 产品目标是让 Template 成为真正的来源蓝图，让 Application Pipeline 成为一次配置后可直接运行的交付单元。
- 本规格采用破坏性模型切换：业务代码、HTTP/Proto 和 Web 只支持新 Pipeline 模型，不保留旧 Template/Stage/Trigger 契约的兼容分支。
- 开发数据库从空库执行迁移链至 version 30，直接得到最终结构和种子数据；不提供旧结构就地转换。
- Webhook 尚未正式投入使用，本规格不定义或实现其新模型。

## Product model

`Pipeline` 是唯一流水线聚合根，通过 `kind` 区分两种产品对象：

```text
Pipeline(kind=template)              不可运行的通用蓝图
  └── PipelineStage[]                模板自有的通用阶段

Pipeline(kind=application)           可运行的交付流水线
  ├── source_pipeline_id -> Template
  ├── source_template_version
  ├── application_id -> Application（可选；仅 Component 映射时存在）
  ├── repository_id -> Repository
  └── PipelineStage[]                从模板物化的独立阶段和制品声明
```

Template 是创建 Pipeline 的唯一来源，不能直接触发、不能配置 Application、Repository 或 Version fork。Template 保留自身 `version`，用于标识蓝图变化和记录实例化来源，但不创建 Snapshot。Application Pipeline 必须从 Template 创建，并永久绑定一个同 Project 的 Repository；Application 可选，只有配置 Component 映射时才绑定同 Project 的 Application 并设置 `latest` 或 `fixed` 来源 Version 策略。运行只提供 ref 和变量，不再选择或修改 Application、Repository、Component 映射或 fork 策略。

`PipelineStage` 不再是全局可复用的独立资源。它由某个 Pipeline 独占，模板与 Application Pipeline 的阶段配置互不共享。这样，Template 的后续修改和其他 Application Pipeline 的调整不会意外改变已配置流水线。

## Aggregate design

### Pipeline

```text
Pipeline {
  id
  project_id
  kind: template | application
  source_pipeline_id: nullable, required when kind=application
  source_template_name: nullable, required when kind=application
  source_template_version: nullable, required when kind=application
  application_id: nullable, required when an Artifact has component_name
  application_name: nullable, required when application_id is set
  repository_id: nullable, required when kind=application
  repository_name: nullable, required when kind=application
  version_fork_strategy: nullable (latest | fixed)
  fixed_version_id: nullable
  fixed_version_label: nullable
  name
  description
  variable_declarations
  version
  created_at
  updated_at
}
```

约束：

- `kind=template` 时，`source_pipeline_id`、`source_template_name`、`source_template_version`、`application_id`、`application_name`、`repository_id`、`repository_name` 均为 `NULL`。
- `kind=application` 时，来源和 Repository ID/名称字段均非空，来源 Pipeline 必须是同 Project 的 Template，Repository 也必须属于同 Project。Application ID/名称必须同时存在或同时为空；存在时必须属于同 Project。
- Application Pipeline 存在至少一个 Component 绑定制品时，Application 和 `version_fork_strategy` 必须存在；策略为 `latest` 或 `fixed`。`fixed` 必须有属于该 Application 的 `fixed_version_id`，`latest` 不得有 `fixed_version_id`。没有任何 Component 绑定制品时，上述策略字段均为 `NULL`。
- `source_pipeline_id`、`repository_id` 是 Application Pipeline 的身份，创建后不可改。`application_id` 可在创建时为空；**仅当当前未绑定时允许首次补绑**，已绑定后不可更改或解绑。绑错应用时应从 Template 重新创建 Pipeline。
- Template 和 Application Pipeline 都以 `(project_id, name)` 唯一；同一 Application/Repository 可有多个 Pipeline，以支持不同构建策略或分支策略。
- Pipeline 使用物理删除，不引入归档或停用状态。数据库中既有 Pipeline/Snapshot/Run/Artifact 历史的具体处置属于后续独立迁移任务；业务代码不为旧结构保留删除兼容逻辑。

### Reference integrity and display snapshots

关系是否使用物理外键由所有权决定，而不是能否通过 ID 查询决定。跨生命周期关系使用逻辑外键：保存稳定 ID，以及在删除目标后仍需展示的名称、标签或版本；业务代码不得因这些逻辑引用阻止物理删除或要求数据库 `SET NULL`。同一聚合的组成关系才使用物理外键。

| 来源 | 目标 | 类型 | 保存的冗余展示字段 | 删除语义 |
| --- | --- | --- | --- | --- |
| Application Pipeline | 来源 Template | 逻辑外键 | `source_template_name`、`source_template_version` | 删除 Template 不要求更新或置空既有 Pipeline 来源信息 |
| Application Pipeline | Application | 逻辑外键 | `application_name` | 删除 Application 不因 Pipeline 引用被阻断；Pipeline 保留最后已知展示信息 |
| Application Pipeline | Repository | 逻辑外键 | `repository_name` | 删除 Repository 不因 Pipeline 引用被阻断；Pipeline 保留最后已知展示信息 |
| Application Pipeline | fixed Version | 逻辑外键 | `fixed_version_label` | 删除 Version 不因 fixed 策略引用被阻断；后续 Run 明确报告目标 Version 不存在 |
| PipelineSnapshot | Pipeline / Template / Application / Repository | 逻辑外键 | `pipeline_name`、`pipeline_version`、Template、Application、Repository 名称和版本 | 删除配置资源不改变既有 Snapshot 执行与展示载荷 |
| PipelineRun | Pipeline | 逻辑外键 | `pipeline_name`、`pipeline_version` | 删除 Pipeline 不要求删除 Run 历史 |
| PipelineRunVersionBinding | Application / source Version / generated Version | 逻辑外键 | Application 名称、Version 标签 | 运行历史不阻断资源删除，也不被回写 |
| VersionComponent | Artifact | 逻辑外键 | `artifact_name`、`artifact_image_ref`、`artifact_local_image_sha256`、`artifact_source_commit_sha` | 删除 Artifact 不需要数据库置空 Component 引用；Component 仍可展示构建来源快照 |
| PipelineStage | Pipeline | 物理外键 | 不适用 | Stage 是 Pipeline 配置组成部分，删除 Pipeline 时级联删除 |
| PipelineRun | PipelineSnapshot | 物理外键 | 不适用 | Snapshot 是 Run 的不可变执行输入，不能在仍被 Run 引用时删除 |
| Artifact | PipelineRun | 物理外键 | 不适用 | Artifact 是 Run 输出，删除 Run 时级联删除 |

空库迁移链必须按此表建立或移除约束。逻辑外键字段在创建时写入冗余展示值；列表和详情优先显示该快照值，不依赖已删除目标的 join。

`VersionComponent` 在关联构建 Artifact 时，同时写入上表定义的 Artifact 展示快照。Artifact 实体继续保存其完整 Run 级记录；Component 的冗余字段只服务于 Version 详情在 Artifact 或 Run 已物理删除后仍可呈现构建来源。

### PipelineStage

```text
PipelineStage {
  id
  pipeline_id
  name
  image
  script
  artifacts
  depends_on
  sort_order
  description
  created_at
  updated_at
}
```

- `pipeline_id` 是 Stage 的唯一所有者；一个 Stage 不能被第二个 Pipeline 或 Template 引用。
- Stage 不保存 Application、Component 或 Version fork 绑定。Application 身份和 Version fork 策略由 Application Pipeline 唯一确定。
- `depends_on` 只引用同一 Pipeline 的 Stage ID。复制 Template 时必须创建新的 Stage ID，并把所有依赖重映射到新 ID。

### Artifact declaration and Component mapping

`PipelineStage.artifacts` 中的每个声明保留现有 collector 载荷，并在 `docker_image` 上增加可选的 `component_name`：

```text
ArtifactConfig {
  name, collector, reference, command, format
  component_name: nullable, only for docker_image
}
```

- Template 中的所有 Artifact `component_name` 必须为空；Template 保持完全通用。
- Application Pipeline 的 `docker_image` Artifact 可选填写 `component_name`，表示该制品在成功 Run 生成的新 Version 中更新的 Component。未填写时，制品只做追溯，不参与 Version 生成。
- 同一 Application Pipeline 内，所有填写的 `component_name` 必须唯一；一个 Stage 可以声明多个镜像制品，多个 Stage 也可以分别声明不同 Component 的镜像制品。
- 每个绑定 Component 的 Artifact 都必须有唯一传递前置 `command/git_object_id` Artifact。一个绑定 Artifact 不再被限制为其所在 Stage 唯一的镜像制品。

### Template instantiation

从 Template 创建 Application Pipeline 是一次物化操作，输入为：

```text
template_id
name
repository_id
application_id (optional)
```

服务端在一个事务内验证输入的 Project 归属并：

1. 创建 `kind=application` 的 Pipeline；
2. 设置 `source_pipeline_id`、`source_template_version`、`repository_id` 与可选的 `application_id`；
3. 深复制 Template 的变量声明和全部 Stage 定义；
4. 用新 Stage ID 重写复制后 Stage 的 DAG 依赖；
5. 清空复制后所有 Artifact 的 `component_name`；
6. 返回新的 Application Pipeline，供用户在镜像制品声明中配置 Component 映射，并在 Pipeline 上配置来源 Version 策略。

创建后不存在隐式继承或同步。Template 的编排和脚本修改只影响之后创建的 Pipeline；Application Pipeline 的 Stage、变量和绑定修改只影响自身。若产品以后需要更新能力，应增加显式的、可预览的 Template reapply 工作流，不得以实时引用实现。

## Version and snapshot lifecycle

Template 与 Application Pipeline 使用相同的 `version` 字段，但它们有不同的业务含义和副作用：

| Pipeline kind | `version` 递增条件 | Snapshot 行为 |
| --- | --- | --- |
| `template` | 修改自有 Stage、DAG、变量、名称或说明 | 不创建、不查询、不复用 Snapshot |
| `application` | 修改自有 Stage、DAG、变量、镜像制品 Component 映射、来源 Version 策略、名称或说明 | 下一次运行前按版本创建或复用 Snapshot |

Template 创建 Application Pipeline 时，新的 Pipeline 从版本 `1` 开始，同时永久记录来源 Template 当时的 `source_template_version`。Template 后续升级不会改变已创建 Pipeline 的 `source_template_version` 或配置；Application Pipeline 后续升级也不会改变来源 Template 的版本。

`PipelineSnapshot` 仅服务于 `kind=application` 的 Pipeline，保留现有不可变快照职责：

```text
Application Pipeline(version=N)
  -> latest PipelineSnapshot(version=N) exists: reuse
  -> otherwise: materialize one immutable Snapshot(version=N)
  -> PipelineRun.snapshot_id references that Snapshot
```

Snapshot 包含 Application Pipeline 的完整 Stage 定义、DAG、变量声明、Repository ID、可选 Application ID、来源 Template ID/版本、来源 Version 策略及 Artifact Component 映射。Run 永远以 `snapshot_id` 重建执行输入，不从当前 Pipeline、Template 或 Stage 表读取可变配置。Template 不得拥有 `PipelineSnapshot` 记录；对 Template 调用 Snapshot 创建或查询接口必须被拒绝。

## Execution design

### Snapshot and Run

`PipelineSnapshot` 改为只关联 Application Pipeline 的 `pipeline_id`，保存该 Pipeline `version` 的完整 Stage 定义、变量声明、Repository 身份、可选的 Application 身份、来源 Template ID/版本、来源 Version 策略及 Artifact Component 映射。它保留现有的“当前版本复用、版本变化新建、Run 不可变引用”业务和数据职责。

手动触发和 Retry 都调用同一个 `CreatePipelineRun` 应用服务：

```text
load application Pipeline
  -> validate Pipeline/Repository Project ownership
  -> get or create PipelineSnapshot
  -> if Snapshot contains Component-bound Artifacts, validate Application ownership and resolve one source Version
  -> persist PipelineRun + PipelineRunVersionBinding atomically
  -> dispatch background task
```

`PipelineRun` 保存 `pipeline_id`、`pipeline_name`、`pipeline_version` 和 `snapshot_id`，不再保存 `template_id`、`template_name`、`template_version`。Artifact 同样保存 Pipeline 身份。Snapshot 中的来源 Template ID/版本和 Pipeline 的 `source_pipeline_id` 用于追溯，不参与运行选择。

快照中存在至少一个 Component-bound Artifact 时，Run 创建时按 Pipeline 的来源 Version 策略解析一个来源 Version：

- `latest` 使用该 Application 当前 `id DESC` 的最后创建 Version；
- `fixed` 使用 Pipeline 中保存的 Version；
- 来源 Version 和全部目标 Component 在 Run 创建时校验并冻结到一条 `PipelineRunVersionBinding`；
- Retry 创建新 Run 并按当时配置重新解析 `latest`，已有 Run 永不被回写。

### Artifact and Version fork

现有 Artifact collector 与 Version fork 语义保留，但 Component 映射来源从 Stage 全局配置改为 Pipeline Snapshot 中的 Artifact 声明：

1. 每个 Stage 成功后收集 `docker_image` Artifact 的镜像引用和本地 SHA，并从唯一传递前置 `command/git_object_id` Artifact 读取实际 source commit；
2. 写入全部 Artifact，但尚不 fork Version；
3. 全部 Stage 成功后，收集快照中所有 Component-bound Artifact，并确认每个映射恰有一个成功镜像制品；
4. 从冻结来源 Version fork 一个未发布 Version，在同一事务中替换所有目标 Component 的镜像并写入各自的 Artifact ID；
5. 回写唯一的 `PipelineRunVersionBinding.generated_version_id`。

`PipelineRunVersionBinding` 继续是历史事实，包含 Application、来源 Version 和生成 Version。Artifact 与 Component 的历史关系由 fork 后 Version Component 的 `artifact_id` 表达；它们都不依赖当前 Pipeline 或 Stage 的可变配置，且不因为 Version 删除而失效。

### Runtime variable preview

`POST /api/pipeline/{pipeline_id}/variable-preview` 是 Application Pipeline 运行时变量的唯一预览入口。它以 Pipeline 固定的 Repository、当前 `trigger_ref` 和用户提供的可覆盖变量为输入，复用 Run 创建时的变量声明解析、Repository 覆盖、Pipeline 自定义值和用户覆盖优先级，返回已解析的变量声明和值。

预览不会创建或复用 `PipelineSnapshot`，不会创建 `PipelineRun`、后台任务或 `PipelineRunVersionBinding`，也不会解析来源 Version 或执行制品绑定。它只校验 Application Pipeline 身份、Repository 可用性与 ref 规范化，并返回可用于界面展示的解析结果。详情页以空 ref 和空覆盖请求默认值；运行对话框则在 ref 或覆盖变化后以当前值重新请求，最终 Trigger 仍独立走 `CreatePipelineRun`。

## External interfaces

旧 Template 运行和 Repository 路径触发接口删除。新接口仅接受 Application Pipeline：

| 用途 | 接口 | 关键输入 |
| --- | --- | --- |
| 管理 Pipeline | `GET/POST /api/pipeline`、`GET/PUT/DELETE /api/pipeline/{pipeline_id}` | `kind`；Application Pipeline 不允许直接创建 |
| 创建 Template 实例 | `POST /api/pipeline/{template_id}/instantiate` | 名称、Repository ID、可选 Application ID |
| 管理阶段 | `POST/PUT/DELETE /api/pipeline/{pipeline_id}/stage` | Stage 与 Artifact 声明；仅已绑定 Application 的 Pipeline 镜像制品可设置 Component 映射 |
| 变量预览 | `POST /api/pipeline/{pipeline_id}/variable-preview` | `trigger_ref`、可覆盖 `variables`；只返回解析值，不产生运行副作用 |
| 手动触发 | `POST /api/pipeline/{pipeline_id}/trigger` | `trigger_ref`、`variables` |
| 运行查询 | `GET /api/pipeline-run`、`GET /api/pipeline-run/{run_id}` | 按 `pipeline_id` 筛选 |

`PipelineRunTriggerReq` 不再含 `template_id` 或 `repository_id`。Webhook 不在本轮接口、领域模型和前端范围内；现有 Repository Webhook 代码不迁移到新 Pipeline 模型。

Proto、TypeScript 类型、MCP 只暴露上述新模型。没有旧字段别名、旧路由转发、兼容 DTO 或“无 Pipeline 时退回 Template”的默认行为。

## UI and workflow

模板列表只展示 `kind=template`，提供编辑蓝图和“创建应用流水线”动作；模板详情不显示运行、Application 选择或 Webhook。

应用流水线列表展示 `kind=application`，并显示固定的 Repository 与可选 Application。详情页包含：

- 不可编辑的来源 Template 与 Repository；Application 创建后若未绑定，可在基本信息中首次补绑，绑定后只读；
- 独立的 Stage 编排和变量编辑；绑定 Application 时才展示构建 Component/fork 策略；
- 仅 Application Pipeline 可用的运行按钮和 Run/Artifact 历史；
- 变量声明表显示同一变量解析器计算的默认运行时值，而不是只显示静态声明；
- 运行对话框只包含 ref 与可覆盖变量，初始与重算期间显示加载状态，预览失败时禁止提交。

Application Pipeline 创建过程按名称、Repository（必填）、Application（可选）的顺序配置；空 Application 不传递为空字符串绑定。未绑定时可在详情基本信息中首次补绑同一 Project 的 Application。随后构建 Stage 映射只在已绑定 Application 的 Pipeline 详情页配置。用户不需要复制或修改 Template Stage 来切换应用；已绑定后不可换绑，需要换应用时从 Template 重新创建。

列表将 Repository 与 Application 拆为独立列，未绑定 Application 明确显示“未绑定”。模板与 Application Pipeline 的每一行均提供删除动作，不再保留无实际语义的“配置”入口；删除确认或删除失败都在其模态窗中反馈，不能将服务端错误穿透为页面级 toast。

## Validation and integrity

- Template 不能触发、拥有 Application/Repository，且其镜像制品不得配置 Component 映射。
- Application Pipeline 必须绑定 Repository；Application 可为空。存在 Component 映射时，Application 必须存在且与 Pipeline、Repository 同属一个 Project。
- Component-bound Artifact、快照创建和 Run 创建都校验唯一上游 `git_object_id` Artifact、目标 Component 的全 Pipeline 唯一性和 Pipeline 来源 Version 策略。
- `variable-preview` 与 Run 创建复用相同的运行时变量解析和覆盖优先级，但其只读语义不得因预览请求创建 Snapshot、Run、任务或 Version 绑定。
- 同一 Repository 的 `running` Run 约束改为通过 Pipeline 的 Repository 执行，保持现有“仅阻止 running、不阻止 waiting_to_run”的产品规则。
- Version 删除检查从旧的全局 Stage 绑定替换为 Application Pipeline 的 `fixed_version_id` 引用；Service、Deployment 和历史 Run 的现有语义保持。
- Application Pipeline 的身份资源不可修改；Application、Repository 或来源 Template 更换必须创建新 Pipeline，避免运行历史与配置含义漂移。

## Database baseline

数据库改造不进入业务 usecase、repository 兼容逻辑或 HTTP handler。空库从 version 1 顺序执行至 version 30 时，必须直接得到新的 `pipeline`、自有 `pipeline_stage`、`pipeline_snapshot`、`pipeline_run`、Run 级 Version 绑定、Artifact 和 VersionComponent 字段；不创建旧 Template/全局 Stage/Webhook 表，也不提供旧库转换路径。

`pipeline_snapshot.stages_snapshot`、`variables_snapshot` 及 Run 对 Snapshot 的不可变引用语义继续保留。业务代码仅依赖最终结构，不提供运行时降级或旧数据补偿。

## Alternatives rejected

1. 保留全局 `PipelineStage`，只把 Application ID 移到 Template/Pipeline 关联：拒绝。Stage 定义仍是跨流水线的可变共享状态，Template 不是独立来源蓝图。
2. Application Pipeline 只绑定 Application、不绑定 Repository：拒绝。手动运行仍需选择代码来源，未形成真正可直接执行的交付流水线。
3. Application Pipeline 实时继承 Template：拒绝。Template 的一次编辑会在下一次运行时隐式改变多个应用的交付内容，无法形成明确变更边界。
4. 每个 Component-bound Artifact 独立 fork Version：拒绝。一个 Run 的多个镜像制品必须共同组成一个 Application Version，不能产生相互覆盖的零散 Version。
5. 在本轮重建 Webhook：拒绝。Webhook 尚未正式投入使用，先从新模型中移除并延后设计。
6. 在业务层保留旧表读取或 Template trigger 回退：拒绝。它违背本任务明确的破坏性切换边界，并会让触发和制品绑定再次产生两套语义。

## Technical questions

暂无阻塞规格的技术问题。独立数据库就地迁移脚本的具体语句、数据审计与 CI 调度属于后续独立任务。

## Risks

1. 这是贯穿 SQL、Proto、HTTP、后台任务和 Web 的破坏性重构；所有入口必须在同一个切换版本更新，不能灰度混用旧模型。
2. 一个绑定制品缺失、失败或无法解析 source commit 时，整个 Run 不得生成 Version；已经记录的通用 Artifact 保留，但不能产生部分 Component 更新的 Version。
3. Pipeline 与 Repository 的一对一绑定提升了运行确定性，但同一应用需要多个源码或分支策略时，应创建多个 Application Pipeline，而不是在运行时切换 Repository。
