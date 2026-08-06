# Go 持续部署 MCP 改写
最后修改时间: 2026-08-06 09:09:55

Review status: Accepted

流程模式: 标准 / standard

## Background

当前 `mcp/src/delivery-mcp/` 是 Python 实现的本地 stdio MCP Server。它对外提供 `orbit_*` 生命周期工具、`runtime_*` 运行态诊断工具和 `verify_deployment`，通过 Orbit HTTP API 与 Docker CLI 完成交付控制和受管运行态查询。

Pomelo Orbit 的领域规则、资源状态、Deployment 创建和异步执行已位于 Go 的 `internal/application` 用例中。本阶段将当前 Python 持续部署 MCP 改写为 Go 实现，并提供持续部署页面内的对话入口。对话 HTTP API 通过 Go MCP Client 调用持续部署 MCP，MCP 直接调用 application usecase，不经由本项目 HTTP API 回环。网页调用携带当前登录用户的 `Authorization`，MCP 将该用户传入 application usecase。无网页的 stdio MCP 认证接入留待后续单独确定，不在本阶段以固定用户配置替代。

## Known issue: Python MCP collection wrapper

Web API 和 HTTP handler 正常：前端将 `VersionComponentMountsUpdateReq` 直接作为 body 发送，实际请求为 `{ "mounts": [ ... ] }`，后端 JSON 解码没有额外嵌套。

问题仅在 Python MCP 的公开 tool schema。`mcp/src/delivery-mcp/src/pomelo_delivery_mcp/version_specs.py` 中的 `VersionComponentMountsUpdate` 定义了 `mounts: list[LogicalMount]`，同时 `orbit_update_version_component_mounts` 又将该 model 作为名为 `mounts` 的 tool 参数接收。因此 MCP 调用参数必须是 `{ "mounts": { "mounts": [ ... ] } }`；工具调用 `mounts.model_dump()` 后，实际发往 Web 的 body 仍是正确的一层 `{ "mounts": [ ... ] }`。

这不是 Web API 缺陷，也不要求变更 HTTP request DTO。Go 改写须以该 Python 包装器行为为兼容性例外：公开 MCP schema 改为扁平集合参数，application DTO 保持既有单层集合字段。`env`、`endpoints`、`dependencies` 等同类工具按相同原则处理。

## Goal

1. 使用官方 `github.com/modelcontextprotocol/go-sdk` 实现 Go 版持续部署 MCP Server。
2. 迁移当前 Python `pomelo_delivery` 的工具表、工具名称、结果语义和结构化错误契约，保留 `pomelo_delivery` 注册名与 `orbit_*` 工具前缀；同名集合更新参数按本需求的扁平 schema 修正。
3. 每个业务工具直接调用对应的 `internal/application` usecase；禁止 Go MCP 通过本项目 `/api/*` HTTP 路由回环调用业务能力。
4. 提供供项目内 HTTP 服务以 Go MCP Client 集成的 Streamable HTTP MCP transport；每个 MCP session 使用发起网页请求的当前用户。
5. 保持持续部署领域行为不变：Application、Version、Component、Service、Gateway 与 Deployment 仍由现有 Go 用例和异步 worker 管理。
6. 在持续部署页面提供对话入口；前端调用 proto 定义的对话 HTTP API，由后端完成 LLM 工具调用循环和 MCP Client 调用，前端不直接调用 `/mcp`。
7. MCP 的业务字段复用现有 proto 生成的 Go DTO；仅保留版本、组件、服务等 MCP 路由字段的轻量 envelope，移除重复的 `XXXInput` 业务结构。

## Non-goal

1. 不改变持续部署领域模型、HTTP API 对外契约、数据库 schema、Deployment worker 或 Docker Compose 生命周期规则。
2. 不迁移独立的 `pipeline-mcp`，也不混合 CI 与 CD 工具。
3. 不新增未在 Python `pomelo_delivery` 中存在的 MCP 工具或隐式编排流程。
4. 不让 MCP tool handler 直接访问 repository、SQLC、数据库连接或绕过 application usecase。
5. 不在本阶段为无网页的 Codex、Claude Code stdio MCP 设计或实现认证方案。
6. 对话历史暂由页面随请求提交，不在本阶段新增持久化会话、消息表或跨设备历史同步。

## User scenarios

1. 登录用户在 Web 对话服务中发起持续部署请求；HTTP 服务以该请求的 `Authorization` 连接 `/mcp`，并发现与 Python 版等价的持续部署工具。
2. 调用方请求 `orbit_deploy` 时，Go MCP 直接调用部署 command usecase 创建 Deployment；现有 worker 随后执行该 Deployment，MCP 不经由 `/api/service/:service_id/deploy` 回环。
3. 调用方查询 Application、Version、Service、Deployment 或其日志时，Go MCP 从相应 application usecase 获得与既有 MCP 契约等价的响应。
4. 调用方使用运行态或验证工具时，工具继续仅作用于 Orbit 管理的运行目标，并保持既有 Docker 运行态查询和结果约束。
5. 项目内 HTTP 服务创建 Go MCP Client，通过 Streamable HTTP 调用 Go MCP 的 `orbit_*` tool，并透传当前用户的 `Authorization`；工具调用 application usecase 时使用该用户 ID。
6. 登录用户从持续部署页面进入对话工作区，提交自然语言请求；HTTP 对话 API 将当前上下文、MCP `tools/list` 返回的 schema 与工具描述交给 LLM，执行 LLM 返回的 tool call，并将工具结果继续提供给 LLM 直到得到回答。
7. 对话需要创建、更新或部署容器时，LLM 只通过 `orbit_*` MCP tool 完成；实际数据写入、Deployment 创建与 Docker 执行仍由既有 application usecase 和 worker 负责。

## Acceptance

1. Go MCP 使用 `github.com/modelcontextprotocol/go-sdk`，认证后的 Streamable HTTP 能完成 `tools/list` 和 `tools/call`。
2. 当前 Python `pomelo_delivery` 所注册的 `orbit_*`、`runtime_*` 和 `verify_deployment` 全部保留为等价的 Go 工具；不得以删减运行态或验证能力作为迁移方案。
3. 保留工具的名称、默认值、主要响应字段、领域终态语义和结构化错误类别与 Python 版兼容；Version Component 和 Service Environment 的同名集合更新工具使用扁平 MCP 参数，不保留 Python 的重复外层包装。`orbit_update_version_component_mounts` 必须接受 `{ "mounts": [ ... ] }`，并映射为 application DTO 的 `{ Mounts: [...] }`。
4. 所有 Application、Version、Component、Service、Gateway 和 Deployment 工具通过 `internal/application` usecase 执行业务操作；代码和测试中不存在对本项目 `/api/*` 的 HTTP 回环调用。
5. 运行态与验证工具不扩大其现有受管目标和 Docker 操作范围。
6. Go MCP 的依赖装配遵循现有分层方向：stdio MCP 是入站适配器，调用 application usecase；生成代码、repository 和基础设施实现不泄漏到 tool handler。
7. 为工具注册、输入 schema、核心成功路径、领域错误映射和 Python/Go 契约对照提供自动化测试。
8. `/mcp` 必须通过现有 Bearer token 认证当前用户；按 MCP session 创建的 Core 只将该用户 ID 传给 application usecase，不读取全局固定 actor 配置。
9. 无网页的标准 stdio MCP、Codex 注册切换与 Python Delivery MCP 删除依赖后续认证方案，不在本阶段完成。
10. 对话 HTTP 请求和响应由 proto 定义并生成 Go、Vue TypeScript 类型；Vue 只调用该 API，后端才可连接 `/mcp`。
11. `internal/api/mcp/delivery` 不保留 `ComponentEnvInput`、`ComponentMountInput`、`ServiceEnvInput` 等重复业务 DTO；Component、Service overlay 及其集合字段使用 `internal/gen/proto` 中生成的叶子消息类型，且同名集合工具仍暴露扁平 MCP 参数。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 本需求是 Python 持续部署 MCP 的 Go 改写，不是为 Web 对话入口临时增加 HTTP 代理。
2. Go MCP 以现有 Go application usecase 为唯一业务入口；不通过本项目 HTTP API 回环。
3. 当前阶段只提供网页登录后的 Streamable HTTP transport；HTTP 服务通过官方 Go MCP Client 集成 HTTP transport，不经过本项目 `/api/*` 路由，并透传当前用户的 `Authorization`。
4. 选择官方 Go MCP SDK，避免同时引入多套 MCP SDK。
5. Python 与 Go MCP 的工具契约在过渡期间持续对照；无网页 stdio 认证方案明确后，再决定 `pomelo_delivery` 的注册切换和 Python 实现删除。
6. `runtime_*` 与 `verify_deployment` 完整迁移。现有 application usecase 未覆盖的 Docker 运行态能力，先补充为最小 application usecase，再由 MCP tool 调用；不得把该逻辑直接放入 tool handler。
7. 保留 `orbit_provision_gateway` 的完整高层语义；其组合流程应迁移为 Gateway application usecase，再由 MCP tool 调用。
8. 现有 HTTP server 在 `/mcp` 挂载 Streamable HTTP handler；handler 用当前请求 Bearer token 创建绑定该用户的 MCP Core。`cmd/server mcp` 在无网页认证方案确定前不暴露。
9. Web API 的请求体保持现状。Go MCP 对 `mounts`、`env`、`endpoints`、`dependencies`、`devices`、`resources`、`tmpfs`、`ulimits` 及 Service `env` 等同名集合更新工具采用扁平参数，避免 `mounts: { mounts: [...] }` 这类双层同名结构。
10. 不配置 `mcp.actor_user_id`。HTTP `/mcp` 的 actor 仅来自当前已认证网页请求；无网页 MCP 的认证策略后续单独设计。
11. 对话入口是持续部署模块的一等页面。其 HTTP 入站适配器调用 dialogue application usecase；usecase 通过 LLM port 和 MCP client port 完成工具调用循环，具体 OpenAI-compatible HTTP client 位于基础设施层。
12. proto 是 HTTP 与 MCP 可共用的业务字段来源。MCP envelope 只表达 `version_id`、`component_id`、`service_id` 等工具路由参数；业务字段直接使用生成的 proto message，不把顶层 HTTP request wrapper 作为同名 MCP 参数，避免重新引入集合嵌套。

## Risk

1. Python MCP 包含 Docker 运行态与验证逻辑，完整迁移将要求补充 application usecase。若直接将 Docker 代码放进 MCP tool handler，将违反本需求的分层目标。
2. 工具的默认参数、字段投影和错误语义存在兼容性风险；仅按工具名称迁移不足以保证 Codex 和后续 Go Agent 的行为一致。
3. 迁移期间同时存在两个实现时，必须避免它们使用同一 `pomelo_delivery` 注册名造成调用方不确定性。
4. Web 对话服务的 MCP client 必须透传当前请求的 `Authorization`；漏传时 `/mcp` 返回未认证，不能以默认用户继续调用。
5. LLM 端点、模型或密钥未配置时，对话 API 应明确返回服务不可用；不得在代码或示例配置中写入实际密钥。

## User review notes

1. 用户要求采用 SpecFlow 标准模式推进。
2. 用户确认先完成 Python 持续部署 MCP 到 Go 的改写；每个 tool 应直接调用现有 `internal/application` 用例，不经由本项目 HTTP API 回环。
3. 用户要求 Go MCP 既能被 Codex 或 Claude Code 作为标准 MCP 使用，也能由项目内 HTTP 服务集成；选择 stdio 与 Streamable HTTP 双 transport。
4. 用户确认允许 Python 与 Go 实现在迁移期间并行，完成切换后移除旧 Python 实现。
5. 用户确认完整补齐 `runtime_*` 与 `verify_deployment` 能力，不以删减工具作为迁移方案。
6. 用户同意保留 `orbit_provision_gateway` 的完整高层语义，并将其流程下沉为 Gateway application usecase。
7. 用户明确要求进入 Plan / 计划，Requirement 自动标记为 Accepted。
8. 用户确认将 Python MCP 的同名集合参数嵌套问题纳入迁移任务；Web 接口不改，Go MCP 采用扁平参数。
9. 用户补充该问题的根因与边界：Python `VersionComponentMountsUpdate` wrapper 造成 MCP 参数双层同名，`model_dump()` 后 Web body 正常；须作为 Go MCP 的公开 schema 与 DTO 映射验收项。
10. 用户明确网页发起 MCP 时必须使用当前登录用户；取消全局 `actor_user_id` 配置，无网页 MCP 的认证接入延后处理。
11. 用户指出前端对话入口未实现且 MCP 存在重复 `XXXInput`，要求纳入当前实现；HTTP DTO 已通过 proto 同时生成 Go 与 Vue 类型，MCP 应复用其业务字段。
