# HTTP Transport 边界收敛验证
最后修改时间: 2026-09-16 10:46:09

Review status: Draft

Mode: standard

## Intent alignment

本次实现将原 `internal/api/http/binding`、`codec`、`response` 三个包收敛到唯一的 `internal/api/http/transport`。HTTP handler、server、router、security、middleware、SSE、task payload 与测试均迁移到新包；旧目录已不存在，源码没有旧 import。

`protojson.MarshalOptions` 和 `protojson.UnmarshalOptions` 仅在 `transport/protojson.go` 持有。application、model、repository 与 infrastructure 未直接导入 `internal/gen/proto`。`transport.WriteError` 是生产 HTTP 代码中唯一的 `AbortWithStatusJSON` owner。

`QueryProjectId` 是迁移前 `binding` 已有的空字符串到 `nil` 语义。本任务保持 HTTP 可观察行为；Project 作用域从可选过滤改为显式租户参数已拆分到 `20260916-project-tenant-scope`，不在本验证范围内。

## Plan alignment

1. transport 已按 `request.go`、`protojson.go`、`output.go`、`error.go` 和 `mapper.go` 的职责建立；请求体仅走 protobuf JSON 解码。
2. 所有 HTTP adapter 调用方已改用 transport，`binding`、`codec`、`response` 已删除，未保留兼容入口。
3. `tx.Middleware` 接收注入的 `ErrorWriter`，不导入 HTTP transport；bootstrap 注入 `transport.WriteError`，事务缓冲、rollback 和 commit 失败处理保留。
4. proto JSON、成功输出、错误契约、CORS、server fallback、logging 以及事务测试均已迁移或补足。

## Actual diff

| 范围 | 实际变更 |
|------|----------|
| HTTP transport | 新增唯一 `internal/api/http/transport` 包及其测试，统一请求、proto JSON、成功输出、错误与 mapper 能力。 |
| 旧包 | 删除 `binding`、`codec`、`response`，没有兼容 import 或包装层。 |
| 调用方 | 迁移 HTTP handler、router、middleware、security、server、SSE 和 task payload 的引用。 |
| 事务边界 | 由 bootstrap 注入 error writer，解除 infrastructure 对 HTTP response 的反向依赖。 |
| 测试诊断 | 修正 `transport/request_test.go` 的 protobuf message 按值格式化，避免复制内部 mutex。 |
| 质量门收尾 | Prettier 格式化 `web/src/api/project/initialization.ts` 与 `web/src/views/project/ProjectInitializationPage.vue`；不改变前端逻辑。 |
| 过程边界 | Plan 记录保留既有 `QueryProjectId` 语义；新的 Project 租户任务不属于本次实现。 |

与 Plan 的预期范围一致。工作区还包含独立的 Project 租户 Intent、其索引登记及用户已有的文档改动；这些不属于本任务的代码 diff。

## Acceptance

| 验收项 | 结果 | 证据 |
|--------|------|------|
| 旧 HTTP 包收敛为唯一 transport | 通过 | 旧目录不存在，旧 import 扫描无命中。 |
| 请求解码仅保留 protobuf JSON 路径 | 通过 | `DecodeJSON` 仅接收 `proto.Message`；unknown field 测试通过。 |
| proto JSON options 只有一个 owner | 通过 | 全仓扫描仅命中 `transport/protojson.go`。 |
| 成功、错误输出和 mapper 职责分离 | 通过 | transport 按职责源文件拆分，输出与错误测试通过。 |
| HTTP 错误与 header 契约保持 | 通过 | 覆盖 400、401 `WWW-Authenticate`、404、405 `Allow`、CORS 拒绝、request ID 与 5xx 脱敏的测试通过。 |
| HTTP adapter 外无生成 proto 依赖 | 通过 | application/model/repository/infrastructure import 扫描无命中。 |
| 相关 Go 测试与完整 Go 测试 | 通过 | focused tests 与 `go test ./cmd/... ./internal/...` 通过。 |
| `task check` | 通过 | Web typecheck、ESLint、Prettier 与 Go config、format、lint 均通过。 |

## HTTP Error Contract

- PASS A1/B1/B4/C1/C1b/D1：错误 body、status 映射、脱敏和统一 writer 由 `transport.WriteError` / `WriteStatusError` 集中处理。
- PASS B3：`server_test.go` 覆盖 405 和 `Allow`；PASS B1：401 bearer challenge、404 和 CORS 403 均有覆盖。
- PASS C2/C3：server fallback、recovery、auth、CORS 与事务失败均调用统一 writer，允许的 CORS preflight 仍由 CORS middleware 返回 204。
- PASS D2/D3：logging 测试覆盖输入 request ID 透传和服务端生成；错误 body 与 header 复用同一 ID。
- N/A E1-E3：前端错误客户端未在本任务中修改。

## Test results

| 命令 | 结果 |
|------|------|
| `go test ./internal/api/http/... ./internal/infrastructure/database/tx` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `./bin/golangci-lint config verify` | 通过 |
| `./bin/golangci-lint fmt ./cmd/... ./internal/... ./sql` | 通过 |
| `./bin/golangci-lint run ./cmd/... ./internal/... ./sql` | 通过，`0 issues` |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web format:fix` | 通过；仅格式化 `src/api/project/initialization.ts` 与 `src/views/project/ProjectInitializationPage.vue`。 |
| `task check` | 通过：Web typecheck、ESLint、Prettier 与 Go config、format、lint 均通过，Go lint 为 `0 issues`。 |
| `git diff --check` | 通过 |

## Risks and incomplete items

- Project scope 的可选 query 语义刻意保留以维持 transport 行为不变；将其收紧为显式租户作用域依赖独立 Intent 的后续规格和实现。

## Conclusion

HTTP transport 收敛的代码、依赖边界、错误契约、完整 Go 测试矩阵和 `task check` 均已验证通过。本记录保持 Draft，等待用户审阅该验证结论。
