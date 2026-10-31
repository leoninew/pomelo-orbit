# 环境 CLI

环境 CLI 提供 SSH 环境列表、远程命令、交互 SSH 和本地端口转发。它使用与实例匹配的程序和配置，[配置根目录与认证方式](./mcp.md#配置客户端)与本地 MCP 相同。

下面命令从本地 Bash 发起。在实例配置根目录运行，`pomelo-orbit` 表示已准备的程序；秘密通过可信环境注入。

## 查找目标

```bash
pomelo-orbit cli env --help
pomelo-orbit cli env list --json
```

列表只含当前成员可见、已配置为 SSH 的项目。后续 `--project` 使用**精确项目编码**，不同于 MCP 的项目 ID。列表可见不代表 SSH 已通过探测。

## 执行命令

```bash
# 将 demo 换为真实项目编码；目标需有对应程序
pomelo-orbit cli env exec --project demo --json -- whoami
```

| 参数或结果 | 规则 |
| --- | --- |
| `--` 后的参数 | 按程序与字面参数执行；管道、重定向需显式调用 shell |
| `--workdir` | 默认环境工作区，也可指定目标绝对路径或 `~/...` |
| `--timeout` | 默认 60 秒，范围 1–300 秒 |
| `--stdin` | 显式读取本地输入，最多 64 KiB；默认发送 EOF |
| 输出 | stdout 与 stderr 合计最多 1 MiB，超限状态为 `output_limit` |

Windows 目标按实际程序使用 `whoami.exe` 等名称。本地 Bash 不改变远端程序或解释器。

JSON 返回状态、远端退出码、时长和文本；非法 UTF-8 会被替换并标记，不用于无损二进制传输。自动化同时检查状态与进程退出码：

| 状态 | 直接运行二进制时的退出码 |
| --- | --- |
| `exited` | 远端 0–255，`exit_code=0` 为正常退出 |
| `timed_out` | 124 |
| `output_limit` / `access_or_target_changed` | 125 |
| `transport_lost` 或一般平台错误 | 255 |
| 参数 / 取消 / 配置错误 | 可能分别为 2 / 130 / 1 |

SSH 使用受管凭据并校验主机指纹，首次连接可能记录指纹。超时、连接丢失或本地取消后，先核对远端实际结果，再重试写入。

## 交互 SSH

```bash
pomelo-orbit cli env ssh --project demo
```

stdin 和 stdout 都需 TTY；省略项目时可交互选择。`Ctrl+]` 本地断开，`Ctrl+C` 通常传给远端 shell。会话有连接、空闲与总时长限制。

## 后台端口转发

```bash
pomelo-orbit cli env forward start --project demo -p 15432:5432
pomelo-orbit cli env forward status -p 15432 --json
pomelo-orbit cli env forward stop -p 15432
```

格式为 `LOCAL:REMOTE`，端口范围 1–65535。远端连接 SSH 主机的回环端口，本地只监听回环地址。

启动会派生后台进程。使用相同 OS 用户、配置根目录运行 `status` / `stop`，本地记录在 `data/cli/forward/` 的用户隔离目录。管理命令不打开业务数据库或校验 PAT；PAT 撤销也不能代替停止转发。
