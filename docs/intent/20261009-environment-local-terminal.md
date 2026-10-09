# Environment 终端入口与本机交互终端
最后修改时间: 2026-10-09 16:29:46

Review status: Accepted

Mode: standard

## Background

当前 Environment 页面只在 `target_type=ssh` 时显示终端按钮并挂载终端抽屉。后端只为就绪的 SSH Environment 签发票据，票据兑换、会话复核和连接日志依赖 SSH 字段；组合入口只注入 SSH PTY runner。前后端测试也将 local 隐藏入口、拒绝连接作为既有行为。

`target_type=local` 表示在 Orbit 控制面所连接的 Docker daemon 上执行，与操作者是否通过 localhost、127.0.0.1、内网 IP 或公网域名访问站点无关。Orbit 部署在远程服务器上并使用 local Environment 时，当前设计仍隐藏终端，遗漏了操作者与该远程 Orbit 运行环境交互的场景。

用户进一步明确：所有已配置的 Environment 都应提供终端入口，无需根据站点访问地址隐藏。local 终端进入 Orbit 进程所在环境；原生运行时进入当前主机，容器运行时进入 Orbit 容器。DooD 场景下，容器内已有的 Docker socket、Docker CLI 和工作区挂载可以继续使用。ssh 终端继续进入配置的目标宿主机。

终端连接与 Docker 检查是独立能力。打开终端不触发 Docker 检查，不以 Docker 或 Compose 可用、Environment Probe 已执行或成功作为前提；该规则同时适用于 local 和 ssh。SSH 的目标配置、凭据与 host key 校验属于连接自身条件，不应与 Docker Probe 结果绑定。

本任务独立记录上述问题和新的产品意图。原 [SSH 终端意图](20260930-environment-ssh-terminal.md) 中排除 local 与 Orbit 容器终端的范围不适用于本任务；实现时须同步校准活文档。

现状依据：

- [Environment 页面](../../web/src/views/environment/EnvironmentPage.vue)与[入口测试](../../web/src/views/environment/EnvironmentPage.test.ts)。
- [终端用例](../../internal/application/environment/usecase/terminal.go)、[目标解析](../../internal/application/environment/usecase/target.go)与[终端端口](../../internal/application/environment/port/terminal.go)。
- [HTTP/WebSocket handler](../../internal/api/http/handler/environment/terminal.go)、[组合入口](../../internal/bootstrap/http.go)与[SSH PTY runner](../../internal/infrastructure/runner/ssh/terminal.go)。
- [本机展示信息](../../internal/bootstrap/environment_display.go)、[Docker 部署指南](../guides/docker-deployment.md)与[原生发布构建](../../scripts/src/pomelo_orbit_cli/commands/release.py)。

## Goal

1. 所有已配置的 Project Environment 均显示终端入口，local 与 ssh 使用一致的打开方式；localhost、127.0.0.1、内网 IP 和公网域名不参与入口显示判断。
2. local 终端以 Orbit 进程的运行用户启动交互式 shell，初始目录为该 Environment 保存的工作区，使用现有路径规则展开 `~`；原生部署运行在主机中，容器部署运行在 Orbit 容器中。
3. 支持当前 Linux 原生、Linux 容器与 Windows 原生运行场景；Unix 使用 PTY，Windows 使用 ConPTY，保留真实终端的输入输出、回显、控制键、尺寸变化和退出状态。
4. ssh 终端继续使用 Environment 保存的目标、SSH 用户、受管密钥和固定 host key，通过现有 SSH PTY 连接目标宿主机。
5. 两类终端复用现有 xterm、WebSocket 协议、一次性票据、Project 成员授权和会话限制。按环境类型分派执行器；local 的票据、会话校验、标题和日志不依赖 SSH 字段。
6. 延续既有生命周期：关闭抽屉只隐藏，保留终端输出和会话；重新打开继续同一会话。离开环境页、切换 Project、目标变更、主动断开或达到服务端限制时释放会话；断线后仅手动重连。
7. 沿用现有 API 路由与数据模型，使用成熟的本地终端库适配既有窄接口，并保持当前无 CGO 的发布方式。
8. 两类终端的票据签发、兑换与持续会话校验均与 Docker Probe 解耦；只检查成员权限、目标身份及修订、工作区和对应 shell/SSH 连接条件。Docker 检查失败不阻止终端连接，也不终止已建立的会话。

## Non-goal

- 不新增 localhost 或站点域名判断，也不因环境尚未就绪而隐藏已配置 Environment 的终端入口。
- 不为 local 增加 SSH host、用户、密钥或凭据绑定；不通过 SSH 连接自身或要求原生本机安装 OpenSSH。
- 不使 local 容器终端自动进入 Docker daemon 宿主机、WSL2 发行版或其他应用容器，不引入提权或命名空间切换。
- 不提供由浏览器选择其他主机、运行用户、shell 或任意启动参数的能力，不引入每个 Orbit 用户独立的操作系统账号。
- 不改变 CI、CD 与 Probe 的固定命令执行行为，不把终端能力扩展到 MCP。
- 不在终端连接过程中自动检查 Docker 或 Compose，不复用 Docker Probe 的成功状态作为终端门槛。
- 不重建已有 SSH 终端、认证或密钥管理流程；不为本任务改造普通环境保存、Wizard 或完整配置恢复的仓储更新机制。
- 不增加文件传输、完整会话回放、跨页面或跨刷新恢复；不将 Project 工作区描述为 shell 的文件系统沙盒。
- 按 SpecFlow 阶段推进；文档接受不隐含远程部署、Git 提交或开发服务器管理授权。

## User scenarios

1. 用户访问远程域名上的 Orbit，在 local Environment 页面打开终端；Orbit 为容器部署时，进入该容器的 shell，查看工作区并使用已有 DooD 能力。
2. 用户通过 localhost 或 127.0.0.1 访问原生 Windows Orbit，在 local Environment 页面看到相同入口，并打开以 Orbit 用户运行的 Windows 交互终端。
3. 用户在原生 Linux Orbit 上打开 local 终端，进入该 Environment 工作区；输入、输出和窗口尺寸与实际 PTY 同步。
4. 用户打开 Linux 或 Windows SSH Environment 的终端，继续以受管 SSH 用户进入目标宿主机，连接不会改为 local。
5. 用户关闭抽屉后继续操作环境页，重新打开仍看到原输出并继续原会话；切换 Project 或修改目标后，旧会话被释放。
6. 用户点击终端时若权限不足、SSH 连接配置或认证失败、本地 shell 启动失败，得到明确的连接失败状态，入口仍保持可见。
7. Docker 不可用或 Environment Probe 尚未执行、检查失败时，用户仍可打开 local/ssh shell 进行操作；在 shell 中执行 Docker 命令的结果独立反馈，不改变终端连接状态。

## Acceptance

- [ ] 已配置的 local 和 ssh Environment 均显示终端入口；通过 localhost、127.0.0.1 和远程站点访问时不按地址隐藏。
- [ ] Linux 原生 local 终端运行在 Orbit 主机，Linux 容器 local 终端运行在 Orbit 容器，Windows 原生 local 终端运行在 Windows 主机；均沿用 Orbit 进程身份，从 Environment 工作区启动。
- [ ] local 不需要 SSH 配置或凭据，票据签发、兑换、会话复核、标题与日志不会读取不存在的 SSH 字段。
- [ ] 本地终端支持双向输入输出、控制键、窗口尺寸变化、手动断开、重连与实际退出码；Windows 使用 ConPTY，不以普通命令管道替代交互终端。
- [ ] 容器终端能访问该容器已有的工作区和 Docker socket，使用当前运行身份执行 Docker CLI；验证不额外授予权限或修改挂载。
- [ ] local/ssh 均不要求 Docker 或 Compose 可用，也不要求 Environment Probe 已执行或成功；票据签发、兑换和会话复核不包含该门槛，打开终端不触发 Docker 检查。
- [ ] Docker 检查失败不阻止终端连接或终止现有会话；Docker 不可用时，shell 仍可交互，Docker 命令只反馈自身执行结果。
- [ ] SSH 仍复用目标用户、受管私钥和固定 host key，执行连接配置、认证及 host key 校验；凭据准备和首次 host key 信任建立不以 Docker Probe 成功为前提。不向浏览器暴露密钥，不根据站点地址或连接失败切换执行器。
- [ ] 两类终端均执行 Project 成员校验、一次性票据和现有会话配额；目标或成员资格变化时终止旧会话，SSH 额外复核凭据修订。
- [ ] 抽屉关闭与重开保留同一会话、输出和终端实例；离开页面、切换 Project、主动断开或达到限制后，释放 shell、受终端会话管理的子进程和 PTY/ConPTY 资源。
- [ ] 记录操作者、Project、Environment、目标类型、时长与结束原因；终端输入输出、票据与认证秘密不进入常规日志。
- [ ] 不修改数据库模型或已执行迁移，不新增终端 API 路由；Windows/Linux/macOS 原生包及 Linux amd64/arm64 镜像的相关构建保持兼容，终端库版本经依赖与构建验证后固定。
- [ ] 原 SSH-only 入口与拒绝 local 的测试按新行为收敛，补充本地交互和会话清理的有效验证，并同步更新相关产品、运行时和操作活文档。

## Open questions

暂无需要用户确认的未决事项。用户明确终端连接与 Docker 检查独立，local 与 ssh 均解除 Docker Probe 门槛；SSH 连接自身的认证与 host key 校验继续保留。

本地 shell、PTY 库版本、清理策略及必要的 Docker 解耦范围已在 [Plan](../plan/20261009-environment-local-terminal.md) 中明确；SSH 复用已有实现，首次指纹记录仅在实际终端连接内解除 Docker 成功条件，不另建握手流程。本意图不新增可配置 shell 入口。当前仅核对了代码和库资料，尚未完成 PTY 实机运行验证。

## Decisions

- 使用标准模式 / standard，按 Intent -> Plan -> Implementation -> Verification 分阶段推进；用户要求开始计划，意图已接受，当前进入 Plan / 计划。
- 终端入口始终对已配置 Environment 可见；是否可连接由服务端授权和对应连接条件决定，不以隐藏入口代替反馈。
- 终端与 Docker 检查完全分开：不在打开终端时检查 Docker，不读取 Docker Probe 成功状态作为连接或维持会话的条件。CI/CD 与 Probe 自身的既有语义不变。
- local 终端的运行位置与 Orbit 进程一致，明确接受 DooD 场景下的 Orbit 容器内 shell；ssh 终端仍运行在配置的 SSH 目标宿主机。
- 复用既有 TerminalRunner/TerminalSession、票据和 WebSocket 协议，增加 local 执行器与环境类型分派，不新建一套终端接口或凭据体系。
- 按用户指定，本地终端统一使用 `github.com/aymanbagabas/go-pty`，适配同一现有终端接口并覆盖 Unix PTY/Windows ConPTY；已有 SSH 终端继续复用现有 runner。库版本、必要依赖升级、无 CGO 构建与资源释放在实施后验证。
- 本任务独立于此前 SSH 终端任务，不改写其过程文档；活文档的新行为在实现阶段同步收敛。
- SSH 是已有能力。本次仅调整其终端连接条件中的 Docker 耦合，不新增独立验证器、预连接或全量配置更新方案。

## Risk

- local shell 继承 Orbit 进程的文件访问、环境和 Docker 权限；Project 工作区只是起始目录，不能承诺 shell 被隔离在该目录内。授权仍沿用当前 Project 成员边界，需保留操作者会话记录。
- Windows ConPTY 与 Unix PTY 的退出、管道排空和句柄关闭行为不同；需验证断开、未消费输出和子进程场景，避免阻塞与残留资源。
- 容器 shell 的可用命令、Docker 权限和工作区访问取决于实际镜像、运行用户及挂载；本任务使用既有能力，不以提权弥补配置问题。
- 当前 TargetResolver 和 HasFreshSuccessfulProbe 包含 Docker Probe 门槛，首次 SSH host key 也只在综合 Probe 成功后记录。计划在既有终端/SSH 链路内处理这些耦合，票据和持续会话复核保持一致；不重设计 SSH 能力或改造全部配置保存接口，不影响 CI/CD 的就绪要求。
- go-pty 的依赖图会要求部分 Go 依赖升级；Windows 进程句柄释放也需按所锁定版本核对。需要验证无 CGO、原生跨平台与 Linux 镜像构建及实际资源回收，不以文档宣称替代编译和运行验证。

## User review notes

- 用户指出：远程站点的服务器可以是 local Environment 的执行主机，按 `target_type=local` 隐藏终端遗漏了该场景；最初要求先了解细节。
- 用户明确：“不隐藏终端入口，orbit 运行在容器里是 DooD 场景、展示容器里终端并无不妥，按这个方向评估。”据此撤销最初按 localhost/127.0.0.1 隐藏入口的想法，并接受容器内 local shell。
- 已评估：新增本地 PTY/ConPTY 执行器并按类型分派，调整票据和日志中的 SSH-only 假设，复用现有前端和协议，无需数据库迁移或新增 API；Docker Probe 与 local shell 连接门槛作为独立问题记录。
- 用户：“采纳意见，$specflow:specflow 标准 记录问题”。记录问题阶段新建意图草稿，按标准模式推进。
- 用户明确：“打开终端就是打开终端，检查 docker 是检查 docker，两者不混在一起”。据此将 local/ssh 终端与 Docker Probe 解耦记为已确认决策，移除该未决事项，保留连接自身所需的授权、身份和 SSH 校验。
- 用户：“开始计划，所以事项需要在计划中理清”。据此将意图标记为 Accepted，进入标准模式 / standard 的 Plan / 计划，要求在计划中明确实施事项。
- 用户指出“ssh 什么的是已有实现，你还加了哪些戏”，随后要求“修正计划”。据此明确复用既有 SSH 能力，撤回额外握手、全量配置条件更新、扩展快照和专门的指纹刷新设计，保留本机终端与 Docker 解耦所需改动。
- 用户明确“终端使用 github.com/aymanbagabas/go-pty”。据此将本地终端库选择统一为 go-pty，并同步计划；当前仍处于 Plan 阶段。
