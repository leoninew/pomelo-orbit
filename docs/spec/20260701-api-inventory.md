# API 盘点规格
最后修改时间: 2026-07-01 15:38:08

## Review status

Accepted

## Requirement basis

基于 `docs/requirement/20260701-api-inventory.md`：

- 范围限定为当前代码库中的后端 HTTP API。
- 每个接口需要记录请求方法、路径、请求体、响应体。
- 请求体和响应体只记录名称、定义来源和形态，不展开字段明细。
- 需要标记未遵从 `Req` / `Resp` 形式的接口。
- 需要标记直接数组、字典、匿名结构、service/repository/model 类型等响应形态。
- 需要补充前端调用侧说明：调用位置、返回类型/请求类型命名形态，以及明显的前后端契约不一致。
- 本次 spec 阶段需要列出后续实现可用的待修改接口 checklist。

## Overview

本任务后续实现不直接改变业务行为，目标是形成 API 契约盘点结果，并为后续规范化接口请求/响应命名提供 checklist。

实现时建议产出一个独立盘点文档，记录：

1. 后端路由清单：method、path、handler、请求体名称/来源/形态、响应体名称/来源/形态。
2. 待规范化接口 checklist：按问题类型分组，作为后续代码修改清单。
3. 前端相关说明：前端调用路径、请求体类型、响应体类型、明显不一致项。
4. 特殊接口说明：SSE、multipart、webhook raw body、void response、错误响应。

## Design decisions

- 不在本阶段修改产品代码，只建立接口契约盘点和后续修改 checklist。
- 成功响应体的规范化目标以命名 `*Resp` 为主；请求体的规范化目标以命名 `*Req` 为主。
- 有响应内容的 action 接口必须返回命名 `XxxxResp`；无响应内容的接口使用 `204 No Content`，并在盘点中明确标注 `void`。
- 分页响应继续使用 `transportresponse.PaginatedResp[T]` 作为通用分页 response；如果分页 item 类型不是 `*Resp`，在 checklist 中作为次级规范化项说明。
- 直接数组响应 `[]T` 统一改为对象包装 `{ items: []T }`，并使用命名 response，例如 `ProjectListResp`、`PermissionListResp`。
- Handler 层 response DTO 应与 service/repository/model 类型隔离：HTTP API 暴露的是稳定契约，不应因为内部存储模型或 service view 调整而被动改变外部响应。该抽取 DTO 的应抽取出来；除非某类型本身就是当前 handler/API 的契约逻辑，不要让 DTO 与业务逻辑混在一起。已有直接暴露内部类型的接口需要在 checklist 中标记；用户已要求无须 Plan，Implementation 阶段按 checklist 分批处理。
- Webhook raw body、multipart upload、SSE 不强行套普通 JSON `Req` / `Resp`，但必须在文档中显式定义形态；SSE event payload 也需要命名。
- 错误响应纳入本次规范化：当前大量 `map[string]string{"detail": ...}` 需要收敛为通用错误响应泛型，例如 `ErrorResp[T]` 或等价结构；具体类型名和字段在 Plan/Implementation 阶段落定。
- 路由路径保持现状，本次不顺手调整路径单复数或兼容别名。

## Affected components

### 后端

- `internal/transport/http/server.go`
- `internal/transport/http/handler/auth/handler.go`
- `internal/transport/http/handler/user/handler.go`
- `internal/transport/http/handler/role/handler.go`
- `internal/transport/http/handler/settings/handler.go`
- `internal/transport/http/handler/project/handler.go`
- `internal/transport/http/handler/task/handler.go`
- `internal/transport/http/handler/ci/repository.go`
- `internal/transport/http/handler/ci/template.go`
- `internal/transport/http/handler/ci/build_stage.go`
- `internal/transport/http/handler/ci/pipeline_run.go`
- `internal/transport/http/handler/ci/artifact.go`
- `internal/transport/http/handler/ci/snapshot.go`
- `internal/transport/http/handler/ci/credential.go`
- `internal/transport/http/handler/cd/handler.go`
- `internal/transport/http/handler/cd/application_extra.go`
- `internal/transport/http/handler/cd/route.go`
- `internal/transport/http/response/response.go`

### 前端

- `web/src/api/auth.ts`
- `web/src/api/project.ts`
- `web/src/api/user.ts`
- `web/src/api/role.ts`
- `web/src/api/settings.ts`
- `web/src/api/ci/artifact.ts`
- `web/src/api/ci/build_stage.ts`
- `web/src/api/ci/repository.ts`
- `web/src/api/ci/webhook.ts`
- `web/src/api/ci/template.ts`
- `web/src/api/ci/run.ts`
- `web/src/api/ci/credential.ts`
- `web/src/api/cd/application.ts`
- `web/src/api/cd/deployments.ts`
- `web/src/api/cd/route.ts`
- `web/src/api/cd/traefik-route.ts`
- `web/src/utils/request.ts`
- `web/src/types/**/*.ts`

## Interfaces and checklist

### 1. 待修改接口 checklist：匿名 map / 字典成功响应

这些接口成功响应直接返回 `map[string]string`、`map[string]any` 或动态字典，缺少命名 `Resp`。

| Checklist | Method | Path | 后端位置 | 当前形态 | 建议方向 |
| --- | --- | --- | --- | --- | --- |
| [ ] | GET | `/api/health` | `internal/transport/http/server.go` | `map[string]string` | 定义 `HealthResp` |
| [ ] | GET | `/api/auth/csrf-token` | `internal/transport/http/handler/auth/handler.go` | `map[string]string` | 定义/统一 `CsrfTokenResp` 或 `CSRFTokenResp` |
| [ ] | GET | `/api/auth/google` | `internal/transport/http/handler/auth/handler.go` | 当前固定 503，`map[string]string` | 明确 placeholder contract 或移出正式 API 清单 |
| [ ] | POST | `/api/auth/google/callback` | `internal/transport/http/handler/auth/handler.go` | `googleCallbackReq` + 503 `map[string]string` | 明确未启用响应，或实现后返回 `TokenResp` |
| [ ] | POST | `/api/ci/webhook/{webhook_id}` | `internal/transport/http/handler/ci/repository.go` | raw body + `map[string]string` | 定义 `RepositoryWebhookReceiveResp`；请求标注 raw payload 特例 |
| [ ] | POST | `/api/cd/application/{app_id}/deploy` | `internal/transport/http/handler/cd/handler.go` | `map[string]string{"deployment_id": ...}` | 定义 `ApplicationDeployResp` 或通用 `DeploymentActionResp` |
| [ ] | GET | `/api/cd/deployment/{deployment_id}/logs` | `internal/transport/http/handler/cd/handler.go` | `map[string]any` | 定义 `DeploymentLogsResp` |
| [ ] | GET | `/api/cd/deployment/{deployment_id}/container-logs` | `internal/transport/http/handler/cd/handler.go` | `map[string]any` | 定义 `DeploymentContainerLogsResp` |
| [ ] | POST | `/api/cd/application/{app_id}/compose-preview` | `internal/transport/http/handler/cd/application_extra.go` | `map[string]string{"compose_yaml": ...}` | 定义 `ApplicationComposePreviewResp` |
| [ ] | GET | `/api/cd/application/{app_id}/file/{file_id}` | `internal/transport/http/handler/cd/application_extra.go` | `map[string]string{"content", "path"}` | 定义 `ApplicationFileContentResp` |
| [ ] | POST | `/api/cd/application/{app_id}/stop` | `internal/transport/http/handler/cd/application_extra.go` | 匿名请求 + `map[string]string{"deployment_id": ...}` | 定义 `ApplicationStopReq` 和 `DeploymentActionResp` |
| [ ] | POST | `/api/cd/application/{app_id}/restart` | `internal/transport/http/handler/cd/application_extra.go` | `map[string]string{"deployment_id": ...}` | 复用 `DeploymentActionResp` |
| [ ] | GET | `/api/cd/application/{app_id}/status` | `internal/transport/http/handler/cd/application_extra.go` | `map[string]string{"status": ...}` | 定义 `ApplicationStatusResp` |
| [ ] | GET | `/api/cd/application/{app_id}/logs` | `internal/transport/http/handler/cd/application_extra.go` | `map[string]string{"logs": ...}` | 定义 `ApplicationLogsResp` |
| [ ] | POST | `/api/cd/route/sync` | `internal/transport/http/handler/cd/route.go` | `map[string]string{"message": ...}` | 定义 `RouteSyncResp` 或通用 `MessageResp` |
| [ ] | POST | `/api/cd/route/{route_id}/enable` | `internal/transport/http/handler/cd/route.go` | `map[string]string{"message": ...}` | 定义 `MessageResp`、返回 `RouteResp` 或改为 void，需决策 |
| [ ] | POST | `/api/cd/route/{route_id}/disable` | `internal/transport/http/handler/cd/route.go` | `map[string]string{"message": ...}` | 同上 |

### 2. 待修改接口 checklist：直接数组响应

这些接口直接返回 `[]T`，缺少外层命名 `Resp`。后续统一改为 `{ items: []T }`，并使用命名 response。

| Checklist | Method | Path | 后端位置 | 当前形态 | 建议方向 |
| --- | --- | --- | --- | --- | --- |
| [ ] | GET | `/api/role/permission` | `internal/transport/http/handler/role/handler.go` | `[]PermissionResp` | 定义 `PermissionListResp { items: []PermissionResp }` |
| [ ] | GET | `/api/project` | `internal/transport/http/handler/project/handler.go` | `[]ProjectResp` | 定义 `ProjectListResp { items: []ProjectResp }` |
| [ ] | GET | `/api/project/{project_id}/member` | `internal/transport/http/handler/project/handler.go` | `[]ProjectMemberResp` | 定义 `ProjectMemberListResp { items: []ProjectMemberResp }` |
| [ ] | POST | `/api/project/{project_id}/member` | `internal/transport/http/handler/project/handler.go` | `[]ProjectMemberResp` | 同上 |
| [ ] | DELETE | `/api/project/{project_id}/member/{user_id}` | `internal/transport/http/handler/project/handler.go` | `[]ProjectMemberResp` | 同上 |
| [ ] | GET | `/api/ci/repository/{repository_id}/webhook` | `internal/transport/http/handler/ci/repository.go` | `[]RepositoryWebhookResp` | 定义 `RepositoryWebhookListResp { items: []RepositoryWebhookResp }` |
| [ ] | GET | `/api/ci/run/{run_id}/artifacts` | `internal/transport/http/handler/ci/pipeline_run.go` | `[]ArtifactResp` | 定义 `PipelineRunArtifactListResp { items: []ArtifactResp }` |
| [ ] | POST | `/api/ci/template/resolve-variables` | `internal/transport/http/handler/ci/template.go` | `[]map[string]any` | 定义 `TemplateVariableResolveResp { items: []VariableDeclarationResp }` |
| [ ] | GET | `/api/cd/application/{app_id}/files` | `internal/transport/http/handler/cd/application_extra.go` | `[]ConfigFileResp` | 定义 `ConfigFileListResp { items: []ConfigFileResp }` |
| [ ] | GET | `/api/cd/application/{app_id}/route` | `internal/transport/http/handler/cd/application_extra.go` | `[]ApplicationRouteResp` | 定义 `ApplicationRouteListResp { items: []ApplicationRouteResp }` |
| [ ] | GET | `/api/cd/application/{app_id}/compose-service` | `internal/transport/http/handler/cd/application_extra.go` | `[]ComposeServiceResp` | 定义 `ComposeServiceListResp { items: []ComposeServiceResp }` |

### 3. 待修改接口 checklist：直接暴露 service / repository / model 类型

这些接口顶层或关键字段直接绑定 service/repository/model 类型，不利于 HTTP API 契约与内部模型解耦。

| Checklist | Method | Path / Type | 后端位置 | 当前形态 | 建议方向 |
| --- | --- | --- | --- | --- | --- |
| [ ] | GET | `/api/settings/config` | `internal/transport/http/handler/settings/handler.go` | `settingssvc.SystemConfig` / alias | 定义 handler 层 `SystemConfigResp` struct |
| [ ] | PUT | `/api/settings/config` | `internal/transport/http/handler/settings/handler.go` | `settingssvc.SystemConfig` | 同上 |
| [ ] | DELETE | `/api/settings/config` | `internal/transport/http/handler/settings/handler.go` | `settingssvc.SystemConfig` | 同上 |
| [ ] | POST | `/api/background/task` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | POST | `/api/background/ci/pipeline-run/{run_id}/execute` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | POST | `/api/background/cd/application/{app_id}/deploy/{deployment_id}` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | POST | `/api/background/cd/application/{app_id}/restart/{deployment_id}` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | POST | `/api/background/cd/application/{app_id}/stop/{deployment_id}` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | GET | `/api/background/task/{id}` | `internal/transport/http/handler/task/handler.go` | `*taskrepo.Task` | 定义 `TaskResp` |
| [ ] | GET | `/api/cd/application/{app_id}/service-config` | `internal/transport/http/handler/cd/application_extra.go` | `[]cdsvc.ApplicationServiceConfigView` | 定义 handler 层 list response 和 item response |
| [ ] | PUT | `/api/cd/application/{app_id}/service-config/{service_name}` | `internal/transport/http/handler/cd/application_extra.go` | `cdsvc.ApplicationServiceConfigView` | 定义 `ApplicationServiceConfigResp` |
| [ ] | GET | `/api/cd/traefik-route/config` | `internal/transport/http/handler/cd/route.go` | `cdsvc.TraefikConfigResp` | 移到 handler 层或建立转换 DTO |
| [ ] | GET | `/api/cd/traefik-route` | `internal/transport/http/handler/cd/route.go` | `cdsvc.TraefikRouteListResp` | 移到 handler 层或建立转换 DTO |
| [ ] | GET | `/api/ci/snapshot/{snapshot_id}` | `internal/transport/http/handler/ci/snapshot.go` | `PipelineSnapshotResp` 内含 repository model 类型 | 定义 HTTP 层嵌套 DTO |
| [ ] | 多个 | `PipelineRunResp` | `internal/transport/http/handler/ci/pipeline_run.go` | 字段含 `model.VariableDeclaration` | 定义 HTTP 层变量声明 DTO |

### 4. 待修改接口 checklist：请求体不是命名 Req

这些接口请求体不是普通命名 `Req`，或者属于需要特殊标注的 request contract。

| Checklist | Method | Path | 后端位置 | 当前形态 | 建议方向 |
| --- | --- | --- | --- | --- | --- |
| [ ] | PUT | `/api/ci/template/{template_id}` | `internal/transport/http/handler/ci/template.go` | `map[string]json.RawMessage` | 定义 `PipelineTemplateUpdateReq` 或明确 JSON patch/merge patch 合约 |
| [ ] | PUT | `/api/ci/build-stage/{stage_id}` | `internal/transport/http/handler/ci/build_stage.go` | `map[string]json.RawMessage` | 定义 `BuildStageUpdateReq` 或明确 patch 合约 |
| [ ] | POST | `/api/ci/webhook/{webhook_id}` | `internal/transport/http/handler/ci/repository.go` | raw body `[]byte` | 标注 webhook passthrough 特例；必要时定义 raw request 说明 |
| [ ] | POST | `/api/cd/application/{app_id}/stop` | `internal/transport/http/handler/cd/application_extra.go` | anonymous struct | 定义 `ApplicationStopReq` |
| [ ] | POST | `/api/cd/route/{route_id}/cert` | `internal/transport/http/handler/cd/route.go` | multipart file `pem` | 标注 multipart request，不按 JSON Req 处理 |

### 5. 待修改接口 checklist：内部弱类型 map / any

这些接口已有命名 Req/Resp，但内部仍使用 `map[string]any` 或 `[]map[string]any`。

| Checklist | Type / Route | 后端位置 | 当前形态 | 建议方向 |
| --- | --- | --- | --- | --- |
| [ ] | `RepositoryResp.VariableDeclarations` | `internal/transport/http/handler/ci/repository.go` | `[]map[string]any` | 定义变量声明 DTO |
| [ ] | `RepositoryCreateReq.VariableOverrides` | `internal/transport/http/handler/ci/repository.go` | `[]map[string]any` | 定义变量覆盖 DTO |
| [ ] | `RepositoryUpdateReq.VariableOverrides` | `internal/transport/http/handler/ci/repository.go` | `*[]map[string]any` | 同上 |
| [ ] | `PipelineTemplateResp.VariableDeclarations` | `internal/transport/http/handler/ci/template.go` | `[]map[string]any` | 定义变量声明 DTO |
| [ ] | `PipelineTemplateCreateReq.VariableDeclarations` | `internal/transport/http/handler/ci/template.go` | `[]map[string]any` | 同上 |
| [ ] | `TemplateVariableResolveReq.VariableDeclarations` | `internal/transport/http/handler/ci/template.go` | `[]map[string]any` | 同上 |
| [ ] | POST `/api/ci/template/resolve-variables` response | `internal/transport/http/handler/ci/template.go` | `[]map[string]any` | 定义 `TemplateVariableResolveResp` |

### 6. 特殊接口 checklist

| Checklist | Method | Path | 当前形态 | Spec 处理 |
| --- | --- | --- | --- | --- |
| [ ] | GET | `/api/cd/deployment/{deployment_id}/stream-log` | SSE `text/event-stream`，event data 为匿名 JSON | 不改为普通 JSON response；定义事件 payload 名称，例如 `DeploymentLogEventResp` / `DeploymentLogCompleteEventResp` |
| [ ] | POST | `/api/cd/route/{route_id}/cert` | multipart file `pem` | 定义 multipart request 说明，响应继续 `RouteResp` |
| [ ] | POST | `/api/ci/webhook/{webhook_id}` | 外部 webhook raw payload | 标注为 raw passthrough webhook，不强制 JSON Req |
| [ ] | 多个 | DELETE / action void 接口 | 204 / no body | 文档中显式标注 `void`，不列为必须修改项 |

### 7. 待修改接口 checklist：错误响应泛型化

当前错误响应多处直接返回 `map[string]string{"detail": ...}` 或类似匿名结构。后续应收敛到 response 包中的通用错误响应泛型，具体字段在 Plan/Implementation 阶段确定。

建议原则：

- `ErrorResp[T]` 或等价泛型类型用于表达错误响应，其中 `T` 表示错误详情载荷类型。
- 常规文本错误可使用 `ErrorResp[string]` 或 `ErrorResp[ErrorDetailResp]`，二选一在 Plan 阶段定稿。
- 字段校验错误可使用结构化详情，例如 `ErrorResp[ValidationErrorResp]` 或 `ErrorResp[[]FieldErrorResp]`。
- 不再在 handler 中散落匿名 `map[string]string` 作为错误响应。
- `transportresponse.JSON` 可继续负责写 JSON；错误响应构造建议集中在 response 包或统一 helper。

| Checklist | 范围 | 当前形态 | 建议方向 |
| --- | --- | --- | --- |
| [ ] | `internal/transport/http/server.go` API 404 | `map[string]string{"detail": "Not Found"}` | 使用通用错误响应泛型 |
| [ ] | `internal/transport/http/handler/**/*.go` 中 `detail` 错误 | `map[string]string{"detail": ...}` | 使用统一错误 helper |
| [ ] | Google OAuth placeholder | 503 `map[string]string` | 使用通用错误响应泛型 |
| [ ] | 认证/鉴权失败 | 当前由 authz/handler 分散返回 | 收敛为统一错误响应形态 |
| [ ] | 参数校验/JSON decode 失败 | 当前多处直接写错误 | 收敛为统一错误响应形态 |

## Frontend notes

### 前端请求封装

- `web/src/utils/request.ts` 使用 axios，`baseURL` 来自 `config.apiBaseUrl`。
- 响应拦截器返回 `response.data`，因此 `web/src/api/**/*.ts` 中 Promise 类型即后端 JSON body 类型。
- `web/src/api/cd/deployments.ts` 中 `streamLogs` 使用裸 `fetch('/api/...')`，不经过 axios `baseURL`，需要在文档和后续实现中单独说明。

### 前端侧明显不一致 checklist

| Checklist | 前端接口 | 前端位置 | 问题 | 建议方向 |
| --- | --- | --- | --- | --- |
| [ ] | `deploymentApi.cancel` | `web/src/api/cd/deployments.ts` | 前端 `Promise<void>`，后端返回 `DeploymentResp` | 改前端类型为 `Deployment`/`DeploymentResp`，或后端改 204，需统一 |
| [ ] | `routeApi.enable` / `disable` / `sync` | `web/src/api/cd/route.ts` | 前端 `Promise<void>`，后端返回 message JSON | 定义 `MessageResp` 或统一 void/action response |
| [ ] | `traefikRouteApi.list` | `web/src/api/cd/traefik-route.ts` | 前端期望 `{ items, total }`，后端 handler 返回 service result，需确认真实 shape | 对齐为 `TraefikRouteListResp` 或数组 |
| [ ] | `repositoryApi.delete` | `web/src/api/ci/repository.ts` | 前端发送 `delete_workspace` query，后端未读取 | 删除前端参数或后端补支持 |
| [ ] | `repositoryApi.list` | `web/src/api/ci/repository.ts` | 前端返回 `PaginatedResp<Repository>`，后端 list item 更接近 list response | 改用 `RepositoryListItem` 或后端返回完整 detail |
| [ ] | `deploymentApi.streamLogs` | `web/src/api/cd/deployments.ts` | 使用裸 fetch，不走 `config.apiBaseUrl` | 如跨域部署，需要改为基于 API base URL 或明确同源要求 |
| [ ] | `ApplicationUpdateReq` | `web/src/api/cd/application.ts` | 后端支持 `code`，前端类型缺少 `code` | 明确是否允许前端更新 code |
| [ ] | `ConfigFile` | `web/src/api/cd/application.ts` / `web/src/types` | 前端类型包含 `updated_at`，后端 `ConfigFileResp` 未显式返回 | 对齐类型或后端补字段 |
| [ ] | `applicationApi.getStatus/getLogs` | `web/src/api/cd/application.ts` | 前端类型含 `error?`，后端错误响应未返回 `error` 字段 | 统一错误响应或移除前端 `error?` |
| [ ] | 多个前端 action 调用 | `web/src/api/ci/*.ts` / `web/src/api/cd/*.ts` | 前端传 `{}`，后端不读取 body | 标注无业务请求体，必要时前端改为无 body |
| [ ] | 多个 list/create 接口 | `web/src/api/ci/*.ts` / `web/src/api/cd/*.ts` | `project_id` 在前端多处可选，后端多处实际依赖 | 逐个确认必填性并对齐类型 |

### 前端侧未封装但后端存在

| Method | Path | 说明 |
| --- | --- | --- |
| GET | `/api/auth/google` | 后端存在，当前前端未封装；后端当前 placeholder 503 |
| POST | `/api/ci/webhook/{webhook_id}` | 外部 webhook 入口，前端不封装合理 |
| GET | `/api/ci/repository/{repository_id}/run` | 后端存在，前端当前主要使用 `/api/ci/run` 过滤 |
| `/api/background/...` | 后端 background task 路由存在，当前 `web/src/api` 未发现封装 |

## Technical questions

- 已决策：直接数组响应统一改为 `{ items: []T }`，并定义命名 list response。
- 已决策：action 类接口有响应内容就返回命名 `XxxxResp`；无响应内容就返回 `204 No Content`。
- 已决策：`transportresponse.PaginatedResp[T]` 继续作为统一分页 response。
- 已解释并暂定：handler 层 response DTO 应与 service/repository/model 类型隔离。原因是 HTTP API 是外部契约，service/repository/model 是内部实现；直接暴露内部类型会导致数据库字段、service view、内部模型命名或字段调整时被动改变 API，并让 handler 失去显式转换和字段筛选边界。后续 Plan 需要决定是一次性全部隔离，还是先处理高风险接口。
- 建议保留为特殊协议例外：Webhook raw body、multipart upload、SSE 不强行套普通 JSON `Req` / `Resp`，但必须文档化并命名事件或响应 payload。
- 已决策：错误响应纳入本次规范化，使用通用错误响应泛型或等价结构，不继续散落匿名 `map[string]string{"detail": ...}`。

## Risks

- 如果后续实现一次性修改所有 checklist 项，可能影响前端调用和测试范围，建议分批进行。
- 将直接数组响应改成对象包装会破坏现有前端契约，必须同步修改前端调用和类型。
- 将 action response 改为 `204 No Content` 或资源 response 会改变现有调用方预期，需要逐个确认是否有 UI 使用响应体。
- 将 service/repository/model 类型复制到 handler DTO 可能引入字段遗漏，需要测试覆盖和代码 review。
- SSE、multipart、webhook raw body 不适合机械套 `Req` / `Resp`，过度统一可能降低可读性或破坏协议语义。

## Alternatives

### Alternative A：只文档化，不修改接口

- 优点：风险最低，不影响前端和调用方。
- 缺点：不能解决 Req/Resp 不统一问题，只能形成现状盘点。

### Alternative B：只补命名类型，不改变 JSON shape

- 优点：大多数接口可保持响应 JSON 不变，只提升代码和文档契约清晰度。
- 缺点：直接数组响应仍不是对象包装；部分前端类型命名仍需同步。
- 当前状态：已被用户决策部分否定；直接数组响应需要改为 `{ items: []T }`，因此不能作为完整实现策略。

### Alternative C：全面统一响应包装

- 优点：接口契约最一致，后续 OpenAPI 或 SDK 生成更容易。
- 缺点：破坏性最大，需要同步修改前端、测试和可能的外部调用方。

当前建议：后续实现优先采用 Alternative B；对已经确认前后端不一致的接口单独做行为对齐决策。

## User review notes

- 2026-07-01 用户要求进入 spec，并明确本阶段要列出待修改接口列表，作为后面实现 checklist。
- 2026-07-01 15:38:08 用户确认关键规格决策：直接数组响应统一为 `{ items: []T }`；有响应内容的 action 返回 `XxxxResp`，无响应内容返回 `204 No Content`；`PaginatedResp[T]` 继续作为通用分页 response；错误响应也应有泛型类；需要解释 handler DTO 与 service/repository/model 类型隔离原则。
- 2026-07-01 15:46:01 用户要求无须 Plan，按标准模式开始实现；补充原则：该抽取 DTO 的抽取出来，除非是它自己的逻辑，不要和业务逻辑混在一起。
