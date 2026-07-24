# backend-go HTTP 请求/响应日志验证
最后修改时间: 2026-06-22 16:32:22

Review status: Draft

## Requirement alignment

- [x] 每个 HTTP 请求输出 `request started` 和 `request completed` 两条访问日志。
- [x] `request started` 在 handler 执行前输出。
- [x] `request completed` 在 handler 返回后输出。
- [x] 两条日志共享同一个 `request_id`。
- [x] started 日志包含 method、path、uri、request_id、remote_addr、user_agent。
- [x] completed 日志包含 method、path、uri、status、bytes、duration_ms、request_id、remote_addr、user_agent。
- [x] 默认 `logging.http_body_enabled=false`，不记录 request/response body。
- [x] `logging.http_body_max_bytes=4096` 作为默认 body 日志字节上限。
- [x] 支持环境变量覆盖 `POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED` 和 `POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES`。
- [x] `logging.http_body_max_bytes <= 0` 时配置校验失败。
- [x] body logging 开启后，JSON `request_body` 输出在 `request started`，JSON `response_body` 输出在 `request completed`，二者共用 `http_body_max_bytes` 截断。
- [x] request body 截断用于日志时，handler 仍能读取完整原始 body。
- [x] response body 截断用于日志时，客户端仍能收到完整原始 response body。
- [x] body 截断结果保持有效 UTF-8，超长追加 `...`。
- [x] 访问日志均使用 INFO 级别。
- [x] panic 经 Recoverer 转为 500 时仍输出 started 和 completed，completed status 为 500。

## Spec alignment

按 `docs/spec/20260622-backend-go-http-logging.md` 核对：

- [x] `LoggingConfig` 增加 `HTTPBodyEnabled` 和 `HTTPBodyMaxBytes`。
- [x] `config.defaults.yaml` 增加 `http_body_enabled: false` 和 `http_body_max_bytes: 4096`。
- [x] `bindEnv` 绑定新增环境变量。
- [x] `Validate()` 校验 `logging.http_body_max_bytes`。
- [x] `Server.Handler()` 将 logging 配置传入 HTTP logging middleware。
- [x] `LogRequest` 接收 `LogRequestConfig`。
- [x] body logging 关闭时不读取 request body、不缓存 response body。
- [x] body logging 开启时 request/response body 共用 max bytes，且 request body 归入 started、response body 归入 completed。
- [x] 未修改 `logging.New` 签名、文件日志、stdout 输出或日志滚动行为。

## Plan alignment

按 `docs/plan/20260622-backend-go-http-logging.md` 核对：

- [x] 修改 `backend-go/config.defaults.yaml`。
- [x] 修改 `backend-go/internal/config/config.go`。
- [x] 修改 `backend-go/internal/config/config_test.go`。
- [x] 修改 `backend-go/internal/service/settings/service.go`。
- [x] 修改 `backend-go/internal/transport/http/server.go`。
- [x] 修改 `backend-go/internal/transport/http/middleware/logging.go`。
- [x] 修改 `backend-go/internal/transport/http/middleware/logging_test.go`。
- [x] 运行 config 和 middleware 包测试。
- [x] 运行 backend-go 全量 Go 测试。

## Actual diff summary

### backend-go config/settings

- `backend-go/config.defaults.yaml`
  - 新增 `logging.http_body_enabled: false`。
  - 新增 `logging.http_body_max_bytes: 4096`。
- `backend-go/internal/config/config.go`
  - `LoggingConfig` 增加 `HTTPBodyEnabled`、`HTTPBodyMaxBytes`。
  - 新增环境变量绑定。
  - 新增 `logging.http_body_max_bytes` 正数校验。
- `backend-go/internal/config/config_test.go`
  - 覆盖默认值、YAML merge、环境变量覆盖和非法配置。
- `backend-go/internal/service/settings/service.go`
  - 系统设置定义新增 `logging__http_body_enabled` 和 `logging__http_body_max_bytes`。

### backend-go HTTP logging

- `backend-go/internal/transport/http/server.go`
  - 将 logging body 配置传入 HTTP logging middleware。
- `backend-go/internal/transport/http/middleware/logging.go`
  - 新增 `LogRequestConfig`。
  - body logging 关闭时跳过 request/response body 处理。
  - request body 只读取 `BodyMaxBytes + 1` bytes 用于日志，并恢复完整 reader 给 handler。
  - `request_body` 和 request body 读取错误输出在 `request started`。
  - response body 只缓存 `BodyMaxBytes + 1` bytes 用于日志。
  - `response_body` 输出在 `request completed`。
  - request/response body 共用按 bytes 截断逻辑，截断后保持有效 UTF-8。
- `backend-go/internal/transport/http/middleware/logging_test.go`
  - 覆盖 body logging disabled。
  - 覆盖 request body 在 started 日志输出。
  - 覆盖 response body 在 completed 日志输出。
  - 覆盖 request/response body enabled 时不串位。
  - 覆盖 request/response 共用 max bytes 截断。
  - 覆盖 request body 截断后 handler 仍读取完整 body。

### SpecFlow docs

- 更新 `docs/requirement/20260622-backend-go-http-logging.md`。
- 更新 `docs/spec/20260622-backend-go-http-logging.md`。
- 更新 `docs/plan/20260622-backend-go-http-logging.md`。
- 更新 `docs/verification/20260622-backend-go-http-logging.md`。

## Expected vs actual changed files

预期改动文件：

- [x] `backend-go/config.defaults.yaml`
- [x] `backend-go/internal/config/config.go`
- [x] `backend-go/internal/config/config_test.go`
- [x] `backend-go/internal/service/settings/service.go`
- [x] `backend-go/internal/transport/http/server.go`
- [x] `backend-go/internal/transport/http/middleware/logging.go`
- [x] `backend-go/internal/transport/http/middleware/logging_test.go`
- [x] `docs/requirement/20260622-backend-go-http-logging.md`
- [x] `docs/spec/20260622-backend-go-http-logging.md`
- [x] `docs/plan/20260622-backend-go-http-logging.md`
- [x] `docs/verification/20260622-backend-go-http-logging.md`

旧文档迁移删除：

- [x] `docs/requirement/20260614-backend-go-http-logging.md`
- [x] `docs/spec/20260614-backend-go-http-logging.md`
- [x] `docs/plan/20260614-backend-go-http-logging.md`
- [x] `docs/verification/20260614-backend-go-http-logging.md`

## Acceptance checklist

- [x] started/completed 两条日志存在且顺序正确。
- [x] request_id 一致。
- [x] 默认 body logging 关闭。
- [x] body logging 开启后 request body 可记录在 started，response body 可记录在 completed。
- [x] request/response body 共用 max bytes 截断。
- [x] request body 截断后 handler 仍读取完整 body。
- [x] response body 截断后客户端仍收到完整 body。
- [x] 非 JSON body 不记录。
- [x] panic/recover 场景符合需求。
- [x] 测试覆盖新增配置和行为。

## Test results

```bash
go -C .\backend-go fmt ./internal/transport/http/middleware
```

结果：通过。

```bash
go -C .\backend-go test -count=1 ./internal/config ./internal/transport/http/middleware ./internal/transport/http
```

结果：通过。

```bash
go -C .\backend-go test -count=1 ./...
```

结果：通过。

## Scope deviation

无范围偏差。

本次未修改 Python `backend/`，未修改 HTTP API，未修改 `logging.New`、文件日志、stdout 输出或日志滚动配置。

## Risks

- body logging 开启后仍可能记录敏感字段；本次只增加开关和大小限制，不做字段级脱敏。
- 按 bytes 截断 JSON body 可能得到非完整 JSON 片段。
- 每个请求仍输出 started/completed 两条访问日志，日志量高于单条 access log。

## Incomplete items

无。

## Conclusion

验证通过。实现与 `docs/requirement/20260622-backend-go-http-logging.md`、`docs/spec/20260622-backend-go-http-logging.md` 和 `docs/plan/20260622-backend-go-http-logging.md` 对齐，`request_body` 已归入 `request started`，`response_body` 已归入 `request completed`，受影响包测试和 backend-go 全量测试均通过。
