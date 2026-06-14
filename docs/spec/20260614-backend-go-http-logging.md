# backend-go HTTP 请求/响应日志规格

Review status: Accepted

## Requirement basis

基于已接受的需求文档：

- `docs/requirement/20260614-backend-go-http-logging.md`

关键约束：

- 技术路线固定为 `slog + chi + 自定义 body capture wrapper`。
- HTTP 请求中间件日志级别统一使用 INFO，不因非 200、4xx 或 5xx 响应改变日志级别。
- JSON request body 完整记录，不截断，不做字段级脱敏。
- JSON response body 记录但截断，不做字段级脱敏。
- 非 JSON request/response body 不记录。
- 实现文件日志，并保留 stdout 输出。
- 使用成熟 Go 组件实现日志滚动，不自行实现文件切分和清理逻辑。
- 不依赖 Python `backend/`。
- 不引入新的日志框架；继续使用 `log/slog`。

## Overview

本需求包含两部分：

1. **应用日志输出增强**：`backend-go/internal/logging` 从仅 stdout 输出扩展为 stdout + rolling file 输出，日志文件路径和滚动参数来自 `logging` 配置。
2. **HTTP 请求/响应访问日志增强**：`backend-go/internal/httpserver` 的 `Server.logRequest` 使用自定义 body capture wrapper 记录 JSON request body、JSON response body 和请求/响应元信息。

最终效果：应用日志、worker 日志和 HTTP 访问日志使用同一个 `slog.Logger`，同时输出到控制台和滚动日志文件。

## Design decisions

### 1. 继续使用 slog，不更换日志框架

当前项目已有 `backend-go/internal/logging/logging.go`，使用 Go 标准库 `log/slog`。

本需求继续使用 `slog`：

- 不引入 zap、zerolog、logrus。
- `Server`、`App`、worker 等继续使用 `*slog.Logger`。
- HTTP 访问日志使用 `s.logger.Info("request completed", ...)` 输出。
- 文件日志通过 `slog.Handler` 的 writer 实现，而不是引入新的 logger API。

### 2. 使用 lumberjack 实现日志滚动

采用成熟 Go 组件：

- `gopkg.in/natefinch/lumberjack.v2`

原因：

- 专用于日志文件 rolling。
- 支持按大小滚动。
- 支持保留文件数量。
- 实现简单，不需要项目自行维护切分、重命名和清理逻辑。

不自行实现日志滚动。

### 3. logging 配置结构

扩展 `backend-go/internal/config/config.go` 中的 `LoggingConfig`。

目标结构：

```go
type LoggingConfig struct {
    Level     string `mapstructure:"level" yaml:"level"`
    File      string `mapstructure:"file" yaml:"file"`
    MaxSizeMB int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
    MaxBackups int   `mapstructure:"max_backups" yaml:"max_backups"`
}
```

配置语义：

- `level`: slog level，继续支持 `debug`、`info`、`warn` / `warning`、`error`。
- `file`: 应用日志文件路径，例如 `logs/backend-go.log`。
- `max_size_mb`: 单个日志文件大小上限，单位 MB。
- `max_backups`: 保留滚动文件数量。

默认配置写入 `backend-go/config.defaults.yaml`：

```yaml
logging:
  level: info
  file: logs/backend-go.log
  max_size_mb: 100
  max_backups: 7
```

环境变量绑定增加：

- `POMELO_ORBIT_BACKEND__LOGGING__MAX_SIZE_MB`
- `POMELO_ORBIT_BACKEND__LOGGING__MAX_BACKUPS`

### 4. logging 配置校验

在 `Config.Validate()` 中增加 logging 校验：

- `logging.file` 必须非空。
- `logging.max_size_mb` 必须大于 0。
- `logging.max_backups` 必须大于 0。

原因：

- 本需求要求启用文件日志；空文件路径代表配置不完整。
- 滚动参数无效时应 fail fast，避免运行时出现不可预期行为。

### 5. logging.New 签名和生命周期

当前：

```go
func New(level string) *slog.Logger
```

目标：

```go
func New(cfg config.LoggingConfig) (*slog.Logger, func() error, error)
```

行为：

1. 解析 `cfg.Level`，使用 `strings.TrimSpace` + `strings.ToUpper`。
2. 创建 `cfg.File` 的父目录。
3. 创建 `lumberjack.Logger`：
   - `Filename: cfg.File`
   - `MaxSize: cfg.MaxSizeMB`
   - `MaxBackups: cfg.MaxBackups`
4. 使用 `io.MultiWriter(os.Stdout, rollingWriter)` 同时输出到 stdout 和文件。
5. 使用 `slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLevel})`。
6. 返回 logger、close 函数和 error。

close 函数：

- 调用 `rollingWriter.Close()`。
- 用于应用退出时关闭文件 writer。

不保留旧签名兼容层；调用方直接更新到新签名。

### 6. main 中 logger 初始化和关闭

修改 `backend-go/cmd/backend-go/main.go`。

目标流程：

```go
logger, closeLogger, err := logging.New(cfg.Logging)
if err != nil {
    slog.Error("init logger failed", "error", err)
    os.Exit(1)
}
defer func() {
    if err := closeLogger(); err != nil {
        slog.Error("close logger failed", "error", err)
    }
}()

backgroundApp := app.New(cfg, logger)
```

说明：

- 配置加载失败仍只能使用默认 `slog.Error`，因为此时 logger 尚未初始化。
- logger 初始化失败时使用默认 `slog.Error` 并退出。
- logger 初始化成功后，`app.New` 继续接收 `*slog.Logger`。

### 7. 请求中间件日志级别固定 INFO

访问日志是请求流水记录，不作为错误日志语义使用。即使响应为 400、404、500，仍统一使用 INFO。

异常筛选通过结构化字段完成：

- `status >= 400`
- `status >= 500`
- `path`
- `uri`
- `request_id`

因此不会实现 status 到 WARN/ERROR 的映射。

### 8. JSON body 判断规则

使用 HTTP `Content-Type` 判断 body 是否可记录。

`isJSONContentType(contentType string) bool` 应支持：

- `application/json`
- `application/json; charset=utf-8`
- `application/problem+json`
- 其他以 `+json` 结尾的 media type

建议使用 `mime.ParseMediaType` 解析，解析失败时回退到小写字符串判断，避免异常 Content-Type 直接导致 body 记录失效。

### 9. request body 完整记录并恢复

`readRequestBodyForLog(r *http.Request) (string, error)`：

- 仅在 JSON request 中调用。
- `r.Body == nil` 时返回空字符串。
- 使用 `io.ReadAll(r.Body)` 读取完整 body。
- 无论 body 是否为空，只要读取成功，都恢复：
  - `r.Body = io.NopCloser(bytes.NewReader(body))`
- body 为空时不输出 `request_body` 字段。
- request body 不截断。
- request body 不脱敏。

读取失败时：

- 不阻断请求。
- 日志中附加 `body_read_error` 字段。

### 10. response body 截断记录

response body 只在响应 `Content-Type` 为 JSON 时记录。

由于 response body 可能较大，必须截断：

- 常量：`maxResponseBodyLogLength = 4096`
- wrapper 内部最多缓存截断所需范围，避免无限内存增长。
- 输出前按 rune 截断，避免切断多字节字符。
- 超长时追加 `...`。

`response_body` 不脱敏。

### 11. 自定义 ResponseWriter wrapper

由于需要捕获 response body，不能只依赖 `chi/middleware.NewWrapResponseWriter` 的 status/bytes 能力。需要自定义 wrapper。

建议类型：

```go
type loggingResponseWriter struct {
    http.ResponseWriter
    status int
    bytes int
    body bytes.Buffer
}
```

核心行为：

- `WriteHeader(status int)`：
  - 只记录第一次 status。
  - 转发给底层 writer。
- `Write(data []byte) (int, error)`：
  - 如果尚未写 status，默认 status 为 200。
  - 转发给底层 writer。
  - 统计实际写入 bytes。
  - 如果响应 Content-Type 是 JSON，则捕获写入数据到内部 buffer，最多保留截断所需范围。
- `Status() int`：
  - 未显式写入时返回 200。
- `BytesWritten() int`：
  - 返回实际写入字节数。
- `Body() string`：
  - 返回截断后的 JSON response body；非 JSON 或空 body 返回空字符串。

兼容可选接口：

- `http.Flusher`
- `http.Hijacker`
- `http.Pusher`

实现时可参考 `D:\SourceCodes\mywork\k12-pubforge\backend-go\internal\httpapi\logging.go` 的 wrapper 模式，但保持当前项目命名和 `slog` 风格。

### 12. middleware 顺序

当前顺序：

```go
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(middleware.Recoverer)
r.Use(s.logRequest)
```

目标顺序：

```go
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(s.logRequest)
r.Use(middleware.Recoverer)
```

目的：

- `RequestID` 先生成 request id。
- `RealIP` 先处理 remote 地址。
- `logRequest` 包在 `Recoverer` 外层。
- handler panic 时由 `Recoverer` 转成 500，`logRequest` 仍能在 `next.ServeHTTP` 返回后记录 500 响应。

### 13. HTTP 访问日志字段

固定字段：

- `method`: HTTP method
- `path`: `r.URL.Path`
- `uri`: `r.URL.RequestURI()`
- `status`: 最终 HTTP status
- `bytes`: response bytes written
- `duration_ms`: 请求耗时毫秒
- `request_id`: `middleware.GetReqID(r.Context())`
- `remote_addr`: `r.RemoteAddr`
- `user_agent`: `r.UserAgent()`

条件字段：

- `request_body`: JSON request body 非空时记录，完整内容，不截断。
- `response_body`: JSON response body 非空时记录，截断后内容。
- `body_read_error`: request body 读取失败时记录。

日志消息：

```go
s.logger.Info("request completed", attrs...)
```

## Affected components

### `backend-go/go.mod` / `backend-go/go.sum`

新增依赖：

- `gopkg.in/natefinch/lumberjack.v2`

### `backend-go/config.defaults.yaml`

新增 logging rolling 默认配置：

- `logging.max_size_mb`
- `logging.max_backups`

### `backend-go/internal/config/config.go`

主要修改点：

- 扩展 `LoggingConfig`。
- 绑定新增环境变量。
- 校验 logging file 和 rolling 参数。

### `backend-go/internal/config/config_test.go`

增加配置默认值、配置合并、环境变量覆盖和非法 rolling 参数测试。

### `backend-go/internal/logging/logging.go`

主要修改点：

- 使用 `config.LoggingConfig` 初始化 logger。
- 创建日志目录。
- 使用 lumberjack rolling writer。
- 使用 `io.MultiWriter` 同时输出 stdout 和 rolling file。
- 返回 logger close 函数。

### `backend-go/internal/logging/logging_test.go`

覆盖：

- level trim。
- 文件日志输出。
- 父目录创建。
- 无效配置返回 error。

### `backend-go/cmd/backend-go/main.go`

修改 logger 初始化和关闭生命周期。

### `backend-go/internal/httpserver/server.go`

主要修改点：

- middleware 顺序。
- `Server.logRequest`。
- JSON Content-Type 判断 helper。
- request body 读取/恢复 helper。
- response writer wrapper。
- response body 截断 helper。

### `backend-go/internal/httpserver/logging_test.go`

新增测试文件，覆盖 request/response logging 行为。

## Interfaces

不新增外部 HTTP API。

新增配置项：

```yaml
logging:
  max_size_mb: 100
  max_backups: 7
```

新增环境变量：

- `POMELO_ORBIT_BACKEND__LOGGING__MAX_SIZE_MB`
- `POMELO_ORBIT_BACKEND__LOGGING__MAX_BACKUPS`

修改内部函数签名：

```go
func logging.New(cfg config.LoggingConfig) (*slog.Logger, func() error, error)
```

## Tests

### HTTP logging tests

新增 `backend-go/internal/httpserver/logging_test.go`，使用 `slog.NewJSONHandler(&buf, nil)` 捕获日志并解析 JSON。

建议测试：

1. `TestLogRequestIncludesMetadata`
   - 请求 `/api/test?x=1`
   - 断言 `method`、`path`、`uri`、`status`、`bytes`、`duration_ms`、`request_id`、`remote_addr`、`user_agent`

2. `TestLogRequestKeepsInfoLevelForErrorStatus`
   - handler 返回 500
   - 断言日志 `level` 为 `INFO`
   - 断言 `status` 为 500

3. `TestLogRequestRecordsJSONRequestBody`
   - request `Content-Type: application/json`
   - body 为 JSON
   - handler 内读取 body 并返回
   - 断言日志 `request_body` 是完整原始 body
   - 断言 handler 实际读取到了完整 body

4. `TestLogRequestSkipsNonJSONRequestBody`
   - request `Content-Type: text/plain`
   - 断言日志不包含 `request_body`

5. `TestLogRequestRecordsJSONResponseBody`
   - response `Content-Type: application/json`
   - handler 写 JSON response
   - 断言日志包含 `response_body`
   - 断言客户端响应体完整

6. `TestLogRequestSkipsNonJSONResponseBody`
   - response `Content-Type: text/plain`
   - 断言日志不包含 `response_body`

7. `TestLogRequestTruncatesResponseBody`
   - response JSON 超过 `maxResponseBodyLogLength`
   - 断言 `response_body` 被截断并带 `...`

8. `TestLogRequestDoesNotTruncateRequestBody`
   - request JSON 超过 `maxResponseBodyLogLength`
   - 断言 `request_body` 保持完整

9. `TestLogRequestLogsRecoveredPanicAsInfo`
   - router 使用 `RequestID`、`RealIP`、`s.logRequest`、`Recoverer`
   - handler panic
   - 断言响应 status 为 500
   - 断言日志 status 为 500
   - 断言日志 level 为 INFO

### logging package tests

更新 `backend-go/internal/logging/logging_test.go`：

- 增加 `level: " warn "`，确认 INFO 不启用。
- 使用 `t.TempDir()` 创建临时日志路径。
- 初始化 logger 后写入一条日志，关闭 writer，断言日志文件存在且包含日志内容。
- 使用嵌套目录路径，断言父目录会自动创建。
- 使用空 `file`、`max_size_mb <= 0` 或 `max_backups <= 0`，断言返回 error。

### config tests

更新 `backend-go/internal/config/config_test.go`：

- 默认配置包含 `Logging.MaxSizeMB == 100`、`Logging.MaxBackups == 7`。
- 自定义配置能覆盖 rolling 参数。
- 环境变量能覆盖 rolling 参数。
- 非法 rolling 参数在 `Validate()` 中返回 error。

## Alternatives considered

### 使用 zap / zerolog / logrus

不采用。原因：当前项目已经使用 `slog`，本需求的核心是 HTTP body capture 和 rolling writer，不是 logger backend。替换日志框架会扩大范围。

### 自行实现日志滚动

不采用。原因：日志滚动涉及切分、重命名、清理和平台文件行为，需求明确要求使用成熟 Go 组件更稳妥。

### 只输出到文件，不输出 stdout

不采用。原因：开发环境和容器运行通常依赖 stdout 观察日志；文件日志是新增能力，不应移除控制台输出。

### 使用 chi/httplog

不采用作为主实现。原因：可提供 request 元信息日志，但不满足完整 JSON request body 和 JSON response body capture 需求，仍需自定义 wrapper。

### 只使用 chi `middleware.NewWrapResponseWriter`

不采用作为完整方案。原因：可捕获 status/bytes，但不能捕获 response body。

### request body 也截断

不采用。用户已明确要求 `request_body` 不截断。

### 根据 status 使用 WARN/ERROR

不采用。用户已明确要求请求中间件日志级别均使用 INFO，不因非 200 响应改变日志级别。

### 字段级脱敏

不采用。用户已明确要求不处理字段脱敏。

## Risks

- 完整 request body 可能导致单条日志较大，且可能包含敏感字段；这是用户明确要求，规格中不做截断和脱敏。
- response body wrapper 如果可选接口实现不当，可能影响 streaming、hijack 或 server push 场景；实现和测试需要覆盖接口兼容的基本行为。
- `Content-Type` 可能在 `Write` 前后才设置；wrapper 应在捕获每次 `Write` 时检查 header，避免过早判断。
- panic/recover 行为依赖 middleware 顺序，需要用测试固定。
- 文件日志引入文件系统依赖，日志路径无权限或磁盘满时会导致 logger 初始化失败；这是期望的 fail fast 行为。
- 新增依赖会修改 `go.mod` / `go.sum`，需要在实现阶段通过 `go test` 验证依赖解析。

## User review notes

- 用户要求严格模式 / strict。
- 用户要求基于 `slog + chi + 自定义 body capture wrapper`。
- 用户要求 `request_body` 不截断。
- 用户要求不处理字段脱敏。
- 用户要求请求中间件日志级别均使用 INFO，不因非 200 响应改变日志级别。
- 用户要求实现文件日志及滚动。
