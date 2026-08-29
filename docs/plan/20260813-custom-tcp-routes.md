# 自定义 TCP 路由实施计划
最后修改时间: 2026-08-29 14:46:35

Review status: Accepted

## Basis

- Requirement：[`20260813-custom-tcp-routes.md`](../requirement/20260813-custom-tcp-routes.md)，状态 `Accepted`。
- Spec：[`20260813-custom-tcp-routes.md`](../spec/20260813-custom-tcp-routes.md)，状态 `Accepted`。

实现限定为：Route 的受管单端口 TCP 转发、Gateway 静态端口 reconcile、Gateway 成功部署后的 Route 全量同步、Endpoint mode 枚举收敛。不得加入 TLS/SNI、Route 状态机、任意地址 TCP target 或旧 mode 兼容。

## Implementation steps

1. 扩展 Route 结构化数据模型与数据库访问。
   - 新增 SQLite/MySQL `000033` 迁移，为 `route` 增加 `protocol`、`listen_port`、`service_id`、`component_name`、`endpoint_name` 及按协议约束所需的索引/检查；随后 `000034` 直接删除 `endpoint_name` 并改为 endpoint 协议和容器端口。
   - 更新 `sql/schema/schema.sql`、`sql/query/route/route.sql` 与 SQLC 生成代码。
   - 保持 HTTP Route 的现有字段和数据行为；TCP target 只存受管引用，不持久化已解析容器地址。

2. 收敛 Endpoint mode 并移除业务 `gateway_tcp` 渲染。
   - 更新 Version Endpoint、Service Endpoint overlay 的 SQL schema、领域校验、有效计划合并、Proto/MCP 描述和 Web mode 选项。
   - 将 `gateway_http` 直接更名为 `gateway`，将 `gateway_tcp` 和旧 `tcp` mode 直接删除并离线更新为 `internal`。
   - 调整 Compose renderer：仅 `gateway` 生成 HTTP labels；TCP `internal` Endpoint 不生成 ports、labels 或 Router。
   - 不修改已执行迁移文件；发布新二进制前，由运维离线执行两张 endpoint 表的旧值转换。

3. 实现 TCP Route 用例、权限校验和端口冲突检测。
   - 扩展 Route DTO、Proto、HTTP handler/mapper、repository port 和 usecase。
   - 对 TCP Route 解析 Service 当前 Version 的 Component/Endpoint，校验项目归属、TCP 协议和 `internal` mode。
   - 在 create/update/enable 前检查已启用 TCP Route 的唯一 `listen_port`、80/443/8080 保留端口，以及正在运行受管 Service 的有效 `local`/`host` 映射端口；停止或故障 Service 不占用宿主机端口。
   - Route 的 create/update/disable/delete 和证书操作只写入业务数据；详情页和列表页启停都只保留前端草稿，统一交由同步入口预览并确认提交。

3a. 将 HTTP Route 收敛为默认受管 Endpoint，并保留高级自定义下游。
   - 使用 `service_id`、`component_name`、`endpoint_protocol`、`endpoint_container_port` 保存受管 HTTP/TCP target；HTTP 受管 target 校验有效 HTTP Endpoint，并在 Route snapshot 生成时解析 `container-alias:container-port`。
   - HTTP Route 仅当三个受管引用均为空时接受 `target_url`，作为高级自定义下游；两种 target 互斥。
   - 前端用项目范围 Service 查询和 Service 详情构建 Service/Component/Endpoint 的级联 `ComboboxSelect`；HTTP 默认展示受管目标，高级开关才显示 URL 输入框。TCP 始终为受管选择。

3b. 移除 Endpoint 名称并用协议加容器端口作为身份。
   - 新增 `000034` SQLite/MySQL 迁移，将 Version Endpoint、Service Endpoint overlay 和 Route 受管 target 结构切换为 `(protocol, container_port)`；既有数据库先离线清空旧 overlay 与受管 Route，不转换旧名称；运行时代码不接受或保留旧名称。
   - 更新 model、SQL/SQLC、Proto/API、有效计划合并与 Route target 校验，使同一 Component 内的协议和容器端口作为唯一关联键。
   - 移除组件 Endpoint 表单的名称输入，所有 UI 以 `http<port>` / `tcp<port>` 派生展示和作为稳定列表 key。

4. 扩展 Traefik REST 发布与 Gateway compile。
   - 将 RouteManager full snapshot 同时渲染 HTTP 和 TCP router/service map；TCP 生成 `HostSNI(*)`、`tcp<listen_port>` entrypoint 和 `container-alias:container-port` server address。同步预览读取全部 REST routers/services，按 router 及其 service 聚合为一条逻辑 Route，以新增、修改或删除及“协议 规则 -> 上游”生成可读差异；仅使用 TLS 布尔值判定 HTTP/HTTPS，不比较证书或 TLS 内部配置。确认和 Gateway deploy/restart 都覆盖完整 REST snapshot。
   - Route dashboard 的 Traefik client 同时读取 HTTP/TCP routers，DTO/UI 依协议展示链接或 `domain:listen_port`。
   - Gateway compile 的 TCP 监听端口来源切换为 enabled TCP Routes。利用已有 `buildManagedGatewayComponent`、`buildTraefikStaticConfig` 和 Compose endpoint 渲染生成 entrypoint 与端口映射。

5. 将 Route 同步接入 Gateway deployment worker。
   - 增加面向 deployment execution 的最小 Route snapshot publisher port，避免 deployment usecase 依赖 Route usecase。
   - 在 `ExecuteApplicationDeploy` 和 `ExecuteApplicationRestart` 的 Gateway 成功 Compose 路径，在标记 Deployment 成功前全量发布 enabled HTTP/TCP Routes。
   - 扩展 worker bootstrap 依赖注入，使 Gateway Detail、MCP 与 provision 最终经过同一个成功后同步钩子。
   - 发布失败应使 Deployment 进入 `faulted`，并记录 REST publish 错误。

6. 完成前端、文档和生成物。
   - 更新 RoutePage/RouteDetail：类型选择、HTTP/TCP 受管 Service/Component/Endpoint Combobox 选择、HTTP 高级自定义下游开关和 TCP 地址展示；两页启停均为同步草稿，复用预览确认弹窗。预览差异表格使用“操作 / 规则”两列，在规则中仅显示有值的业务数据和 Traefik 数据；HTTPS 证书操作仅对 HTTP Route 可用。
   - 更新 Version/Service Component detail 页的 endpoint mode 文案与模式驱动字段可见性。
   - 更新活文档 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/routing-and-certificates.md`，移除 `gateway_tcp`，记录 Route TCP/Gateway sync 行为。
   - 使用 `task sqlc` / `task proto` 或等效项目命令更新 SQLC、Go Proto 和 Web TypeScript 生成物。

## Files to change

| 区域 | 预期文件 |
|---|---|
| 数据库 | `sql/migration/{sqlite,mysql}/000033_*`、`sql/schema/schema.sql`、`sql/query/route/route.sql`、`internal/gen/sqlc/route/*` |
| Endpoint | `internal/model/*`、`internal/application/application/usecase/version_validation.go`、`internal/application/service/usecase/service.go`、`internal/application/deployment/usecase/compose_renderer.go`、相关 SQL/Proto/MCP/Web 文件 |
| Route | `internal/model/route.go`、`internal/repository/route.go`、`internal/application/route/{dto,port,usecase}`、`internal/api/http/handler/route/*`、`proto/orbit/v1/route/*` |
| Traefik/Gateway | `internal/infrastructure/external/traefik/route.go`、`internal/application/gateway/usecase/{compile,exposure,service}.go` |
| 部署 | `internal/application/deployment/usecase/{service,command,deployment_execution}.go`、`internal/bootstrap/{http,worker}.go`、相关 port 与测试 |
| 前端 | `web/src/views/route/RoutePage.vue`、`web/src/api/route/route.ts`、Component detail/form 与生成 proto |
| 文档 | 上述活 SoT 文档及本任务 verification 文档 |

## Verification plan

1. 数据库与生成物：运行 SQLC 和 Proto 生成，确认 schema/query/生成代码一致。
2. Go 单元/集成测试：
   - Route HTTP 与 TCP 输入校验、项目权限、endpoint 引用、端口冲突、停用/删除后端口集合；
   - Traefik HTTP/TCP full REST snapshot、空 namespace 清理；
   - Gateway compile 的 entrypoint/port mapping；
   - Gateway deploy/restart 成功后的 Route publish，以及 publish failure 转 deployment fault；
   - 四个现行 mode 的协议关联与 TCP Route 对 `internal` TCP Endpoint 的 target 校验。
3. 前端测试和检查：Route 条件表单、TCP 地址展示、Endpoint mode 选择；运行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。
4. 后端检查：运行 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...` 和 `go test ./cmd/... ./internal/...`。
5. 交付前人工核验：HTTP Route 在 Gateway 重部署后恢复；TCP Route 对应端口已由 Gateway 监听，业务 Component 未直出该端口。

## Deployment prerequisites

在 `000035` 收紧 Endpoint mode 约束前，由运维执行同版本交付的离线脚本：

| 数据库 | 脚本 |
|---|---|
| SQLite | `sql/migration/offline/000033_endpoint_mode_rename.sqlite.sql` |
| MySQL | `sql/migration/offline/000033_endpoint_mode_rename.mysql.sql` |

离线脚本将 `gateway_http` 更新为 `gateway`，将 `gateway_tcp` 与旧 `tcp` 更新为 `internal`。离线脚本不放入已执行迁移、应用启动逻辑或业务 fallback。执行前必须备份，并在转换后确认两张 Endpoint 表只含四个现行 mode。

## Risks

- Route create/update 后静态 Gateway entrypoint 只有在 Gateway 后续 deploy/restart 后才生效。
- Gateway route publish 失败会使部署显示失败，即使容器已运行；部署日志必须清晰表明该条件。
- 新 Route 字段会影响 SQLC 与 Proto 的多端生成物，需避免手工编辑生成文件造成漂移。
- 离线旧 mode 转换遗漏会在新校验生效后阻断受影响 Service 的部署。

## Rollback

- 代码回滚前先确认数据库 `000033` 的 down migration 与已发布 schema 可逆；不回滚已执行的历史迁移。
- 若 Gateway 发布失败，保留现有 Route 数据和 Gateway volumes；修复配置后重新部署 Gateway 触发全量 Route 同步。
- 离线 mode 数据转换若需回退，按发布记录执行反向 SQL，不在应用中恢复旧 mode 支持。

## User review notes

- 用户已要求进入 Plan。
- 用户已要求开始实现，Plan 已接受。
