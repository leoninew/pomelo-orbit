# Pomelo Orbit 项目约束

- 不主动启动、停止或重启开发服务器；由用户管理。
- 项目处于活跃开发期，不做兼容层、别名、默认值兜底或新旧逻辑并存。
- API 路由统一使用单数形式，例如 `/api/ci/repository`。
- 不修改已执行的迁移文件。
- 前端目录为 `web`；修改前端后必须运行：
  - `yarn --cwd web lint:fix`
  - `yarn --cwd web typecheck`
- 前端时间展示使用 `web/src/utils/time.ts`，不要在业务代码中散落 `dayjs()`。
- 前端页面优先使用项目共享组件，不直接散落底层 Reka primitives。
- Go 后端检查使用：
  - `go fmt ./cmd/... ./internal/...`
  - `go vet ./cmd/... ./internal/...`
  - `go test ./cmd/... ./internal/...`
