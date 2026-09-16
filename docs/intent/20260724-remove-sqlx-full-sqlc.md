# 完全移除 sqlx，转为 sqlc 生成 SQL 与 Repository
最后修改时间: 2026-07-24 11:27:01

Review status: Accepted

## Background

后端数据访问当前为 **sqlx + 部分 sqlc** 双轨：

| 区域 | 现状 |
|------|------|
| 连接与迁移 | `*sqlx.DB`（`infrastructure/database`、`bootstrap`） |
| user / role / project / task | 包在 `impl/sqlc`，仍持有 `*sqlx.DB`，列表/事务/部分写路径手写 SQL |
| 粗粒度桶 | `impl/sqlx/{ci,cd}` 全量手写（合计约 1500+ 行），以流程名而非资源领域分包 |
| 查询源 | `sql/query/*.sql` 约 40 条命名查询，远未覆盖全部方法 |
| 方言 | 约 60 处 `db.NowExpr` / `DurationMillisExpr` 运行时拼进 SQL |

既有需求 [`20260705-backend-go-sqlc`](./20260705-backend-go-sqlc.md) 仅引入工具链并部分静态迁移。本任务**收紧边界**：彻底移除 `github.com/jmoiron/sqlx`，生产 SQL 与 repository 一律 sqlc 生成路径，**不做兼容**。

领域边界以共识文档为准：

- [`docs/analyze/20260724-domain-split-consensus-共识.md`](../analyze/20260724-domain-split-consensus-共识.md)

**禁止**继续以 `ci` / `cd` 作为目标业务包或 sqlc 领域名；`ci`/`cd` 仅可保留为历史 HTTP 路由或 migration 分组标签。

## Goal

1. **完全移除** `github.com/jmoiron/sqlx`（`go.mod` / `go.sum`、全部 import、`impl/sqlx` 目录）。
2. 生产连接类型统一为 **`*sql.DB`**。
3. 生产 SQL **仅** 来自 `sql/query/<domain>/**/*.sql`，经 sqlc **按领域**生成到 `internal/gen/sqlc/<domain>`，由 `impl/sqlc/<domain>` 适配；路径至少体现 `<domain>`（对齐 proto 领域分层精神）。
4. Repository 实现落在 `internal/repository/impl/sqlc/<domain>/`；对齐共识 §7；**禁止** `impl/sqlc/ci` 或 `impl/sqlc/cd`。
5. **不做兼容**：无 sqlx 包装、无双轨、无 `CIStore`/`CDStore` 别名；粗接口 **一次切除**（特性分支一次合主干）。
6. 跨域读用**窄接口**；禁止总线 Store。
7. **事务默认**：HTTP 请求级切面 UoW；少量需独立保存失败状态/记录的路径用细粒度事务（worker/claim 等同理）。
8. 保持 **SQLite 与 MySQL** 可运行；单套 `?`；时间戳应用层传入。
9. 对外 HTTP / 业务语义不变；usecase 改依赖细 Store，API 契约不变。
10. 最终验收：全库零 sqlx；query/gen/impl 无 `ci`/`cd` 业务域名。

## Non-goal

1. 不改 HTTP 路由路径、proto 消息字段语义、前端（Proto 目录重组若未单独立项则不在本任务强做）。
2. **不在本任务内**做表 rename migration（如 `build_stage`→`pipeline_stage`、`stage_run`→`pipeline_stage_run`）；SQL 标识符对齐**当前 schema 物理表名**，Go/领域命名用共识名。rename 另项按共识推进。
3. 不删除/迁移 proto 文件（`config_file` 等清理属共识其它工作流）。
4. 不引入 ORM / scany 等 scan 库。
5. 测试 fixture 的 ad-hoc SQL 不强制进 sqlc；测试不得依赖 sqlx。
6. `settings` 无业务表：不建 sqlc 领域。
7. **不实现** `login_attempt` 业务读写（表可保留；本任务不建其 sqlc/Store 路径）。
8. 不处理 migration squash（见 `20260724-sql-migration-squash`）。
9. 不执行 Git 写操作（除非用户另行授权）；不主动启停开发服务器。

## Definition: 「完全 sqlc」

| 允许 | 禁止 |
|------|------|
| SQL 在 `sql/query/**/*.sql` | Go 生产路径手写 SQL 字符串 |
| `sqlc.Queries` / `WithTx` | `*sqlx.*`、`sqlx.In`、`GetContext`、`SelectContext` |
| 事务内多次生成方法 | `impl/sqlx`、双轨实现 |
| 参数化可选过滤 | 运行时拼接 WHERE |
| `sqlc.slice` 表达 IN | 依赖 sqlx 展开 IN |
| Go 传入时间戳 / duration | `NowExpr` / `DurationMillisExpr` 插入 SQL 文本 |
| `dbmodel` → `model.*` | gen 类型泄漏 usecase |
| 共识领域包名 | 目标业务包名为 `ci` / `cd` |

## Domain map（共识对齐）

### 平台核心

| 领域 | 写入职责 | query / 实现 |
|------|----------|----------------|
| `auth` | 已实现的 login_history（**不含** login_attempt，见 Decisions） | `sql/query/auth/*` / `gen/sqlc/auth` |
| `user` | user、user_role | `sql/query/user/*` / `gen/sqlc/user` |
| `role` | role、permission、role_permission | `sql/query/role/*` / `gen/sqlc/role` |
| `project` | project、project_member | `sql/query/project/*` / `gen/sqlc/project` |
| `task` | background_task | `sql/query/task/*` / `gen/sqlc/task` |
| `settings` | 无表 | **无 sqlc** |

### 流水线与交付

| 领域 | 写入职责 | query / 实现 |
|------|----------|----------------|
| `credential` | credential | `sql/query/credential/*` / `gen/sqlc/credential` |
| `repository` | repository、repository_webhook | `sql/query/repository/*` / `gen/sqlc/repository` |
| `pipeline` | template、stage 定义、template_stage、snapshot | `sql/query/pipeline/*` / `gen/sqlc/pipeline` |
| `pipeline_run` | pipeline_run、stage 执行实例、artifact | `sql/query/pipeline_run/*` / `gen/sqlc/pipeline_run` |
| `application` | application、version、component、expose | `sql/query/application/*` / `gen/sqlc/application` |
| `environment` | environment | `sql/query/environment/*` / `gen/sqlc/environment` |
| `service` | service 运行时绑定 | `sql/query/service/*` / `gen/sqlc/service` |
| `deployment` | deployment 操作历史 | `sql/query/deployment/*` / `gen/sqlc/deployment` |
| `gateway` | gateway_config（及载体 application 查询） | `sql/query/gateway/*` / `gen/sqlc/gateway` |
| `route` | route | `sql/query/route/*` / `gen/sqlc/route` |

物理表名示例（当前库，query 内使用）：`build_stage`、`stage_run` 等；领域与 Go 命名使用 `pipeline_stage` / `pipeline_stage_run` 语义，不在本任务改表。

## User scenarios

1. **开发者改查询**：只改对应领域 `sql/query` → `task sqlc` → 编译通过。
2. **业务读写**：各领域行为与迁移前一致（分页、搜索、状态机、事务）；`ErrNotFound` 语义保留。
3. **Worker claim**：task 域事务 claim 不退化。
4. **装配**：bootstrap 按领域构造多个 Store，注入 usecase；无 sqlx 类型。
5. **检查**：全仓无 sqlx 依赖与 import。

## Acceptance

### 全局

1. `go.mod` / `go.sum` 无 sqlx。
2. 全仓无 `jmoiron/sqlx`；无 `impl/sqlx`。
3. 生产 repository 无手写生产 SQL；均走 sqlc 生成 API。
4. 连接与 bootstrap 使用 `*sql.DB`。
5. sqlc 实现目录与 query 按共识领域划分，**无** `ci`/`cd` 业务包。
6. 跨域读为窄接口，无总线 Store 回潮。
7. `go fmt` / `go vet` / `go test` 于 `./cmd/...` `./internal/...` 通过。
8. MySQL e2e 有条件则通过，否则记录跳过原因。

### 分域（可分 PR）

每个领域 PR：

1. 该域零 sqlx、零该域生产手写 SQL。
2. 相关测试通过；关键路径缺测在 plan 补。
3. 对应 `sql/query` + generate 产物入库。

## Work breakdown（按共识领域）

### T0 — 约定与工具

1. 冻结新增 sqlx / 禁止 `impl/sqlx` 与 `impl/sqlc/{ci,cd}`。
2. 约定：时间戳与 duration 应用层化；可选过滤参数化；`sqlc.slice`；`TranslateError`。
3. `sqlc.yaml` 改为**按领域多段**；`sql/query/<domain>/` → `internal/gen/sqlc/<domain>/`；`task sqlc` 一次生成。
4. 目标构造：`NewRepository(db *sql.DB)`（无 `driver string`）；运行期 Queries 绑定 ctx 中的 `DBTX`。
5. 目录与命名对齐共识 §7 + Spec D3。

### T1 — 连接层、UoW 切面与测试基建

1. `Open` / migration / bootstrap / worker → `*sql.DB`。
2. **HTTP 请求级事务切面**（默认 UoW）+ ctx `DBTX` 解析；worker 消息级 UoW。
3. 标明细粒度独立事务路径（claim、必要的失败/终态落库等）。
4. 测试 `sql.Open`；覆盖 Commit/Rollback 与独立事务例外。

### T2 — `task`

1. Claim / Complete / Fail 等全部进 `background_task.sql`。
2. ClaimNext = 事务 + 生成方法。
3. 测试更新；域零 sqlx。

### T3 — `role`

1. 列表/CRUD/权限绑定事务/IN 查询全 sqlc。
2. 域零 sqlx。

### T4 — `user` + `auth`

1. **user**：账户 CRUD、list、user_role 分配、roles/permissions 读。
2. **auth**：仅 **login_history** 写入与列表（从 user 挪出）；**不含 login_attempt**（未实现功能，移出本任务）。
3. `UserRolesByUserIds`：`sqlc.slice`。
4. 两域均零 sqlx。

### T5 — `project`

1. project/member CRUD、list by member、create 事务（含默认 environment 写入时：environment **写入**属 environment 域方法，由应用层或 project 用例编排调用，project query 不越权写异域表——若现状单事务写 environment，Spec/Plan 定「project 用例编排 multi-repo 事务」或临时同 tx 多 Queries，**写入职责归属 environment**）。
2. 提供 **project-read** 窄接口供下游复用。
3. 域零 sqlx。

### T6 — `credential`

1. CRUD、list/search、exists、name、referenced。
2. 删除原 CI 桶中对应手写实现。

### T7 — `repository`

1. repository CRUD/list/by code/has running pipelines。
2. webhook 全路径（可独立 `webhook.sql`，包仍 `sqlc/repository`）。

### T8 — `pipeline`

1. pipeline_stage（物理表 `build_stage`）、template、template_stage、snapshot。
2. update-with-stages / duplicate 等事务。
3. 领域命名 pipeline_*；SQL 表名跟当前 schema。

### T9 — `pipeline_run`

1. pipeline_run 状态机、list 多条件、stage 执行实例（物理表 `stage_run`）、artifact。
2. 执行写路径全 sqlc。

### T10 — `application`

1. application CRUD/list；version 子聚合（component/expose replace/delete 事务）。
2. 包 `sqlc/application`；query 可拆 `application.sql` + `version.sql`。

### T11 — `environment`

1. environment CRUD/list/by code/count services。

### T12 — `service`

1. upsert、status、after-deploy、list by app、list by project（join query 属 service 读模型，不改变 version 归属）。

### T13 — `deployment`

1. create、running/complete/cancel、list；duration 应用层计算。

### T14 — `gateway`

1. gateway_config 读写、resolve active、list gateway apps、create app+config 编排（application 写入走 application 域方法）、has active gateway service。

### T15 — `route`

1. route CRUD/list/enabled/by domain。

### T16 — 装配与 usecase 接线（一次切）

1. 删除 `CIStore`/`CDStore`（及 Execution 粗接口）；替换为细粒度领域 Store，**无别名**。
2. bootstrap / worker / usecase 一次换齐注入。
3. 跨域读窄端口；默认走请求/消息 UoW，不在 usecase 普遍自开 tx。

### T17 — 清零

1. 全仓无 sqlx；`go mod tidy`。
2. 删除 SQL 方言插入 helper（无引用后）。
3. 扁平 `sql/query/*.sql` 旧路径清除；文档改为 sqlc 领域路径 + database/sql。

### T18 — 验证预期

1. 全量测试；关键路径：登录与 login_history、task claim、pipeline 写回、version 多表、service/deploy、gateway、route、请求 Rollback、分页边界。

## Open questions

1. **与 migration squash / 表 rename 并行**：本任务锁定当前物理表名，rename 合并后再改 query 标识符（默认：是）。其余原 open questions 已在 Spec 用户答复中关闭。

## Decisions

1. 流程：**标准模式 / standard**；额外产出 Spec（用户点名）。
2. **领域边界以共识为准**；禁止 `ci`/`cd` 目标业务包。
3. **sqlc 路径体现领域**：`sql/query/<domain>/` + `internal/gen/sqlc/<domain>/` + `impl/sqlc/<domain>/`（不必强行 `orbit/v1` 前缀）。
4. **事务**：默认 HTTP **请求级切面 UoW**；少量失败落库/claim 等**细粒度独立事务**；worker 用消息级 UoW。
5. **login_attempt**：表/model 有、业务未实现 → **移出本任务**；auth 只迁 login_history。
6. **一次切**：特性分支完成后再合主干；不做主干分域渐进替换粗接口。
7. **不做兼容**；完全 sqlc；时间戳应用层化；参数化过滤。
8. **表 rename 不在本任务**；settings 无 sqlc。
9. 覆盖 `20260705-backend-go-sqlc` 的「动态手写可保留」。
10. 任务 **T0–T18** 为分支内工作序，不是多阶段上线切片。

## Risk

1. usecase 依赖面从 2 个大 Store 扩为多 Store，接线 diff 大。
2. 参数化列表与旧 where builder 语义差。
3. 多域同事务编排易漏 Rollback/Commit。
4. 物理表名与领域名不一致带来阅读成本（rename 前）。
5. 与 squash/rename 并行导致 query 与 schema 短暂漂移。
6. 回归面：流水线执行、部署状态机、gateway 创建事务。

## Assumptions

1. 活跃开发期，允许一次性切换存储实现与 Store 粒度。
2. 共识文档在本任务期间保持有效；若共识变更，先改过程文档再改代码。
3. sqlc 版本支持 slice 与 `database/sql` 生成。

## User review notes

- 用户：按共识修正需求；标准模式；产出 Spec。
- 用户答复并已同步 Spec：sqlc 体现领域目录、请求级 UoW、login_attempt 未实现移出、粗接口一次切。
- Requirement / Spec 已按上述答复更新。
