# Cloudflare 灰云 API 的 CORS 与前端 baseURL 适配规格
最后修改时间: 2026-07-01 15:45:26

Review status: Accepted

## Requirement basis

基于已接受需求：

```text
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
```

用户补充决策：

1. CORS 和运行时配置命名按现有配置体系合理评估。
2. CORS 允许多组 Origin。
3. 不需要 `Access-Control-Allow-Credentials`。
4. 需要实现 `window.__CONFIG__.apiBaseUrl` 的运行时注入。

目标部署形态：

```text
https://orbit.preflite.cn      静态资源 / 前端页面，Cloudflare 黄云 / Proxied
https://orbit-api.preflite.cn  API，Cloudflare 灰云 / DNS only
```

## Overview

本变更由三个部分组成：

1. 后端配置与 CORS 中间件。
2. 前端 API URL 构造统一化。
3. 前端运行时公开配置注入。

整体请求路径：

```text
浏览器加载页面:
https://orbit.preflite.cn
  -> Cloudflare 黄云
  -> 源站
  -> Go 静态文件服务返回 index.html / JS / CSS

浏览器调用 API:
https://orbit-api.preflite.cn/api/...
  -> DNS only 灰云，直连源站
  -> Traefik Host(`orbit-api.preflite.cn`)
  -> Go API
```

由于页面 Origin 与 API Origin 不同，后端需要在 API 请求上按配置返回 CORS 头，并处理 preflight `OPTIONS`。

## Design decisions

### Decision 1: CORS 配置放在 `server` 配置段

现有配置已经有：

```yaml
server:
  host: 127.0.0.1
  port: 9020
```

CORS 属于 HTTP server 行为，而不是业务服务、Traefik 或安全密钥，因此放在 `server` 下最贴近现有配置结构。

新增配置：

```yaml
server:
  cors_allowed_origins: []
```

配置文件中使用数组表达多 Origin：

```yaml
server:
  cors_allowed_origins:
    - https://orbit.preflite.cn
    - https://other.example.com
```

环境变量仍通过逗号分隔字符串覆盖数组配置：

```text
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn,https://other.example.com
```

设计为配置结构数组，原因：

1. CORS allowlist 语义天然是 Origin 列表，数组比逗号拼接字符串更准确。
2. YAML 配置中能清晰表达多组 Origin。
3. 环境变量天然是字符串，因此加载时将逗号分隔值 decode 为 `[]string`。
4. 用户要求 `cors_allowed_origins` 应该是数组。

后端内部将该数组解析为规范化 Origin 集合。

### Decision 2: 未配置 CORS 时完全保持现状

当 `server.cors_allowed_origins` 为空时：

- 不设置 `Access-Control-Allow-Origin`。
- 不拦截 `OPTIONS`。
- 不默认返回 `*`。

这样保持当前同源部署行为，不扩大跨域访问面。

### Decision 3: 只对 `/api/` 请求应用 CORS

CORS 目标是跨域 API。静态资源仍由 `orbit.preflite.cn` 访问，不需要为所有静态资源设置 CORS。

中间件应只处理：

```text
PathPrefix(`/api/`)
```

包括：

```text
/api/auth/...
/api/ci/...
/api/cd/...
/api/project/...
/api/settings/...
```

### Decision 4: 不支持 credentials

根据用户决策，本阶段不设置：

```text
Access-Control-Allow-Credentials: true
```

当前认证主要使用 Bearer token：

```text
Authorization: Bearer <token>
```

因此 CORS 需要允许 `Authorization` header，但不需要 cookie credentials。

### Decision 5: 使用运行时注入配置 API base URL

现有前端已经有运行时配置入口：

```ts
window.__CONFIG__?.apiBaseUrl || import.meta.env.VITE_API_BASE_URL
```

`web/index.html` 也已有占位符：

```html
<!-- __RUNTIME_CONFIG__ -->
<script>
  window.__CONFIG__ = window.__CONFIG__ || {};
</script>
```

新增后端配置，与 CORS 配置合并在 `server` 配置段：

```yaml
server:
  api_base_url: ""
```

环境变量：

```text
POMELO_ORBIT_SERVER__API_BASE_URL=https://orbit-api.preflite.cn
```

Go 服务在返回 `index.html` 时注入：

```html
<script>
  window.__CONFIG__ = {"apiBaseUrl":"https://orbit-api.preflite.cn"};
</script>
```

运行时配置优先于 Vite 构建期变量，因此镜像不需要为不同 API 域名重复构建。

### Decision 6: 运行时注入使用既有 HTML 占位符并保留 fallback

`web/index.html` 已满足运行时配置注入要求：

```html
<!-- 运行时配置注入点：Nginx/entrypoint 可替换为 window.__CONFIG__。只放公开配置，不能放密钥。 -->
<!-- __RUNTIME_CONFIG__ -->
<script>
  // 保底为空对象；真实运行时配置必须在应用入口脚本加载前注入。
  window.__CONFIG__ = window.__CONFIG__ || {};
</script>
```

该注入点位于 `<head>` 内，早于前端入口脚本，因此可以保证 `window.__CONFIG__.apiBaseUrl` 在应用启动前可读。

运行时注入策略：

1. 优先替换 `<!-- __RUNTIME_CONFIG__ -->`。
2. 如果构建产物意外丢失占位符，则 fallback 插入到 `</head>` 前。
3. 注入内容只包含公开配置，不包含密钥。
4. 使用 `json.Marshal` 生成配置 JSON，避免手写拼接导致转义问题。

### Decision 7: 前端所有非 Axios API 调用改用 `buildApiUrl`

已发现绕过 Axios baseURL 的调用点：

1. `web/src/api/cd/deployments.ts` 中的 `fetch('/api/cd/deployment/.../stream-log')`。
2. `web/src/views/ci/components/WebhookList.vue` 中的 `window.location.origin` 生成 API URL。

改为统一使用：

```ts
buildApiUrl('/api/...')
```

Webhook URL 使用用户确认的路径：

```text
/api/ci/webhook/{webhook_id}
```

不引入 `/api/webhooks/` 别名或兼容路径。

## Affected components

### Backend config

文件：

```text
internal/config/config.go
configs/config.yaml
internal/config/config_test.go
```

新增结构：

```go
type Config struct {
    Server ServerConfig `mapstructure:"server" yaml:"server"`
    // existing fields...
}

type ServerConfig struct {
    Host               string   `mapstructure:"host" yaml:"host"`
    Port               int      `mapstructure:"port" yaml:"port"`
    CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins" yaml:"cors_allowed_origins"`
    APIBaseURL         string   `mapstructure:"api_base_url" yaml:"api_base_url"`
}
```

新增 env binding：

```go
"server.cors_allowed_origins"
"server.api_base_url"
```

### Backend HTTP middleware

文件建议：

```text
internal/transport/http/middleware/cors.go
internal/transport/http/middleware/cors_test.go
internal/transport/http/server.go
```

中间件行为：

1. 输入配置数组。
2. 解析 Origin 数组。
3. 请求路径不是 `/api/` 时不处理。
4. 请求无 `Origin` 时不处理。
5. Origin 命中允许列表时设置：

```text
Access-Control-Allow-Origin: <request Origin>
Vary: Origin
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Max-Age: 600
```

6. 对命中的 preflight `OPTIONS` 返回 `204 No Content`。
7. 对未命中的 preflight `OPTIONS` 返回 `403 Forbidden`，原因是失败更明确，避免落入 API NotFound 产生误导。

### Backend runtime config injection

文件：

```text
internal/transport/http/server.go
internal/transport/http/server_test.go
```

当前静态服务函数：

```go
func serveStatic(w http.ResponseWriter, r *http.Request, staticDir string) bool
```

需要能访问 `s.appCfg.Server.APIBaseURL`。建议改为 Server 方法：

```go
func (s Server) serveStatic(w http.ResponseWriter, r *http.Request, staticDir string) bool
```

当最终服务的是 `index.html` 时：

1. 读取 `static/index.html`。
2. 使用 `json.Marshal` 生成公开配置，避免手写 JS 字符串转义。
3. 替换 `<!-- __RUNTIME_CONFIG__ -->`。
4. 设置 `Content-Type: text/html; charset=utf-8`。
5. 写出 HTML。

注入内容示例：

```html
<script>window.__CONFIG__ = {"apiBaseUrl":"https://orbit-api.preflite.cn"};</script>
```

如果 `server.api_base_url` 为空，可以注入空对象或不替换；推荐仍注入：

```html
<script>window.__CONFIG__ = {};</script>
```

这样行为稳定，且符合现有前端 fallback。

静态资源文件仍使用 `http.ServeFile`，避免影响 JS/CSS/favicon 缓存与 Range 行为。

### Frontend URL construction

文件：

```text
web/src/api/cd/deployments.ts
web/src/views/ci/components/WebhookList.vue
```

`deployments.ts`：

```ts
import { buildApiUrl } from '@/config';

return fetch(buildApiUrl(`/api/cd/deployment/${id}/stream-log`), {
  headers: token ? { Authorization: `Bearer ${token}` } : {},
  signal,
});
```

`WebhookList.vue`：

```ts
import { buildApiUrl } from '@/config';

function webhookUrl(webhookId: string) {
  return buildApiUrl(`/api/ci/webhook/${webhookId}`);
}
```

## Interfaces

### Config YAML

默认配置：

```yaml
server:
  host: 127.0.0.1
  port: 9020
  cors_allowed_origins: []
  api_base_url: ""
```

生产配置示例：

```env
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn
POMELO_ORBIT_SERVER__API_BASE_URL=https://orbit-api.preflite.cn
```

多 Origin 示例：

```env
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn,https://orbit-preview.preflite.cn
```

### HTTP response headers

普通跨域 API 响应：

```text
Access-Control-Allow-Origin: https://orbit.preflite.cn
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Max-Age: 600
Vary: Origin
```

Preflight：

```text
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://orbit.preflite.cn
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Max-Age: 600
Vary: Origin
```

### Runtime config

HTML 注入：

```html
<script>window.__CONFIG__ = {"apiBaseUrl":"https://orbit-api.preflite.cn"};</script>
```

前端读取：

```ts
config.apiBaseUrl === 'https://orbit-api.preflite.cn'
```

## Technical questions

1. 已决策：`OPTIONS` 未命中 Origin 时返回 `403 Forbidden`。
2. 已决策：Origin normalize 时去除尾部 `/`，请求头 Origin 通常不带路径和尾斜杠。
3. 已验证：`web/index.html` 已有专门的 `<!-- __RUNTIME_CONFIG__ -->` 注入点，且位于前端入口脚本之前。实现时仍需保留 `</head>` fallback，以避免构建产物或未来插件移除注释导致注入失败。

## Risks

1. CORS header 配置过宽会扩大 API 暴露面。默认空配置必须保持不开放。
2. `OPTIONS` 处理过宽可能影响非 API 路由；中间件必须限制在 `/api/`。
3. `window.__CONFIG__` 注入必须使用 JSON 编码，不能手写拼接，避免引号或脚本注入问题。
4. 如果只改 Axios 以外的两个调用点，但后续新增直接 `fetch('/api')`，可能再次绕过 baseURL；建议后续 code review 关注该模式。
5. 跨域 API 依赖 token 存储在前端状态/localStorage；本任务不改变认证模式。

## Alternatives

### Alternative 1: Docker build arg 写死 `VITE_API_BASE_URL`

方案：

```dockerfile
ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN yarn build
```

拒绝原因：

- 镜像与部署环境绑定。
- API 域名变化需要重建镜像。
- 项目已有 `window.__CONFIG__` 运行时配置入口。

### Alternative 2: 全站 `orbit.preflite.cn` 改灰云

拒绝原因：

- 用户明确希望静态资源由 `orbit.preflite.cn` 承载并保持 Cloudflare 黄云。
- 本任务目标是静态/动态分离。

### Alternative 3: 后端默认 `Access-Control-Allow-Origin: *`

拒绝原因：

- 不符合“有配置就设置好”的要求。
- 扩大 API 暴露面。
- 未来如果引入 credentials 会产生安全和浏览器兼容问题。

## User review notes

待用户审查。
