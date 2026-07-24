# backend-go 引入 golang-migrate 数据库迁移验证
最后修改时间: 2026-07-05 22:21:38

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260705-backend-go-golang-migrate.md` 核对：

1. 已引入 `github.com/golang-migrate/migrate/v4`。
2. 后端迁移入口已改为 `db.MigrateUp`，通过 golang-migrate 的 `iofs` source 与 SQLite/MySQL database driver 执行迁移。
3. 迁移 SQL 已放在仓库根目录：
   - `sql/migration/sqlite`
   - `sql/migration/mysql`
4. 旧 `internal/db/migrator.go`、旧 migrator 测试和 `internal/migrations` 迁移包已删除。
5. 不做历史兼容、不迁移已有数据；旧 `__migration_history` 不再创建，使用 golang-migrate 默认 `schema_migrations`。
6. 旧 checksum、手动排序、执行耗时记录和迁移阶段模板渲染逻辑已移除。
7. MySQL 打开连接时强制启用 `multiStatements`，满足 golang-migrate MySQL driver 执行多语句迁移文件的要求。
8. Docker 构建上下文已补充 `COPY sql ./sql`，确保 embed 的 `backend/sql` 包可参与构建。

## Spec alignment

不适用。light / 轻量模式未创建独立 Spec 文档，按 Requirement 核对。

## Plan alignment

不适用。light / 轻量模式未创建独立 Plan 文档，按 Requirement 与实现 diff 核对。

## Actual diff summary

主要实际改动：

1. 新增 golang-migrate 接入：
   - `internal/db/migration.go`
   - `sql/migrations.go`
2. 新增迁移验证测试：
   - `internal/db/migration_test.go`
3. 调整调用点：
   - `internal/bootstrap/bootstrap.go`
   - `internal/app/app.go`
   - `cmd/backend-go/main.go`
   - 多个 repository / worker / HTTP 测试
4. 调整 MySQL DSN：
   - `internal/db/db.go` 中解析 DSN 并设置 `MultiStatements = true`。
5. 迁移脚本从旧目录迁出并按 golang-migrate 命名：
   - 旧：`internal/migrations/{sqlite,mysql}/v0.1.x__*.sql`
   - 新：`sql/migration/{sqlite,mysql}/00000x_*.up.sql`
6. 删除旧迁移历史表创建语句：
   - 新 SQL 中未再创建 `__migration_history`。
7. 更新 Docker 构建：
   - `Dockerfile`
   - `Dockerfile.cn`
8. 更新依赖：
   - `go.mod`
   - `go.sum`
9. 创建流程文档：
   - `docs/requirement/20260705-backend-go-golang-migrate.md`
   - `docs/verification/20260705-backend-go-golang-migrate.md`

## Expected vs actual changed files

| 预期 | 实际 |
| --- | --- |
| 替换旧手写迁移器 | 已删除 `internal/db/migrator.go`，新增 `internal/db/migration.go` |
| 脚本放到 `sql/migration/sqlite` 和 `sql/migration/mysql` | 已新增 `sql/migration/{sqlite,mysql}/000001` 至 `000007` `.up.sql` 文件 |
| 不做历史兼容 | 未实现 `__migration_history` 到 `schema_migrations` 的状态迁移，旧表创建语句已移除 |
| 保持 SQLite/MySQL 支持 | SQLite 使用 golang-migrate sqlite driver；MySQL 使用 golang-migrate mysql driver，并启用 multiStatements |
| 更新测试为业务语义断言 | `internal/db/migration_test.go` 验证 task queue 可用、admin 权限数据可用、deployment command_text 可写、schema_migrations version=7、旧历史表不存在 |
| Go 后端检查通过 | `go fmt`、`go vet`、`go test` 均通过 |

未发现超出本次迁移机制替换范围的业务 API 或 service 重构。

## Acceptance criteria checklist

- [x] `go.mod` 引入 `github.com/golang-migrate/migrate/v4`。
- [x] 后端数据库初始化路径使用 golang-migrate 执行迁移。
- [x] 不再调用旧 `NewMigrator(...).Up()` 生产路径。
- [x] 迁移 SQL 位于 `sql/migration/sqlite` 与 `sql/migration/mysql`。
- [x] 迁移文件符合 golang-migrate up 文件命名格式。
- [x] SQLite 与 MySQL 均有对应 driver/source 选择逻辑。
- [x] 旧 `__migration_history` 不再作为执行依据。
- [x] 自定义 checksum、排序、耗时记录等旧逻辑已移除。
- [x] 测试断言迁移后的业务可用性，而不只断言内部历史表存在。
- [x] `go fmt ./cmd/... ./internal/... ./sql` 通过。
- [x] `go vet ./cmd/... ./internal/...` 通过。
- [x] `go test ./cmd/... ./internal/...` 通过。

## Command results

```text
$ go fmt ./cmd/... ./internal/... ./sql
# 无输出，命令成功
```

```text
$ go vet ./cmd/... ./internal/...
# 无输出，命令成功
```

```text
$ go test ./cmd/... ./internal/...
?    backend/cmd/backend-go [no test files]
ok   backend/internal/app (cached)
?    backend/internal/apperror [no test files]
?    backend/internal/bootstrap [no test files]
ok   backend/internal/config (cached)
ok   backend/internal/db (cached)
?    backend/internal/infrastructure/logstore [no test files]
ok   backend/internal/logging (cached)
ok   backend/internal/repository (cached)
?    backend/internal/repository/cd [no test files]
ok   backend/internal/repository/ci (cached)
?    backend/internal/repository/model [no test files]
?    backend/internal/repository/project [no test files]
?    backend/internal/repository/role [no test files]
ok   backend/internal/repository/task (cached)
?    backend/internal/repository/user [no test files]
ok   backend/internal/runtimepath (cached)
ok   backend/internal/security (cached)
ok   backend/internal/service/auth (cached)
ok   backend/internal/service/cd (cached)
ok   backend/internal/service/ci (cached)
?    backend/internal/service/project [no test files]
?    backend/internal/service/role [no test files]
?    backend/internal/service/settings [no test files]
?    backend/internal/service/task [no test files]
?    backend/internal/service/user [no test files]
?    backend/internal/status [no test files]
ok   backend/internal/templatex (cached)
ok   backend/internal/transport/http (cached)
?    backend/internal/transport/http/handler/auth [no test files]
ok   backend/internal/transport/http/handler/authz (cached)
?    backend/internal/transport/http/handler/cd [no test files]
?    backend/internal/transport/http/handler/ci [no test files]
?    backend/internal/transport/http/handler/project [no test files]
?    backend/internal/transport/http/handler/role [no test files]
?    backend/internal/transport/http/handler/settings [no test files]
?    backend/internal/transport/http/handler/task [no test files]
?    backend/internal/transport/http/handler/user [no test files]
ok   backend/internal/transport/http/middleware (cached)
ok   backend/internal/transport/http/response (cached)
ok   backend/internal/worker (cached)
ok   backend/internal/worker/handler/cd (cached)
ok   backend/internal/worker/handler/ci (cached)
```

## Missed or expanded scope

1. 未补 `.down.sql` 文件。Requirement 中已记录本次实现按当前自动 up 需求迁移为 `.up.sql`，不额外补空 down 脚本。
2. 未做已有数据库历史状态迁移。用户已明确“不历史兼容，不迁移已有数据”。
3. 未修改前端，未触及 API 路由契约。
4. 额外更新 `Dockerfile` / `Dockerfile.cn` 是必要构建修正：迁移脚本现在在根目录 `sql` 包中，镜像构建必须复制该目录。

## Risks

1. 已有由旧 migrator 初始化过的数据库不会自动平滑升级；需要重建或人工处理。
2. MySQL 多语句迁移依赖 DSN 解析后统一打开 `multiStatements`；如果外部对 DSN 字符串有严格比对，应注意最终连接参数会被标准化。
3. 当前没有 down 迁移能力，回滚仍依赖数据库备份或手工处理。
4. `migrate-status` 输出从逐文件状态变为当前 `version` / `dirty`，这是迁移到 golang-migrate 后的行为变化。

## Incomplete items

无。按本次 light Requirement 范围，代码实现与验证命令均已完成。

## Conclusion

验证通过。实现符合本次需求：后端迁移机制已从自编写 migrator 替换为 golang-migrate，迁移脚本已移动到 `sql/migration/{sqlite,mysql}`，旧历史兼容与数据迁移未实现且符合用户明确约束，Go 后端格式化、静态检查和测试均通过。
