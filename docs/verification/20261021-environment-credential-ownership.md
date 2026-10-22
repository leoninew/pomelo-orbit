# Verification / 验证：环境凭据以环境为归属主表
最后修改时间: 2026-10-21 14:13:03

## Review status / 审查状态

Accepted

主会话已按用户“完成 SpecFlow 验证”的授权完成当前交付范围审查。结论为通过；业务数据库升级及外部数据库运行验证不包含在该结论中。

## Intent alignment / 意图对齐

- 对照[意图](../intent/20261021-environment-credential-ownership.md) A01–A08，凭据已通过 `environment_id` 直接归属环境，环境先保存，随后才创建凭据。
- 当前关系为一环境零或一份凭据；主表删除凭据存储列，凭据子表删除 Project 归属列。任务快照继续保存执行时的凭据身份。
- 保存、初始化、类型切换、Probe、目标解析、Web/CLI 会话及项目交接均采用新的关系；权限、加密方案和公开数据格式沿用现有实现。
- 遵循用户后续明确的测试边界：已删除本次新增的历史数据迁移测试和 `credentialOwnershipLegacyDatabase` 夹具；本轮没有重新引入。数据迁移采用 SQL 审查和发布前专项检查。

## Plan alignment / 计划对齐

- 依据[最新计划](../plan/20261021-environment-credential-ownership.md)，核对基线 `ef610ac483bb7f15d64a9c418fda4bfde9d571df` 到当前工作树的完整 diff，包含已暂存变更。
- 已落实 `environment_id NOT NULL UNIQUE`、物理外键 `ON DELETE RESTRICT`、只读派生凭据身份、既有 UoW 内先主后从，以及指纹 UPDATE 中的 `EXISTS` 身份校验。
- 交接恢复采用删除旧子行并创建新 ID，包内密钥与 revision 保留；相同 revision 也不能复用旧会话身份。
- 本轮仅补验证记录和计划状态，未改变产品代码。复用同一会话中删除迁移测试后已通过的固定检查；重新生成 SQLC 并比较文件哈希，确认无生成漂移。

## Task deliveries and integration / 子任务交付与集成

不适用。该任务为主会话顺序实施，没有子任务 DAG 或子 agent 交付。

已核对 HTTP 写事务装配、MCP `addWriteTool`、交接应用调用与仓储 context 事务传播。数据库写入仍参与已有调用方 UoW，未新增事务框架或把 SSH 远端操作纳入数据库事务。

## Actual diff summary / 实际差异摘要

- 存储：新增三库 000054 up/down，更新两个 schema 快照、环境/凭据查询及 SQLC 输出。
- 应用：按环境查找凭据，初始化先保存环境，local 切换删除子行；交接替换建立新凭据身份；各运行时入口继续比较当前 ID/revision。
- 测试：同步当前 schema 夹具，补充当前关系约束和真实调用方事务回滚验证；删除无生产调用方的凭据更新方法及相关 fake。
- 工具：Python 工作区库存关联子表，Go helper 核对 EnvironmentId、凭据 ID 和 revision。
- 文档：同步产品、架构、决策和相关指南，新增迁移指南与本验证报告。

## Expected vs actual changed files / 预期与实际修改文件对比

本轮开始时，相对基线共 65 个变更文件，均与本任务相关；本轮新增同名验证报告并更新计划记录。

| 范围 | 实际文件 / 位置 | 与计划比较 |
| --- | --- | --- |
| 迁移与查询 | `sql/migration/{sqlite,mysql,postgres}/000054_environment_credential_ownership.{up,down}.sql`、`sql/query/environment/`、`sql/query/environment_credential/` | 一致；既有编号迁移未修改 |
| Schema / 生成结果 | `sql/schema/mysql.sql`、`sql/schema/sqlite.sql`、`internal/gen/sqlc/` | SQLite 快照为检索发现后的必要同步；共享 model 变化均来自 SQLC |
| 模型与仓储 | 两个环境 model、凭据 Repository 接口、两个 SQLC Repository | 一致；原地更新接口因失去生产调用方一并移除 |
| 环境应用 | `internal/application/environment/usecase/` 的初始化、配置、保存、密钥、Probe、目标、终端及测试 | 一致；Command/Forward/CI/CD 通过既有目标解析和派生身份消费，无须无关改写 |
| 相关测试 | 环境 handler、两个 Repository 测试、Windows SSH 外部用例夹具 | 与新归属同步；本次数据迁移测试已按用户要求删除 |
| 运维工具 | `scripts/migrate_workspace.py`、对应 Python 测试、`scripts/workspace_migration_runtime.go` | 一致 |
| 文档 | 产品模型、后端/CD 运行时、决策账本、部署/CLI/工作区指南、迁移指南、索引、同名过程记录 | 一致 |

前端、Proto、仓库凭据和 Deployment/PipelineRun 的快照存储字段没有本任务手工修改。

## Acceptance criteria checklist / 验收标准检查清单

| 验收 | 证据 | 结果 |
| --- | --- | --- |
| A01 环境归属与唯一性 | [凭据查询](../../sql/query/environment_credential/environment_credential.sql)仅按环境定位；[Repository 测试](../../internal/repository/impl/sqlc/environment_credential/repository_test.go)验证无父环境拒绝、同环境唯一、按环境隔离读取、限制父表删除和环境写入不能覆盖子表身份 | 通过 |
| A02 初始化顺序与回滚 | [初始化用例](../../internal/application/environment/usecase/ssh_initialization.go)先 Create/Update Environment 再 ensure credential；[生命周期集成测试](../../internal/application/environment/usecase/credential_lifecycle_integration_test.go)验证重复复用，以及新建、已有未初始化环境和交接替换写失败的真实事务回滚 | 通过 |
| A03 类型切换与显式生成 | [保存用例](../../internal/application/environment/usecase/service.go)删除 local 的原凭据；生命周期测试验证切回 SSH 不生成凭据，显式初始化生成新 ID；[初始化单测](../../internal/application/environment/usecase/initialization_save_test.go)覆盖保存不生成、重复复用及目标变化 | 通过 |
| A04 运行时与旧身份失效 | Probe、TargetResolver、TerminalService 按环境读取；Deployment/PipelineRun、Command/Forward 继续核对派生身份；生命周期测试验证相同 revision 替换后旧会话拒绝；[指纹 Repository 测试](../../internal/repository/impl/sqlc/environment/repository_test.go)验证目标/凭据身份条件 | 通过 |
| A05 交接与密钥边界 | [配置恢复](../../internal/application/environment/usecase/configuration.go)先环境后子行并重新加密；[配置集成测试](../../internal/application/environment/usecase/configuration_integration_test.go)覆盖密文持久化与恢复；交接服务保留 `encryptPackagePrivateKey` 并清空包内明文，公开 API/Proto 无新增密钥字段 | 通过 |
| A06 迁移与 SQLC 一致 | 审查三库 000054：按旧有效绑定复制原 ID/密文/revision/时间，异常在替换业务表前失败；未绑定旧行不迁入，down 仅恢复当前关系；SQLite/PostgreSQL 事务与 MySQL DDL 边界、PostgreSQL 主键命名已核对；SQLC 再生成无漂移 | SQL 审查通过；实际数据升级/回退留作专项验证 |
| A07 冗余关系与文档 | 全部当前凭据读取改为 environment ID；检索未发现旧 Latest/Update 接口及已删除迁移夹具；旧名称仅在迁移、历史快照、读取投影等预期位置保留；活文档描述一致 | 通过 |
| A08 工程检查 | 下表列出本会话实际命令结果与本轮补充检查；没有以测试缓存代替实际测试执行 | 通过 |

## Test / command results / 测试 / 命令结果

工程使用 Go、SQLC、三库编号迁移、Python 运维脚本，以及 web 的 Vue/TypeScript/Yarn。检查入口来自 `Taskfile.yml` 与项目 AGENTS.md。

| 命令 / 检查 | 结果与证据时点 |
| --- | --- |
| `task check` | 删除本次迁移测试后重新执行通过，包含前端 typecheck/lint/format、Go 格式检查与 lint；0 issues |
| `go test -count=1 ./cmd/... ./internal/... ./sql` | 删除本次迁移测试后重新执行通过；覆盖现存本地持久化、事务、协议和进程测试；外部场景沿既有条件跳过 |
| `task test:go:unit -- -count=1` | 实施阶段通过；删除迁移测试后的完整 Go 测试再次覆盖当前业务用例 |
| `python -m unittest discover -s scripts -p test_migrate_workspace.py` | 实施阶段通过，28 项中 23 通过、5 跳过；脚本随后无行为修改 |
| `go build -o <系统临时目录>/orbit-credential-ownership-runtime.exe scripts/workspace_migration_runtime.go` | 实施阶段通过，helper 随后未修改 |
| `task sqlc` + 生成文件哈希比较 | 本轮通过，重新生成没有文件漂移 |
| `git diff --check`、`git diff --cached --check` | 本轮通过 |
| 新增迁移测试 / 历史升级夹具 | 按用户要求删除，本轮未运行或重新引入 |
| MySQL/PostgreSQL 隔离库、Python PostgreSQL/Cygwin 外部场景 | 既有启用变量未配置，未执行；未连接业务库，也未启动外部服务 |

上述通过项均来自实际工具返回。当前产品代码与最近通过检查的实现一致，本轮仅文档变更，无须重复完整测试。

## Missed or expanded scope / 遗漏或扩大的范围

- 无未解释的产品范围扩大。同步 SQLite schema 快照和移除无调用方的凭据更新接口，已记录于计划。
- 用户修正了测试边界，历史升级/回退测试不再属于本次交付；当前关系与事务测试仍使用当前 migration 建库，验证的是现行业务约束。
- 没有新增兼容层、轮换产品能力、额外 API 或多环境模型。

## Risks / 风险

- 当前没有实际业务库的数据预检查证据。异常绑定或未绑定密钥数量须在升级前按[迁移指南](../archive/guides/environment-credential-migration.md)确认。
- 未绑定历史密钥不会迁入新表，down 无法恢复这些行；完整恢复依赖升级前数据库备份和对应加密配置。
- MySQL DDL 有隐式提交，失败不能假设整体自动回滚；三库真实升级/回退及外部 SSH 场景未获得本轮运行证据。

## Incomplete items / 未完成事项

当前代码交付与本地验证无待修复项。发布前仍需完成实际库预检查、备份，以及目标数据库上的专项升级/回退验证；这些不是单元测试覆盖事项，本轮没有执行业务数据迁移。

## Conclusion / 结论

按最新计划及用户限定的测试范围，整体验证通过，环境与凭据主从关系修正可作为当前实现交付。A06 的结论限于 SQL 审查与生成一致性；实际数据迁移的运行结论须由发布前专项验证给出。未执行暂存、提交、发布或业务库写入。
