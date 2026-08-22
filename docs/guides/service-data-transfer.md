# Orbit 服务导出导入

Orbit 的服务迁移只处理一个 Service 的部署闭包，不承担通用数据库传输。
底层 SQLite/MySQL 连接、JSONL 解析、日期时间转换、主键冲突策略和事务由已安装的
dbtalk CLI 提供。Orbit 与 dbtalk 通过 `dbtalk database` 子命令通信，不导入 dbtalk
Python 包。

## 前置条件

1. 在本机安装 uv：https://docs.astral.sh/uv/
2. 在本机安装 dbtalk，并确认 `dbtalk database --help` 可用。
3. 目标 Orbit 数据库先执行当前版本的迁移初始化。服务文件不包含 schema、索引、
   触发器、函数、存储过程或权限。
4. 数据库连接使用 dbtalk 的 canonical DSN；优先将 DSN 放在环境变量中，只把变量名传给脚本。

## 导出

```bash
uv run --project scripts python scripts/database_transfer.py export \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --service-code <service-code> --output service.jsonl --tz UTC
```

对于 MySQL，将 canonical DSN 放在环境变量中，并使用
`--source mysql --dsn-env <ENV_NAME>`；环境变量值应使用
`mysql+pymysql://user:password@host:3306/database` 形式。

导出脚本让 dbtalk 读取服务闭包所需表并生成临时 JSONL，然后按 Orbit 领域规则裁剪为
目标服务的 Project、Application、完整 Version lineage、Version/Service components、
Gateway 配置、环境覆盖和 Route。临时文件只在进程内存在，完成后自动删除，不读取无关
表或携带 `schema_migrations`。

## 导入

```bash
uv run --project scripts python scripts/database_transfer.py import \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --input service.jsonl --mode upsert --tz UTC
```

必须显式指定 `--mode`。`insert` 遇到主键、唯一键或其他约束冲突时失败；`upsert`
按文件中声明的主键或联合主键更新已有行。目标执行正常 Orbit 迁移后已包含种子
Project，因此标准服务迁移使用 `upsert`。导入前，Orbit 会拒绝缺表、多个 Service、
跨 Application 的版本谱系、循环 lineage、缺失引用或混入服务闭包外数据的文件。通过
校验后才调用 dbtalk。

服务 JSONL 不包含 `schema_migrations` 或其他无关表，导入不传表筛选参数。

`--tz` 使用 IANA 时区名称，应在导出和导入时按同一业务约定传递。具体 JSONL 记录格式、
日期时间解释、DATE/TIME 规则和事务语义以 dbtalk 的 `dbtalk-database` skill 为准。
