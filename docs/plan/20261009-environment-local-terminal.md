# Environment 终端入口与本机交互终端实施计划
最后修改时间: 2026-10-09 18:08:00

Review status: Accepted

Mode: standard

## Intent basis

依据 [已接受的意图](../intent/20261009-environment-local-terminal.md)。所有已配置的 Environment 均显示终端入口。`local` 指 Orbit 进程所在的执行环境，与浏览器访问 localhost、127.0.0.1、内网 IP 或公网域名无关；容器部署接受进入 Orbit 容器。终端连接和 Docker 检查独立，规则同时覆盖 local 和 ssh。

当前阶段为标准模式 / standard 的验证 / Verification。用户已反馈“测试无问题”，要求暂存相关变更并完成验证文档；实施范围及开发检查保留如下，验收结果见 [Verification](../verification/20261009-environment-local-terminal.md)。

本次范围收敛为终端入口开放、local 终端执行器和终端与 Docker 检查解耦。SSH PTY、受管密钥、host-key 校验、WebSocket、票据、授权与抽屉生命周期均是已有实现，直接复用。SSH 侧只调整实际阻碍解耦的连接条件与首次指纹记录，不另建 SSH 验证流程，不改造环境保存、Wizard 或配置恢复。

## Current findings

- `EnvironmentPage.vue` 的终端按钮和抽屉均以 SSH 为条件，抽屉标题直接读取 SSH 用户和主机。
- `TerminalService` 使用部署的 `TargetResolver`；签发只接受 SSH，兑换和持续复核都要求最新成功 Probe，并直接读取 SSH binding。HTTP handler 的连接日志同样直接读取 SSH 用户。
- `sshrunner.Runtime` 已实现现有 `TerminalRunner/TerminalSession`。`localrunner.Runtime` 没有交互终端能力；`targetrunner.Runtime` 已按持久化的 target type 分派其他执行能力，可以复用该分派方式。
- `bootstrap/http.go` 单独注入 SSH terminal runner。`newDeploymentRuntime` 同时由 HTTP 和 worker 组合入口使用，调整返回类型时必须验证两处编译。
- SSH runner 的终端连接严格要求已有 host key；目前首次指纹只在综合 Probe 成功后持久化。仅移除终端的 Probe 状态判断，仍无法让 Docker 检查失败的首次 SSH 目标打开终端。
- 首次指纹记录属于既有 SSH 连接能力，本任务只把该记录从 Docker 成功条件中分离。环境整行保存的并发覆盖问题作为既有风险记录，不扩展成全量仓储条件更新任务。

## Design decisions

### 1. 入口、执行位置与身份

| 类型与部署 | shell 执行位置 | 操作系统身份 | 初始目录与 shell |
| --- | --- | --- | --- |
| local，Linux 原生 | Orbit 所在主机 | Orbit 进程用户 | 保存的工作区，`/bin/sh -i` |
| local，Linux 容器 / DooD | Orbit 容器 | Orbit 容器进程用户 | 容器可见的工作区，`/bin/sh -i` |
| local，Windows 原生 | Orbit 所在 Windows 主机 | Orbit 进程用户 | 保存的工作区，系统自带 Windows PowerShell，`-NoLogo -NoProfile` |
| ssh，Linux/Windows | Environment 配置的 SSH 目标宿主机 | 保存的 SSH 用户 | 保持现有 OpenSSH 默认 shell 和登录起始目录 |

local 按 Orbit 实际操作系统选择实现，不使用 Docker daemon 的操作系统或 SSH platform 字段判断。Windows 从系统目录定位内置 PowerShell，通过 ConPTY 启动，不要求安装 `pwsh`，不增加 shell 搜索或回退链。Unix 使用固定 `/bin/sh`，不从浏览器或 `$SHELL` 接收启动命令。shell 继承 Orbit 的进程环境，Unix 设置终端类型为 `xterm-256color`。

local 工作区使用现有 `workspacepath.ExpandLocalHomePath` 展开 `~`，按实际运行平台确认绝对路径。不存在时以 Orbit 当前权限创建根目录，沿用本地目录创建的 `0o750` 约定；无法创建或进入时返回连接失败。不准备 pipeline/deployment 子目录、Compose 文件或 Docker 挂载，不调用 `PhysicalPathResolver`、Docker inspect 或部署工作区准备逻辑。

SSH 仍要求已显式准备且有效的受管凭据。终端不生成或轮换密钥，不执行初始化命令，不修改 SSH 起始目录。工作区只作为 local 起始目录，不能承诺文件系统隔离。macOS 维持既有原生包的编译兼容，不扩展 Environment 平台枚举或新增 macOS 部署支持承诺。

### 2. 终端目标解析独立于 Docker

在环境应用层为 `TerminalService` 增加专用目标解析逻辑，复用 `environmentForProject`、配置校验及凭据解密 helper；移除对部署 `TargetResolver` 的依赖。部署、CI、Route 等继续使用现有 resolver 及 `HasFreshSuccessfulProbe` 规则，不给该接口增加“忽略 Probe”的参数。

终端解析检查 Project 成员、环境存在、目标配置、工作区，以及 SSH 分支的凭据归属、binding/revision 和私钥可解密。local 不读取 SSH 字段。签发、兑换和会话复核都不检查 `last_probe_*`，也不调用 Prober 或 Docker/Compose 命令。

沿用现有票据中的 Environment ID、target revision 和 SSH credential ID/revision，仅增加 target type 区分 local/ssh。local 不检查 SSH binding，SSH 保留现有 binding 校验；不扩展为全字段目标快照，不为配置恢复新增终端规则。

票据签发和兑换均检查成员；兑换原子消费一次性票据，重新加载当前目标，按快照比对后分配现有会话配额。持续复核保持周期检查及输入前检查；local 只复核成员和目标，SSH 额外读取凭据记录核对有效 binding/revision。Probe 状态或诊断变化不使票据、会话或前端实例失效。

### 3. 已有 SSH 链路中的必要解耦

SSH 的连接、认证、PTY、默认 shell、resize、退出与密钥管理沿用现有实现。票据签发和兑换只读取、校验已有配置，不执行 SSH 预连接。

当前首次 host key 只在综合 Probe 成功后保存，不能留成“先检查 Docker 才能打开首次 SSH 终端”的隐含要求。处理限定在实际终端连接内：已有指纹继续严格比较；没有指纹时复用现有首次观察 callback，在同一次 SSH 连接中用受管密钥认证并取得观察指纹。认证失败不记录指纹，不新增独立 verifier 或第二次 SSH 连接。

将当次连接观察到的指纹交给环境用例，在发送 ready、接受浏览器输入前复核成员、目标修订和凭据并保存；固定失败或目标变化时关闭本次会话。仅为传回观察指纹对连接结果做必要适配，保留现有 TerminalSession 输入输出、票据与 WebSocket 协议。

指纹持久化采用只写指纹和 `updated_at` 的窄条件操作，匹配 Environment ID、target type/revision、SSH credential ID/revision；当前指纹只能为空或与本次观察相同，不覆盖不同指纹，也不修改 Probe 结果。Probe 的现有首次指纹写入复用该操作，Docker 检查命令与结果记录逻辑继续沿用现状。

本次 SQL/仓储变更仅服务于上述指纹记录，现有 `UpdateEnvironment` 接口及普通保存、Wizard、SSH 初始化、配置恢复调用点不纳入改造。新查询使用命名 SQLC 参数并通过 `task sqlc` 生成。终端连接仍在请求事务外，保留现有超时、配额和复核策略。

### 4. PTY 实现与资源归属

按用户指定，local 终端统一使用 [`github.com/aymanbagabas/go-pty`](https://github.com/aymanbagabas/go-pty)，基于已核对的 `v0.2.3` API 实施。Unix PTY 和 Windows ConPTY 均通过该库创建及读写，适配现有 `TerminalRunner/TerminalSession`；平台文件仅处理 shell 选择和进程清理差异。已有 SSH 终端继续使用现有 runner。

使用 `pty.New()` 创建终端；Unix 使用 `Pty.Command(...)`、`Cmd.Dir/Env`、`Cmd.Start/Wait` 与 `Cmd.ProcessState` 管理 shell，启动后释放父进程持有的 slave 端。Windows 经 go-pty 暴露的 ConPTY 句柄启动系统 PowerShell，由平台适配器持有并释放 CreateProcess 返回的进程/线程句柄，避免 v0.2.3 的 Cmd.Start 遗漏原始进程句柄。进程等待和 PTY 关闭分别完成，由本地会话协调 context 取消、输出泵与关闭顺序，不把关闭 PTY 等同于等待进程。

`v0.2.3` 要求 Go 1.25，并依赖 `creack/pty v1.1.24`、`u-root v0.16.0`、`x/crypto v0.51.0`、`x/sys v0.44.0`；实施时锁定 go-pty 版本，并接受由该依赖图要求的必要升级，检查实际 `go.mod/go.sum` 结果及无 CGO 构建。库的 Windows 进程句柄释放情况列入下方风险与资源验证，不因该风险改换用户指定的库。

`localrunner.Runtime.StartTerminal` 校验 target type 和尺寸，实现现有 Input、Output、Resize、Wait、Close。shell、PTY/ConPTY、输出泵和清理回调由会话对象统一拥有，关闭幂等，启动失败时释放已取得资源。

- Unix：通过 PTY 建立 shell 会话及控制终端；取消/关闭时向 shell 与相关前台进程组发送结束信号，必要时强制终止，关闭 PTY 并回收 shell。正确处理交互 shell 作业控制使前台命令不一定与 shell 同组的情况。主动脱离终端会话的 daemon 不纳入进程树清理承诺。
- Windows：以 suspended 状态启动 shell，加入设置了 kill-on-close 的 Job Object 后再恢复执行。失败即终止并释放本次资源；正常结束或断开时关闭 Job 清理所属子进程。平台适配器等待并读取实际退出码，释放其持有的进程/线程句柄；使用 go-pty 创建、调整尺寸并关闭 ConPTY，输出泵持有复制的输出管道句柄以保留尾输出。
- 会话内部持续读取 PTY 输出，使用有界流向 handler 交付。显式关闭时先解除对外输出的背压并切为排空/丢弃，再结束进程并关闭 ConPTY；不能等待已经退出的 WebSocket 消费者，不能先停止读取再调用可能阻塞的 `ClosePseudoConsole`。
- 自然退出时从 `Cmd.ProcessState` 取得真实退出码，非零退出按正常进程退出处理。Unix 先让输出泵读完 PTY 剩余数据，再关闭 master，不能在进程 Wait 返回后立即截断输出；Windows 在输出泵仍运行时完成 ConPTY 关闭与排空。最后结束 Output，handler 保持先发送尾输出、再发送 exit 的顺序。context 取消属于断开流程。等待、关闭和 Resize 的竞态由会话对象协调。

继续复用当前 WebSocket 桥接，若 handler 的结束顺序妨碍本地清理，只调整释放/取消顺序，保持消息协议、结束原因和 SSH 行为。

### 5. 分派、页面与日志

`targetrunner.Runtime` 增加 `StartTerminal`，通过现有 `forTarget` 按持久化类型选择 runtime，再断言窄 `environmentport.TerminalRunner` 能力。local/ssh runner 与 dispatcher 添加编译期接口断言；不扩展 deployment port，不根据 SSH hostname 或连接失败切换目标。

`newDeploymentRuntime` 保持返回 local runtime 与 dispatcher，但第二项返回具体 `targetrunner.Runtime`，让 HTTP 同一实例可注入部署接口和终端接口。`applicationServices.EnvironmentTerminalRunner` 改为 `environmentport.TerminalRunner`，替换单独 SSH runner；worker 继续用于原部署路径。

Environment 页面仅以环境已配置为终端按钮和抽屉的显示条件。初始化密钥按钮继续仅用于 SSH。标题按类型使用 `environment.local` 的启动展示快照或 `environment.ssh` 的用户/主机，标明本机或 SSH；缺少展示字段时不拼出虚假的 `user@localhost`，也不影响连接。

抽屉 key 包含 Project ID、Environment ID、target type/revision；沿用关闭仅隐藏、重新打开保留会话、离开页面/目标切换销毁、断线手动重连的行为。标题展示刷新不作为独立会话身份，避免元数据更新误触发 disconnect。

沿用现有环境数据读取和刷新方式，不新增首次连接 ready 后的指纹刷新事件或跨操作响应排序机制。Probe 结果变化不改变终端实例 key；真实目标变化继续按现有生命周期释放会话。

日志统一记录 actor、Project、Environment、target type、运行用户、时长与结束原因。local 身份复用启动时的 LocalDisplaySnapshot，SSH 使用保存的 SSH 用户，不无条件解引用 `Environment.SSH`。保留认证、Origin、票据响应不捕获正文和终端流不记录的现有保护。

继续使用现有两个路由及 `orbit-terminal-v1` 协议，保留 30 秒票据、15 秒 WebSocket 启动、15 分钟空闲、2 小时会话、2 秒复核、全局 16/每用户 2 会话以及现有尺寸/消息限制。保持票据容量和配额回收逻辑。不新增 API、proto 字段、前端依赖或数据库迁移。

## Implementation steps

- [x] **终端用例解耦**：移除部署 resolver 注入，复用环境读取和凭据 helper，增加 local 分支；调整签发、兑换及持续复核的 Probe 门槛。SSH 只在现有实际连接中完成必要的首次指纹记录解耦。
- [x] **本地 PTY 与组合入口**：接入并锁定 `github.com/aymanbagabas/go-pty`，实现 local 会话与平台清理，接入现有 target dispatcher 和 HTTP 注入；HTTP/worker 构建通过。
- [x] **入口与展示**：开放所有已配置环境入口，按类型提供已有用户/主机展示信息，移除日志中的 SSH-only 解引用；沿用 AppDrawer、xterm、票据与会话生命周期。
- [x] **测试与活文档**：相关测试及四份活文档已补充，真实 Windows/Linux 容器/DooD PTY、前端 15 个测试、Go 全量测试及四个平台构建已通过；仓库固定检查已执行，`task check` 的最终结果与其他开发检查如下记录。

以上记录 Implementation 的实施进度与开发检查；不据此创建 Verification 文档或宣称未执行的浏览器/真实远端 SSH 场景通过。实现完成后按 SpecFlow 由用户要求进入 Verification。

## Files to change

| 范围 | 计划文件与责任 |
| --- | --- |
| 终端用例/端口 | `internal/application/environment/usecase/terminal.go`、必要时拆出的 `terminal_target.go`、`internal/application/environment/port/terminal.go`；终端解析、local 分支和既有会话校验适配 |
| 首次指纹记录 | `internal/application/environment/usecase/probe.go`、`internal/repository/environment.go`、`internal/repository/impl/sqlc/environment/repository.go`、`sql/query/environment/environment.sql`、生成的 `internal/gen/sqlc/environment/`；仅增加/复用窄指纹记录，无普通配置更新接口改造 |
| 本地执行器 | `internal/infrastructure/runner/local/` 新增 `terminal.go`、`terminal_unix.go`、`terminal_windows.go` 及对应测试；统一 go-pty 适配，平台 shell/进程清理、工作区与资源管理 |
| 已有 SSH 链路适配 | `internal/infrastructure/runner/ssh/terminal.go`、`runtime.go`、`environment_probe.go` 中相关 helper 及已有测试；复用连接和 callback，仅传回首次观察指纹，不新增验证器 |
| 分派/注入 | `internal/infrastructure/runner/target/runtime.go`、`internal/bootstrap/deployment_runtime.go`、`http.go`；窄终端能力和组合；验证 `worker.go` 编译 |
| HTTP 与日志 | `internal/api/http/handler/environment/terminal.go` 及测试；移除 SSH-only 日志假设，保证统一清理与既有协议 |
| Go 依赖 | `go.mod`、`go.sum`；新增固定 go-pty 依赖及其要求的必要依赖升级和 tidy 结果 |
| 页面 | `web/src/views/environment/EnvironmentPage.vue`、`EnvironmentTerminalDrawer.vue`、对应测试、`web/src/i18n/locales/zh-CN.ts` 与 `en-US.ts` |
| 有效验证 | 现有 `usecase/terminal_test.go`、相关 Probe/SSH/HTTP 测试与替身，新增 local PTY 测试及窄指纹写入的实际仓储验证 |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`、`docs/guides/docker-deployment.md`，按需在 `docs/decisions/ledger.md` 记录取代 SSH-only 范围的有效决策 |

不改写此前 SSH 终端的过程文档或归档材料；不修改本次工作之外的 settings 过程文档。普通环境保存、Wizard、SSH 凭据初始化、完整配置恢复、路由、DTO、MCP、迁移和 Docker/Compose 检查命令不列入变更范围。

## Verification plan

### 自动化验证

1. **用例与 HTTP**：local 无 SSH 字段可签发、兑换、运行复核；local/ssh 在 Probe 未运行、失败或修订过期时仍可连接，Probe 状态变化后会话仍有效；验证打开终端未调用 Prober/Docker。保留成员、一次性票据、目标变化、SSH 凭据变化、Origin、配额释放与日志脱敏的有效回归。
2. **已有 SSH 回归**：复用现有测试 SSH server，证明已有受管用户、密钥、host-key 校验及 PTY 行为保持有效。首次无指纹且 Docker 不可用时，使用同一次终端连接取得指纹，认证失败不记录；指纹记录或目标复核失败时不发送 ready、不接受输入。票据签发不产生额外 SSH 连接。
3. **窄指纹写入**：在 SQLite 上通过生成查询与仓储执行指纹固定，证明旧 target revision、凭据变化或不同指纹不能被覆盖，指纹记录不改变 Probe/Gateway 字段。沿用项目已有 SQLC 参数规则；不增加全量配置条件更新及配置恢复竞态测试任务。
4. **local PTY**：按平台验证工作区/`~`、输入输出与回显、控制键、resize、非零退出码及尾输出；关闭含前台长命令的会话、大输出但消费者已经退出的会话，确认在有限时间内清理子进程、输出泵和句柄。只终止测试创建的进程，不管理用户服务。
5. **前端**：local/ssh 均有入口，local 无初始化密钥入口；不引入站点地址判断。复用现有关闭/重开、隐藏期间连接、手动重连、Project/目标切换测试，并证明 local 采用相同行为、Probe 失败不终止终端。
6. **部署回归**：保留 `target_test.go` 中部署需要最新成功 Probe 的断言，运行已有相关测试，证明终端解耦不改变部署 resolver 的要求。

实施时运行：

```powershell
task sqlc
yarn --cwd web lint:fix
yarn --cwd web typecheck
yarn --cwd web test src/views/environment/EnvironmentPage.test.ts src/views/environment/EnvironmentTerminalDrawer.test.ts
task check
go test ./cmd/... ./internal/...
```

沿用现有发布入口/构建参数，以 `CGO_ENABLED=0` 编译 Windows/Linux/Darwin amd64，以及 Linux arm64；检查 go-pty 的平台构建与实际依赖升级，确认未引入 CGO 或无关版本升级。编译不代替终端实机运行。

### 端到端场景与证据

| 场景 | 必须证明的行为 |
| --- | --- |
| Windows 原生 local | localhost/127.0.0.1 页面可见入口；ConPTY PowerShell 以 Orbit 用户进入工作区；控制键、resize、退出与 Job 清理正确 |
| Linux 原生 local | PTY shell 的用户、工作区、交互、控制键和进程组清理正确 |
| Linux 容器 / DooD local | hostname/用户和路径证明进入 Orbit 容器；可访问既有工作区与 Docker socket，Docker CLI 使用既有权限；没有进入宿主机或其他应用容器 |
| Docker 不可用 | 使用隔离测试配置/目标使 Docker 连接失败，local/ssh 仍打开并交互；shell 内 Docker 命令独立失败，已运行终端不被 Probe 失败关闭 |
| Linux/Windows SSH | 即使尚未 Probe 或 Docker 检查失败，已授权受管密钥仍可认证、固定指纹并进入目标默认 shell；后续严格 pinning，目标/凭据变化断开 |
| 浏览器与生命周期 | 桌面/窄屏可操作，无标题或控件重叠；关闭重开保留会话，切换环境释放；Probe 结果变化不重建会话 |

测试使用已有用户管理的服务，或独立测试进程/容器，不主动启动、停止或重启开发服务器，也不停止 Docker Desktop 来制造故障。记录实际目标、命令结果、退出/资源观察及浏览器截图；未执行的场景明确列为未完成，不以替身或交叉编译宣称端到端通过。

## Blockers

无需要用户补充产品决策的事项。当前已在 Windows 主机及独立 Linux 容器完成真实 PTY 检查，DooD 测试使用继承 Docker CLI 的独立容器和已有 socket；没有管理用户开发服务器或停止 Docker Desktop。Linux 原生、真实 Linux/Windows SSH 目标及浏览器完整端到端场景尚未执行，不以替身、容器测试或交叉编译替代这些证据。

## Assumptions

- 沿用当前单节点票据/配额存储与 Project 成员授权。local shell 使用 Orbit 进程权限，不为 Project/Orbit 用户创建操作系统账户或隔离沙盒。
- Windows 部署使用支持 ConPTY 的系统；不支持时明确失败，不以普通 stdin/stdout 管道替代。系统内置 PowerShell 可执行；Unix 镜像/主机具有 `/bin/sh`。
- 使用当前工作区保存规则与挂载。容器 shell 继承已有环境、文件和 socket 权限，DooD 可用性由实际配置决定。
- SSH 首次信任沿用当前首次成功受管密钥认证后观察 host key 的策略；已有指纹始终严格比较，不新增浏览器确认指纹的流程。
- 当前开发工具已确认 Windows PowerShell、Go 1.25.8、Node 22、Yarn 1.22、Task 3.46 与无 CGO；这些只是计划依据，不代表修改后的检查已经通过。

## Risks

- local 终端使已有 Project 成员能够以 Orbit 用户操作整个运行环境，包括该用户可见的环境变量和 Docker socket；工作区只限定启动目录。入口和权限沿用已接受意图，需保持成员复核及元信息日志。
- ConPTY 关闭可能等待输出排空；作业控制可能使 Unix 前台命令处于另一个进程组。内部输出泵、取消顺序和实际子进程清理是本次重点验证项。
- 现有整行配置保存可能覆盖并发写入的指纹，这是已发现的既有并发风险。本任务保护当次指纹记录不覆盖新目标或不同指纹；不据此改造全部保存和恢复接口，也不声称已解决整行配置保存的所有竞争。
- 首次 SSH 指纹必须在同一次实际终端连接中记录，固定失败时不得继续向浏览器提供交互。已有指纹保持严格校验，不能用移除 Docker 门槛来放宽认证或 host-key 校验。
- 已确认 go-pty `v0.2.3` 的 Windows `Cmd.start` 会遗漏 CreateProcess 返回的原始进程句柄。本次平台适配器直接管理该进程/线程句柄，真实 Windows 重复创建/关闭测试已通过；ConPTY 的创建、尺寸调整与释放继续使用 go-pty。提前转移输出管道所有权，避免关闭原始管道取消复制句柄的读取并截断尾输出。库上游问题本身未修改。
- go-pty 要求提升当前 `x/crypto`、`x/sys` 并引入 `u-root` 等依赖；升级影响通过项目固定检查和跨平台构建验证。Windows 编码、尾输出及长期运行行为仍需实测。
- 既有原生 macOS 包继续交付不等于新增 macOS Environment 平台支持；真实 SSH、反向代理 Upgrade 和 DooD 挂载验证须注明实际范围。

## Rollback

实现后如需回退，以本任务实际提交/变更范围恢复代码、依赖和活文档，不回退用户其他改动，不引入长期功能开关或两套逻辑。回退后的 SSH-only 和 Probe 门槛会恢复，local 入口也会回到旧行为，应明确告知该行为差异。

无数据库模型或迁移变更。既有 SSH 连接记录的合法指纹与旧模型兼容，回退不清空该数据；local 首次连接创建的工作区目录保留，不删除用户目录或终端执行的文件。正在运行的会话按关闭路径释放；如需要部署回退，由用户授权管理相应服务，计划阶段不操作。

## User review notes

- 用户要求不隐藏终端入口，并接受 DooD 下 Orbit 容器内的终端；localhost/127.0.0.1 也保留入口。
- 用户明确“打开终端就是打开终端，检查 docker 是检查 docker，两者不混在一起”；本计划同时覆盖 local/ssh 的签发、兑换、持续复核及 SSH 首次信任建立。
- 用户要求“开始计划，所以事项需要在计划中理清”；Intent 已 Accepted，本 Plan 为 Draft，local shell、库版本、执行位置、清理归属、注入方式及终端解耦范围已明确。
- 用户指出 SSH 已有实现，并要求修正计划。据此删除独立 SSHHostKeyVerifier、票据前额外握手、全量环境配置条件更新、扩展票据目标快照、专门的指纹刷新和异步响应排序；SSH 只保留已有链路内解除 Docker 依赖所需的调整。
- 用户明确“终端使用 github.com/aymanbagabas/go-pty”。据此 local 终端统一使用该库，替换原按平台选择两个库的方案；同步 API 适配、依赖影响和资源验证要求，不扩展已有 SSH 实现的范围。
- 当前没有需要用户确认的未决产品事项；待完成的是计划审查及后续实施/实机验证。此阶段未修改产品代码，也未运行构建、产品测试或管理开发服务器。
- 用户要求“开始实现”，据此接受本计划并进入 Implementation。已确认 go-pty v0.2.3 的 Windows 原始进程句柄遗漏，采用上文平台适配器处理进程所有权，终端核心仍使用用户指定的 go-pty；不扩展产品范围。
- 终端实现后，用户另外要求初始化和编辑环境提交匹配目标配置时只警告、不阻止。这是原终端计划之外的明确追加变更；实际文件和验收证据见 Verification 的范围变化记录，不倒写原计划范围。
- 用户反馈“测试无问题”，授权暂存相关变更并完成 SpecFlow Verification 文档，进入验证阶段；未授权提交或推送。

## Implementation record

以下保留 Implementation 完成时的记录，后续用户追加的目标警告及 Verification 结果另见验证文档。终端代码覆盖计划中的用例、窄指纹仓储/SQLC、local PTY、既有 SSH callback、dispatcher/bootstrap、HTTP、页面与测试；终端部分没有新增路由、proto 字段或迁移，没有改造普通配置保存或重新实现 SSH 认证。既有首次观察 callback 在第一次观察后固定同一连接后续 rekey 的 host key，维持严格校验。

开发检查结果：

- `task sqlc`、`go mod tidy`：通过，锁定 go-pty `v0.2.3` 及其要求的依赖升级。
- `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`：通过。
- 两个 Environment 前端测试文件：15 个测试通过，涵盖 local/ssh 入口、抽屉隐藏/重开、手动重连及失败 Probe 不重建会话；本任务页面 Prettier 检查通过。
- `go test ./cmd/... ./internal/...`：通过；仓库全量 Go 格式检查、golangci-lint 配置校验及 lint 均通过。
- 真实 Windows PTY：工作区与 `~`、输入输出、resize、非零退出码和尾输出、Ctrl-C 恢复交互、前台子进程清理、大输出消费者退出后关闭、重复连接句柄回收通过。
- 独立 Linux 容器：相同 PTY 测试通过；DooD 测试使用 `docker:29.4` 和已有 socket，通过 shell hostname 与当前容器一致、工作区和 Docker CLI 可访问的检查。仅创建并自动清理本任务测试容器。
- Windows/Linux/Darwin amd64、Linux arm64 的 `go build ./cmd/...`，`CGO_ENABLED=0`：通过。首次并发构建的临时导入错误经顺序重跑消失。
- `git diff --check`：通过。
- `task check`：最终完整通过，包含前端类型、lint、Prettier，以及 Go 配置校验、格式和 lint。

实施过程中全仓检查曾受到同时进行的 settings 变更的格式问题影响；这些并行改动完成整理后，最后一次 `task check` 完整通过。本任务未修改这些 settings 文件，并保留同文件内其他人的 bootstrap 和 i18n 改动。测试生成的 Linux 二进制和本任务独立容器已清理。

实施结束时未执行浏览器完整端到端、Linux 原生和真实远端 SSH 验收，也未管理用户开发服务器。既有环境整行保存的并发指纹覆盖风险仍按 Risks 保留；当时尚未创建 Verification 文档或暂存代码。用户已在后续反馈人工测试通过并要求完成验证文档和暂存，结果以 Verification 记录为准。
