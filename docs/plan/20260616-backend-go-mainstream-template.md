# backend-go 主流模板架构改进计划
最后修改时间: 2026-06-17 14:22:30

Review status: Accepted

## Basis

- Requirement: `docs/requirement/20260616-backend-go-mainstream-template.md`，状态 `Accepted`。
- Spec: `docs/spec/20260616-backend-go-mainstream-template.md`，状态 `Accepted`。

本计划只覆盖 `backend-go` 主流 Go 分层改造和开发阶段 `air` hot reload。明确不修改：

- Python `backend`；
- `jingjia` 模板相关内容；
- migration SQL；
- frontend；
- git 状态。

## Implementation strategy

采用“完整 checklist + 逐项勾选”的方式实施。每完成一个 checklist 项，应同步更新本 Plan 文档中的勾选状态，直到全部完成。

整体顺序：

1. 先建立分层基础设施和错误/响应/路由骨架。
2. 再按模块迁移 handler / service / repository。
3. 同步迁移 worker/task、CI/CD 后台执行依赖。
4. 最后移除 `internal/orbit` 和旧 `internal/httpserver` 残留。
5. 引入 `air` API/worker hot reload 配置和开发命令。
6. 运行验证，创建 verification 文档。

## Checklist

### 0. 基线和保护

- [x] 记录当前工作区已有改动，避免覆盖用户变更。
- [x] 运行或至少记录基线 Go 测试状态：`cd backend-go && go test ./...`。
- [x] 确认本次不修改 `backend-go/internal/migrations/**/*.sql`。

### 1. 分层目录和公共基础

- [x] 创建 `internal/transport/http` 分层目录：`handler`、`middleware`、`response`。
- [x] 创建 `internal/service` 分层目录。
- [x] 创建 `internal/repository` 分层目录及 `internal/repository/model`。
- [x] 创建 `internal/bootstrap`。
- [x] 创建 `internal/worker`。
- [x] 引入轻量应用错误包或等价机制，用于 `validation / unauthorized / forbidden / not found / conflict / internal`。
- [x] 将通用 JSON 响应、错误响应、分页响应、时间格式化迁移到 `transport/http/response`。
- [x] 确保新公共包不 import 旧 `internal/httpserver` 或 `internal/orbit`。

### 2. HTTP server / router 骨架

- [x] 将 `internal/httpserver/server.go` 的 server 壳和 router 注册职责迁移到 `internal/transport/http`。
- [x] 将 request logging / recover / auth context 等 HTTP 横切能力迁移到 `transport/http/middleware`。
- [x] 保持 `/api/health` 行为不变。
- [x] 保持 NotFound 响应结构为 `{"detail":"Not Found"}`。
- [x] 新 router 通过 handler 的 `Register` 方法注册路由，而不是在一个巨型 server 文件中集中堆 handler 方法。

### 3. Bootstrap / app 组装

- [x] 将 `internal/app/app.go` 中 DB、migrator、repository、service、handler、router、worker 的散装逻辑迁移到 `internal/bootstrap`。
- [x] 保留 `app.App` 的外部命令入口：`Serve`、`RunWorker`、`Migrate`、`MigrationStatus`。
- [x] `Serve` 通过 bootstrap 构建 HTTP server，不再直接创建 `orbit.Store`。
- [x] `RunWorker` 通过 bootstrap 构建 worker，不再直接创建 `orbit.Store`。
- [x] 保持现有 CLI 命令和生产启动方式不变。

### 4. Auth / User / Role 模块迁移

- [x] 将 `internal/httpserver/auth.go` 拆到 `transport/http/handler/auth` 和 `service/auth`。
- [x] 将 JWT、CSRF、login、logout、password、login-history 的业务编排从 handler 下沉到 service。
- [x] 将 Turnstile HTTP 配置读取和验证保留在合适的 handler/service 边界，避免 handler 直接访问全局 Store。
- [x] 将 `internal/httpserver/user.go` 拆到 `handler/user` 和 `service/user`。
- [x] 将 `internal/httpserver/role.go` 拆到 `handler/role` 和 `service/role`。
- [x] 将 `internal/orbit/user_store.go` 迁移到 `repository/user` 或 `repository/auth`。
- [x] 将 `internal/orbit/role_store.go` 迁移到 `repository/role`。
- [x] 将相关 DB model 从 `orbit/model.go` 迁移到 `repository/model`。
- [x] 保持 auth/user/role API 路径、响应字段和主要错误语义不变。
- [x] 更新 auth/user/role 相关测试包和 imports。

### 5. Project 模块迁移

- [x] 将 `internal/httpserver/project.go` 拆到 `transport/http/handler/project` 和 `service/project`。
- [x] 将项目创建、更新、deprecate、成员管理规则迁移到 service。
- [x] 将 `internal/orbit/project_store.go` 迁移到 `repository/project`。
- [x] 将 `Project`、`ProjectMember` 相关 DB model 迁移到 `repository/model`。
- [x] 保持 `/api/project` 及 member 路由行为不变。
- [x] 更新 project route/service/repository 测试。

### 6. Settings 模块迁移

- [x] 将 `internal/httpserver/settings.go` 拆到 `handler/settings` 和 `service/settings`。
- [x] 保持配置读取/写入行为不变。
- [x] 保持权限语义和错误响应不变。
- [x] 更新 settings 测试。

### 7. Task / background task 模块迁移

- [x] 将 `internal/task/repository.go` 中 SQL repository 迁移到 `repository/task`。
- [x] 将 task model 迁移到 `repository/model` 或 `service/task` 的输出结构。
- [x] 创建 `service/task`，封装 enqueue、find/query 等用例。
- [x] 将 `internal/task/worker.go` 和 `internal/task/router.go` 迁移到 `internal/worker`。
- [x] 将 background task HTTP endpoints 迁移到 `transport/http/handler/task`。
- [x] 保持 `/api/background/task` 和 typed enqueue endpoints 行为不变。
- [x] 更新 task repository / worker / HTTP 测试。

### 8. CI HTTP/API 模块迁移

- [ ] 拆分 `internal/httpserver/dashboard.go` 中 CI 相关 DTO 和 handler。
- [x] 将 repository API 迁移到 `transport/http/handler/ci` + `service/ci`。
- [x] 将 webhook API 迁移到 `handler/ci` + `service/ci`。
- [x] 将 pipeline template API 迁移到 `handler/ci` + `service/ci`。
- [x] 将 build stage API 迁移到 `handler/ci` + `service/ci`。
- [x] 将 pipeline run API 迁移到 `handler/ci` + `service/ci`。
- [x] 将 snapshot API 迁移到 `handler/ci` + `service/ci`。
- [x] 将 artifact API 迁移到 `handler/ci` + `service/ci`。
- [ ] 将 credential API 迁移到 `handler/ci` + `service/ci` 或共享 credential service。
- [ ] 将 `internal/orbit/repository_store.go` 迁移到 `repository/ci`。
- [ ] 将 `internal/orbit/pipeline_template_store.go` 迁移到 `repository/ci`。
- [ ] 将 `internal/orbit/pipeline_run_store.go` 迁移到 `repository/ci`。
- [ ] 将 `internal/orbit/build_stage_store.go` 迁移到 `repository/ci`。
- [ ] 将 `internal/orbit/credential_store.go` 迁移到 `repository/ci` 或 `repository/credential`。
- [ ] 将 CI 相关 DB model 从 `orbit/model.go` 迁移到 `repository/model`。
- [ ] 保持 `/api/ci/*` API 路径、响应字段、分页结构和主要错误语义不变。
- [ ] 更新 CI HTTP route tests、store tests 和 service/repository tests。

### 9. CI worker / executor 迁移

- [ ] 调整 `internal/ci/handler.go`，使其不依赖 `orbit.Store`。
- [ ] 将 CI task handler 迁移到 `internal/worker/handler/ci` 或按 Spec 中等价位置组织。
- [ ] 保留 `internal/ci` 中 executor / runner / template resolution 等底层执行能力。
- [ ] 将 CI 执行所需数据访问改为依赖 `service/ci` 或 `repository/ci` 的最小接口。
- [ ] 保持 pipeline run 状态流转、stage run、artifact 写入行为不变。
- [ ] 更新 CI handler/executor 测试。

### 10. CD HTTP/API 模块迁移

- [ ] 拆分 `internal/httpserver/dashboard.go` 中 CD application/deployment 相关 DTO 和 handler。
- [ ] 将 `internal/httpserver/application_routes_extra.go` 迁移到 `handler/cd` + `service/cd`。
- [ ] 将 `internal/httpserver/route.go` 迁移到 `handler/cd` + `service/cd`。
- [ ] 将 `internal/httpserver/traefik_route.go` 迁移到 `handler/cd` + `service/cd`。
- [ ] 将 application CRUD、config file、service config、route、compose preview/export/import 等用例放入 `service/cd`。
- [ ] 将 deployment 相关用例放入 `service/cd`。
- [ ] 将 route / Traefik route 管理用例放入 `service/cd`。
- [ ] 将 `internal/orbit/application_store.go` 迁移到 `repository/cd`。
- [ ] 将 `internal/orbit/api_store.go` 中 CD/API route 相关 SQL 迁移到 `repository/cd`。
- [ ] 将 CD 相关 DB model 从 `orbit/model.go` 迁移到 `repository/model`。
- [ ] 保持 `/api/cd/*` API 路径、响应字段和主要错误语义不变。
- [ ] 更新 CD HTTP route tests、store tests 和 service/repository tests。

### 11. CD worker / runner 迁移

- [ ] 调整 `internal/cd/handler.go`，使其不依赖 `orbit.Store`。
- [ ] 将 CD deploy/restart task handler 迁移到 `internal/worker/handler/cd` 或按 Spec 中等价位置组织。
- [ ] 保留 `internal/cd` 中 compose/render/runner 等底层执行能力。
- [ ] 将 CD 执行所需数据访问改为依赖 `service/cd` 或 `repository/cd` 的最小接口。
- [ ] 保持 application/deployment 状态流转、部署日志和 compose 写入行为不变。
- [ ] 更新 CD handler/compose/runner 测试。

### 12. `internal/orbit` 退出和清理

- [ ] 将 `orbit/model.go` 中所有仍被使用的结构迁移到新归属包。
- [ ] 将 `orbit/status.go` 中状态常量迁移到 `internal/status` 或模块内常量。
- [ ] 将 `orbit/store.go` 中 worker/CI/CD 通用方法迁移到对应 repository。
- [x] 删除所有对 `backend/internal/orbit` 的 import。
- [x] 删除 `internal/orbit` 包及其测试，或确认该目录为空并移除。
- [x] 运行全局搜索确认没有 `internal/orbit`、`orbit.` 残留。

### 13. 旧 `internal/httpserver` 退出和清理

- [x] 将所有 HTTP handler 和 test 迁移到 `internal/transport/http`。
- [x] 删除所有对 `backend/internal/httpserver` 的 import。
- [x] 删除旧 `internal/httpserver` 包及其测试，或确认该目录为空并移除。
- [x] 运行全局搜索确认没有 `internal/httpserver` 残留。

### 14. Air hot reload

- [x] 新增 `backend-go/.air.api.toml`，构建 `./cmd/backend-go` 并运行 `serve`。
- [x] 新增 `backend-go/.air.worker.toml`，构建 `./cmd/backend-go` 并运行 `worker`。
- [x] API 和 worker 使用不同输出文件，例如 `./tmp/air/api.exe` 与 `./tmp/air/worker.exe`。
- [x] air 配置排除 `tmp`、`bin`、`data`、日志目录和其它构建产物目录。
- [x] 更新 `.gitignore`，忽略 air 临时产物，例如 `backend-go/tmp/` 和 `backend-go/bin/`（如当前未被忽略）。
- [x] 更新 `justfile`：将 `dev-backend` 切换或新增为 `air -c .air.api.toml`。
- [x] 更新 `justfile`：将 `dev-worker` 切换或新增为 `air -c .air.worker.toml`。
- [x] 评估并更新 `scripts/dev.py`：开发组合启动时是否直接调用 air API/worker，或继续使用一次性 build binary。若改用 air，应保持前端/API/worker 同时启动和退出清理行为。
- [x] 记录 `air` 安装前置条件；若项目不 vendoring 工具，则不把 air 加入生产依赖。

### 15. Imports、格式化和测试迁移

- [x] 更新所有 Go import 路径。
- [x] 运行 `gofmt` / `go fmt ./...`。
- [x] 运行 `go vet ./...`。
- [x] 运行 `go test ./...`。
- [x] 若 route tests 因包路径变更失败，迁移测试 helper 并保持原断言语义。
- [x] 确认无 migration SQL diff。
- [x] 确认无 frontend diff，除非用户另行授权。

## Files to change

预计会修改或新增：

- `backend-go/internal/app/app.go`
- `backend-go/internal/bootstrap/**`
- `backend-go/internal/transport/http/**`
- `backend-go/internal/service/**`
- `backend-go/internal/repository/**`
- `backend-go/internal/worker/**`
- `backend-go/internal/ci/**`
- `backend-go/internal/cd/**`
- `backend-go/internal/task/**`（迁移后可删除或缩减）
- `backend-go/internal/httpserver/**`（迁移后删除）
- `backend-go/internal/orbit/**`（迁移后删除）
- `backend-go/.air.api.toml`
- `backend-go/.air.worker.toml`
- `.gitignore`
- `justfile`
- `scripts/dev.py`（如采用 air 作为组合开发启动方式）
- 相关 Go 测试文件路径随包迁移调整。

明确不修改：

- `backend-go/internal/migrations/**/*.sql`
- `backend/migrations/**`
- `frontend/**`

## Verification plan

从仓库根目录或 `backend-go` 目录运行：

```powershell
cd backend-go && go test ./...
```

```powershell
cd backend-go && go vet ./...
```

```powershell
cd backend-go && go fmt ./...
```

如果使用项目统一入口：

```powershell
just check
```

注意：`just check` 会运行 frontend 检查；若本次没有前端变更但命令失败，需要记录失败是否与本次后端改造无关。

Air 配置验证：

```powershell
cd backend-go && air -c .air.api.toml
```

```powershell
cd backend-go && air -c .air.worker.toml
```

不主动长期启动/停止开发服务器；若需要实际运行 hot reload，应在 Verification 阶段按用户授权执行，或记录需用户本地运行验证。

结构验证：

- 搜索确认无 `backend/internal/orbit` import。
- 搜索确认无 `backend/internal/httpserver` import。
- 搜索确认 `backend-go/internal/migrations/**/*.sql` 无 diff。
- 搜索确认 handler 不直接访问 `sqlx.DB` 或 repository 具体 SQL。
- 搜索确认 service 不 import `transport/http`。
- 搜索确认 repository 不 import `service` 或 `transport/http`。

## Blockers

暂无已知阻塞项。

如果实现中发现必须修改 migration SQL、前端或 Python backend，应停止当前实现并回到 Requirement / Spec 更新范围。

## Assumptions

- 现有 `chi + sqlx + viper + slog` 技术栈继续保留。
- `air` 作为开发工具使用，不作为生产依赖或生产启动方式。
- API 和 worker 现有 CLI 子命令 `serve` / `worker` 保持可用。
- 当前 API 路径、响应结构和主要错误语义应保持稳定。
- 可以在迁移过程中短暂保留 `orbit`，但最终必须删除。

## Risks

- 全量架构改造影响范围大，单次 diff 可能很大；通过 checklist 降低不可见风险。
- 大量文件移动会导致 import、测试包名、helper 迁移错误。
- CI/CD 和 worker 的行为回归风险高，需要优先保留现有测试语义。
- `air` 配置若 watch 构建产物可能循环重启，需要严格 exclude。
- `scripts/dev.py` 当前一次构建同一个 binary 同时启动 API 和 worker；切换到 air 后需要确保两个进程独立构建输出且退出清理正常。

## Rollback

- 不执行 git 写操作；如需要回滚，由用户使用 git 工具处理。
- 每个 checklist 项完成后应保持 `go test ./...` 尽量可恢复通过，减少大段不可回滚改动。
- 若某个模块迁移风险过大，应先停止并报告，不继续扩大变更范围。
- 若发现必须修改 migration SQL，则停止实现并更新 Spec/Plan，不在本计划内直接修改。

## User review notes

- 待用户 review。
