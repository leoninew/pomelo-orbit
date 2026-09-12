# 部署环境目标需求
最后修改时间: 2026-09-12 09:50:42

Review status: Accepted

Mode: strict

## Background

Project 继续承担资源归属、成员授权、HTTP/Proto/MCP scope 和 Web active-project 状态。每个 Project 恰好有一个 Environment，Environment 恰好绑定一个 Gateway；切换 Project 就切换部署目标和默认 Gateway。

此前本机 CD 直接使用控制面 Docker daemon 与 `workspace.deployment`。SSH 改造后错误地将 Environment 收窄为 SSH 连接，并删除了本机执行能力。实际产品模型中，本机和 SSH 远程机都是明确的部署目标：`ssh` 到 `127.0.0.1` 仍是 SSH 远程机，不是本机的别名。

## Goal

1. Environment 明确支持 `local` 与 `ssh` 两种 target type；二者复用 Project 1:1 Environment 1:1 Gateway、部署任务、Route 与权限模型。
2. `local` 在控制面宿主机直接执行 Docker Compose，工作目录取控制面 `workspace.deployment`；不保存、不展示也不要求 SSH host、port、user、credential、host key 或 SSH 初始化入口。
3. `ssh` 维持现有 Linux OpenSSH 与 Windows native OpenSSH + WSL2 Docker Desktop 能力。对于已能以用户密码或私钥连接的 Linux target，Orbit 直接使用一次性认证检查 Docker daemon/Compose、无交互 `sudo` 与 OpenSSH 公钥能力，按需写入部署公钥和工作目录，再用受管私钥 Probe；一次性认证不得持久化或出现在 API、MCP、日志。Windows SSH target 可由 Web Wizard 或 Environment detail 生成本地主机 PowerShell helper，用户通过 SSH 写入受管部署公钥、创建工作目录并检查 WSL2/Docker Desktop/Linux containers/Compose，随后仍须回到页面执行 SSH 测试和 Probe。Orbit 不安装、启停或配置 Docker、Docker Compose、Docker Desktop、WSL 或 Docker 用户组；这些外部前置条件不满足时直接失败并给出诊断。
4. Deploy、restart、stop、运行时查询、容器日志、Gateway 网络创建和 Traefik REST publish 都通过同一个显式目标运行时执行；不能出现只让 Compose 本机化而 Route/Gateway 仍走 SSH 的半套实现。
5. Deployment 保存 Environment、target type、target revision、Gateway snapshot；仅 SSH deployment 保存 SSH credential identity/revision。目标或 revision 改变后，排队任务必须失败。
6. Project 创建请求只包含名称和编码；后端在同一事务中固定创建 active `local` Environment，不创建部署 SSH credential。用户随后在 Environment 页面切换为 `ssh` 并完成配置。

## Non-goal

- 不删除 Project、Project member、Project scope、active-project selector 或既有 Project resource isolation。
- 不允许一个 Project 多个 Environment，也不允许一个 Environment 多个 Gateway。
- 不把 `127.0.0.1`、`localhost` 或任意 SSH 配置解释为 local；不在 local 与 ssh 间做隐式 fallback。
- 不为 local 填充虚构 SSH 字段，不为 ssh 回退到控制面 Docker，也不保留旧 SSH-only API 字段的业务兼容分支。
- 不支持 macOS SSH target、Cygwin/MSYS/Git Bash/WSL SSH server、Windows Containers 或 registry credential 管理。
- 不提供 Environment delete。
- 不为 Linux SSH 初始化保留可复制的宿主机命令，也不将 bootstrap 密码、私钥或私钥口令保存为 Credential。

## User scenarios

1. 本机 Environment：成员选择 `local`，打开环境页面能看到本机目标、控制面工作目录和 Gateway 入口；Probe 检查控制面 Docker / Docker Compose，然后部署、查询和 Route publish 都在控制面执行。
2. Linux SSH Environment：成员选择 `ssh` 并填写平台、host、port、user、workspace root；若能以自己的密码或私钥连接目标，成员在页面输入一次性认证，Orbit 先确认现成 Docker/Compose、无交互 `sudo` 和 OpenSSH 公钥能力，再写入受管部署公钥与工作目录，随后 Probe 并 pin host key。所有 CD 操作仍在该目标端执行。
3. 回环 SSH：成员配置 `ssh` / `127.0.0.1`，系统按 SSH 连接、认证和 host-key contract 执行，不转为 local。
4. 创建 Project：同一事务创建 Project 与 active local Environment，不创建部署私钥；Web 将新 Project 设为当前 Project 并转到 Environment 页面。空库 seed 的 default Project 获得 local Environment 与其绑定 Gateway。
5. 目标变更：将 local 改为 ssh 或反向修改会递增 target revision、清除旧 Probe；已排队的 deployment 因快照不一致失败，不能落到另一台机器。

## Acceptance

- [ ] Environment 公开 target type；local 与 ssh 的输入、响应和 UI 只出现各自适用字段。
- [ ] local CD 在控制面 Docker/workspace 上完整运行：Compose 生命周期、runtime/log query、Gateway network、Traefik REST Route publish 均可用。
- [ ] ssh CD 继续通过 SSH/SFTP 运行；SSH 到 loopback 不会走 local runner。
- [ ] local 不生成或持有部署 SSH credential；ssh 的私钥不进入 API、MCP 或日志。
- [ ] local Probe 只检查控制面 Docker prerequisites；SSH Probe 继续 pin host key。
- [ ] Linux SSH 自动初始化只接受一次性密码或私钥认证，并在请求完成后丢弃；它不安装或管理 Docker、Compose、Docker Desktop、WSL 或 Docker 用户组，缺失 Docker prerequisites 或 passwordless sudo 时先失败并提示。
- [ ] 自动初始化仅按需写入部署公钥和工作目录；不改写 sshd、重启服务、修改防火墙或 Docker 网络规则。
- [ ] Deployment snapshot 显式校验 target type 与 revision；SSH credential snapshot 只在 ssh 时存在。
- [ ] default seed Project 的 Environment 是 local，Gateway 绑定保留，无占位 SSH credential。
- [ ] `POST /api/project` 只接受名称和编码；每个新 Project 都有 active local Environment，随后可在 Environment 页面配置 SSH。
- [ ] `GET /api/project/:id/environment` 的 local target 返回控制面平台、主机名、当前用户和工作目录；编辑 local Environment 切到 ssh 时，仅当所选 SSH 平台与控制面平台一致才预填主机和用户。
- [ ] `task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- `local` 是一等 Environment target type，不是 SSH 的快捷路径或降级路径。
- local workspace 由控制面 `workspace.deployment` 配置决定；SSH workspace 仍是 Environment SSH target 的字段。
- local response 的预填信息直接来自控制面进程：`runtime.GOOS`、`os.Hostname()` 和 `os/user.Current()`；它们不属于 Environment 持久化配置，也不用于改变 SSH 的运行时语义。
- 本次采用无兼容基线：`000039` 从空库直接建立最终 local/ssh schema，已有开发库由运维就地同步版本记录；应用运行时不基于空字段猜测类型，也不保留兼容执行分支。
- SQL seed 随新模型收敛为 default local Environment，不再插入 placeholder credential。
- Project 创建不承载 Environment target 配置，Environment 页面是切换和配置 SSH target 的唯一入口。
- Docker Engine/Compose 与 Windows Docker Desktop/WSL2 是 SSH target 的外部前置条件；Linux 自动初始化负责检查，不负责包办安装。Windows 因普通 SSH 无法可靠完成 UAC 提权，不提供自动特权初始化，但提供由用户执行的 PowerShell helper；helper 不能替代网络、账户权限或后续页面 Probe。

## Risk

- 本机 Docker daemon、控制面容器挂载和 `workspace.deployment` 必须可用；Probe 将在实际部署前阻止不满足前置条件的环境。
- Environment type 切换会使已有 deployment snapshot 失效，这是防止任务落到错误目标的必要行为。
- Linux/Windows SSH 真实目标仍需实机验证。
- Linux 自动初始化还要求 bootstrap SSH user 已可连接并具备 passwordless sudo；这不是 Orbit 可以安全代办的主机权限提升。

## User review notes

- 用户确认 Environment 可以是 local 或 ssh；ssh 到 `127.0.0.1` 仍然是 ssh。
- 用户要求立即恢复 local，便于与历史本机实现对照；不要等当前改造提交后再从历史提交回看。
- 用户拒绝业务逻辑兼容、隐式 fallback、`reserved` 字段和旧新逻辑并存。
- 用户要求：已有用户密码或私钥能连接时，Orbit 必须直接完成 Linux 部署 key 与工作目录初始化，而不是让用户复制命令。
