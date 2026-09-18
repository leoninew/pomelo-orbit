# Route REST 快照防误删
最后修改时间: 2026-08-29 14:46:35

Review status: Accepted

Flow mode: light

## Superseded

本需求的初版方案是发现未登记的 Traefik `@rest` router 后拒绝全量覆盖。该方案已被 2026-08-29 用户确认的统一 Route 发布入口取代；当前仍保留“避免静默误删”的目标，但通过同步预览、差异展示、hash 校验和用户确认实现。

## Background

Orbit 通过 Traefik `providers.rest` 管理自定义 HTTP/TCP Route。REST provider 使用完整 PUT 替换语义，因此业务 Route 与 Traefik 动态配置可能出现不一致；未登记的 router 也可能在确认覆盖时被清理。

## Goal

让 Route 变更统一由全量同步入口发布，并在覆盖前让用户看见业务数据与 Traefik 数据的差异，避免无感知地覆盖或误报同步成功。

## Non-goal

- 不从 Traefik 自动导入任意动态配置。
- 不在 Orbit 中合并或保留未由 Route 表管理的 router、service、middleware 或证书配置。
- 不改变 Traefik `providers.rest` 的全量 PUT 模型。

## User scenarios

1. 用户在 `/routes` 列表页启用或禁用 Route 时，只修改前端未提交草稿，不调用后端启停接口。
2. 用户点击同步后，系统按自定义 Route 可见的域名、路径、目标地址、协议和 TCP 监听端口读取并比较 Traefik REST routers、services；不读取或展示证书、证书解析器和其他 Traefik 内部配置。预览以一条逻辑 Route 一行展示新增、修改或删除，规则列只展示有值的业务数据和 Traefik 数据，值使用可读的“协议 规则 -> 上游”形式，不输出 JSON；用户确认后以业务数据覆盖 Traefik。
3. 用户在详情页或列表页启用或禁用 Route 时，都只修改前端未提交草稿；创建、编辑和证书操作先写入业务数据，后续统一同步才会发布完整快照。
4. Gateway deploy/restart 成功后，worker 自动发布完整启用 Route 快照，避免 Gateway 重建后动态配置丢失。

## Acceptance

- Route 创建、编辑、证书和详情页启停不直接调用 Traefik 增量发布接口。
- 列表页和详情页启停只维护前端草稿；列表草稿跨分页保留，并在同步预览/确认时一并提交。
- 同步预览只读读取业务和全部 Traefik REST 路由状态，返回稳定 hash、匹配结果和按逻辑 Route 聚合的可读差异；每行含新增、修改或删除及有值一侧的“协议 规则 -> 上游”，不输出完整 JSON 配置。
- 同步确认校验预览期间业务 hash 与 Traefik hash 未变化；校验失败时不更新业务数据、不覆盖 Traefik。
- 确认成功后以完整 enabled Route 集合覆盖 REST provider；未知 Traefik 配置不自动保留。
- Gateway deploy/restart 成功路径继续发布完整启用 Route 快照。

## Decisions

- 采用 `/api/route/sync/preview` 与 `/api/route/sync/confirm` 两阶段接口。
- 采用“预览差异后确认覆盖”替代“发现未知 router 即拒绝”，因为 REST provider 的产品语义就是由 Orbit 全量管理。
- 未由业务 Route 管理的 Traefik REST router/service 也在预览中展示为业务数据缺失；确认后仍按全量快照覆盖，不自动保留。
- 详情页和列表页启停都不调用后端启停接口，统一延迟到同步确认。
- 同步差异表格采用“操作 / 规则”两列；规则内以“业务数据：”和“Traefik 数据：”标记双方值，避免“当前 / 目标”在全量覆盖语义下产生歧义。

## Risk

- 预览和确认之间仍可能发生外部 Traefik 写入；确认 hash 可拒绝已发生的变更，但 PUT 前后仍需由受管路径避免并发外部写入。
- 同步失败可能使业务数据已更新而 Traefik 尚未覆盖；界面必须保留错误反馈，用户可重新预览并确认。
- 确认覆盖会删除预览中展示的未受管 Traefik REST 配置；手工配置需在确认前迁移为业务 Route 或移至非 REST provider。

## User review notes

- 用户确认 Route 发布统一走全量同步。
- 用户确认列表页启用/禁用仅为前端未提交数据，详情页启停保留现状，基本信息表单移除启停字段。
- 用户确认同步采用预览、确认两阶段；差异经用户确认后覆盖 Traefik 数据。
- 用户补充：Traefik 手工创建的 REST 路由必须展示为可读差异，不输出 JSON；详情页启停改为与列表页一致的前端草稿，并提供同步入口。
