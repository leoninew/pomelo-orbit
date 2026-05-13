# Google OAuth 国内网络访问解决方案

## 问题背景

Google OAuth 登录在国内环境下主要会遇到两类网络问题：

1. 后端请求 `https://oauth2.googleapis.com/token` 交换 token 超时。
2. 后端验证 `id_token` 时需要访问 Google 的证书或 JWKS 端点，可能超时。

错误日志示例：

```text
2026-05-13 05:44:32 [WARNI] main.py:73 Business error: POST /api/auth/google/callback, status=503, message=Google 授权请求失败, 请检查网络或稍后重试
```

当前项目已经由后端完成 OAuth 的关键安全流程：

```text
前端拿到 Google code
-> 前端提交 code 到后端 /api/auth/google/callback
-> 后端用 code 向 Google 换取 id_token
-> 后端验证 id_token
-> 后端创建或关联用户
-> 后端签发本系统 JWT
```

因此解决国内网络问题时，应尽量保留这个信任边界：后端仍然是身份验证者，Worker 只解决后端访问 Google 的网络连通性。

## 推荐方案：Cloudflare Worker 透明代理

### 架构流程

```text
1. 用户点击 Google 登录。
2. 后端 /api/auth/google 生成 Google 授权 URL。
3. Google 回调到前端 /google/callback?code=xxx。
4. 前端把 code 提交到后端 /api/auth/google/callback。
5. 后端通过 Worker 代理访问 Google token endpoint。
6. 后端通过 Worker 代理获取 Google 证书或 JWKS。
7. 后端验证 id_token，并签发本系统 JWT。
```

Worker 不接收浏览器提交的 OAuth code，也不返回用户身份信息给浏览器。它只代理后端访问 Google 的固定端点。

### Worker 路由范围

Worker 必须是白名单代理，只允许必要路径：

```text
POST /token            -> https://oauth2.googleapis.com/token
GET  /oauth2/v1/certs  -> https://www.googleapis.com/oauth2/v1/certs
GET  /oauth2/v3/certs  -> https://www.googleapis.com/oauth2/v3/certs
GET  /health           -> Worker 健康检查
```

不要实现任意 URL 转发，避免变成开放代理。

### Worker 示例

```javascript
export default {
  async fetch(request) {
    const url = new URL(request.url);

    if (url.pathname === "/health" && request.method === "GET") {
      return new Response("ok", { status: 200 });
    }

    if (url.pathname === "/token" && request.method === "POST") {
      return proxy(request, "https://oauth2.googleapis.com/token");
    }

    if (url.pathname === "/oauth2/v1/certs" && request.method === "GET") {
      return proxy(request, "https://www.googleapis.com/oauth2/v1/certs");
    }

    if (url.pathname === "/oauth2/v3/certs" && request.method === "GET") {
      return proxy(request, "https://www.googleapis.com/oauth2/v3/certs");
    }

    return new Response("not found", { status: 404 });
  },
};

async function proxy(request, target) {
  const headers = new Headers(request.headers);
  headers.delete("host");

  return fetch(target, {
    method: request.method,
    headers,
    body: request.method === "GET" || request.method === "HEAD" ? undefined : request.body,
  });
}
```

这个 Worker 只供后端调用，不需要开放浏览器 CORS。

### 后端配置

建议新增配置项：

```yaml
google:
  client_id: ""
  client_secret: ""
  redirect_uri: ""
  token_proxy_url: ""
```

环境变量示例：

```bash
POMELO_ORBIT_GOOGLE__TOKEN_PROXY_URL=https://oauth.example.com
```

后端 token 交换时使用：

```python
token_url = (
    f"{settings.google.token_proxy_url.rstrip('/')}/token"
    if settings.google.token_proxy_url
    else "https://oauth2.googleapis.com/token"
)
```

`id_token` 验证时也必须通过代理获取 Google 证书或 JWKS。实现方式可以是：

1. 使用支持自定义证书 URL 的验证入口，并把 certs/JWKS URL 指向 Worker。
2. 或者保留标准 Google 验证逻辑，但通过真实 HTTP proxy 机制代理 Google 请求。
3. 不允许只解析 JWT payload 后信任其中的 `sub`、`email`、`name`。

无论采用哪种实现，后端都必须继续验证：

```text
signature
aud == GOOGLE_CLIENT_ID
iss in ("accounts.google.com", "https://accounts.google.com")
exp 未过期
```

### workers.dev 被墙时

如果 `*.workers.dev` 在目标网络不可访问，给 Worker 绑定 Cloudflare 自定义域名，例如：

```text
https://oauth.example.com
```

要求：

1. 域名托管在 Cloudflare。
2. Worker 绑定 Custom Domain。
3. 后端配置 `POMELO_ORBIT_GOOGLE__TOKEN_PROXY_URL=https://oauth.example.com`。
4. Google Console 的 redirect URI 不需要改，仍然指向前端回调页，例如 `https://orbit.example.com/google/callback`。

`wrangler.toml` 示例：

```toml
name = "google-oauth-proxy"
main = "src/index.js"
compatibility_date = "2026-05-13"

routes = [
  { pattern = "oauth.example.com", custom_domain = true }
]
```

## 可选方案：Worker 完整处理后端交换

如果希望 `client_secret` 只保存在 Worker，或者希望所有 Google 访问都发生在 Cloudflare 边缘，可以让 Worker 完成 Google OAuth 交互。但这时后端不能接收浏览器转发的裸用户信息，而必须认证 Worker。

### 架构流程

```text
1. 用户点击 Google 登录。
2. 后端 /api/auth/google 生成 Google 授权 URL，redirect_uri 仍指向前端 /google/callback。
3. Google 回调到前端 /google/callback?code=xxx&state=yyy。
4. 前端把 code 和 state 提交给 Worker。
5. Worker 校验 state，用 code 向 Google 换取 token。
6. Worker 验证 Google id_token 的 signature、aud、iss、exp。
7. Worker server-to-server 调用后端内部交换接口。
8. 后端验证调用确实来自 Worker。
9. 后端创建或关联用户，签发本系统 JWT。
10. Worker 把本系统 JWT 返回给前端。
```

这条链路中，浏览器只负责转交 Google code 和接收最终登录结果，不负责向后端提交 `{provider_id, email, name}`。

### 后端交换接口

后端新增一个只接受 Worker 调用的接口，例如：

```text
POST /api/auth/google/worker-exchange
Authorization: Bearer <WORKER_BACKEND_SECRET>
Content-Type: application/json
```

请求体可以是 Worker 已验证过的 Google 身份：

```json
{
  "provider_id": "google-sub",
  "email": "user@example.com",
  "name": "User Name",
  "email_verified": true,
  "google_aud": "GOOGLE_CLIENT_ID",
  "google_iss": "https://accounts.google.com",
  "google_exp": 1770000000
}
```

后端必须先验证 `Authorization`，再使用请求体创建或关联用户。未通过 Worker 认证的请求一律返回 401。

最小可用的 Worker 认证方式是共享密钥：

```text
Worker secret: WORKER_BACKEND_SECRET
Backend env:   POMELO_ORBIT_GOOGLE__WORKER_BACKEND_SECRET
```

更强的方式是 HMAC 或 Worker 签名 JWT：

```text
Worker 对 body + timestamp 做 HMAC
后端校验签名、timestamp 窗口和 nonce/jti 防重放
```

如果用 Worker 签名 JWT，payload 至少应包含：

```json
{
  "iss": "pomelo-orbit-google-oauth-worker",
  "aud": "pomelo-orbit-backend",
  "exp": 1770000060,
  "iat": 1770000000,
  "jti": "one-time-id",
  "provider_id": "google-sub",
  "email": "user@example.com",
  "name": "User Name",
  "email_verified": true
}
```

后端必须校验签名、`iss`、`aud`、`exp`、`jti`，并拒绝重复使用的 `jti`。

### Worker 处理要点

Worker 必须完成真正的 Google ID token 验证，不能只解码 JWT payload。至少验证：

```text
signature 使用 Google JWKS/certs 验证
aud == GOOGLE_CLIENT_ID
iss in ("accounts.google.com", "https://accounts.google.com")
exp 未过期
state 与发起登录时生成的 state 匹配
```

Worker 调后端时应使用后端内网或公网 HTTPS 地址，并带上 Worker 凭据。Worker 日志不能记录 authorization code、Google token、本系统 JWT 或共享密钥。

### 与透明代理的取舍

透明代理更适合当前项目，因为它保留现有后端 OAuth 实现，改动小，信任边界清晰。

Worker 后端交换适合这些场景：

1. 不希望 `client_secret` 存在后端。
2. 愿意新增 Worker 身份认证、后端交换接口和重放防护。
3. 愿意在 Worker 中实现完整的 Google ID token 验证。

如果只是为了修复国内网络访问，优先采用透明代理。

## 不采用的方案

### 浏览器转发裸用户信息

不采用这个方案的原因不是 Worker 不能访问 Google，而是它把后端的身份验证边界改坏。

错误设计通常是：

```text
前端把 code 发给 Worker
-> Worker 换 token 并解析 id_token
-> Worker 返回 {provider_id, email, name}
-> 前端把这些字段 POST 给后端
-> 后端直接签发 JWT
```

这个流程不可直接采用，因为 `/api/auth/google/callback-worker` 是公网 API。攻击者可以绕过 Worker，直接 POST 任意 `{provider_id, email, name}` 给后端。如果后端按 email 关联已有用户并签发 JWT，就会形成账号冒充。

即使 Worker 真的完整验证了 Google `id_token`，只要最终交给后端的是浏览器转发的普通 JSON，后端仍然无法证明这些字段来自 Worker。

如果要采用 Worker 完整处理 OAuth，必须改成上面的“Worker 完整处理后端交换”模式：后端认证 Worker，或验证 Worker 签发的短期断言。

### 只做离线验证 id_token

只缓存 Google 公钥并在本地验证 `id_token`，不能单独解决问题。因为授权码换 token 仍然需要访问 `https://oauth2.googleapis.com/token`。

离线验证还需要正确处理公钥缓存、过期、轮换、issuer、audience 和时间偏移。它可以作为后端验证实现的一部分，但不能替代 token endpoint 的网络解决方案。

## 必须补齐的安全项

### state 校验

OAuth 登录流程应加入 `state` 参数：

```text
后端生成 state
-> 保存到安全 cookie 或服务端会话
-> 授权 URL 携带 state
-> 回调时校验 state
-> 校验通过后才交换 token
```

这用于降低 CSRF 和登录混淆风险。网络代理方案不应绕过这一步。

### 代理访问控制

Worker 应保持最小能力：

1. 只允许固定路径。
2. 只允许固定 method。
3. 不代理任意 URL。
4. 记录错误状态，但不要记录 `client_secret`、authorization code、token 等敏感值。

## 测试

本地测试 Worker：

```bash
wrangler dev
```

健康检查：

```bash
curl https://oauth.example.com/health
```

测试 token endpoint 是否可达：

```bash
curl -X POST https://oauth.example.com/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "grant_type=authorization_code"
```

这个请求会因为缺少 `code`、`client_id` 等参数返回 Google 错误，但只要能收到 Google 的错误响应，就说明代理链路可达。

完整流程测试：

```text
访问 /login
-> 点击 Google 登录
-> 完成 Google 授权
-> 回到 /google/callback
-> 后端完成 token exchange 和 id_token 验证
-> 前端进入已登录状态
```

## 监控和日志

1. Cloudflare Worker 日志：`wrangler tail`
2. 后端日志：查看 `/api/auth/google/callback` 请求日志
3. 前端错误：浏览器控制台

## 参考资料

- [Cloudflare Workers 文档](https://developers.cloudflare.com/workers/)
- [Google OAuth 2.0 文档](https://developers.google.com/identity/protocols/oauth2)
- [JWT 规范](https://jwt.io/)
