# Spec：完全移除 sqlx，共识领域下全量 sqlc Repository
最后修改时间: 2026-07-24 11:45:41

Review status: Accepted

## Requirement basis

- 需求：[`docs/requirement/20260724-remove-sqlx-full-sqlc.md`](../requirement/20260724-remove-sqlx-full-sqlc.md)（Accepted）
- 领域共识：[`docs/analyze/20260724-domain-split-consensus-共识.md`](../analyze/20260724-domain-split-consensus-共识.md)
- 收紧并覆盖：[`docs/requirement/20260705-backend-go-sqlc.md`](../requirement/20260705-backend-go-sqlc.md)

## Overview

```text
*sql.DB
  → internal/gen/sqlc/<domain>     # 按领域生成
  → impl/sqlc/<domain>             # 按领域适配 model
  → 领域 Store 注入 usecase
  → 默认：请求级事务切面；例外：细粒度事务
```

1. 零 sqlx。
2. 生产 SQL 100% sqlc。
3. query / gen / impl **路径均体现 `<domain>`**（对齐 proto 的领域分层精神，不必强行 product/v1）。
4. 粗粒度 `CIStore`/`CDStore` **一次切除**，无别名并存。
5. 表 rename 不在范围；`login_attempt` **不在范围**（见 D11）。

## Design decisions

### D1 — 连接

| 项 | 决策 |
|----|------|
| 连接类型 | `*sql.DB` only |
| 时间戳 | Go 传入 UTC `time.Time`；禁止 `NowExpr` 拼进 SQL |
| duration_ms | Go 侧计算后写入 |
| 双方言 | 单套 `?` query |

### D2 — 「完全 sqlc」

| 模式 | 做法 |
|------|------|
| CRUD / 固定 list | 命名 query |
| 可选过滤 | 参数化；like 串 Go 预组 |
| IN | `sqlc.slice` |
| 批量子行 | 循环 `:exec` |
| 禁止 | Go 拼 WHERE；sqlx API |

### D3 — sqlc 目录结构（体现领域，对齐 proto 精神）

共识目标 proto：`proto/orbit/v1/<domain>/<entity>.proto` → `internal/gen/proto/orbit/v1/<domain>/`。

sqlc **至少体现 `<domain>`**（不做 API 版本层；不强制 `orbit/v1` 前缀，避免与迁移/工具链无谓耦合）：

```text
sql/query/<domain>/<entity>.sql
internal/gen/sqlc/<domain>/          # package = 领域名，如 user、pipeline
internal/repository/impl/sqlc/<domain>/
```

示例：

```text
sql/query/
  auth/login_history.sql
  user/user.sql
  role/role.sql
  project/project.sql
  task/background_task.sql
  credential/credential.sql
  repository/repository.sql
  repository/webhook.sql
  pipeline/template.sql
  pipeline/stage.sql              # SQL 表: build_stage
  pipeline/snapshot.sql
  pipeline_run/pipeline_run.sql   # 可含 stage 执行；表: stage_run
  pipeline_run/artifact.sql
  application/application.sql
  application/version.sql
  environment/environment.sql
  service/service.sql
  deployment/deployment.sql
  gateway/gateway.sql
  route/route.sql

internal/gen/sqlc/
  auth/
  user/
  role/
  ...
```

`sqlc.yaml`：**按领域多段 `sql:` 配置**（共享同一 `schema: sql/migration/sqlite`），每段：

- `queries: sql/query/<domain>`
- `gen.go.package: <domain>`（或 `sqlc<domain>` 若关键字冲突时再定）
- `gen.go.out: internal/gen/sqlc/<domain>`
- `sql_package: database/sql`

`task sqlc` 一次生成全部领域。

横切转换：`internal/repository/impl/sqlc/dbmodel`（或 `internal/repository/impl/sqlc/internal/dbmodel`），不生成 SQL。

删除：`internal/repository/impl/sqlx/`；扁平的 `sql/query/*.sql` 迁入领域目录后删除旧路径。

`settings`：无 sqlc。

### D4 — Repository 接口

按领域 Store；删除最终态：

- `CIStore` / `PipelineExecutionStore`
- `CDStore` / `DeploymentExecutionStore`

**合入策略（用户确认）：一次切** — 特性分支内完成全部替换后**一次合入主干**；不在主干上长期「半粗半细」接口。开发仍可按领域提交到特性分支，但对外发布/合主干不做分域渐进兼容。

跨域**读**用窄端口；**写**只在归属域 Store。

### D5 — 跨域事务（主流实践 + 本项目选择）

#### 主流常见三档

| 档位 | 做法 | 典型场景 |
|------|------|----------|
| A. 请求/消息级 UoW | 入口切面 `Begin`，ctx 携带 `tx`/`DBTX`，成功 Commit、失败 Rollback | 一次 HTTP/RPC 内多仓储写入，要全成全败 |
| B. 应用服务显式事务 | usecase 内 `BeginTx`，多 Store `WithTx` | 无 HTTP 的 worker、或只要局部原子性 |
| C. 仓储内短事务 | 单聚合多表（version+components）在 repo 内 tx | 不泄露给调用方的聚合一致性 |

反模式：每个 repo 方法各自隐式开全局长事务又互相嵌套；或「总线 Store」吞掉所有跨域写。

#### 本项目决策（用户倾向）

1. **默认（绝大多数）**：在 **HTTP 请求边界** 以切面处理事务（请求级 UoW）。
   - 中间件/过滤器：请求开始绑定可提交的 `Tx`（或 lazy begin），handler/usecase 经 ctx 取 `DBTX`。
   - 请求成功（无 error、非 5xx 策略由 Plan 钉死）→ Commit；失败 → Rollback。
   - 各领域 sqlc `Queries` 从 ctx 取 `DBTX`（`*sql.Tx` 或 `*sql.DB`），**禁止**在默认路径里再嵌套 `BeginTx`。
2. **例外（少量、细粒度）**：需要**在主事务失败后仍落库**，或**独立保存失败状态/审计/执行记录**时，显式使用独立连接/独立短事务，例如：
   - 登录失败审计、限流计数类写入（若存在）
   - pipeline/deployment **终态**在编排失败后仍要更新 status/error（若与主流程隔离）
   - worker claim：lease 更新与业务执行生命周期不同，保持显式短事务
3. **Worker / 非 HTTP**：无请求切面；对齐「消息处理边界」——默认一个 task handler 一次 UoW，同样允许「失败状态独立提交」的细粒度例外。
4. **单聚合多表**（version 替换 components）：优先走当前 UoW 的同一 `tx`；若调用栈无 UoW，才允许 repo 内短事务（实现时优先保证 HTTP/worker 已提供 UoW，减少 repo 自开 tx）。

Plan 需落地：

- ctx key 与 `DBTX` 解析 helper
- 哪些路径标为 `@NoRequestTx` / 独立事务（清单）
- 与 Gin 错误处理、panic recover 的 Commit/Rollback 顺序

### D6 — 物理名 vs 领域名

| 层 | 规则 |
|----|------|
| SQL 文本 | 当前物理表名（`build_stage`、`stage_run`…） |
| 路径/包/Go 领域名 | 共识名（`pipeline`、`pipeline_run`…） |
| 表 rename | 另项 |

### D7 — 错误与分页

- `sql.ErrNoRows` → `repository.ErrNotFound`
- `Page[T]` + `NormalizePage` 保留

### D8 — Usecase / bootstrap

- bootstrap 装配各领域 gen Queries + Store
- usecase 依赖细 Store；HTTP 路由可仍用历史 `/ci`、`/cd` 前缀
- 请求级 UoW 中间件挂在 HTTP server 链上

### D9 — 测试

- memory SQLite + MigrateUp；`sql.Open`
- 集成测需覆盖：请求级 Commit/Rollback；细粒度独立事务路径
- 提交各 `internal/gen/sqlc/<domain>` 生成物

### D10 — 交付与合入

- **主干一次切**：特性分支完成 T0–T17 后一次合并；不做主干上的分域渐进替换粗接口。
- 分支内可按领域提交便于审阅，但合并门禁是「零 sqlx + 无 CIStore/CDStore + 领域路径齐全」。

逻辑完成顺序（分支内工作序，非多阶段上线）：

```text
T0 约定 + sqlc 多包目录
T1 *sql.DB + 请求级 UoW 切面骨架
T2–T15 各领域 sqlc 与 Store
T16 删粗接口、usecase/bootstrap 一次换齐
T17 清零 sqlx
T18 验证
```

### D11 — `login_attempt` 范围

代码核对结论：

| 项 | 状态 |
|----|------|
| migration 表 `login_attempt` | 存在 |
| `model.LoginAttempt` | 存在 |
| repository / usecase 读写 | **无**（仅 `login_history` 有 `SaveLoginHistory` / list） |
| sqlc query | **无**（生成 models 可能来自 schema 扫描，无业务 API） |

按用户规则：**未实现功能 → 移出本任务**。

- 不写 `login_attempt` 的 sqlc query/Store 方法。
- 不删表、不改 migration。
- auth 域本任务只承接**已实现**的 `login_history`（及从 user 挪出的相关方法）。

### D12 — IN / 其它已定

- `UserRolesByUserIds` 等：优先 `sqlc.slice`。
- 不保留 Execution 专用第三套接口。

## Affected components

| 区域 | 影响 |
|------|------|
| `sqlc.yaml` | 多段按领域 queries/out |
| `sql/query/**` | 迁入 `<domain>/` |
| `internal/gen/sqlc/**` | 按领域子包 |
| `internal/repository/**` | 接口一次拆分 |
| `internal/api/http` 或 middleware | 请求级 UoW |
| `internal/bootstrap` | 装配 + 中间件顺序 |
| `internal/application/**` | 依赖细 Store；默认不自开 tx |
| `internal/queue/worker` | 消息级 UoW / claim 细粒度 |
| `go.mod` | 去 sqlx |

## Interfaces

### 连接与 UoW

```text
Open(cfg) (*sql.DB, error)
MigrateUp(db *sql.DB, driver string) error

// 概念 API（Plan 定包路径与命名）
WithTx(ctx) DBTX          # 从 ctx 取当前事务或 db
BeginRequestTx / middleware
RunInTx(ctx, fn)          # 细粒度例外用
```

### 领域 Store 搬迁（摘要）

| 原粗接口方法簇 | 目标 |
|----------------|------|
| Credential* | CredentialStore |
| Repository*、Webhook* | RepositoryStore |
| BuildStage*、Template*、Snapshot* | PipelineStore |
| PipelineRun*、StageRun*、Artifact* | PipelineRunStore |
| Application*、Version*、Component*、Expose* | ApplicationStore |
| Environment* | EnvironmentStore |
| Service* | ServiceStore |
| Deployment* | DeploymentStore |
| Gateway* | GatewayStore |
| Route* | RouteStore |
| LoginHistory* | AuthStore（不含 login_attempt） |
| User*、UserRole* | UserStore |
| Project*、Member* | ProjectStore |
| Task* | TaskStore |

## Technical questions

（用户已答复，归档）

| # | 结论 |
|---|------|
| sqlc 是否按领域分层 | **是**；`sql/query/<domain>/` + `internal/gen/sqlc/<domain>/` |
| 跨域事务 | **默认 HTTP 请求切面 UoW**；少量失败落库/claim 等**细粒度独立事务** |
| login_attempt | **未实现 → 移出任务**；只迁 login_history |
| 粗接口切换 | **一次切**（特性分支一次合主干） |

暂无阻塞实现的新未决项；Plan 阶段细化 UoW 中间件与 `@NoRequestTx` 清单即可。

## Risks

1. 请求级 UoW 与「失败仍要写入」路径若标错，会丢审计或误 rollback 状态写。
2. SQLite `MaxOpenConns=1` 下请求长事务放大锁等待 → 事务必须短、避免在 tx 内做 Docker/IO。
3. 一次切 diff 大 → 特性分支 + 全量测试门禁。
4. sqlc 多包配置维护成本 → 用统一 Task 与目录约定降低。
5. 物理表名与领域目录名不一致 → query 文件头注释标明物理表。

## Alternatives considered

| 方案 | 结论 |
|------|------|
| 单扁平 `sql/query` + 单 gen 包 | **拒绝**；用户要求至少体现领域 |
| 完全照抄 `orbit/v1` 前缀 | **不必要**；领域层足够 |
| 仅 usecase 开事务、无请求切面 | **非默认**；用户倾向请求切面 |
| 分域渐进替换 CIStore 上主干 | **拒绝**；一次切 |
| 本任务实现 login_attempt | **拒绝**；功能未实现 |
| scany / 手写 database/sql | **拒绝** |

## Implementation guidance（Plan 输入）

1. `sqlc.yaml` 多领域段模板与目录迁移步骤。
2. 请求 UoW 中间件设计、ctx 契约、与 worker 对齐方式。
3. 细粒度独立事务路径清单（claim、必要的终态写、等）。
4. 旧 `CIStore`/`CDStore` 方法 → 新 Store → query 对照表。
5. 一次切的分支策略与全量 `go test` 门禁。
6. 验证清单 T18。

## User review notes

- 用户答复已纳入 D3 / D5 / D10 / D11。
- `login_attempt`：表与 model 有、业务路径无 → 移出。
- 用户「开始计划」：本 Spec 标为 Accepted，进入 Plan。
