# backend-go HTTP 请求/响应日志
最后修改时间: 2026-06-22 16:22:32

Review status: Accepted

## Background

`backend-go` 当前使用 `log/slog`、`chi` middleware 和自定义 body capture wrapper 输出 HTTP 访问日志。访问日志已经拆分为 `request started` 和 `request completed` 两条结构化日志，并通过同一个 `request_id` 关联。

从生产日志最佳实践看，默认记录 HTTP request/response body 风险较高：body 可能包含敏感信息，也可能导致日志膨胀。用户明确要求增加一个请求和响应共用的 body logging 配置，默认生产关闭，开发环境可开启，并用同一个最大字节数限制 request body 和 response body 日志大小。

本需求沿用 `20260622-backend-go-http-logging.md` 文档，继续维护同一 HTTP logging 主题。

## Goal

- `backend-go` 每个 HTTP 请求进入 logging middleware 时输出一条 `request started` 结构化日志。
- `backend-go` 每个 HTTP 请求处理完成后继续输出一条 `request completed` 结构化日志。
- 两条日志通过相同 `request_id` 关联同一个请求。
- `request started` 用于表达请求开始，至少记录 method、path、uri、request id、remote addr、user agent。
- `request completed` 记录 status、bytes、duration、request id、remote addr、user agent。
- 增加请求和响应共用的 HTTP body logging 配置：
  - `logging.http_body_enabled`
  - `logging.http_body_max_bytes`
- `logging.http_body_enabled` 默认为 `false`，即默认不记录 `request_body` 和 `response_body`。
- `logging.http_body_max_bytes` 默认为 `4096`，用于限制 request body 和 response body 的日志字节数。
- body logging 开启后，仅 JSON body 会进入访问日志：`request_body` 记录在 `request started`，`response_body` 记录在 `request completed`。请求体读取后 handler 仍能读取完整原始 body，响应体捕获后客户端仍能收到完整原始 response body。
- 访问日志继续使用 `slog` 和 INFO 级别，不因 4xx/5xx 改变级别。
- 保留现有文件日志、stdout 输出和日志滚动行为。

## Non-goal

- 不更换日志框架，不引入 zap、zerolog、logrus。
- 不改变现有 HTTP API 行为和响应格式。
- 不修改 Python `backend/` 的日志实现。
- 不新增远程日志上报、日志查询系统或外部日志服务。
- 不新增路径 prefix 控制、采样策略或字段级脱敏。
- 不把 `response_body` 移到 `request started`；响应体只能在请求完成后记录。
- 不因为本次日志配置修改而修改数据库迁移或业务 handler。

## User scenarios

- 作为开发者，我发起 API 请求后，可以先看到 `request started`，确认请求已进入 backend-go。
- 作为开发者，我可以用同一个 `request_id` 将 `request started` 和 `request completed` 对应起来。
- 作为开发者，接口耗时较长时，我可以先从开始日志看到该请求，再等待完成日志查看 status 和 duration。
- 作为开发者，当接口返回 4xx/5xx 时，我可以从完成日志的 `status` 字段筛选异常请求；日志级别仍统一为 INFO。
- 作为运维者，生产默认不记录 HTTP body，降低敏感信息泄露和日志膨胀风险。
- 作为开发者，我可以在开发环境打开 `logging.http_body_enabled`，并通过 `logging.http_body_max_bytes` 控制 request/response body 日志大小。
- 作为调用方，日志中间件不会消耗 request body，也不会破坏 response body，接口行为保持不变。

## Acceptance

- 每个 HTTP 请求输出两条访问日志：
  - handler 执行前输出一条 `request started`。
  - handler 执行后输出一条 `request completed`。
- `request started` 至少包含：
  - `method`
  - `path`
  - `uri`
  - `request_id`
  - `remote_addr`
  - `user_agent`
- `request completed` 至少包含：
  - `method`
  - `path`
  - `uri`
  - `status`
  - `bytes`
  - `duration_ms`
  - `request_id`
  - `remote_addr`
  - `user_agent`
- 同一请求的 `request started` 和 `request completed` 必须使用同一个 `request_id`。
- `request started` 必须在调用下游 handler 前输出；`request completed` 必须在下游 handler 返回后输出。
- 默认配置包含：
  - `logging.http_body_enabled: false`
  - `logging.http_body_max_bytes: 4096`
- 支持环境变量覆盖：
  - `POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED`
  - `POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES`
- `logging.http_body_max_bytes` 必须为正数，配置非法时启动前 fail fast。
- `logging.http_body_enabled=false` 时，即使 request/response 是 JSON，也不记录 `request_body` 或 `response_body`。
- `logging.http_body_enabled=true` 且 request `Content-Type` 为 JSON、body 非空时，`request started` 包含按 `logging.http_body_max_bytes` 截断后的 `request_body`。
- `request completed` 不包含 `request_body`。
- `logging.http_body_enabled=true` 且 response `Content-Type` 为 JSON、body 非空时，`request completed` 包含按 `logging.http_body_max_bytes` 截断后的 `response_body`。
- `request started` 不包含 `response_body`。
- 请求或响应 `Content-Type` 非 JSON 或 body 为空时，对应日志不包含对应 body 字段。
- request body 日志截断不能破坏 handler 读取完整原始 body。
- response body 日志截断不能破坏客户端收到完整原始 response body。
- body 截断应按字节数限制并保持日志字符串为有效 UTF-8；超长时追加 `...`。
- 所有访问日志均使用 INFO 级别，不因非 200、4xx 或 5xx 响应改变日志级别。
- panic 被现有 recovery 处理为 500 时，访问日志仍能记录 `request started` 和 status=500 的 `request completed`。
- 更新 Go 测试覆盖：开始/完成日志、配置默认值、配置覆盖、非法配置、body logging 关闭、request/response 共用 max bytes、body 截断后 request/response 行为不变。

## Open questions

暂无必须阻塞下一阶段的问题。

后续可以单独讨论但不阻塞本需求的问题：

- 是否需要对 `password`、`token`、`secret`、`key` 等字段做递归脱敏。
- 是否需要路径 prefix 控制或采样策略。
- 是否需要按环境自动启用开发环境 body logging。

## Decisions

- 访问日志保持两条：`request started` + `request completed`。
- `request started` 记录请求开始所需的基础元信息；body logging 开启时，JSON `request_body` 也记录在 `request started`。
- `request completed` 保留完成态字段；body logging 开启时，只记录 JSON `response_body`，不重复记录 `request_body`。
- `request_body` 和 `response_body` 共用同一组开关和大小配置：`logging.http_body_enabled` 和 `logging.http_body_max_bytes`。
- `logging.http_body_enabled` 默认 `false`，生产默认关闭 body logging。
- `logging.http_body_max_bytes` 默认 `4096`。
- body 截断按字节数执行，而不是按 rune 数执行；截断结果必须保持有效 UTF-8。
- request body 只读取最多 `http_body_max_bytes + 1` 字节用于日志，随后恢复 body reader，避免为了日志完整读入大请求。
- 两条访问日志均使用 INFO 级别。
- 不调整文件日志、stdout 输出、日志滚动配置和 logger 初始化生命周期。

## Risk

- 每个请求从一条访问日志变为两条访问日志，日志量会增加。
- body logging 开启后，`request_body` / `response_body` 仍可能包含敏感字段；本次只增加开关和大小限制，不做字段级脱敏。
- 如果开始日志字段和完成日志字段不一致，排查时可能难以关联；测试必须固定 `request_id` 一致性。
- middleware 顺序仍需保证 panic 被 recover 后，完成日志能记录 500 响应。
- 按字节截断 JSON body 可能得到非完整 JSON 片段；这是日志体积控制的预期结果。

## User review notes

- 2026-06-22：用户指出 backend-go 现有实现只有 `request completed`，要求修改为请求开始和结束分开记录。
- 2026-06-22：用户要求使用轻量模式 / light，并将 `20260614-backend-go-http-logging.md` 系列文档移动为 `20260622-backend-go-http-logging.md` 后同步修改。
- 2026-06-22：用户要求落实 body logging 开关，请求和响应共用配置，默认生产关闭，开发环境可开。
- 2026-06-22：用户澄清“请求和响应拆开”要求 `request_body` 属于 `request started`，`response_body` 属于 `request completed`。
