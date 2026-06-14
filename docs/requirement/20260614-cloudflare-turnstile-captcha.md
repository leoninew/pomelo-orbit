# Cloudflare Turnstile 验证码可行性审视

Review status: Accepted

## Background

当前 `backend-go` 登录验证码仍是本地固定答案实现：

- `backend-go/internal/httpserver/auth.go` 中 `getCaptcha` 返回固定 SVG 图片，内容为 `1234`。
- `login` 请求体包含 `captcha_token` 和 `captcha_answer`。
- 后端只校验 `captcha_answer == "1234"`，`captcha_token` 仅要求非空，没有实际绑定或服务端验证。
- 前端 `frontend/src/views/Login.vue` 在登录页调用 `/api/auth/captcha` 获取图片验证码，并在登录时提交 `captcha_token` 与 `captcha_answer`。

这套实现适合开发占位，但不具备真实验证码防护能力。Cloudflare Turnstile 可以替代图片验证码，由前端 widget 生成一次性 token，后端调用 Cloudflare siteverify 接口验证 token。

## Goal

- 审视当前验证码功能是否可以改为 Cloudflare Turnstile。
- 明确前后端需要调整的接口、配置和验证流程。
- 添加 Turnstile 启用开关，默认启用，可通过配置覆盖关闭。
- 给出是否可行的结论和主要风险。

## Non-goal

- 本阶段不实现代码。
- 本阶段不申请 Cloudflare Turnstile site key / secret key。
- 本阶段不设计多验证码供应商兼容层。
- 本阶段不保留旧图片验证码作为兼容分支；如后续决定实现，应直接切换到目标方案。
- 关闭 Turnstile 时不回退到旧图片验证码，只是不执行验证码校验。

## Current implementation

### Backend

关键文件：

- `backend-go/internal/httpserver/auth.go`

当前路由：

- `GET /api/auth/csrf-token`
- `GET /api/auth/captcha`
- `POST /api/auth/login`

当前登录请求：

```go
type loginReq struct {
    Username      string `json:"username"`
    Password      string `json:"password"`
    CSRFToken     string `json:"csrf_token"`
    CaptchaToken  string `json:"captcha_token"`
    CaptchaAnswer string `json:"captcha_answer"`
}
```

当前校验：

```go
if strings.TrimSpace(req.CaptchaAnswer) != captchaAnswer {
    writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid captcha"})
    return
}
```

问题：

- 验证码答案固定为 `1234`。
- `captcha_token` 不参与真实校验。
- `/api/auth/captcha` 返回本地 SVG 图片，与外部验证服务无关。

### Frontend

关键文件：

- `frontend/src/views/Login.vue`
- `frontend/src/api/auth.ts`
- `frontend/src/types/auth.ts`
- `frontend/src/stores/auth.ts`

当前流程：

1. 登录页加载时获取 CSRF token。
2. 调用 `authApi.getCaptcha()` 获取 `{ token, image }`。
3. 用户手动输入图片验证码答案。
4. 登录时提交 `captcha_token` 和 `captcha_answer`。
5. 登录失败后刷新图片验证码。

## Cloudflare Turnstile fit

Cloudflare Turnstile 的模型与当前验证码流程可以对应：

- 前端不再显示图片验证码和输入框，而是渲染 Turnstile widget。
- Turnstile widget 生成 `response token`。
- 登录请求提交该 token 给后端。
- 后端使用 secret key 调用 Cloudflare `siteverify` 接口进行服务端验证。
- 验证通过后再继续用户名/密码校验。

Cloudflare 官方要求：

- 必须做服务端验证，不能只信任前端 widget。
- 验证接口：`POST https://challenges.cloudflare.com/turnstile/v0/siteverify`。
- 必填参数：`secret`、`response`。
- 可选参数：`remoteip`、`idempotency_key`。
- token 5 分钟过期。
- token 一次性使用，重复使用或过期会失败。

## Proposed target flow

### Frontend flow

1. 登录页加载 CSRF token。
2. 登录页渲染 Cloudflare Turnstile widget。
3. Turnstile 成功后返回 token。
4. 前端将 token 存入表单状态。
5. 登录时提交：
   - `username`
   - `password`
   - `csrf_token`
   - `turnstile_token`
6. 登录失败或 token 过期时重置 Turnstile widget。

### Backend flow

1. `login` 解析登录请求。
2. 校验 `username`、`password`、`csrf_token`、`turnstile_token` 非空。
3. 使用配置中的 Turnstile secret key 调用 Cloudflare siteverify。
4. 请求参数包含：
   - `secret`
   - `response`: 前端提交的 `turnstile_token`
   - `remoteip`: 调用方 IP，建议从请求 remote address / RealIP 结果获取。
5. siteverify `success == true` 时继续用户名密码校验。
6. siteverify 失败时返回 `Invalid captcha` 或更中性的验证失败信息。
7. 记录验证失败日志，但不得记录 secret key。

## Required changes if implemented

### Backend changes

- `backend-go/internal/config/config.go`
  - 新增 Turnstile 配置，例如：
    - `turnstile.enabled`，默认 `true`，可通过配置覆盖为 `false`
    - `turnstile.site_key`
    - `turnstile.secret_key`
    - `turnstile.verify_url`，可选，默认 Cloudflare endpoint
  - 绑定环境变量：
    - `POMELO_ORBIT_BACKEND__TURNSTILE__ENABLED`
    - `POMELO_ORBIT_BACKEND__TURNSTILE__SITE_KEY`
    - `POMELO_ORBIT_BACKEND__TURNSTILE__SECRET_KEY`
    - `POMELO_ORBIT_BACKEND__TURNSTILE__VERIFY_URL`
  - 当 `turnstile.enabled=true` 时校验 site key 和 secret key 非空；关闭时不强制校验。
- `backend-go/config.defaults.yaml`
  - 增加 Turnstile 配置占位。
- `backend-go/internal/httpserver/auth.go`
  - 移除固定 `captchaAnswer` 图片验证码校验。
  - 登录请求字段从 `captcha_token` / `captcha_answer` 调整为 `turnstile_token`。
  - 增加 Turnstile siteverify 调用逻辑。
  - `GET /api/auth/captcha` 可删除或替换为返回 Turnstile site key 的接口。
- `backend-go/internal/httpserver/server_test.go`
  - 更新登录测试，使用可注入/可 mock 的 Turnstile verifier。

### Frontend changes

- `frontend/src/views/Login.vue`
  - 移除图片验证码输入框和刷新逻辑。
  - 引入 Turnstile widget 渲染。
  - 保存 widget 返回 token。
  - 登录失败后 reset widget。
- `frontend/src/types/auth.ts`
  - `LoginReq` 将 `captcha_token` / `captcha_answer` 替换为 `turnstile_token`。
  - 如保留配置接口，则新增 Turnstile config response 类型。
- `frontend/src/api/auth.ts`
  - 移除或替换 `getCaptcha()`。
- `frontend/src/stores/auth.ts`
  - 登录参数改为传递 `turnstile_token`。
- i18n 文案
  - 将“验证码”相关文案调整为 Turnstile 验证失败/请完成验证。

## Design considerations

### 1. 启用开关

新增 `turnstile.enabled`：

- 默认值为 `true`。
- 可通过配置文件或环境变量覆盖为 `false`。
- `enabled=true` 时，登录必须提交并通过 `turnstile_token` 服务端验证。
- `enabled=false` 时，登录不执行 Turnstile 校验，也不回退到旧图片验证码。
- `enabled=false` 主要用于开发环境或外部网络不可达环境，生产环境建议保持启用。

### 2. 是否仍需要 `/api/auth/captcha`

可选方案：

- 删除 `/api/auth/captcha`，前端通过构建环境变量注入 Turnstile site key。
- 删除 `/api/auth/captcha`，新增 `/api/auth/turnstile-config` 返回 site key。

建议后续实现时优先考虑后端返回 site key：

- site key 不是 secret，可公开。
- 前端部署不需要单独注入 site key。
- 方便不同环境使用不同 site key。

### 3. 是否需要保留 `captcha_answer`

不建议保留。项目处于活跃开发期，按当前规范不做兼容层。后续实现应直接改为 `turnstile_token`。

### 4. Turnstile verifier 可测试性

建议在后端设计一个小接口，例如：

```go
type TurnstileVerifier interface {
    Verify(ctx context.Context, token string, remoteIP string) error
}
```

`Server` 注入 verifier 或由配置构造 verifier，测试中使用 fake verifier，避免单元测试访问 Cloudflare。

### 5. 网络失败策略

建议：

- siteverify 网络失败时登录失败，返回验证码验证失败或稍后重试。
- 使用短超时，例如 3s。
- 不因为 Cloudflare 临时不可用而绕过验证码。

## Feasibility conclusion

可行，且比当前固定图片验证码更适合生产使用。

主要原因：

- 当前验证码实现是固定答案，占位性质明显。
- Turnstile 的前端 token + 后端 siteverify 模型能直接替代当前 `captcha_token` / `captcha_answer` 流程。
- 前端登录页改动集中在 `Login.vue`。
- 后端登录校验集中在 `auth.go`，改造边界清晰。
- 当前配置系统可扩展 Turnstile secret/site key。

## Acceptance for this review

- [x] 明确当前验证码实现位置和限制。
- [x] 明确 Cloudflare Turnstile 服务端验证要求。
- [x] 明确前端改造点。
- [x] 明确后端改造点。
- [x] 明确是否可行。
- [x] 记录风险和后续设计问题。

## Risks

- Turnstile 依赖 Cloudflare 外部服务，登录路径会新增外部网络依赖。
- 中国大陆或部分网络环境访问 Turnstile widget / siteverify 可能存在可用性风险。
- 完整替换会影响登录页前端交互和登录请求 DTO，需要前后端同步修改。
- 测试需要 mock verifier，不能依赖真实 Cloudflare 服务。
- secret key 必须只保存在后端配置中，不能下发到前端。

## Open questions

- site key 由前端环境变量注入，还是由后端接口返回？建议后端接口返回。
- 是否需要保留开发环境 bypass？按当前项目规范不建议默认 bypass；如需要，应作为显式开发配置并严格限制。
- 已决定删除 `/api/auth/captcha`，新增 `/api/auth/turnstile-config`。

## Sources

- [Cloudflare Turnstile server-side validation](https://developers.cloudflare.com/turnstile/get-started/server-side-validation/)
- [Cloudflare Turnstile getting started](https://developers.cloudflare.com/turnstile/get-started/)
