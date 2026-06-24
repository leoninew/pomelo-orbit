# backend-go Docker 镜像迁移需求
最后修改时间: 2026-06-24 12:19:47

Review status: Accepted

## Background

当前根目录 `Dockerfile` 和 `Dockerfile.cn` 仍构建并启动 Python backend，发布 workflow 也使用该 Dockerfile 构建镜像。前端已依赖 `backend-go` 提供的接口，例如 `GET /api/auth/turnstile-config`，因此容器环境继续运行 Python backend 会导致登录页初始化 404。

## Goal

- 将根目录 `Dockerfile` 和 `Dockerfile.cn` 的后端构建与启动目标迁移到 `backend-go`。
- 保留前端构建产物进入单镜像的部署形态。
- 保留容器内 Docker CLI 和 Docker Compose plugin，满足现有 CI/CD/CD 功能依赖。
- 让容器内 `backend-go` 监听 `0.0.0.0:80`，并保留 `/api/health` healthcheck。
- 补齐 `backend-go` 单镜像场景下的静态前端文件服务与 SPA fallback。
- 更新 GitHub docker publish workflow，使发布前检查改为 Go backend 相关检查。

## Non-goal

- 不迁移旧 Python backend 的数据目录或历史运行时数据。
- 不拆分为前端 Nginx 容器 + API 容器的多容器部署。
- 不删除 Python backend 源码。
- 不改动业务 API 行为，除新增静态文件服务外不调整现有路由语义。

## User scenarios

- 用户构建根目录 Docker 镜像后，容器启动的是 `backend-go serve`，不是 `python -m pomelo_orbit.main`。
- 用户访问容器根路径时可以加载前端 SPA。
- 登录页初始化请求 `/api/auth/turnstile-config` 命中 `backend-go` 并返回 200。
- GitHub Actions 发布镜像时使用 backend-go 测试作为后端检查。

## Acceptance

- `Dockerfile` 使用 Go build stage 构建 `backend-go/cmd/backend-go`，final image 不再依赖 Python backend runtime。
- `Dockerfile.cn` 与 `Dockerfile` 保持同等 backend-go 迁移逻辑，并保留国内镜像加速意图。
- final image 复制 `backend-go` binary、`backend-go/config.defaults.yaml` 和前端 `dist` 到运行目录。
- final image 设置 `POMELO_ORBIT_SERVER__HOST=0.0.0.0` 与 `POMELO_ORBIT_SERVER__PORT=80`。
- final image 创建运行所需目录，但不复制或迁移旧数据目录。
- `backend-go` 对非 `/api/` 路径提供静态文件或 `index.html` fallback；对 `/api/` 未命中路径仍返回 JSON 404。
- `.github/workflows/docker-publish.yml` 不再运行 Python backend tests，改为运行 backend-go tests。

## Open questions

暂无需要用户确认的未决事项。用户已明确不需要迁移数据目录，并采纳单镜像补齐 backend-go 静态文件服务的建议。

## Decisions

- 使用轻量模式 / light。
- 继续维护单镜像部署形态：前端构建产物由 backend-go 进程服务。
- 运行时数据目录只创建，不从旧 backend 复制。

## Risk

- Docker build 使用的 Go 版本需要与 `backend-go/go.mod` 的 `go 1.25.0` 匹配；基础镜像 tag 需要可用。
- backend-go 静态文件服务需要避免影响 `/api/` JSON 404 行为。
- 发布 workflow 从 Python 测试切到 Go 测试后，Python backend regressions 不再阻塞镜像发布，这是符合迁移方向但会改变 CI 覆盖边界。
