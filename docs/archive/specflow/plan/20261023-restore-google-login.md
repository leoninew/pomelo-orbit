# Plan / 计划：恢复 Google 登录
最后修改时间: 2026-10-23 11:52:34

## 审查状态

Accepted

## 意图依据与授权

依据 [意图](../intent/20261023-restore-google-login.md) A01–A06。用户“将功能补回来”授权本次 Implementation，先补齐意图及计划，再直接实施。工作基线 `6bdc4880`；用户已有未跟踪的 EasyTier 意图文档与本次工作无关。

## 技术设计与决策

- 集中增加 `GoogleConfig` 的 ClientId、ClientSecret、RedirectUri。三项均为空时功能不可用，任一已填则启动校验完整性和 URL；本地 HTTP 仅允许 localhost/loopback。secret 纳入系统设置脱敏。保留现有配置加载入口。
- `auth/usecase` 增加独立 Google 登录用例，复用现有认证依赖；通过认证端口调用 Google 适配器及现有短事务 runner。bootstrap 注入，Gin handler 只负责输入、cookie 和协议响应。
- Google 适配器使用 `golang.org/x/oauth2` 和 `coreos/go-oidc/v3`，固定 Google OAuth/JWKS 地址、RS256 验签、客户端 audience/issuer/expiry 与 nonce 校验，并限定 HTTP 请求超时。不将 Google 原始响应或 token 返回前端。
- 授权生成随机 state、nonce、PKCE verifier；带用途标记的会话经现有 Fernet 加密，10 分钟有效，通过 HttpOnly、SameSite=Lax cookie 传递，HTTPS 回调使用 Secure。回调校验 state 后交换授权码，清除 cookie。
- 扩展 `GoogleCallbackReq` 为 code + state，新增 Google 专用响应包含 Orbit token、token_type、redirect；回调目标存入加密会话，只接受站内相对路径，前端继续用现有 `resolveLoginRedirect` 校验路由。
- SQLC 增加按 provider + subject 查询，沿用现有 UserStore 和 OAuth 字段。查 subject → 查已验证邮箱 → 关联或创建；禁用账号、身份绑定冲突拒绝。新账号无可用密码、不自动授予角色/成员资格。
- 将 POST Google 回调排除在 HTTP 全请求事务外。Google 请求完成后，通过应用事务端口提交账号、最近登录时间与历史；提交成功后签发 JWT，失败回滚。
- 保持现有 `{code,error,requestId}` 错误出口，授权码交换依赖不可用为 503、无效授权或身份验证失败为 400、禁用为 403、绑定冲突为 409，内部错误安全输出。

## 实现步骤与预期文件

1. 配置模型、启动校验、默认 YAML/环境变量模板及本地回调端口修正；增加配置测试。
2. `internal/application/auth/{dto,port,usecase}`、`internal/infrastructure/external/google`：完成登录编排和 OIDC 适配，增加业务及协议测试。
3. `sql/query/user/user.sql`、UserStore、SQLC 生成与仓储实现：增加身份查询，验证账号关联/创建和回滚。
4. `internal/api/http/{handler/auth,routes/auth.go}`、bootstrap：装配登录用例、cookie、错误和短事务；增加协议与装配检查。
5. `proto/orbit/v1/auth/auth.proto`、生成的 Go/TS DTO、`web/src/{api/auth,views/auth/GoogleCallback.vue}`：接通 state 和登录后目标，使用现有生成入口。
6. 运行必要开发检查并 review 实际 diff，将结果、偏差及未完成项登记本计划。

## 子任务与协调

不适用。由当前主会话串行实现单一交付。

## 开发检查

- Go 核心业务测试使用事实端口；Google 适配器以本地 HTTP/JWKS 验证协议与签名，无真实账号或凭据。
- 真实临时 SQLite 验证 provider 查询、关联/创建、禁用账号与失败回滚，标记为本地集成测试。
- Gin 协议验证授权重定向、cookie、无效 state、禁用功能及安全错误契约。
- 运行 `task check`、`go test -count=1 ./cmd/... ./internal/...`、`task test:go:unit -- -count=1`，以及前端 `corepack yarn lint:fix`、`corepack yarn typecheck`；前端变更不引入整页或端到端测试。

## 风险、回滚与未完成项

真实 Google 控制台和交互授权需要外部人工确认；当前配置回调地址从 9002 修正为 9020 后，控制台也需登记相同地址。无需数据库迁移；回滚本次代码及配置即可。开发服务器不由本次任务重启。

## 实现记录

2026-10-23 10:58:17：前置记录完成，进入 Implementation。

2026-10-23 11:19:18：Implementation 已完成并交付，整体 Verification 尚未进入。

### 实际交付与范围

- 集中 Google 配置和校验已接入运行时；默认关闭，部分配置阻止启动。Client Secret 纳入系统设置的敏感字段，环境变量示例注明 Google 控制台回调登记要求。
- Google 用例、外部端口和 OIDC 适配器已装配。恢复按 subject 登录、已验证邮箱关联、创建账号、最近登录时间、历史与 JWT；新增用户没有密码凭据、角色或 Project 成员关系。
- 使用现有 Fernet 保护 10 分钟浏览器会话，验证 state、PKCE、nonce 和签名/issuer/audience/expiry；取消和失败返回登录页，成功恢复经过前端路由校验的站内目标。前端使用 router.replace 清除地址栏授权参数。
- POST 回调排除全请求事务；账号写入与历史使用现有短事务 runner，提交成功后返回 JWT。SQLite 失败注入验证关联和创建均回滚。
- 使用 `task sqlc` 和 `task proto` 重新生成查询及 Go/TS DTO。新增依赖 `coreos/go-oidc/v3 v3.16.0`，现有 `x/oauth2` 改为直接依赖；未改数据库结构或迁移。
- 旧 security 测试替身同步新增 UserStore 查询方法，属于接入所需范围。用户主动撤销 README 和 HTTP logging 的改动；这两项未纳入最终交付，日志行为沿用现有实现。
- 被 Git 忽略的本机 `.env.development` 回调已改为 `http://localhost:9020/google/callback`；现有 ID/Secret 未变更。该本地修改不会随 Git diff 交付。

### 开发检查结果

| 检查 | 结果与证据 |
| --- | --- |
| `task sqlc` / `task proto` | 通过；buf 提示未配置依赖可更新，两个生成步骤成功，无阻塞 |
| 认证、Google、handler、middleware、config、bootstrap 针对性 Go 测试 | 通过，包含临时 SQLite 持久化与回滚、本地 JWKS 验签和 HTTP 错误契约 |
| `go test -count=1 ./cmd/... ./internal/...` | 全范围复跑通过；初次暴露旧测试替身缺方法，已修正 |
| `task test:go:unit -- -count=1` | 通过，Google 用例只依赖事实端口，未将 SQLite/迁移引入核心单测 |
| `task check` | 最终通过，0 issues；初次静态检查报一个错误文本大小写问题，修正后 Google 包测试及完整质量检查通过 |
| `corepack yarn lint:fix`，以及 `task check` 内的 `corepack yarn typecheck` | 在 `web/` 执行并通过 |
| `corepack yarn test src/stores/auth.test.ts src/utils/login-redirect.test.ts` | 2 个文件、14 项轻量测试通过 |
| `git diff --check` | 通过 |
| 当前运行入口的只读开发检查 | GET `http://localhost:9020/api/auth/google` 返回 302，授权目标 `accounts.google.com`；回调为 9020，包含 state、nonce、S256 PKCE；未跟随跳转或操作真实账号 |

### 审查与待验收事项

实际 diff 与 A01–A06 的实现范围一致，额外文件为接入所需的既有替身同步和环境变量配置说明；没有扩大到其他 OAuth 提供商或权限模型。配置沿唯一加载入口、缺项提前校验；Gin 失败均调用现有 transport writer，协议测试确认 code/error/requestId。

真实 Google 控制台回调登记及浏览器账号授权尚未验证。MySQL/PostgreSQL、Docker、远端 SSH 等显式外部联调未额外启用。现有开发服务由原运行方式加载变更，本会话没有执行启动、停止或重启命令。按 SpecFlow 默认阶段边界交付实现和开发检查，后续进入整体 Verification 时继续使用本意图与计划。

## 整体验证与交付确认

2026-10-23 11:52:34：用户反馈“实测无问题”，并要求完成 SpecFlow 验证文件，授权进入 Verification。真实 Google 登录的待验收项据用户实测闭合；本会话未读取或修改 Google Cloud Console 配置。用户同时明确 README 和 logging 为主动撤销，最终范围以当前 diff 为准。

核对意图 A01–A06、计划、当前工作区差异及既有自动检查证据，整体交付接受。详见 [验证报告](../verification/20261023-restore-google-login.md)。本次只补齐验收记录并执行差异检查，未重复运行已通过的开发检查；没有产品代码变更。意图、计划、验证三份记录归档到 `docs/archive/specflow/` 的对应阶段目录。
