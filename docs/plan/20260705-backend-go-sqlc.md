# backend-go 引入 sqlc 数据库查询工具链计划
最后修改时间: 2026-07-05 23:15:44

Review status: Accepted

## Basis

已接受需求文档：

```text
docs/requirement/20260705-backend-go-sqlc.md
```

## Implementation steps

### Step 1: 确认工具链边界

1. 使用 sqlc v2 配置。
2. schema 输入指向现有迁移目录，不修改已执行迁移文件。
3. query 输入新增 `sql/query`。
4. 生成代码输出到 `internal/db/sqlc`。
5. 使用 `database/sql` 兼容接口，repository 保持现有 `*sqlx.DB` 字段以便事务和动态查询继续可用。

### Step 2: 添加 sqlc 配置和 query 文件

新增：

```text
sqlc.yaml
sql/query/*.sql
internal/db/sqlc/*.go  # 由 sqlc 生成
```

优先整理静态查询：

1. 用户、角色、项目等基础权限域的按 ID / code / name 查询。
2. CI/CD repository 中简单按主键读取、存在性检查、删除等静态 SQL。
3. 后台任务 repository 中 dequeue / mark done / mark failed 等核心静态 SQL，如 sqlc 支持可读实现。

### Step 3: 接入生成代码

1. 在 `internal/repository.Store` 中可选持有 sqlc `Queries`，或各 repository 内部创建 `sqlc.New(db)`。
2. 保持现有 repository 构造函数签名，减少 service/bootstrap 改动。
3. 将适合迁移的方法替换为 sqlc 调用，并在 repository 内转换为现有 `model` 类型。
4. 对动态 WHERE、`sqlx.In`、事务内批量循环等保留现状，但避免新增同类散落 SQL。

### Step 4: 工具依赖记录

1. 若项目没有 tools 记录方式，新增 `tools.go` 使用 build tag 记录 `github.com/sqlc-dev/sqlc/cmd/sqlc`。
2. 更新 `go.mod` / `go.sum`。
3. 生成代码后纳入仓库工作区。

### Step 5: 格式化和检查

按项目约束运行：

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

如果 sqlc 生成工具不可用，优先通过 `go run github.com/sqlc-dev/sqlc/cmd/sqlc generate` 执行，并记录结果。

## Files to change

预计新增/修改：

```text
sqlc.yaml
sql/query/*.sql
internal/db/sqlc/*.go
tools.go 或 internal/tools/tools.go
go.mod
go.sum
internal/repository/**/repository.go
```

不应修改：

```text
sql/migration/**/*.sql
```

## Verification plan

Implementation 阶段先运行生成、格式化和 Go 后端检查。Verification 阶段若用户后续要求进入，再创建 `docs/verification/20260705-backend-go-sqlc.md` 并对照 diff、验收标准和命令结果做完整验收记录。

## Blockers

无必须等待用户决策的 blocker。TermBridge 未找到 sqlc 现成实现，已按 requirement 记录为假设和风险。

## Assumptions

1. sqlc 生成代码可以提交到仓库，而不是要求部署环境安装 sqlc 后再生成。
2. 本次“收拾数据库查询”以生产 repository 层为主，不包含测试 fixture SQL 的全面迁移。
3. 保留 `sqlx` 作为连接、事务和动态查询辅助是允许的；本次不是替换数据库驱动。

## Risks

1. sqlc 对 MySQL 与 SQLite 双 schema 的支持可能需要取舍；若单配置无法覆盖，优先确保当前测试数据库和生产查询语义一致。
2. 迁移范围过大可能引入回归，需控制在静态查询和清晰 CRUD。
3. 生成类型可能与现有 model 不一致，需要显式转换，避免把生成层泄露到 service 层。

## Rollback

1. 删除 `sqlc.yaml`、`sql/query` 和 `internal/db/sqlc`。
2. 从 repository 恢复直接 sqlx 查询调用。
3. 移除 tools 依赖和 go.mod/go.sum 中仅 sqlc 引入的依赖。

## User review notes

- 用户已明确要求直接实现，本计划按阶段流转规则标记为 Accepted。
