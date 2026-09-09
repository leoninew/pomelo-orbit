# Pomelo Orbit 工作目录与挂载
最后修改时间: 2026-09-09 11:52:41

Doc role: living guide。与代码冲突时以代码为准。

## 目录边界

| 配置 / 字段 | 内容 |
| --- | --- |
| `workspace.pipeline` | Orbit 控制面上的 Repository checkout、Run artifacts 与 Stage logs；CI Docker runner 继续使用该目录 |
| `workspace.deployment` | local Environment 的 CD 根目录，也是控制面 Deployment execution logs 根目录（`<root>/logs/<service-code>/<deployment-id>.log`） |
| SSH Environment `workspace_root` | SSH 目标宿主机上的 CD 根目录；Service Compose、组件 logical mount、Gateway 证书和 Route REST snapshot 都在此目录下 |

`workspace.*` 相对值相对 `orbit.root` 解析。local 没有可编辑的 Environment workspace；它直接使用控制面 `workspace.deployment`。SSH Environment `workspace_root` 可以是目标平台的绝对路径，也可以是 `~/<path>`；后者在远端执行和 SFTP 写入时解析为登录 SSH 用户的主目录，Linux 和 Windows 均使用 `~/.pomelo-orbit` 作为默认值。

Service 目录为 `<target-workspace>/<service-code>/`，包含 `docker-compose.yml`、组件受控文件/目录，以及 Gateway 的 `gateway/certs/` 和 `.orbit/traefik-rest.json`。local 的 target workspace 是 `workspace.deployment`，由控制面 Docker daemon 执行；SSH 的 target workspace 是 Environment `workspace_root`，由 SSH/SFTP 执行。两类目标不互相 fallback。

## 挂载语义

除 `named_volume` 外，Version mount source 必须是绝对路径或显式 `./` 相对路径。部署渲染将 `./` source 解析到 target Service 目录，并由 local filesystem 或 SFTP materialize 受控文件或目录；absolute source 保持目标宿主机路径语义。`ignore_if_exists` 只影响受控文件首次写入。

Orbit 运行在容器内时，`workspace.pipeline` 仍需满足 CI Docker-outside-of-Docker 的 host path 映射。local Environment 的 `workspace.deployment` 必须同时是控制面可写与 Docker daemon 可见的 bind mount，Orbit 通过 Docker daemon path resolver 渲染本机 Compose mount source；SSH Environment 不使用该 resolver。

## Traefik 证书

Gateway Version 声明 cert/acme directory mount。Route 同步通过对应 Project Environment target runtime 将 PEM 与私钥写入 `<target-workspace>/<gateway-service-code>/gateway/certs/`，Traefik REST snapshot 引用容器内 `/etc/traefik/certs/`。

Gateway deployment 不创建 mount；它只使用 Version 已声明的拓扑，并对既有 static config 和 DNS profile environment 应用当前 Gateway 配置。
