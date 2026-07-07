# HTTP 静态资源访问日志过滤开关需求
最后修改时间: 2026-07-07 09:54:16

Review status: Accepted

## Background

当前 HTTP access log middleware 全局挂在后端 router 上，前端构建产物请求也会输出 `request started` 和 `request completed`。例如正常访问 `/assets/route-*.js` 会产生访问日志，导致日志中出现大量静态资源 200 请求噪音。

## Goal

新增一个默认启用的日志配置开关，用于跳过 `/assets` 下 `.js`、`.css`、`.html` 文件的 HTTP 200 访问日志，减少正常静态资源访问带来的日志噪音。

## Non-goal

- 不关闭 API 请求日志。
- 不关闭静态资源非 200 请求日志。
- 不改变静态文件服务、SPA fallback、CORS 或 panic recovery 行为。
- 不修改 HTTP body logging 的现有语义。
- 不支持 `/asserts` 拼写；本需求按示例和前端构建目录实现 `/assets`。

## User scenarios

- 作为运维或开发者，正常访问前端页面时，不希望日志中充满 `/assets/*.js`、`/assets/*.css`、`/assets/*.html` 的 200 请求。
- 作为排障者，当 `/assets/app.js` 返回 404、500 或其他非 200 状态时，仍希望保留访问日志。
- 作为开发者，需要能通过配置或环境变量关闭该过滤行为，以恢复完整 access log。

## Acceptance

- 默认配置启用该过滤行为。
- 新增配置项可通过 YAML、环境变量和 settings 服务展示。
- `configs/config.yaml` 为新增配置提供注释说明。
- 删除不再维护的 `configs/config.example.yaml`。
- `/assets` 下 `.js`、`.css`、`.html` 的 200 请求不输出 `request started` 和 `request completed`。
- `/assets` 下目标扩展名的非 200 请求仍输出完整 started/completed 日志。
- 非目标路径或非目标扩展名请求仍按原逻辑输出访问日志。
- API 请求访问日志保持不变。
- 现有 HTTP body logging、panic recovery 和 response writer optional interface 行为保持不变。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 配置命名采用 `logging.http_skip_assets_200_enabled`。
- 环境变量命名采用 `POMELO_ORBIT_LOGGING__HTTP_SKIP_ASSETS_200_ENABLED`。
- 仅匹配 URL path 中 `/assets/` 前缀，不匹配 `/assets-old/` 或 `/asserts/`。
- 仅跳过状态码恰好为 200 的请求，不跳过 304。

## Risk

- 为了完全不记录匹配请求，需要在候选 `/assets/*.js|css|html` 请求上延迟写出 `request started`，直到拿到最终状态码。候选路径的非 200 请求仍会写 started/completed，但写出时间会在 handler 返回后。
