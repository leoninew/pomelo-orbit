# CD 部署原理
最后修改时间: 2026-09-23 16:21:10

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

Service 是 Application 下的独立运行绑定。一个 Application 可以按需要创建多条 Service；`service.code` 在当前 Project 内唯一，并作为部署工作目录、Compose project、日志目录和派生路由名的稳定标识。创建页面可建议 `<application-code>-default`，但该 code 可编辑且 `default` 不表示实例或默认 Service。部署、停止和运行时查询均以显式 `service_id` 定位目标。

## Environment target

Environment 的 target type 是显式 `local | ssh`：

- `local` 直接在 Orbit 控制面宿主机的 Docker daemon 执行，工作目录为 Environment 保存的 `workspace_root`；页面原样展示该值，`~` / `~/...` 只在 Probe 和部署时展开为控制面用户主目录。不显示 SSH 表单、主机指纹或初始化入口。
- `ssh` 支持 Linux OpenSSH + Docker Engine/Compose，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。它在目标端按 Environment 保存的 `workspace_root` materialize workspace，`~` / `~/...` 展开为远端登录用户主目录，并使用 `environment_credential` 中的私钥和 pinned host key 执行。

`ssh` 到 `127.0.0.1` 仍是 SSH，不会被解释为 local。没有 hostname heuristic 或 local/SSH fallback。两类目标的 Compose 生命周期、运行时查询、证书同步、Gateway network 与 Traefik REST publish 均通过同一 target runtime 执行。

不支持 macOS、其他 Windows Docker 组合或任意 SSH command 执行。宿主机应自行完成 registry 配置和登录；Orbit 不管理多 registry 或 registry credential。

## 配置与 Probe

创建 Project 时只填写名称和编码。Environment 与 Gateway 只能由 Web Project Initialization Wizard 写入。local 不创建 `environment_credential`。SSH target 可以先保存为未初始化状态；用户在完整表单后点击“生成初始化命令”时，Orbit 在同一写事务中保存 target，并创建或复用该 Environment 的受管密钥。重复生成命令始终返回同一有效公钥，保存、编辑、状态读取和 Probe 都不会隐式创建或轮换密钥。

Linux 和 Windows SSH target 都在 Wizard 与已保存的 Environment detail 中提供可复制的初始化命令。用户必须通过云控制台、已有 SSH 登录或其他带外方式在目标端执行命令，随后回到页面运行 Probe。Linux Bash 命令由所配置的 SSH 用户执行：幂等创建该用户的 `~/.ssh`、修正 `700`/`600` 权限并追加 Orbit 公钥；同时按所选 `workspace_root` 创建 `pipeline` 与 `deployment` 目录，将新建的工作区根目录及这两个子目录 `chown` 为该用户。非 root 用户执行目录准备时使用 `sudo`，不递归更改现有目录下的文件，不修改 `sshd`、Docker、Docker Compose、Docker 用户组或防火墙。Windows PowerShell 命令仍须在目标 Windows 主机的管理员 PowerShell 执行，用于准备 Windows OpenSSH、受管公钥、工作目录和 Docker Desktop 前置条件。

Docker Engine/Compose（Windows 上包括 Docker Desktop、WSL2 Linux engine）是 SSH target 的外部前置条件。Orbit 不接收、读取或持久化操作者个人私钥、密码或一次性 bootstrap 认证。未生成初始化命令时，SSH Probe 会写入明确的恢复诊断且不会尝试裸网络连接或生成密钥。

“环境”是独立页面，始终对应顶部当前 Project；项目详情可跳转到该页面。local Probe 只检查控制面 Docker 与 Docker Compose。SSH 首次 Probe 使用受管私钥认证并记录当次 host key `SHA256:` fingerprint，后续 Probe 和部署严格校验该指纹。SSH Probe 随后运行固定的 Docker 先决条件检查：

- Linux：验证 `docker`、Docker Engine、`docker compose` 与 Docker daemon。
- Windows：验证 native OpenSSH 可调用 `wsl.exe`、存在 WSL2 `docker-desktop`，并通过宿主机 `docker.exe` 验证 Docker Desktop Engine、Compose、daemon 和 Server OS 为 `linux`。部署与 Traefik REST 同样在 noninteractive PowerShell 中调用 `docker.exe` / `curl.exe`，不进入 Docker Desktop 内部发行版执行 Compose。

SSH Probe 不使用密码认证、PTY、端口转发或用户输入的远端命令；网络、认证和 Docker 失败将记录为脱敏诊断。切换 target type 或修改 SSH target/工作目录会递增 Environment target revision。SSH target 变更会清除已记录指纹，旧 Probe 结果不会覆盖新配置。

## 网络与 Gateway

Gateway 创建或复用部署宿主上的 Docker bridge network `traefik`。加入 Traefik 的普通 Service 以 external network 方式接入同一共享网络；Gateway 的 `providers.docker.network` 也固定为 `traefik`。网络名不由 Environment code 派生。

Gateway 仍通过普通 Service/Deployment 生命周期部署。Route 同步及 Gateway deploy/restart 后的 Traefik REST snapshot 发布在当前 Environment target 内执行，确保操作只作用于当前 Project 的 Gateway。

TCP entrypoint/host port 是 Gateway Version Component endpoint。需要新的 TCP 端口时，在 Version 中声明、部署该 Version，再创建 TCP Route。
