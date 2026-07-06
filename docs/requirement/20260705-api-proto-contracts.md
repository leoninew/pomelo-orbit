# API 请求响应 Proto 契约化需求
最后修改时间: 2026-07-05 23:15:38

## Review status

Accepted

## Background

当前后端 HTTP API 的请求体和响应体主要由 Go handler 包内的 `*Req` / `*Resp` struct 定义，前端在 `web/src/types` 中维护对应 TypeScript interface。两端契约存在人工同步成本，字段名、可选性和列表包装等约束容易漂移。参考项目 `D:\SourceCodes\mywork\TermBridge-go` 已使用 `buf` + proto 文件生成 Go 与 TypeScript 类型。

## Goal

- 为 Pomelo Orbit 的 API 请求体和响应体引入 proto 定义与生成链路。
- 将现有 API 请求中的 `Req` 与响应中的 `Resp` 类型切换为 proto 生成类型。
- 前端 API 类型优先引用 proto 生成的 TypeScript 类型，减少手写契约重复。
- 保持现有 HTTP JSON 传输、路由、状态码、权限、业务校验和响应包装语义不变。

## Non-goal

- 不把路由、HTTP method、path 参数、query 参数、header、cookie、鉴权规则或错误码定义进 proto。
- 不引入 gRPC、Connect、twirp 或新的 RPC runtime。
- 不修改已执行迁移文件。
- 不主动启动、停止或重启开发服务器。
- 不为了兼容保留手写 DTO 与生成 DTO 的双轨契约；项目处于活跃开发期，直接切换。

## User scenarios

- 后端开发者修改 API 请求或响应字段时，先修改 proto，再生成 Go/TypeScript 类型。
- 前端开发者在 `web/src/api` 与页面类型中直接复用生成类型，避免手工同步字段结构。
- 后续 review 可以通过 proto 文件集中检查请求体/响应体契约。

## Acceptance

- 仓库根目录具备 proto 生成工具链配置，参考 TermBridge 的 `buf.yaml` / `buf.gen.yaml` / `web/buf.gen.yaml` 主流实践。
- 只在 proto 中定义请求体和响应体 message；不定义路由、query/path/header、错误响应或 RPC service。
- 现有后端 `*Req` / `*Resp` DTO 类型迁移为 generated Go 类型或其别名，handler 编解码仍输出现有 snake_case JSON 字段。
- 前端请求/响应类型迁移为 generated TypeScript 类型，现有调用代码类型检查通过。
- 必要的生成产物或生成入口纳入仓库，保证本地检查可复现。
- 运行项目约定检查：Go `fmt` / `vet` / `test`；若修改前端则运行 `yarn --cwd web lint:fix` 和 `yarn --cwd web typecheck`。

## Open questions

- 是否未来要把 path/query/header 也纳入 OpenAPI 或 proto 注解：本次不做，按用户要求待议。
- 是否引入 Buf managed mode、breaking check CI、远程插件 pinning 策略：本次采用参考项目的最小可用配置，后续按 CI 需求扩展。

## Decisions

- 流程模式采用严格模式 / strict，因为该变更跨后端、前端、构建工具链和 API 契约。
- 用户已明确“直接实现”，因此本需求文档标记为 `Accepted`，并继续进入规格、计划与实现。
- proto package 按业务域拆分，生成目录与参考项目保持相近结构；JSON 字段名保持 snake_case。

## Risk

- proto3 scalar 字段无法原生表达“未传”和“传空值”的差异；对 PATCH/Update 请求中的可选字段必须使用 wrapper 或 optional，并确认 JSON 解码能保留 presence。
- `any` / `json.RawMessage` 类字段需要使用 protobuf `Value`、`Struct` 或字符串承载，避免破坏现有动态变量语义。
- ts-proto 生成类型对 optional、wrapper、map、repeated 的表现会影响前端赋值方式，需要通过 typecheck 验证。
- 大范围契约替换可能暴露历史前后端字段漂移，需要以现有测试和类型检查为准收敛。

## User review notes

- 用户原始要求：参考 `D:\SourceCodes\mywork\TermBridge-go` 引入 proto 工具链，将 API 请求里的 Req/Resp 全部使用 proto 定义和生成；只定义请求体和响应体，其他待议事项参考主流实践；直接实现。
