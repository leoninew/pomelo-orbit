# CI Pipeline 变量设计
最后修改时间: 2026-08-12 11:29:20

Doc role: living guide。与代码冲突时以代码为准。

## 归属

变量声明有明确来源。Template 的 Pipeline 声明在实例化时深复制到 Application Pipeline；之后两个对象独立演进。运行时还会合并仓库变量、当前 Application Pipeline 的变量和从冻结 Application Pipeline 阶段提取的变量。Snapshot 冻结本 Pipeline 版本的阶段定义、执行结构及当时可获得的 Repository/Pipeline/Stage 变量声明，并保存 runtime/system 声明元数据；该变量声明只用于历史展示与追溯，不读取当前 Template，也不用作后续运行的变量解析输入。

`default` 表示声明提供的回退值，`value` 是来源配置中已持久化的取值；`secret` 只影响展示与日志脱敏，不改变传递规则。最终运行时列表按变量名去重，来源和可编辑性由服务端解析。

| 来源 | 含义 | 运行弹窗 |
|---|---|---|
| `repository` | 仓库变量配置 | 可编辑 |
| `pipeline` | Application Pipeline 变量配置 | 可编辑 |
| `pipeline_stage` | 阶段脚本和制品模板提取的变量，可能含 Liquid `default` | 可编辑 |
| `runtime` | 本次运行生成但允许用户控制的上下文，例如 `repository_ref` | 可编辑 |
| `system` | 本次运行的结构性上下文 | 只读 |

## 运行时上下文

| 变量名 | 运行时来源 |
|---|---|
| `repository_code` / `repository_url` | Application Pipeline 已绑定的 Repository；`system`，不可改 |
| `repository_ref` | 本次运行的源码 ref；`runtime`，可改 |
| `runtime_datetime` | 本次 Run 创建时生成的 UTC 时间戳；`system`，不可改 |

不再存在 `template_id`、`template_name`、`template_version` 作为运行期选择输入。来源 Template 仅在 Pipeline/Snapshot 的追溯信息中保存。

## 运行时合并

可配置变量从高到低：

```text
本次提交的完整运行表单
  -> Repository 自定义变量
  -> Application Pipeline 自己的声明 value/default
  -> Stage 提取的默认值
```

系统上下文不参与覆盖链，始终由服务端生成。Trigger 与变量预览请求仅传递完整的非
`system` 变量表单；`repository_ref` 在 `variables` 中，是唯一的源码 ref 输入。请求
不得携带 `trigger_ref`；`system` 变量也不得出现在 `variables` 中。

手动运行的请求仅有：

```json
{ "variables": { "repository_ref": "main", "IMAGE_TAG": "20260807" } }
```

Repository、可选的 Application、目标 Component 与来源 Version 策略均在 Pipeline 配置中冻结，不能作为运行参数传入。Retry 与手动触发共用相同的变量解析和 Snapshot/Run 创建路径。

## 运行前预览

`POST /api/pipeline/{pipeline_id}/variable-preview` 接受与手动 Trigger 相同的完整非
`system` 变量表单。它使用同一条运行时合并链路返回完整变量声明及其解析后的
`value`，但不创建 Snapshot、PipelineRun、任务或 Version 绑定。

运行弹窗以预览结果填充全部可配置项，并提交完整的非 `system` 表单。`repository_ref` 是该表单中的唯一源码 ref 输入；服务端不会把它解析为 commit SHA。系统上下文可展示但必须只读且不提交。

## Snapshot 与历史

`PipelineSnapshot.variables_snapshot` 保存创建 Snapshot 时可获得的完整声明：Repository/Pipeline/Stage 的来源、配置值和默认值，以及 runtime/system 声明元数据，用于历史展示与追溯。Preview 从当前 Pipeline 阶段定义提取 Stage 声明；创建 Run 或 Retry 时，即使复用既有 Snapshot，也从其阶段定义提取 Stage 声明。三条路径都读取当前 Repository 与当前 Application Pipeline 变量配置，并与本次完整表单和系统上下文解析最终变量快照，保存到 `PipelineRun.variables_snapshot`；不得用 Snapshot 的变量声明或值替代上述实时解析。因为 Snapshot 可复用，用户表单值和本次生成的 system 值不写入 Snapshot。执行器必须使用 Run 快照；之后修改任何配置不影响已创建的 Run。Retry 以原 Run 的完整非 `system` 变量快照作为表单创建新 Run。删除配置资源不要求重写历史变量快照。
