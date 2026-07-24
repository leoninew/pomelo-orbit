---
Review status: Accepted
Flow mode: light
Stage: Requirement
---

# backend-go CD Route 接口迁移

## Background

`backend-go` 是 Python `backend` 的 Go 版本实现和改良，目前 `/api/cd/route` 相关接口在 `docs/todo.md` 中仍标记为待迁移。前端 `frontend/src/api/cd/route.ts` 已依赖这些端点。

## Goal

- 对照 `backend/src/pomelo_orbit/interfaces/api/cd/route.py`，在 `backend-go` 补齐 `/api/cd/route` 相关端点。
- 覆盖 CRUD、启用/停用、同步路由配置、证书上传、禁用 HTTPS、启用 Let's Encrypt、启用 mkcert。
- 更新 `docs/todo.md` 中 Route 相关迁移状态。

## Non-goal

- 不迁移 `/api/cd/traefik-route` 或 `/api/cd/deployment/{id}` 等其他 CD 端点。
- 不启动或停止开发服务器。
- 不执行 git 写操作。

## Acceptance

- `GET /api/cd/route` 支持按项目分页列出，并支持搜索。
- `POST /api/cd/route` 能创建项目路由。
- `GET/PUT/DELETE /api/cd/route/{route_id}` 能按权限获取、更新和删除路由，其中启用中的路由不可删除。
- `POST /api/cd/route/{route_id}/enable` 与 `/disable` 能更新状态并同步路由配置文件。
- `POST /api/cd/route/sync` 能按项目同步路由和证书文件。
- `POST /api/cd/route/{route_id}/cert`、`DELETE /api/cd/route/{route_id}/https`、`POST /api/cd/route/{route_id}/letsencrypt`、`POST /api/cd/route/{route_id}/mkcert` 能维护 HTTPS 状态与证书类型。
- Go 代码格式化并通过相关 `backend-go` 测试。
- `docs/todo.md` Route 小节反映已迁移状态。

## Risk / Assumptions

- backend-go 当前没有 Python 版 `TraefikManager` 的完整等价实现，需要在本次补齐路由配置与证书文件同步的最小完整能力。
- 本次不实际启动 Traefik 或 Docker；验证以 API/数据库/文件生成逻辑和测试为主。
- mkcert 能力依赖本机 `mkcert` 命令和 CA 安装状态；接口会在不可用时返回错误。

## User review notes

用户已明确要求使用 light 模式直接实现，不要打折扣；因此本需求记录直接标记为 Accepted 并进入实现。
