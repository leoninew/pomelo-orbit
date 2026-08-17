# Gateway Service Lifecycle
最后修改时间: 2026-08-15 12:03:03

Review status: Accepted

## Background

当前 Gateway 已使用 `Application(kind=gateway)`、`Version`、`VersionComponent`、`Service` 和通用 `Deployment` 模型，但资源创建与部署入口尚未统一：HTTP 创建仅写入 Application、GatewayConfig、Version 和受管 Traefik Component；Gateway 详情页在用户点击部署时才创建或改绑 Service；MCP `orbit_provision_gateway` 又在独立编排中创建 Service、发布 Version 后调用通用 Service 部署。

这使同一个 Gateway 的资源生命周期、Version 状态和 Service 创建时机取决于入口。Gateway 的实际运行单元已经是 Service，因此创建和部署边界应以 Service 为中心收敛。

## Goal

1. 创建 Gateway 时，在同一个业务操作中创建其 Application、GatewayConfig、受管 Version/Component 和一个已停止的默认 Gateway Service。默认 Service 的 `instance_key=default`、code 为 `<application-code>-default`；它是初始部署目标，而非 Gateway 唯一允许的 Service binding。
2. Gateway 的 Deploy、Restart、Stop、日志、状态和 Deployment 记录全部复用通用 Service / Deployment 路径。Gateway 页面不得在“部署”操作中隐式创建或改绑 Service；显式 Provision/Service 创建操作可以使用通用 Service 创建能力建立指定实例。
3. 保留 Gateway 的必要平台业务扩展，但优先复用通用 Service/Deployment 校验。Gateway 特有逻辑仅限创建初始资源组，以及启动后的 REST 就绪等待和全量 Route 快照发布；不增加 Gateway peer 状态或 Gateway 可用性的部署前预检，也不在部署时覆盖常规 Version 配置。
4. 所有入口在 Version 发布策略、Service 选择和异步部署结果上产生相同结果；`orbit_provision_gateway` 仅作为幂等资源准备入口，创建/解析资源后返回默认 Service，不再隐式发布或部署。
5. 初始 Gateway Version 的 Traefik 组件、端点、挂载和静态配置必须是普通 Version 规格：用户创建 Gateway 后可继续从 Application、Version 和 Service 页面按常规能力修改、发布、选择和部署，不能被 Gateway Deploy 过程重写。
6. 初始 Version 必须复用已有的 Gateway、`traefik.*` 与 `cert.letsencrypt.*` 配置作为默认来源；不新增 `image_pull_policy`、默认入口/TLS 或共享网络配置，保持当前 `missing`、`web`、`none` 与 `traefik` 默认。它们分别可由 Version 或 GatewayConfig 在创建后修改；Traefik 容器的固有协议路径、端口和 provider 模板同样作为 Version 默认内容，不做无差别进程配置。

## Non-goal

1. 不引入新的 Gateway 专用 Deployment、任务、Service 状态或 Docker 生命周期命令。
2. 不改变 Route 的 HTTP/TCP 动态快照模型，或引入多 Gateway/Kubernetes 的产品能力；多个 Gateway Service binding 不表示支持多个并发运行的 Traefik。
3. 不为旧的“部署时才创建 Gateway Service”数据或 API 保留兼容分支；开发数据库和当前业务数据按项目既有迁移/重建策略处理。
4. 不在本需求中把标准 Application 的 Service 创建流程改为创建 Application 时自动产生 Service。
5. 不把 Gateway Version/Component 变成只能由 Gateway usecase 编译、隐藏或覆盖的内部配置；也不因 Gateway 而缩减常规 Application、Version、Service 编辑能力。

## User scenarios

1. 用户创建 Gateway 后，详情页即可看到一个 `instance_key=default`、code 为 `<application-code>-default`、状态为 `stopped` 的 Gateway Service，不必先发起部署才能拥有运行绑定。
2. 用户从 Gateway 详情页部署选定的 Gateway Service 时，系统默认使用其已绑定 Version；选择其他 Version 时先按普通 Service 规则更新绑定，再调用通用 `DeployService`。Deployment 详情、日志、取消和故障状态与标准 Service 一致。
3. 用户显式创建 `instance_key=staging` 的 Gateway Service 时，其 code 为 `<application-code>-staging`。它可作为另一个 Service binding 保存或部署；当同 Application 的 default 实例正在运行时，由与常规应用相同的单运行实例规则拒绝并发启动，而不是 Gateway 专用校验。
4. MCP 在空项目调用 `orbit_provision_gateway` 时创建完整 Gateway 资源组并返回默认停止态 Service；重复调用复用同一资源组和同一 Service。用户可先在 Application/Version 中配置，随后显式调用通用 Service 部署；指定非默认实例的显式 Provision 使用通用 Service 创建/查询语义。
5. Gateway Compose 启动成功后，系统等待 Traefik REST 控制面可用，再发布完整 HTTP/TCP Route 快照；任一步失败均以通用 Service/Deployment faulted 结果呈现。
6. 用户创建 Gateway 后进入其 Application 页面，修改未发布 Version 的 Traefik Component、endpoint、mount、镜像或拉取策略，并按普通 Version/Service 流程发布、绑定和部署；Gateway 部署使用该选定 Version 的完整规格，不自动重新编译、替换或补回受管内容。
7. 用户更新 GatewayConfig 后，后续 Gateway route 同步和需要 GatewayConfig 的通用渲染使用该配置；它不隐式改写既有 Version。若要改变 Traefik 进程静态配置，用户在 Application/Version 中显式修改并部署目标 Version。

## Acceptance

1. Gateway 创建成功后，持久化存在同属一个 Gateway 的 Application、GatewayConfig、Version、受管 Traefik Component、默认 Service 及其 Component mapping；默认 Service 为 `stopped`，`instance_key=default`，code 为 `<application-code>-default`。
2. 上述创建为原子操作：任一持久化步骤失败后，不遗留部分 Gateway Application、Version、GatewayConfig 或 Service 数据。
3. Gateway 详情页以已存在的 Service ID 调用通用 Deployment command；不再在页面部署事件中创建、更新或猜测 Service 绑定。显式 Provision 为给定 instance key 创建 Service 时必须调用通用 Service 创建能力，但不得发布 Version 或发起部署。
4. Gateway 的部署、重启和停止继续产生普通 Deployment，使用相同的队列、取消、日志、状态转换和工作目录规则。
5. Gateway 与标准 Service 的部署前只使用通用的权限、Service/Version 所属、同 Service 活跃 Deployment 和同 Application 单运行实例校验。系统不再因 Gateway peer 状态或 Gateway 是否运行而额外拒绝入队；端口、外部网络和容器名冲突由同一 Compose 执行路径返回错误。
6. Gateway 成功启动后仍执行 REST readiness 和 Route 全量快照发布；发布失败时 Service 与 Deployment 都为 `faulted`。
7. Gateway Service 部署严格使用选定 Version 及其 Service overlays 的通用有效计划；Gateway usecase、Route usecase 和 worker 不得在部署中重新编译、替换或补回 Version Component 的端点、挂载、镜像或静态文件。
8. Web、HTTP、MCP、Proto、Go usecase 与测试对默认 Gateway Service 的识别和部署行为保持一致。
9. Gateway 创建所生成的初始 Version 默认值可追溯到既有配置：`traefik.image`、REST 地址、域名、证书目录、就绪超时和 `cert.letsencrypt.*` 必须接入。拉取策略、默认入口/TLS 和共享网络保持当前默认值，但创建后的实际部署规格始终以用户保存的 Version 为准，不能再以进程配置覆盖。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. Gateway 仍是 `kind=gateway` 的 Application，GatewayConfig 仍是平台控制面配置；“Gateway 是普通 Service”指其运行、部署和状态机归 Service/Deployment 所有，不是删除 Gateway 领域策略。
2. Gateway 支持普通 Service 的实例模型：创建时建立 `default` binding，Service code 始终为 `<application-code>-<instance-key>`；可以存在多个停止态或候选实例，其并发行为由通用的同 Application 单运行实例规则约束。
3. Gateway 的专用业务只能通过通用部署链路的 Render/后置扩展实现，不新增平行的 Gateway deploy executor 或 Gateway 专用部署前状态预检。
4. 本需求是独立的 `20260815-` SpecFlow 任务；既有 Gateway runtime、TCP Route 和删除任务文档不改写历史结论。
5. 初始 `traefik.yml`、ACME 挂载、TCP entrypoint、共享网络和 Dashboard 相关规格在 Gateway 创建时写入初始 Version Component，之后由标准 Application/Version 编辑与发布流程所有。Gateway Deploy 不注入、覆盖或恢复这些内容；Route 变更也不隐式编译 Gateway Version。Version 发布前置遵从所有 Service 共用的规则；本任务不为 Gateway 增加例外，Provision 也不隐式发布或部署。
6. `traefik.image`、`rest_api_url`、`base_domain`、`cert_dir`、`rest_ready_timeout` 和 `cert.letsencrypt.*` 是既有配置来源，必须进入创建初始 Version 或部署后等待的对应环节。GatewayConfig 持久化每个 Gateway 的 REST 地址、域名、默认入口和 TLS 模式。`missing`、`web`、`none` 与 `traefik` 保持当前默认；Docker socket、默认端口、静态文件路径和 provider 细节作为可编辑 Version 默认模板，不扩张为进程配置。

## Risk

1. 创建由当前多次写入加补偿删除改为原子资源组，需要跨 Application、Gateway 和 Service 仓储建立明确的事务边界。
2. 默认及额外 Gateway Service 的 Version 更新、组件 mapping 对齐和删除约束必须覆盖完整生命周期；通用单运行实例规则必须不再排除 `kind=gateway`。
3. GatewayConfig 与 Version 分属不同配置面：前者控制 Gateway 领域路由语义，后者控制 Traefik 容器规格。两者边界不清会导致保存 GatewayConfig 时错误重写用户 Version，或以 Gateway 部署名义绕过 Version 发布流程。
4. 去除 Gateway 状态预检后，端口、外部网络或容器名冲突会在 Compose 执行时而非 API 入队时反馈；部署日志和错误映射必须保持可诊断。
5. 初始配置只负责创建默认 Version；配置更新不会追溯修改已经保存的 Version。需要调整某个 Gateway 的 socket、端口或 provider 等静态规格时，用户编辑并发布该 Gateway 的 Version；不为这些 Version 字段增加全局配置镜像。

## User review notes

- 用户确认：Gateway 创建应同时创建附带的 Application、Version 和 Service；Gateway 部署本质上是调用 Service 部署，Gateway 只保留部署前后及渲染中的特定业务。
- 用户补充：Gateway 仍使用 Service 实例模型；当前实例均为 `default`，Service code 使用 `<application-code>-default`，未来可能使用其他 instance key。
- 用户确认切换到标准模式，并确认上述受管静态物料与 Version 状态边界不存在未决事项。
- 用户补充：Gateway 应尽量复用常规应用校验；不为部署前 Gateway 状态或实例并发另建专用校验，常规约束能覆盖时优先使用常规约束。
- 用户补充：除受管资源的通用名称外，大多数 Gateway 运行参数已有配置来源；先复用这些来源，未覆盖的参数补入配置，禁止在代码或静态 YAML 模板中写死。
- 用户补充：Gateway 创建后仍要在 Application、Version 和 Service 中按常规能力继续配置；它不是一次性受管 Render，Gateway 部署不得覆盖用户保存的 Version 规格。
- 用户进一步明确：创建 Gateway 后先配置 Application/Version；Provision 不能把资源创建、Version 发布和默认 Service 部署做成一把梭。
- 用户最终确认：无需为拉取策略、默认入口/TLS 和共享网络新增配置，保持当前默认；既有配置继续作为初始默认来源，用户仍可在 Application/Version 修改最终规格。
- 用户确认：本次采用新的 `20260815-` 需求，既有 Gateway SpecFlow 过程文档保持不动。
- 用户要求进入 Spec / 规格阶段；本 Requirement 视为 Accepted。
