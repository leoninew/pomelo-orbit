# CD 部署原理
最后修改时间: 2026-09-04

Doc role: living guide。权威模型见 [CD 领域模型](../product/cd-model.md) 与 [CD 运行时](../architecture/cd-runtime.md)。

## 部署边界

选择 Project 即选择唯一的 Environment 和 Gateway：

```text
Project -> Environment -> Gateway
Application + Version + Service
  -> Deployment snapshot
  -> remote workspace
  -> docker compose over SSH
```

一个 Environment 只能部署一个 Gateway。Environment 与已绑定 Gateway 不能删除；它们与 Project、Gateway Application 的关系均为逻辑外键。

## 支持的宿主

- Linux OpenSSH，Docker Engine 与 Docker Compose 可用。
- Windows native OpenSSH，WSL2 Docker Desktop 可用。

不支持 macOS、其他 Windows Docker 组合、本机 Docker fallback 或任意 SSH command 执行。宿主机应自行完成 registry 配置和登录；Orbit 不管理多 registry 或 registry credential。

## 配置与 Probe

创建 Project 时必须同时提供部署 Environment：SSH host、port、user、平台、工作目录、部署私钥和远端 host-key `SHA256:` fingerprint。私钥由 Environment 管理，API 与 MCP 不会返回它，HTTP 日志会脱敏。

在项目详情的“部署环境”区执行 Probe。Probe 使用私钥认证并严格校验 host key，随后运行固定的 Docker 先决条件检查：

- Linux：验证 `docker`、Docker Engine、`docker compose` 与 Docker daemon。
- Windows：验证 native OpenSSH 可调用 `wsl.exe`、存在 WSL2 `docker-desktop`，并通过宿主机 `docker.exe` 验证 Docker Desktop Engine、Compose、daemon 和 Server OS 为 `linux`。部署与 Traefik REST 同样在 noninteractive PowerShell 中调用 `docker.exe` / `curl.exe`，不进入 Docker Desktop 内部发行版执行 Compose。

Probe 不使用密码认证、PTY、端口转发或用户输入的远端命令。网络、认证和 Docker 失败将记录为脱敏诊断；修改 SSH target、工作目录、host key 或私钥会递增 Environment target revision，旧 Probe 结果不会覆盖新配置。

## 网络与 Gateway

Gateway 创建 Environment-scoped Docker bridge network：`orbit-<environment-code>-traefik`。加入 Traefik 的普通 Service 使用该 external network；缺少环境网络 identity 会阻止渲染，不允许回退到全局 `traefik` 网络。

Gateway 仍通过普通 Service/Deployment 生命周期部署。Route 同步及 Gateway deploy/restart 后的 Traefik REST snapshot 发布在 Environment SSH target 内执行，确保操作只作用于当前 Project 的 Gateway。

TCP entrypoint/host port 是 Gateway Version Component endpoint。需要新的 TCP 端口时，在 Version 中声明、部署该 Version，再创建 TCP Route。
