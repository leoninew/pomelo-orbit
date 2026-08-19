# Route REST 快照防误删验证
最后修改时间: 2026-08-19 18:21:40

Review status: Accepted

Flow mode: light

## Requirement Alignment

- Route 发布前读取 Traefik HTTP/TCP router，只检查 `provider=rest` 的项。
- 发现未登记 `@rest` router 时，应用服务返回 `409/unmanaged_traefik_route`，写操作发生前中止。
- 新建或从禁用状态启用的同名 Route 可接管历史 router；已启用 Route 改名遇到未知同名 router 会拒绝。
- Gateway 部署完成后的 `PublishSnapshot` 也使用相同保护。

## Spec / Plan Alignment

不适用。light 模式按 Requirement 实现，未创建 Spec 或 Plan。

## Actual Diff Summary

- Route use case 在创建、更新、删除、启停、同步、证书变更和部署后发布前检查 REST router 归属。
- 未受管 router 使用现有统一 HTTP 错误契约输出 `409/unmanaged_traefik_route`。
- 补充集成测试：拒绝禁用时覆盖未知 router、允许同名接管、拒绝改名接管，以及保留高级自定义 target。
- 路由指南和 CD 运行时文档说明保护和迁移方式。

## Expected And Actual Files

| 预期文件 | 实际文件 | 结果 |
| --- | --- | --- |
| Route use case / tests | `internal/application/route/usecase/service.go`、相关测试 | 符合 |
| 活路由文档 | `docs/guides/routing-and-certificates.md`、`docs/architecture/cd-runtime.md` | 符合 |
| light 过程记录 | 本 Requirement 与 Verification | 符合 |

## Acceptance Checklist

- [x] 未登记的 `@rest` router 不会因 Route 操作被静默清除。
- [x] 冲突操作不更新数据库且不发布快照。
- [x] 同名历史 router 可被新的启用 Route 接管。
- [x] 已启用 Route 改名不会接管同名未知 router。
- [x] 高级 URL target Route 在禁用其他 Route 后保留在快照中。
- [x] 文档说明限制、错误码与迁移要求。

## Command Results

| 命令 | 结果 |
| --- | --- |
| `go test ./internal/application/route/usecase ./internal/infrastructure/external/traefik` | PASS |
| `go test ./cmd/... ./internal/...` | PASS |
| `task check` | 未完成：运行 124 秒后工具超时，未输出诊断信息 |

## Scope Deviation

无。未修改迁移、Gateway 静态配置或 Traefik REST provider 的全量 PUT 语义。

## Risks And Incomplete Items

- 外部直接写入 Traefik 与预检查后的全量 PUT 之间仍有竞态；受管路径应避免绕过 Orbit 写入 `@rest`。
- `task check` 未能在当前会话完成，需在本地 CI 或更长执行预算下复跑。

## Conclusion

实现满足 Requirement。定向和完整 Go 测试通过；`task check` 的超时作为未完成验证项保留。
