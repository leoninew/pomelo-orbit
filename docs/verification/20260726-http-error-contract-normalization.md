# HTTP 错误契约规范化验证
最后修改时间: 2026-07-29 21:28:10

Review status: Accepted

Flow mode: standard

## 需求对齐

- 失败响应统一为 `code`、`error`、`requestId`；HTTP status 仅在状态行表达。
- 未分类错误固定为 `500/internal_error`，内部 cause 仅进入日志；已确认 Traefik 连接错误改为安全的 `503/service_unavailable` 摘要。
- 新请求 ID 使用 ULID，入站 ID、响应 header、错误 body 与日志使用同一值；**Web client 不生成 request id**（服务端生成）。
- 应用负责的 NoRoute、NoMethod、recovery、认证、授权和 CORS 拒绝使用统一 writer。401 返回 `WWW-Authenticate: Bearer`；405 返回 `Allow`。
- Web 与 MCP client 严格校验新契约，保留 status、code、request ID；旧 body、空 body 和非 JSON body 进入 contract mismatch。
- 可复用实践已沉淀到独立 best-practices 仓库（含后续 2026-07-29 文档/skill 收口）。

## 计划对齐

- 已完成应用错误分类、ULID request ID、集中 writer、handler/security/fallback 迁移、Web/MCP 调用方迁移和测试覆盖。
- 实现阶段的独立前测发现并修复了 401 challenge、405 `Allow`、Traefik cause 泄露风险和 Web 网络失败测试缺口。
- 未修改并行的 environment、proto、SQL、部署能力和 MCP runtime credential 改动。

## 实际差异摘要

- Go HTTP 边界：统一 writer（`internal/api/http/response`）、应用错误映射、request ID middleware、fallback/recovery 和全部 handler 错误出口。
- 调用方：Web `ApiError` 与 MCP `OrbitAPIError` 解析稳定错误契约。
- 文档：更新 requirement、plan、HTTP 错误契约指南与操作手册；新增通用检查 skill。
- 验证补充：401 `WWW-Authenticate`、405 `Allow`、安全的 Traefik unavailable 映射及 Axios 网络失败测试。
- **2026-07-29 精修**：`apperror.KindRateLimited`（429 能力预留）；各 handler `writeServiceError` 统一为转发 `WriteError`（不再分叉日志签名）；best-practices 明确 request id 生成 owner 与 Go writer 包布局。

## 预期与实际文件

计划内的 HTTP、Web、MCP、Go module 和过程文档均已变更。实际额外变更含 best-practices 指南与 skill，以及验证过程中补充的 route usecase 安全映射和测试。environment 相关文件保持未暂存，未归入本任务。

## 验收清单

- [x] 纳入范围的 HTTP 错误使用 `code/error/requestId`。
- [x] 覆盖 validation、401、403、404、405、409、503 和 500 默认映射；`rate_limited` Kind 已具备（产品无限流路径时 429 用例 N/A）。
- [x] 500 不向调用方泄露 cause；日志可按 request ID 检索。
- [x] request ID 透传或生成 ULID，header/body/log 一致；Web 默认不造 id。
- [x] Web 和 MCP 严格归一化错误契约；Web 保持 401 边界。
- [x] API fallback、method mismatch 和 recovery 使用统一契约；401 和 405 保留协议 header。
- [x] 本 feature 定向测试覆盖已运行并通过。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/common/errors ./internal/api/http/handler/... ./internal/api/http/response`（2026-07-29 精修） | PASS（见本轮） |
| `go test ./cmd/... ./internal/... ./sql`（原验收） | PASS |
| `go vet ./cmd/... ./internal/... ./sql`（原验收） | PASS |
| 全仓 golangci-lint / web typecheck / web format | **不作为本 feature 门禁**：失败文件属 environment/version 并行改动 |
| `yarn --cwd web test`（原验收） | PASS |
| MCP pytest / ruff / mypy（原验收） | PASS |

## 范围偏差

无功能范围偏差。后续精修仅统一 handler 错误转发与 Kind 表，不改变外部契约字段。

## 风险与未完成项

- 全仓 lint/typecheck/format 仍可能因**并行 environment 改动**失败；与本 HTTP 错误契约 feature 解耦，不阻塞本 verification Accepted。
- 历史 validation public message 文案仍可按业务逐条收紧（不改变分类机制）。

## 结论

**HTTP 错误契约 feature：Accepted。**  
实现、定向验证与 2026-07-29 与 best-practices 对齐的精修通过。全仓无关门禁失败不纳入本 feature 未完成项。
