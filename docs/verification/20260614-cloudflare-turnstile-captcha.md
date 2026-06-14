# Cloudflare Turnstile 验证码验证

Review status: Accepted

## Requirement alignment

- [x] 使用 Cloudflare Turnstile 替换当前固定答案图片验证码。
- [x] 删除 `GET /api/auth/captcha`。
- [x] 新增 `GET /api/auth/turnstile-config` 返回 `enabled` 和 `site_key`。
- [x] 添加 `turnstile.enabled` 开关，默认启用，可通过配置覆盖关闭。
- [x] `enabled=true` 时登录必须提交并通过 `turnstile_token` 服务端验证。
- [x] `enabled=false` 时不执行 Turnstile 校验，也不回退旧图片验证码。
- [x] 不保留 `captcha_token` / `captcha_answer` 兼容层。
- [x] secret key 只保存在后端配置，不下发前端。
- [x] 不依赖 Python `backend/`。

## Spec alignment

- [x] 后端新增 `TurnstileConfig`，包含 `enabled`、`site_key`、`secret_key`、`verify_url`。
- [x] 后端配置支持环境变量覆盖 Turnstile 配置。
- [x] 后端在 `enabled=true` 时校验 Turnstile site key / secret key / verify URL 非空。
- [x] 后端新增 Cloudflare siteverify verifier，使用 JSON POST 请求。
- [x] 登录 DTO 改为 `turnstile_token`。
- [x] 登录校验在 `enabled=true` 时调用 verifier。
- [x] 登录校验在 `enabled=false` 时跳过 Turnstile。
- [x] 前端删除图片验证码输入和刷新逻辑。
- [x] 前端动态加载 Turnstile script 并显式渲染 widget。
- [x] 前端登录失败后 reset Turnstile widget。
- [x] 前端卸载时 remove Turnstile widget。
- [x] 前端 i18n 从 captcha 文案切换为 verification 文案。

## Plan alignment

计划中的实现步骤均已执行：

- [x] 扩展后端配置。
- [x] 更新后端配置测试。
- [x] 新增 Turnstile verifier。
- [x] 调整 Server 结构和构造。
- [x] 调整 auth.go 路由和 DTO。
- [x] 更新后端认证测试。
- [x] 更新前端类型和 API。
- [x] 新增前端 Turnstile 类型声明。
- [x] 更新 auth store。
- [x] 更新 Login.vue。
- [x] 更新 i18n 文案。
- [x] 删除旧 captcha 引用。
- [x] 执行后端 Go 测试和前端 lint/typecheck。

## Actual diff summary

### Backend

- `backend-go/config.defaults.yaml`
  - 新增默认 Turnstile 配置，默认启用并使用 Cloudflare 官方测试 key。
- `backend-go/internal/config/config.go`
  - 新增 `TurnstileConfig`。
  - 绑定 Turnstile 环境变量。
  - 增加启用时必填校验。
- `backend-go/internal/config/config_test.go`
  - 覆盖默认值、环境变量覆盖、启用/关闭校验。
- `backend-go/internal/httpserver/turnstile.go`
  - 新增 Cloudflare siteverify verifier。
- `backend-go/internal/httpserver/turnstile_test.go`
  - 覆盖 siteverify 成功、失败、非 2xx、非法 JSON。
- `backend-go/internal/httpserver/auth.go`
  - 删除固定图片验证码逻辑。
  - 删除 `/api/auth/captcha` 路由。
  - 新增 `/api/auth/turnstile-config`。
  - 登录请求改为 `turnstile_token`。
  - 登录时按 `turnstile.enabled` 决定是否调用 verifier。
- `backend-go/internal/httpserver/server.go`
  - `Server` 持有 Turnstile verifier。
- `backend-go/internal/httpserver/server_test.go`
  - 更新登录测试和 Turnstile 开关测试。
  - 覆盖 `/api/auth/captcha` 已删除。
- `backend-go/internal/app/mysql_e2e_test.go`
  - E2E 登录路径关闭 Turnstile，避免真实外部依赖。

### Frontend

- `frontend/src/types/auth.ts`
  - 删除 `CaptchaResp`。
  - 新增 `TurnstileConfigResp`。
  - `LoginReq` 改为 `turnstile_token?: string`。
- `frontend/src/api/auth.ts`
  - 删除 `getCaptcha()`。
  - 新增 `getTurnstileConfig()`。
- `frontend/src/stores/auth.ts`
  - 登录参数改为 `turnstileToken?: string`。
- `frontend/src/types/turnstile.ts`
  - 新增 Cloudflare Turnstile browser API 类型声明。
- `frontend/src/views/Login.vue`
  - 移除图片验证码 UI。
  - 动态加载 Turnstile script。
  - 根据后端 config 渲染/跳过 Turnstile。
  - 登录失败时 reset widget。
  - 组件卸载时 remove widget。
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`
  - 替换验证码文案为人机验证/verification 文案。

## Planned vs actual changed files

计划内文件：

- [x] `backend-go/config.defaults.yaml`
- [x] `backend-go/internal/config/config.go`
- [x] `backend-go/internal/config/config_test.go`
- [x] `backend-go/internal/httpserver/auth.go`
- [x] `backend-go/internal/httpserver/server.go`
- [x] `backend-go/internal/httpserver/server_test.go`
- [x] `backend-go/internal/app/mysql_e2e_test.go`
- [x] `backend-go/internal/httpserver/turnstile.go`
- [x] `backend-go/internal/httpserver/turnstile_test.go`
- [x] `frontend/src/views/Login.vue`
- [x] `frontend/src/api/auth.ts`
- [x] `frontend/src/types/auth.ts`
- [x] `frontend/src/types/turnstile.ts`
- [x] `frontend/src/stores/auth.ts`
- [x] `frontend/src/i18n/locales/zh-CN.ts`
- [x] `frontend/src/i18n/locales/en-US.ts`
- [x] `docs/verification/20260614-cloudflare-turnstile-captcha.md`

## Acceptance criteria checklist

- [x] `/api/auth/captcha` 不再注册，测试覆盖 404。
- [x] `/api/auth/turnstile-config` 返回配置。
- [x] `turnstile.enabled` 默认启用。
- [x] `turnstile.enabled=false` 时登录不要求 token。
- [x] `turnstile.enabled=true` 时缺少 token 返回 400。
- [x] `turnstile.enabled=true` 且 verifier 失败返回 400。
- [x] `turnstile.enabled=true` 且 verifier 成功后可继续登录。
- [x] 前端无旧 captcha API / DTO / UI 引用。
- [x] 前端 lint 和 typecheck 通过。

## Test results

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/backend-go && go fmt ./internal/config ./internal/httpserver ./internal/app && go test ./internal/config ./internal/httpserver ./internal/app
```

结果：通过。

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/backend-go && go test ./...
```

结果：通过。

第一次在 `backend-go` 目录误执行前端命令：

```bash
yarn lint --fix && yarn typecheck
```

结果：失败，原因是当前目录没有 `package.json`。

随后在正确目录执行：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn lint --fix && yarn typecheck
```

结果：通过。

## Missed or expanded scope

- 未保留旧图片验证码兼容层，符合计划。
- 未新增 npm 依赖，符合计划。
- 未实现开发 bypass 以外的多供应商兼容，符合计划。
- 默认配置使用 Cloudflare 官方测试 key，生产环境需要覆盖真实 key。

## Risks

- 默认启用 Turnstile 后，登录依赖浏览器加载 Cloudflare script 和后端访问 Cloudflare siteverify。
- `turnstile.enabled=false` 会关闭登录验证码校验，应只用于开发或受控环境。
- 生产环境必须覆盖测试 site key / secret key，否则不能提供真实防护。

## Conclusion

实现与已接受的 Requirement / Spec / Plan 对齐。后端目标测试、后端完整测试、前端 lint --fix 和 typecheck 均已通过。
