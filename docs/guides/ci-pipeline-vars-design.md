# CI Pipeline 变量设计
最后修改时间: 2026-08-15 13:57:20

Doc role: living guide。与代码冲突时以代码为准。

## 归属

Template Pipeline 的变量声明会在实例化时深复制到 Application Pipeline；之后两个对象独立演进。Template Pipeline 保留既有变量配置：内置变量和阶段声明只读，`pipeline_custom` 可新增、编辑、删除。Application Pipeline 详情在此基础上显示可管理变量和只读内置项：

| 管理来源 | 含义 | 操作 |
|---|---|---|
| `pipeline_custom` | 已持久化在当前 Application Pipeline 的变量 | 新增、编辑、删除 |
| `pipeline_stage` | 从阶段脚本和制品 `reference`、`name`、`command` 提取的非内置 Liquid 变量 | 覆盖为同名 Pipeline 配置 |
| `pipeline` | `repository_ref`、`repository_code`、`repository_url`、`runtime_datetime` | 只读 |

Repository 配置不显示也不能在 Pipeline 中覆盖。删除一个覆盖阶段变量的 `pipeline_custom` 配置后，该变量恢复为 `pipeline_stage`，继续使用阶段中声明的 Liquid `default`（若有）。变量名必须是非内置、唯一的 Liquid 标识符。

Repository 变量只在 Repository 管理，不出现在 Application Pipeline 的管理列表，也保持更高优先级。`runtime` 和 `system` 上下文由服务端生成，不可在 Pipeline 管理。

`default` 是回退值，`value` 是持久化配置值，`secret` 只影响展示元数据，不改变变量解析顺序。

## 运行时上下文

| 变量名 | 运行时来源 |
|---|---|
| `repository_code` / `repository_url` | Application Pipeline 绑定 Repository；`system`，由服务端注入 |
| `repository_ref` | 绑定 Repository 的 `default_branch`；`runtime`，由服务端注入 |
| `runtime_datetime` | 本次 Run 创建时生成的 UTC 时间戳；`system`，由服务端注入 |

来源 Template 只保存在 Pipeline/Snapshot 的追溯信息中；它不是运行期输入。

## 运行时合并

同名变量按以下顺序取第一个非空值：

```text
Repository 自定义变量的 value/default
  -> Application Pipeline 自己的 value/default
  -> Pipeline Stage 提取的 Liquid default
```

系统上下文不参与覆盖链。手动 Trigger 不再接受变量表单，`POST /api/pipeline/{pipeline_id}/trigger` 使用空请求体；服务端以当前 Repository、当前 Application Pipeline 配置和冻结阶段定义解析最终变量。任何非 system 变量最终无值时，Trigger 返回 validation error，用户需要先在变量管理区域补齐 Pipeline 配置。

应用流水线运行没有变量预览 API、运行弹窗、配置表单或配置列表。

## Snapshot 与历史

`PipelineSnapshot.variables_snapshot` 保存创建 Snapshot 时可获得的 Repository/Pipeline/Stage 声明与 runtime/system 元数据，用于历史展示和追溯；它不作为之后新 Run 的变量取值来源。

创建 Run 或 Retry 时，都从当前 Repository、当前 Application Pipeline 配置和该次使用的冻结阶段定义解析最终变量，保存到 `PipelineRun.variables_snapshot`。执行器只使用 Run 快照；之后修改任何配置不影响已创建的 Run。Retry 以原 Run 的非 system 快照值构造内部 overrides，再创建新的 Run；该内部路径不对手动 Trigger 暴露变量表单。
