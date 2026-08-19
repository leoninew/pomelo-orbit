# Route REST 快照防误删
最后修改时间: 2026-08-19 18:21:40

Review status: Accepted

Flow mode: light

## Background

Route 禁用会向 Traefik `providers.rest` 提交完整 HTTP/TCP 快照。`orbit.think.preflite.cn` 曾仅存在于 Traefik 的 `@rest` provider 而没有对应的 Orbit Route 记录；禁用另一条 Route 时，该未知 router 被全量 PUT 清除并返回默认 404。

## Goal

在任何会发布 Route 快照的操作前，阻止 Orbit 覆盖未由 Route 表管理的 Traefik `@rest` router，避免自定义域名再次被静默删除。

## Non-goal

- 不从 Traefik 自动导入任意动态配置。
- 不合并或保留未知 REST provider 的 router、service、middleware 或证书配置。
- 不改变 `providers.rest` 的全量快照模型，也不修改 Gateway 静态 entrypoint。

## User scenarios

1. 用户禁用、删除、编辑、同步或修改证书前，Traefik 有未登记的 `@rest` router：操作返回明确冲突，Route 数据和 Traefik 配置保持不变。
2. 运维将遗留 router 按相同名称创建为启用的 Orbit Route：该 Route 可接管同名 router，之后由完整快照管理。
3. 已入库且启用的高级 URL target Route，在禁用另一条 Route 后继续存在于快照中。

## Acceptance

- 所有直接发布 Route 快照的写操作和 Gateway 重建后的快照发布，均在 PUT 前检查 HTTP/TCP `@rest` router。
- 未登记 router 返回 `409`、稳定错误码 `unmanaged_traefik_route`，且不会修改 Route 表或调用快照发布。
- 允许新建或重新启用的同名 Route 接管历史 router；已启用 Route 改名不得接管同名未知 router。
- 高级 URL target Route 不会因禁用其他 Route 而从快照遗漏。
- 活文档说明 REST provider 的全量替换限制和人工迁移方式。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 以检测并拒绝覆盖替代读取后合并：Traefik REST provider 只有整体 PUT 接口，不能可靠读取并保留动态配置的完整来源。
- 采用 `409` 而非服务端错误：未知 router 是可由运维登记或迁移解决的配置归属冲突。

## Risk

- 检查与随后 PUT 之间仍存在外部直接写入 Traefik 的竞态；受管 Gateway 的常规路径应只通过 Orbit 修改 `@rest`。
- 现存未知 `@rest` router 会阻断后续 Route 发布，直到运维将其登记为 Route 或迁移到非 REST provider。这是避免再次静默删除的有意保护。
