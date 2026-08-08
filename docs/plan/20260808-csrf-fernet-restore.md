# 登录 CSRF Fernet 校验恢复计划
最后修改时间: 2026-08-08 14:30:00

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

- [需求](../requirement/20260808-csrf-fernet-restore.md) 已接受。
- 后端用 Fernet（非 JWT）签发/校验登录 CSRF；明文类型固定为 `csrf`；有效期 **3 分钟**，过期判定用 **Fernet 内置 timestamp + TTL**。
- 密钥源：`cfg.Jwt.SecretKey`（与 Credential 加密一致）。
- 登录页：初始化完成前 Login 禁用；初始化成功后 3 分钟内可登录；到期 Login 不可继续并提示刷新；**不**静默续签。
- 硬编码 `csrf_token: "csrf"` 的测试/调用方全部改为真实签发；不做兼容层。

## 实施步骤

### 1. 扩展 Fernet：支持按内置 timestamp 校验 TTL

1. 在 `internal/common/crypto/fernet.go`（package `security`）新增：
   - `DecryptStringWithTTL(secretKey, encryptedValue string, maxAge time.Duration) (string, error)`
   - 在现有 HMAC 校验与解密路径上，读取 token 内 8 字节 big-endian timestamp；若 `now - ts > maxAge` 或 `ts` 明显异常（如远超 now，可允许极小 clock skew，例如 ≤60s）则返回明确错误（如 `fernet token expired`）。
2. 保持 `EncryptString` / `DecryptString` 行为不变（Credential 等调用方零改动）。
3. 在 `fernet_test.go` 覆盖：
   - 未过期 round-trip；
   - 通过包内 `encryptFernet(..., pastTimestamp)` 构造过期 token 并断言 TTL 拒绝；
   - 篡改仍拒绝；非法 key 仍拒绝。
4. 不引入第三方 Fernet 依赖。

### 2. 实现真实 CSRF 签发/校验（auth 域）

1. 重写 `internal/auth/csrf`（替换当前仅 `rand` hex 的 `NewCSRFToken`）：
   - 常量：`TokenTTL = 3 * time.Minute`、`payloadType = "csrf"`（明文即为该字符串，简单且可防与 Credential JSON 误用）。
   - `Issue(secretKey string) (string, error)`：`security.EncryptString(secretKey, payloadType)`。
   - `Verify(secretKey, token string) error`：`DecryptStringWithTTL(..., TokenTTL)`，且明文必须 **精确等于** `payloadType`；否则统一视为无效。
2. 扩展 `authsvc.Service`：
   - 增加字段 `secretKey string`；
   - `New(..., secretKey string)` 注入密钥（bootstrap / 测试全部改签名）；
   - `NewCSRFToken()` 调用 `csrf.Issue(s.secretKey)`；
   - `Login`：在用户名密码校验**之前**调用 `csrf.Verify`；失败返回明确 validation 错误（文案建议：`请求令牌无效或已过期，请刷新页面重试` 或英文等价，与 HTTP 错误契约一致），**不得**落成凭据错误以免掩盖 CSRF 问题。
3. `ValidateLoginInput` 仍要求 `csrf_token` 非空；真实校验在 `Login`（或抽出 `ValidateAndConsume` 仅校验，无服务端消费存储）。
4. 删除 usecase 内重复的 `crypto/rand`+hex 生成逻辑；禁止保留“非空即过”路径。
5. 错误类型：新增 `ErrInvalidCSRFToken`（`KindValidation` 或项目惯例的 400），handler 走现有 `WriteError`，不单独泄露密钥/内部解密细节。

### 3. 接线 bootstrap 与 HTTP 登录路径

1. `internal/bootstrap/http.go`：`authsvc.New(..., cfg.Jwt.SecretKey)`。
2. 其它 `authsvc.New` 调用点（如 `internal/api/http/security/authz_test.go`、auth 集成测试）同步传入 Fernet 测试密钥（可用现有 `AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=`）。
3. Handler 保持：
   - `GetCSRFToken` → `service.NewCSRFToken()`；
   - `Login` 仍先 `ValidateLoginInput`，再 Turnstile（若启用），再 `service.Login`（内部验 CSRF）。
4. **不改** Proto 字段形状：`CSRFTokenResp.token`、`LoginReq.csrf_token` 仍为 string。
5. MCP 本地授权走浏览器 grant，不经密码+CSRF 硬编码；确认无 `csrf_token:"csrf"` 残留即可。

### 4. 前端登录页：就绪门闩 + 3 分钟失效 UI

文件主改：`web/src/views/auth/Login.vue`，文案 `web/src/i18n/locales/{zh-CN,en-US}.ts`。

1. 状态：
   - `initLoading` / `ready`：初始化未成功前 Login **disabled**；
   - `sessionExpired`：3 分钟到时置 true；
   - `pageTimer`：`setTimeout(3 * 60 * 1000)`，`onBeforeUnmount` 必须 `clearTimeout`。
2. 初始化：`Promise.all([getCsrfToken, getTurnstileConfig])` → 应用配置 →（若启用）`renderTurnstile` 成功 → 置 `ready`、启动 3 分钟 timer。任一步失败：保持未 ready，toast/文案提示初始化失败与刷新。
3. Login 按钮：
   - `:disabled="loading || !ready || sessionExpired"`（或等价）；
   - `sessionExpired` 时按钮 **不可见**（`v-if`），并展示“会话/令牌已过期，请刷新页面重试”固定提示区。
4. 提交：
   - 无 ready / 无 csrf / 已 expired 时本地直接拦截；
   - **删除**登录失败后静默 `getCsrfToken()` 续签逻辑；CSRF 类失败与页面超时统一引导 **整页刷新**（`location.reload` 可选按钮，或文案要求用户刷新）。
   - 账号密码错误等非 CSRF 失败：若仍在 3 分钟窗口内，可重置 Turnstile 并允许重试，**不**换新 CSRF。
5. i18n 增加：`login.initFailed`、`login.sessionExpired`、`login.refreshRequired`（或合并一条）；去掉/替换硬编码中文 toast 中与令牌相关的零散文案，纳入 i18n。
6. 不新增自动续签 interval。

### 5. 修正测试与 e2e 硬编码 CSRF

1. `internal/application/auth/usecase/service_integration_test.go`：
   - `New(..., authTestSecretKey)`；
   - 登录前 `token, _ := service.NewCSRFToken()`，`LoginInput.CSRFToken = token`；
   - 增补用例：过期/错误 token 登录失败；合法 token 成功。
2. `internal/test/e2e/mysql_e2e_test.go`：先 `GET /api/auth/csrf-token` 解析 `token`，再 POST login；禁止 `"csrf"` 字面量。
3. `internal/api/http/security/authz_test.go`：更新 `authsvc.New` 参数；若测试走 Login，改用真实 CSRF。
4. 全仓再扫 `csrf_token":"csrf"` / `CSRFToken: "csrf"`，清零残留。
5. 可选：`internal/auth/csrf` 包测 Issue/Verify/过期/类型不符。

### 6. 质量门与文档回写

1. 命令（仓库约定）：
   - `go fmt ./cmd/... ./internal/...`
   - `go vet ./cmd/... ./internal/...`
   - `go test ./cmd/... ./internal/...`（至少 `./internal/common/crypto`、`./internal/auth/csrf`、`./internal/application/auth/...`、`./internal/test/e2e` 若环境允许）
   - 前端有改：`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`
2. 若有活文档提及 CSRF 为占位或 JWT 10 分钟，回写为 Fernet 3 分钟（优先 `docs/guides` 中登录/安全相关；archive 不改）。
3. 不改已执行迁移；本任务无 DB schema 变更。

## 预计涉及文件

| 区域 | 路径 |
| --- | --- |
| Fernet | `internal/common/crypto/fernet.go`、`fernet_test.go` |
| CSRF 域 | `internal/auth/csrf/csrf.go`（重写）、可选 `csrf_test.go` |
| Auth 用例 | `internal/application/auth/usecase/service.go`、`service_integration_test.go` |
| Bootstrap / HTTP | `internal/bootstrap/http.go`、`internal/api/http/handler/auth/handler.go`（若需错误映射微调）、`internal/api/http/security/authz_test.go` |
| E2E | `internal/test/e2e/mysql_e2e_test.go` |
| Web | `web/src/views/auth/Login.vue`、`web/src/i18n/locales/zh-CN.ts`、`en-US.ts` |
| 过程文档 | 本 plan；verification 待实现后 |

**预计不改**：`proto/orbit/v1/auth/auth.proto`、Credential 加解密调用点、Turnstile 服务端校验主体、JWT access token 签发。

## 验证计划

| 场景 | 预期 |
| --- | --- |
| GET csrf-token | 返回可被 `DecryptStringWithTTL(key, token, 3m)` 解出且明文为 `csrf` 的 Fernet token |
| 合法登录 | 先取 CSRF，3 分钟内登录成功得 access_token |
| 过期 CSRF | 用过去 timestamp 签发的 token 登录 → 400/validation，无 access_token |
| 篡改 / 错误 key / 明文非 csrf / 空 token | 全部拒绝 |
| 硬编码 `"csrf"` | 登录失败（回归旧行为已移除） |
| 登录页未 ready | Login disabled |
| 登录页 3 分钟后 | Login 不可见或不可用 + 刷新提示；无静默换票 |
| 密码错误且仍在窗口内 | 可重试，不自动换 CSRF |
| Credential 加密 | 既有 round-trip 测试仍绿（DecryptString 无 TTL 行为不变） |

## Blockers、假设与风险

- **Assumption**：CSRF 明文固定为字符串 `csrf` 即可满足类型隔离；不需要 JSON 或 nonce。
- **Assumption**：`jwt.secret_key` 在运行环境已是合法 Fernet key（运维文档要求）；若只是 ≥32 字符非 Fernet，签发会直接失败——属配置错误，不在本任务做兼容降级。
- **Assumption**：登录失败后的“刷新”以用户整页刷新为准，前端不提供后台自动 re-issue。
- **Risk**：3 分钟窗口对慢操作用户不友好，已是需求确认行为。
- **Risk**：`authsvc.New` 签名变更会打断所有构造点；实施时全量编译覆盖。
- **Risk**：e2e 若跳过 MySQL 环境，须在单元/集成路径仍覆盖 CSRF 真校验。
- **Scope**：不做一次性服务端 nonce 消费表；同一 token 在 3 分钟内可被重复提交（与旧 Python JWT CSRF 类似），可接受。

## Rollback

- 纯代码回退：恢复随机 CSRF + 非空校验即可（不推荐）；无迁移回滚。
- 若线上 `jwt.secret_key` 非 Fernet 导致无法签发，应修正密钥配置而非放宽校验。

## User Review Notes

- 2026-08-08：用户要求开始 plan；Requirement 自动 Accepted。
- 计划默认：TTL API 放在 `common/crypto`；CSRF 逻辑放在 `internal/auth/csrf`；Service 注入 secretKey；前端 3 分钟与后端一致且无静默续签。
- 2026-08-08：用户要求开始实现，Plan 标记为 Accepted。
