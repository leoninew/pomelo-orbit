# CI Pipeline 设计文档
最后修改时间: 2026-09-07 22:04:24

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
  ├── application_id + 名称快照（模板含 Docker 制品时必填）  目标 Application
  ├── repository_id + 名称快照                  固定的源码 Repository
  ├── version_fork_strategy                     latest | fixed（模板含 Docker 制品时）
  └── PipelineStage(kind=application)[]         从引用快照物化的独立执行节点
```

阶段库只定义通用执行步骤；`PipelineStage(kind=template)` 不保存 DAG、排序、Application、Component 或来源 Version 策略，但保存不带 `component_name` 的制品声明。Template Pipeline 通过 `PipelineStageReference` 保存阶段引入时的来源 ID/名称/版本/说明、镜像、脚本和制品声明快照，以及自己的节点名称、说明、DAG 和排序。

Application Pipeline 必须从同项目 Template 创建。实例化只读取 Template Pipeline 已关联的引用快照，重映射引用节点 ID 到新的应用阶段 ID，复制制品声明。模板含 Docker 制品时，必须在同一请求中选择 Application、来源 Version 策略，并把每个 Docker 制品绑定到唯一的 Component；它不会重新读取可变阶段库。Template 或阶段模板后续变更、删除都不会影响已创建的 Application Pipeline。

## 阶段与制品

`PipelineStage(kind=application)` 归属于单个 Application Pipeline，包含私有名称、执行镜像、脚本、制品声明、`depends_on`、排序和说明，并强制保存非空的来源模板阶段 ID、名称、已应用版本和说明快照。依赖只能引用同一 Pipeline 的本地节点。

Template Pipeline 的 `PipelineStageReference` 与 Application Stage 都可对来源模板版本执行显式更新。普通保存不会同步模板；当来源模板存在更高版本时，Pipeline 详情的构建阶段列表会在阶段名称后展示“有更新”标签，悬停可查看实际变更字段及当前值到模板值的差异，卡片右上角提供“更新”命令。更新前预览来源名称、镜像、脚本、制品声明和模板说明差异。引用会直接替换制品声明；应用阶段按制品名称保留仍有效的 Docker 组件映射，并保留私有名称、说明、DAG、排序和 Pipeline 的来源 Version 策略。

`ArtifactConfig` 的 `component_name` 只允许用于 `docker_image`：

- 模板阶段和 Template Pipeline 引用保存无 `component_name` 的制品声明。
- Application Pipeline 在实例化时复制声明；每个 Docker 制品必须映射到一个 Application Component。
- Docker 制品的 `component_name` 表示成功 Run 要更新的 Application Component；同一 Pipeline 中 Component 不可重复。
- 每个组件映射镜像必须经由该 Stage 的传递依赖恰好关联一个 `command/git_object_id` 制品，作为 source commit。

来源 Version 策略归 Pipeline 所有：`latest` 在 Run 创建时读取该 Application 的最新 Version，`fixed` 固定一个该 Application 的 Version。模板含 Docker 制品时，它与制品组件映射在 Template Pipeline 实例化为 Application Pipeline 时一并保存，避免出现不能运行的中间配置。

## 阶段变量

Stage 的 `script` 以及 Artifact 的 `reference`、`name`、`command` 使用同一套 Liquid 变量语法：`{{ NAME }}` 是必填变量，`{{ NAME | default: "value" }}` 是当前位置的阶段默认值。`${NAME:-value}` 等 shell 风格默认表达式不受支持。

Repository 的 `value/default` 是全局覆盖；Application Pipeline 可以保存带 `stage_id` 的 Stage 覆盖，也可以保存不带 `stage_id` 的全局覆盖。缺少覆盖时，每个 Stage 独立应用自身 Liquid default。变量详情按 `(name, stage_id)` 展示，因此前端和后端 Stage 可以共用 `working_dir` 但分别默认 `web` 与 `webapi`，并可分别设置值。

变量配置的 `value` 还可由简单 `{{ NAME }}` 引用和文本拼接组成，并在 Run 创建前递归展开；例如 `image_repository = {{ repository_code }}-web`。这不是阶段字段的完整 Liquid 模板：变量 `value` 不接受 filter、tag、条件、循环、dotted path 或 Shell 插值，`default` 必须是无 Liquid 标记的字面量。保存时检查未知引用和循环，Stage 值只可读取同 Stage 和全局可见值。Liquid 的渲染、引用提取及嵌套解析均由通用模板模块负责，Pipeline 仅提供来源优先级与作用域。

## Snapshot、Run 与 Version

只有 `kind=application` 的 Pipeline 会在运行前按 Pipeline 版本创建或复用 Snapshot。Snapshot 冻结完整阶段定义、创建时可获得的 Repository/Pipeline/Stage 变量声明、来源 Template、Repository，以及可选的 Application 和 Version 策略，并保存 runtime/system 的声明元数据和阶段默认值来源。Snapshot 的变量声明仅用于历史展示与追溯；变量在每次 Run 和 Retry 时由当前 Repository、当前 Pipeline 配置及冻结阶段定义解析。全局有效值和带 `stage_id` 的 Stage 有效值分别保存到 Run，阶段默认值保留在变量元数据并由冻结 Stage 文本独立渲染，因此不存在多阶段同名变量被压缩为一个 Run 全局值的情况。手动运行会打开分支或标签弹窗，默认读取绑定 Repository 的 `repository_ref`，并允许用户在触发前修改。Template 有自己的 `version` 用于来源追溯，但不拥有 Snapshot。

```text
应用流水线
  -> 按 pipeline.version 创建/复用 Snapshot
  -> 解析并冻结一次来源 Version（有组件映射时）
  -> 原子创建 PipelineRun + PipelineRunVersionBinding
  -> 异步执行 Stage DAG、收集 Artifact
  -> 所有 Stage 成功后，只 fork 一次 Version 并写入全部映射 Component
```

Retry 与手动触发共享 Run 创建路径。Retry 创建新 Run；`latest` 会按重试时刻重新解析，历史 Run 不被回写。

## Runtime workspace

Pipeline checkout、Stage log 和 Run artifact 都位于当前 Project `Environment.workspace_root/pipeline`。`workspace.root` 只作为启动配置基准，实际执行按 Project Environment 解析；配置值表示 Orbit 进程可见路径。Pipeline 容器的 `/workspace` 与 `/artifacts` bind mount 会解析为 Docker daemon 可见的宿主路径。因此 DooD 部署必须将该工作区根目录显式挂入 Orbit 容器，不能把容器内路径直接传给 Docker。SSH Environment 的远端 workspace 仅用于 CD，不能作为控制面 Pipeline 的本地 bind source。

## Git 凭据

远程仓库的 `git clone` 阶段只接受 HTTPS URL，不识别托管平台。Stage 脚本包含仓库 URL 且仓库绑定了凭据时，执行器用 `GIT_CONFIG ... insteadOf` 把明文 URL 改写成认证 URL，不把 token 写入脚本。

| 类型 | 凭据内容 | 执行时改写 |
|---|---|---|
| `github_token` | token | `https://<token>@host/owner/repo.git` |
| `gitee_token` | `username:token` | `https://<username>:<token>@host/owner/repo.git` |
| `gitea_token` | Gitea 登录用户名:`token` | 与 `gitee_token` 相同 |
| `git_ssh` | 私钥 | 可保存，流水线执行拒绝 |

`gitea_token` 来自 Gitea 用户设置 → Applications → Manage Access Tokens，不是同页 OAuth2 Application，也不是仓库 Deploy Key。非 `https://` 的仓库 URL 执行失败。公开 HTTPS 仓库可以不绑定凭据。

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
