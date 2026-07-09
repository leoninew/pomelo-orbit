# internal 分层后续逐条拆分验证
最后修改时间: 2026-07-09 08:45:23

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
