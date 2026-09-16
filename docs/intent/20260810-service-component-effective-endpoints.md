# 服务组件有效端点响应
最后修改时间: 2026-08-10

Review status: Accepted

## Background

服务详情的每个 `ServiceComponentResp` 仅返回稀疏的 `endpoints` 覆盖。未覆盖时，前端无法获得 Version 声明的协议、容器端口及最终 mode，因此不能在服务详情中正确展示类似 Docker Desktop 的端口信息。

## Goal

- 服务详情响应中的每个组件返回 `effective_endpoints`。
- 值必须来自 `BuildEffectiveServicePlan` 的 `plan.Components`，即 Version 声明与 Service 稀疏覆盖合并后的唯一运行时投影。
- 原有 `endpoints` 继续只表达可编辑的稀疏 Service 覆盖；不得用它冒充有效端点，也不得由前端合并两份数据。
- 服务详情组件表新增端口列，按“模式标签、协议/容器端口”显示已发布端点；`internal` 端点不作为已发布端口显示。

## Non-goal

- 本次不实现可点击的网关 URL 或运行时端口探测。
- 不新增持久化字段、迁移、默认值、兼容字段或按组件 N+1 的详情请求。

## Acceptance

- `GET /api/service/{service_id}` 及其他返回 Service view 的响应中，每个成功构建有效计划的组件都有与 `plan.Components` 一致的 `effective_endpoints`。
- Service 端点覆盖可改变 mode、监听地址、监听端口、entrypoint 与 path prefix，但不能改变协议或容器端口；响应反映合并结果。
- 有效计划无法构建时，保留现有 `effective_error`，不伪造有效端点。
- Go mapper 与 service usecase 测试覆盖合并结果和响应投影。
- 服务详情组件表显示生效的非 `internal` 端点；没有已发布端点时端口单元格留空。
