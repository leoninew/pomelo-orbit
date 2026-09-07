# 流水线变量嵌套 Liquid 值验收
最后修改时间: 2026-09-07 23:10:24

Review status: Accepted

Mode: standard

## Basis

- Requirement: [流水线变量嵌套 Liquid 值](../requirement/20260907-pipeline-variable-nested-liquid-values.md)（`Accepted`）
- Plan: [流水线变量嵌套 Liquid 值实施计划](../plan/20260907-pipeline-variable-nested-liquid-values.md)（`Accepted`）

## Requirement Alignment

- 通用 `internal/common/template.ResolveNestedValues` 仅接受简单 `{{ NAME }}`，通过依赖图递归展开命名字符串模板和只读根上下文；Pipeline、Repository、Artifact、Version 等领域概念未进入该实现。
- `pipelinevariable` 仅装配既有 Repository、Pipeline 全局、当前 Stage 与系统根变量的可见域，再调用通用解析器。全局域不包含 Stage 私有值，Stage 域不包含其他 Stage 私有值。
- `default` 经 `ValidateLiteralValue` 保持字面量；只有缺少有效 `value` 时，既有 `EffectiveVariableValue` 才将其作为叶子回退。
- `CreatePipeline`、Template Pipeline 配置校验和 Application Pipeline 配置校验均在持久化前调用 `ValidateNestedVariableValues`。运行时在渲染 Stage 字段及生成 Run 快照前再次防御性解析。
- Stage `script`、Artifact 的 `reference`、`name`、`command` 仍使用既有完整 Liquid 渲染入口；变量配置值的受限嵌套语法没有扩展为第二套领域表达式。

## Actual Diff

本需求的实现范围：

| 预期区域 | 实际改动 |
| --- | --- |
| 通用嵌套解析 | `internal/common/template/template.go`、`internal/common/template/template_test.go` |
| Pipeline 变量薄适配 | `internal/application/pipeline/rule/pipelinevariable/runtime.go`、`runtime_test.go`；已删除迁移后的业务表达式解析文件 `stage_expression.go` |
| Pipeline 保存校验 | `internal/application/pipeline/usecase/pipeline.go`、`pipeline_test.go`、`stage_template.go` |
| 运行时 Stage 渲染 | `internal/application/pipeline_run/usecase/stage_resolve.go` |
| 活文档 | `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md` |

工作区同时存在 `000033` seed SQL、迁移测试和 Pipeline/Repository 前端页面的改动。这些是相邻的既有工作，不属于本需求的实现或验收结论，未在本次验收中扩大为额外需求。

## Acceptance Checklist

- [x] 变量 `value` 支持简单 `{{ NAME }}` 引用及文本拼接，并在 Run 创建前展开。
- [x] `default` 仅作为无 `value` 的叶子字面量回退，包含 Liquid 分隔符的默认值在保存时失败。
- [x] 系统、Repository、Pipeline 全局和当前 Stage 可见性沿用既有优先级；跨 Stage 私有引用被拒绝。
- [x] 运行时变量在 Stage 字段渲染与 Run 快照前已解析；回归测试断言快照不包含 `{{`。
- [x] 多级引用、未知引用、直接循环和间接循环由通用解析器返回包含变量名或依赖链的 validation error。
- [x] Create、Update 及 Stage 配置写路径在持久化前校验；Update 用例断言无效配置不调用持久化且版本不变。
- [x] filter、tag、dotted path、不完整 Liquid 与 Shell 风格默认值按各自边界拒绝；`${NAME}` 在嵌套值中保持普通文本。
- [x] 非嵌套变量、Stage 字段的既有 Liquid `default`、来源优先级与字段渲染由原有及定向回归测试覆盖。

## Validation Evidence

| 命令 | 范围 | 结果 |
| --- | --- | --- |
| `go test ./internal/common/template ./internal/application/pipeline/rule/pipelinevariable ./internal/application/pipeline/usecase ./internal/application/pipeline_run/usecase` | 通用解析、运行时变量、保存校验、Stage 渲染 | 通过 |
| `yarn --cwd web lint:fix` | 已变更前端源码的 ESLint 问题修复 | 完成，无错误输出 |
| `yarn --cwd web typecheck` | 前端 TypeScript 类型检查 | 通过 |
| `task check` | 前端 typecheck/lint/format 与 Go 格式、静态检查 | 通过 |
| `go test ./cmd/... ./internal/...` | Go 全量测试 | 本次未重复执行；用户明确确认已运行且无问题 |

未启动或重启开发服务器，未执行数据库迁移、种子导入或修改运行数据。

## Scope Deviations And Risk

- 无产品行为范围扩张：未增加 Shell 插值、Liquid filter、业务命名规则、节点参数持久化、API/Proto 契约或数据迁移。
- 活跃工作区含有本需求外的未提交变更；提交时应将其与本需求的通用模板和 Pipeline 校验变更分组审查，避免混入无关 seed 或 UI 修改。
- 历史数据未迁移。旧配置若绕过保存校验而包含不支持的嵌套值，运行时防御性校验会拒绝执行，需在下一次保存时修正。

## Incomplete Items

无。用户已确认全量检查无问题；本次未重复执行其已完成的 Go 全量测试。

## Conclusion

实现与 Requirement 和 Plan 一致，验收标准满足。标准模板层承担嵌套语法、引用解析与循环检测，Pipeline 层只负责可见性、优先级和接入，配置错误在保存前阻断且最终运行时快照保存已展开值。
