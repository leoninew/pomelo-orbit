# 流水线模板与应用流水线分离验证
最后修改时间: 2026-08-07 18:55:18

Review status: Draft

流程模式: 严格 / strict

## Requirement alignment

- `Pipeline(kind=template|application)` 已取代旧 Template、全局 Stage 和 Stage 级 Application/Version 绑定；Application Pipeline 在创建时固定来源 Template 和 Repository，Application 仅在制品存在 Component 映射时绑定。
- Template 不再运行或创建 Snapshot；Application Pipeline 使用版本化 Snapshot，Run 只引用 Snapshot。
- Component 映射位于 `docker_image` 制品声明；Run 成功后一次 fork Version 并更新全部映射 Component，Run binding 记录生成 Version。
- 旧 Repository Template trigger/Webhook 入口和前端已移除；本轮没有引入替代 Webhook。
- 离线 SQLite 切换脚本已完成并在两个开发库完成完整性、外键和旧表清理检查；旧运行历史仅在显式确认后删除。
- 详情变量声明与运行对话框均通过同一个无副作用预览接口解析运行时值，保留 Repository 覆盖、Pipeline 自定义值和用户覆盖的既有优先级；预览不会创建 Snapshot、Run、任务或 Version binding。

## Spec and plan alignment

- SQLC 的 CI 聚合已使用独立 target schema 和 query 目录；Repository 只调用生成 Query，不手写查询。
- 跨生命周期引用采用 ID 与展示快照的逻辑外键；Stage、Snapshot、Run、Artifact 的聚合内关系保留物理外键。
- HTTP 写事务使用标准 `Request.Context` 传播，避免 SQLite 单连接下事务占连接、服务查询却回落根 DB 的请求挂起。
- Run fork Version 与 `generated_version_id` 回写已在同一短事务中执行；嵌套持久化调用复用 context transaction。
- Application Pipeline 变量预览复用 Run 创建的声明解析和运行时变量规则；详情以默认输入加载，运行对话框按当前 ref 与覆盖重新计算，加载或失败期间不会允许提交。

## Actual diff

- 重建 Pipeline 领域、SQLC、Repository、Usecase、Proto、HTTP 路由与 Web 页面，完成 Template 与 Application Pipeline 分离。
- 新增 `scripts/migrate_pipeline_template_instances.py` 和操作说明，执行开发 SQLite 的就地切换。
- 修正 PipelineRun/Artifact 分页查询的 SQLC 命名参数；新增全仓 SQL query 静态检查，禁止单语句混用命名参数与匿名 `?`。
- 修正 Pipeline 列表/计数查询：使用完整命名参数，生成 API 不再含 `ColumnN`，并以 SQLite 回归测试执行过滤和分页。
- 移除 Version 删除检查对旧 `pipeline_stage_build_version_binding` 的残留读取，保留运行中 Service 对 Version 的阻断语义。
- 对按 ID 查询的 PipelineSnapshot、PipelineRun 与 Artifact 强制 Project 存在并执行成员校验，避免 `NULL project_id` 绕过鉴权。
- 将 Application Pipeline 的 Application 绑定改为可选：实例化时空值不传递为空字符串；仅 Component 映射存在时校验 Application 与来源 Version 策略。Snapshot 的 Application 展示字段和 Proto 契约相应改为可空/optional。
- 为运行时变量增加 `variable-preview` 入口，并在 Pipeline 详情和运行对话框复用同一前端解析调用；运行对话框使用加载状态，预览失败时禁用确认。
- 流水线列表拆分 Repository 与 Application 列，并以“未绑定”展示空 Application；模板和 Application Pipeline 均提供删除动作，删除失败保留在确认模态窗中反馈，列表不再提供“配置”动作。
- 在 `D:\SourceCodes\mywork\best-practices` 新增 `check-sqlc-parameter-binding`、`check-transaction-management` 两个已校验 Skill，沉淀本次 SQLC 与事务发现。

## Acceptance checklist

- [x] Template 与 Application Pipeline 分离，Application Pipeline 不在运行时重选 Application/Repository；Repository 固定且 Application 可选。
- [x] Component 映射由制品声明维护，单个成功 Run 最多生成一个 Version。
- [x] Snapshot、Run、Artifact 和 Version Component 保留来源与展示追溯字段。
- [x] 手动 Trigger 与 Retry 收敛到同一 Run 创建路径。
- [x] SQLC 列表、制品分页和按 Run 制品查询不再混用命名/匿名参数，并有运行时覆盖。
- [x] 旧表残留检查通过，Version 删除不再读取删除后的 CI 旧表。
- [x] 变量预览复用既有运行时变量解析和覆盖机制；详情展示默认值，运行对话框重算并在加载/失败时阻止提交。
- [x] 列表清晰展示 Repository 与可选 Application，所有 Pipeline 提供物理删除入口，删除错误不穿透为页面级 toast。
- [ ] 浏览器手工验收：从 Template 创建 Application Pipeline、运行只输入 ref/变量、制品 Component 映射、Snapshot/Artifact 追溯和 Webhook 入口移除。

## Validation results

| Command or check | Result |
| --- | --- |
| `./bin/sqlc generate` | PASS |
| `./bin/buf generate` | PASS |
| `./bin/buf generate . --template web/buf.gen.yaml --output web` | PASS |
| `go fmt ./cmd/... ./internal/...` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| Pipeline 相关定向 Go 测试 | PASS |
| `go test ./cmd/... ./internal/...` | FAIL：仅 `internal/bootstrap.TestMigrateAndMigrationVersion` 仍断言迁移版本为 `30`，当前 schema 版本为 `31` |
| `yarn --cwd web lint:fix` | PASS（前端变更完成后执行） |
| `yarn --cwd web typecheck` | PASS（前端变更完成后执行） |
| `yarn --cwd web test` | PASS，10 files / 62 tests（前端变更完成后执行） |
| 主开发库和备份库 SQLite `integrity_check` / `foreign_key_check` | PASS |
| SQLC 命名与匿名参数全仓静态扫描 | PASS |
| 新增 SQLC/事务 Best Practices Skill 校验 | PASS |

## Risks and incomplete items

1. 完整 Go 测试被 `internal/bootstrap.TestMigrateAndMigrationVersion` 的旧迁移版本期望阻断；这是测试基线跟随当前 `31` 号 schema 的后续修复，不应通过回退业务模型或 schema 解决。
2. CI target schema 和切换脚本仍将 `project_id` 声明为 nullable；业务入口已经拒绝缺失 Project 的记录。若要将“Pipeline 必属 Project”提升为物理约束，需要单独设计 SQLite 表重建和已切换开发库的离线变更，不能在运行时兼容。
3. 手工 UI 验收尚未执行，尤其应覆盖可选 Application 的实例化、无 Application 时隐藏 Component/Version 策略、详情默认变量值、运行时 ref/覆盖重算，以及预览失败反馈。

## Conclusion

主要业务切换、SQLC 参数故障、事务原子性、事务端口分层和变量预览行为已通过自动验证，且 Go/TypeScript 生成均已完成。由于完整 Go 测试的迁移版本断言仍待修复，以及 UI 手工验收尚未完成，本验证保持 Draft。
