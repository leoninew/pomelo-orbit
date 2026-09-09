# Environment 分层边界收敛计划
最后修改时间: 2026-09-09 17:52:42

Review status: Accepted

Mode: standard

## Basis

- Requirement: `docs/requirement/20260909-environment-architecture-boundaries.md`（Accepted）
- 无独立 Spec；标准模式直接计划。
- 并行需求 `docs/requirement/20260909-environment-workspace-ownership.md` 不在本计划实现。本计划只收口分层与装配，不定义工作目录产品来源。

## Delivery boundary

Environment 读/改/Probe/Initialize 对外返回 `environment/dto` View；HTTP 与 MCP 只消费 View。local 平台/主机/用户由组合根注入控制面快照。Probe 升为 `environment/port`。HTTP 与 Worker 共用一个 Deployment Runtime 工厂。HTTP status 映射离开 `common/errors`。gateway/settings 不再吃整份 `config.Config`。`routes.Dependencies` 不再持有 `*sql.DB`。`docs/architecture/backend.md` 改为现行 `application/<domain>`。

不改工作目录保存/校验/runtime 取路径；不恢复 install command；不改 Initialize 一次性认证；不把 Probe 放进请求 UoW；不改 HTTP 错误 JSON 形状 `{code,error,requestId}`。

## Parallel file locks

本会话可改：

- `internal/common/errors/**`
- `internal/api/http/response/**`
- `internal/api/http/routes/routes.go`、`environment.go`
- `internal/application/gateway/**`、`internal/application/settings/**`
- `docs/architecture/backend.md`
- `internal/application/environment/port/prober.go`（新建）
- Environment View DTO、handler/MCP 映射形状
- bootstrap 共享 Runtime 工厂；local runner 路径语义保持，暂时仍传入 `cfg.Workspace.Deployment`

并行会话可改：sql/proto/web 环境页、`internal/model/environment.go`、environment usecase 的保存/校验/probe 使用库存 workspace、local runner `ServiceDir`、execution log 拆分、config 对 `workspace.deployment` 的 CD 用途、product/cd-runtime/deployment 文档。

共享并需协调：

- `internal/application/environment/usecase/*.go`：本会话返回 View、依赖 Prober、`toView()` / `WithLocalDisplay`
- `internal/application/environment/dto/environment.go`：本会话加 View；workspace 会话可加 local `workspace_root` 到 `UpdateInput`
- `internal/bootstrap/http.go`、`worker.go`：本会话工厂 + settings/gateway 构造 + local display 注入；workspace 会话停止把 `cfg.Workspace.Deployment` 当 CD 根
- handler/MCP environment：本会话改成消费 View；workspace 会话只加 local `workspace_root` 读写，不引入 View

交接约定：workspace 会话继续让 usecase 返回 `model.Environment`，直到本会话落地 View。冲突时保住「工作目录来自 Environment 库存」，本会话不得写回 yaml 投影。

## Implementation steps

1. 把 HTTP status 从 `common/errors` 挪到 `api/http/response`。
   - `Classification` 只保留 `Code` + `Message`。
   - 删除 `StatusCode`、`NewForHTTPStatus`，以及 `classificationForKind` / `kindForHTTPStatus` 对 `net/http` 的依赖。
   - `Classify` 仍给 MCP `toolError` 与 dialogue 使用 Code/Message。
   - `response.HTTPStatus(err)`、`statusForKind`、`kindForHTTPStatus` 承接映射；`WriteStatusError` 改为 `apperror.New(kind, message)`。
   - JSON 形状、401 `WWW-Authenticate`、5xx 记入 `c.Error` 保持不变。
   - application 测试把 `apperror.StatusCode(err) != 400/401/404/409/http.Status*` 改成 `apperror.IsKind(...)`：400 对应 `KindValidation`，401 对应 `KindUnauthorized`，404 对应 `KindNotFound`，409 对应 `KindConflict`，503 对应 `KindUnavailable`。`delete_test.go` 的 `Classify(err).StatusCode` 同样改 `IsKind`。
   - `internal/common/errors/error_test.go` 只断言 Code/Message；HTTP 映射测试放到 `internal/api/http/response`。

2. 收口构造注入。
   - `settingssvc.New(definitions []settingsdto.Definition, envStore)`；导出 `Definitions(cfg config.Config)`（若 `cfg.Base` 非空则用 Base）。构造函数不再持有 `config.Config`。bootstrap 与 settings 测试改调 `Definitions(cfg)`。
   - `gatewaysvc.New(..., traefik config.TraefikConfig, ...)`，不再收整份 `config.Config`。`resolvePath` 参数保持现签名位置，即使仍未使用也不删。`testGatewayConfig()` 改为返回 `TraefikConfig`。
   - `routes.Dependencies` 用 `MutatingUnitOfWork gin.HandlerFunc` 替换 `Database *sql.DB`。`routes.go` 去掉 `database/sql` 与 `tx` import。bootstrap：`databasetx.Middleware(database, "/api/route/sync/preview", "/api/route/sync/confirm", "/api/project/:project_id/environment/probe")`。Probe 仍排除；Initialize 事务行为不动。

3. Probe 端口化。
   - 新增 `internal/application/environment/port/prober.go`，定义 `Prober`：`Probe(ctx, Environment, DeploymentSSHPrivateKey) (string, error)` 与 `ProbeLocal(ctx) error`。
   - 删除 usecase 未导出的 `environmentProber` / `localEnvironmentProber` 以及 `probeOutcome` 里的类型断言。`environmentrunner.Prober` 已具备两个方法，直接注入。
   - 测试 double `probeEnvironmentProber` 补 `ProbeLocal`。
   - `New(...)` 保持 6 个参数，prober 类型改为 `environmentport.Prober`。

4. Environment View。
   - 在 `environment/dto` 增加 `View`、`LocalTargetView`、`SSHTargetView`、`LocalDisplaySnapshot`。SSH View 不含 credential / private key。
   - `EnvironmentForUser` / `UpdateForUser` / `ProbeForUser` / `InitializeForUser` 返回 `environmentdto.View`。`BootstrapForProject` 仍返回 `model.Environment`。
   - `Service.WithLocalDisplay(snapshot) Service` 注入快照，避免所有测试改构造。bootstrap：`environmentsvc.New(...).WithLocalDisplay(localEnvironmentDisplay())`。OS 读取（GOOS / hostname / current user）只在 bootstrap，不在 handler。
   - `toView()`：local 平台/主机/用户来自快照；local `workspace_root` 只取库存（当前 local 映射为空则空字符串）。禁止再用 yaml 或 OS 填工作目录。

5. HTTP / MCP 消费 View。
   - handler `New(logger, service, authenticator)`，删除 `localWorkspaceRoot` / `resolveLocalTargetInfo` / OS 读取。`environmentResponse(view)` 直接映射 proto。
   - `routes/environment.go` 不再把 `r.cfg.Workspace.Deployment` 传给 handler。
   - MCP 删除 `Dependencies.LocalWorkspaceRoot`。`environmentOutput(view)` 输出与 HTTP 同一套 local 字段（含 platform/host/username）。fake `environmentToolService` 返回 `environmentdto.View`。原断言 yaml `/srv/orbit/deployment` 的测试改为把该路径放在 fake View 上。
   - `TestUpdateForUserKeepsDeploymentCredentialForSSHTargetChange` 若当前断言 `updated.SSH.CredentialId`，改为断言 `store.environment.SSH.CredentialId`，因为 View SSH 无 credential。
   - MCP `EnvironmentService` 接口返回类型同步为 View。

6. Deployment Runtime 工厂。
   - 在 bootstrap 增加 `newDeploymentRuntime(workspaceRoot string, resolvePath localrunner.PhysicalPathResolver) (*localrunner.Runtime, deploymentport.Runtime)`。
   - `http.go` 与 `worker.go` 都调用它。本会话仍传入 `cfg.Workspace.Deployment`，不改 localrunner 路径语义。不要合并 command/execution service。

7. 活文档。
   - `docs/architecture/backend.md` 模块表去掉 `application/cd`、`application/ci`，改为现行域：`application`、`auth`、`credential`、`deployment`、`dialogue`、`environment`、`gateway`、`pipeline`、`pipeline_run`、`project`、`repository`、`role`、`route`、`service`、`settings`、`user`。不趁机清理 sqlx。

## Files to change

- `internal/common/errors/error.go`、`error_test.go`
- `internal/api/http/response/response.go`，新增 HTTP status 映射测试
- 所有 `apperror.StatusCode` / `Classify(...).StatusCode` 测试：user/auth/credential/repository/gateway/project/route/role
- `internal/application/settings/usecase/service.go` 及 settings 测试
- `internal/application/gateway/usecase/service.go`、`initial_test.go`、factory/delete 测试
- `internal/api/http/routes/routes.go`、`environment.go`
- `internal/application/environment/port/prober.go`（新）
- `internal/application/environment/dto/environment.go`
- `internal/application/environment/usecase/service.go`、`probe.go`、`probe_test.go`、`target_test.go`、`service_integration_test.go`
- `internal/api/http/handler/environment/handler.go`、`environment.go`、`environment_test.go`
- `internal/api/mcp/delivery/types.go`、`mapper.go`、`environment_tools.go`、`server_test.go`
- `internal/bootstrap/http.go`、`worker.go`、必要时 `app.go` 的 display 快照辅助函数
- `docs/architecture/backend.md`

不改：`web/`、sql/migration、proto、`internal/model/environment.go`、local runner 路径算法、execution log、`workspace.deployment` 的产品含义。

## Verification plan

实现完成后由 Verification 阶段执行；本计划阶段不跑门禁。

- View：GET/Update/Probe/Initialize 返回 View；local 展示来自注入快照，不读 handler OS，不把 yaml 填进 `workspace_root`。新增 `view_test.go` 覆盖「有快照 / 无 yaml」。
- Prober：usecase 只依赖 `environment/port.Prober`；local/ssh probe 测试 double 带 `ProbeLocal`。
- Runtime：HTTP 与 Worker 走同一 `newDeploymentRuntime`。
- 错误：`common/errors` 无 `net/http`；HTTP JSON 仍为 `{code,error,requestId}`；MCP `toolError` 仍用 Classify 的 Code/Message。
- 注入：settings/gateway 构造无 `config.Config`；routes 无 `*sql.DB`；Probe 路径仍排除请求 UoW。
- 文档：`backend.md` 无 `application/cd` / `application/ci`。
- 私钥不进 View/HTTP/MCP。
- 命令：`task check` 与 `go test ./cmd/... ./internal/...`。不跑 frontend。

## Blockers

无。workspace 会话并行改共享文件时，以库存工作目录为准，本会话不回退 yaml 投影。

## Assumptions

- 当前 local 库存 `workspace_root` 为空；本会话落地后 local GET/MCP 可能暂时显示空工作目录，直到 workspace 会话写入库存。这是拆分后的预期窗口，不在本计划回填 yaml。
- `environmentrunner.Prober` 已实现 `Probe` 与 `ProbeLocal`，无需改 adapter 行为。
- Runtime 工厂本会话仍接收 `cfg.Workspace.Deployment`，只消除重复 `New`；真正改参数归 workspace 会话。
- settings 定义列表从 `Definitions(cfg)` 一次性注入后，服务不再回读进程配置。

## Risks

- 与 workspace 会话同时改 bootstrap/handler/dto/usecase，可能产生合并冲突。
- `StatusCode` 散落在多个 application 测试，漏改会编译失败；必须一次清干净。
- settings 改快照类型时可能漏键；`Definitions(cfg)` 必须覆盖现有 `settingDefinitions` 全量键。
- 拿掉 routes `*sql.DB` 时若漏传 Probe 排除名单，Probe 会重新进入请求事务。
- Runtime 工厂若被扩成合并 command/execution，会超出本需求。

## Rollback

回退本计划提交即可。不通过重新把 HTTP status 放回 `common/errors`、或让 handler 再读 yaml/OS 来部分回滚。

## User review notes

- 用户要求标准模式进入计划，并与 workspace 会话并行。
- 用户已否决把工作目录放回 yaml / Settings；本计划只禁止 adapter 私自拼装。
