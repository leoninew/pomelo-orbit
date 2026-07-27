# MCP 直接部署测试可用性规格
最后修改时间: 2026-07-27 18:03:59

Review status: Accepted

## Requirement basis

- Requirement: `docs/requirement/20260727-mcp-deployment-testability.md` (`Accepted`)
- Flow mode: `standard` / 标准模式；用户明确要求补充本 Spec 阶段。
- 范围只包含项目 MCP 注册、Application 到 Service 查询、工具调用异常的一致处理，以及 Go HTTP 事务错误响应的修正。它不新增脚本、journal、`cleanup` 复合动作或任何 `data/` 目录策略。

## Verified findings

1. 当前 Go HTTP 层已有可复用的统一错误契约。`internal/common/errors` 将受控错误分类为 HTTP status、稳定 code 和安全 message；`response.WriteError` 返回 `{code, error, requestId}`，内部错误不会向客户端暴露 cause。
2. `tx.Middleware` 有两条绕过或破坏该契约的路径：`BeginTx` 失败直接返回 `{"message":"failed to begin transaction"}`；`Commit` 失败发生在 handler 已写出成功响应之后，目前只记录 `c.Error`，客户端会收到错误的 2xx。
3. Go 端已存在 `GET /api/application/:app_id/service`。其返回的 `ServiceResp` 恰好包含 id、application_id、instance_key、version_id、last_successful_version_id、status 和时间字段，不含 runtime config；MCP 仍必须做显式允许字段投影，避免未来 API 扩展意外透传敏感字段。
4. 本地锁定的 MCP SDK 是 `mcp` 1.28.1。协议原型经 stdio `ClientSession` 验证：工具可返回 `CallToolResult(isError=true, content=[TextContent(...)], structuredContent={...})`，客户端完整接收 `isError`、摘要和结构化对象。
5. FastMCP 的 `Tool.run` 会把工具函数异常包装为 `ToolError`，但保留原异常为 `__cause__`。覆写项目 Server 的 `call_tool` 并在该边界分类异常，可以在低层 Server 将异常压缩成纯文本之前返回统一 `CallToolResult`。原型已验证输入校验、Orbit API、Docker 失败和正常领域结论四种路径。
6. 当前 `verify_deployment` 会捕获 Orbit/Docker/runtime 异常并把 `str(error)` 写入 `inconclusive`。这既把工具异常误报为正常结论，也可能把 Docker stderr 暴露到工具成功响应中，必须改正。
7. 当前 Codex CLI 0.145.0 的本地配置解析接受 stdio MCP 的 `command`、`args`、`cwd`、`startup_timeout_sec` 和 `tool_timeout_sec` 字段。新的 Codex 会话验证表明 `cwd` 相对工作区根目录解析，因此 manifest 使用 `cwd = "."`；当前会话不能热加载尚未创建的项目 manifest。

## Overview

本变更让 `pomelo-orbit-mcp` 作为受信任项目的直接 Codex stdio MCP Server 注册，并使调用方能够从 Application 枚举 Service 后执行用户明确要求的独立动作。

工具调用失败统一为标准 MCP `CallToolResult.isError=true`，带安全文本摘要和结构化错误对象。已经成功取得领域状态后的 deployment terminal state 与 verification conclusion 保持为正常工具结果。Go HTTP 控制面在事务开始或提交失败时也必须返回现有的统一 HTTP 错误契约，不能把未提交的写操作报告为成功。

本变更不定义测试结束流程。deploy、wait、verify、stop、delete 和 `remove_volumes` / `remove_dir` 都是独立的、由用户明确选择的动作。

## Design decisions

### Project MCP manifest

在仓库根目录新增 `.codex/config.toml`：

```toml
[mcp_servers.pomelo_orbit]
command = "uv"
args = ["--directory", "mcp", "run", "pomelo-orbit-mcp"]
cwd = "."
startup_timeout_sec = 10.0
tool_timeout_sec = 300.0
```

`cwd = "."` 指向 Codex 当前项目工作区根目录；`--directory mcp` 使入口解析到 `mcp/pyproject.toml` 中的 `pomelo-orbit-mcp`。manifest 不包含 URL、用户名、密码、JWT、`POMELO_ORBIT_*`、用户绝对路径或 Docker 参数。

前置条件是项目已受信任，`uv` 可由启动 Codex 的环境解析，且现有本地 `mcp/.env` 已配置。启动失败发生在 MCP session 建立之前，协议上不可能返回 `CallToolResult`；客户端将其按“Server unavailable”处理，禁止继续后续操作。它与已建立 session 后的工具调用异常不同。

本功能不修改现有 `Settings.load` 的 data-root 行为，也不让 manifest 设置 data root。真实运行时若需要 data root，只能使用操作者已配置、已拥有的路径；自动化测试不启动真实 Docker 或操作仓库 `data/`。

### Application Service 查询

新增 `orbit_list_application_services(application_id)`。它调用既有 `OrbitClient.list_application_services` 和既有 HTTP 路由，不新增 Go API。

响应固定为：

```json
{
  "application_id": "app-id",
  "services": [
    {
      "id": "service-id",
      "application_id": "app-id",
      "instance_key": "default",
      "status": "stopped",
      "version_id": "version-id",
      "last_successful_version_id": "version-id",
      "created_at": "2026-07-27T00:00:00Z",
      "updated_at": "2026-07-27T00:00:00Z"
    }
  ]
}
```

可选字段缺失时省略。实现使用允许字段映射，不返回 `runtime_config`、Service 全量详情或其他未来新增字段。调用方使用每项 `id` 作为 `orbit_stop`、`orbit_restart` 或 `orbit_deploy` 的 `service_id`；不再借助 runtime 工具间接取得该 ID。

### MCP tool error contract

统一错误只适用于已建立 MCP session 的 `tools/call`。项目新增一个共享 `FastMCP` 子类并覆写 `call_tool`：它调用既有 FastMCP 工具管理器，捕获 `ToolError`，检查保留的根因，然后返回 `CallToolResult`。工具函数、工具模块和业务编排不得各自拼接错误文本或构造协议结果。

所有工具调用异常使用以下对象；`status` 与 `request_id` 只在来源提供时出现：

```json
{
  "error": {
    "kind": "validation|orbit_api|runtime|internal",
    "code": "stable_machine_code",
    "message": "safe operator-facing message",
    "status": 400,
    "request_id": "optional-request-id"
  }
}
```

`CallToolResult` 的规则如下：

| Source | kind / code | Safe message and metadata |
| --- | --- | --- |
| FastMCP/Pydantic input validation or tool `ValueError` | `validation` / `validation_failed` | `Invalid tool input.`；不回显原始异常。 |
| Unknown tool | `validation` / `tool_not_found` | `Tool is not available.` |
| `OrbitAPIError` | `orbit_api` / Orbit code or `orbit_request_failed` | 使用 OrbitClient 已脱敏的 message；保留 status 与 request_id（若有）。 |
| `RuntimeTargetError` or `DockerRuntimeError` | `runtime` / `runtime_failed` | `Runtime operation failed.`；不回显 Docker stderr、stdout、命令或工作目录。 |
| 其他未预期异常 | `internal` / `internal_error` | `Internal server error.`；不回显异常或 traceback。 |

文本内容由同一安全 message 生成，用于不读取 `structuredContent` 的 MCP 客户端。机器处理读取 `structuredContent.error`。`isError=true` 一律表示该调用失败；任何依赖该结果的后续动作不得自动执行或记为成功。

`SettingsError`、stdio 进程崩溃、启动超时、传输断开和格式错误的 MCP JSON-RPC 请求不属于一个已完成的 `tools/call`，因此不能伪装为该对象。客户端把它们统一按未获得工具结果的失败处理，并停止依赖该结果的操作。

### Domain conclusions versus tool errors

以下值是已成功读取或计算出的领域结论，必须保持 `isError=false`：

- `orbit_wait_deployment` 的 deployment status（包括 `faulted`、`canceled`）和 `timed_out`。
- `verify_deployment` 的 `consistent`、`failed`、`drift`、`inconclusive`，前提是工具已取得足够业务证据并完成该比较。

以下不是领域结论，必须传播到统一 MCP 错误映射：Orbit HTTP/网络失败、Docker CLI/Compose 失败、runtime target 不合法、输入校验失败和未预期异常。为满足该边界，`verify_deployment` 与 `observe_stability` 不再捕获这类异常并把它们转写为 `inconclusive`；现有由实际 deployment 状态、容器状态或比较结果得出的结论保持不变。

### Go HTTP transaction error contract

Go 端沿用既有 `apperror` 和 `transportresponse.WriteError`，不创建第二套 HTTP 错误 DTO。

1. `BeginTx` 失败使用 `apperror.KindInternal` 和 `WriteError`，因此返回 `internal_error`、`Internal server error.` 和 request ID，而不是自定义 `message` 字段。
2. 事务中间件仅包裹会变更状态的 API 请求（POST、PUT、PATCH、DELETE）。健康检查、读请求、OPTIONS 与静态资源不创建事务，也不缓冲响应。
3. 对被事务包裹的请求，中间件缓冲 handler 的 headers、status 和 body。只有 `Commit` 成功后才将成功响应写入网络。
4. handler 已中止、记录错误或返回 4xx/5xx 时，回滚并原样写出其既有错误响应。`Commit` 失败时丢弃已缓冲的成功响应，写出统一的 500 错误，再记录服务端原因。
5. handler panic 时，事务中间件丢弃缓冲内容、回滚并恢复原始 writer，然后重新抛出；既有 Recovery middleware 负责使用 `WriteError` 返回统一内部错误。这样不会泄露或混入任何已缓冲的成功 body。
6. rollback 失败只作为服务器日志事件附加到已经确定的失败路径；它不能把已经完成的客户端错误改写为第二个响应。

### Direct MCP operation guide

新增指南只把用户描述映射到单个 MCP 动作，不定义自动编排：

| Explicit user request | Eligible tool | No implied action |
| --- | --- | --- |
| enumerate an Application's Services | `orbit_list_application_services` | deploy, stop, delete |
| deploy | `orbit_deploy` | wait, verify, stop, delete |
| wait | `orbit_wait_deployment` | verify, stop, delete |
| verify | `verify_deployment` | stop, delete |
| stop a Service | `orbit_list_application_services`, then `orbit_stop` | volume removal, Application deletion |
| remove managed volumes | `orbit_stop(remove_volumes=true)` | Application deletion |
| delete an Application | `orbit_delete_application` | directory removal unless `remove_dir=true` is explicit |

示例只展示参数来源与错误处理，不调用 RAGFlow 初始化器，也不创建“cleanup”工具或测试脚本。真实 Docker/RAGFlow 操作仅在用户明确要求的独立集成测试目标中执行。

## Affected components

| Component | Change |
| --- | --- |
| `.codex/config.toml` | 新增项目级 stdio MCP manifest。 |
| `mcp/src/pomelo_orbit_mcp/server.py` | 使用共享的错误映射 FastMCP 子类。 |
| `mcp/src/pomelo_orbit_mcp/mcp_errors.py` | 新增安全错误分类、`CallToolResult` 构造和文本摘要。 |
| `mcp/src/pomelo_orbit_mcp/tools/orbit.py` | 新增受限 Service 列表工具和允许字段投影。 |
| `mcp/src/pomelo_orbit_mcp/verification.py` | 不再把工具异常吞为 `inconclusive`。 |
| `mcp/tests/` | 覆盖 Service 投影、所有错误类别、stdio 协议结果和领域结论边界。 |
| `internal/infrastructure/database/tx/request.go` | 统一 begin/commit 失败响应，并实现事务响应缓冲。 |
| `internal/infrastructure/database/tx/request_test.go` | 覆盖 begin、commit、rollback、panic 和响应不泄露。 |
| `go.mod`, `go.sum` | 加入 test-only SQL mock 依赖，以稳定模拟 Begin/Commit 失败。 |
| `mcp/README.md`, `docs/guides/` | 说明 manifest 前置条件和按显式请求选择 MCP 动作。 |

## Interfaces

| Interface | Request | Response / behavior |
| --- | --- | --- |
| `orbit_list_application_services` | `application_id: string` | `{application_id, services}`；每项仅为允许的 Service 摘要。 |
| MCP error result | 已建立 session 后的 validation、Orbit API、runtime、internal 失败 | `isError=true`、安全 TextContent 和 `structuredContent.error`。 |
| `orbit_wait_deployment` | `deployment_id`, optional timeout | `isError=false`；返回 deployment 与 `timed_out`。 |
| `verify_deployment` | `application_id`, `deployment_id` | 领域比较成功时 `isError=false`；依赖系统失败时统一 MCP error result。 |
| Go HTTP write route | 任意事务性 API 写请求 | 仅在 Commit 成功时返回 2xx；begin/commit/panic 使用既有 `{code,error,requestId}`。 |
| `.codex/config.toml` | Codex trusted project configuration | 启动名为 `pomelo_orbit` 的 stdio Server，不携带秘密或用户路径。 |

## Verification design

1. MCP 协议测试经真实 stdio `ClientSession` 调用，断言每个错误类别同时具有 `isError=true`、安全文本和同形 `structuredContent.error`。
2. 协议测试注入包含伪造 Docker stderr、认证字样和异常详情的错误，断言它们不出现在文本或结构化对象中。
3. MCP 测试断言 Pydantic 输入错误、直接 `ValueError`、`OrbitAPIError`、`RuntimeTargetError`、`DockerRuntimeError` 与未知异常均进入相应类别；OrbitAPIError 保留 status、code、request_id。
4. 验证工具测试断言 API/Docker/runtime 异常为 `isError=true`；deployment `faulted`、`canceled`、timeout、以及完成比较后的 `failed`、`drift`、`inconclusive` 仍为正常结果。
5. Service 列表测试使用包含 `runtime_config` 和额外字段的 client fixture，断言输出只含允许字段和正确 `service_id`。
6. Go 事务测试用 test-only SQL mock 强制 Begin 与 Commit 失败，断言统一 500 契约、request ID、无成功 body、无提交；同时覆盖成功提交、业务 4xx rollback 与 panic recovery。
7. manifest 验收在新的受信任 Codex 会话完成：检查 `pomelo_orbit` 被发现、schema 可见、Server unavailable 时不会把后续动作报告为成功。该验收不部署 Docker Application。
8. 真实 Docker 集成测试保持显式、隔离且默认不运行；不读取、修改或删除仓库 `data/`。

## Risks and controls

1. Codex 会话不会热加载新 manifest。控制：实现后使用新会话完成 manifest 验收，不把旧会话的工具缺失误判为 Server 缺陷。
2. FastMCP 是外部依赖。控制：将已验证的 `CallToolResult` stdio 测试固定在测试套件，升级 MCP 版本时先运行该测试。
3. 事务响应缓冲若包裹静态或流式响应会改变语义。控制：只用于事务性 API 写请求；代码库当前没有流式 API，静态 `ServeFile` 路径明确绕过。
4. Docker 的原始 stderr 可能包含敏感值。控制：错误映射永不使用 `str(DockerRuntimeError)`；验证工具不得把异常转为正常 evidence。
5. 项目 MCP Server 启动仍依赖操作者已有的 `mcp/.env`、Orbit API、Docker context 和 data-root 配置。控制：manifest 不伪造这些值，文档将它们列为运行前置条件，而非执行期自动修复或脚本化。

## Alternatives

1. 每个工具自行捕获并返回错误 dict：不采用。输入 schema 和未知异常仍会绕过该逻辑，且会产生不一致文本。
2. 解析 FastMCP 默认 `ToolError` 文本：不采用。它丢失类型、status 和 request ID，且可能回显敏感 stderr。
3. 把 verification 的环境错误继续记为 `inconclusive`：不采用。它错误地把工具失败写为成功调用，调用方无法中止依赖操作。
4. 新建 `cleanup` 复合工具或把 stop/delete 固化成结束流程：不采用。生命周期动作必须由用户独立选择。
5. 为本功能调整 `Settings`、data root 或 RAGFlow 初始化脚本：不采用。它们不属于项目 MCP 注册和工具契约范围。

## User review notes

用户确认：项目 MCP 使用 manifest 声明；Service 查询是应补足的接口自洽性；所有异常响应和处理方式必须一致；停止、删除与其他生命周期动作独立；不引入脚本、journal 或 data 目录策略。

本次规格补充已基于当前 Go 实现、MCP 1.28.1 stdio 协议原型和 Codex CLI 0.145.0 配置字段解析完成开发前分析。尚未修改产品代码、创建 MCP manifest、启动业务 Server 或执行部署。
