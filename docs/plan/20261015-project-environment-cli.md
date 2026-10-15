# Plan / 计划：通过项目别名连接环境并执行命令
最后修改时间: 2026-10-15 23:00:14

## Review status / 审查状态

Accepted

用户“开始实现”接受方案；Implementation 已完成，必要开发检查通过。用户于 2026-10-15 确认“实测可用”，[验证记录](../verification/20261015-project-environment-cli.md)已完成；Windows 非交互链路实测通过，跨平台物理交互终端矩阵仍保留为未验证项。

## Intent basis / 意图依据

- [已接受意图](../intent/20261015-project-environment-cli.md)，验收 A01—A10。
- 用户要求脚本读取环境变量中的 PAT，以项目标识连接已有环境，提供交互式和非交互式模式及大模型调用方式；明确不需要 SCP。
- Plan 授权为“开始计划”；后续“开始实现”接受本方案并授权实施与必要开发检查。工作基线为 `00fcbc99`，实施开始时只有本功能尚未跟踪的意图与计划文档。
- 以 SSH Environment 为范围；复用服务端目标、用户、凭据和主机指纹，不创建另一套 SSH 配置，不开放 local 执行或远程 MCP。

## Technical decisions / 技术设计与决策

### 1. 入口、配置与项目定位

在已有 `pomelo-orbit-cli` 下新增 `environment` 命令组，与读取 `SSH_HOST/SSH_USER` 的独立 `remote` 工具分开。

```bash
pomelo-orbit-cli environment list
pomelo-orbit-cli environment list --json
pomelo-orbit-cli environment ssh
pomelo-orbit-cli environment ssh --project production
pomelo-orbit-cli environment exec --project production -- docker ps
pomelo-orbit-cli environment exec --project production --timeout 30 --json -- docker ps
pomelo-orbit-cli environment exec --project production -- sh -lc 'df -h; free -m'
```

- `--project` 只接受精确的 Project code；不混用名称、id、模糊匹配或持久化的上次选择。code 在服务端解析并重新检查成员资格，客户端项目列表不构成授权。
- `list` 列出全部当前成员可见项目，包含 `id/name/code/is_active/environment_id/target_type/platform/host/port/username` 摘要，按 code 排序；未配置环境的字段为空并标为 `unconfigured`。保持既有成员可访问范围，不额外引入 Project active 门槛。
- `ssh` 一律要求 stdin/stdout 为 TTY；未指定项目时列出可见 SSH 目标并按编号选择，没有目标则失败，不能自动选择唯一目标。指定项目时不再选择。local 和未配置环境不可连接，返回明确原因。
- `exec` 必须指定项目与至少一个命令参数，参数使用 Click 的 `UNPROCESSED` 集合，`--` 后的 `--help` 等内容属于远端参数。永不进入项目选择、密码输入或确认提示。
- 新增客户端配置 `POMELO_ORBIT_CLI__BASE_URL`，表示 Orbit 服务根地址，例如 `https://orbit.example.com`，允许部署路径前缀；统一在此前缀后追加 `/api/cli/...`，不把它解释为 API endpoint。拒绝 userinfo、query、fragment 及非 HTTP(S) scheme。
- PAT 继续使用 `POMELO_ORBIT_MCP__ACCESS_TOKEN`，不增加 token 别名或命令行 token 参数。
- 扩展 scripts 的集中 typed settings，加入 `ClientSettings`，只在 `environment` 命令执行前集中校验两项必填值；其他已有命令不要求 PAT。PAT 按现有前缀及 32 字节 base64url 随机部分校验格式，实际有效性由服务端认证。
- 配置沿用 scripts 的 `configs/config.yaml < configs/config.<env>.yaml < .env[.<env>] < OS env` 顺序，环境选择仍来自启动 OS env；只把所需字段纳入白名单。CLI 不读取服务端的 `overwrite.env`，文档明确与 Go 服务端配置的区别，不在命令模块直接读环境变量。
- 根命令及子命令 `--help`、`--version` 继续无需凭据；远端负载中包含 `--help` 时仍按现有延迟初始化机制正确加载配置并执行。

### 2. 专用 PAT HTTP 边界

仅为下列路由增加 PAT 认证，普通 Web JWT API 与 stdio MCP 的现有行为保持不变：

| 方法与路由 | 输入 | 行为 |
| --- | --- | --- |
| `GET /api/cli/project` | `Authorization: Bearer <PAT>` | 返回成员可见项目及环境摘要 |
| `GET /api/cli/environment/terminal?project_code=...` | PAT header；WebSocket subprotocol `orbit-terminal-v1` | 校验、获取环境会话资格后升级为交互终端 |
| `POST /api/cli/environment/exec` | PAT header；下文定义的 JSON body | 执行一次无 PTY 命令并返回有界结果 |

- `security.Authenticator` 增加独立的 `CurrentPATUser` 入口，复用 `AuthService.AuthenticateMCPAccessToken`；不改 `CurrentUser` 的 JWT 语义，不按 token 前缀自动切换整个 API 的认证方式。
- 新建 `cli` HTTP handler，依赖窄应用接口；JWT 不能代替这里的 PAT，PAT 不能调用普通 Web API。每个 HTTP 请求或 WebSocket 升级请求均重新校验 PAT、用户存在且 enabled、未过期和未撤销。
- 项目 code 不存在返回 404，已知项目的非成员沿用现有 403。local、未配置环境或无效凭据绑定返回 400；目标修订冲突返回 409；SSH 建连、认证和容量不可用返回安全的 503；未知内部错误返回 500。
- 所有失败走现有 `transport.WriteError`，契约为 `{code,error,requestId}`，401 带 `WWW-Authenticate: Bearer`，不向响应复制底层 SSH error、命令参数或内部 cause。
- 客户端统一归一化 HTTP、网络、超时及契约错误；保留 status、code 和 requestId。非 JSON、HTML 或错误结构不符时报告 `contract_mismatch`，不直接输出代理返回的原文。
- 使用标准 TLS 证书校验，不提供跳过证书验证参数；HTTP 与 WebSocket 客户端均禁止跟随重定向，避免 PAT 被带到另一地址。不读系统代理配置，直接连接显式配置的 Orbit 地址。
- 三个 CLI 路由在请求 UoW 中间件注册之前装配，方式与现有 Dialogue 一致；网络执行及终端不得进入请求级 SQL 长事务。首次主机指纹写入继续使用现有窄条件更新。

### 3. 应用边界、连接资格与交互终端复用

- 在 `application/environment` 增加 `AccessService`，承担成员可见项目/环境摘要、按 code 解析目标及 CLI 终端资格获取。通过窄 `ProjectAccessReader` 使用已有 `ListProjectsByMember/ProjectByCode/IsProjectMember`；环境读取复用现有存储，不改 SQL 或 schema。
- 项目列表仅查询库存，不解密密钥、不建连、不读取 Probe 结果作为门槛。已有项目逐项读取环境足够完成此用途，不为此新增批量查询框架。
- 将 `TerminalService.resolveTarget` 的现有连接目标解析移为 Environment Service 内共享 helper；将成员、目标、凭据修订和指纹持续复核规则提为同包 helper，供终端与命令用例共用。保留部署 `TargetResolver.ResolveProjectTarget` 的成功 Probe 要求，不增加通用的绕过校验开关。
- `TerminalService` 新增直接按已认证 actor 和 project 获取 grant 的方法，与票据兑换共用配额领取/释放逻辑。既有 Web 入口继续兑换一次性票据；CLI 入口用 PAT 直接获取 grant，不再增加客户端票据交换。
- `AccessService` 在最终 grant 上再次确认 SSH 类型，防止两次读取之间由 SSH 改为 local 后错误启动本机 shell；失败立即释放资格。
- 将当前 HTTP handler 中的 WebSocket 读写、ready/resize/exit 控制、会话结束和日志处理提取为 `internal/api/http/terminal` 中的共享协议实现。Web 和 CLI 两个 handler 只负责各自认证与 grant 获取，随后调用同一个协议实现，不复制两个会话循环。
- 协议继续使用二进制帧传送输入/合并终端输出，文本控制帧保持现有 `ready`、`resize`、`exit`、`connection_failed`、`idle_timeout`、`access_or_target_changed`、`session_limit`；CLI 不传浏览器 `ticket.*` subprotocol，也不带 Origin。
- SSH 交互仍进入已配置用户的默认 shell/home，不自动进入工作区、容器或 WSL。收到 `ready` 后客户端发送实际尺寸，服务端尺寸限制仍为列 1—500、行 1—200。
- 保留现有终端限制：建连 15 秒、空闲 15 分钟、最长 2 小时、每用户 2 个/进程共 16 个会话，每 2 秒复核成员与目标。Probe 状态变化不影响连接。
- PAT 在升级请求时校验；已建立终端继续沿用现有成员/目标/凭据会话失效规则。撤销或到期阻止新请求，不新增持续扫描 PAT 或承诺立即终止已经建立的会话。

### 4. 非交互命令契约与执行

新增 `proto/orbit/v1/cli/cli.proto` 定义项目摘要、命令请求和结果 DTO，沿用仓库 proto JSON snake_case 及集中 writer。HTTP DTO 留在 handler，应用 DTO 和执行端口不依赖 protobuf、Gin 或 SSH SDK。

请求字段：

| 字段 | 约束 |
| --- | --- |
| `project_code` | 必填，精确目标；执行时重新授权 |
| `argv` | 非空数组，首项为可执行程序；最多 128 项，UTF-8 总量不超过 32 KiB，拒绝 NUL；不拼接成未经转义的脚本文本 |
| `workdir` | 可选，默认 Environment 的 `workspace_root`；提供时按目标平台接受绝对路径或 `~` / `~/...`，不接受相对目录 |
| `stdin` | protobuf bytes，JSON 为 base64；最多 64 KiB，缺省即 EOF |
| `timeout_seconds` | optional uint32，缺省 60，显式值 1—300；0 不表示无限 |

- HTTP 请求 body 上限为 128 KiB，在 decode 前限制，超限按项目 400 校验契约返回。使用现有严格 proto JSON decode，未知字段报错。
- 默认 argv 模式保留每个参数的边界，不自动解释管道、重定向、变量或通配符。Linux 使用现有 POSIX quote 组合调用；Windows 使用已有 PowerShell 编码传输与逐项 quote，明确将 native exit code 或 cmdlet/异常失败转为真实进程退出状态。
- 新建通用命令构造 helper，复用现有 quote/编码函数；不直接调用部署专用 `remoteCommand`，避免其中 Docker/Curl 名称改写及 argv 中 `~` 的自动展开影响通用命令。`workdir` 的 `~` 按既有路径规则展开；argv 内 `~`、`$VAR` 等保持字面值。
- 需要 shell 功能时由调用者显式指定解释器：Linux `-- sh -lc '...'`，Windows `-- powershell.exe -NoProfile -NonInteractive -Command '...'`。不新增 shell 字符串与 argv 两套接口。
- workdir 不存在时不创建、不切换为其他目录，返回远端执行失败；它只是起始目录，命令可以按 SSH 用户权限访问其他位置。
- CLI 仅在 `--stdin` 时读取本地 stdin，读取上限 64 KiB，超限在请求前失败；未指定时不读取父进程 stdin，避免模型调用挂起。

执行用例采用独立 `CommandService`，通过 `environment/port.CommandRunner` 打开一次无 PTY SSH 连接；不扩大 deployment Runtime 接口，也不把 SSH SDK 类型暴露到应用。

1. 校验输入和成员资格，解析当前 SSH 环境及受管密钥，领取命令配额（每用户 2 个/进程共 16 个，与交互终端分别计数）。
2. `CommandRunner.Open` 只建立已认证 SSH 连接，返回具有 `HostKeyFingerprint/Run/Close` 能力的 session，此时不发出命令。
3. 使用当前连接观察到的指纹，复用 `recordHostKey` 完成首次固定；已有指纹严格匹配。随后再次校验成员、目标和凭据修订，再允许执行。指纹提交冲突时关闭连接，不运行命令。
4. `Run` 在同一连接创建无 PTY session，分别消费 stdout/stderr，输入完成后发送 EOF。`ssh.ExitError` 属于真实远端退出状态，不包装成平台错误；缺失退出状态或连接中断不能当作 exit 0。
5. 应用拥有总超时、输出预算和每 2 秒一次的连接资格复核；任一失败取消执行并关闭拥有的 SSH session/client。总超时从用例受理开始包含解析和建连，建连额外不超过 15 秒。收尾最多等待 5 秒，不无限等待远端。
6. 返回结果、记录元信息并在所有分支释放配额。调用方断开时取消 request context 并收尾；不自动重试或后台排队。

结果包含 `project_id/project_code/environment_id/target_revision/state/exit_code/stdout/stderr/duration_ms/output_truncated`。`exit_code` 为 optional int32，仅确认远端退出时存在；`stdout/stderr` 使用 protobuf bytes，在 JSON 传输中为 base64，避免丢失原始字节。

- `state=exited` 表示获得真实退出状态，exit 0 才表示成功；非零退出也以 HTTP 200 返回真实命令结果。
- 命令已发出后，超时、超出输出预算、成员/目标失效、SSH 中断分别返回 `timed_out/output_limit/access_or_target_changed/transport_lost`，exit_code 缺省，不能伪造远端已退出或已清理干净。它们是明确的未完成执行结果，CLI 一律非零退出。
- 建连或发命令前失败使用前述标准 HTTP error，不返回假的命令结果。客户端在已发送请求后失联则报告结果未知，不推断命令未执行。
- stdout/stderr 共用 1 MiB 原始字节预算，由并发安全的有界 writer 保存各自前缀；超限立即取消并标明 `output_truncated=true/state=output_limit`，不继续无限收集或静默丢弃后声称成功。
- 总超时或取消是结束本次等待、关闭 SSH 连接，不保证任意远端派生进程已物理退出；不新增远端 agent、进程树管理或强制提权。

### 5. Python 客户端与输出规则

- HTTP 使用标准库 `urllib.request` 的统一 client：禁用代理和重定向，设置超时，解码 DTO/base64，归一化错误。项目列表超时 15 秒，exec 请求等待为指定总超时加 20 秒，以覆盖结果传输和有界收尾。
- 交互使用 `websockets==15.0.1` 的 sync client，PAT 放 `additional_headers`，subprotocol 仅 `orbit-terminal-v1`；`proxy=None`、关闭压缩、建连 15 秒、关闭 5 秒、帧上限 64 KiB、ping interval/timeout 均为 20 秒。此 client 不自动跟随重定向。
- 跨平台终端输入/输出使用 `prompt-toolkit==3.0.52` 的 `create_input/raw_mode/attach/read_keys` 和 `create_output/write_raw/flush`，不自行手写 Windows Console 模式管理。Windows 使用支持 VT 的现代控制台；不满足 TTY/VT 条件时明确失败。
- 在 asyncio 生命周期内绑定输入，WebSocket recv/send 通过有界任务和 `asyncio.to_thread` 调用 sync client；关闭时先关闭 WebSocket，使阻塞读写返回，再回收任务与输入绑定，并用 context manager 恢复本地终端。
- 输入按库的原始 data 发送；空 data 的功能键用库自带 `REVERSE_ANSI_SEQUENCES` 映射；保留 bracketed paste 包裹，按库 flush 机制处理单独 Escape。终端输出用增量 UTF-8 decoder，避免多字节字符跨帧损坏。
- Ctrl+C/Ctrl+D 转发远端，Ctrl+] 作为本地断开键；连接期间每 250ms 比较终端尺寸，变化时发 resize，不为每个平台新增一套 resize 协议。输入队列上限 64 条，溢出明确关闭，不能静默丢控制键。
- `exec` 默认将 base64 解码的字节分别写到 stdout/stderr 的 binary buffer，不夹杂状态标题、项目列表或日志。必要的平台诊断写 stderr。
- `list --json` 输出一个 JSON 对象；`exec --json` 输出一个包含目标、state、exit_code、stdout/stderr 文本、耗时和 requestId 的 JSON 对象，不再混写远端 stderr。文本按 UTF-8 decode，替换无效字节时增加 `output_decoding_replaced=true`，原始 API 仍保留字节。
- CLI exit：真实远端 0—255 原样传递；超范围状态保留在 JSON 并以 255 退出；配置/用法错误 2，平台/连接错误或状态未知 255，超时 124，输出超限或执行中权限/目标变化 125，本地中断 130。远端可能使用相同数值，需严格区分时读取 JSON state，不只看退出码。
- root `--verbose` 不开启第三方 WebSocket header/frame debug 日志，不打印 PAT、命令参数、stdin 或输出内容；不增加自动登录、认证刷新或执行重试。
- 新依赖只在对应命令运行时导入；更新 `scripts/pyproject.toml` 和 `uv.lock`，不升级无关依赖。

### 6. 日志、文档与现有行为

- HTTP body 日志对两个 CLI 环境路径禁用请求和响应正文，避免 argv、stdin、stdout/stderr 进入常规日志；只记录 method/path/status/requestId 等元信息。Authorization header 不进入日志。
- 连接/命令应用日志记录 actor、Project、Environment、target_revision、凭据修订、耗时、执行 state 和已知 exit code；不记录私钥、PAT、命令正文或完整流。底层错误进入受控 cause 时同样不拼接命令/输出。
- 新增 `docs/guides/project-environment-cli.md`，作为人和大模型共用的调用契约；更新 scripts README、`.env.example`、docs INDEX、产品模型、后端架构与客户端调用指南。明确 PAT 现在也可用于这些专用 CLI 路由，修正“只能用于本地 stdio”的绝对表述，但继续说明它不能当作普通 Web JWT。
- 大模型通过该指南及 CLI help 调用，不新增 MCP 工具或项目 skill。说明每次明确项目、无默认重试、参数与显式 shell 示例，以及命令可能修改远端状态；诊断请求不隐含修复、重启或删除授权。
- Web 终端、CI/CD、Probe、stdio MCP、现有 `remote` 工具及已执行迁移不改变产品行为。proto 的标准生成会增加 Go 与 TypeScript DTO，不修改业务 Web UI。

## Implementation steps / 实施步骤

1. **协议与认证**（A01、A02、A05）：添加 CLI proto、生成 DTO；实现 PAT 专用认证和路由骨架、客户端 typed config 与错误类型。完成条件为 HTTP 权限与错误协议、配置优先级和缺失配置行为可直接验证，普通 JWT/PAT 边界保持独立。
2. **环境访问与交互入口**（A02、A03、A05—A07）：实现项目摘要与 code 解析，提取连接规则和配额领取，提取共用终端传输，接通 PAT 升级入口。完成条件为现有 Web 协议测试继续通过，新入口能按精确 code 获取 SSH 会话，不依赖 Probe/Gateway，也不能误走 local。
3. **一次性命令执行**（A04—A08）：增加窄 CommandRunner 与 SSH adapter，实现同连接指纹确认、真实 exit code、字节输出、输入 EOF、限额、资格复核、取消和清理。完成条件为成功、远端非零及真实失败边界均有直接应用/适配器证据。
4. **CLI 命令与终端客户端**（A01—A04、A07—A09）：加入所定版本依赖，完成 list/ssh/exec、TTY 选择、控制键与 resize、stdin/JSON/退出码及错误归一化。完成条件为 help 不依赖 PAT，非交互调用无 prompt，输出可直接被模型消费且无认证泄露。
5. **文档、开发检查与交付审查**（A09、A10）：同步活文档和大模型示例，运行下节固定检查与必要协议/实机验证。将实际 diff、检查证据、未运行项及环境风险写回本计划的交付记录；实施完成后交付摘要，按授权进入整体 Verification。

## Files to change / 修改范围

| 范围 | 主要文件与职责 |
| --- | --- |
| 协议 | 新增 `proto/orbit/v1/cli/cli.proto`；标准生成 `internal/gen/proto/orbit/v1/cli/cli.pb.go`、`web/src/gen/proto/orbit/v1/cli/cli.ts` |
| HTTP | 新增 `internal/api/http/handler/cli/`、`routes/cli.go`、`terminal/`；修改 `routes/routes.go`、`security/authz.go`、`handler/environment/terminal.go` 及相关 handler 装配；调整 `middleware/logging.go` 的正文排除规则 |
| 应用 | `internal/application/environment/{dto,port,usecase}`：AccessService、CommandService、窄项目 reader、命令 runner、共享连接校验；调整 TerminalService 直接领取与共用配额；不扩大 deployment Runtime |
| 基础设施与装配 | `internal/infrastructure/runner/ssh/command.go` 及测试；复用既有 handshake/quote；`internal/bootstrap/http.go` 注入能力、注册无外层长事务路由；现有 auth PAT 仅更新用途说明，不改 token 存储 |
| CLI | 新增 `commands/environment.py`、`environment_client.py`、`environment_terminal.py`；修改 `cli.py`、`settings.py`、`context.py` 中所需 typed config 装配、`errors.py` 与相应 tests |
| 依赖 | `scripts/pyproject.toml`、`scripts/uv.lock`；不新增 Go SSH/PTY 库 |
| 文档 | `scripts/README.md`、根 `.env.example`、新调用指南、`docs/INDEX.md`、`docs/product/cd-model.md`、`docs/architecture/backend.md`、`docs/guides/client-cloud-invocation.md`、必要的 MCP 认证表述 |

## Tasks and coordination / 子任务与协调

不适用。不建立独立 task 或 DAG，不启动子 agent；按上述顺序完成一次完整交付。当前计划是实施与进展记录的唯一位置，共享协议、应用装配与客户端变更由同一实施会话维护。

后续会话接收时读取本意图和计划，核对用户阶段授权、Git 基线与实际工作树，在本计划的实施记录中登记实际开始基线、进展、检查和交付版本。方案为 Draft 时不得把“已有开发步骤”视为实施授权。

## Verification plan / 检查与验证安排

| 验收 | 最小有效证据 |
| --- | --- |
| A01 | settings 单测验证 dotenv/OS 优先级、PAT/URL 校验、其他命令不要求 PAT；help 与远端 `--help` 参数用例；检查日志不含 token |
| A02、A07 | AccessService 端口替身验证项目摘要、未配置/local 标识和精确 code 授权；CLI list/选择测试；不存在/非成员/错误目标不会运行 runner |
| A03 | 共享终端协议测试覆盖 PAT 升级与 Web 票据两入口、ready/resize/exit 与资源释放；Python 用输入替身验证控制键、UTF-8 分帧及退出恢复；Linux/Windows 目标实机终端证据 |
| A04 | 真实应用用例配 runner 替身，验证 stdout/stderr、非零/未知 exit、stdin EOF、argv/显式 shell；SSH adapter 用进程内 SSH server 验证真实协议与退出状态，不伪称单元测试 |
| A05、A06 | PAT 缺失/过期/撤销/禁用、JWT/PAT 入口分离；成员与目标/凭据变更；首次指纹同连接记录且冲突前不发命令；Probe 失败、无 Gateway/Service 仍可执行 |
| A08 | 有界 writer、输出超限、超时与配额释放的直接用例；真实 SSH 组件验证断开取消、流排空与关闭；body 日志测试证明命令/输入/输出不被捕获；不使用进程句柄总量推断资源释放 |
| A09 | 指南中的命令与实际 help、环境变量名、JSON/exit 规则逐项核对；示例含 Linux/Windows shell，未出现 SCP、私钥导出或自动修复 |
| A10 | 固定检查通过，生成结果可重现，实机验证和未验证项有明确记录，所有证据对应最终 diff |

实施中的固定检查与生成命令：

```powershell
task proto
task test:go:unit -- -count=1
task check
go test ./cmd/... ./internal/...
yarn --cwd web lint:fix
yarn --cwd web typecheck
uv --directory scripts run --locked --no-sync ruff format --check .
uv --directory scripts run --locked --no-sync ruff check .
uv --directory scripts run --locked --no-sync mypy
uv --directory scripts run --locked --no-sync pytest
```

- scripts 命令等价于 `make -C scripts check/test`；当前 Windows 环境没有 make，直接使用其 Makefile 中的 uv 入口。新增锁定依赖后使用已有 `uv sync --all-groups --locked` 同步，再执行检查；不安装无关工具。
- proto 生成采用现有 `task proto`，检查并排除无关生成差异；因为标准生成会改动 web DTO，执行前端固定 lint/typecheck，不增加前端业务测试。
- 开发期间按改动运行相关 Go/Python 包测试；完整固定检查通过后不无故重复。涉及真实时间的规则用端口替身和 `testing/synctest`，协议/真实进程验证保留实际边界。
- 实机目标为用户已有且可授权访问的 Linux SSH 与 Windows native OpenSSH 环境；只读 smoke 命令取主机/用户、工作目录、stdout/stderr 和非零退出结果，交互核对 Ctrl+C、尺寸变化及退出恢复。不主动启动或重启开发服务器，也不在 Plan 阶段发起连接。
- 真实 SSH 测试遵循仓库现有显式启用条件。目标或 PAT 不可用时，明确记录对应验收未验证及解除条件；不能用替身通过替代实机终端结论。

## Risks and recovery / 风险与恢复

- **命令副作用与结果未知**：这是通用 SSH 权限，客户端断线不保证没有执行。返回 state 并禁止重试；指南明确诊断授权边界，不增加命令黑名单或伪只读承诺。
- **跨平台终端**：已从指定版本源代码核对 Windows/Unix 输入、raw mode、VT 输出和键映射接口；当前未安装新库或做控制台实测。实施验证 Windows VT 与 Linux TTY，失败时修复客户端适配，不放宽交互验收。
- **取消与输出**：新命令端口必须实际关闭自己的 SSH 连接，不能套用现有无限 bytes.Buffer Query。总超时与取消结果只承诺本次等待和连接收尾，不承诺派生进程树退出。
- **长请求代理限制**：exec 是有界结果请求，默认 60 秒、最大 300 秒；调用方需让 Orbit 前方代理等待覆盖所需执行时间。代理断连按结果未知处理，不通过自动重试补偿；本次不新增异步任务/恢复机制。
- **凭据与日志**：API body 捕获与第三方 DEBUG 是两个实际泄露入口，实施必须同时排除；不存在自动脱敏任意命令输出的承诺。
- **共享终端回归**：先提取并保留 Web 协议测试，再接 PAT 入口；不改 Web 票据、local 行为或 Probe 门槛。SSH 命令用独立构造 helper，避免影响部署专用命令规则。
- **恢复**：不变更数据库 schema 或已执行迁移，无数据转换。按本功能 diff 撤回新增 CLI 路由、命令、依赖及共享终端提取并恢复生成 DTO即可；PAT 数据、Environment 凭据与业务运行态保持原存储。已在远端发生的命令副作用不属于代码回滚可撤销范围。

## Implementation record / 实施记录

- 状态：Implementation 已完成，必要开发检查通过；后续 Verification 结果见[验证记录](../verification/20261015-project-environment-cli.md)。实施交付为基线 `00fcbc99` 上的未提交工作树，实施会话未暂存或提交代码；验证接收时功能变更已在暂存区，本轮不改变其暂存状态。
- 授权与基线：用户“开始实现”接受本计划；开始时只有本功能 intent/plan 未跟踪文档。按单会话顺序实现，未启动子 agent 或开发服务器，未执行远端业务命令。
- A01、A02、A07：新增集中 ClientSettings 与 environment list/ssh/exec；现有 PAT 与新 BASE_URL 的 dotenv/OS 优先级、help 无配置、精确 code 定位、成员可见 inactive/local/未配置摘要、exec 必填参数及无隐式选择均已实现。其他 remote 工具沿用原入口。
- A03：Web 票据与 CLI PAT 共用 `internal/api/http/terminal` 协议；直接获取 grant 共用终端配额。Python 使用锁定的 websockets / prompt-toolkit，实现 raw 输入、控制键、Ctrl+]、resize、UTF-8 增量输出与关闭恢复；大段粘贴按 16 KiB 分帧，64 条队列溢出明确断开。
- A04—A08：CommandService 与独立 SSH CommandRunner 已落地，包含同连接首次主机指纹固定、命令前资格复核、无 PTY 执行、stdin EOF、分离字节输出、真实退出状态、未知状态、总超时、每 2 秒资格复核、1 MiB 共享输出预算、独立配额及 5 秒清理预算。CLI 路由注册在请求 UoW 之前；PAT/JWT 分离，终端/命令正文不进入 HTTP 日志。部署 Probe 门槛保持原规则。
- Windows 实施细化：沿用 PowerShell 编码和路径/字符串 quote；native 程序使用 ProcessStartInfo + 显式 Windows CRT argv 编码，继承 SSH 标准句柄，避免 Windows PowerShell native 参数绑定丢失空参数或内嵌引号。cmdlet/异常失败仍映射为非零退出。此为既定字面 argv/字节流承诺的具体实现，不增加第二种命令接口。
- A09：新增 [调用指南](../guides/project-environment-cli.md)，同步 scripts README、环境变量模板、文档索引、产品模型、后端架构与 MCP/客户端指南。示例覆盖项目发现、交互、模型 JSON 调用及显式 Linux/Windows shell；标明无自动重试与诊断授权边界。
- 依赖：新增 `websockets==15.0.1`、`prompt-toolkit==3.0.52`，更新 uv.lock 并执行 `uv --directory scripts sync --all-groups --locked`；未升级无关依赖。uv 因跨磁盘缓存使用普通文件复制，未影响检查。

开发证据（对应最终实现）：

| 检查 | 结果 |
| --- | --- |
| `task proto` | 通过；再次生成并比较新增 Go/TS DTO 的 SHA-256，完全一致；buf 提示没有依赖可更新，不影响生成 |
| `task test:go:unit -- -count=1` | 通过；包含新 Access/Command 用例与虚拟时间下的超时/撤权验证 |
| `go test ./cmd/... ./internal/...` | 通过；包含原 Web 终端回归、PAT 入口、严格 body/字节协议、日志正文排除及 SSH 组件验证 |
| `task check` | 通过；Web types/lint/format、Go formatter/linter，0 issues |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 均通过；只增加标准生成 TS DTO，无业务前端改动 |
| scripts Ruff format check / Ruff check / mypy | 均通过；使用计划中的 locked/no-sync 入口 |
| scripts pytest | 85 passed；配置、help/argv、stdin、stdout/stderr/JSON/exit、HTTP 契约、输入/resize/UTF-8/溢出与退出恢复 |
| 本地 SSH 组件 | 通过；真实 SSH channel 验证 Open 不发送命令、stdin EOF、独立字节流、远端 exit 7、缺失退出状态和取消；不是远端实机验收 |
| Windows 本地进程组件 | 通过；实际 Windows PowerShell 启动原生 helper，验证空参数、引号、末尾反斜线、Unicode、字面变量/通配符、二进制 stdin/stdout/stderr 和 exit 7 |
| CLI help、文档引用、`git diff --check` | 通过 |

交付审查与限制：

- 生产调用链已核对：专用 PAT → 成员/code → 受管目标/密钥 → 当前连接主机指纹确认 → 命令；local 不可进入 CLI，库存不解密凭据、不要求 Probe/Gateway；应用与 SSH/HTTP/protobuf 的边界保持分开。
- SSH 协议与 Windows 本地进程证据不等同于真实 Linux/Windows OpenSSH 终端联调。本会话未选定并启用实机目标，未运行真实远端命令或物理 TTY 的 raw mode/resize/控制键联调；A03 的对应实机证据及 A10 的整体验收仍待 Verification。解除条件为可访问的已授权 Linux/Windows 目标、运行新代码的 Orbit 服务与物理 TTY；开发服务器继续由用户管理。
- Python 检查使用当前 Windows Python 3.14.2；声明的 Python 3.12 下限已核对锁定库 API/支持范围，未在实际 Python 3.12 或 Linux TTY 下运行。
- 反向代理长请求等待仍需覆盖执行时长。超时/断线只关闭连接和结束等待，不承诺远端派生进程已终止；结果未知不自动重试。上述边界已写入指南。
- 没有 schema/迁移、MCP 工具、SCP、固定诊断库或业务 UI 范围扩展；未执行发布、提交或服务器管理。

## User review notes / 用户审查记录

- 用户“开始计划”接受前置意图并授权本轮 Plan；没有授权实施、Git 提交、发布或开发服务器管理。
- 决策已覆盖命令名、配置与 PAT 来源、专用 API、共享终端协议、argv/stdin/workdir、退出状态、输出与取消边界、日志、依赖及检查方式；没有把技术选择留给后续调查 task。
- 意图目标与范围未扩大，仍为项目别名下的交互 SSH 与非交互命令。复用现有终端处理和环境凭据，不新增文件传输、固定诊断工具库或远程 MCP。
- 用户“开始实现”接受本计划并授权实施及必要开发检查；不包含提交、发布或开发服务器管理。
- 用户后续授权诊断 `guohua`，已通过 CLI 完成真实 Windows SSH、stdin 脚本、错误输出及超时结果检查；这些实测补充在验证记录，前述“未执行远端命令”仅描述 Implementation 交付时点。
- 用户于 2026-10-15 明确反馈“实测可用，完成验证文档”，据此完成验证记录并接受实际使用结果；未提供的 Linux/物理交互终端细项不推定为通过。本轮未修改实现、暂存、提交、部署或启动开发服务器。
