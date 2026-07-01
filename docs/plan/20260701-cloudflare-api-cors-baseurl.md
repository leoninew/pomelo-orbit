# Cloudflare 灰云 API 的 CORS 与前端 baseURL 适配计划
最后修改时间: 2026-07-01 15:45:26

Review status: Accepted

## Basis

已接受文档：

```text
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
docs/spec/20260701-cloudflare-api-cors-baseurl.md
```

目标形态：

```text
https://orbit.preflite.cn      静态资源 / 前端页面，Cloudflare 黄云 / Proxied
https://orbit-api.preflite.cn  API，Cloudflare 灰云 / DNS only
```

## Implementation steps

### Step 1: 扩展配置结构

修改：

```text
internal/config/config.go
configs/config.yaml
internal/config/config_test.go
```

计划：

1. 在 `Config` 中新增 `Web WebConfig`。
2. 在 `ServerConfig` 中新增 `CORSAllowedOrigins []string`。
3. 新增 `WebConfig`：

```go
type WebConfig struct {
    APIBaseURL string `mapstructure:"api_base_url" yaml:"api_base_url"`
}
```

4. 在 `bindEnv` 中新增：

```go
"server.cors_allowed_origins"
"web.api_base_url"
```

5. 在 `configs/config.yaml` 中新增默认空配置：

```yaml
server:
  cors_allowed_origins: []

web:
  api_base_url: ""
```

6. 更新配置测试，覆盖：
   - 默认配置为空。
   - 环境变量能覆盖 CORS origins 和 web API base URL。

### Step 2: 添加 CORS 中间件

新增：

```text
internal/transport/http/middleware/cors.go
internal/transport/http/middleware/cors_test.go
```

计划行为：

1. 解析 origins 数组。
2. normalize：trim 空白，去掉尾部 `/`，忽略空项。
3. 如果未配置 origins，中间件 no-op。
4. 只处理 `/api/` 路径。
5. 请求没有 `Origin` 时 no-op。
6. Origin 命中允许列表时设置：

```text
Access-Control-Allow-Origin: <request Origin>
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Vary: Origin
```

7. 命中且为 `OPTIONS` 时返回 `204 No Content`。
8. 未命中且为 `/api/` preflight `OPTIONS` 时返回 `403 Forbidden`。
9. 不设置 `Access-Control-Allow-Credentials`。

测试覆盖：

1. 未配置时不设置任何 CORS header。
2. 配置单个 Origin 时，匹配 Origin 返回 headers。
3. 配置多个 Origin 时，任一匹配 Origin 返回 headers。
4. Origin 带尾斜杠的配置能匹配正常 Origin。
5. 匹配的 preflight 返回 `204`。
6. 未匹配的 preflight 返回 `403`。
7. 非 `/api/` 路径不处理 CORS。

### Step 3: 注册 CORS 中间件

修改：

```text
internal/transport/http/server.go
```

计划：

1. 在 request logging / recoverer 周围保持现有行为，新增 CORS 中间件。
2. 中间件注册位置建议在 `Recoverer` 后、路由注册前；CORS preflight 能提前返回，避免落入 NotFound。
3. 使用配置：

```go
transportmiddleware.CORS(s.appCfg.Server.CORSAllowedOrigins)
```

### Step 4: 实现运行时配置注入

修改：

```text
internal/transport/http/server.go
internal/transport/http/server_test.go
```

计划：

1. 将 `serveStatic` 从自由函数改为 `Server` 方法，或新增可接收 runtime config 的 helper。
2. 当请求静态文件命中具体文件且不是 `index.html` 时，继续使用 `http.ServeFile`。
3. 当 SPA fallback 或根路径返回 `index.html` 时：
   - 读取 `static/index.html`。
   - 构造公开配置：

```go
map[string]string{"apiBaseUrl": s.appCfg.Web.APIBaseURL}
```

   - 如果 APIBaseURL 为空，输出空对象 `{}`。
   - 用 `json.Marshal` 生成 JSON。
   - 优先替换 `<!-- __RUNTIME_CONFIG__ -->`。
   - 如果占位符不存在，fallback 插入到 `</head>` 前。
   - 设置 `Content-Type: text/html; charset=utf-8` 后写出。
4. 如果读取 `index.html` 失败，保持现有行为语义：静态不存在时返回 API 风格 NotFound 或 false，由外层 NotFound 处理。

测试覆盖：

1. 配置 `Web.APIBaseURL` 后，返回的 index HTML 包含 `window.__CONFIG__` 和 apiBaseUrl。
2. APIBaseURL 为空时，返回空配置，不影响页面。
3. 占位符缺失时能插入到 `</head>` 前。
4. 普通静态文件仍能正常返回，不被 runtime config 注入逻辑改写。

### Step 5: 修复前端绕过 baseURL 的调用点

修改：

```text
web/src/api/cd/deployments.ts
web/src/views/ci/components/WebhookList.vue
```

计划：

1. `deployments.ts` 导入 `buildApiUrl`：

```ts
import { buildApiUrl } from '@/config';
```

2. `streamLogs` 改为：

```ts
return fetch(buildApiUrl(`/api/cd/deployment/${id}/stream-log`), {
  headers: token ? { Authorization: `Bearer ${token}` } : {},
  signal,
});
```

3. `WebhookList.vue` 导入 `buildApiUrl`。
4. `webhookUrl` 改为：

```ts
return buildApiUrl(`/api/ci/webhook/${webhookId}`);
```

5. 不新增 `/api/webhooks/` 路径，不改后端 webhook 路由语义。

### Step 6: 搜索防遗漏

执行内容搜索，确认没有其他绕过 API baseURL 的调用：

```text
fetch(`/api
fetch('/api
window.location.origin
/api/webhook
/api/webhooks
```

若发现同类调用点，按同一规则修复；如不是 API URL 场景，在实现摘要中说明跳过原因。

## Files to change

预计修改：

```text
configs/config.yaml
internal/config/config.go
internal/config/config_test.go
internal/transport/http/server.go
internal/transport/http/server_test.go
internal/transport/http/middleware/cors.go
internal/transport/http/middleware/cors_test.go
web/src/api/cd/deployments.ts
web/src/views/ci/components/WebhookList.vue
```

过程文档已修改：

```text
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
docs/spec/20260701-cloudflare-api-cors-baseurl.md
docs/plan/20260701-cloudflare-api-cors-baseurl.md
```

## Verification plan

### Backend checks

```bash
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

### Frontend checks

```bash
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

### Targeted behavior checks

可用 `httptest` 单元测试覆盖以下行为：

1. CORS allowed Origin。
2. CORS denied preflight。
3. Runtime config 注入。
4. SPA fallback 仍正常。

如需要手工验证部署形态，可在部署后使用：

```bash
curl -i -X OPTIONS \
  -H 'Origin: https://orbit.preflite.cn' \
  -H 'Access-Control-Request-Method: GET' \
  -H 'Access-Control-Request-Headers: authorization' \
  https://orbit-api.preflite.cn/api/health
```

预期包含：

```text
HTTP/2 204
Access-Control-Allow-Origin: https://orbit.preflite.cn
Access-Control-Allow-Headers: Authorization, Content-Type
```

运行时配置验证：

```bash
curl -s https://orbit.preflite.cn/ | grep 'window.__CONFIG__'
```

预期包含：

```text
https://orbit-api.preflite.cn
```

## Blockers

暂无阻塞项。用户已确认：

1. 多 Origin。
2. 不需要 credentials。
3. 需要运行时配置注入。
4. `/api/ci/webhook/` 路径没问题。

## Assumptions

1. 生产环境会设置：

```text
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn
POMELO_ORBIT_WEB__API_BASE_URL=https://orbit-api.preflite.cn
```

2. `orbit-api.preflite.cn` 已由云端路由转发到同一后端服务。
3. 源站证书覆盖 `orbit-api.preflite.cn`。
4. 认证仍使用 Bearer token，不依赖跨站 cookie。

## Risks

1. CORS 只解决浏览器跨域限制，不改变服务端权限模型；API 仍必须依赖现有鉴权。
2. 如果生产环境忘记设置 CORS origins，浏览器跨域 API 会失败。
3. 如果生产环境忘记设置 `web.api_base_url`，前端仍会同源请求 `orbit.preflite.cn/api/...`。
4. 如果未来新增直接 `fetch('/api/...')`，可能再次绕过 baseURL；后续 code review 需关注。

## Rollback

1. 移除或清空：

```text
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS
POMELO_ORBIT_WEB__API_BASE_URL
```

可回到同源 API 行为。

2. 代码层面可回滚本变更提交。
3. DNS / Cloudflare 层面可临时把 `orbit.preflite.cn` 改灰云作为性能止血方案，但这不属于本代码任务。

## User review notes

待用户审查。
