# MCP 调用认证边界重构验收记录
最后修改时间: 2026-09-07 16:04:46

Review status: Accepted

Flow mode: standard

## 需求对齐

- 对照已接受的 [MCP 调用认证边界重构需求](../requirement/20260906-mcp-standard-authentication.md) 与 [实施计划](../plan/20260906-mcp-standard-authentication.md)。本功能没有独立 Spec，按 standard 流程以 Requirement 和 Plan 为依据验收。
- Web Dialogue 的 HTTP 入站认证只将当前用户 ID 传入 Dialogue Service；Dialogue MCP factory 用该 ID 建立每 turn 独立的内存 transport session，不再传递浏览器 `Authorization` 或访问 HTTP `/mcp`。
- 本地 stdio MCP 通过 `POMELO_ORBIT_MCP__ACCESS_TOKEN` 获取 MCP PAT。`initialize` 与 `tools/list` 不触发认证；每次 `tools/call` 重新认证 PAT 并校验固定 session actor，认证失败不会执行业务工具。
- 已登录用户可在“系统管理 / 访问令牌”创建、查看和撤销命名 PAT。创建响应只返回一次明文，数据库只保存 SHA-256 摘要；PAT 支持有限有效期，撤销、过期、篡改或用户禁用均被拒绝。
- 已删除浏览器 grant、loopback callback、token file、`/mcp` Streamable HTTP transport 及其 Web 授权页。没有保留 OAuth/OIDC、远程 MCP HTTP、CLI 密码登录或凭据缓存兼容路径。

## 计划对齐

| 计划范围 | 实际结果 |
| --- | --- |
| Dialogue actor 收敛与内存 MCP transport | `MCPClientFactory.Connect` 接收 actor user ID；`internal/infrastructure/mcp/delivery` 使用 SDK `NewInMemoryTransports` 建立短会话。 |
| PAT 生命周期与持久化 | 新增 `mcp_access_token` 模型、SQLC 查询、Auth Service 用例及 SQLite/MySQL/PostgreSQL `000038` 迁移；不修改既有迁移。 |
| stdio 认证门禁 | `App.RunMCP` 从 typed config 读取 PAT，并在每次工具调用通过 `AuthenticateMCPAccessToken` 认证；首次通过后固定 actor，后续认证不得更换该 actor。 |
| Web 管理入口 | 新增受保护的单数 `/api/auth/mcp-access-token` 的创建、列表与撤销接口、Proto DTO、前端 API 与“访问令牌”页面。 |
| 废弃路径删除 | 删除 `mcp-grant` DTO/usecase/路由、旧 MCP HTTP handler、浏览器授权器与 token store，以及授权/回调页面。 |
| 配置与活文档 | `.codex/config.toml` 仅声明 `env_vars` 透传，环境样例、后端架构、调用与 MCP 指南已更新；未写入真实凭据。 |

## 实际 Diff 摘要

本验收核对以下任务相关范围：MCP bootstrap、Dialogue 适配、认证服务与仓储、认证 API/Proto、SQL schema/query/三种新增迁移、前端访问令牌管理页、配置及活文档。

- 认证层新增 `CreateMCPAccessToken`、`ListMCPAccessTokens`、`RevokeMCPAccessToken` 和 `AuthenticateMCPAccessToken`；认证结果仍复用用户存在性、enabled 状态及当前角色/权限装配。
- MCP Server 从首次授权缓存改为每次工具调用认证；并发调用时固定 actor 不可被后续凭据认证结果替换。
- 访问令牌页面支持创建、可选 30/90/365 天到期、列表、撤销和一次性复制。创建后显示的令牌输入不接收默认焦点，底部复制命令显示为“复制”。
- SQLC 与 Go/TypeScript Proto 生成产物已随 schema 和契约更新。

工作区还存在相邻历史 MCP 过程文档、RAGFlow 文档及 `skills/deploy-sub2api-orbit/**` 的并行改动。这些文件不在本 Requirement/Plan 的交付边界内，本记录不以其内容作为本功能验收依据，也未修改或回退它们。

## 验收清单

### Web Dialogue

- [x] HTTP 入站认证向 Dialogue Service 传递当前用户 ID，原始 Authorization 不进入其 MCP factory。
- [x] Dialogue 使用共享 MCP Core 的内存 transport，Web 与 stdio 共用 tool registry 和 application usecase。
- [x] 现有 `dialogue:write` 与项目成员检查保持在 HTTP/application 边界。
- [x] 用户确认 Web Dialogue 实测无问题。

### Codex CLI stdio

- [x] `.codex/config.toml` 通过 `env_vars` 将 `POMELO_ORBIT_MCP__ACCESS_TOKEN` 传给 `go run ./cmd/server mcp`，仓库未包含实际 token。
- [x] PAT 创建、列表、撤销、可选到期、摘要持久化和用户 enabled 校验均已实现。
- [x] discovery 无认证副作用；`tools/call` 每次认证并阻断认证失败后的业务执行。
- [x] 同一 session 固定为首次认证的 actor，认证结果不能通过工具参数或后续不同用户改变。
- [x] 用户确认 Codex stdio 实测无问题。

### 删除与契约收敛

- [x] 旧 grant、浏览器授权/回调、token file、远程 `/mcp` HTTP transport、旧前端页面和对应测试已删除。
- [x] 配置、环境样例、架构说明和 MCP 操作指南已同步为“访问令牌”与显式环境凭据流程。
- [x] 自动化覆盖真实 PAT 生命周期（篡改、撤销、过期、禁用用户）、工具发现不认证、认证失败阻断、actor 固定/并发与内存 transport。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `task check` | 通过：`vue-tsc`、ESLint、Prettier、golangci-lint 配置/格式/lint 均通过。 |
| `task test` | 通过：Vitest 19 个测试文件、92 个用例；`go test ./cmd/... ./internal/... ./sql` 通过。 |
| `go test -race ./internal/api/mcp/delivery ./internal/infrastructure/mcp/delivery` | 未完成：默认环境禁用 CGO；以 `CGO_ENABLED=1` 重试后，系统缺少 `gcc`。普通并发测试与全量 Go 测试已通过。 |
| `git diff --check` | 通过。 |

## 范围偏差

无功能范围偏差。用户在实施期间将面向用户的中文名称收敛为“访问令牌”，并调整一次性令牌窗口的焦点与复制按钮文案；这属于新增管理页面的可用性细化，不改变 PAT 或 MCP 认证边界。

## 风险与未完成项

- race 检查需要安装可用的 Windows C 编译器后补跑；当前没有将此环境限制误报为 race 测试通过。
- stdio 仍需在能访问同一 Orbit 数据库、配置、Docker 和 workspace 的可信运行环境中执行，不支持任意工作站直连远程 MCP HTTP。
- PAT 是长期敏感凭据。明文仅在创建响应展示一次；使用者仍需通过环境变量或系统秘密管理保存，并在泄露时撤销。
- 本次人工验收基于用户确认的 Codex 与 Web Dialogue 结果，未由本记录启动或控制真实服务、Worker 或现有 MCP session。

## 结论

本实现满足已接受的 MCP 认证边界重构需求和计划：Web Dialogue 已收敛为内部 actor 绑定与内存 MCP transport，Codex stdio 已改用可管理的 PAT，旧浏览器授权与 HTTP 回环路径已删除。静态检查、全量自动化测试及用户确认的 Codex/Dialogue 实测均通过；仅 race 检查受本机缺少 `gcc` 限制，需在具备 CGO 工具链的环境补充执行。
