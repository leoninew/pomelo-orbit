# HTTP Transport 边界收敛实施计划
最后修改时间: 2026-09-16 10:34:38

Review status: Accepted

Mode: standard

## Intent basis

依据已接受的 `docs/intent/20260915-http-transport-boundaries.md`：将 HTTP 入站适配层的共享能力收敛至单一 `internal/api/http/transport` 包，删除无调用方的泛化和一次性转发，不改变任何 HTTP 可观察行为或 application 边界。

现状审计确认：所有 HTTP request body 均为 protobuf message；`codec` 是 proto JSON options 的唯一 owner，SSE 与 task payload 直接调用其 marshal helper；NoRoute、NoMethod、认证授权、CORS 拒绝与 recovery 均走统一错误 writer。`internal/infrastructure/database/tx` 的请求事务 middleware 当前直接依赖 `response`，需要在迁移时解除这一反向 HTTP 依赖。

## Implementation steps

### 1. 建立单一 transport 包及明确职责

- 新建 `internal/api/http/transport/`，以源文件而非子包划分职责：
  - `request.go`：protobuf request body 解码和保留的 query 转换；`DecodeJSON` 改为仅接收 `proto.Message`，删除无调用方的普通 `encoding/json` fallback 与 `DecodeJSONReader`。
  - `protojson.go`：唯一的 protobuf JSON marshal/unmarshal options owner、`MarshalProtoJSON` / `UnmarshalProtoJSON` 和 Gin renderer。options 不再作为可被其他包修改的导出变量。
  - `output.go`：输出 protobuf HTTP 成功响应。成功输出函数使用不与 renderer 类型冲突的名称，例如 `WriteProtoJSON`。
  - `error.go`：错误 body、`WriteError`、`WriteStatusError` 及其私有 status/kind 映射。
  - `mapper.go`：分页、slice pointer、`structpb.Value`、optional scalar 和 RFC3339 时间转换。
- 保留当前 proto JSON 的 `UseProtoNames`、`EmitUnpopulated` 和未知字段拒绝设置；保留 `WriteError` 的 status、code、401 challenge、request ID 与 5xx 脱敏逻辑。
- `QueryProjectId` 是迁移前已有的空字符串到 `nil` query 语义，保留以满足本任务的 HTTP 行为不变约束；将 Project 从可选过滤收敛为显式租户作用域的设计和实现，移交独立的 `20260916-project-tenant-scope` 任务。`QueryInt` 和仍有多处调用的 mapper 辅助函数保留在职责对应文件。

### 2. 迁移 HTTP adapter 调用方并删除旧包

- 将 `internal/api/http/handler/**` 中的 binding、codec 和 response imports 改为单一 transport import；将成功响应调用切换至新的输出函数，错误输出和 mapper 使用对应 transport API。
- 将 dialogue SSE 与 task payload 的 protobuf marshal 调用迁入 transport，确保它们和常规 HTTP 响应复用同一个 options owner。
- 迁移 `server.go`、`routes/routes.go`、`middleware/cors.go`、`middleware/logging.go`、`security/authz.go` 及相关测试的错误/成功输出 import。
- 所有调用迁移完成后删除 `internal/api/http/binding`、`codec`、`response`；不保留旧 import path、deprecated alias 或 forwarding wrapper。

### 3. 解除数据库请求事务对 HTTP 包的反向依赖

- 保留 `internal/infrastructure/database/tx` 对 SQL transaction、context 绑定、response buffering 与 rollback 的现有职责和行为。
- 将 `tx.Middleware` 改为接收由组合根提供的统一错误写入函数；该模块不再导入 `internal/api/http/response` 或新的 transport 包。
- 在 `internal/bootstrap/http.go` 由 bootstrap 将 `transport.WriteError` 注入 `tx.Middleware`，保持 `routes.Dependencies.MutatingUnitOfWork`、skip path、事务提交失败和 panic rollback 语义不变。
- 更新事务 middleware 测试，继续证明 begin/commit/panic 失败返回同一安全错误契约且不会泄漏被缓冲的成功 body。

### 4. 迁移和补足 transport 测试

- 将 `codec/protojson_test.go` 和 `response/response_test.go` 移入 transport，并按新职责与名称调整。
- 保留并补足以下断言：snake_case proto JSON、`EmitUnpopulated`、未知字段拒绝、`structpb.NullValue`、成功响应 content type、错误 body、401 `WWW-Authenticate`、kind-to-status 映射与 5xx cause 脱敏。
- 保持 `server_test.go` 的 404/405/`Allow` 契约、CORS 测试、logging 的 proto JSON response 测试以及事务 middleware 的回滚测试，改用新的 import path。
- 完成后用源码搜索确认不存在旧包 import、第二个 proto JSON options owner，且 application/model/repository/infrastructure 没有直接生成 proto 依赖。

## Files to change

- 新增：`internal/api/http/transport/{request.go,protojson.go,output.go,error.go,mapper.go}` 及对应 `*_test.go`。
- 删除：`internal/api/http/binding/binding.go`、`internal/api/http/codec/{protojson.go,protojson_test.go}`、`internal/api/http/response/{response.go,response_test.go}`。
- 迁移调用方：`internal/api/http/handler/**`、`internal/api/http/{server.go,server_test.go}`、`internal/api/http/routes/routes.go`、`internal/api/http/middleware/{cors.go,cors_test.go,logging.go,logging_test.go}`、`internal/api/http/security/authz.go`。
- 解除反向依赖：`internal/infrastructure/database/tx/{request.go,request_test.go}` 与 `internal/bootstrap/http.go`。
- 过程文档：本 Plan，以及实施后按阶段更新的验证文档。

## Verification plan

1. 运行 `gofmt` 覆盖新增和修改的 Go 文件，并运行 `go test ./internal/api/http/... ./internal/infrastructure/database/tx` 验证 transport、router、middleware、SSE、task payload 和请求事务的核心路径。
2. 运行 `task check` 与 `go test ./cmd/... ./internal/...`，满足仓库固定 Go 后端检查。
3. 静态审计：确认源码不再导入 `internal/api/http/binding`、`codec` 或 `response`；生产代码中 protobuf JSON options 只有 transport owner；application、model、repository 与 infrastructure 没有直接导入 `internal/gen/proto`。
4. 对照 Intent 验证可观察行为：无效 proto JSON 的 400 契约、proto JSON 字段与空值规则、SSE/task payload 编码、404、405 `Allow`、401 `WWW-Authenticate`、CORS 拒绝、panic recovery、request ID 和 5xx 脱敏。
5. 确认事务 begin/commit/panic 失败继续 rollback，commit 失败不输出已缓冲成功 body，并经注入 writer 使用同一错误契约。

## Blockers

无。

## Assumptions

- 当前所有 HTTP JSON request body 都是 protobuf message；因此删除普通 JSON fallback 不影响已支持 API。
- `QueryProjectId` 在迁移前已存在；transport 任务保留其空字符串到 `nil` 行为，避免改变 Project scope 语义。显式 Project tenant scope 由独立任务处理。
- `routes.Dependencies.MutatingUnitOfWork` 继续接收 `gin.HandlerFunc`，无需重设计 request-scoped UoW 或 application transaction runner。
- 现有 `task check` 与 `go test ./cmd/... ./internal/...` 是本次完整后端验证入口。

## Risks

- Go package 级合并会暴露当前成功响应函数与 renderer 的同名概念；必须直接改为无冲突的 transport API，不能以兼容 wrapper 规避。
- handler、middleware 和测试调用点广泛，漏迁移会造成编译失败或第二个 options owner；需先统一迁移再删除旧目录。
- request transaction middleware 的错误 writer 改为注入时，必须覆盖 begin 和 commit 失败，以免 writer 未接线或 response buffering 顺序变化。
- 本任务明确不改变 HTTP 语义；任何发现的既有日志或业务行为问题必须单独记录，不能随包收敛一并修改。

## Rollback

- 本次不涉及 schema、数据或外部协议迁移。若实现出现行为回归，回退该实现提交即可恢复原包布局和组合根接线。
- 不在运行时保留旧包兼容层；回退是唯一恢复路径。

## User review notes

- 2026-09-16：用户要求开始计划 / Plan。
- 计划采用单一 transport 包、删除旧包，并以 bootstrap 注入错误 writer 解除数据库请求事务模块对 HTTP transport 的反向依赖。
- 2026-09-16：用户要求开始实现；本 Plan 已接受。
- 2026-09-16：审计确认 `QueryProjectId` 是原 `binding` 包已有能力，不是本次迁移新增。用户将 Project 租户语义收敛拆分至 `20260916-project-tenant-scope`；本 Plan 保持该 helper 的既有 HTTP 行为，不在包收敛中修改其业务语义。
