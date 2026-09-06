# 流水线阶段变量默认值规范验证
最后修改时间: 2026-09-06 22:43:30

Review status: Accepted

Mode: standard

## Basis

- Requirement: [流水线阶段变量默认值规范](../requirement/20260906-pipeline-stage-variable-defaults.md)（`Accepted`）
- Plan: [流水线阶段变量默认值规范实施计划](../plan/20260906-pipeline-stage-variable-defaults.md)（`Accepted`）

## Requirement Alignment

- 阶段文本统一使用 Liquid `{{ NAME }}` 和 `{{ NAME | default: "value" }}`；未新增 `${NAME:-default}` 或其他兼容分支。
- 变量提取覆盖阶段 `script` 及 Artifact 的 `reference`、`name`、`command`，并复用 Liquid 解析与运行时渲染规则。
- 变量声明、运行时和 Run 快照均按 `(name, stage_id)` 区分全局与阶段作用域；多阶段同名 `working_dir` 不再合并。
- Repository 全局值、Pipeline 阶段值、Pipeline 全局值和当前 Liquid 默认值按既定优先级生效；无默认值的必填变量保留阶段上下文报错。
- HTTP/Proto 请求增加 `stage_id`；前端提交前将响应 DTO 显式转换为请求 DTO，避免 `stage_defaults`、`stage_name` 等展示字段导致严格 Proto JSON 绑定失败。

## Actual Diff Summary

- Liquid 模板解析与变量规则：新增默认值引用提取、阶段来源和运行时全局/阶段变量集合。
- Pipeline Run：冻结并恢复阶段变量作用域，按阶段独立渲染，补充缺失变量和 Stage 删除覆盖的校验。
- API：变量响应返回 `stage_id`、`stage_name` 和阶段默认值；变量请求只接受可写字段与 `stage_id`。
- 前端：变量表以变量名、来源阶段和来源类型为行身份；列顺序为“默认值、当前值”，并更新中英文文案。Pipeline、Repository 的变量更新不再透传响应专用字段。
- 活文档：CI Pipeline 及变量设计说明已同步 Liquid 默认值和 Stage 作用域规则。

## Expected And Actual Files

| 范围 | 计划 | 实际 |
|---|---|---|
| Liquid、变量规则、Run 解析 | `internal/common/template`、`pipelinevariable`、`pipeline_run/usecase` | 已实现并覆盖测试 |
| HTTP/Proto | `common.proto`、HTTP mapper、生成 DTO | 已实现；补充 JSON `null` 绑定测试 |
| Pipeline/Repository 前端 | 变量表、详情编辑、i18n | 已实现；请求 DTO 转换与列调整已完成 |
| 活文档 | CI Pipeline/变量指南 | 已同步 |

## Acceptance Checklist

- [x] 唯一默认语法为 Liquid `default` filter，不兼容 shell-style 默认值。
- [x] 阶段脚本和 Artifact 字段统一提取 Liquid 变量引用。
- [x] 详情识别并展示 `repository_dockerfile`、`working_dir` 等阶段变量与来源阶段。
- [x] 同名变量在不同阶段以独立条目保存、展示和编辑。
- [x] 无覆盖时，多阶段分别使用自身 Liquid 默认值。
- [x] Repository、Pipeline Stage、Pipeline 全局和 Liquid 默认值按规定优先级解析。
- [x] Pipeline Stage 覆盖通过 `stage_id` 限定；空 `stage_id` 表示全局覆盖。
- [x] 无默认值的变量在运行解析时返回变量和阶段上下文。
- [x] 非法或不支持的变量表达式返回可读校验/渲染错误。
- [x] Run 快照保留全局和 Stage 变量作用域，Retry 沿用该语义。
- [x] 前端变量更新请求不包含响应专用字段，`default: null` 可由 Proto JSON 正确解析。

## Test Results

| Command | Result |
|---|---|
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `yarn --cwd web test` | Passed: 19 files, 92 tests |
| `task check` | Passed: typecheck, lint, format, golangci-lint |
| `go test ./cmd/... ./internal/...` | Passed |
| `go test ./internal/api/http/codec` | Passed |

## Scope And Risks

- 变量响应 DTO 到请求 DTO 的显式转换是本次 API 契约收敛的一部分，修复了严格 Proto JSON 对 `stage_defaults`、`stage_name` 的拒绝。
- 未启动开发服务器或执行浏览器端到端操作；页面逻辑由 TypeScript 类型检查、全量前端测试和请求 DTO 单测覆盖。
- 已有阶段文本若仍使用 `${NAME:-default}`，必须显式改写为 Liquid；本次不会在运行时兼容该历史语法。

## Conclusion

需求、计划与实际差异一致，验收项和既定检查均已通过。本任务可交付。
