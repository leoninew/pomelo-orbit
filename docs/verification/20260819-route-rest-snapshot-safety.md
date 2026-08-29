# Route REST 快照防误删验证
最后修改时间: 2026-08-29 14:53:32

Review status: Accepted

Flow mode: light

## Requirement Alignment

原“未登记 router 阻断”方案已由统一 Route 全量同步方案取代。当前验证依据是：Route 业务变更以及列表、详情页的启停草稿不直接发布 Traefik；同步先预览业务/Traefik 对等性，再经 hash 校验确认覆盖；Gateway deploy/restart 继续自动发布完整快照。

## Actual Diff Summary

- 新增 Route sync preview/confirm HTTP API、Proto 和前端同步模态窗。
- 列表页和详情页启停均改为前端草稿；禁用 Route 名称始终可进入详情，详情页提供同一同步入口。
- Route 创建、编辑、证书及启停不再直接发布 REST snapshot。
- 移除发布路径中“发现未登记 router 即返回 `unmanaged_traefik_route`”的阻断逻辑，确认后允许完整覆盖。
- 同步预览比较业务快照与全部 Traefik REST routers/services；router 及其 service 聚合为一条逻辑 Route，以“操作 / 规则”两列展示新增、修改、删除及有值一侧的“协议 规则 -> 上游”，不比较证书或 TLS 内部配置，确认前校验双方 hash。

## Expected And Actual Files

| 预期范围 | 实际文件 | 结果 |
| --- | --- | --- |
| Route sync use case / tests | `internal/application/route/usecase/sync.go`、相关测试 | 符合 |
| HTTP/Proto/Web | `internal/api/http/handler/route/*`、`proto/orbit/v1/route/route.proto`、`web/src/views/route/*`、生成物 | 符合 |
| Route publish / Gateway hook | `internal/application/route/usecase/service.go`、Traefik adapter、worker 注入 | 符合 |
| 活文档与决策 | Route guide、CD runtime、`docs/decisions/ledger.md` | 符合 |

## Acceptance Checklist

- [x] 禁用 Route 名称可点击进入详情。
- [x] 列表页和详情页启停都不调用后端 enable/disable 接口，只保留前端草稿。
- [x] 详情页同步按钮会反映启停草稿，基本信息编辑表单不包含启停字段。
- [x] Route 变更不直接发布 Traefik，全量发布统一由同步确认或 Gateway deploy/restart 完成。
- [x] 同步弹窗执行只读预览，以一条逻辑 Route 一行的“操作 / 规则”表格展示业务/Traefik 值；未受管 REST 项不输出 JSON。
- [x] 确认阶段校验预览 hash，过期预览返回冲突且不发布。
- [x] 确认后以完整 Route 集合覆盖 Traefik REST provider。
- [x] 未登记 router 不再被静默合并；其差异会在确认前展示，确认后按全量覆盖语义处理。

## Command Results

| 命令 | 结果 |
| --- | --- |
| `go test ./cmd/... ./internal/...` | PASS |
| `task check` | PASS：typecheck、ESLint、Prettier、golangci-lint 均通过 |
| `git diff --check` | PASS |
| Pomelo PW | 未完成最新版视觉核验：先前进程仍提供旧前端资源；随后 `localhost:9020` 返回 `502`。未确认同步。 |

## Scope Deviations

初版“未知 router 返回 `409/unmanaged_traefik_route`”保护按用户最新决策废止；该偏差已同步写入决策账本和本记录的 Superseded 说明。

## Risks And Incomplete Items

- 未确认同步或执行真实 Gateway deploy/restart；运行中的本地页面未能加载最新版前端资源，需待用户恢复服务后重新核验同步预览表格。
- 同步确认在数据库提交后执行外部 PUT，外部 PUT 失败时需要用户重新执行同步；该一致性边界已在 usecase 注释和活文档中说明。

## Conclusion

统一 Route 全量同步入口、列表/详情页前端启停草稿和同步预览/确认流程已实现，未受管 REST 配置会以可读差异展示，既定检查全部通过。初版未知 router 阻断策略已明确废止并完成文档回写。
