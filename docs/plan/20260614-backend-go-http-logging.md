# backend-go HTTP 请求/响应日志实施计划

Review status: Accepted

## Requirement / Spec basis

已接受文档：

- `docs/requirement/20260614-backend-go-http-logging.md`
- `docs/spec/20260614-backend-go-http-logging.md`

核心要求：

- 使用 `slog + chi + 自定义 body capture wrapper`。
- HTTP 请求中间件访问日志统一 INFO。
- `request_body` 完整记录，不截断，不脱敏。
- `response_body` 记录但截断，不脱敏。
- 非 JSON request/response body 不记录。
- 实现文件日志及按大小滚动。
- stdout 和 rolling file 同时输出。
- 使用成熟 Go 组件 `gopkg.in/natefinch/lumberjack.v2` 实现滚动。
- 不依赖 Python `backend/`。

## Implementation steps

### 1. 引入滚动日志依赖

在 `backend-go` 模块内引入：

- `gopkg.in/natefinch/lumberjack.v2`

预期改动：

- `backend-go/go.mod`
- `backend-go/go.sum`

方式：

```bash
cd backend-go
go get gopkg.in/natefinch/lumberjack.v2
```

如果后续 `go test ./...` 触发依赖整理，以最终 `go.mod` / `go.sum` 为准。

### 2. 扩展 logging 配置

修改 `backend-go/internal/config/config.go`：

- 扩展 `LoggingConfig`：

```go
type LoggingConfig struct {
    Level     string `mapstructure:"level" yaml:"level"`
    File      string `mapstructure:"file" yaml:"file"`
    MaxSizeMB int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
    MaxBackups int   `mapstructure:"max_backups" yaml:"max_backups"`
}
```

- 在 `bindEnv` 中增加：
  - `logging.max_size_mb`
  - `logging.max_backups`
- 在 `Validate()` 中增加校验：
  - `logging.file` 非空
  - `logging.max_size_mb > 0`
  - `logging.max_backups > 0`

修改 `backend-go/config.defaults.yaml`：

```yaml
logging:
  level: info
  file: logs/backend-go.log
  max_size_mb: 100
  max_backups: 7
```

### 3. 更新配置测试

修改 `backend-go/internal/config/config_test.go`：

- 默认配置测试断言：
  - `cfg.Logging.File == "logs/backend-go.log"`
  - `cfg.Logging.MaxSizeMB == 100`
  - `cfg.Logging.MaxBackups == 7`
- 自定义配置合并测试覆盖 rolling 参数。
- 环境变量覆盖测试覆盖：
  - `POMELO_ORBIT_BACKEND__LOGGING__MAX_SIZE_MB`
  - `POMELO_ORBIT_BACKEND__LOGGING__MAX_BACKUPS`
- 增加非法配置测试：
  - 空 `logging.file`
  - `max_size_mb <= 0`
  - `max_backups <= 0`

### 4. 重构 logging.New

修改 `backend-go/internal/logging/logging.go`：

- 新签名：

```go
func New(cfg config.LoggingConfig) (*slog.Logger, func() error, error)
```

- 实现：
  - level 使用 `strings.TrimSpace` + `strings.ToUpper` 解析。
  - 创建 `cfg.File` 父目录：`os.MkdirAll(filepath.Dir(cfg.File), 0o755)`。
  - 创建 `lumberjack.Logger`：

```go
rollingWriter := &lumberjack.Logger{
    Filename:   cfg.File,
    MaxSize:    cfg.MaxSizeMB,
    MaxBackups: cfg.MaxBackups,
}
```

  - 用 `io.MultiWriter(os.Stdout, rollingWriter)` 同时输出控制台和文件。
  - 创建 `slog.NewTextHandler`。
  - 返回 logger、`rollingWriter.Close`、nil。

注意：

- 不保留旧签名兼容层。
- 不实现 `max_age` / `compress`。
- 不改变日志格式为 JSON。

### 5. 更新 logging 测试

修改 `backend-go/internal/logging/logging_test.go`：

- 更新调用为新签名。
- 保留 level 行为测试，包括 `" warn "`。
- 使用 `t.TempDir()` 验证文件输出：
  - 创建临时日志路径。
  - 初始化 logger。
  - 写入一条日志。
  - 关闭 writer。
  - 断言日志文件存在且包含日志内容。
- 验证嵌套父目录会自动创建。
- 验证空 `file`、`max_size_mb <= 0`、`max_backups <= 0` 返回 error。

### 6. 更新 main logger 生命周期

修改 `backend-go/cmd/backend-go/main.go`：

- 将：

```go
logger := logging.New(cfg.Logging.Level)
```

改为：

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
```

- `app.New(cfg, logger)` 保持不变。

### 7. 增强 HTTP logRequest

修改 `backend-go/internal/httpserver/server.go`。

#### middleware 顺序

改为：

```go
r.Use(middleware.RequestID)
r.Use(middleware.RealIP)
r.Use(s.logRequest)
r.Use(middleware.Recoverer)
```

#### 新增 helper / wrapper

新增常量：

```go
const maxResponseBodyLogLength = 4096
```

新增 helper：

- `isJSONContentType(contentType string) bool`
- `readRequestBodyForLog(r *http.Request) (string, error)`
- `truncateLogBody(value string, limit int) string`

新增 `loggingResponseWriter`：

- 捕获 status。
- 统计 bytes。
- 在响应 `Content-Type` 为 JSON 时捕获 response body。
- response body 最多保留截断所需内容。
- 支持 `Flush`、`Hijack`、`Push`。

#### 更新 logRequest 主流程

- 请求开始时读取 JSON request body 并恢复。
- 使用 `loggingResponseWriter` 包装 response writer。
- 调用 `next.ServeHTTP`。
- 构造 `[]any` 字段：
  - `method`
  - `path`
  - `uri`
  - `status`
  - `bytes`
  - `duration_ms`
  - `request_id`
  - `remote_addr`
  - `user_agent`
  - 条件：`request_body`
  - 条件：`response_body`
  - 条件：`body_read_error`
- 始终使用：

```go
s.logger.Info("request completed", attrs...)
```

### 8. 新增 HTTP logging 测试

新增 `backend-go/internal/httpserver/logging_test.go`。

测试工具：

- `bytes.Buffer` 捕获 `slog.NewJSONHandler` 输出。
- `encoding/json` 解析单条日志。
- 直接构造 `Server{logger: logger}` 或使用小型 chi router。

测试用例：

1. `TestLogRequestIncludesMetadata`
2. `TestLogRequestKeepsInfoLevelForErrorStatus`
3. `TestLogRequestRecordsJSONRequestBody`
4. `TestLogRequestSkipsNonJSONRequestBody`
5. `TestLogRequestRecordsJSONResponseBody`
6. `TestLogRequestSkipsNonJSONResponseBody`
7. `TestLogRequestTruncatesResponseBody`
8. `TestLogRequestDoesNotTruncateRequestBody`
9. `TestLogRequestLogsRecoveredPanicAsInfo`

### 9. 格式化和测试

执行：

```bash
cd backend-go
go fmt ./cmd/backend-go ./internal/config ./internal/logging ./internal/httpserver
go test ./internal/config ./internal/logging ./internal/httpserver
go test ./...
```

### 10. Verification 文档

实现和验证完成后新增：

- `docs/verification/20260614-backend-go-http-logging.md`

记录：

- requirement/spec/plan 对齐情况。
- 实际 diff 摘要。
- 测试命令和结果。
- 剩余风险。
- 不执行 git commit/push。

## Files to change

- `backend-go/go.mod`
- `backend-go/go.sum`
- `backend-go/config.defaults.yaml`
- `backend-go/cmd/backend-go/main.go`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/logging/logging.go`
- `backend-go/internal/logging/logging_test.go`
- `backend-go/internal/httpserver/server.go`
- `backend-go/internal/httpserver/logging_test.go`
- `docs/verification/20260614-backend-go-http-logging.md`

## Verification plan

最小验证：

```bash
cd backend-go
go test ./internal/config ./internal/logging ./internal/httpserver
```

完整验证：

```bash
cd backend-go
go test ./...
```

格式化：

```bash
cd backend-go
go fmt ./cmd/backend-go ./internal/config ./internal/logging ./internal/httpserver
```

## Assumptions

- 使用 `gopkg.in/natefinch/lumberjack.v2` 可以接受新增 Go 依赖。
- 默认滚动参数采用 `max_size_mb: 100`、`max_backups: 7`。
- `response_body` 截断长度采用 4096 rune。
- `request_body` 不截断，即使较大也完整进入日志。
- 文件日志路径相对 `backend-go` 进程工作目录解析。

## Risks

- 完整 request body 可能导致日志很大，且可能包含敏感字段；这是明确需求，不做截断和脱敏。
- 文件日志路径无权限或磁盘不可写时，logger 初始化会失败并阻止启动。
- 新增依赖会修改 `go.mod` / `go.sum`。
- response writer wrapper 需要谨慎实现可选接口，避免破坏特殊 HTTP 行为。

## Rollback

如需回滚：

- 恢复 `logging.New` 旧签名和调用。
- 移除 `lumberjack` 依赖及 rolling 配置字段。
- 恢复 `Server.logRequest` 简单实现。
- 删除新增 `logging_test.go` / verification 文档中对应内容。

## User review notes

- 用户要求进入 Plan / 计划阶段。
