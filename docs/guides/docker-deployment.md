# Docker 部署指南
最后修改时间: 2026-09-11 14:48:17

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

默认 `workspace.pipeline=data/pipeline` 与 `logging.deployment_root=data/deployment-logs` 分别解析为 `/app/data/pipeline` 和 `/app/data/deployment-logs`。Wizard 的 local 默认工作目录是 `~/.pomelo-orbit`，配置和界面都原样展示；使用时展开为容器内进程用户主目录。若 CD 根需要落在已挂载路径上，应在 Wizard 里填写该容器内绝对路径，并保证 Docker daemon 可见。不要只挂 Docker socket。

在 Gateway 创建或编辑时选择 ACME profile。DNS-01 token 在 Gateway 配置中填写，部署 DNS profile 时由 Orbit 作为 `CF_DNS_API_TOKEN` 写入 Traefik Compose environment；无需为 Orbit 配置全局 Cloudflare token。
