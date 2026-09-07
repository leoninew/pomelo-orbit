# MCP 调用认证边界重构需求
最后修改时间: 2026-09-07 13:31:14

Review status: Accepted

Flow mode: standard

## Background

当前实际使用场景只有两类：

1. 用户在 Web Dialogue 页面发起部署对话。浏览器请求 `/api/dialogue`，Orbit 服务端认证当前用户后，由 Dialogue Service 调用 MCP 工具。
2. 用户在 Codex CLI 中使用 `pomelo-orbit-mcp`。Codex 启动 `go run ./cmd/server mcp`，通过 stdio 调用本地 Orbit MCP Server。

当前实现存在两处不必要的认证耦合：

- Web Dialogue 先认证用户，再把原始 `Authorization` 转发给 Orbit 自己的 HTTP `/mcp` 入口。
- stdio 在首次 `tools/call` 时打开浏览器，通过 loopback 和 MCP grant 获取 JWT，并将 JWT 保存到本地 `mcp-token` 文件。
- 仅改为由用户手动注入 24 小时 Web JWT，虽然消除了旧的非标准授权流程，却把 MCP 的长期配置降级成每日重新获取和替换 Web 会话凭据。

当前没有“Codex CLI 直接连接远程 `/mcp`”的产品场景。因此，本需求不建设远程 MCP OAuth Authorization Server，也不引入 OIDC。MCP stdio 没有 OAuth 握手协议，认证应采用显式的进程凭据；Web Dialogue 则应使用已认证的 Orbit 用户身份进行内部 actor 绑定。

## Goal

1. Web Dialogue 认证一次后，直接以当前用户作为 actor 调用共享 MCP Core，不再通过 HTTP `/mcp` 自调用。
2. Codex CLI stdio 使用显式配置的、仅用于 MCP 的 Orbit Personal Access Token（PAT）完成 actor 认证，不自动打开浏览器，不实现私有 OAuth 交接协议，也不复用短期 Web JWT。
3. stdio MCP 的凭据校验与用户状态校验遵循清晰、可测试的会话边界，避免无凭据或失效凭据执行工具。
4. Web Dialogue 和 stdio 继续共用同一套 MCP tool registry、application usecase、项目成员检查和业务授权规则。
5. 直接删除当前 MCP 浏览器 grant、loopback callback、本地 token 文件和未被实际场景使用的远程 HTTP MCP 认证路径，不提供兼容层。

## Non-goal

- 不实现 OAuth 2.0/2.1 Authorization Server。
- 不实现 OIDC Provider、ID Token、UserInfo 或外部身份提供商接入。
- 不实现远程 Streamable HTTP MCP 的 OAuth discovery、PKCE、动态客户端注册或 token introspection。
- 不改变用户、角色、权限码、项目成员和 application usecase 的领域模型。
- 不重构 Web 邮箱密码登录，只调整 Dialogue 内部的 MCP 调用边界。
- 不为 Codex CLI 实现用户名密码登录、Turnstile 绕过、令牌自动续期、浏览器 cookie/localStorage 读取或本地 credential cache。
- 不保留旧的 `/api/auth/mcp-grant`、`/api/auth/mcp-grant/exchange` 或等价浏览器授权接口。
- 不保留旧的 MCP HTTP Bearer 入口作为兼容路径。

## User scenarios

1. 用户已登录 Web 并拥有 `dialogue:write` 权限时，发起 Dialogue 不会触发第二次 MCP 登录；所有 MCP tool 调用绑定到该 Web 用户。
2. Web Dialogue 调用 MCP 工具时，不能通过请求参数或模型输入改变 actor；项目资源仍由现有项目成员检查限制。
3. 用户在已登录的 Orbit Web 控制台生成命名 MCP PAT，仅在创建响应中复制一次，并将其配置为 Codex stdio 父进程环境变量。
4. Codex CLI 启动 stdio MCP 时，凭据由 MCP 客户端通过环境变量或等价进程配置注入；同一 PAT 可长期使用直到用户撤销或其显式到期。
4. Codex 执行 `initialize` 或 `tools/list` 时，不打开浏览器、不监听 loopback、不写本地凭据文件。
5. Codex 首次及后续 `tools/call` 均不能在缺少、过期、签名错误或对应用户已禁用时执行 Orbit 业务用例。
6. 用户修改 stdio 配置中的凭据后，重启 MCP session 即可使用新凭据；旧 session 不保留旧 actor 的隐式状态。

## Acceptance

### Web Dialogue

- [ ] `/api/dialogue` 只在 HTTP 入站边界认证当前用户，并将用户 ID 传入 Dialogue Service。
- [ ] Dialogue Service 使用进程内、用户 actor 绑定的 MCP Core 或等价内部适配，不再将原始 Web `Authorization` 转发到 `/mcp`。
- [ ] Web Dialogue 与 stdio 注册完全相同的 MCP tools，并调用相同 application usecase。
- [ ] Dialogue 的 `dialogue:write` 权限、项目成员检查和领域错误行为保持有效。
- [ ] Web Dialogue 不依赖 MCP grant、浏览器授权页、loopback callback 或本地 token 文件。

### Codex CLI stdio

- [ ] stdio 仅从明确约定的环境变量或等价进程凭据读取 Orbit MCP PAT；变量名称和配置注入方式在 Plan 中确定。
- [ ] 已登录用户可创建、列出和撤销自己的命名 MCP PAT；明文 token 只在创建响应中返回一次，持久层只保存不可逆摘要。
- [ ] PAT 可配置显式到期时间；撤销或到期后，下一次 `tools/call` 必须被拒绝。
- [ ] stdio 不实现 OAuth authorize、token exchange、loopback callback、浏览器启动或本地 credential cache。
- [ ] `initialize` 和 `tools/list` 不产生认证交互或业务副作用；首次真实 `tools/call` 前必须建立有效 actor。
- [ ] 每次 `tools/call` 都校验 credential 的签名、过期时间、用户存在性和 enabled 状态；失效时不得调用 application usecase。
- [ ] 一个 stdio session 的所有工具调用只能绑定到凭据验证得到的用户，不能由工具参数指定或覆盖 actor。
- [ ] stdio 认证失败以安全的 MCP tool error 返回，不向 stdout、MCP 响应或日志输出 credential 内容。
- [ ] stdio 运行环境能够使用与 Orbit 服务一致的用户数据和 credential 验证配置；环境不一致时明确失败，不静默降级。

### Removal and contract convergence

- [ ] 删除 `mcp-grant` 路由、DTO、内存 grant store、BrowserAuthorizer、loopback callback 和 `mcp-token` TokenStore。
- [ ] 删除当前远程 `/mcp` Streamable HTTP 入口及其自定义 Bearer 认证适配；Web Dialogue 不再依赖该入口。
- [ ] 删除旧 MCP 授权 Web 页面、路由、API client、配置项和相关文档；不保留别名、回退或双路径。
- [ ] 更新 `.codex/config.toml`、环境示例、MCP 指南和客户端调用文档，准确说明 Web Dialogue 内部 actor 绑定与 stdio 显式凭据。
- [ ] 自动化测试覆盖 Web actor 绑定、stdio discovery 无认证副作用、缺失/过期/错误 credential、禁用用户、actor 不可覆盖和业务失败阻断。

## Open questions

1. stdio 进程是否只允许运行在与 Orbit 数据库和 credential 验证配置一致的主机上？若需要从任意工作站连接远程 Orbit，当前“直接执行本地 application usecase”的架构必须另行调整。
2. 删除远程 `/mcp` 后，是否保留一个明确标记为内部测试的 transport adapter，还是完全删除 HTTP MCP 代码和测试？默认按完全删除处理。

## Decisions

- 当前决策：标准模式 / `standard`，按 Requirement → Plan → Implementation → Verification 推进。
- 当前决策：当前产品场景不包含 Codex CLI 直连远程 `/mcp`，不为该场景建设 OAuth2/OIDC。
- 当前决策：Web Dialogue 走内部 actor 绑定，不重复执行 MCP 网络认证。
- 当前决策：Codex CLI 使用 stdio 显式进程凭据；stdio 不实现 OAuth 浏览器授权。
- 2026-09-07：为消除手工轮换 24 小时 Web JWT 的使用断层，stdio 凭据确定为用户自行管理的 MCP PAT。PAT 采用高熵随机值，数据库只保存 SHA-256 摘要，可选到期、可随时撤销，且不能调用普通 Web API。
- 当前决策：删除现有 MCP grant、loopback、token file 和远程 HTTP Bearer 认证路径，不提供兼容层。
- 当前决策：MCP tools、application usecase、项目成员检查和业务授权是两种入口的共同执行边界。

## Risk

- PAT 一经在创建时复制便应作为长期敏感凭据处理；遗失无法找回，只能撤销并新建。环境变量仍是明文进程配置，操作者应使用系统秘密管理能力且不提交到仓库。
- 当前 stdio 直接装配本地数据库和 application usecase。若 Codex 在非 Orbit 主机运行，用户数据和 credential 验证配置可能不一致。
- 删除远程 `/mcp` 会影响任何未记录在当前产品场景中的调用者；这是明确的不兼容变更，不能通过保留旧入口规避。
- Web Dialogue 改为进程内 MCP Core 后，需要确保 tool registry 和 actor 绑定与现有 MCP server 行为一致，避免出现 Web 与 stdio 权限分叉。
- 当前 JWT 签发和验证逻辑不是专用 stdio credential 体系；若继续使用，必须在 Spec 阶段明确有效期、撤销、轮换和多实例隔离。

## User review notes

- 2026-09-06：用户明确要求“进入 plan”，按 SpecFlow 将当前 Requirement 标记为 `Accepted` 并进入 Plan，不开始实现。
- 当前没有该功能的独立 Spec 文档；按用户指定进入目标阶段，不补写或追认未经审查的 Spec。
- Open questions 保留讨论记录，其处理方式、假设和风险由 [同名 Plan](../plan/20260906-mcp-standard-authentication.md) 承接，不再重复阻止阶段推进。每次 `tools/call` 重新认证和删除 HTTP MCP 以本需求 Acceptance 为准，不采用与之冲突的 TTL 缓存或旧 transport 保留方案。
- 2026-09-06：用户要求切换为标准模式 / standard 并开始 Implementation；Requirement 保持 Accepted，不补建 Spec。
- 2026-09-07：用户指出手动提取和每日替换 JWT 的体验问题，要求继续优化实现。以 MCP PAT 替换 Web JWT 复用，不恢复已删除的浏览器授权、回环或 token 文件。
