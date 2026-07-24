# backend-go 日志系统改进
最后修改时间: 2026-06-25 12:34:54

Review status: Accepted

## Background

当前 `backend-go` 已有 `internal/logging`、HTTP 请求日志中间件和 `slog` 注入，但日志入口仍然分散：配置 logger 不是进程默认 logger，部分包级 `slog.*` 调用会绕开统一输出；HTTP 请求日志虽然已记录 started / completed，但还没有完全对齐 TermBridge-go 的 requestlog 行为。

用户要求对比 `D:\SourceCodes\mywork\TermBridge-go` 的实现直接收敛，不要求向后兼容，也不要求保留旧配置分支。目标是把 backend-go 的日志系统改成“应用启动创建唯一 logger，HTTP 层统一请求日志，默认 logger 也进入同一输出”的模式。

## Goal

1. 让 `internal/logging` 创建的 logger 成为进程默认 logger。
2. 让 HTTP 请求日志行为更接近 TermBridge-go 的 requestlog 中间件。
3. 让包级 `slog.Error(...)`、`slog.Info(...)` 和默认构造路径都落到配置好的日志输出。
4. 补齐相关回归测试，防止日志路径再次分叉。

## Non-goal

1. 不保留旧的日志兼容分支。
2. 不维持旧的“默认 logger 继续绕过配置输出”的行为。
3. 不新增额外的日志输出通道或独立日志系统。
4. 不把日志改动扩展到无关业务逻辑。

## User scenarios

### 场景一：应用启动后所有日志统一输出

作为开发者，我启动 backend-go 后，后续的包级 `slog.*` 调用和显式注入 logger 的调用都应该写到同一套配置日志中，而不是一部分走默认 stderr、一部分走文件。

### 场景二：HTTP 请求能看到统一结构化日志

作为开发者，我希望每次 HTTP 请求都能看到 started / completed 两条结构化日志，字段与 TermBridge-go 的 requestlog 风格保持一致，方便定位请求路径、状态码、耗时和 body。

### 场景三：日志回归测试可直接捕获

作为开发者，我希望日志系统的默认 logger、HTTP middleware 和 response helper 都有回归测试，避免以后改动时重新引入分叉。

## Acceptance

1. `internal/logging.New` 创建 logger 后，进程默认 logger 被切换到该实例。
2. 包级 `slog.Info(...)` / `slog.Error(...)` 会写入 backend-go 配置的日志输出。
3. HTTP 请求日志继续记录 started / completed 两条日志，并保留 request_id、method、path、uri、status、bytes、duration_ms、remote_addr、user_agent 等字段。
4. 请求 body / 响应 body 仍只在 JSON 或 `+json` 内容类型下记录，并保持截断与 UTF-8 合法性。
5. `loggingResponseWriter` 继续保留可选接口能力，不破坏 flush / hijack / push / unwrap。
6. 相关单元测试通过，并能证明默认 logger、HTTP middleware、response helper 的日志路径都被覆盖。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 直接按 TermBridge-go 的思路收敛日志行为，不保留旧兼容分支。
2. 保持现有配置结构和调用方式，不额外引入新的日志子系统。
3. HTTP 请求日志继续使用现有 chi 中间件链，只把 request metadata 和 writer 行为对齐到更稳定的实现。

## Risk

1. 进程默认 logger 变更会影响所有包级 `slog.*` 调用，测试需要显式恢复默认 logger。
2. HTTP middleware 行为收敛可能改变已有日志字段或顺序，相关测试必须同步更新。
3. response helper 依赖默认 logger，若默认 logger 初始化失败，错误日志将回退到标准库默认行为。
