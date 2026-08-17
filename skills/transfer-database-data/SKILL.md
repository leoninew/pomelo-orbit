---
name: transfer-database-data
description: "使用仓库内 scripts/database_transfer.py 安全地导出、转换或导入 SQLite/MySQL 的全表数据或完整服务配置（项目、应用、版本、组件、服务覆盖、网关配置和路由）。用于跨库恢复服务配置、生成可恢复传输文件、SQLite/MySQL 转换或替换目标库数据；涉及 schema、其他部分业务表或直接执行生成 SQL 时，先确认脚本能力与用户授权。 Safely export, convert, or import complete SQLite/MySQL table data or complete service configuration (projects, applications, versions, components, service overrides, gateway configuration, and routes) with scripts/database_transfer.py. Use for cross-database service-configuration recovery, transfer archives, SQLite/MySQL conversion, or replacing target data; confirm script capability and user authorization for schema, other partial-table, or direct-SQL workflows."
---

# 传输数据库表数据 / Transfer Database Table Data

从仓库根目录运行 `scripts/database_transfer.py`。需要更完整的格式说明时阅读 [`scripts/database_transfer.md`](../../scripts/database_transfer.md)；行为冲突时以脚本代码为准。

Run `scripts/database_transfer.py` from the repository root. Read [`scripts/database_transfer.md`](../../scripts/database_transfer.md) for format details; the script is authoritative when documentation differs.

## 操作边界 / Operational Boundaries

- 使用默认 `--scope all` 传输源数据库中的全部普通表；用户要求恢复应用、版本、组件、服务覆盖、网关配置和路由时，使用 `--scope service-config`。该范围包含其完整依赖表，不包含部署历史、任务或日志。 / Use the default `--scope all` for every ordinary source table. Use `--scope service-config` to restore applications, versions, components, service overrides, gateway configuration, and routes. This scope includes its complete dependency tables but excludes deployment history, tasks, and logs.
- 脚本不迁移 DDL。导入前确认目标库已经通过当前迁移建立兼容 schema。 / The script does not migrate DDL; ensure the target has a compatible current schema before import.
- 传输文件保留原始字段值，不重写时间文本、时区、JSON、环境变量、密钥、受控文件内容或证书。它可能含敏感配置或凭据；默认只报告路径和摘要，用户明确要求查看内容时先确认展示范围并对凭据脱敏。 / Transfer files preserve original field values without rewriting timestamp text, time zones, JSON, environment variables, secrets, controlled-file content, or certificates. They may contain sensitive configuration or credentials; report paths and summaries by default, and confirm inspection scope and redact credentials when content is explicitly requested.
- `export` 只读数据库但会覆盖已有输出文件。写入前解析绝对路径；文件已存在且用户未明确要求覆盖时先请求确认。 / `export` is database-read-only but replaces an existing output file. Resolve the absolute path and confirm an unrequested overwrite.
- `import` 会写目标库；`--replace` 会先删除传输文件覆盖的所有表数据。确认精确目标库，并且仅在用户明确要求替换时添加 `--replace`。 / `import` writes the target; `--replace` first deletes all data from every table in the archive. Confirm the exact target and add it only when explicitly requested.
- 生成文件中的可读 SQL 始终包含 `DELETE`。默认通过本脚本导入或转换；仅当用户明确要求直接执行并确认删除影响时，才交给数据库客户端。 / Rendered SQL always contains `DELETE`. Use the script by default; execute it through a database client only after the user explicitly requests it and confirms the deletion impact.

## 工作流 / Workflow

1. 确认源、目标、驱动、文件路径、导出范围，以及操作是追加还是替换。服务配置恢复选择 `service-config`；其他部分数据不将其替代为该范围。先检查路径和目标 schema；默认不输出数据内容或密钥，用户明确要求时按确认范围展示并脱敏。 / Confirm source, target, drivers, paths, export scope, and append-versus-replace intent. Select `service-config` for service-configuration recovery; do not substitute it for unrelated partial data. Inspect paths and target schema first; hide data and secrets by default, and disclose only the confirmed, redacted scope when explicitly requested.
2. MySQL 操作前确认 Python 环境可导入 `pymysql`。从现有配置安全取得连接字段；默认不在日志或回复中回显密码。 / Before MySQL operations, ensure Python can import `pymysql`. Obtain connection fields safely and avoid echoing passwords by default.
3. 同驱动传输：导出源文件，然后用相同驱动导入目标。跨驱动传输需要依次完成导出、转换和导入。 / For same-driver transfers, export then import with the same driver. Cross-driver transfers require export, conversion, and import in sequence.
4. 导入成功后按表比较行数，并运行目标数据库完整性检查。默认只报告表名、计数和检查结果；用户明确要求时再扩大报告范围。 / After import, compare per-table row counts and run target integrity checks. Report table names, counts, and check results by default; expand only when explicitly requested.
5. 保留传输文件供用户确认。除非用户明确要求，不删除包含数据的传输文件。 / Retain the transfer file for user confirmation; do not delete it unless explicitly requested.

## 命令 / Commands

SQLite 导出与导入 / SQLite export and import:

```powershell
python scripts/database_transfer.py export --source sqlite --sqlite-path <source.db> --output <transfer.sqlite.sql>
python scripts/database_transfer.py import --target sqlite --sqlite-path <target.db> --input <transfer.sqlite.sql> [--replace]

# 服务配置、应用版本组件与路由
python scripts/database_transfer.py export --source sqlite --scope service-config --sqlite-path <source.db> --output <service-config.sqlite.sql>
python scripts/database_transfer.py import --target sqlite --sqlite-path <target.db> --input <service-config.sqlite.sql> [--replace]
```

MySQL 导出与导入 / MySQL export and import:

```powershell
python scripts/database_transfer.py export --source mysql --mysql-host <host> --mysql-port <port> --mysql-user <user> --mysql-password <password> --mysql-database <database> --output <transfer.mysql.sql>

# MySQL 服务配置、应用版本组件与路由
python scripts/database_transfer.py export --source mysql --scope service-config --mysql-host <host> --mysql-port <port> --mysql-user <user> --mysql-password <password> --mysql-database <database> --output <service-config.mysql.sql>
python scripts/database_transfer.py import --target mysql --mysql-host <host> --mysql-port <port> --mysql-user <user> --mysql-password <password> --mysql-database <database> --input <transfer.mysql.sql> [--replace]
```

跨驱动转换 / Cross-driver conversion:

```powershell
python scripts/database_transfer.py convert --from sqlite --to mysql --input <transfer.sqlite.sql> --output <transfer.mysql.sql>
python scripts/database_transfer.py convert --from mysql --to sqlite --input <transfer.mysql.sql> --output <transfer.sqlite.sql>
```

## 完成条件 / Completion

- 命令成功退出，目标表计数符合预期，完整性检查无错误。 / Commands succeed, target table counts match expectations, and integrity checks pass.
- SQLite 使用 `PRAGMA integrity_check` 与 `PRAGMA foreign_key_check`；MySQL 至少确认导入事务成功、外键检查已恢复并核对表计数。 / For SQLite, run `PRAGMA integrity_check` and `PRAGMA foreign_key_check`; for MySQL, confirm transaction success, restored foreign-key checking, and table counts.
- 报告源和目标驱动、数据库标识、传输文件路径、追加或替换模式及验证结果，隐藏密码和数据内容。 / Report source and target drivers, database identifiers, transfer-file path, append/replace mode, and verification results while hiding passwords and data contents.
