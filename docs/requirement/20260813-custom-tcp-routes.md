# 自定义 TCP 路由
最后修改时间: 2026-08-13 10:36:10

Review status: Draft

## Background

当前平台自定义 Route 仅发布 Traefik `http.routers` 和 `http.services`，可将自定义域名和路径代理到 HTTP(S) 上游。组件 Endpoint 另有 `gateway_tcp` mode，意图为将 Redis 等内部 TCP 服务通过 Gateway 以域名加端口的形式公开。

这两套入口机制的边界不对称：`gateway_tcp` 的公开监听端口需要 Gateway 静态 entrypoint 和容器端口映射配合，但业务 Endpoint 中还要求用户手工填写 entrypoint；部署链路没有把标准服务的 TCP 端口集合统一 reconcile 到 Gateway。普通 TCP 协议也不携带 Host，不能按域名在同一个端口分流。结果是服务详情中的 `host` 端口暴露与 `gateway_tcp` 入口容易冲突，用户难以判断何时该使用 Traefik。

## Goal

将平台自定义路由扩展为 HTTP Route 和 TCP Route 两类，由 Route 成为唯一的 Gateway 公开入口控制面：

- HTTP Route 维持域名、路径和 HTTP(S) target 的现有能力。
- TCP Route 能将受管 Service Component 声明的 TCP Endpoint 通过 `domain:listen_port` 对外公开，并由平台管理 Traefik 动态 TCP 路由、Gateway 静态 entrypoint 和 Gateway 宿主机端口映射。
- TCP Route 清楚区分明文 TCP 的端口独占规则与 TLS/SNI 的同端口按域名分流规则。
- 业务 Component Endpoint 不再承担 `gateway_tcp` 公开入口配置；公开 TCP 的生命周期、权限和冲突校验收敛到 Route。

## Non-goal

- 不改变 HTTP Route 的 Host / PathPrefix 路由语义、证书管理和 `providers.rest` 全量发布方式。
- 不实现任意非受管的 `host:port` 作为 TCP Route 默认 target；目标必须可由平台权限、部署状态和网络可达性管理。
- 不尝试让明文 Redis、MySQL 等 TCP 协议依据域名在同一个监听端口中分流。
- 不在本需求中实现 UDP、HTTP/3、PROXY protocol、TCP middleware 或多 Gateway 并行运行。
- 不保留 `gateway_tcp` 与 TCP Route 的长期双重业务 SoT、兼容别名或默认值回退。

## User scenarios

1. 用户为 Redis Service 的 `redis` Component 声明 TCP Endpoint `6379`，创建 TCP Route：`redis.example.com:16379 -> redis:6379`。平台部署/更新 Gateway 后，客户端可以用该地址连接；业务 Redis 容器不直接映射宿主机 `16379`。
2. 用户为启用 TLS 的两个 TCP 服务在相同的 `:443` 或其他相同 TCP 监听端口创建不同域名的 TCP Route。平台仅在 TLS/SNI 可用时允许这样共享端口，并为每条 Route 生成对应 `HostSNI(domain)` 规则。
3. 用户尝试为两个明文 TCP Route 使用同一监听端口。平台在保存或部署前拒绝，并指出端口已被另一条明文 TCP Route 独占。
4. 用户创建 HTTP Route：`termbridge.example.com/ -> termbridge component:80`，现有 HTTP/HTTPS Route 行为不受 TCP Route 引入影响。
5. 用户停用或删除最后一条使用某 TCP 监听端口的 Route。平台在后续 Gateway reconcile 中移除对应的静态 entrypoint 和宿主机端口映射，不影响其他 HTTP/TCP Route。

## Acceptance

- [ ] Route 规格、API、持久化和 Web UI 能区分 `http` 与 `tcp`，并根据类型仅展示有效字段。
- [ ] TCP Route 引用项目内受管 Service Component 的已声明 TCP Endpoint；跨项目、未声明、非 TCP 和无权限目标均被拒绝。
- [ ] HTTP Route 仍产生顶层 `http.routers` / `http.services` 的 REST 快照，保留 Host、PathPrefix 和现有 HTTPS/证书行为。
- [ ] TCP Route 在 REST 快照中产生顶层 `tcp.routers` / `tcp.services`；动态 TCP target 不要求业务容器直接发布宿主机端口。
- [ ] 启用 TCP Route 的 Gateway 静态配置和 Compose 端口映射包含其 `listen_port`；停用或删除后可在 reconcile 中移除未使用端口。
- [ ] 明文 TCP Route 对一个 `listen_port` 仅允许一个启用 Route，生成 `HostSNI(*)`。
- [ ] TLS TCP Route 生成 `HostSNI(domain)`；仅在可验证的 TLS/SNI 模式下允许多个不同域名共享同一 `listen_port`。
- [ ] 同一 `listen_port` 不得同时被 Gateway TCP Route 与业务 Component 的 `host` / `local` Endpoint 绑定；错误信息说明冲突资源和端口。
- [ ] `gateway_tcp` 从业务 Endpoint 的产品/API/渲染主路径移除，且不存在与 TCP Route 并存的公开 TCP 配置来源。
- [ ] Gateway 部署、TCP Route 保存/启停/删除失败时，不得将数据库 Route 状态与已发布 Gateway 配置静默置为不一致；失败状态和恢复动作应可观测。
- [ ] 对 HTTP 路由、明文 TCP、TLS TCP/SNI、端口冲突、Gateway reconcile、权限和目标有效性提供覆盖测试。

## Open questions

1. TCP Route 的 TLS 策略需要区分 Traefik TLS termination 与 TLS passthrough。二者的证书来源、上游是否必须 TLS、以及对 Redis TLS 的推荐配置需要在 Spec 定稿。
2. Gateway 静态配置变更需要重新部署 Gateway。Route 保存操作应同步等待该部署，还是先持久化为待生效状态并由异步任务驱动，需要在 Spec 定稿。
3. `gateway_tcp` 是直接删除现有枚举/存量数据，还是先进行一次明确的数据迁移后删除，需在 Spec 中依据现有数据和迁移约束确定；不得引入长期兼容层。
4. HTTP Route target 是否也应从任意 `target_url` 收敛为受管 Component 引用，不属于本需求的必做范围，但该决策会影响统一 Route 模型的接口设计。
5. TCP Route 的公开端口范围、80/443/8080 等保留端口，以及防火墙策略的产品责任边界需要明确。

## Decisions

- 采纳：将自定义路由升级为 HTTP/TCP 公开入口，TCP Route 可表达域名加端口的连接地址。
- 采纳：普通 TCP 的域名仅用于 DNS；没有 TLS/SNI 时 Route 必须独占监听端口。
- 采纳：TLS/SNI TCP Route 才允许按不同域名共享同一个监听端口。
- 采纳：Redis 等服务的默认公开路径是 TCP Route；组件保留 `internal`、`local`、`host`，不将 `gateway_tcp` 保留为并行公开入口。
- 采纳：Gateway entrypoint、宿主机端口映射和 Traefik TCP 动态配置由 Route 控制面统一 reconcile。

## Risks

- TCP Route 改变 Gateway 的静态监听端口，需要滚动/重新部署 Gateway；短暂连接中断、宿主机端口占用和防火墙未放行都必须在设计和运行反馈中处理。
- Traefik TLS termination 与 passthrough 的安全边界差异较大；错误的默认选择可能导致上游协议不匹配或证书暴露。
- 删除 `gateway_tcp` 触及版本、服务覆盖、渲染、MCP、HTTP/Proto、前端、文档和已存数据，需要完整迁移设计和跨层回归验证。
- 公开 Redis 等数据服务会扩大攻击面；TCP Route 的可访问性不能替代 ACL、认证、TLS 和网络防火墙策略。

## User review notes

- 用户已采纳“将自定义路由扩展为域名加端口的服务暴露入口”的方向。
- 本需求仅记录任务；尚未授权进入 Spec、Plan 或 Implementation。
