# internal 分层后续逐条拆分验证
最后修改时间: 2026-07-09 09:28:24

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
