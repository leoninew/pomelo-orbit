# Orbit 服务数据迁移

Orbit 的服务迁移只处理一个 Service 的部署闭包，不承担通用数据库传输。
底层 SQLite/MySQL 连接、JSONL 解析、日期时间转换、主键冲突策略和事务由已安装的
Housekeeper CLI 提供。Orbit 与 Housekeeper 通过 `housekeeper database` 子命令通信，
不导入 Housekeeper Python 包。

## 前置条件

1. 在本机安装已发布的 Housekeeper，并确认 `housekeeper database --help` 可用。
2. 目标 Orbit 数据库先执行当前版本的迁移初始化。服务文件不包含 schema、索引、
   触发器、函数、存储过程或权限。
3. MySQL DSN 放在环境变量中，只把变量名传给脚本。

## 导出

```bash
python scripts/database_transfer.py export \
  --source sqlite --sqlite-path data/db/pomelo-orbit.db \
  --service-code <service-code> --output service.jsonl --tz UTC
```

导出脚本让 Housekeeper 生成临时全库 JSONL，然后按 Orbit 领域规则裁剪为目标服务的
Project、Application、完整 Version lineage、Version/Service components、Gateway
配置、环境覆盖和 Route。临时全库文件只在进程内存在，完成后自动删除。该调用固定
排除 `schema_migrations`，避免携带目标 Orbit 需要自行初始化的迁移元数据。

## 导入

```bash
python scripts/database_transfer.py import \
  --target sqlite --sqlite-path data/db/pomelo-orbit.db \
  --input service.jsonl --mode upsert --tz UTC
```

必须显式指定 `--mode`。`insert` 遇到主键、唯一键或其他约束冲突时失败；`upsert`
按文件中声明的主键或联合主键更新已有行。目标执行正常 Orbit 迁移后已包含种子
Project，因此标准服务迁移使用 `upsert`。导入前，
Orbit 会拒绝缺表、多个 Service、跨 Application 的版本谱系、循环 lineage、缺失引用
或混入服务闭包外数据的文件。通过校验后才调用 Housekeeper。

服务 JSONL 不包含 `schema_migrations`，导入时不能传
`--exclude-table schema_migrations`。Housekeeper 对导入排除项要求该表确实存在于
JSONL 文件中。

`--tz` 使用 IANA 时区名称，应在导出和导入时按同一业务约定传递。具体 JSONL 记录格式、
日期时间解释、DATE/TIME 规则和事务语义以 Housekeeper 的 database-transfer 文档为准。
Housekeeper 源码目录中的示例可使用 `uv run housekeeper`；Orbit 运行时使用已安装的
`housekeeper` CLI。
