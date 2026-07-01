# Cloudflare 灰云 API 的 CORS 与前端 baseURL 适配需求
最后修改时间: 2026-07-01 14:44:08

Review status: Accepted

## Background

当前目标部署形态为：

- 静态资源和前端页面由 `https://orbit.preflite.cn` 承载，Cloudflare 上保持黄云 / Proxied。
- API 请求走 `https://orbit-api.preflite.cn`，Cloudflare 上保持灰云 / DNS only，直连源站，避免动态 API 经 Cloudflare 绕路导致 TTFB 升高。
- 云端路由已支持两个 Host：`Host(`orbit.preflite.cn`) || Host(`orbit-api.preflite.cn`)`。

该形态会让浏览器页面 Origin 与 API Origin 不同：

```text
页面 Origin: https://orbit.preflite.cn
API Origin:  https://orbit-api.preflite.cn
```

因此需要后端按配置返回 Access-Control-Allow 相关响应头，并处理浏览器 preflight；同时前端所有 API 调用点都必须通过统一 API base URL，而不能继续隐式使用 `window.location.origin` 或同源 `/api/...`。

## Goal

1. 添加后端 Access-Control-Allow 相关配置：当配置存在时，后端为 API 请求设置正确的 CORS 响应头，并支持跨域预检请求。
2. 修复前端绕过统一 Axios baseURL 的调用点，确保灰云 API 域名配置后，相关 API 请求也走 `config.apiBaseUrl`。
3. 保持 `/api/ci/webhook/` 路径判断不变：用户已确认该路径没问题，本任务不把它当作路径错误修复。

## Non-goal

1. 不改变 Cloudflare DNS 或 Traefik 云端路由配置；这些已由运维侧处理。
2. 不在本任务中实现完整运行时配置注入方案，除非后续 Spec / Plan 阶段确认这是完成 API baseURL 配置的必要手段。
3. 不重构所有 API 模块；仅处理已发现的绕过统一 baseURL 的调用点。
4. 不修改 `/api/ci/webhook/` 的后端路由语义。
5. 不做兼容层、别名或新旧 API 路径并存。

## User scenarios

### Scenario 1: 前端页面从黄云域名加载，API 走灰云域名

用户打开：

```text
https://orbit.preflite.cn
```

前端页面中的 API 请求应发送到：

```text
https://orbit-api.preflite.cn/api/...
```

浏览器不得因 CORS preflight 或 `Authorization` header 被后端拒绝。

### Scenario 2: SSE / stream-log 也走灰云 API 域名

当用户查看部署日志流时，前端不应直接 `fetch('/api/...')` 到 `orbit.preflite.cn`，而应使用统一 API base URL 构造请求。

### Scenario 3: Webhook URL 展示使用 API base URL

Webhook URL 展示或复制逻辑不应依赖 `window.location.origin` 生成前端站点域名下的 API URL，而应与 API base URL 配置保持一致。

### Scenario 4: 未配置 CORS 时保持当前行为

如果未配置 Access-Control-Allow 相关配置，后端不应无条件放开跨域访问，当前同源访问行为应保持不变。

## Acceptance

1. 后端新增明确的 Access-Control-Allow 配置入口，至少能配置允许的 Origin。
2. 当允许 Origin 配置存在且请求携带匹配的 `Origin` 时，API 响应包含必要 CORS 头，例如：

```text
Access-Control-Allow-Origin: https://orbit.preflite.cn
Access-Control-Allow-Headers: Authorization, Content-Type
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
```

3. 后端能处理浏览器 preflight `OPTIONS` 请求；对允许的 Origin 返回成功状态和 CORS 头。
4. 未配置允许 Origin 时，不默认返回宽泛的 `Access-Control-Allow-Origin: *`。
5. Axios 已覆盖的 API 调用继续通过统一 `config.apiBaseUrl` 工作。
6. 直接 `fetch('/api/...')` 的调用点改为通过统一 URL 构造方式访问 API。
7. 依赖 `window.location.origin` 生成 API/Webhook URL 的调用点改为通过 API base URL 构造。
8. `/api/ci/webhook/` 路径不被错误改名或替换为其他路径。
9. 前端修改后应通过项目约定检查：

```bash
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

10. 后端修改后应通过项目约定检查：

```bash
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

## Open questions

1. Access-Control 配置项的最终命名待 Spec 阶段确定。候选方向包括：
   - `POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS`
   - `POMELO_ORBIT_HTTP__CORS_ALLOWED_ORIGINS`
   - 新增专门的 `cors` 配置段
2. 是否只允许单个 Origin `https://orbit.preflite.cn`，还是支持多个 Origin 列表，待 Spec 阶段确定。
3. 是否需要把 `Access-Control-Allow-Credentials` 纳入配置，取决于当前和短期认证方式是否只使用 Bearer token。
4. 是否在本任务中顺带实现 `window.__CONFIG__.apiBaseUrl` 的运行时注入，还是仅修复前端绕过 baseURL 的调用点，待 Spec 阶段确定。

## Decisions

1. 使用严格模式 / strict，因此本阶段只记录需求，不直接修改产品代码。
2. 用户已确认 `/api/ci/webhook/` 路径没问题，本任务不把 webhook 路径作为错误处理。
3. 目标部署形态明确为：`orbit.preflite.cn` 承载静态资源并走 Cloudflare 黄云；`orbit-api.preflite.cn` 承载 API 并走 Cloudflare 灰云。

## Risk

1. 跨域 API 使用 `Authorization` header 会触发 preflight；如果后端 OPTIONS 或 Allow-Headers 不完整，浏览器请求会失败，而 curl 仍可能成功。
2. 如果 CORS 配置过宽，例如默认 `*`，会扩大 API 暴露面。
3. 如果只修复 Axios baseURL，遗漏 `fetch`、跳转、Webhook URL 展示等非 Axios 调用点，会导致部分流量仍走 `orbit.preflite.cn` 黄云路径。
4. 如果 API base URL 采用 Vite 构建期变量写死，镜像会与环境绑定；是否引入运行时注入需要在后续方案阶段决策。

## User review notes

待用户审查。
