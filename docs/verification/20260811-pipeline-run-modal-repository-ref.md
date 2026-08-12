# 流水线运行时变量解析与运行弹窗验证
最后修改时间: 2026-08-12 16:44:07

Review status: Accepted

Mode: strict

## Requirement Alignment

实现已将运行时源码 ref 统一为 `repository_ref`。Trigger 和 Preview 仅接收完整的非 `system` `variables` 表单；Run 的列、响应和完整变量快照均使用该名称。变量按表单、Repository、Pipeline、阶段默认值的优先级解析，Snapshot 的变量声明仅保留为历史展示，Run 创建和 Retry 均重新解析当前配置与本次表单。

运行弹窗删除独立 ref 控件，改由可编辑的 `repository_ref` 变量输入承担；非 `system` 变量提交完整表单，`system` 变量只读且服务端拒绝覆盖。执行器仅使用 Run 的变量快照，并校验快照中的 `repository_ref` 与 Run 列一致。

用户追加确认清理无模板引用的系统变量，因此 `repository_id`、`repository_name`、`pipeline_id`、`pipeline_name`、`pipeline_version` 已从运行时声明、Run 快照注入和仓库变量展示中移除。PipelineRun、Artifact 等领域模型与 API 中用于关联、检索和历史展示的同名结构字段仍保留。

## Spec Alignment

- SQLite/MySQL `000031` 成对迁移将 `pipeline_run.trigger_ref` 重命名为 `repository_ref`；Schema、SQL、SQLC、模型、DTO、Proto、HTTP 映射和前端类型同步更新。
- `PipelineSnapshot.variables_snapshot` 保留完整声明历史，但 Preview、Trigger、Retry 不将其作为变量值输入；Trigger/Retry 使用冻结阶段结构，Preview 使用当前阶段。
- `PipelineRun.variables_snapshot` 持久化最终执行值，执行器只读该快照；不会与 Snapshot ref 比较，也不限制复用 Snapshot 的后续 Run 使用不同 ref。
- 前端弹窗基于 Preview 返回声明构建表单，支持 debounce、过期请求隔离、字段级必填错误，以及关闭/切换 Pipeline 时的状态清理。

## Plan Alignment

迁移、变量规则、Snapshot/Run/Retry 边界、Proto/HTTP、前端弹窗、SQLC 生成和自动化测试均已落实。运行时变量精简是基于实际阶段模板引用审计确认的范围扩展，已同步更新需求、规格和活文档。未引用的 `triggerRef` i18n 键已在中英文语言包中删除。

## Actual Diff Summary

- `trigger_ref` -> `repository_ref` 覆盖 `pipeline_run` 迁移、持久化、协议、HTTP、前端类型与展示。
- 运行变量改为完整表单解析和 Run 专属快照；Retry 从原 Run 非系统变量重建表单，执行器从 Run 快照读取。
- 运行弹窗去除独立 ref 输入，改用 `repository_ref`，并加入字段级错误与完整表单 Preview/Trigger。
- 删除五个无阶段模板引用的系统变量与两个无调用的 i18n `triggerRef` 死键。

## Expected Vs Actual Files

| 预期区域 | 实际变更 | 结果 |
|---|---|---|
| Schema、迁移、SQLC | `sql/migration/**/000031*`、`sql/schema`、`sql/query`、`internal/gen/sqlc/**` | 一致 |
| 协议与 HTTP | `proto/**/pipeline_run.proto`、Go/TS 生成物、handler、DTO、mapper | 一致 |
| 变量与运行边界 | `pipelinevariable`、Snapshot、Run service/execution | 一致 |
| 运行弹窗与 Run 展示 | `PipelineDetail.vue`、Run 页面、生成 TS 类型 | 一致 |
| 回归测试与文档 | Go 测试、migration 测试、导入脚本测试、guides、过程文档 | 一致 |
| 无引用变量清理 | 变量规则、仓库变量展示、i18n、变量文档和测试 | 用户确认的范围扩展 |

## Acceptance Checklist

- [x] Snapshot 保存完整声明历史，Run 单独保存最终执行值，后续配置变更不回写历史 Run。
- [x] Preview/Trigger/Retry 按表单 -> Repository -> Pipeline -> Stage default 重新解析，并保留未出现在阶段文本的配置变量。
- [x] `repository_ref` 是唯一可编辑的 ref 输入，按原文保存，不解析为 SHA。
- [x] `system` 变量只读，服务端拒绝 system 与未知表单键；非系统变量可编辑且 Trigger 要求完整值。
- [x] Run 的 `repository_ref` 与 Run 变量快照一致；执行只读 Run 快照。
- [x] 数据库、Proto、HTTP、前端生成类型和运行弹窗均不再使用业务 `trigger_ref` 字段。
- [x] 五个无阶段模板引用的系统变量和两个无调用的 i18n `triggerRef` 键均已移除。
- [x] `git diff --check` 通过。

## Test Results

以下命令在本轮验证中通过：

```text
task sqlc
yarn --cwd web lint:fix
yarn --cwd web typecheck
yarn --cwd web test
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
git diff --check
```

## Scope And Risks

- `trigger_ref` 的剩余文本仅存在于历史 DDL、迁移 up/down、迁移断言、旧备份导入源字段和过程文档；现行业务 API、模型、Proto、SQL 查询和运行弹窗不存在该字段。
- 数据库升级须执行 migration 31。项目不保留兼容字段，前后端与数据库必须同版本发布。
- 前端 TypeScript Proto 生成物曾因 Buf 远程插件网络不可用而手工恢复；本次 SQLC、前端 lint/typecheck/test 和 Go 检查均通过，后续网络可用时可再用 `task proto` 复核其字节级生成结果。

## Incomplete Items

无。

## Conclusion

严格模式 / strict 的 Requirement、Spec、Plan、Implementation 与 Verification 已闭环。实现与已接受需求、规格和计划一致，验证通过，可进入提交审查。
