# Traefik 多域名 Router Label
最后修改时间: 2026-07-01 09:55:53

Review status: Accepted

## Background

应用详情中的“路由配置”会在应用启用路由托管时，把 `application_route` 记录注入到部署时生成的 `docker-compose.yml` labels 中。目前每条路由只包含一个 `domain`，并生成单个 `Host(...)` rule。

用户需要支持同一应用服务配置多个域名，并生成 Traefik 支持的 OR rule，例如：

```text
traefik.http.routers.pomelo-orbit.rule=Host(`orbit.typing-island.site`) || Host(`orbit.preflite.cn`)
```

## Goal

- 支持在应用详情中给同一个 service 添加多条域名路由。
- 部署/预览生成 labels 时，将同一个 service 的多个域名合并为一个 router rule。
- 生成规则保持 Traefik label 形式：`Host(`domain1`) || Host(`domain2`)`。
- 保持现有 UI/API/数据库结构不变，优先复用当前多条 `application_route` 记录能力。

## Non-goal

- 不新增前端“多域名输入框”。
- 不调整 `application_route` 表结构。
- 不改造 Traefik runtime routers 只读查看页面。
- 不实现跨 service 的路由合并。

## User scenarios

1. 用户在应用详情中为 `pomelo-orbit` service 添加域名 `orbit.typing-island.site`。
2. 用户继续为同一个 `pomelo-orbit` service 添加域名 `orbit.preflite.cn`。
3. 预览或部署应用时，生成一个 `traefik.http.routers.pomelo-orbit.rule` label，其值包含两个 Host 条件并以 `||` 连接。

## Acceptance

- 同一 service、同一端口的多条路由生成一条合并后的 router rule。
- 不同 service 仍分别生成各自 labels。
- 同一 service 多域名不会因为循环赋值导致只保留最后一条。
- 同一 service 若配置了不同 container port，不应静默选择其中一个端口。
- 前端和后端都要校验 host/domain 合法性，至少拒绝包含空格的值。
- 预览逻辑和部署生成逻辑必须共用同一套路由 label 注入逻辑，避免多 host 预览与实际部署不一致。
- 现有“无路由时清理托管 labels”的行为保持不变。
- 补充后端单元测试覆盖多域名合并行为和非法 host 拒绝行为。

## Open questions

暂无需要用户确认的未决事项。本次按“同一 service + 同一 port 合并域名；同一 service + 不同 port 报错”的假设实现。

## Decisions

- 采用轻量模式 / light。
- 用户已要求“直接实现”，Requirement 视为 Accepted 并进入 Implementation / 实现。
- 不改 API 和表结构，只修改 label 注入逻辑。

## Risk

- 如果用户未来希望一个 service 的不同端口也能生成多个 router/service label，需要新增 router/service 命名策略，不能复用当前单 router 名称。
- 当前表不限制重复域名；实现中会对同 service 的重复 domain 去重，避免生成重复 Host 条件。
