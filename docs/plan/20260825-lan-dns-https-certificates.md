# 局域网 DNS-01 HTTPS 与 Gateway 配置收敛实施计划
最后修改时间: 2026-08-29 15:14:37

Review status: Accepted

流程模式: 严格 / strict

## Implementation Steps

1. 收敛 Gateway 持久化契约。
   - 就地更新未提交的 `000036` SQLite/MySQL migration：增加 `acme_profile`、`dns_api_token` 和 `gateway_acme_profile_version`。
   - 删除历史 `000034_seed_gateway` 的 SQLite/MySQL up/down 文件；新增递增的 `000037_seed_gateway`，在已应用旧 seed 的开发库中删除旧数据后，插入当前完整 Gateway 拓扑。
   - 更新 schema、Gateway SQL、sqlc、model、repository、DTO、proto、HTTP mapper 和 web types。Gateway 读写 profile/email/token/profile Version binding，不再读写 listener 或 credential availability。
   - 移除 `POMELO_ORBIT_CF_DNS_API_TOKEN`、Cloudflare Settings 配置、secret workspace、Compose secret、redaction 和相关 bootstrap wiring/tests。

2. 将 Gateway Compose 拓扑还原到 Version/Component。
   - Gateway 创建事务生成 base、`http`、`dns`、`http-dns` 四个普通 Version，以及四个角色 binding 和默认 base Service。
   - 每个 Version 的 `traefik` Component 声明 socket、controlled `traefik.yml`、cert/acme directories、TCP 80/443 和 local HTTP 8080；profile Version 仅在其文件中声明固定 resolver 结构。
   - 保持 TCP 的既有 Version 声明工作流，不创建 Gateway listener 行、端口校验或部署期 endpoint 投影。

3. 以 Gateway profile 选择普通 Service Version。
   - Gateway deploy command 在创建 Deployment 前解析 `acme_profile`，将 Service Version 与 Component mappings 更新为绑定 Version；空 profile 保持 base Version。
   - Deployment snapshot 保存 GatewayConfig 的 profile/email/token；worker 只执行持久化 Version ID。
   - 用窄范围 Gateway enrichment 替代 `projectGatewayPlan`：验证已声明 Component/file/resolver 后写 email；DNS profile 仅追加 `CF_DNS_API_TOKEN` 到 Traefik Component effective env。

4. 让 Route 读取 profile capability，保留 Version TCP 约束。
   - HTTP/DNS Let's Encrypt 校验与详情提示从 `acme_profile` 推导；DNS 额外要求 `dns_api_token`。
   - TCP Route 对 selected Gateway Version 的预声明 entrypoint/host port 保持既有校验，删除 Gateway listener 依赖。

5. 更新 Gateway UI。
   - 删除 listener 表与 HTTP/DNS checkbox/provider availability。
   - 增加可空 ACME profile 选择；选择 profile 后显示 email，DNS profile 显示 DNS token。
   - 详情只展示 base/profile Version bindings、Gateway 业务属性和固定 TCP 应由 Version 管理的说明；部署由 profile 自动选择绑定 Version。

6. 覆盖回归。
   - Gateway create/profile binding、profile 切换、基础 Version 无 email、HTTP/DNS/email/token enrichment、Route capability、Version TCP endpoint 与 Compose 输出。
   - `000036` 的新 schema、`000037` 的完整 Gateway seed、Gateway create/profile binding，以及移除环境变量/secret workspace/redaction 的配置回归。

## Files To Change

| Area | Files |
| --- | --- |
| Database | `sql/migration/{sqlite,mysql}/000036_gateway_config_runtime.*`、`000037_seed_gateway.*`、`sql/schema/`、`sql/query/gateway/`、`internal/gen/sqlc/gateway/` |
| Gateway | `internal/model/gateway.go`、Gateway repository/DTO/usecase、factory/deployment tests |
| Deployment | command/execution/renderer/enrichment、bootstrap wiring、secret/redaction removal |
| Route | profile capability and TCP Version validation |
| API/UI | Gateway proto/mapper/handler/generated TS、Gateway form/detail/edit/locales |
| Docs/config | current SoT guides、`.env.example`、`configs/config.yaml`、release env files |

## Verification Plan

1. 运行 Gateway、Route、deployment、migration 聚焦 Go tests。
2. 运行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 及 Gateway form tests。
3. 运行 `task check`、`go test ./cmd/... ./internal/...` 与 `git diff --check`。
4. 手工核对：基础 Gateway 无 email 可部署；三种 profile 选择对应 Version；DNS token 出现在 DNS Gateway Compose environment；TCP 端口只能由 Version 声明后被 Route 使用。

## Risks And Rollback

- 本地开发环境中的未提交 migration 直接收敛到当前模型，不保留旧 schema 的迁移、适配或回滚路径。
- DNS token 直通 Gateway database/API/snapshot/Compose 的可见性是已接受的产品边界。
- 开发期若需撤销，使用版本控制回退完整未执行变更并重建开发数据库；不修改已执行 migration。

## User Review Notes

- 2026-08-26：确认按“Version/Component 为 Compose 拓扑来源，Gateway 为业务属性和 profile 选择器”实施。
- 2026-08-26：确认移除 `POMELO_ORBIT_CF_DNS_API_TOKEN`，DNS token 交由 Traefik Gateway 管理。
- 2026-08-27：确认历史 Gateway seed 文件直接删除，不保留占位 migration；完整新 seed 使用递增的 `000037`。
- 2026-08-29：确认将 Route 与 Traefik REST provider 的同步拆分为独立轻量 SpecFlow 任务；本计划不实现其 API、前端交互或验收，见 [Route REST 快照防误删](../requirement/20260819-route-rest-snapshot-safety.md)。
