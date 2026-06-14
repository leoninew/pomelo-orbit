# Cloudflare Turnstile 验证码实施计划

Review status: Accepted

## Requirement / Spec basis

已接受文档：

- `docs/requirement/20260614-cloudflare-turnstile-captcha.md`
- `docs/spec/20260614-cloudflare-turnstile-captcha.md`

核心要求：

- 使用 Cloudflare Turnstile 替换当前固定答案图片验证码。
- 删除 `GET /api/auth/captcha`。
- 新增 `GET /api/auth/turnstile-config` 返回 `enabled` 和 `site_key`。
- 添加 `turnstile.enabled` 开关，默认启用，可配置关闭。
- `enabled=true` 时登录必须提交并通过 `turnstile_token` 服务端验证。
- `enabled=false` 时不执行 Turnstile 校验，也不回退旧图片验证码。
- 不保留 `captcha_token` / `captcha_answer` 兼容层。
- 遵从 Turnstile 主流实践：前端 widget 生成 token，后端 siteverify 验证，secret 仅在后端保存。

## Implementation steps

### 1. 扩展后端配置

修改 `backend-go/internal/config/config.go`：

- 在 `Config` 中新增：

```go
Turnstile TurnstileConfig `mapstructure:"turnstile" yaml:"turnstile"`
```

- 新增结构体：

```go
type TurnstileConfig struct {
    Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
    SiteKey   string `mapstructure:"site_key" yaml:"site_key"`
    SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
    VerifyURL string `mapstructure:"verify_url" yaml:"verify_url"`
}
```

- 在 `bindEnv` 中增加：
  - `turnstile.enabled`
  - `turnstile.site_key`
  - `turnstile.secret_key`
  - `turnstile.verify_url`
- 在 `Validate()` 中增加：
  - `turnstile.enabled=true` 时，`site_key`、`secret_key`、`verify_url` 必须非空。
  - `turnstile.enabled=false` 时，不强制要求 key。

修改 `backend-go/config.defaults.yaml`：

```yaml
turnstile:
  enabled: true
  site_key: "1x00000000000000000000AA"
  secret_key: "1x0000000000000000000000000000000AA"
  verify_url: "https://challenges.cloudflare.com/turnstile/v0/siteverify"
```

### 2. 更新后端配置测试

修改 `backend-go/internal/config/config_test.go`：

- 默认配置断言 `Turnstile.Enabled == true`。
- 默认配置断言 test site key / secret key / verify URL 非空。
- 环境变量覆盖测试覆盖：
  - `POMELO_ORBIT_BACKEND__TURNSTILE__ENABLED=false`
  - `POMELO_ORBIT_BACKEND__TURNSTILE__SITE_KEY`
  - `POMELO_ORBIT_BACKEND__TURNSTILE__SECRET_KEY`
  - `POMELO_ORBIT_BACKEND__TURNSTILE__VERIFY_URL`
- 校验测试覆盖：
  - enabled=true 且 site_key 空时报错。
  - enabled=true 且 secret_key 空时报错。
  - enabled=true 且 verify_url 空时报错。
  - enabled=false 且 key 为空时通过。

### 3. 新增 Turnstile verifier

新增文件：

- `backend-go/internal/httpserver/turnstile.go`

实现内容：

```go
type turnstileVerifier interface {
    Verify(ctx context.Context, token string, remoteIP string) error
}
```

实现 Cloudflare verifier：

```go
type cloudflareTurnstileVerifier struct {
    secretKey string
    verifyURL string
    client    *http.Client
}
```

构造函数：

```go
func newTurnstileVerifier(cfg config.TurnstileConfig) turnstileVerifier
```

siteverify 行为：

- 使用 `POST`。
- `Content-Type: application/json`。
- body 包含：
  - `secret`
  - `response`
  - `remoteip`
- 使用 `http.Client{Timeout: 3 * time.Second}`。
- 非 2xx 返回 error。
- JSON 解析失败返回 error。
- `success=false` 返回 error，并包含 error codes 信息。
- `success=true` 返回 nil。
- 不记录或暴露 `secret_key`。

### 4. 调整 Server 结构和构造

修改 `backend-go/internal/httpserver/server.go`：

- `Server` 增加字段：

```go
turnstileVerifier turnstileVerifier
```

- `New(...) Server` 中根据 `cfg.Turnstile` 创建 verifier。
- 为测试保留包内可覆盖方式：测试可直接构造 `Server{turnstileVerifier: fakeVerifier}`，或在 `New` 后覆盖字段。

### 5. 调整 auth.go 路由和 DTO

修改 `backend-go/internal/httpserver/auth.go`：

- 删除：
  - `const captchaAnswer = "1234"`
  - `getCaptcha`
  - `r.Get("/api/auth/captcha", s.getCaptcha)`
- 新增：

```go
r.Get("/api/auth/turnstile-config", s.getTurnstileConfig)
```

- 新增 response：

```go
type turnstileConfigResp struct {
    Enabled bool   `json:"enabled"`
    SiteKey string `json:"site_key"`
}
```

- 修改 `loginReq`：

```go
type loginReq struct {
    Username       string `json:"username"`
    Password       string `json:"password"`
    CSRFToken      string `json:"csrf_token"`
    TurnstileToken string `json:"turnstile_token"`
}
```

- 登录校验逻辑：
  - `username`、`password`、`csrf_token` 必填。
  - `cfg.Turnstile.Enabled == true` 时：
    - `turnstile_token` 必填。
    - 调用 `s.turnstileVerifier.Verify(r.Context(), req.TurnstileToken, clientIP(r))`。
    - 失败返回 `400 Invalid captcha`。
  - `enabled=false` 时跳过 Turnstile 校验。

- `clientIP(r)` helper：
  - 使用 `net.SplitHostPort(r.RemoteAddr)` 提取 host。
  - 失败时回退 `r.RemoteAddr`。

### 6. 更新后端认证测试

修改：

- `backend-go/internal/httpserver/server_test.go`
- `backend-go/internal/app/mysql_e2e_test.go`

主要调整：

- 不再请求 `/api/auth/captcha`。
- 请求 `/api/auth/turnstile-config` 并断言 200。
- 登录 body 改为：

```json
{"username":"admin","password":"admin","csrf_token":"csrf","turnstile_token":"test-token"}
```

- 测试 server 使用 fake verifier，避免真实 Cloudflare 调用。
- 增加覆盖：
  - enabled=true + missing token => 400。
  - enabled=true + verifier failure => 400。
  - enabled=false + no token => login success。

新增 verifier 测试：

- `backend-go/internal/httpserver/turnstile_test.go`
  - httptest server success。
  - success=false。
  - 非 2xx。
  - invalid JSON。

### 7. 更新前端类型和 API

修改 `frontend/src/types/auth.ts`：

- 删除 `CaptchaResp`。
- 新增：

```ts
export interface TurnstileConfigResp {
  enabled: boolean
  site_key: string
}
```

- 修改 `LoginReq`：

```ts
export interface LoginReq {
  username: string
  password: string
  csrf_token: string
  turnstile_token?: string
}
```

修改 `frontend/src/api/auth.ts`：

- 删除 `getCaptcha()`。
- 新增：

```ts
getTurnstileConfig(): Promise<TurnstileConfigResp> {
  return request.get('/api/auth/turnstile-config')
}
```

### 8. 新增前端 Turnstile 类型声明

新增：

- `frontend/src/types/turnstile.ts`

内容：

```ts
export {}

declare global {
  interface Window {
    turnstile?: {
      render: (
        container: string | HTMLElement,
        options: {
          sitekey: string
          callback?: (token: string) => void
          'expired-callback'?: () => void
          'error-callback'?: () => void
        }
      ) => string
      reset: (widgetId?: string) => void
      remove: (widgetId: string) => void
    }
  }
}
```

### 9. 更新 auth store

修改 `frontend/src/stores/auth.ts`：

- 登录参数改为：

```ts
async function login(username: string, password: string, csrfToken: string, turnstileToken?: string)
```

- 提交：

```ts
authApi.login({
  username,
  password,
  csrf_token: csrfToken,
  turnstile_token: turnstileToken,
})
```

### 10. 更新 Login.vue

修改 `frontend/src/views/Login.vue`：

- 移除图片验证码输入框和 `<img>`。
- 页面加载时调用：
  - `authApi.getCsrfToken()`
  - `authApi.getTurnstileConfig()`
- `turnstile.enabled=true` 时：
  - 动态加载 `https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit`
  - 渲染 widget 到容器。
  - callback 保存 token。
  - expired/error callback 清空 token。
- `turnstile.enabled=false` 时：
  - 不渲染 widget。
  - validate 不要求 token。
- 登录失败后：
  - 如果 enabled，调用 `window.turnstile.reset(widgetId)` 并清空 token。
- 组件卸载时：
  - 如果 widget 已渲染，调用 `window.turnstile.remove(widgetId)`。

遵从主流实践：

- site key 可公开，来自后端 config 接口。
- secret key 不进入前端。
- token 只在登录提交时使用。
- token 过期/失败后 reset widget。

### 11. 更新 i18n 文案

修改：

- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`

替换旧 captcha 文案为 Turnstile 语义：

中文建议：

- `verification`: `人机验证`
- `verificationRequired`: `请完成人机验证`
- `verificationFailed`: `人机验证失败，请重试`
- `verificationExpired`: `人机验证已过期，请重试`

英文建议：

- `verification`: `Verification`
- `verificationRequired`: `Please complete verification`
- `verificationFailed`: `Verification failed, please try again`
- `verificationExpired`: `Verification expired, please try again`

### 12. 删除旧 captcha 引用

全局搜索并删除/替换：

- `getCaptcha`
- `CaptchaResp`
- `captcha_token`
- `captcha_answer`
- `captchaAnswer`
- `/api/auth/captcha`

不保留兼容 alias。

### 13. 验证

后端：

```bash
cd backend-go
go fmt ./internal/config ./internal/httpserver ./internal/app
go test ./internal/config ./internal/httpserver ./internal/app
go test ./...
```

前端：

```bash
cd frontend
yarn lint --fix && yarn typecheck
```

注意：项目指令要求前端代码变更后必须执行 `yarn lint --fix && yarn typecheck`。

### 14. Verification 文档

实现完成后新增：

- `docs/verification/20260614-cloudflare-turnstile-captcha.md`

记录：

- requirement/spec/plan 对齐情况。
- 实际 diff 摘要。
- 后端测试结果。
- 前端 lint/typecheck 结果。
- 剩余风险。

## Files to change

### Backend

- `backend-go/config.defaults.yaml`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/httpserver/auth.go`
- `backend-go/internal/httpserver/server.go`
- `backend-go/internal/httpserver/server_test.go`
- `backend-go/internal/app/mysql_e2e_test.go`
- `backend-go/internal/httpserver/turnstile.go`
- `backend-go/internal/httpserver/turnstile_test.go`

### Frontend

- `frontend/src/views/Login.vue`
- `frontend/src/api/auth.ts`
- `frontend/src/types/auth.ts`
- `frontend/src/types/turnstile.ts`
- `frontend/src/stores/auth.ts`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`

### Docs

- `docs/verification/20260614-cloudflare-turnstile-captcha.md`

## Verification plan

后端目标测试：

```bash
cd backend-go
go test ./internal/config ./internal/httpserver ./internal/app
```

后端完整测试：

```bash
cd backend-go
go test ./...
```

前端强制检查：

```bash
cd frontend
yarn lint --fix && yarn typecheck
```

## Assumptions

- 默认启用 Turnstile，并使用 Cloudflare 官方测试 key 作为开发默认值。
- 生产环境会覆盖真实 `site_key` / `secret_key`。
- 关闭 Turnstile 是显式配置行为，主要用于开发或受控环境。
- 删除 `/api/auth/captcha` 不需要兼容旧前端。
- 前端可从浏览器访问 Cloudflare Turnstile script。

## Risks

- Turnstile 外部网络不可达会影响登录；这是默认启用带来的预期风险。
- `enabled=false` 会关闭登录验证码校验，不应在生产环境使用。
- 直接删除旧 captcha 接口要求前后端同步发布。
- 前端动态加载第三方 script，需处理加载失败、过期和重置。

## Rollback

如需回滚：

- 恢复旧 `captcha_token` / `captcha_answer` DTO。
- 恢复 `GET /api/auth/captcha` 和固定 SVG 验证码逻辑。
- 移除 Turnstile config / verifier / frontend widget 代码。

仅在用户明确要求回滚时执行；当前项目规范下不保留兼容层。

## User review notes

- 用户要求删除 `GET /api/auth/captcha`。
- 用户要求其他设计遵从主流实践。
- 用户要求进入 Plan / 计划阶段。
