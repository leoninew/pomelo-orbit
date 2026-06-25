# backend-go 主流模板架构改进规格

Review status: Accepted

## Requirement basis

基于已接受需求：`docs/requirement/20260616-backend-go-mainstream-template.md`。

本次只改进 `backend-go` 自身架构，不涉及：

- Python `backend` 修改；
- `jingjia` 模板相关修改；
- migration SQL 修改；
- 前端修改；
- git 写操作。

已确认方向：不引入 go-zero / Kratos / GoFrame 等大框架，保留现有 `chi + sqlx + viper + slog` 技术栈，改成主流 Go 分层结构。

新增范围：引入 `https://github.com/air-verse/air`，用于开发阶段 hot reload，并且 API 与 worker 都要生效。

## Overview

当前 `backend-go` 的主要问题不是技术栈，而是职责集中：

- `internal/httpserver` 同时承担路由注册、请求解析、业务编排、权限判断、DTO 映射、错误响应和部分数据规则。
- `internal/orbit` 同时承担 DB model、全局 Store、repository、状态常量和部分业务语义。
- `internal/app/app.go` 直接组装 DB、migration、Store、HTTP server、task worker 和 CI/CD handler。
- CI/CD worker 已经有局部接口抽象，但仍依赖 `orbit` 模型和 Store。

目标规格是把依赖方向收敛为：

```text
transport/http handler -> service/usecase -> repository -> db/sqlx
bootstrap -> 组装所有依赖
worker -> service/usecase 或 worker handler -> repository / runner
```

HTTP 层不直接访问 SQL Store；业务编排不放在 handler；SQL 不散落在 HTTP 文件；`orbit` 不作为长期中心包保留。

开发阶段通过 `air` 提供 API 与 worker 的热重载入口，但不改变生产启动方式。

## Target package layout

目标采用朴素 Go 分层，不照搬重型 DDD 目录。

```text
backend-go/
  cmd/backend-go/

  internal/
    app/                      # 应用生命周期：serve / worker / migrate / status
    bootstrap/                # 依赖组装，替代 app.go 中的大量手工散装逻辑

    config/
    logging/
    db/                       # 保留现有 DB 打开、方言、migrator；本次不改 migration SQL
    migrations/               # 保留现状

    transport/http/
      server.go               # HTTP server 壳
      router.go               # 路由集中注册
      middleware/             # logging / recover / auth context 等
      response/               # JSON、分页、错误响应、时间格式
      handler/
        auth/
        user/
        role/
        project/
        settings/
        ci/
        cd/
        task/

    service/
      auth/
      user/
      role/
      project/
      settings/
      ci/
      cd/
      task/

    repository/
      model/                  # sqlx db tag model
      auth/
      user/
      role/
      project/
      ci/
      cd/
      task/

    worker/
      router.go
      worker.go
      handler/
        ci/
        cd/

    ci/                       # CI 底层执行能力：executor / runner / template resolution
    cd/                       # CD 底层执行能力：compose / runner / deploy file rendering
    security/
    status/
    templatex/
```

说明：

- `transport/http` 替代当前 `httpserver`，只保留 HTTP 相关职责。
- `service` 承担用例编排和业务规则，不 import `transport/http`。
- `repository` 承担 SQL 和持久化模型，不 import `service` 或 `transport/http`。
- `bootstrap` 是唯一允许同时知道 handler、service、repository、config、db 的组装层。
- `db` 和 `migrations` 保留现有行为；本次不调整 migration 文件。

## Design decisions

### 1. 保留轻量技术栈

继续使用：

- `chi`：HTTP router；
- `sqlx`：数据库访问；
- `viper`：配置；
- `slog`：日志。

不引入大框架，不把问题转移成框架迁移。

### 2. 采用 handler / service / repository 分层

每个业务模块使用一致结构：

```text
handler:    HTTP request/response、路径参数、query、body 解析
service:    业务用例、权限/状态/流程编排、调用多个 repository
repository: SQL、事务、model 映射
model:      持久化结构体，包含 db tag
```

handler 可以做基础请求格式解析，但不承载跨 repository 的业务规则。

### 3. DTO 边界

- HTTP request / response DTO 放在对应 `transport/http/handler/<module>` 包内。
- service 输入输出使用 service DTO 或业务结构，避免暴露 `http.Request` / `http.ResponseWriter`。
- repository 返回 repository/model 或模块实体，由 service 做必要组合。
- 时间格式、分页响应、错误响应集中到 `transport/http/response`。

### 4. 错误处理

引入轻量应用错误类型，例如：

```text
service.ErrValidation
service.ErrUnauthorized
service.ErrForbidden
service.ErrNotFound
service.ErrConflict
```

或等价的 `apperror` 包。

handler 统一调用 response 写错误：

```text
err -> HTTP status + {"detail": "..."}
```

避免每个 handler 分散写重复的 400/401/403/404/500 判断。

### 5. 权限和当前用户

当前 `currentUser`、`requirePermission` 等逻辑从 `httpserver` 中拆出：

- HTTP middleware 或 auth helper 负责解析 token、加载当前用户上下文；
- service 接收 `currentUserID` 或 `Actor`，执行业务权限判断；
- handler 不直接访问全局 Store 查询权限。

### 6. `internal/orbit` 退出策略

`internal/orbit` 不作为长期兼容层保留。拆分方向：

- DB model -> `internal/repository/model`；
- Store SQL -> 各 `internal/repository/<module>`；
- API/业务状态常量 -> `internal/status` 或对应 service/module；
- `NewId` 等通用工具 -> 独立小包，或放到需要它的 service/repository 内；
- CI/CD 使用的结构 -> 迁入 `service/ci`、`service/cd` 或 `repository/model`，按职责归属拆分。

实施阶段可以短暂保留 `orbit` 以保证可编译，但 plan checklist 必须明确每个遗留项的删除路径。用户已接受该退出策略。

### 7. Worker 分层

当前 `internal/task` 同时包含 repository、worker、router 和 task model。目标拆分为：

- `repository/task`：`background_task` SQL；
- `service/task`：enqueue / query 等用例；
- `worker`：claim/execute/fail/complete 循环和 handler router；
- `transport/http/handler/task`：背景任务 HTTP API。

CI/CD 后台任务 handler 不再依赖 `orbit.Store`，改依赖对应 service 或 repository interface。

### 8. CI/CD 模块拆分

CI/CD 是最大风险区，规格上不改变业务行为，只改变边界：

- CI HTTP API -> `transport/http/handler/ci`；
- CI 用例，如 repository、template、pipeline run、artifact、webhook -> `service/ci`；
- CI SQL -> `repository/ci`；
- CI 执行器、容器 runner、模板渲染 -> 保留或整理到 `internal/ci`；
- CD HTTP API -> `transport/http/handler/cd`；
- CD 用例，如 application、deployment、route、traefik route -> `service/cd`；
- CD SQL -> `repository/cd`；
- CD compose/render/runner -> 保留或整理到 `internal/cd`。

### 9. Bootstrap

`internal/app/app.go` 不再直接散装所有依赖。目标：

- `app.App` 保留命令入口：`Serve`、`RunWorker`、`Migrate`、`MigrationStatus`；
- `bootstrap` 创建 DB、repository、service、handler、router、worker；
- HTTP server 构造只接收 config/logger/router 或 handler 集合，不接收全局 Store。

### 10. Air hot reload

引入 `air-verse/air` 作为开发阶段 hot reload 工具。

设计原则：

- API 和 worker 使用独立配置，避免共用 build 输出互相覆盖。
- hot reload 只服务开发阶段，不改变生产构建和运行命令。
- air 生成的临时文件放入 backend-go 内部临时目录，并在 `.gitignore` 或等价忽略配置中排除。
- watch 范围覆盖 Go 源码和必要配置文件，排除 data/tmp/log/build 产物，避免循环触发。

建议文件形态：

```text
backend-go/.air.api.toml
```

建议命令形态：

```powershell
air -c .air.api.toml
```

或通过项目已有任务入口包装成：

```powershell
just backend-go-dev-api
```

具体命令命名在 Plan 阶段结合 `justfile` / 当前开发脚本确认。

API 配置应构建并运行：

```text
go build -o ./tmp/air/api.exe ./cmd/backend-go
./tmp/air/api.exe serve
```

原计划包含独立 worker air 配置；后续单节点运行时已删除 worker air 文件，worker loop 并入 `serve` 启动路径。

Windows 下实际扩展名和 air 配置字段在实现阶段以 air 当前配置格式为准。

## Affected components

### 当前重点文件

- `backend-go/internal/app/app.go`
- `backend-go/internal/httpserver/server.go`
- `backend-go/internal/httpserver/dashboard.go`
- `backend-go/internal/httpserver/auth.go`
- `backend-go/internal/httpserver/project.go`
- `backend-go/internal/httpserver/user.go`
- `backend-go/internal/httpserver/role.go`
- `backend-go/internal/httpserver/settings.go`
- `backend-go/internal/httpserver/*_routes*.go`
- `backend-go/internal/orbit/model.go`
- `backend-go/internal/orbit/store.go`
- `backend-go/internal/orbit/*_store.go`
- `backend-go/internal/task/repository.go`
- `backend-go/internal/task/worker.go`
- `backend-go/internal/ci/*.go`
- `backend-go/internal/cd/*.go`
- `backend-go/.gitignore` 或仓库根 `.gitignore`
- `backend-go/.air.api.toml`
- `backend-go/.air.worker.toml`（后续已删除，worker loop 并入 `serve`）
- 项目任务入口文件，如 `justfile` / `scripts/dev.py`，若 Plan 阶段确认需要包装 hot reload 命令

### 不应修改的范围

- `backend/migrations`、`backend-go/internal/migrations/**/*.sql`；
- Python `backend`；
- frontend；
- git 状态写操作。

## Interface shape

### Handler 构造

示例形态：

```go
type Handler struct {
    svc Service
    logger *slog.Logger
}

func New(svc Service, logger *slog.Logger) Handler
func (h Handler) Register(r chi.Router)
```

其中 `Service` 是 handler 所需的最小接口，方便测试。

### Service 构造

```go
type Service struct {
    repo Repository
    users UserRepository
    tasks TaskService
    logger *slog.Logger
}

func New(repo Repository, ...) Service
```

service 不接收 `http.ResponseWriter`、`*http.Request` 或 chi router。

### Repository 构造

```go
type Repository struct {
    db *sqlx.DB
    driver string
}

func New(db *sqlx.DB, driver string) Repository
```

repository 可以继续使用现有 `db.NowExpr`、`db.QuoteIdent`、`db.IsNoRows` 等工具。

### Error response

统一响应保持现有风格：

```json
{"detail":"..."}
```

分页响应保持：

```json
{"items":[],"total":0,"page":1,"per_page":10,"pages":0}
```

## Technical questions

暂无必须阻塞计划阶段的问题。

后续 Plan 阶段需要细化：

- checklist 的模块顺序和粒度；
- `orbit` 删除的最终检查项；
- 测试迁移策略；
- 哪些文件只移动，哪些文件需要拆职责；
- air 配置文件字段、临时目录、忽略规则和开发命令包装方式。

## Alternatives considered

### 引入 go-zero / Kratos / GoFrame

不采用。原因：当前问题是职责边界，不是缺框架；引入大框架会造成大量无关迁移和 review 噪音。

### 照搬 Python DDD 四层

不采用。Go 端保持朴素分层即可，避免过重目录和抽象。

### 只改目录名，不拆职责

不采用。验收标准要求核心业务不再主要堆在 `httpserver` 和 `orbit.Store`，必须改变依赖方向和职责边界。

### 只先做 project 纵向切片后暂停

不采用。用户要求 Plan 阶段列出完整 checklist，Implementation 阶段做一项勾选一项，直到全部完成。

### API / worker 共用一个 air 配置

不采用。API 和 worker 运行命令不同，共用配置容易互相覆盖 build 输出，也不便于并行开发。

## Risks

- 改造范围大，Plan 阶段必须把 checklist 拆到可验证粒度。
- 大量移动文件会造成 import 和测试包名变更，容易引入机械错误。
- API 行为必须保持稳定，尤其是 auth、CI/CD、background task。
- `orbit` 被多处依赖，删除路径需要按模块推进，不能残留长期双轨实现。
- `air` 配置如果 watch 产物目录，可能出现循环重启；需要明确 exclude_dir。
- API 和 worker hot reload 若同时运行，需要隔离输出二进制和临时目录。
- 如果实现中发现必须修改 migration SQL，应停止并回到需求/规格阶段确认，因为当前范围明确排除 migration。

## User review notes

- 用户新增范围：引入 `air-verse/air` hot reload，开发阶段对 API 和 worker 都生效。
- 用户接受 `internal/orbit` 的退出策略：可短暂保留以保证迁移过程可编译，但必须有 checklist 删除路径，最终不作为长期兼容层保留。
- 待用户 review。
