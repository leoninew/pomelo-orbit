# PomeloOrbit Vite 运行时 API Origin 改进验证
最后修改时间: 2026-07-03 14:16:38

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260703-runtime-api-origin-refinement.md` 核对，本次实现覆盖轻量模式 / light 需求：

- 已将运行时公开 API URL 前缀从 `api_base_url` / `apiBaseUrl` 收敛为 `public_url` / `publicUrl`。
- 已移除 `web/index.html` 中额外的 `window.__CONFIG__ = window.__CONFIG__ || {}` fallback script，仅保留 `<head>` 内唯一 `<!-- __RUNTIME_CONFIG__ -->` marker。
- 已将前端 `buildApiUrl` 改为 API path namespace 断言 + `publicUrl + path` 拼接，不再补救 `api/...` 这类错误输入。
- 已将后端 API missing route 与 CORS path 判断改为配置化 prefix helper，默认 namespace 为 `/api`，覆盖 `/api` 与 `/api/...`，拒绝 `/apix`。
- 已在配置加载阶段 normalize / validate `server.public_url`、`server.cors_allowed_origins` 与 `server.api_path_prefixes`。
- 已补充 Go 测试和前端 Vitest，覆盖 runtime config 优先级、路径断言、CORS prefix 与配置校验。
- 已按用户要求不修改历史过程文档事实记录，也未检查或修改 CI/CD 配置。

## Spec alignment

不适用。当前流程为轻量模式 / light，本功能未创建独立 Spec 文档。

## Plan alignment

不适用。当前流程为轻量模式 / light，本功能依据已接受的 Requirement 直接实现。

## Actual diff summary

当前项目变更范围如下：

- 过程文档
  - `docs/requirement/20260703-runtime-api-origin-refinement.md`
  - `docs/verification/20260703-runtime-api-origin-refinement.md`
- 配置与示例
  - `.env.example`
  - `configs/config.yaml`
  - `configs/config.example.yaml`
  - `web/.env.development`
  - `web/.env.remote`
- 后端配置加载
  - `internal/config/config.go`
  - `internal/config/config_test.go`
- 后端 HTTP server 与 middleware
  - `internal/transport/http/server.go`
  - `internal/transport/http/server_test.go`
  - `internal/transport/http/middleware/cors.go`
  - `internal/transport/http/middleware/cors_test.go`
- 前端 runtime config
  - `web/index.html`
  - `web/src/config.ts`
  - `web/src/config.spec.ts`
  - `web/src/env.d.ts`
  - `web/src/utils/request.ts`

主要行为变化：

- 配置结构使用 `server.public_url` 与 `server.api_path_prefixes`，环境变量使用 `POMELO_ORBIT_SERVER__PUBLIC_URL` 与 `POMELO_ORBIT_SERVER__API_PATH_PREFIXES`。
- `Load()` 和 base config 加载会 normalize `server.public_url`、`server.cors_allowed_origins` 与 `server.api_path_prefixes`。
- `server.public_url` 非空时要求 absolute `http` / `https` URL，不允许 query 或 fragment。
- `server.cors_allowed_origins` 要求 absolute `http` / `https` origin，不允许 path、query 或 fragment。
- `server.api_path_prefixes` 必须非空，prefix 必须以 `/` 开头，且不允许使用根路径 `/`。
- 后端注入 runtime config 时输出 `window.__CONFIG__ = {"publicUrl":"..."}`；空 `public_url` 注入 `{}`。
- `web/index.html` 只保留唯一 runtime config marker，不再保留第二段默认空对象脚本。
- 前端读取 `window.__CONFIG__.publicUrl`，且只有在 `window.__CONFIG__` 不存在时才使用构建期 `VITE_PUBLIC_URL`。
- 前端 `buildApiUrl` 只接受 `/api` 或 `/api/...`，并返回 `getPublicUrl() + path`。
- Axios request 的 `baseURL` 改为 `config.publicUrl`。
- 后端 API 404 和 CORS path 判断统一使用 `server.api_path_prefixes` 配置驱动的 API prefix helper。

## Expected vs actual changed files

| 预期 | 实际 | 结论 |
| --- | --- | --- |
| 配置结构、YAML 示例和 `.env.example` 改为 `public_url` / `POMELO_ORBIT_SERVER__PUBLIC_URL`，并增加 `api_path_prefixes` / `POMELO_ORBIT_SERVER__API_PATH_PREFIXES` | `.env.example`、`configs/config.yaml`、`configs/config.example.yaml` 已更新 | 符合 |
| 后端配置加载支持新字段并校验 URL / Origin / API prefix | `internal/config/config.go`、`config_test.go` 已更新 | 符合 |
| 后端注入 runtime config 字段为 `publicUrl` | `internal/transport/http/server.go`、`server_test.go` 已更新 | 符合 |
| HTML 模板只有一个 `<head>` marker，无额外 fallback script | `web/index.html` 已更新 | 符合 |
| CORS 与 API 404 使用配置化 prefix helper | `server.go`、`middleware/cors.go` 及测试已更新 | 符合 |
| 前端 runtime config 与 URL builder 使用 `publicUrl` 和断言式 path | `web/src/config.ts`、`config.spec.ts` 已更新 | 符合 |
| 当前实现路径不再使用旧运行时 API Origin 命名 | 已检查 `internal/`、`web/`、`configs/`、`.env.example`，无旧命名匹配 | 符合 |
| 不修改历史过程文档和 CI/CD 配置 | 历史 `docs/spec`、`docs/plan`、`docs/verification` 与 CI/CD 配置未改 | 符合 |

## Acceptance checklist

- [x] 配置结构、YAML 示例和 `.env.example` 使用 `server.public_url` / `POMELO_ORBIT_SERVER__PUBLIC_URL` 表达公开 API URL 前缀，并使用 `server.api_path_prefixes` / `POMELO_ORBIT_SERVER__API_PATH_PREFIXES` 表达 API path namespace。
- [x] 当前实现路径中不再使用 `api_base_url`、`APIBaseURL`、`apiBaseUrl`、`VITE_API_BASE_URL` 作为运行时 API Origin 推荐配置名。
- [x] `web/index.html` 中仅保留一个 `<head>` 内 `<!-- __RUNTIME_CONFIG__ -->` marker，不包含 `window.__CONFIG__ = window.__CONFIG__ || {}` fallback script。
- [x] 后端注入 `window.__CONFIG__` 时输出字段为 `publicUrl`；空 `public_url` 注入 `{}`。
- [x] 前端 runtime config 读取使用 `publicUrl`，并通过 `typeof window !== 'undefined'` 容忍非浏览器测试环境。
- [x] `buildApiUrl` 只接受当前允许的 API path prefix：`/api` 和 `/api/...`，并返回 `getPublicUrl() + path`。
- [x] CORS middleware 对 API path 的判断使用配置化 prefix helper，默认覆盖 `/api`、`/api/...`，拒绝 `/apix`。
- [x] API missing route 判断与 CORS path 判断语义一致，避免 SPA fallback 吃掉 API 404。
- [x] 配置加载对 `server.public_url`、`server.cors_allowed_origins` 和 `server.api_path_prefixes` 做 normalize / validate。
- [x] 前端 Vitest、Go 单元测试和主要显式检查已通过。

## Test results

已运行并通过：

```bash
go test ./internal/config ./internal/transport/http/...
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
yarn --cwd web run test
yarn --cwd web run typecheck
yarn --cwd web run lint
yarn --cwd web prettier --check src/config.ts src/config.spec.ts src/utils/request.ts
yarn --cwd web run build
git diff --check
```

其中：

- `go test ./cmd/... ./internal/...` 通过所有 Go 包测试。
- `yarn --cwd web run test` 通过 3 个 test files / 14 个 tests。
- `yarn --cwd web run build` 构建成功。
- `git diff --check` 通过。

未全量通过但与本次改动无关：

```bash
yarn --cwd web run format:check
```

失败文件为既有未格式化文件：

- `src/api/cd/deployments.ts`
- `src/views/cd/ApplicationDetail.vue`
- `src/views/Settings.vue`

本次修改的前端文件已单独通过 Prettier 检查：

```bash
yarn --cwd web prettier --check src/config.ts src/config.spec.ts src/utils/request.ts
```

已知构建输出：`yarn --cwd web run build` 仍会输出来自 `reka-ui/node_modules/@vueuse/core` 的 Rollup `#__PURE__` annotation warning，但构建成功。该 warning 来自依赖包注释位置，不是本次实现引入的失败。

## Missed or expanded scope

- 未修改历史过程文档中旧 `api_base_url` / `apiBaseUrl` 事实记录，符合用户“历史文档不用改”的指令。
- 未检查或修改 CI/CD 配置，符合用户“ci/cd 配置无须关心”的指令。
- 曾尝试同步当前前端架构文档中的示例命名，但该 Markdown 文件存在 CRLF / whitespace 检查噪音；为避免扩大范围并保持 `git diff --check` 通过，已撤回该文档改动。
- 未引入多后端服务公开 URL 配置；当前只将 `/api` 整理成可配置的 `server.api_path_prefixes` 模型。
- 未启用 CORS credentials，未改变认证、鉴权、数据库或业务 API 语义。

## Risks

- 这是配置合同变更：已有部署若设置 `POMELO_ORBIT_SERVER__API_BASE_URL` 或前端 `VITE_API_BASE_URL`，需要同步改为 `POMELO_ORBIT_SERVER__PUBLIC_URL` / `VITE_PUBLIC_URL`；本任务按用户要求未检查 CI/CD 配置。
- `server.host` / `server.port` 仍只表示后端监听地址，不能推导浏览器公开 API URL；拆分 Origin 部署必须显式配置 `server.public_url`。
- 当前 API namespace 默认配置为 `/api`。未来新增 `/graphql`、`/report-api` 或多后端服务时，需要同步扩展 `server.api_path_prefixes`、前端 prefix 列表与公开 URL 前缀模型。
- 前端 Axios client 在模块加载时读取 `config.publicUrl`；本次默认 runtime config 在入口脚本执行前已注入。

## Incomplete items

暂无本次需求范围内未完成项。

外部既有问题：`yarn --cwd web run format:check` 会因 3 个非本次修改文件失败，未在本任务中修复。

## Conclusion

验证通过。当前实现满足轻量模式 / light requirement 中的运行时 API Origin 命名收敛、唯一 runtime marker、断言式 API URL builder、后端 prefix 判断、配置 normalize / validate、示例配置和测试覆盖要求。
