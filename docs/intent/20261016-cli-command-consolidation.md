# Intent / 意图：共享业务的 Go CLI 远程环境管理
最后修改时间: 2026-10-16 12:37:36

## Review status / 审查状态

Accepted

用户已接受 Go CLI 从本地直接 SSH 操作远程环境，与 WebAPI 共用配置、数据和业务，沿用现有密钥与身份；同一二进制使用 `cli env`，配置沿用现有 cwd 加载方式。用户“开始实施”接受配套计划及旧专属 HTTP 清理建议，授权本会话顺序实现并执行必要开发检查。实施先移除完整 Python CLI，再建设 Go CLI；标准库单文件 `scripts/release.py` 接续打包。

## Background / 背景

实施前 Python CLI 的 `environment list/ssh/exec` 通过 Orbit HTTP/WebSocket 访问受管环境，`remote ssh/exec/scp/tunnel` 使用本机 OpenSSH 和独立主机配置。两组远程动作重叠，调用路径与执行语义不同。

目标 Go CLI 是 Orbit 的新入站入口：使用 WebAPI 的配置加载、数据库及应用用例，在进程内通过已有业务读取环境和使用受管凭据，由 SSH 基础设施连接目标主机。项目与环境数据、授权规则和业务实现共同复用。

仓库已有 `RunMCP` 本地入口，可作为共用应用服务的参考。环境业务已有受管密钥、主机指纹、目标及凭据修订校验，SSH 基础设施已有终端和 Windows/Linux 命令编码。

依据：[应用入口](../../internal/bootstrap/app.go)、[环境访问用例](../../internal/application/environment/usecase/access.go)、[环境业务](../../internal/application/environment/usecase/service.go)、[SSH 连接](../../internal/infrastructure/runner/ssh/runtime.go)、[现行 CLI 指南](../guides/project-environment-cli.md)。

## Goals / 目标

1. 实施时先移除完整 Python CLI 实现、专属测试、依赖和调用入口，再建设 Go CLI；同一步以单文件、标准库 `scripts/release.py` 接续现有打包链，不保留完整 uv 项目。
2. 与 WebAPI 共用配置加载和数据，通过共享应用用例完成环境管理与远程操作。
3. 保持项目权限、目标身份、凭据绑定、主机指纹和执行规则的一致性；新增业务能力也落在应用层。
4. CLI 仅操作远程 SSH 环境，沿用既有密钥和身份业务，命令围绕 `list`、`ssh`、`exec` 和 `forward start/status/stop` 组织；端口转发采用 `-L` 语义，目的为 SSH 主机自身端口。
5. 形成可检查的业务复用、配置定位、资源生命周期和后续实施安排。

## Non-goals / 非目标

- 不改用户运行配置、密钥或业务数据，不运行实际数据库迁移，不启动或管理用户开发服务器。
- 不增加经 WebAPI 领取/下发 SSH 密钥的流程，不以 HTTP/WebSocket 代理作为 Go CLI 的环境访问路径。
- 不在 CLI 中复制项目权限、凭据解密、指纹校验或执行规则，不直接拼 SQL 绕过应用用例。
- 不因执行 CLI 命令启动 HTTP/Worker；移除 Python CLI 不代表将每个旧工具逐项重写为 Go。首版只做环境查询、交互 SSH、命令执行和端口转发，不迁移独立 `remote`、scp、cert、ulid、version 等工具命令。
- 当前不为打包分发设计配置根参数或路径机制，不增加 `--work-dir`、根目录环境变量或目录查找兜底。
- 不支持 CLI 操作 `local` 环境，不设计本地环境与远程环境的执行位置转换，也不另建密钥或身份体系。
- 端口转发只覆盖本地到 SSH 目标主机自身服务的 TCP 转发，不增加反向转发、动态代理或其他主机的跳板/转发管理。

## User stories / scenarios / 用户故事 / 场景

- 运维者使用同一套 Orbit 配置和数据，从 Go CLI 查询项目、进入环境和执行命令。
- WebAPI 与 CLI 对同一目标使用一致的项目权限、受管凭据、指纹及修订规则。
- 自动化明确指定 Project code，获得可解释的输出与退出码；命令语义与现有业务保持一致。
- 运维者用 `forward start/status/stop` 管理本机到受管 SSH 主机自身服务的后台端口转发，CLI 退出后仍可查看和停止。

## Acceptance criteria / 验收标准

- A01：Go CLI、WebAPI 的调用链体现为各自入站适配 → 共享应用用例 → Repository / 基础设施；共用业务是核心验收，不能只证明使用同一数据库。
- A02：直接共用现有 `internal/config` 的加载与校验契约、数据库和加密配置；按当前工作目录读取配置，相对本地资源路径、profile、覆盖顺序及 `POMELO_ORBIT_` 前缀语义沿用现状。必需配置读取或校验失败时直接报错退出，CLI 不增加目录查找或路径兜底。
- A03：沿用既有身份及认证业务，将 Orbit 操作者身份传入共享用例，复用项目选择和成员权限；SSH 连接复用 Web 终端使用的环境账号、受管密钥、主机指纹及修订规则，不另建 CLI 身份体系，也不绕过成员校验。
- A04：凭据由共享应用业务在进程内解析和使用，SSH 由基础设施执行；不存在新增的密钥领取、导出接口或客户端凭据副本。
- A05：实施第一步移除完整 Python CLI 实现及专属测试、依赖、配置入口、Makefile/Taskfile/CI 调用和活文档中的旧用法，不保留旧命令或兼容入口；保留非 Python 打包资源，并用不依赖 uv/Click 的单文件标准库 `scripts/release.py` 接续当前打包。随后在同一 `pomelo-orbit` 二进制中建设 `cli env list/ssh/exec` 和 `cli env forward start/status/stop`，保留 `serve`、`worker`、`mcp`，操作范围仅限远程 SSH 环境。
- A06：保持需要迁移的真实行为，包括 Windows/Linux 参数编码、workspace、stdin、stdout/stderr、退出码、取消及结果未知时不重放；语义变化单独说明。
- A07：端口转发采用本地 `-L` 语义，本机回环地址监听，经 SSH 连接访问该 SSH 主机的回环地址端口；规划后台启动、实际状态核对、单条停止、断线退出、半关闭及资源清理责任。同一映射重复启动幂等，冲突报错，不自动重连、不接管旧 Python 后台进程；终端/端口转发不套用部署专属的 Docker/Probe 前置条件。
- A08：完成最小有效开发检查和用法说明；方案接受、实施交付和整体验证状态分别记录。

## Workload assessment / 工作量评估

后续实现预计涉及 Python CLI 清理、单文件打包接续、Taskfile/CI 调用处理、Go CLI 入站适配、共享应用用例与 bootstrap 边界及 SSH 能力扩展，工作量中等。复杂度主要在业务复用、跨平台终端和后台转发生命周期；风险集中在清理后调用残留、CLI 重复实现业务、另建配置机制、权限遗漏及需要承接的真实行为丢失。

当前按用户“开始实施”授权，在同一会话顺序完成清理、打包接续、共享业务和 CLI/forward 适配。保持意图与计划两份记录，不派发子任务；实际检查和交付写入计划。

## Candidate tasks and dependencies / 候选子任务与依赖

不适用，本会话顺序实现。

## Open questions / 待定问题

- Q01 已收敛：用户“开始实施”按已展示建议一并删除旧 `/api/cli/...` 路由及专属 handler、proto、生成代码和测试，保留浏览器终端、共享环境业务和 PAT/MCP 认证。

暂无需要用户确认的未决事项。首版命令、端口转发生命周期和失败语义以已接受计划为准。

## Decisions / 决策

- 已确认：以 Go CLI 直接 SSH 为主要方向，与 WebAPI 共用配置、数据和业务用例。
- 已确认：CLI 只覆盖从本地通过 SSH 操作受管远程环境的路径，忽略 `local` 环境；密钥、SSH 账号及 Orbit 操作者身份沿用既有配置、数据与认证业务。
- 已确认：端口转发仅采用本地 `-L` 语义，由 CLI 在本机回环地址监听，经 SSH 访问目标主机自身的回环地址端口；目标主机对其他主机的访问不属于应用新增能力范围。
- `-L` 不要求在远端新增监听端口或对公网开放被转发的服务端口；前提是 SSH 连接可达、服务端允许 TCP 转发，且远端服务可在其本机访问。本地监听端用于访问远程服务，不代表支持 `local` 环境。
- 端口转发使用 `forward start/status/stop`；`start` 后台运行且就绪后返回，`status` 核对实际受管运行状态，`stop` 只停止对应受管条目。具体参数和生命周期以已接受计划为准。
- 配置、数据访问及凭据使用在本地应用进程内复用；本轮不再按独立远程 API 客户端设计。
- 已确认：在现有 `pomelo-orbit` 二进制中增加 `cli env` 命令组，与 `serve`、`worker`、`mcp` 共存；环境管理命令表达为 `pomelo-orbit cli env ...`。
- 已确认：服务与 CLI 直接共用现有配置加载器，基于调用时的当前工作目录读取配置；由调用者在正确目录启动，必需配置读取或校验失败时直接报错退出。
- 当前不新增打包分发设计，撤回新增显式根目录的方案；相对本地资源路径沿用现有解析规则，不为 CLI 增加配置目录发现或兜底。接续仓库已有发布包构建不改变这一约定。
- 已确认：实施第一步移除完整 Python CLI，覆盖旧 environment/remote 以及 cert/ulid/version/release 等包内命令、专属测试、依赖和调用入口；完成清理后再新增 Go CLI，不保留双实现或兼容入口。
- 用户允许的打包例外：可用最小依赖、单文件 `scripts/release.py` 承接现有构建；计划采用 Python 标准库，CI 直接 `python scripts/release.py`，不保留 `pyproject.toml`、`uv.lock` 或 Python CLI 包结构。
- scripts 专属 dotenv 读取与 Makefile development 默认值约定随 Python 工具退出；Go CLI 直接共用后端配置。清理实现与依赖时保留现有配置、密钥和业务数据。
- 首版围绕 `list`、`ssh`、`exec`、`forward start/status/stop` 六个动作组织，不继续泛列环境配置、初始化、Probe 或文件传输等候选范围。

## Risks / assumptions / 风险 / 假设

- 从相同工作目录和环境启动 WebAPI 与 CLI，使用同一配置和数据库；必需配置读取失败即退出是已接受的行为，不作为待修复问题。
- 共享业务需保留操作者、成员权限和目标复核；数据可访问不替代用例授权。
- `local` 环境、反向转发和其他主机的跳板能力已排除在范围外；后台转发的运行记录不是业务数据库状态，也不能代替真实进程核验。
- Python CLI 清理先于 Go CLI 建设，两者之间旧 CLI 功能暂不可用；须同步处理 Taskfile、两条 CI 工作流及活文档引用，并接续现有 ZIP 打包。删除源码不会终止已有后台 SSH 进程，不擅自停止或接管这些进程。

## User review notes / 用户审查记录

- 用户：“以 CLI 直接 SSH + Go CLI 的策略评估”。
- 用户：“用 go cli 实现的话，肯定与现有 go webapi 共用配置、复用数据，不存在‘领取密钥’的说法”。
- 用户：“$specflow:specflow 记录这个任务，我们共用的是数据配置、更是业务”。本轮据此记录目标与评估计划，不开始实现。
- 用户：“采纳，不过 pomelo-orbit environment 改为 pomelo-orbit cli env”。据此接受单二进制并将环境管理命令组确定为 `cli env`；不扩大为产品实现授权。
- 用户后续明确：当前不准备打包，暂时没有 `--work-dir` 需求，沿用现有配置读取方式，读不到直接失败没有问题。此确认取代此前新增显式根目录的方案。
- 用户：“忽略本地环境，cli 只考虑本地以 ssh 操作远程环境 这条路径，沿用密钥和身份”，并说明 SSH 或隧道操作本地环境没有意义。据此移除 local 环境及独立身份设计的待定项；端口转发本轮已规划为后台启停，见计划。
- 用户：“python 相关实现全部移除，而且上来就要做这件事情”。据此将完整 Python CLI 清理确定为实施第一步，覆盖全部旧工具与依赖调用，随后再建设 Go CLI；后续单文件打包例外见下条记录。
- 用户明确采用本地 `-L`，目标主机访问其他主机属于其自身能力，应用不扩展管理；当前只需端口转发，要求评估以 `forward` 配合状态、启动和停止命令表达。此轮记录固定转发范围及命令建议，未将候选命令方案视为已接受。
- 用户要求更新文档、开发前确认清楚；随后补充打包命令复杂时可用 scripts 下最小依赖、单文件 `release.py`，不使用完整 uv 项目。此补充作为此前“全部 Python 移除”的明确打包例外，其他 Python CLI 实现仍首先移除。
- 用户再次明确 Go CLI 当前用于环境管理，要求列出讨论形成的命令和旧 Python HTTP 接口配置；配置与业务共用、SSH 身份复用、不连接 local 环境均按已确认处理，重点完善 `forward` 状态、启动、停止规划。
- 用户：“开始实施”。据此接受完整计划和旧专属 HTTP 清理建议，进入 Implementation，授权本会话顺序实施与必要开发检查。
