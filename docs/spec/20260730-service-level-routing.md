# 服务级路由与显式网关变更
最后修改时间: 2026-07-30 12:54:42

Review status: Accepted

## Requirement Basis

依据已接受的 [Requirement](../requirement/20260730-service-level-routing.md)：Version 是可复用组件规格；Service 是实例和环境相关的运行配置。标准应用部署继续要求存在且运行中的 Gateway，但 public TCP 缺少 Gateway entrypoint 只产生提示，绝不触发 Gateway 写入或部署。

## Current State

- `version_expose` 通过 Version API、Version 详情、Compose renderer 和 Gateway deployment preparation 共同使用，导致业务部署可改写 Gateway 托管 Version。
- `POST /api/application/:app_id/deploy` 接收 `version_id` 和 `instance_key`，会按该组合创建或更新 Service；Version 详情也直接调用该接口。
- `POST /api/version/:version_id/preview` 按 Version 和 instance key 渲染 Compose，忽略 Service 专属配置。
- 标准应用部署在入队前已经调用 `EnsureGatewayRunning`，Gateway 缺失或未运行的既有阻断语义保持不变。
- 开发 SQLite 中有 3 条 `version_expose`，每条均恰好匹配一个当前 Service；普通应用没有 `version_component_port`，现有 port mappings 都属于 Gateway。

## Ownership

```text
Version
  components, image, command, mount, env, dependency

Service
  version selection, runtime configuration, service exposes
  local host binding or public Traefik routing intent

Gateway
  static entrypoints and published ports, configured and deployed explicitly
```

`ServiceExpose` 是持久化的运行绑定，不保存任何 Docker label。Compose renderer 根据它和当前 Gateway 配置临时派生端口发布、网络与 Traefik labels。

`VersionComponentPort` 保留给 `kind=gateway` 的静态网关端口配置。标准应用 Version 不再允许组件端口直接发布到宿主机；其 local/public 端口行为全部经 `ServiceExpose` 表达。这样 Redis 等标准组件可由不同 Service 选择 local `127.0.0.1:6379:6379` 或 public TCP，而 Gateway 的 `6379:6379` 仍只能由用户编辑并部署 Gateway 后生效。

## Data Model

移除 `model.VersionExpose`，新增 `model.ServiceExpose`：

| Field | Meaning |
| --- | --- |
| `id` | Stable service-expose ID |
| `service_id` | Owning Service |
| `component_name` | Component name in the Service's selected Version |
| `protocol` | `http` or `tcp` |
| `container_port` | Target component port |
| `path_prefix` | HTTP-only route prefix |
| `access` | `local` or `public` |
| `listen_port` | Optional local/public TCP listener; omitted means container port |
| timestamps | Standard audit timestamps |

`service_expose` has a foreign key to `service(id)` with cascade delete, an index on `service_id`, and a unique key `(service_id, component_name, protocol, container_port)`. The existing `version_expose` table and all Version-scoped repository methods are removed.

The Service repository gains `ServiceExposesByService` and `ReplaceServiceExposes`. The Service use case owns validation, replacement and authorization. Validation verifies component membership against the Service's selected Version, protocol/access values, port ranges, HTTP-only path prefixes, duplicate local binds and public TCP listen conflicts. Version component rename/delete must reject references from Service exposes; it must not rewrite another aggregate's configuration.

## API And UI Contracts

### Version

- Remove `VersionExposeReq`, `VersionExposeResp`, `exposes` from Version create/update/response messages, Version import/export bundles, mappers and Version detail UI.
- Remove Version-detail expose controls, direct deploy dialog and Compose preview drawer.
- Remove `POST /api/version/:version_id/preview` and `VersionPreviewReq/Resp`; there is no Version-only Compose document after this change.

### Service

Service becomes the configuration and operation surface:

- `ServiceCreateReq` accepts the initial Version, instance key, runtime configuration and repeated `ServiceExposeReq`.
- Add `ServiceBasicUpdateReq(version_id, instance_key)`. Application ownership remains immutable. The update validates the persisted runtime configuration and exposes against the selected Version, then persists the basic fields atomically.
- The runtime configuration update request contains only runtime configuration and repeated `exposes`; it validates the entire desired configuration against the Service's already selected Version before replacing its expose collection, and it does not deploy.
- `ServiceResp` and Service detail responses include repeated `ServiceExposeResp`.
- Add Service Compose preview endpoint and messages, taking `service_id` only.
- Add Service deploy endpoint and messages, taking `service_id` and `force_recreate` only. The response includes `deployment_id` plus non-blocking warnings.

Service detail puts Version selection and instance key in an editable basic-information section opened through a modal, matching other detail pages. Environment variables and exposes render as separate cards with independent edit, save and cancel controls. Because the configuration contract is atomically validated and persisted, each card saves its edited field together with the other field's latest persisted value; no defaults or fallbacks are introduced at submit time. The Service list/detail deploy dialogs deploy the selected Service's saved configuration and no longer show a Version selector or instance-key input. Version detail contains specification actions only. Application-detail Version Compose previews are removed.

The Gateway detail keeps its explicit Version and Service deployment controls. Its component ports remain the manual way to add static TCP entrypoints and Docker published ports.

No compatibility aliases are retained for the removed Version preview or application-level Version deployment API. Web and MCP clients are updated atomically to Service contracts: `orbit_preview_version` becomes a Service preview operation, and `orbit_deploy(application_id, version_id, instance_key, ...)` becomes Service-targeted deployment.

## Deployment And Rendering

`DeployService` loads the persisted Service, its selected Version, components, runtime configuration and Service exposes. It creates a deployment record with the Version ID as execution history, but it never accepts a Version or expose collection from the deploy request and never creates an implicit Service.

The deployment worker passes `[]model.ServiceExpose` to `RenderInput`. The renderer behavior is:

| Service expose | Generated service Compose |
| --- | --- |
| local | `127.0.0.1:<listen>:<container>` on the selected component |
| public HTTP | Gateway network plus derived HTTP labels using the Gateway base domain and default entrypoint |
| public TCP | Gateway network plus derived `tcp<listen>` labels; no host port is published by the business component |

Labels remain generated artifacts. They are never stored in Version or Service records.

Before a Service deploy command is queued, the Gateway coordinator retains `EnsureGatewayRunning`. It also compares each public TCP `listen_port` against the currently deployed Gateway Service's static component ports. Missing ports produce a response warning such as “Gateway is running but does not expose tcp6379; configure port 6379 and deploy Gateway.” The command still queues the Service deployment.

The execution path removes `PrepareDeployment` behavior that compiles Gateway listeners, modifies the Gateway Version, sets `RolloutConfig`, or calls `deployGatewayInPlace`. Gateway configuration is read only for rendering and warnings. A later explicit Gateway Service deployment is the only way to apply a static entrypoint change.

## Version-Scoped Entry Points To Remove

| Current entry point | Replacement |
| --- | --- |
| Version detail Deploy dialog | Service detail Deploy action |
| Version detail Compose preview | Service detail Compose preview |
| Application detail Version Compose preview | Navigate to/select a Service and preview it |
| `POST /api/application/:app_id/deploy` with Version + instance key | `POST /api/service/:service_id/deploy` |
| `POST /api/version/:version_id/preview` | `POST /api/service/:service_id/preview` |
| MCP `orbit_deploy(application_id, version_id, instance_key, ...)` | Service-targeted deploy tool |
| MCP `orbit_preview_version(version_id, instance_key)` | Service-targeted preview tool |

## Baseline SQL And Development Database

No new numbered migration is created. No Go business migration and no standalone data-migration script is added.

The existing baseline SQL files are edited in place for both SQLite and MySQL:

1. `000023_application` no longer creates, indexes or drops `version_expose`.
2. `000026_service` creates `service_expose` and drops it in its paired down script.
3. Service and application sqlc queries are updated to match their new ownership, then generated code is regenerated through the repository's normal toolchain.

The existing development SQLite database is changed in place during implementation. Before mutating it, inspect the version-to-Service mapping; continue only when each `version_expose` has exactly one current Service. For the current development database, all three records meet this condition. Create `service_expose`, copy each row using that Service ID, validate counts and foreign keys, then drop `version_expose` and its index. Existing Gateway component ports, including the legacy `6379` port, are preserved because the system cannot infer whether a user now intends to keep them; cleanup remains an explicit Gateway edit.

## Verification Design

- Service use-case tests cover create/update validation, component membership, local listener conflict and public TCP collision rules.
- Renderer tests cover one Service configuration rendered in local, public HTTP and public TCP modes, and assert standard component ports do not independently publish host ports.
- Deployment command tests assert Gateway absent/not-running remains blocking, while running Gateway lacking a requested TCP listener returns a warning and queues only the Service deployment.
- Execution tests assert public TCP deployment never calls Gateway compile/reconcile or force recreation and never writes Gateway ports/configuration.
- HTTP/proto/frontend tests cover removal of Version expose/deploy/preview contracts and the Service config, preview and deploy replacements.
- SQLite migration verification checks `integrity_check`, foreign-key check, three copied Service exposes and absence of `version_expose` in the development database.

## Risks And Constraints

- This is an intentionally breaking local API and UI contract change. No compatibility fallback is permitted because retaining Version deployment paths would preserve the incorrect ownership model.
- A Service deployment may complete while its public TCP endpoint remains unreachable until the user explicitly configures and deploys Gateway. The warning must be visible but non-blocking.
- Gateway static port removals are never inferred from Service changes, avoiding accidental shared-ingress disruption but allowing explicit cleanup work to remain with Gateway operators.
- Existing Service configuration is the source of intent. Deployment requests must not introduce an alternate, transient expose payload.

## Alternatives Rejected

- **业务部署自动同步 Gateway**：会让 Service 部署拥有共享入口的生命周期和停机风险，违背显式 Gateway 控制边界。
- **保留 Version expose，部署时复制到 Service**：会形成两份可变配置，无法定义哪一份是 Compose 的真实输入。
- **仅在部署请求中提交 expose**：无法在 Service 详情、Compose 预览、重试和运行时查询中复用同一配置。
- **为旧 Version deploy/preview API 保留兼容别名**：会继续允许绕过 Service 配置的运行时操作，与模型迁移目标冲突。

## User Review Notes

- 2026-07-30：用户要求将 Traefik 路由能力从 Version 转到 Service，禁止业务部署自动变更 Gateway。
- 2026-07-30：用户澄清 Gateway 存在且运行仍是部署前置条件；缺少 public TCP entrypoint 仅提示。
- 2026-07-30：用户要求废弃从 Version 发起的部署和预览入口。
- 2026-07-30：用户要求修改既有 baseline SQL，并就地修改开发 SQLite；不新增迁移文件、业务迁移或迁移脚本。
- 2026-07-30：用户要求服务基本信息支持修改，Version 不得出现在运行时配置编辑区或其请求中。
