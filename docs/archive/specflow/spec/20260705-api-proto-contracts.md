# API 请求响应 Proto 契约化规格
最后修改时间: 2026-07-06 17:07:25

## Review status

Accepted

## Requirement basis

基于 `docs/requirement/20260705-api-proto-contracts.md`。用户已要求直接实现，Requirement 已标记为 `Accepted`。

## Overview

引入以 `buf` 为入口的 proto 工具链：`proto/orbit/api/v1/` 保存 API 请求体/响应体 message，后端生成 Go 类型，前端生成 TypeScript 类型。HTTP transport 继续使用 JSON 编解码，proto 只作为 schema 与类型生成来源，不定义 service、route 或 transport 细节。

## Design decisions

1. **只定义 body DTO**：proto 文件只包含现有 `*Req` / `*Resp`、列表/分页响应体和必要的嵌套 message；不包含 route、method、query、path、header、cookie 或错误 envelope。
2. **JSON 字段名保持 snake_case**：proto 字段使用 snake_case；Go JSON 编解码应使用 proto 生成字段上的 json tag 或兼容配置，前端 ts-proto 使用 `snakeToCamel=false`。
3. **Go 生成目录**：Go 生成产物统一放入 `internal/gen/...`；service/repository 不依赖 API DTO，HTTP transport 直接 import 生成类型，不额外 re-export。
4. **前端生成目录**：TypeScript 生成产物放入 `web/src/gen`，前端 API 和页面使用点直接从生成文件 import API Req/Resp 类型；`web/src/types` 仅保留 UI-only 类型、常量或工具层错误 envelope。
5. **presence 处理**：Update/PATCH 请求中的指针字段使用 wrapper 类型或 optional message 字段，确保 JSON 解码后后端仍能判断字段是否传入。
6. **动态 JSON 值处理**：变量声明中的 `default` / `value` 和 task payload 使用 `google.protobuf.Value` / `Struct` 或字符串方案，以最小破坏现有 JSON 语义为准。
7. **外层响应包装**：`ListResp`、`PaginatedResp` 属于响应体结构，可在 proto 中定义具体业务列表响应 message；错误响应不纳入本次范围。

## Affected components

- 根目录工具链配置：`buf.yaml`、`buf.gen.yaml`、`web/buf.gen.yaml`、必要的生成脚本。
- Proto schema：`proto/orbit/api/v1/*.proto`，按业务域拆分，文件名使用领域名，不额外添加 `ci_` / `cd_` 前缀。
- 后端生成 DTO：`internal/gen/orbit/api/v1/*.pb.go`。
- 后端 handler：请求 decode、响应 encode 和 DTO 构造处。
- 前端生成类型：`web/src/gen/**`。
- 前端类型与 API 调用：`web/src/types/**`、`web/src/api/**`，必要时页面组件引用类型。

## Interfaces

### Proto

- package 示例：`orbit.api.v1`。
- Go package 示例：`backend/internal/gen/orbit/api/v1;apiv1`。
- message 命名沿用现有 `LoginReq`、`TokenResp` 等，便于迁移和 review。

### Go

- handler 包直接引用 `apiv1.<Message>` generated DTO；不通过 handler 局部 type alias / re-export 间接暴露 API DTO。
- JSON 编解码仍通过标准 HTTP helper；如 wrapper/presence 与标准 `encoding/json` 不兼容，则增加局部转换/解码辅助，不能改变 HTTP 对外契约。

### TypeScript

- ts-proto 只生成 types：`onlyTypes=true`，不生成 encode/decode/client/json methods。
- `snakeToCamel=false`，保证字段名与现有 API JSON 一致。
- `web/src/types` 不 re-export generated API DTO；前端 API 和页面直接从 `web/src/gen/orbit/api/v1/<domain>` import generated types。

## Technical questions

- 本地是否已有 `buf` 和 `protoc-gen-go`：实现阶段先探测；若缺失，优先通过项目脚本或 `go install` / package devDependency 建立可复现生成入口，并如实记录未能生成的原因。
- ts-proto 插件采用远程插件还是本地 devDependency：参考 TermBridge 使用 remote plugin，减少前端依赖膨胀；如果环境不支持则改用本地 devDependency。

## Risks

- 大范围 DTO 替换可能导致 handler 构造代码编译错误较多，需要按域逐步收敛。
- wrapper 类型与前端普通 `string | undefined` 的类型不一致时，需要写清晰转换，而不是隐藏兼容层。
- proto 生成字段名与既有 Go 命名可能不同，需避免业务代码中混用旧字段名。

## Alternatives

- OpenAPI 生成 TypeScript：能覆盖 HTTP route，但用户明确要求参考 proto 工具链，且本次只定义 body。
- 手写 JSON Schema：不能直接生成 Go DTO，无法解决双端同步问题。
- 全量 gRPC/Connect：超出本次目标，会改变 transport 架构。

## User review notes

- 用户已要求直接实现；本规格随前置需求一起标记为 `Accepted`。
