# 服务级路由与显式网关变更
最后修改时间: 2026-07-30 12:54:42

Review status: Accepted

## Background

当前版本（Version）保存组件暴露配置。标准应用的部署命令已在持久化前校验 Gateway 存在且处于运行状态。部署 `public + tcp` 暴露时，Gateway 用例还会将其汇总为 Traefik 的静态 TCP entrypoint 和宿主机端口发布，并在业务部署中隐式编译、修改及强制重建 Gateway。

这使同一版本无法按 Service 选择不同访问方式，也使业务部署拥有了 Gateway 的生命周期副作用。例如同一 Redis 组件既应能以本地 `6379:6379` 运行，也应能以公网 TCP `tcp6379` 经 Traefik 运行；这应由 Service 的运行配置决定，而非由 Version 决定。

## Goal

1. 将路由和端口暴露的配置归属从 Version 移至 Service 详情。
2. 让同一个 Version 可由不同 Service 分别配置 local 或 public 暴露、监听端口、协议，以及 HTTP 路由所需的运行参数。
3. 由 Service 配置在部署渲染阶段派生 Docker 端口发布与 Traefik labels；labels 不作为 Version 配置持久化。
4. 移除标准应用部署中由 `public + tcp` 暴露隐式编译、修改或重建 Gateway 的行为。
5. 保持既有“Gateway 必须存在且运行”的部署前置校验；当 public TCP 所需网关监听端口尚未由已部署 Gateway 提供时，仍允许业务部署，仅提示用户自行配置并部署 Gateway。
6. 以 Service 作为部署和 Compose 预览的唯一入口；移除从 Version 直接部署或预览运行时 Compose 的入口与契约。
7. Service 基本信息支持修改实例标识和所选 Version；Version 不属于运行时配置编辑区。

## Non-goal

- 不实现自动 Gateway 重建、自动端口开通、自动回滚或通用 Gateway 变更集工作流。
- 不改变明文 TCP 的协议限制：域名仅是客户端连接地址，不能像 HTTP Host 一样用于明文 TCP 路由。
- 不在本需求中引入新的 L4 代理、预分配端口池或 TLS/SNI 多路复用方案。
- 不允许业务 Service 修改 Gateway 的 Version、组件 ports、静态 `traefik.yml` 或部署状态。
- 不保留 Version 部署、Version Compose 预览或 `ApplicationDeployReq(version_id, instance_key)` 的兼容入口。

## User Scenarios

1. 用户编辑 Redis Service 详情，选择组件 `redis`、容器端口 `6379`、协议 `tcp` 和 `local`，监听端口为 `6379`。部署后仅 Redis Service 生成本地端口发布，Gateway 配置与部署不变。
2. 用户编辑同一 Redis Version 的另一 Service，选择 `public + tcp`、监听端口 `6379`。业务 Compose 加入 Gateway 网络并生成 TCP labels；若 Gateway 已显式配置并部署 `tcp6379`，该 Service 可以部署。
3. 用户选择 public TCP，但当前运行的 Gateway 未提供所需监听端口。业务 Service 仍可部署；部署结果提供必要提示，说明需在 Gateway 中配置并部署 `tcp6379`。完成 Gateway 操作后，该端点才能对外连接。
4. 用户部署 public HTTP Service。Service Compose 从 Service 路由配置派生 HTTP labels；固定 HTTP/HTTPS entrypoint 已存在时，不修改 Gateway。
5. 用户查看 Version 详情时，只看到组件规格及其容器内端口契约；不再编辑或持有 local/public、监听端口、Traefik 路由和标签等环境运行绑定。
6. 用户在 Service 详情的基本信息中修改实例标识和 Version，在独立的环境变量卡片与接入暴露卡片中分别修改运行参数与暴露绑定；从该 Service 发起部署或 Compose 预览，Version 详情不再提供部署和 Compose 预览操作。

## Acceptance

- Version 的 API、持久化模型和详情 UI 不再以 Version 暴露配置承载 Traefik 路由或宿主机端口绑定。
- Service 详情支持管理每个组件端口的访问绑定，至少覆盖 `protocol`、`access`、`container_port` 和 `listen_port`；HTTP 场景保留所需的路由参数。
- 部署渲染只从 Service 的有效暴露绑定生成该 Service 的 Docker `ports` 和 Traefik labels。
- local 暴露只影响目标 Service 的 Compose，不触发 Gateway 编译、配置写入或部署。
- public HTTP 暴露不触发 Gateway 编译、配置写入或部署。
- public TCP 暴露不会修改 Gateway Version、托管组件 ports、静态 Traefik 配置或启动 Gateway 部署。
- 保留标准应用既有的 Gateway 存在且运行校验。
- public TCP 所需 entrypoint 缺失时，业务部署仍可成功；部署结果提供非阻断提示，包含所需监听端口与需要用户完成的 Gateway 配置/部署动作。
- Gateway 已部署并提供所需监听端口时，public TCP Service 的 Compose 生成正确的 `tcp<listen_port>` entrypoint label 和后端容器端口，且自身不发布该公网监听端口。
- 现有 Version 暴露数据的迁移、保留或显式废弃策略得到定义；不得静默丢失已有本地或公网访问配置。
- 服务配置变更、部署重试和运行时详情使用同一份持久化 Service 暴露配置，而非一次性部署请求参数。
- 部署 API、Web 与 MCP 的部署和 Compose 预览入口以 Service 为目标；不再接收 Version ID 与 instance key 组合来创建或预览运行时。
- Service 基本信息可修改 `instance_key` 与 Version，且编辑通过与其他详情页一致的模态窗完成；应用归属保持不可编辑，运行时配置请求不携带 Version。
- Service 详情将环境变量和接入暴露呈现为独立卡片，分别编辑、保存和取消，不将它们混入同一个运行时配置卡片。

## Open Questions

1. 现有 `version_expose` 记录如何迁移到 Service：同一 Version 被多个 Service 引用时，是否复制到每个 Service、仅迁移当前运行 Service，或要求用户重新配置？需要在规格阶段按现有数据和产品兼容性确定。
2. Gateway 未提供 public TCP entrypoint 时，前端提示应仅给出端口和操作说明，还是需要提供跳转到 Gateway 详情的入口？
3. Service 暴露配置是否随 Service 直接更新，还是需要引入可回滚的 Service 配置修订版本？本轮至少要求部署可复现。
4. 删除或改为 local 的 public TCP Service 配置后，Gateway 上遗留的静态监听端口由用户何时、以何种 Gateway 操作清理？本轮不应自动清理。

## Decisions

- Version 表达可复用的组件规格；Service 表达环境和实例相关的网络访问绑定。
- Traefik labels 是 Service 配置的渲染产物，不是可编辑或持久化的业务对象。
- Gateway 是独立的受控资源。业务部署只能校验其是否满足需求，不能替用户改变或部署它。
- Gateway 缺失或未运行继续阻止标准应用部署；本轮仅将“缺少 public TCP entrypoint”改为非阻断提示。
- 对明文 public TCP，`listen_port` 是网关唯一的路由资源；`<app>.lvh.me` 仅作为连接地址呈现给客户端。
- Version 仅保留规格操作；部署和 Compose 预览必须以已有 Service 为目标。
- Service 的 Version 和 `instance_key` 是基本信息；基本信息变更必须复用当前 runtime config/exposes 进行原子校验，不能留下与 Version 不匹配的暴露配置。

## Risk

- 这是跨 Version、Service、Deployment、Gateway、数据库迁移、HTTP/Proto 契约和前端工作流的模型迁移，旧数据与并发部署需要在规格阶段明确。
- Service 暴露配置变化可能改变已有 Compose 输出；需要覆盖 local、public HTTP、public TCP、Gateway 未就绪和 Gateway 已就绪的回归场景。
- Gateway 端口状态与数据库期望配置可能不一致。业务部署必须以已部署 Gateway 的实际/已应用监听集合判断，而不能把待配置端口视为可用。

## User Review Notes

- 2026-07-30：用户明确要求将 Traefik 路由能力从 Version 挪到 Service 详情；移除 TCP 暴露修改 Gateway 配置的行为，改为提示用户自行配置和部署 Gateway。
- 2026-07-30：用户澄清标准应用已有 Gateway 存在且运行前置校验，本轮不改变它；public TCP entrypoint 缺失时不应阻止应用部署，只做必要提示，不自动修改或部署 Gateway。
- 2026-07-30：用户要求找出并废弃从 Version 发起部署的入口；部署和预览转为 Service 入口。
- 2026-07-30：用户要求 Service 基本信息可修改并参考创建字段组织；Version 从运行时配置区与请求中移出。
- 2026-07-30：用户要求服务详情基本信息使用模态窗编辑，并将环境变量与接入暴露拆成独立卡片。
