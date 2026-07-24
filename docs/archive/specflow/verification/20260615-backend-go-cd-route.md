---
Review status: Accepted
Flow mode: light
Stage: Verification
---

# backend-go CD Route 接口迁移验证

## What changed

- 在 `backend-go/internal/httpserver/route.go` 新增 `/api/cd/route` 端点实现：列表、创建、详情、更新、删除、启用、停用、同步、证书上传、禁用 HTTPS、启用 Let's Encrypt、启用 mkcert。
- 在 `backend-go/internal/httpserver/dashboard.go` 注册 `/api/cd/route` 相关路由。
- 在 `backend-go/internal/orbit/api_store.go` 增加 `route` 表的分页查询、全量查询、详情、创建、更新、删除仓储方法。
- 在 backend-go 配置中补齐 Traefik 动态路由目录、证书目录、容器名配置，并支持环境变量覆盖。
- 新增 `backend-go/internal/httpserver/route_routes_test.go` 覆盖 CRUD、启停、证书、同步、鉴权与 Traefik 配置生成。
- 更新 `docs/todo.md` 中 Route 小节，将 12 个 `/api/cd/route` 端点标记为 backend-go 已迁移。

## Acceptance

- [x] `GET /api/cd/route` 支持按项目分页列出，并支持搜索。
- [x] `POST /api/cd/route` 能创建项目路由。
- [x] `GET/PUT/DELETE /api/cd/route/{route_id}` 能按权限获取、更新和删除路由，启用中的路由不可删除。
- [x] `POST /api/cd/route/{route_id}/enable` 与 `/disable` 能更新状态并同步/撤销路由配置文件。
- [x] `POST /api/cd/route/sync` 能按项目同步路由与证书文件。
- [x] `POST /api/cd/route/{route_id}/cert` 能解析合并 PEM、保存证书并写入证书文件。
- [x] `DELETE /api/cd/route/{route_id}/https` 能关闭 HTTPS、清理证书并重写路由配置。
- [x] `POST /api/cd/route/{route_id}/letsencrypt` 能校验配置与域名并启用 Let's Encrypt。
- [x] `POST /api/cd/route/{route_id}/mkcert` 能调用本机 mkcert 生成证书；不可用时返回明确错误。
- [x] `docs/todo.md` Route 小节已更新为已迁移。

## Commands

```text
go -C "backend-go" test ./internal/httpserver ./internal/orbit ./internal/config
# ok


go -C "backend-go" test ./...
# ok
```

首次直接在仓库根目录执行 `go test ./internal/httpserver ./internal/orbit ./internal/config` 失败，原因是 Go module 位于 `backend-go/`；随后使用 `go -C "backend-go" ...` 通过。

## Remaining risk

- 本次未启动 Traefik 或 Docker，路由配置和证书同步通过文件生成逻辑与 HTTP 测试验证。
- mkcert 依赖本机 `mkcert` 命令和 CA 安装状态，测试未实际调用外部 mkcert。
- 工作区存在本任务之外的 CD deployment 相关未提交改动，应在提交前和本次 route 改动一起审视或拆分。
