# CD 部署原理
最后修改时间: 2026-09-09 20:22:00

Doc role: living guide。权威模型见 [CD 领域模型](../product/cd-model.md) 与 [CD 运行时](../architecture/cd-runtime.md)。

## 部署边界

选择 Project 即选择唯一的 Environment 和 Gateway：

```text
Project -> Environment -> Gateway
Application + Version + Service
  -> Deployment snapshot
  -> target runtime
     local -> Environment.workspace_root + Docker daemon
     ssh   -> Environment.workspace_root + docker compose over SSH
```

一个 Environment 只能部署一个 Gateway。Environment 与已绑定 Gateway 不能删除；它们与 Project、Gateway Application 的关系均为逻辑外键。

## Environment target

Environment 的 target type 是显式 `local | ssh`：

- `local` 直接在 Orbit 控制面宿主机的 Docker daemon 执行，工作目录为 Environment 保存的 `workspace_root`。页面显示该目录和 Gateway 入口，不显示 SSH 表单、主机指纹或初始化入口。
- `ssh` 支持 Linux OpenSSH + Docker Engine/Compose，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。它在目标端按 Environment 保存的 `workspace_root` materialize workspace，并使用受管私钥和 pinned host key 执行。

`ssh` 到 `127.0.0.1` 仍是 SSH，不会被解释为 local。没有 hostname heuristic 或 local/SSH fallback。两类目标的 Compose 生命周期、运行时查询、证书同步、Gateway network 与 Traefik REST publish 均通过同一 target runtime 执行。

不支持 macOS、其他 Windows Docker 组合或任意 SSH command 执行。宿主机应自行完成 registry 配置和登录；Orbit 不管理多 registry 或 registry credential。

## 配置与 Probe

创建 Project 时只填写名称和编码。Orbit 在同一事务中创建 active local Environment，Web 随即切换到新 Project 并打开“环境”页；用户可在那里将 target type 切换为 ssh 并填写 host、port、user、平台和工作目录。local 不创建部署私钥；切换到 SSH 后 Orbit 才生成并保存部署私钥。active Linux SSH target 显示“初始化部署主机”入口：输入能登录目标的 SSH 用户与一次性密码或私钥后，Orbit 在该连接内完成初始化。一次性认证不写入数据库、响应、MCP 或日志。

Docker Engine/Compose（Windows 上包括 Docker Desktop、WSL2 Linux engine）是 SSH target 的外部前置条件。Linux 自动初始化会先检查配置 SSH user 的 Docker daemon/Compose、bootstrap user 的 passwordless `sudo` 及 OpenSSH 公钥认证能力；不满足时直接失败并给出诊断。它只按需写入 Orbit 的部署公钥和 `.pomelo-orbit` 工作目录，不安装、启停或配置 Docker/Compose/Docker Desktop/WSL，不修改 Docker 用户组、sshd、firewall 或 Docker 网络。

Windows SSH target 保留已配置受管私钥后的 Probe 与运行时能力，但不提供自动初始化入口：普通 SSH 不能可靠完成 Windows UAC 提权。Windows Docker Desktop/WSL/OpenSSH 与受管公钥必须由目标主机的管理员预先配置。

“环境”是独立页面，始终对应顶部当前 Project；项目详情可跳转到该页面。local Probe 只检查控制面 Docker 与 Docker Compose。SSH 首次 Probe 使用生成的私钥认证并记录当次 host key `SHA256:` fingerprint，后续 Probe 和部署严格校验该指纹。SSH Probe 随后运行固定的 Docker 先决条件检查：

- Linux：验证 `docker`、Docker Engine、`docker compose` 与 Docker daemon。
- Windows：验证 native OpenSSH 可调用 `wsl.exe`、存在 WSL2 `docker-desktop`，并通过宿主机 `docker.exe` 验证 Docker Desktop Engine、Compose、daemon 和 Server OS 为 `linux`。部署与 Traefik REST 同样在 noninteractive PowerShell 中调用 `docker.exe` / `curl.exe`，不进入 Docker Desktop 内部发行版执行 Compose。

SSH Probe 不使用密码认证、PTY、端口转发或用户输入的远端命令。Linux 自动初始化仅在一次性 bootstrap session 中使用用户提供的密码或私钥，随后仍由受管私钥 Probe；网络、认证和 Docker 失败将记录为脱敏诊断。切换 target type 或修改 SSH target/工作目录会递增 Environment target revision。SSH target 变更会清除已记录指纹，旧 Probe 结果不会覆盖新配置。

## 网络与 Gateway

Gateway 创建或复用部署宿主上的 Docker bridge network `traefik`。加入 Traefik 的普通 Service 以 external network 方式接入同一共享网络；Gateway 的 `providers.docker.network` 也固定为 `traefik`。网络名不由 Environment code 派生。

Gateway 仍通过普通 Service/Deployment 生命周期部署。Route 同步及 Gateway deploy/restart 后的 Traefik REST snapshot 发布在当前 Environment target 内执行，确保操作只作用于当前 Project 的 Gateway。

TCP entrypoint/host port 是 Gateway Version Component endpoint。需要新的 TCP 端口时，在 Version 中声明、部署该 Version，再创建 TCP Route。
