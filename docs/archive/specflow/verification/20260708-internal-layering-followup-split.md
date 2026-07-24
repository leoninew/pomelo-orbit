# internal 分层后续逐条拆分验证
最后修改时间: 2026-07-09 13:46:57

Review status: Accepted

## Verification target

本次验证范围为 light / 轻量模式下的 issue 5：CD workspace 拆分。

对应 requirement 文档：

- `docs/requirement/20260708-internal-layering-followup-split.md`

本次只验证 issue 5，不验证 issue 6 及后续待处理项。

## Requirement alignment

需求要求：

- 将 `internal/application/cd/workspace.go` 与 `internal/workflow/activity/cd/workspace.go` 中重复的 CD workspace 规则拆到共享位置。
- 共享位置应遵循架构边界，放在 `internal/infrastructure/storage/local/cdworkspace`。
- application 与 workflow activity 直接依赖共享包。
- 不保留 wrapper、别名、适配、兼容转发或新旧逻辑并存。
- 保持既有路径行为不变。

实际实现与 requirement 对齐：

- 新增 `internal/infrastructure/storage/local/cdworkspace`，承载 CD workspace 规则。
- 删除 application 与 workflow activity 下的重复 workspace 文件。
- `internal/application/cd/service.go` 和 `internal/workflow/activity/cd/service.go` 直接使用 `*cdworkspace.Workspace`。
- 测试中的 resolver 注入直接使用 `cdworkspace.NewWithResolver(...)`。
- 未新增 wrapper-only 文件、类型别名或兼容构造函数。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 issue 5 拆分策略实施。

## Actual diff summary

当前 diff 覆盖以下文件：

- `docs/requirement/20260708-internal-layering-followup-split.md`
  - 记录 issue 5 架构依据、拆分策略、实施结果、验收标准和验证结果。
- `internal/infrastructure/storage/local/cdworkspace/workspace.go`
  - 新增共享 CD workspace 实现。
- `internal/infrastructure/storage/local/cdworkspace/workspace_test.go`
  - 迁移并保留原 workspace 行为测试。
- `internal/application/cd/service.go`
  - workspace 字段与构造函数改为直接依赖 `cdworkspace`。
- `internal/application/cd/compose_test.go`
  - resolver 注入改为 `cdworkspace.NewWithResolver(...)`。
- `internal/workflow/activity/cd/service.go`
  - workspace 字段与构造函数改为直接依赖 `cdworkspace`。
- `internal/workflow/activity/cd/deployment_execution_test.go`
  - resolver 注入改为 `cdworkspace.NewWithResolver(...)`。
- `internal/application/cd/workspace.go`
  - 删除重复 workspace 实现。
- `internal/workflow/activity/cd/workspace.go`
  - 删除重复 workspace 实现。
- `internal/workflow/activity/cd/workspace_test.go`
  - 删除原位置测试；测试内容迁移到共享包。

## Expected vs actual changed files

预期改动：

- 新增 `internal/infrastructure/storage/local/cdworkspace`。
- 删除 application/activity 下重复 workspace 实现。
- 更新 application/activity 调用点。
- 迁移 workspace 行为测试。
- 更新 requirement 记录 issue 5 状态。

实际改动与预期一致。

未发现 issue 5 范围外的产品代码重构；未触碰前端、迁移文件、API 路由、runner 合并或后续 issue。

## Acceptance checklist

- [x] `internal/infrastructure/storage/local/cdworkspace` 新增并承载 CD workspace 规则。
- [x] `internal/application/cd/workspace.go` 删除。
- [x] `internal/workflow/activity/cd/workspace.go` 删除。
- [x] application 与 workflow activity 都直接依赖 `*cdworkspace.Workspace`。
- [x] 不保留 wrapper、别名、适配、兼容转发或新旧逻辑并存。
- [x] workspace 行为保持不变：
  - app dir 仍为 `dataRoot/cd/<appCode>`；
  - deployment log path 仍为 `dataRoot/cd/<appCode>/deployments/<deploymentId>.log`；
  - `PhysicalDir` 仍返回 slash 格式 physical root；
  - `PhysicalAppDir` 仍返回 slash 格式 `<physicalRoot>/cd/<appCode>`；
  - physical root resolver 仍只执行一次并缓存错误。
- [x] 后端检查通过。

## Test results

已运行命令：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint run ./cmd/... ./internal/...`：通过，输出 `0 issues.`。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/cdworkspace
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd
```

## Missed or expanded scope

未发现范围偏离。

本次没有处理以下事项，符合 issue 5 边界：

- 未拆 `CommandRunner`。
- 未移动 `os.RemoveAll(s.workspace.AppDir(...))`。
- 未调整 deployment execution 编排。
- 未处理 issue 6 及后续条目。
- 未修改前端或数据库迁移。

## Risks

当前未发现遗留实现风险。

注意：`git status --short` 当前显示本次变更位于 index 侧，包括 rename/delete 识别；本次验证未执行 `git add`、`git commit` 或 `git push`。

## Incomplete items

无。

## Conclusion

Issue 5 CD workspace 拆分已完成并通过验证。实现符合 requirement 中的分层目标：CD workspace 规则归入 `internal/infrastructure/storage/local/cdworkspace`，application 与 workflow activity 直接依赖共享包，未保留兼容层或 wrapper-only 文件，既有路径行为和后端检查均通过。

---

# Issue 6：task runtime model 边界验证

## Verification target

本次追加验证范围为 light / 轻量模式下的 issue 6：task runtime model 边界拆分。

对应 requirement 文档：

- `docs/requirement/20260708-internal-layering-followup-split.md`

本节只验证 issue 6，不重新验证 issue 5，也不验证 issue 7 及后续待处理项。

## Requirement alignment

需求要求：

- `Task` runtime model 不应定义在 `internal/repository/impl/sqlc/task`。
- `Task` 应迁移到 queue 边界，即 `internal/queue/task`。
- sqlc task repository 只保留持久化、transaction 和 SQLC row conversion 职责。
- queue service、worker、worker handlers、HTTP task handler、application CI/CD 不再为了 `Task` 类型依赖 sqlc repository impl。
- 不保留类型别名、wrapper-only 文件、适配层或新旧模型并存。
- 保持任务入队、claim、complete、fail、HTTP create/get task、CI/CD worker handler 行为不变。

实际实现与 requirement 对齐：

- 新增 `internal/queue/task/model.go`，定义 `Task` runtime model。
- `internal/repository/impl/sqlc/task/repository.go` 删除本地 `Task` 定义，并直接返回 `*tasksvc.Task`。
- `internal/queue/task/service.go` 使用本包 `Task`，不再 import sqlc task repository 只是为了模型类型。
- worker、worker handler、HTTP task handler、application CI/CD 接口均直接使用 `tasksvc.Task`。
- 未新增类型别名、wrapper-only 文件或兼容适配层。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 issue 6 拆分策略实施。

## Actual diff summary

当前 issue 6 diff 覆盖以下文件：

- `docs/requirement/20260708-internal-layering-followup-split.md`
  - 记录 issue 6 架构依据、拆分策略、实施结果、验收标准和验证结果。
- `internal/queue/task/model.go`
  - 新增 queue runtime task model。
- `internal/repository/impl/sqlc/task/repository.go`
  - 删除本地 `Task` 定义；`ClaimNext`、`FindById`、`taskFromSQLC` 改用 `tasksvc.Task`。
- `internal/queue/task/service.go`
  - repository interface 和 service 返回值改用本包 `Task`。
- `internal/worker/worker.go`
  - worker handler、router 和 repository interface 改用 queue task model。
- `internal/worker/handler/ci/handler.go`
  - CI worker handler 参数改用 `tasksvc.Task`。
- `internal/worker/handler/cd/handler.go`
  - CD worker handler 参数改用 `tasksvc.Task`。
- `internal/api/http/handler/task/handler.go`
  - task response 转换直接接收 `*tasksvc.Task`。
- `internal/application/ci/repository.go`
  - application CI 的 `TaskService` interface 返回 `*tasksvc.Task`。
- `internal/application/cd/service.go`
  - application CD 的 `TaskService` interface 返回 `*tasksvc.Task`。
- `internal/worker/worker_test.go`
  - handler test callback 改用 `tasksvc.Task`；保留 concrete sqlc repository 构造用于集成式 worker 测试。
- `internal/worker/handler/ci/handler_test.go`
  - 测试 task 构造改用 `tasksvc.Task`。
- `internal/worker/handler/cd/handler_test.go`
  - 测试 task 构造改用 `tasksvc.Task`。

## Expected vs actual changed files

预期改动：

- 新增 `internal/queue/task/model.go`。
- 删除 sqlc task repository 中的 runtime model 定义。
- 更新 queue、worker、worker handler、HTTP task handler、application CI/CD 调用点。
- 更新相关测试中直接构造 task 的位置。
- 更新 requirement 记录 issue 6 状态。

实际改动与预期一致。

仍保留 `internal/repository/impl/sqlc/task` import 的位置仅用于构造或 wiring concrete repository：

- `internal/bootstrap/provider.go`
- `internal/api/http/server.go`
- `internal/api/http/server_test.go`
- `internal/test/e2e/mysql_e2e_test.go`
- `internal/worker/worker_test.go`

这些位置不是为了使用 `Task` model，因此不构成 issue 6 的分层泄漏。

未发现 issue 6 范围外的产品代码重构；未触碰前端、数据库迁移、API 路由形态、worker 生命周期或 retry 策略。

## Acceptance checklist

- [x] `internal/repository/impl/sqlc/task/repository.go` 不再定义 `Task`。
- [x] `Task` model 位于 `internal/queue/task/model.go`。
- [x] queue service、worker、worker handlers、HTTP task handler、application CI/CD 不再为了 `Task` 类型 import `internal/repository/impl/sqlc/task`。
- [x] sqlc task repository 直接返回 `*tasksvc.Task`，只保留持久化与 row conversion 职责。
- [x] 不保留类型别名、wrapper-only 文件、适配层或新旧模型并存。
- [x] 任务入队、claim、complete、fail、HTTP create/get task、CI/CD worker handler 行为不变。
- [x] 后端检查通过。

## Test results

已运行命令：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint run ./cmd/... ./internal/...`：通过，输出 `0 issues.`。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/task
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/ci
?    gitee.com/leoninew/PomeloOrbit-go/internal/queue/task [no test files]
ok   gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task
ok   gitee.com/leoninew/PomeloOrbit-go/internal/worker
ok   gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/ci
```

## Missed or expanded scope

未发现范围偏离。

本次没有处理以下事项，符合 issue 6 边界：

- 未调整 task API 路由或 request/response contract。
- 未重构 worker 生命周期、并发模型、polling 或 retry 策略。
- 未迁移 task type constants。
- 未修改数据库 schema 或已执行迁移。
- 未处理 issue 7 及后续条目。
- 未修改前端。

## Risks

当前未发现遗留实现风险。

注意：本次验证观察到 issue 6 代码变更和 requirement 文档变更已处于 index 侧；追加 verification 文档后，verification 文档本身为新的工作区变更。本次验证未执行 `git add`、`git commit` 或 `git push`。

## Incomplete items

无。

## Conclusion

Issue 6 task runtime model 边界拆分已完成并通过验证。实现符合 requirement 中的分层目标：`Task` runtime model 已迁入 `internal/queue/task`，sqlc task repository 只保留持久化与 row conversion，queue、worker、handler 和 application 调用点不再为了模型类型依赖 sqlc repository impl，未保留兼容层、别名或适配层，后端检查全部通过。

---

# Issue 7：logger/logstore 命名与边界验证

## Verification target

本次追加验证范围为 light / 轻量模式下的 issue 7：logger/logstore 命名与边界拆分。

对应 requirement 文档：

- `docs/requirement/20260708-internal-layering-followup-split.md`

本节只验证 issue 7，不重新验证 issue 5/6，也不验证 issue 8 及后续待处理项。

## Requirement alignment

需求要求：

- 区分 application/server runtime logger 与 CI/CD execution log storage。
- 将实际承载 execution log 本地文件读写的实现从 `internal/infrastructure/logger/logstore` 移到更准确的本地存储边界。
- 新位置为 `internal/infrastructure/storage/local/executionlog`，concrete type 表达为 `executionlog.Store`。
- application / workflow 消费侧不继续暴露旧 concrete infrastructure type，而是按“谁消费，谁定义”定义最小接口。
- 删除无调用点的 `Exists` 方法。
- 不保留旧包、类型别名、转发 wrapper、兼容层或新旧实现并存。
- 不修改 runtime logger、CI/CD workspace 路径规则、execution 编排或 API contract。
- 保持 execution log 读写行为不变。

实际实现与 requirement 对齐：

- 新增 `internal/infrastructure/storage/local/executionlog/store.go`，提供 `Store.Writer` 与 `Store.Read`。
- 删除 `internal/infrastructure/logger/logstore/logstore.go`。
- application CI/CD 定义并依赖本包 `LogReader` 接口，只暴露 `Read` 能力。
- workflow activity CI 定义并依赖本包 `ExecutionLogStore` 接口，暴露 `Writer` + `Read` 能力。
- workflow activity CD 定义并依赖本包 `LogWriter` 接口，只暴露 `Writer` 能力。
- bootstrap 与 HTTP server 作为组装位置构造并注入 `executionlog.Store{}`。
- 未迁移 `Exists`，且新包中不存在该方法。
- 未改 `internal/infrastructure/logger/logging`。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 issue 7 拆分策略实施。

## Actual diff summary

当前 issue 7 diff 覆盖以下文件：

- `docs/requirement/20260708-internal-layering-followup-split.md`
  - 记录 issue 7 架构依据、拆分策略、实施结果、验收标准和验证结果。
- `internal/infrastructure/storage/local/executionlog/store.go`
  - 从旧 logstore 迁入 execution log 本地读写实现，并将类型命名为 `Store`。
  - 保留 `Writer` / `Read` 行为；不迁移未使用的 `Exists`。
- `internal/infrastructure/logger/logstore/logstore.go`
  - 删除旧 package。
- `internal/application/ci/repository.go`
  - 删除旧 logstore import；新增消费侧 `LogReader` 接口；Service 构造和字段改用接口。
- `internal/application/cd/service.go`
  - 删除旧 logstore import；新增消费侧 `LogReader` 接口；Service 构造和字段改用接口。
- `internal/workflow/activity/ci/service.go`
  - 删除旧 logstore import；新增消费侧 `ExecutionLogStore` 接口；Service 构造和字段改用接口。
- `internal/workflow/activity/ci/executor.go`
  - executor 字段改用 `ExecutionLogStore`。
- `internal/workflow/activity/cd/service.go`
  - 删除旧 logstore import；新增消费侧 `LogWriter` 接口；Service 构造和字段改用接口。
- `internal/bootstrap/provider.go`
  - concrete implementation 改为 `executionlog.Store{}`。
- `internal/api/http/server.go`
  - server 字段、构造和 CD service wiring 改用 `executionlog.Store`。
- `internal/workflow/activity/ci/execution_test.go`
  - 测试注入改为 `executionlog.Store{}`。
- `internal/workflow/activity/cd/deployment_execution_test.go`
  - 测试注入改为 `executionlog.Store{}`。

## Expected vs actual changed files

预期改动：

- 新增 `internal/infrastructure/storage/local/executionlog`。
- 删除 `internal/infrastructure/logger/logstore`。
- 更新 application / workflow / bootstrap / HTTP server / 测试调用点。
- 更新 requirement 记录 issue 7 状态。

实际改动与预期一致。

`git diff --cached --name-status` 将旧文件到新文件识别为 rename：

```text
R068 internal/infrastructure/logger/logstore/logstore.go internal/infrastructure/storage/local/executionlog/store.go
```

这符合“迁移实现并删除旧包”的预期；新包没有保留旧 package、类型别名或兼容 wrapper。

未发现 issue 7 范围外的产品代码重构；未触碰前端、数据库迁移、API contract、CI/CD workspace 路径规则、deployment/stage execution 编排或 runtime logger 配置。

## Acceptance checklist

- [x] `internal/infrastructure/logger/logstore/logstore.go` 删除。
- [x] `internal/infrastructure/storage/local/executionlog` 新增并承载 execution log 本地文件读写实现。
- [x] `internal/infrastructure/logger/logging` 保持只负责应用 runtime logger 初始化，不混入 execution log storage。
- [x] application CI/CD 不再直接依赖 concrete `executionlog.Store`；只依赖本包消费侧读取接口。
- [x] workflow activity CI/CD 不再直接依赖旧 `logger/logstore` 包；写入/读取 execution log 通过本包消费侧最小接口完成。
- [x] 未保留旧包、类型别名、转发 wrapper、兼容层或新旧实现并存。
- [x] execution log 行为保持不变：缺失文件读取返回空内容与原 offset；写入自动创建目录并 append；offset 读取返回新 offset。
- [x] 未修改 CI/CD workspace 路径规则、deployment/stage execution 编排、应用 runtime logger 配置或 API contract。
- [x] 后端检查通过。

## Test results

已运行命令：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint run ./cmd/... ./internal/...`：通过，输出 `0 issues.`。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/api/http
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/ci
ok   gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap
?    gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog [no test files]
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci
```

## Missed or expanded scope

未发现范围偏离。

本次没有处理以下事项，符合 issue 7 边界：

- 未重构 `internal/infrastructure/logger/logging`、`slog` 或 `lumberjack` 初始化。
- 未修改 CI/CD workspace 路径规则。
- 未调整 deployment/stage execution 编排。
- 未改变 API contract 或路由。
- 未修改数据库 schema 或已执行迁移。
- 未处理 issue 8 及后续条目。
- 未修改前端。

## Risks

当前未发现遗留实现风险。

注意：本次验证观察到 issue 7 代码变更和 requirement 文档变更已处于 index 侧；追加 verification 文档后，verification 文档本身为新的工作区变更。本次验证未执行 `git add`、`git commit` 或 `git push`。

## Incomplete items

无。

## Conclusion

Issue 7 logger/logstore 命名与边界拆分已完成并通过验证。实现符合 requirement 中的分层目标：CI/CD execution log 本地读写已迁入 `internal/infrastructure/storage/local/executionlog`，旧 `logger/logstore` 删除，application 与 workflow activity 通过消费侧最小接口使用 execution log 能力，未保留兼容层、别名或 wrapper，runtime logger 配置与 execution log 行为保持不变，后端检查全部通过。

---

# Issue 9：HTTP server 路由与依赖组装边界验证

## Verification target

本次追加验证范围为 light / 轻量模式下的 issue 9：`internal/api/http/server.go` 中 HTTP adapter 与 composition-root/bootstrap 职责拆分。

对应 requirement 文档：

- `docs/requirement/20260708-internal-layering-followup-split.md`

本节只验证 issue 9，不重新验证 issue 5/6/7，也不验证 issue 10。

## Requirement alignment

需求要求：

- `api/http` 只保留 Gin 入站适配职责：engine、middleware、route registration、health、static SPA fallback、runtime config 注入。
- repository implementation、application service、infrastructure adapter 的选择与组装移到 `bootstrap/provider.go`。
- `transporthttp.New` 不保留旧签名，不新增兼容 wrapper 或新旧构造路径并存。
- `Handler()` 不再重复构造 CI/CD services，并拆成清晰私有 route registration 函数。
- Cloudflare Turnstile concrete HTTP verifier 不继续留在 `api/http`，由 infrastructure 包承载并通过 auth handler 的接口消费。
- 清理 root `internal/api/http/*_routes_test.go` 中与源码职责不匹配的孤儿测试；需要保留的业务覆盖迁到对应 application 包。
- 不修改 API route path、request/response contract、application service 行为、repository SQL/mapper、worker/bootstrap lifecycle、前端或数据库迁移。

实际实现与 requirement 对齐：

- `internal/api/http/server.go` 当前不再 import sqlc repository implementation、envfile、executionlog、traefik、turnstile infrastructure、jwt 或 task repository implementation。
- `transporthttp.New` 当前签名为 `New(cfg config.Config, logger *slog.Logger, deps ServerDependencies) Server`，未保留旧 `store/taskRepo/maxAttempts` 构造路径。
- `ServerDependencies` 显式接收 HTTP route registration 所需的 application services、authenticator、handler stores、task service 和 turnstile verifier。
- `Handler()` 只创建 Gin engine，并调用 `registerMiddleware`、`registerHealthRoutes`、`registerAuthRoutes`、`registerUserRoutes`、`registerRoleRoutes`、`registerSettingsRoutes`、`registerProjectRoutes`、`registerCIRoutes`、`registerCDRoutes`、`registerTaskRoutes`、`registerFallbackRoutes`。
- `internal/bootstrap/provider.go` 负责构造 token service、sqlc repositories、application services、envfile store、executionlog store、Traefik route manager、mkcert generator、Turnstile verifier，并注入 `transporthttp.New`。
- Cloudflare Turnstile verifier concrete implementation 已迁到 `internal/infrastructure/turnstile/verifier.go`；`api/http` 下没有旧 concrete verifier 或 wrapper。
- `internal/test/e2e/mysql_e2e_test.go` 改用 `bootstrap.NewHTTPServer`，不继续调用旧 `transporthttp.New` 签名。
- root HTTP route tests 中实际覆盖 application service / repository / filesystem / task enqueue 业务组合的孤儿测试已删除；保留业务覆盖已按源码归属重建到 `internal/application/{auth,cd,ci,project,role,settings,user}` 的 service integration tests。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 issue 9 拆分策略实施。

## Actual diff summary

当前 issue 9 diff 覆盖以下文件：

- `docs/requirement/20260708-internal-layering-followup-split.md`
  - 记录 issue 9 架构依据、拆分策略、实施结果、验收标准和验证结果。
- `internal/api/http/server.go`
  - 新增 `ServerDependencies`；`New` 接收已装配依赖；`Handler()` 拆成私有 route registration 方法；删除 repository/service/infrastructure adapter 构造。
- `internal/bootstrap/provider.go`
  - 将 HTTP server 所需 repository、application service、infra adapter、authenticator、Turnstile verifier 的组装集中到 bootstrap。
- `internal/infrastructure/turnstile/verifier.go`
  - 从 `internal/api/http/turnstile.go` 迁出 Cloudflare Turnstile concrete HTTP client。
- `internal/infrastructure/turnstile/verifier_test.go`
  - Turnstile verifier 测试随 concrete implementation 迁到 infrastructure 包。
- `internal/api/http/server_test.go`
  - 调整为只覆盖 HTTP server/static/runtime config/fallback 等 adapter 责任，并适配 `ServerDependencies` 构造。
- `internal/test/e2e/mysql_e2e_test.go`
  - 改用 `bootstrap.NewHTTPServer` 装配 HTTP server。
- 删除 root HTTP route 集成测试：
  - `internal/api/http/application_routes_test.go`
  - `internal/api/http/artifact_routes_test.go`
  - `internal/api/http/auth_routes_test.go`
  - `internal/api/http/build_stage_routes_test.go`
  - `internal/api/http/ci_api_compatibility_test.go`
  - `internal/api/http/credential_routes_test.go`
  - `internal/api/http/deployment_routes_test.go`
  - `internal/api/http/pipeline_run_routes_test.go`
  - `internal/api/http/project_routes_test.go`
  - `internal/api/http/repository_routes_test.go`
  - `internal/api/http/route_routes_test.go`
  - `internal/api/http/settings_routes_test.go`
  - `internal/api/http/snapshot_routes_test.go`
  - `internal/api/http/template_routes_test.go`
  - `internal/api/http/traefik_route_test.go`
  - `internal/api/http/user_role_routes_test.go`
- 新增 application service integration tests：
  - `internal/application/auth/service_integration_test.go`
  - `internal/application/cd/service_integration_test.go`
  - `internal/application/ci/service_integration_test.go`
  - `internal/application/project/service_integration_test.go`
  - `internal/application/role/service_integration_test.go`
  - `internal/application/settings/service_integration_test.go`
  - `internal/application/user/service_integration_test.go`

## Expected vs actual changed files

预期改动：

- 修改 `internal/api/http/server.go`，让 HTTP server 只接收已装配依赖并注册 Gin routes/middleware/static fallback。
- 修改 `internal/bootstrap/provider.go`，集中承担 HTTP server 依赖组装。
- 迁移 Turnstile concrete verifier 到 infrastructure。
- 调整受构造签名影响的 HTTP server/e2e 测试。
- 清理 root HTTP route orphan tests，并把需要保留的业务覆盖放到对应 application package。
- 更新 requirement 和 verification 文档。

实际改动与预期一致。

`git diff --name-status` 将 Turnstile concrete implementation 识别为 rename：

```text
R072 internal/api/http/turnstile.go internal/infrastructure/turnstile/verifier.go
R066 internal/api/http/turnstile_test.go internal/infrastructure/turnstile/verifier_test.go
```

未发现 issue 9 范围外的产品代码重构；未触碰前端、数据库迁移、API route path、request/response contract、worker lifecycle 或 bootstrap app lifecycle。

## Acceptance checklist

- [x] `internal/api/http/server.go` 不再 import sqlc repository implementation packages。
- [x] `internal/api/http/server.go` 不再构造 application services、repository implementation 或 infrastructure adapters。
- [x] HTTP server 只负责 Gin engine、middleware、route registration、static fallback 和 HTTP adapter 依赖持有。
- [x] `bootstrap/provider.go` 负责选择并装配 HTTP server 所需 repository/service/infrastructure dependencies。
- [x] `Handler()` 不再重复构造 CI/CD services。
- [x] route registration 已拆为清晰私有函数。
- [x] Cloudflare Turnstile concrete HTTP client 不再作为 concrete implementation 留在 `api/http`；auth handler 仍通过接口消费。
- [x] 不保留旧 `transporthttp.New` 签名、兼容 wrapper、别名或新旧构造路径并存。
- [x] API 路由、middleware 顺序、static SPA fallback、runtime config 注入行为保持不变。
- [x] root HTTP orphan route tests 已清理；保留的业务覆盖已迁到对应 application service tests。
- [x] 后端检查通过。

## Test results

已运行命令：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint run ./cmd/... ./internal/...`：通过，输出 `0 issues.`。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/api/http
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/auth
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/ci
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/project
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/role
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/settings
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/user
ok   gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap
ok   gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/turnstile
ok   gitee.com/leoninew/PomeloOrbit-go/internal/test/e2e
```

## Missed or expanded scope

未发现范围偏离。

本次没有处理以下事项，符合 issue 9 边界：

- 未修改 API route path 或 request/response contract。
- 未修改 handler 内部业务映射逻辑。
- 未修改 application service 行为。
- 未修改 repository SQL/mapper。
- 未修改 worker lifecycle 或 bootstrap app lifecycle。
- 未拆分 `internal/bootstrap/app.go` / `internal/bootstrap/provider.go` 的生命周期边界；该事项属于 issue 10。
- 未修改前端或数据库迁移。

## Risks

当前未发现遗留实现风险。

注意：本次验证观察到 issue 9 代码变更和 requirement 文档变更已处于 index 侧；追加 verification 文档后，verification 文档本身为新的工作区变更。本次验证未执行 `git add`、`git commit` 或 `git push`。

## Incomplete items

无。

## Conclusion

Issue 9 HTTP server 路由与依赖组装边界拆分已完成并通过验证。实现符合 requirement 中的分层目标：`api/http` 不再选择 repository implementation、不再构造 application services 或 infrastructure adapters，HTTP server 只负责 Gin adapter 相关职责；依赖装配集中到 bootstrap；Turnstile concrete verifier 已迁入 infrastructure；孤儿 HTTP route tests 已清理且需要保留的业务覆盖已放回对应 application 包；未保留兼容层、别名、wrapper 或新旧构造路径，后端检查全部通过。

---

# Issue 10：bootstrap 生命周期与 provider 边界验证

## Verification target

本次追加验证范围为 light / 轻量模式下的 issue 10：`internal/bootstrap/app.go` 与 `internal/bootstrap/provider.go` 的 bootstrap 生命周期与 provider 边界拆分。

对应 requirement 文档：

- `docs/requirement/20260708-internal-layering-followup-split.md`

本节只验证 issue 10，不重新验证 issue 5/6/7/9。

## Requirement alignment

需求要求：

- `bootstrap` 仍作为唯一组合根，允许看见 repository impl、infrastructure、api/http、worker、workflow activity，但不承载业务规则。
- `cmd/server` 与 `cmd/migrate` 保持薄入口，不直接 new repository、handler、worker handler 或 infrastructure client。
- 在 `internal/bootstrap` 内部按职责拆分 database/migration、repository store、HTTP server factory、worker factory、runtime lifecycle。
- 删除 provider 大杂烩；不保留空 provider、wrapper-only 文件、兼容构造、别名或新旧装配路径并存。
- `App.Serve()`、`App.RunWorker()`、`App.Migrate()`、`App.MigrationVersion()` 行为保持不变。
- 测试用例跟随源码职责，bootstrap 只测试 composition/lifecycle，不恢复 root HTTP orphan route tests。
- 不修改 API contract、application service 行为、repository SQL/mapper、worker handler / workflow activity 业务逻辑、task retry/lease/concurrency、config schema、前端或数据库迁移。

实际实现与 requirement 对齐：

- `internal/bootstrap/database.go` 承载 `OpenDatabase`、`RunMigrations`、`MigrationVersion`。
- `internal/bootstrap/repository.go` 承载 `NewRepositoryStore`、`NewTaskRepository`。
- `internal/bootstrap/http.go` 承载 `NewHTTPServer` 和私有 `newHTTPServerDependencies`。
- `internal/bootstrap/worker.go` 承载 `NewTaskRouter`、`NewWorker`。
- `internal/bootstrap/runtime.go` 承载私有 `runHTTPServerAndWorker`，只处理 context、HTTP shutdown、error propagation，不知道业务 service。
- `internal/bootstrap/app.go` 保留 `App` 对 `cmd` 的生命周期入口，`Serve()` 继续打开数据库、执行迁移、构造 HTTP server 与 worker，但并发运行和 shutdown 协调下沉到 runtime helper。
- `internal/bootstrap/provider.go` 已删除；未留下空占位或 wrapper-only 文件。
- `internal/bootstrap/runtime_test.go` 新增 bootstrap lifecycle 测试；没有恢复 root HTTP orphan route tests。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 issue 10 拆分策略实施。

## Actual diff summary

当前 issue 10 diff 覆盖以下文件：

- `docs/requirement/20260708-internal-layering-followup-split.md`
  - 记录 issue 10 架构依据、拆分策略、实施结果、验收标准和验证结果。
- `internal/bootstrap/app.go`
  - 移除 HTTP server + background worker 并发生命周期协调细节；改为调用 `runHTTPServerAndWorker`。
- `internal/bootstrap/database.go`
  - 新增 database/migration helper。
- `internal/bootstrap/repository.go`
  - 新增 repository store/task repository factory。
- `internal/bootstrap/http.go`
  - 从原 provider 拆出 HTTP server factory 和 HTTP server dependency assembly。
- `internal/bootstrap/worker.go`
  - 从原 provider 拆出 task router / worker factory 和 workflow activity execution service 装配。
- `internal/bootstrap/runtime.go`
  - 新增 HTTP server 与 background worker 并发生命周期协调 helper。
- `internal/bootstrap/runtime_test.go`
  - 新增 bootstrap lifecycle 测试。
- `internal/bootstrap/provider.go`
  - 删除 provider 大杂烩文件；当前 diff 中被识别为 `provider.go -> http.go` rename 加新增文件。

## Expected vs actual changed files

预期改动：

- 在 `internal/bootstrap` 内拆分 database/migration、repository store、HTTP server factory、worker factory、runtime lifecycle。
- 删除或收敛 provider 大杂烩文件。
- 更新 `App.Serve()` 以使用 runtime lifecycle helper。
- 新增 bootstrap package 内 lifecycle 测试。
- 更新 requirement 和 verification 文档。

实际改动与预期一致。

`git diff --name-status` 将 provider 到 HTTP factory 拆分识别为 rename：

```text
R057 internal/bootstrap/provider.go internal/bootstrap/http.go
```

同时新增 `database.go`、`repository.go`、`runtime.go`、`runtime_test.go`、`worker.go`。这符合“拆分 provider 大杂烩并删除空占位”的预期；没有保留单独的 `provider.go` 空文件或兼容装配入口。

未发现 issue 10 范围外的产品代码重构；未触碰前端、数据库迁移、API route path、request/response contract、application service、repository SQL/mapper、worker handler / workflow activity 业务逻辑、task retry/lease/concurrency 或 config schema。

## Acceptance checklist

- [x] `bootstrap` 仍是唯一组合根；repository/service/infrastructure 装配未回流到 `api/http`、`application` 或 `cmd`。
- [x] `cmd/server` 与 `cmd/migrate` 保持薄入口，不直接 new repository、handler、worker handler 或 infrastructure client。
- [x] database/migration helper 与 repository store factory、HTTP server factory、worker factory、runtime lifecycle 职责在 bootstrap 内部分文件或私有函数中清晰分离。
- [x] 不保留空 provider、wrapper-only 文件、兼容构造、别名或新旧装配路径并存。
- [x] `App.Serve()` 行为保持不变：启动 HTTP server 与 background worker；任一失败时取消另一侧；context 取消时 shutdown HTTP；正常退出返回原有语义。
- [x] `App.RunWorker()` 行为保持不变：打开数据库、执行迁移、构造 task router/worker 并运行。
- [x] `App.Migrate()` / `App.MigrationVersion()` 行为保持不变。
- [x] 测试用例跟随源码职责：bootstrap 只测试 composition/lifecycle，application/api/infrastructure 继续测试各自业务或 adapter；未恢复 root HTTP orphan route tests。
- [x] 后端检查通过。

## Test results

已运行命令：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint fmt ./cmd/... ./internal/...`：通过。
- `./bin/golangci-lint run ./cmd/... ./internal/...`：通过，输出 `0 issues.`。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap
ok   gitee.com/leoninew/PomeloOrbit-go/internal/api/http
ok   gitee.com/leoninew/PomeloOrbit-go/internal/test/e2e
ok   gitee.com/leoninew/PomeloOrbit-go/internal/worker
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci
```

## Missed or expanded scope

未发现范围偏离。

本次没有处理以下事项，符合 issue 10 边界：

- 未修改 API route path 或 request/response contract。
- 未修改 application service 行为。
- 未修改 repository SQL/mapper。
- 未修改 worker handler 或 workflow activity 执行业务逻辑。
- 未修改 task retry、lease、concurrency 语义。
- 未修改 config schema。
- 未修改前端。
- 未修改数据库迁移。

## Risks

当前未发现遗留实现风险。

注意：本次验证观察到 issue 10 代码变更和 requirement 文档变更已处于 index 侧；追加 verification 文档后，verification 文档本身为新的工作区变更。本次验证未执行 `git add`、`git commit` 或 `git push`。

## Incomplete items

无。

## Conclusion

Issue 10 bootstrap 生命周期与 provider 边界拆分已完成并通过验证。实现符合 requirement 中的分层目标：bootstrap 继续作为唯一组合根，`cmd` 保持薄入口，database/migration、repository store、HTTP server factory、worker factory、runtime lifecycle 已在 bootstrap 内部按职责分离，provider 大杂烩文件已删除，未保留兼容层或新旧装配路径；新增测试留在 bootstrap 并只覆盖 lifecycle，后端检查全部通过。
