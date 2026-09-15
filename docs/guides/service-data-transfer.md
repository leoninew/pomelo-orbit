# Orbit 服务与环境导出导入

Orbit 的服务迁移只处理一个 Service 的部署闭包，不承担通用数据库传输。
Environment 有独立的备份恢复命令，传输指定 Environment 及其 SSH 密钥。
底层 SQLite/MySQL/PostgreSQL 连接、JSONL 解析、日期时间转换、主键冲突策略和事务由已安装的
dbtalk CLI 提供。Orbit 与 dbtalk 通过根级 `dbtalk export` / `dbtalk import` 命令通信，不导入 dbtalk
Python 包。

## 前置条件

1. 在本机安装 uv：https://docs.astral.sh/uv/
2. 在本机安装 dbtalk，并确认 `dbtalk export --help` 与 `dbtalk import --help` 可用。
3. 目标 Orbit 数据库先执行当前版本的迁移初始化。服务文件不包含 schema、索引、
   触发器、函数、存储过程或权限。
4. 数据库连接使用 dbtalk 的 canonical DSN；优先将 DSN 放在环境变量中，只把变量名传给脚本。

## 导出

```bash
uv run --project scripts python scripts/database_transfer.py export \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <current-project-id> --service-code <service-code> \
  --output service.jsonl --tz UTC
```

对于 MySQL 或 PostgreSQL，将 canonical DSN 放在环境变量中，并分别使用
`--source mysql --dsn-env <ENV_NAME>` 或
`--source postgresql --dsn-env <ENV_NAME>`；环境变量值应分别使用
`mysql+pymysql://user:password@host:3306/database` 或
`postgresql+psycopg://user:password@host:5432/database` 形式。

`--project-id` 必须是当前选中的 Project ID。服务 code 只在该 Project 内定位；脚本会拒绝
Service、Application 或 Route 跨项目的输入数据。

导出脚本让 dbtalk 读取服务闭包所需表并生成临时 JSONL，然后按 Orbit 领域规则裁剪为
目标服务的 Project、Application、完整 Version lineage、Version/Service components、
Gateway 配置、环境覆盖和 Route。临时文件只在进程内存在，完成后自动删除，不读取无关
表或携带 `schema_migrations`。

## 导入

```bash
uv run --project scripts python scripts/database_transfer.py import \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <current-project-id> --input service.jsonl --mode upsert --tz UTC
```

对于 PostgreSQL，使用
`--target postgresql --dsn-env <ENV_NAME>`，其中环境变量值为
`postgresql+psycopg://user:password@host:5432/database`。

导入的 `--project-id` 是当前选中的目标 Project。适配器先校验文件中的源 Project 闭包，
再将 Application、Service 和 Route 绑定到目标 Project；源 Project 行不会写入目标数据库。
目标 Project 必须存在。Project 的 Environment 是目标运行时配置，不包含在服务传输文件中，
避免导入改变 SSH 凭据绑定、宿主机或工作区。

必须显式指定 `--mode`。`insert` 遇到主键、唯一键或其他约束冲突时失败；`upsert`
按文件中声明的主键或联合主键更新已有行。目标执行正常 Orbit 迁移后已包含种子
Project，因此标准服务迁移使用 `upsert`。导入前，Orbit 会拒绝缺表、多个 Service、
跨 Application 的版本谱系、循环 lineage、缺失引用或混入服务闭包外数据的文件。通过
校验后才调用 dbtalk。

服务 JSONL 不包含 `schema_migrations` 或其他无关表，导入不传表筛选参数。

`--tz` 使用 IANA 时区名称，应在导出和导入时按同一业务约定传递。具体 JSONL 记录格式、
日期时间解释、DATE/TIME 规则和事务语义以 dbtalk 的 `dbtalk-database` skill 为准。

## 环境备份与恢复

环境备份恢复使用独立命令，不和服务部署闭包混用。导出时指定源 Project 与 Environment：

```bash
uv run --project scripts python scripts/database_transfer.py export-environment \
  --source sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <source-project-id> --environment-id <environment-id> \
  --source-secret-env <SOURCE_JWT_SECRET_ENV> \
  --output data/<environment-id>-<timestamp>.jsonl --tz UTC
```

`SOURCE_JWT_SECRET_ENV` 是源 Orbit 实例 `jwt.secret_key` 的环境变量名。脚本用该 Fernet
key 解开 `environment_credential.encrypted_private_key`，在环境 JSONL 中写成明文
`private_key`，以便目标实例重新加密。不要把 Fernet key 写到命令行、文件名或日志里。

恢复时指定目标 Project：

```bash
uv run --project scripts python scripts/database_transfer.py import-environment \
  --target sqlite --dsn sqlite:///./data/db/pomelo-orbit.db \
  --project-id <target-project-id> --input data/<environment-id>-<timestamp>.jsonl \
  --target-secret-env <TARGET_JWT_SECRET_ENV> --tz UTC
```

`TARGET_JWT_SECRET_ENV` 是目标 Orbit 实例的 `jwt.secret_key` 环境变量名。导入前会确认目标
Project 存在且没有 Environment；存在 Environment 时拒绝，环境恢复没有 `--mode` 覆盖选项。
导入会将 Environment 和关联 SSH 密钥绑定到目标 Project，并使用目标 Fernet key 写回密文。
Environment 的 code 同步使用目标 Project 的 code，避免保留源 Project 身份。
这使同一把 SSH 私钥在目标实例中仍可用于已初始化的远程主机。

环境文件的 scope 固定为 `environment`。SSH 环境含一条匹配的
`environment_credential`；local 环境没有凭据块。为满足 dbtalk 外键检查，文件有零行的
`project` 表块，但不包含 Project 行、`repository_credential`、Gateway 绑定、workspace 文件或
容器。恢复时会清空
`gateway_application_id`，Gateway 需在目标 Project 按正常流程建立。

环境 JSONL 包含明文 SSH 私钥。应限制该文件的权限和保存位置，仅通过批准的安全通道传递，
恢复成功后立即删除；不要将文件内容、私钥、Fernet key 或密文输出到日志、Issue 或聊天记录。
