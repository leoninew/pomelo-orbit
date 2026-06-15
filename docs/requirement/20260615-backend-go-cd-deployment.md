# backend-go CD Deployment API 补齐需求

Review status: Accepted

## Background

`backend-go` 是 Python `backend` 的 Go 版本实现和改良，目前 `/api/cd/deployment` 相关接口仅完成列表路由，详情、日志、流式日志、取消等端点仍未补齐。前端已有对应 API 调用，需要 Go 后端对齐 Python 后端行为。

## Goal

补齐 `backend-go` 中 `/api/cd/deployment` 相关接口：

- `GET /api/cd/deployment`：补齐鉴权、项目校验与过滤参数。
- `GET /api/cd/deployment/{deployment_id}`：获取部署详情。
- `GET /api/cd/deployment/{deployment_id}/logs`：按 offset 增量读取部署日志。
- `GET /api/cd/deployment/{deployment_id}/stream-log`：SSE 流式读取部署日志。
- `POST /api/cd/deployment/{deployment_id}/cancel`：取消 waiting/running 部署。
- 更新 `docs/todo.md` 中 backend-go API 迁移状态。

## Non-goal

- 不迁移 `/api/cd/route` 与 `/api/cd/traefik-route`。
- 不重构 CD worker、Docker 执行流程或前端页面。
- 不修改已执行迁移文件。
- 不新增兼容层或旧接口别名。

## User scenarios

- 用户在部署列表按项目、应用、状态、名称和时间范围查询部署记录。
- 用户进入部署详情页查看部署记录和日志。
- 用户在部署执行中通过轮询或 SSE 查看新增日志。
- 用户取消尚未执行或执行中的部署记录。

## Acceptance

- 上述 `/api/cd/deployment` 端点在 `backend-go` 注册并返回与 Python 后端兼容的 JSON 结构。
- 列表与详情接口要求登录，并按 `project_id` / 部署所属项目做成员校验。
- 日志接口支持 `offset >= 0`，返回 `logs`、`offset`、`is_complete`、`status`。
- 取消接口仅允许 `waiting_to_run`、`running` 状态，并将状态更新为 `canceled`。
- 增加或更新相关 Go 测试。
- `docs/todo.md` 中 Deployment 小节标记为已迁移。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 轻量模式下直接实现，不额外创建 plan/spec 文档。
- 以 Python `backend/src/pomelo_orbit/interfaces/api/cd/deployment.py` 为接口清单。
- 日志读取沿用 Go worker 已生成的文件位置：`data/cd/<application_code>/deployments/<deployment_id>.log`。

## Risk

- 取消接口仅更新部署记录状态，不直接中断已被 worker 拉起的外部 Docker 命令；这与当前 Go worker 架构相关，后续如需强制终止需单独设计任务取消机制。
