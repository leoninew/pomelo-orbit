# 流水线阶段变量默认值规范实施计划
最后修改时间: 2026-09-06 15:12:00

Review status: Accepted

Mode: standard

## Basis

- Requirement: [流水线阶段变量默认值规范](../requirement/20260906-pipeline-stage-variable-defaults.md)（`Accepted`）
- 当前实现：Go Liquid 渲染器位于 `internal/common/template`；流水线变量提取、合并和运行时解析位于 `internal/application/pipeline/rule/pipelinevariable`。

本计划采用 Liquid 单一变量规范：`{{ NAME }}` 表示必填，`{{ NAME | default: "value" }}` 表示阶段位置级默认值。不增加 `${NAME:-default}` 或其他默认语法的兼容路径。

## Overview

当前实现将多个阶段中同名变量压缩到一个 `map[name]`，详情和运行时都丢失来源 Stage。实施后使用作用域键分离变量：

```text
Repository value/default
  -> 全局运行变量

Pipeline Stage value/default
  -> 对应 Stage 的运行变量

Pipeline global value/default
  -> 未配置 Stage 值时的全局运行变量

Stage occurrence default
  -> 当前 Liquid Render 独立应用
```

变量管理按 `(name, stage_id)` 展示 Stage 条目；Pipeline 阶段覆盖值写回现有 JSON 配置中的 `stage_id` 字段，不新增数据库迁移。

## Implementation Steps

1. **建立共享 Liquid 变量引用解析规则**
   - 在 `internal/common/template` 集中定义支持的 Liquid 输出变量引用和 `default` filter 提取结果，复用当前渲染器的变量命名与默认值语义。
   - 明确识别 `{{ NAME }}` 和 `{{ NAME | default: "value" }}`；不识别 `${NAME:-default}` 或其他默认运算符。
   - 解析结果至少包含变量名、是否有默认值和默认值字面量；非法或无法完整识别的默认表达式返回明确错误，避免详情提取与运行渲染使用不同规则。
   - 保持 Liquid 条件、循环和 dotted path 的现有渲染行为；本任务只统一流水线变量默认值提取与校验。

2. **扩展内部变量声明和配置作用域**
   - 在 `internal/model/pipeline.go` 为 `VariableDeclaration` 增加 `stage_id/stage_name`；阶段声明按 `(name, stage_id)` 生成独立条目，阶段默认值只属于该条目。
   - 在 `internal/application/pipeline/rule/pipelinevariable/runtime.go` 保留全局变量与 Stage 变量两个运行时作用域，使用同一作用域键合并 Repository、Pipeline 和阶段解析结果。
   - `ResolvePipelineVariables`、`ResolveTemplatePipelineVariables` 和 `RuntimeVariableDeclarations` 不再按变量名压缩不同 Stage；阶段 Pipeline 配置通过 `stage_id` 与阶段声明匹配。
   - Pipeline 自定义变量仍保存于 `pipeline.variable_declarations`，但阶段覆盖必须携带 `stage_id`；不带 `stage_id` 的自定义变量表示全局配置。

3. **调整运行时解析与 Stage 渲染**
   - 修改 `ResolveRuntimeVariables`：返回全局变量和按 Stage 的变量集合；Repository 全局值优先，其次是 Pipeline Stage 值、Pipeline 全局值，最后由当前 Liquid 表达式应用 default。
   - 对所有使用 `{{ NAME | default: ... }}` 的 occurrence，在没有高优先级值时允许由当前 Stage 的 Liquid Render 应用自身默认值；对使用 `{{ NAME }}` 且无高优先级值的 occurrence，在运行创建或执行前返回包含 Stage 名称和变量名的 validation error。
   - 保持 `resolveStages` 对 `script`、Artifact `reference`、`name`、`command` 的统一 Liquid 渲染入口；为渲染错误补充阶段和字段上下文。
   - 调整 `MarshalRuntimeVariableSnapshot` / `UnmarshalRuntimeVariableSnapshot`，使同名不同 Stage 的值按 `stage_id` 保存并恢复；仅依赖阶段 default 的条目不被误判为缺少全局运行值。
   - Snapshot 创建、Trigger、Retry 和 Executor 继续使用既有生命周期；只替换变量声明/解析细节，不改变 Run、Artifact、Version fork 流程。

4. **扩展 HTTP/Proto 变量展示和配置契约**
   - 在 `proto/orbit/v1/common/common.proto` 为响应增加 `stage_id/stage_name`，为请求增加 `stage_id`；阶段条目和阶段覆盖可被外层准确区分。
   - 更新 `internal/api/http/handler/pipeline/mapper.go`，映射 Stage 来源和请求中的 `stage_id`；阶段名称和默认值仍为派生展示信息。
   - 执行 `task proto`，审核 Go/TypeScript 生成物只包含本次 Proto 变化。

5. **更新流水线详情展示**
   - 在 `web/src/views/pipeline/components/VariableDeclarationsTable.vue` 以 `name + stage_id + source` 作为行身份，直接展示每个 Stage 条目，不再把多阶段同名变量合并为一行。
   - `PipelineDetail.vue` 的新增、编辑、删除和覆盖操作按 Stage 条目提交 `stage_id`；全局自定义变量不带 `stage_id`。
   - 同步中英文变量表头/辅助文案及生成 TypeScript 类型。

6. **补充测试并校准活文档**
   - 扩展 `internal/common/template` 测试，锁定 Liquid `default` 对缺失值和空字符串的行为，以及不支持 shell-style 默认表达式的边界。
   - 扩展 `internal/application/pipeline/rule/pipelinevariable/runtime_test.go`，覆盖：`repository_dockerfile`/`working_dir` 识别、脚本与 Artifact 字段提取、单阶段默认值、同名变量多阶段不同默认值、Repository/Pipeline 覆盖、必填变量缺失和非法表达式。
   - 扩展 `internal/application/pipeline_run/usecase` 测试，验证前端阶段和后端阶段使用不同 `working_dir` 默认值，且 Run 执行快照/Stage 渲染结果正确。
   - 增加 HTTP mapper/前端相关测试或最小类型约束，验证阶段默认值字段不会进入更新请求。
   - 更新 `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md`，将流水线阶段默认值规范统一为 Liquid，并说明阶段 occurrence 语义；不修改归档文档。

## Files To Change

| 区域 | 主要路径 |
|---|---|
| Liquid 解析 | `internal/common/template/template.go`、`internal/common/template/template_test.go` |
| 变量规则与模型 | `internal/model/pipeline.go`、`internal/application/pipeline/rule/pipelinevariable/runtime.go`、`internal/application/pipeline/rule/pipelinevariable/runtime_test.go` |
| Run 执行 | `internal/application/pipeline_run/usecase/stage_resolve.go`、`internal/application/pipeline_run/usecase/execution.go`、相关测试 |
| Snapshot | `internal/application/pipeline/usecase/snapshot_create.go`、相关测试 |
| HTTP/Proto | `proto/orbit/v1/common/common.proto`、`internal/api/http/handler/pipeline/mapper.go`、生成代码 |
| 前端 | `web/src/views/pipeline/components/VariableDeclarationsTable.vue`、`web/src/views/pipeline/PipelineDetail.vue`、`web/src/gen/proto/**`、`web/src/i18n/locales/{zh-CN,en-US}.ts` |
| 活文档 | `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md` |

不新增数据库迁移；不修改已执行迁移文件；不触碰与本任务无关的工作区改动。

## Verification Plan

1. 生成检查：执行 `task proto`，审查生成物和实际 Proto diff。
2. Go 定向测试：执行 `go test ./internal/common/template/... ./internal/application/pipeline/rule/pipelinevariable/... ./internal/application/pipeline_run/usecase/... ./internal/api/http/handler/pipeline/...`。
3. Go 项目检查：按仓库约定执行 `task check` 与 `go test ./cmd/... ./internal/...`。
4. 前端检查：执行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`；若已有前端测试入口覆盖变量表，再执行对应测试。
5. 行为验收：验证单阶段默认值、多阶段同名 `working_dir` 分别解析、Pipeline 覆盖、必填变量错误上下文、Artifact 引用解析和详情展示。
6. 全文检查：确认活文档、代码和 seed 示例不再把 `${NAME:-default}` 描述为流水线阶段默认语法；归档材料不作修改。

## Blockers, Assumptions And Risks

- `VariableDeclaration` 当前同时用于 Pipeline detail、Snapshot detail 和 Run 快照；新增阶段默认值字段必须保证 JSON/Proto 展示与运行时反序列化都能区分“阶段 fallback”与“全局有效 value”。
- Liquid `default` 的具体空值行为以当前 `github.com/osteele/liquid` 实测为准，测试锁定后不在业务层重复实现第二套判断。
- 变量管理当前按变量名渲染单行；阶段默认值集合需要在不破坏已有编辑/覆盖操作的前提下展示，不能把派生字段误提交。
- 当前数据库 seed 和已有未归档活文档可能仍使用无默认的 `{{ NAME }}`，或历史文本使用 `${NAME:-default}`；本任务不提供运行时兼容，需在实际 diff 中明确需要改写的活跃 seed/示例。

## Rollback

本任务不涉及数据库迁移。代码回退以同一版本的 Proto/Go/Web 生成物一致为前提；不得通过重新加入 `${NAME:-default}` 兼容分支回退语义。若实施中发现 API 字段设计不足，应在未交付前继续调整本次 Proto，不保留双轨字段。

## User Review Notes

用户已确认：流水线阶段继续使用 Liquid；唯一默认写法为 `{{ NAME | default: "value" }}`；不做 `${NAME:-default}` 或其他语法兼容；多阶段同名变量默认值按阶段出现位置保留和解析。用户要求计划完成后直接进入实现。
