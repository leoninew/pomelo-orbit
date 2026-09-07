# 后端架构（现行）
最后修改时间: 2026-09-07 13:31:14

Doc role: living SoT  
权威：与代码冲突时以代码为准。  
代码锚点：`cmd/server`、`internal/bootstrap`、`internal/api/http`、`internal/application`、`internal/repository`、`internal/infrastructure`。

## 技术栈

| 层 | 选型 |
|----|------|
| 语言 | Go（见 `go.mod`） |
| HTTP | Gin |
| API 契约 | Protobuf（`proto/orbit`）+ 生成代码 `internal/gen/proto`；HTTP 映射在 handler |
| 配置 | Viper + 环境变量；启动校验（`internal/config`） |
| DB | SQLite / MySQL / PostgreSQL；**golang-migrate** embed（`sql/migration`） |
| SQL | **sqlc**（部分域）+ **sqlx**（CD 等仍用 sqlx 实现，以 `internal/repository/impl` 为准） |
| 队列 | DB 后台任务 + 同进程 worker（单节点；API 与 worker 可同启） |
| 模板 | Liquid（CI/部分渲染路径） |
| 前端静态 | `web/dist` 等由部署配置决定 |

**已废止文档中的栈**：Python / FastAPI / SQLAlchemy / Alembic — 见归档 `docs/archive/designs/architecture.md`。

## 分层与依赖方向

```text
cmd/server, cmd/migrate
  → bootstrap（组装 DB、HTTP、worker、迁移）
  → api/http（handler / routes / middleware）  # 入站适配
  → application/*（usecase、dto、port）       # 应用服务
  → repository（接口）+ model
  → repository/impl/{sqlx,sqlc} + infrastructure  # 出站适配
  → gen/{proto,sqlc}  # 生成物，勿手改业务逻辑于此
```

约定：

- **API 路由单数**，例如 `/api/ci/repository`（`CLAUDE.md`）。  
- 不在业务层散落读环境变量；配置集中 `internal/config`。  
- 生成代码隔离在 `internal/gen`。

## 模块边界（application）

| 包 | 职责 |
|----|------|
| `application/cd` | 应用、版本、环境、网关、服务、部署、平台路由、compose 渲染 |
| `application/ci` | 仓库、模板、快照、运行、产物、凭证、变量 |
| `application/auth` / `user` / `role` / `project` / `settings` | 平台能力 |

## 数据与迁移

- Schema 与 seed 均为普通编号迁移，统一位于 `sql/migration/{sqlite,mysql,postgres}/`，由 `MigrateUp` 按版本顺序执行和跟踪；不设独立 data migration 加载流程。
- **不修改已执行的迁移文件**（项目约束）。开发期重建库可接受时，以当前迁移链终态为准。  
- 终态 CD 表见 `*_cd_schema*` 类迁移（application/version/component/expose/environment/gateway_config/service/deployment/route 等）；**无** 旧表 `application_config_file`、`application_service`、`application_route`、`environment_binding` 作为现行 schema。

## 运行形态

- `App.Serve`：普通迁移（含 seed）→ HTTP → **同进程** background worker（见 `internal/bootstrap/app.go`）。
- `App.RunWorker`：普通迁移（含 seed）→ 可单独跑 worker。
- `App.RunMCP`：本地 stdio MCP；工具发现无认证副作用，每次 `tools/call` 校验 `POMELO_ORBIT_MCP__ACCESS_TOKEN` 中的 MCP PAT，并固定 session actor。它直连 application usecase，不暴露 HTTP `/mcp`，也不使用浏览器 grant、loopback callback、本地 token 文件或 Web JWT。
- Deployment Dialogue：HTTP 入站认证后的 actor 通过每 turn 的内存 MCP session 调用与 stdio 相同的 Core；不转发浏览器 Authorization。
- 单节点挂载 Docker socket 执行 CI/CD 容器操作；不做 API→远程 worker 协议主路径。

## 相关

- [CD 运行时](./cd-runtime.md)  
- [产品 CD 模型](../product/cd-model.md)  
- [决策账本](../decisions/ledger.md)  
