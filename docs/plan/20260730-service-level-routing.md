# 服务级路由与显式网关变更计划
最后修改时间: 2026-07-30 12:54:42

Review status: Accepted

流程模式: 严格 / strict

## 需求与规格依据

- [需求](../requirement/20260730-service-level-routing.md) 与 [规格](../spec/20260730-service-level-routing.md) 已接受。
- Version 只保存可复用组件规格；Service 保存选定 Version、运行配置和端口/路由暴露绑定。
- Gateway 的存在且运行校验必须保留。缺少 public TCP entrypoint 只产生必要的非阻断提示，不得触发 Gateway 写入、编译、重建或部署。
- 不新增 migration 编号、业务数据迁移代码或独立迁移脚本；修改既有 baseline SQL，并在实施时就地更新开发 SQLite。

## 实施步骤

### 1. 重置持久化归属并重新生成 SQLC

修改 `sql/migration/{sqlite,mysql}/000023_application.*`，移除 `version_expose` 的建表、索引和回滚定义；修改 `000026_service.*`，增加带 `service_id` 外键、级联删除、查询索引和 `(service_id, component_name, protocol, container_port)` 唯一约束的 `service_expose`。

修改 `sql/query/application/version.sql`，删除 Version expose 查询、写入、删除和重命名联动；修改 `sql/query/service/service.sql`，增加 Service expose 的读取、全量替换、按组件引用检查以及 Service 详情加载所需查询。新增 `model.ServiceExpose`，将仓储接口和 SQLC 实现改为由 Service 提供 expose 读写；`VersionExpose`、对应应用仓储方法及其生成代码一并删除。执行 `task sqlc`，只提交由这些查询产生的必要生成代码。

### 2. 将配置校验和编辑原子性收敛到 Service

扩展 `internal/application/service/dto` 和 `internal/application/service/usecase`：Service 创建保存 Version、instance key、runtime config 与 expose 集合。基本信息更新原子验证并保存 Version 和 instance key，同时复用既有 runtime config/exposes 验证新 Version；运行时配置更新只验证并保存 runtime config 与整个 expose 集合。读取 Service 时携带 exposes，运行时配置不再有独立于 Service 配置快照的替代来源。

校验组件名称确实属于当前 Service Version，协议和 access 为允许值，容器/监听端口在有效范围内，HTTP 路径仅用于 HTTP，且不存在重复 expose、重复 local host binding 或 public TCP listener 冲突。改变 Service 的 Version 时先对新的完整组件集合验证 expose。所有缺失值均是验证错误：后端、提交处理和部署请求均不得补默认值或使用 fallback。

版本组件重命名或删除时查询 Service expose 引用并拒绝操作，不跨聚合静默改写 Service 配置。`VersionComponentPort` 继续只允许 Gateway 的静态端口；标准应用的本地发布只能来自 `ServiceExpose`。

### 3. 替换 Proto、DTO、HTTP 映射和路由契约

在 `proto/orbit/v1/service/service.proto` 定义 `ServiceExposeReq/Resp`、`ServiceBasicUpdateReq(version_id, instance_key)`、不携带 Version 的运行时配置更新、Service preview 和 Service deploy 请求/响应。部署响应包含 `deployment_id` 和 warnings；部署输入只能包含 Service ID 和 `force_recreate`。

从 `application.proto`、`version.proto`、`application_bundle.proto` 及相应映射中删除 `ApplicationDeployReq`、Version preview 和 Version expose 契约。移除 Version 导入/导出 exposes，确保应用 bundle 只表达 Version 的静态组件规格。运行 `task proto` 后更新 Go/TypeScript 生成代码、DTO mapper 和 ProtoJSON 测试。

在 Service 路由与 handler 下新增 Service 基本信息更新、运行时配置更新、Compose preview、Service deploy 路由；删除 `/api/application/:app_id/deploy` 和 `/api/version/:version_id/preview`。停止、重启等既有 Service-target 路由保持现有授权与语义。HTTP bootstrap 的依赖与 handler 注入同步到新用例边界。

### 4. 将渲染、预览和部署执行改为 Service 输入

修改 Deployment usecase 的命令、查询、预览和 worker 执行路径：先授权并加载已有 Service，再加载其选定 Version、runtime config 和 `[]model.ServiceExpose`。部署不再接收 Version ID、instance key 或临时 expose，且不再隐式创建/覆盖 Service；Deployment 历史仍记录当时的 Version ID 与 Service ID。

把 `RenderInput.Exposes` 和 renderer 辅助函数改为 `ServiceExpose`。local 只在目标业务组件生成 `127.0.0.1:<listen>:<container>`；public HTTP 使用 Gateway 网络和派生 HTTP labels；public TCP 使用 Gateway 网络和 `tcp<listen>` labels，但业务容器不发布该 TCP 宿主机端口。标签只存在于渲染后的 Compose，绝不写入数据库。

保留标准应用队列前的 `EnsureGatewayRunning`。当 Service 有 public TCP expose 时，读取当前已部署 Gateway 的静态组件端口，逐个生成缺少 `tcp<listen>` 的 warning，并继续入队。移除标准应用部署路径中 `PrepareDeployment` 的 Gateway listener 汇总、`CompileGatewayToVersion`、`RolloutConfig` 设置和 `deployGatewayInPlace` 调用；显式 Gateway Service 部署仍是编译并应用其静态 entrypoint 的唯一路径。

### 5. 清理 Version 运行时入口，并完成 Web 服务配置工作流

删除 Version 详情的暴露表单、直接部署对话框和 Compose 预览，以及 Application 详情基于 Version 的预览。Version 页面只保留组件规格与版本生命周期操作。

在 `web/src/views/service/ServiceDetail.vue` 组织 Service 基本信息卡片与编辑模态窗（应用只读，Version 与 instance key 可修改），使其与其他详情页编辑模式一致。环境变量与接入暴露渲染为两张独立卡片，各自维护编辑草稿、保存和取消；保存单一卡片时复用服务中另一部分的已持久化值，以满足现有原子配置契约，且不在提交阶段增加默认值或 fallback。从同一已保存 Service 进行 preview、deploy、重试和详情读取。部署对话框只保留该 Service 的 `force_recreate`，不再展示 Version 选择或 instance key。部署响应带 warning 时显示必要的 Gateway 配置/部署提示，但不阻断跳转到 deployment 详情。

新增 expose 表单时只在初始化阶段设置产品默认值：组件使用当前 Version 的第一个组件，access 为 `local`；协议、容器端口及其他必填字段按表单已选择值严格验证。编辑已有 expose 时完全回填已保存值。提交、API client 和服务端均不得补默认值或 fallback。同步更新 `web/src/api`、Proto 生成类型、i18n 文案及相关页面测试。

### 6. 转换 MCP 的运行时目标

修改 `mcp/src/pomelo_orbit_mcp/orbit_client.py` 与 `tools/orbit.py`：Service 创建/配置传递 exposes；Compose preview 和 deploy 都接收 `service_id`，deploy 仅再接收 `force_recreate`。移除 `orbit_deploy(application_id, version_id, instance_key, ...)` 和 `orbit_preview_version`，不保留兼容别名。

更新 `version_specs.py`、MCP 测试、工具注册/schema 断言以及用户文档，使其不再传输 Version expose 或发起 Version 运行时操作。MCP 仅通过新的控制面 Service API 部署，不直接修改 Docker 或 Gateway。

### 7. 就地更新开发 SQLite

实施时先停止会访问 `data/db/pomelo-orbit.db` 的开发进程并做本地备份；用只读查询再次验证每条 `version_expose` 都能唯一映射到一个当前 Service。若映射不再一一对应，停止数据变更并更新规格/计划，不猜测复制目标。

在单个显式 SQLite 事务内创建与 baseline 一致的 `service_expose`，按 Version ID 关联 Service 复制三个既有 expose，检查复制数、重复键和外键后删除 `version_expose`。不改动任何 Gateway `version_component_port`，包括遗留 `6379`。完成后执行 `PRAGMA foreign_key_check`、`PRAGMA integrity_check`、表/索引盘点以及 Service expose 查询，确认原表不存在、新表有三个归属正确的记录。

## 预计涉及文件

| 区域 | 主要路径 |
| --- | --- |
| Baseline 与查询 | `sql/migration/{sqlite,mysql}/000023_application.*`、`000026_service.*`、`sql/query/application/version.sql`、`sql/query/service/service.sql` |
| 模型与仓储 | `internal/model/{application,service}.go`、`internal/repository/{application,service}.go`、`internal/repository/impl/sqlc/{application,service}/repository.go`、`internal/gen/sqlc/**` |
| 业务与渲染 | `internal/application/service/**`、`internal/application/deployment/usecase/**`、`internal/application/gateway/usecase/deployment.go`、`internal/application/application/usecase/version*.go` |
| API 与生成契约 | `proto/orbit/v1/{service,application}/*.proto`、`internal/api/http/{routes,handler}/**`、`internal/gen/proto/**`、`web/src/gen/proto/**` |
| Web | `web/src/api/{service,application}/*.ts`、`web/src/views/service/**`、`web/src/views/application/{ApplicationDetail,VersionDetail}.vue`、相关 i18n/测试 |
| MCP | `mcp/src/pomelo_orbit_mcp/{orbit_client.py,version_specs.py,tools/orbit.py}`、`mcp/tests/**`、相关文档 |
| 开发数据 | `data/db/pomelo-orbit.db`，仅在实施阶段人工就地修改，不新增仓库迁移脚本 |

## 验证计划

1. 执行 `task sqlc` 与 `task proto`，确认生成结果只反映 Service expose 和 Service-target 运行时契约。
2. 对 Service usecase 覆盖创建/完整更新、组件归属、协议/路径/端口校验、冲突拒绝、组件重命名/删除引用保护和无默认值/fallback 的错误分支。
3. 对 renderer 覆盖 local、public HTTP、public TCP，以及标准应用绝不从 `VersionComponentPort` 发布端口；断言标签是 Compose 产物而非持久化字段。
4. 对 deployment command/execution 覆盖 Gateway 不存在或未运行仍阻断、Gateway 已运行但缺 entrypoint 返回 warning 且业务仍入队、以及 public TCP 路径绝不编译、写入、强制重建或部署 Gateway。
5. 对 HTTP/Proto/前端/MCP 覆盖 Service 配置、preview、deploy、warning 呈现和已删除 Version/app 运行时入口；检查 Service 表单默认值只在初始化设置。
6. 按第 7 步完成开发 SQLite 的复制、计数、外键和完整性检查；确认 Gateway 端口记录未变化。
7. 运行 `task test` 和 `task check`，并按项目现有前端/MCP 测试入口补充相关定向检查；如环境不足，记录未运行项目及原因。

## 风险、假设与回退

- API、Web 与 MCP 均为有意的破坏性变更，不提供 Version deploy/preview 或 Version expose 兼容层。
- Service 部署在 Gateway entrypoint 缺失时可成功，但外部 TCP 连接在用户显式配置并部署 Gateway 前不可达；warning 必须明确这一状态。
- Gateway 端口删除永不从 Service 变更推断，避免影响共享入口；遗留端口由 Gateway 操作者显式清理。
- 开发数据库回退仅在实施前的本地备份和同步恢复 baseline SQL 后执行，不能用应用逻辑自动回滚；不触碰已部署 Gateway 配置或容器。
- 暂无阻塞项。计划接受后才进入 Implementation / 实现。

## User Review Notes

- 2026-07-30：用户要求将 Traefik 路由从 Version 移至 Service，并禁止 public TCP 业务部署变更或重建 Gateway。
- 2026-07-30：用户确认 Gateway 存在且运行的部署校验保留；public TCP 缺 entrypoint 时只提示，仍允许部署。
- 2026-07-30：用户要求移除 Version 发起的部署与 Compose preview；Service 是唯一运行时入口。
- 2026-07-30：用户要求 baseline SQL 与开发 SQLite 就地修改，不新增迁移版本、业务迁移或脚本；保留 Gateway 的遗留 `6379` 配置。
- 2026-07-30：用户要求开始 Implementation / 实现，计划自动接受。
- 2026-07-30：用户要求 Service 基本信息可修改，并将 Version 从运行时配置区和请求中移出。
