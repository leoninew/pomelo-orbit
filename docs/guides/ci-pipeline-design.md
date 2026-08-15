# CI Pipeline 设计文档
最后修改时间: 2026-08-15 14:30:51

Doc role: living guide。与代码冲突时以代码为准。

## 产品模型

`Pipeline` 是唯一的流水线聚合根：

```text
PipelineStage(kind=template, project scoped)    可复用阶段定义
  └── name / image / script / artifacts / description / version

Pipeline(kind=template)                         只作为来源，不可运行
  └── PipelineStageReference[]                  冻结的阶段引用与 DAG 节点

Pipeline(kind=application)                      可运行的交付单元
  ├── source_pipeline_id + 名称/版本快照        来源 Template
  ├── application_id + 名称快照（可选）          目标 Application（仅组件映射时必填）
  ├── repository_id + 名称快照                  固定的源码 Repository
  ├── version_fork_strategy                     latest | fixed（仅有组件映射时）
  └── PipelineStage(kind=application)[]         从引用快照物化的独立执行节点
```

阶段库只定义通用执行步骤；`PipelineStage(kind=template)` 不保存 DAG、排序、Application、Component 或来源 Version 策略，但保存不带 `component_name` 的制品声明。Template Pipeline 通过 `PipelineStageReference` 保存阶段引入时的来源 ID/名称/版本/说明、镜像、脚本和制品声明快照，以及自己的节点名称、说明、DAG 和排序。

Application Pipeline 必须从同项目 Template 创建。实例化只读取 Template Pipeline 已关联的引用快照，重映射引用节点 ID 到新的应用阶段 ID，复制制品声明，并在同一请求中选择 Application、来源 Version 策略及 Docker 制品到 Component 的绑定；它不会重新读取可变阶段库。Template 或阶段模板后续变更、删除都不会影响已创建的 Application Pipeline。

## 阶段与制品

`PipelineStage(kind=application)` 归属于单个 Application Pipeline，包含私有名称、执行镜像、脚本、制品声明、`depends_on`、排序和说明，并强制保存非空的来源模板阶段 ID、名称、已应用版本和说明快照。依赖只能引用同一 Pipeline 的本地节点。

Template Pipeline 的 `PipelineStageReference` 与 Application Stage 都可对来源模板版本执行显式更新。普通保存不会同步模板；当来源模板存在更高版本时，Pipeline 详情的构建阶段列表会在阶段名称后展示“有更新”标签，悬停可查看实际变更字段及当前值到模板值的差异，卡片右上角提供“更新”命令。更新前预览来源名称、镜像、脚本、制品声明和模板说明差异。引用会直接替换制品声明；应用阶段按制品名称保留仍有效的 Docker 组件映射，并保留私有名称、说明、DAG、排序和 Pipeline 的来源 Version 策略。

`ArtifactConfig` 的 `component_name` 只允许用于 `docker_image`：

- 模板阶段和 Template Pipeline 引用保存无 `component_name` 的制品声明。
- Application Pipeline 在实例化时复制声明；未映射的制品只保留构建追溯。
- Docker 制品的非空 `component_name` 表示成功 Run 要更新的 Application Component；同一 Pipeline 中 Component 不可重复。
- 每个组件映射镜像必须经由该 Stage 的传递依赖恰好关联一个 `command/git_object_id` 制品，作为 source commit。

来源 Version 策略归 Pipeline 所有：`latest` 在 Run 创建时读取该 Application 的最新 Version，`fixed` 固定一个该 Application 的 Version。它与制品组件映射在 Template Pipeline 实例化为 Application Pipeline 时一并保存，避免出现不能运行的中间配置。

## Snapshot、Run 与 Version

只有 `kind=application` 的 Pipeline 会在运行前按 Pipeline 版本创建或复用 Snapshot。Snapshot 冻结完整阶段定义、创建时可获得的 Repository/Pipeline/Stage 变量声明、来源 Template、Repository，以及可选的 Application 和 Version 策略，并保存 runtime/system 的声明元数据。Snapshot 的变量声明仅用于历史展示与追溯；变量在每次 Run 和 Retry 时由当前 Repository、当前 Pipeline 配置及冻结阶段定义解析，最终执行值只保存到 Run。手动运行没有变量预览或表单，`repository_ref` 使用绑定 Repository 的默认分支。Template 有自己的 `version` 用于来源追溯，但不拥有 Snapshot。

```text
应用流水线
  -> 按 pipeline.version 创建/复用 Snapshot
  -> 解析并冻结一次来源 Version（有组件映射时）
  -> 原子创建 PipelineRun + PipelineRunVersionBinding
  -> 异步执行 Stage DAG、收集 Artifact
  -> 所有 Stage 成功后，只 fork 一次 Version 并写入全部映射 Component
```

Retry 与手动触发共享 Run 创建路径。Retry 创建新 Run；`latest` 会按重试时刻重新解析，历史 Run 不被回写。

`PipelineRun` 和 `Artifact` 保存 `pipeline_id`、名称和版本快照。`PipelineRunVersionBinding` 保存 Application、来源 Version 与生成 Version 的 ID 和展示标签。Version Component 写入 Artifact ID 及镜像、SHA、source commit 等展示快照。

## 引用与删除

跨生命周期关系是逻辑外键：存稳定 ID，也存删除目标后仍需展示的名称、标签或版本。Template、Application、Repository、Version 与 Pipeline 允许物理删除；历史 Snapshot、Run、Artifact 和 Version Component 不需要回写或置空。

只有聚合内部组成关系使用物理外键：Template Pipeline -> PipelineStageReference、Application Pipeline -> PipelineStage、PipelineSnapshot -> PipelineRun，PipelineRun -> Artifact。阶段来源使用逻辑快照引用，允许删除模板阶段后继续使用已保存的引用快照。空库通过迁移链至 version 30 直接创建该模型；旧阶段配置由独立离线脚本处置，业务代码不保留兼容路径。

## HTTP API

| 端点 | 用途 |
|---|---|
| `GET/POST /api/pipeline` | 查询 Pipeline；只能创建 Template |
| `GET/PUT/DELETE /api/pipeline/:pipeline_id` | Pipeline 详情、更新、物理删除 |
| `POST /api/pipeline/:pipeline_id/instantiate` | 用 Template 创建 Application Pipeline |
| `GET/POST /api/pipeline-stage` | 项目内阶段库查询与创建 |
| `GET/PUT/DELETE /api/pipeline-stage/:stage_id` | 阶段库详情、更新与删除 |
| `POST /api/pipeline/:pipeline_id/stage` | 从阶段库引入节点 |
| `PUT/DELETE /api/pipeline/:pipeline_id/stage/:stage_id` | 更新/删除 Pipeline 节点 |
| `GET/POST /api/pipeline/:pipeline_id/stage/:stage_id/template-update-preview|template-update` | 预览并显式应用模板阶段更新 |
| `GET /api/pipeline/snapshot/:snapshot_id` | Application Pipeline 的不可变快照 |
| `POST /api/pipeline/:pipeline_id/trigger` | 运行 Application Pipeline |
| `GET /api/pipeline-run` | 按 Repository 或 Pipeline 查询 Run |
| `POST /api/pipeline-run/:run_id/retry` | 从历史 Run 重试 |
| `GET /api/pipeline-run/artifact` | 按 Repository 或 Pipeline 查询制品 |

Webhook 尚未投入使用，本模型不定义 Repository Webhook、公开 Webhook 接收入口或自动触发行为。
