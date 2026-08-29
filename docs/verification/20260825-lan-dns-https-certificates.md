# 局域网 DNS-01 HTTPS 与 Gateway 配置收敛验收记录
最后修改时间: 2026-08-29 15:14:37

Review status: Draft

流程模式: 严格 / strict

## Requirement Alignment

1. Gateway 的 Compose 拓扑已收敛到普通 Application、Version、Component 与 Service。创建 Gateway 会建立 `base`、`http`、`dns`、`http-dns` 四个普通 Version；基础 Version 声明 80/443/8080、Docker socket、静态 `traefik.yml` 与 cert/acme mount。
2. `GatewayConfig` 只保存 Gateway 业务配置、ACME profile、email、DNS token 和 profile Version binding；未引入 listener、mount、endpoint 或 resolver 行。
3. Gateway 部署先按保存的 profile 切换默认 Service 到绑定 Version 并写入 deployment snapshot。worker 的 enrichment 只校验并更新已声明 resolver 的 `acme.email`，DNS profile 仅在有效环境变量中设置 `CF_DNS_API_TOKEN`。
4. Route 的 HTTP-01/DNS-01 可用性来自 Gateway profile，DNS-01 额外要求 token；`letsencrypt` 与 `letsencrypt-dns` resolver 名保持不变。TCP Route 继续核对当前 Version 已声明的 endpoint。
5. `POMELO_ORBIT_CF_DNS_API_TOKEN`、全局 Cloudflare Settings、Compose secret 和 provider availability 路径已移除。代码搜索仅保留部署 enrichment 中的 `CF_DNS_API_TOKEN`。
6. 历史 `000034_seed_gateway` 文件已删除。`000037_seed_gateway` 在新库中创建一个标准 Traefik Application、四个绑定 Version、四个 Component、停止态默认 Service 和 Dashboard Route；ACME profile、email 与 DNS token 保持为空，按需由用户填写。

## Spec Alignment

- SQLite/MySQL `000036_gateway_config_runtime` 及 schema/query/sqlc/model/repository/proto/HTTP/web type 已同步 `acme_profile`、`acme_email`、`dns_api_token` 与 `gateway_acme_profile_version`；递增的 `000037_seed_gateway` 创建完整默认 Gateway 数据，并清理已执行的旧 seed。
- `gateway_enrichment.go` 用 YAML AST 修改已有 email 标量，缺失 component、controlled file、resolver、email 或 DNS token 时在 Compose 前失败；不再投影任何 Component 拓扑。
- DNS profile 的静态配置使用公共 resolver `1.1.1.1:53`、`8.8.8.8:53` 和 `delayBeforeChecks: 60s`，对应 DNS-01 传播等待场景。
- Gateway UI 改为详情与编辑分离：编辑按卡片/模态窗独立保存；详情不展示无业务价值的 ID，不再使用假脱敏，DNS token 使用既有可见性切换组件。

## Plan Alignment

计划中的数据库、Gateway 创建/部署、Route capability、前端 profile 表单和回归测试均有实现。用户随后要求删除历史 seed 文件后，完整默认数据以递增的 `000037` 落地，并由既有迁移集成测试和 `dbtalk` 新库查询核对。额外完成了用户随后要求的运行日志可见性：Gateway 和 Route 详情提供 Traefik 日志入口，Gateway 部署和 Route 操作后自动打开日志抽屉；Service 详情复用同一 runtime 容器日志抽屉，Deployment 详情复用纯展示组件而保留自身 API/lifecycle。

## Actual Diff Summary

当前暂存范围覆盖 Gateway profile、Route capability、运行日志与文档；本次补充的 seed 变更删除 `000034_seed_gateway`，新增 SQLite/MySQL `000037_seed_gateway`，并扩展既有迁移测试和 Gateway 工厂测试。主要变更如下：

| 范围 | 实际变更 |
| --- | --- |
| Gateway/部署后端 | profile Version 创建及 binding、部署前 Version 选择、snapshot、窄范围 Gateway enrichment、Compose/network/runtime 适配与测试 |
| Route | ACME capability、challenge 参数、HTTP mapper/proto、Traefik REST snapshot 与 TCP Version 校验 |
| 数据库 | SQLite/MySQL `000035`/`000036`/`000037`、schema/query/sqlc；历史 `000034_seed_gateway` 已删除 |
| 前端 Gateway/Route | profile 表单校验、详情/独立编辑 dialog、Route challenge dialog、导航和 i18n |
| 运行日志 | `ContainerLogView`、`RuntimeContainerLogsDrawer`、Gateway/Route/Service/Deployment 接入及 Monaco 高度链修复 |
| 文档/配置 | runtime/product/guides/decision/config 说明同步 |

## Expected Vs Actual Changed Files

| 预期范围 | 实际文件族 | 结果 |
| --- | --- | --- |
| migration/schema/query | `sql/migration/**/000035*`、`000036*`、`000037*`、`sql/schema/*`、`sql/query/{gateway,route}`、sqlc 生成物 | 符合 |
| Gateway 与部署 | `internal/application/gateway/**`、`internal/application/deployment/**`、repository/model/handler/proto | 符合 |
| Route | `internal/application/route/**`、Traefik adapter、route proto/mapper | 符合 |
| Gateway UI | `web/src/views/gateway/**`、generated TS、router/navigation、locale、form/navigation tests | 符合 |
| 用户追加的日志体验 | `web/src/components/{ContainerLogView,RuntimeContainerLogsDrawer}.vue`、`runtimeContainerLogs.ts`、Gateway/Route/Service/Deployment detail | 范围扩展，已实现 |
| 无关暂存项 | `docs/requirement/20260825-project-automation-review.md` | 与本 feature 无关，应从该功能提交中拆出 |

## Acceptance Checklist

- [x] 基础 Version 保留固定 80/443/8080、socket、static config、cert/acme mount 与手工证书能力，且不要求 ACME email。
- [x] 每个 Gateway 创建并绑定 `http`、`dns`、`http-dns` profile Version；profile resolver 结构仅存在于 Version/Component。
- [x] GatewayConfig 仅保存控制面、profile、email、DNS token 与 binding，不保存 listener/resolver 布局。
- [x] 部署在创建 snapshot 前选择普通 Service Version；worker 不变更 Component mount、endpoint、listener 或 resolver 结构。
- [x] 部署期只修改声明的 `acme.email`，DNS profile 只传入 `CF_DNS_API_TOKEN`。
- [x] 全局 Cloudflare token/settings、secret workspace、Compose secret、redaction 和 availability API 已从当前代码路径移除。
- [x] Route 的 HTTP/DNS challenge capability 由 profile 和 token 推导，resolver 名稳定。
- [x] TCP listener 只来自选中 Version 的 endpoint；Gateway API/UI/schema 无 listener 行。
- [x] 新库的 Gateway seed 包含标准 Application、四个 profile Version/Component、完整 endpoint/mount、base Service binding 和禁用的自定义 Dashboard Route；旧 `000034` seed 在升级时清理。
- [x] Gateway、Route 提供手动 Traefik 日志入口；部署/Route 操作会自动打开同一日志抽屉。
- [x] 运行日志 Monaco 容器恢复完整 flex 高度链，不再是数像素高的直接 flex item。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `task check` | 通过：web typecheck/lint/Prettier，golangci-lint config/fmt/run，0 issues |
| `go test ./cmd/... ./internal/...` | 通过 |
| `go test ./internal/infrastructure/database ./internal/application/gateway/usecase ./internal/application/route/usecase -count=1` | 通过：既有迁移集成测试覆盖完整新 seed 与旧 seed 清理 |
| `yarn --cwd web test` | 通过：15 个测试文件、75 个测试 |
| `git diff --cached --check` | 通过，无空白错误 |
| `dbtalk database query --dsn sqlite:///./.tmp-gateway-seed-validation.sqlite ...` | 通过：由迁移单测初始化的新 SQLite 库包含 4 个 profile、12 个 endpoint、16 个 mount，默认 Service/Route 与 Gateway 创建模型一致 |

此前的真实运行时排查还验证了 DNS-01 成功路径：修正域名 A 记录至当前 Traefik IP 后，Traefik 已获取 Let's Encrypt 证书，直接 HTTPS 返回有效证书和 HTTP 200。该结果证明异步签发路径可工作，但不替代本次 UI 人工验收。

## Scope Expansion And Missed Scope

- 用户在实现期间追加了 Gateway/Route/Service 的 Traefik runtime 日志抽屉、部署及 Route 操作后的自动展开，以及日志区域高度修复；这部分不在原 plan 中，已纳入本次实际交付。
- Route 与 Traefik REST provider 的统一全量同步、前端启停草稿和 preview/confirm 模态窗已拆分为独立轻量任务，见 [Route REST 快照防误删](../verification/20260819-route-rest-snapshot-safety.md)。它不计入本 DNS-01 验收的范围或未完成项。
- `docs/requirement/20260825-project-automation-review.md` 是独立的 Taskfile 审查，不属于 LAN DNS-01/Gateway feature，当前仅因共用暂存区而出现。
- 未发现需求、规格或计划中遗漏而未实现的功能项。

## Risks And Incomplete Items

1. 尚未对当前构建进行浏览器级人工复验：需要确认 Gateway 部署、Route 更新/证书操作后日志抽屉自动打开，手动入口可用，关闭后停止轮询，Monaco 在真实 Drawer 中保持完整高度。
2. DNS-01 证书签发依赖公开 DNS 委派、Cloudflare token scope、DNS 传播和 Let's Encrypt 外部校验；API 操作成功仅表示 Traefik 接受 snapshot，签发应通过 Traefik 日志与实际 HTTPS 证书继续观察。

## Conclusion

自动化验证、迁移检查、新库 `dbtalk` 对照和实现核对均通过，功能已具备进入人工 happy-path 验收的条件。本记录保持 `Draft`，等待完成浏览器交互复验后由用户确认接受。
