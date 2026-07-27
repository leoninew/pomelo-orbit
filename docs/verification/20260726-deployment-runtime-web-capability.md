# 部署运行时能力 Web 支持验证
最后修改时间: 2026-07-27 10:26:43

## Review status

Draft

## Verification basis

- Requirement: `docs/requirement/20260726-deployment-runtime-web-capability.md` (`Accepted`)
- Plan: `docs/plan/20260726-deployment-runtime-web-capability.md` (`Accepted`)
- Flow mode: 标准模式 / `standard`
- 用户限定：不执行 Go integration、Docker integration、浏览器 E2E 或新的运行时部署。

## Requirement alignment

核心 Web 实现与已接受 Requirement 对齐：Credential 页面支持 `runtime_env` 的结构化键值编辑；Version Component 支持高级运行时字段和 `secret_env_refs`；Service 详情通过受权的 Service ID 读接口展示当前持久化 Credential 配置，并明确其用于下一次部署。

前端 diff 未引入浏览器到 MCP stdio、Docker CLI 或运行时探测的调用。`runtime_env` 值通过既有 Credential/Service HTTP 模型读取，不写入 Version、Deployment 或浏览器日志。

## Plan alignment

Plan Step 1 至 Step 5 的后端契约、受权读模型、Credential 体验、Version Component 高级配置与 Service 卡片均有对应实现。Proto 的 Go 和 TypeScript 生成物已由当前生成命令重建并保持与暂存内容一致。

本轮还有后续用户明确授权的范围扩展，不属于原 Plan 的 Web 实现边界：

- 停止 Service 删除、无 Component 时的部署前校验、Deployment 操作日志与容器日志分栏、Gateway 默认 REST API URL。
- MCP `runtime_compose_ps` 改为包含已停止容器，便于读取刚退出服务的日志。
- RAGFlow 拆分/聚合 Compose 示例、对应 `.env.example`、初始化器的聚合 Version 支持及聚合组件 logical mount 隔离。

这些扩展与原 Web 目标无冲突，但应在提交时与核心 Web 变更分组审查。

## Actual diff summary

- 扩展 Service Proto、HTTP route/handler/mapper、依赖装配和 usecase，新增 Service runtime env 响应与停止 Service 删除能力。
- 扩展 Credential、Version、Service、Deployment、Gateway 页面及中英文文案；新增 `RuntimeEnvEditor`、Component 高级配置 Dialog 与运行时环境变量工具测试。
- 为 Deployment 增加前置 Component 校验与操作/容器日志展示；Gateway 创建默认使用 `http://localhost:8080`。
- 新增 MCP 已停止容器诊断覆盖及单测；新增 RAGFlow 拆分/聚合 Compose 样例、环境变量模板和初始化器聚合 Version 流程，并为聚合组件分离持久化目录。

## Expected vs actual changed files

| 范围 | 结果 |
|---|---|
| Service Proto、Go/TypeScript 生成物、Service HTTP/usecase/装配 | 已修改，符合 Plan Step 1 |
| Credential、Version、Service Web 页面、i18n 与运行时工具单测 | 已修改，符合 Plan Step 2 至 Step 5 |
| 部署前 Component 校验、停止 Service 删除、Deployment 日志与 Gateway 默认值 | 后续用户授权的范围扩展 |
| MCP 退出容器诊断 | 后续用户授权的范围扩展；不新增浏览器调用路径 |
| `scripts/ragflow-*`、`ragflow_initialize.py` | 后续用户授权的 RAGFlow 验证资产，不属于原 Web Requirement |
| `web/vite.config.ts.timestamp-1785110012835-eae8931ad9815.mjs` | Vite 临时文件，非功能实现，应从提交中排除或单独处理 |

## Acceptance criteria checklist

- [x] `runtime_env` Credential 的键值解析、环境变量键、重复键和空值校验有 Vitest 覆盖；Web lint/typecheck/format/test 通过。
- [x] Version Component 的高级运行时字段和 `secret_env_refs` 已进入既有 update payload，且 Go/TypeScript 静态检查通过。
- [x] Service runtime env 的 Proto、HTTP route、handler、受权 usecase 和 Service 详情类型契约已实现并通过编译/静态检查。
- [x] 静态 diff 核对未发现浏览器直连 MCP/Docker；Deployment 表单不要求重新输入 `runtime_env` 值。
- [x] 空 Component 部署被前后端校验拦截；停止 Service 的删除路径已添加单测。
- [ ] Service runtime env 解析的集成测试、Credential/Version/Service 页面级组件测试和浏览器人工流程未执行。
- [x] 聚合 RAGFlow 的 MySQL、Redis、MinIO、Elasticsearch 与 RAGFlow 数据目录已隔离；五个组件通过 MCP 稳定性验证，固定 HTTP Probe 可达。

## Test results

| 类别 | 命令或证据 | 结果 |
|---|---|---|
| Proto 生成 | `./bin/buf generate`；`./bin/buf generate . --template web/buf.gen.yaml --output web` | 通过。首次 Go 生成受 Windows 文件锁影响中断，单独重试后成功，未产生未暂存生成物差异。 |
| Proto lint | `./bin/buf lint` | 未通过。全仓 Proto package 的既有 `PACKAGE_VERSION_SUFFIX` 规则失败，与本次新增 Service 消息无关。 |
| Go 格式 | 对暂存 Go 文件执行 `gofmt -d` | 通过。 |
| Go 静态检查 | `go vet ./cmd/... ./internal/... ./sql` | 通过。 |
| Go 非 integration 测试 | 排除含 `*_integration_test.go` 的 usecase package 后执行 `go test`；Service package 仅运行 `TestDeleteService*` | 通过。筛选误包含 `internal/test/e2e`，因未配置 E2E 环境而跳过，未访问外部环境。 |
| Web | `yarn --cwd web lint`、`typecheck`、`format`、`test` | 全部通过；Vitest 为 9 个文件、45 个用例。 |
| MCP | `make -C mcp check` | 通过；27 项通过，1 项 Docker 集成测试按标记排除。 |
| 初始化器 | `uv --directory mcp run python ../scripts/test_ragflow_initialize.py` 与 Ruff | 通过；17 项测试通过。 |
| Compose 样例 | 各目录 `.env.example` 配合 `docker compose config --quiet` | 6 份 Compose 文件均通过静态解析；未创建容器、网络或卷。 |
| 聚合 RAGFlow 运行时 | MCP 创建 `v0.26.4-cpu-elasticsearch-bundled-isolated-data`、部署并核验 | 通过。五个组件均为 `healthy`，稳定性窗口无重启，`ragflow-cpu:80/` HTTP Probe 为 `reachable`。 |
| Diff | `git diff --cached --check`、`git diff --check` | 通过。 |

## Runtime verification evidence

聚合 Version 曾直接复用拆分 Fixture 的裸 `data` logical mount；由于同一 Application 的组件共用物理 Service 目录，MySQL、Redis、MinIO 和 Elasticsearch 会写入同一宿主目录。初始化器现在将 logical mount 改为 `<component>/data` 或 `<component>/logs`，聚合 Compose 示例对应使用 `./data/<component>` bind mount。

通过 MCP 创建并部署 `v0.26.4-cpu-elasticsearch-bundled-isolated-data` 后，MySQL 首次初始化期间 RAGFlow 曾因连接被拒绝重启；MySQL 正式监听后，RAGFlow 自动重试并完成数据库迁移。最终 `verify_deployment` 返回 `consistent`，五个容器健康，固定 `runtime_http_probe` 返回 `reachable`。未清理任何旧 Version、实例或数据目录。

## Missed or expanded scope

- 用户明确排除 integration/E2E、Docker integration 与浏览器人工回归；Service runtime env 的主要行为测试位于 integration 文件，未执行。
- RAGFlow 的 MCP/Compose/初始化器工作是后续用户请求的扩展，不能作为原 Web Requirement 已完成运行时验收的证据。
- 全量 Proto lint 的 package-version 规则是当前仓库基线问题，尚未纳入本任务修复。

## Risks

- Service 卡片展示当前持久化 Credential 值；Credential 轮换到重新部署之间仍可能与容器进程环境不同。
- 聚合部署首次启动时，RAGFlow 可能早于 MySQL 正式监听而重试；当前 Version 依赖 `unless-stopped` 重试恢复，Orbit 尚未表达 Compose `depends_on.condition: service_healthy`。
- 工作区包含多个后续任务的 MCP/RAGFlow 变更和 Vite 临时文件，直接整体提交会混合交付边界。

## Incomplete items

- 未执行 Service runtime env integration 测试、页面级组件测试和浏览器人工验收。
- 未修复全仓 `buf lint` 基线失败。

## Conclusion

核心 Web 实现的编译、静态检查、前端测试和相关非集成测试通过；MCP 与 RAGFlow 扩展的静态和运行时检查也通过。由于 integration/E2E 与页面人工验证未执行、全仓 Proto lint 仍失败，Verification 保持 `Draft`，等待用户审阅与后续处置。
