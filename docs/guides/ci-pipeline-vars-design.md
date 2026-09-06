# CI Pipeline 变量设计
最后修改时间: 2026-09-06 15:12:00

Doc role: living guide。与代码冲突时以代码为准。

## 归属

Template Pipeline 的变量声明会在实例化时深复制到 Application Pipeline；之后两个对象独立演进。Template Pipeline 保留既有变量配置：内置变量和阶段声明只读，`pipeline_custom` 可新增、编辑、删除。Application Pipeline 详情在此基础上显示可管理变量和只读内置项：

| 管理来源 | 含义 | 操作 |
|---|---|---|
| `pipeline_custom` | 已持久化在当前 Application Pipeline 的变量 | 新增、编辑、删除 |
| `pipeline_stage` | 从阶段脚本和制品 `reference`、`name`、`command` 提取的非内置 Liquid 变量 | 覆盖为同名 Pipeline 配置 |
| `pipeline` | `repository_ref`、`repository_code`、`repository_url`、`runtime_datetime` | 只读 |

Repository 配置不显示也不能在 Pipeline 中覆盖。删除一个覆盖阶段变量的 `pipeline_custom` 配置后，该变量恢复为 `pipeline_stage`，继续使用阶段中声明的 Liquid `default`（若有）。变量名必须是非内置 Liquid 标识符；Pipeline 配置的唯一键是 `(name, stage_id)`，其中空 `stage_id` 表示全局配置。变量表按 Stage 来源展示独立条目；阶段默认值是派生展示数据，Stage 覆盖通过同一条目的 `stage_id` 提交。

Repository 变量只在 Repository 管理，不出现在 Application Pipeline 的管理列表，也保持更高优先级。`runtime` 和 `system` 上下文由服务端生成，不可在 Pipeline 管理。

`default` 是 Repository 或 Pipeline 配置的全局回退值，`value` 是持久化配置值，`secret` 只影响展示元数据，不改变变量解析顺序。

## 阶段 Liquid 变量

流水线 Stage 的 `script`、Artifact `reference`、`name`、`command` 统一使用 Liquid：

```liquid
{{ NAME }}
{{ NAME | default: "value" }}
```

`{{ NAME }}` 是必填引用。`{{ NAME | default: "value" }}` 在变量缺失或为空字符串时使用该位置的字面量默认值；默认字面量可使用单引号或双引号，不做二次模板展开。`${NAME:-value}`、`${NAME-value}`、`${NAME:=value}` 等 shell 风格默认语法不属于流水线变量规范，系统会拒绝包含这些写法的阶段模板。

阶段默认值属于表达式所在的 Stage，而不是同名变量的全局 `default`。例如前端和后端 Stage 分别使用 `{{ working_dir | default: "web" }}`、`{{ working_dir | default: "webapi" }}` 时，变量管理显示两个带 Stage 来源的 `working_dir` 条目；未配置覆盖时，两个 Stage 分别使用自己的目录。

## 运行时上下文

| 变量名 | 运行时来源 |
|---|---|
| `repository_code` / `repository_url` | Application Pipeline 绑定 Repository；`system`，由服务端注入 |
| `repository_ref` | 绑定 Repository 的 `default_branch`；`runtime`，由服务端注入 |
| `runtime_datetime` | 本次 Run 创建时生成的 UTC 时间戳；`system`，由服务端注入 |

来源 Template 只保存在 Pipeline/Snapshot 的追溯信息中；它不是运行期输入。

## 运行时合并

同名变量在每个表达式位置按以下顺序取第一个非空值：

```text
Repository 自定义变量的全局 value/default
  -> Application Pipeline 当前 Stage 的 value/default
  -> Application Pipeline 全局 value/default
  -> 当前 Stage 表达式的 Liquid default
```

Repository 值进入全局运行时变量；Pipeline 值根据是否带 `stage_id` 进入当前 Stage 或全局运行时变量；不同 Stage 的 Liquid default 不会被压缩为一个全局值。系统上下文不参与覆盖链。手动 Trigger 不再接受变量表单，`POST /api/pipeline/{pipeline_id}/trigger` 使用空请求体；服务端以当前 Repository、当前 Application Pipeline 配置和冻结阶段定义解析最终变量。仅使用阶段 default 的变量无需补齐 Pipeline 配置；某个表达式使用 `{{ NAME }}` 且没有更高优先级值时，Trigger 返回包含变量名和 Stage 的 validation error。

应用流水线运行没有变量预览 API、运行弹窗、配置表单或配置列表。

## Snapshot 与历史

`PipelineSnapshot.variables_snapshot` 保存创建 Snapshot 时可获得的 Repository/Pipeline/Stage 声明与 runtime/system 元数据，用于历史展示和追溯；它不作为之后新 Run 的变量取值来源。

创建 Run 或 Retry 时，都从当前 Repository、当前 Application Pipeline 配置和该次使用的冻结阶段定义解析最终变量，保存到 `PipelineRun.variables_snapshot`。全局有效值与带 `stage_id` 的 Stage 有效值分别保存到 Run；仅由 Stage default 提供的变量保留其阶段默认值元数据，并由冻结 Stage 文本在执行时独立渲染。执行器只使用 Run 快照和冻结 Stage；之后修改任何配置不影响已创建的 Run。Retry 以原 Run 的非 system 全局值构造内部 overrides，再创建新的 Run；该内部路径不对手动 Trigger 暴露变量表单。
