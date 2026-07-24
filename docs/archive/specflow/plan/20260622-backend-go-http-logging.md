# backend-go HTTP 请求/响应日志实施计划
最后修改时间: 2026-06-22 16:22:32

Review status: Draft

## Requirement / Spec basis

当前文档：

- `docs/requirement/20260622-backend-go-http-logging.md`
- `docs/spec/20260622-backend-go-http-logging.md`

核心要求：

- 每个 HTTP 请求输出 `request started` 和 `request completed` 两条结构化访问日志。
- 两条日志使用相同 `request_id` 关联。
- `request started` 在下游 handler 执行前输出。
- `request completed` 在下游 handler 返回后输出。
- 默认不记录 HTTP request/response body。
- 通过 `logging.http_body_enabled` 开启 request/response body logging。
- 通过 `logging.http_body_max_bytes` 统一限制 request/response body 日志字节数。
- `request_body` 记录在 `request started`，`response_body` 记录在 `request completed`。
- 访问日志统一 INFO。
- 不改变现有文件日志、stdout 输出、日志滚动和 logger 生命周期。

## Implementation steps

### 1. 扩展 logging 配置

修改 `backend-go/internal/config/config.go`：

- `LoggingConfig` 增加：
  - `HTTPBodyEnabled bool`
  - `HTTPBodyMaxBytes int`
- `bindEnv` 增加：
  - `logging.http_body_enabled`
  - `logging.http_body_max_bytes`
- `Validate()` 增加：
  - `logging.http_body_max_bytes > 0`

修改 `backend-go/config.defaults.yaml`：

```yaml
logging:
  http_body_enabled: false
  http_body_max_bytes: 4096
```

### 2. 更新配置测试

修改 `backend-go/internal/config/config_test.go`：

- 默认配置断言 `HTTPBodyEnabled == false`。
- 默认配置断言 `HTTPBodyMaxBytes == 4096`。
- YAML merge 覆盖 `http_body_enabled` 和 `http_body_max_bytes`。
- 环境变量覆盖：
  - `POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED`
  - `POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES`
- 非法 `http_body_max_bytes <= 0` 返回 error。

### 3. 暴露系统设置项

修改 `backend-go/internal/service/settings/service.go`：

- 增加 `logging__http_body_enabled`。
- 增加 `logging__http_body_max_bytes`。

### 4. 更新 HTTP logging middleware 配置

修改 `backend-go/internal/transport/http/middleware/logging.go`：

- 新增 `LogRequestConfig`：

```go
type LogRequestConfig struct {
    BodyEnabled  bool
    BodyMaxBytes int
}
```

- `LogRequest` 接收该配置。
- body logging 关闭时：
  - 不读取 request body。
  - 不缓存 response body。
  - 不输出 `request_body` / `response_body`。
- body logging 开启时：
  - JSON request body 最多读取 `BodyMaxBytes + 1` bytes 用于日志。
  - request body reader 通过已读前缀 + 原 reader 组合恢复，确保 handler 可读取完整 body。
  - `request_body` 和 request body 读取错误输出在 `request started`。
  - JSON response body 最多缓存 `BodyMaxBytes + 1` bytes 用于日志。
  - `response_body` 输出在 `request completed`。
  - request/response body 均按 `BodyMaxBytes` bytes 截断，超长追加 `...`。
  - 截断结果保持有效 UTF-8。

### 5. 更新 server middleware 接线

修改 `backend-go/internal/transport/http/server.go`：

- 将 `s.appCfg.Logging.HTTPBodyEnabled` 和 `s.appCfg.Logging.HTTPBodyMaxBytes` 传入 `transportmiddleware.LogRequest`。

### 6. 更新 HTTP logging 测试

修改 `backend-go/internal/transport/http/middleware/logging_test.go`：

- body logging disabled 时不输出 request/response body。
- body logging enabled 时，started 日志记录 JSON request body，completed 日志记录 JSON response body。
- 非 JSON body 不记录。
- request/response body 共用 `BodyMaxBytes` 截断。
- request body 截断用于日志时，handler 仍可读取完整原始 body。
- response body 截断用于日志时，客户端仍收到完整原始 response body。
- 保留 started/completed、request_id、INFO、panic/recover 测试。

### 7. 格式化和测试

执行：

```bash
go -C .\backend-go fmt ./internal/config ./internal/service/settings ./internal/transport/http ./internal/transport/http/middleware
go -C .\backend-go test -count=1 ./internal/config ./internal/transport/http/middleware
go -C .\backend-go test -count=1 ./...
```

## Files to change

预期产品代码文件：

- `backend-go/config.defaults.yaml`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/service/settings/service.go`
- `backend-go/internal/transport/http/server.go`
- `backend-go/internal/transport/http/middleware/logging.go`
- `backend-go/internal/transport/http/middleware/logging_test.go`

预期过程文档文件：

- `docs/requirement/20260622-backend-go-http-logging.md`
- `docs/spec/20260622-backend-go-http-logging.md`
- `docs/plan/20260622-backend-go-http-logging.md`
- `docs/verification/20260622-backend-go-http-logging.md`

旧文档路径继续保持迁移删除状态：

- `docs/requirement/20260614-backend-go-http-logging.md`
- `docs/spec/20260614-backend-go-http-logging.md`
- `docs/plan/20260614-backend-go-http-logging.md`
- `docs/verification/20260614-backend-go-http-logging.md`

## Verification plan

- 对照 requirement 检查默认 body logging 是否关闭。
- 对照 requirement 检查 request/response 是否共用配置。
- 对照 spec 检查 `http_body_max_bytes` 是否同时作用于 request 和 response body。
- 运行 config 和 middleware 包测试。
- 运行 backend-go 全量 Go 测试。
- 检查实际 diff 是否只包含本次日志配置、middleware 行为和文档更新。

## Assumptions

- `logging.http_body_enabled` 默认 false 可作为生产默认关闭。
- 开发环境通过 YAML 或环境变量显式开启 body logging。
- request/response body 共用同一 max bytes 配置，不分别配置。
- 本次不做字段级脱敏。

## Risks

- body logging 开启后仍可能记录敏感字段。
- 按 bytes 截断 JSON body 可能得到非完整 JSON 片段。
- 每个请求仍输出 started/completed 两条 access log，日志量高于单条访问日志。

## Rollback

如需回滚：

- 删除 `LoggingConfig` 新增字段和默认配置。
- 恢复 `LogRequest` 单参数签名。
- 恢复 body logging 始终启用的测试断言。
- 恢复文档中关于 body logging 配置的描述。

## User review notes

- 2026-06-22：用户要求增加请求和响应共用的 body logging 开关，默认生产关闭，开发环境可开。
- 2026-06-22：用户澄清请求/响应拆开要求：`request_body` 输出在 `request started`，`response_body` 输出在 `request completed`。
