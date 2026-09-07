# CI 制品关联应用版本规格
最后修改时间: 2026-08-05 17:37:49

Review status: Accepted

流程模式: 标准 / standard

## Requirement basis

- 依据 [CI 制品关联应用版本 Requirement](../requirement/20260805-ci-artifact-version-fork.md)，该需求已接受。
- 范围是本地构建镜像制品、Application Version fork 和制品血缘；不涉及 Service 部署、registry 或 MCP 行为。

## Overview

将可选的 `BuildVersionBinding` 加入 PipelineStage。一个有绑定的构建阶段只产生一个 `docker_image` 制品，并指定要替换的 Version Component 名称。

PipelineStage 的制品声明统一采用 collector 模型。PipelineRun 创建时从冻结的阶段绑定解析来源 Version：`latest` 选该 Application ULID 最大的 Version，`fixed` 使用配置中的 Version。解析后的来源 Version ID 存入该 Run 的绑定记录。阶段成功后，系统采集声明的制品值或位置；镜像 collector 查询本地镜像并通过 `source_artifact_id` 取得 Git object ID，随后 fork 来源 Version、替换目标 Component 的镜像，并将新 Component 与制品关联。

新 Version 的 label 采用 `build-<runtime_datetime>`。它只用于展示；所有关系均使用资源 ID。

## Design decisions

### BuildVersionBinding

`PipelineStage` 增加可选的构建版本绑定：

```text
application_id
component_name
fork_strategy: latest | fixed
fixed_version_id: nullable
```

- 未配置绑定的阶段保持现有行为。
- 配置绑定的阶段必须声明且只能声明一个 `docker_image` 制品。
- `application_id` 必须与 PipelineStage 属于同一 Project。
- `component_name` 是 fork 后 Version 中要替换镜像的 Component 名称。
- `latest` 不允许设置 `fixed_version_id`；`fixed` 必须设置它，且该 Version 属于 `application_id`。

首版采用“一阶段、一个镜像、一个 Component”的映射。Version 可以包含多个 Component；fork 后只替换名称匹配的目标 Component，其余 Component 及其运行配置由现有 fork 逻辑复制。一个阶段产出多个镜像以及一个镜像映射多个 Component 不在首版范围内。

### Snapshot and source resolution

阶段绑定属于 StageDefinition，必须随 PipelineSnapshot 冻结。后续修改 PipelineStage 的 Application、策略或目标 Component 不影响已有 PipelineRun。

在创建 PipelineRun 时，为 Snapshot 中每个绑定建立一条 Run 级记录：

```text
pipeline_run_id
pipeline_stage_id
application_id
component_name
source_version_id
generated_version_id: nullable
artifact_id: nullable
```

该记录以 `(pipeline_run_id, pipeline_stage_id)` 唯一。`latest` 以 `id DESC` 选择该 Application 最后创建的 Version。项目使用 `ulid.Make()` 生成 Version ID；ULID 的文本排序按毫秒时间排序，同一毫秒使用单调熵排序，因此可直接作为本地单进程场景下的创建顺序。该查询不按 label、SemVer 或发布状态过滤。

来源 Version 在 PipelineRun 创建时即固定。若 `latest` 对应的 Application 没有 Version，或来源 Version 中不存在 `component_name`，触发失败且不创建 PipelineRun。`fixed` 在 Stage 配置保存时和 Run 创建时都重新校验其 Version 与 Component。

### Artifact collectors and source commit

制品声明统一为 `name`、`collector`、`reference`、`command` 与 `format`：

- `file`：`reference` 为工作区制品文件路径，成功后记录其实际位置。
- `command`：在阶段脚本成功后，以相同镜像、环境和挂载执行 `command`，将去除首尾空白的标准输出存入 `artifact.value`；格式为 `text` 或 `git_object_id`。
- `docker_image`：`reference` 为渲染后的本地镜像引用，成功后通过 Docker inspect 读取本地 image SHA。

Git clone 阶段固定声明 `source_commit` 为 `command` collector，命令为 `git rev-parse HEAD`，格式为 `git_object_id`。命令输出必须是 40 或 64 位十六进制 Git object ID。关联 Application 的构建阶段必须在 DAG 中传递依赖唯一的该类型上游制品；不使用 `trigger_ref` 回退，因为远程仓库的该字段可能是分支名。

### Image artifact normalization

`artifact` 保存稳定的 Run/Stage/名称身份、collector 与 collector 专属载荷：

```text
file
  location
command
  value
  value_format
docker_image
  image_ref                   -- 本地 Docker tag
  local_image_sha256          -- docker image inspect 得到的本地 image ID
  source_artifact_id          -- 唯一的 git_object_id Artifact
```

镜像到 Git commit 的追溯通过 `source_artifact_id` 自关联，而不是复制 commit 字段。新制品不再使用 `type`、`path` 或 `/artifacts/source_commit` 约定；基线 schema 与 seed 数据原地更新，不保留运行时兼容分支。已有数据库另行迁移。

镜像构建命令成功后，服务端必须以 `image_ref` 查询本机 Docker，取得 `local_image_sha256`，并通过 `source_artifact_id` 解析 Git object ID。无法获得其中任一值时，该绑定阶段的制品保存失败，不能创建 Version。镜像 tag 允许随后被覆盖，SHA 仅作为新 Version 创建时的来源记录，不作为部署前置校验。

### Repository running task assertion

系统不支持同一 Repository 的并发构建。触发和重试 PipelineRun 时，查询该 `repository_id` 是否已存在状态为 `running` 的 PipelineRun；存在时以业务校验错误拒绝创建新的 Run。`waiting_to_run` 不阻塞后续请求。

这是业务断言，不建立活动运行锁、lock table、lease 或其他跨 worker 的互斥机制。它的范围是阻止在已观察到运行中任务时再发起构建。

### Fork and association transaction

绑定阶段的制品保存包含一次数据库事务：

1. 创建包含镜像 metadata 与 `source_artifact_id` 的 Artifact。
2. 从 Run 已保存的 `source_version_id` fork 一个未发布 Version。
3. 在新 Version 中定位 `component_name`，将其 `image` 更新为 `image_ref`。
4. 在新 Version Component 的 `artifact_id` 写入 Artifact。
5. 回写 Run 级记录中的 `artifact_id` 和 `generated_version_id`。

事务失败时不保留任何上述数据库记录；本地 Docker 镜像不回滚。因 `(pipeline_run_id, pipeline_stage_id)` 唯一约束，重试不会为同一个成功的阶段重复 fork Version；已有成功结果直接返回。

### Version reference integrity

删除 Version 前必须检查会影响运行资源配置的入向引用，至少包括：

- Service 和 Deployment；
- `fixed` 阶段绑定；

Run 级 `source_version_id`、`generated_version_id` 与对应 label 是不可变历史快照，不建立到 Version 的外键，也不参与删除校验；删除 Version 后仍保留原值。存在子 Version 的 `created_from_version_id` 时，先将其清空再删除来源 Version。Component 的 `artifact_id` 随 Version Component 删除；通用 Artifact 保留，不作为反向删除 Version 的理由。

## Affected components

| 区域 | 预期职责 |
| --- | --- |
| PipelineStage / PipelineSnapshot | 保存和冻结 `BuildVersionBinding`。 |
| PipelineRun | 在创建时解析来源 Version，并持久化 Run 级绑定。 |
| Artifact | 保留完整制品记录及 collector 专属载荷。 |
| Application Version | fork 后更新目标 Component，并保存其 `artifact_id` 血缘。 |
| Version deletion | 扩展引用计数与删除约束。 |
| HTTP Proto / handler / Web | 暴露阶段绑定、镜像制品元数据和 Version Component 制品来源。 |

## Interfaces

PipelineStage HTTP/Proto 的创建、更新和详情响应增加可选 `BuildVersionBinding`：

```text
BuildVersionBinding {
  application_id: string
  component_name: string
  fork_strategy: latest | fixed
  fixed_version_id: optional string
}
```

PipelineStage Artifact 声明响应公开 collector 配置；Artifact 运行记录公开 `pipeline_stage_id`、`collector`、`location`、可选 `value`/`value_format`，镜像制品额外公开 `image_ref`、`local_image_sha256`、通过血缘投影的 `source_commit_sha` 和关联的 Version/Component ID。Version Component 详情增加其关联制品的只读来源信息。

不新增 CI 到 CD 的自动动作，不扩展 `pomelo-orbit-mcp`，也不要求 Pipeline MCP 获得写权限。

## Technical questions

暂无阻塞 Plan 的技术问题。`git_object_id` 去除首尾空白后接受 40 或 64 位十六进制 object ID。

## Risks

1. `latest` 故意继承 ULID 最大的 Version，包括先前构建产生的未发布 Version；串行 Run 的链路依赖于 Run 创建顺序。
2. 收紧 Version 删除行为会改变现有“清空 fork 血缘后删除”的行为，需要迁移与用户提示。
3. 构建脚本虽成功但 Docker 本地镜像或实际 commit 无法读取时，阶段将不能生成关联 Version；错误必须明确区分构建失败和制品归档失败。
4. 本地 tag 可变，因此 Version 的 `image` 表达的是运行标签；制品 SHA 只表达该 Version 创建时的镜像来源。
5. Repository 约束是状态查询的业务断言而非跨 worker 互斥；这是用户明确选择的实现范围。

## Alternatives

1. 在 Application 上增加 base/current Version：拒绝。现有模型没有此语义，且会把多 Service/多 Version 的选择隐式化。
2. 使用 Version label 或 SemVer 选择来源 Version：拒绝。label 可编辑，当前没有语义化版本契约。
3. 由构建阶段直接修改已有 Version：拒绝。必须保留现有 fork 血缘并生成未发布 Version。
4. 将多个镜像映射作为首版能力：延后。首版的单镜像单 Component 约束更容易验证和重试。

## User review notes

1. 用户确认 `latest` 基于最后创建的 Version，`fixed` 基于用户选择的已有 Version。
2. 用户确认 Version 删除必须断言没有引用。
3. 用户接受在 PipelineRun 创建时冻结 `latest` 的来源，并接受首版以 `component_name` 显式映射单个镜像制品。
4. 用户接受建议的非语义化构建 Version label 格式。
5. 用户提出 Git clone 阶段生成 commit SHA 制品供下游构建阶段使用；当前草稿据此设计。
6. 用户确认同一 Repository 不支持并发构建。
7. 用户询问 ULID 是否可用于排序；当前草稿采用 `id DESC`。
8. 用户明确并发约束只需断言不存在 `running` PipelineRun，`waiting_to_run` 不阻塞，且不需要活动运行锁或类似实现。
9. 用户拒绝以文件作为 Git commit 制品载体，采纳通用 collector、标量值与 Artifact 自关联血缘模型；不保留旧制品字段或运行时兼容路径。
