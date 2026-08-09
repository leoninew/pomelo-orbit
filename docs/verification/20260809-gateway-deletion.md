# 网关删除清理验证
最后修改时间: 2026-08-09 13:29:57

Flow mode: light

Review status: Accepted

## 需求对齐

- 已实现网关删除时自动清理停止的 Service、托管 Version、GatewayConfig 和内部 Application。
- 已实现非停止 Service 的前置检查；错误明确包含 Service code，并返回验证错误。
- 已移除 Gateway HTTP handler 与 Web API 中的 `remove_dir`，Gateway 用例不再依赖 Workspace。
- 删除事务只执行正常领域关系下的固定清理步骤，不增加异常引用分析、专门冲突分支或数据库约束兜底。

## 规格与计划对齐

不适用。light 模式依据已接受的 Requirement 实现，未创建独立 Spec 或 Plan。

## 实际差异摘要

- Gateway usecase 增加清晰的非停止 Service 提示，并调用专用 `DeleteGatewayApplication`。
- Application repository 在单一事务中按 Service、Version、GatewayConfig、Application 顺序清理网关部署载体；新增显式 GatewayConfig 删除查询及 sqlc 生成物。
- Gateway 删除 API 和 Web 客户端不再声明或发送 `remove_dir`；bootstrap 移除 Gateway 的 Workspace 注入。
- 新增 Gateway usecase 和 repository 测试，覆盖停止 Service 删除、running/faulted Service 拒绝，以及 GatewayConfig 的显式删除。

## 预期与实际文件

| 预期范围 | 实际文件 | 结果 |
| --- | --- | --- |
| Gateway 删除语义与 HTTP 参数 | `internal/api/http/handler/gateway/`、`internal/application/gateway/`、`internal/bootstrap/`、`web/src/api/gateway/` | 一致 |
| 原子清理与 sqlc 查询 | `internal/repository/`、`internal/repository/impl/sqlc/application/`、`sql/query/application/`、`internal/gen/sqlc/application/` | 一致 |
| 自动化测试 | `internal/application/gateway/usecase/delete_test.go`、`internal/repository/impl/sqlc/application/repository_test.go` | 一致 |
| 过程与后续记录 | `docs/requirement/`、`docs/verification/`、`docs/decisions/ledger.md` | 一致；Backlog 为用户明确要求的额外记录 |

## 验收清单

- [x] 停止 Service 的网关删除会清理内部部署载体及 GatewayConfig。
- [x] running 或 faulted Service 会阻止删除，并提示需先停止的 Service。
- [x] Gateway 删除不读取或发送 `remove_dir`，也不调用 Workspace。
- [x] 通用 Application 删除语义未改变。
- [x] sqlc 生成物与查询源一致。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `./bin/sqlc generate` | PASS |
| `go fmt ./cmd/... ./internal/...` | PASS |
| `go vet ./cmd/... ./internal/...` | PASS |
| `go test ./cmd/... ./internal/...` | PASS |
| `yarn --cwd web lint:fix` | PASS |
| `yarn --cwd web typecheck` | PASS |
| `git diff HEAD --check` | PASS |

## 范围偏差

无功能范围偏差。`docs/decisions/ledger.md` 新增的删除路径防御规则整理 Backlog 由用户在验证前明确要求，用于后续独立处理。

## 风险与未完成项

- 已登记的其他删除路径防御逻辑不在本次范围，后续按 Backlog 独立审视。
- 未启动开发服务器；验证以单元测试、全量 Go 测试和 Web 静态检查为准。

## 结论

本次网关删除清理满足已接受 Requirement，验证通过。
