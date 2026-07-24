# backend-go HTTP 请求/响应日志规格
最后修改时间: 2026-06-22 16:22:32

Review status: Draft

## Requirement basis

基于当前需求文档：

- `docs/requirement/20260622-backend-go-http-logging.md`

关键约束：

- HTTP 访问日志拆分为 `request started` 和 `request completed` 两条结构化日志。
- 两条日志通过同一个 `request_id` 关联。
- 继续使用 `slog + chi + 自定义 body capture wrapper`。
- HTTP 访问日志统一使用 INFO，不因非 200、4xx 或 5xx 响应改变日志级别。
- body logging 由 `logging.http_body_enabled` 控制，默认关闭。
- request body 和 response body 共用 `logging.http_body_max_bytes`，默认 4096 bytes。
- body logging 开启时，只记录 JSON body：`request_body` 归入 `request started`，`response_body` 归入 `request completed`，并按字节数截断，截断后保持有效 UTF-8。
- 不改变现有文件日志、stdout 输出和日志滚动行为。

## Overview

本需求包含三部分：

1. 请求生命周期日志：每个 HTTP 请求输出 `request started` 和 `request completed`。
2. body logging 配置：通过 `logging.http_body_enabled` 控制 request/response body 是否记录。
3. body 日志大小控制：通过 `logging.http_body_max_bytes` 统一限制 request/response body 日志字节数。

最终效果：生产默认只记录访问元信息；开发或排障时可以打开 body logging，并用同一最大字节数限制请求体和响应体日志大小。开启后，请求体随 `request started` 输出，响应体随 `request completed` 输出。

## Design decisions

### 1. 日志消息拆分

新增开始日志：

```go
logger.Info("request started", attrs...)
```

保留完成日志：

```go
logger.Info("request completed", attrs...)
```

不新增第三种访问日志消息，不改变业务日志消息。

### 2. 开始日志字段

`request started` 记录请求开始时已确定的请求侧信息：

- `method`: HTTP method
- `path`: `r.URL.Path`
- `uri`: `r.URL.RequestURI()`
- `request_id`: `middleware.GetReqID(r.Context())`
- `remote_addr`: `r.RemoteAddr`
- `user_agent`: `r.UserAgent()`

条件字段：

- `request_body`: `logging.http_body_enabled=true` 且 JSON request body 非空时记录。
- `body_read_error`: request body 读取失败时记录。

开始日志不记录：

- `status`
- `bytes`
- `duration_ms`
- `response_body`

### 3. 完成日志字段

`request completed` 固定字段：

- `method`
- `path`
- `uri`
- `status`
- `bytes`
- `duration_ms`
- `request_id`
- `remote_addr`
- `user_agent`

条件字段：

- `response_body`: `logging.http_body_enabled=true` 且 JSON response body 非空时记录。

完成日志不记录：

- `request_body`
- `body_read_error`

### 4. logging 配置扩展

扩展 `backend-go/internal/config/config.go` 中的 `LoggingConfig`：

```go
type LoggingConfig struct {
    Level            string `mapstructure:"level" yaml:"level"`
    File             string `mapstructure:"file" yaml:"file"`
    MaxSizeMB        int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
    MaxBackups       int    `mapstructure:"max_backups" yaml:"max_backups"`
    HTTPBodyEnabled  bool   `mapstructure:"http_body_enabled" yaml:"http_body_enabled"`
    HTTPBodyMaxBytes int    `mapstructure:"http_body_max_bytes" yaml:"http_body_max_bytes"`
}
```

默认配置：

```yaml
logging:
  level: info
  file: logs/backend-go.log
  max_size_mb: 100
  max_backups: 7
  http_body_enabled: false
  http_body_max_bytes: 4096
```

环境变量：

- `POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED`
- `POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES`

校验：

- `logging.http_body_max_bytes > 0`

### 5. middleware 配置对象

新增 middleware 内部配置：

```go
type LogRequestConfig struct {
    BodyEnabled  bool
    BodyMaxBytes int
}
```

`Server.Handler()` 将应用配置传给 middleware：

```go
transportmiddleware.LogRequest(
    s.logger,
    transportmiddleware.LogRequestConfig{
        BodyEnabled:  s.appCfg.Logging.HTTPBodyEnabled,
        BodyMaxBytes: s.appCfg.Logging.HTTPBodyMaxBytes,
    },
)
```

### 6. request body 日志读取策略

body logging 关闭时：

- 不读取 request body。
- 不记录 `request_body`。

body logging 开启时：

- 仅 JSON request body 会读取用于日志。
- 为避免日志中间件完整读入大请求，只读取最多 `BodyMaxBytes + 1` bytes。
- 使用读取到的前缀和原始 reader 组合恢复 `r.Body`，确保 handler 仍能读取完整原始 body。
- `request started` 输出前完成 request body 日志前缀读取，因此 started 日志仍在 handler 前输出。
- `request started` 输出时按 `BodyMaxBytes` bytes 截断，超长追加 `...`。
- 截断后修剪不完整 UTF-8 尾部，确保日志字符串有效。

### 7. response body 捕获策略

body logging 关闭时：

- wrapper 不缓存 response body。
- 不记录 `response_body`。

body logging 开启时：

- 仅 JSON response body 会缓存用于日志。
- wrapper 最多缓存 `BodyMaxBytes + 1` bytes。
- 日志输出时按 `BodyMaxBytes` bytes 截断，超长追加 `...`。
- 截断后修剪不完整 UTF-8 尾部，确保日志字符串有效。
- 客户端仍收到完整原始 response body。

### 8. INFO 级别保持不变

访问日志是请求流水记录，不作为错误日志语义使用。即使响应为 400、404、500，两条访问日志也统一使用 INFO。

## Affected components

### `backend-go/config.defaults.yaml`

新增默认配置：

- `logging.http_body_enabled: false`
- `logging.http_body_max_bytes: 4096`

### `backend-go/internal/config/config.go`

主要修改点：

- 扩展 `LoggingConfig`。
- 绑定新增环境变量。
- 校验 `logging.http_body_max_bytes`。

### `backend-go/internal/config/config_test.go`

主要修改点：

- 覆盖默认值。
- 覆盖 YAML merge。
- 覆盖环境变量覆盖。
- 覆盖非法 `http_body_max_bytes`。

### `backend-go/internal/service/settings/service.go`

主要修改点：

- 在系统设置定义中暴露 `logging__http_body_enabled`。
- 在系统设置定义中暴露 `logging__http_body_max_bytes`。

### `backend-go/internal/transport/http/server.go`

主要修改点：

- 将 `cfg.Logging.HTTPBodyEnabled` 和 `cfg.Logging.HTTPBodyMaxBytes` 传入 HTTP logging middleware。

### `backend-go/internal/transport/http/middleware/logging.go`

主要修改点：

- `LogRequest` 接收 `LogRequestConfig`。
- body logging 关闭时不读取 request body、不缓存 response body。
- request/response body 共用 `BodyMaxBytes` 截断。
- request body 日志只读前缀并恢复完整 reader。

### `backend-go/internal/transport/http/middleware/logging_test.go`

主要修改点：

- 覆盖 body logging 关闭时不输出 body 字段。
- 覆盖 body logging 开启时 request body 输出在 started 日志、response body 输出在 completed 日志。
- 覆盖 request/response body 共用 max bytes 截断。
- 覆盖截断后 request body handler 仍可读取完整原始 body。

## Interfaces

不新增外部 HTTP API。

新增配置项：

```yaml
logging:
  http_body_enabled: false
  http_body_max_bytes: 4096
```

新增环境变量：

- `POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED`
- `POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES`

不修改 `logging.New` 签名。

## Tests

HTTP logging 测试覆盖：

1. 单次请求产生 started/completed 两条日志，且 request_id 一致。
2. started 日志包含基础元信息，不包含完成态字段；body logging 开启时可包含 `request_body`。
3. completed 日志包含完成态字段，不包含 `request_body`。
4. body logging disabled 时不输出 `request_body` / `response_body`。
5. body logging enabled 时在 started 日志记录 JSON request body，并保持 handler 读取完整 body。
6. body logging enabled 时在 completed 日志记录 JSON response body，并保持客户端响应完整。
7. 非 JSON request/response body 不记录。
8. request/response body 按同一个 max bytes 截断，并保持有效 UTF-8。
9. panic 被 Recoverer 转为 500 后仍输出 started/completed，completed 为 INFO 且 status=500。

配置测试覆盖：

1. 默认 `http_body_enabled=false`。
2. 默认 `http_body_max_bytes=4096`。
3. YAML 配置覆盖。
4. 环境变量覆盖。
5. 非法 `http_body_max_bytes <= 0` 报错。

## Risks

- body logging 开启后仍可能记录敏感字段；本次只提供开关和大小限制，不做脱敏。
- 按 bytes 截断可能得到非完整 JSON 片段；这是日志体积控制的预期结果。
- 每个请求仍会输出 started/completed 两条日志，日志量高于单条 access log。

## User review notes

- 2026-06-22：用户要求将请求开始和结束分开记录，并同步移动/更新 20260614 系列文档到 20260622。
- 2026-06-22：用户要求增加请求和响应共用的 body logging 开关，默认生产关闭，开发环境可开。
- 2026-06-22：用户澄清请求/响应拆开要求：`request_body` 输出在 `request started`，`response_body` 输出在 `request completed`。
