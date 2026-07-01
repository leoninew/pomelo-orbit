# Cloudflare 灰云 API 的 CORS 与前端 baseURL 适配验证
最后修改时间: 2026-07-01 16:50:47

Review status: Accepted

## Basis

验证依据：

```text
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
docs/spec/20260701-cloudflare-api-cors-baseurl.md
docs/plan/20260701-cloudflare-api-cors-baseurl.md
```

当前流程：严格模式 / strict，Verification / 验证。

## Requirement alignment

### 需求 1：后端新增明确的 CORS 配置入口

已满足。

实现新增：

```yaml
server:
  cors_allowed_origins: []
```

Go 配置结构使用数组：

```go
CORSAllowedOrigins []string `mapstructure:"cors_allowed_origins" yaml:"cors_allowed_origins"`
```

环境变量绑定支持：

```text
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn,https://preview.preflite.cn
```

并通过 `mapstructure.StringToSliceHookFunc(",")` 将环境变量字符串 decode 为 `[]string`。

### 需求 2：匹配 Origin 的 API 响应返回 CORS 头

已满足。

新增 `internal/transport/http/middleware/cors.go`，在请求满足以下条件时设置响应头：

1. 路径以 `/api/` 开头。
2. 请求带 `Origin`。
3. Origin 命中 `server.cors_allowed_origins`。

返回头包括：

```text
Access-Control-Allow-Origin: <request Origin>
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Vary: Origin
```

### 需求 3：支持 preflight OPTIONS

已满足。

行为：

1. 命中 allowlist 的 `/api/` preflight 返回 `204 No Content` 和 CORS 头。
2. 未命中 allowlist 的 `/api/` preflight 返回 `403 Forbidden`。
3. 未配置 origins 时中间件 no-op，不默认放开跨域。

### 需求 4：未配置时不默认 `*`

已满足。

`server.cors_allowed_origins` 为空时，中间件直接返回原 handler，不设置 `Access-Control-Allow-Origin`，不默认 `*`。

### 需求 5：Axios API 继续通过统一 baseURL 工作

已满足。

现有 Axios request 层继续使用 `config.apiBaseUrl`，本次没有破坏该路径。

### 需求 6：直接 `fetch('/api/...')` 调用点处理

已处理并扩展清理。

原计划中发现的 `streamLogs` 是 SSE `fetch` 调用点。验证实现时进一步确认该接口无前端调用点，因此实际处理不是保留并改成 `buildApiUrl`，而是删除未使用的前后端 SSE 接口：

```text
GET /api/cd/deployment/{deployment_id}/stream-log
```

删除范围包括前端 `deploymentApi.streamLogs`、后端 route、handler、SSE response struct 和对应 route 测试。

### 需求 7：Webhook URL 不再依赖 `window.location.origin`

已满足。

`web/src/views/ci/components/WebhookList.vue` 改为通过 `buildApiUrl('/api/ci/webhook/{id}')` 构造展示/复制 URL。

### 需求 8：不改 `/api/ci/webhook/` 路径

已满足。

路径仍为：

```text
/api/ci/webhook/{webhook_id}
```

未新增 `/api/webhooks/` 兼容路径或别名。

## Spec alignment

### CORS 配置位于 `server` 配置段

已按 Spec 实现。

```yaml
server:
  cors_allowed_origins: []
```

### `cors_allowed_origins` 是数组

已按用户修正和 Spec 实现。

YAML 使用数组，环境变量仍允许逗号分隔字符串覆盖。

### 未配置 CORS 时保持现状

已按 Spec 实现并有单元测试覆盖。

### CORS 只处理 `/api/`

已按 Spec 实现并有单元测试覆盖。

### 不设置 credentials

已按 Spec 实现。

测试覆盖不设置：

```text
Access-Control-Allow-Credentials
```

### 运行时配置注入

已按 Spec 实现。

新增：

```yaml
web:
  api_base_url: ""
```

返回 `index.html` 时注入：

```html
<script>window.__CONFIG__ = {"apiBaseUrl":"https://orbit-api.preflite.cn"};</script>
```

当 `api_base_url` 为空时注入：

```html
<script>window.__CONFIG__ = {};</script>
```

实现使用 `json.Marshal`，避免手写 JS 字符串拼接。

### 注入点和 fallback

已按 Spec 实现。

优先替换：

```html
<!-- __RUNTIME_CONFIG__ -->
```

占位符缺失时 fallback 插入到 `</head>` 前。

## Plan alignment

### Step 1：扩展配置结构

已完成。

涉及：

```text
configs/config.yaml
internal/config/config.go
internal/config/config_test.go
```

### Step 2：添加 CORS 中间件

已完成。

涉及：

```text
internal/transport/http/middleware/cors.go
internal/transport/http/middleware/cors_test.go
```

### Step 3：注册 CORS 中间件

已完成。

涉及：

```text
internal/transport/http/server.go
```

### Step 4：实现运行时配置注入

已完成。

涉及：

```text
internal/transport/http/server.go
internal/transport/http/server_test.go
```

### Step 5：修复前端绕过 baseURL 的调用点

部分按计划完成，部分因实际代码审计调整。

1. `WebhookList.vue` 按计划改为 `buildApiUrl`。
2. `deployments.ts` 中的 `streamLogs` 经使用审计确认无调用点，已删除，而不是保留并改为 `buildApiUrl`。

该调整符合用户后续要求：检查接口是否使用，如果没有就删除，同时检查后端业务逻辑。

### Step 6：搜索防遗漏

已执行相关搜索与审计。

结论：

1. 删除后，当前源码中未再发现 `streamLogs`、`stream-log`、`sseDeploymentLog`、`VITE_FEATURE_SSE_DEPLOYMENT_LOG` 残留。
2. `/api/cd/deployment/{deployment_id}/logs` 是普通部署执行日志接口，不是 SSE，未纳入本次删除范围。
3. 当前部署详情页实际使用 `/container-logs` 展示容器日志。

## Actual diff summary

### Staged diff summary

当前本任务相关 staged 文件：

```text
configs/config.yaml
docs/plan/20260701-cloudflare-api-cors-baseurl.md
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
docs/spec/20260701-cloudflare-api-cors-baseurl.md
internal/config/config.go
internal/config/config_test.go
internal/transport/http/deployment_routes_test.go
internal/transport/http/handler/cd/handler.go
internal/transport/http/middleware/cors.go
internal/transport/http/middleware/cors_test.go
internal/transport/http/server.go
internal/transport/http/server_test.go
web/src/api/cd/deployments.ts
web/src/config.ts
web/src/env.d.ts
web/src/views/ci/components/WebhookList.vue
```

Staged 统计：

```text
16 files changed, 1345 insertions(+), 131 deletions(-)
```

### Unstaged diff summary

工作区仍存在其他未 stage 变更，主要来自相邻任务或既有工作区状态，不属于本次 CORS/runtime baseURL staged 交付范围。

验证过程中按项目约束运行 `go fmt ./cmd/... ./internal/...`，该命令对部分未 stage Go 文件产生或保留了格式化后的 unstaged 变更。需要在提交前由用户决定是否拆分、回滚或单独处理这些无关改动。

未 stage 统计：

```text
31 files changed, 385 insertions(+), 154 deletions(-)
```

未 stage 中包含的典型文件：

```text
internal/transport/http/handler/cd/application_extra.go
internal/transport/http/handler/cd/route.go
internal/transport/http/handler/ci/pipeline_run.go
internal/transport/http/handler/ci/template.go
internal/transport/http/handler/project/handler.go
internal/transport/http/handler/role/handler.go
internal/transport/http/response/response.go
scripts/docker-compose.yml
web/src/api/cd/application.ts
web/src/api/cd/route.ts
web/src/api/ci/run.ts
web/src/api/ci/template.ts
web/src/api/project.ts
web/src/api/role.ts
web/src/views/cd/ApplicationDetail.vue
web/src/views/ci/PipelineRunDetail.vue
web/src/views/ci/PipelineTemplateDetail.vue
```

另有未跟踪文档：

```text
docs/requirement/20260701-api-inventory.md
docs/spec/20260701-api-inventory.md
```

这些不是本 verification 的交付对象。

## Expected vs actual changed files

### Expected in plan

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
docs/requirement/20260701-cloudflare-api-cors-baseurl.md
docs/spec/20260701-cloudflare-api-cors-baseurl.md
docs/plan/20260701-cloudflare-api-cors-baseurl.md
```

### Actual staged additions beyond original plan

```text
internal/transport/http/deployment_routes_test.go
internal/transport/http/handler/cd/handler.go
web/src/config.ts
web/src/env.d.ts
```

原因：用户在实现阶段要求检查 SSE `stream-log` 是否使用，未使用则删除，并检查后端业务逻辑。上述文件用于删除无调用点的 SSE 接口及其前端配置残留。

### Actual unstaged unrelated changes

存在，不属于本次 staged 交付范围。详见 “Unstaged diff summary”。

## Acceptance checklist

- [x] 后端新增明确 CORS allowed origins 配置入口。
- [x] `cors_allowed_origins` 使用数组表达。
- [x] 环境变量支持逗号分隔多 Origin。
- [x] 匹配 Origin 的 API 响应返回必要 CORS 头。
- [x] preflight `OPTIONS` 命中 allowlist 返回 `204`。
- [x] preflight `OPTIONS` 未命中 allowlist 返回 `403`。
- [x] 未配置 allowed origins 时不默认返回 `Access-Control-Allow-Origin: *`。
- [x] 不设置 `Access-Control-Allow-Credentials`。
- [x] CORS 只处理 `/api/`。
- [x] 新增 `web.api_base_url` 运行时配置。
- [x] `index.html` 返回时注入 `window.__CONFIG__`。
- [x] 注入使用 `json.Marshal`。
- [x] 占位符缺失时 fallback 到 `</head>` 前。
- [x] Webhook URL 使用 `buildApiUrl`，不再依赖 `window.location.origin`。
- [x] `/api/ci/webhook/` 路径未被错误替换。
- [x] 未使用 SSE `stream-log` 接口已按用户要求删除。
- [x] 前端 lint/typecheck 通过。
- [x] Go fmt/vet/test 通过。

## Test results

### Go

```text
go fmt ./cmd/... ./internal/...
```

结果：通过。

说明：命令输出了被格式化文件：

```text
internal\transport\http\handler\ci\pipeline_run.go
```

该文件属于未 stage 的相邻工作区变更，不属于本次 staged 交付范围。

```text
go vet ./cmd/... ./internal/...
```

结果：通过。

```text
go test -count=1 ./cmd/... ./internal/...
```

结果：通过。

关键包结果包括：

```text
ok backend/internal/config
ok backend/internal/transport/http
ok backend/internal/transport/http/middleware
ok backend/internal/worker
ok backend/internal/worker/handler/cd
ok backend/internal/worker/handler/ci
```

```text
go test ./internal/config ./internal/transport/http/middleware ./internal/transport/http -run 'TestLoadDefaultConfigFile|TestLoadConfigEnvOverrides|TestCORS|TestStaticFiles|TestDeployment' -count=1
```

结果：通过。

输出：

```text
ok backend/internal/config
ok backend/internal/transport/http/middleware
ok backend/internal/transport/http
```

### Frontend

```text
yarn --cwd web lint:fix
```

结果：通过。

```text
yarn --cwd web typecheck
```

结果：通过。

## Missed or expanded scope

### Expanded scope: 删除未使用 SSE stream-log 接口

原 Requirement/Spec/Plan 假设 SSE `stream-log` 是需要保留并改造 baseURL 的直接 `fetch` 调用点。

实现阶段用户追问后进行使用审计，确认：

1. 前端没有调用 `deploymentApi.streamLogs`。
2. 部署详情页实际使用 `deploymentApi.getContainerLogs(...)`。
3. 后端存在真实 SSE route 和 handler，但没有前端消费者。

因此删除：

```text
GET /api/cd/deployment/{deployment_id}/stream-log
```

该扩展范围由用户明确要求触发，且与“修复绕过 API baseURL 的调用点”目标一致：没有使用者的接口不应为了兼容而保留。

### Not removed: 普通部署日志 `/logs`

保留：

```text
GET /api/cd/deployment/{deployment_id}/logs
```

原因：它是普通 JSON 部署执行日志接口，支持 offset 增量读取，不是 SSE `stream-log`。虽然当前部署详情页主要使用 `container-logs`，但 `/logs` 是不同业务接口，本次没有用户授权删除。

## Risks

1. 当前工作区存在大量未 stage 的相邻改动。虽然本次相关 staged diff 已隔离，但提交前仍建议使用 staged diff 进行审查，避免混入无关变更。
2. `go fmt ./cmd/... ./internal/...` 按项目约束运行后影响了未 stage 的相邻 Go 文件；这些文件需要单独处理。
3. 生产环境必须设置：

```text
POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS=https://orbit.preflite.cn
POMELO_ORBIT_WEB__API_BASE_URL=https://orbit-api.preflite.cn
```

否则浏览器跨域 API 或前端 API base URL 行为不会达到目标部署形态。

4. CORS 只解决浏览器跨域限制，不改变后端鉴权模型；API 仍依赖现有 Bearer token 鉴权。
5. 后续新增直接 `fetch('/api/...')` 或 `window.location.origin + '/api/...'` 时，仍可能再次绕过 API base URL，需要 code review 继续关注。

## Incomplete items

1. 未处理工作区中非本任务的 unstaged/untracked 变更。
2. 未进行真实线上域名手工验证，例如：

```text
curl -i -X OPTIONS https://orbit-api.preflite.cn/api/health
curl -s https://orbit.preflite.cn/ | grep 'window.__CONFIG__'
```

原因：本次验证在本地代码与测试层完成，未对外部部署环境执行验证。

## Conclusion

本次实现与已接受的 Requirement / Spec / Plan 对齐，并按用户后续要求删除了未使用的 SSE `stream-log` 接口。核心本地验证通过：

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test -count=1 ./cmd/... ./internal/...
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

结论：本任务代码层验证通过，可以进入人工审查与后续提交准备。提交前建议重点确认 staged diff，避免混入工作区中其他未 stage 的相邻改动。
