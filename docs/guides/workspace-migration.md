
最后修改时间: 2026-10-21 13:48:17

在仓库根目录运行：

```sh
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit'
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit' --no-dry-run
```

`apply --target <目录>` 只评估；加 `--no-dry-run` 自动评估并执行，有未完成的执行记录时自动继续。`--target` 必填，缺失或为空时直接报错，不读取应用配置或写入迁移记录。无需指定计划文件、登录 token、数据库连接或控制面启停命令。PowerShell 可以使用 `python scripts/migrate_workspace.py` 加相同参数。

执行前自行停止 Orbit API 和 worker，迁移期间保持停止。迁移完成后，日志提示使用原来的启动方式自行启动 Orbit，并输出 profile、启动工作目录及迁移目标目录。脚本负责业务数据与容器迁移，Orbit 进程由用户管理。

工具要求使用当前 schema：环境库存左连接 `environment_credential.environment_id` 派生凭据 ID/修订，连接按环境读取并复核归属。升级 000054 前应完成旧 schema 下未结束的搬迁批次，或按本指南恢复后撤销批次；升级后重新预演，不能复用旧库存。升级步骤见[环境凭据迁移指南](./environment-credential-migration.md)。

## 自动发现

入口默认选择 `POMELO_ORBIT_APP__ENV=development`，保留已有非空 profile。脚本自动构建迁移专用 Go helper，复用应用的配置加载器，读取 YAML、dotenv、进程变量和 `overwrite.env`；数据库及 SSH 凭据使用已有配置和业务库中的绑定，不需要另建迁移配置。

通过当前配置的工作区及宿主机识别 Environment，源目录使用该 Environment 的 `workspace_root`。目标目录由 `--target` 显式指定，没有默认规则。SSH 环境使用已有账号、托管密钥、host key 和目标平台，在目标机器读取目录与 Docker 信息；SSH 指向本机时仍通过 SSH 操作。

服务目录、挂载和制品位置中的 `~/` 在数据宿主机按运行账号展开后比较和改写；同一宿主机其他环境的根目录也先展开再检查重叠，即使没有运行容器也会识别共用目录。保存的业务快照保留原始路径，用于执行前复核与数据库旧值校验。

`--environment` 可选，用于主动选择其他环境；`--target` 始终必填：

```sh
sh scripts/migrate_workspace.sh apply --environment other-env --target /srv/orbit
```

`--environment` 接受 Environment ID 或 code；无法唯一识别时列出可用 code。Cygwin / Git Bash 使用原生 Windows Python，Windows 目录支持 `D:/Documents/orbit` 和当前 shell 的挂载路径，带空格时加引号。先识别目标系统，再转换 Windows 路径；Linux SSH 目标 `/srv/orbit` 会原样传递。相对 Windows 路径按调用 shell 的目录解析。

## 迁移顺序

1. 读取业务快照、Docker 容器、Compose、目录和空间，确认没有活动 CI/CD 任务、其他环境共用目录或其他容器占用待迁数据。
2. 复核业务快照、Compose 和容器状态；用户已停止 Orbit API 和 worker，保持业务停止写入。
3. 备份 PostgreSQL，冻结现有 Compose 配置，停止原本运行的容器。
4. 冷复制 `pipeline/`、`deployment/`、`deployment-logs/`，逐文件校验 SHA-256，保留源目录。更新新目录内 Compose 的宿主机路径。
5. 先恢复原本运行的数据库等基础服务，等待业务库可用；在单个数据库事务中检查旧值并更新 Environment 根目录、修订、服务目录、挂载覆盖、文件制品位置及仍有效的 CI Run 绑定。
6. 更新已有控制面的 `overwrite.env`，使用已有 Compose 逐个重建其余原本运行的组件，保留副本数量；停止的组件继续保持停止。
7. 核对环境根目录、修订、容器挂载和运行组件数量，保存验证结果，输出自行启动 Orbit 的提示。

迁移保留已有 Probe 状态、时间和诊断，不发起 Probe，也不要求已有状态为成功。Compose 重建只恢复容器，不等待应用 healthcheck；原有环境失联或业务健康问题不作为迁移的额外验收条件。读写目录、执行必要的 SSH 命令和访问业务库仍须实际可用。

已发布 Version 和历史 Deployment 执行输入保持原值。迁移直接重建现有 Compose 拓扑，不通过 HTTP API 提交部署队列。Gateway 的 dynamic、certs、acme 文件随工作区复制，容器内路径保持原值，无需修改 Route 业务数据。Named volume 复用现有卷，卷内容不复制。

## 记录与恢复

计划、说明和 SQL 写入仓库 `data/workspace-migration/<批次>/`，执行后同目录保存数据库备份、Compose 快照、覆盖文件备份、`state.json` 和 `verification.json`。`active.json` 自动指向当前批次，不需要传入路径。迁移完成且释放工作区占用后清除活动指针，保留批次历史；已有完成记录的残留指针在下次运行时自动清除，可继续迁往其他目录或迁移其他 Environment。批次目录限制为当前用户私有；Compose 快照及覆盖文件备份可能含有敏感配置，应与数据库备份一起保管。

迁移中断后，重新运行：

```sh
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit' --no-dry-run
```

恢复时仍须传入同一目标目录，路径会在目标宿主机解析后与保存的计划核对，不匹配即拒绝。已完成的阶段跳过。新目录数据库启动后，继续执行不会重新复制源目录覆盖新写入；已完成迁移重跑不会重复部署。仅预演而未执行时，下次自动重新评估。失败时保留备份与进度，不自动删除旧目录或回切旧数据库。

预演和执行均可能发现真实阻塞，例如活动任务、空间不足或路径重叠；日志输出原因，阻塞退出码为 2，普通失败为 1。

## 运行条件与边界

控制机需要 Python ≥3.12、Go 和 `dbtalk`；目标宿主机需要 Python ≥3.12、Docker Compose，以及读写目录和操作容器所需的权限。迁移通过 SSH 发送同一份 Python 实现，远端不要求有仓库或安装 dbtalk。

当前实现支持 PostgreSQL 业务库。数据库备份复用 dbtalk 的 PostgreSQL 客户端发现机制。Docker 使用目标机器的本地 socket；不跨 Docker endpoint 操作另一台机器的文件系统。业务数据之外的 `data/config`、SQLite 和临时材料不复制。工作区内的链接或 reparse point 会阻止迁移，已知 MySQL 临时 socket 链接和 Unix socket 除外。

路径改写仅覆盖会复制的 `pipeline/`、`deployment/`、`deployment-logs/`。若业务路径或 Compose 的 bind mount、构建目录、配置及 secret 文件引用了工作区内其他位置（例如 `<source>/database`），预演会报出该路径并停止；冻结 Compose 时在停止容器前再次校验，避免路径切换后读到未复制的空目录。工作区之外的挂载及 named volume 沿用原位置。

脚本只修改已识别的路径字段及 Compose 文件路径，不对嵌入业务脚本或业务环境变量的文本做全文替换。

## 文件职责

| 文件 | 用途 |
| --- | --- |
| `scripts/migrate_workspace.sh` | Cygwin / Git Bash / Linux 的简短入口，默认选择 development profile |
| `scripts/migrate_workspace.py` | 评估、文件复制、业务事务更新、容器重建和断点继续的主程序 |
| `scripts/workspace_migration_runtime.go` | 复用应用配置加载、SSH 凭据解密和命令执行；主程序自动编译，无需单独操作 |
| `scripts/test_migrate_workspace.py` | 迁移与恢复的回归测试，不参与运行 |

迁移运行代码集中在上述三个入口及辅助文件。应用侧的初始化进度与目标解析独立于 Probe 历史，迁移后可直接使用已有环境；实际连接、目录、凭据及修订校验仍由相应业务操作执行。

## 验证脚本

```powershell
python -m unittest discover -s scripts -p test_migrate_workspace.py -v
python -m py_compile scripts/migrate_workspace.py
```

`ORBIT_MIGRATION_TEST_CONTAINER` 可指定独立 PostgreSQL 测试容器；事务测试会重建其中的 `migration_test` schema，不能指向业务容器。`ORBIT_MIGRATION_TEST_CYGWIN` 可指定 Cygwin 安装目录以验证真实 shell 入口及路径转换。这两个变量仅用于测试，与迁移命令无关。
