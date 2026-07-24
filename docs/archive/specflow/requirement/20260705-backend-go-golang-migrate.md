# backend-go 引入 golang-migrate 数据库迁移
最后修改时间: 2026-07-05 21:51:11

Review status: Accepted

## Background

当前后端使用项目自编写的迁移器执行 embedded SQL：

- 迁移执行逻辑位于 `internal/db/migrator.go`。
- SQL 脚本当前位于 `internal/migrations/sqlite` 和 `internal/migrations/mysql`。
- `internal/migrations/migrations.go` 通过 `embed.FS` 暴露 `sqlite/*.sql` 与 `mysql/*.sql`。
- 现有迁移历史表为 `__migration_history`，自维护文件名、checksum 和执行耗时。

本次需求希望引入官方/社区通用迁移库 `github.com/golang-migrate/migrate/v4` 替换当前自编写迁移执行机制，并将迁移脚本组织到仓库根目录 `sql\migration` 下的 `sqlite` 与 `mysql` 目录。

## Goal

1. 使用 `github.com/golang-migrate/migrate/v4` 作为数据库迁移执行入口，替换当前手写的迁移顺序、历史记录、checksum 校验与执行逻辑。
2. 保持 SQLite 与 MySQL 两类数据库驱动的迁移能力。
3. 将迁移 SQL 按用户要求放置到：
   - `sql\migration\sqlite`
   - `sql\migration\mysql`
4. 后端启动或初始化数据库时继续能自动执行必要迁移。
5. 迁移脚本命名、目录结构与 golang-migrate 的 source driver 约定保持一致，避免继续依赖自定义 `v0.1.0__name.sql` 解析规则。
6. 删除或停止使用项目自编写迁移器中与 golang-migrate 重叠的执行历史、checksum、排序和事务封装逻辑。

## Non-goal

1. 不修改已经执行过的迁移 SQL 的业务含义或表结构语义。
2. 不引入新旧迁移机制并存的兼容层；项目处于活跃开发期，迁移机制应完成替换而非双轨运行。
3. 不在本次需求中重构业务 repository、service 或 HTTP API。
4. 不更改数据库配置模型，除非 golang-migrate 接入必须补充最小必要参数。
5. 不执行 `git add`、`git commit`、`git push` 等 Git 写操作。

## User scenarios

1. **新环境启动后端**
   - 后端连接 SQLite 或 MySQL。
   - 系统通过 golang-migrate 读取对应数据库类型的迁移目录。
   - 所有待执行迁移按版本顺序执行完成。

2. **已有开发环境继续启动**
   - 如果已有数据库已经由旧迁移器初始化，本次实现需要明确处理策略。
   - 在项目活跃开发且不保留兼容层的约束下，默认可以要求开发环境重建数据库，或由用户确认是否需要一次性切换策略。

3. **新增迁移脚本**
   - 开发者按 golang-migrate 文件命名约定添加 `up` / `down` 脚本。
   - 后端迁移执行不再需要修改 Go 代码中的自定义脚本列表或 checksum 逻辑。

## Acceptance

1. `go.mod` 引入 `github.com/golang-migrate/migrate/v4` 及 SQLite/MySQL/file 或 io/fs source 所需驱动依赖。
2. 后端数据库初始化路径使用 golang-migrate 执行迁移，不再调用当前自编写迁移器的 `Up()` 执行流程。
3. 迁移 SQL 放置在 `sql\migration\sqlite` 和 `sql\migration\mysql` 目录下，并符合 golang-migrate 对版本号、方向和文件名的要求。
4. SQLite 与 MySQL 均有对应 source 路径选择逻辑，不能硬编码只支持单一数据库。
5. 旧的 `__migration_history` 自维护历史表不再作为新迁移机制的执行依据；使用 golang-migrate 默认 schema migrations 表或明确配置后的等价机制。
6. 自定义 checksum 校验、手动排序、手动记录执行耗时等重复逻辑被移除或停止暴露给生产路径。
7. 相关测试更新为验证业务语义：迁移执行后关键表/初始数据/约束可用，而不是只断言内部历史表存在。
8. Go 后端检查通过：
   - `go fmt ./cmd/... ./internal/...`
   - `go vet ./cmd/... ./internal/...`
   - `go test ./cmd/... ./internal/...`

## Open questions

1. golang-migrate 要求常规文件名形如 `<version>_<name>.up.sql` 与 `<version>_<name>.down.sql`。本次实现先按当前只执行 up 的需求迁移为 `.up.sql`，不额外补空 down 脚本。
2. 已有开发数据库从 `__migration_history` 切换到 golang-migrate 的 `schema_migrations` 不做自动迁移状态；用户已明确不做历史兼容、不迁移已有数据。
3. 现有迁移中包含的 `{{ ... }}` 和 `{% ... %}` 是业务模板文本，不是迁移阶段变量；迁移执行阶段按纯 SQL 文件交给 golang-migrate。

## Decisions

1. 采用轻量模式 / light，仅先沉淀 Requirement，不在本阶段修改产品代码。
2. 本需求目标是替换迁移执行机制，不做新旧迁移双轨兼容。
3. 测试断言应聚焦迁移后的业务结果和可用性，避免只检查迁移历史表存在等实现细节。

## Risk

1. golang-migrate 的版本表与现有 `__migration_history` 不兼容；已有数据库如果不做状态迁移，可能重复执行初始化 SQL 或报对象已存在。
2. 将单文件迁移改为 up/down 文件可能造成大量文件重命名；若未保留语义一致性，可能影响迁移顺序和可读性。
3. 如果现有 SQL 依赖 Go 模板渲染，直接交给 golang-migrate 执行会失败或产生错误 SQL。
4. `sql\migration` 目录当前不存在，路径决策会影响 embed/source driver 选择和包结构。
5. MySQL 与 SQLite 的 golang-migrate database driver 接入方式不同，错误处理和 dirty version 恢复策略需要在实现时明确。

## User review notes

- 用户请求：`light 引入 github.com/golang-migrate/migrate/v4 替换自己编写的数据库迁移，脚本放在 k12-force\sql\migration 下的 sqlite 和 mysql 目录`。
- 用户已要求开始实现，并明确：不做历史兼容，不迁移已有数据。
- 用户后续修正脚本路径：脚本放在仓库根目录 `sql\migration` 下的 `sqlite` 和 `mysql` 目录。
- 用户指出不应把旧 `migrator.go` 包装成新方案；实现应迁移到 golang-migrate 新方案并去除旧迁移器抽象。
