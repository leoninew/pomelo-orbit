# CI Pipeline 变量设计
最后修改时间: 2026-08-07 16:35:00

Doc role: living guide。与代码冲突时以代码为准。

## 归属

变量声明归 Pipeline 所有。Template 的声明在实例化时深复制到 Application Pipeline；之后两个对象独立演进。Application Pipeline 运行时以它自己的 Snapshot 作为唯一执行输入，不读取当前 Template。

变量的来源字段和可编辑性仍由服务端解析。`default` 表示系统或脚本给出的默认值，`value` 是持久化覆盖；`secret` 只影响展示与日志脱敏，不改变传递规则。

## 内置变量

| 变量名 | 运行时来源 |
|---|---|
| `repository_id` / `repository_name` / `repository_url` | Application Pipeline 已绑定的 Repository |
| `repository_ref` | 本次 Trigger 的 ref |
| `pipeline_id` / `pipeline_name` / `pipeline_version` | Application Pipeline 与其运行 Snapshot |

不再存在 `template_id`、`template_name`、`template_version` 作为运行期选择输入。来源 Template 仅在 Pipeline/Snapshot 的追溯信息中保存。

## 运行时合并

高到低：

```text
系统内置变量（不可覆盖）
  -> 触发请求中的 variables
  -> Repository 自定义变量
  -> Application Pipeline 自己的声明 value/default
  -> Stage 提取的默认值
```

手动运行的请求仅有：

```json
{ "trigger_ref": "main", "variables": { "IMAGE_TAG": "20260807" } }
```

Repository、可选的 Application、目标 Component 与来源 Version 策略均在 Pipeline 配置中冻结，不能作为运行参数传入。Retry 与手动触发共用相同的变量解析和 Snapshot/Run 创建路径。

## 运行前预览

`POST /api/pipeline/{pipeline_id}/variable-preview` 接受与手动 Trigger 相同的
`trigger_ref` 和 `variables`。它使用同一条运行时合并链路返回完整变量声明及其
解析后的 `value`，但不创建 Snapshot、PipelineRun、任务或 Version 绑定。

运行弹窗以预览结果填充可编辑项；实际 Trigger 只提交用户相对预览初值修改过的
变量，避免将 UI 中展示的默认值误作为最高优先级的手工覆盖。

## Snapshot 与历史

`PipelineSnapshot.variables_snapshot` 保存该 Application Pipeline 版本的声明。`PipelineRun.variables_snapshot` 保存本次实际输入的脱敏快照，并与 Pipeline、Repository 及可选的 Application/Version 绑定一同保留历史追溯。删除配置资源不要求重写历史变量快照。
