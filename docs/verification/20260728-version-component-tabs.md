# 版本组件结构化表单验证
最后修改时间: 2026-07-28 21:34:15

Review status: Draft

## Requirement alignment

实现已移除 Version Component API 和领域模型中的 `*_json` 配置字段，以结构化 Proto、DTO、持久化关联表和 Compose Renderer 取代。Version 详情缩减为组件目录，Component 独立详情页支持未发布版本的一次保存编辑和已发布版本的同布局只读展示。

## Spec alignment

- `000023_application` 的 SQLite/MySQL 建表 SQL、查询和 sqlc 代码包含 Component 关联表。
- HTTP、usecase、repository、renderer、Gateway 编译路径均直接处理结构化 Component 数据。
- MCP 输入规格与客户端 payload 使用结构化项；补充了列表模型序列化的泛型类型约束。
- 前端包含独立 Component 路由、结构化表单、国际化文案和旧编辑模态窗移除。

## Plan alignment

计划的后端、MCP、前端、SQL、Proto、测试和过程文档变更均在暂存范围内。两份迁移脚本已在工作区验证，但按用户要求不纳入本次暂存。开发 SQLite 库本轮未执行 `--apply`：只读核对显示它已经是目标规范化 schema，因此迁移脚本正确拒绝将非旧 JSON schema 再次转换。

## Actual diff summary

暂存范围覆盖过程文档、SQLite/MySQL schema、Proto 和生成代码、Go 的 HTTP/usecase/repository/renderer、MCP，以及前端路由、API、表单、视图和测试。旧 Component 编辑模态窗和 JSON 表单工具已删除。两份迁移脚本保留在未跟踪工作区。

## Expected vs actual files

| 预期范围 | 实际情况 |
| --- | --- |
| Component 结构化存储和迁移脚本 | SQL/query/generated code 已暂存；SQLite 转换脚本及测试已验证但未暂存 |
| API、领域校验和 Compose Renderer | 已修改 HTTP mapper/route、usecase、repository、runtime config、renderer 和测试 |
| MCP Component 输入 | 已修改 client/tool/spec/test；补齐 mypy 泛型问题 |
| 独立 Component 前端页面 | 已新增详情页和表单工具，更新路由/API/i18n，删除旧模态窗 |
| 需求、规格、计划与验证记录 | 已包含同一 feature 的四份阶段文档 |

未发现与该 feature 无关的 `scripts/tei-bge-m3` 缓存差异。

## Acceptance checklist

- [x] Component 请求/响应模型不再暴露 `*_json` 字段，改用嵌套结构化消息。
- [x] Component 结构化配置使用关联表持久化，Compose Renderer 不再解析 Component JSON。
- [x] SQLite 转换脚本支持只读预检、备份和事务性 `--apply`，并覆盖支持/拒绝路径。
- [x] Version 页面提供 Component 目录，独立页面提供结构化编辑与只读展示。
- [x] 全部后端、MCP、前端自动化检查和格式/diff 检查通过。

## Test results

| 命令 | 结果 |
| --- | --- |
| `python scripts/test_migrate_sqlite_version_component.py` | 2 passed |
| `go test ./...` | passed |
| `go vet ./...` | passed |
| `gofmt -l internal` | no output |
| `uv run --directory mcp pytest` | 35 passed, 1 Docker test deselected |
| `uv run --directory mcp ruff check src tests` | passed |
| `uv run --directory mcp mypy` | passed, 14 source files |
| `yarn --cwd web test` | 9 files, 44 tests passed |
| `yarn --cwd web lint` | passed |
| `yarn --cwd web typecheck` | passed |
| `git diff --cached --check` | passed |

## Development SQLite check

`data/db/pomelo-orbit.db` 已含 8 个 `version_component` 行及全部目标关联表。关联记录计数为：arguments 17、dependencies 4、env 33、healthchecks 6、healthcheck args 12、mounts 9、networks 0、ports 3、resources 1、tmpfs 1、ulimits 1；`PRAGMA foreign_key_check` 返回 0 项。

## Risks and incomplete items

- 开发库已经不是旧 JSON schema，因此本轮无法以其作为脚本 `--apply` 的输入；脚本的读写转换路径已由临时数据库测试覆盖。
- 实际开发应用的 application list API 当前返回 500，组件页面的浏览器复核使用了 API mock；真实后端联调需在该既有 API 问题恢复后补做。

## Conclusion

自动化验证通过，结构化 Component 改造可进入人工验收。上述两个外部状态限制不影响源码测试结论，但应在合并前完成真实应用 API 的浏览器联调。
