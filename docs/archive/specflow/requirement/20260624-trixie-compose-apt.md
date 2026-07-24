# Debian Trixie apt 安装 Docker Compose 改造需求
最后修改时间: 2026-06-24 18:34:07

## Review status

Accepted

## Background

服务器执行 `docker build -t pomelo-orbit:20260624-100115 -f Dockerfile.cn .` 时，最终镜像阶段通过 `curl` 从 `ghproxy.net` 下载 Docker Compose 二进制文件失败：

```text
curl: (92) HTTP/2 stream 1 was not closed cleanly: INTERNAL_ERROR
```

失败点位于 `Dockerfile.cn` 中直接下载 Docker Compose release binary 的步骤。该步骤依赖 GitHub 代理链路，服务器网络环境下稳定性不足，导致构建失败。

前期确认：Debian bookworm 默认源不能直接通过 `docker-compose-plugin` 达到同等目的；`docker-compose-plugin` 属于 Docker 官方 apt 源。Debian trixie 的官方 `docker-compose` 包已经是 Compose v2，可通过 apt 安装并使用 `docker compose` 命令。

## Goal

- 将 `Dockerfile.cn` 和 `Dockerfile` 的最终运行阶段基础镜像更新为 `debian:trixie-slim`。
- 通过 apt 安装 Debian trixie 官方源中的 `docker-compose` 包，移除直接从 GitHub 或 ghproxy 下载 Docker Compose 二进制文件的步骤。
- 保留构建期断言，确保最终镜像内 `docker compose` 命令可执行。
- 本地分别对 `Dockerfile.cn` 和 `Dockerfile` 执行 `docker build`，确认构建可用并展示结果。

## Non-goal

- 不引入 Docker 官方 apt 源。
- 不固定安装 Docker Compose v5.1.0 release binary。
- 不调整前端构建、Go 后端构建、运行时目录、环境变量、健康检查和启动命令。
- 不修改部署脚本中的 `docker compose` 调用方式。
- 不执行 git commit、push、merge 等 git 写操作。

## User scenarios

- 作为维护者，我希望服务器使用 `Dockerfile.cn` 构建镜像时不再依赖不稳定的 ghproxy Compose 下载链路。
- 作为维护者，我希望标准 `Dockerfile` 和国内构建用 `Dockerfile.cn` 在运行时基础依赖策略上保持一致。
- 作为部署者，我希望镜像内仍然可执行 `docker compose`，满足现有 CI/CD/CD 功能依赖。

## Acceptance

- `Dockerfile.cn` 最终阶段使用 `debian:trixie-slim`。
- `Dockerfile` 最终阶段使用 `debian:trixie-slim`。
- 两个 Dockerfile 都通过 apt 安装 `docker-compose`。
- 两个 Dockerfile 都删除直接下载 Docker Compose release binary 的 `curl ... github.com/docker/compose ...` 步骤。
- 两个 Dockerfile 都在构建期执行 `docker compose version` 作为可用性断言。
- 本地 `docker build -f Dockerfile.cn ... .` 成功。
- 本地 `docker build -f Dockerfile ... .` 成功。

## Open questions

暂无需要用户确认的未决事项。用户已明确要求轻量模式、直接实现并完成本地 Docker build 验证。

## Decisions

- 使用 Debian trixie 官方 `docker-compose` 包，而不是 Docker 官方 apt 源的 `docker-compose-plugin`。
- 接受 Compose 版本由 Debian trixie apt 仓库决定，不再固定为 release binary `v5.1.0`。
- 同时改造 `Dockerfile.cn` 和 `Dockerfile`，避免两个镜像定义在最终阶段依赖策略上分叉。

## Risk

- `debian:trixie-slim` 会把最终阶段基础系统从 Debian 12 升级到 Debian 13，`docker.io`、`curl`、系统库等包版本会随之变化。
- Debian trixie apt 源中的 Compose v2 版本与原先下载的 `v5.1.0` 不一致；现有项目使用的 `docker compose --env-file .env up -d`、`ps`、`logs` 等基础能力预计兼容，但仍以本地 build 和后续运行时验证为准。
- `Dockerfile.cn` 仍依赖国内 Debian 镜像源可用性；本次变更只移除 ghproxy/GitHub release binary 下载依赖。
