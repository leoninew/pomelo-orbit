# 服务级路由与显式网关变更验证
最后修改时间: 2026-07-30 18:41:20

Review status: Draft

流程模式: 严格 / strict

## 对齐结论

- [需求](../requirement/20260730-service-level-routing.md)、[规格](../spec/20260730-service-level-routing.md) 与[计划](../plan/20260730-service-level-routing.md)均为 `Accepted`。
- 实现将访问暴露的持久化归属从 Version 移至 Service：移除 `VersionExpose`/`version_expose`，新增 `ServiceExpose`/`service_expose`，并由 Service 配置驱动 Compose 端口与 Traefik labels。
- 标准 Service 部署继续校验 Gateway 存在且运行。缺少 public TCP entrypoint 时返回 warning 并继续入队；业务部署路径不再编译、写入、重建或部署 Gateway。
- Version 的运行时部署与 Compose preview 契约、HTTP 路由及 Web/MCP 调用已移除，Service 成为 preview/deploy 的唯一入口。
- Service 基本信息、环境变量与接入暴露已分离为独立编辑面；基本信息可修改 Version 与 instance key，运行时配置请求不再携带 Version。

## 实际改动与范围

| 区域 | 结果 |
| --- | --- |
| Baseline SQL、模型与仓储 | 既有 `000023_application`/`000026_service` baseline 与 SQLC 查询改为 Service expose 归属；生成 SQLC 代码同步更新。 |
| Service、Version 与 HTTP/Proto | 新增 Service basic/config/preview/deploy 契约；移除 Version expose 与 Version/application 运行时入口；Go 与 TypeScript 生成代码已更新。 |
| 部署与 Gateway | renderer 从持久化 Service expose 派生 local/public HTTP/public TCP Compose；Gateway 缺 entrypoint 为非阻断 warning，Gateway 配置只读。 |
| Web | Service 详情承担基本信息、环境变量、暴露、预览和部署；Version 详情只保留组件规格。 |
| MCP | preview/deploy 改以 `service_id` 为目标，删除 Version 运行时工具与请求字段。 |
| 测试 | 覆盖 Service 配置、Gateway 运行前置条件、public TCP warning、渲染和 Gateway 行为边界。 |

与计划预计文件一致的主要区域包括 `sql/`、`internal/{model,repository,application,api,gen}/`、`proto/`、`web/src/{api,gen,views}/` 和 `mcp/`。

当前工作树还包含本功能之外的 BGE-M3/RAGFlow 部署 SQL、技能、代理脚本，以及全局详情页排版/交互调整。这些不应被解释为服务级路由需求本身；提交前应按可审计边界拆分。当前也存在 42 个已暂存文件的工作区格式/文案差异和未跟踪部署资产，本验证未改写、暂存或取消暂存它们。

## 验收清单

- [x] Version 不再承载路由、Traefik labels 或宿主机端口绑定；Service 管理持久化 expose。
- [x] Service expose 覆盖协议、访问方式、容器端口、监听端口与 HTTP 路由参数，并对组件归属、端口、路径与冲突进行校验。
- [x] local 仅发布目标 Service 端口；public HTTP/TCP 仅为目标 Compose 派生 Gateway network 与 labels。
- [x] Gateway 不存在或未运行仍阻断标准 Service 部署。
- [x] Gateway 缺少 public TCP entrypoint 时，仅给出所需端口和显式配置/部署 Gateway 的 warning，业务部署仍入队。
- [x] Service 部署与 public TCP 执行路径不调用 Gateway compile/reconcile/force recreate/deploy。
- [x] Version/application deploy/preview 契约和 MCP 入口已移除，Service 为唯一运行时入口。
- [x] Service 基本信息通过模态窗修改；环境变量与接入暴露为独立卡片，提交阶段不补默认值或 fallback。
- [x] 既有 baseline SQL 与开发 SQLite 已就地改为 `service_expose`，未新增 migration 编号、业务迁移或数据迁移脚本。
- [!] 无法重新证明计划中的历史“三条 expose 均复制”与 Gateway 遗留 `6379` 保留：随后环境重建后，当前数据库只有两条 local HTTP Service expose，且没有 `6379` 的 `version_component_port` 记录。

## 验证结果

| 命令或检查 | 结果 |
| --- | --- |
| `task test` | 通过：Web 8 个测试文件、48 个测试通过；`go test ./cmd/... ./internal/... ./sql` 通过。 |
| `task check` | 通过：`vue-tsc`、ESLint 自动修复、Prettier、Go format、`golangci-lint` 均无问题。 |
| `uv run pytest`（`mcp/`） | 通过：56 passed，1 deselected。 |
| `task sqlc` | 通过：SQLC 生成成功。 |
| `task proto` | 通过：Buf 生成成功；仅提示未配置外部依赖可更新。 |
| `git diff --check`、`git diff --cached --check` | 通过，无空白错误。 |
| `go test ./internal/application/deployment/usecase ./internal/application/service/usecase ./internal/application/gateway/usecase` | 通过。定向覆盖 `TestDeployServiceRejectsGatewayNotRunningBeforePersisting`、`TestDeployServiceWarnsForMissingPublicTCPGatewayEntrypoint` 及 Service/Gateway 用例包。 |
| SQLite `PRAGMA integrity_check` | `ok`。 |
| SQLite `PRAGMA foreign_key_check` | 无结果行。 |
| SQLite 架构与数据盘点 | `version_expose` 不存在；`service_expose` 具备 Service 外键级联删除、查询索引和唯一约束。当前共有 2 条 local HTTP expose：RAGFlow `ragflow-cpu:80`/`9380` 与 BGE-M3 `tei:80`/`8081`。 |

## 风险与未完成项

- 本次 API、Web 和 MCP 合约是有意的破坏性变更；旧 Version deploy/preview 不保留兼容入口。
- public TCP 的业务 Service 在 Gateway entrypoint 缺失时可以部署，但外部连接会在用户显式配置并部署 Gateway 前不可用；这依赖可见 warning。
- 数据库在后续环境操作中被重建或覆盖，无法从当前状态复核实施时的三条 `version_expose` 迁移和 Gateway `6379` 保留。当前结构、外键和完整性均正常。
- 当前索引与工作区混有服务级路由外的部署资产和 UI 改动；在不进行片段级索引操作的约束下，应避免将它们误标为单一功能提交。

## 结论

服务级路由与显式 Gateway 控制的实现和可重复验证项均通过。历史开发数据迁移计数与 Gateway `6379` 只能记录为不可回溯的证据缺口，不影响当前数据库完整性或代码路径验证。等待审阅本验证记录后，可将其状态更新为 `Accepted`。
