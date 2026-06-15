# backend-go CD Deployment API 补齐验证

Review status: Accepted

## What changed

- 补齐 `backend-go` 的 `/api/cd/deployment` 端点注册：详情、日志、SSE 日志、取消。
- `GET /api/cd/deployment` 增加登录校验、`project_id` 必填校验、项目成员校验，以及 `application_id`、`status`、`search`、`date_from`、`date_to` 过滤。
- 增加部署详情读取与按部署所属项目/应用项目的访问校验。
- 增加部署日志按 offset 增量读取，日志路径使用 `data/cd/<application_code>/deployments/<deployment_id>.log`。
- 增加部署日志 SSE 输出，完成态输出 `event: complete`。
- 增加取消部署记录能力，仅允许 `waiting_to_run` / `running`，取消后写入 `canceled`、`finished_at`、`duration_ms`、`Cancelled by user`。
- 增加 `deployment_routes_test.go` 覆盖列表、详情、日志、SSE、取消、未登录访问。
- 更新 `docs/todo.md` 中 Deployment API 迁移状态。

## Acceptance

- [x] `GET /api/cd/deployment` 支持项目鉴权和过滤参数。
- [x] `GET /api/cd/deployment/{deployment_id}` 返回部署详情。
- [x] `GET /api/cd/deployment/{deployment_id}/logs` 返回 `logs`、`offset`、`is_complete`、`status`。
- [x] `GET /api/cd/deployment/{deployment_id}/stream-log` 支持 SSE 日志与完成事件。
- [x] `POST /api/cd/deployment/{deployment_id}/cancel` 支持取消 waiting/running 部署记录。
- [x] `docs/todo.md` 已更新 Deployment 小节。

## Commands

```powershell
go -C backend-go fmt ./...
go -C backend-go test ./internal/httpserver -run Deployment -count=1 -v
go -C backend-go test ./...
```

结果：全部通过。

说明：曾直接在仓库根目录运行 `go fmt ./...`，因根目录不是 Go module 失败；随后改用 `go -C backend-go fmt ./...` 成功。

## Remaining risk

- 取消接口当前只更新部署记录状态，不强制中断已经被 worker 启动的外部 Docker 命令；如需真正停止正在运行的后台任务，需要后续单独设计 worker/task cancellation。
- 工作树中存在与本次 Deployment API 不直接相关的配置/Route 变更，验证时未将其作为本次需求范围审查。
