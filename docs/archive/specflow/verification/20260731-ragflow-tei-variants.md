# RAGFlow 内嵌 TEI CPU/GPU Version 验证记录
最后修改时间: 2026-07-31 12:58:47

Review status: Draft

## 需求对齐

- CPU/GPU RAGFlow+TEI 基线保持两个 Version、一个默认 Service，以及独立 Gateway Application；GPU 只验证数据和 Compose 规格，不宣称运行时验证。
- 初始化 SQL 为 RAGFlow Service 新生成五项运行时变量，不复制当前运行实例的配置；Service 全部导出为停止态，Gateway 保持空运行时配置。
- Version Component 的 `pull_policy` 在 SQLite/MySQL schema、Go 领域模型、Proto/HTTP、MCP 和前端均为必填的 `always|missing|never`；新建表单默认值为 `missing`。

## 实际改动

- `000023_application` 现有迁移合并了 `pull_policy` 的非空枚举约束，并保留开发库已有的 `content_masked` 挂载列，使迁移空库可导入导出 SQL。
- 开发 SQLite 已原地重建 `version_component`，全部 14 条记录具有有效拉取策略；外键检查通过。
- 导出器为标准 seed Project 使用 `INSERT OR IGNORE`，避免新库迁移预置默认 Project 后产生主键或唯一键冲突。

## 验收结果

- `task proto`、`task sqlc`：通过。
- `python scripts/test_export_ragflow_tei_baseline.py`：3 个测试通过。
- 真实导出 SQL 导入完整 SQLite 迁移空库：2 个 Application、3 个 Version、13 个组件、2 个停止态 Service、1 个 GPU device request；RAGFlow 运行时变量键集正确，`foreign_key_check` 为 0。
- `go test ./internal/api/http/handler/application ./internal/application/application/usecase ./internal/application/deployment/usecase ./internal/application/gateway/usecase ./internal/repository/impl/sqlc/application`：通过。
- `uv run pytest tests/test_version_specs.py tests/test_server.py tests/test_orbit_client.py`（`mcp/`）：30 个测试通过。
- `yarn --cwd web test componentForm`：9 个测试通过；`yarn --cwd web typecheck`：通过。
- `git diff --check`：通过。

## 范围与风险

- 当前 GPU 机器条件仍不具备，GPU 容器运行验证仍未执行。
- 基线每次重新导出都会重新生成 RAGFlow 初始化变量，预期产生 SQL diff；测试只检查键集和可导入性，不比较密文内容。
- 工作区存在本轮之外的未提交修改；本记录不将其视为本次交付的一部分。

## 结论

实现和验证符合当前已接受需求。初始化 SQL 可用于基于标准迁移的下一次开发实例初始化。
