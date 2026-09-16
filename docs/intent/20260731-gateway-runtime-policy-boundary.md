# Gateway Runtime Policy Boundary
最后修改时间: 2026-08-02 13:06:01

Review status: Accepted

## Background

Gateway 是 `kind=gateway` 的 Application，已使用 Application -> Version -> VersionComponent -> Service 的统一生命周期模型，但其 Compose 渲染仍在通用部署层用 `kind` 分支同时决定网络、端口发布、Dashboard 标签和部分运行时限制。

在正式环境升级 Traefik 3.6 并使用 `providers.rest` 重部署时，Gateway Compose 将共享网络渲染为顶层键 `default`、物理名 `traefik`。远端已有网络的 `com.docker.compose.network` 标签为 `traefik`，导致 Compose 拒绝接管该网络。该故障暴露出 Gateway 平台策略与 Component 运行时定义的边界不清。

当前还存在多处重复配置来源：`gateway_config.image` 与 Gateway Component 的 `image` 重复；`application.image_pull_policy` 与 Component `pull_policy` 重复；`VersionComponentPort` 与 `ServiceExpose` 都描述了宿主机如何到达 Component 的端口，却分别挂在 Version 和 Service 上。前者当前仅在 Gateway 分支渲染，后者负责普通服务的 local/public 暴露，形成重叠且不对称的端口模型。

## Goals

1. 保持 Gateway 作为有明确平台职责的 `kind=gateway` Application，同时让其 Version、VersionComponent 和 Service 继续成为唯一的运行时配置与部署目标。
2. 明确三层职责：Version/Component 声明镜像的使用方式，包括镜像、命令、挂载、健康检查、资源、拉取策略和候选的容器端点契约；Service 选择 Version 后保存实例绑定、运行时变量与候选的统一 Service Endpoint Binding；GatewayConfig 保存 REST Provider、域名、TLS 默认策略和共享网络策略等平台职责。
3. 消除 `VersionComponentPort` 与 `ServiceExpose` 的重叠。Version 可声明 Component 可提供的容器端点及默认参数，但不能直接拥有宿主机端口绑定；Service Endpoint Binding 是渲染宿主机绑定、local/public 访问与路由所需运行时表达的唯一来源。Gateway 的入口监听是其 Service Binding 的受控投影，而不是独立的第三个运行时真相。
4. 让通用 Compose 渲染器消费显式的部署拓扑/渲染策略，而不是通过单个 `kind` 布尔分支隐式改变 Component 的端口和网络语义。
5. 修复并验证 Gateway 对共享 `traefik` 网络的创建、复用和标准服务外接行为；通过一次就地网络配置迁移保持正式环境的 `preflite.cn` 域名。
6. 保证 `providers.rest` 所需的 Traefik 静态配置、Docker socket 挂载、ACME 状态挂载仍是受控平台配置，不能因普通 Component 编辑而被意外移除。

## Non-goals

1. 不在本需求中引入多 Gateway 并行运行或多个独立平台网络。
2. 不改变现有正式域名、证书账户、业务数据迁移结果或已初始化的 Application/Version/Service 清单。
3. 不通过直接 Docker Compose 生命周期命令绕过 Orbit MCP 执行正式部署。
4. 不把共享平台网络开放为任意 Version Component 的自由编辑字段。

## User scenarios

1. 运维人员创建或升级 Version 时，在 Component 中选择镜像、挂载、拉取策略和可提供的端点；Service 在选定 Version 后配置实例变量与需要发布或绑定的端点。Gateway Service 的端点绑定定义 Traefik 的 80/443 等平台监听，并安全生成受控静态配置。
2. Gateway 首次部署到空主机时，能够创建稳定命名为 `traefik` 的 bridge 网络；后续标准 Application 以 external 网络方式加入它。
3. Gateway 部署到带有既有 `traefik` 网络的主机时，Compose 网络键与稳定标签一致，不会因为项目名或网络键差异失败。
4. 普通 Application 部署前必须有运行中的 Gateway；其 HTTP/TCP 暴露由自身 Service Endpoint Binding 与 Gateway 的已运行入口能力共同决定。
5. 变更 Gateway 的域名、TLS 或 REST Provider 地址不会无意覆盖用户已选择的 Gateway Component 镜像、挂载和其他 Version 参数；入口监听的修改有独立、可审计的策略输入。
6. 运维人员在 Service 详情中按与 Version Component 相同的 Runtime、Connectivity、Mounts、Advanced 分块查看每个 Component 的配置，并能看出一个值来自 Version、Service 覆盖，还是已被 Service 删除。
7. 运维人员只能修改或删除 Version 已声明配置的值字段，不能新增未声明的环境变量、挂载、资源项或 Endpoint。网络配置可以先保存，端口/Gateway/域名/网络策略问题在 Preview 或 Deploy 时处理。保存或删除后，页面显示待部署状态；用户显式部署后才改变运行中的容器。

## Acceptance criteria

1. Gateway 和标准 Application 的 Compose 网络声明拥有明确且可测试的所有权语义；Gateway 网络键、物理名和 Compose 标签稳定为 `traefik`。
2. Compose renderer 不再使用单个 `injectGatewayNetwork` / `publishComponentPorts` 布尔值同时改变多个独立行为；Effective Endpoint、平台网络和 Dashboard 注入均有独立且显式的策略来源。
3. Gateway Component 的运行时字段只有一个权威来源；`gateway_config.image` 与 Application/Component 拉取策略的重复来源得到移除或迁移为清晰的默认值机制。
4. 没有两个模型可同时产生同一个宿主机端口绑定：Version Endpoint 的模板参数与 Service Endpoint 的稀疏覆盖合成为唯一的 Effective Endpoint。`container_port` 是不可覆盖的接口契约，Service 只能改写已声明 Endpoint 的值字段。
5. Gateway 受控静态配置持续启用 Docker Provider、REST Provider、必要入口、ACME 挂载和安全权限；REST 路由发布继续可用。
6. SQLite/MySQL 原始 DDL、SQLC、HTTP/Proto、MCP、前端和测试在字段或契约调整后保持一致；开发数据库重建为当前 schema，不提供旧 schema/API/renderer 或历史数据的兼容路径。
7. 在隔离或正式验证环境中，Traefik 3.6 通过 Orbit MCP 的部署能够完成，并验证 `80/443`、REST Provider 与至少一个公开服务路由。
8. Service 运行时配置支持与现有 Component 详情一致的修改和删除交互。删除有效 Version 配置须显式压制默认值，而不是被误解为继承。
9. Service 组件配置页面与 Version Component 使用相同的功能分块；每个可调值显示 Version 模板值、Service 覆盖值、Effective 值和来源状态，Version 契约字段保持只读。
10. 保存、修改或删除 Service 配置不自动重启 Service。页面明确标记待部署配置，Preview 与下一次 Deploy 使用同一份 EffectiveServicePlan。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. `kind=gateway` 保留为领域能力和平台策略的选择器；目标不是消除它，而是禁止它在通用渲染器中隐式重定义 Component 运行时语义。
2. 共享 `traefik` 网络由 Gateway 平台策略拥有，标准 Application 只以 external 网络消费者身份加入。
3. 正式环境继续使用 `preflite.cn`，并保持 REST Provider 作为动态路由管理机制。
4. 当前未部署的网络键热修复仅作为故障缓解候选，必须纳入后续 Spec 评估，不能替代完整设计。
5. 渲染结果由 Version Component 声明、Service 运行时 Binding 和 Gateway 平台策略组成；Service 只改写 Version 已声明配置的值字段，并通过统一的 `inherit` / `override` / `deleted` 语义持久化。
6. `version_component_endpoint` 与 `service_component_endpoint` 沿用同名字段；后者是字段级稀疏 overlay，唯一运行时结果为 Effective Endpoint。
7. `gateway_config.image` 迁移到 Gateway VersionComponent 后删除；`application.image_pull_policy` 用于补齐现有 Component 的 `pull_policy` 后删除。Gateway 初始创建必须在 VersionComponent 中明确提供镜像和拉取策略。
8. GatewayConfig 是平台策略；其变更生成新的 EffectiveServicePlan 并重新部署 Gateway Service，不派生 Version。Deployment 快照承担该次变更审计。
9. 既有 `traefik` 网络必须在部署前满足网络键和 Compose 标签均为 `traefik`；不满足时网络前置检查失败，不提供运行时兼容分支。
10. Service 运行时配置修改/删除沿用 Component 详情交互。没有 Service 子记录表示继承；删除有效配置持久化为 `state=deleted`，未来恢复默认时才删除该 tombstone 或覆盖记录。所有操作必须引用 Version 已声明的条目。
11. Service 详情以 Component 为入口，并提供与 Version Component 对齐的配置详情页。前端消费服务端返回的声明、overlay 与 Effective 值，不自行合成配置；保存后由用户显式部署。
12. 待部署状态由服务端比较当前 EffectiveServicePlan fingerprint 与最近成功部署快照得出；前端只展示该结果，Preview 与 Deploy 使用同一 fingerprint。
13. 网络相关配置不在保存阶段阻止用户；端口冲突、Gateway/域名/entrypoint 和网络策略在 Preview/Deploy 处理。Gateway Endpoint 与其他 Endpoint 一样按 Version 声明和 Service 覆盖值生效，不按 `kind=gateway` 与 `api` 名称重写 mode；首期仅使用 Deployment 快照审计，不新增覆盖变更历史。

## Risks and assumptions

1. Gateway 接管涉及 `80/443`、ACME 状态和现网 Traefik 容器；任何网络或容器所有权迁移均可能产生短暂中断。
2. 删除 VersionComponentPort 或重建重复字段会影响原始 DDL、生成代码、MCP 与前端表单。开发数据库必须重建；不能依赖历史数据映射或旧数据兼容策略。
3. `providers.rest` 的控制面地址必须从运行中的 Orbit 容器可达；不以本机开发地址作为正式环境假设。
4. 现有远端共享网络及其连接容器必须在实施前完成只读盘点，不能依赖历史 Compose 文件推断其所有权。

## User review notes

- 用户要求使用 SpecFlow 严格模式记录本任务。
- 用户指出 VersionComponentPort 与 ServiceExpose 并存造成双重端口/网络暴露模型，并明确 Version/Component 只声明镜像使用方式、Service 是实际运行时表达；需求已调整为消除重复宿主机绑定并评估 Version Endpoint 与统一 Service Endpoint Binding。
- 用户确认不做兼容层；开发数据库重建后使用更新过的原始 DDL，数据库、API 与渲染路径一并切换。
- 用户确认 Service 运行时配置需要支持修改与删除，并沿用 Component 详情交互；删除 Version 默认配置使用显式 tombstone，不能与继承混淆。
- 用户要求将 Service UI 组织为与 Version Component 相似的功能分块，并支持运行时值的改写操作。
- 用户确认所有 Service 覆盖均基于 Version 已声明条目的值字段；不新增挂载等例外。网络策略延后到 Preview/Deploy，保存不阻止用户；Gateway `api` 沿用通用 Endpoint 覆盖与部署语义，不建立独立覆盖审计。
- 用户要求进入 Spec / 规格阶段，Requirement 已接受。
