# Pomelo Orbit CLI

项目内运维 CLI，依赖 Python `>=3.12` 和 uv，无需全局安装。

## 使用

在 `scripts/` 目录执行：

```bash
make deps
make run -- --help
make check
make test
```

仓库根目录可用 `uv --directory scripts run --locked pomelo-orbit-cli <命令>`。`make check fix=1` 自动修复 Ruff 问题。

## 命令

以下命令接在 `make run --` 后，可加 `--help` 查看参数。

| 命令 | 用途 |
| --- | --- |
| `cert new -n <域名> --cert-dir <目录>` | 生成证书和私钥 |
| `cert check -n <域名> --cert-dir <目录>` | 检查证书、信任链和 HTTPS 连接 |
| `ulid -n <数量>` | 生成 ULID，默认一个 |
| `environment list [--json]` | 列出 PAT 用户成员可见项目和环境摘要 |
| `environment ssh [--project <code>]` | 按项目 code 连接受管 SSH 环境；缺省交互选择 |
| `environment exec --project <code> [--json] -- <命令及参数>` | 非交互执行，保留 stdout/stderr 和真实退出状态 |
| `remote ssh` | 连接远程主机 |
| `remote exec -- <命令及参数>` | 执行远程命令，`--workdir` 指定工作目录 |
| `remote scp to-remote <本地> <远程>` | 上传文件，`-r` 传输目录 |
| `remote scp from-remote <远程> <本地>` | 下载文件 |
| `remote tunnel start <远程端口>:<本地端口>` | 启动隧道，支持多个映射 |
| `remote tunnel status` | 查看隧道 |
| `remote tunnel stop` | 停止隧道 |
| `version` | 按完整 Git 历史预览版本，`--apply` 同步版本文件 |
| `release` | 按 `VERSION` 构建 Windows/Linux/macOS x64 ZIP，输出到 `dist/`，依赖 Go 和 Yarn |

`cert` 首次使用前执行 `mkcert -install`；网关配置见[证书管理指南](../docs/guides/certificate-management.md)。`remote` 的 `SSH_HOST`、`SSH_USER` 配置在 `scripts/.env`，环境变量优先；隧道要求免交互密钥认证。

`make run` 默认使用 `development` 环境，可通过 `POMELO_ORBIT_APP__ENV` 覆盖。参数含空格或 `=` 时使用 `make run ARGS='...'`。

项目环境 CLI 使用 `POMELO_ORBIT_CLI__BASE_URL` 与现有 `POMELO_ORBIT_MCP__ACCESS_TOKEN`；服务端读取已保存 SSH 凭据。配置、交互终端及大模型调用契约见[项目环境 CLI 指南](../docs/guides/project-environment-cli.md)。
