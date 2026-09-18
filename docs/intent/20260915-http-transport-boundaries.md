# HTTP Transport 边界收敛
最后修改时间: 2026-09-16 09:56:29

Review status: Accepted

Mode: standard

## Background

HTTP 入站适配层目前将请求解码、protobuf JSON 编解码、成功输出、错误契约和通用 proto 字段转换拆在 `internal/api/http/binding`、`codec`、`response` 三个包。`response` 同时承担成功 proto 输出、错误分类与输出、分页、时间和 `structpb` 转换，包名无法表达这些职责。

现有 HTTP 请求体均为生成的 protobuf message。`codec` 是 `protojson` options 的唯一 owner；常规 HTTP 响应、SSE 和 task payload 均复用其编码规则。所有应用负责的失败边界，包括 NoRoute、NoMethod、认证授权、CORS 拒绝和 panic recovery，均调用 `response.WriteError` 或 `response.WriteStatusError`。HTTP adapter 外的 application、model、repository 与 infrastructure 当前未依赖 `internal/gen/proto`。

## Goal

1. 将共享 HTTP transport 能力收敛到唯一的 `internal/api/http/transport` 包，以职责清楚的源文件组织请求解码与 query 读取、protobuf JSON 编解码、成功输出、错误映射与输出、mapper 辅助转换。
2. 删除没有实际调用方的泛化能力和仅提供一次转发价值的 helper；不以新的包装层替代它们。
3. 让 handler 保持既有的适配器职责：解码 proto 请求、映射 DTO、调用 usecase、输出成功或错误响应；生成 proto 类型继续限定在 HTTP transport adapter。
4. 保持所有可观察 HTTP 行为不变，包括 proto JSON、错误响应、HTTP status、headers、request ID、校验和 query 默认值语义。

## Non-goal

- 不让 application、model、repository 或 infrastructure 依赖 `internal/gen/proto`。
- 不改用 Gin 标准 JSON binding 或 `encoding/json` 替代 protobuf JSON。
- 不改变 proto JSON 的 snake_case 字段名、`EmitUnpopulated`、未知字段拒绝，或新增另一套 options owner。
- 不重设计应用错误分类、HTTP status 映射、错误 body、请求校验、分页规则、HTTP 路由、middleware 顺序或 MCP 契约。
- 不改变 CORS 预检成功响应、非法 query 参数回退、请求体读取、SSE 或 task payload 的现有语义。
- 不顺带修正或重写与本次包收敛无关的日志字段、业务 mapper 或前端行为。

## User scenarios

1. HTTP handler 接收任一 proto 请求时，仍通过统一 protobuf JSON 解码入口读取 body；无效 JSON 继续返回既有 `400` 错误契约。
2. HTTP handler 返回 proto message 时，继续以既有字段命名和空值规则输出；SSE 与 task payload 继续复用同一 protobuf JSON 编解码规则。
3. 应用错误、认证授权失败、CORS 拒绝、路由 fallback 和 recovery 继续通过同一错误 writer 输出既有 status、code 和 request ID；5xx 继续脱敏。
4. handler mapper 继续可使用分页、时间、optional scalar 和 `structpb.Value` 辅助转换，但这些转换不再归入语义为 response 的包。

## Acceptance

- [ ] `internal/api/http/binding`、`codec`、`response` 的职责与调用方收敛到 `internal/api/http/transport`；不保留并行 transport 子包或旧包兼容入口。
- [ ] 请求解码仅保留实际需要的 protobuf JSON 路径；不存在无调用方的普通 JSON fallback 或仅转发一次的同义 helper。
- [ ] protobuf JSON marshal/unmarshal options 只有一个受控 owner；常规 HTTP、SSE 和 task payload 的 JSON 语义保持一致。
- [ ] 成功 proto 输出、错误输出和 mapper 辅助转换在 transport 包内职责分明，不再混杂在 `response` 包。
- [ ] 错误 body `{code, error, requestId}`、HTTP status、401 `WWW-Authenticate`、405 `Allow`、request ID 透传或生成、5xx 脱敏、请求校验与 query 语义保持不变。
- [ ] HTTP adapter 之外仍无 `internal/gen/proto` 依赖；handler 不引入业务错误到 HTTP status 的分支表。
- [ ] 相关 Go 测试证明重组前后的 HTTP 可观察行为等价，并通过 `task check` 和 `go test ./cmd/... ./internal/...`。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard。
- 使用单一 `internal/api/http/transport` 包，不保留 `binding`、`codec`、`response` 等并行包或兼容导入路径。
- 包内按职责划分源文件，而不是为每项能力继续创建 transport 子包。
- protobuf 生成类型只在 HTTP transport adapter 使用；应用层继续使用 DTO、model 与 port。
- 保持已有 `{code, error, requestId}` 错误契约；RFC 9457 不在本次范围内。
- 以删除无用泛化和一次性转发为准，不用别名、deprecated wrapper 或新旧 API 并存维持旧 import path。

## Risk

- `WriteError` 和 `WriteStatusError` 被 handler、认证、middleware、router fallback 广泛调用；移动时必须保证所有入口仍复用一个错误 writer。
- 当前成功输出函数与 protobuf renderer 类型有同名概念；合并为一个 Go package 时需要选择职责清楚且不冲突的标识符，不能用仅转发的兼容符号规避。
- `DecodeJSONReader`、query pointer helper 和通用 mapper 的保留与删除必须以真实调用方为依据，不能因重组而改变空值或默认值语义。
- 本任务是包结构收敛，现有日志中的 `uri` 字段等与响应契约无关的观察项不在范围内。

## User review notes

- 2026-09-15：用户要求以标准模式记录本任务。
- 2026-09-15：用户明确本任务仅整理已有包结构与消除冗余，不重新实现或改变响应格式、校验规则及其他既有行为。
- 2026-09-15：用户选择单一 `internal/api/http/transport` 包。
- 2026-09-16：用户要求推进工作；本 Intent 已接受。
- 2026-09-16：用户要求开始 Plan；本 Intent 作为计划依据继续有效。
