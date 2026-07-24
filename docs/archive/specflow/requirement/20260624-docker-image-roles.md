# Docker 镜像角色启动方式改进需求
最后修改时间: 2026-06-24 22:14:10

Review status: Accepted

## Background

当前同一个 Docker 镜像需要同时服务于 HTTP API 和后台 worker 两类进程。现有 Dockerfile / Dockerfile.cn 使用 `CMD ["backend-go", "serve"]` 作为默认命令，虽然可以被覆盖，但 Compose 中需要写完整命令；同时 Dockerfile 内置 HTTP healthcheck 会被 worker 容器继承，导致 worker 作为非 HTTP 进程运行时 healthcheck 不匹配。

## Goal

- 改进 `Dockerfile` 和 `Dockerfile.cn`，让镜像默认启动 HTTP API，同时能用更简洁的 command 启动 worker。
- 避免镜像层内置 HTTP 专用 healthcheck 影响 worker 角色。
- 在文档中展示同一镜像启动 API 与 worker 的 Docker / Compose 使用方式。

## Non-goal

- 不新增单独 worker 镜像。
- 不改 backend-go 命令行参数或 worker 执行逻辑。
- 不引入 docker-compose.yml 部署文件。
- 不调整数据库、任务表或 CI/CD 业务逻辑。

## User scenarios

- 用户直接运行镜像时，默认启动 HTTP API。
- 用户需要启动 worker 时，可以覆盖 command 为 `worker`。
- 用户使用 Compose 部署时，可以用同一个 image 分别定义 `api` 和 `worker` 服务。

## Acceptance

- `Dockerfile` 和 `Dockerfile.cn` 使用 `ENTRYPOINT ["backend-go"]` 与 `CMD ["serve"]`。
- Dockerfile 内不保留 HTTP API 专用 healthcheck，避免 worker 继承错误健康检查。
- README 展示 `docker run` 默认启动 API、覆盖 command 启动 worker 的方式。
- README 展示 Compose 中 `api` 和 `worker` 使用同一镜像、不同 command 的示例，并将 HTTP healthcheck 放到 `api` 服务上。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 使用单镜像多角色启动模式，不拆分镜像。
- API healthcheck 由具体部署服务声明，不放在通用镜像层。

## Risk

- 移除 Dockerfile 内置 healthcheck 后，直接 `docker run` 不再自带容器健康状态；使用 Compose / 生产编排时需要在 API 服务声明 healthcheck。
- `EXPOSE 80` 仍保留在镜像中；worker 角色不会监听该端口，但这是镜像元数据，不影响运行。
