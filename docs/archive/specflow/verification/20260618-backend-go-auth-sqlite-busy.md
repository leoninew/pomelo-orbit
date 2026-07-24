# backend-go 认证查询 SQLITE_BUSY 修复 验证
最后修改时间: 2026-06-18 18:02:37

Review status: Draft

## Requirement alignment / 需求对齐

需求：`docs/requirement/20260618-backend-go-auth-sqlite-busy.md`（Review status: Accepted）

核对结论：

1. `authz.CurrentUser` 对 `UserById` 的非 `sql.ErrNoRows` 错误不再返回 401 `Invalid token`：已实现。`backend-go/internal/transport/http/handler/authz/authz.go` 中，先判断 `errors.Is(err, sql.ErrNoRows)`，仅该情况返回 401；其他错误记录 `Error` 日志并返回 `503 Authentication service unavailable`。
2. `sql.ErrNoRows` 或用户状态非 `enabled` 仍返回 401：已实现。
3. token verify 失败仍返回 401：已实现。
4. 数据库查询当前用户失败时记录 error 日志，并返回非 401 状态码，建议使用 503：已实现，返回 `http.StatusServiceUnavailable`。
5. SQLite 初始化设置 busy timeout：已实现，`PRAGMA busy_timeout = 5000`。
6. SQLite 初始化启用 WAL：已实现，`PRAGMA journal_mode = WAL`。
7. SQLite 初始化启用 foreign_keys：已实现，`PRAGMA foreign_keys = ON`。
8. authz 测试覆盖：已实现，包含普通数据库错误、`sql.ErrNoRows`、disabled 用户、invalid token 四种场景。
9. db 测试覆盖 SQLite PRAGMA 设置：已实现，验证 busy_timeout、journal_mode、foreign_keys 三项。
10. backend-go 相关测试和全量测试通过：已实现。

## Spec / Plan alignment

无 spec / plan 文档。当前流程模式为 light / 轻量模式，按 requirement / 需求核对。

## Actual diff summary / 实际 diff 摘要

已提交 / staged 的改动文件：

- `M  backend-go/internal/db/db.go`
- `A  backend-go/internal/db/db_test.go`
- `M  backend-go/internal/transport/http/handler/authz/authz.go`
- `A  backend-go/internal/transport/http/handler/authz/authz_test.go`
- `A  docs/requirement/20260618-backend-go-auth-sqlite-busy.md`

实际行为层 diff：

- `authz.CurrentUser` 把原来的 `if err != nil || user.Status != "enabled"` 合并为两个独立判断；`sql.ErrNoRows` 返回 401，其他数据库错误返回 503。
- `openSQLite` 在 `SetMaxOpenConns(1)` 之后、`Ping` 之前调用 `configureSQLite`；若 PRAGMA 执行失败则关闭数据库并返回错误。
- `configureSQLite` 顺序执行三条 PRAGMA：busy_timeout、journal_mode、foreign_keys。

## Expected vs actual changed files / 预期与实际改动对比

预期改动：

- `backend-go/internal/transport/http/handler/authz/authz.go`
- `backend-go/internal/db/db.go`
- 对应测试文件
- 需求文档

实际改动：与预期一致，无额外业务代码扩散。

## Acceptance criteria checklist / 验收清单

- [x] `authz.CurrentUser` 对 `UserById` 的非 `sql.ErrNoRows` 错误不再返回 401 `Invalid token`
- [x] `sql.ErrNoRows` 或用户状态非 `enabled` 仍返回 401
- [x] token verify 失败仍返回 401
- [x] 数据库查询当前用户失败时记录 error 日志，并返回非 401 状态码（实现为 503）
- [x] SQLite 初始化设置 busy timeout
- [x] SQLite 初始化启用 WAL
- [x] SQLite 初始化启用 foreign_keys
- [x] authz 测试覆盖：普通数据库错误、`sql.ErrNoRows`、disabled 用户、invalid token
- [x] db 测试覆盖 SQLite PRAGMA 设置
- [x] backend-go 相关测试和全量测试通过

## Test results / 测试结果

- `go test -C backend-go ./...`：全部通过（cached）
- `go vet -C backend-go ./...`：无输出，未报错
- `gofmt -l` 对四个 changed Go 文件：无输出，已全部格式化

说明：本次复用了项目 `justfile` 中的 backend-go 工具链入口，即 `go test ./...`、`go vet ./...`、`go fmt ./...`。

## Missed or expanded scope / 范围偏差

无范围扩散。未修改前端代码，未触碰 CI worker 并发模型，未修改业务状态机，符合 requirement 中的 non-goal。

## Risk / 风险

1. `PRAGMA journal_mode = WAL` 在网络文件系统或特殊 SQLite 环境下可能受限；当前开发和常规部署预期可用。
2. `busy_timeout = 5000` 只能缓解短暂锁冲突，无法覆盖长期写锁或多进程错误使用 SQLite 的情况。
3. 503 仍会在 UI 上提示失败，但不会触发前端清除登录态；这与 requirement 目标一致。
4. `configureSQLite` 若执行 PRAGMA 失败，会关闭数据库并返回错误；这是启动期失败，不会静默降级。

## Incomplete items / 未完成项

无明显 incomplete items。

## Conclusion / 结论

实现与 light / 轻量模式需求一致，验收项全部通过，测试与 vet 均未报错，代码已格式化。可进入交付审视。

建议 git commit message:

```text
fix(backend-go): return 503 instead of 401 on sqlite busy and improve sqlite init

- authz.CurrentUser returns 530 for non-ErrNoRows UserById failures to avoid
  frontend clearing the token on transient database locks
- keep 401 for invalid token, missing user, and disabled user to preserve
  existing client behavior
- configure sqlite with busy_timeout=5000, WAL, and foreign_keys enabled at
  open time to reduce lock contention between CI worker writes and frontend
  polling reads
- add authz tests covering database error, ErrNoRows, disabled user, and
  invalid token paths
- add db test asserting sqlite pragmas are applied
```
