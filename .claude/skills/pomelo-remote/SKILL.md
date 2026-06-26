---
name: pomelo-remote
description: 管理 Pomelo Orbit 远程服务器环境
allowed-tools: Bash
---

# Pomelo Orbit 远程环境管理

统一管理远程服务器的命令执行、数据库查询、容器管理和部署操作。

## 前置条件

**本地环境：**
- 项目根目录包含 `scripts/manage.py` 脚本
- `scripts/.env` 配置了 SSH 连接信息（`SSH_HOST`, `SSH_USER`）

**远程环境：**
- 远程工作目录：`/opt/pomelo-orbit`
- 已安装 `pomelo-db` 工具（数据库查询 CLI）
- `/opt/pomelo-orbit/.env` 包含 pomelo-db 数据源配置

**关于 pomelo-db 配置：**
- 数据源名称：`pomelo-orbit`（在远程 .env 中配置）
- pomelo-db 支持通过 `-a` 参数添加数据源配置（详见 `/pomelo-db` skill）
- 配置格式：`pomelo-db -a <name>=<dsn>`，例如 `pomelo-db -a orbit=sqlite:///opt/pomelo-orbit/data/pomelo-orbit.db`

## 基础命令格式

```bash
# 在项目根目录执行（包含 scripts/manage.py 的目录）
python scripts/manage.py <command> [options]
```

**注意**：所有命令必须在项目根目录执行，即 `scripts/` 目录的父目录。

## 1. 命令执行

### 基本用法

```bash
# 在远程 home 目录执行
python scripts/manage.py exec '<command>'

# 在指定工作目录执行
python scripts/manage.py exec -w /opt/pomelo-orbit '<command>'
```

### 常用示例

```bash
# 查看远程工作目录
python scripts/manage.py exec -w /opt/pomelo-orbit 'pwd'

# 查看远程文件列表
python scripts/manage.py exec -w /opt/pomelo-orbit 'ls -la'

# 查看环境变量
python scripts/manage.py exec -w /opt/pomelo-orbit 'cat .env | grep -v "SECRET"'
```

## 2. 数据库操作

**关于 pomelo-db：**
- 轻量级数据库查询 CLI 工具，支持 SQLite、MySQL、PostgreSQL 等
- 通过 `-d <datasource>` 指定数据源，`-e "<sql>"` 执行查询
- 默认只读模式，写入操作需添加 `-w` 参数
- 输出格式：`-o json`（默认）或 `-o table`（表格）
- 详细用法参见 `/pomelo-db` skill

### 验证 pomelo-db 可用性

```bash
# 检查 pomelo-db 是否安装
python scripts/manage.py exec -w /opt/pomelo-orbit 'which pomelo-db'

# 查看已配置的数据源
python scripts/manage.py exec -w /opt/pomelo-orbit 'pomelo-db -l'
```

### 查询迁移历史

```bash
# 查看所有迁移记录
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT id, filename, checksum FROM __migration_history ORDER BY id" -o table'

# 统计迁移数量
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT COUNT(*) as count FROM __migration_history" -o table'
```

### 查询业务数据

```bash
# 查看用户列表
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT id, username, is_active FROM user" -o table'

# 查看应用列表
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT id, name, code, status FROM application" -o table'

# 查看代码仓库
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT id, name, code, repository_url FROM repository" -o table'

# 查看流水线模板
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT id, name, version FROM pipeline_template" -o table'
```

### 写入操作（需要 -w 参数）

```bash
# 更新迁移记录 checksum
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -w -e "UPDATE __migration_history SET checksum=\"xxx\" WHERE filename=\"v0.6.0__ci_schema.sql\""'

# 删除记录
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -w -e "DELETE FROM __migration_history WHERE id > 10"'
```

## 3. 容器管理

### 查看状态

```bash
# 查看容器状态
python scripts/manage.py docker-compose ps

# 查看容器日志（最近 50 行）
python scripts/manage.py docker-compose logs --tail=50

# 实时查看日志
python scripts/manage.py docker-compose logs -f
```

### 容器操作

```bash
# 重启服务
python scripts/manage.py docker-compose restart

# 停止服务
python scripts/manage.py docker-compose down

# 启动服务
python scripts/manage.py docker-compose up -d

# 重新构建并启动
python scripts/manage.py docker-compose up -d --build
```

### 故障排查

```bash
# 查看迁移相关日志
python scripts/manage.py docker-compose logs --tail=100 | grep -i migration

# 查看错误日志
python scripts/manage.py docker-compose logs --tail=100 | grep -i error

# 进入容器 shell
python scripts/manage.py exec -w /opt/pomelo-orbit 'docker compose exec pomelo-orbit sh'
```

## 4. 部署操作

### 升级部署

```bash
# 升级到指定镜像
python scripts/manage.py upgrade --image ghcr.io/leoninew/pomelo-orbit:latest

# 升级并跳过拉取镜像（镜像已存在）
python scripts/manage.py upgrade --image ghcr.io/leoninew/pomelo-orbit:v1.0 --skip-pull
```

### 备份数据

```bash
# 备份远程数据目录
python scripts/manage.py backup

# 备份到指定目录
python scripts/manage.py backup --remote-dir /opt/pomelo-orbit/data
```

## 5. 文件传输

```bash
# 上传本地文件到远程服务器
python scripts/manage.py scp to-remote ./local-file.txt /opt/pomelo-orbit/local-file.txt

# 下载远程文件到本地
python scripts/manage.py scp from-remote /opt/pomelo-orbit/.env ./remote.env

# 递归上传本地目录到远程服务器
python scripts/manage.py scp to-remote --recursive ./dist /opt/pomelo-orbit/dist

# 递归下载远程目录到本地
python scripts/manage.py scp from-remote -r /opt/pomelo-orbit/logs ./logs
```

说明：
- SSH 连接信息来自 `scripts/.env` 中的 `SSH_HOST` 和 `SSH_USER`。
- 使用 `--recursive` / `-r` 时，脚本会先断言源路径是目录。
- 递归复制采用 `scp -r` 原生行为，目标已存在时覆盖同名文件或合并目录，不删除目标端多余文件。
- 路径中包含空格或特殊字符时，需要按当前 shell 规则加引号。
- Windows/Git Bash 下建议使用相对路径或 `/c/Users/...` 风格路径。

## 6. 直接 SSH 连接

```bash
# SSH 连接到远程服务器
python scripts/manage.py ssh
```

## 常见问题排查

使用以下命令组合进行问题诊断：

```bash
# 查看容器状态和日志
python scripts/manage.py docker-compose ps
python scripts/manage.py docker-compose logs --tail=100

# 查看迁移历史
python scripts/manage.py exec -w /opt/pomelo-orbit \
  'pomelo-db -d pomelo-orbit -e "SELECT * FROM __migration_history ORDER BY id" -o table'

# 检查环境配置
python scripts/manage.py exec -w /opt/pomelo-orbit 'cat .env | grep -v SECRET'

# 查看远程文件
python scripts/manage.py exec -w /opt/pomelo-orbit 'ls -la'
```

## 注意事项

1. **工作目录**：大部分操作需要 `-w /opt/pomelo-orbit` 参数
2. **引号转义**：命令中包含双引号时会自动转义
3. **写入操作**：数据库写入操作必须添加 `-w` 参数
4. **日志查看**：使用 `docker-compose logs` 而非 `exec` 进入容器
5. **备份数据**：定期备份远程数据目录到本地 `scripts/backup/`
6. **文件传输**：使用 `scp to-remote` / `scp from-remote`，目录复制需显式添加 `--recursive` / `-r`

---

## 备用方案：SSH MCP

当 `scripts/manage.py` 无法解决问题时，可以使用 SSH MCP 直接连接远程服务器。

### 安装 SSH MCP

从 `scripts/.env` 读取配置变量：
- `SSH_HOST`：远程服务器地址
- `SSH_USER`：SSH 用户名
- SSH 密钥路径：通常为 `~/.ssh/id_rsa`

```bash
# 1. 移除旧的 MCP 配置（如果存在）
claude mcp remove pomelo-orbit-ssh

# 2. 添加 SSH MCP（从 scripts/.env 读取配置后替换占位符）
claude mcp add --transport stdio pomelo-orbit-ssh -- npx -y ssh-mcp -- --host=<SSH_HOST> --user=<SSH_USER> --key=<key-path>
```

### 重启会话

安装 MCP 后需要重启 Claude Code 会话：

```bash
# 1. 重启会话（Ctrl+Shift+P -> "Claude Code: Restart Session"）
# 2. 使用 -c 恢复会话上下文
```

### 使用 SSH MCP 执行远程命令

重启后可以直接使用 MCP 工具：

```bash
# 执行远程命令
mcp__ssh-mcp__exec "cd /opt/pomelo-orbit && pwd"

# 查询数据库
mcp__ssh-mcp__exec "cd /opt/pomelo-orbit && pomelo-db -d pomelo-orbit -e 'SELECT COUNT(*) FROM user' -o table"

# 查看容器状态
mcp__ssh-mcp__exec "cd /opt/pomelo-orbit && docker compose ps"
```

### 何时使用 SSH MCP

- `scripts/manage.py` 命令转义问题无法解决
- 需要更直接的远程命令执行
- 需要在 AI 对话中直接操作远程服务器
- 调试复杂的 shell 命令

### SSH MCP vs scripts/manage.py

| 特性 | scripts/manage.py | SSH MCP |
|------|-------------------|---------|
| 安装 | 无需额外安装 | 需要安装 MCP |
| 使用 | 命令行工具 | AI 对话中调用 |
| 命令转义 | 自动处理 | 需要手动处理 |
| 适用场景 | 日常运维 | 复杂调试、AI 辅助 |
| 会话依赖 | 无 | 需要重启会话生效 |
