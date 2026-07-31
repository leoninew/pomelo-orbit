# `manage.py` — 远程部署与管理入口

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

通过 `scripts/.env` 配置的 SSH 连接执行 Pomelo Orbit 的远程部署、隧道、命令、Compose、文件复制、备份和日志清理操作。

## 前置条件

`scripts/.env` 必须定义 `SSH_HOST`、`SSH_USER` 与 `REMOTE_DEPLOY_DIR`；`GHCR_TOKEN` 为可选配置，用于需要认证的镜像拉取。执行主机需要 `ssh`，文件复制操作还需要 `scp`。

## 命令族

```bash
python scripts/manage.py upgrade --image IMAGE
python scripts/manage.py tunnel start <remote:local> [remote:local...]
python scripts/manage.py tunnel stop|status
python scripts/manage.py exec [-w REMOTE_DIR] <command>
python scripts/manage.py docker-compose <args>
python scripts/manage.py scp to-remote|from-remote [-r] <source> <destination>
python scripts/manage.py ssh
python scripts/manage.py backup [--remote-dir DIRECTORY]
python scripts/manage.py clean
```

运行 `python scripts/manage.py <subcommand> --help` 获取当前子命令参数。

## 安全边界

`upgrade`、`exec`、`docker-compose`、`scp`、`backup` 与 `clean` 都可能影响远程主机或其文件；`tunnel` 会创建或停止本地 SSH 隧道。执行前应明确目标环境与命令影响，特别是 `exec` 和透传的 Docker Compose 参数。

## 相关文档

远程运维流程、Docker 操作和数据库查询请参阅 [`.claude/skills/pomelo-remote/SKILL.md`](../.claude/skills/pomelo-remote/SKILL.md)。