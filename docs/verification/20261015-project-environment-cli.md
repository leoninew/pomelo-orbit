# Verification / 验证：通过项目别名连接环境并执行命令
最后修改时间: 2026-10-16 12:37:36

历史记录：本文只证明当时的 Python/HTTP 实现，不替代 [Go CLI 共享业务任务](../plan/20261016-cli-command-consolidation.md)的开发检查和验收；现行用法见 [CLI 指南](../guides/project-environment-cli.md)。

## Review status / 审查状态

Accepted

用户于 2026-10-15 明确反馈“实测可用，完成验证文档”，接受实际使用结果并授权完成验证记录。开发检查与 Windows 远程非交互链路已有证据；Linux 实机及物理交互终端矩阵未完整验证，保留为未验证项，不将用户总体反馈推定为这些细项全部通过。

## Intent alignment / 意图对齐

- 依据[已接受意图](../intent/20261015-project-environment-cli.md) A01—A10，交付 `environment list/ssh/exec`、环境变量 PAT 认证、精确 Project code 定位和模型调用指南。
- `guohua`（国华图书）的实测通过 Orbit 已保存的 Windows OpenSSH 目标和凭据执行；客户端未接收私钥，Docker Desktop 未运行时仍可诊断宿主机。
- 未新增 SCP、远程 MCP、local 执行或环境初始化。后续 Docker 启动项设置是用户单独授权的运维动作；Casdoor 服务复制使用已有 MCP，不计为本功能实现或验收证据。

## Plan alignment / 计划对齐

- 依据[已接受计划](../plan/20261015-project-environment-cli.md)：三个 PAT 专用路由、共享 WebSocket 终端协议、独立 CommandService/SSH CommandRunner、同连接指纹确认、stdout/stderr 字节流、执行状态和资源限制均已落地。
- 核对当前 diff 的路由与装配、HTTP handler、命令用例、SSH adapter、Python client，以及已有认证、终端和命令测试；未发现需改变既定方案的范围偏差。
- 交付基线为 `00fcbc99`。核对时已有 47 个变更文件处于暂存区；这是接收时的状态，本轮没有执行暂存、提交或发布。当前轮次只新增本验证文档并同步计划状态。
- 实现快照：`git diff HEAD --name-only` 中排除 `docs/**` 和 `scripts/README.md` 后的 38 个文件，按路径排序，对每项 `UTF-8(path) + NUL + SHA256(file bytes)` 拼接再取 SHA-256，结果为 `7ced9ba7564be20cc88d22b59c1197e5cc7399f89c80b48333da8657c995f074`。用于定位本轮文档核对对应的代码、生成物、依赖和配置，不包含 PAT 或远端配置。

## Task deliveries and integration / 子任务交付与集成

不适用。按总计划单会话交付，无独立 task 或子 agent。

## Corrections and scope / 修复与范围

| 对齐依据 | 发现的偏差或现象 | 实际处理 | 复验结果与证据 |
| --- | --- | --- | --- |
| A01 配置来源 | 初次调用未选择 development profile，缺少 BASE_URL/PAT | 本次调用进程设置 `POMELO_ORBIT_APP__ENV=development` 和本地 Orbit BASE_URL，读取既有 `.env.development` PAT | 项目列表成功；未输出 PAT，未新增凭据 |
| A04 显式 shell 与 stdin | Windows 较长 `-Command` 参数返回 `The command line is too long.` | 改用已有 `--stdin`，远端执行 `powershell.exe -NoProfile -NonInteractive -Command -`；未修改代码或重试结果未知的写命令 | 同一诊断脚本通过 stdin 执行成功；保留 Windows 命令行长度限制 |
| A08 超时结果 | Docker Desktop 进程出现后，后续 Docker 引擎查询未在时限内返回 | 保留实际 `timed_out` 结果和已完成命令的部分输出，不宣称全部成功 | 45 秒超时结果不含伪造的成功 exit code |
| 验证文档与实施记录 | 原计划仍标记整体 Verification 未开展 | 新增本记录，计划补充用户验收、实测与剩余证据边界 | 文档引用与 diff 空白检查通过 |

本轮未修改实现代码。工具策略曾在本地拒绝 Docker 启动命令，未到达远端；这不能证明 SSH 用户缺少管理员权限，也不是 Orbit CLI 返回的认证失败。Docker Desktop 后来由用户自行启动，不能记为代理启动成功。

## Acceptance results / 验收结果

| 验收 ID | 结果 | 证据或未满足原因 |
| --- | --- | --- |
| A01 | 通过 | 集中配置与认证测试；development profile 下读取既有 PAT，项目查询及命令调用成功 |
| A02 | 通过 | 实际列表返回 `guohua` 的 code、Windows/SSH 目标摘要；AccessService 测试覆盖 local、未配置和 inactive 项目，不返回私钥 |
| A03 | 部分通过 | 共享 Web/CLI 协议及 Python 控制键、resize、UTF-8、退出恢复测试通过；用户反馈实测可用，但未提供逐项物理 TTY 和 Linux/Windows 交互矩阵结果，相关实机项仍未验证 |
| A04 | 通过 | Windows 真实 SSH 执行主机、用户、服务及日志查询；独立 stderr、非零退出和 stdin 脚本可用；本地 SSH/Windows 进程组件覆盖 exit 7、二进制流与字面 argv |
| A05 | 通过 | PAT/JWT 入口分离、成员/目标/凭据变更及同连接指纹确认测试；实测按已保存目标执行，客户端未覆盖目标或提供私钥 |
| A06 | 通过 | 实测 Docker Desktop 进程缺失、WSL 实例停止且引擎管道不存在时，SSH 宿主机诊断仍成功；无 Probe/Gateway 前置条件另有用例证据 |
| A07 | 通过 | 每次实测显式 `--project guohua`；精确 code、访问拒绝、local 禁止及无自动回退/重试由用例与客户端实现覆盖 |
| A08 | 通过 | 实测 45 秒 `timed_out`、部分输出和无 exit code；输出预算、配额、撤权和取消测试通过；命令正文日志排除测试通过 |
| A09 | 通过 | [调用指南](../guides/project-environment-cli.md)与 CLI 参数一致；实测使用 list、exec、JSON、显式 shell 和 stdin，不依赖 SCP |
| A10 | 通过（开发检查与记录） | Implementation 固定检查通过，本轮未变更代码；已明确记录真实 Windows 实测及 A03 的未验证矩阵，不将其计为全平台通过 |

## Check results / 检查结果

2026-10-16 配置目录调整后，CLI 改为读取 `scripts/.env[.<env>]`，所有配置使用 `POMELO_ORBIT_` 前缀；`make run` 仍默认 `development`。当前配置约定见 [CLI README](../../scripts/README.md#配置)。本机无 `make`，在 `scripts/` 下执行 Makefile 中的等效检查：`uv run --locked --no-sync ruff format --check .`、`ruff check .`、`mypy` 和 `pytest`（后三项使用相同 uv 前缀）均通过，pytest 为 91 passed。测试覆盖目录定位、profile 选择、环境变量优先级及 remote 共用配置；未重复远程操作。

以下表格及远程实测沿用 2026-10-15 Implementation 的完成记录，不作为本次重新执行的证据：

| 命令 / 检查 | 实际结果 | 本轮未重跑原因 |
| --- | --- | --- |
| `task proto` | 通过；重复生成的新增 Go/TS DTO SHA-256 一致 | 本轮无协议或生成物修改 |
| `task test:go:unit -- -count=1` | 通过 | 本轮无 Go 修改 |
| `task check`、`go test ./cmd/... ./internal/...` | 均通过；lint 0 issues | 本轮无实现修改 |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 均通过 | 本轮无前端修改 |
| scripts Ruff format / Ruff check / mypy | 均通过 | 本轮无 Python 修改 |
| scripts pytest | 85 passed | 本轮无 Python 修改 |
| 原 Web 终端回归、PAT 升级、本地 SSH channel、Windows 原生进程组件 | 通过 | 证据见计划实施记录；不替代物理终端实测 |

真实调用证据（2026-10-15，本会话已执行，不为写文档重复远程操作）：

| 操作 | 实际结果 | requestId |
| --- | --- | --- |
| `environment list --json` | 成功发现 `guohua`，无私钥输出 | `01M4ZXERDHQZ9W3KME4KJ1S3T3` |
| `exec --project guohua --json` 查询 hostname、whoami、Docker/WSL | `state=exited`；Docker 引擎不可用错误在 stderr，SSH 查询成功 | `01M4ZXFC2BYAY6VTXYCHHYEX5P` |
| 较长 PowerShell `-Command` | `state=exited, exit_code=1`；远端命令行过长 | `01M4ZXG30JQXV7PJZMGFYXNJM8` |
| `--stdin` 输入同类 PowerShell 诊断脚本 | 成功读取开机时间、登录状态、启动配置和 Docker 日志 | `01M4ZXGM65YYZ10RV5D4B3JRWY` |
| Windows 事件日志查询 | 成功定位计划系统更新重启及用户注销 | `01M4ZXH52V35JTMBRK6R716EBG` |
| Docker 启动项设置及后续引擎查询 | 启动项写入并回读成功，随后查询返回 `state=timed_out, duration_ms=45000`，未报告整体成功 | `01M4ZY2H6QZBEVGEVWREMFCX7P` |

本轮文档检查：`git diff --check`、`git diff --cached --check` 通过；本验证文档与计划的相对 Markdown 文件链接有效，实现快照未改变。

## Risks and incomplete items / 风险与未完成项

- 未完成 Linux 真实 SSH 和物理 TTY 下的交互选择、控制键、resize、断开恢复矩阵；需在相应平台连接真实交互终端补验。用户本次总体可用确认不补造这些细项证据。
- Windows OpenSSH 外层命令行对编码后的长脚本可能先于 API 的 32 KiB argv 上限失败。长 PowerShell 脚本本轮已使用 `--stdin` 成功执行；API 参数上限不等同于目标 shell 的可执行长度保证。
- 部分 Windows 程序输出 UTF-16 或本地编码，JSON 按 UTF-8 展示可能出现 NUL 或替换字符；实测已出现。原始 API 字节保留，调用者可要求远端输出 UTF-8 或使用原始输出自行解码。
- Python 检查在 Windows Python 3.14.2 完成，Python 3.12 下限及 Linux TTY 未实机验证。
- 超时/断线不保证远端进程树终止，不自动重放结果未知的命令；代理等待时间需覆盖执行时限。

## Conclusion / 结论

用户确认实际使用可用，Windows 项目发现及非交互 SSH 诊断链路验证通过，本次验证文档已完成。开发检查、实测成功与失败结果均已记录；A03 的跨平台交互实机证据仍不完整，不能据此宣称所有平台和终端操作全部通过。保留活过程文档供后续补验，不在本轮归档或执行 Git 写操作。
