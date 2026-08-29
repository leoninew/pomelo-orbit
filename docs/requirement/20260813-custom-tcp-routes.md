# 自定义 TCP 路由
最后修改时间: 2026-08-29 14:31:44

Review status: Accepted

## Background

当前平台自定义 Route 仅发布 Traefik `http.routers` 和 `http.services`，可将自定义域名和路径代理到 HTTP(S) 上游。组件 Endpoint 另有 `gateway_tcp` mode，意图为将 Redis 等内部 TCP 服务通过 Gateway 以域名加端口的形式公开。

这两套入口机制的边界不对称：`gateway_tcp` 的公开监听端口需要 Gateway 静态 entrypoint 和容器端口映射配合，但业务 Endpoint 中还要求用户手工填写 entrypoint；部署链路没有把标准服务的 TCP 端口集合统一 reconcile 到 Gateway。普通 TCP 协议也不携带 Host，不能按域名在同一个端口分流。结果是服务详情中的 `host` 端口暴露与 `gateway_tcp` 入口容易冲突，用户难以判断何时该使用 Traefik。

## Goal

将平台自定义路由扩展为 HTTP Route 和 TCP Route 两类，由 Route 成为唯一的 Gateway 公开入口控制面：

- HTTP Route 维持域名、路径和 HTTP(S) target 的现有能力。
- HTTP Route 默认选择项目内受管 Service Component 的 HTTP Endpoint；直接填写任意 HTTP(S) 下游仅作为明确开启的高级功能。
- TCP Route 能将受管 Service Component 声明的 TCP Endpoint 通过 `domain:listen_port` 对外公开，并由平台管理 Traefik 动态 TCP 路由、Gateway 静态 entrypoint 和 Gateway 宿主机端口映射。
- TCP Route 按公开监听端口独占路由，将域名作为客户端连接地址和 DNS 身份。
- 业务 Component Endpoint 不再承担 `gateway_tcp` 公开入口配置；公开 TCP 的生命周期、权限和冲突校验收敛到 Route。
- Endpoint mode 的 `gateway_http` 重命名为 `gateway`，保留其 HTTP Gateway 语义。

## Non-goal

- 不改变 HTTP Route 的 Host / PathPrefix 路由语义、证书管理和 `providers.rest` 全量发布方式。
- 不实现任意非受管的 `host:port` 作为 TCP Route 默认 target；目标必须可由平台权限、部署状态和网络可达性管理。
- 不尝试让明文 Redis、MySQL 等 TCP 协议依据域名在同一个监听端口中分流。
- 不在本需求中实现 UDP、HTTP/3、PROXY protocol、TCP middleware 或多 Gateway 并行运行。
- 不保留 `gateway_tcp` 与 TCP Route 的长期双重业务 SoT、兼容别名或默认值回退。

## User scenarios

1. 用户为 Redis Service 的 `redis` Component 声明 TCP Endpoint `6379`，创建 TCP Route：`redis.example.com:16379 -> redis:6379`。平台部署/更新 Gateway 后，客户端可以用该地址连接；业务 Redis 容器不直接映射宿主机 `16379`。
2. 用户尝试为两个 TCP Route 使用同一监听端口。平台在保存或部署前拒绝，并指出端口已被另一条 TCP Route 独占。
3. 用户创建 HTTP Route：`termbridge.example.com/ -> termbridge component:80`，现有 HTTP/HTTPS Route 行为不受 TCP Route 引入影响。
4. 用户停用或删除最后一条使用某 TCP 监听端口的 Route。平台在后续 Gateway reconcile 中移除对应的静态 entrypoint 和宿主机端口映射，不影响其他 HTTP/TCP Route。
5. 用户在 Gateway 详情页发起部署。Gateway 容器成功启动后，平台重新发布全部已启用 HTTP/TCP Route 到 `providers.rest`，既有自定义路由不会因 Gateway 重建而丢失。
6. 用户添加 HTTP 或 TCP Route 时，先用可搜索的 Combobox 选择受管 Service、Component 和匹配协议的 Endpoint；HTTP Route 仅在主动展开高级设置后才能改为填写自定义 HTTP(S) 下游。

## Acceptance

- [ ] Route 规格、API、持久化和 Web UI 能区分 `http` 与 `tcp`，并根据类型仅展示有效字段。
- [ ] TCP Route 引用项目内受管 Service Component 的已声明 TCP Endpoint；跨项目、未声明、非 TCP 和无权限目标均被拒绝。
- [ ] TCP Route 的监听端口字段位于 Service/Component/Endpoint 选择之后；选择 Endpoint 时预填其容器端口，用户仍可修改公开监听端口。
- [ ] HTTP Route 默认引用项目内受管 Service Component 的已声明 HTTP Endpoint；受管目标与 TCP 一样在服务当前 Version 和有效 Endpoint overlay 中解析。
- [ ] 自定义 HTTP(S) 下游仅在 HTTP Route 的高级模式中可用，并维持既有 URL 格式校验；受管目标与自定义下游互斥。
- [ ] Route 创建与编辑使用 Combobox 选择 Service、组件和 Endpoint，不依赖当前列表页的分页结果；Service 已确定所属应用，无需额外选择应用。
- [ ] Component Endpoint 与 Service Endpoint overlay 不再接受或持久化用户自定义名称；端点身份为 `protocol + container_port`，界面统一派生展示为 `http<port>` 或 `tcp<port>`。
- [ ] HTTP Route 仍产生顶层 `http.routers` / `http.services` 的 REST 快照，保留 Host、PathPrefix 和现有 HTTPS/证书行为。
- [ ] TCP Route 在 REST 快照中产生顶层 `tcp.routers` / `tcp.services`；动态 TCP target 不要求业务容器直接发布宿主机端口。
- [ ] 启用 TCP Route 的 Gateway 静态配置和 Compose 端口映射包含其 `listen_port`；停用或删除后可在 reconcile 中移除未使用端口。
- [ ] TCP Route 对一个 `listen_port` 仅允许一个启用 Route，生成 `HostSNI(*)`；不支持同端口按域名分流。
- [ ] 同一 `listen_port` 不得同时被 Gateway TCP Route 与正在运行的业务 Component 的 `host` / `local` Endpoint 绑定；停止的 Service 不占用宿主机端口。错误信息说明冲突资源和端口。
- [ ] `gateway_tcp` 从 Endpoint mode、产品/API/渲染主路径直接删除；已有库表数据在发布前离线更新为 `internal`，该存量转换不进入业务逻辑。
- [ ] `gateway_http` 从 Endpoint mode、产品/API/渲染主路径直接重命名为 `gateway`；已有库表数据在发布前离线更新为 `gateway`，该存量转换不进入业务逻辑。
- [ ] Route 创建、编辑和证书只写入业务数据，不直接发布 Traefik；列表页和详情页的启停只保留前端未提交草稿；不新增 Route 状态机或异步协调模型。
- [ ] `/routes` 列表页和详情页启用/禁用都保留前端未提交草稿；同步入口先预览业务数据与 Traefik 数据的差异，确认后以完整 enabled Route 集合覆盖 REST provider。
- [ ] 同步确认校验预览期间的业务数据与 Traefik 数据未变化；预览失败、数据过期或覆盖失败时不误报成功。
- [ ] 每次 Gateway Service 的 deploy/restart 成功后，均在同一异步部署流程中发布全量 HTTP/TCP Route 快照；Gateway 详情页发起的部署不例外。
- [ ] 对 HTTP 路由、独占端口 TCP、端口冲突、Gateway reconcile、权限和目标有效性提供覆盖测试。

## Open questions

不适用。本需求已确认：`gateway_tcp` 直接删除，历史数据离线转换为 `internal`；`gateway_http` 重命名为 `gateway`，历史数据离线转换；Route 变更先写业务数据，由统一同步入口预览并确认后全量覆盖 Traefik；Gateway 成功部署后自动同步全量自定义 Route。TLS/SNI、异步协调、HTTP target 收敛、端口策略和防火墙不扩展本需求范围。

## Decisions

- 采纳：将自定义路由升级为 HTTP/TCP 公开入口，TCP Route 可表达域名加端口的连接地址。
- 采纳：TCP Route 的域名仅用于 DNS 和客户端连接地址；每个公开监听端口只能属于一条 TCP Route，不支持 TLS/SNI 同端口分流。
- 采纳：Redis 等服务的默认公开路径是 TCP Route；组件保留 `internal`、`local`、`host`，不将 `gateway_tcp` 保留为并行公开入口。
- 采纳：Gateway entrypoint、宿主机端口映射和 Traefik TCP 动态配置由 Route 控制面统一 reconcile。
- 采纳：直接删除 `gateway_tcp`；发布前对现有 Version 与 Service Endpoint 表执行离线数据更新，将值改为 `internal`，不在业务逻辑中承载迁移或兼容。
- 采纳：将 `gateway_http` 直接更名为 `gateway`；发布前对现有 Version 与 Service Endpoint 表执行离线数据更新，将值改为 `gateway`，不保留旧枚举值。
- 采纳：Route 发布入口统一为全量同步；Route 创建、编辑和证书只更新业务数据，列表页和详情页启停都只更新前端草稿，不调用 Traefik 增量启停接口。
- 采纳：同步使用预览/确认两阶段；预览返回业务与 Traefik 的 hash 及差异列表，确认时校验 hash，确认后覆盖完整 REST provider 数据。
- 采纳：Gateway Service 的 deploy/restart 成功后调用 Route 全量同步；此步骤位于异步部署执行路径，覆盖 Gateway 详情页和其他调用通道。

## Risks

- TCP Route 改变 Gateway 的静态监听端口，需要滚动/重新部署 Gateway；短暂连接中断、宿主机端口占用和防火墙未放行都必须在设计和运行反馈中处理。
- 删除 `gateway_tcp` 触及版本、服务覆盖、渲染、MCP、HTTP/Proto、前端、文档和已存数据，需要完整迁移设计和跨层回归验证。
- 公开 Redis 等数据服务会扩大攻击面；TCP Route 的可访问性不能替代 ACL、认证、TLS 和网络防火墙策略。

## User review notes

- 用户已采纳“将自定义路由扩展为域名加端口的服务暴露入口”的方向。
- 用户确认 `gateway_tcp` 直接删除、`gateway_http` 改名为 `gateway`，存量通过业务逻辑外的离线数据更新完成；Route 复用既有同步逻辑。
- 用户补充：Gateway 详情发起部署后必须同步全量自定义 Route，避免 Gateway 重建导致路由丢失，并要求进入 Spec。
- 用户补充：HTTP Route 默认应选择应用和组件的 HTTP 网络，自定义下游是高级功能；TCP Route 同样选择应用和组件的 TCP 网络；目标选择使用 Combobox，避免分页列表截断候选项。
- 用户补充：应用选择是多余层级；选定 Service 后直接选择该 Service 的组件和端口。
- 用户补充：TCP Route 在组件与 Endpoint 选定后展示监听端口，并以 Endpoint 使用的端口预填，允许用户改写。
- 用户补充：已停止的 Service 不应以其静态 Endpoint 配置阻止 TCP Route；冲突检查仅针对正在运行的 Service。
- 用户补充：移除组件和服务网络的名称字段，改由协议和容器端口派生 `http<port>` / `tcp<port>` 标识。
