# Project 租户作用域显式传递规格
最后修改时间: 2026-09-16 16:32:37

Review status: Accepted

Mode: strict

## Intent basis

本规格落实 [Project 租户作用域显式传递 Intent](../intent/20260916-project-tenant-scope.md)。`Project` 是成员权限、资源隔离和部署环境切换边界。项目归属资源的 scope 缺失不是“不过滤”，而是无效请求。

用户已确认：采用逐 HTTP API 以 `project_id` query string 显式表达 scope；不使用前端/后端自动注入切面，不从资源 ID 或已读取资源反向推导 scope。历史空归属数据仅在任务完成后用一次 `UPDATE ... WHERE project_id IS NULL` 回填默认 Project，不产生运行时代码路径。

## Overview

```text
Pinia activeProjectId
  -> 前端 scoped API 的 projectId 必填参数
  -> ?project_id=<projectId>
  -> Gin handler 原样读取 query
  -> application usecase(projectId, targetId, input)
  -> 成员/权限校验
  -> repository 的 projectId 参数
  -> WHERE direct.project_id = ?
     或 JOIN/EXISTS 父资源 project_id = ?
```

`project_id` 是调用方声明，不是可信授权信息。只有成员资格/权限校验和 SQL 的同值过滤共同成立时，Project 隔离才成立。资源 ID 仅标识目标，不能代替 scope。

## Design decisions

### 显式接口契约

1. 每个 Project-owned HTTP 操作都接受 `?project_id=<id>`，包括列表、详情、创建、修改、删除、导入、批量、嵌套资源、运行时操作、日志和 SSE。
2. scoped 前端 API 统一将 `projectId: string` 作为显式参数，而不是藏在可选 `params` 中。参数顺序为 `projectId`、目标 ID、业务 input/filter，例如 `get(projectId, id)`、`update(projectId, id, input)`、`list(projectId, filters)`。
3. 前端 API 方法各自在请求配置中写入 `{ params: { project_id: projectId, ... } }`。不得增加 Axios interceptor、scoped request wrapper 或 router state 作为自动传递机制。
4. handler 使用 `c.Query("project_id")` 取得原始值并传给对应 usecase；不增加 `QueryProjectId`、`OptionalProjectId` 或其他字段专属解析 helper。
5. application 对外用例将 `projectId string` 作为独立、必填参数，紧随 `userId`。创建 DTO 不再同时保存另一个可冲突的 `ProjectId` 输入字段；Project 归属只来自 usecase 参数。
6. repository port 的 Project-owned 读取、更新和删除同样接收非空 `projectId string`。生成 SQLC 参数由 SQL 定义产生，禁止手改 `internal/gen/sqlc/**`。

Project 管理、认证、用户、角色、系统设置和无 Project 归属的后台 task 不加该参数。它们不是 scoped HTTP API。

### 授权、可见性与错误语义

1. 每个 public scoped usecase 先 `TrimSpace(projectId)`；空值返回既有 validation error，HTTP 映射为 `400` 和既有 `{code,error,requestId}` body。
2. usecase 以该值验证 Project 存在、actor 成员资格和现有操作权限。请求 Project 不存在或 actor 非成员沿用当前 `404`/`403` 语义。
3. 成员校验通过后，repository 以请求 `projectId` 和目标 ID 联合查询。目标属于其他 Project 时结果为 `ErrNotFound`，HTTP 返回 `404`，不泄露目标是否存在。
4. 不允许先按目标 ID 查出资源，再从资源记录中的 Project 做成员校验或 scope 比对。该模式不满足请求边界与数据访问边界一致的要求。
5. 继续使用 `transport.WriteError`。本任务不新增 error body、status 映射或 request ID 机制。

### SQL 查询策略

| 归属形式 | 资源 | 查询和写入条件 |
| --- | --- | --- |
| 直接 `project_id` | Application、Repository、Repository Credential、Pipeline、Pipeline Stage、Pipeline Snapshot、Pipeline Run、Artifact、Service、Deployment、Route、Deployment Dialogue Conversation、Environment | 集合以 `project_id = sqlc.arg(project_id)`；详情/写入以 `id = sqlc.arg(id) AND project_id = sqlc.arg(project_id)`。删除、更新和状态变更使用同一谓词。 |
| Application 派生 | Version、Version Component 与各 Component 配置子表、Application Services、Gateway/GatewayConfig | 从 Version/Component/Service/Gateway join Application，并以 `application.project_id = sqlc.arg(project_id)` 约束；若直接查询 Service，额外保持其 `project_id` 与 Application 一致。 |
| Pipeline 派生 | Pipeline Stage Reference、Application Pipeline Stage Node、Pipeline Run Stage Log、Pipeline Run Version Binding | join Pipeline 或 Pipeline Run，并以其 `project_id = sqlc.arg(project_id)` 约束；嵌套 path ID 仍与父 ID 同时参与过滤。 |
| 运行时派生 | Application runtime、Service deployment/preview、Deployment logs/container logs、Route sync/Traefik config | 先由同一条 scoped repository 查询取得 Application/Service/Deployment/Route，再执行运行时操作；不允许运行时工具按无 scope 的 ID 加载目标。 |

Project filter 本身永远不是 `sqlc.narg`。搜索、日期、repository、pipeline、application 等业务过滤仍可使用 `sqlc.narg`，但 `project_id` 必须是必填 `sqlc.arg(project_id)`。同一 SQLC 语句保持命名参数风格；MySQL `LIMIT ? OFFSET ?` 的已有解析器例外不扩展到 scope 条件。

### HTTP 与前端 API 矩阵

下表中的每一项均在 URL query 显式携带 `project_id`。列出的路径集合覆盖现有 route 中所有 Project-owned 操作；`*` 仅表示同一资源下的具体 HTTP method 均遵循该契约，不是未审计的通配规则。

| API 组与前端模块 | HTTP 路径集合 | usecase/repository 归属 | SQL scope |
| --- | --- | --- | --- |
| `applicationApi` | `/api/application`、`/api/application/:app_id`、`/stop`、`/restart`、`/status`、`/logs`、`/service`、`/version` | application 与 deployment/service usecase 均接收 `projectID` | Application 直接过滤；Service/Version 通过 Application 关联 |
| `applicationApi` Version/Component | `/api/version/:version_id`、`/preview`、`/publish`、`/unpublish`、`/fork`、`/component` 及 component 的 `basic/runtime/endpoints/env/mounts/dependencies/devices/advanced`、删除 | application、deployment usecase 接收 `projectID`、version/component ID | Version -> Application；Component -> Version -> Application |
| `repositoryApi` | `/api/repository`、`/api/repository/:repository_id` | repository usecase 接收 `projectID` | Repository 直接过滤；code uniqueness 以 `project_id + code` |
| `credentialApi` | `/api/repository-credential`、`/import`、`/:credential_id`、`/export` | credential usecase 接收 `projectID` | Credential 直接过滤；关联 Repository 的检查同 scope |
| `pipelineApi` 与 `pipelineStageApi` | `/api/pipeline`、`/:pipeline_id`、`/instantiate`、`/stage`、`/stage/:stage_id`、template-update preview/apply、`/snapshot/:snapshot_id`、`/api/pipeline-stage` 与 `/:stage_id` | pipeline usecase 接收 `projectID` | Pipeline/Stage/Snapshot 直接过滤；引用和节点 join Pipeline |
| `pipelineRunApi` 与 `artifactApi` | `/api/pipeline-run`、`/:run_id`、`/cancel`、`/retry`、`/:run_id/artifact`、`/:run_id/stage/:stage_run_id/log`、`/api/pipeline/:pipeline_id/trigger`、`/api/pipeline-run/artifact`、`/:artifact_id` | pipeline-run usecase 接收 `projectID` | Run/Artifact 直接过滤；stage log/binding join Run |
| `serviceApi` 与 `deploymentApi` | `/api/service`、`/:service_id`、`/basic`、`/env`、`/component/:component_id`、`/preview`、`/deploy`；`/api/deployment`、`/:deployment_id`、`/logs`、`/container-logs`、`/cancel` | service/deployment usecase 接收 `projectID` | Service/Deployment 直接过滤，并 join Application where required |
| `gatewayApi` | `/api/gateway`、`/:gateway_id` | gateway usecase 接收 `projectID` | Gateway Application join/`EXISTS` on Application Project |
| `routeApi` 与 `traefikApi` | `/api/route`、`/sync/preview`、`/sync/confirm`、`/:route_id`、`/enable`、`/disable`、`/cert`、`/https`、`/letsencrypt`、`/mkcert`、`/traefik`、`/traefik/config` | route usecase 接收 `projectID` | Route 直接过滤； managed target and gateway lookup bind same Project |
| `dialogueApi` | `/api/deployment-dialogue/conversation`、`/:conversation_id`、`/turn`、`/turn/stream` | dialogue usecase receives `projectID`; turn input no longer supplies a conflicting source | Conversation direct filter; created conversation uses request scope; SSE query also includes it |
| `environmentApi` | `GET/PUT /api/environment`、`POST /api/environment/probe`、`POST /api/environment/initialize` | environment usecase already accepts `projectID`; handler source changes from path to query | Environment direct `project_id` query |
| `projectInitializationApi` | `GET /api/project-initialization` and `POST /environment/test`、`/environment`、`/environment/windows-command`、`/bootstrap`、`/probe`、`/gateway` | initialization usecase already accepts `projectID`; handler source changes from path to query | Project/Environment/Gateway operations bind request scope |

The former `/api/project/:project_id/environment...` and `/api/project/:project_id/initialization...` routes are removed directly. They are replaced by the query-scoped paths in this table; no alias or compatibility route remains.

### Frontend current-Project behavior

`useProjectStore().activeProjectId` remains the only Web source of the selected Project. Each Project-owned view or composable must:

1. read it before invoking a scoped API and require a nonempty value;
2. pass the captured value to that API method explicitly;
3. capture it for asynchronous loads and discard the result when `activeProjectId` has changed, or cancel the old request;
4. clear or reload page-local Project data after a Project switch.

No store state is injected below the page/composable call site. The API module receives a string and renders it into the request URL; the server remains the authorization boundary.

### MCP boundary

`orbit_select_project` remains the only MCP tool schema that receives `project_id`. The connection's selected Project is retained as MCP protocol state, but every call from MCP delivery into a scoped application usecase passes that selected value as its ordinary `projectID` argument.

`applicationInScope`/`versionInScope`/`serviceInScope`/`gatewayInScope`/`routeInScope`/`deploymentInScope` must stop loading an unscoped resource then comparing its Project. They should call the scoped usecase directly. Tool schemas remain unchanged; this is an MCP protocol exception, not an unscoped application or repository path.

## Affected components

- `web/src/stores/project.ts`, all Project-owned views/composables, and `web/src/api/{application,repository,credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,project}/**`.
- `internal/api/http/routes/{application,repository,credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,environment,project_initialization}.go` and their handlers.
- `internal/application/{application,repository,credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,environment,project_initialization}/**` plus corresponding ports.
- `internal/repository/*.go`, `internal/repository/impl/sqlc/**`, `sql/query/{application,repository,repository_credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,environment}/**`, generated SQLC output, and focused tests.
- `internal/api/mcp/delivery/**` and its scope tests.

## Alternatives considered

| Alternative | Decision | Reason |
| --- | --- | --- |
| Optional list filter or pointer Project ID | Rejected | Produces an unscoped branch and has already exposed cross-Project collection data. |
| Resource ID lookup followed by Project derivation | Rejected | Request scope is no longer explicit at the data boundary and can disclose or authorize the wrong object. |
| Axios/Gin middleware/context propagation | Rejected | Hides the actual scoped API contract and makes call-site audit incomplete. |
| Path-scoped Project routes | Rejected for this task | Query string is the accepted consistent HTTP representation; existing path-scoped environment/initialization routes are removed. |
| Database RLS or a new cross-cutting authorization framework | Rejected | The supported multi-database architecture and current boundaries do not provide one common mechanism; explicit application authorization and SQL predicates are the selected control. |

## Risks and test design

| Risk | Required evidence |
| --- | --- |
| A route or front-end method omits scope | Route/API inventory tests and TypeScript call-site compilation; focused handler tests assert missing `project_id` is `400`. |
| ID operation crosses Project | For each direct and derived relation class, member actor requests a foreign Project ID with its own Project scope and receives `404`. |
| Membership is bypassed | Test an existing Project requested by a non-member and retain its established `403`/`404` behavior. |
| SQLC parameter binding regresses | Regenerate via `task sqlc`; repository tests exercise scope plus search/pagination filters and assert no `narg(project_id)` full-list branch remains. |
| Project switch races | Frontend tests cover an old asynchronous response arriving after `activeProjectId` changes and verify it does not write state. |
| MCP returns to post-load scope checks | MCP scope tests assert selected Project reaches each scoped application entrypoint and an out-of-scope target is not loaded through an unscoped lookup. |

## Technical questions

暂无阻塞性技术问题。现有 error writer 已将 application validation error 映射到既有 `400` 契约。历史 `NULL` 行回填仅是任务完成后的运维动作，不改变本设计。

## User review notes

- 2026-09-16：用户确认以主流多租户与 object-level authorization 实践审视后，整体决策成立。
- 2026-09-16：用户要求完成 Plan 后直接进入 Implementation；本规格无待决事项，按该授权继续。
