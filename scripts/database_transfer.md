# `database_transfer.py` - SQLite / MySQL 表数据传输

Doc role: local script reference. 与代码冲突时以代码为准。

## 能力

一个脚本提供六项操作，所有普通表均从源数据库动态读取，不包含业务表白名单、默认数据库路径或领域规则：

- SQLite 导出文件
- SQLite 文件导入
- MySQL 导出文件
- MySQL 文件导入
- SQLite 文件转换为 MySQL 文件
- MySQL 文件转换为 SQLite 文件

传输文件包含所有普通表和全部字段值。BLOB 以可逆 Base64 payload 保存，并同时渲染为目标方言的 SQL 字面量。生成文件只能由本脚本导入或转换；目标数据库必须预先具备兼容 schema。

MySQL 导出会排除数据库自动计算的生成列，由目标 schema 在导入时重新计算。SQLite 导入 MySQL 时，Go 偏移时间文本和 RFC 3339 时间文本会转换为 UTC `DATETIME` 值；其他文本保持原样。

## 用法

```bash
# SQLite 导出（--sqlite-path 与 --output 均为必填）
python scripts/database_transfer.py export --source sqlite --sqlite-path path/to/source.db --output path/to/transfer.sqlite.sql

# SQLite 文件导入
python scripts/database_transfer.py import --target sqlite --sqlite-path path/to/target.db --input path/to/transfer.sqlite.sql --replace

# SQLite 文件转 MySQL 文件
python scripts/database_transfer.py convert --from sqlite --to mysql --input path/to/transfer.sqlite.sql --output path/to/transfer.mysql.sql

# MySQL 文件导入
python scripts/database_transfer.py import --target mysql --mysql-host 127.0.0.1 --mysql-port 3306 --mysql-user USER --mysql-password PASSWORD --mysql-database DATABASE --input path/to/transfer.mysql.sql --replace

# MySQL 导出
python scripts/database_transfer.py export --source mysql --mysql-host 127.0.0.1 --mysql-port 3306 --mysql-user USER --mysql-password PASSWORD --mysql-database DATABASE --output path/to/transfer.mysql.sql

# MySQL 文件转 SQLite 文件
python scripts/database_transfer.py convert --from mysql --to sqlite --input path/to/transfer.mysql.sql --output path/to/transfer.sqlite.sql
```

SQLite 导出在只读事务内运行 `integrity_check` 和 `foreign_key_check`。导入时，`--replace` 会在同一数据库连接内暂时关闭外键检查、清空文件包含的表、插入完整数据后恢复检查；不使用 `--replace` 时只追加记录。

## 边界

该脚本处理表数据和表字段，不迁移 DDL、容器、卷、工作区、运行日志或外部服务数据。跨 SQLite/MySQL 时，目标 schema 由目标数据库自行管理。
