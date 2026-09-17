# 事务管理与边界规范治理
最后修改时间: 2026-09-17 16:55:00

Review status: Accepted

Mode: standard

## Background

系统目前具备分工明确的三层事务管理体系：
1. **HTTP 请求切面事务（全局写 UoW）**：在 `internal/infrastructure/database/tx/request.go` 中通过 `tx.Middleware` 作为 Gin 中间件（`MutatingUnitOfWork`），对所有 `POST`、`PUT`、`PATCH`、`DELETE` 请求统一开启事务，使用标准 `context.Context` 传播，并通过 `bufferedResponseWriter` 实现两阶段提交保障（仅在 Handler 正常且 `sqlTx.Commit()` 成功后才 flush 响应）。**常规 HTTP 业务写请求（包括跨多个 Repository 的编排）默认直接复用此切面事务，应用层无须引入额外的 `TransactionRunner`。**
2. **应用层显式短 UoW（`TransactionRunner`）**：其使用场景严格限定在**缺乏入站请求事务切面**的场景：
   - 异步 Worker 后台任务：Worker 仅持有 `*sql.DB`，容器运行与长任务在事务外执行，仅在最终多表持久化阶段（如 `forkBuildVersion`）使用显式短 UoW。
   - 被排除在切面外的特殊请求（`skipPaths`）：如 `/api/route/sync/confirm`（前后有 Traefik 网络 I/O）与 `/api/dialogue`（包含 LLM 外部调用），业务需要将短数据库写入包裹在显式 UoW 中。
3. **Repository 内部复合持久化（`RunInTx`）**：Repository 通过 `dbmodel.Queries(ctx, r.db, ...)` 与 `tx.DbTXFrom(ctx)` 获取连接；对于单聚合内部的多语句写入，通过 `tx.RunInTx` 感知上下文事务：有外层事务时直接复用执行（不发 `BEGIN`），无外层事务时（如测试、离线脚本）自洽开启独立短事务。

经全库代码与事务边界审查，已精确定位以下具体违规与隐患点，本规范文档定义确定的改造任务项：

1. **外部网络与不可逆本地 I/O 处于 HTTP 事务生命周期内（高风险）**：
   - `POST /api/project-initialization/environment/test`：调用 `TestSSHReachability` 执行真实远程 SSH 网络连接，未加入 `skipPaths`，导致网络探测与超时等待期间无谓占用数据库事务与写锁。
   - `POST /api/auth/login`：Handler 先调用 Cloudflare Turnstile 外网 HTTP 验证接口，再计算 CPU 密集的 `bcrypt.CompareHashAndPassword`，全程处于 HTTP 切面事务中。
   - `DELETE /api/pipeline-run/:run_id`：Handler 在切面事务内调用 `workspace.RemoveRunFiles` 执行物理文件删除。文件删除不可逆，且耗时不确定。
2. **纯计算或无写操作的 POST 接口空跑事务**：
   - `POST /api/service/:service_id/preview` 与 `POST /api/version/:version_id/preview`：仅生成 Docker Compose YAML 文本，纯读无写，未加入 `skipPaths`，每次请求空跑 `BeginTx`、缓冲与 `Commit`。
   - `POST /api/auth/logout`：仅做请求解析并返回 204，无任何数据库操作，未加入 `skipPaths`。
3. **Repository 内部事务使用不规范**：
   - `service.Repository.DeleteService`：内部仅执行单条 `q.DeleteService`，冗余包裹了 `tx.RunInTx`，违反“不为单表、单语句写入引入显式 UoW”的原则。
   - `application.Repository.DeleteGatewayApplication`：内部直接调用 `q.DeleteGatewayConfigByApplication` 越界删除网关表数据，破坏聚合边界，且与 `gateway.Repository.DeleteGatewayConfig` 重复。
   - `role.Repository.CreateRole` 与 `UpdateRole`：包含写主表与删除并重写权限表的多语句操作，但未包裹 `tx.RunInTx`，脱离 HTTP 切面时无法保证原子性。
   - `user.Repository.SetUserRoles`：包含清空角色、批量插入角色和更新时间戳的多语句操作，未包裹 `tx.RunInTx`。

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
   - `DELETE /api/pipeline-run/:run_id` 移出 HTTP 全局事务切面后，由 `pipeline_run.Repository.DeletePipelineRun` 内部原生的 `tx.RunInTx` 保障数据库删除的独立原子提交；数据库删除成功后，再在事务外执行本地磁盘文件清理 `workspace.RemoveRunFiles`。
3. **消除 Repository 冗余与越界操作**：
   - 改造 `service.Repository.DeleteService`：去除 `tx.RunInTx` 包装，直接通过 `r.q(ctx).DeleteService` 执行单语句删除。
   - 改造 `application.Repository.DeleteGatewayApplication`：删除该越界方法，统一使用 `DeleteApplication`；网关配置的清理完全收敛到 `gateway.Repository.DeleteGatewayConfig` 负责。
4. **补齐 Repository 复合写入原子性**：
   - 为 `role.Repository.CreateRole` 与 `UpdateRole` 包裹 `tx.RunInTx(ctx, r.db, ...)`，确保角色与权限关联原子写入。
   - 为 `user.Repository.SetUserRoles` 包裹 `tx.RunInTx(ctx, r.db, ...)`，确保用户角色清空、批量写入与时间戳更新原子完成。

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
4. 离线脚本、数据初始化或自动化测试直接调用 `roleRepo.CreateRole` 或 `userRepo.SetUserRoles` 时，即使插入权限或角色中途出错，也会完整回滚，数据库不残留脏数据。
5. 删除流水线运行记录时，先在独立短事务内原子删除数据库记录与绑定，提交成功后再执行文件删除；文件物理操作彻底与数据库事务解耦。

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
- [x] **流水线删除与文件清理时序重构** (`internal/application/pipeline_run/usecase/service.go`)：
  - [x] `DeletePipelineRun` 调整执行顺序：先执行 `s.store.DeletePipelineRun(ctx, projectId, run.Id)`；在数据库成功删除并提交后，再执行 `workspace.RemoveRunFiles(run.Id)`。
  - [x] 若数据库删除失败，磁盘文件完全不被修改。
- [x] **Repository 事务精简与规范**：
  - [x] `service.Repository.DeleteService` 去除 `tx.RunInTx`，改为单语句直调。
  - [x] `application.Repository` 移除 `DeleteGatewayApplication` 及内部对 `gateway_config` 表的操作（已于前序提交 `d2a81f62` 收敛移除，现状统一使用 `DeleteApplication` 与 `gatewayCore.RemoveGateway`）。
  - [x] `role.Repository.CreateRole` 与 `UpdateRole` 内部包裹 `tx.RunInTx`。
  - [x] `user.Repository.SetUserRoles` 内部包裹 `tx.RunInTx`。
- [x] **验证通过**：
  - [x] `task check` 通过。
  - [x] `go test ./cmd/... ./internal/...` 全量通过。
  - [x] 新增针对 `roleRepo` 与 `userRepo` 复合写入在独立调用下的事务回滚测试。

## Decisions

1. **HTTP 请求事务边界**：面向切面的 `tx.Middleware` 是常规 HTTP 同步写请求的默认且唯一的 UoW。跨聚合编排（如 `gateway.Service.RemoveGateway`、`user.Service.DeleteUser`）直接通过标准 Context 复用该切面事务，不为常规 HTTP 用例引入 `TransactionRunner` 样板代码（沿用 `20260902-user-email-authentication` 既定原则）。
2. **显式 UoW 适用场景**：`TransactionRunner` 仅用于 Worker 异步短写阶段以及被排除在切面外的特殊请求（如 `route/sync/confirm`、`dialogue`）。
3. **排除路由决策**：所有包含外部网络通信（SSH、Cloudflare）、不可逆文件物理 I/O 以及纯读/无写的写动词路由，全部通过 `skipPaths` 显式移出 HTTP 请求级事务切面。
4. **文件删除与数据库一致性决策**：数据库记录是系统可信事实源。删除流水线采用“先原子提交数据库删除，后异步/后续清理磁盘文件”的时序。
5. **复合持久化自治决策**：单聚合的多语句写入在 Repository 内部自治使用 `tx.RunInTx`，保证脱离切面环境时的自洽原子性。

## Risk

- **路径模式匹配**：`skipPaths` 中的 `/api/service/:service_id/preview`、`/api/version/:version_id/preview` 与 `/api/pipeline-run/:run_id` 包含路径参数，必须依赖 `matchesPathTemplate` 准确识别与匹配。已在 `internal/infrastructure/database/tx/request_test.go` 中有基准测试覆盖。
- **孤儿文件清理**：若流水线数据库记录删除后服务异常退出导致文件未清理，依赖已有的工作区孤儿清理机制兜底。

## User review notes

- 2026-09-17：纠正对 `gateway.Service.RemoveGateway` 的误判，澄清常规 HTTP 写请求统一由切面事务兜底、无需手动 `TransactionRunner` 的边界规则，收敛文档任务项。
