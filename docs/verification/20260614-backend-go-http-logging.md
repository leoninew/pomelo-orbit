# backend-go HTTP 请求/响应日志验证

Review status: Draft

## Requirement alignment

- [x] 使用 `slog + chi + 自定义 body capture wrapper`。
- [x] HTTP 请求中间件访问日志统一使用 INFO，不因非 200、4xx 或 5xx 响应改变日志级别。
- [x] JSON request body 完整记录，不截断，不做字段级脱敏。
- [x] JSON response body 记录并按 4096 rune 截断，不做字段级脱敏。
- [x] 非 JSON request/response body 不记录。
- [x] 读取 JSON request body 后恢复 `r.Body`，handler 可继续读取完整 body。
- [x] 捕获 JSON response body 后，客户端仍收到完整响应。
- [x] 实现文件日志并保留 stdout 输出。
- [x] 使用成熟 Go 组件 `gopkg.in/natefinch/lumberjack.v2` 实现按大小滚动。
- [x] 不依赖 Python `backend/`。

## Spec alignment

- [x] `logging.New` 改为使用 `config.LoggingConfig` 初始化 logger。
- [x] `logging.New` 返回 `*slog.Logger`、close 函数和 error。
- [x] `logging.file` 指定日志文件路径。
- [x] `logging.max_size_mb` 指定单文件大小上限。
- [x] `logging.max_backups` 指定保留滚动文件数量。
- [x] logger 使用 `io.MultiWriter(os.Stdout, rollingWriter)` 同时输出到 stdout 和滚动文件。
- [x] `cmd/backend-go/main.go` 管理 logger close 生命周期。
- [x] `Server.logRequest` 使用自定义 `loggingResponseWriter` 捕获 status、bytes 和 JSON response body。
- [x] `readRequestBodyForLog` 读取 JSON request body 后恢复 `r.Body`。
- [x] middleware 顺序调整为 `RequestID` → `RealIP` → `logRequest` → `Recoverer`。

## Plan alignment

计划中的实现步骤均已执行：

- [x] 引入 `gopkg.in/natefinch/lumberjack.v2`。
- [x] 扩展 `LoggingConfig`。
- [x] 更新 `config.defaults.yaml`。
- [x] 更新配置校验和配置测试。
- [x] 重构 `logging.New`。
- [x] 更新 logging 测试。
- [x] 更新 main logger 生命周期。
- [x] 增强 HTTP request/response logging。
- [x] 新增 HTTP logging 测试。
- [x] 执行格式化和测试。

## Actual diff summary

### backend-go logging/config

- `backend-go/go.mod` / `backend-go/go.sum`
  - 新增 `gopkg.in/natefinch/lumberjack.v2`。
- `backend-go/config.defaults.yaml`
  - 新增 `logging.max_size_mb: 100`。
  - 新增 `logging.max_backups: 7`。
- `backend-go/internal/config/config.go`
  - `LoggingConfig` 增加 `MaxSizeMB`、`MaxBackups`。
  - 绑定新增环境变量。
  - 校验 `logging.file`、`logging.max_size_mb`、`logging.max_backups`。
- `backend-go/internal/logging/logging.go`
  - logger 初始化改为 stdout + rolling file 双输出。
  - 创建日志父目录。
  - 返回 close 函数。
- `backend-go/cmd/backend-go/main.go`
  - 使用新 `logging.New(cfg.Logging)`。
  - 启动失败时 fail fast。
  - defer 关闭 logger writer。

### backend-go HTTP logging

- `backend-go/internal/httpserver/server.go`
  - 增加 JSON request body 读取和恢复。
  - 增加 JSON response body capture wrapper。
  - response body 截断长度为 4096 rune。
  - 访问日志包含 method/path/uri/status/bytes/duration_ms/request_id/remote_addr/user_agent。
  - 访问日志固定 INFO。
  - middleware 顺序调整以记录 panic 后的 500。
- `backend-go/internal/httpserver/logging_test.go`
  - 新增 HTTP logging 行为测试。

### tests

- `backend-go/internal/config/config_test.go`
  - 覆盖 logging 默认值、merge、env override、非法配置。
- `backend-go/internal/logging/logging_test.go`
  - 覆盖文件输出、目录创建、配置校验、level trim。

### SpecFlow docs

- `docs/requirement/20260614-backend-go-http-logging.md`
- `docs/spec/20260614-backend-go-http-logging.md`
- `docs/plan/20260614-backend-go-http-logging.md`
- `docs/verification/20260614-backend-go-http-logging.md`

## Planned vs actual changed files

计划内文件：

- [x] `backend-go/go.mod`
- [x] `backend-go/go.sum`
- [x] `backend-go/config.defaults.yaml`
- [x] `backend-go/cmd/backend-go/main.go`
- [x] `backend-go/internal/config/config.go`
- [x] `backend-go/internal/config/config_test.go`
- [x] `backend-go/internal/logging/logging.go`
- [x] `backend-go/internal/logging/logging_test.go`
- [x] `backend-go/internal/httpserver/server.go`
- [x] `backend-go/internal/httpserver/logging_test.go`
- [x] `docs/requirement/20260614-backend-go-http-logging.md`
- [x] `docs/spec/20260614-backend-go-http-logging.md`
- [x] `docs/plan/20260614-backend-go-http-logging.md`
- [x] `docs/verification/20260614-backend-go-http-logging.md`

工作区中还存在本任务未修改的文件：

- `backend-go/.gitignore`
- `scripts/dev.py`

这些文件不是本次日志需求的交付内容。

## Test results

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/backend-go && go fmt ./cmd/backend-go ./internal/config ./internal/logging ./internal/httpserver && go test ./internal/config ./internal/logging ./internal/httpserver
```

结果：通过。

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/backend-go && go test ./...
```

结果：通过。

## Missed or expanded scope

- 未实现 `max_age`、`compress`，符合计划中的最小滚动配置范围。
- 未实现字段级脱敏，符合用户明确要求。
- 未截断 `request_body`，符合用户明确要求。
- 未修改 Python `backend/`，符合约束。

## Risks

- 完整 `request_body` 可能导致单条日志较大，也可能包含敏感字段；这是明确需求，本次不截断、不脱敏。
- 文件日志路径不可写时，logger 初始化会失败并阻止服务启动；这是 fail fast 行为。
- 当前工作区存在与本任务无关的 `backend-go/.gitignore` 和 `scripts/dev.py` 改动，提交前需要注意边界。

## Conclusion

实现与已接受的 Requirement / Spec / Plan 对齐。目标测试和完整 Go 测试均通过。