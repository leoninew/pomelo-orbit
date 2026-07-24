# CD 部署失败时展示完整错误原因验证
最后修改时间: 2026-07-21 10:37:35

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260721-cd-deployment-failure-error-detail.md` 核对：

- 命令失败时 `error_message` 包含真实命令输出，而不仅是 `exit status N`。
- `error_message` 对超长输出截取尾部约 4KB，完整过程仍以操作日志文件为准。
- 详情页 `faulted` 时错误信息完整换行展示，日志区自动切换为操作日志（方案 A）。
- 成功 / 运行中路径仍以容器日志为主。
- 不回填历史 `error_message`；不改 compose 行为与状态机；未扩展到 CI 错误链路。

用户已确认本地实测符合预期。

## Spec and plan alignment

不适用。轻量模式 / light 仅有 Requirement；实现按 requirement 与现有 API/runner 直接完成。

## Actual diff summary

- `internal/infrastructure/runner/cd/runner.go`
  - `ShellRunner.Run` 使用 `io.MultiWriter` 同时写操作日志与内存缓冲。
  - 失败时将缓冲尾部（最多 4KB）并入返回 error，供 `CompleteDeployment` 写入 `error_message`。
- `internal/infrastructure/runner/cd/runner_test.go`
  - 覆盖失败错误含命令输出、超长输出截断、空输出。
- `internal/application/cd/usecase/deployment_execution_test.go`
  - 确认部署失败路径把 runner 错误消息写入 complete message。
- `web/src/views/cd/DeploymentDetail.vue`
  - 错误信息改为完整多行展示。
  - `faulted` 拉取 `getLogs` 操作日志；非失败主路径仍用容器日志；stop 保持不适用说明。
  - 运行中自动刷新进入终态后按状态重新拉取对应日志源。
- `web/src/api/cd/deployments.ts`
  - `getLogs` 支持 Axios request config（含 AbortSignal）。
- `docs/requirement/20260721-cd-deployment-failure-error-detail.md`
  - 需求文档（Accepted）。

## Expected vs actual changed files

| 预期 | 实际 | 说明 |
| --- | --- | --- |
| CD ShellRunner 错误输出并入 error | `runner.go` + `runner_test.go` | 符合 |
| 部署失败路径测试 | `deployment_execution_test.go` | 符合 |
| 详情页错误完整展示 + faulted 操作日志 | `DeploymentDetail.vue` | 符合 |
| 操作日志 API 支持取消/配置 | `deployments.ts` | 符合 |
| 需求过程文档 | `docs/requirement/...` | 符合 |
| 验证文档 | `docs/verification/...` | 本文件 |

未改 migration、proto、compose 执行语义、CI runner、历史数据回填。

## Acceptance checklist

- [x] 失败 `error_message` 含命令真实输出，不只 `exit status N`（代码 + 单元测试 + 用户实测）。
- [x] 超长输出截取尾部（`maxErrorOutputBytes = 4KB`）并保留关键尾部内容。
- [x] 详情页错误信息完整展示（无单行 truncate）。
- [x] `faulted` 日志区展示操作日志（`GET /api/cd/deployment/:id/logs`）。
- [x] 非失败主路径默认容器日志策略保持。
- [x] 列表页仍展示 `error_message`；新失败记录内容更具诊断价值（后端增强，UI 列仍 truncate 以适配表格密度）。
- [x] Go runner/失败路径测试覆盖。
- [x] 前端 lint/typecheck 与 Go fmt/vet/相关 test 通过。
- [x] 用户实测确认符合预期。

## Test results

| Command | Result |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./internal/infrastructure/runner/cd/ ./internal/application/cd/usecase/ -count=1` | 通过 |
| `yarn --cwd web lint:fix` | 通过；0 error，5 条项目既有 import warning |
| `yarn --cwd web typecheck` | 通过 |
| 用户本地实测失败详情页 | 通过（用户确认符合预期） |

## Scope deviation

无范围外扩张。

- 未实现 Tab 双日志（方案 B）或仅后端增强（方案 C）；按默认方案 A 落地。
- 列表页错误列保持 `max-w-56 truncate` + title 悬停，详情页才完整展开；与表格布局约束一致，不视为验收缺口。
- 历史失败记录的 `error_message` 未回填；打开旧失败详情时依赖操作日志展示完整原因，符合 Non-goal。

## Risks and incomplete items

- 需重启后端进程后，新失败记录才会写入增强后的 `error_message`。
- 列表页对超长 `error_message` 仍截断展示，完整内容看详情页。
- 未运行全量 `go test ./cmd/... ./internal/...` 与前端 vitest 全量套件；本次相关包测试与静态检查已覆盖改动面。

## Conclusion

实现与已接受 requirement 对齐，自动化检查与用户实测均通过。轻量模式交付完成，可提交。
