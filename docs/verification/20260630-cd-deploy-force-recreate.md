# CD 应用部署支持强制重建验证
最后修改时间: 2026-06-30 09:44:42

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260630-cd-deploy-force-recreate.md` 核对，本次实现覆盖需求中的核心目标：

1. 应用详情页部署按钮已调整为复合控件：主按钮执行普通部署，附加下拉菜单提供强制重建部署入口。
2. 强制重建部署从前端请求携带 `force_recreate: true`，后端创建 deployment 后将该布尔值写入后台任务 payload。
3. Worker 解析任务 payload 后将 `force_recreate` 透传到部署执行层。
4. 部署执行层仅在 `forceRecreate == true` 时向 `docker compose up` 参数追加 `--force-recreate`。
5. 默认部署路径仍创建普通 deployment、进入异步任务、跳转部署详情，执行层命令不包含 `--force-recreate`。
6. 旧 payload 或请求 body 缺省 `force_recreate` 时，Go bool 零值按 `false` 处理。

需求中的 Non-goal 保持一致：未改变应用列表页部署入口，未新增独立部署路由，未修改 deployment 数据表结构或迁移文件，未改变停止、重启、删除应用等其他操作行为，未主动启动、停止或重启开发服务器。

## Spec alignment / 规格对齐

不适用。当前流程为轻量模式 / light，本次没有单独创建 spec 文档，按 requirement / 需求核对。

## Plan alignment / 计划对齐

不适用。当前流程为轻量模式 / light，本次没有单独创建 plan 文档，按 requirement / 需求核对实现和验证范围。

## Actual diff summary / 实际差异摘要

验证前 `git diff HEAD --stat` 显示的功能和过程文档变更范围：

```text
 .../20260630-cd-deploy-force-recreate.md           | 58 ++++++++++++++++++++
 internal/service/cd/deployment_execution.go        | 12 +++--
 internal/service/cd/deployment_execution_test.go   | 34 ++++++++++--
 internal/service/cd/service.go                     |  8 ++-
 internal/transport/http/application_routes_test.go | 63 ++++++++++++++++++++++
 internal/transport/http/handler/cd/handler.go      | 14 ++++-
 internal/worker/handler/cd/handler.go              |  5 +-
 internal/worker/handler/cd/handler_test.go         | 24 +++++++--
 web/src/api/cd/application.ts                      |  4 +-
 web/src/i18n/locales/en-US.ts                      |  2 +
 web/src/i18n/locales/zh-CN.ts                      |  2 +
 web/src/views/cd/ApplicationDetail.vue             | 58 ++++++++++++++++----
 12 files changed, 254 insertions(+), 30 deletions(-)
```

实际功能改动包括：

- `docs/requirement/20260630-cd-deploy-force-recreate.md`
  - 新增轻量模式 requirement，并在进入验证阶段时将 `Review status` 更新为 `Accepted`。
- `web/src/views/cd/ApplicationDetail.vue`
  - 将原部署按钮改为主按钮 + 下拉菜单的复合部署控件。
  - 主按钮调用普通部署；下拉菜单项调用强制重建部署。
  - 为下拉触发按钮增加 `deployOptions` aria-label，并复用禁用态。
- `web/src/api/cd/application.ts`
  - `deploy` API 增加可选 `{ force_recreate?: boolean }` 参数，默认仍发送空对象。
- `web/src/i18n/locales/en-US.ts`, `web/src/i18n/locales/zh-CN.ts`
  - 新增部署选项和强制重建部署文案。
- `internal/transport/http/handler/cd/handler.go`
  - 现有 deploy endpoint 增加可选 JSON body 解析，非法 JSON 返回 400，空 body 保持普通部署。
- `internal/service/cd/service.go`
  - 新增 `ApplicationDeployInput`，创建 deployment 后将 `force_recreate` 写入后台任务 payload。
- `internal/worker/handler/cd/handler.go`
  - 任务 payload 增加 `ForceRecreate`，deploy operation 透传到部署执行接口。
- `internal/service/cd/deployment_execution.go`
  - `ExecuteApplicationDeploy` 和内部 `writeAndDeploy` 增加 `forceRecreate` 参数。
  - 仅在 `forceRecreate` 为 true 时追加 `--force-recreate`。
- `internal/service/cd/deployment_execution_test.go`
  - 覆盖普通部署命令不包含 `--force-recreate`。
  - 新增强制重建部署命令包含 `--force-recreate` 的测试。
- `internal/worker/handler/cd/handler_test.go`
  - 覆盖旧 payload 默认 false 和新 payload true 的透传行为。
- `internal/transport/http/application_routes_test.go`
  - 覆盖 HTTP deploy 请求携带 `force_recreate: true` 后，后台任务 payload 保留该布尔值。

## Expected vs actual changed files / 预期与实际改动对比

| 预期范围 | 实际文件 | 结论 |
| --- | --- | --- |
| 需求文档 | `docs/requirement/20260630-cd-deploy-force-recreate.md` | 符合 |
| 应用详情页部署复合控件 | `web/src/views/cd/ApplicationDetail.vue` | 符合 |
| 前端 deploy API 可选参数 | `web/src/api/cd/application.ts` | 符合 |
| 前端中英文文案 | `web/src/i18n/locales/en-US.ts`, `web/src/i18n/locales/zh-CN.ts` | 符合 |
| 现有 deploy endpoint 可选 body | `internal/transport/http/handler/cd/handler.go` | 符合 |
| service 创建任务 payload | `internal/service/cd/service.go` | 符合 |
| worker payload 透传 | `internal/worker/handler/cd/handler.go` | 符合 |
| 部署执行层追加 compose 参数 | `internal/service/cd/deployment_execution.go` | 符合 |
| 后端测试覆盖 | `internal/service/cd/deployment_execution_test.go`, `internal/worker/handler/cd/handler_test.go`, `internal/transport/http/application_routes_test.go` | 符合 |
| 应用列表页部署入口 | 无修改 | 符合 Non-goal |
| 数据库迁移 / deployment schema | 无修改 | 符合 Non-goal |
| 停止、重启、删除应用行为 | 无业务逻辑修改 | 符合 Non-goal |

## Acceptance criteria checklist / 验收清单

- [x] 应用详情页部署控件包含默认部署主按钮和强制重建部署菜单项。
- [x] 默认部署执行层命令不包含 `--force-recreate`。
- [x] 强制重建部署请求携带 `force_recreate: true`，后台任务 payload 保留该布尔值。
- [x] Worker 能把 `force_recreate` 从任务 payload 透传到部署执行层。
- [x] 部署执行层仅在 `force_recreate == true` 时追加 `--force-recreate`。
- [x] 缺省或旧 payload 中没有 `force_recreate` 字段时按 `false` 处理。
- [x] 前端 lint/typecheck 按项目约束通过。
- [x] Go fmt/vet/test 按项目约束通过。

## Test results / 命令结果

已按项目约束运行以下检查：

```text
yarn --cwd web lint:fix
结果：通过
输出摘要：eslint . --fix --cache，Done in 0.98s.
```

```text
yarn --cwd web typecheck
结果：通过
输出摘要：vue-tsc --noEmit，Done in 4.63s.
```

```text
go fmt ./cmd/... ./internal/...
结果：通过
输出摘要：无输出
```

```text
go vet ./cmd/... ./internal/...
结果：通过
输出摘要：无输出
```

```text
go test ./cmd/... ./internal/...
结果：通过
输出摘要：cmd 和 internal 下所有 package 测试通过；无测试文件的 package 正常跳过。
```

## Missed or expanded scope / 范围偏差

- 未发现超出 requirement 的产品功能变更。
- 本次验证未启动开发服务器，也未做浏览器手工点击验证，符合项目约束中“不主动启动、停止或重启开发服务器”。
- 前端主部署按钮当前会调用 `handleDeploy()` 并发送 `force_recreate: false`；执行语义符合“普通部署不启用强制重建”，但如果后续希望严格做到默认请求 body 不出现该字段，可再将默认调用调整为不传 data。

## Risks / 风险

- 强制重建部署会触发 Docker Compose 重新创建容器，可能导致短暂服务中断；当前 UI 已将其放入显式下拉菜单项，仍需用户操作时理解语义。
- 当前验证主要依赖单元/路由测试、类型检查和静态 diff 核对；未在真实 Docker 环境中执行一次实际 `docker compose up --force-recreate`。
- 当前 git index 中已有暂存变更，验证阶段对 requirement 状态和新增 verification 文档的变更未自动执行 `git add`；提交前需要用户自行确认暂存区与工作区边界。

## Incomplete items / 未完成项

- 无代码层未完成项。
- 如需更高置信度，可由用户启动本地环境后，在应用详情页手工点击主部署与强制重建菜单项各一次，并观察请求 payload 与部署日志。

## Conclusion / 结论

本次实现与轻量模式 requirement 对齐。默认部署路径、强制重建参数传递链路、worker 透传和部署执行层命令拼接均已有测试或静态核对覆盖；前端 lint/typecheck 与 Go fmt/vet/test 均通过。当前可以进入人工验收和提交前 review。