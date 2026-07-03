# PomeloOrbit Vite 运行时 API Origin 改进需求
最后修改时间: 2026-07-03 14:05:00

Review status: Accepted

## Background

基于 `D:\SourceCodes\mywork\k12-force` 的落地经验，以及 `D:\SourceCodes\mywork\best-practices\docs\guides\vite-runtime-api-origin-integration.md` 与 `vite-runtime-api-origin-integration.manual.md` 的修订结果，需要检查并改进 PomeloOrbit-go 现有 Vite runtime API Origin 实现。

当前 PomeloOrbit-go 已经具备运行时配置注入、CORS middleware、Vite dev proxy 和前端 API URL 构造能力，但仍保留了部分旧口径或可改进点：`api_base_url` / `apiBaseUrl` 命名、HTML fallback script、`buildApiUrl` 自动补路径、CORS 只用硬编码 `/api/` 判断，以及配置 URL / Origin 缺少启动期校验。

## Goal

- 将运行时公开 API URL 前缀语义从 `api_base_url` / `apiBaseUrl` 收敛为 `public_url` / `publicUrl`，不新增影子配置。
- 保持 `web/index.html` 中只有一个 `<head>` 内 runtime config marker，不在 body 中注入，不保留额外 fallback script。
- 将前端 `buildApiUrl` 改为 API path namespace 断言 + 公开 API URL 前缀拼接，不做路径补救。
- 将后端 CORS 和 API 404 判断改为可配置的项目级 API path prefix 规则，当前默认 namespace 为 `/api`。
- 将公开 API URL 前缀和 CORS allowlist 的 normalize / validate 前移到配置加载与启动校验阶段。
- 补充或修正前后端测试，覆盖 runtime config 优先级、路径断言、CORS prefix 和配置校验。
- 清理当前实现文档与配置示例中的旧命名，避免继续传播 `api_base_url` / `apiBaseUrl` 作为推荐字段。

## Non-goal

- 不引入 Gin，不改变当前 chi / net/http 架构。
- 不启用 CORS credentials，也不引入跨站 cookie、CSRF 或认证重构。
- 不新增多后端服务公开 URL 配置；本次只把现有 `/api` 规则整理成可配置的 prefix 模型。
- 不保留 `api_base_url` / `apiBaseUrl` / `VITE_API_BASE_URL` 的兼容 alias 或过渡字段。
- 不重构业务 API、数据库、CI/CD 领域逻辑。
- 不修改历史过程文档的事实记录。

## User scenarios

- 同源部署：`server.public_url` 为空，前端请求 `/api/...`。
- 拆分 Origin 部署：`server.public_url` 为 API origin，后端向 `index.html` 注入 `window.__CONFIG__.publicUrl`，前端请求 `https://api.example.com/api/...`。
- 错误调用：前端调用 `buildApiUrl('api/health')` 或 `buildApiUrl('/apix/health')` 时立即失败，而不是静默补救。
- 跨域浏览器请求：只有 Origin 命中 `server.cors_allowed_origins` 且 path 属于 API prefix 时，后端返回 CORS header；allowed preflight 返回 `204`，denied preflight 返回 `403`。
- 错误配置：`server.public_url` 或 `server.cors_allowed_origins` 中的非法 URL / Origin 在启动期失败，而不是运行期静默降级。

## Acceptance

- 配置结构、YAML 示例和 `.env.example` 使用 `server.public_url` / `POMELO_ORBIT_SERVER__PUBLIC_URL` 表达公开 API URL 前缀。
- 当前实现路径中不再使用 `api_base_url`、`APIBaseURL`、`apiBaseUrl`、`VITE_API_BASE_URL` 作为运行时 API Origin 推荐配置名。
- `web/index.html` 中仅保留一个 `<head>` 内 `<!-- __RUNTIME_CONFIG__ -->` marker，不包含 `window.__CONFIG__ = window.__CONFIG__ || {}` fallback script。
- 后端注入 `window.__CONFIG__` 时输出字段为 `publicUrl`；空 `public_url` 注入 `{}`。
- 前端 runtime config 读取使用 `publicUrl`，并通过 `typeof window !== 'undefined'` 容忍非浏览器测试环境。
- `buildApiUrl` 只接受项目允许的 API path prefix，当前为 `/api` 和 `/api/...`，并返回 `getPublicUrl() + path`。
- CORS middleware 对 API path 的判断使用配置化 prefix helper，默认覆盖 `/api`、`/api/...`，拒绝 `/apix`。
- API missing route 判断与 CORS path 判断语义一致，避免 SPA fallback 吃掉 API 404。
- 配置加载对 `server.public_url`、`server.cors_allowed_origins` 和 `server.api_path_prefixes` 做 normalize / validate。
- 前端 Vitest、Go 单元测试和项目显式检查入口通过。

## Open questions

暂无必须阻塞实现的未决事项。

需要实现阶段核对的事项：

- 当前仓库是否有 CI / 部署脚本仍显式设置 `POMELO_ORBIT_SERVER__API_BASE_URL` 或 `VITE_API_BASE_URL`；若有，应同步改名而不是保留 alias。
- 历史 SpecFlow 文档是否只作为历史记录存在；若是，不应为了“全仓库无旧词”而大规模改写历史记录。

## Decisions

- 公开 API URL 前缀字段使用 `server.public_url` 和 runtime `window.__CONFIG__.publicUrl`。
- 不保留 `api_base_url` / `apiBaseUrl` / `VITE_API_BASE_URL` 的 temporary alias。
- `buildApiUrl` 不做路径补救，只做 API namespace 断言和前缀拼接。
- 当前项目 API namespace 默认配置为 `/api`；未来新增 namespace 时扩展 `server.api_path_prefixes`。
- 后端继续使用当前 chi / net/http middleware 形态，不引入 Gin。
- `</head>` fallback 可以作为后端容错保留，但模板中只允许一个显式 marker。

## Risk

- 这是配置合同变更：如果现有运行环境使用 `POMELO_ORBIT_SERVER__API_BASE_URL` 或 `VITE_API_BASE_URL`，需要在运行配置中同步改为新字段名；本任务不检查 CI/CD 配置。
- 历史过程文档不作为本次清理范围。
- 前端 request client 当前在模块加载时读取 baseURL；若未来 runtime config 不是在入口脚本前注入，需要额外设计动态读取机制。本次默认 runtime config 在应用入口前可用。
- 当前只规划 `/api` prefix；未来多后端服务需要扩展 path prefix 与公开 URL 前缀模型。
