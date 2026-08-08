# 登录 CSRF Fernet 校验恢复需求
最后修改时间: 2026-08-08 14:22:57

Review status: Accepted

流程模式: 标准 / standard

## Background

Python 时代登录 CSRF 为基于密钥的短时令牌：签发后登录路径验签并拒绝过期令牌（历史上约 10 分钟 JWT 形态）。Go 迁移后该能力被掏空：

1. `GET /api/auth/csrf-token` 仅返回 `crypto/rand` 十六进制串，不使用密钥。
2. `POST /api/auth/login` 只检查 `csrf_token` 非空，不验签、不过期、不消费。
3. 前端登录页仍会拉取 CSRF 并提交，造成“有防护字段、无真实防护”的假象。

仓库已具备可复用的 Fernet 实现（`internal/common/crypto`，包名 `security`），Credential 等场景已用 `cfg.Jwt.SecretKey` 作为 Fernet 密钥。补 CSRF 应走 Fernet，而不是再造 JWT 形态的 CSRF。

## Goal

1. 恢复登录 CSRF 的真实签发与校验：后端用 Fernet 生成/验证 CSRF，令牌密码学有效期为 **3 分钟**（基于 Fernet 内置 timestamp + TTL）。
2. CSRF 明文 payload 含固定类型标记 `csrf`，防止与其它 Fernet 用途（如 Credential）混用。
3. 登录页完成初始化之前，**Login 按钮不可用**；初始化任务包含 **CSRF 必选**，以及 **Turnstile 配置**（与既有并行/一并完成后启用；若启用 Turnstile 还需完成 widget 就绪语义，细节在 Plan 落地）。
4. 初始化成功后启动 **3 分钟** 页面级 `setTimeout`：到期后 Login **失败态/不可继续登录**（按钮不可用或不可见，以实现一致为准），页面提示 **刷新后重试**；**不做**页面内静默续签 CSRF。
5. 登录提交必须携带当前 CSRF；后端拒绝无效、篡改、类型不符或过期令牌，并返回明确错误。
6. 删除“随机串 + 仅非空”实现；同步改掉硬编码 `csrf_token: "csrf"` 的测试/脚本/调用方，不保留兼容。

## Non-goal

- 不把 CSRF 做成 JWT（不复用 `internal/auth/jwt` 的 access token 形态签发 CSRF）。
- 不引入 Cookie 双提交、服务端 session 存储或 Redis 一次性 nonce 表。
- 不在页面存活期内自动静默刷新 CSRF。
- 不在本需求中改造 Turnstile / Google OAuth / MCP grant 的主体产品逻辑（仅保证登录主路径 CSRF 正确；复用 login 契约的入口一并遵守校验）。
- 不扩大到全局 API CSRF（仅覆盖登录相关 anti-token）。
- 不改 JWT access token 的 24h 有效期与签发逻辑。

## User Scenarios

1. 用户打开登录页：页面进入加载中状态，Login 按钮禁用，不可提交。
2. 登录页加载后请求 CSRF（可与 Turnstile config 一并初始化）；后端用 Fernet 与 `jwt.secret_key` 签发 3 分钟有效、payload 类型为 `csrf` 的令牌；初始化全部成功后启用 Login，并启动 **3 分钟** 页面超时计时。
3. 用户在 3 分钟内提交正确账号密码与当前 CSRF：登录成功。
4. 用户使用过期、篡改、伪造、类型不符或空 CSRF 提交：后端拒绝，不签发 access token；前端提示令牌无效/过期并引导刷新页面。
5. 用户停留满 3 分钟：前端 timer 触发，Login 不可继续使用，页面提示刷新重试；用户刷新后重新初始化。
6. CSRF 或必要初始化失败：Login 保持禁用，提示初始化失败；不得以空 token 放行提交。

## Acceptance

- [ ] `GET /api/auth/csrf-token` 返回 Fernet 密文；密钥为 `jwt.secret_key`；明文含类型标记 `csrf`；**不**使用随机 hex 占位。
- [ ] 过期判定使用 **Fernet 内置 timestamp**，max-age = **3 分钟**（扩展 `DecryptStringWithTTL` 或等价 API，而不是业务明文 `exp` 字段作为唯一时效来源）。
- [ ] 校验时同时验证：可解密、未超 3 分钟、明文类型为 `csrf`；任一项失败即无效。
- [ ] `POST /api/auth/login` 在账号密码校验前（或同等安全顺序）强制校验 CSRF；无效 CSRF 不得登录成功。
- [ ] 登录页：CSRF + Turnstile 配置等必要初始化未完成前，Login `disabled`。
- [ ] 初始化成功后 `setTimeout(3 * 60 * 1000)`：到期 Login 不可继续、提示刷新；`onBeforeUnmount` 清理 timer。
- [ ] **不做** 3 分钟内的静默 CSRF 续签。
- [ ] 所有硬编码 `csrf_token: "csrf"` 或跳过真实 CSRF 的测试/脚本/调用方改为先取 token 或使用测试辅助签发；无兼容旧任意字符串。
- [ ] 单元/集成测试：签发可解密、未过期可通过、过期拒绝、篡改拒绝、类型不符拒绝、空 token 拒绝；前端禁用/超时提示有测试或明确手工验收点。
- [ ] 清理死代码与重复生成路径，无新旧并存。

## Open Questions

暂无。先前 5 项均已由用户确认关闭。

## Decisions

- 补 CSRF **使用 Fernet**，**不使用 JWT CSRF**。
- CSRF 有效期与页面可登录窗口均为 **3 分钟**；到期 Login 不可继续，提示刷新重试；**无**静默续签。
- 过期：使用 **Fernet 内置 timestamp + TTL=3m**。
- 明文 payload：固定类型标记 **`csrf`**（采纳）。
- 初始化时序：CSRF 必选 + Turnstile 配置等必要项成功后才启用 Login（采纳）。
- 硬编码 `"csrf"` 的调用方：**全部改掉**（采纳）。
- 密钥源：`jwt.secret_key`（与 Credential 一致）。
- 项目约束：不做兼容层/默认值兜底/新旧逻辑并存。

## Risk

- `jwt.secret_key` 若非合法 Fernet key，Credential 与 CSRF 会同时失败；需与运维文档的 Fernet 格式要求一致。
- 3 分钟窗口较短，用户慢填表单可能触发刷新提示；属已确认产品行为。
- e2e/集成/脚本登录路径遗漏修改会直接失败。
- 前端 timer 必须在卸载时清理。

## User Review Notes

- 2026-08-08：排查登录页 anti-token 后确认 Go 侧 CSRF 为空壳；历史 Python 为密钥短时令牌，用户明确补逻辑不用 JWT、应用 Fernet。
- 2026-08-08：用户要求 SpecFlow 标准模式记录登录 CSRF 恢复。
- 2026-08-08：用户确认开放问题：
  1. CSRF 3 分钟有效；3 分钟后 Login 失败，页面提示刷新重试（取消原先 10 分钟页面超时设想）。
  2. 过期使用 Fernet 内置 timestamp/TTL。
  3. 采纳 payload 类型标记 `csrf`。
  4. 采纳初始化全部成功后才启用 Login。
  5. 改掉硬编码 `csrf_token: "csrf"` 的调用方。
- 2026-08-08：用户要求开始 plan，Requirement 标记为 Accepted。
