# Pomelo Orbit 工作目录与挂载
最后修改时间: 2026-09-04

Doc role: living guide。与代码冲突时以代码为准。

## 目录边界

| 配置 / 字段 | 内容 |
| --- | --- |
| `workspace.pipeline` | Orbit 控制面上的 Repository checkout、Run artifacts 与 Stage logs；CI Docker runner 继续使用该目录 |
| `workspace.deployment` | Orbit 控制面上的 Deployment execution logs，仅为 `<root>/logs/<service-code>/<deployment-id>.log` |
| Environment `workspace_root` | 目标宿主机上的 CD 根目录；Service Compose、组件 logical mount、Gateway 证书和 Route REST snapshot 都在此目录下 |

`workspace.*` 相对值相对 `orbit.root` 解析。Environment `workspace_root` 必须是目标平台的绝对路径：Linux 使用 POSIX absolute path，Windows 使用 drive absolute path。

远端 Service 目录为 `<environment.workspace_root>/<service-code>/`，包含 `docker-compose.yml`、组件受控文件/目录，以及 Gateway 的 `gateway/certs/` 和 `.orbit/traefik-rest.json`。Orbit 不在本机生成可供 Docker 执行的 CD workspace，也不回退到控制面 Docker daemon。

## 挂载语义

除 `named_volume` 外，Version mount source 必须是绝对路径或显式 `./` 相对路径。部署渲染将 `./` source 解析到远端 Service 目录，并通过 SFTP materialize 受控文件或目录；absolute source 保持目标宿主机路径语义。`ignore_if_exists` 只影响受控文件首次写入。

Orbit 运行在容器内时，`workspace.pipeline` 仍需满足 CI Docker-outside-of-Docker 的 host path 映射。`workspace.deployment` 只需作为控制面日志持久卷，不参与远端 Compose mount 解析。

## Traefik 证书

Gateway Version 声明 cert/acme directory mount。Route 同步通过对应 Project Environment 的 SFTP 将 PEM 与私钥写入 `<environment.workspace_root>/<gateway-service-code>/gateway/certs/`，Traefik REST snapshot 引用容器内 `/etc/traefik/certs/`。

Gateway deployment 不创建 mount；它只使用 Version 已声明的拓扑，并对既有 static config 和 DNS profile environment 应用当前 Gateway 配置。