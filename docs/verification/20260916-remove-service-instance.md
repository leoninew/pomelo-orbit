# 移除服务实例语义验收
最后修改时间: 2026-09-16 18:19:07

Review status: Accepted

Mode: standard

## Intent alignment

实现已移除 Service 的 `instance_key` 和由 `default` 实例键派生的领域、部署、HTTP、Proto、MCP 与 Web 语义。一个 Application 仍可创建多条 Service；运行态身份为 Project 内唯一的 `service.code`，而不是 Application 加实例键。

创建页面在选中 Application 后仅预填 `<application-code>-default` 作为可编辑的 code 建议。该字面量不代表默认实例；服务端按当前 Project 检查 code 冲突，不阻止同 Application 下第二条 Service，也不提前检查端口或挂载冲突。

Gateway 通过唯一受管 Service 的 ID 操作，既有 `traefik-default` code 保持不变但不再具有默认实例含义。`Environment.state` 已从存储、模型和对外契约删除，Probe 状态字段保留。

## Spec alignment

标准模式不单独创建 Spec；按已接受的 Intent 核对。实现未提供旧 HTTP、Proto 或 MCP 字段的 alias、fallback、双路径或 Proto `reserved`。

## Plan alignment

计划中的持久化迁移、SQLC/Proto 生成、Service/Deployment/Gateway/MCP 收敛、Web 创建与展示清理、Environment state 删除、活文档和回归测试均已完成。实施期间删除了随默认 Service 筛选失效的 `isActiveServiceStatus`，使 Go lint 恢复通过。

## Actual diff summary

- 新增 000043/000044 三数据库迁移，分别移除 `service.instance_key` 和 `environment.state`；SQLite 表重建保留 Service 子表关系、Environment 目标与 Probe 数据，MySQL/PostgreSQL 使用对应的 `ALTER TABLE`。
- 从 Service、Deployment、Environment 模型、SQL 查询、SQLC repository、HTTP/Proto 映射、MCP schema 与运行时目标解析中删除实例键和环境 state。
- Service 创建、基础更新和部署链路只使用 Service ID、Application ID、Version ID 与 code；Project 内 code 冲突仍拒绝，多 Service 不再受 Application 级单例 guard 限制。
- Gateway、Route、初始化与运行时工具使用其受管 Service ID/code，不再按 `default` 或 instance key 定位。
- Web 删除 Service 实例字段及 `xxx / default` Service 展示；创建表单保留可编辑 code 输入并在选择 Application 时预填 code 建议。
- 更新 CD 模型、后端架构、部署和 MCP 操作活文档，生成 SQLC 与 Proto 客户端/服务端代码。

## Expected vs actual changed files

实际任务变更覆盖计划中的 `sql/migration/**`、`sql/schema/**`、`sql/query/**`、`proto/orbit/v1/**`、`internal/{model,application,repository,api}/**`、生成 SQLC/Proto、`web/src/{gen,i18n,views}/**`、活文档和相关测试。未增加兼容层，也未修改归档文档。

工作区同时存在早于本任务的 Project scope 改造、HTTP mapper 文件命名调整、分析文档和 `PipelinePage.vue` 格式化等变更；它们没有作为本任务的验收对象，也未被回退或暂存。

## Acceptance criteria checklist

- [x] Service schema、模型、repository 查询和部署投影不再包含 `instance_key`；Project 内 `service.code` 保持唯一。
- [x] 同一 Application 可以创建多条 Service；第二条 Service 只在 code 与当前 Project 已有 code 冲突时被拒绝。
- [x] Service 创建、更新、列表、详情、部署和 Gateway 响应不再暴露实例键或默认服务字段。
- [x] Web 不再展示服务实例、实例键或 `xxx / default` 格式的 Service 标识；选择 Application 时预填且允许编辑 Service code。
- [x] Gateway 通过唯一受管 Service 操作，不使用 `instance_key == "default"` 识别目标；`traefik-default` code 保持稳定。
- [x] MCP schema、输入和输出不含 `instance_key`，运行时操作要求显式 Service ID。
- [x] Proto 未新增 `reserved`，未保留旧字段或运行时 fallback。
- [x] `Environment.state` 不再持久化、读取或输出；目标和 Probe 行为继续使用 target/probe 字段。
- [x] SQLite 从迁移 42 升级的测试验证旧 Service/子表数据保留、两个 Service 可共用 Application、Project code 唯一约束保留，以及两个被删除列不存在。

## Test results

| 命令或检查 | 结果 |
| --- | --- |
| `task sqlc` | 通过；SQL 定义与 SQLC 生成代码同步。 |
| `task proto` | 通过；删除契约字段后的 Proto 代码同步。 |
| `yarn --cwd web lint:fix` | 通过。 |
| `yarn --cwd web format:fix` | 通过。 |
| `yarn --cwd web typecheck` | 通过。 |
| `task check -- --fix` | 执行并修复 Go 格式。 |
| `task check` | 通过：Web typecheck、ESLint、Prettier、golangci-lint 配置、格式与静态检查均通过。 |
| `go test ./cmd/... ./internal/... ./sql` | 通过：全部 Go 包通过。 |
| `go test ./internal/application/deployment/usecase` | 通过：删除失效 Gateway 辅助函数后的受影响包复验。 |
| `go test ./internal/infrastructure/database -run TestMigrateUpSQLiteRemovesServiceInstanceAndEnvironmentState -count=1` | 通过。 |
| `yarn --cwd web test` | 通过：25 个测试文件、114 个测试。 |
| `rg` 残留检查 | 通过：生产 Go、SQL schema/query 与 Web source 无实例字段、默认服务字段或环境 state；仅迁移旧 schema fixture 与 MCP 的“schema 不应包含字段”断言保留 `instance_key` 字符串。 |
| `rg -n "reserved" proto` | 通过：无输出。 |
| 缩写大小写审查 | 通过：本次新增手写 Go 标识符保持项目 `Id` 与协议名 `SSH`/`HTTP`/`TLS` 约定。 |
| `git diff --check` | 通过。 |

## Missed or expanded scope

未修改已有 Service code，因此历史 code 中的 `-default` 仍是稳定运行标识的一部分，不是实例语义。未做离线重命名或资源冲突预检，符合 Intent 非目标。

迁移行为的自动化升级测试在 SQLite 执行。MySQL 与 PostgreSQL 的 000043/000044 DDL 已按各自方言审阅，但当前验证环境没有连接实际 MySQL/PostgreSQL 实例执行迁移。

没有启动、停止或重启开发服务器，也没有执行 Git 暂存、提交或推送。

## Risks and incomplete items

- 这是有意的破坏性 schema 与 API 变更，仍在使用旧 `instance_key` 或 `environment.state` 的外部 HTTP、Proto、MCP 调用方必须与本版本同步升级。
- 多 Service 的端口、挂载、容器资源或用户自定义网络冲突会在部署时由 Docker 暴露，不会在 Service 创建阶段拦截。
- MySQL/PostgreSQL migration 尚缺一次连接真实目标数据库的执行验证；SQLite 的升级、数据保留和约束已自动覆盖。
- 未启动浏览器进行人工 UI 流程；前端表单和文本收敛由 typecheck、Vitest、lint/format 与 API 契约测试覆盖。

## Conclusion

Service 实例与 Environment state 语义已按 Intent 和 Plan 直接删除。系统以 Project 内唯一的 Service code 和显式 Service ID 维持多 Service 运行边界，Gateway 不再依赖默认实例；生成、静态检查、Go/Web 测试、SQLite 升级测试及 diff 检查均通过。除真实 MySQL/PostgreSQL 迁移执行尚待环境验证外，没有未完成项。
