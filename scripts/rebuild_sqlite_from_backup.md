# rebuild_sqlite_from_backup 使用说明

`rebuild_sqlite_from_backup.py` 用于把旧版 Pomelo Orbit SQLite 备份库重建为符合当前 `backend/migrations` 要求的新库。

适用场景：迁移脚本被重写后，旧库中的 `__migration_history` 与当前迁移文件名不一致，应用启动会重复执行当前迁移并失败。

## 脚本做什么

脚本会执行以下步骤：

1. 使用当前 schema 文件创建一个全新的 SQLite 库。
2. 从旧备份库只读复制业务数据。
3. 如果旧库没有 `project` 表，则创建默认项目。
4. 为新增 `project_id` 的表回填默认项目 ID。
5. 不复制旧库的 `__migration_history`。
6. 计算当前迁移文件 checksum，并写入新的 `__migration_history`。
7. 执行校验：
   - `PRAGMA foreign_key_check`
   - `project_id` 非空检查
   - 当前迁移历史 checksum 检查
   - 调用 `run_migrations()` 确认启动迁移不会失败
8. 可选生成结构化 insert JSON，方便远程导入。

## 获取备份

先从远程环境生成并拉取备份：

```bash
python scripts/manage.py backup
```

`backup` 命令会读取 `scripts/.env` 中的 `REMOTE_DEPLOY_DIR`，在远程执行：

```bash
tar -czf /tmp/pomelo-orbit-backup-<时间戳>.tar.gz --exclude=./data/ci/*/workspace -C <REMOTE_DEPLOY_DIR> .
```

然后通过 `scp` 下载到本地：

```text
scripts/backup/data-<时间戳>.tar.gz
```

如果需要指定其他远程目录，可以使用：

```bash
python scripts/manage.py backup --remote-dir /path/to/remote/deploy/dir
```

由于压缩包内容以远程部署目录为根，数据库在归档内的路径是 `./data/db/pomelo-orbit.db`。确认备份文件名后，提取旧库到脚本输入路径：

```bash
tar -xOzf "scripts/backup/data-<时间戳>.tar.gz" "./data/db/pomelo-orbit.db" > "backend/data/db/pomelo-orbit.db"
```

## 基本用法

```bash
python scripts/rebuild_sqlite_from_backup.py \
  --backup backend/data/db/pomelo-orbit.db \
  --output backend/data/db/pomelo-orbit.rebuilt.db \
  --json-output backend/data/db/pomelo-orbit.import.json
```

## 参数说明

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `--backup` | 是 | 旧 SQLite 备份库路径。脚本以只读方式打开该文件。 |
| `--output` | 是 | 生成的新 SQLite 库路径。默认不允许覆盖已有文件。 |
| `--json-output` | 否 | 生成结构化 insert JSON 的路径。可用于远程导入。 |
| `--schema` | 否 | 当前 schema SQL 文件，默认 `backend/migrations/v0.7.0__schema.sql`。 |
| `--migrations-dir` | 否 | 当前迁移目录，默认 `backend/migrations`。 |
| `--default-project-id` | 否 | 旧数据回填使用的默认项目 ID。 |
| `--default-project-name` | 否 | 默认项目名称，默认 `默认项目`。 |
| `--default-project-code` | 否 | 默认项目编码，默认 `default`。 |
| `--force` | 否 | 如果输出文件已存在，允许覆盖。谨慎使用。 |

## 覆盖已有输出文件

默认情况下，如果 `--output` 或 `--json-output` 指向的文件已经存在，脚本会直接失败，避免误覆盖。

确认可以覆盖时再使用：

```bash
python scripts/rebuild_sqlite_from_backup.py \
  --backup backend/data/db/pomelo-orbit.db \
  --output backend/data/db/pomelo-orbit.rebuilt.db \
  --json-output backend/data/db/pomelo-orbit.import.json \
  --force
```

## 成功输出示例

```text
Rebuilt database: backend/data/db/pomelo-orbit.rebuilt.db
Import JSON: backend/data/db/pomelo-orbit.import.json
user: 1
project: 1
application: 7
repository: 5
pipeline_run: 44
artifact: 21
__migration_history: 3
Validation: ok
```

看到 `Validation: ok` 表示新库已通过脚本内置校验。

## 结果判断

生成的新库可以认为是符合当前 `backend/migrations` 要求的库，当且仅当：

- 脚本输出 `Validation: ok`
- `__migration_history` 只有当前迁移文件记录
- `run_migrations()` 校验通过

当前脚本写入的迁移历史来自 `CURRENT_MIGRATION_FILES`：

```python
CURRENT_MIGRATION_FILES = [
    "v0.7.0__schema.sql",
    "v0.7.1__init_data.json",
    "v0.7.2__business_data.json",
]
```

如果后续迁移文件发生变化，需要同步更新脚本中的 `CURRENT_MIGRATION_FILES`，或改造成自动发现策略。

## 注意事项

- 脚本不会修改 `--backup` 指向的旧库。
- 脚本默认不会覆盖输出文件。
- `--force` 会删除已有输出文件后重新生成，使用前请确认目标路径无误。
- 生成的 JSON 是结构化 insert 数据，不包含旧迁移历史。
- 远程导入前建议先在本地使用生成的新库启动应用验证。