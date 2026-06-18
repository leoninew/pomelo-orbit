# Cloudflare Turnstile 验证码规格

Review status: Accepted

## Requirement basis

基于已接受的需求文档：

- `docs/requirement/20260614-cloudflare-turnstile-captcha.md`

关键要求：

- 使用 Cloudflare Turnstile 替换当前固定答案图片验证码。
- 增加 `turnstile.enabled` 开关，默认启用，可通过配置覆盖关闭。
- `enabled=true` 时登录必须通过 Turnstile 服务端验证。
- `enabled=false` 时登录不执行 Turnstile 校验，也不回退到旧图片验证码。
- 不保留 `captcha_token` / `captcha_answer` 兼容层。
- 不依赖 Python `backend/`。

## Overview

当前验证码实现是本地固定答案 `1234`，不具备真实防护能力。目标方案改为：

1. 后端暴露 Turnstile 配置接口，返回公开的 `site_key` 和 `enabled` 状态。
2. 前端登录页根据 `enabled` 决定是否渲染 Turnstile widget。
3. `enabled=true` 时，前端提交 Turnstile widget 生成的 token。
4. 后端登录接口使用 `secret_key` 调用 Cloudflare siteverify 验证 token。
5. 验证成功后继续用户名/密码校验。
6. `enabled=false` 时，前端不渲染 Turnstile，后端不要求 token。

## Design decisions

### 1. 配置开关默认启用

新增后端配置：

```yaml
turnstile:
  enabled: true
  site_key: "1x00000000000000000000AA"
  secret_key: "1x0000000000000000000000000000000AA"
  verify_url: "https://challenges.cloudflare.com/turnstile/v0/siteverify"
```

字段语义：

- `enabled`: 是否启用 Turnstile 登录校验，默认 `true`。
- `site_key`: 前端 widget 使用，可公开。
- `secret_key`: 后端 siteverify 使用，必须保密。
- `verify_url`: Cloudflare siteverify endpoint，默认官方地址，测试可覆盖为本地 httptest server。

环境变量：

- `POMELO_ORBIT_TURNSTILE__ENABLED`
- `POMELO_ORBIT_TURNSTILE__SITE_KEY`
- `POMELO_ORBIT_TURNSTILE__SECRET_KEY`
- `POMELO_ORBIT_TURNSTILE__VERIFY_URL`

校验规则：

- `turnstile.enabled=true` 时：
  - `site_key` 必须非空。
  - `secret_key` 必须非空。
  - `verify_url` 必须非空。
- `turnstile.enabled=false` 时：
  - 不强制要求 `site_key` / `secret_key`。
  - `verify_url` 可为空。

### 2. 开发默认使用 Cloudflare test keys

开发默认配置可使用 Cloudflare 官方测试 key：

- site key: `1x00000000000000000000AA`
- secret key: `1x0000000000000000000000000000000AA`

这样默认启用时仍能本地开发和测试真实 Turnstile 验证流程。

注意：生产环境必须覆盖为真实 widget 的 site key / secret key。

### 3. 配置接口替代图片验证码接口

当前 `GET /api/auth/captcha` 返回 `{ token, image }`。

目标改为新的配置接口：

```http
GET /api/auth/turnstile-config
```

响应：

```json
{
  "enabled": true,
  "site_key": "1x00000000000000000000AA"
}
```

设计理由：

- `site_key` 可公开，由后端返回便于不同环境配置。
- 不继续使用 `/api/auth/captcha`，避免旧图片验证码语义残留。
- 不保留旧接口兼容层。

### 4. 登录 DTO 直接切换为 turnstile_token

当前：

```json
{
  "username": "admin",
  "password": "admin",
  "csrf_token": "...",
  "captcha_token": "...",
  "captcha_answer": "1234"
}
```

目标：

```json
{
  "username": "admin",
  "password": "admin",
  "csrf_token": "...",
  "turnstile_token": "..."
}
```

后端 `loginReq`：

```go
type loginReq struct {
    Username       string `json:"username"`
    Password       string `json:"password"`
    CSRFToken      string `json:"csrf_token"`
    TurnstileToken string `json:"turnstile_token"`
}
```

校验：

- `username`、`password`、`csrf_token` 始终必填。
- `turnstile.enabled=true` 时，`turnstile_token` 必填。
- `turnstile.enabled=false` 时，`turnstile_token` 不要求。

### 5. Turnstile verifier

新增后端内部 verifier，负责调用 Cloudflare siteverify。

建议接口：

```go
type TurnstileVerifier interface {
    Verify(ctx context.Context, token string, remoteIP string) error
}
```

实现：

```go
type cloudflareTurnstileVerifier struct {
    secretKey string
    verifyURL string
    client    *http.Client
}
```

siteverify 请求：

```http
POST https://challenges.cloudflare.com/turnstile/v0/siteverify
Content-Type: application/json
```

请求体：

```json
{
  "secret": "...",
  "response": "turnstile-token",
  "remoteip": "client-ip"
}
```

响应体只关心核心字段：

```go
type turnstileVerifyResp struct {
    Success    bool     `json:"success"`
    ErrorCodes []string `json:"error-codes"`
    Hostname   string   `json:"hostname"`
    Action     string   `json:"action"`
}
```

验证规则：

- HTTP 请求失败：返回验证失败 error。
- 非 2xx：返回验证失败 error。
- JSON 解析失败：返回验证失败 error。
- `success=false`：返回验证失败 error。
- `success=true`：验证通过。

日志：

- 验证失败可记录 error code、hostname、action。
- 不记录 `secret_key`。
- 不记录完整 Cloudflare response 中可能不必要的敏感字段。

### 6. remote IP 获取

后端请求 Cloudflare siteverify 时传入 `remoteip`。

建议取值：

- 优先使用 `r.RemoteAddr` 中的 host 部分。
- 当前 HTTP middleware 已使用 `middleware.RealIP`，后续也可以从 request context/headers 读取真实 IP；实现阶段应复用当前 chi RealIP 行为。

### 7. 登录校验顺序

后端 `login` 推荐顺序：

1. 解析 JSON。
2. 校验 `username`、`password`、`csrf_token`。
3. 如果 `turnstile.enabled=true`：
   - 校验 `turnstile_token` 非空。
   - 调用 verifier。
   - verifier 失败则返回 `Invalid captcha`。
4. 查询用户并验证密码。
5. 签发 token。

理由：

- 先验证 Turnstile，可以减少无效登录对用户密码校验路径的消耗。
- 失败信息保持中性，不暴露具体 Cloudflare 验证细节。

### 8. 前端接入方式

前端不新增 npm 依赖，直接加载 Cloudflare script：

```html
https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit
```

`Login.vue` 目标行为：

- 页面加载时获取 CSRF token 和 Turnstile config。
- `enabled=true` 时渲染 Turnstile widget。
- widget callback 返回 token，写入状态。
- 登录前要求 token 非空。
- 登录失败后调用 `turnstile.reset(widgetId)`。
- `enabled=false` 时不渲染 widget，也不要求 token。

类型声明：

由于不引入 npm 包，需要在前端添加最小全局类型声明，例如：

```ts
declare global {
  interface Window {
    turnstile?: {
      render: (container: string | HTMLElement, options: Record<string, unknown>) => string
      reset: (widgetId?: string) => void
      remove: (widgetId: string) => void
    }
  }
}
```

### 9. 前端 API / types

`frontend/src/api/auth.ts`：

- 移除或替换 `getCaptcha()`。
- 新增：

```ts
getTurnstileConfig(): Promise<TurnstileConfigResp>
```

`frontend/src/types/auth.ts`：

```ts
export interface TurnstileConfigResp {
  enabled: boolean
  site_key: string
}

export interface LoginReq {
  username: string
  password: string
  csrf_token: string
  turnstile_token?: string
}
```

`frontend/src/stores/auth.ts`：

- 登录参数从 `captchaToken/captchaAnswer` 改为 `turnstileToken`。
- `enabled=false` 时可不传 `turnstile_token`。

### 10. i18n 和 UI 文案

将旧“图片验证码”语义调整为 Turnstile：

- 中文：
  - `请完成人机验证`
  - `人机验证失败`
  - `人机验证已过期，请重试`
- 英文：
  - `Please complete verification`
  - `Verification failed`
  - `Verification expired, please try again`

### 11. 测试策略

后端：

- config 测试：
  - 默认 `enabled=true`。
  - `enabled=false` 可覆盖。
  - `enabled=true` 时缺少 key 报错。
  - `enabled=false` 时允许 key 为空。
- verifier 测试：
  - httptest server 模拟 Cloudflare success。
  - httptest server 模拟 `success=false`。
  - httptest server 模拟非 2xx。
  - httptest server 模拟非法 JSON。
- auth 测试：
  - enabled=true 且 token 缺失返回 400。
  - enabled=true 且 verifier 失败返回 400。
  - enabled=true 且 verifier 成功后可正常登录。
  - enabled=false 时不要求 token，正常登录。

前端：

- 如果已有测试框架覆盖登录页，则补充：
  - enabled=true 时渲染 Turnstile 容器并要求 token。
  - enabled=false 时不渲染 Turnstile。
- 变更后必须执行：
  - `yarn lint --fix`
  - `yarn typecheck`

## Affected components

### Backend

- `backend-go/config.defaults.yaml`
- `backend-go/internal/config/config.go`
- `backend-go/internal/config/config_test.go`
- `backend-go/internal/httpserver/auth.go`
- `backend-go/internal/httpserver/server.go`
- `backend-go/internal/httpserver/server_test.go`
- 新增 Turnstile verifier 文件，例如：
  - `backend-go/internal/httpserver/turnstile.go`
  - `backend-go/internal/httpserver/turnstile_test.go`

### Frontend

- `frontend/src/views/Login.vue`
- `frontend/src/api/auth.ts`
- `frontend/src/types/auth.ts`
- `frontend/src/stores/auth.ts`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`
- 可能新增全局类型声明文件，例如：
  - `frontend/src/types/turnstile.ts`

## Interfaces

### `GET /api/auth/turnstile-config`

Response:

```json
{
  "enabled": true,
  "site_key": "1x00000000000000000000AA"
}
```

### `POST /api/auth/login`

Request when enabled:

```json
{
  "username": "admin",
  "password": "admin",
  "csrf_token": "csrf-token",
  "turnstile_token": "token-from-widget"
}
```

Request when disabled:

```json
{
  "username": "admin",
  "password": "admin",
  "csrf_token": "csrf-token"
}
```

Response unchanged:

```json
{
  "access_token": "...",
  "token_type": "bearer"
}
```

## Technical questions

暂无阻塞实现的问题。

需在 Plan 阶段进一步细化：

- Turnstile script 动态加载是否抽为 composable。
- fake verifier 注入点放在 `Server` 结构体还是 package-level helper。

## Risks

- Turnstile 依赖 Cloudflare 外部服务，登录链路会新增网络依赖。
- `enabled=false` 会关闭登录验证码校验，只适合开发或明确受控环境。
- 默认启用时，生产环境必须覆盖官方测试 key，否则实际防护不成立。
- 中国大陆或受限网络环境可能无法加载 Turnstile widget 或访问 siteverify。
- 直接删除旧图片验证码会要求前后端同步发布。

## Alternatives

### 保留旧图片验证码作为 fallback

不采用。项目当前阶段不做兼容层；关闭 Turnstile 时直接不做验证码校验，而不是回退到固定答案图片验证码。

### 前端环境变量注入 site key

不采用为默认方案。site key 由后端配置接口返回，便于环境统一管理。

### 默认关闭 Turnstile

不采用。用户要求默认启用，可覆盖配置关闭。

## User review notes

- 用户要求严格模式 / strict。
- 用户要求添加开关，默认启用，可以覆盖配置关闭。
- 用户要求删除 `GET /api/auth/captcha`。
- 用户要求其他设计遵从主流实践。
- 用户要求进入 Spec / 规格阶段。
