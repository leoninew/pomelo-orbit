# 通过项目别名连接远程环境
最后修改时间: 2026-10-15 21:18:08

`pomelo-orbit-cli environment` 使用 Project **code** 作为 SSH alias。Orbit 服务端按当前用户的项目成员资格读取已保存的 SSH 目标与受管私钥，再代理交互终端或执行一次命令。客户端只需要 Orbit 地址与 PAT，无需私钥、Docker、数据库访问或 SCP。

## 配置与启动

在已登录的 Web 控制台“系统管理 / 访问令牌”创建 PAT，存入父进程环境或 gitignored 的项目根 `.env` / `.env.<env>`：

```dotenv
POMELO_ORBIT_CLI__BASE_URL=https://orbit.example.com
POMELO_ORBIT_MCP__ACCESS_TOKEN=orbit_mcp_pat_...
```

`BASE_URL` 是服务根地址，可包含部署路径前缀，例如 `https://host/orbit`；客户端追加 `/api/cli/...`。使用标准 TLS 证书校验，直接连接该地址，不读取系统代理，不跟随重定向。不要把 PAT 写到命令参数、可提交文件或模型提示词。

配置顺序为 `configs/config.yaml < configs/config.<env>.yaml < .env[.<env>] < OS env`；profile 由启动时 `POMELO_ORBIT_APP__ENV` 选择。CLI **不读取 Go 服务端的 `overwrite.env`**。其他 CLI 命令和 `--help` 不要求 PAT。

在仓库根目录运行（Python ≥3.12，uv）：

```bash
uv --directory scripts sync --locked
uv --directory scripts run --locked pomelo-orbit-cli environment list --json
```

以下示例省略 `uv --directory scripts run --locked` 前缀；可在 scripts 虚拟环境中直接使用 `pomelo-orbit-cli`。

## 人工交互

```bash
pomelo-orbit-cli environment list
pomelo-orbit-cli environment ssh
pomelo-orbit-cli environment ssh --project production
```

`list` 返回当前用户所有成员可见项目，包括 inactive、local 和未配置环境的项目，按 code 排序。摘要没有密钥。`ssh` 未指定项目时列出 SSH 目标并要求编号选择；指定项目则直接连接，不使用名称、id、模糊匹配或上次选择。

`ssh` 要求 stdin/stdout 都是 TTY，Windows 需要支持 VT 的现代控制台。进入 SSH 用户默认 shell/home，不自动进入 workspace、容器或 WSL。Ctrl+C / Ctrl+D 发送到远端，Ctrl+] 本地断开；尺寸变化会同步。终端建连上限 15 秒、空闲 15 分钟、总时长 2 小时；每用户最多 2 个、每服务进程共 16 个。

## 大模型与自动化调用

模型先 `environment list --json` 获取精确 code，然后每次显式传入 `--project`。`exec` 永不提示选项目、密码或确认，不分配 PTY；只有 `--stdin` 才读取本地输入，缺省发送 EOF。

```bash
pomelo-orbit-cli environment exec --project production -- docker ps
pomelo-orbit-cli environment exec --project production --timeout 30 --json -- df -h
pomelo-orbit-cli environment exec --project production --workdir /var/log -- ls -lah
pomelo-orbit-cli environment exec --project production -- sh -lc 'df -h; free -m'
pomelo-orbit-cli environment exec --project windows-prod -- powershell.exe -NoProfile -NonInteractive -Command 'Get-Process | Select-Object -First 5'
```

`--` 后每项按字面 argv 发送，包括 `--help`、空格、空参数、`$HOME`、`~`、`*`、`|` 和 `>`。需要 shell 语法时显式调用 `sh -lc`、`powershell.exe -Command` 或其他解释器，遵循调用者本地 shell 的引号规则。Windows native 程序使用标准 Windows CRT 参数规则；程序自身若使用其他解析规则，按该程序要求传参。

`--workdir` 缺省为环境保存的 workspace；显式值必须是目标平台绝对路径或 `~` / `~/...`，仅该目录展开 home。缺失目录不会被创建或替换为其他目录。它不限制命令访问 SSH 用户有权访问的其他位置。

给模型的调用约定可直接引用本文：每次确认目标 code，使用非交互 `exec --json`，检查 `state` 与 `exit_code` 后解释结果；不要自动重试结果未知的请求。诊断授权不隐含修复、重启、删除或其他写操作授权，按用户明确范围执行。命令具有所配置 SSH 用户的实际权限。

## 输出与状态

默认模式将原始 stdout/stderr 字节分别写入本地对应流，远端非零退出如实传递。平台错误和未完成状态诊断写 stderr。

`exec --json` 输出一个对象，包含 `project_id/project_code/environment_id/target_revision/state/exit_code`（仅已知时）、`stdout/stderr` 文本、`duration_ms/output_truncated/requestId`。文本按 UTF-8 解码；无效字节被替换时 `output_decoding_replaced=true`。平台错误也是单个对象，`state=platform_error` 并带 `code/error/status/requestId`；用法与配置错误仍走 CLI 用法提示。

| state | 含义 | CLI 退出码 |
| --- | --- | --- |
| `exited` | 已确认远端退出，只有 exit_code=0 成功 | 真实 0–255；范围外映射 255，JSON 保留原值 |
| `timed_out` | 已发出命令后超过总超时 | 124 |
| `output_limit` | stdout+stderr 超过 1 MiB，保留各自前缀并关闭连接 | 125 |
| `access_or_target_changed` | 执行期间成员资格、目标、凭据或指纹变化 | 125 |
| `transport_lost` | 命令已发送，但未获得可靠退出状态 | 255 |
| `platform_error` | 认证、目标解析、连接或协议失败 | 255 |
| 本地用法/配置错误、本地中断 | 请求无效或用户中断 | 2、130 |

远端也可使用 124/125/255；需要区分时读取 JSON state。命令一旦发出，超时、断线和取消均不能证明它未执行或远端派生进程已终止。连接关闭只结束本次等待和 SSH 连接，不管理远端进程树。客户端网络失败提示结果未知，禁止据此自动重试。

## 服务端协议与限制

PAT 仅用于本地 stdio MCP 和专用 CLI 路由，不能替代普通 Web API 的 JWT。每个请求 / WebSocket 升级均检查 PAT 未撤销、未过期且用户 enabled，随后检查 Project 成员资格。既有终端每 2 秒复核成员/目标/凭据；PAT 撤销阻止新请求，不承诺立即结束已建立终端。

| 路由 | 契约 |
| --- | --- |
| `GET /api/cli/project` | `{projects:[...]}`，项目及环境摘要 |
| `GET /api/cli/environment/terminal?project_code=...` | PAT Authorization header，WebSocket 子协议仅 `orbit-terminal-v1`，无 Origin；共用 Web 终端二进制数据与 ready/resize/exit 协议 |
| `POST /api/cli/environment/exec` | `project_code/argv/workdir/stdin/timeout_seconds`，protobuf JSON snake_case |

命令请求只支持 SSH，未配置/local 返回 400；未知 code 404、非成员 403、无效 PAT 401、修订冲突 409、SSH 连接/认证/容量失败 503。错误契约为 `{code,error,requestId}`，401 带 `WWW-Authenticate: Bearer`。执行结果（包括远端非零、超时和连接丢失）以 HTTP 200 返回，通过 state 判定。

argv 最多 128 项、UTF-8 共 32 KiB、不含 NUL；stdin 最多 64 KiB，HTTP JSON 中为 base64；body 最多 128 KiB。总超时缺省 60 秒，显式 1–300 秒（包含查询和建连），连接另受 15 秒上限；清理最多等 5 秒。stdout/stderr 在 API 中为 base64 bytes，以保留原始字节；客户端 JSON 则面向模型输出文本。

非交互配额每用户 2、每服务进程共 16，与终端分别计数；不排队、不重试。目标凭据与首次主机指纹复用现有存储，无需成功 Docker Probe、Gateway 或部署 Service。命令在当前连接的主机指纹确认并保存后才发出。

长请求的反向代理等待时间需覆盖执行超时与结果回传；客户端等待为指定超时加 20 秒。常规 HTTP 日志不记录这两个环境端点的请求/响应正文；应用日志只记录执行元信息，不记录 PAT、密钥、argv、stdin 或输出。输出内容自身不会自动脱敏。
