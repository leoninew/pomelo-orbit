# 应用流水线跟进模板流水线更新验证
最后修改时间: 2026-09-24 12:10:12

Review status: Accepted

Mode: strict

## Intent alignment

- 复用既有 `pipeline_snapshot`，未新增快照表或第二种快照抽象。
- 应用流水线继续通过 `source_pipeline_id + source_template_version` 记录来源。
- 普通模板编辑路径不调用快照创建；实例化、升级成功和运行前路径复用快照创建逻辑。
- 未恢复 `source_variables_baseline` 或 `source_reference_baseline`。
- 同步 HTTP Handler 未自行开启事务，写入依赖既有 HTTP UoW middleware。

## Plan alignment

- 已完成快照按 `pipeline_id + pipeline_version` 精确读取。
- 已完成模板升级预览、Apply、Proto、HTTP 路由和前端 API 契约。
- 已完成按 `source_template_stage_id` 匹配阶段、模板新增阶段合并、应用变量保留和版本 CAS 校验。
- 已修复流水线级升级替换制品定义时丢失 Application `component_name` 绑定的问题，并补充回归测试。
- 已同步 SQLC、Go Proto 和 Web Proto 生成物。

## Actual diff summary

- 新增模板升级合并逻辑及单元测试：`internal/application/pipeline/usecase/template_update.go`、`internal/application/pipeline/usecase/template_update_test.go`。
- 扩展快照创建、来源快照读取、应用版本条件更新和运行前快照路径。
- 新增 Pipeline 级升级预览/Apply 的 HTTP、Proto、前端 API 契约。
- 更新流水线过程文档、产品文档和 CI Pipeline 指南。
- 本任务未新增数据库迁移文件；开发库通过 `dbtalk` 就地清理和归一化。
- 验证期间工作区并行出现 `sql/migration/*/000049_global_pipeline_templates.*` 和对应迁移测试，属于全局模板归属任务的扩展变更，未归入本任务实现结论。

## Acceptance checklist

- [x] 两个 baseline 字段不存在；代码与 SQL 查询无残留方案。
- [x] `pipeline_snapshot` 仍是唯一快照表，未按 `kind` 拒绝快照。
- [x] Application 来源 ID+版本字段存在，开发库现有 Application 来源引用有效。
- [x] 开发库模板 Pipeline 2 条、模板 Stage 5 条均为全局记录；Application 记录仍带项目归属。
- [x] 升级预览读取旧来源快照和当前模板，不调用创建快照入口。
- [x] Apply 校验应用版本和来源版本，并通过 `UpdateApplicationPipelineIfVersion` 执行 CAS 更新。
- [x] 阶段合并按 `source_template_stage_id`，阶段级更新入口保持独立。
- [x] 模板升级更新阶段内容时保留 Application 的制品 `component_name` 映射。
- [x] 开发库迁移元数据为 `version=48, dirty=false`，无不存在的迁移版本记录。

## Test results

- `task sqlc`：通过。
- `task check`：通过；包含 Web typecheck、lint、format、Go format 和 golangci-lint。
- `go test ./cmd/... ./internal/...`：通过。
- `go test -count=1 ./internal/application/pipeline/usecase ./internal/repository/impl/sqlc/pipeline ./internal/application/pipeline_run/usecase`：通过。
- `yarn --cwd web lint:fix`：通过。
- `yarn --cwd web typecheck`：通过。
- `git diff --cached --check`：通过。
- `task proto`：通过；Go Proto 和 Web Proto 均生成完成。

## Database verification

- `DBTALK_DSN_APP` 的 `schema_migrations` 当前为 `48 / dirty=false`；工作区并行的 `000049_global_pipeline_templates` 可在下次启动时继续归一化迁移。
- `source_variables_baseline` 和 `source_reference_baseline` 均不存在。
- 当前 Application 来源引用无失效模板。
- 模板 Pipeline 和模板 Stage 的 `project_id` 已就地归一为 `NULL`；Application Pipeline/Stage 保留项目归属。
- 本次数据库验收未新增快照记录；现有库中快照数量为 0，未用当前应用值伪造历史来源快照。

## Risks and incomplete items

- 不执行真实浏览器端到端流程；该流程不属于本任务验收范围。
- 当前开发库没有快照数据，无法通过 live data 证明旧来源快照读取；代码单测和 SQLC/Go 检查已覆盖对应接口，但仍建议在开发库创建一个新实例后手工验证预览与 Apply。
- 工作区存在未归属本任务的 `000049_global_pipeline_templates` 新迁移及测试，提交前必须由负责人决定是否拆分或纳入全局模板任务。

## Conclusion

实现与自动化验收通过，制品映射缺口已修复；按本次验收范围，验证标记为 `Accepted`。工作区并行的 `000049_global_pipeline_templates` 变更仍不属于本任务交付范围。
