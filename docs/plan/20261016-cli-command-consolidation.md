# Plan / 计划：共享业务的 Go CLI 远程环境管理
最后修改时间: 2026-10-16 13:55:49

## Review status / 审查状态

Accepted

用户“开始实施”接受本计划及 Q01 旧专属 HTTP 清理建议，本会话顺序实施并执行必要开发检查。范围为本地 Go CLI 直接 SSH 操作远程环境，与 WebAPI 共用配置、数据和应用用例，沿用现有身份及 cwd 加载/失败行为。首版包含 `list`、`ssh`、`exec`、`forward start/status/stop`；实施先移除 Python CLI，以标准库单文件 `scripts/release.py` 接续打包，再建设 Go CLI。

## Intent basis / 意图依据

- [父级意图](../intent/20261016-cli-command-consolidation.md) A01—A08。
- 用户明确要求：Go CLI 直接 SSH，与 Go WebAPI 共用配置、数据，更要共用业务；使用 SpecFlow 记录任务。
- 用户已采纳同一二进制，指定环境管理命令组为 `pomelo-orbit cli env`；保留现有配置读取方式，不引入根目录参数或路径发现。
- 用户明确忽略 local 环境，只考虑本地 CLI 通过 SSH 操作远程环境，沿用既有密钥和身份。
- 用户明确要求 Python CLI 相关实现首先移除；后续允许以最小依赖、单文件 `scripts/release.py` 承接复杂打包，不保留完整 uv 项目。
- 用户明确端口转发使用本地 `-L`，不扩展目标主机访问其他主机的管理能力；要求规划 `forward` 状态、启动、停止。
- 代码基线：HEAD `4e83dda1639cd7995f2eaa1d1031779672f64058`；实现前重新核对工作区。
- 分层依据：[后端架构](../archive/architecture/backend.md)、[项目架构治理 skill](../../.agents/skills/backend-architecture-governance/SKILL.md)。

## Technical decisions / 技术设计与决策

### 已确认的共享边界

```text
WebAPI 入站适配 ─┐
                ├→ 共享应用用例 ─→ Repository → 同一数据库
Go CLI 入站适配 ─┘             └→ SSH 基础设施 → 目标主机
```

- 直接复用 WebAPI 的配置、数据及应用用例。CLI 负责参数、交互、终端和输出；项目选择、操作者权限、凭据绑定/解密、主机指纹和目标修订规则在共享 application 内执行。
- CLI 只操作受管远程 SSH 环境。SSH 连接使用同一环境保存的账号、受管密钥和指纹，不增加 local 分支、另一套主机配置或独立身份体系。
- 凭据在本地应用进程内部使用，SSH 由现有 Go 基础设施及明确端口承担，不经 WebAPI 领取密钥，不新增 HTTP/WebSocket 隧道协议。
- 终端、执行及新增端口转发按业务职责复用。现有连接业务位于 `terminal.go` 本身不是重构理由；需要扩展时提取实际共享职责，不为 CLI 加一层转发 facade，不在 CLI 组合仓储与解密。
- 端口转发仅为本地 `-L`：本机回环监听，经 SSH 连接访问 SSH 主机自身回环地址端口；不增加反向转发、动态代理、任意目的主机或跳板管理。本地监听是远程服务入口，不属于 local 环境支持。

### 入口、共享配置与身份装配

- 同一 `pomelo-orbit` 二进制，保留默认 serve 行为及 `serve`、`worker`、`mcp`，新增 `cli env`。使用 go.mod 已有版本的 Cobra 分发；帮助及参数错误在加载运行配置前处理，保持 MCP stdout 的协议隔离。
- 直接使用 `config.Load()`，按当前 cwd 读取 `configs/config.yaml`、profile YAML、对应 dotenv 和 `overwrite.env`，保留 `POMELO_ORBIT_APP__ENV`、`POMELO_ORBIT_` 前缀、覆盖顺序、可选文件及校验契约。
- 相对数据库、日志和其他本地资源路径沿用现有解析方式。调用者从正确目录和环境启动 WebAPI/CLI；必需配置读取或校验失败直接退出，不增加 `--work-dir`、根目录环境变量或目录发现/兜底。
- 各入口按需装配，CLI 不启动 HTTP/Worker、挂载检查或自动数据库迁移；不因复用 RunMCP 而整体启动 MCP。
- 具体身份接入复用 RunMCP：从共享配置 `cfg.MCP.AccessToken`（`POMELO_ORBIT_MCP__ACCESS_TOKEN`）调用已有认证用例，在本地得到 Orbit 操作者并传入项目成员权限用例。这是已有身份配置，不新增 CLI token、登录命令或服务 URL；SSH 认证则继续使用环境保存的账号和受管密钥。
- `forward status/stop` 只管理本机当前系统用户的受管进程，不打开业务数据库或验证 PAT，避免数据库不可用或凭据失效时无法停止自己的转发；仍沿用共享配置加载及读取失败行为。

### 首版命令清单与用法

| 命令 | 用途与参数约定 |
| --- | --- |
| `pomelo-orbit cli env list [--json]` | 查询成员可见的 SSH 环境，显示 project code、名称和 SSH 端点；过滤 local 与未配置环境 |
| `pomelo-orbit cli env ssh [--project CODE]` | 交互式远程终端；省略 project 时在 TTY 中选择，只接受 SSH 环境 |
| `pomelo-orbit cli env exec --project CODE [--workdir PATH] [--timeout SECONDS] [--stdin] [--json] -- PROGRAM ARG...` | 非 PTY 字面 argv 执行，复用现有工作目录、输入输出、超时和退出结果契约 |
| `pomelo-orbit cli env forward start --project CODE -p LOCAL:REMOTE` | 后台启动本地 `-L` 转发，就绪后返回 |
| `pomelo-orbit cli env forward status [-p PORT] [--json]` | 查看本机受管转发，省略端口则列出全部 |
| `pomelo-orbit cli env forward stop -p PORT` | 停止一条本机受管转发，已停止时幂等成功 |

`-p/--port` 统一端口参数：start 使用 `LOCAL:REMOTE`，status/stop 使用本地端口；status 省略端口仍列出全部。`--project` 保持启动必填，不给旧公开端口参数增加兼容入口。

首版不增加环境配置、初始化、Probe、scp 或通用 remote 命令，不迁移 cert、ulid、version 等旧工具。打包为独立开发脚本，不作为 Go CLI 子命令。命令拼写使用 `forward`。

以下命令已接入 Go 入口。后台转发从稳定路径的二进制启动，开发时先构建再运行：

```powershell
# 从仓库根运行，通过现有共享配置选择器指定 development
$env:POMELO_ORBIT_APP__ENV = "development"
go build -o ./bin/pomelo-orbit.exe ./cmd/server

./bin/pomelo-orbit.exe cli env list
./bin/pomelo-orbit.exe cli env ssh --project production
./bin/pomelo-orbit.exe cli env exec --project production -- whoami
./bin/pomelo-orbit.exe cli env forward start --project production -p 15432:5432
./bin/pomelo-orbit.exe cli env forward status
./bin/pomelo-orbit.exe cli env forward stop -p 15432
```

这是本地编译，不要求先打发布包；Linux/macOS 使用对应平台二进制路径及相同参数。不依赖 `go run` 临时二进制或 Air 自动重启持有长期转发。Windows 运行进程会占用可执行文件，重新构建前先停止该二进制持有的转发；不设计执行 status/stop 前强制 build 的包装命令。

### 旧 Python CLI 的 HTTP 配置与接口

以下为已删除的旧实现，仅作清理和语义对照：

| 配置 | 现有用途 |
| --- | --- |
| `POMELO_ORBIT_CLI__BASE_URL` | Orbit HTTP(S) 服务根地址，可带部署路径前缀；客户端追加 `/api/cli/...` |
| `POMELO_ORBIT_MCP__ACCESS_TOKEN` | 既有 Orbit PAT，通过 `Authorization: Bearer ...` 认证 HTTP/终端连接 |
| `POMELO_ORBIT_APP__ENV` | 选择 `scripts/.env.<profile>`，未指定时读取 `scripts/.env`；进程环境覆盖文件，旧 scripts Makefile 默认 development |

| 现有接口 | Python 命令 |
| --- | --- |
| `GET /api/cli/project` | `environment list`，也是交互选择 SSH 目标的来源 |
| `GET /api/cli/environment/terminal?project_code=...` | `environment ssh`，升级为 WebSocket，子协议 `orbit-terminal-v1` |
| `POST /api/cli/environment/exec` | `environment exec`，提交 project code、argv、workdir、stdin 和 timeout |

旧代码基线为 HEAD `4e83dda` 中的 `scripts/src/pomelo_orbit_cli/settings.py`、`environment_client.py` 和 `internal/api/http/routes/cli.go`；这些文件现已删除。旧 remote 另用 `POMELO_ORBIT_SSH__HOST/USER` 和本机 OpenSSH，不走上述 HTTP；独立目标配置也已移除。

Go CLI 不需要 `POMELO_ORBIT_CLI__BASE_URL` 或 scripts 专属 dotenv，直接使用 WebAPI 的配置、数据库和应用用例。既有 PAT 字段可在本地解析 Orbit 身份，SSH 账号和密钥取自同一受管环境，分别复用现有业务。

Q01 已按用户“开始实施”授权删除专属路由、handler、CLI proto、生成代码、测试及接线，并移除无生产调用方的 HTTP PAT helper。保留浏览器终端、环境应用用例及 PAT/MCP 认证业务。

### 现有复用依据

| 现有实现 | 复用重点 |
| --- | --- |
| [config.Load](../../internal/config/runtime.go) | 直接共用 cwd 加载、profile、覆盖顺序、相对路径和校验；读取失败退出 |
| [RunMCP](../../internal/bootstrap/app.go) | 本地数据库、共享服务与既有 PAT 认证装配，CLI 按需复用 |
| [应用装配](../../internal/bootstrap/http.go)、[仓储装配](../../internal/bootstrap/stores.go) | 共用依赖按职责装配，入口拥有所创建资源的关闭责任 |
| [AccessService](../../internal/application/environment/usecase/access.go) | Project code 解析、成员权限和 SSH 目标限制 |
| [连接业务](../../internal/application/environment/usecase/terminal.go) | 目标、凭据绑定、指纹确认和连接资格复核，不套部署 Probe 前置条件 |
| [SSH 终端](../../internal/infrastructure/runner/ssh/terminal.go)、[命令执行](../../internal/infrastructure/runner/ssh/command.go)、[命令用例](../../internal/application/environment/usecase/command.go) | 取消、退出码、跨平台编码、输入输出限制；CLI 适配本地 TTY、窗口和信号 |
| [单文件打包](../../scripts/release.py)、[Taskfile](../../Taskfile.yml)、[发布流水线](../../.github/workflows/package.yml)、[Go 验证流水线](../../.github/workflows/go-verify.yml) | 首步清理完整 Python CLI/uv 项目，抽出标准库打包脚本，同步替换调用 |

已有只读调查确认 7 个环境中有 6 个 SSH，从当前 CLI 主机检查 SSH 端口均可达。结果不替代登录、转发及跨平台验证；本轮不重复连接或读取实际凭据。

### 端口转发启停、状态和生命周期

- 范围：TCP `127.0.0.1:<local-port>` → 已保存的 SSH 主机 → 该主机 `127.0.0.1:<remote-port>`，端口为 1—65535。不提供方向、任意目的主机、监听地址或前台/后台模式选择；start 即后台运行。
- 启动：共享用例解析项目、操作者权限、目标及凭据；后台完成 SSH 认证、主机指纹确认和本地监听后，start 才返回成功。失败回收子进程、监听与记录，不留下半启动实例。不要求目标服务立即健康，服务不可达由实际连接反馈。
- 重复与冲突：同一受管项目、映射及目标修订仍在运行时幂等成功；同一 local-port 被其他映射、进程或配置实例占用时报错，不替换或停止。并发启动由本地排他控制和真实监听结果约束。
- 后台所有权：同一 Go 二进制通过内部工作入口持有转发，一条映射一个进程，不建常驻通用 daemon。继承启动 cwd 与环境，经共享加载入口读取配置，在进程内使用凭据；私钥和 PAT 不写入命令参数或运行记录。
- 运行记录：放在当前 cwd 的 `data/cli/forward/`，与业务数据库分开，按本机当前系统用户管理，记录项目、映射、PID、启动身份及状态。通过进程创建信息、可执行身份和运行标识核验归属，不仅凭记录、PID 存在或端口占用认定受管进程。就绪通知和停止控制由本地进程适配器实现，不增加对外 HTTP/WebSocket 服务。
- 状态：status 显示项目、映射、PID 和实际状态，区分运行、已退出及失效记录，报告能确认的退出原因；归属不符时显示异常，不将端口上的其他服务认作本应用。running 证明转发进程、SSH 连接和监听有效，不宣称远端服务健康。
- 停止：stop 仅停止已核验归属的一条受管转发，关闭监听、活动 TCP 流和 SSH，清理对应记录；不存在或已退出时清理残留并幂等成功。端口由非受管进程占用时不操作该进程。status/stop 不要求数据库或 PAT 可用。
- 断线及资格变化：沿用 SSH 心跳与共享成员权限、目标/凭据修订复核，连接丢失或资格复核失败即关闭资源并退出，不自动重连、不随系统启动。资格复核沿用现有 2 秒周期；不套终端票据、15 分钟空闲及 2 小时终端期限，也不把现有进程内终端配额宣称为跨进程全局配额。
- 技术清理：支持多个 TCP 连接和双向半关闭，启动失败、连接丢失、主动停止均明确资源关闭责任；不在数据库长事务中持有 SSH。应用层组织资格和生命周期，基础设施实现 TCP/SSH 流与平台进程操作，bootstrap 负责装配。
- 网络前提：`-L` 不在远端新增监听，不要求向公网开放远端服务端口；需 SSH 可达、sshd 允许 TCP 转发，且服务在远端主机回环地址可访问。
- 旧进程：不读取或接管旧 Python `.remote-tunnels.json`，不自动终止旧 SSH 进程；端口被旧进程占用时按占用报错，由操作者处理。

### 终端和命令执行的既有行为

- ssh 要求 stdin/stdout TTY；保留交互选择、窗口变化、远端默认 shell、Ctrl+C 传递和 `Ctrl+]` 本地脱离，退出恢复终端状态。复用终端成员权限、指纹、修订及现行会话规则，不使用浏览器票据流程。
- exec 保留字面 argv 和 Windows/Linux 编码，复用现有命令编码器。默认工作目录为保存的 workspace，可传远端绝对路径或 `~/...`；`--workdir` 是远端命令目录，不是已排除的本地 `--work-dir`。
- 沿用限制：timeout 默认 60 秒、允许 1—300 秒，stdin 最大 64 KiB，stdout/stderr 总上限 1 MiB，argv 最多 128 项/32 KiB；不传 `--stdin` 即发送 EOF。普通输出分写 stdout/stderr，JSON 包含执行结果并明确非 UTF-8 替换情况，日志和诊断不混入 stdout。
- 沿用退出映射：远端正常退出码 0—255，超时 124，输出超限或权限/目标变化 125，平台/传输失败 255，取消 130，参数错误 2。结果未知时不自动重放命令。

### Python CLI 清理与单文件打包例外

- 第一项实施改动移除 `scripts/src/pomelo_orbit_cli/`、`scripts/tests/`、`scripts/pyproject.toml`、`scripts/uv.lock` 及 Python CLI 专属 Makefile/配置示例和入口，覆盖旧 environment/remote、cert/ulid/version/release 包内命令，不保留兼容层。
- 同一步将现有 release 核心逻辑抽为 `scripts/release.py`，只用 Python 标准库，去除 Click、包上下文、dotenv 与 uv 依赖。通过脚本自身位置定位仓库并以仓库为构建 cwd；这是开发打包脚本的定位，不改变 Go 配置加载。
- 直接调用 `python scripts/release.py`，读取现有 VERSION，沿用 Yarn 前端构建、CGO=0 的 Windows/Linux/macOS amd64 Go 构建、ZIP 命名/内容和 Unix 执行权限，全部平台构建成功后才发布构建产物。不新增版本计算或 Go release 命令。
- 保留 `scripts/package/` 的 `.sh`、`.cmd`、`env.release` 等非 Python 资源，不删除整个 scripts。保留根配置、用户 dotenv、密钥与业务数据，不清理数据目录内未跟踪的本地辅助文件。
- Taskfile 移除 uv sync 等旧 CLI 依赖；两条 CI 设置 Python 解释器并直接运行脚本，移除 setup-uv、sync 与旧命令调用。保持 ZIP 校验/上传及 package 工作流 image/publish 的依赖和制品契约。
- 清理并接续打包后再建设 Go CLI，不等 Go 功能齐全再删除 Python CLI；活文档改用新命令。源码清理不停止既有后台 SSH 进程。

### 开发前范围确认

| 编号 | 范围项 | 决策 |
| --- | --- | --- |
| Q01 | 旧 Python 专用 `/api/cli/...` 接口及专属协议代码 | 用户要求开始实施，按已展示建议一并移除；保留 Web 终端、共享用例和 PAT/MCP 认证 |

六个命令、后台生命周期、身份接入、开发启动和打包方案均在本次实施授权内，暂无需要用户确认的未决事项。

## Implementation steps / 实施步骤

按用户“开始实施”授权顺序执行：

1. 首先移除完整 Python CLI、专属测试、依赖和调用；同一步抽出标准库 `scripts/release.py`，接续 Taskfile/CI、修正活文档，按 Q01 确认处理旧专用 HTTP 代码，覆盖 A05。
2. 装配 Go CLI，复用共享用例、身份上下文与配置入口，覆盖 A01—A04。
3. 实现 `cli env list/ssh/exec` 与 `forward start/status/stop`，补足终端适配及后台转发增量，覆盖 A05—A07。
4. 完成最小有效检查和用法说明，覆盖 A08；不恢复 Python CLI 入口。

## Files to change / 需修改文件

实际修改同时覆盖产品实现、必要测试及活文档。

先清理完整 Python CLI、专属测试、依赖、Makefile 及运行入口，新增 `scripts/release.py`，调整 Taskfile、两条 CI 和活文档；随后修改 Go 命令入口、`internal/api/cli`、`internal/bootstrap`、`internal/application/environment`、`internal/infrastructure/runner/ssh`、`internal/infrastructure/runner/forwardprocess` 及必要测试。直接复用 `internal/config`，不改配置根/路径机制；CI 只接续既有能力。

Q01 清理 `internal/api/http/routes/cli.go`、`internal/api/http/handler/cli/`、CLI proto、Go/Web 生成代码和孤立的 HTTP PAT helper，使用现有生成链检查；保留浏览器终端。不新增路由、迁移或广泛重构。发布包启动脚本限定服务进程匹配，避免停止服务时误停独立转发。

## Tasks and coordination / 子任务与协调

不拆分或派发子任务。用户已明确“开始实施”，本会话顺序完成清理、共享用例、CLI/forward 适配及必要开发检查；Implementation 已完成并交付开发检查结果；整体 Verification 尚未开展。该授权接受本计划及 Q01 的清理建议，无需重复确认。

记录修订：13。此前 T01—T03 是尚未分配、没有接收和交付的评估草稿，现已合并撤下；ID 不复用于后续节点。不拆分或派发子任务。

## Verification plan / 验证计划

- 核对文档一致性、链接、时间、空白和过时表述；开发检查以实际执行结果登记，不以文档检查宣称功能通过。
- 架构验证沿实际调用链检查共享业务、依赖方向及资源所有权，不用 CLI 复制规则的测试证明一致性。规则/用例使用既有端口替身直接测试，CLI 协议及真实 socket/SSH 按实际边界验证。
- Python CLI 清理检查旧包、专属依赖和测试已移除，Taskfile/CI/活文档不依赖 uv/旧 CLI。单文件脚本不引入第三方 Python 依赖；通过现有真实打包及 ZIP 内容、权限和版本校验验证接续，不恢复旧测试。
- 修改 Go 后执行 `task check`、`go test ./cmd/... ./internal/...`；核心反馈使用 `task test:go:unit -- -count=1`，不替代固定检查。
- 能力验证聚焦配置/数据库一致、权限及目标变化、指纹、跨平台 argv、TTY，以及 forward 就绪、重复/冲突启动、真实/失效状态、断线退出、多连接/半关闭和停止清理；覆盖数据库/PAT 不可用时仍可 status/stop。不穷举未承诺行为。
- Web 生成代码删除后执行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`；确认浏览器终端、PAT/MCP 仍保留。
- 不启动用户开发服务器、不运行实际数据库迁移、不处理已有后台 SSH 进程；必要组件测试只管理自己的夹具和子进程，不自动生成整体验证报告。

## Implementation delivery / 实现交付与开发检查

本会话按 HEAD `4e83dda1639cd7995f2eaa1d1031779672f64058` 的初始基线顺序实施，未提交 Git；当前工作树即交付 diff。初始两份未跟踪过程文档已沿用并更新。

- 首先移除完整 Python CLI、专属测试、uv 依赖与入口，抽出 [release.py](../../scripts/release.py)，接续 Taskfile/两条 CI。删除旧 CLI HTTP 路由、handler、proto/生成代码及无生产调用方的 HTTP PAT helper，保留浏览器终端和 PAT 业务。
- [Go CLI](../../internal/api/cli/command.go) 实现六个命令，[bootstrap](../../internal/bootstrap/cli.go) 复用共享配置、数据库、PAT 认证和环境业务。list 只列 SSH；exec 保留字面 argv、二进制输入输出和退出状态，stdin 与交互选择等待均可取消。
- [ForwardService](../../internal/application/environment/usecase/forward.go) 组织身份、目标、同连接指纹、就绪和持续资格复核；[SSH 转发](../../internal/infrastructure/runner/ssh/forward.go) 处理本地监听、多连接、半关闭、心跳与断线清理。
- [进程适配](../../internal/infrastructure/runner/forwardprocess/manager.go) 实现用户私有目录、端口文件锁、独立子进程、两阶段启动、PID/创建时间/可执行身份/运行标识核验和本地控制。控制凭据单独随机生成，只保存在私有运行记录中，不进入 argv 或公开状态；错误凭据不能停止进程。
- 发布包服务脚本只匹配默认 serve、serve 和 worker，避免停止服务时误停 forward。活文档已换成 Go CLI 使用方式；旧过程记录标明已被取代。

以下为首次实施检查；三平台打包对应端口参数调整前的实现，本次调整另行登记开发检查。

| 开发检查 | 实际结果 |
| --- | --- |
| Go 和 Web buf 重新生成 | 通过；只删除旧 CLI 专属生成代码，无其他生成漂移 |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 通过 |
| `task check` | 最终源码通过，golangci-lint 为 0 issues |
| `go test ./cmd/... ./internal/...` | 最终源码通过；外部场景仍按既有条件显式启用 |
| `task test:go:unit -- -count=1` | 通过；核心规则与用例未使用测试结果缓存 |
| 应用、CLI、SSH、forwardprocess 组件检查 | 通过；覆盖资格撤销、指纹先于监听、取消、真实并发启停、冲突/外部监听保护、失效 PID、私有控制认证、多连接、半关闭和断线释放 |
| `python scripts/release.py` | 最终源码 Windows/Linux/macOS amd64 构建成功；ZIP 内容、VERSION、二进制格式、Unix 执行权限与 CRC 校验通过，产物在 `dist/` |
| 当前 development 配置的 CLI 实测 | `cli env list --json` 退出 0，返回 6 个 SSH 环境；`forward status --json` 退出 0；数据库临时设为不可达且 PAT 无效时状态查询仍成功，随后恢复进程环境 |
| 发布包脚本检查 | Windows status 及默认/serve/worker/CLI/MCP 命令匹配夹具通过；POSIX shell 语法通过；未执行服务 stop/restart |

实际增量与计划一致。额外修正服务脚本的进程筛选是为保持新 forward 的独立生命周期；孤立 HTTP PAT helper 清理随旧协议删除，未删除共享认证业务。未修改配置加载器、用户配置/密钥、迁移或实际业务数据，未启动或停止用户开发服务器，也未接管旧 SSH 进程。

未运行：真实受管远端 SSH 命令/端口联调、物理 TTY 交互，以及 Linux/macOS 后台进程运行矩阵；三平台构建成功不替代这些运行验证。本次完成 Implementation 与必要开发检查，不将其标为整体 Verification 已接受。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- cwd 与环境决定加载配置，调用者从正确位置启动；读取失败退出是已接受行为，不作为路径改造或打包工作的阻塞项。
- 项目成员授权和 SSH 账号分别复用各自现有业务；能读取数据库不代表可以绕过成员权限。
- 后台转发只管理本机受管进程，不把本地记录当作业务数据库或全局运行状态，不接管旧 Python 隧道。
- Python CLI 清理到 Go CLI 建成之间旧功能暂不可用，需同步修正调用并以单文件脚本接续 CI。
- Windows 可执行文件锁定、子进程脱离和停止控制属于平台适配，不能只依赖 Linux signal 行为；验证覆盖实际支持的桌面平台。

## Rollback / recovery / 回滚 / 恢复

按实际增量和基线恢复代码，保留用户其他工作区改动、配置、数据及后台进程，不整体还原仓库。后台转发先用自身 stop 命令关闭后再替换 Windows 二进制。

## User review notes / 用户审查记录

- 已确认：Go CLI 直接 SSH，与 WebAPI 共用配置、数据和业务；用户调用 SpecFlow 记录任务。
- 命令确认：“采纳，不过 pomelo-orbit environment 改为 pomelo-orbit cli env”。
- 配置确认：当前不需 --work-dir，按现有方式读取配置，读取失败退出可接受，撤回新增显式根目录方案。
- 范围确认：只考虑本地 CLI 通过 SSH 操作远程环境，沿用密钥和身份，忽略 local。
- 清理顺序：“python 相关实现全部移除，而且上来就要做这件事情”。首先清理 Python CLI，再建设 Go CLI。
- 转发范围：本地 -L，只到 SSH 目标主机自身端口，不管理该主机访问其他主机；当前只需端口转发，规划 forward 状态、启动和停止。
- 打包补充：允许 scripts 中最小依赖、单文件 release.py，不使用完整 uv 项目；据此以标准库脚本接续既有三平台打包，是此前 Python 清理要求的明确例外。
- 前序评估：用户要求列出环境管理命令和旧 Python HTTP 配置，重申共用配置与业务、SSH 身份、不连接 local；据此形成六个动作及 forward 生命周期方案。当时保留的 Q01 已由下一条实施授权收敛。
- 用户：“开始实施”。据此接受完整计划及 Q01 建议，本会话顺序完成实现和必要开发检查。

### 端口参数调整

用户要求 start 使用 `-p/--port 15432:5432`，status/stop 使用 `-p/--port 15432`。直接调整 CLI 参数适配及用法，保留 `forward/status` 标准拼写和启动 `--project` 必填约定；不改共享应用输入、后台进程协议或 SSH 转发业务。

- `go test -count=1 ./internal/api/cli` 通过，覆盖短/长参数映射、无效端口对及 status/stop 的本地进程适配。
- `task check` 通过，golangci-lint 为 0 issues；`go test ./cmd/... ./internal/...` 通过。
- 已重建 `bin/pomelo-orbit.exe`，三个动作的帮助输出均显示 `-p/--port`；development 配置下，`status -p 15432 --json` 与 `status --port 15432 --json` 均退出 0，返回空转发列表。
- `git diff --check` 通过。本轮未重建三平台 ZIP，未启动或停止真实转发。
