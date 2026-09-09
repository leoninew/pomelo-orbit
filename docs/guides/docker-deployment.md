# Docker 部署指南
最后修改时间: 2026-09-09 20:22:00

Doc role: living guide（运维向）。领域模型见 [CD 模型](../product/cd-model.md)。

Pomelo Orbit 需要 Docker socket、数据库目录、pipeline workspace 与部署日志目录。local Environment 的 `workspace_root` 还必须选择一个同时对 Orbit 容器和 Docker daemon 可见的挂载路径。Docker-outside-of-Docker 部署示例：

```yaml
services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /srv/pomelo-orbit/db:/app/data/db
      - /srv/pomelo-orbit/ci:/app/data/pipeline
      - /srv/pomelo-orbit/deployment-logs:/app/data/deployment-logs
    environment:
      - POMELO_ORBIT_JWT__SECRET_KEY=${POMELO_ORBIT_JWT__SECRET_KEY}
```

默认 `workspace.pipeline=data/pipeline` 与 `logging.deployment_root=data/deployment-logs` 分别解析为 `/app/data/pipeline` 和 `/app/data/deployment-logs`。Environment 的 `workspace_root` 没有 YAML 默认值；配置 local Environment 时应填写容器内已挂载且 Docker daemon 可见的路径。不要只挂 Docker socket，也不要让进程配置或 Environment 指向未挂载路径。

在 Gateway 创建或编辑时选择 ACME profile。DNS-01 token 在 Gateway 配置中填写，部署 DNS profile 时由 Orbit 作为 `CF_DNS_API_TOKEN` 写入 Traefik Compose environment；无需为 Orbit 配置全局 Cloudflare token。
