# CD 部署详情展示执行命令与容器日志计划
最后修改时间: 2026-06-30 12:18:00

Review status: Accepted

## Requirement / Spec basis

- Requirement: `docs/requirement/20260630-cd-deployment-detail-container-logs.md`
- Spec: `docs/spec/20260630-cd-deployment-detail-container-logs.md`

本轮用户已明确要求“基于本次操作开始后的日志，开始实现”，因此视为接受 Spec 并进入 Implementation。

## Implementation steps

1. 数据库迁移
   - 新增 SQLite / MySQL migration，为 `deployment` 表添加 `command_text` 字段。
   - 不修改已执行 migration 文件。
   - 更新 migration 数量相关测试期望。

2. 后端模型与 repository
   - `model.Deployment` 增加 `CommandText`。
   - `CreateDeployment` 写入 `command_text`。
   - `ListDeployments`、`Deployment` 查询返回 `command_text`。
   - HTTP `DeploymentResp` 增加 `command_text`。

3. 命令构造同源化
   - 抽取 compose command 构造逻辑：deploy / restart / stop。
   - 创建 deployment 时持久化完整命令字符串。
   - 执行层使用同一个构造结果调用 runner，避免展示命令与实际命令漂移。

4. 容器日志服务与接口
   - 新增 deployment 维度 container logs service 方法。
   - 基于 deployment `started_at` 调用 `docker compose logs --since <started_at>`。
   - `--since` 失败时回退 `--tail N`，响应标记 `source: tail`。
   - stop deployment 返回 validation 或前端不调用；实现以 service 防御性拒绝 stop 为准。
   - 新增 HTTP route：`GET /api/cd/deployment/{deployment_id}/container-logs?tail=200`。

5. 前端 API / type / 页面
   - `Deployment` / `DeploymentDetail` 增加 `command_text`。
   - `deploymentApi` 增加 `getContainerLogs`。
   - `DeploymentDetail.vue` 基本信息展示执行命令。
   - 日志卡片标题改为“容器日志”。
   - deploy / restart 轮询 container logs；stop 显示“不展示实时容器日志”说明。
   - `source === 'tail'` 时展示历史日志提示。
   - 增加自动刷新 / 暂停刷新切换按钮；运行中 deployment 进入详情时默认开启自动刷新，终态 deployment 进入详情时只拉取一次日志，不默认轮询；暂停时停止轮询，恢复时立即刷新并继续轮询。终态不自动刷新仅作为初始行为，用户手动开启后不因终态状态阻止持续刷新。
   - 移除容器日志区域原独立“刷新”按钮；错误态重试直接调用容器日志拉取。

6. 测试与验证
   - 更新后端路由测试：detail 返回 `command_text`；未认证 container logs 返回 401。
   - 更新 deploy / restart / stop 路径相关测试覆盖 command text。
   - 更新 execution 测试确保 force recreate 命令一致。
   - 运行项目约束检查。

## Files to change

- `internal/migrations/sqlite/v0.1.6__deployment_command_text.sql`
- `internal/migrations/mysql/v0.1.6__deployment_command_text.sql`
- `internal/repository/model/cd.go`
- `internal/repository/cd/repository.go`
- `internal/service/cd/service.go`
- `internal/service/cd/application_extra.go`
- `internal/service/cd/deployment_execution.go`
- `internal/transport/http/handler/cd/handler.go`
- `internal/transport/http/deployment_routes_test.go`
- `internal/transport/http/application_routes_test.go`
- `internal/service/cd/deployment_execution_test.go`
- `internal/worker/handler/cd/handler_test.go`（如接口签名影响）
- `internal/db/migrator_test.go`
- `internal/app/mysql_e2e_test.go`
- `web/src/types/cd/deployment.ts`
- `web/src/api/cd/deployments.ts`
- `web/src/views/cd/DeploymentDetail.vue`

## Verification plan

按项目约束运行：

- `yarn --cwd web lint:fix`
- `yarn --cwd web typecheck`
- `go fmt ./cmd/... ./internal/...`
- `go vet ./cmd/... ./internal/...`
- `go test ./cmd/... ./internal/...`

## Blockers

暂无阻塞项。

## Assumptions

- `docker compose logs --since <RFC3339>` 在目标环境可用；失败时按 spec 回退到 `--tail`。
- MySQL migration 若不支持 `TEXT NOT NULL DEFAULT ''`，使用 `VARCHAR(2048) NOT NULL DEFAULT ''` 存储命令字符串。
- 第一版容器日志采用近实时轮询，不实现 SSE。

## Risks

- 新增 migration 会改变数据库 schema，需要确保 SQLite / MySQL 一致。
- 容器日志轮询返回完整日志，日志量大时可能增加前端和后端压力；通过 tail 默认值控制。
- 持久化命令字符串需要与 runner 参数同源，否则会形成错误审计信息。

## Rollback

- 如实现出现问题，回滚新增 migration、model/repository 字段、container logs API 和前端展示逻辑。
- 已执行 migration 的环境不能直接删除迁移文件，需要按项目迁移策略通过后续迁移回退字段或停用前端展示。

## User review notes

- 用户明确要求基于“本次操作开始后的日志”实现。
- 用户确认执行命令来源为持久化完整命令字符串。
- 用户确认停止操作不展示实时容器日志。