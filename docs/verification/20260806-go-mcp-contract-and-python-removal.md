# Go MCP 契约补齐与 Python MCP 移除验证
最后修改时间: 2026-08-06 14:08:25

Review status: Draft

流程模式: 标准 / standard

## 验证依据

依据已接受的 [Requirement](../requirement/20260806-go-mcp-contract-and-python-removal.md) 与 [Plan](../plan/20260806-go-mcp-contract-and-python-removal.md) 验证。本次以 Go delivery MCP、浏览器授权交接、对话编排和活文档更新为范围；用户已澄清，除 Python MCP 目录本身外，所有 `scripts/` 必须保留。

## 实际差异摘要

- `cmd/server mcp` 复用 Go bootstrap 与 delivery Core；本地 token 无效时通过浏览器授权、loopback callback 和单次 grant 取得凭据。
- auth application 与 HTTP 路由新增 MCP grant 签发、交换和浏览器授权页面；登录及 Google 回调恢复原始 redirect。
- delivery MCP 增加 `controlled_file`、端点 mode 和 runtime doctor 的模型可发现说明；服务端测试保持 48 个工具。
- dialogue usecase 增加每回合执行状态，覆盖更新后的回读、每个 Service 单次 deploy、deployment ID 保留和失败时的部分结果。
- `.codex/config.toml`、`AGENTS.md`、活指南和 RAGFlow inventory 统一为 Go stdio `go run ./cmd/server mcp` 与浏览器认证流程。
- 工作区中已不存在 `mcp/`，活动配置、指南和 skills 中也不存在 Python MCP 启动命令或目录引用。

## 计划对齐

| 计划范围 | 实际结果 |
| --- | --- |
| Go stdio MCP 与本地认证基础设施 | 已实现，相关 Go unit test 通过。 |
| 单次授权码与 Web 授权页 | 已实现，grant、callback、token store 和 redirect 有定向测试或类型检查覆盖。 |
| MCP schema 与工具说明 | 已实现，`controlled_file`、端点 mode、runtime doctor 和 48 tool registry 有 server test 覆盖。 |
| 对话纠偏与单次部署 | 已实现，dialogue usecase 测试覆盖部署 guard 与部分执行结果。 |
| Python MCP 与活文档清理 | `mcp/` 已不存在，活引用扫描通过。 |
| 其他 scripts 清理 | 不适用。用户已澄清该范围不属于本任务；已恢复此前误删的 scripts、测试和文档。 |

## 验收清单

- [x] 工作区不含 `mcp/`，没有 Python MCP 兼容入口。
- [x] Go delivery MCP 的 48 个工具保留，且继续走 application usecase 边界。
- [x] `controlled_file` 与端点 mode 的契约及现有 Go input 映射有定向测试覆盖。
- [x] 组件相关离散值与组合参数通过 MCP schema/description 向模型暴露。
- [x] 对话纠偏、回读、单次 deploy guard 和部分失败结果有 usecase 测试覆盖。
- [x] `.codex/config.toml`、`AGENTS.md`、活指南和 skills 不含旧 Python MCP 引用。
- [x] MCP schema、认证、对话编排和引用清理具备自动化检查。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `task check` | 通过：Web typecheck、eslint、prettier、golangci-lint format 与 lint 均成功；formatter 未修改文件。 |
| `task test` | 通过：Vitest 10 个文件、62 项测试通过；Go `./cmd/... ./internal/... ./sql` 通过。 |
| `go test -count=1 ./internal/api/mcp/delivery ./internal/application/auth/usecase ./internal/application/dialogue/usecase ./internal/infrastructure/mcp ./internal/bootstrap` | 通过。 |
| `python -m unittest discover -s scripts -p 'test_*.py'` | 通过：19 项测试通过。 |
| Python MCP 目录与活引用扫描 | 通过：`mcp/` 不存在，`.codex`、`AGENTS.md`、`docs/guides/` 与 `skills/` 未命中旧 Python MCP 入口。 |

## 范围与风险

- 两份未跟踪的 `scripts/migrate-deployment-snapshot-refs.*.sql` 是用户已有迁移文件，不属于本任务，未修改。
- `task check` 和 `task test` 都通过；无格式化残留。
- 未执行真实浏览器登录、loopback callback、stdio MCP 重启复用凭据及 token 失效再授权的端到端人工验收，避免使用或创建真实用户凭据。该路径已有自动化单元测试，但仍应在受控登录环境中手工确认 stdout 不含日志或 token。

## 结论

自动化验证通过，实际实现与已澄清后的范围一致。浏览器认证与真实 stdio 会话的人工验收尚待完成，因此本验证记录保持 Draft。
