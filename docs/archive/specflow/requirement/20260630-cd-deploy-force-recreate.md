# CD 应用部署支持强制重建
最后修改时间: 2026-06-30 09:40:53

Review status: Accepted

## Background

应用详情页面 `/cd/applications/{id}` 原有部署按钮只支持普通 `docker compose up -d` 部署路径。部分场景下需要显式让 Docker Compose 重新创建容器，以处理容器配置、镜像或运行时状态未按预期刷新的问题，因此需要在不改变默认部署行为的前提下提供可选的 `--force-recreate` 能力。

## Goal

- 将应用详情页部署按钮调整为复合控件：默认主按钮保持普通部署，附加菜单提供强制重建部署。
- 强制重建部署触发后，后端执行 Docker Compose 部署时追加 `--force-recreate`。
- 默认部署路径保持兼容，不因新增能力改变现有普通部署行为。
- 部署触发后仍沿用现有 deployment 记录、异步任务、部署详情跳转和日志查看链路。

## Non-goal

- 不改变应用列表页的部署入口行为。
- 不新增独立部署路由；沿用现有 `/api/cd/application/{id}/deploy`。
- 不修改 deployment 数据表结构，不新增迁移。
- 不改变停止、重启、删除应用等其他操作行为。
- 不主动启动、停止或重启开发服务器。

## User scenarios

1. 用户在应用详情页点击主部署按钮，系统执行普通部署，行为与原有部署一致。
2. 用户在应用详情页打开部署按钮附加菜单，选择强制重建部署，系统创建一条部署记录并执行带 `--force-recreate` 的 Docker Compose 部署。
3. 用户进入部署详情页后，可以像普通部署一样查看部署状态和日志。
4. 旧客户端或旧调用方继续发送空 body 或不传 `force_recreate` 时，仍执行普通部署。

## Acceptance

- 应用详情页部署控件包含默认部署主按钮和强制重建部署菜单项。
- 默认部署请求不会设置强制重建，最终 Docker Compose 参数不包含 `--force-recreate`。
- 强制重建部署请求携带 `force_recreate: true`，后台任务 payload 保留该布尔值。
- Worker 能把 `force_recreate` 从任务 payload 透传到部署执行层。
- 部署执行层仅在 `force_recreate == true` 时追加 `--force-recreate`。
- 缺省或旧 payload 中没有 `force_recreate` 字段时按 `false` 处理。
- 前端 lint/typecheck 和 Go fmt/vet/test 按项目约束通过。

## Open questions

暂无必须由用户确认的未决事项。

## Decisions

- 使用 light / 轻量模式记录本需求。
- 使用现有 deploy endpoint 的可选 JSON body 承载 `force_recreate`，而不是新增 API 路由。
- 使用后台任务 payload 传递执行选项，不落库到 deployment schema。
- 复合控件使用项目已有的 Reka UI Dropdown Menu 模式实现。
- 应用列表页部署入口保持普通部署，避免扩大范围。

## Risk

- 强制重建会导致 Docker Compose 重新创建容器，可能带来短暂服务中断；该行为必须由用户显式选择。
- 如果未来新增更多部署选项，需要继续保持默认部署路径不变，并避免把执行选项隐式持久化为全局状态。
- UI 复合控件需要保持键盘可访问性和禁用态一致性，避免用户在部署进行中重复触发。
