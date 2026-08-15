# 应用流水线变量管理与直接运行
最后修改时间: 2026-08-15 13:57:20

Review status: Accepted

Mode: standard

## Background

当前 Application Pipeline 详情页把运行时变量预览同时用于变量配置展示。它使 Pipeline 自身配置、从阶段脚本和制品模板提取的变量、Repository 变量及运行时上下文混在同一张表中；并且只有 `pipeline_custom` 来源可以编辑。结果是用户无法在 Application Pipeline 中接管阶段变量，也必须在每次运行前填写一份完整的变量表单。

变量应有明确的管理归属：Repository 变量在 Repository 管理，Application Pipeline 变量在 Pipeline 管理，阶段文本只提供可被覆盖的声明与回退值。运行不应再成为人工配置入口。

## Goal

1. 将非 Template 的 Application Pipeline 变量区域调整为变量管理能力。
2. 用户可以新增 Application Pipeline 自身变量，编辑或删除已存在的 Pipeline 自身变量。
3. 用户可以将阶段脚本或制品模板提取出的变量覆盖为 Application Pipeline 配置；覆盖后 Pipeline 配置优先于阶段中的 Liquid `default`。
4. 点击 Application Pipeline 的“运行”后不展示变量预览、变量配置表单或变量列表；服务端直接使用当前持久化配置及固定运行时上下文创建 Run。
5. PipelineRun 仍保存最终变量快照供执行和历史追溯，但该快照不再是触发前的交互界面。

## Non-goal

- 不在 Application Pipeline 中编辑、删除或覆盖 `repository` 来源变量；Repository 配置继续保持更高优先级并只在 Repository 管理。
- 不允许配置或覆盖 `runtime`、`system` 上下文。`repository_ref` 直接取绑定 Repository 的 `default_branch`；`repository_code`、`repository_url`、`runtime_datetime` 继续由服务端生成。四个内置变量仍显示在 Application Pipeline 详情中，但只能只读查看。
- 不改变 Template Pipeline 在实例化时复制其变量声明到 Application Pipeline 的既有行为，也不移除其既有变量配置卡片；模板继续只能新增、编辑和删除自身 `pipeline_custom` 配置。
- 不改变既有 Run、Snapshot 的持久化结构或历史数据。
- 不在本任务处理变量值的加密、日志脱敏或权限模型。

## User scenarios

### 新增 Pipeline 变量

用户在 Application Pipeline 详情的变量管理区域新增 `IMAGE_TAG=20260815`。该配置保存为 Pipeline 自身变量，并在后续 Run 中注入执行环境；同名阶段变量存在时，该值覆盖阶段 `default`。

### 接管阶段变量

阶段脚本含有 `{{ BUILD_ARGS | default: "--no-cache" }}`，而 Pipeline 尚未配置 `BUILD_ARGS`。变量管理区域显示该变量及其阶段来源，用户选择覆盖并设置 `--pull`。系统持久化 Pipeline 配置，之后该变量按 Pipeline 值解析；删除该 Pipeline 配置后，变量回退为阶段默认值。

### Repository 变量保持归属

Repository 已配置 `REGISTRY_HOST`。Application Pipeline 的变量管理区域不提供覆盖、编辑或删除该变量的操作；运行时仍优先采用 Repository 配置。

### 直接运行

用户点击 Application Pipeline 的“运行”，前端不再打开变量弹窗，也不调用变量预览接口。服务端以当前 Repository/Pipeline/冻结阶段配置创建 Run；若存在没有 Pipeline 或阶段默认值的必填变量，创建失败并返回可读错误，用户应先在变量管理区域补齐配置。

## Acceptance

- [ ] Application Pipeline 详情提供变量管理区域；Template Pipeline 保留既有变量配置区域，但不使用 Application Pipeline 的阶段覆盖交互。
- [ ] 管理区域可新增任意非内置的 Pipeline 变量，并可编辑、删除已保存的 Pipeline 变量。
- [ ] 从当前 Application Pipeline 阶段脚本、制品 `reference`、`name` 或 `command` 提取的非内置变量，在没有同名 Pipeline 配置时可被“覆盖”；覆盖操作创建同名 Pipeline 配置而不修改阶段文本。
- [ ] 删除同名 Pipeline 配置后，变量重新显示为阶段来源并回退到阶段 `default`（若有）。
- [ ] Repository 变量在该区域不可被 Pipeline 覆盖，运行时优先级保持 Repository > Pipeline > Stage default。
- [ ] `repository_ref` 固定为绑定 Repository 的默认分支；`repository_code`、`repository_url`、`runtime_datetime` 在 Application Pipeline 详情中作为只读内置项显示，均不能被配置或请求覆盖。
- [ ] Application Pipeline 的运行按钮不打开变量弹窗、不展示变量表单或列表，且前端不在触发前调用变量预览。
- [ ] 直接触发时服务端根据当前持久化配置解析最终值；任何非系统变量最终无值时返回 validation error，不创建 Run。
- [ ] 新创建 Run 的变量快照仍包含执行所用的全部最终值，执行器和重试继续以 Run 快照为执行输入。
- [ ] Pipeline 变量保存后 Pipeline 版本递增，后续 Run 使用新配置，已创建 Run 不受影响。

## Decisions

- “覆盖已有非 `repository`、`pipeline` 配置”在本需求中具体指接管 `pipeline_stage` 来源变量：它们是阶段文本给出的可配置声明和默认值。Pipeline 配置作为更高优先级来源保存。
- `runtime` 与 `system` 不是用户可管理的配置。由于运行时不再提供表单，`repository_ref` 不再由用户提交，统一使用 Repository 默认分支；它们继续作为只读内置项显示在 Application Pipeline 详情中。
- 运行历史中的变量快照保留；“不展示详情配置表单或列表”只约束触发 Application Pipeline 时的交互，不删除 Snapshot/Run 历史查看能力。

## Open questions

暂无需要用户确认的未决事项。

## Risk

- 原有 Trigger 要求完整的非 system 表单；改为直接运行后，服务端需要把“请求完整”校验替换为“解析结果完整”校验，避免已保存的默认值被错误拒绝。
- 变量管理显示不能再依赖带有本次运行值的 preview 结果，否则会重新引入运行时状态并混淆 `pipeline_stage` 与 Pipeline 配置归属。
