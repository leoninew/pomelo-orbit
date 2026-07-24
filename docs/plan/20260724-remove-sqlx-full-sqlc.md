# Plan：完全移除 sqlx，共识领域全量 sqlc
最后修改时间: 2026-07-24 11:50:00

Review status: Accepted

## Basis

| 文档 | 状态 |
|------|------|
| [requirement](../requirement/20260724-remove-sqlx-full-sqlc.md) | Accepted |
| [spec](../spec/20260724-remove-sqlx-full-sqlc.md) | Accepted |
| [domain consensus](../analyze/20260724-domain-split-consensus-共识.md) | 已确认基线 |

流程：标准模式 / standard。合入策略：**特性分支一次切主干**（分支内可按步提交，不在主干分域渐进）。

## Goal (plan-level)

1. 分支内完成 `*sql.DB` + 请求/消息 UoW + 按领域 sqlc + 细 Store + 删 sqlx。
2. 无 `CIStore`/`CDStore`/`PipelineExecutionStore`/`DeploymentExecutionStore`。
3. 无 `jmoiron/sqlx`；query/gen/impl 路径含 `<domain>`。
4. `go fmt` / `go vet` / `go test ./cmd/... ./internal/...` 全绿后一次合主干。

## Implementation steps

### Step 0 — 分支与冻结

1. 从当前主干拉特性分支（名称由执行者定，如 `feature/remove-sqlx-full-sqlc`）。
2. 冻结：禁止新增 sqlx API、禁止新建 `impl/sqlx` 或 `impl/sqlc/{ci,cd}`。
3. 与 migration squash / 表 rename 协调：本分支 **SQL 标识符锁当前物理表名**；squash 若改 schema 路径，同步 `sqlc.yaml` 的 `schema` 指向，不改业务表名语义。

### Step 1 — sqlc 多领域工程骨架（T0）

1. 新建目录结构（先可空文件占位）：
   - `sql/query/<domain>/`
   - 目标 `internal/gen/sqlc/<domain>/`
   - `internal/repository/impl/sqlc/<domain>/`
2. 重写 `sqlc.yaml`：每个共识领域一段 `sql:`（共享 `schema: sql/migration/sqlite`）：

| domain | queries | package | out |
|--------|---------|---------|-----|
| auth | `sql/query/auth` | `auth` | `internal/gen/sqlc/auth` |
| user | `sql/query/user` | `user` | `internal/gen/sqlc/user` |
| role | `sql/query/role` | `role` | `internal/gen/sqlc/role` |
| project | `sql/query/project` | `project` | `internal/gen/sqlc/project` |
| task | `sql/query/task` | `task` | `internal/gen/sqlc/task` |
| credential | `sql/query/credential` | `credential` | `internal/gen/sqlc/credential` |
| repository | `sql/query/repository` | `repository` | `internal/gen/sqlc/repository` |
| pipeline | `sql/query/pipeline` | `pipeline` | `internal/gen/sqlc/pipeline` |
| pipeline_run | `sql/query/pipeline_run` | `pipelinerun` 或 `pipeline_run` | `internal/gen/sqlc/pipeline_run` |
| application | `sql/query/application` | `application` | `internal/gen/sqlc/application` |
| environment | `sql/query/environment` | `environment` | `internal/gen/sqlc/environment` |
| service | `sql/query/service` | `service` | `internal/gen/sqlc/service` |
| deployment | `sql/query/deployment` | `deployment` | `internal/gen/sqlc/deployment` |
| gateway | `sql/query/gateway` | `gateway` | `internal/gen/sqlc/gateway` |
| route | `sql/query/route` | `route` | `internal/gen/sqlc/route` |

3. package 名若与 Go 关键字/冲突：用 `pipelinerun` 等合法标识，**目录仍用共识域名**。
4. 将现有扁平 `sql/query/*.sql` **迁入**对应领域目录（内容先原样搬，再补齐）。
5. 删除旧 `internal/gen/sqlc` 单包布局（生成后整体替换）。
6. `task sqlc` 必须一次成功；生成物入库。
7. 扩展 `internal/repository/impl/sqlc/dbmodel`（横切 Null 转换），供各域 impl 引用。

**检查：** `task sqlc` 退出 0。

### Step 2 — 连接层 `*sql.DB`（T1 前半）

1. `internal/infrastructure/database/database.go`：`Open` 返回 `*sql.DB`；保留 SQLite pragma / MySQL 池参数。
2. `migration.go`：入参 `*sql.DB`（现有 `sqlDB.DB` 解包删除）。
3. 删除生产路径对 `NowExpr`/`DurationMillisExpr` 的依赖（可暂留函数至 T17 再删，但新 query 禁止使用）。
4. `bootstrap/database.go`、`app.go`、`http.go`、`worker.go`：类型改为 `*sql.DB`（此步可先仍编译失败，与 Step 3–5 同分支推进）。

**检查：** 包级编译可在后续步骤闭合；本步至少 `database` 包自测可过。

### Step 3 — 请求/消息 UoW（T1 后半）

新建（建议路径，实现时可微调但须单点）：

```text
internal/infrastructure/database/tx/
  context.go    # ctx key, DBTXFrom(ctx), WithDBTX
  request.go    # HTTP 中间件
  run.go        # RunInTx / RunWithoutRequestTx 细粒度
```

#### 3.1 契约

```text
type DBTX interface {
  ExecContext(...)
  PrepareContext(...)
  QueryContext(...)
  QueryRowContext(...)
}

// ctx 中：
// - 必有 *sql.DB（根连接）
// - 可选 *sql.Tx（请求/消息 UoW 内）
DBTXFrom(ctx) DBTX  // 有 tx 用 tx，否则用 db（只读默认可走 db；写路径应在 UoW 内）
```

领域 repo：`queries := gen.New(tx.DBTXFrom(ctx))`，**构造时注入 `*sql.DB`**，执行时优先 ctx DBTX。

#### 3.2 HTTP 中间件行为

1. 每个请求：`BeginTx`（或 lazy：第一次写时 begin——Plan 推荐 **eager begin** 简单可测；若性能有虑再改 lazy）。
2. `defer`：panic 或 handler 返回 error → Rollback；成功 → Commit。
3. 与现有 Gin 错误处理对齐：以 `c.Errors` / 统一 response 是否已写失败状态为准；**Plan 钉死规则**：
   - handler 调用链返回 error 或 abort 为 5xx → Rollback
   - 业务 4xx（校验失败）→ **Rollback**（默认；避免半写入）
   - 仅当路径显式 `tx.CommitPartial`/`RunInTx` 独立提交时例外
4. 挂载位置：`bootstrap`/`api/http` 在 auth 之后、业务路由之前（具体插槽实现时对齐现有 middleware 序）。

#### 3.3 细粒度例外清单（初始）

| 路径 | 原因 | 做法 |
|------|------|------|
| `task.ClaimNext` | lease 生命周期 ≠ 整次 HTTP/业务 | `RunInTx` 独立短事务；**不参与**外层请求 tx 或 worker 长事务 |
| `task.Complete` / `Fail` | worker 消息边界内可走消息 UoW；若需在业务失败后仍标记 fail | 同一消息 UoW 末态写，或独立短事务（实现选一种并单测） |
| 未来登录失败计数等 | 主流程失败仍要审计 | 独立短事务（本任务无 login_attempt，预留模式即可） |

**禁止**在 UoW 内：Docker、长文件 IO、外部 HTTP（已有 runner 应在 tx 外）。

#### 3.4 Worker

1. 每个 task handler 外层：消息级 UoW（Begin/Commit/Rollback）。
2. Claim 仍用独立短事务（见上表）。

**检查：** 中间件单测（成功 commit / error rollback）；`RunInTx` 单测。

### Step 4 — Repository 接口一次定义（T16 前置设计落地）

在 `internal/repository/` **新增/调整**接口文件（建议一文一域或按现分包）：

| 新/调整接口 | 来源方法 |
|-------------|----------|
| `AuthStore` | `SaveLoginHistory`、`ListLoginHistory`（从 UserStore 挪出） |
| `UserStore` | 账户与 user_role；去掉 login_history |
| `RoleStore` | 保持并补全 |
| `ProjectStore` | 保持；暴露 project-read 所需方法 |
| `TaskStore` 或 queue 侧接口 | enqueue/claim/complete/fail/find（与 `queue/task` 对齐） |
| `CredentialStore` | 自 CIStore |
| `RepositoryStore` | repository + webhook |
| `PipelineStore` | stage/template/snapshot |
| `PipelineRunStore` | run/stage_run/artifact + 执行状态写 |
| `ApplicationStore` | application/version/component/expose |
| `EnvironmentStore` | environment |
| `ServiceStore` | service |
| `DeploymentStore` | deployment + 执行状态写 |
| `GatewayStore` | gateway_config 及相关查询 |
| `RouteStore` | route |

**删除文件/类型（在 Step 10 接线时删除，但接口定义阶段先写新接口）：**

- `CIStore`、`PipelineExecutionStore`（`ci.go`）
- `CDStore`、`DeploymentExecutionStore`（`cd.go`）

可选：`ProjectReader` 小接口 = `Project` + `IsProjectMember`，供下游依赖，避免依赖完整 ProjectStore。

### Step 5 — 平台核心域 sqlc + impl（T2–T5）

对每个域重复模式：

1. 在 `sql/query/<domain>/` 写全命名 SQL（参数化 list/count；时间戳参数；IN 用 `sqlc.slice`）。
2. `task sqlc`。
3. 实现 `impl/sqlc/<domain>/repository.go`：`New(db *sql.DB)`，方法内 `gen.New(tx.DBTXFrom(ctx))`，`dbmodel` 转 `model`。
4. 迁移/重写测试：`sql.Open("sqlite", ":memory:")` + `MigrateUp`。

#### 5.1 task（T2）

| Query（建议名） | 用途 |
|-----------------|------|
| EnqueueTask | insert |
| TaskToClaim | select candidate |
| ClaimTask | optimistic update |
| CompleteTask | success |
| FailTask | fail/retry |
| FindTaskByID | get |

ClaimNext：仅用 `RunInTx` + TaskToClaim + ClaimTask。

#### 5.2 role（T3）

List/Create/Update/Delete、权限 list、set permissions、PermissionByCodes（slice）。

#### 5.3 user + auth（T4）

- user：CRUD、list、roles、permissions、SetUserRoles、UserRolesByUserIds（slice）。
- auth：`SaveLoginHistory`、`ListLoginHistory` only。
- **不做** login_attempt。

#### 5.4 project（T5）

- project/member 全路径。
- CreateProject：usecase 在**同一请求 UoW** 内调 `ProjectStore` + `EnvironmentStore.Create`（默认 env）；**不**在 project query 写 environment 表。
- 提供 ProjectReader。

**检查：**  
`go test ./internal/repository/impl/sqlc/task/... ./internal/repository/impl/sqlc/role/... ./internal/repository/impl/sqlc/user/... ./internal/repository/impl/sqlc/auth/... ./internal/repository/impl/sqlc/project/...`

### Step 6 — 流水线域（T6–T9）

#### 6.1 credential

CRUD、list/search、exists、name、ReferencedByRepositories。

#### 6.2 repository

CRUD、list/search、by code、HasRunningPipelines；webhook CRUD/list。

#### 6.3 pipeline

物理表 `build_stage` → query 文件 `pipeline/stage.sql`；template + template_stage + snapshot。  
UpdateWithStages / Duplicate：同 UoW 多语句（请求或消息 tx）。

#### 6.4 pipeline_run

物理表 `stage_run`；run 状态机、list 多条件、artifact。  
执行写：MarkRunning / Complete / Cancel / InsertStage / UpdateStage / InsertArtifact — 全部命名 query；duration Go 算。

**检查：** 原 `impl/sqlx/ci` 测试迁到各域或 `pipeline`/`pipeline_run` 包；`go test` 对应包。

### Step 7 — 交付域（T10–T15）

#### 7.1 application

application CRUD/list；version + component + expose（replace/delete 多语句同一 UoW）。

#### 7.2 environment

CRUD/list/by code/CountServices。

#### 7.3 service

Upsert、status、AfterDeploy、list by app、list by project（join 读模型仍归 service query）。

#### 7.4 deployment

Create、MarkRunning、Complete、Cancel、List；`duration_ms`/`finished_at` Go 传入。

#### 7.5 gateway

gateway_config；ResolveActive；ListGatewayApps；HasActiveGatewayService。  
Create gateway app：usecase 编排 `ApplicationStore.Create` + `GatewayStore.UpsertConfig`（同请求 UoW）。

#### 7.6 route

CRUD/list/all/enabled/by domain。

**检查：** 原 `impl/sqlx/cd` 测试迁移；各域 `go test`。

### Step 8 — 删除 sqlx 实现目录

1. 删除整个 `internal/repository/impl/sqlx/`。
2. 确认无引用。

### Step 9 — Usecase / bootstrap / worker 一次接线（T16）

1. 改构造函数（示例，以实现为准）：

| 组件 | 依赖 |
|------|------|
| auth usecase | AuthStore, UserStore, token… |
| user usecase | UserStore, RoleStore |
| role usecase | RoleStore |
| project usecase | ProjectStore, EnvironmentStore, UserStore(读) |
| ci usecase | Credential, Repository, Pipeline, PipelineRun, ProjectReader, dispatcher… |
| cd usecase | Application, Environment, Service, Deployment, Gateway, Route, ProjectReader, … |
| task service | TaskStore |
| worker handlers | 对应域 Store + 消息 UoW |

2. `bootstrap/http.go`：构造全部领域 repo；注册 UoW 中间件；注入 usecase。
3. `bootstrap/worker.go`：同上。
4. 删除对 `cirepo`/`cdrepo` sqlx 包的 import。
5. 修全部 fake/mock（handler 测试、authz_test 等）。

**检查：** `go build ./...` 或 `go test ./internal/bootstrap/...` 编译通过。

### Step 10 — 清零（T17）

1. `rg jmoiron/sqlx` / `sqlx\.` 全仓应为 0（除文档）。
2. `go mod tidy`；确认 `go.mod` 无 sqlx。
3. 删除无用 `NowExpr`/`DurationMillisExpr`/`QuoteIdent`（若无引用）。
4. 删除扁平 `sql/query` 遗留文件。
5. README/设计文档中 sqlx 表述改为 sqlc 领域路径（若有）。
6. `model` 上 `db` tag：可不删；不得再依赖其做生产扫描。

### Step 11 — 全量验证（T18，实现收尾；正式 Verification 阶段再出报告）

见下文 Verification plan。分支门禁全绿后合并主干。

## Files to change (expected)

### 新增

```text
sql/query/<domain>/**/*.sql
internal/gen/sqlc/<domain>/**          # generated
internal/repository/impl/sqlc/<domain>/**
internal/infrastructure/database/tx/**
internal/repository/auth.go            # 或并入既有文件
internal/repository/credential.go
internal/repository/repository_vcs.go  # 避免与 package 名混淆时的命名
internal/repository/pipeline.go
internal/repository/pipeline_run.go
internal/repository/application.go
internal/repository/environment.go
internal/repository/service.go
internal/repository/deployment.go
internal/repository/gateway.go
internal/repository/route.go
# 具体文件名以实现时清晰为准
```

### 大改

```text
sqlc.yaml
Taskfile.yml                           # 如需
internal/infrastructure/database/*.go
internal/bootstrap/*.go
internal/repository/*.go               # 接口
internal/repository/impl/sqlc/**       # 旧 user/role/project/task 重写
internal/application/**/usecase/**
internal/queue/**/**
internal/api/http/**                   # middleware 挂载、fake stores
go.mod / go.sum
```

### 删除

```text
internal/repository/impl/sqlx/**
internal/repository/ci.go              # 粗接口
internal/repository/cd.go
internal/gen/sqlc/*.go                 # 旧单包（被领域包替换）
sql/query/*.sql                        # 扁平旧路径（迁走后）
```

### 不改

```text
sql/migration/**                       # 不改已执行迁移内容（除非 squash 并行另项）
web/** 业务
proto 目录搬家
login_attempt 业务
```

## Method migration map (summary)

完整方法表在实现时从现接口抄录；摘要：

| 旧 | 新 Store |
|----|----------|
| CIStore.Credential* | CredentialStore |
| CIStore.Repository* / Webhook* | RepositoryStore |
| CIStore.BuildStage* / PipelineTemplate* / Snapshot* | PipelineStore |
| CIStore.PipelineRun* / StageRun* / Artifact* | PipelineRunStore |
| PipelineExecutionStore.* | PipelineRunStore（+ 必要读） |
| CDStore.Application* / Version* / Component* / Expose* | ApplicationStore |
| CDStore.Environment* | EnvironmentStore |
| CDStore.Service* | ServiceStore |
| CDStore.Deployment* | DeploymentStore |
| CDStore.Gateway* | GatewayStore |
| CDStore.Route* | RouteStore |
| DeploymentExecutionStore.* | DeploymentStore + Application/Service 读 |
| UserStore.LoginHistory* | AuthStore |
| CIStore/CDStore.Project* | ProjectStore / ProjectReader |

## Verification plan

### 命令（实现完成时 / Verification 阶段）

```text
task sqlc
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

可选：`go test ./internal/test/e2e/...`（有 MySQL DSN 时）。

### 静态门禁

```text
rg "jmoiron/sqlx|github.com/jmoiron/sqlx" --glob "*.go"
rg "impl/sqlx|CIStore|CDStore|PipelineExecutionStore|DeploymentExecutionStore" --glob "*.go"
rg "GetContext|SelectContext|BeginTxx|sqlx\\." --glob "*.go"
```

均应无匹配（测试与生产）。

### 行为清单

| # | 场景 | 期望 |
|---|------|------|
| 1 | 登录成功 | 写 login_history；请求 tx commit |
| 2 | 业务校验 4xx | 无脏写（rollback） |
| 3 | task claim | 独立短事务；并发下不双 claim（相对现状） |
| 4 | pipeline 执行写回 | running/complete/stage/artifact 正确 |
| 5 | version 替换 components | 同请求原子 |
| 6 | create project | project + 默认 environment 同请求原子 |
| 7 | gateway create | app + config 同请求原子 |
| 8 | deploy complete/cancel | duration_ms 合理；状态正确 |
| 9 | 分页/search 空串 | 与旧行为一致或单测钉死 |
| 10 | worker 消息失败 | 消息 UoW rollback 或 Fail 路径按设计落库 |

### 包级建议测试顺序

1. `database` + `database/tx`
2. 各 `impl/sqlc/<domain>`
3. `application/*/usecase` 集成测
4. `queue/worker`
5. 全量 `./internal/...`

## Blockers

1. 无 — 工具链已有 sqlc v1.30、`task sqlc`。
2. 软阻塞：与 migration squash 并行时需同步 schema 路径；**不**阻塞启动实现。

## Assumptions

1. 在特性分支完成前，主干可继续其它工作；合并时解决冲突。
2. sqlc multi-package 单 `sqlc.yaml` 可用（v1.30）。
3. 请求级 eager `BeginTx` 可接受；SQLite 单连接要求事务短。
4. 4xx 业务错误默认 Rollback。
5. `login_attempt` 保持表结构，无代码路径。
6. 物理表名锁定至 rename 另项。

## Risks

| 风险 | 缓解 |
|------|------|
| 一次切 diff 过大难审 | 分支内按 Step 提交；PR 描述挂本 Plan 步骤 |
| UoW 误 rollback 丢状态写 | 例外清单 + 单测；Fail/Complete 路径人工过一遍 |
| tx 内跑 runner | code review 禁止；执行放 tx 外 |
| 参数化 list 语义差 | 迁移旧 list 测试 |
| 多 gen 包 import 噪音 | 统一 alias；dbmodel 集中转换 |
| 合并冲突 | 缩短分支寿命；先合 squash 或先合本分支二选一协调 |

## Rollback

1. **合主干前**：废弃特性分支即可。
2. **合主干后**：`git revert` 合并提交；恢复旧依赖与 `impl/sqlx` 仅当 revert 完整。
3. 数据：无 migration 变更则无数据回滚；仅代码回滚。
4. 不做双轨运行期开关。

## Implementation order checklist

```text
[ ] Step 0  分支
[ ] Step 1  sqlc 多领域骨架 + 搬迁 query
[ ] Step 2  *sql.DB
[ ] Step 3  UoW 中间件 + worker + 单测
[ ] Step 4  新 Store 接口
[ ] Step 5  task/role/user/auth/project
[ ] Step 6  credential/repository/pipeline/pipeline_run
[ ] Step 7  application/environment/service/deployment/gateway/route
[ ] Step 8  删 impl/sqlx
[ ] Step 9  usecase/bootstrap 一次接线
[ ] Step 10 清零 sqlx
[ ] Step 11 全量测试门禁
```

## User review notes

- 用户指令：开始计划 → Spec Accepted，本 Plan 为 Draft 待确认。
- 确认 Plan 后即可「开始实现」。
- 本阶段不写产品代码。
