# HTTP 错误契约规范化验证
最后修改时间: 2026-07-26 17:43:34

Review status: Draft

Flow mode: standard

## 需求对齐

- 失败响应统一为 `code`、`error`、`requestId`；HTTP status 仅在状态行表达。
- 未分类错误固定为 `500/internal_error`，内部 cause 仅进入日志；已确认 Traefik 连接错误改为安全的 `503/service_unavailable` 摘要。
- 新请求 ID 使用 ULID，入站 ID、响应 header、错误 body 与日志使用同一值。
- 应用负责的 NoRoute、NoMethod、recovery、认证、授权和 CORS 拒绝使用统一 writer。401 返回 `WWW-Authenticate: Bearer`；405 返回 `Allow`。
- Web 与 MCP client 严格校验新契约，保留 status、code、request ID；旧 body、空 body 和非 JSON body 进入 contract mismatch。
- 新增 best-practices 文档和 `check-http-error-contract` skill 不包含本项目业务背景。

## 计划对齐

- 已完成应用错误分类、ULID request ID、集中 writer、handler/security/fallback 迁移、Web/MCP 调用方迁移和测试覆盖。
- 实现阶段的独立前测发现并修复了 401 challenge、405 `Allow`、Traefik cause 泄露风险和 Web 网络失败测试缺口。
- 未修改并行的 environment、proto、SQL、部署能力和 MCP runtime credential 改动。

## 实际差异摘要

- Go HTTP 边界：统一 writer、应用错误映射、request ID middleware、fallback/recovery 和全部 handler 错误出口。
- 调用方：Web `ApiError` 与 MCP `OrbitAPIError` 解析稳定错误契约。
- 文档：更新 requirement、plan、HTTP 错误契约指南与操作手册；新增通用检查 skill。
- 验证补充：401 `WWW-Authenticate`、405 `Allow`、安全的 Traefik unavailable 映射及 Axios 网络失败测试。

## 预期与实际文件

计划内的 HTTP、Web、MCP、Go module 和过程文档均已变更。实际额外变更仅为用户明确要求的 `best-practices` 指南与 skill，以及验证过程中补充的 route usecase 安全映射和测试。environment 相关文件保持未暂存，未归入本任务。

## 验收清单

- [x] 纳入范围的 HTTP 错误使用 `code/error/requestId`。
- [x] 覆盖 validation、401、403、404、405、409、503 和 500 默认映射。
- [x] 500 不向调用方泄露 cause；日志可按 request ID 检索。
- [x] request ID 透传或生成 ULID，header/body/log 一致。
- [x] Web 和 MCP 严格归一化错误契约；Web 保持 401 边界。
- [x] API fallback、method mismatch 和 recovery 使用统一契约；401 和 405 保留协议 header。
- [x] 后端、Web、MCP 与 skill 的定向和全量测试覆盖已运行。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `go test ./cmd/... ./internal/... ./sql` | PASS |
| `go vet ./cmd/... ./internal/... ./sql` | PASS |
| `bin/golangci-lint.exe config verify` | PASS |
| `bin/golangci-lint.exe fmt --diff ./cmd/... ./internal/... ./sql` | FAIL，范围外 3 个 environment 文件未格式化 |
| `bin/golangci-lint.exe run ./cmd/... ./internal/... ./sql` | FAIL，除上述 3 个格式问题外，另有 1 个 environment integration test 的 staticcheck 问题 |
| `yarn --cwd web test` | PASS，8 files / 43 tests |
| `yarn --cwd web lint` | PASS |
| `yarn --cwd web typecheck` | FAIL，`VersionDetail.vue` 缺少 `secret_env_refs` |
| `yarn --cwd web format` | FAIL，19 个范围外 Web 文件存在 Prettier 差异 |
| `uv run pytest` | PASS，18 passed / 1 deselected |
| `uv run ruff check src tests` | PASS |
| `uv run mypy` | PASS |
| `quick_validate.py skills/check-http-error-contract` | PASS |

## 范围偏差

无功能范围偏差。根据独立前测，补充了协议 header、安全错误映射和网络失败测试；根据用户要求，将可复用实践沉淀到独立 best-practices 仓库。

## 风险与未完成项

- 当前全量 lint、Web typecheck 和 Web format 不能作为整个工作区的通过门槛，失败文件均属于 environment/version component 并行改动，不在本任务暂存范围内。
- 对仍将 validation error 文本设为公开 message 的历史调用点已进行审计；本次修复了已确认的外部 Traefik cause。其余命中需要在后续业务需求中逐项确认是否为安全、用户可读的 validation 文案。

## 结论

HTTP 错误契约任务的实现、定向验证和回归检查通过。由于当前工作区仍存在范围外的质量门禁失败，verification 保持 `Draft`，待相关 owner 修复后可将全量仓库质量门禁作为独立验收项关闭。
