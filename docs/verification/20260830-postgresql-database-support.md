# PostgreSQL 数据库支持验证
最后修改时间: 2026-08-30 23:07:45

流程：标准模式 / standard

Stage: `Verification`

Review status: `Accepted`

## Requirement Alignment

- `database.driver` 现支持 `sqlite`、`mysql`、`postgres`；PostgreSQL 使用 `database.postgres.dsn`，配置校验、环境变量绑定、启动 secret 注册和示例配置已同步。
- `internal/infrastructure/database` 使用 `lib/pq` 打开、Ping 和管理 PostgreSQL 连接池；SQLC 的 MySQL `?` 占位符与反引号标识符在 PostgreSQL connector 边界转换，根连接、Prepare 和事务路径共享该行为。
- `sql/migration/postgres/` 已包含与 SQLite/MySQL 一致的 40 个 up/down 迁移文件；嵌入与 `golang-migrate` PostgreSQL driver 已接线。
- 可空 SQLC 参数在空筛选时使用显式类型上下文：文本为 `CAST(sqlc.narg(...) AS CHAR) IS NULL`，时间为 `CAST(sqlc.narg(...) AS DATE) IS NULL`。E2E 覆盖 nil、非空 ULID、搜索字符串和时间筛选，避免重新出现 `could not determine data type of parameter`。

## Spec Alignment

不适用。该任务采用 standard 流程，未创建独立 Spec。

## Plan Alignment

- 配置、connector、migration 嵌入、SQLC 查询/生成物、E2E 和活文档均已落地，符合 Plan 的五个实施步骤及后续“可空参数显式类型”设计决策。
- 三套 migration 目录均含 40 个文件；`migration_test.go` 覆盖集合与方向一致性。
- 真实 PostgreSQL E2E 使用配置中的 PostgreSQL 环境运行，且不启动或重启开发服务。

## Actual Diff Summary

- 配置与依赖：新增 `lib/pq`，并扩展配置结构、示例、README 与设置服务的 PostgreSQL 选项。
- 数据库适配：新增 `postgres.go` 及其测试；迁移 runner、嵌入资源与迁移集合校验支持 PostgreSQL。
- SQL 兼容：共享 SQLC 查询使用跨方言 Boolean 字面量和对 `user` 的安全引用；56 个文本可空筛选使用 `CAST(... AS CHAR)`，时间筛选使用显式日期 cast。守护测试拒绝无类型或函数型的 `sqlc.narg(...) IS NULL` 检查，随后由 `task sqlc` 重新生成。
- 数据库资源：新增 PostgreSQL 迁移链；按后续明确要求，同步修改三种方言 Gateway seed 的 `restart_policy` 为 `unless-stopped`。
- 验收：新增 PostgreSQL migration/repository/transaction/optional-filter E2E；可空文本筛选同时覆盖 nil、非空 ULID 和搜索字符串，并通过真实 MySQL 到 PostgreSQL JSONL 覆盖导入核对 45 张表、1007 行。

## Expected vs Actual Files

Plan 预期的配置、数据库基础设施、migration、SQL query/生成物、E2E、README 与架构文档均出现在实际 diff 中。

实际额外项：`sql/migration/sqlite/000037_seed_gateway.up.sql` 与 `sql/migration/mysql/000037_seed_gateway.up.sql` 因用户后续明确要求修改既有 seed 而变更；没有无关的生产代码重构。

## Acceptance Checklist

- [x] PostgreSQL driver 与 DSN 配置可校验、打开、Ping，失败路径保留数据库类型上下文。
- [x] `MigrateUp`、`MigrateTo` 和 migration version 读取支持 PostgreSQL，三种方言的 migration 集合一致。
- [x] 真实 PostgreSQL 环境迁移状态为 `version=37`、`dirty=false`，关键 seed、用户与项目可查询。
- [x] PostgreSQL 执行 SQLC 查询、事务回滚、分页/可空筛选参数和时间范围筛选；nil 与非空文本/时间参数均无未定类型参数错误。
- [x] SQLite/MySQL 共享 SQLC 生成流程与全量 Go 测试未回归。
- [x] `task sqlc` 可从查询定义重复生成生成物，未手工修改 SQLC 业务代码。
- [x] 配置样例、README 与活架构文档已说明 SQLite、MySQL、PostgreSQL 三种数据库。

## Test Results

| 命令 / 检查 | 结果 |
| --- | --- |
| `task sqlc` | 通过；生成物由当前 SQL query 重建。 |
| `go test ./cmd/... ./internal/...` | 通过。 |
| `task check` | 通过；web typecheck/lint/format 与 Go format/lint 均无问题。 |
| `go test ./internal/test/e2e -run '^TestPostgreSQLMigrationE2E$' -count=1` | 通过；覆盖 migration、seed、SQLC、事务，以及 nil/非空文本/时间可空筛选。 |
| DBTalk 只读类型语法探测 | PostgreSQL 与 MySQL 均可执行 `CAST(... AS CHAR)`、`CAST(... AS DATE)`；不读写业务表。 |
| PostgreSQL 日志（E2E 后 90 秒） | 仅 checkpoint，无本轮数据库错误。 |
| MySQL -> PostgreSQL JSONL 覆盖核对 | 45 表、1007 行逐表计数一致；`project.is_active` 为 PostgreSQL `boolean`。 |

## Scope Deviations

- Requirement 原则是不修改已执行 SQLite/MySQL migration；用户后续明确授权就地修改各方言 seed，因此 `000037_seed_gateway.up.sql` 的目标组件重启策略同步变更。现有已迁移数据库不会因文件改写自动回填。
- 外部 `pomelo-dbtalk` 的 Boolean 导入适配为完成真实 MySQL -> PostgreSQL 覆盖验收而修复，不属于本仓库生产 diff。

## Risks And Incomplete Items

- PostgreSQL E2E 目前由本地授权实例执行，CI 尚未提供 PostgreSQL service；持续集成仍不能自动复验该路径。
- MySQL 未提供应用层 E2E；本轮仅以实际只读 cast 语法探测、SQLC MySQL 生成和现有 SQLite/Go 回归覆盖共享查询变更。
- PostgreSQL connector 将业务 SQL 中的 `?` 视为参数占位符。未来若引入 PostgreSQL JSON `?` 操作符，需要先扩展方言边界与测试。
- 本次 JSONL 验收产生的系统临时文件因终端策略拒绝删除，仍需在工作站层面清理；该文件不在仓库或配置中。

## Conclusion

实现与已接受的 Requirement、Plan 对齐；当前生成、静态检查、全量 Go 测试和真实 PostgreSQL E2E 均通过，显式类型声明替代了函数型参数推断。用户已接受 Verification，并知悉 CI 覆盖、MySQL 应用层 E2E 和临时数据清理风险。
