# `database_transfer.py` - SQLite / MySQL 表数据传输

Doc role: local script reference. 与代码冲突时以代码为准。

## 能力

一个脚本提供全表或服务配置范围的 SQLite/MySQL 数据传输，不迁移 DDL：

- SQLite 全表或服务配置导出文件
- SQLite 文件导入
- MySQL 全表或服务配置导出文件
- MySQL 文件导入
- SQLite 文件转换为 MySQL 文件
- MySQL 文件转换为 SQLite 文件

默认 `--scope all` 传输所有普通表。`--scope service-config` 传输服务配置完整依赖集合：`project`、应用、版本、版本组件及其 env/端点/挂载/依赖/健康检查/资源/tmpfs/ulimit/设备明细、网关配置、服务及其 env/组件覆盖，以及路由；不包含部署历史、任务、日志或容器卷数据。

传输文件保留源字段值：文本不会改写为其他时间格式或时区，JSON、环境变量、受控文件内容、服务密钥和证书字段均按原值保存和导入。BLOB 以可逆 Base64 payload 保存。生成文件只能由本脚本导入或转换；目标数据库必须预先具备可接受这些原始值的兼容 schema。

## 用法

```bash
# SQLite 导出（--sqlite-path 与 --output 均为必填）
python scripts/database_transfer.py export --source sqlite --sqlite-path path/to/source.db --output path/to/transfer.sqlite.sql

# 仅导出可完整恢复的服务配置、应用版本组件与路由
python scripts/database_transfer.py export --source sqlite --scope service-config --sqlite-path path/to/source.db --output path/to/service-config.sqlite.sql

# SQLite 文件导入
python scripts/database_transfer.py import --target sqlite --sqlite-path path/to/target.db --input path/to/transfer.sqlite.sql --replace

# SQLite 文件转 MySQL 文件
python scripts/database_transfer.py convert --from sqlite --to mysql --input path/to/transfer.sqlite.sql --output path/to/transfer.mysql.sql

# MySQL 文件导入
python scripts/database_transfer.py import --target mysql --mysql-host 127.0.0.1 --mysql-port 3306 --mysql-user USER --mysql-password PASSWORD --mysql-database DATABASE --input path/to/transfer.mysql.sql --replace

# MySQL 导出
python scripts/database_transfer.py export --source mysql --mysql-host 127.0.0.1 --mysql-port 3306 --mysql-user USER --mysql-password PASSWORD --mysql-database DATABASE --output path/to/transfer.mysql.sql

# MySQL 服务配置导出
python scripts/database_transfer.py export --source mysql --scope service-config --mysql-host 127.0.0.1 --mysql-port 3306 --mysql-user USER --mysql-password PASSWORD --mysql-database DATABASE --output path/to/service-config.mysql.sql

# MySQL 文件转 SQLite 文件
python scripts/database_transfer.py convert --from mysql --to sqlite --input path/to/transfer.mysql.sql --output path/to/transfer.sqlite.sql
```

SQLite 导出在只读事务内运行 `integrity_check` 和 `foreign_key_check`。导入时，`--replace` 会在同一数据库连接内暂时关闭外键检查、清空文件包含的表、插入完整数据后恢复检查；不使用 `--replace` 时只追加记录。

## 边界

`service-config` 要求源库包含当前服务配置集合的全部表；缺失任一表会阻止导出，避免产生不完整的可恢复配置。该脚本处理表数据和表字段，不迁移 DDL、容器、卷、工作区、运行日志或外部服务数据。跨 SQLite/MySQL 时，目标 schema 由目标数据库自行管理，并必须接受归档中的原始字段值。
