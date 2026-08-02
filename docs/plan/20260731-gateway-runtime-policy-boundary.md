# Gateway Runtime Policy Boundary Plan
最后修改时间: 2026-08-02 13:06:01

Review status: Accepted

流程模式: 严格 / strict

## 需求与规格依据

- [Requirement](../requirement/20260731-gateway-runtime-policy-boundary.md) 与 [Spec](../spec/20260731-gateway-runtime-policy-boundary.md) 已接受。
- VersionComponent 与 `version_component_xx` 保存可复用声明和模板值；ServiceComponent 与 `service_component_xx` 保存稀疏运行时值。没有 Service 子记录表示继承，`override` 表示改写值字段，`deleted` 表示压制声明值。
- Service 只能覆盖或删除 Version 已声明的条目，不能新增环境变量、挂载、资源项或 Endpoint。可覆盖的是值字段；标识、接口和结构字段仍来自 Version。
- 网络配置允许保存。端口冲突、Gateway/entrypoint/域名/网络可用性由 Preview 和 Deploy 的部署处理报告；Gateway `api` 与其他已声明 Endpoint 一样按保存的 Version/Service 值生成计划，不因名称被强制为 `internal`。
- 开发数据库通过原始 DDL 重建，不保留旧 schema、旧 API、旧 renderer、双读、历史数据转换或兼容适配。

## 实施步骤

### 1. 更新原始 DDL 并重建开发数据库

在 `sql/migration/{sqlite,mysql}/000023_application.*.sql`、`000025_gateway.*.sql`、`000026_service.*.sql` 和 `000027_deployment.*.sql` 中直接定义 `version_component_endpoint`、`service_component`、`service_component_env`、`service_component_mount`、`service_component_resource`、`service_component_endpoint` 及 EffectiveServicePlan fingerprint 的最终 schema。同步删除 `version_component_port`、`service_expose`、`service.runtime_config_json`、`gateway_config.image` 和 `application.image_pull_policy` 的原始定义。

开发数据库重新建库后从空库执行现有 migration 序列，schema 版本保持 `30`。不新增 `000031`，不转换历史端口、Expose 或 JSON 数据，也不提供旧 schema/API/renderer 的双读或兼容路径；原始 DDL 的 up/down 定义必须保持一致。

### 2. 重建模型、查询与仓储边界

更新 `sql/query/application/version.sql`、`sql/query/service/service.sql` 和 `sql/query/deployment/deployment.sql`，为新的声明、Service overlay、Version 切换重映射、Effective plan hash 与最近成功部署查询提供 SQLC 查询。移除 ServiceExpose 与 JSON runtime config 查询、写入和冲突查询。

在 `internal/model/{application,service,deployment}.go` 定义 Version Endpoint、ServiceComponent、各 Service 子配置及 `EffectiveServicePlan` 读模型。Service 子记录只含与 Version 对应的值字段和 `state`；仓储在事务中执行 Version 切换时的 Component/name 重映射，缺少声明条目时拒绝切换。

修改 `internal/repository/{application,service,deployment}.go`、`internal/repository/impl/sqlc/**` 和相关 port/store 适配器，使 Service 是 overlay 的唯一所有者。执行 `task sqlc`，仅提交由变更查询生成的 SQLC 文件。

### 3. 用 EffectiveServicePlan 统一预览、渲染与部署

在 `internal/application/deployment/usecase/` 新增 EffectiveServicePlan 构造器：加载 Service、Version、ServiceComponent 与子表，按 `inherit` / `override` / `deleted` 合成 Component、资源、挂载和 Endpoint，并对规范化结果计算稳定 fingerprint。Version 端点的协议/容器端口保持不可改写；Service Endpoint 的值字段只引用已声明 Endpoint。

将 `compose_renderer.go`、`runtime_query.go`、`deployment_execution.go`、`command.go` 和相关 DTO/port 改为只消费 EffectiveServicePlan，删除 `RenderInput.Exposes`、`RuntimeConfig`、`injectGatewayNetwork`、`publishComponentPorts` 与 VersionComponentPort 的渲染语义。所有 Compose `ports`、Traefik labels、网络和受控文件只能由 plan 投影产生。

保留 Gateway 平台策略构造器，但将其输入改为 Gateway Service 的 Effective Endpoint。共享网络的 Compose key/物理名/标签固定为 `traefik`；Gateway Endpoint 不按名称产生额外端口策略。将原先配置保存时的端口、public TCP、entrypoint 和 Gateway 冲突判断移至 Preview/Deploy preflight：保存成功，Preview/Deploy 返回可操作的问题；无法生成或执行 Compose 时部署失败并记录原因。

在部署成功记录中保存 fingerprint 和渲染摘要。Service 配置读取时比较当前 fingerprint 与最近成功部署 fingerprint，返回 `pending_deploy`；Preview 与 Deploy 必须使用同一构造器，避免状态、预览和实际部署不一致。

### 4. 替换应用、Service 与 Gateway 用例契约

更新 `internal/application/{application,service,gateway,deployment}/dto`、usecase、port 和 HTTP handler mapper。Version Component 的 ports 更新契约迁移为 Endpoint 声明；Service 创建/配置改为读取和写入 ServiceComponent overlays，不再接收 `runtime_config` 或 `ServiceExpose` 集合。

Service 配置更新只校验 Version 声明引用、字段类型和 Endpoint 自身字段组合。环境、资源、挂载和 Endpoint 的新增未知条目必须拒绝；同名声明值等于 Version 时删除 overlay；删除有效配置写入 tombstone。网络冲突不在保存路径拒绝。

Gateway 的 image 与 pull policy 只从 Gateway VersionComponent 读取。GatewayConfig 更新重新构造并部署 Gateway Service plan，不覆写 Component 声明。移除所有旧 `gateway_config.image`、Application pull policy、ServiceExpose 和 VersionComponentPort 的领域入口。

### 5. 更新 Proto、HTTP、MCP 与生成代码

在 `proto/orbit/v1/application/version.proto` 以 Endpoint 声明取代 `ComponentPort` / `VersionComponentPortsUpdateReq`；在 `proto/orbit/v1/service/service.proto` 定义 ServiceComponent 配置读取、值覆盖、删除和待部署状态的请求/响应。Service 读取响应必须携带声明、overlay、Effective 值、来源状态、fingerprint 与 `pending_deploy`，但不得暴露旧 JSON 或 ServiceExpose。

同步修改 `internal/api/http/{routes,handler}/application`、`service`、`gateway` 和 bootstrap 注入，更新 `web/src/gen/proto/**`、`internal/gen/proto/**` 及 `web/src/api/{application,service}`。执行 `task proto`。更新控制面 MCP 的 Service 配置、Preview 和 Deploy 调用及其 schema 测试，使其只使用新 Service 契约，不保留旧参数或别名。

### 6. 重组 Service 配置 UI

将 `web/src/views/service/ServiceDetail.vue` 的扁平 runtime config 与 expose 编辑卡片替换为组件配置入口。它列出当前 Service 的 Component、覆盖/删除数量和服务端 `pending_deploy`，并连接新的 `ServiceComponentDetail` 路由与页面。

新增 `web/src/views/service/ServiceComponentDetail.vue`，沿用 `VersionComponentDetail.vue` 的 Runtime、Connectivity、Mounts、Advanced 分块和行编辑模式。每个可修改行显示 Version 模板值、Service 值、Effective 值和 `inherited` / `override` / `deleted` 状态；基础执行、健康检查、依赖、设备、tmpfs、ulimit、mount target/type/权限、Endpoint protocol/container port 只读。编辑仅定位 Version 已声明条目的值字段，删除写 tombstone；不显示任意新增入口。

保存后只刷新服务端配置视图并显示“待部署”，不自动重启。Preview 与 Deploy 保持现有入口但使用新 fingerprint；更新 router、i18n、API client 和视图测试。删除旧 runtime config/expose 表单、API 类型和文案。

### 7. 清理旧引用并建立回归测试

删除旧 ServiceExpose、runtime config JSON、VersionComponentPort 与 Gateway image/pull policy 的 API、仓储、用例、renderer、MCP 和前端引用。通过 `rg` 确认它们不再作为生产代码或协议字段存在；保留历史文档不作操作依据。

补充数据库、仓储、用例、renderer、Gateway、HTTP/Proto 和 Vue 测试。覆盖迁移成功/失败、ServiceComponent 重映射、值覆盖、tombstone、未知声明拒绝、Gateway `api` 保留声明或覆盖的 Endpoint 配置、网络配置保存不阻断、Preview/Deploy 处理网络失败、fingerprint 待部署状态、UI 三值展示和显式部署。

## 预计涉及文件

| 区域 | 主要路径 |
| --- | --- |
| Schema 与数据迁移 | `sql/migration/{sqlite,mysql}/{000023_application,000025_gateway,000026_service,000027_deployment}.*.sql`、`sql/query/{application/version,service/service,deployment/deployment}.sql` |
| 模型与仓储 | `internal/model/{application,service,deployment}.go`、`internal/repository/**`、`internal/repository/impl/sqlc/**`、`internal/gen/sqlc/**` |
| 业务与渲染 | `internal/application/{application,service,gateway,deployment}/**`、`internal/common/runtimeconfig/**` |
| API 与生成契约 | `proto/orbit/v1/{application/version,service/service,gateway/gateway}.proto`、`internal/api/http/**`、`internal/gen/proto/**`、`web/src/gen/proto/**` |
| Web | `web/src/views/service/{ServiceDetail,ServiceComponentDetail}.vue`、`web/src/views/application/VersionComponentDetail.vue`、`web/src/{router,api,i18n}/**` |
| MCP 与测试 | 控制面 MCP Service client/schema、各层 `*_test.go`、Web 测试 |

## 验证计划

1. 在 SQLite 与 MySQL 空库执行全部 migration，验证最终 schema、外键、唯一键、`PRAGMA foreign_key_check` / `integrity_check` 与 MySQL 等价检查；不测试历史数据升级或 `000031` 兼容迁移。
2. 运行 `task sqlc`、`task proto`，确认生成代码与源查询/Proto 一致，旧 `runtime_config`、`ServiceExpose`、`ComponentPort` 契约不存在。
3. 覆盖 EffectiveServicePlan 对继承、值覆盖和 tombstone 的合成；确认未知 env/mount/resource/Endpoint 被拒绝，Service 不会新增未声明条目，Gateway `api` 保留已声明或覆盖的 mode、监听地址和端口。
4. 覆盖 Compose renderer 对 Gateway 网络 `traefik` 标签、按 Effective Endpoint 生成的端口/路由，以及 Preview/Deploy 才处理端口/entrypoint/Gateway 冲突的行为。
5. 覆盖 Version 切换时 ServiceComponent/name 重映射和错误拒绝；覆盖成功部署写 fingerprint、配置更新后 `pending_deploy=true`、再次成功部署后清除待部署。
6. 覆盖 HTTP/Proto/MCP 的新 Service 配置读写，以及 Vue 的组件配置入口、四分块详情、三值状态、编辑/删除和待部署提示。
7. 运行 `task test`、`task check` 与前端项目约定的 lint/test 命令；记录无法运行的检查和原因。

## 风险、假设与回退

- 这是跨 SQLite/MySQL、SQLC、Proto、Go、MCP 与 Web 的破坏性切换；所有消费者必须与 schema 一同升级，不能滚动混用。
- `runtime_config_json` 的 placeholder 展开可能一对多写入多个 Component 环境变量。迁移前必须对每个 key 的 Version 引用做只读盘点；不猜测未引用或缺失值的目标。
- Service 允许保存会在部署阶段失败的网络意图。Preview 必须尽早清晰呈现问题，Deploy 必须保留失败日志和可重试状态。
- 开发数据库按当前原始 DDL 重建；不提供历史 schema 或数据的应用内回退。
- Plan 接受后才进入 Implementation / 实现。

## User Review Notes

- 用户确认所有 Service 覆盖都锚定 Version 已声明的条目，只修改值字段；不提供新增挂载等例外。
- 用户要求网络策略不阻止配置保存，延后到 Preview/Deploy 处理；Gateway `api` 不再拥有名称特判，沿用通用 Endpoint 语义。
- 用户确认首期不需要独立覆盖变更审计，Deployment 的 EffectiveServicePlan snapshot/fingerprint 足够。
- 用户接受 Plan，并要求开始 Implementation / 实现。
