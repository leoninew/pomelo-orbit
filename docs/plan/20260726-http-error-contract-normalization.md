# HTTP 错误契约规范化实施计划
最后修改时间: 2026-07-26 17:14:55

Review status: Accepted

Flow mode: standard

## 需求依据

- [HTTP 错误契约规范化需求](../requirement/20260726-http-error-contract-normalization.md)（`Accepted`）
- `D:\SourceCodes\mywork\best-practices\docs\guides\http-error-contract.manual.md`
- `D:\SourceCodes\mywork\best-practices\docs\guides\http-error-contract.md`

## 实施概览

将失败 HTTP 响应收敛为唯一的 JSON writer。该 writer 从结构化应用错误中解析 HTTP status、稳定 `code` 和安全 `error`，并从 request context 写入同一个 `requestId`。业务 cause 仅进入服务端日志；未知错误一律作为 `500/internal_error` 输出。

错误码与默认状态映射如下。显式业务 code 只能覆盖对应业务语义，不能绕过 status 或把内部 cause 放入响应。

| Kind / 场景 | HTTP status | 默认 `code` |
| --- | ---: | --- |
| validation | 400 | `validation_failed` |
| unauthorized | 401 | `unauthorized` |
| forbidden | 403 | `forbidden` |
| not found | 404 | `not_found` |
| conflict | 409 | `conflict` |
| method mismatch | 405 | `method_not_allowed` |
| dependency unavailable | 503 | `service_unavailable` |
| internal 或未分类错误 | 500 | `internal_error` |

所有错误 body 固定为 `code`、`error`、`requestId` 三个字段；本期不输出 `detail`、`details` 或重复的 `status` 字段。默认文案必须是对外安全的通用文案，只有经过确认的业务提示才能作为安全 `error` 覆盖默认值。

## 实施步骤

1. 建立应用错误与 request ID 基础设施。
   - 扩展 `apperror`，使错误对象分别保存 Kind、可公开 message、可选稳定 code 与内部 cause；提供集中分类函数，未知错误回退到 `internal_error`。
   - 保留 `Error()`/`Unwrap()` 的日志可观测性，但禁止 HTTP writer 通过 `err.Error()` 形成 body。
   - 新建独立 request-id 包，集中保存 context key、`X-Request-ID` header 名称、ULID 生成和读取逻辑，避免 `response` 与 `middleware` 产生 import cycle。
   - 将 request-id middleware 迁移到该包：非空入站 header 原样透传；否则生成 ULID。移除项目中唯一的 `github.com/google/uuid` 使用，加入 ULID 依赖并更新 lock sum；若 SQLite 的间接依赖仍需要 UUID，则保留其模块图条目。

2. 实现唯一 HTTP 错误 writer，并把完整 cause 关联到日志。
   - 在 `internal/api/http/response` 定义非泛型错误 DTO 与 `WriteError`/受控的已知错误写入入口；始终从 request context 填充 `requestId`，并使用 `application/json` 返回统一 body。
   - 由 writer 对 `apperror` 进行映射；直接 binding、必填参数和静态业务提示先构造成结构化安全错误，再交给同一个 writer，杜绝 handler 自行映射 status/code/body。
   - writer 将原始错误作为 Gin context error 保存，`LogRequest` 在请求完成时为包含 cause 的失败记录带 `request_id` 的错误日志。500 的日志保留 cause，HTTP body 只保留默认安全文案。
   - `Recovery` 使用 logger 记录 panic 与 request ID，且在尚未写出响应时调用同一 writer 返回 `500/internal_error`；已写出响应时不重复写 body。

3. 统一路由层的非业务异常出口。
   - 在 Gin router 启用 `HandleMethodNotAllowed`，为已注册 API route 的 method mismatch 注册统一 `405/method_not_allowed` 响应。
   - 将 API `NoRoute` 改为 `404/not_found`；维持非 API 的静态资源与 SPA fallback 现有行为，无法提供静态资源时也使用受控的 JSON 错误输出。
   - 保留 middleware 顺序，确保 request ID 已设置后才处理 recovery、fallback 与日志。

4. 迁移所有 Go HTTP 调用点。
   - 删除各 handler package 重复的 `writeError` helper，将 `internal/api/http/handler/**` 中的 `transportresponse.Error`、`err.Error()` 响应和手工 status 分支全部替换为统一 writer。
   - 逐一标注即时错误的安全分类：JSON/binding/必填字段为 validation，已知授权错误为 unauthorized/forbidden，资源缺失为 not found，竞争冲突为 conflict，依赖不可用为 service unavailable；其余原始错误作为 internal cause。
   - 同步迁移 `internal/api/http/security/authz.go` 及 server fallback 等 handler 之外的 HTTP 错误出口，确认全仓不再保留旧 `{ "detail": ... }` 写入路径。

5. 迁移 Web Axios client。
   - 在 `web/src/utils/request.ts` 定义新错误 body 的运行时 guard 与 `ApiError` 扩展字段，保留 `status`、`code`、`requestId` 和安全 message。
   - 响应拦截器只接受完整新契约；HTML、空响应、旧 `detail` 或字段类型不合法的错误响应返回明确的 contract-mismatch `ApiError`，不向页面传递原始 body。
   - 维持 401 的现有 `handleUnauthorized()` 边界，同时在契约有效时保留服务端 `code` 与 `requestId`；网络失败继续使用本地安全摘要。

6. 迁移 MCP Orbit client，且不触碰 environment 移除工作。
   - 将 `OrbitAPIError` 改为保存 status、可选 `code`、可选 `request_id` 和安全摘要；`__str__` 仅渲染这些受控字段。
   - 用严格的新错误 body 解析替换 `_error_detail`。认证端点仍不读取错误 body；其他端点仅在 `code`、`error`、`requestId` 类型完整时消费，契约不匹配时使用通用安全摘要。
   - 保持既有 401 重新认证与网络错误行为，不修改 MCP tools、环境变量、工作区或部署能力的移除改动。

7. 补齐测试与迁移检查。
   - 为应用错误映射、统一 writer、request ID middleware、fallback、NoMethod 和 recovery 添加覆盖；逐项验证 400、401、403、404、405、409、500、503、request ID 透传、ULID 生成及 500 不泄露 cause。
   - 为 Web interceptor 添加 Vitest，覆盖有效契约、401 既有边界、网络错误与 contract mismatch。
   - 为 MCP client 添加 pytest，覆盖新 body 的 status/code/request ID 保留、401 重试、认证响应脱敏和旧 `detail`/非 JSON 的安全降级。

## 预期改动文件

| 范围 | 文件 |
| --- | --- |
| 应用错误 | `internal/common/errors/error.go`、新增 `internal/common/errors/error_test.go` |
| request ID 与 middleware | 新增 `internal/api/http/requestid/`、`internal/api/http/middleware/logging.go`、`internal/api/http/middleware/logging_test.go`、`go.mod`、`go.sum` |
| HTTP 输出与路由 | `internal/api/http/response/response.go`、`internal/api/http/response/response_test.go`、`internal/api/http/routes/routes.go`、`internal/api/http/server.go`、`internal/api/http/server_test.go` |
| HTTP handler/security 迁移 | `internal/api/http/handler/**/*.go`、`internal/api/http/security/authz.go`、`internal/api/http/security/authz_test.go`；只修改存在错误输出的文件 |
| Web client | `web/src/utils/request.ts`、新增 `web/src/utils/request.test.ts` |
| MCP client | `mcp/src/pomelo_orbit_mcp/orbit_client.py`、`mcp/tests/test_orbit_client.py` |
| 过程文档 | 本计划及其对应 requirement；不修改其他 feature 的过程文档 |

## 验证计划

1. Go 定向测试：`go test ./internal/common/errors ./internal/api/http/response ./internal/api/http/middleware ./internal/api/http/security ./internal/api/http`。
2. Go 全量检查：`go test ./cmd/... ./internal/... ./sql`，随后执行 `go vet ./cmd/... ./internal/... ./sql` 与项目版本固定的 `golangci-lint run ./cmd/... ./internal/... ./sql`。
3. Web：在 `web` 目录执行 `yarn test`、`yarn typecheck`、`yarn lint` 和 `yarn format`。
4. MCP：在 `mcp` 已安装开发依赖的环境执行 `uv run pytest`、`uv run ruff check src tests` 和 `uv run mypy`。
5. 迁移审计：检索 `google/uuid`、`uuid.NewString`、`json:"detail"`、`response.data.detail`、`_error_detail` 和 HTTP handler 中的旧 `transportresponse.Error` 调用；预期仅保留与新实现无关的词汇或零结果，并人工复核每个命中。
6. 由于工作树已有 environment 移除改动，先运行上述定向测试；全量失败时按失败文件与本计划改动范围归因，不修改或回退无关变更来使检查通过。

## 假设与风险

- `X-Request-ID` 的 HTTP header 名大小写不敏感；实现使用项目统一常量，响应语义仍为 `X-Request-ID`。
- ULID 只用于没有入站 request ID 的请求；不会校验、重写或替换上游提供的非空 ID。
- 这是破坏性 API 契约变更。外部 API 消费者需要在部署说明中获知旧 `detail` 已移除；本仓库内的 Web 与 MCP 将同步迁移。
- `mcp/src/pomelo_orbit_mcp/orbit_client.py` 已有用户的 environment 移除改动。实施时只在错误解析及异常 DTO 的最小区块合并，先检查最新 diff，绝不覆盖其余变更。
- 在日志中记录 cause 可能包含敏感信息，因此只写到服务端日志，不写入 response 或 MCP 用户可见摘要；日志访问控制仍由现有运维策略负责。
- `github.com/google/uuid` 是 `modernc.org/sqlite` 的间接依赖。`go mod tidy` 会保留它，但项目源代码不再导入或生成 UUID；本期不替换数据库驱动以移除该传递依赖。

## 回滚

- 若调用方兼容性问题阻断发布，回滚本 feature 所改的 HTTP writer、handler 调用点、Web client 和 MCP parser 到同一提交，不回滚并行的 environment 移除变更。
- 回滚后恢复旧 `detail` 只作为紧急兼容措施；正常修复应优先恢复新契约的实现完整性而不是长期双写两种错误 body。
- 同步移除新增 ULID 依赖并恢复 UUID 依赖仅限完整回滚时执行；不要在部分迁移状态下混用两种 request ID 生成器。

## 阻塞项

暂无阻塞项。实施前必须基于最新工作树重读 MCP 和 environment 移除重叠文件，确保最小化合并。

## User review notes

- 2026-07-26：用户要求进入 Plan / 计划阶段，Requirement 自动标记为 `Accepted`。
- 2026-07-26：用户确认 request ID 使用 ULID，HTTP status 遵循最佳实践，MCP 只需满足新错误契约的使用要求。
- 2026-07-26：实现确认项目自身的 UUID 使用已清除；SQLite 的传递模块依赖保留，原因已记录在风险项。
