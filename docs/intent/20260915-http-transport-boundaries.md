# HTTP Transport 边界收敛
最后修改时间: 2026-09-15 22:46:20

Review status: Draft

Mode: standard

## Background

当前 HTTP 入站适配层将请求解码、protobuf JSON 编解码、成功响应、错误契约、分页与 proto 字段转换分散在 `internal/api/http/binding`、`codec`、`response` 三个包中。`response` 还同时承载 HTTP status 映射、request ID 错误 body、`structpb.Value`、可选标量和 RFC3339 格式化，成为跨 handler 的杂项依赖。

现有 HTTP API 的所有 JSON 请求体均为生成的 protobuf message。成功响应以 `protojson` 输出，错误响应使用稳定的 `{code, error, requestId}` JSON 契约；NoRoute、NoMethod、CORS 拒绝、认证授权和 panic recovery 已复用统一错误 writer。

## Goal

将已有共享 HTTP transport 支持做包结构收敛和冗余消除，使 handler 能明确完成“解码 proto 请求、映射 DTO、调用 usecase、输出成功或错误响应”的既有流程。

保持 HTTP 对外 proto JSON 与错误响应契约、应用层边界、请求校验规则和所有现有行为不变。

## Non-goal

- 不让 application、model、repository 或 infrastructure 直接依赖 `internal/gen/proto`。
- 不改用 Gin 标准 JSON binding，也不以 `encoding/json` 替代 protobuf JSON。
- 不迁移至 RFC 9457 Problem Details，或改变既有错误字段、snake_case proto JSON、`EmitUnpopulated` 和未知字段拒绝语义。
- 不重设计或重新实现业务错误分类、HTTP status 映射、响应格式、请求校验、分页业务规则、HTTP 路由或 MCP 契约。
- 不改变非法 query 参数回退默认值、请求体读取或 middleware 顺序等现有行为。

## User scenarios

1. HTTP handler 接收任一 proto 请求时，继续通过统一 protobuf JSON 解码入口读取 body，并在无效 JSON 时返回既有 400 错误契约。
2. HTTP handler 返回 proto message 时，继续以既有字段命名和空值规则输出；SSE 与 task payload 继续复用相同 protobuf JSON 编解码规则。
3. 应用错误、认证授权失败、CORS 拒绝、路由 fallback 和 recovery 继续通过同一错误 writer 输出既有 status、code 和 request ID，且 5xx 继续脱敏。

## Acceptance

- 消除未被实际调用的泛化和仅转发一次的 helper，且不引入等价的新冗余。
- protobuf JSON 的 marshal/unmarshal options 仍只有一个受控 owner；常规 HTTP、SSE 和 task payload 的 JSON 语义保持一致。
- 成功响应、错误响应和字段转换不再混杂在语义为 `response` 的杂项包中；目录和导出 API 能表达请求、编码、输出、错误映射和 mapper 辅助转换的职责。
- 保持既有错误 body、HTTP status、401 `WWW-Authenticate`、405 `Allow`、request ID 透传/生成、5xx 脱敏、请求校验和 query 参数语义。
- 相关 Go 测试通过，并证明重组前后的 HTTP 可观察行为等价。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- protobuf 生成类型继续限定在 HTTP transport adapter，应用层使用 DTO、model 与 port。
- 保留项目既有 `{code, error, requestId}` 错误契约；RFC 9457 仅作为外部参考，不作为本次迁移目标。
- Gin custom renderer 与 `protojson` 是保留能力；任务目标是收敛 owner 和调用入口，而非替换协议实现。
- 本任务只做包结构重组与冗余消除；不顺带改变 response、error、validation、query 或 body handling 行为。
- 共享 HTTP transport 能力收敛到单一 `internal/api/http/transport` 包；以职责清楚的源文件划分内部实现，不保留多个 transport 子包。

## Risk

- 当前 `WriteStatusError` 有大量 handler、认证和 middleware 调用点；必须分阶段迁移，避免改变 400、401、403、404 和 503 现有语义。
- 跨包移动 protobuf JSON、错误 writer 和 mapper 辅助函数时，必须避免 import cycle，并保持 SSE 与 task payload 的复用路径。
- 工作区存在与本任务无关的进行中修改；实施时不得覆盖或混入这些变更。

## User review notes

- 2026-09-15：用户要求以标准模式记录本任务。
- 2026-09-15：用户明确本任务仅整理已有包结构与消除冗余，不重新实现或改变响应格式、校验规则及其他既有行为。
- 2026-09-15：用户选择单一 `internal/api/http/transport` 包。
