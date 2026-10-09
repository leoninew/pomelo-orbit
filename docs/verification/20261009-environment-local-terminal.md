# Environment 本机终端与共用目标警告验收
最后修改时间: 2026-10-09 18:08:00

Review status: Accepted

Mode: standard

## Basis

- [Intent](../intent/20261009-environment-local-terminal.md)：Accepted。
- [Plan](../plan/20261009-environment-local-terminal.md)：Accepted。
- 用户指定本机终端使用 `github.com/aymanbagabas/go-pty`，并要求开始实现。
- 实现完成后，用户另行要求修改环境独占行为：初始化和项目环境编辑模态窗提交相同目标配置时只警告、不拒绝。该追加要求已实现，并纳入本文的范围变化记录；不将其倒写为原终端计划的内容。
- 用户于 2026-10-09 明确反馈“测试无问题”，授权暂存相关变更并完成 SpecFlow 验证文档。本文区分用户人工验收、已执行自动化检查和未逐项收集的端到端证据。

## Intent alignment

- 所有已配置的 local/ssh Environment 显示终端入口，浏览器访问地址不参与判断。
- local shell 与 Orbit 进程处于同一运行环境、使用相同操作系统用户；原生运行在主机，容器运行在 Orbit 容器。保存的工作区是启动目录，使用时展开 `~`，不构成文件系统隔离。
- local 使用 go-pty `v0.2.3`，Unix 为 `/bin/sh -i`，Windows 为系统 PowerShell 和 ConPTY；不需要 SSH 配置或凭据。
- local/ssh 的票据、兑换、会话复核均与 Docker Probe 独立。SSH 继续检查受管凭据及主机指纹，部署、CI 和 Route 的成功 Probe 要求保留。
- 复用既有 SSH 连接、PTY、票据、WebSocket、Project 成员授权、会话配额和抽屉生命周期；首次指纹在实际认证连接内固定，不新增预连接或独立 SSH 验证流程。

## Spec alignment

标准模式未创建独立 Spec，依据 Intent、Plan 及用户后续明确要求核对。共用目标警告属于实现后的追加变更，其依据和验证结果单独列出。

## Plan alignment

| 计划内容 | 实际实现与证据 |
| --- | --- |
| 终端目标独立解析 | `TerminalService` 不再依赖部署 TargetResolver，读取环境与对应凭据；终端用例测试覆盖未 Probe、失败 Probe 和过期 Probe |
| 本机 PTY/ConPTY | local runner 通过 go-pty 创建终端，使用平台文件管理进程与清理；真实 Windows 和独立 Linux 容器 PTY 测试通过 |
| 工作区与身份 | local 使用进程身份和展开后的保存路径；工作区、home 展开及输入输出测试通过；DooD 检查证明进入容器 |
| SSH 最小适配 | 复用已有 SSH runner 和首次观察 callback；同一连接后续 rekey 仍受指纹约束，认证或指纹固定失败不提供交互 |
| 窄指纹写入 | 条件更新仅写指纹和更新时间，绑定目标与凭据修订；SQLite 仓储测试验证不覆盖不同指纹、Probe 或 Gateway |
| 分派与注入 | 既有 target runtime 分派 local/ssh，HTTP 注入 TerminalRunner 窄接口；HTTP 和 worker 构建与 Go 测试通过 |
| 页面与生命周期 | local/ssh 入口和标题统一；关闭重开保留实例，目标变化释放，Probe 状态变化不重建；前端测试通过 |
| 日志与协议 | 保留现有终端路由、WebSocket 协议、票据与配额；日志支持 local，不采集终端流或认证秘密 |
| 活文档与构建 | 同步产品、运行时及原生/Docker 部署指南；四个目标的无 CGO 构建已在实施阶段通过 |

终端部分没有新增 API 路由、proto 字段或数据库模型。追加的共用目标警告增加了响应字段，未增加迁移、持久化警告状态或 Docker 身份探测。

## Actual diff summary

- 终端应用层新增 local 支持，移除 Docker Probe 门槛，并在 SSH 交互开始前确认首次观察的指纹。
- 增加 local PTY 会话、Unix/Windows 平台适配和真实终端资源回收测试；通过既有 target dispatcher 注入。
- 增加只更新主机指纹的 SQLC 查询、仓储方法及实际 SQLite 验证，Probe 复用该窄写入。
- 环境页始终提供终端入口，抽屉按目标类型显示身份，沿用现有连接、关闭和手动重连行为。
- 移除环境目标配置匹配造成的 409 拒绝。读取和保存环境时动态计算 `target_may_be_shared`，初始化、编辑和生成 SSH 命令保存成功后显示非阻塞警告。
- 警告沿用其他活跃 Project 的配置匹配：local 与 local 匹配，SSH 比较保存的 host + port，类型之间不交叉匹配；不宣称已识别同一个 Docker daemon，不暴露其他 Project 的身份或凭据。
- 更新 Environment 和初始化快照 proto、对应 Go/TypeScript 生成代码、中英文文案与活文档。

## Expected versus actual files

| 范围 | 实际文件 | 对齐情况 |
| --- | --- | --- |
| 终端用例及端口 | `internal/application/environment/usecase/terminal*.go`、`port/terminal.go` | 按计划；直接在既有用例中解析目标，未新增独立验证器 |
| 指纹与仓储 | `usecase/probe*.go`、`internal/repository/environment.go`、`impl/sqlc/environment/repository*.go`、环境 SQL 与 SQLC 生成代码 | 按计划；没有改造全量配置条件更新 |
| 本机终端 | `internal/infrastructure/runner/local/terminal*.go` | 按计划；Windows 管理进程句柄与 Job，Unix 管理 shell/前台进程组 |
| SSH 与分派 | `internal/infrastructure/runner/ssh/{runtime,terminal,environment_probe}*.go`、`runner/target/runtime.go` | 按计划，保留既有 SSH 认证与 PTY |
| HTTP 与组合根 | `internal/api/http/handler/environment/terminal*.go`、`internal/bootstrap/{http,deployment_runtime}.go` | 按计划；`http.go` 仅暂存终端修改 |
| 依赖 | `go.mod`、`go.sum` | 锁定 go-pty 及必要依赖升级，保留无 CGO 构建 |
| 页面及抽屉 | `web/src/views/environment/EnvironmentPage*`、`EnvironmentTerminalDrawer*`、两份 locale 文件 | 按计划；locale 只暂存终端及目标警告文案 |
| 目标警告应用层 | `usecase/{service,configuration,ssh_initialization,view}.go`、`dto/environment.go`、相关集成测试 | 用户追加范围，保存匹配配置不再返回 conflict |
| 目标警告传输层 | Environment/Project Initialization HTTP handler、两个 proto 和对应生成代码 | 用户追加范围，动态布尔响应字段，不持久化 |
| 初始化界面 | `web/src/views/project/ProjectInitializationPage.vue` 及测试 | 用户追加范围，警告后继续正常流程 |
| 文档 | 本任务 Intent、Plan、本文及四份活文档 | 按计划；Docker 指南只暂存终端说明和标题时间 |

同时进行的 settings/config、覆盖文件存储、settings proto/页面、其他文档和配置样例不属于本次暂存。混合文件按修改片段拆分，不撤销工作区中的其他改动。

## Acceptance checklist

- [x] local/ssh 都显示终端入口，local 不显示 SSH 密钥初始化入口；不依据浏览器地址隐藏。
- [x] local 无 SSH 配置可签发、兑换、运行并复核票据，展示和日志不解引用 SSH 字段。
- [x] Windows ConPTY 的工作区、home、输入输出、resize、退出码、尾输出和 Ctrl-C 恢复交互通过真实 PTY 测试。
- [x] Windows Job/句柄回收、Unix shell/前台进程组终止、未消费大输出关闭通过 Windows 或 Linux 容器测试。
- [x] Linux 容器及 DooD 测试确认 shell hostname 与当前容器一致、工作区和既有 Docker socket/CLI 可访问；未改变用户权限或挂载。
- [x] local/ssh 在 Probe 未执行、失败或过期时仍可连接；打开终端不调用 Docker 检查，失败 Probe 不重建现有终端。
- [x] SSH 受管身份、凭据、主机指纹和实际 PTY 行为通过受控 SSH server 测试；首次固定失败不发送 ready，不接受输入。
- [x] 条件指纹写入不覆盖目标/凭据变化、不同指纹或 Probe/Gateway 字段；实际 SQLite 仓储测试通过。
- [x] 成员复核、一次性票据、配额释放、目标或凭据变化断开、日志保护通过既有及更新测试。
- [x] 抽屉关闭与重开保留终端和会话，离开或目标变化清理，断线仅手动重连；前端测试通过。
- [x] 部署 resolver 的最新成功 Probe 要求保留，相关现有 Go 回归通过。
- [x] 没有数据库模型或已执行迁移修改，没有新增终端 API；Windows/Linux/Darwin amd64、Linux arm64 无 CGO 构建通过。
- [x] 重复 local 或 SSH 配置能创建和编辑，真实 SQLite 检查确认保存落库；修改工作区/SSH 用户仍提示可能共用目标，废弃 Project 不再触发警告。
- [x] 生成 SSH 初始化命令及保存已知环境定义不因匹配配置拒绝；受管凭据流程继续适用。
- [x] 初始化保存后显示警告并进入检查步骤，编辑保存后显示警告并关闭模态窗；没有匹配时维持普通成功提示。
- [x] 用户反馈“测试无问题”，接受当前实现并要求完成验证文档、暂存相关变更。
- [ ] Linux 原生、真实远端 Linux/Windows SSH、浏览器桌面/窄屏和生产代理 Upgrade 尚未逐项收集端到端证据；不将容器、替身或编译结果当作这些场景的实测。

## Test results

| 检查 | 结果与证据范围 |
| --- | --- |
| `task sqlc`、`go mod tidy` | 实施阶段通过，生成窄指纹查询并锁定依赖 |
| `task proto` | 目标警告实现阶段通过，更新 Environment/初始化响应契约 |
| `yarn --cwd web lint:fix` | 目标警告完成后通过 |
| `task check` | 最后一次实现检查完整通过，包含前端 typecheck/lint/format、Go 格式及静态检查；0 issues |
| `go test ./cmd/... ./internal/...` | 目标警告完成后完整通过，部分既有包使用 Go 缓存 |
| 三个 Environment/Project Initialization 前端测试文件 | 本次 Verification 重跑：3 个文件、23 项测试全部通过 |
| 真实 Windows PTY | 实施阶段工作区、home、I/O、控制键、resize、尾输出、子进程/句柄回收通过 |
| 独立 Linux 容器及 DooD PTY | 实施阶段真实交互和清理通过；DooD 使用 `docker:29.4` 与已有 socket，仅清理本任务测试容器 |
| `CGO_ENABLED=0`，Windows/Linux/Darwin amd64、Linux arm64 的 `go build ./cmd/...` | 实施阶段顺序构建通过，编译不代替对应平台交互实测 |
| 用户人工测试 | 用户明确反馈“测试无问题”，未提供逐平台测试清单 |
| 仅暂存内容的独立 Go 检查 | 通过；从暂存树导出独立源码快照，以 `CGO_ENABLED=0` 运行 `go test ./cmd/... ./internal/...`，确认不依赖未暂存的 settings/config 改动 |
| 暂存范围及 `git diff --cached --check` | 通过；共 61 个相关文件，四个混合文件只暂存相关片段，settings/config 修改保留在工作区 |

本次没有产品代码改动，收集已通过的项目检查并重跑三个相关前端测试。未管理用户开发服务器，也未重新执行不变的依赖生成和跨平台构建。

## Scope changes and risks

- 原终端计划明确排除普通环境保存、Wizard 和 proto 变更。用户在终端实现后明确追加“环境提交只警告、不阻止”，因此本次暂存增加相应保存路径、响应契约和界面修改，本文显式记录这一偏离。
- 目标匹配只是配置层面的提示。不同地址可指向同一 daemon，同一 SSH 地址不同用户也可能使用不同 daemon；本次没有引入身份识别或将启发式判断改成新的阻塞规则。
- 允许多个 Project 保存匹配目标配置后，实际部署仍可能发生端口、容器名称及工作目录冲突；当前保存操作不保证资源隔离。
- local shell 继承 Orbit 用户权限，工作区仅是启动目录。容器终端不会进入宿主命名空间，也不会增加 DooD 权限。
- go-pty v0.2.3 的 Windows 原始进程句柄遗漏通过平台适配器处理；ConPTY 仍由该库管理，上游库本身未修改。
- 现有环境整行保存仍可能覆盖并发记录的指纹。窄指纹写入只保护自身条件更新，按用户已确认范围保留这一既有风险。

## Incomplete items

没有已知产品实现未完成项。Linux 原生、真实远端 SSH、浏览器逐视口布局/焦点/中文交互及生产代理端到端证据未单独补齐；用户当前人工验收通过，不据此推断所有平台均已实测。

独立验证使用的系统临时目录 `orbit-environment-staged-fec8f45a5b9a4ffe95419a5f87a4243c` 保留源码快照与归档。已核对它是本次创建的普通目录；自动审批仍拒绝清理操作，仅返回 `blocked by policy`，未提供详细原因。该临时材料不在仓库或暂存区中。

## Conclusion

终端实现及用户追加的共用目标警告已通过自动化检查和用户人工验收，本文据用户确认标记为 Accepted。相关变更已暂存，混合文件已按范围拆分，暂存源码的独立 Go 测试及空白检查通过。特定平台的端到端证据边界如上保留，临时快照清理受到工具限制。未创建提交或推送。
