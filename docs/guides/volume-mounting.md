# Pomelo Orbit 工作目录与挂载
最后修改时间: 2026-08-18

Doc role: living guide。与代码冲突时以代码为准。

## 目录边界

Orbit 只管理两个可配置的运行工作根：

| 配置 | 内容 |
| --- | --- |
| `workspace.pipeline` | Repository checkout、Run artifacts 与 Stage logs |
| `workspace.deployment` | Service Compose、部署日志、组件 logical mount、Gateway 证书与 ACME 数据 |

相对值相对 `orbit.root` 解析，绝对值可位于项目外。SQLite 数据库、进程日志、环境文件、用户本地 Repository 和导出文件由各自配置或调用方管理，均不属于 workspace。

Pipeline 内部路径保持为 `<workspace.pipeline>/<repository-code>/workspace` 及 `<workspace.pipeline>/runs/<run-id>/...`。Deployment 以 Service code 为根：`<workspace.deployment>/<service-code>/`，其中包含 `docker-compose.yml`、`deployments/*.log` 与组件相对挂载源。

## Docker 路径

Orbit 原生运行时，workspace 的绝对路径用于 Orbit 文件读写；除 `named_volume` 外，Version 中的挂载源必须是绝对路径或显式写成 `./data`、`./config/app.env` 这类 Compose 相对路径，并原样渲染为 bind mount。裸的 `data` 会被 Compose 解释为卷名，不属于目录、文件或 controlled file 的合法 source；`named_volume` 保持裸卷名。

在 Docker-outside-of-Docker 部署中，配置值是 Orbit 容器内路径，两个 workspace root 必须各自从 Docker daemon 主机 bind mount 进容器。启动时 Orbit 用 Docker inspect 验证映射；Compose 和 Pipeline Stage 只接收解析后的主机路径，而逻辑目录和 controlled file 仍在 Orbit 可见目录物化。

```yaml
services:
  pomelo-orbit:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /srv/pomelo-orbit/ci:/app/data/pipeline
      - /srv/pomelo-orbit/cd:/app/data/deployment
```

这与默认 `configs/config.yaml` 的 `data/pipeline`、`data/deployment` 一致，无需额外的环境变量。只有刻意修改 workspace 配置为其他容器内路径时，才同时调整其 bind mount。

## Traefik 证书

手工上传或 mkcert 生成的 Route 证书遵循一条固定链路：

1. Route 的 PEM 和私钥保存到数据库。
2. 同步 Route 时，Orbit 将它们写入 `<workspace.deployment>/traefik/data/certs/<route-name>.pem` 与 `<route-name>-key.pem`。
3. 同一 REST snapshot 在顶层 `tls.certificates` 中引用 Traefik 容器内的 `/etc/traefik/certs/<route-name>.pem` 与对应私钥。
4. 初始 Gateway Version 把该目录作为 Docker daemon 可见 host path 同时挂载到 `/etc/traefik/certs` 和 `/letsencrypt`。后者保存 ACME 的 `acme.json`。

Route 同步使用 `providers.rest` 的全量 PUT，不使用 `dynamic/routes.yml`、`tls.yml` 或 file provider watch。手工复制 PEM 或设置 `POMELO_TRAEFIK_DATA_DIR` 不会将证书登记到 REST snapshot。

Gateway 的 Version 是可编辑的普通资源。若用户变更或移除上述挂载，Route 证书同步不会重写 Version；恢复此约定后重新部署 Gateway，再同步 Route。

## 故障排查

- 容器启动时报 `workspace.pipeline` 或 `workspace.deployment must be bind mounted`：将对应宿主目录显式 bind mount 到 Orbit 容器，使它覆盖当前配置的 workspace 路径。
- DooD 部署后 Compose 找不到挂载源：检查 Docker daemon 可见的主机路径，而不是 Orbit 进程看到的容器内路径。原生运行时检查 Service 目录下与 `docker-compose.yml` 同级的相对源。
- 手工证书未生效：确认 Gateway Version 挂载 `/etc/traefik/certs`，确认 Route 已启用并同步，再检查 `<workspace.deployment>/traefik/data/certs/` 中的 PEM 文件和 Traefik REST API 日志。
