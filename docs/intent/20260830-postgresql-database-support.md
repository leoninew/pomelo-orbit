# PostgreSQL 数据库支持
最后修改时间: 2026-08-30 20:37:18

- Flow mode: `standard`
- Stage: `Requirement`
- Review status: `Accepted`

## Background

项目当前支持 SQLite 和 MySQL。数据库连接由 `internal/infrastructure/database` 统一打开，迁移由 `golang-migrate` 加载 `sql/migration/{sqlite,mysql}`，仓储查询主要由 SQLC 生成。

当前 SQLC 配置以 MySQL 为 engine，生成查询使用 `?` 占位符；SQLite 和 MySQL 均可直接执行该形式。PostgreSQL 使用不同的参数占位符和部分不同的 DDL/类型语法，因此仅增加一个连接驱动不足以形成可用支持。

## Goal

增加 PostgreSQL 作为第三种可选数据库，使用户可以通过配置选择 PostgreSQL，并完整运行服务、worker 和迁移命令，同时保持 SQLite/MySQL 的现有行为不变。

支持范围包括：

- 配置结构、环境变量绑定和启动校验。
- PostgreSQL `database/sql` 驱动的连接、Ping、连接池和错误处理。
- PostgreSQL 对应的完整迁移链及当前 seed 数据。
- SQLC 生成查询、手写数据库访问、事务和后台任务在 PostgreSQL 上的参数绑定与类型兼容。
- PostgreSQL 配置示例、活文档和针对性测试。

## Non-goal

- 不重新设计现有领域模型、表结构或仓储接口。
- 不修改已经执行过的 SQLite/MySQL 迁移文件，也不在本需求中压缩迁移历史。
- 不移除或改变 SQLite/MySQL 支持，不引入兼容别名或新旧数据库逻辑并存。
- 不改变前端功能、API 路由或数据库之外的部署架构。
- 不要求本地开发环境默认启动 PostgreSQL；真实数据库验收是否进入 CI 需要单独确认。

## User scenarios

### PostgreSQL 配置启动

用户在配置中设置：

```yaml
database:
  driver: postgres
  postgres:
    dsn: "postgres://user:password@127.0.0.1:5432/pomelo_orbit?sslmode=disable"
```

或使用对应的 `POMELO_ORBIT_DATABASE__POSTGRES__DSN` 环境变量。配置加载和校验成功后，server、worker、MCP 和 migrate 命令均能使用该数据库。

### PostgreSQL 初始化和重复迁移

在空 PostgreSQL 数据库上执行迁移，所有当前版本的 up migration 顺序成功，迁移状态为 clean，并包含现有系统 seed 数据。重复执行迁移不报错；查询迁移状态可以返回版本和 dirty 状态。

### 服务业务访问

使用 PostgreSQL 运行服务后，现有仓储的读取、创建、更新、删除、分页、搜索和事务操作能够执行。SQLC 查询和直接执行的 SQL 均使用正确的 PostgreSQL 参数绑定；时间、可空字段、布尔值和 JSON 文本的读写结果与现有数据库保持一致。

### 既有数据库回归

SQLite 默认配置和 MySQL 配置继续通过原有连接、迁移和仓储测试，不因 PostgreSQL 适配而改变行为。

## Acceptance

1. `database.driver` 接受 `sqlite`、`mysql` 和 `postgres`；选择 PostgreSQL 时必须要求非空 `database.postgres.dsn`，错误信息明确指向 PostgreSQL 配置。
2. PostgreSQL DSN 能被连接层打开并 Ping；连接失败会在启动阶段返回带数据库类型的可诊断错误。
3. `MigrateUp`、`MigrateTo` 和 `ReadMigrationVersion` 支持 PostgreSQL，且 PostgreSQL 迁移目录与现有迁移版本/方向完整对应。
4. PostgreSQL 空库可完成当前迁移链和 seed；重复迁移保持 clean，关键表和系统 seed 数据可查询。
5. SQLC 生成查询、手写 SQL 和事务路径均能在 PostgreSQL 执行，尤其覆盖参数占位符、分页、可空时间/字符串、布尔值、JSON 文本和任务领取事务。
6. 现有 SQLite 单元/迁移测试和 MySQL 配置路径不回归；新增 PostgreSQL 配置、迁移文件一致性和参数适配测试。
7. SQLC 配置或生成流程仍可从仓库中的 schema/query 重复生成，生成物不需要手工维护业务逻辑。
8. `configs/config.yaml`、`.env.example`、README/后端架构活文档中数据库支持列表和 PostgreSQL 配置方式保持一致。

## Open questions

- PostgreSQL 的最低支持版本是否有部署约束？当前草稿按主流 PostgreSQL 版本、无需使用特定新版本特性处理。
- DSN 是否只接受 `postgres://` / `postgresql://` URL，还是同时要求支持 lib/pq 的 keyword/value 格式？当前示例使用 URL 形式，具体实现应以选定驱动的稳定输入格式为准。
- 是否需要在 CI 中启动真实 PostgreSQL 服务执行迁移和仓储 E2E？若没有现成 CI 服务，本地/环境变量驱动的可选 E2E 可以先作为验收路径。
- SQLC 是否继续以 MySQL 作为唯一生成 engine，并在数据库适配层转换参数，还是改为 PostgreSQL engine 并为旧数据库增加适配？该选择应在 Plan 阶段根据代码量、生成类型和三种数据库兼容性验证确定。

## Decisions

- PostgreSQL driver 名称统一使用 `postgres`，配置节点统一使用 `database.postgres.dsn`。
- 既有 SQLite/MySQL migration 文件视为已执行历史，不修改；新增 PostgreSQL migration 文件与现有版本号对齐。
- 迁移和 seed 的最终语义以当前 SQLite/MySQL 迁移链和代码实际使用的 schema 为准，不引入新的业务表或数据流程。
- 过程文档使用中文为主，代码标识符和数据库标准术语保留英文。

## Risks and assumptions

- PostgreSQL 与 MySQL 的参数占位符不同，SQLC 生成代码和直接 SQL 的适配是主要实现风险；必须覆盖事务中的 `*sql.Tx`，不能只处理根连接。
- MySQL 的 `DATETIME`、`TINYINT(1)`、`JSON`、反引号和多列 DDL 不能机械复制到 PostgreSQL；迁移需要逐版本检查并通过真实 PostgreSQL 执行验证。
- 当前仓库没有默认的 PostgreSQL 服务配置；没有可连接实例时，只能完成静态、生成和 SQLite 回归验证，真实 PostgreSQL 迁移/业务验收将标记为未完成。
- 假定 PostgreSQL 数据库由用户提前创建，应用只负责连接该数据库并执行 schema migration，不负责创建数据库本身。

## User review notes

2026-08-30：用户要求开始 Plan，视为接受当前需求范围和未决事项按计划阶段的工程假设推进。
