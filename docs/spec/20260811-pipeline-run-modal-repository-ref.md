# 流水线运行时变量解析与运行弹窗规格
最后修改时间: 2026-08-12 11:29:20

Review status: Accepted

Mode: strict

## Requirement Basis

本规格落实 [Requirement](../requirement/20260811-pipeline-run-modal-repository-ref.md) 已确认的运行时变量模型。改动跨 Proto/HTTP、应用用例、变量规则、持久化、执行器和 Vue 运行弹窗；不实施兼容字段、别名或新旧逻辑并存。

## Overview

运行变量分为可配置变量和固化系统上下文。运行弹窗始终编辑并提交完整的非 `system` 变量表单；后端在预览和创建 Run 时用同一解析规则生成完整变量声明。每个 Run 保存不可变的最终变量快照，执行器只读取此快照。

`PipelineSnapshot` 继续冻结阶段定义、DAG、制品模板、Pipeline 身份/版本及创建时可获得的完整变量声明。其变量声明包含 Repository、Pipeline 与 Stage 的配置值/默认值，并保存 `runtime`/`system` 的声明元数据。Snapshot 变量声明只服务于历史展示与追溯，不参与运行时值解析；用户表单值和本次生成的系统值只能保存到 Run。复用同一 Snapshot 时，变量必须按当前 Repository、当前 Application Pipeline、当前或冻结阶段定义和本次表单重新解析。

## Variable Resolution

### 声明装配

手动 Trigger 与 Retry 依次装配以下声明；Preview 不创建 Snapshot，使用当前 Pipeline 阶段定义替代第 1 步，其他规则相同：

| 顺序 | 输入 | 作用 |
|---|---|---|
| 1 | `PipelineSnapshot.stages_snapshot` | 从冻结阶段的脚本和制品模板提取 Liquid 变量及 `default`，产生 `pipeline_stage` 声明。 |
| 2 | 当前 Application Pipeline `variable_declarations` | 产生或覆盖同名 `pipeline` 声明。未被阶段引用的 Pipeline 声明也必须保留。 |
| 3 | 当前 Repository `variable_overrides` | 产生或覆盖同名 `repository` 声明。 |
| 4 | 运行时声明 | 注入唯一的 `repository_ref`，来源 `runtime`，初始默认值为 Repository `default_branch`。 |
| 5 | 系统声明 | 生成 Repository、Pipeline 和运行时间上下文，来源 `system`。 |

最终列表按变量名唯一。对于同名可配置变量，保留最高优先级来源的 `description`、`secret` 和 `default` 元数据；阶段的 Liquid `default` 只在高优先级声明没有 `value` 或 `default` 时补入，不能删掉 Repository/Pipeline 的未引用声明。`{{ IMAGE_TAG | default: "latest" }}` 必须产生或补全 `pipeline_stage` 的 `IMAGE_TAG` 默认值。

运行时 Preview 和 Run 快照对外只使用 `repository`、`pipeline`、`pipeline_stage`、`runtime`、`system` 五种 `source`。现有配置编辑接口中的 `repository_custom` 与 `pipeline_custom` 是配置层存储来源，不得泄漏为运行时解析结果。

### 值解析与可编辑性

可配置值按以下优先级取第一个非空值：

```text
提交表单中的 variables[name]
  -> Repository 声明的 value/default
  -> Pipeline 声明的 value/default
  -> Pipeline Stage Liquid default
```

`repository_ref` 只从 `variables.repository_ref` 取得；预览的空或缺失候选值用 Repository `default_branch` 初始化。Trigger 需要非空的 `repository_ref`，并要求所有最终可配置变量具有值。其余 `runtime`、`repository`、`pipeline` 与 `pipeline_stage` 变量均为 `editable=true`。

系统变量不参与上述覆盖链，均为 `editable=false`，由服务端生成：

| 名称 | 值来源 |
|---|---|
| `repository_code`、`repository_url` | 当前 Pipeline 绑定 Repository 的运行时地址与镜像标识。 |
| `runtime_datetime` | 本次预览的展示时间或 Run 创建时的 UTC 时间；Run 快照以创建 Run 时产生的值为准。 |

服务端拒绝 `variables` 中的任意 `system` 名称，避免静默丢弃错误输入。请求中未知变量同样返回 validation error，避免拼写错误被无声忽略。Repository 变量、Pipeline 变量、阶段变量和 `repository_ref` 都可被表单覆盖。

### Source Ref

`trigger_ref` 完全删除。`repository_ref` 是唯一源码 ref 输入，并保留用户提交的分支、标签或其他 Git ref 原文：

- 不调用本地目录 Repository 的 `ResolveRevision`，不替换成 SHA。
- 不新增 `repository_sha`。
- Git commit 追溯继续由现有 `command/git_object_id` 制品提供。
- 本地目录克隆仍可读取 `DockerHostPath` 挂载源码；这不是 ref 解析。

## API Contract

路由保持不变：`POST /api/pipeline/:pipeline_id/variable-preview` 与 `POST /api/pipeline/:pipeline_id/trigger`。

两个请求都只保留：

```json
{
  "variables": {
    "repository_ref": "main",
    "IMAGE_TAG": "20260812"
  }
}
```

- Preview 可接收空或部分 `variables` 以引导默认值；返回完整变量声明及已解析 `value`。响应不再返回顶层 `trigger_ref`。
- Trigger 必须接收运行弹窗可见的完整非 `system` 表单。服务端仍以本次解析结果做完整校验，不能信任客户端完整性。
- `PipelineRunResp` 的 `trigger_ref` 改为 `repository_ref`，字段语义和数据库列一致。
- HTTP handler 只负责 Proto/DTO 映射与错误写出；合并、校验和快照构造留在 `pipeline_run` application usecase 与 `pipelinevariable` rule 中。

Proto 请求/预览响应删除字段后采用最小字段布局：两种请求的 `variables` 使用 tag `1`，预览响应的 `variable_declarations` 使用 tag `1`；`PipelineRunResp` 直接将原 tag `10` 改名为 `repository_ref`。该服务处于活跃开发期，前后端需同版本发布。

## Persistence And Snapshot Boundary

### Pipeline Snapshot

保留 `PipelineSnapshot.variables_snapshot`，它在创建 Snapshot 时保存所有可快照化的完整声明：Repository、Pipeline、从当时阶段定义提取的 Stage 变量的来源、配置值和默认值，以及 runtime/system 声明元数据。它用于 Snapshot API 的历史展示和追溯，不能被 Preview、Trigger、Retry 或 Execute 用作变量解析来源。由于一个 Snapshot 可被多个 Run 复用，用户表单值和每次生成的 system 值不写入该 Snapshot，只写入对应 Run。

- 不修改 `pipeline_snapshot` 的 schema、Model、Proto snapshot response、SQL 查询或 SQLC 映射。
- Preview 从当前 Pipeline 阶段定义提取阶段声明；Trigger/Retry 从 `stages_snapshot` 提取阶段声明；三者都从当前 Pipeline 与 Repository 配置和值以及本次表单解析运行变量。

### Pipeline Run

`PipelineRun.trigger_ref` 和 `pipeline_run.trigger_ref` 重命名为 `repository_ref`。新增 SQLite/MySQL 成对迁移完成列重命名；查询、SQLC、Model、Repository、DTO、Proto、HTTP mapper 和前端生成类型同步改名。

`PipelineRun.variables_snapshot` 保存该次 Run 的完整最终声明列表，包括所有可配置变量及 `system` 声明，且每项的 `value` 都是执行时使用的值。它是唯一的变量执行输入，也是历史展示来源。Run `repository_ref` 必须等于该快照中 `repository_ref.value`。

## Run Flows

### Variable Preview

1. 校验调用者和可运行的 Application Pipeline，读取其当前 Repository、Pipeline 配置与当前阶段定义。
2. 装配声明并以候选 `variables` 解析值；缺失 `repository_ref` 使用 `default_branch`，系统值仅由服务端生成。
3. 返回完整、唯一的变量声明；不创建 Snapshot、Run、后台任务或 Version binding。

### Manual Trigger

1. 校验 Pipeline、Repository 与并发运行限制，取得或创建结构 Snapshot。
2. 用当前 Repository/Pipeline 配置、Snapshot 阶段和提交的完整非系统表单解析最终变量；拒绝系统键、未知键、空 `repository_ref` 及缺失必填值。
3. 创建 Run、Stage Runs、Version binding 和调度任务时原子持久化 `repository_ref` 与完整 `variables_snapshot`。
4. 排队执行；之后配置变更不影响此 Run。

### Execute

1. 加载 Run、其 Snapshot 和当前 Repository 的执行连接/路径信息。
2. 反序列化 `PipelineRun.variables_snapshot`，校验其 `repository_ref` 与 Run 列一致；不再调用运行时变量装配，也不读取可变 Repository/Pipeline 变量配置。
3. 用快照变量渲染冻结 `stages_snapshot`，并把变量传入 Stage Executor。

### Retry

1. 读取原 Run 的完整变量快照，取出全部非 `system` 项的 `value`，包含 `repository_ref`。
2. 以该集合作为新 Run 的完整表单；重新读取当前 Repository/Pipeline 配置，重新取得当前 Pipeline 结构 Snapshot，并重新解析完整变量快照。
3. 新 Run 记录 `retry_of`，原 Run 与其快照绝不回写。

## Frontend Run Dialog

运行弹窗以 Preview 返回的声明为单一表单模型：

- 删除独立的“分支或标签”控件和所有 `runForm.trigger_ref` 状态；将 `repository_ref` 作为普通但必填的 `runtime` 变量输入。
- 初始化 Preview 后，用返回的有效 `value` 填充所有非 `system` 字段；系统字段只读显示，且永不加入请求。
- 每次编辑任意非系统字段，提交当前完整非系统表单进行 debounce Preview。异步响应按请求序号防止旧响应覆盖新输入。
- 提交时发送完整非系统表单，不再只发送相对初值的 diff。Preview 加载中、预览失败或字段校验失败时禁用确认。
- `repository_ref` 为空时在该输入下显示“请输入分支或标签”；其他空值归属各自字段。每项字段错误使用 `aria-invalid`、错误边框及控件下方文本；编辑当前字段只清除该字段错误。网络、权限和无法归属单一字段的错误保留为表单级反馈。
- 关闭、取消或切换 Pipeline 时清理本模态实例的字段错误、预览计时器与过期请求状态。

## Affected Components

| 层 | 主要文件/区域 |
|---|---|
| Proto 与生成物 | `proto/orbit/v1/pipeline_run/pipeline_run.proto`、`proto/orbit/v1/pipeline/snapshot.proto`、`task proto` 生成的 Go/TS。 |
| 变量规则 | `internal/application/pipeline/rule/pipelinevariable/runtime.go` 及测试；拆分系统声明、声明合并、表单校验和 Run 快照序列化职责。 |
| Pipeline Snapshot | `internal/application/pipeline/usecase/snapshot_create.go` 与变量规则；保留 Snapshot 声明快照的历史展示，切断其与运行时解析的依赖。 |
| Run 用例与执行 | `internal/application/pipeline_run/usecase/service.go`、`execution.go`、DTO、测试。 |
| HTTP | `internal/api/http/handler/pipeline_run/*`；路由不变。 |
| Model 与持久化 | `internal/model/pipeline.go`、`sql/query/pipeline/*.sql`、`sql/query/pipeline_run/*.sql`、Repository、SQLC、两种数据库的新迁移。 |
| 前端 | `web/src/views/pipeline/PipelineDetail.vue`、生成 Proto 类型、按需要的运行详情展示。 |
| 活文档 | `docs/guides/ci-pipeline-design.md` 与 `docs/guides/ci-pipeline-vars-design.md` 同步为“Snapshot 保留声明快照用于追溯，Run 保存最终执行值”。 |

## Test Matrix

| 场景 | 预期 |
|---|---|
| 阶段 `{{ IMAGE_TAG | default: "latest" }}`，无 Repository/Pipeline 配置 | 产生唯一的 `pipeline_stage` 声明，值为 `latest`。 |
| Pipeline 声明未在阶段引用 | 仍出现在 Preview 和 Run 快照中。 |
| Repository、Pipeline、Stage 同名 | 值和声明元数据按提交、Repository、Pipeline、Stage 的优先级解析，最终仅一项。 |
| `repository_ref` 初始预览/手动输入/本地目录 | 初始为 `default_branch`；输入原样保存与执行，不解析为 SHA。 |
| 提交系统键、未知键、空 ref 或缺失变量 | 服务端 validation error；系统键不被静默忽略。 |
| 同 Snapshot 两次 Trigger，期间更新 Repository/Pipeline 变量 | 两次 Run 分别使用创建时解析值；后一次看到当前配置。 |
| 同 Snapshot 两次 Trigger，期间更新配置 | Snapshot 详情仍展示创建时的声明历史；后一次 Run 按当前配置重新解析，不能从 Snapshot 声明取值。 |
| 创建 Run 后修改配置 | 原 Run 的执行与详情仍使用它的 `variables_snapshot`。 |
| Retry | 使用原 Run 非系统表单重新解析，生成新 Run 快照并保留原 Run。 |
| 迁移后查询和创建 Run | `repository_ref` 替代所有 `trigger_ref` 持久化和响应路径。 |
| 弹窗交互 | 无独立 ref 输入；完整表单 Preview/Trigger；字段级错误、只读 system、关闭清理和过期 Preview 防护成立。 |

实施完成时按项目约束运行 `task proto`、`task sqlc`、Go 格式化/vet/test，以及前端 `yarn --cwd web lint:fix` 和 `yarn --cwd web typecheck`。

## Risks And Alternatives

- 该改动移除 Proto 字段和数据库列，必须前后端同版本部署。由于项目处于活跃开发期，不提供 `trigger_ref` 兼容层。
- Preview 与 Trigger 使用当前配置但 Run 执行使用 Run 快照，必须保持同一解析函数，避免预览与创建结果分叉。
- `runtime_datetime` 的 Preview 值只是展示结果；提交 Run 时重新生成，避免预览停留时间被误认为执行时间。
- Snapshot 声明快照必须明确标注为历史/追溯数据，防止未来代码重新把它当作运行时权威来源。

## User Review Notes

- 已确认删除 `trigger_ref`，并接受客户端提交完整非 `system` 表单。
- 已确认 `repository_ref` 不解析 SHA，不新增 `repository_sha`，Git commit 由 `command/git_object_id` 制品追溯。
- 已确认 Snapshot 复用不复用变量；变量在每次运行时重新解析，Retry 使用原 Run 的非系统表单重新创建 Run。
