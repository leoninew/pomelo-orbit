# CD 部署详情展示执行命令与容器日志验证
最后修改时间: 2026-06-30 13:03:31

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260630-cd-deployment-detail-container-logs.md` 核对，本次实现覆盖需求中的核心目标：

1. 部署详情页“基本信息”已新增“执行命令”，直接展示后端返回的 `command_text`，历史记录为空时显示“未记录”。
2. 执行命令作为完整字符串持久化到 `deployment.command_text`，不依赖前端临时拼接，也不依赖 `/cd/deployments/{id}?from=application`。
3. 普通部署、强制重建部署、重启、停止和删除 volumes 停止都通过同源的 compose command 构造逻辑生成展示命令和实际执行参数。
4. 部署和重启类详情页日志区域已调整为“容器日志”，不再把 deployment 操作日志作为默认主日志内容。
5. 容器日志优先通过 deployment `started_at` 执行 `docker compose logs --since <started_at>` 获取本次操作开始后的日志；失败时回退 `--tail N` 并返回 `source: tail`。
6. 页面在 `source === 'tail'` 时展示“可能包含本次操作前的历史输出”的提示。
7. 停止类操作不展示实时容器日志，页面显示“不展示实时容器日志”的说明，后端 service 也防御性拒绝 stop deployment 的 container logs 请求。
8. 容器日志区域保留自动刷新 / 暂停刷新切换，移除了原独立“刷新”按钮；错误态重试直接拉取一次容器日志。
9. 自动刷新初始行为符合用户补充要求：运行中的 deploy / restart 详情默认开启自动刷新；已完成、失败、取消等终态详情只初始拉取一次，不默认轮询；用户后续手动点击自动刷新时允许持续轮询，不因终态状态阻止。
10. 现有返回应用、返回部署列表、取消运行中任务能力保留。

需求中的 Non-goal 保持一致：未重做整体部署详情页信息架构或视觉风格，未主动启动、停止或重启开发服务器，未改变应用列表页、应用详情页操作入口设计，未改变 Docker Compose 实际执行行为，未实现完整历史日志归档系统。

## Spec alignment / 规格对齐

依据 `docs/spec/20260630-cd-deployment-detail-container-logs.md` 核对：

1. 数据模型按规格新增 `deployment.command_text` 字段，并通过 Deployment detail response 返回。
2. 后端抽取 `composeCommand`，使用同一结构的 `String()` 写入 `command_text`，并使用同一 `Name` / `Args` 执行 runner，满足“命令生成与实际执行参数同源”。
3. 新增 deployment 维度接口：`GET /api/cd/deployment/{deployment_id}/container-logs?tail=200`。
4. Container logs response 返回 `logs`、`source`、`is_realtime_supported`。
5. `deploy` / `restart` 使用 `docker compose -f docker-compose.yml logs --since <deployment.started_at RFC3339>`；失败后回退 `docker compose -f docker-compose.yml logs --tail <N>`。
6. `stop` 操作由前端避免调用实时日志接口，后端收到请求时返回 validation error。
7. 前端采用近实时轮询而非 SSE；每次用接口返回的完整日志替换编辑器内容，避免 offset 追加错位。
8. 页面卸载时停止轮询；用户点击按钮可暂停或恢复轮询；终态详情默认不自动刷新但允许用户手动开启。

已识别一个规格文案与最终用户补充之间的变化：规格原先曾写“如果刷新后发现 deployment 已进入终态，则停止轮询”，后续用户明确要求“不自动刷新是初始行为，后面用户要修改为自行刷新不要阻止”，文档已更新为“终态不阻止用户手动持续刷新”，实现按更新后的规格执行。

## Plan alignment / 计划对齐

依据 `docs/plan/20260630-cd-deployment-detail-container-logs.md` 核对：

1. 数据库迁移：已新增 SQLite / MySQL migration，为 `deployment` 表添加 `command_text` 字段，未修改已有迁移文件。
2. 后端模型与 repository：`model.Deployment`、`CreateDeployment`、`ListDeployments`、`Deployment` 查询和 HTTP response 均覆盖 `command_text`。
3. 命令构造同源化：已抽取 `internal/service/cd/compose_command.go`，部署、重启、停止创建记录和执行层复用同一命令构造。
4. 容器日志服务与接口：已新增 deployment container logs service 和 HTTP route，包含 `--since` 优先、`--tail` 回退、stop 防御性拒绝。
5. 前端 API / type / 页面：已增加 `Deployment.command_text` 类型、`deploymentApi.getContainerLogs`、详情页执行命令展示、容器日志展示、tail 提示、自动刷新 / 暂停刷新、终态初始不自动刷新、移除独立刷新按钮。
6. 测试与验证：已更新 migration 数量、路由、命令文本和执行命令相关测试，并按项目约束运行检查。

## Actual diff summary / 实际差异摘要

验证前 `git diff HEAD --stat --` 显示的变更范围：

```text
 ...20260630-cd-deployment-detail-container-logs.md | 109 ++++++++
 ...20260630-cd-deployment-detail-container-logs.md |  82 ++++++
 ...20260630-cd-deployment-detail-container-logs.md | 290 +++++++++++++++++++++
 internal/app/mysql_e2e_test.go                     |   4 +-
 internal/db/migrator_test.go                       |   8 +-
 .../mysql/v0.1.6__deployment_command_text.sql      |   1 +
 .../sqlite/v0.1.6__deployment_command_text.sql     |   1 +
 internal/repository/cd/repository.go               |   8 +-
 internal/repository/model/cd.go                    |   1 +
 internal/service/cd/application_extra.go           |  40 ++-
 internal/service/cd/compose_command.go             |  47 ++++
 internal/service/cd/deployment_execution.go        |  17 +-
 internal/service/cd/service.go                     |   7 +
 internal/transport/http/application_routes_test.go |  22 +-
 internal/transport/http/deployment_routes_test.go  |   5 +-
 internal/transport/http/handler/cd/handler.go      |  17 +-
 web/src/api/cd/deployments.ts                      |  12 +
 web/src/i18n/locales/zh-CN.ts                      |   4 +-
 web/src/types/cd/deployment.ts                     |   1 +
 web/src/views/cd/DeploymentDetail.vue              | 185 ++++++-------
 20 files changed, 715 insertions(+), 146 deletions(-)
```

实际功能改动包括：

- `docs/requirement/20260630-cd-deployment-detail-container-logs.md`
  - 新增并维护标准模式 requirement，记录执行命令、容器日志、停止操作日志策略、自动刷新初始行为和终态手动刷新行为。
- `docs/spec/20260630-cd-deployment-detail-container-logs.md`
  - 新增并维护 spec，记录数据库字段、命令同源化、container logs API、轮询策略和备选方案。
- `docs/plan/20260630-cd-deployment-detail-container-logs.md`
  - 新增并维护 plan，记录迁移、后端、前端、测试和验证步骤。
- `internal/migrations/sqlite/v0.1.6__deployment_command_text.sql`
  - SQLite `deployment` 表新增 `command_text TEXT NOT NULL DEFAULT ''`。
- `internal/migrations/mysql/v0.1.6__deployment_command_text.sql`
  - MySQL `deployment` 表新增 `command_text VARCHAR(2048) NOT NULL DEFAULT ''`。
- `internal/repository/model/cd.go`, `internal/repository/cd/repository.go`
  - Deployment model 和 repository create/select 覆盖 `command_text`。
- `internal/service/cd/compose_command.go`
  - 新增 compose command 构造对象和 deploy / restart / stop / container logs 命令构造函数。
- `internal/service/cd/service.go`, `internal/service/cd/application_extra.go`, `internal/service/cd/deployment_execution.go`
  - 创建 deployment 时写入命令文本；执行层复用同源命令；新增 deployment container logs service。
- `internal/transport/http/handler/cd/handler.go`
  - Deployment response 增加 `command_text`；新增 container logs route 和 handler。
- `web/src/types/cd/deployment.ts`, `web/src/api/cd/deployments.ts`
  - 前端类型和 API 增加 `command_text` / `getContainerLogs`。
- `web/src/views/cd/DeploymentDetail.vue`
  - 展示执行命令；日志卡片改为容器日志；实现容器日志拉取、tail 提示、stop 不适用说明、自动刷新 / 暂停刷新、终态初始不自动刷新、用户手动开启不受终态阻止、错误态重试。
- `internal/db/migrator_test.go`, `internal/app/mysql_e2e_test.go`, `internal/transport/http/deployment_routes_test.go`, `internal/transport/http/application_routes_test.go`
  - 更新迁移数量和 HTTP / command_text 相关断言。
- `web/src/i18n/locales/zh-CN.ts`
  - 调整 CI 运行详情自动刷新 / 暂停刷新中文文案，与本次容器日志按钮参考文案一致。

## Expected vs actual changed files / 预期与实际改动对比

| 预期范围 | 实际文件 | 结论 |
| --- | --- | --- |
| Requirement / Spec / Plan 过程文档 | `docs/requirement/...`, `docs/spec/...`, `docs/plan/...` | 符合 |
| SQLite / MySQL migration | `internal/migrations/sqlite/v0.1.6__deployment_command_text.sql`, `internal/migrations/mysql/v0.1.6__deployment_command_text.sql` | 符合 |
| Migration 数量测试 | `internal/db/migrator_test.go`, `internal/app/mysql_e2e_test.go` | 符合 |
| Deployment model / repository | `internal/repository/model/cd.go`, `internal/repository/cd/repository.go` | 符合 |
| CD service / execution 命令同源化 | `internal/service/cd/service.go`, `internal/service/cd/application_extra.go`, `internal/service/cd/deployment_execution.go`, `internal/service/cd/compose_command.go` | 符合 |
| HTTP Deployment response / container logs route | `internal/transport/http/handler/cd/handler.go` | 符合 |
| 后端 HTTP 测试 | `internal/transport/http/deployment_routes_test.go`, `internal/transport/http/application_routes_test.go` | 符合 |
| 前端 Deployment type / API | `web/src/types/cd/deployment.ts`, `web/src/api/cd/deployments.ts` | 符合 |
| 部署详情页面 | `web/src/views/cd/DeploymentDetail.vue` | 符合 |
| CI 运行详情中文文案 | `web/src/i18n/locales/zh-CN.ts` | 轻微扩展范围：用于统一自动刷新 / 暂停刷新文案；如不希望影响 CI 页面，可单独回退该文件中的两行文案 |
| 应用列表页 / 应用详情页操作入口 | 无本需求相关修改 | 符合 Non-goal |
| 开发服务器启停 | 未执行 | 符合项目约束 |

## Acceptance criteria checklist / 验收清单

- [x] 部署详情页“基本信息”区域新增“执行命令”字段。
- [x] 普通部署的执行命令不包含 `--force-recreate`。
- [x] 强制重建部署的执行命令包含 `--force-recreate`。
- [x] 重启操作的执行命令展示为 `docker compose -f docker-compose.yml restart`。
- [x] 停止操作的执行命令展示为 `docker compose -f docker-compose.yml down`；删除 volumes 时包含 `-v`。
- [x] 执行命令作为完整命令字符串持久化，并由部署详情 API 返回；页面不临时拼接命令，不依赖 URL query。
- [x] 部署和重启类详情页日志区域展示容器日志，而不是把操作日志作为主内容。
- [x] 容器日志优先展示“本次操作开始后”的日志，基于 deployment `started_at` 调用 `docker compose logs --since <started_at>`。
- [x] 无法按时间范围获取时回退最近 N 行日志，并在页面提示可能包含历史输出。
- [x] 停止类操作不展示实时容器日志，不显示误导性的“部署日志”标题，并明确说明“不展示实时容器日志”。
- [x] 运行中的部署或重启详情页默认自动刷新，并提供“自动刷新 / 暂停刷新”切换按钮。
- [x] 已完成或失败等终态详情页初始只拉取一次日志，不默认轮询；用户后续点击自动刷新时允许手动启动并持续刷新。
- [x] 容器日志区域移除原独立“刷新”按钮；错误态重试直接重新拉取容器日志。
- [x] 现有返回应用、返回部署列表、取消运行中任务能力保持可用。
- [x] 前端 lint/typecheck 和 Go fmt/vet/test 按项目约束通过。

## Test results / 命令结果

已按项目约束运行以下检查：

```text
yarn --cwd web lint:fix
结果：通过
输出摘要：eslint . --fix --cache，Done in 17.50s.
```

```text
yarn --cwd web typecheck
结果：通过
输出摘要：vue-tsc --noEmit，Done in 10.43s.
```

```text
go fmt ./cmd/... ./internal/...
结果：通过
输出摘要：无输出。
```

```text
go vet ./cmd/... ./internal/...
结果：通过
输出摘要：无输出。
```

```text
go test ./cmd/... ./internal/...
结果：通过
输出摘要：cmd/internal packages 均通过或无测试文件。
```

补充检查：

```text
git diff --check HEAD --
结果：通过
输出摘要：无输出。
```

## Missed or expanded scope / 范围偏差

- 轻微扩展范围：`web/src/i18n/locales/zh-CN.ts` 中 CI run 自动刷新中文文案从“自动刷新中 / 已暂停刷新”调整为“自动刷新 / 暂停刷新”。这与本需求“参考 `/ci/run/{id}` 的自动刷新 / 暂停刷新能力”有关，但会影响流水线详情页既有中文文案。当前判断为低风险；如产品上不希望改动 CI 文案，可单独回退该文件中的两行变更。
- 未做真实 Docker Compose 环境手工验证；本次验证覆盖代码路径、类型检查、Go 测试和接口行为测试。`docker compose logs --since <RFC3339>` 在目标运行环境中的实际表现仍依赖部署环境，代码已提供 `--tail` 回退。
- 未主动启动、停止或重启开发服务器，符合项目约束。

## Risks / 风险

- 容器日志接口每次返回完整日志文本，长时间运行或日志量较大时可能增加后端命令执行、网络传输和 Monaco 渲染压力；当前通过 2 秒轮询间隔、终态初始不自动刷新和 `tail=200` 回退降低风险，但 `--since` 成功时仍可能返回较大输出。
- `docker compose logs --since` 可能返回多服务混合输出，符合当前应用级 compose 视角，但页面文案应避免让用户误解为单容器日志。
- MySQL 使用 `VARCHAR(2048) NOT NULL DEFAULT ''` 存储命令文本，当前 Docker Compose 命令长度足够；若未来命令极长，需要重新评估字段长度。
- 真实容器日志读取依赖工作目录、Compose 文件和 Docker 环境可用性；测试未覆盖真实 Docker daemon。

## Incomplete items / 未完成项

- 无必须阻塞交付的未完成项。
- 可选后续优化：为容器日志接口增加更明确的输出量控制或分页 / offset 机制，避免 `--since` 在长时间运行场景下返回过大日志。
- 可选后续优化：如用户需要同时排查 Orbit 执行过程，可另开需求增加“执行日志”折叠面板或 debug 入口。

## Conclusion / 结论

本次实现与已接受的 Requirement、Spec 和 Plan 对齐；验证命令均通过。除已记录的轻微文案扩展范围和真实 Docker Compose 环境未手工验证外，没有发现阻塞交付的问题。当前可以进入用户验收或提交准备阶段。
