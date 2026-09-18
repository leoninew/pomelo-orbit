# Gateway REST API 访问上下文实施计划
最后修改时间: 2026-09-18 14:54:44

Review status: Accepted

Mode: standard

## Intent basis

本计划落实已接受的 [Gateway REST API 访问上下文 Intent](../intent/20260918-gateway-rest-api-access-context.md)。Traefik 管理 API 有两个明确且固定的访问端点：容器网络中的 `http://traefik:8080`，以及部署宿主机 loopback 上的 `http://127.0.0.1:8080`。SSH 只是让 Orbit 在目标宿主机执行命令，不改变 host endpoint 的含义。

当前初始化已经把 `8080` 作为 `local` 端口发布到 `127.0.0.1:8080`，但 Gateway 只保存一个 container endpoint，RouteManager 又始终在 Environment Runtime 根命名空间执行 curl，导致 SSH/宿主机模式无法解析 `traefik`。此外，Gateway 生命周期维护一条关闭状态的 `traefik-dashboard` 业务 Route；该 Route 不是 Traefik 本地管理 API 所必需的产品数据。

## Target design

- `GatewayConfig.rest_api_url` 保留为容器端点，`GatewayConfig.rest_api_host_url` 新增为宿主机端点。
- 新建 Gateway 和 Project Initialization 固定生成 `http://traefik:8080` 与 `http://127.0.0.1:8080`。版本组件继续用 `local` 模式将容器 `8080` 绑定到宿主机 `127.0.0.1:8080`；Service 不创建重复 overlay，继承 Version 定义。
- RouteManager 的 GET、ready check 和 PUT snapshot 都经由同一请求帮助方法。local Environment 且 Orbit 运行在容器内时仅请求 container endpoint；local Environment 的宿主机 Orbit 与 SSH Environment 均仅请求 host endpoint。请求失败不切换地址、不重试同一业务请求。
- 删除初始化时固定生成的 `traefik-dashboard.<base_domain>` 自定义 Route，以及 Gateway 对这条 Route 的恢复、导出和删除所有权；不扩大为删除现有 dashboard API/UI。保留 Traefik 的 `api.dashboard: true` 与 `api.insecure: true`，它们服务于 loopback 管理 API，而不是业务 Route。
- 接管包升级为 v2，导出两个 endpoint 且不包含 dashboard Route。已导出的 v1 包不在业务代码中迁移或兼容；数据库历史记录只通过 SQL migration 补齐固定 host endpoint。普通 Route 上游不受推断或改写。

## Implementation steps

1. Gateway 数据模型、DTO、SQLC query/repository 和三方言 `000047` migration 增加 `rest_api_host_url`。迁移为已有 Gateway 写入固定 loopback endpoint，空库 schema 同步更新。
2. 初始化和 Gateway 生命周期保存、返回两个 endpoint；移除 `DashboardRoute` 及所有 Gateway 对 Route 的所有权。更新 Project Initialization、Gateway HTTP/proto/MCP 和 Web 表单传递及展示该字段。
3. 重构 Traefik RouteManager，使所有 REST 请求复用按执行上下文选择的单一 endpoint 逻辑：容器化的 local Orbit 使用 container endpoint，宿主机 local Orbit 与 SSH Environment 使用 host endpoint。保留实际请求失败的原始语义，不做 fallback 或循环重试。
4. 新增 migration 仅删除精确匹配的系统生成历史 `traefik-dashboard.<base_domain>` Route，不触碰用户定义的 Route 或既有 dashboard API/UI。
5. Handover v2 导入导出携带完整 Gateway 定义；不为旧导出包保留业务层迁移或兼容路径。
6. 覆盖初始化端口模型、容器化 local Orbit、宿主机 local Orbit、SSH Environment 的单次 endpoint 选择、snapshot 发布、接管包 round-trip 和 dashboard Route 不再生成的主流程测试。

## Files to change

- `internal/model/gateway.go`、Gateway use case/DTO、初始化 use case
- `internal/infrastructure/external/traefik/route.go`
- Gateway SQL query、repository、schema 和 `000047` migrations
- Gateway、Project Initialization、Route proto/HTTP/MCP 及 Web 调用方
- `internal/application/project_handover/**`
- Gateway/Route 相关测试和 CD 活文档

## Assumptions and risks

- 目标 Traefik 保持将 8080 发布为宿主机 loopback；该端口不对公网暴露。
- endpoint 选择依赖 Orbit 的实际执行位置；该位置检测必须可靠，且请求失败不能改变已选 endpoint。
- 已导出 v1 接管包不属于兼容范围；数据库中的历史系统 Route 只由 SQL migration 按精确结构删除。
- 不修改普通 HTTP/TCP Route 的目标限制和可达性语义。

## Rollback

`000047` down migration 仅移除新增 host endpoint 字段。回退前导出的 v2 接管包应在支持 v2 的版本恢复；不引入长期的新旧运行时并存逻辑。

## User review notes

- 用户确认 `http://traefik:8080` 为 container endpoint，`http://127.0.0.1:8080` 为 host endpoint，并要求按此方向实现和复核初始化逻辑。
- 用户要求删除 `traefik-dashboard` 自定义路由，并确认 8080 在 Version 与 Service 的有效定义中为 local 模式。
- 用户要求测试覆盖主流程即可，且不接受通过 URL 主机名猜测、改写或限制用户业务 Route。
- 用户明确确认：local Environment 中 Orbit 容器使用 `http://traefik:8080`，Orbit 宿主机使用 `http://127.0.0.1:8080`；SSH Environment 一律在远程宿主机调用 `http://127.0.0.1:8080`，不做 fallback 或循环重试。
