# Gin HTTP Transport 迁移需求
最后修改时间: 2026-07-07 11:30:07

Review status: Accepted

## Background

当前后端 HTTP transport 使用 chi 作为 router，并通过 chi middleware 提供 request id、real ip、panic recovery 等能力。业务 handler 虽然大多仍以 `net/http` 的 `http.ResponseWriter` 和 `*http.Request` 组织，但路由注册语法、路径参数读取和部分 middleware 已经与 chi 绑定。

本任务目标是将 HTTP transport 切换到 Gin，并按 Gin 的方式组织实现。用户明确要求不做适配或兼容，因此最终状态不应保留 chi 兼容层、旧 handler adapter 或 Gin/chi 混合模式。

## Goal

- 使用 Gin 作为唯一 HTTP router。
- 将 HTTP handler 改为 Gin 原生 `gin.HandlerFunc` / `*gin.Context` 组织方式。
- 将路由声明改为 Gin 原生路径参数语法，例如 `:repository_id`。
- 将路径参数读取改为 `c.Param(...)`。
- 将 query、header、body、response、authz、middleware 等 HTTP transport 横切能力迁移到 Gin context。
- 删除 chi router 和 chi middleware 依赖。
- 保持现有 API contract、静态资源服务、SPA fallback、CORS、panic recovery、access log 和 `/assets` 200 日志过滤语义不变。

## Non-goal

- 不做 Gin 包装旧 `http.HandlerFunc` 的最终适配方案。
- 不模拟 chi route context，不保留 `chi.URLParam`。
- 不保留 Gin 与 chi 并存的混合模式。
- 不借机重写 service、repository、model、proto DTO 或数据库逻辑。
- 不改变 API 路由的单数形式约束。
- 不改变 response JSON/protojson 字段命名、unknown field 处理或默认值输出语义。
- 不改动已执行迁移文件。
- 不主动启动、停止或重启开发服务器。

## User scenarios

- 作为维护者，希望 HTTP transport 统一基于 Gin，避免 router 层新旧框架并存。
- 作为开发者，希望后续新增路由直接使用 Gin 原生注册、参数读取和 middleware 组织方式。
- 作为前端/API 调用方，希望迁移后现有 API 请求、错误响应、静态资源和 SPA fallback 行为不发生可见变化。
- 作为排障者，希望 access log、request id、panic recovery、CORS 和静态资源 200 日志过滤继续可用。

## Acceptance

- `go.mod` 不再需要 `github.com/go-chi/chi/v5`。
- 生产代码不再 import `github.com/go-chi/chi/v5` 或 `github.com/go-chi/chi/v5/middleware`。
- `internal/transport/http/server.go` 使用 `gin.New()`，不得使用 `gin.Default()` 叠加默认 logger/recovery。
- HTTP route registration 使用 Gin 原生命名和方法：`GET`、`POST`、`PUT`、`DELETE`，路径参数使用 `:param`。
- HTTP handler 使用 `*gin.Context`，不以旧 `http.HandlerFunc` 作为最终业务 handler 形态。
- 认证鉴权入口使用 Gin context，未认证/未授权错误响应保持现有语义。
- response helper 保持当前 protojson 行为：`UseProtoNames: true`、`EmitUnpopulated: true`，不得直接用 Gin 默认 JSON 行为破坏 API contract。
- JSON request decode 保持当前 protojson unknown field 严格处理语义。
- CORS 行为保持：仅作用于 API path，允许 origin 的 OPTIONS 返回 204，不允许 origin 的 OPTIONS 返回 403。
- panic recovery 后仍返回 500，access log 仍能记录 completed 500。
- 静态资源服务和 SPA fallback 保持现有行为，包括 runtime config 注入。
- `/assets` 下 `.js`、`.css`、`.html` 的 200 请求在默认配置下仍不输出 `request started` 和 `request completed`。
- `/assets` 目标扩展名的非 200 请求仍输出完整 access log。
- API route tests、middleware tests、response tests 和项目后端检查通过。

## Open questions

暂无需要用户确认的未决事项。用户已明确要求不做适配或兼容，按 Gin 原生方式组织实现。

## Decisions

- 采用 light / 轻量模式推进：Requirement -> Implementation -> Verification。
- 本需求文档因用户直接要求“开始任务”并指定实现边界，初始状态记录为 `Accepted`，随后进入 Implementation / 实现阶段。
- Gin migration 的最终形态必须删除 chi 依赖和 `chi.URLParam` 调用。
- 第一轮迁移不采用 Gin 默认 JSON binding/response，以避免改变 proto DTO 的 JSON contract；仅将项目现有 decode/encode 语义迁移到 Gin context。
- 保留 service/repository 层调用方式，迁移范围限制在 HTTP transport 层。

## Risk

- 迁移范围大，涉及全部 HTTP handler、authz、middleware、response helper 和路由测试，容易出现机械替换遗漏。
- Gin 与 chi 在 trailing slash、method mismatch、OPTIONS、path normalization、panic recovery 写出状态等细节上可能存在差异，需要通过现有和新增测试兜底。
- 如果 response helper 不谨慎，可能破坏 protojson 字段名或默认值输出，造成 API contract 变化。
- 如果 logging middleware 顺序或 recovery 语义变化，可能影响最近新增的 `/assets` 200 日志过滤和 panic completed 500 日志。
- Gin 引入后依赖树会增加，需要通过 `go mod tidy` 和全量后端检查确认依赖状态。
