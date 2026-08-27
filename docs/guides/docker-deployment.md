# Docker 部署指南
最后修改时间: 2026-08-26 23:01:37

Doc role: living guide（运维向）。领域模型见 [CD 模型](../product/cd-model.md)。

Pomelo Orbit 需要 Docker socket、数据库目录、pipeline workspace 与 deployment workspace。Docker-outside-of-Docker 部署示例：

```yaml
services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /srv/pomelo-orbit/db:/app/data/db
      - /srv/pomelo-orbit/ci:/app/data/pipeline
      - /srv/pomelo-orbit/cd:/app/data/deployment
    environment:
      - POMELO_ORBIT_JWT__SECRET_KEY=${POMELO_ORBIT_JWT__SECRET_KEY}
```

默认 `workspace.pipeline=data/pipeline` 与 `workspace.deployment=data/deployment` 分别解析为 `/app/data/pipeline` 和 `/app/data/deployment`。不要只挂 Docker socket，也不要让 workspace 指向未挂载路径。

在 Gateway 创建或编辑时选择 ACME profile。DNS-01 token 在 Gateway 配置中填写，部署 DNS profile 时由 Orbit 作为 `CF_DNS_API_TOKEN` 写入 Traefik Compose environment；无需为 Orbit 配置全局 Cloudflare token。
