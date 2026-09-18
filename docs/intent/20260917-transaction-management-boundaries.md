# 事务管理与边界规范治理
最后修改时间: 2026-09-17 21:15:00

Review status: Accepted

Mode: standard

## Background

系统采用清晰收敛的两层事务管理架构（路线 B：仓储去事务化）：
1. **HTTP 请求切面事务（全局写 UoW）**：在 `internal/infrastructure/database/tx/request.go` 中通过 `tx.Middleware` 作为 Gin 中间件（`MutatingUnitOfWork`），对所有常规 `POST`、`PUT`、`PATCH`、`DELETE` 写请求统一开启事务，使用标准 `context.Context` 传播，并通过 `bufferedResponseWriter` 实现两阶段提交保障（仅在 Handler 正常且 `sqlTx.Commit()` 成功后才 flush 响应）。**常规 HTTP 业务写请求（包括跨多个 Repository 的应用层编排）默认直接复用此切面事务，应用层无须引入额外的 `TransactionRunner`。**
2. **应用层显式短 UoW（`TransactionRunner`）**：其使用场景严格限定在**缺乏入站请求事务切面**的场景：
   - 异步 Worker 后台任务：Worker 仅持有 `*sql.DB`，容器运行与长任务在事务外执行，仅在最终多表持久化阶段（如 `forkBuildVersion`）使用显式短 UoW。
   - 被排除在切面外的特殊请求（`skipPaths`）：如 `/api/route/sync/confirm`（前后有 Traefik 网络 I/O）、`/api/dialogue`（包含 LLM 外部调用），以及 `/api/pipeline-run/:run_id`（多表删除后有物理文件清理）。应用层 UseCase 在需要保证多表原子写入时显式调用 `TransactionRunner.RunInTransaction`。
3. **仓储层定位（彻底去事务化）**：领域仓储（Domain Repositories）彻底消除内部 `tx.RunInTx` 调用，纯粹作为数据访问层，只通过 `r.q(ctx)` 执行 SQL 语句。仓储自身不开启、不提交、也不回滚事务，其原子性完全遵循外部传入的 `context.Context`。

经全库代码与事务边界审查，已精确定位以下具体违规与隐患点，本规范文档定义确定的改造任务项：

1. **外部网络与不可逆本地 I/O 处于 HTTP 事务生命周期内（高风险）**：
   - `POST /api/project-initialization/environment/test`：调用 `TestSSHReachability` 执行真实远程 SSH 网络连接，未加入 `skipPaths`，导致网络探测与超时等待期间无谓占用数据库事务与写锁。
   - `POST /api/auth/login`：Handler 先调用 Cloudflare Turnstile 外网 HTTP 验证接口，再计算 CPU 密集的 `bcrypt.CompareHashAndPassword`，全程处于 HTTP 切面事务中。
   - `DELETE /api/pipeline-run/:run_id`：Handler 在切面事务内调用 `workspace.RemoveRunFiles` 执行物理文件删除。文件删除不可逆，且耗时不确定。
2. **纯计算或无写操作的 POST 接口空跑事务**：
   - `POST /api/service/:service_id/preview` 与 `POST /api/version/:version_id/preview`：仅生成 Docker Compose YAML 文本，纯读无写，未加入 `skipPaths`，每次请求空跑 `BeginTx`、缓冲与 `Commit`。
   - `POST /api/auth/logout`：仅做请求解析并返回 204，无任何数据库操作，未加入 `skipPaths`。
3. **仓储层违规自治事务与隐式嵌套**：
   - 多个领域仓储（`application`, `service`, `gateway`, `pipeline`, `pipeline_run` 等）内部残留 `tx.RunInTx`，造成职责混淆与隐式嵌套隐患。
   - `role.Repository` 与 `user.Repository` 在不同方法中事务使用不一致。

## Goal

1. **补齐 `tx.Middleware` 的跳过路径（`skipPaths`）**：
   - 在 `internal/bootstrap/http.go` 中，将以下 6 个路由加入 `skipPaths`：
     - `"/api/project-initialization/environment/test"`（SSH 网络握手探测）
     - `"/api/auth/login"`（Turnstile 外网 HTTP 校验与 bcrypt 计算）
     - `"/api/auth/logout"`（无写操作的 204 请求）
     - `"/api/service/:service_id/preview"`（纯计算 Compose 预览）
     - `"/api/version/:version_id/preview"`（纯计算 Compose 预览）
     - `"/api/pipeline-run/:run_id"`（包含磁盘物理文件清理）
2. **重构流水线删除执行顺序与事务边界**：
   - `DELETE /api/pipeline-run/:run_id` 移出 HTTP 全局事务切面后，在应用层 `pipelinerunsvc.Service.DeletePipelineRun` 中通过显式注入的 `TransactionRunner.RunInTransaction` 保障数据库四表级联删除的原子提交；在数据库成功提交后，再在事务外执行本地磁盘文件清理 `workspace.RemoveRunFiles`。
3. **彻底消除领域仓储中的事务控制（路线 B）**：
   - 彻底清除所有领域仓储（`application`, `service`, `gateway`, `pipeline`, `pipeline_run`, `role`, `user` 等）内部的 `tx.RunInTx`。
   - 仓储仅通过 `r.q(ctx)` 执行 SQL，其事务生命周期完全由外部上下文决定。
4. **适配相关单元测试**：
   - 仓储层复合写入回滚测试直接通过外部传入带有事务的 context（`database.BeginTx` + `tx.WithTx`）验证回滚行为。
   - 应用层用例测试注入 `TransactionRunner` 验证事务边界。

## Non-goal

- 不废除基于 HTTP 动词拦截与标准 Context 传播的 `tx.Middleware` 全局切面机制。
- 不在已有 HTTP 写请求切面的常规业务用例（如 `RemoveGateway`、`DeleteUser`、`CreateProject` 等）中重复注入或调用 `TransactionRunner`。
- 不废弃基于 Context 的事务传播与 `dbmodel.Queries` 连接动态感知机制。
- 不改变任何现有 API 路径、请求/响应结构体与 HTTP 状态码。
- 不为单表、单语句写入引入显式 UoW。
- 不在 Repository 接口层暴露泛化的 `RunInTransaction`。
- 不修改已执行的历史数据库迁移文件。

## User scenarios

1. 用户点击测试 SSH 连通性时，若目标机器离线导致连接超时，请求全程不开启数据库事务，不占用数据库连接池。
2. 用户登录时，Cloudflare Turnstile 验证与密码哈希计算不持有数据库连接；登录历史记录与时间更新在验证完成后以普通上下文直接写入。
3. 用户调用 Compose 预览或点击登出时，系统不开启事务也不进行响应缓冲，直接快速返回结果。
4. 删除流水线运行记录时，在应用层短事务内原子删除数据库记录与绑定，提交成功后再执行本地文件删除；文件物理操作彻底与数据库事务解耦。
5. 仓储层测试通过标准 `context` 注入事务并触发回滚，清晰验证多表操作在事务边界下的原子性。

## Acceptance

- [x] **HTTP 事务切面名单收敛** (`internal/bootstrap/http.go`)：
  - [x] `MutatingUnitOfWork` 的 `skipPaths` 新增：
    - `"/api/project-initialization/environment/test"`
    - `"/api/auth/login"`
    - `"/api/auth/logout"`
    - `"/api/service/:service_id/preview"`
    - `"/api/version/:version_id/preview"`
    - `"/api/pipeline-run/:run_id"`
- [x] **登录用例事务解耦** (`internal/api/http/handler/auth/handler.go`, `internal/application/auth/usecase/service.go`)：
  - [x] 确认登录接口在无切面事务状态下，Turnstile 验证、密码比对与 JWT 签发正常执行。
  - [x] `MarkUserLoggedIn` 与 `SaveLoginHistory` 在无外层事务时正常写入数据库。
- [x] **流水线删除应用层短 UoW 与文件清理时序重构** (`internal/application/pipeline_run/usecase/service.go`)：
  - [x] `pipelinerunsvc.Service` 注入 `TransactionRunner`。
  - [x] `DeletePipelineRun` 中先通过 `s.transactionRunner.RunInTransaction` 原子删除数据库 4 张表并提交；提交成功后，再在事务外执行 `workspace.RemoveRunFiles(run.Id)`。
  - [x] 若数据库删除失败，磁盘文件完全不被修改。
- [x] **领域仓储彻底去事务化（路线 B）**：
  - [x] 清除 `internal/repository/impl/sqlc/role/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/user/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/service/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/gateway/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/pipeline/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/pipeline_run/repository.go` 中的 `tx.RunInTx`。
  - [x] 清除 `internal/repository/impl/sqlc/application/repository.go` 中的 `tx.RunInTx`。
- [x] **验证通过**：
  - [x] `task check` 通过（Vue/Prettier/ESLint/Go linter 0 issues）。
  - [x] `go test ./cmd/... ./internal/...` 全量通过。
  - [x] `pipeline_run_deletion_test.go` 验证应用层短 UoW 与文件清理时序。
  - [x] `role/repository_test.go` 与 `user/repository_test.go` 验证上下文事务下的原子性与回滚。

## Decisions

1. **路线 B 仓储去事务化**：彻底消除领域仓储内部的 `tx.RunInTx`。仓储是单纯的数据访问对象（DAO），只依赖 `r.q(ctx)`，其原子性完全由调用方传入的 context 决定。
2. **HTTP 请求事务边界**：面向切面的 `tx.Middleware` 是常规 HTTP 同步写请求的默认且唯一的 UoW。跨聚合编排（如 `gateway.Service.RemoveGateway`、`user.Service.DeleteUser`）直接通过标准 Context 复用该切面事务，不为常规 HTTP 用例引入 `TransactionRunner` 样板代码。
3. **显式 UoW 适用场景**：`TransactionRunner` 仅用于 Worker 异步短写阶段以及被排除在切面外的特殊写请求（如 `route/sync/confirm`、`dialogue`、`pipeline-run/:run_id`）。
4. **排除路由决策**：所有包含外部网络通信（SSH、Cloudflare）、不可逆文件物理 I/O 以及纯读/无写的写动词路由，全部通过 `skipPaths` 显式移出 HTTP 请求级事务切面。
5. **文件删除与数据库一致性决策**：数据库记录是系统可信事实源。删除流水线采用“先原子提交数据库删除，后异步/后续清理磁盘文件”的时序。

## Risk

- **路径模式匹配**：`skipPaths` 中的 `/api/service/:service_id/preview`、`/api/version/:version_id/preview` 与 `/api/pipeline-run/:run_id` 包含路径参数，必须依赖 `matchesPathTemplate` 准确识别与匹配。已在 `internal/infrastructure/database/tx/request_test.go` 中有基准测试覆盖。
- **孤儿文件清理**：若流水线数据库记录删除后服务异常退出导致文件未清理，依赖已有的工作区孤儿清理机制兜底。

## User review notes

- 2026-09-17：纠正对 `gateway.Service.RemoveGateway` 的误判，澄清常规 HTTP 写请求统一由切面事务兜底、无需手动 `TransactionRunner` 的边界规则。
- 2026-09-17：采纳用户决策“路线 B”，彻底消除所有领域仓储内的 `tx.RunInTx`，将事务生命周期管理 100% 收敛到入站切面与应用层显式 UoW。
