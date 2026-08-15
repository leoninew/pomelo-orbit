# 应用流水线变量管理与直接运行规格
最后修改时间: 2026-08-15 13:57:20

Review status: Accepted

Mode: standard

## Requirement Basis

本规格落实 [Requirement](../requirement/20260815-application-pipeline-variable-management.md)。Application Pipeline 的变量配置从运行前临时表单迁移到 Pipeline 详情中的持久化管理；运行只负责按当前配置创建 Run。Template Pipeline 保留既有变量配置交互，不增加 Application Pipeline 的阶段覆盖能力。

## Overview

变量处理拆成两个互不混淆的视图：

```text
Application Pipeline 阶段定义 + Pipeline 自身变量
  -> 变量管理列表（只用于详情页配置）

Repository + Pipeline + 冻结阶段 + 服务端上下文
  -> 最终运行变量快照（只用于创建、执行和追溯 Run）
```

详情变量列表包含两个可管理来源和一组只读内置项：

| 来源 | 含义 | 管理操作 |
|---|---|---|
| `pipeline_custom` | 已持久化在当前 Pipeline 的配置 | 编辑、删除 |
| `pipeline_stage` | 从当前 Application Pipeline 阶段脚本、制品模板提取，尚未被 Pipeline 接管 | 覆盖 |
| `pipeline` | `repository_ref`、`repository_code`、`repository_url`、`runtime_datetime` | 只读 |

Repository 变量不进入详情列表，且继续有更高优先级。运行时和系统变量由后端产生，不能通过 Pipeline 配置覆盖；详情只保留它们的只读内置声明，不显示本次运行的值。

## Variable Management

### 管理列表

`GET /api/pipeline/:pipeline_id` 的 `variable_declarations` 对 Application Pipeline 表示详情变量列表，不再代表完整运行时预览：

1. 从当前 Pipeline 阶段的 `script` 及制品的 `reference`、`name`、`command` 提取符合现有 Liquid 变量语法的非内置变量和可选 `default`。
2. 读取该 Pipeline 已保存的 `variable_declarations`，规范化为 `pipeline_custom` 配置。
3. 按变量名合并：有同名 Pipeline 配置时仅返回该配置，且阶段 `default` 仅在 Pipeline 配置没有 `value` 或 `default` 时作为回退；否则返回 `pipeline_stage` 条目。
4. 追加 `repository_ref`、`repository_code`、`repository_url`、`runtime_datetime` 四个只读内置条目；不追加 Repository 配置，也不返回本次运行的动态值。

列表中的 `pipeline_custom` 条目可编辑和删除；`pipeline_stage` 条目不能直接编辑或删除，但提供“覆盖”操作。覆盖打开与新增共用的编辑表单，变量名固定为被覆盖的名称，保存后创建同名 `pipeline_custom` 配置。删除一个同名 `pipeline_custom` 后，下一次详情读取恢复显示对应 `pipeline_stage` 条目和它的阶段默认值。

新增变量必须是非空、非内置且符合当前 Liquid 变量名规则的唯一名称。后端承担该校验和去重，不能只依赖前端；变量值继续使用当前 `value`、`default`、`description`、`secret` 数据结构。详情 UI 的新增/编辑表单保持只编辑 `value`、`description`、`secret`，不新增单独的 `default` 控件。

Template Pipeline 的接口存储模型不变，详情页保留既有“变量配置”卡片：内置变量和阶段声明只读，`pipeline_custom` 可新增、编辑、删除；不提供阶段变量“覆盖”操作。Template Pipeline 不可运行，因此不展示运行时变量表。

### 优先级

运行时同名变量的有效值顺序调整为：

```text
Repository 的 value/default
  -> Application Pipeline 的 value/default
  -> Pipeline Stage 的 Liquid default
```

手动 Trigger 不再接受用户覆盖，因此移除“本次提交表单”这一优先级层。Repository 同名配置始终赢过 Pipeline；Pipeline 只能覆盖 `pipeline_stage` 的值或默认值。变量没有任意有效值时，Run 创建失败。

## Direct Trigger

Application Pipeline 的运行按钮直接调用 Trigger API，不再打开运行弹窗、加载变量预览或构造变量表单。

`PipelineRunTriggerReq` 改为空消息，`PipelineRunTriggerInput` 不再携带 `Variables`。`POST /api/pipeline/:pipeline_id/variable-preview` 及其 Handler、DTO、Proto、前端 API 和调用点全部删除；项目处于活跃开发期，不保留旧路由或兼容字段。

手动 Trigger 使用当前 Repository、当前 Application Pipeline 配置和该 Pipeline 版本的冻结阶段定义解析变量：

- `repository_ref` 由绑定 Repository 的 `default_branch` 直接填入，并在 Run 与变量快照中一致保存。
- `repository_code`、`repository_url`、`runtime_datetime` 保持服务端生成。
- 所有非系统声明在合并后必须有非空最终值；否则返回 validation error，且不创建 Snapshot、Run、Version binding 或任务。

预览消失后，不再为前端保留“部分输入可预览”的解析路径。变量规则可保留内部的 overrides 参数，只服务于 Retry 从历史 Run 快照重建输入，不能通过 HTTP 手动 Trigger 传入。

## Snapshot, Execution And Retry

`PipelineSnapshot.variables_snapshot` 继续保存快照创建时可得的 Repository/Pipeline/Stage 声明及 runtime/system 元数据，用于历史追溯。它不作为新 Run 取值来源。

`PipelineRun.variables_snapshot` 继续保存该 Run 的完整最终值，执行器只读取该快照。Pipeline 或 Repository 配置在 Run 创建后的修改不影响该 Run。

Retry 保持既有历史语义：从原 Run 的非 system 变量快照构造内部 overrides，再与当前 Repository/Pipeline 配置和本次结构 Snapshot 解析为新的 Run 快照。该内部路径不重新引入用户运行表单；对新建的直接运行，`repository_ref` 始终来自 Repository 默认分支。

## API And Frontend Changes

| 范围 | 变更 |
|---|---|
| Pipeline detail | Application Pipeline 的 `variable_declarations` 改为详情变量列表：包含只读内置项、Pipeline 自身配置和阶段声明，不含 Repository 配置。 |
| Pipeline detail UI | Template Pipeline 保留原有变量配置卡片，只允许新增、编辑、删除 `pipeline_custom`；Application Pipeline 的内置项只读，并额外支持覆盖 `pipeline_stage`。 |
| Trigger API | `POST /api/pipeline/:pipeline_id/trigger` 请求体为空。 |
| Preview API | 删除 `/api/pipeline/:pipeline_id/variable-preview` 及所有类型、调用和测试。 |
| Run UI | 删除运行弹窗、预览状态、表单、字段错误、debounce 和相关 watch；点击运行直接创建 Run，成功后跳转 Run 详情。 |
| Run/Snapshot detail | 保留变量快照历史表，不作为触发交互的一部分。 |

## Affected Components

| 层 | 主要位置 |
|---|---|
| Proto/HTTP | `proto/orbit/v1/pipeline_run/pipeline_run.proto`、`internal/api/http/routes/pipeline_run.go`、`internal/api/http/handler/pipeline_run/` |
| Pipeline 变量规则 | `internal/application/pipeline/rule/pipelinevariable/runtime.go` 及其测试 |
| Pipeline detail | `internal/application/pipeline/usecase/pipeline.go`、`internal/api/http/handler/pipeline/mapper.go` |
| Run usecase | `internal/application/pipeline_run/usecase/service.go`、相关 DTO 与测试 |
| Frontend | `web/src/views/pipeline/PipelineDetail.vue`、`web/src/api/pipeline_run/pipeline_run.ts`、生成 Proto 类型 |
| 活文档 | `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md` |

## Risks

- 移除 Preview 后，原本的“完整表单”校验不能原样保留；必须校验合并后的最终值，不能要求请求 map 中出现每个键。
- 现有详情页以 Preview 结果作为变量表数据。改用管理列表后需确保 stage 覆盖、删除回退和 Pipeline 配置版本递增都在保存后立即刷新。
- 删除公共 Preview endpoint 与 Trigger `variables` 字段要求前后端、Proto 生成物和测试在同一改动中同步更新。

## Alternatives

- 保留 Preview endpoint 供变量管理区域使用：不采纳。它携带运行时值和系统上下文，会再次混淆配置管理与运行解析。
- 在运行按钮保留只读变量确认列表：不采纳。需求明确运行时不展示详情配置表单或列表。
- 允许 Pipeline 覆盖 Repository 变量：不采纳。Repository 是该变量的管理归属，并保持最高优先级。

## Open questions

暂无需要用户确认的未决事项。
