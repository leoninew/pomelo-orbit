# Go MCP 契约补齐与 Python MCP 移除计划
最后修改时间: 2026-08-06 14:07:49

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

本计划实施已接受的 [Go MCP 契约补齐与 Python MCP 移除](../requirement/20260806-go-mcp-contract-and-python-removal.md)。Python 不是迁移来源；现有 Go application usecase、Go delivery MCP 与 Web 认证流程是唯一实现依据。

## 实施架构

### 认证和 Transport

保留网页登录发起的 Streamable HTTP `/mcp`，同时为 Codex 增加 Go stdio 入口 `cmd/server mcp`。两种 transport 都构造同一个 `internal/api/mcp/delivery` Core，并通过 application usecase 执行业务；stdio 不调用 `/mcp` 或任何 `/api` 业务工具。

stdio 启动时从用户私有凭据存储读取 Bearer token，并用现有认证服务校验。没有有效 token 时：

1. 在 loopback 随机端口启动一次性 callback listener，生成高熵 `state`，仅接受本机回调且只完成一次。
2. 使用可替换的浏览器启动器打开已配置的 Web `mcp-authorize` 路由，携带 callback URL 与 `state`。
3. 未登录用户先经现有登录页认证；登录页保留原始 redirect，授权页面使用当前浏览器 Bearer token 向后端申请短时、一次性的 MCP 授权码。
4. 授权页仅将授权码和 `state` 重定向至经过 loopback 校验的 callback。stdio 进程以授权码调用专用交换接口取得 Bearer token，写入用户配置目录中的私有凭据文件，然后启动 MCP server。
5. token 失效、回调超时、state 不匹配、授权码重放或浏览器无法启动均返回明确错误并停止启动；不读取浏览器 localStorage、不要求用户复制 token、不回退到 Python 邮箱/密码登录。

浏览器授权码只处理认证。直接 MCP tool 调用继续由已认证 actor 的 Go Core 处理，保持当前 application-usecase 边界。

### MCP 领域契约和对话编排

保持当前 48 个 Go delivery tools。针对现有 proto 推导出的宽泛 schema，新增可复用的 schema augmentation 与工具描述规范，不新增重复的业务 input DTO：

- 组件挂载明确 `directory`、`file`、`named_volume`、`controlled_file` 的 source type、字段限制和示例；受控文件说明相对 source、`source_is_host_path=false`、可为空的 content、最大内容和必填四位八进制 `mode`。
- 端点明确 protocol、port、`internal`、`local`、`host`、`gateway_http`、`gateway_tcp` 等可选 mode 及相关字段；其它已有离散契约如 pull policy、restart policy、dependency condition、healthcheck、device request 同步从现有 Go 校验导出到工具说明或 schema。
- `runtime_doctor` 等具有组合字段的工具在 schema 与描述中直接表达成对参数和互斥约束，避免模型通过错误调用发现规则。

对话 usecase 增加每回合执行状态：读取到已有资源但其声明或 Service overlay 不符合用户目标时，先做最小更新，回读相关 Version/Service 后才允许部署；每个 Service 在同一用户回合只允许一个 `orbit_deploy`，部署后等待终态并将 Deployment ID 保留在终态结果中。工具轮次耗尽或模型失败时，返回可分类的对话失败和已执行 tool/resource 摘要，不将成功副作用笼统包装为服务不可用。

## 实施步骤

1. 建立 Go stdio MCP composition 与本地认证基础设施。
   - 在 `cmd/server` 和 `internal/bootstrap` 增加 `mcp` 命令路径，复用现有配置、数据库、任务和 application-service 组装，不让 tool handler 接触 repository 或 SQLC。
   - 在 `internal/infrastructure/mcp/` 增加浏览器启动、loopback callback、私有 token store 与授权码交换 HTTP client；将外部 I/O 放在基础设施层，使用接口隔离以便测试。
   - 扩展配置和 `.env.example`，为 stdio MCP 明确配置 API base URL 与 Web login URL；只在 `mcp` 命令启动时校验这些字段。凭据文件路径使用用户配置目录，不放在仓库、`data/` 或日志中。
   - 使用官方 Go MCP SDK `StdioTransport` 启动 actor-bound delivery Core；stdio 的协议输出仅写 stdout，诊断写 stderr 或现有受控日志。

2. 实现浏览器授权码交接。
   - 在 auth application/HTTP 边界增加短时单次 MCP grant 的签发和交换能力，包含过期、哈希存储或等价的不可逆校验、一次性消费和用户绑定；不得在 URL、响应日志或错误中暴露 Bearer token。
   - 新增 Web `mcp-authorize` 路由与页面，校验 callback 仅为 loopback，调用 grant 接口后重定向。登录成功和 Google 登录完成后均恢复原始 redirect，而不是无条件跳转首页。
   - 在 `.codex/config.toml` 将 `pomelo_delivery` 从 `uv ... pomelo-delivery-mcp` 切换为 Go `cmd/server mcp` 命令，保留注册名和合理的启动/工具超时。

3. 补齐 Go MCP 的模型可发现契约。
   - 重构 `internal/api/mcp/delivery` 的工具注册辅助函数，使其可保留类型安全的 proto 解码，同时为需要领域条件的工具附加自定义 JSON Schema、enum、description 和示例。
   - 优先改造 mount、endpoint、runtime target/doctor 工具；字段含义、默认值和约束以 application usecase 校验为源，不重建 Python request wrapper。
   - 扩展 MCP server tests，断言公开 tool schema 的受控文件、端点和成对参数语义，而非仅检查工具数量和集合扁平化。

4. 收敛部署对话的写入流程和失败结果。
   - 在 `internal/application/dialogue/usecase` 为单次 Turn 维护已更新资源、Deployment ID 和每个 Service 的 deploy guard；拒绝重复部署并把可行动的工具错误返回给模型。
   - 更新系统指令，要求将用户的配置描述视为目标状态：存在资源必须比对具体字段，不匹配必须更新、回读，再部署一次并等待。
   - 扩展 dialogue DTO、HTTP stream handler 与前端 store/page，使轮次耗尽或 provider 失败时仍显示已完成的操作、Deployment ID 和明确失败原因；不在失败时伪造成功回答。
   - 为已有错误配置的 Component 建立 fake MCP/LLM 场景，验证 `file` 到 `controlled_file`、`host` 到 `gateway_http` 的更新、回读和单次部署顺序。

5. 删除 Python MCP 并更新活文档、Skill 与失效运维脚本。
   - 在确认 Go stdio 可认证、列出工具并执行受控调用后，删除整个跟踪的 `mcp/` 目录；实施时先校验精确路径。该目录下被忽略的 Python venv/cache 可删除，用户本地凭据不迁移至仓库。
   - 更新 `AGENTS.md`、`.codex/config.toml`、`docs/guides/mcp-direct-operations.md`、RAGFlow skills 及其活动引用，移除 Python/uv/旧 stdio 说明，写入 Go MCP 的真实登录、schema 更新和会话重启行为。
	- 不改写 `docs/archive/**` 或上一任务正在暂存的 `20260805-go-delivery-mcp` 过程记录；历史 Python 实现结论保留为历史上下文。
	- 保留所有与 Python MCP 无关的 `scripts/`、测试和同名参考文档；本任务不清理远程 SSH、Docker Compose 或其他运行维护工具。

## 预期修改文件

| 范围 | 预期变更 |
| --- | --- |
| `cmd/server/`、`internal/bootstrap/` | 增加 Go stdio MCP 启动模式并复用现有依赖装配。 |
| `internal/application/auth/`、`internal/api/http/handler/auth/`、`internal/api/http/routes/auth.go` | 单次授权码签发/交换的领域、入站适配和路由。 |
| `internal/infrastructure/mcp/` | 浏览器启动、回调、凭据持久化和认证交接。 |
| `internal/api/mcp/delivery/` | 自定义 schema augmentation、工具说明及其测试。 |
| `internal/application/dialogue/`、`internal/api/http/handler/dialogue/`、`proto/orbit/v1/dialogue/` | 单次部署 guard、部分结果和失败事件契约。 |
| `web/src/views/auth/`、`web/src/router/`、`web/src/api/auth/`、`web/src/views/deployment/`、`web/src/stores/` | MCP 授权页、登录 redirect、授权调用和对话失败状态展示。 |
| `configs/config.yaml`、`.env.example`、`.codex/config.toml` | Go MCP 启动与认证所需的显式配置及 Codex 注册切换。 |
| `mcp/` | 整个目录删除。 |
| `AGENTS.md`、`docs/guides/`、`skills/` | 更新当前 MCP 操作与 RAGFlow 指令，删除 Python MCP 相关的旧入口。 |

## Verification plan

1. Go unit/integration tests覆盖：有效/失效本地 token、callback state、回调来源限制、授权码过期和重放、凭据文件权限与不泄密、stdio tool discovery，以及 stdio 与 HTTP transport 的相同 tool registry。
2. MCP schema tests覆盖：48 个 tool 保留、扁平集合字段保持、`controlled_file` 的正确示例与缺失/非法 mode 拒绝、端点 mode 和 `runtime_doctor` 的组合参数说明。
3. Dialogue tests覆盖：已有资源的差异纠偏、回读、同一 Service 的单次 deployment、wait 后的终态说明，以及轮次耗尽时保留部分 tool/result。
4. 认证授权页和登录 redirect 的前端类型检查与针对性测试；人工验证一次浏览器登录、stdio MCP 重启复用凭据、token 失效重新认证，确认 stdout 不含日志或凭据。
5. 执行项目约定的 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`，以及修改前端后的 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。
6. 删除后用 `rg` 确认活配置、AGENTS、guide 和 skill 不再引用 `mcp/`、`uv --directory mcp`、`pomelo-delivery-mcp` 或 `pomelo-pipeline-mcp`；archive 历史引用不作为失败条件。

## Assumptions and blockers

1. Go stdio MCP 在本地执行时能加载与 Orbit HTTP 服务相同的配置、数据库和 JWT secret；服务端可用时，配置中的 API URL 和 Web login URL 指向同一受信任 Orbit 环境。
2. 浏览器能够访问 Web login URL，并允许本机 loopback callback；Turnstile 或第三方登录仍由既有 Web 登录页处理。
3. `mcp/` 中存在的被忽略本地凭据和虚拟环境在删除前会按精确路径检查，避免扩大到仓库外的用户配置目录。

## Risks and rollback

1. 浏览器登录交接、loopback callback 和 token 持久化扩大认证攻击面。实现必须限制 callback host、绑定随机 state、使用短时单次 code、限制凭据文件权限，并在失败时拒绝启动而不是降级认证。
2. Application 校验和 MCP schema 可能逐步漂移。schema augmentation 必须有测试锚定，并在修改领域枚举时同步更新。
3. Python MCP 不作为回滚路径。若 Go stdio 或登录交接出现问题，修复 Go 实现和 Codex 配置，不重新引入 Python/uv compatibility layer。
4. 对话侧的 deploy guard 必须按 Service 和单个 Turn 隔离，不能阻止下一次用户明确请求的部署，也不能因为模型重试掩盖真实部署失败。

## User review notes

1. 用户要求使用新的标准 / standard SpecFlow 任务记录。
2. 用户明确所有 Python MCP 实现位于 `mcp/`，该目录直接移除；`pipeline-mcp` 无须关注或替代。
3. 用户明确 Go MCP 不复刻 Python，而是基于当前 Go 实现补齐能力，`controlled_file` 是优先缺口。
4. 用户决定 Codex MCP 认证通过浏览器打开 Orbit 登录页并记录本地凭据。
5. 用户澄清本任务只删除 Python MCP 相关的不适用内容；所有其他 `scripts/` 必须保留。
6. 用户明确要求开始 Implementation / 实现，Plan 视为接受。
