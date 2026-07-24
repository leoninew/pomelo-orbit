# Gin HTTP Transport 迁移验证
最后修改时间: 2026-07-07 14:00:37

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260707-gin-http-transport-migration.md` 核对：

- 已使用 Gin 作为 HTTP transport router：`internal/transport/http/server.go` 使用 `gin.New()`，没有使用 `gin.Default()`。
- 生产 handler 已迁移为 Gin context 形态，业务 handler 使用 `*gin.Context`，路由注册使用 Gin 原生 `GET`、`POST`、`PUT`、`DELETE`。
- 路径参数读取已迁移为 `c.Param(...)`，路由参数使用 Gin `:param` 语法。
- 未保留 chi/Gin 混合模式；生产代码未发现 `github.com/go-chi/chi/v5`、`chi.URLParam`、`chi.NewRouter` 或 chi middleware 调用。
- `go.mod` 使用 `github.com/gin-gonic/gin v1.12.0`，未发现 `github.com/go-chi/chi/v5` 依赖。
- JSON request decode 保持 strict protojson unknown field 语义：`internal/transport/http/response/response.go` 通过 `codec.UnmarshalProtoJSON` 处理 proto message。
- JSON response 对 protobuf DTO 使用 `internal/transport/http/codec.ProtoJSON`，配置 `UseProtoNames: true`、`EmitUnpopulated: true`，避免 Gin 默认 `encoding/json` 吞掉默认值和空列表。
- 错误响应继续使用 Gin 原生 `c.JSON(..., gin.H{"detail": ...})`，保持 `detail` shape。
- CORS、recovery、request id、real ip、access log、静态资源和 SPA fallback 均迁移到 Gin middleware / Gin handler 形态。
- `/assets` 下 `.js`、`.css`、`.html` 的 200 请求日志过滤逻辑保留在 Gin logging middleware 中。

## Spec alignment

不适用。本任务按 light / 轻量模式推进，无单独 spec 文档；按 requirement / 需求核对。

## Plan alignment

不适用。本任务按 light / 轻量模式推进，无单独 plan 文档；按 requirement / 需求核对。

## Actual diff summary

本次迁移实际改动集中在 HTTP transport 层和 Go module 依赖：

- `go.mod` / `go.sum`
  - 引入 Gin 依赖及其依赖树。
  - 移除 chi 直接依赖。
- `internal/transport/http/server.go`
  - 使用 `gin.New()` 构造 router。
  - 挂载 Gin middleware：request id、real ip、logging、recovery、CORS。
  - 路由注册改为 Gin 原生方法和 `:param` 语法。
  - `NoRoute` 处理 API 404 与 SPA fallback。
- `internal/transport/http/middleware/*.go`
  - CORS 和 logging middleware 改为 `gin.HandlerFunc`。
  - logging 保留 body capture、request completed、panic 500、`/assets` 200 skip 行为。
- `internal/transport/http/handler/**`
  - auth、authz、user、role、project、settings、task、CI、CD handler 改为 `*gin.Context`。
  - 路径参数使用 `c.Param(...)`，query/header/body/context 从 Gin context 读取。
  - protobuf 成功响应使用 `c.Render(..., codec.ProtoJSON{...})`，错误响应使用 `c.JSON(..., gin.H{"detail": ...})`。
- `internal/transport/http/codec/*.go`
  - 新增 protojson codec，统一 protobuf JSON marshal/unmarshal 选项。
  - 新增 Gin `ProtoJSON` renderer，保持 proto field name 和默认值输出。
- `internal/transport/http/response/response.go`
  - 保留 request decode 与 DTO 辅助函数。
  - proto request decode 复用 codec strict unmarshal。
- `internal/transport/http/*_test.go` 和 middleware/authz tests
  - 更新 Gin router、middleware、route compatibility、JSON shape 和相关行为测试。
- `docs/requirement/20260707-gin-http-transport-migration.md`
  - 记录并接受轻量需求。

## Expected vs actual changed files

预期改动范围：HTTP transport、Gin 依赖、相关测试和 SpecFlow requirement/verification 文档。

实际改动文件符合预期，主要包括：

- `internal/transport/http/server.go`
- `internal/transport/http/middleware/cors.go`
- `internal/transport/http/middleware/logging.go`
- `internal/transport/http/handler/**`
- `internal/transport/http/codec/protojson.go`
- `internal/transport/http/response/response.go`
- `internal/transport/http/*_test.go`
- `go.mod`
- `go.sum`
- `docs/requirement/20260707-gin-http-transport-migration.md`
- `docs/verification/20260707-gin-http-transport-migration.md`

未发现 service、repository、model、migration 或前端业务代码被纳入迁移范围。

## Acceptance criteria checklist

- [x] `go.mod` 不再需要 `github.com/go-chi/chi/v5`。
- [x] 生产代码未发现 `github.com/go-chi/chi/v5`、`github.com/go-chi/chi/v5/middleware`、`chi.URLParam`、`chi.NewRouter`。
- [x] `server.go` 使用 `gin.New()`，未使用 `gin.Default()`。
- [x] route registration 使用 Gin 原生方法和 `:param` 语法。
- [x] HTTP handler 使用 `*gin.Context`，未保留旧 `http.HandlerFunc` 作为生产业务 handler 形态。
- [x] 认证鉴权入口使用 Gin context，未认证/未授权错误响应仍为 `detail` shape。
- [x] protobuf response 使用 protojson codec，保持 `UseProtoNames: true`、`EmitUnpopulated: true`。
- [x] JSON request decode 保持 `DiscardUnknown: false`。
- [x] CORS Gin middleware 测试通过。
- [x] panic recovery 和 access log 相关测试通过。
- [x] 静态资源服务和 SPA fallback 测试通过。
- [x] `/assets` 目标扩展名 200 skip 和非 200 logging 行为测试通过。
- [x] API route tests、middleware tests、response/codec tests 和后端检查通过。

## Command results

已运行：

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
git diff --check HEAD --
```

结果：

- `go fmt ./cmd/... ./internal/...`：通过。
- `go vet ./cmd/... ./internal/...`：通过。
- `go test ./cmd/... ./internal/...`：通过。
- `git diff --check HEAD --`：通过，无 whitespace error 输出。

额外检查：

```text
Grep: github.com/go-chi/chi / chi.URLParam / chi.NewRouter / chi middleware
Grep: protobuf success response still using c.JSON
```

结果：

- 未发现生产代码残留 chi import / chi URL param / chi router。
- 未发现 protobuf 成功响应继续使用 Gin 默认 `c.JSON` 输出。
- `http.HandlerFunc` 命中仅出现在测试中的 test server / middleware test handler，不属于生产业务 handler 残留。

## Missed or expanded scope

- 新增 `internal/transport/http/codec` 是为保持原 response JSON contract 所需，属于 requirement 中“不得直接用 Gin 默认 JSON 行为破坏 API contract”的范围内。
- 错误响应仍用 `gin.H{"detail": ...}`，未强行改成 protobuf error message；这是为了保留当前错误响应 shape，同时避免无意义包装。
- 未修改前端代码，不需要运行 `yarn --cwd web lint:fix` 或 `yarn --cwd web typecheck`。
- 未修改已执行 migration 文件。
- 未主动启动、停止或重启开发服务器。

## Risks

- 迁移覆盖面大，未来新增 HTTP handler 时需要继续遵守：protobuf 成功响应使用 `codec.ProtoJSON`，错误响应可用 Gin `c.JSON(gin.H{"detail": ...})`。
- Gin 与 chi 在路径清理、method mismatch、redirect/trailing slash 等边界行为上存在框架差异；当前已通过现有 route/middleware/server 测试覆盖主要契约，但若后续发现具体边界差异，应以 Gin 原生行为和 API contract 为准补测试。
- `docs/spec/20260614-cloudflare-turnstile-captcha.md` 中仍有历史文字提到 chi `middleware.RealIP`，这是旧文档背景，不是生产代码残留。

## Incomplete items

无。当前 requirement 验收项均已验证通过。

## Conclusion

Gin HTTP transport 迁移验证通过。实现符合“切换到 Gin，不做 chi/Gin 混合，不做旧 handler 适配”的目标；后端格式化、vet、全量 Go 测试和 diff whitespace 检查均通过。