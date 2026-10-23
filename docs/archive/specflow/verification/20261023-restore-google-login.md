# Verification / 验证：恢复 Google 登录
最后修改时间: 2026-10-23 11:52:34

## 审查状态

Accepted

2026-10-23：用户反馈“实测无问题”，要求完成 SpecFlow 验证文件，确认实现交付并授权本次整体验证。真实 Google 登录结果来自用户实测；其余结论来自实现阶段已执行的开发检查和本次代码、差异核对。

## 意图对齐

依据 [意图](../intent/20261023-restore-google-login.md) A01–A06，已恢复 Go 后端 Google 授权、回调验证、账号复用或关联、新账号创建及 Orbit JWT 签发，前端接通 state 与站内目标恢复。密码登录、角色与 Project 成员隔离沿用既有业务规则。

原 503 原因是 Go 迁移留下占位路由。后续 Google `redirect_uri_mismatch` 要求控制台登记与应用一致的 `http://localhost:9020/google/callback`；用户随后确认实测无问题，真实登录待验收项闭合。本会话未读取或修改控制台配置。

## 计划对齐

依据 [计划](../plan/20261023-restore-google-login.md)，集中配置校验、认证用例与 Google 端口、OIDC 外部适配器、现有仓储与 SQLC 查询、HTTP 装配和前端回调均已交付。协议使用 state、nonce、S256 PKCE、ID token 验签与 10 分钟加密 cookie；Google 网络请求位于事务外，用户和历史写入在短事务内完成，提交成功后才返回 JWT。

用户明确撤销 README 和 HTTP logging 改动，最终 diff 中未包含这两项，计划实现记录已据实修正。配置操作说明保留在 `.env.example`。本次验证仅修改过程文档。

## 子任务交付与集成

不适用。本功能未拆分 task，由主会话串行完成各层实现并检查整体调用链。

## 实际差异摘要

- 核对基线为 `6bdc4880`，交付版本为当前工作区相对该基线的 diff，包含已暂存产品变更与本次验证文档；本次未执行 Git 暂存、提交或推送。
- 配置新增 Google 三项参数、启动校验和 Client Secret 设置脱敏；无配置时保持结构化 503。
- 新增 Google 认证 DTO、端口、用例及 OIDC 适配器，按 subject 优先、已验证邮箱其次的顺序解析身份，拒绝禁用账号与绑定冲突。
- 复用现有 OAuth 字段，增加 provider + subject 查询及生成代码；账号、最近登录时间与登录历史原子提交，没有迁移变更。
- 保留 GET `/api/auth/google`、POST `/api/auth/google/callback` 和前端 `/google/callback`，补齐 cookie、state、专用响应、事务装配与合法站内跳转。
- 新增业务、临时 SQLite 持久化与回滚、HTTP/OIDC 协议、配置和装配测试；引入 `coreos/go-oidc/v3` 及其间接依赖，将既有 `x/oauth2` 改为直接依赖。

## 预期与实际修改文件对比

| 预期范围 | 实际交付 | 核对结果 |
| --- | --- | --- |
| 配置与示例 | `internal/config/config.go`、`google_test.go`、`configs/config.yaml`、`.env.example` | 一致；本机被忽略的 `.env.development` 回调在实现阶段改为 9020，不随 Git 交付 |
| 认证用例及 Google 适配器 | `internal/application/auth/{dto,port,usecase}`、`internal/infrastructure/external/google` 及测试 | 一致；依赖同步于 `go.mod`、`go.sum` |
| 身份查询与持久化 | `sql/query/user/user.sql`、UserStore、SQLC 仓储实现及生成文件 | 一致；复用现有表结构 |
| HTTP 路由和装配 | auth handler、routes、bootstrap 及测试 | 一致；另同步 `internal/api/http/security/authz_test.go` 的既有替身方法 |
| 回调契约与前端 | auth proto、生成的 Go/TS DTO、`web/src/api/auth/auth.ts`、`web/src/views/auth/GoogleCallback.vue` | 一致；前端携带 state 与 cookie，清除地址栏授权参数 |
| 过程文档 | 同名意图、计划、验证 | 齐全；完成后按阶段归档到 `docs/archive/specflow/` |
| README、HTTP logging | 当前无变更 | 用户明确撤销，未纳入最终交付 |

## 验收标准检查清单

| 标准 | 证据与结果 |
| --- | --- |
| A01：配置与授权入口 | 通过。配置测试覆盖统一加载、禁用、缺项及 URL 校验；handler 测试覆盖配置缺失的 503 契约；实现阶段只读实测入口返回 Google 302，回调端口为 9020。 |
| A02：浏览器绑定与身份验证 | 通过。用例测试覆盖 state、缺失/篡改/过期会话及未验证邮箱；本地签名 JWT/JWKS 协议测试覆盖 PKCE 参数、issuer、audience、expiry、nonce、签名与缺失 ID token。 |
| A03：账号解析与权限边界 | 通过。用例测试覆盖 subject 复用、邮箱关联、新建、禁用、冲突和存储失败；SQLite 测试确认同一 subject 复用、旧密码与角色保留、新用户无密码与角色。代码核对未增加角色或 Project 成员写入。 |
| A04：事务与签发时序 | 通过。用例测试确认 Google 请求在事务前、提交失败不返回 token；SQLite 登录历史失败注入确认账号关联或创建回滚；bootstrap 测试确认回调排除请求级事务。 |
| A05：前端与真实登录 | 通过。核对前端传递 code/state、携带 cookie、清除 URL 参数及恢复合法站内目标，取消和异常分支保留可见反馈；站内目标工具轻量单测通过。真实 Google 登录由用户实测确认，未将失败分支代码核对记为逐项浏览器实测。 |
| A06：检查与错误边界 | 通过。下表记录已执行的质量检查；HTTP 测试确认 `{code,error,requestId}`，Google 适配器测试确认错误不传播上游敏感响应或 Google access token，配置错误不暴露 secret。通用 HTTP logging 沿用既有实现。 |

## 测试 / 命令结果

工具链依据 `go.mod`、`Taskfile.yml` 和 `web/package.json`：Go 1.25、Task、Vue/TypeScript、Corepack Yarn、Vitest，以及项目现有 SQLC/Proto 生成入口。下列开发检查已在实现阶段实际执行，证据沿用计划中的交付记录；本次仅完成文档与差异核对，未重复运行这些检查。

| 阶段与检查 | 结果 |
| --- | --- |
| 实现：`task sqlc`、`task proto` | 通过；buf 未配置依赖的提示不阻塞生成 |
| 实现：认证用例、Google 适配器、handler、config、bootstrap 针对性 Go 测试 | 通过；覆盖业务、临时 SQLite、JWKS 和 HTTP 协议边界 |
| 实现：`go test -count=1 ./cmd/... ./internal/...` | 通过 |
| 实现：`task test:go:unit -- -count=1` | 通过 |
| 实现：`task check` | 通过，0 issues；包含前端类型、lint、格式及 Go 格式、lint 检查 |
| 实现：`corepack yarn lint:fix`，以及 `task check` 内的 `corepack yarn typecheck`（`web/`） | 通过 |
| 实现：`corepack yarn test src/stores/auth.test.ts src/utils/login-redirect.test.ts`（`web/`） | 2 个文件、14 项轻量测试通过 |
| 实现：现有服务的只读授权入口检查 | 302 到 `accounts.google.com`，回调为 `http://localhost:9020/google/callback`，携带 state、nonce 和 S256 PKCE |
| 验证：意图、计划、代码与 `git diff HEAD` 核对 | A01–A06 均有对应交付和证据，README 与 logging 撤销已反映到文档 |
| 验证：`git diff --cached --check`、`git diff --check` | 通过；归档后再次检查文档格式及相对链接 |
| 用户实测：真实 Google 登录 | 用户明确反馈“实测无问题”，接受主流程结果 |

未额外运行 MySQL/PostgreSQL、Docker、远端 SSH 联调，它们不属于本功能的交付范围。未新增前端整页或端到端自动化测试，遵循项目轻量单测约定。本次未操作开发服务器生命周期。

## 遗漏或扩大的范围

没有验收范围遗漏。README 和 logging 由用户主动撤销；必要的 UserStore 测试替身同步、依赖和生成代码属于认证接入范围。原有未跟踪文档 `docs/intent/20261022-easytier-host-network-access.md` 与本次无关，未修改或归档。

## 风险

无阻塞交付的问题。其他部署地址仍需在 Google Cloud Console 登记对应的前端 `/google/callback`；本次真实登录验收覆盖用户当前本地环境，不推定其他部署环境已配置。

## 未完成事项

无。原真实账号授权待验收项已据用户实测闭合；Git 提交与发布未包含在本次授权范围。

## 结论

恢复 Google 登录的目标已完成，A01–A06 验收通过，交付接受。按项目约定将同名意图、计划、验证记录归档到 `docs/archive/specflow/`，保留相互引用。
