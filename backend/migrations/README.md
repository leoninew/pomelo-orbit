# Database Migrations

This directory contains migration files for database schema management.

## Naming Convention

```
vMAJOR.MINOR.PATCH__description.sql   # DDL / 原生 SQL
vMAJOR.MINOR.PATCH__description.json  # 结构化数据操作（insert / update / delete）
```

Examples:
- `v0.1.0__init_schema.sql` - 初始化表结构
- `v0.4.1__init_admin_user.json` - 插入初始数据
- `v0.4.4__nginx.sql` - 插入应用配置

## How It Works

1. 应用启动时自动按文件名排序执行所有未执行的迁移
2. 每个迁移只执行一次，执行记录存储在 `__migration_history` 表
3. MD5 校验和确保已执行的迁移文件不被修改

## SQL 格式（.sql）

适用于 DDL 和复杂 SQL，直接写标准 SQL 语句：

```sql
-- v0.1.0__init_schema.sql
CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE
);
```

## JSON 格式（.json）

适用于结构化数据操作，支持 `insert` / `update` / `delete`，所有值均参数化执行。

文件为 JSON 数组，每个元素为一个操作：

```json
[
  {
    "type": "insert",
    "table": "表名",
    "data": [
      { "列名": "值", ... }
    ],
    "comment": "可选说明"
  }
]
```

### insert

```json
{
  "type": "insert",
  "table": "user",
  "data": [
    { "id": "abc", "username": "admin", "password_hash": "..." }
  ],
  "comment": "初始化管理员"
}
```

### update

```json
{
  "type": "update",
  "table": "user",
  "data": { "username": "new_name" },
  "where": { "id": "abc" }
}
```

### delete

```json
{
  "type": "delete",
  "table": "user",
  "where": { "id": "abc" }
}
```

> 注意：JSON 格式中的值均为字面量，不支持 SQL 函数（如 `datetime('now')`），时间字段请使用 ISO 8601 字符串（如 `"2024-01-01T00:00:00Z"`）。

## Important Notes

- **不要修改已执行的迁移文件** — 会导致 checksum 校验失败
- 开发阶段如需修改，手动更新 `__migration_history` 中对应记录的 checksum

## View Migration History

```sql
SELECT * FROM __migration_history ORDER BY executed_at;
```
