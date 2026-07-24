# backend-go 引入 sqlc 数据库查询工具链验证
最后修改时间: 2026-07-06 10:38:59

Review status: Accepted

## Basis

已接受过程文档：

```text
docs/requirement/20260705-backend-go-sqlc.md
docs/plan/20260705-backend-go-sqlc.md
```

## Requirement alignment

| Requirement acceptance | Verification result |
| --- | --- |
| 仓库根目录新增 sqlc 配置文件，配置 schema、query 和生成代码位置。 | 已新增 `sqlc.yaml`，schema 指向 `sql/migration/sqlite`，queries 指向 `sql/query`，生成输出到 `internal/db/sqlc`。 |
| `sql/query` 下存在按领域组织的 sqlc query 文件。 | 已新增并按领域拆分 `sql/query/*.sql`，包括 `user.sql`、`role.sql`、`project.sql`、`background_task.sql`、`repository.sql`、`webhook.sql`、`credential.sql`、`pipeline_template.sql`、`application.sql`、`config_file.sql`、`service_config.sql`、`route.sql`。 |
| 生成代码进入内部包，不暴露为公共 API。 | 已生成到 `internal/db/sqlc`，repository 内部调用并通过 `internal/repository/dbmodel` 转换，不向 service/handler 暴露 sqlc 类型。 |
| 至少核心生产 repository 的静态查询改为调用 sqlc；动态 WHERE、动态 IN、批量事务循环保留时边界清晰。 | `project`、`user`、`role`、`task` repository 的静态查询已接入 sqlc；动态分页搜索、`sqlx.In`、方言时间表达式和事务内循环写入继续保留在 repository 内。 |
| `go.mod` / `go.sum` 引入 sqlc 运行所需依赖或 tool 记录方式。 | 已新增 `tools.go` 记录 `github.com/sqlc-dev/sqlc/cmd/sqlc`，并更新 `go.mod` / `go.sum`。 |
| 不修改 `sql/migration/**` 已有迁移文件。 | staged diff 未包含 `sql/migration/**`。 |
| 后端检查目标：`go fmt`、`go vet`、`go test`。 | 本次 Verification 已运行并通过。 |

## Plan alignment

| Plan step | Verification result |
| --- | --- |
| 使用 sqlc v2 配置，schema 指向现有迁移目录，query 输入新增 `sql/query`，生成输出 `internal/db/sqlc`。 | 已实现。 |
| 优先整理静态查询，保留动态 WHERE、`sqlx.In`、事务内批量循环。 | 已实现。 |
| repository 构造函数签名保持稳定，各 repository 内部创建 `sqlc.New(db)`。 | 已实现，`project`、`user`、`role`、`task` repository 均保留原构造签名并持有 `queries *dbsqlc.Queries`。 |
| 生成代码后纳入仓库工作区。 | 已生成并暂存 `internal/db/sqlc/*.go`。 |
| 运行生成、格式化、vet、test。 | 已运行并通过。 |

## Actual diff summary

本任务相关 staged diff：

```text
38 files changed, 2805 insertions(+), 160 deletions(-)
```

主要范围：

```text
docs/requirement/20260705-backend-go-sqlc.md
docs/plan/20260705-backend-go-sqlc.md
go.mod
go.sum
sqlc.yaml
tools.go
sql/query/*.sql
internal/db/sqlc/*.go
internal/repository/dbmodel/convert.go
internal/repository/project/repository.go
internal/repository/role/repository.go
internal/repository/task/repository.go
internal/repository/user/repository.go
```

## Expected vs actual changed files

### Expected

```text
sqlc.yaml
sql/query/*.sql
internal/db/sqlc/*.go
tools.go 或 internal/tools/tools.go
go.mod
go.sum
internal/repository/**/repository.go
```

### Actual task-related staged files

```text
docs/plan/20260705-backend-go-sqlc.md
docs/requirement/20260705-backend-go-sqlc.md
go.mod
go.sum
internal/db/sqlc/application.sql.go
internal/db/sqlc/background_task.sql.go
internal/db/sqlc/config_file.sql.go
internal/db/sqlc/credential.sql.go
internal/db/sqlc/db.go
internal/db/sqlc/models.go
internal/db/sqlc/pipeline_template.sql.go
internal/db/sqlc/project.sql.go
internal/db/sqlc/querier.go
internal/db/sqlc/repository.sql.go
internal/db/sqlc/role.sql.go
internal/db/sqlc/route.sql.go
internal/db/sqlc/service_config.sql.go
internal/db/sqlc/user.sql.go
internal/db/sqlc/webhook.sql.go
internal/repository/dbmodel/convert.go
internal/repository/project/repository.go
internal/repository/role/repository.go
internal/repository/task/repository.go
internal/repository/user/repository.go
sql/query/application.sql
sql/query/background_task.sql
sql/query/config_file.sql
sql/query/credential.sql
sql/query/pipeline_template.sql
sql/query/project.sql
sql/query/repository.sql
sql/query/role.sql
sql/query/route.sql
sql/query/service_config.sql
sql/query/user.sql
sql/query/webhook.sql
sqlc.yaml
tools.go
```

### Scope notes

当前工作区仍存在大量未暂存的 HTTP handler / proto / web 相关改动。这些不属于本 sqlc 任务 staged diff，但本次全量 Go 检查会编译当前工作区的未暂存 Go 文件；本轮检查结果已通过。

## Acceptance checklist

- [x] 新增 sqlc 配置文件。
- [x] 新增按领域拆分的 query 文件。
- [x] 生成代码位于 `internal/db/sqlc`。
- [x] repository 外部方法签名保持不变。
- [x] sqlc 生成类型未泄露到 service/handler。
- [x] 核心静态查询接入 sqlc。
- [x] 动态 WHERE、`sqlx.In`、事务内批量循环、方言时间表达式保留清晰边界。
- [x] 未修改 `sql/migration/**`。
- [x] `go fmt ./cmd/... ./internal/...` 通过。
- [x] `go vet ./cmd/... ./internal/...` 通过。
- [x] `go test ./cmd/... ./internal/...` 通过。

## Command results

### sqlc generate

```text
$env:CGO_ENABLED='0'; go run github.com/sqlc-dev/sqlc/cmd/sqlc generate
```

结果：通过，无输出。

### Go format

```text
go fmt ./cmd/... ./internal/...
```

结果：通过，无输出。

### Go vet

```text
go vet ./cmd/... ./internal/...
```

结果：通过，无输出。

### Go test

```text
go test ./cmd/... ./internal/...
```

结果：通过。关键输出摘要：

```text
?    backend/cmd/backend-go [no test files]
ok   backend/internal/app (cached)
ok   backend/internal/config (cached)
ok   backend/internal/db (cached)
?    backend/internal/db/sqlc [no test files]
ok   backend/internal/repository (cached)
ok   backend/internal/repository/ci (cached)
ok   backend/internal/repository/task (cached)
ok   backend/internal/service/auth (cached)
ok   backend/internal/service/cd (cached)
ok   backend/internal/service/ci (cached)
ok   backend/internal/transport/http (cached)
ok   backend/internal/worker (cached)
ok   backend/internal/worker/handler/cd (cached)
ok   backend/internal/worker/handler/ci (cached)
```

## Missed or expanded scope

- 未迁移所有手写 SQL；CI/CD 复杂动态查询、分页搜索、`IN (?)` 展开、事务内循环写入仍保留在 repository 中。这符合 requirement 和 plan 中的边界。
- 本次新增了 CI/CD query 的生成基础，但并未大规模接入 CI/CD repository；该部分主要为后续迁移提供具名 SQL 和生成代码。
- 用户在实现阶段要求进一步拆分 query 文件，并明确 CI/CD 文件名不需要 `ci_` / `cd_` 前缀；最终拆分已按该要求执行。

## Risks

1. sqlc 当前配置基于 SQLite migration 生成；MySQL 路径依赖现有 `?` 参数和字段语义保持一致，尚未做真实 MySQL 端到端运行验证。
2. `go.mod` 因 sqlc tool 依赖引入较多间接依赖，这是 `go run github.com/sqlc-dev/sqlc/cmd/sqlc generate` 的工具链成本。
3. 当前工作区含有大量非本任务未暂存改动；提交前应继续保持 staged diff 与其它任务分离。

## Incomplete items

无本任务范围内未完成项。

## Conclusion

本次 sqlc 工具链引入和首批 repository 静态查询迁移符合已接受的 Requirement 与 Plan。生成、格式化、vet 和测试均已通过。建议保持当前暂存边界，只将 sqlc 任务相关 staged diff 作为独立提交处理。