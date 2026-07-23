# CD：网关详情提供部署与停止入口
最后修改时间: 2026-07-23 22:29:09

Review status: Accepted

Flow mode: light

Parent / 关联：
- `docs/requirement/20260722-cd-application-kind-gateway.md`（Application.kind=gateway）
- `docs/requirement/20260721-cd-application-version-deploy.md`（应用部署/停止契约）
- 前端现状：`web/src/views/cd/GatewayDetail.vue` 仅配置 + exposures +「版本与部署」跳转

## Background

1. 网关是 `kind=gateway` 的 Application，仍走统一的 Version / Deploy / Service 生命周期。
2. 网关已有独立路由与详情页（`/cd/gateway/:id`），但详情页没有直接的 **部署** / **停止** 操作。
3. 运维需要在网关详情内完成常见启停，而不必绕到版本详情或服务详情。

## Goal

1. 在 **网关详情** 顶栏提供 **部署** 与 **停止** 入口。
2. 复用既有 `applicationApi.deploy` / `applicationApi.stop`，不新增后端契约。
3. 部署成功 / 停止成功后，若返回 `deployment_id`，跳转部署详情；否则刷新本页运行态。
4. 保留「版本与部署」跳转（进入通用应用工作负载面），作为高级入口。

## Non-goal

1. 不在网关详情内重建完整版本编辑 / 组件挂载 UI。
2. 不新增 gateway 专用 deploy/stop API。
3. 不改变 standard 应用详情或服务详情既有行为。
4. 不做日志抽屉 / 容器列表（服务详情能力）；本期只提供操作入口。

## User scenarios

1. 用户打开网关详情，选择已发布版本与环境，触发部署，进入部署记录查看进度。
2. 用户在网关已有运行中/故障服务时，从详情触发停止，可选是否删除 volumes。
3. 用户仍可通过「版本与部署」进入应用/版本面做 fork、发布、预览等高级操作。

## Acceptance

1. 网关详情顶栏可见 **部署**、**停止** 按钮（与编辑/返回并列）。
2. 部署弹窗至少包含：版本、环境、instance_key、force_recreate；校验失败时有明确错误提示。版本优先列出已发布；**无已发布版本时回退为全部版本并默认选最新**（按 `created_at`）。
3. 停止按钮仅在存在可停止服务（`running` / `faulted`）时可用；确认弹窗支持 remove_volumes。
4. 操作调用既有 application deploy/stop API；成功后跳转 deployment 详情或刷新。
5. 中英文 i18n 覆盖新增文案。

## Open questions

不适用（light：按既有 ServiceDetail / VersionDetail 交互收敛）。

## Decisions

1. 部署表单对齐 VersionDetail + ServiceDetail 的字段并集：version + environment + instance_key + force_recreate。
2. 停止目标为该网关 application 下可停止的 Service；若多个，弹窗内选择 service。
3. kind 字段规范值为 `gateway`（用户口语 “gate” 与此同义）。
4. 部署版本选择：有 published 时只用 published；否则用全部版本 newest-first，默认最新。

## Risk

1. 多实例网关服务选择错误可能停错 instance — 弹窗展示 environment / instance_key 降低误操作。
2. ~~无 published 版本时部署应失败并提示~~ — **已由 Decision §4 取代**：无 published 时回退最新版本；仅在「无任何版本」时提示。
3. 网关详情页脚本会变重；保持仅前端 UI 编排，不抽公共 composable（light 范围）。
