# HTTP 错误契约规范化需求
最后修改时间: 2026-07-26 16:45:43

Review status: Accepted

Flow mode: standard

## Background

当前 HTTP API 的错误响应仍使用 `{ "detail": ... }`。业务 handler 在多个包内分别通过 `h.writeError` 或直接调用 `transportresponse.Error` 写响应；部分路径把 `err.Error()` 直接返回给调用方。现有 `apperror.Error` 会将业务文案与根因拼接，无法区分可公开的错误信息和只应记录日志的 cause。

前端 Axios client 直接读取 `response.data.detail`，只保留 HTTP status，不能稳定消费应用错误码或请求关联 ID。项目已经具备 Request ID middleware：可透传或生成 request id，写入响应 header，并记录到日志；当前新生成 ID 使用 UUID，需要迁移为 ULID。

本需求以 `D:\SourceCodes\mywork\best-practices\docs\guides\http-error-contract.manual.md` 和 `D:\SourceCodes\mywork\best-practices\docs\guides\http-error-contract.md` 为错误契约基线。

## Goal

- 所有纳入范围的 HTTP API 错误响应统一为：

  ```json
  {
    "code": "not_found",
    "error": "Resource not found.",
    "requestId": "..."
  }
  ```

- `code` 使用稳定、机器可读的 snake_case 应用错误码；HTTP status 仅存在于状态行，不在 body 重复。
- 在应用错误模型中区分可安全返回的 message 与仅用于日志的 cause；未知错误和内部错误统一映射为 `500` / `internal_error`，不得泄露 SQL、路径、第三方响应、token、stack 或原始 cause。
- 集中维护 `Kind -> HTTP status -> default code` 映射，并提供统一 error writer；handler 不再自行拼接错误 body 或直接将 `err.Error()` 写入 HTTP 响应。
- 错误响应 body 中的 `requestId` 与 `X-Request-ID` response header 和日志中的 `request_id` 保持同一值；没有上游 request id 时生成 ULID。
- 前端 API client 校验新错误契约，输出带 status、code 和 requestId 的 typed error；非契约错误响应归类为 contract mismatch。
- MCP 的 Orbit HTTP client 能读取新错误 body，保留可用的 status、code 和 requestId，并向 MCP 调用方输出安全错误摘要。

## Non-goal

- 不为成功响应引入通用 envelope，protobuf 成功 DTO 与 `204 No Content` 语义保持不变。
- 本期不定义 `details` 的语义或返回空 `details`；字段级 validation、冲突对象和 retry hint 需在后续需求中单独定义。
- 不预先枚举无明确消费者的细粒度业务错误码；先实现状态类别对应的默认 code，特殊 code 按后续业务需要增加。
- 不改变认证产品流程；401 仍由现有前端认证边界按 HTTP status 处理。
- 不处理 environment 移除、MCP 工具参数迁移、部署领域或无关页面逻辑；MCP 仅在其 Orbit HTTP client 错误解析处适配新契约。

## User scenarios

1. 前端提交无效 JSON 或业务字段不合法时，收到 400 和稳定 `validation_failed`，并展示安全摘要。
2. 用户访问无权限资源、不可见资源或发生并发冲突时，前端可分别获得 403 / `forbidden`、404 / `not_found`、409 / `conflict`。
3. 数据库、第三方服务或未分类异常失败时，调用方获得 500 / `internal_error` 与 requestId，服务端日志可用同一 request_id 检索完整 cause。
4. API client 收到 HTML、空 body 或旧 `{ "detail": ... }` 响应时，将其标记为 contract mismatch，而不是向页面暴露原始响应。

## Acceptance

- 所有纳入范围的业务 HTTP API 错误出口不再返回 `{ "detail": ... }`，而是返回 `code`、`error` 和 `requestId`。
- 默认映射至少覆盖 validation、unauthorized、forbidden、not found、conflict 和 internal；未知异常固定为 500 / `internal_error`。
- 每个 500 响应均不包含 cause 或底层错误文本；完整错误仍以 request_id 写入日志。
- response header `X-Request-ID` 与错误 body `requestId` 一致；请求带该 header 时保持透传语义。
- 新生成的 request id 为 ULID；不再用 UUID 生成 request id。
- handler 不再维护重复的 status/code/body 映射；直接 transport、binding 和 usecase 错误最终使用统一 error writer 或等价集中出口。
- 前端 client 不再读取 `response.data.detail`，而是校验并保留 `status`、`code`、`requestId`；非契约响应产生 contract mismatch。
- MCP Orbit client 不再依赖 `detail`，可消费新错误契约且不向 MCP 调用方暴露原始 error body。
- 项目负责的 API fallback 和 panic recovery 使用同一错误契约：未知 API route 为 404 / `not_found`，method mismatch 为 405 / `method_not_allowed`，panic 为 500 / `internal_error`。
- 后端测试覆盖 400、401、403、404、409、500、request ID 透传/生成和 500 不泄露 cause；前端测试覆盖结构化错误转换、401 既有边界和 contract mismatch。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard：Requirement -> Plan -> Implementation -> Verification。
- 使用 `code`、`error`、`requestId` 作为失败响应基线字段；不沿用 `detail`。
- `details` 不作为本期兼容字段或占位字段。
- 已有 Request ID middleware 是本期的关联 ID 基础；不在 handler 内重复生成 ID。新 ID 使用 ULID，保留已有 header 的透传语义。
- 项目负责 API fallback、method mismatch 和 panic recovery 的标准 HTTP status 与错误 code 必须符合最佳实践，并使用同一错误 body。
- 前端 Web API client 和 MCP Orbit HTTP client 都纳入调用方迁移范围；MCP 仅处理错误契约解析，不在本期处理环境移除相关工具变更。

## Risk

- 这是破坏性 HTTP 契约变更；所有直接消费错误响应的前端代码、测试、脚本和外部调用方均需同步更新。
- 若 handler 迁移不完整，可能同时存在旧 `{ "detail": ... }` 和新错误 body，导致 client 行为不确定。
- 若 safe message 与 cause 分离不彻底，仍可能在 500 响应中泄露内部信息，或因过度脱敏而失去必要的业务提示。
- 将 Gin 的 fallback、method mismatch 或 recovery 改为统一 JSON body 时，必须避免破坏 SPA fallback、CORS 和已经写出响应的 panic 边界。
- 现有工作区含有与本需求无关的 environment 移除改动；后续实现和验证必须隔离其 diff，避免误归属或回退。

## User review notes

- 2026-07-26：用户要求按 `specflow` 标准模式启动 HTTP 异常处理规范化任务。
- 2026-07-26：用户指定错误契约参考 best-practices 的 HTTP error contract 文档。
- 2026-07-26：用户确认 Request ID 使用 ULID；项目负责的状态码遵循最佳实践；MCP 需满足新错误契约的使用要求。
