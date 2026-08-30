# PostgreSQL 数据库支持
最后修改时间: 2026-08-30 22:54:33

流程：标准模式 / standard

Review status: Accepted

## Basis

本计划实施已接受的 [Requirement](../requirement/20260830-postgresql-database-support.md)。当前数据库边界如下：

- internal/infrastructure/database 以 *sql.DB 作为连接和事务入口。
- SQLite、MySQL 共用以 ? 为参数占位符的 SQLC 生成代码；SQLC 当前以 MySQL schema/query 生成。
- golang-migrate 从嵌入的 sql/migration/{sqlite,mysql} 读取按版本组织的 up/down migration。
- 现有 PostgreSQL 支持必须覆盖生成查询和事务，而不是只让 Open 或迁移命令能够连接。

## Design decisions

### 1. 保留一套 SQLC 生成物

继续以 MySQL 作为 SQLC 的生成 engine 和类型基线，不为 PostgreSQL 生成另一套同路径的代码，也不在仓储层维护三套实现。原因是现有仓储、tx.DbTX、SQLC DBTX 和数据库类型已经围绕一套共享生成物组织。

PostgreSQL 连接使用 database/sql connector 适配器，在 driver connection 的 ExecContext、QueryContext、Prepare/PrepareContext 路径把 SQL 中未处于字符串、标识符或注释内的 ? 转换为 PostgreSQL $1、$2 参数。该适配器同时覆盖根 *sql.DB 和由它创建的 *sql.Tx，迁移 SQL 不经过该转换。

参数扫描器只处理当前代码实际需要的问号参数语法，并正确跳过单引号字符串、双引号标识符、-- 行注释和 /* */ 块注释；不把 PostgreSQL 原生操作符语义扩展为新的业务能力。

### 2. 使用 lib/pq 和 migrate PostgreSQL driver

增加固定版本的 github.com/lib/pq 直接依赖，使用其 NewConnector 打开 PostgreSQL，并复用 golang-migrate/migrate/v4/database/postgres 的 WithInstance 迁移适配器。连接池参数沿用 MySQL 的服务型默认值，并在 Ping 失败时关闭数据库返回包含 postgres 上下文的错误。

DSN 接受驱动支持的 PostgreSQL URL/keyword-value 格式；文档默认展示 postgres://...?...，不在配置层复制解析逻辑或保存明文 secret。

### 3. PostgreSQL migration 单独维护

新增 sql/migration/postgres/，与当前最高版本和每个 up/down 文件一一对应。文件按现有 SQLite/MySQL migration 的业务语义重写为 PostgreSQL DDL/DML：

- DATETIME/DATETIME(3) 使用 PostgreSQL timestamp 类型或等价精度，并保留当前时间默认值语义。
- LONGTEXT、TINYINT(1)、MySQL JSON、ENGINE=InnoDB 和 UNIQUE KEY 使用 PostgreSQL 等价写法；布尔/整数列的 SQLC 扫描结果与现有 Go 模型保持兼容。
- MySQL 反引号改为 PostgreSQL 标识符引用；保留 "user" 的引用，避免 PostgreSQL 保留字与现有物理表名冲突。
- seed 的数据、顺序、唯一约束和外键关系与当前迁移链一致，不引入新业务数据。

既有 SQLite/MySQL migration 不修改。嵌入声明增加 PostgreSQL glob，迁移 driver 分支增加 PostgreSQL。

### 4. 收敛保留字查询

共享 SQL query 保持 SQLC 当前 MySQL parser 可接受的反引号形式，connector 在 PostgreSQL 执行边界将反引号标识符转换为双引号（包括 user、project 相关查询）。SQLite 与当前 MySQL ANSI_QUOTES session 配置均能继续执行生成 SQL，PostgreSQL 可直接解析转换后的标识符。

### 5. 可空参数显式类型

SQLC/MySQL 生成的重复 `?` 在 PostgreSQL connector 中会成为独立的 `$n`。仅出现在 `IS NULL` 左侧的占位符没有列类型上下文，必须在共享 SQL 中显式声明类型。字符串参数使用 `CAST(sqlc.narg(...) AS CHAR) IS NULL`，时间参数使用 `CAST(sqlc.narg(...) AS DATE) IS NULL`；转换结果只用于空值判断。

不得使用 `LOWER`、`UPPER`、`TRIM` 等无关业务函数推断参数类型。该形式在 SQLite、MySQL 与 PostgreSQL 中均可解析，且将类型约束写入 SQL，而不是隐藏在 connector 规则中。

## Implementation steps

### Step 1 — 配置契约

1. 在 internal/config/config.go 增加 DatabaseDriverPostgres、PostgresConfig 和 DatabaseConfig.Postgres。
2. 在环境变量绑定、启动校验和 secret key 列表中增加 database.postgres.dsn / database__postgres__dsn。
3. 更新 configs/config.yaml、.env.example 和配置单测，覆盖 PostgreSQL DSN 成功加载、空 DSN 失败、未知 driver 错误信息和默认 SQLite 不变。

### Step 2 — PostgreSQL 连接与参数适配

1. 在 internal/infrastructure/database/database.go 增加 PostgreSQL connector 打开流程、连接池设置、Ping 和关闭失败连接的处理。
2. 新增同包内的 PostgreSQL placeholder rebind 实现；保留现有 MySQL ANSI_QUOTES connector 和 SQLite pragma 行为。
3. 在 internal/infrastructure/database/database_test.go 增加不依赖真实数据库的 connector/rebind 测试：多参数转换、字符串/标识符/注释中的 ? 保留、Prepare 路径转换、底层连接错误传播。
4. 用 fake driver connection 覆盖 database/sql 调用的 ExecContext/QueryContext 入口，确保适配不是只对 SQLC 直接调用生效。

### Step 3 — migration 接线和 PostgreSQL 文件

1. 在 sql/migrations.go 的 go:embed 中加入 migration/postgres/*.sql。
2. 在 internal/infrastructure/database/migration.go 引入 PostgreSQL migrate driver，并为 migrationDatabaseDriver 增加 postgres 分支；保持 migration runner 的 source/instance 名称与 driver 常量一致。
3. 基于当前 SQLite/MySQL migration 逐版本新增 PostgreSQL up/down 文件，覆盖现有全部版本和 seed；重点人工检查 user、时间默认值、唯一约束、外键删除顺序、seed 中的字符串/JSON 内容。
4. 扩展 internal/infrastructure/database/migration_test.go 的文件方向一致性检查，使 SQLite、MySQL、PostgreSQL 三套 migration 版本集合一致。

### Step 4 — SQLC 查询兼容

1. 修改 sql/query/user/user.sql 和 sql/query/project/project.sql 中的 user 表引用为 "user"。
2. 执行 task sqlc，检查生成物只包含预期的引用变化和生成器版本不变；不手改 internal/gen/sqlc 的业务逻辑。
3. 用现有 repository 集成测试确认 SQLite 查询仍可执行；PostgreSQL E2E 重点覆盖用户/项目查询，以验证保留字处理、占位符转换和类型扫描。

### Step 5 — PostgreSQL E2E 与文档

1. 在 internal/test/e2e/ 增加 PostgreSQL 配置驱动的可选迁移 E2E，使用独立环境变量（BACKEND_GO_POSTGRES_E2E_CONFIG 或仅进程内传递的 BACKEND_GO_POSTGRES_E2E_DSN），没有配置时跳过并说明原因。
2. E2E 在空库执行迁移两次、读取 migration version/dirty 状态、检查系统 seed 数量和关键 Gateway 拓扑；覆盖代表性 repository/事务操作，以及可选筛选为空和非空时间范围时的 SQLC list/count 参数绑定。
3. 更新 README.md、docs/architecture/backend.md、configs/config.yaml 和 .env.example 中的数据库支持列表、配置名称和 PostgreSQL 运行说明。
4. 不启动开发服务器；真实 PostgreSQL 实例由验收环境通过 DSN 提供。

## Files to change

- 配置：internal/config/config.go、internal/config/config_test.go、configs/config.yaml、.env.example。
- 连接与迁移：internal/infrastructure/database/database.go、对应 connector/rebind 测试、internal/infrastructure/database/migration.go、migration_test.go、sql/migrations.go。
- 依赖与生成：go.mod、go.sum、sql/query/user/user.sql、sql/query/project/project.sql、由 task sqlc 产生的受影响 internal/gen/sqlc/** 文件。
- PostgreSQL schema migration：新增 sql/migration/postgres/*.sql 全部当前版本的 up/down 文件。
- E2E：新增或调整 internal/test/e2e/postgres_e2e_test.go 及共享测试辅助代码。
- 文档：README.md、docs/architecture/backend.md、本 Plan；实现完成后再创建 Verification 文档。

## Verification plan

### 静态和单元验证

- task sqlc：确认查询和 schema 可重复生成。
- go test ./internal/config ./internal/infrastructure/database：覆盖配置、连接适配、占位符转换和 migration 文件集合。
- go test ./cmd/... ./internal/...：覆盖现有 Go 单测和集成测试。
- task check：按仓库入口执行前端/Go format、lint、typecheck；确认生成代码和手写适配器均通过。

### SQLite 回归

- 现有 SQLite migration 单测执行完整迁移两次、seed 断言和外键检查。
- 现有 repository 测试继续使用 SQLite，重点检查更新后的 "user" 查询和事务路径。

### PostgreSQL 验收

- 使用 PostgreSQL 实例和 BACKEND_GO_POSTGRES_E2E_CONFIG 或 BACKEND_GO_POSTGRES_E2E_DSN 执行 go test ./internal/test/e2e -run PostgreSQL（具体测试名以实现为准）。
- 验证空库完整 migration、重复 migration、version clean、seed 拓扑、用户/项目查询、分页参数、可选筛选为空和非空时间范围、布尔/JSON 文本和任务领取事务。
- 若环境没有 PostgreSQL，记录可执行的静态/SQLite 结果，并明确真实 PostgreSQL migration/业务 E2E 未完成，不将其伪称为通过。

## External verification environment

- dbtalk executable 已在当前环境 PATH 中可用。
- 用户已提供 PostgreSQL 测试 DSN，并授权通过环境变量 DBTALK_LEON_K12_DSN 供 dbtalk 使用；DSN 内容不写入仓库、配置样例、日志或命令参数。
- 实施/验证时先用 dbtalk database query 只读检查目标数据库状态；如需清空目标库，使用 dbtalk database exec 的单条显式写操作，并在执行前确认清理范围只限于该测试数据库。
- Go E2E 不读取 dbtalk 的 Python DSN 格式。运行时从同一授权值生成仅存在于临时目录的测试配置文件，测试结束后清理；不新增凭据文件或提交测试配置。
- dbtalk 用于数据库状态检查和必要的测试库准备，应用 schema 与 seed 仍由 Go migration 代码执行；不得用 dbtalk 代替本次 PostgreSQL migration 验收。

## Blockers

- 当前没有已知代码阻塞。
- 真实 PostgreSQL 迁移和业务 E2E 依赖外部实例或 CI service；没有 DSN 时只能跳过该部分并记录风险。

## Assumptions

- PostgreSQL 数据库和账号由部署者预先创建，应用只负责连接目标数据库并执行迁移。
- PostgreSQL 版本使用主流受支持版本，不依赖特定版本专属语法；驱动 DSN 由 lib/pq 解析。
- ? 在当前业务 SQL 中仅表示参数占位符；若未来引入 PostgreSQL ? JSON 操作符，应先扩展明确的 SQL 方言边界，而不是让通用扫描器猜测。
- 外部 PostgreSQL E2E 为可选环境依赖，不改变默认本地 SQLite 流程。

## Risks

1. driver-level placeholder rebind 覆盖面不足会导致 SQLC、手写 SQL 或事务路径中某一类调用仍失败；通过 fake driver 的所有调用入口和真实 PostgreSQL E2E 降低风险。
2. 40 个 PostgreSQL migration 文件存在 DDL/DML 细节差异；通过文件集合测试、逐版本审查和空库真实执行发现问题。
3. "user" 物理表名在 PostgreSQL 中需要一致引用；任何未更新的 query 或 seed 引用都会在业务路径暴露，需用全仓 rg 和 PostgreSQL 查询 E2E 检查。
4. 未连接真实 PostgreSQL 时，Go 编译和 SQLite 测试不能证明 PostgreSQL DDL、类型扫描和锁语义正确；验证报告必须保留该边界。

## Rollback

- 实现阶段若尚未部署 PostgreSQL，可回退连接/config/code/doc 变更并删除未执行的 PostgreSQL migration 新文件；不触碰已执行的 SQLite/MySQL migration。
- PostgreSQL migration 一旦被实际环境执行，不能通过删除文件回滚；必须使用对应 down migration 或运维备份恢复，并遵循现有迁移状态管理。
- 不执行 git add、git commit 或 git push。

## User review notes

2026-08-30：用户授权使用 DBTALK_LEON_K12_DSN 对 PostgreSQL 测试数据库执行空库准备和真实测试；凭据只通过环境变量传递。

2026-08-30：用户确认开始 Implementation，采用保留 MySQL SQLC 生成物并在 PostgreSQL connector 边界转换占位符的方案。

2026-08-30：用户要求移除以 `LOWER(sqlc.narg(...))` 推断 PostgreSQL 参数类型的实现，改用共享 SQL 中的显式 `CAST` 类型声明。
