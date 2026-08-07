# `migrate_pipeline_template_instances.py` - Pipeline 模板实例切换

Doc role: 一次性离线 SQLite 迁移工具说明。与代码冲突时以代码为准。

## 用途

将旧 SQLite v30 的 `pipeline_template`、全局 `pipeline_stage` 和 `pipeline_template_stage` 改造成新的 `Pipeline(kind=template)` 及其自有 `pipeline_stage`。旧 Template ID、名称、版本、变量声明和关联行 ID 会保留；关联行 ID 成为新 Stage ID，旧依赖会重映射为这些新 ID。所有制品声明中的 `component_name` 均会被移除，因此迁移后的对象始终是通用 Template。

该工具不尝试把旧 Template、全局 Stage 绑定或运行记录猜测成 Application Pipeline。旧模型没有可靠的 Application + Repository 身份组合，猜测会产生错误的可运行配置。因此迁移不会创建任何 `kind=application` Pipeline；需要在切换后从 Template 明确创建它们。

## 数据处置

下列记录与新运行链路不兼容，存在数据时只能在显式确认后删除：

- `repository_webhook`；
- Template `pipeline_snapshot`；
- `pipeline_run`、`pipeline_stage_run`、`artifact`；
- Stage/Run 级的旧 Application Version binding。

`version_component` 不会删除。脚本重建该表，移除 `artifact_id -> artifact` 的物理外键和 `ON DELETE SET NULL`，保留原 `artifact_id` 作为逻辑引用；若旧 Artifact 存在，则把制品名称、镜像引用、SHA256 与来源 commit 写入新增的展示快照字段。迁移后删除 Artifact 不会再回写 Version Component。

脚本不会修改 `schema_migrations`，只接受当前项目旧 v30、`dirty=0` 的完整 SQLite 基线。已迁移的数据库、缺表、字段集合不符、重复的同项目 Template 名称、无法重映射的 Stage 依赖或循环 DAG 都会被拒绝。

## 用法

先执行默认的只读预检：

```bash
python scripts/migrate_pipeline_template_instances.py \
  --database data/db/pomelo-orbit.db
```

预检会执行 SQLite 完整性与外键检查，并输出将转换的 Template/Stage 数量及待删除旧记录数。若存在旧历史，预检以非零状态退出，要求操作人查看计数后再显式确认。

离线切换默认必须指定一个尚不存在的备份路径，并确认删除旧历史：

```bash
python scripts/migrate_pipeline_template_instances.py \
  --database data/db/pomelo-orbit.db \
  --backup data/db/pomelo-orbit-before-pipeline-instance-cutover.db \
  --apply \
  --drop-legacy-history
```

`--apply` 会先获得 SQLite 排他事务锁，再用 SQLite backup API 创建并校验备份，随后在同一事务内重建表并在提交前执行完整性、外键、目标列和数据计数检查。备份路径必须不存在，且不能与数据库路径相同，脚本不会覆盖已有备份。数据库仍被其他进程占用时，脚本会快速失败；应停止相关应用写入后重试。

仅当目标库本身是可丢弃副本或已有独立备份时，才可显式跳过这一步：

```bash
python scripts/migrate_pipeline_template_instances.py \
  --database scripts/backup/data-20260807-084023/data/db/pomelo-orbit.db \
  --apply \
  --drop-legacy-history \
  --no-backup
```

`--no-backup` 与 `--backup` 互斥，且只能和 `--apply` 一起使用。它不是默认快捷方式：选择该参数后，迁移事务失败时没有脚本自动创建的回退副本。

## 回退与边界

迁移本身不修改既有编号迁移文件，也不引入业务层兼容逻辑。若切换后需要回退，应停止使用新业务版本，使用本次产生的 SQLite 备份恢复数据库，再部署旧版本；不能在同一个业务版本中并存新旧 Pipeline 模型。

该工具只处理 SQLite 控制面数据，不修改工作区、镜像、Registry、容器、卷或外部 Git 服务。执行前仍应按常规离线变更流程确认应用不再写入该数据库。
