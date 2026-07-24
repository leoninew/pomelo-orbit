# manage.py scp 文件传输功能需求
最后修改时间: 2026-06-26 11:53:17

Review status: Accepted

## Background

`scripts/manage.py` 已经通过 `scripts/.env` 读取远程 SSH 配置，并提供远程执行、Docker Compose 管理、SSH 连接、部署和备份能力。当前脚本内部已有部署用的 `scp` 上传逻辑，以及备份用的远程文件下载逻辑，但没有面向用户的通用文件复制命令。

本轮需要为该脚本补充通用 `scp` 能力，使用户可以基于 `scripts/.env` 中现有的 SSH 配置，在本地与远程服务器之间复制文件。

## Goal

1. 在 `scripts/manage.py` 中新增面向用户的 `scp` 子命令。
2. 支持将本地文件复制到远程服务器。
3. 支持将远程服务器文件复制到本地。
4. 支持通过 `--recursive` / `-r` 显式递归复制目录。
5. 使用 `--recursive` / `-r` 时断言源路径是目录，再执行递归复制。
6. 目标已存在时按 `scp` 原生行为覆盖或合并写入，不实现 `rsync --delete` 式删除目标端多余文件。
7. 复用 `scripts/.env` 中的 `SSH_HOST`、`SSH_USER` 等现有连接配置，不新增远程连接配置项。
8. 保持实现简单直接，优先复用脚本内已有的 `copy_to_remote` 和备份下载逻辑。

## Non-goal

1. 不实现任意远程主机选择；目标主机固定来自 `scripts/.env`。
2. 不实现批量同步、rsync 或增量同步。
3. 不实现 `rsync --delete` 能力；递归复制目录时不删除目标端已存在但源端不存在的文件。
4. 不自动创建本地或远程目标父目录。
5. 不新增 `--port`、`--mkdirs` 等额外选项。
6. 不重构 `manage.py` 的整体命令结构。
7. 不执行 git commit / push / merge 等 git 写操作。

## User scenarios

1. 用户希望把本地临时文件上传到远程服务器，例如复制到 `/tmp` 或远程部署目录下。
2. 用户希望把远程服务器上的配置、日志或排查用文件下载到本地。
3. 用户不需要手动拼接 `user@host:`，只需指定复制方向和源/目标路径。
4. 用户希望复制目录时，可以显式添加 `--recursive` 或 `-r`，并且脚本会先确认源路径是目录。
5. 用户通过 `python scripts/manage.py scp --help` 能看到清晰的命令用法。

## Acceptance

1. `python scripts/manage.py scp to-remote <local_path> <remote_path>` 可以将本地文件复制到 `scripts/.env` 配置的远程服务器。
2. `python scripts/manage.py scp from-remote <remote_path> <local_path>` 可以将远程文件复制到本地。
3. `python scripts/manage.py scp to-remote --recursive <local_dir> <remote_dir>` 可以递归上传本地目录。
4. `python scripts/manage.py scp from-remote --recursive <remote_dir> <local_dir>` 可以递归下载远程目录。
5. 使用 `--recursive` 或 `-r` 时，脚本必须先断言源路径是目录：
   - `to-remote` 对本地源路径执行本地目录检查。
   - `from-remote` 对远程源路径执行远程目录检查。
6. 未使用 `--recursive` / `-r` 时按文件复制处理，不额外自动识别目录。
7. 递归复制目录时，目标已存在则按 `scp -r` 原生行为覆盖或合并写入；不删除目标端多余文件。
8. 命令实现复用 `Config.ssh_target`，用户不需要在参数中输入 SSH 用户名或主机名。
9. 复制命令使用 `subprocess.run([...], check=True)` 形式调用 `scp`，不使用本地 shell 拼接执行。
10. `scripts/manage.py` 顶部用法说明包含新增 `scp` 命令。
11. `.claude/skills/pomelo-remote/SKILL.md` 记录上传、下载和递归目录复制示例。
12. 代码变更后运行脚本检查命令：`python -m mypy scripts/`、`python -m ruff check scripts/ --fix`、`python -m ruff format scripts/`。
13. 至少验证 `python scripts/manage.py --help` 和 `python scripts/manage.py scp --help` 能正常显示新增命令。

## Open questions

暂无必须由用户决定的未决事项。

可继续推进但需要用户知晓的假设：

1. 命令名称采用 `scp to-remote` / `scp from-remote`，避免 Windows 路径盘符中的冒号与 scp 远程路径语法混淆。
2. 本轮支持文件复制和显式 `--recursive` / `-r` 目录递归复制，但不支持同步语义。
3. 递归目录复制的覆盖行为采用 `scp -r` 原生语义：存在则覆盖同名文件或合并目录，不删除目标端多余文件。
4. 路径中包含空格或特殊字符时，由用户按当前 shell 规则加引号；脚本不额外实现复杂转义层。

## Decisions

1. 新命令挂在 `scripts/manage.py scp` 下，而不是新增独立脚本。
2. 连接信息继续由 `scripts/.env` 统一提供。
3. 使用显式方向参数 `to-remote` / `from-remote`，不根据路径中是否包含 `:` 自动判断方向。
4. 支持 `--recursive` / `-r` 显式目录递归复制，并在复制前断言源路径是目录。
5. 失败处理沿用脚本现有风格：`scp` 失败时让 `subprocess.run(..., check=True)` 抛出错误并终止命令。

## Risk

1. 远程路径仍会经过远程 shell/scp 解析，包含特殊字符的路径可能需要用户正确加引号。
2. Windows/Git Bash 环境下，绝对路径风格差异可能影响本地路径识别；建议优先使用相对路径或 Git Bash 风格路径。
3. 不自动创建父目录会让目标目录不存在的复制操作失败；这是符合原生 `scp` 预期的显式失败。
4. 远程目录断言需要额外执行一次远程 `test -d`，如果远程路径包含特殊字符，仍可能受远程 shell 解析影响。
