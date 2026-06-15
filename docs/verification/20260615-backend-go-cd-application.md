# backend-go cd application 相关接口迁移验证

Review status: Accepted

## What changed

- `backend-go/internal/httpserver/dashboard.go`
  - 注册并实现 `/api/cd/application` 基础 CRUD 路由。
  - 列表、详情、创建、更新、删除接口接入当前用户鉴权和项目成员校验。
  - 创建/更新请求补充字段校验：名称、编码、镜像拉取策略、路由托管开关。
  - 删除接口按应用状态阻止 deploying/deployed 应用删除，并支持 `remove_dir=true` 删除应用目录。
- `backend-go/internal/httpserver/application_routes_extra.go`
  - 注册表中剩余 `/api/cd/application/...` 子资源接口已实现：
    - `POST /api/cd/application/import`
    - `GET /api/cd/application/{app_id}/export`
    - `POST /api/cd/application/{app_id}/compose-preview`
    - `GET /api/cd/application/{app_id}/files`
    - `POST /api/cd/application/{app_id}/file`
    - `GET /api/cd/application/{app_id}/file/{file_id}`
    - `PUT /api/cd/application/{app_id}/file/{file_id}`
    - `DELETE /api/cd/application/{app_id}/file/{file_id}`
    - `POST /api/cd/application/{app_id}/deploy`
    - `POST /api/cd/application/{app_id}/stop`
    - `POST /api/cd/application/{app_id}/restart`
    - `GET /api/cd/application/{app_id}/status`
    - `GET /api/cd/application/{app_id}/logs`
    - `GET /api/cd/application/{app_id}/route`
    - `POST /api/cd/application/{app_id}/route`
    - `PUT /api/cd/application/{app_id}/route/{route_id}`
    - `DELETE /api/cd/application/{app_id}/route/{route_id}`
    - `GET /api/cd/application/{app_id}/compose-service`
    - `GET /api/cd/application/{app_id}/service-config`
    - `PUT /api/cd/application/{app_id}/service-config/{service_name}`
  - 实现 import/export、配置文件、compose preview、deploy/stop/restart/status/logs、应用 route、compose-service、service-config 处理逻辑。
- `backend-go/internal/orbit/application_store.go`
  - 新增 application 相关仓储写操作：按名称/编码查询、创建、批量导入、更新、删除。
  - 新增配置文件、应用路由、service-config、deployment 创建等仓储操作。
- `backend-go/internal/httpserver/application_routes_test.go`
  - 覆盖 application CRUD、参数校验、重复名称、未认证、运行中删除拦截。
  - 覆盖 application 子资源接口认证、import/export、file、compose-service、compose-preview、service-config 可测路径。
- `backend-go/internal/httpserver/server_test.go`
  - 调整 dashboard 列表测试，为 `/api/cd/application` 增加认证头。
- `docs/todo.md`
  - 将所有 `/api/cd/application` 相关接口更新为已迁移，并调整迁移统计。

## Acceptance

- [x] backend-go 已注册 `docs/todo.md` 中所有 `/api/cd/application` 相关路由。
- [x] CRUD、导入导出、配置文件、部署运行、route、compose-service、service-config 子资源接口均有 handler。
- [x] application 相关仓储操作已补齐。
- [x] 新增/更新 application 接口测试。
- [x] 运行 backend-go Go 测试并通过。
- [x] `docs/todo.md` 已更新迁移进度。

## Commands

- `gofmt -w "backend-go/internal/httpserver/dashboard.go" "backend-go/internal/httpserver/application_routes_test.go" "backend-go/internal/orbit/application_store.go" && go test ./...`
  - 结果：失败。
  - 原因：在仓库根目录执行，根目录不是 Go module。
  - 输出要点：`pattern ./...: directory prefix . does not contain main module or its selected dependencies`。
- `go -C "backend-go" test ./internal/httpserver -run "TestApplication" -v`
  - 结果：通过。
- `gofmt -w "backend-go/internal/httpserver/dashboard.go" "backend-go/internal/httpserver/application_routes_extra.go" "backend-go/internal/httpserver/application_routes_test.go" "backend-go/internal/httpserver/server_test.go" "backend-go/internal/orbit/application_store.go" && go -C "backend-go" test ./...`
  - 结果：通过。

## Remaining risk

- `status`、`logs`、`stop` 会调用本机 `docker compose`，测试覆盖以路由注册、认证和可测业务路径为主，没有在测试中实际启动 Docker。
- `/api/cd/application/{app_id}/stop` 在 handler 内同步执行 docker compose down；现有 Python 后端也是同步 stop。deploy/restart 继续使用 backend-go 既有后台任务队列。
- `docs/todo.md` 在本次工作前已有未提交变更；本次只更新其中 Backend API Migration TODO 的 application 相关接口状态与统计。
- 工作区还存在本次未处理的无关变更/未跟踪文件，例如 Makefile、backend-go/bin、justfile 等，提交前建议拆分核对。
