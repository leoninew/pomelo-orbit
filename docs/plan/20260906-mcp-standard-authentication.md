# MCP 调用认证边界重构计划
最后修改时间: 2026-09-07 13:31:14

Review status: Accepted

Flow mode: standard

## Requirement basis

- 依据：[MCP 调用认证边界重构需求](../requirement/20260906-mcp-standard-authentication.md)，状态 `Accepted`。
- 用户从 Requirement 明确要求进入 Plan。本功能没有独立 Spec；本轮只记录实施方案和待审查假设，不补写已接受的 Spec，不实施产品代码，不进入 Verification。
- 实际场景仅为 Web Dialogue 内部调用与 Codex CLI 启动 `go run ./cmd/server mcp` 的 stdio 调用。不建设 OAuth2/OIDC，也不改为 Codex 直连 HTTP MCP。
- stdio 的环境凭据形式是 Orbit 的项目选择；不把 JWT 格式、删除浏览器辅助流程或删除 HTTP transport 说成 MCP 规范对所有实现的强制要求。

## Current implementation

| 边界 | 当前代码与行为 |
| --- | --- |
| Dialogue 入站 | `internal/api/http/handler/dialogue/dialogue.go` 校验 `dialogue:write`，把用户 ID 和原始 Authorization 一起传入 usecase。 |
| Dialogue 出站 | `internal/application/dialogue/port/port.go` 的 `MCPClientFactory.Connect` 接收 Authorization；`internal/infrastructure/mcp/delivery/client.go` 通过 Streamable HTTP 回环访问 `/mcp`。 |
| MCP Core | `internal/api/mcp/delivery/server.go` 注册共享工具。固定 actor 或首次调用 `ActorAuthorizer` 后缓存 actor；后续 stdio 调用不再认证。 |
| stdio 装配 | `internal/bootstrap/app.go:RunMCP` 打开配置数据库，复用 application services；无有效 token 时使用浏览器授权器和 token 文件。 |
| 身份校验 | `internal/application/auth/usecase/service.go:Authenticate` 校验 JWT、用户存在性和 enabled 状态，并查当前角色与权限。 |
| 凭据生命周期 | `internal/auth/jwt/token.go` 签发 24 小时 JWT；当前无单 token 撤销和自动刷新，退出或改密不撤销已签发 JWT。 |
| 可复用实现 | `internal/api/mcp/delivery/server_test.go:connectInMemory` 已使用 Go MCP SDK `NewInMemoryTransports`，可用于内部 client/server 会话。 |

## Assumptions and decisions

以下方案用于让计划可执行，随本 Plan 一起接受审查，不冒充用户已单独确认的设计决策。

### Credential and delivery

1. stdio 使用 MCP Personal Access Token（PAT），不再复用 `/api/auth/login` 签发的 24 小时 Web JWT。PAT 有独立随机值、可选到期和撤销边界，但不引入 OAuth/OIDC、远程 HTTP MCP 或 CLI 密码登录。
2. 已登录用户可在 Web 中创建、列出和撤销自己的 PAT；创建时只返回一次明文。数据库只存 SHA-256 摘要、所属用户、显示名称、创建时间和可选到期时间。不能查询、导出或恢复明文 token。
3. 唯一 stdio 输入保持为 `POMELO_ORBIT_MCP__ACCESS_TOKEN`，映射集中 typed config 的 `MCP.AccessToken`。业务层和 tool handler 不直接调用 `os.Getenv`，缺失值不使用默认账号或缓存凭据兜底。
4. `.codex/config.toml` 保留 stdio command、cwd、超时与现有环境选择器，仅声明将父进程的凭据变量传给子进程，不写入真实 token，也不使用 TOML 字符串插值假装环境透传。拟采用 Codex 的 `env_vars` 项，实施前核实安装版本的配置 schema 并以假凭据子进程检查实际传递结果。
5. 新变量只通过集中配置入口处理；配置样例仅写变量名和空值说明，不新增 credential cache，也不把该个人凭据加入系统设置展示或持久化项。实现时检查配置解码错误、日志和测试输出均不会回显该字段。
6. PAT 默认不过期，也可在创建时选择有限天数；撤销、过期、用户禁用或删除后，下一次 `tools/call` 被拒绝。运行中 session 不热读新环境、不更换 actor，且不自行续期；用户替换 PAT 后重启 MCP session。

### Execution boundaries

1. Web Dialogue 使用 SDK 内存 transport 连接相同 `delivery.NewServer`。每个对话 turn 创建独立、固定 actor 的 MCP server/client 会话，结束或取消时关闭双方；不为不同用户复用一个可变 actor 的全局 Core。
2. Dialogue 的 application 层仅依赖已有 `MCPClientFactory` 端口。基础设施适配器接收 bootstrap 注入的 `serverForActor` 工厂，不能反向 import `internal/api/mcp/delivery`；Core 创建和 usecase 装配留在 bootstrap。
3. stdio 每次 `tools/call` 调用 `AuthService.AuthenticateMCPAccessToken`，成功后才执行工具。认证门禁集中在 MCP 接收中间件，不散落到各工具；不存在“已经绑定 actor 就跳过认证”的分支。
4. stdio 首次成功认证固定用户 ID，后续每次校验须匹配该 actor。绑定与并发校验通过受控同步完成，不在工具执行过程中修改共享 actor；失败不得继续使用历史 actor 执行业务。
5. 缺失凭据不阻止协议初始化与工具发现；认证发生在工具执行前。配置/数据库/工作目录等现有启动前提仍可能导致进程启动失败，不承诺完全离线 discovery。
6. 固定 Web actor 来源于该 turn 的入站认证，不做第二次网络登录；业务成员权限按既有用例检查。stdio 重新认证阻止的是下一次工具调用，不取消已接受的长调用、已创建任务或正在运行的 Worker。
7. 保留当前可信运行拓扑：stdio 进程访问目标 Orbit 的同一逻辑数据库、签名配置、Docker 和 workspace；不引入从任意工作站访问远程 Orbit 的 bridge。运行 stdio 的人已处于可接触 DB/签名密钥的可信主机边界，JWT 不是对本机管理员的隔离措施。
8. 完全删除 HTTP `/mcp` transport，包括它的专用测试和网络客户端行为；不保留内部测试专用 HTTP 通道。SDK 内存 transport 不监听任何端口。

## Implementation steps

### 1. 收敛 Dialogue 的身份契约

- 从 `Service.CompleteTurn`、`CompleteTurnWithProgress` 和内部 `completeTurn` 移除 Authorization 参数；handler 只传 `current.User.Id`。
- 将 `MCPClientFactory.Connect(ctx, authorization)` 收敛为 `Connect(ctx, actorUserId)`，同步真实实现、fake、调用点与参数名称，不保留旧重载或兼容函数。
- 保持 JSON/Proto 请求响应、SSE 事件、对话历史、LLM 多轮执行和失败后结果记录的当前语义；用户 ID 不进入模型可指定的工具参数。
- 保留 HTTP `dialogue:write` 和项目成员校验，保留 Dialogue 路由不持有长 HTTP 写事务的现状；不要因为改为进程内调用，把整个 LLM turn 包在一个 DB 事务中。

### 2. 将内部 MCP 客户端改为内存 transport

- 原地改造 `internal/infrastructure/mcp/delivery/client.go` 的 factory，移除 endpoint、HTTP client 和 Authorization transport，复用已有工具 schema/result 映射。
- factory 通过注入的构造函数获得固定 actor 的 MCP server，使用官方 SDK `NewInMemoryTransports` 建立双向会话，不手写 MCP 分发器或复制 tool registry。
- client 包装同时管理 client/server session；连接中途失败、context 取消和 `Close` 均释放已经创建的资源，避免 goroutine 残留。
- 在 bootstrap 先构造交付业务依赖，再注入按 actor 创建 Core 的函数和 Dialogue Service；不为依赖装配引入 application 到 api 的反向依赖，不扩大为全仓 DI 重构。
- 增加两个用户并发 turn 的 actor 隔离与取消测试，确认模型输入无法覆盖 actor。

### 3. 替换 stdio 认证和配置

- 新增 MCP PAT 模型、SQLC 查询和 SQLite/MySQL/PostgreSQL 迁移；不修改既有迁移。Persisted token 仅保存 SHA-256 摘要和管理元数据，按 user ID 查询并由数据库 CASCADE 清理。
- 在 Auth Service 中实现创建、列出、撤销和认证 PAT 的用例，重用用户启用状态与角色/权限装配；token 摘要不存在、到期或用户不可用统一返回安全的 unauthorized。
- 新增已认证的单数 `/api/auth/mcp-access-token` 管理入口和对应 Web 页面；创建响应明文 token 只出现一次，列表和日志绝不返回摘要或明文。
- `RunMCP` 从已加载 typed config 捕获 PAT，注入每次调用的认证函数；不读旧 token store、不创建浏览器授权器。
- 重命名或更新 `ActorAuthorizer` 为表达“每次调用认证”的接口和注释，删除首次授权缓存语义。固定 Web actor 与 stdio 凭据门禁是两个受信入口，不是新旧认证兼容路径。
- 调整 `ensureActor`/接收中间件：先检查当次认证结果，再绑定或核对 actor，再交给工具 handler。并发情况下不重复写固定 actor，不出现失败后使用缓存 actor 的路径。
- 无效、过期、用户删除/禁用等返回安全 `unauthorized` tool error；数据库不可用等返回安全的基础设施错误，不吞成“请打开浏览器”。不把原始 credential 或未经脱敏的错误写到 MCP result。
- `internal/config` 删除 `APIUrl`、`WebUrl`、`AuthTimeout`、相关 bind keys 和 MCP URL 推导/校验函数，增加新 credential 字段。HTTP/Worker 启动不要求 stdio token；stdio 缺失凭据的检查留到 `tools/call`。
- PAT 验证复用当前 auth service 的用户启用状态检查；保留现有登录、CSRF、Turnstile、改密和登录历史行为，不改变 Web JWT 格式或生命周期。

### 4. 删除废弃路径

- 删除 `internal/api/mcp/delivery/http.go`、`http_test.go` 及 server tests 中 HTTP transport 专属场景；保留或迁移共享工具契约的测试意图。
- 移除 `internal/api/http/server.go` 的可选 MCP handler 参数、字段与 mux 分支；移除 bootstrap 的 HTTP MCP handler/endpoint 装配。保留给内存 transport 和 stdio 共用的 Core 构造能力。
- 删除 `internal/infrastructure/mcp/auth.go`、`auth_test.go`，仅删除这些文件，保留改造后的 `internal/infrastructure/mcp/delivery/` 子包。
- 删除 `internal/application/auth/usecase/mcp_grant.go`、专用测试和 `dto/mcp.go`；删除 auth service 的 grant store 初始化，以及 handler/routes 中两个 `mcp-grant` 入口。
- 删除 Web `MCPAuthorize.vue`、`MCPCallback.vue`、对应路由、`MCPGrantResponse`、`createMcpGrant` 和专属 i18n 文案。
- 更新登录重定向测试，使用仍存在的受保护页面验证一般 redirect 行为，不为已删除的业务建立新断言。保留登录和未授权处理公共工具。
- 不删除用户配置目录里的真实 `mcp-token` 文件，不读取或迁移其中内容；代码不再消费它即可。清理个人旧文件由用户自行决定。

### 5. 交付配置和活文档

- 更新 `.codex/config.toml`、`.env.example` 和 `configs/config.yaml`，清除旧 handoff 设置；示例不包含真实凭据，不自动改写用户 `.env.*`。
- 操作指南明确完整步骤：登录后创建 MCP PAT、一次性复制到父进程变量、启动 Codex/stdio，以及到期或撤销后的替换和重启；保留同环境 DB/workspace/Worker 前提，不能只写“配置一个 token”。
- 更新 MCP 操作指南、客户端调用机制和后端架构中的相关链路；决策账本记录旧浏览器交接和 HTTP 回环被替代，不改写相邻历史 SpecFlow 文档。
- 扫描当前运行时代码、配置和活指南中的 `mcp-grant`、`BrowserAuthorizer`、`TokenStore`、`MCPAPIUrl`、`StreamableClientTransport` 等遗留引用，区分历史文档与现行承诺，不作无关清理。

### 6. 实现自检与阶段交付

- 执行下述必需检查和定向自动化，汇报实际 diff、已运行/未运行检查及凭据生命周期限制。
- Implementation 完成后停留在实现阶段，等待用户明确进入 Verification；届时再形成同名验证文档和验收结论。此 Plan 不等于已授权实现或完整验收。

## Files to change

| 范围 | 预期变更 |
| --- | --- |
| `internal/application/dialogue/port/port.go`、`usecase/service.go`、相关测试 | 删除 Authorization 传递，改为 actor 工厂契约。 |
| `internal/api/http/handler/dialogue/`、`routes/routes.go` | 入站 actor 传递；校准内部调用/事务注释，保持原有 UoW 边界。 |
| `internal/infrastructure/mcp/delivery/client.go`、`client_test.go` | 内存 transport、双端生命周期、隔离与错误测试。 |
| `internal/api/mcp/delivery/server.go`、`types.go`、`server_test.go` | 每次 stdio 调用认证、固定 actor 同步；复用工具注册。 |
| `internal/bootstrap/app.go`、`http.go`、相关测试 | 新认证回调、内存 Core 工厂、移除 HTTP MCP 装配。 |
| `internal/api/http/server.go`、相关测试 | 移除 MCP handler 参数与 mux 分支。 |
| `internal/config/config.go`、相关测试 | 新输入变量、删除旧 MCP URL/timeout 设置。 |
| `internal/model/user.go`、`internal/repository/auth.go`、`internal/application/auth/**`、`internal/repository/impl/sqlc/auth/**`、`sql/{schema,query,migration}/**` | MCP PAT 的哈希持久化、用户状态认证、创建/列出/撤销和三种数据库迁移。 |
| `proto/orbit/v1/auth/auth.proto`、`internal/api/http/handler/auth/**`、`internal/api/http/routes/auth.go`、`web/src/{api,views,router,navigation,i18n}/**` | 已认证用户的 PAT 管理 API 与页面，明文 token 仅在创建结果中展示一次。 |
| `internal/infrastructure/mcp/auth*`、`internal/application/auth/**/mcp*`、`internal/api/mcp/delivery/http*` | 按上述精确文件列表删除废弃代码及专属测试，不递归删父目录。 |
| `web/src/views/auth/MCP*.vue`、`api/auth/auth.ts`、`router/index.ts`、`i18n/locales/{zh-CN,en-US}.ts`、`utils/login-redirect.test.ts` | 删除专属页面与引用，保留一般登录行为。 |
| `.codex/config.toml`、`.env.example`、`configs/config.yaml` | stdio 显式 credential 配置；不提交 token。 |
| `docs/guides/{mcp-direct-operations,client-cloud-invocation}.md`、`docs/architecture/backend.md`、`docs/decisions/ledger.md` | 更新本次相关活文档，不顺带清理其他架构描述。 |

新增 MCP PAT 所需的 SQL、三种新 migration 和 auth Proto DTO；不修改已执行 migration、用户/权限模型或 MCP tool 业务 schema。生成 SQLC 与 Go/TypeScript Proto 产物必须由项目命令产生。

## Verification plan

### 自动化行为证据

| 行为 | 最小有效证据 |
| --- | --- |
| Web actor 传递 | handler/usecase 测试只传已认证用户 ID；无权限请求不调用 Dialogue；保留一次完整 LLM 工具调用与 SSE 事件测试。 |
| 内存 MCP | 使用真实 SDK 内存 transport 列出工具并调用一个代表性工具，结果和 schema 保持现行契约；两个用户并发调用各自记录正确 actor。 |
| 生命周期 | 内存 client 连接失败、主动 Close、turn 取消时双方释放；不访问 HTTP `/mcp`，不等待远程网络超时。 |
| stdio discovery | 真正 stdio transport 子进程或测试 helper 使用临时 DB/config，无 token 仍完成 initialize/tools/list；stdout 仅 MCP 消息，工具调用失败且业务未执行。 |
| PAT 生命周期 | 使用真实 auth service 和临时用户仓储，创建后只返回一次明文、数据库只保留摘要；缺失、篡改、过期、撤销、用户不存在和禁用均拒绝下一次调用。 |
| 并发与项目授权 | 同一 stdio session 并发调用稳定绑定同一 actor；用户不属于目标项目时，现有 application 检查阻断调用，不因内部 transport 失去授权。 |
| 凭据输入 | config 测试模拟环境值与未设置值；验证 Codex 父子进程实际透传时只用哨兵假值，检查响应、日志和 stdout 不包含该值。 |
| 普通 Web 登录 | 登录重定向、401 处理、邮箱密码/CSRF/Turnstile 现有测试继续有效；不保留已删除 MCP grant/回调业务测试。 |

不穷举已删除协议的负向参数，也不把密码修改自动撤销 PAT 等当前未承诺行为写成验收断言。PAT 到期测试使用确定性时间写入或测试 token，不等待真实有效期。

### 必需命令

以下命令在后续 Implementation 自检或 Verification 阶段运行，本 Plan 阶段未运行：

```text
go test ./internal/api/mcp/delivery ./internal/infrastructure/mcp/delivery ./internal/application/auth/usecase ./internal/application/dialogue/usecase ./internal/config ./internal/bootstrap
yarn --cwd web test src/utils/login-redirect.test.ts src/utils/handle-unauthorized.test.ts src/router/index.test.ts
yarn --cwd web lint:fix
yarn --cwd web typecheck
task check
go test ./cmd/... ./internal/...
git diff --check
```

共享 actor 和会话生命周期涉及并发，额外对 MCP Core/内存适配器运行定向 `go test -race`；若当前 Windows/CGO 工具链不可用，记录未执行原因，使用正常并发测试且在具备 race 环境时补证据，不为此自动安装依赖。

### 人工验收

- 用户管理的现有 Web 服务中完成一次 Dialogue 查询和一个获授权的代表性写操作，确认无第二次登录、无访问 `/mcp`，审计主体为当前用户。涉及真实部署的验收另需用户明确授权。
- Codex 使用 stdio 配置，在隔离测试账号/环境中完成工具发现和只读项目查询；更换凭据后重建 session，确认 actor 与新凭据一致。不得读取用户已有秘密代为操作。
- 未经用户要求不启动、停止或重启开发服务、Worker 或现有 Codex MCP session；真实环境未提供时报告人工验收未完成，不能用模拟测试声称实际 Codex 端到端已通过。

## Blockers

- 当前无阻止编写 Plan 的事项。凭据交付和可信部署拓扑是本计划的显式假设，等待 Plan 审查，不再创建 OAuth 方案分支。
- 官方 Codex MCP 文档本轮 HTTP 获取返回 403，`env_vars` 拟定配置尚需在实施时依据安装版本帮助/schema 核实并验证透传。不能在配置未核实的情况下宣称首次使用已可用，也不能以把真实 token 写入仓库为退路。
- 若实际要求自动续期、无环境变量的凭据交付或异地 stdio 客户端，此计划不能满足，应先更新本功能需求/计划，而不是实现时悄悄加 OAuth 或 HTTP bridge。

## Risks

- PAT 仅作为本地 MCP actor 凭据，不能充当 Web API Bearer JWT。它仍是高权限的长期机密，操作者需在创建后立即存入环境/系统秘密管理，并在泄露时撤销。
- PAT 不随 Web logout 或改密自动撤销；禁用/删除用户、显式撤销或到期可阻断后续 stdio 调用。替换 PAT 后旧进程仍持有旧值，用户应重启该 session。
- `cfg.Jwt.SecretKey` 还用于 CI/CD credential 加密和 CSRF。不能把随意轮换该密钥当作本次 token 撤销方案，否则可能破坏存量加密凭据读取。
- 现有 JWT 无实例标识；同环境数据和密钥由部署保证，不承诺自动发现所有跨实例误配。不同实例不应共享验证密钥并复制同一用户数据作为身份隔离手段。
- stdio 仍直接使用 DB、Docker、workspace；这一点会限制远程工作站部署，环境凭据并不能解决该拓扑问题。Worker 和事务职责不随 transport 切换改变。
- Web 固定 actor 的有效范围是一轮已认证请求，不在 turn 内增加重新登录；中途禁用用户不承诺撤销已经执行的操作，项目资源操作继续经过现有成员检查。
- 删除 HTTP transport 后，普通静态站点 fallback 对未知 GET 路径可能返回页面，不能把页面响应误判成 MCP 端点仍可用；不为已删 `/mcp` 新建兼容或专用墓碑路由。

## Rollback

- 采用一次性发布收敛后的代码、前端和配置，无旧 grant/token file/HTTP transport 兼容开关；实施过程中的步骤拆分不代表同时交付两套认证。
- 本计划不新增 DB schema，不需要数据回填或历史迁移重写；不触碰用户旧 token 文件和其他私有配置。
- 发布出现阻断时停止推广并优先修复新实现。需要回退时由用户授权整套代码与配置回退到已知版本，不能在新版本内恢复旧认证分支；不自动执行 Git 回退或生产服务操作。

## User review notes

- 2026-09-06：用户要求“进入 plan”；Requirement 已标记 `Accepted`，本计划为 `Draft`，未开始 Implementation。
- 2026-09-06：最初计划待审查的“复用现有 24 小时 JWT 并由用户显式取得/注入”假设，已于 2026-09-07 被 MCP PAT 方案取代。
- 每次工具重新认证、完整删除 HTTP MCP、Web 内存 transport 及共享业务授权已按接受的 Requirement 安排，不重新提出与其冲突的选项。
- 2026-09-06：用户明确切换为标准模式 / standard 并开始 Implementation；本 Plan 视为 Accepted，实施后默认等待用户进入 Verification。
- 2026-09-07：用户要求继续优化，确认手工 Web JWT 注入不能作为实际可用的 CLI 凭据交付。实施范围改为 MCP PAT 管理与校验，不恢复旧浏览器授权路径。
