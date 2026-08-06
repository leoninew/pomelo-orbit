# Go 持续部署 MCP 改写验证
最后修改时间: 2026-08-06 16:26:52

Review status: Draft

流程模式: 标准 / standard

## 需求对齐

- Go Delivery MCP 使用官方 `github.com/modelcontextprotocol/go-sdk` 注册 48 个与 Python 版同名的 `orbit_*`、`runtime_*` 和 `verify_deployment` 工具。
- `/mcp` 使用 Streamable HTTP，并从当前请求 Bearer token 解析用户后按会话构造 MCP Core；对话 usecase 通过 Go MCP Client 透传该 Authorization。
- 工具调用 application usecase；Component 与 Service 的集合更新参数为扁平数组，未保留 Python 的同名嵌套 wrapper。
- `runtime_*`、`verify_deployment` 和 `orbit_provision_gateway` 已分别下沉为 deployment 或 gateway application usecase。
- 对话 HTTP 契约由 proto 生成 Go 与 Vue DTO；前端调用对话 API，后端执行 LLM tool-calling loop 并展示工具进度。

## 规格对齐

不适用。当前 feature 采用标准模式 / standard，已有 Requirement 与 Plan，没有独立 Spec。

## 计划对齐

提交 `095749d3` 已实现共享 MCP Core、认证 Streamable HTTP transport、application-usecase 边界、对话 API/客户端/页面、Gateway 编排与运行态验证能力。

Requirement 和 Plan 的早期文字同时出现“stdio + Streamable HTTP”与“无网页 stdio 延后”的表述。实际实现遵循后者：仅提供已认证的 `/mcp` Streamable HTTP；无网页的 stdio、Codex/Claude Code 注册切换和 Python MCP 删除仍未实现。

## 实际变更

### 预期范围

- Go MCP adapter、认证 HTTP transport 与项目内 MCP client。
- deployment/gateway runtime usecase、对话 application usecase 与 OpenAI-compatible client。
- dialogue proto、HTTP handler、Vue API、Pinia store 和页面。
- 配置、依赖、导航与相关测试。

### 实际范围

- 验证对象为提交 `095749d3`（55 个文件，5161 行新增、36 行删除），覆盖预期的 Go MCP、对话、runtime/verification、Gateway、配置、proto、Vue 页面、测试与过程文档范围。
- 当前工作区存在后续未提交工作，不属于本 feature；本验证不把它们计入实际范围，也不调整其暂存状态。

## 验收清单

- [x] Go MCP 使用官方 SDK，认证 HTTP MCP 可完成 `tools/list` 与 `tools/call`。
- [x] Python Delivery MCP 的工具面、运行态工具和验证工具已在 Go MCP 保留；工具表测试断言 48 个工具。
- [x] Component/Service 集合参数为扁平 schema，覆盖 mounts 映射与 schema 断言。
- [x] MCP handler 通过 application usecase 工作，未发现向本项目 HTTP API 发起调用的代码；响应中的 `/api/*` 仅作兼容性元数据投影。
- [x] Gateway provision、runtime 与 verification 通过 application usecase 提供。
- [x] 对话 API、MCP client Authorization 转发和当前用户 MCP Core 有单元测试覆盖。
- [x] 对话 proto 已生成 Go 与 Vue DTO，Web 不直接连接 `/mcp`。
- [ ] Python/Go 自动化 fixture 契约对照尚未建立；当前测试覆盖 Go 工具表、schema、关键调用和 HTTP transport，不比较 Python 进程的实际输出。
- [ ] 无网页 stdio transport、Codex/Claude Code 注册和 Python Delivery MCP 移除，按已接受的延期决定未实现。
- [ ] 真实 LLM、MCP、Docker/Deployment Worker 的集成验收，由用户执行。

## 自动化检查

| 命令 | 结果 |
| --- | --- |
| `task check` | 通过：Vue typecheck、ESLint、Prettier、golangci-lint fmt/run 均通过。 |
| `go test ./internal/api/mcp/delivery ./internal/application/dialogue/usecase ./internal/infrastructure/mcp/delivery ./internal/infrastructure/external/openai ./internal/application/gateway/usecase ./internal/application/deployment/usecase ./internal/application/application/usecase ./internal/application/pipeline_run/usecase` | 通过；OpenAI client 包无测试文件。 |
| `yarn --cwd web test` | 通过：10 个测试文件、62 个用例。 |

首次 `task check` 报告 5 处未处理的 `Close()` 返回值，已按仓库既有显式忽略模式修复；首次 Web 单测的 2 个导航断言未包含新增 `/dialogue` 首入口，已更新后通过。

当前工作区已有后续未提交改动，且 `task check` 会执行 `lint:fix`、`format:fix` 等写入操作。为避免影响该工作，本验证未在当前工作区重复运行该命令；以上结果对应提交 `095749d3` 的实现阶段检查。

## 用户集成验收

由用户执行，不在本次验证中启动或操作部署环境：

1. 在持续部署对话页验证无 LLM 配置、普通回复、工具调用成功和工具调用失败的展示。
2. 验证请求以当前登录用户访问 `/mcp`，且对话中的读写操作符合该用户可见资源。
3. 对创建、更新、部署、停止和运行态诊断分别确认 MCP 返回、异步 Deployment 状态与页面工具进度一致。
4. 验证路由切换后对话历史保留，并确认刷新后的预期清空行为。

## 风险与未完成项

- 真实 Docker、Gateway、LLM provider 和 Deployment Worker 集成未由自动化检查覆盖。
- Python/Go fixture 契约对照尚未自动化；48 个 Go 工具的名称、schema 与核心路径已有单元测试，但仍需在切换 Python MCP 前补齐跨实现比较。
- 无网页标准 MCP transport 仍待认证方案明确；不能将当前 `/mcp` HTTP 端点等同于已完成的 stdio MCP。
- 当前工作区的后续改动未纳入本次结论；若需再次验证，应在其独立提交或隔离工作区中运行。

## 结论

提交 `095749d3` 的自动化质量门禁与受影响单元测试均通过，静态核对确认 SDK、认证转发、48 工具注册和 mounts 扁平 schema。Verification 保持 Draft，等待用户完成集成验收；Python/Go 自动化契约对照亦需在切换和删除 Python MCP 前补齐。
