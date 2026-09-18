# 流水线运行时变量解析与运行弹窗
最后修改时间: 2026-08-12 11:29:20

Review status: Accepted

Mode: strict

## Background

运行流水线时的变量并不只来自弹窗输入。它由仓库配置、Application Pipeline 配置、Application Pipeline 的构建阶段，以及系统在本次运行中解析的上下文共同构成。当前实现有两个缺口：

- 阶段模板中的 `{{ VARIABLE | default: "value" }}` 仍可被解析，但与 Pipeline 预定义变量合并时，未被阶段直接引用的 Pipeline 变量会丢失；这使曾有的默认值链路不完整。
- 运行弹窗把 `trigger_ref` 作为独立的「分支或标签」字段，而同一概念也应以 `repository_ref` 出现在完整的运行时配置中。并且当前将所有内置变量一概标为只读，不符合“除固化上下文外，运行时配置均可修改”的规则。

本任务重建完整的运行时变量模型，并以 `repository_ref` 作为运行弹窗中唯一的 ref 输入。

## Business Model

### 声明来源

运行时变量声明按来源产生，名称在最终列表中唯一：

| 来源 | 产生方式 | 典型变量 | 默认值 / 优先级 |
|---|---|---|---|
| `repository` | 仓库变量配置 | `repository_dockerfile` | 仓库的 `value`，否则 `default`；高于 Pipeline 与阶段默认值 |
| `pipeline` | Application Pipeline 的变量配置 | `IMAGE_TAG` | Pipeline 的 `value`，否则 `default`；高于阶段默认值 |
| `pipeline_stage` | 从 Application Pipeline 阶段脚本及制品模板提取 | `{{ IMAGE_TAG | default: "latest" }}` | 模板内 `default`；同名声明缺少配置默认值时补入 |
| `runtime` | 为本次运行生成的可配置上下文 | `repository_ref` | 默认值来自仓库 `default_branch` |
| `system` | 为本次运行生成的结构性上下文 | `repository_code`、`repository_url`、`runtime_datetime` | 固化值，不接受用户覆盖 |

`repository_ref` 是运行时配置，不是仓库变量覆盖。Trigger 与变量预览请求只提交完整的非 `system` 变量表单，其中 `repository_ref` 是唯一的源码 ref 输入；不再有 `trigger_ref`。无论仓库类型，后端保留用户提交的 ref 作为运行时值，并持久化到 `PipelineRun.repository_ref`，它必须与该 Run 完整变量快照中的 `repository_ref` 一致；不解析为 commit SHA，也不新增 `repository_sha`。源码 commit 的追溯继续以已有的 `command/git_object_id` 制品为准。

### 合并规则

对同名的可配置变量，从高到低采用第一个有值的来源：

```text
本次运行的用户覆盖
  -> Repository 变量的 value/default
  -> Pipeline 变量的 value/default
  -> Pipeline Stage 模板中的 default
```

- `value` 是来源配置中已保存的取值，`default` 是未显式配置时的回退值。
- 阶段提取只补充声明和默认值，不能删除未被阶段文本引用的 Repository 或 Pipeline 变量。
- 同一变量只能有一条最终声明；保留最终生效声明的来源、说明、`secret` 与默认值。
- 系统上下文不参与上述覆盖链。若被包含在运行环境中，始终由服务端生成，且 `editable=false`；请求在 `variables` 中携带任意 `system` 变量时必须被拒绝。
- 除 `system` 外，最终运行时配置均为 `editable=true`。运行弹窗允许覆盖 Repository、Pipeline、Pipeline Stage 和 `runtime` 来源的值。

### Snapshot 与历史

```text
Repository / Pipeline / Pipeline Stage 配置
  -> PipelineSnapshot: 冻结本 Pipeline 版本的阶段定义与变量声明快照，用于历史和追溯
  -> Variable Preview: 叠加当前仓库配置与本次完整表单，生成完整运行时配置
  -> PipelineRun: 引用 PipelineSnapshot，并持久化本次完整的有效变量快照
  -> Executor: 使用 PipelineRun 的变量快照执行，不因之后的配置变更改变该 Run
```

- `PipelineSnapshot.variables_snapshot` 保存创建 Snapshot 时可获得的完整变量声明历史：Repository、Pipeline 与 Stage 的配置值/默认值，以及 `runtime`/`system` 的声明元数据。它用于历史展示与追溯，不是运行时变量取值来源；用户表单值和本次生成的系统值只保存到 Run。
- 每次预览、创建 Run 或 Retry 都读取当前 Repository 与当前 Application Pipeline 的变量配置和本次表单重新解析变量；不得复用 Snapshot 的变量声明或值作为运行时输入。Snapshot 的阶段定义仍作为 Trigger/Retry 的结构执行输入，Preview 使用当前阶段定义。
- `PipelineRun.variables_snapshot` 必须持久化本次所有最终注入值，包含可配置项及系统上下文。
- 已创建 Run 的变量快照不可回写。后续修改仓库、Pipeline 或阶段配置只影响后续新建的 Run。
- Pipeline Snapshot 只冻结 Pipeline / Stage 定义；复用同一 Snapshot 创建 Run 时，仍须重新读取当前 Repository 变量配置，并与本次提交的完整表单重新解析。Snapshot 不复用任何一次运行的变量取值。
- Retry 创建新的 Run，以原 Run 的完整非 `system` 变量快照作为表单重新解析；新的 Run 仍持久化自己的完整变量快照。

## Goal

1. 恢复并统一 Repository、Pipeline 与 Pipeline Stage 的变量声明与默认值合并。
2. 变量预览始终返回完整、去重的运行时变量列表；不得因阶段是否引用某变量而漏掉 Repository、Pipeline 或 `repository_ref`。
3. 运行弹窗移除独立「分支或标签」字段，使用可编辑的 `repository_ref` 作为唯一 ref 输入。
4. 运行弹窗允许修改全部非 `system` 的运行时配置，并在 debounce 后以最新输入刷新预览。
5. Trigger 与变量预览 API 只接受 `{ variables }`，其中 `variables` 是完整的非 `system` 运行表单，`repository_ref` 是唯一的源码 ref 输入；Run 模型、存储与响应同样使用 `repository_ref`。
6. Run 创建时持久化完整的最终变量快照，执行和历史查看均以该快照为准。

## Non-goal

- 不保留 `trigger_ref` 作为请求、响应、模型或存储字段，不新增 `repository_ref` 的第二个传输字段。
- 不允许覆盖 `system` 来源的结构性上下文，不通过 `variables` map 覆盖它们。
- 不新增远程分支或标签存在性校验、自动补全或 Webhook 触发。
- 不解析 `repository_ref` 为 commit SHA，也不新增 `repository_sha`；源码 commit 的产物追溯保持既有 `command/git_object_id` 机制。
- 不改变配置页编辑 Repository / Pipeline 配置的职责；本任务只改变运行前预览与运行弹窗的可编辑性。

## Interaction Rules

- 打开运行弹窗时，先请求变量预览。变量列表必须包含 `repository_ref`，默认值为仓库 `default_branch`；其余可配置变量预填其有效值，系统变量只读展示。
- 修改任一可配置变量都触发 debounce 预览，并提交当前完整的非 `system` 表单，包括 `repository_ref`。
- 提交时校验所有可配置变量的最终值。`repository_ref` 为空时在该输入下显示「请输入分支或标签」；其他缺失值在各自输入下显示错误。网络和解析失败保留为表单级错误。
- 预览失败时禁用确认；关闭运行弹窗时清除本实例的字段错误和 debounce 请求。

## Acceptance

- [ ] Pipeline Snapshot 保存创建时完整的可快照化变量声明以供历史追溯，包括 Repository、Pipeline 与 Stage 的配置值/默认值；阶段模板的 `default` 未被遗漏，当前 Pipeline 声明不因未被阶段引用而丢失；用户表单值和系统实际值只保存到 Run，Snapshot 声明不作为后续运行的变量取值来源。
- [ ] 变量预览对“无阶段变量”“仅阶段变量”“Pipeline 变量未被阶段引用”“Repository/Pipeline/Stage 同名变量”均返回完整、唯一且按合并规则解析的声明。
- [ ] 预览始终包含唯一的 `repository_ref`，`editable=true`，其值保持用户提交的 ref；不会被解析为 SHA。
- [ ] `system` 上下文为只读，请求 `variables` 携带任意 `system` 键时返回校验错误；其他来源均可在运行弹窗编辑。
- [ ] 运行弹窗没有独立「分支或标签」字段；`repository_ref` 是唯一 ref 输入。修改它连同其他非 `system` 值触发 debounce 预览。
- [ ] Trigger 与预览请求不含 `trigger_ref`；`variables` 含完整非 `system` 表单和 `repository_ref`，不含任意 `system` 变量。
- [ ] `PipelineRun` 的模型、数据库存储与响应使用 `repository_ref`；它与该 Run 变量快照中的 `repository_ref` 相同。
- [ ] 本地的必填错误展示在对应输入下，并带错误边框和 `aria-invalid`；修正当前字段后只清除该字段错误。
- [ ] 每个新建 Run 引用 Pipeline Snapshot，并持久化包含最终有效值的完整变量快照；之后修改任何变量配置不会改变已创建 Run 的执行环境或历史展示。
- [ ] 复用同一 Pipeline Snapshot 创建后续 Run 时，仍以当前 Repository 变量配置和本次完整表单重新解析变量。
- [ ] Retry 使用原 Run 的完整非 `system` 变量快照创建新的 Run，并保持历史 Run 不变。

## Risk

- 前后端需要同版本发布：新前端依赖预览返回完整的 `repository_ref` 与 `editable` 声明。
- 现有 `PipelineRun.variables_snapshot` 只保存声明列表中的项目；改为完整运行环境后，执行器和 Retry 必须直接使用该快照，避免在执行时重新读取已变化的配置。
