# CI Pipeline 设计文档
最后修改时间: 2026-08-07 16:35:00

Doc role: living guide。与代码冲突时以代码为准。

## 产品模型

`Pipeline` 是唯一的流水线聚合根：

```text
Pipeline(kind=template)                         只作为来源，不可运行
  └── PipelineStage[]                           模板独占阶段

Pipeline(kind=application)                      可运行的交付单元
  ├── source_pipeline_id + 名称/版本快照        来源 Template
  ├── application_id + 名称快照（可选）          目标 Application（仅组件映射时必填）
  ├── repository_id + 名称快照                  固定的源码 Repository
  ├── version_fork_strategy                     latest | fixed（仅有组件映射时）
  └── PipelineStage[]                           从模板深复制的独立阶段
```

Template 只定义通用的步骤、DAG、变量和制品收集方式。它没有 Application、Repository、来源 Version 或 Component 绑定，也不能创建 Run 和 Snapshot。

Application Pipeline 必须从同项目 Template 创建。实例化会复制变量与全部阶段、重映射 Stage ID 的 DAG 依赖，并清空所有制品的 `component_name`。Template 后续变更不会影响已创建的 Application Pipeline。

## 阶段与制品

`PipelineStage` 归属于单个 Pipeline，包含：执行镜像、脚本、制品声明、`depends_on`、排序和说明。依赖只能引用同一 Pipeline 的 Stage。

`ArtifactConfig` 的 `component_name` 只允许用于 `docker_image`：

- Template 中必须为空。
- Application Pipeline 中为空时，制品只保留构建追溯。
- 非空时，表示成功 Run 要更新的 Application Component；同一 Pipeline 中 Component 不可重复。
- 每个组件映射镜像必须经由该 Stage 的传递依赖恰好关联一个 `command/git_object_id` 制品，作为 source commit。

来源 Version 策略归 Pipeline 所有：`latest` 在 Run 创建时读取该 Application 的最新 Version，`fixed` 固定一个该 Application 的 Version。制品组件映射与来源 Version 策略在同一个阶段更新事务内保存，避免出现不能运行的中间配置；删除最后一个组件映射时自动清除策略。

## Snapshot、Run 与 Version

只有 `kind=application` 的 Pipeline 会在运行前按 Pipeline 版本创建或复用 Snapshot。Snapshot 冻结完整阶段定义、变量、来源 Template、Repository，以及可选的 Application 和 Version 策略。Template 有自己的 `version` 用于来源追溯，但不拥有 Snapshot。

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

只有聚合内部组成关系使用物理外键：Pipeline -> PipelineStage，PipelineSnapshot -> PipelineRun，PipelineRun -> Artifact。空库通过迁移链至 version 30 直接创建该模型，不在业务代码中保留旧模型兼容路径。

## HTTP API

| 端点 | 用途 |
|---|---|
| `GET/POST /api/pipeline` | 查询 Pipeline；只能创建 Template |
| `GET/PUT/DELETE /api/pipeline/:pipeline_id` | Pipeline 详情、更新、物理删除 |
| `POST /api/pipeline/:pipeline_id/instantiate` | 用 Template 创建 Application Pipeline |
| `POST /api/pipeline/:pipeline_id/stage` | 新增 Pipeline 自有阶段 |
| `PUT/DELETE /api/pipeline/:pipeline_id/stage/:stage_id` | 更新/删除阶段与制品配置 |
| `GET /api/pipeline/snapshot/:snapshot_id` | Application Pipeline 的不可变快照 |
| `POST /api/pipeline/:pipeline_id/trigger` | 运行 Application Pipeline |
| `GET /api/pipeline-run` | 按 Repository 或 Pipeline 查询 Run |
| `POST /api/pipeline-run/:run_id/retry` | 从历史 Run 重试 |
| `GET /api/pipeline-run/artifact` | 按 Repository 或 Pipeline 查询制品 |

Webhook 尚未投入使用，本模型不定义 Repository Webhook、公开 Webhook 接收入口或自动触发行为。
