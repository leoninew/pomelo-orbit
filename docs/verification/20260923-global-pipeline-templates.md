# 全局流水线模板与构建阶段验收
最后修改时间: 2026-09-24 11:22:09

Review status: Accepted

Mode: standard

Stage: Verification

## 验证范围

- 验收对象为 `docs/intent/20260923-global-pipeline-templates.md` 与 `docs/plan/20260923-global-pipeline-templates.md` 对应的实现。
- 本次交付边界以暂存区为准，只包含全局 Template Pipeline、全局构建阶段模板及其数据库、SQL、SQLC、应用层和文档变更。
- `20260923-application-pipeline-template-update` 相关 API、快照并发更新和前端契约仍保留在工作区，未纳入本次验收或暂存。

## Intent alignment

- Template Pipeline 使用 `kind=template AND project_id IS NULL` 表示全局资源。
- Template Stage 使用 `kind=template AND project_id IS NULL` 表示全局资源；Application Pipeline 和 Application Stage 继续按 Project 隔离。
- Project ID 仍作为成员资格和资源校验上下文，未取消 API 授权边界。
- 未新增 `source_variables_baseline`、`source_reference_baseline` 或其他模板归属字段。

## Plan alignment

- 三种数据库的 pipeline schema 和 migration 已就地调整，`pipeline_stage.project_id` 支持模板行使用 `NULL`。
- Pipeline SQL、模板阶段 SQL 和 SQLC 代码已区分全局 Template 与项目内 Application 数据。
- UseCase 的模板创建、查询、更新、删除、实例化及名称冲突检查已按全局范围收敛；Application Pipeline 名称仍按 Project 检查。
- Repository 与 UseCase 测试覆盖全局模板跨 Project 可见、全局名称查询和 Application Pipeline 项目隔离。
- 产品概览与 CI Pipeline 活文档已同步全局资源语义。

## Actual diff summary

- Schema/migration：三种数据库将模板 Pipeline 和模板 Stage 的归属调整为全局，保留 Application 数据的 Project 归属。
- Query/SQLC：全局模板查询忽略 Project 归属；Application 查询保持 Project 条件；模板 Stage 结果将数据库 `NULL` 映射为模型层空 Project ID。
- Application：列表输入在 UseCase 层执行 `TrimSpace`；Repository 不再承担流水线输入清洗，仅执行数据库参数映射和空值转换。
- Tests：新增 Repository 全局可见性与项目隔离测试，并更新名称冲突测试的 `kind` 维度。
- Documentation：Intent、Plan、产品概览和 CI Pipeline 指南已描述全局模板与成员授权上下文。

## Expected vs actual

| 预期范围 | 实际结果 |
| --- | --- |
| `sql/migration/{mysql,postgres,sqlite}/000020_pipeline.up.sql` | 已暂存并包含模板全局化 schema 变更 |
| `sql/query/pipeline/pipeline.sql` | 已暂存并包含全局 Template Pipeline/Stage 查询与写入边界 |
| `internal/application/pipeline/usecase/{pipeline.go,stage_template.go}` | 已暂存并包含全局资源校验和应用层输入清洗 |
| `internal/repository/{pipeline.go,impl/sqlc/pipeline/*}` | 已暂存并包含全局查询适配、nullable Stage 映射和测试 |
| `internal/gen/sqlc/pipeline/*` | 已暂存全局模板相关生成代码；未暂存快照版本和并发更新生成代码 |
| `docs/verification/20260923-global-pipeline-templates.md` | 本文新增，记录本次 Verification 证据 |
| Application Pipeline Template Update API、proto、web、快照并发更新 | 未纳入本次暂存区，仍属于另一会话工作 |

## Acceptance checklist

- [x] 任意已加入的 Project 查询 Pipeline 时可看到相同的全局 Template Pipeline；Repository 测试覆盖两个 Project 查询同一模板。
- [x] 任意已加入的 Project 查询构建阶段模板时可看到相同的全局 Template Stage；Repository 测试覆盖跨 Project 查询。
- [x] 创建、更新、删除模板资源的数据库边界不再使用模板 Project 归属；Project 仅保留为成员资格上下文。
- [x] Application Pipeline 只按当前 Project 查询，Application Stage 继续保留所属 Pipeline 的 Project 归属。
- [x] 迁移后的开发库模板记录均为全局记录，Application 记录仍保留 Project 归属。
- [x] 未增加 `source_variables_baseline`、`source_reference_baseline` 或兼容性双轨字段。

## Verification results

- `go test ./cmd/... ./internal/...`：通过；包含 `internal/application/pipeline/usecase`、`internal/repository/impl/sqlc/pipeline` 及其依赖包。
- `task check`：通过；`web` 的 `typecheck`、`lint`、Prettier 检查以及 `golangci-lint` 配置、格式和全量扫描均通过，最终报告 `0 issues`。
- `git diff --cached --check`：通过。
- `DBTALK_DSN_APP` migration：最新版本 `50`，`dirty=false`。
- `DBTALK_DSN_APP` Pipeline 数据：Template global `4` 条，Application project-scoped `4` 条。
- `DBTALK_DSN_APP` Stage 数据：Template global `10` 条，Application project-scoped `9` 条。
- `DBTALK_DSN_APP` 归属完整性检查：Template 带 Project `0` 条，Template Stage 带 Project `0` 条，Application 无 Project `0` 条，Application Stage 无 Project `0` 条。

## Scope deviations

- 当前工作区还包含另一会话的 Application Pipeline Template Update 实现、proto/web 生成代码、快照和运行相关改动；这些文件未进入本次暂存区。
- 开发库中存在历史同名全局模板：Pipeline 有 `Go 构建流水线`、`镜像构建流水线` 两组重复名称，Stage 模板有五组重复名称。由于脚本、版本或引用快照不同，本次未擅自合并或删除。

## Risks

- 历史重复名称会使按名称读取的结果存在歧义；UseCase 的新建和更新名称检查已按全局范围拒绝新增冲突，但历史记录的合并策略仍需产品或数据治理决策。
- 本次验收验证了 Repository、UseCase、数据库状态和工程检查，未额外执行已被另一会话修改的 HTTP/前端 Template Update 流程。

## Incomplete items

- 全局模板代码、schema、开发库归一化和验证均已完成。
- 历史重复模板未处理，作为后续数据治理事项保留，不阻断本次全局归属验收。

## Conclusion

全局流水线模板与构建阶段的实现满足 Intent、Plan 和 Acceptance 要求；代码检查、全量 Go 测试、工程检查及 `DBTALK_DSN_APP` 数据归属检查均通过。本次交付范围可按全局模板任务验收，暂存区未包含另一会话的 Application Pipeline Template Update 工作。
