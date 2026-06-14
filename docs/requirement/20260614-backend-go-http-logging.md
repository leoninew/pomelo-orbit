# backend-go HTTP 请求/响应日志

Review status: Accepted

## Background

`backend-go` 当前已经使用 Go 标准库 `log/slog` 输出应用日志，并在 `internal/httpserver/server.go` 中有基础 HTTP request logging。但现有访问日志只记录 `method`、`path` 和 `duration_ms`，缺少响应状态码、响应体、请求体、query string、request id 等排障所需信息。当前配置已有 `logging.file: logs/backend-go.log`，但 logger 仍固定输出到 stdout，尚未实现文件日志和日志滚动。

参考项目 `D:\SourceCodes\mywork\k12-pubforge\backend-go` 已有 HTTP logging middleware 模式：读取 JSON request body 后恢复 `r.Body`，包装 `ResponseWriter` 捕获 status/body。当前项目应吸收 request/response body capture 模式，但保持现有技术栈：`slog + chi + 自定义 body capture wrapper`，且请求中间件日志级别固定使用 INFO。

## Goals

- 为 `backend-go` 引入结构化 HTTP 请求/响应访问日志。
- 基于当前项目已有 `log/slog`，继续输出结构化日志，不切换到 zap、zerolog、logrus 或参考项目的 `log.Logger` 文本格式。
- 基于当前 `chi` HTTP router/middleware 链实现，不更换 Web 框架。
- 使用自定义 body capture wrapper 捕获 JSON request body 和 JSON response body。
- JSON request body 读取后必须恢复，确保 handler 仍可正常读取请求体。
- JSON response body 捕获后必须保证客户端仍收到完整响应。
- 访问日志应保留关键请求/响应元信息，便于定位接口错误和慢请求。
- 应实现应用日志文件输出，使用现有 `logging.file` 配置指定日志文件路径。
- 应实现日志滚动，避免单个日志文件无限增长。

## Non-goals

- 不依赖、不调用、不读取 Python `backend/` 项目的运行时接口或文件。
- 不引入新的日志框架或替换现有 `slog`。
- 不改变现有 `logging.file` 配置语义；该字段用于指定应用日志文件路径。
- 不引入远程日志上报、日志查询系统或外部日志服务。
- 不记录非 JSON 请求体或非 JSON 响应体。
- 不在本需求中实现字段级脱敏、敏感字段配置或 body logging 开关。
- 不重构业务 handler 或 API 响应格式。
- 不修改已执行迁移文件。

## User scenarios

- 作为开发者，我访问 API 后，可以从 `backend-go` 日志中看到该请求的 method、uri、status、duration、request id 和响应大小。
- 作为开发者，当接口返回非 200、4xx 或 5xx 时，我可以通过 `status` 字段筛选异常请求；请求中间件日志级别仍统一为 INFO。
- 作为开发者，当请求 `Content-Type` 为 JSON 时，我可以在日志中看到完整 request body，用于排查前端传参问题。
- 作为开发者，当响应 `Content-Type` 为 JSON 时，我可以在日志中看到截断后的 response body，用于排查接口返回内容问题。
- 作为调用方，日志中间件不会消耗 request body，也不会破坏 response body，接口行为保持不变。
- 作为开发者，我可以在 `logging.file` 指定的文件中查看 backend-go 应用日志和 HTTP 访问日志。
- 作为运维者，我可以依赖日志滚动策略限制单个日志文件大小，避免日志无限增长。

## Acceptance criteria

- 每个 HTTP 请求完成后输出一条 `request completed` 结构化访问日志。
- 访问日志至少包含：
  - `method`
  - `path`
  - `uri`
  - `status`
  - `bytes`
  - `duration_ms`
  - `request_id`
  - `remote_addr`
  - `user_agent`
- 请求 `Content-Type` 为 JSON 且 body 非空时，日志包含完整 `request_body`。
- 请求 `Content-Type` 非 JSON 或 body 为空时，日志不包含 `request_body`。
- 响应 `Content-Type` 为 JSON 且 body 非空时，日志包含截断后的 `response_body`。
- 响应 `Content-Type` 非 JSON 或 body 为空时，日志不包含 `response_body`。
- 读取 JSON request body 后，业务 handler 仍能读取完整原始 body。
- 捕获 JSON response body 后，客户端仍能收到完整原始 response body。
- response body 日志必须有长度上限，避免日志过大；request body 不截断。
- 所有访问日志均使用 INFO 级别，不因非 200、4xx 或 5xx 响应改变日志级别。
- panic 被现有 recovery 处理为 500 时，访问日志仍能记录该 500 响应，且日志级别仍为 INFO。
- `logging.New(level string)` 继续可用，并能正确处理带空格的 level 字符串。
- 应使用 `logging.file` 指定应用日志文件路径，并在启动时创建缺失的父目录。
- 应同时保留 stdout 输出，避免开发环境和容器运行时失去控制台日志。
- 应实现按文件大小滚动日志，避免单个日志文件无限增长。
- 日志滚动参数应可配置，至少包含单文件大小上限和保留文件数量。
- 新增/更新的 Go 测试覆盖 JSON request body、JSON response body、body 恢复、response body 截断、request body 不截断、固定 INFO 日志级别、基础元信息字段、文件日志输出和日志滚动配置。

## Decisions

- 使用 `slog + chi + 自定义 body capture wrapper`，不更换日志框架。
- 应基于 `slog` handler 输出到 stdout 和滚动文件 writer。
- 日志滚动应使用成熟 Go 组件实现，不自行实现文件切分和清理逻辑。
- JSON body 是否记录以 HTTP `Content-Type` 判断。
- 本需求记录 request body 和 response body，但只记录 JSON body。
- `request_body` 不截断，不做字段级脱敏；`response_body` 做长度截断，不做字段级脱敏。
- 应保持访问日志为单条 `request completed`，request/response body 作为字段附加，不为同一请求拆出多条 body 日志。

## Open questions

暂无必须阻塞下一阶段的问题。

后续可以单独讨论但不阻塞本需求的问题：

- 是否需要对 `password`、`token`、`secret`、`key` 等字段做递归脱敏。
- 是否需要 body logging 配置开关或路径 prefix 控制。
- 日志滚动参数的具体默认值，例如单文件大小、保留文件数量、保留天数和是否压缩。

## Risks

- JSON request/response body 可能包含敏感字段；本需求按用户明确要求记录 body，不做字段级脱敏，且 `request_body` 不截断。
- response body capture 需要包装 `http.ResponseWriter`，必须保持 `Flush`、`Hijack`、`Push` 等可选接口兼容，避免破坏特殊响应场景。
- middleware 顺序调整需要通过测试确认 panic/recover 与访问日志记录行为符合预期。
- 文件日志会引入文件句柄生命周期管理；应用退出时应关闭日志文件 writer，避免缓冲或句柄泄漏。
- 日志滚动参数如果设置不合理，可能导致日志保留不足或磁盘占用过高，需要在配置默认值中平衡。

## User review notes

- 用户指定严格模式 / strict。
- 用户指定技术路线：`slog + chi + 自定义 body capture wrapper`。
- 用户明确要求记录请求头为 JSON 的请求 body。
- 用户明确要求记录响应。
- 用户要求更新需求，实现文件日志及滚动。
