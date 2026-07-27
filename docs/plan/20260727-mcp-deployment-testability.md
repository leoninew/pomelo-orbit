# MCP 直接部署测试可用性实施计划
最后修改时间: 2026-07-27 17:11:07

Review status: Accepted

## Basis

- Requirement: `docs/requirement/20260727-mcp-deployment-testability.md` (`Accepted`)
- Spec: `docs/spec/20260727-mcp-deployment-testability.md` (`Accepted`)
- Flow mode: `standard` / 标准模式。
- 本计划只实现已接受规格，不新增部署脚本、初始化器、journal、`cleanup` 工具或 `data/` 目录行为。

## Implementation steps

1. 修正 Go 事务中间件的错误响应时序。
   - 修改 `internal/infrastructure/database/tx/request.go`：将事务限定于 POST、PUT、PATCH、DELETE API 请求；读请求、OPTIONS、health 和静态响应直接透传。
   - 为事务写请求引入可丢弃的 Gin response buffer。handler 成功时先 commit，再将 2xx headers/body 写入原始 writer；commit 失败时清空 buffer，通过 `transportresponse.WriteError` 返回统一 500。
   - 将 `BeginTx` 失败改为 `apperror.KindInternal` 加 `WriteError`。handler panic 时丢弃 buffer、回滚、恢复原 writer 后重新抛出，由既有 Recovery middleware 输出统一错误。
   - 保留 handler 已产生的 4xx/5xx；这些路径回滚后原样 flush。rollback 失败只记录服务器错误，不生成第二个 HTTP response。
   - 扩展 `request_test.go`。引入 test-only SQL mock 依赖到 `go.mod`/`go.sum`，稳定模拟 Begin/Commit 失败，而不依赖真实数据库的偶然行为。

2. 建立 MCP 的单一工具错误出口。
   - 新建 `mcp/src/pomelo_orbit_mcp/mcp_errors.py`：定义安全错误分类、可选 status/request_id 投影、文本摘要和 `CallToolResult` 构造。
   - 修改 `mcp/src/pomelo_orbit_mcp/server.py`：以项目 FastMCP 子类覆写 `call_tool`，从 FastMCP `ToolError.__cause__` 分类 validation、Orbit API、runtime 和 internal 错误；不修改 FastMCP 依赖的低层源码。
   - 仅把安全内容放入 TextContent 和 `structuredContent.error`。Docker stderr/stdout、命令、HTTP body、认证信息、runtime config、traceback 和原始未知异常不得进入错误结果。
   - 对启动前 `SettingsError`、stdio 断开及超时保持“Server unavailable”处理；它们没有 MCP session，不能伪造为 `CallToolResult`。

3. 补足 Application 到 Service 的 MCP 查询接口。
   - 修改 `mcp/src/pomelo_orbit_mcp/tools/orbit.py`，新增 `orbit_list_application_services(application_id)`。
   - 调用已有 `OrbitClient.list_application_services`，为每个 Service 显式投影允许字段：id、application_id、instance_key、status、version_id、last_successful_version_id、created_at、updated_at。
   - 修改 `mcp/tests/test_server.py` 并新增或扩展工具测试，锁定工具发现、input schema、响应形状以及 runtime config 不泄露。

4. 纠正 verification 的错误与领域结论边界。
   - 修改 `mcp/src/pomelo_orbit_mcp/verification.py`：Orbit API、Docker 和 runtime target 异常向上抛出，由统一 MCP 错误出口处理；删除把它们写入 `inconclusive` evidence/differences 的分支。
   - 保持 deployment `faulted`、`canceled`、`timed_out` 与完成验证后的 `failed`、`drift`、`inconclusive` 为 `isError=false` 的正常领域结果。
   - 更新 `mcp/tests/test_verification.py` 和新增协议测试，覆盖该边界与错误内容脱敏。

5. 增加项目 manifest 与直接操作文档。
   - 新增 `.codex/config.toml`，使用已接受的 `uv --directory mcp run pomelo-orbit-mcp` stdio 配置、项目相对 cwd 和启动/调用超时；不写入任何环境变量或秘密。
   - 更新 `mcp/README.md`，说明 trusted project、`mcp/.env` 前置条件、重新打开会话和 Server unavailable 的处理。
   - 新增 `docs/guides/` 下的直接 MCP 操作指南。按用户显式请求列出单个工具，不定义 deploy/wait/verify/stop/delete 的隐式链路。

## Verification plan

1. Go 单元测试：覆盖 begin 失败、commit 失败、成功提交、4xx rollback 和 panic recovery；断言 `{code,error,requestId}`、无成功 body 泄露及正确事务终态。
2. MCP 单元和 stdio 协议测试：通过真实 `ClientSession` 断言 validation、Orbit API、runtime、internal 均返回 `isError=true`、安全 TextContent 和同形 `structuredContent.error`。
3. 脱敏测试：将 Docker stderr、认证字样和未知异常细节注入失败路径，断言它们不出现在任何 MCP error content 中。
4. 领域结论测试：断言 wait 的 terminal/timeout 和 verification 的比较结论保持 `isError=false`；其依赖 API/Docker/runtime 失败则为 `isError=true`。
5. Service 列表测试：fixture 中含 runtime config 和额外字段，断言响应只包含允许字段，且可直接取得 `orbit_stop` 所需的 `service_id`。
6. 运行 `make -C mcp check`，再运行项目 Go 测试入口 `task test`；Docker 集成测试不作为默认检查运行。
7. 在新的受信任 Codex 会话中验收 manifest：确认 `pomelo_orbit` 工具被发现，且 Server 启动失败不会导致调用方继续依赖操作。该验收不执行 Docker 部署。

## Files to change

| File or directory | Planned change |
| --- | --- |
| `internal/infrastructure/database/tx/request.go` | 事务范围、response buffer、统一 begin/commit/panic 错误处理。 |
| `internal/infrastructure/database/tx/request_test.go` | 事务失败和 HTTP response 契约测试。 |
| `go.mod`, `go.sum` | test-only SQL mock。 |
| `mcp/src/pomelo_orbit_mcp/mcp_errors.py` | 新增共享 MCP error mapper。 |
| `mcp/src/pomelo_orbit_mcp/server.py` | 挂接统一 `call_tool` 边界。 |
| `mcp/src/pomelo_orbit_mcp/tools/orbit.py` | Service 列表工具和字段投影。 |
| `mcp/src/pomelo_orbit_mcp/verification.py` | 异常传播与领域结论分离。 |
| `mcp/tests/` | Service、error mapper、stdio protocol、verification 测试。 |
| `.codex/config.toml` | 新增受版本控制的 MCP manifest。 |
| `mcp/README.md`, `docs/guides/` | 注册前置条件与直接操作说明。 |

## Rollback

1. manifest 发现或启动不兼容时，只移除 `.codex/config.toml`；不影响 Orbit API、Docker 或用户级配置。
2. MCP 映射出现兼容问题时，回退 `mcp_errors.py`、Server 接入和对应测试，不改变 Orbit HTTP 错误契约。
3. response buffer 出现不兼容时，回退事务中间件与 SQL mock 测试；不要以保留错误 2xx 的方式绕过 commit 失败。
4. 本计划没有创建、停止、删除 Docker Application 或操作 `data/`，因此无需资源回滚。

## Assumptions and risks

- `mcp` 1.28.1 的 `CallToolResult` 支持已由 stdio 原型确认；升级依赖必须重新运行协议测试。
- Codex CLI 0.145.0 接受所需 manifest 字段，但新配置只能由新受信任会话加载并验收。
- 现有项目没有流式 API；事务 buffer 只用于写 API，避免影响静态 `ServeFile` 和未来流式路径。
- MCP Server 的本地配置、Docker context 和 data root 由操作者已有配置提供。本功能不创建或管理这些资源。

## User review notes

用户已接受规格并要求进入 Plan。实现开始前无待决产品选择；Plan 本身等待用户审查。
