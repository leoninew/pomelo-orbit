# Orbit 服务级 JSONL 导入导出适配验证
最后修改时间: 2026-08-22 21:19:38

Review status: Accepted

Mode: standard

## Requirement Alignment

- 服务适配层只调用 `dbtalk database export` / `import`，并保留 Orbit 服务部署闭包的选择和领域校验；不重新实现全库连接、SQL 方言、值编码或导入事务。
- 导出/导入使用 canonical `--dsn` 或 `--dsn-env`，二者互斥；服务文件仍在调用 dbtalk 前完成表集、单 Service、引用和版本 lineage 校验。
- `scripts/` 新增不打包的 uv 项目。`cryptography` 与 `ulid-py` 是运行时依赖，pytest、Ruff 和 MyPy 仅用于开发检查；pytest 直接收集现有 `unittest.TestCase`。

## Plan Alignment

Plan 中没有独立 Spec，按已接受的 Requirement 与 Plan 核对。

| Plan 项 | 结果 |
| --- | --- |
| dbtalk CLI 适配与服务闭包校验 | 已实现并由定向测试覆盖。 |
| 文档与 `transfer-database-data` skill 收敛 | 已完成；统一使用 dbtalk canonical DSN 与 uv 运行入口。 |
| scripts uv 工具环境 | 已完成；`package = false`，无构建后端、`src/` 布局或包入口。 |
| 定向 Python 与 Go 验证 | 脚本检查和 Go 测试通过。 |

## Actual Diff Summary

- `scripts/database_transfer.py` 收敛为 dbtalk CLI 服务闭包适配器，使用 `--dsn` / `--dsn-env`，并修复版本 lineage 的可空类型收窄。
- `scripts/test_database_transfer.py` 保留服务闭包、输入拒绝、CLI 参数、临时文件与敏感错误测试；由 pytest 收集执行。
- `scripts/pyproject.toml`、`scripts/uv.lock`、`scripts/.gitignore` 与 `Taskfile.yml` 建立局部 uv 环境及 `task scripts:check`。
- 脚本文档、服务迁移指南和 skill 改为使用 `uv run --project scripts`；`version` Task 也使用锁定环境。
- Ruff 对 `cert.py`、`manage.py`、服务传输脚本及其测试应用了机械格式化；Ruff lint 仅启用基础语法和未定义名称检查，未借此重构既有运维逻辑。

## Expected Vs Actual Files

| 预期范围 | 实际文件 |
| --- | --- |
| 服务传输适配与测试 | `scripts/database_transfer.py`、`scripts/test_database_transfer.py` |
| 服务迁移文档与 skill | `docs/guides/service-data-transfer.md`、`skills/transfer-database-data/**`、`docs/INDEX.md` |
| 脚本工具环境 | `scripts/{pyproject.toml,uv.lock,.gitignore,README.md}`、`Taskfile.yml` |
| 运行依赖脚本文档 | `scripts/{cert.py,cert.md,gen_ulid.md,version-calc.py,version-calc.md}`、`docs/guides/certificate-management.md` |
| 过程文档 | `docs/{requirement,plan,verification}/20260819-jsonl-database-transfer.md` |

工作区还包含独立的 `docs/requirement/20260820-scheduled-tasks.md`；它不属于本任务。`docs/INDEX.md` 和 `scripts/README.md` 的先前清理内容也不构成服务传输实现行为。

## Acceptance

- [x] 服务数据只通过 dbtalk JSONL CLI 读写，Orbit 不再维护全库 SQL 或方言实现。
- [x] 导出选择请求服务及其 Project、Application、Version lineage、组件、Gateway、覆盖和 Route 闭包，并排除无关服务。
- [x] 导入在调用 CLI 前验证表集、单个服务、应用归属、引用和版本谱系。
- [x] 文档与 skill 明确 dbtalk 边界、目标 schema 前置条件和 canonical DSN 用法。
- [x] uv 管理运行时及开发依赖；pytest、Ruff 和 MyPy 具有可重复的脚本检查入口。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `task scripts:check` | Passed: Ruff format/lint、MyPy（5 个实现文件）和 pytest（18 passed）。 |
| `uv --directory scripts lock --check` | Passed。 |
| `uv run --project scripts python -c "import cryptography, ulid"` | Passed。 |
| `uv run --project scripts python scripts/database_transfer.py --help` | Passed。 |
| `uv run --project scripts python scripts/cert.py --help` | Passed。 |
| `task version` | Passed，使用 uv 环境完成只读版本计算。 |
| `go test ./cmd/... ./internal/...` | Passed。 |
| `dbtalk database --help` | Passed，使用本地 dbtalk 环境验证。 |
| Orbit SQLite service export/import | Passed：默认 `dbtalk` 命令导出 22 张表后，导入临时数据库副本成功。 |
| `task check` | 未完成：此前运行 124 秒后超时且无输出。 |

## Risks And Incomplete Items

1. 已使用本地 dbtalk 对 Orbit SQLite 服务闭包执行真实导出和导入验证。
2. `task check` 未完成。它不覆盖本次修改的前端或 Go 源码；脚本专用检查和 Go 测试已单独通过。

## Conclusion

服务级 dbtalk 适配、外键拓扑表顺序和脚本 uv/pytest 工具环境已完成验证。
