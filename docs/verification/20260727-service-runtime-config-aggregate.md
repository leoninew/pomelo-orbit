# Service 聚合运行时配置验证记录
最后修改时间: 2026-07-27 16:44:42

Review status: Draft

## Verification scope

- 流程模式：标准模式 / `standard`
- 验证对象：暂存区的 Service 聚合运行时配置改造，关联 Requirement 和 Plan 均为 `Accepted`。
- 未暂存的首次部署 `instance_key` 改动会影响编译和 Web 类型检查，作为当前工作区上下文参与检查；不属于本次暂存交付范围，也没有单独判为问题。
- 按用户要求，不运行 SQLite/MySQL、Docker、浏览器或其他集成测试。

## Requirement alignment

1. `service.runtime_config_json`、`model.Service.RuntimeConfig`、Service Repository 和 Service HTTP 接口已改为普通 K/V 配置读写。
2. `runtime_env` Credential、组件秘密引用、`.runtime` 环境文件和 Compose `env_file` 路径已从暂存实现中移除。
3. 部署和重启在创建 Deployment 时复制 Service K/V 到 `options_json.runtime_config`；worker 使用该快照解析 Version placeholder，并对多余 key 写入日志 warning。
4. Service 创建、读取、整体替换配置，以及 Version 与 Service 配置匹配的校验路径均已实现并通过静态/定向单元检查。

## Plan alignment

- 暂存变更覆盖迁移与 SQLC、Service 聚合用例、Version 校验、部署快照、Proto/HTTP、Web、MCP、RAGFlow 脚本和活文档，符合计划列出的主要范围。
- 当前未暂存部署改动把部署输入从计划中的 `service_id` 调整为 `instance_key`。这是另一项正在进行的改动；当前工作区的 Go、TypeScript 与 MCP 检查均通过，因此不作为本次暂存验证的范围偏差或缺陷。

## Actual diff summary

- 暂存区：110 个文件，约 1,830 行新增、2,701 行删除。
- 数据层：Service 表增加 `runtime_config_json`，查询、SQLC 和 Repository 全程携带 K/V。
- 服务层：增加 Service 创建与运行时配置 GET/PUT；配置替换不改 Service 状态、Version 或 Deployment。
- 部署层：Deploy/Restart 快照配置，worker 仅从快照解析并渲染；Credential 和 `.runtime` 依赖删除。
- 前端与 MCP：删除 runtime_env/secret-ref 流程，改为 Service K/V 编辑和 Service 定位的部署流程。

## Acceptance checklist

- [x] Service K/V 持久化与读写路径已由 Go 编译、vet、定向单测和类型检查覆盖。
- [x] 运行时 Credential 与组件秘密引用从持续部署链路移除。
- [x] Deploy/Restart 快照 Service 配置，worker 从快照解析；相关 Deployment 用例定向单测通过。
- [x] Version 与已关联 Service 的必填 placeholder 校验代码已接入并完成编译检查。
- [x] Web 与 MCP 的 API 类型、lint 和定向单测通过。
- [ ] SQLite/MySQL 实库迁移、现有 `orbit` 数据库清理和 Docker 渲染未执行，按用户要求跳过集成测试。
- [ ] Web Prettier 格式检查失败，需先处理后才能将本验证记录改为 `Accepted`。

## Command results

| 命令 | 结果 |
| --- | --- |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./internal/common/runtimeconfig ./internal/application/deployment/usecase -run ...` | 通过 |
| `go test ./internal/application/service/usecase -run '^TestDeleteService.*$'` | 通过 |
| `go test ./internal/application/application/usecase -run '^$'` | 通过，仅编译，不执行集成用例 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web lint` | 通过 |
| `yarn --cwd web format` | 失败，8 个文件不符合 Prettier |
| `uv run ruff check src tests` | 通过 |
| `uv run mypy` | 通过 |
| `uv run pytest tests/test_service_runtime_config.py tests/test_server.py -q` | 2 通过 |
| `python -m unittest scripts/test_ragflow_initialize.py` | 17 通过 |
| `git diff --cached --check` / `git diff --check` | 均通过 |

## Formatting issue

`yarn --cwd web format` 报告以下文件需要 Prettier 格式化：

- `web/src/api/application/application.ts`
- `web/src/api/service/service.ts`
- `web/src/i18n/locales/en-US.ts`
- `web/src/i18n/locales/zh-CN.ts`
- `web/src/views/application/components/VersionComponentRuntimeDialog.vue`
- `web/src/views/application/VersionDetail.vue`
- `web/src/views/service/ServiceDetail.vue`
- `web/src/views/service/ServicePage.vue`

未自动执行 `format:fix`，避免在验证阶段改写暂存和未暂存文件。

## Risks and incomplete items

1. Service 配置读写与部署快照已完成非集成检查，但实际 SQLite/MySQL 迁移和 Docker 运行路径本轮未执行。
2. 暂存和未暂存代码共同参与编译；提交前应再次在最终暂存状态运行同一组检查。
3. Web 格式问题是当前唯一已发现的检查失败项。

## Conclusion

功能、类型、lint 和定向单元检查通过，未发现运行时配置聚合改造的功能性问题。由于 Web Prettier 检查失败，且集成测试按要求未执行，本验证记录保持 `Draft`。
