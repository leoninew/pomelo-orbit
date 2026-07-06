# backend-go 引入 sqlc 数据库查询工具链
最后修改时间: 2026-07-05 23:15:44

Review status: Accepted

## Background

当前后端 repository 层大量直接使用 `sqlx` 拼接和执行 SQL，查询分散在 `internal/repository/**/repository.go` 中。项目已经在上一轮工作中将迁移脚本整理到 `sql/migration/{sqlite,mysql}`，但查询层仍缺少统一、可生成、可检查的 SQL 组织方式。

用户要求参考 `D:\SourceCodes\mywork\TermBridge-go` 引入 sqlc 工具链，收拾本项目数据库查询，并直接实现。实际调研发现 TermBridge 当前仓库未检索到 `sqlc.yaml` 或 sqlc 生成代码；可参考的是其以 `database/sql` 为基础、按领域 repository 收口数据库访问的方向，而 sqlc 工具链需在本项目按 sqlc 官方约定补齐。

## Goal

1. 在本项目引入 sqlc 工具链配置，让数据库查询可以从 `sql/query` 下的 SQL 文件生成 Go 代码。
2. 优先整理生产 repository 层的静态 SQL，减少业务代码中散落的手写 `SELECT` / `INSERT` / `UPDATE` / `DELETE` 字符串。
3. 保持现有 SQLite 与 MySQL 双数据库运行能力；生成查询应兼容现有 `?` 参数风格。
4. 保持对外 API、service 行为和迁移脚本语义不变。
5. 不修改已经执行的迁移文件。
6. 不执行 `git add`、`git commit`、`git push` 等 Git 写操作。

## Non-goal

1. 不重构 HTTP API 路由、DTO 或前端调用。
2. 不在本次变更中改变表结构或新增业务迁移。
3. 不引入新旧逻辑并存的兼容层；若某类动态 SQL 不适合一次性迁入 sqlc，应明确保留为动态查询并说明边界，而不是制造双轨封装。
4. 不主动启动、停止或重启开发服务器。
5. 不追求把测试 fixture 中的 SQL 全部迁入 sqlc；测试辅助建表/插数 SQL 可继续留在测试内。

## User scenarios

1. **开发者修改数据库查询**
   - 在 `sql/query` 下修改具名 SQL。
   - 运行 sqlc 生成命令。
   - Go 编译期能发现参数或返回列不匹配。

2. **后端服务调用 repository**
   - service 层仍通过现有 repository 方法访问数据库。
   - repository 内部使用 sqlc 生成的查询方法承接静态 SQL。
   - 分页、搜索、动态 IN、事务等复杂动态场景保持现有明确代码路径或逐步拆分。

3. **SQLite / MySQL 环境运行**
   - 生成查询不破坏现有 SQLite 测试。
   - MySQL 仍保留当前 `?` 参数与字段名语义。

## Acceptance

1. 仓库根目录新增 sqlc 配置文件，配置 schema、query 和生成代码位置。
2. `sql/query` 下存在按领域组织的 sqlc query 文件。
3. 生成代码进入 `internal/db/sqlc` 或等价内部包，不暴露为公共 API。
4. 至少核心生产 repository 的静态查询改为调用 sqlc 生成方法；不适合 sqlc 的动态 WHERE、动态 IN、批量事务循环需保留时必须有清晰边界。
5. `go.mod` / `go.sum` 引入 sqlc 运行所需依赖或 tool 记录方式，避免工具链隐式依赖。
6. 不修改 `sql/migration/**` 已有迁移文件。
7. 后端检查目标：
   - `go fmt ./cmd/... ./internal/...`
   - `go vet ./cmd/... ./internal/...`
   - `go test ./cmd/... ./internal/...`

## Open questions

1. TermBridge 未找到可直接复制的 sqlc 工具链文件，本次按 sqlc 官方配置和本项目结构落地。
2. sqlc 对动态 SQL 拼接、可变 `IN (?)` 展开和部分事务内循环插入并不总是更简洁；此类查询本次以不牺牲清晰度为边界。
3. SQLite 与 MySQL schema 类型存在差异，生成代码以当前 Go model 和 sqlx 使用方式为约束，优先确保 SQLite 测试与通用 `database/sql` 接口可用。

## Decisions

1. 采用标准模式 / standard；用户已明确“直接实现”，因此 Requirement 与 Plan 记录后标记为 `Accepted` 并进入 Implementation。
2. sqlc 查询文件放在仓库根目录 `sql/query`，生成包放在 `internal/db/sqlc`。
3. repository 对外方法签名保持不变，避免 service / handler 层连锁重构。
4. 对动态过滤、分页搜索、事务内批量写入暂不强行抽象成低可读性的 sqlc 查询。

## Risk

1. 一次性迁移所有手写 SQL 可能造成大范围行为风险；实现需优先选择静态查询和简单 CRUD，避免重写复杂业务流程。
2. sqlc 生成类型与现有 `model` 类型可能不完全一致，需要 adapter 或 overrides；不应为了生成类型破坏领域模型。
3. sqlc 工具若未安装，生成命令可能失败；实现阶段需记录实际命令结果。
4. 双数据库方言差异可能限制单套 query 的覆盖范围。

## User review notes

- 用户请求：`参考 D:\SourceCodes\mywork\TermBridge-go 引入 sqlc 工具链，收拾本项目的数据库查询，直接实现`。
- 用户已授权直接进入实现；本需求文档按该指令标记为 Accepted。
