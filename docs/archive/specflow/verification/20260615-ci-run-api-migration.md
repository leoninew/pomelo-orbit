# backend-go CI Run API 迁移验证

## What changed

本次对照 `docs/requirement/20260615-ci-run-api-migration.md`，完成并验证 `/api/ci/run` 相关接口迁移。

已实现或补齐：

- `GET /api/ci/run`
  - 要求登录。
  - `project_id` 必填。
  - 支持 `repository_id`、`template_id`、`date_from`、`date_to`。
  - 默认 `per_page=20`。
  - 返回结构化 `variables_snapshot`。
- `GET /api/ci/run/{run_id}`
  - 返回 run 详情。
  - 内嵌 `stage_runs`。
  - 执行项目成员校验。
- `GET /api/ci/run/{run_id}/artifacts`
  - 统一复用 run 加载与权限校验。
- `GET /api/ci/run/{run_id}/stages/{stage_run_id}/log`
  - 支持 `offset`。
  - 返回 `logs`、`offset`、`is_complete`。
- `POST /api/ci/run/{run_id}/cancel`
  - 仅允许取消 `waiting_to_run` / `running`。
  - 返回更新后的 run 详情。
- `POST /api/ci/run/{run_id}/retry`
  - 仅允许重试 `faulted` / `ran_to_completion`。
  - 创建新 run，设置 `retry_of`，并入队后台执行任务。

相关内部变化：

- 新增 stage run store 查询和 run cancel store 方法。
- 调整 CI handler，使其能从数组形式 `variables_snapshot` 还原执行变量。
- 调整 CI handler 结束时的状态检查，避免取消后的 run 被后台执行完成覆盖状态。
- 新增 run route tests，覆盖详情、列表过滤、日志、取消、重试和鉴权。
- 更新 `docs/todo.md` 中 Backend API Migration TODO 的 CI Run 小节和统计。

## Acceptance

- [x] 逐个检查 `/api/ci/run` 相关 Python endpoint，并补齐 backend-go 缺失同 method/path 路由。
- [x] 响应字段、状态码、错误语义尽量对齐 Python backend，并保持 backend-go 项目内权限校验风格。
- [x] 不存在的 run 返回 404；stage log 对不存在/不属于该 run 的 stage 按 Python 行为返回空日志完成状态。
- [x] 需要项目权限的 run 操作执行项目成员校验。
- [x] 添加 backend-go route tests，覆盖成功路径、鉴权和关键错误路径。
- [x] 更新 `docs/todo.md` 中 CI Run 小节状态和统计。
- [x] backend-go 相关测试通过。

## Commands

已执行并通过：

```powershell
go -C "D:\SourceCodes\mywork\pomelo-orbit\backend-go" test ./internal/httpserver -run "TestPipelineRun|TestProjectAndDashboardLists|TestRepositoryTriggerRoute"
```

结果：

```text
ok  backend/internal/httpserver  (cached)
```

已执行并通过：

```powershell
go -C "D:\SourceCodes\mywork\pomelo-orbit\backend-go" test ./internal/httpserver ./internal/orbit ./internal/ci
```

结果：

```text
ok  backend/internal/httpserver  (cached)
ok  backend/internal/orbit       (cached)
ok  backend/internal/ci          (cached)
```

已执行并通过：

```powershell
go -C "D:\SourceCodes\mywork\pomelo-orbit\backend-go" test ./...
```

结果：

```text
?   backend/cmd/backend-go       [no test files]
ok  backend/internal/app         (cached)
ok  backend/internal/cd          (cached)
ok  backend/internal/ci          (cached)
ok  backend/internal/config      (cached)
ok  backend/internal/db          (cached)
ok  backend/internal/httpserver  (cached)
ok  backend/internal/logging     (cached)
?   backend/internal/migrations  [no test files]
ok  backend/internal/orbit       (cached)
ok  backend/internal/security    (cached)
?   backend/internal/status      [no test files]
ok  backend/internal/task        (cached)
ok  backend/internal/templatex   (cached)
```

## Remaining risk

- 当前工作区包含多个功能批次的未提交变更：artifact、credential、snapshot、CI run、SpecFlow 文档和 `docs/todo.md`。提交前建议按功能边界拆分审查。
- `Makefile` 仍有既有修改，不属于本次 CI Run API 迁移范围，本次未处理。
- `cancel` 已避免后台执行结束覆盖 `canceled` 状态，但当前 backend-go worker 没有 Python `cancel_task(run_id)` 那样的运行中任务主动中断机制；运行中的容器任务可能仍会执行到自然结束，只是不再覆盖最终 run 状态。
- `retry` 复用原 run 的 `variables_snapshot` 创建新 run；若后续要完全复刻 Python 的变量重新解析逻辑，可单独增强。
