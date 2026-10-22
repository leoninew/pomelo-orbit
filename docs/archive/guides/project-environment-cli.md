# Go CLI 远程环境管理
最后修改时间: 2026-10-21 13:48:17

`pomelo-orbit cli env` 使用 Project **code** 定位已保存的 SSH 环境。本地 Go 进程与 WebAPI 共用配置、数据库和应用用例，复用项目成员权限、SSH 账号、受管密钥、主机指纹及目标修订规则，并直接建立 SSH 连接。

CLI 按 Environment ID 读取唯一的当前凭据，并复核凭据 ID/修订。SSH→local 或交接替换凭据后，旧连接与 forward 记录失效；切回 SSH 必须在 Web 显式生成新初始化命令。CLI 与服务端必须使用同一 schema 版本，000054 升级前按[环境凭据迁移指南](environment-credential-migration.md)结束连接并备份。

## 配置与开发启动

从 WebAPI 使用的同一工作目录和 profile 启动 CLI。直接调用 `config.Load()`，沿用默认 YAML、profile YAML、启动 dotenv、进程环境、`overwrite.env` 的覆盖顺序和 `POMELO_ORBIT_` 前缀；相对数据库和本地资源路径按现有规则解析。配置读取或校验失败即退出，不提供目录发现或 `--work-dir`。

Orbit 操作者使用共享 `mcp.access_token` / `POMELO_ORBIT_MCP__ACCESS_TOKEN`，通过现有 PAT 认证业务在本地解析；令牌必须有效且用户 enabled。SSH 认证使用环境绑定的账号、受管私钥和主机指纹。需要访问 WebAPI 所用数据库，并使用相同的加密配置。PAT 不替代 SSH 身份，不需要服务 URL。

在仓库根目录的 PowerShell 中：

```powershell
go build -o ./bin/air-api/pomelo-orbit.exe ./cmd/server
task cli -- env list
task cli -- env list --json
task cli -- env ssh --project production
task cli -- env exec --project production -- docker ps
```

`task cli` 自动设置 `POMELO_ORBIT_APP__ENV=development`，调用已构建的 `./bin/air-api/pomelo-orbit.exe cli`，与 Air 共用开发产物。第一个 `--` 后的子命令和参数原样透传；执行远程命令时保留第二个 `--`，用于分隔 CLI 参数与远端程序。更新 Go 代码后需重新构建二进制。

以下示例使用已构建的 `pomelo-orbit`；Windows 开发时替换为 `./bin/pomelo-orbit.exe`。Linux/macOS 使用同一命令参数。CLI 不启动 HTTP、worker 或自动迁移数据库。`--help` 不加载配置。默认运行和 `serve/worker/mcp` 入口保留。

## 命令

| 命令 | 用途 |
| --- | --- |
| `cli env list [--json]` | 查询成员可见的 SSH 环境，按 code 排序 |
| `cli env ssh [--project CODE]` | 交互 SSH；省略项目时选择编号 |
| `cli env exec --project CODE [--workdir PATH] [--timeout SECONDS] [--stdin] [--json] -- PROGRAM ARG...` | 非交互、无 PTY 的字面 argv 执行 |
| `cli env forward start --project CODE -p LOCAL:REMOTE` | 后台启动本地 TCP 转发 |
| `cli env forward status [-p PORT] [--json]` | 核对一条或全部本机受管转发 |
| `cli env forward stop -p PORT` | 停止一条受管转发 |

仅支持 SSH 环境，过滤 local 和未配置环境。库存可包含 inactive、尚未 Probe 或缺少凭据的 SSH 环境；实际连接仍检查配置、成员资格、密钥和指纹。不要求 Docker Probe 或 Gateway 就绪。

## 交互终端

```sh
pomelo-orbit cli env ssh
pomelo-orbit cli env ssh --project production
```

要求 stdin/stdout 为 TTY；Windows 需要支持 VT 的现代控制台。进入 SSH 用户默认 shell/home，不自动进入 workspace、容器或 WSL。Ctrl+C / Ctrl+D 发送到远端，Ctrl+] 本地断开；窗口尺寸变化会同步，退出恢复本地终端。

沿用终端规则：建连上限 15 秒、空闲 15 分钟、总时长 2 小时；成员权限、目标和凭据每 2 秒复核。终端及 exec 的配额为进程内配额，不是多个 CLI 进程之间的全局限制。

## 非交互执行

```sh
pomelo-orbit cli env exec --project production -- docker ps
pomelo-orbit cli env exec --project production --timeout 30 --json -- df -h
pomelo-orbit cli env exec --project production --workdir /var/log -- ls -lah
pomelo-orbit cli env exec --project production -- sh -lc 'df -h; free -m'
pomelo-orbit cli env exec --project windows-prod -- powershell.exe -NoProfile -NonInteractive -Command 'Get-Process | Select-Object -First 5'
```

`--` 后各项作为字面 argv，包括空格、空参数、`$HOME`、`~`、`*`、`|` 和 `>`。需要 shell 语法时显式调用解释器，并遵循本地 shell 的引号规则。Windows native 程序沿用标准 Windows CRT 参数编码。

`--workdir` 缺省为环境保存的 workspace；显式值必须为目标平台绝对路径或 `~` / `~/...`。不存在的目录不会被创建或替换。仅传 `--stdin` 才读取本地输入，缺省发送 EOF。

限制沿用共享命令业务：argv 最多 128 项/32 KiB、不含 NUL；stdin 最大 64 KiB；stdout/stderr 合计最多 1 MiB。执行超时默认 60 秒、允许 1–300 秒，连接另受 15 秒限制。超时从调用共享执行用例开始计算，本地读取 stdin 不计入其中，但可用 Ctrl+C 取消。

默认模式将原始 stdout/stderr 字节分写本地对应流。`--json` 输出一个对象，包含 `project_id/project_code/environment_id/target_revision/state/exit_code`（已知时）、`stdout/stderr` 文本、`duration_ms/output_truncated`；非 UTF-8 字节替换为 U+FFFD，标记 `output_decoding_replaced=true`。应用错误输出 `state=platform_error` 和 `error`。参数解析错误输出到 stderr；日志不混入 stdout。

| state / 情况 | 退出码 |
| --- | --- |
| `exited` | 远端真实 0–255，范围外映射 255 |
| `timed_out` | 124 |
| `output_limit` / `access_or_target_changed` | 125 |
| `transport_lost` / 应用、连接失败 | 255 |
| 本地用法错误 / 配置加载失败 / 本地中断 | 2 / 1 / 130 |

自动化先用 `list --json` 确认精确 code，每次显式指定 `--project`，结合 JSON `state` 和 `exit_code` 判断结果。命令一旦发出，超时、断线或取消均不能证明它未执行或远端派生进程已停止；结果未知时不得自动重放。

## 本地端口转发

```sh
pomelo-orbit cli env forward start --project production -p 15432:5432
pomelo-orbit cli env forward status
pomelo-orbit cli env forward status -p 15432 --json
pomelo-orbit cli env forward stop -p 15432
```

`-p` 与 `--port` 等价。启动传入 `本地端口:远程端口`，状态和停止只传本地端口；`status` 不传端口则列出全部。`start` 仍需显式指定 `--project`。

固定转发方向为本机 `127.0.0.1:15432` → 项目 SSH 主机 → 该主机自身 `127.0.0.1:5432`，相当于本地 `-L`。端口范围 1–65535；不在远端新增监听。需 SSH 可达、sshd 允许 TCP 转发、远端服务可在回环地址访问。

`start` 完成 SSH 认证、同连接指纹确认及本地监听后返回，一条映射一个后台子进程。相同操作者、项目、映射和目标/凭据修订重复启动幂等；端口冲突报错。支持多个 TCP 连接及双向半关闭。连接丢失或每 2 秒的资格复核失败时关闭监听、活动流和 SSH 并退出，不自动重连；没有终端的空闲或 2 小时期限。

运行记录及日志位于当前 cwd 的 `data/cli/forward/` 下，按系统用户隔离，不保存 PAT 或私钥。`status` 核对运行标识、PID、OS 进程创建信息和本地控制响应，区分 `starting/running/stopping/stopped/failed/stale/unresponsive`；`running` 不代表远端服务健康。`stop` 通过已核验的本地控制连接停止一条转发并清理记录，已退出或不存在时幂等成功，不终止非受管进程。

文本模式下，无记录的 `status/stop` 输出 `No managed forward(s) found`，停止完成输出 `Stopped managed forward on local port PORT.`；`status --json` 无记录时仍返回 `{"forwards":[]}`。无记录只表示当前 cwd、当前系统用户下没有受管记录，不代表端口空闲。如果端口仍被占用，应核对监听 PID、可执行路径及命令行；运行记录丢失的后台进程无法通过 `stop` 的控制协议停止，需确认身份后手动结束该进程。不要在转发运行期间删除记录。

`status/stop` 仍加载共享配置，但不打开业务数据库或验证 PAT。应从启动转发时相同的 cwd、同一二进制路径执行。不要用 `go run` 的临时文件或 Air 重启进程持有长期转发；Windows 重建前先停止该二进制持有的转发。

旧 Python CLI、独立 remote 工具及其 HTTP 专属路由已移除；旧后台 SSH 进程不被接管或停止。开发打包由标准库单文件脚本承接，见 [scripts 说明](../../../scripts/README.md)。
