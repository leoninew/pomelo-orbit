# MCP 按需授权与开发配置继承
最后修改时间: 2026-08-06 21:03:18

Review status: Accepted

流程模式: 轻量 / light

## Background

`pomelo_delivery` 作为 Codex stdio MCP 在初始化时即启动。当前 `RunMCP` 在 Server 创建前同步验证本地凭据，并在令牌缺失或失效时立即打开浏览器授权页；仅进行工具发现也会产生登录副作用。

同时，Go 程序统一通过 `internal/config.Load` 读取 typed `Config`。本地开发环境由进程环境变量 `POMELO_ORBIT_APP__ENV=development` 选择 `.env.development`，但 Codex MCP 入口直接运行 `go run ./cmd/server mcp`，未继承该选择变量，导致 `mcp.web_url` 回退到基线配置。

## Goal

1. stdio MCP 的 `initialize` 与 `tools/list` 不打开浏览器、不读取或验证用户凭据。
2. 首次真实 `tools/call` 才验证本地令牌；令牌缺失或失效时再执行现有浏览器授权，并将成功用户绑定到该 stdio 会话。
3. Codex 的 `pomelo_delivery` 启动环境明确选择 `development`，让 `mcp.api_url`、`mcp.web_url` 与 `mcp.auth_timeout` 继续通过现有 typed `Config.MCP` 读取 `.env.development`。

## Non-goal

- 不改变 HTTP `/mcp` 的 Bearer 认证、领域 usecase、数据库 schema 或浏览器授权协议。
- 不增加第二套配置 loader、MCP 配置格式或认证凭据存储。
- 不启动、停止或重启开发服务器。

## User scenarios

1. Codex 打开仓库并初始化 `pomelo_delivery` 时，MCP 工具可被发现但不会打开 Orbit 页面。
2. 用户首次实际调用任一持续部署或运行态工具时，缓存令牌可用则直接执行；不可用时才打开配置的 Orbit Web UI 完成授权。
3. 本地开发的 Codex MCP 使用 `.env.development` 中配置的 Orbit URL，而不是 `configs/config.yaml` 的通用基线。

## Acceptance

- `tools/list` 前后授权器均未被调用。
- 首次 `tools/call` 恰好完成一次认证，并将返回的用户 ID 传入 application usecase；同一会话的后续调用不重复授权。
- 认证失败以 MCP tool error 返回，且不执行业务 usecase。
- `.codex/config.toml` 为 MCP 子进程设置 `POMELO_ORBIT_APP__ENV=development`；MCP URL 仍由 `Config.MCP` 提供。
- 为按需授权和 actor 绑定增加自动化测试，并保留现有 HTTP MCP 的静态 actor 行为。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 用户要求以轻量模式记录并直接实现，本 Requirement 视为已接受。
2. 认证发生在 MCP 接收 `tools/call` 的统一 middleware，而不散落到 48 个 tool handler。
3. `POMELO_ORBIT_APP__ENV` 是启动时配置选择器；Codex MCP 配置只负责注入该选择器，不承载 `mcp.*` 业务配置值。
4. 活 MCP 操作指南以首次实际工具调用授权为准；`AGENTS.md` 已有的“首次使用”表述无需修改。

## Risk

- 浏览器授权可能被取消或超时；该次 tool call 必须失败且不得产生业务副作用，后续调用允许重新发起授权。
- stdio 会话可并发发起 tool call；actor 绑定必须只执行一次并对全部调用一致可见。
