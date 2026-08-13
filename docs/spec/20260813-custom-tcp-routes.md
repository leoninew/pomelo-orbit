# 自定义 TCP 路由规格
最后修改时间: 2026-08-13 19:25:34

Review status: Accepted

## Requirement basis

依据已接受的 [自定义 TCP 路由需求](../requirement/20260813-custom-tcp-routes.md)。Route 是 Gateway 唯一的公开 TCP 控制面：`gateway_tcp` 直接删除、`gateway_http` 直接更名为 `gateway`，历史值均由发布前离线更新处理，不提供运行时兼容。Gateway 每次成功 deploy/restart 后必须全量同步自定义 Route，避免动态 REST 配置因 Gateway 重建丢失。

## Overview

Route 保持现有 HTTP Route 能力，并新增 TCP Route。TCP Route 绑定一个受管 Service Component 的已声明 TCP Endpoint，以唯一 `listen_port` 经 Traefik 转发。域名用于客户端 DNS 连接地址，不参与明文 TCP 分流；因此每个启用 TCP Route 的监听端口全局唯一。

```text
Route (HTTP/TCP) changes
  -> persist Route
  -> collect enabled TCP Route listen ports
  -> compile Gateway unpublished Version with tcp<port> entrypoints
  -> existing Route full snapshot publish (http + tcp)

Gateway Service deploy/restart succeeds
  -> publish the complete enabled Route snapshot (http + tcp)
  -> complete Deployment
```

Gateway 静态 entrypoint 和宿主机 ports 仍由 Gateway compile 管理；Traefik REST provider 仅接收动态 router/service 快照。

## Design decisions

### 1. Endpoint modes

Version Component Endpoint 与 Service Component Endpoint overlay 的 mode 枚举收敛为：

```text
internal | local | host | gateway
```

- `gateway` 取代 `gateway_http`，必须匹配 `protocol=http`，继续生成组件派生域名的 Docker HTTP labels。
- `internal`、`local`、`host` 保持现有语义。`host` / `local` 继续由 Component Endpoint 直接生成 Compose port mapping。
- TCP Route 必须引用 `protocol=tcp` 且 mode 为 `internal` 的 Endpoint。该 Endpoint 不生成 Docker 端口映射、Traefik labels、entrypoint 或公开路由；标准 Service 因 Route target 依赖加入受管 `traefik` 网络，以稳定 container alias 供动态 TCP Route 访问。
- 删除 `gateway_tcp` 的所有验证、渲染、MCP schema、Proto、Web 选项和文档引用；不接受旧值。

发布前先执行 schema migration `000033`，再由离线运维 SQL 更新既有 `version_component_endpoint` 与 `service_component_endpoint`：

```text
gateway_http -> gateway
gateway_tcp  -> internal
tcp          -> internal
```

`000033` 的结构约束在离线转换窗口内同时容纳旧、新 mode 值，避免存量数据使 schema migration 失败；新二进制仍只接受新值。转换脚本位于 `sql/migration/offline/000033_endpoint_mode_rename.{sqlite,mysql}.sql`，不由应用迁移、业务启动或 fallback 调用；不修改已执行迁移文件。

### 1a. Endpoint identity

Component Endpoint 不再有可编辑或持久化的 `name`。端点身份固定为 `(protocol, container_port)`，该组合已在一个 Component 内唯一；所有界面和 API response 统一派生只读标识 `http<container_port>` 或 `tcp<container_port>`。Service Component Endpoint overlay 使用同一对字段关联 Version 声明，Route 的受管 target 同样保存 `endpoint_protocol` 与 `endpoint_container_port`，不保存派生名称。

新增 `000034` 迁移移除 Endpoint 名称列并建立新主键/唯一约束。既有数据库必须先执行同版本离线清理：删除全部 Service Endpoint overlay 和具有受管 target 的 Route，再应用结构迁移。运行时不转换旧名称、不提供兼容别名或 fallback。

### 2. Route model

`route` 新增的持久化字段表达 route 类型、公开监听端口和受管 endpoint target。HTTP 与 TCP 默认都引用受管 Service Component Endpoint；HTTP 的 `target_url` 仅在高级自定义下游时使用：

| 字段 | HTTP Route | TCP Route |
|---|---|---|
| `protocol` | `http` | `tcp` |
| `domain` | 必填 | 必填，作为 DNS/客户端地址 |
| `path_prefix` | 必填 | 不适用 |
| `target_url` | 高级自定义下游时必填 | 不适用 |
| `listen_port` | 不适用 | 必填，1-65535 |
| `service_id` | 默认受管目标时必填 | 必填 |
| `component_name` | 默认受管目标时必填 | 必填 |
| `endpoint_protocol` / `endpoint_container_port` | 默认受管目标时必填 | 必填 |
| `https_enabled` / certificate | 保留现有语义 | 不适用 |

受管 HTTP/TCP target 都解析到关联 Service 当前 Version 中同名 Component 的 `(protocol, container_port)` Endpoint。保存时必须验证：

- 操作者拥有 Route Project 和目标 Service 所属 Project 的权限，且两者相同；
- Service、Component、Endpoint 均存在；
- HTTP Route 的 Endpoint `protocol=http`；TCP Route 的 Endpoint `protocol=tcp` 且有效 mode 为 `internal`；
- TCP Route 的 `listen_port` 在所有启用 TCP Route 中唯一；
- `listen_port` 不与 Gateway 固定入口 `80`、`443`、`8080` 冲突；
- `listen_port` 不与正在运行的受管 Service 中有效 `local` / `host` port mapping 冲突；停止或故障 Service 的静态 Endpoint 不占用宿主机端口。

HTTP Route 选择受管目标时由系统生成动态 REST 上游 `http://<container-alias>:<container-port>`；高级自定义下游沿用 `target_url`。受管引用与自定义 URL 互斥。TCP Route 不包含 TLS、passthrough、SNI 或同端口多域名语义；这些字段不进入本次模型、接口或 UI。

Web 选择器以 Service 为入口，再级联 Component 和对应协议 Endpoint。Service 候选从项目范围的服务查询提供，组件和端点从选中 Service 的详情中得到；三层均采用项目既有 `ComboboxSelect`，不从 Route 当前分页结果推导候选项。Service 已唯一确定所属 Application，界面不再添加应用选择层。Endpoint 选项统一派生显示为 `http<port>` 或 `tcp<port>`，不显示或编辑用户名称。TCP 的公开 `listen_port` 位于 Endpoint 选择之后，Endpoint 选定时预填其 `container_port`，用户可手工覆盖。HTTP 高级自定义下游由开关控制，只有打开时显示 URL 输入框并清空受管引用。

### 3. Traefik REST snapshot

`RouteConfigPublisher.ApplySnapshot` 继续接受全部 enabled Routes，并构造单个完整 REST payload：

```json
{
  "http": { "routers": {}, "services": {} },
  "tcp": { "routers": {}, "services": {} }
}
```

- HTTP 项完全复用当前 Host / PathPrefix、`web` / `websecure`、证书和 HTTP load balancer 生成规则。受管 HTTP target 使用与 TCP 相同的容器别名和 Endpoint container port；高级 HTTP Route 使用持久化 `target_url`。
- TCP 项从受管 target 生成 `tcp.routers.<route>.rule=HostSNI(\`*\`)`、`entryPoints=["tcp<listen_port>"]` 和 `tcp.services.<route>.loadBalancer.servers=[{address:"<container-alias>:<container-port>"}]`。
- 受管 HTTP/TCP 上游使用标准 Service Component 连接 `traefik` 网络时已有的稳定 container alias；不得要求业务 Component 发布宿主机端口。
- 空的 HTTP 或 TCP router/service map 必须随快照发送，以清空相应 `@rest` namespace 的旧动态配置。

Route Dashboard 读取需要同时显示 HTTP 与 TCP routers；HTTP URL 构造只适用于 HTTP 项，TCP 项展示 `domain:listen_port`。

### 4. Gateway TCP entrypoint reconcile

Gateway compile 的 `tcpListens` 输入改为由所有 enabled TCP Route 的 `listen_port` 计算，而不再从业务 Endpoint `gateway_tcp` 收集。计算后的端口集合使用已有规范化规则：去重、排序、排除 80/443/8080。

对每个端口，编译的 Gateway Component：

- 增加 `tcp<listen_port>` 静态 Traefik entrypoint；
- 增加同端口 `0.0.0.0:<listen_port>:<listen_port>` 的 Gateway Compose port mapping；
- 在没有任何 enabled TCP Route 引用时移除该端口。

Route 的 create/update/enable/disable/delete 在写入后复用既有同步流程，先确保 Gateway 未发布 Version 已按完整 enabled TCP Route 集合 compile；其余 Route HTTP/TCP 内容均由同一次全量 REST snapshot 发布。不增加 Route 专属后台任务或状态机。

### 5. Gateway deploy/restart route sync

Gateway 详情页调用的是通用 Service deploy API，因此同步逻辑必须放在 deployment worker 的成功路径，不能放在 Vue 页面或单独的 Gateway provision helper。

当 `Application.kind=gateway` 的 deploy 或 restart 完成 Compose 操作、Service 状态写为 `running` 后：

1. 从数据库读取全部 enabled Route；
2. 通过该 Gateway 的 `rest_api_url` 调用 `RouteConfigPublisher.ApplySnapshot`；
3. 成功后将 Deployment 标记为 `ran_to_completion`。

同步失败使该 Gateway Deployment 进入 `faulted` 并记录失败原因，不写为成功，确保管理面可观测到 Gateway 已启动但 Route 未发布。该钩子覆盖 Gateway Detail、MCP `orbit_deploy` 和 Gateway provision 所有最终进入 deployment worker 的通道。

## Affected components

| 区域 | 设计变更 |
|---|---|
| SQL / sqlc | 新 Route 字段、查询参数与 target 查询；新增迁移文件，不修改已执行迁移 |
| Model / repository | Route 的 protocol、TCP target 与端口；Endpoint mode 枚举收敛 |
| Route usecase | TCP target 权限/引用/端口冲突验证、Gateway compile 触发、全量 HTTP/TCP snapshot |
| Traefik adapter | REST payload 同时生成 `http` 和 `tcp`；读取 HTTP/TCP router 状态 |
| Gateway usecase | 用 enabled TCP Routes 的端口集编译受管 Gateway |
| Deployment worker | Gateway deploy/restart 成功后的 Route snapshot 同步 |
| API / Proto / MCP | Route TCP 输入输出；移除旧 endpoint modes，保留 `internal`、`local`、`host`、`gateway` |
| Web | Route 类型化表单和展示；Component Endpoint mode 选项收敛 |
| Docs / tests | 更新活 SoT，覆盖单端口 TCP、冲突、快照和 Gateway 重建同步 |

## Interfaces

Route API 的 HTTP create/update 字段保留既有 Route 语义。TCP create/update 传递 `protocol=tcp`、`domain`、`listen_port`、`service_id`、`component_name`、`endpoint_protocol` 和 `endpoint_container_port`，并不传 `path_prefix`、`target_url`、HTTPS 或证书字段。

受管 TCP target 使用稳定 ID/name 引用，Route 展示可以返回解析后的目标容器端口和 `domain:listen_port`，但 REST payload 只使用部署期间确定的 container alias 与容器端口。

## Risks

- Route 写入后的 Gateway compile 只影响未发布 Gateway Version；实际新监听端口在用户部署 Gateway 前不会对外生效。
- Gateway 成功启动后 Route snapshot 发布失败会将 Deployment 标为失败，需确保日志直接包含 REST publish 错误。
- 端口冲突检查需要覆盖所有正在运行 Service 的有效 endpoint overlays，避免 Gateway 和业务容器竞争宿主机端口。
- 存量离线更新必须在部署新二进制前完成，否则新枚举校验会拒绝旧数据。

## Alternatives

- 保留 `gateway_tcp`：拒绝。它和 Route 形成双公开 TCP SoT，且无法可靠承担 Gateway entrypoint reconcile。
- TCP Route 支持 TLS/SNI 共享端口：不采用。该能力未在需求中授权，会引入证书、TLS termination/passthrough 和上游协议模型。
- TCP Route 允许任意地址 target：不采用。会绕过 Service 权限、版本声明和部署网络保证。
- 在 Gateway Detail 前端同步路由：拒绝。MCP 和其他 deploy 调用会漏过，且 Traefik 尚未启动时无法正确发布。

## User review notes

- Requirement 已由用户接受并要求进入 Spec。
- 本规格遵循用户已确认的最小范围，不扩展 TLS/SNI、Route 异步状态机或 HTTP target 重构。
