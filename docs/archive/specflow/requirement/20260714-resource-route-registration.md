# HTTP 资源路由注册拆分
最后修改时间: 2026-07-14 14:37:49

Review status: Accepted

## Background

此前 `internal/api/http/routes/ci.go` 与 `cd.go` 按前端 URL 命名空间集中注册路由。虽然前端仍使用 `/api/ci/*`、`/api/cd/*`，但它们不应成为后端的模块聚合边界。

本次将 HTTP 入站适配器中的路由注册按实际资源拆分：每个资源路由文件自行创建对应 HTTP handler，并绑定本资源的 Gin endpoints；`routes.go` 仅保留 HTTP router 的全局装配和资源注册顺序。

## Goal

- 删除 `internal/api/http/routes/ci.go` 和 `cd.go`。
- 移除所有 `registerCI*`、`registerCD*` 聚合注册符号。
- 让 `repository`、`template`、`build_stage`、`pipeline_run`、`snapshot`、`artifact`、`credential`、`application`、`deployment`、`application_extra`、`route`、`traefik_route` 等资源文件各自负责 handler 构造和 endpoint 绑定。
- 保持 HTTP 路径、方法、handler 绑定及整体注册顺序不变。
- 保持分层依赖方向：`bootstrap` 装配依赖，`api/http/routes` 负责 Gin 路由和 handler，handler 调用 application service；Gin 不进入 `bootstrap` 或 `application`。

## Non-goal

- 不修改前端 URL 命名空间或 API 契约。
- 不修改 handler、application service、repository、infrastructure 或 bootstrap 的业务行为和依赖装配。
- 不引入 Gin route group、通用注册器、兼容 wrapper、别名或新旧注册逻辑并存。
- 不修改已执行迁移，不启动或停止开发服务器，不执行 Git 写操作。

## User scenarios

- 前端继续通过既有 `/api/ci/*` 和 `/api/cd/*` 调用接口，行为无差异。
- 后端维护者查看任一资源路由文件时，可以在同一处定位该资源的 handler 创建与 endpoint 声明。
- 后端维护者查看 `routes.go` 时，只看到 HTTP adapter 的全局中间件、健康检查、资源注册顺序和 fallback，不看到 CI/CD 人为聚合层。

## Acceptance

- [ ] `ci.go` 与 `cd.go` 不存在。
- [ ] 路由包中没有 `registerCI`、`registerCD`、`registerCIRoutes` 或 `registerCDRoutes` 符号。
- [ ] 每个资源 registrar 是 `Router` 方法，并只绑定该资源的 endpoints。
- [ ] 所有原有 CI/CD endpoint 的 HTTP method、path、handler method 和注册顺序保持不变。
- [ ] `internal/bootstrap` 和 `internal/application` 不包含 Gin import。
- [ ] `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...` 均通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- `/api/ci/*` 与 `/api/cd/*` 是前端 API URL 命名空间，不是后端路由注册模块边界。
- `routes.go` 是 HTTP adapter 的全局 composition spine；资源文件是 endpoint ownership 边界。
- 由于 HTTP handler 为注入 application-facing service/authenticator 的无状态协议适配器，各资源 registrar 可独立创建其 handler，不需要跨资源共享 handler 实例。

## Risk

- 路由拆分时可能遗漏、重复或重排 endpoint，尤其是静态路径与参数路径的相对顺序。
  - 缓解：保留原始资源注册顺序和各文件内直接 `engine.GET/POST/PUT/DELETE` 声明顺序；运行 Go 全量检查，并审阅最终 diff。
- 现有工作区包含本次任务之外的 `.claude/worktrees/` 未跟踪目录。
  - 缓解：不纳入本次文档和代码变更范围。
