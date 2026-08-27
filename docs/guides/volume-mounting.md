# Pomelo Orbit 工作目录与挂载
最后修改时间: 2026-08-26 23:01:37

Doc role: living guide。与代码冲突时以代码为准。

## 目录边界

| 配置 | 内容 |
| --- | --- |
| `workspace.pipeline` | Repository checkout、Run artifacts 与 Stage logs |
| `workspace.deployment` | Service Compose、部署日志、组件 logical mount、Gateway 证书与 ACME 数据 |

相对值相对 `orbit.root` 解析。Deployment 以 Service code 为根：`<workspace.deployment>/<service-code>/`，其中包含 `docker-compose.yml`、`deployments/*.log` 与组件相对挂载源。

## Docker 路径

Orbit 原生运行时，workspace 绝对路径用于文件读写；除 `named_volume` 外，Version 挂载源必须是绝对路径或显式 `./` 相对路径。在 Docker-outside-of-Docker 部署中，pipeline 与 deployment workspace 必须从 Docker daemon 主机 bind mount 进 Orbit 容器；启动时 Orbit 以 Docker inspect 验证映射。

## Traefik 证书

Gateway Version 声明 cert/acme directory mount。Route 同步将 PEM 与私钥写入 `<workspace.deployment>/<gateway-runtime-service-code>/gateway/certs/`，REST snapshot 引用容器内 `/etc/traefik/certs/`。

Gateway deployment 不创建 mount；它只对已声明 static config 设置 email，并在 DNS profile 的 Compose environment 写入 `CF_DNS_API_TOKEN`。
