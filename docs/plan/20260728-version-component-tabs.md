# 版本组件结构化表单计划
最后修改时间: 2026-07-28 18:30:43

Review status: Accepted

## Implementation steps

1. 原地更新 SQLite/MySQL `000023_application` 建表 SQL，新增 Component 关联表；不改变其他 JSON 配置的表结构。
2. 编写 `scripts/migrate_sqlite_version_component.py`，实现 `--check`、备份、事务转换、结构/引用校验和 `--apply`；以临时数据库副本覆盖脚本测试后，再原地更新 `data/db/pomelo-orbit.db`。
3. 将 Component JSON 字段替换为结构化 Proto、DTO、模型、SQL 查询和仓储写入；新增 Component CRUD API，并移除旧 JSON mapper/校验路径。
4. 让 Renderer 直接消费结构化领域数据，补齐依赖环、引用改名/删除和有限策略字段的校验与测试。
5. 将版本详情缩减为组件目录，新增独立组件详情/编辑页面及结构化表单；移除旧组件编辑模态窗和相关前端 JSON 表单工具。
6. 运行 Python 脚本检查/转换验证、Go 格式化/vet/测试、前端 test/lint/typecheck 和 diff 检查。

## Files to change

`000023_application` SQLite/MySQL 建表 SQL、`sql/query/application/version.sql`、`scripts/migrate_sqlite_version_component.py`、Component 相关 Proto/DTO/model/repository/usecase/renderer、前端 API/router/views/components/i18n/tests，以及开发 SQLite 数据库。

## Verification plan

- Python 脚本针对临时数据库副本的 `--check` / `--apply` 测试，以及开发库转换后的结构和数据核对。
- Go 格式化、vet、应用/部署 usecase 测试和 HTTP mapper 测试。
- `yarn --cwd web test`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。
- `git diff --check`。

## Assumptions and risks

只处理 Component 配置结构化；不扩展 Compose 资源或运行时字段。Component 改名和删除的关联处理必须由后端事务执行。

## Rollback

SQLite 脚本在提交前失败时回滚；提交后使用脚本创建的备份恢复。源码回退需将 `000023_application` 和相关代码整体恢复，不保留旧/新模型并存。
