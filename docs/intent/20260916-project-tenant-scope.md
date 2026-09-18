# Project 租户作用域显式传递
最后修改时间: 2026-09-16 16:32:37

Review status: Accepted

Mode: strict

## Background

`Project` 是 Orbit 中业务资源的租户作用域，不是一个可有可无的列表筛选项。当前部分集合用例（已确认包括 Application 列表）将 `project_id` 表示为 `*string`：缺失时跳过成员校验，并在 repository 层取消项目过滤。更广泛地，许多项目资源的详情、修改、删除和运行操作只携带资源 ID；现有授权通过读取资源后推导其 Project 完成，HTTP 协议本身没有表达当前 Project 过滤条件。

Web 已经有唯一的当前 Project 来源：Pinia `useProjectStore().activeProjectId`。该值由用户选择并持久化，页面切换 Project 时会重新请求对应数据。后续实现应由各项目资源 API 方法将该值作为 `project_id` query string 显式传递，不通过 Axios interceptor、Vue router state、Gin middleware、context value 或其他隐式机制自动注入。

现有部分 `project_id` 数据列允许 `NULL`，而新建资源已要求非空 Project。历史空归属数据在任务完成后统一归入默认 Project；这不产生运行时默认值或兼容分支。

## Evidence inventory

### 已确认的越界路径

| 范围 | 当前行为 | 问题 | 目标 |
| --- | --- | --- | --- |
| Application 集合 | HTTP 空 `project_id` 经 `transport.QueryProjectId` 变为 `nil`；usecase 跳过成员校验；SQLC `narg` 取消筛选 | 可读取跨 Project Application | HTTP、usecase、repository 都接收必填 `string projectId`，查询固定以该值过滤 |
| Repository 集合 | handler 手工把空 `project_id` 转为 `nil`；usecase 跳过成员校验；SQLC `narg` 取消筛选 | 可读取跨 Project Repository | 与 Application 一致收敛为必填 scope |
| Repository code 查重 | `RepositoryByCode` 以 `*string` 接收可选 Project | 代码唯一性检查仍保留无 scope 分支 | 按 Project 查重的内部 port 使用必填 scope |
| 按 Pipeline Run 列 Artifact | public usecase 已先按 `run_id` 读取并授权，再把 `run.ProjectId` 指针传给 repository | 入站 HTTP 未表达当前 Project，repository port 仍把项目归属表达为可选 | HTTP 显式接收 scope，usecase 与 repository 以 Run ID 和 scope 联合过滤 |

现有 SQL repository 测试明确断言 `ListApplications(..., nil, ...)` 和 `ListRepositories(..., nil, ...)` 返回全量数据。这些是待移除的历史行为，不是后续兼容性承诺。

### 资源归属初步清单

产品模型将 Project 定义为成员权限、资源隔离和部署环境切换边界。以当前 schema 和用例为准，直接或派生属于 Project 的资源分为三类：

| 分类 | 资源 | 结论 |
| --- | --- | --- |
| 已有必填直接归属 | `project_member`、`environment`、`environment_credential`、`service`、`pipeline_stage`、`deployment_dialogue_conversation` | 保持现有归属语义 |
| 当前可空但产品上属于 Project | `repository_credential`、`application`、`repository`、`pipeline`、`pipeline_snapshot`、`pipeline_run`、`artifact`、`deployment`、`route` | 不能继续把 `NULL` 解释为全局资源；任务完成后统一回填默认 Project |
| 通过父资源关联过滤 | Version/Component 及其配置、Service 子表、Pipeline stage run、Pipeline run version binding、GatewayConfig 等 | 不新增冗余 `project_id`；repository 通过父资源 join/`EXISTS` 将请求 scope 写入 SQL 条件 |

其中 `pipeline_snapshot` 可由 Pipeline 派生，Artifact 可由 Pipeline Run 派生；Pipeline Run 和 Deployment 的 Project 还须与其 Repository/Pipeline、Service/Application 等关联交叉核对。Application、Repository、Repository Credential、Pipeline、Route 的 `NULL` 根记录没有可靠的单一父资源可推导归属，因此按已确认的默认 Project 策略回填，不以关联猜测其归属。

### 入站与授权边界

- HTTP 的项目资源接口一律以 `?project_id=<current-project-id>` 表达当前 Project。范围包括集合、创建、导入、批量、同步、详情、修改、删除、运行操作和嵌套资源操作；资源路径中的 `:id` 只标识目标，不能代替租户 scope。
- 前端由每个项目资源 API 方法显式接收必填 `project_id` 参数，调用页从 `projectStore.activeProjectId` 取得值后传入。不得以 Axios request interceptor 或通用 wrapper 自动拼接参数；这样请求签名、页面调用与实际 HTTP URL 都能直接审查。
- HTTP handler 逐个读取 query string 并传给 application usecase；usecase 逐个校验非空和成员资格；repository 逐个把 scope 写入查询。不得通过 Gin middleware、context、全局当前 Project 或先读取资源再隐式取得 Project 来完成过滤。
- 直接拥有 `project_id` 的资源以 `id AND project_id` 查询和写入。通过父资源归属的 Version、Component、Service 子表、Pipeline stage run、Pipeline run version binding、GatewayConfig 等，使用明确 join/`EXISTS` 条件把父资源的 Project 与 query 参数绑定，而不是新增冗余列或在应用层先查询后推导。
- `Project` 管理本身（列出、创建、维护 Project 与成员）以及认证、用户、角色、系统设置和无 Project 归属的 task 不接受当前 Project scope。Environment 与 Initialization 是当前 Project 的运行时资源，改用 query string scope，不继续以 `/api/project/:project_id/...` 作为其 API 过滤实现。
- MCP 的 `orbit_select_project` 仍是唯一面向 MCP 用户接收 `project_id` 的工具。后续 tool schema 继续不暴露该字段，但必须将已选 Project 作为显式普通参数传给 application usecase；不能借由无 scope 的 repository 查询取得数据。

### HTTP 和前端审计范围

| 资源 API 组 | 当前前端状态 | 目标 |
| --- | --- | --- |
| Application、Version、Component 及运行操作 | 仅 list/create 带或可带 `project_id`；详情、版本、组件、日志和运行方法仅收 ID | 每个方法要求 scope；Application 自表或 Application join 参与过滤 |
| Repository、Repository Credential | list 的 `project_id` 可选；详情、修改、删除、导出仅收 ID | 每个方法要求 scope；Repository code 查重和 CRUD 都带 scope |
| Pipeline、Stage Template、Snapshot、Pipeline Run、Artifact | list 中多处可选；触发、运行详情、重试、日志、Artifact by run 和嵌套操作仅收 ID | 每个方法要求 scope；通过 Pipeline/Run 关联的查询显式 join Project |
| Service、Deployment、Gateway、Route | list 或同步接口已有 scope；详情、修改、部署、日志、证书和运行接口仅收 ID | 每个方法要求 scope；Service/Application/Deployment/Gateway/Route 查询联合过滤 |
| Dialogue、Environment、Initialization | Conversation list 带 scope；详情、turn/stream、删除和当前 `/api/project/:project_id/...` 路径未统一 query string | 所有 Project 资源操作使用 query string；SSE 请求也将 scope 放入 URL query |

### 目标接口表达

- Project scope 在前端 API 方法、HTTP handler、application usecase 和 repository port 中以非空 `string projectId` 表达；不再用 `*string`、`sql.NullString.Valid=false`、空字符串或资源读取后的派生值表达“不筛选 Project”。
- 前端 API 类型将 `project_id` 从 optional 改为必填。页面先验证 `projectStore.activeProjectId` 非空，再将它传给每一个 scoped API 调用；这包括 fetch/SSE 的 URL query，不只 Axios 请求。
- 对 Project-owned 查询和写操作，SQL 使用必填 `sqlc.arg(project_id)`。直接归属表使用相等条件；派生资源使用关联表的相等条件。不得保留 `narg` 全量分支，生成的 SQLC 参数和 repository 调用只从 SQL 定义重新生成。
- 不新增 `QueryProjectId`、`OptionalProjectId` 或任何字段专属 helper，也不创建前端/后端的 scope 切面。HTTP adapter 只在各 handler 读取原始 query；application usecase 负责 trim、必填校验和成员授权；repository 只执行已确定 scope 的查询。
- 遇到本应归属 Project 却为 `NULL` 的记录不得被解释为全局资源，也不能成为绕过成员校验、显示全量数据或在请求中省略 scope 的理由。

### 主流实践审视

以多租户隔离与 object-level authorization 的通行做法审视，本任务的核心决策正确，但安全边界必须落在服务端，而不是前端 store 或 query string 本身：

| 实践标准 | 本任务决策 | 结论与约束 |
| --- | --- | --- |
| 每个对象访问都做授权，不能仅相信对象 ID | 请求同时携带资源 ID 与 `project_id` | 正确。application usecase 先验证 actor 是请求 Project 的成员，再以 `id + project_id` 读取或写入；只改 ID 或只改 scope 都不能越界 |
| 客户端传入的 tenant/scope 不可信 | `activeProjectId` 仅是 HTTP 参数来源 | 正确。Pinia 负责用户意图和请求可见性，不能替代 membership/permission 校验或数据库过滤 |
| tenant 边界应在数据访问时强制 | 直接表使用 `project_id`，派生表通过明确关联条件过滤 | 正确。SQL 必须把请求 scope 纳入 `WHERE`/`EXISTS`，不能先按 ID 读出对象再以其归属决定 scope |
| API scope 来源应一致、可审计 | HTTP 使用 query string，MCP 使用已选 scope 的显式普通参数 | 正确。许多系统会选择 path scope 或 middleware/context；本任务选择逐 API query 参数是刻意的可审计性取舍，不降低服务端授权要求 |
| 租户数据访问必须受查询条件约束 | Project-owned 直接列或父资源关联列均绑定请求 scope | 正确。直接表以 `id + project_id` 或 `project_id` 查询；派生表以 join/`EXISTS` 绑定请求 scope |

`project_id` 是不透明但非秘密的资源归属标识，放入受认证 HTTPS API 的 query string 可以接受；日志、错误响应、前端状态和遥测不得把它与 credential、token 或其他秘密混合输出。将 Environment/Initialization 的 path scope 改为 query string 是一致性选择，不是额外的安全机制。

前端还要处理 Project 切换竞态：每次请求捕获发起时的 `activeProjectId`，响应写入状态前确认当前值未变，或取消旧请求。否则即使每个 HTTP 请求都正确过滤，旧 Project 的慢响应仍可能覆盖当前页面。

历史 `NULL project_id` 数据不作为本任务的设计问题处理。任务完成后，对相应表执行一次 `UPDATE <table> SET project_id = <default-project-id> WHERE project_id IS NULL` 即可；不增加运行时默认值或兼容分支。

## Goal

1. 将 Project 定义为项目归属资源的明确租户作用域，而非可选过滤条件。
2. 对所有需要 Project 过滤的项目资源 HTTP 操作，在 query string、前端 API 方法、入站 adapter、application usecase 与 repository 查询边界显式传递非空 `project_id`。
3. 在执行任何项目归属资源操作前，以请求显式提供的 Project 执行成员资格和权限校验，并以同一个值约束资源查询和写入，消除无作用域与跨 Project 访问绕过。
4. 任务完成后将现有 `NULL project_id` 数据统一归入默认 Project，不增加运行时兼容逻辑。

## Non-goal

- 不将用户、认证配置、系统级设置等无 Project 归属的资源人为租户化。
- 不在本任务中改变 HTTP transport、protobuf JSON、错误 body 或请求 ID 契约；缺失作用域仅复用既有 validation 错误出口。
- 不以 `QueryProjectId`、`OptionalProjectId` 等字段专属 helper 代替清晰的用例和 repository 参数。
- 不通过前端请求拦截器、Vue router state、后端 middleware 或 context 携带当前 Project；`project_id` query string 是 HTTP 层唯一 scope 传递方式。

## User scenarios

1. Web 用户切换当前 Project 后，所有 Application、Repository、Pipeline、Service、Deployment 等项目归属资源 API 调用都从 `projectStore.activeProjectId` 显式带上 `?project_id=`；服务仅处理该 Project 数据。
2. 直接 HTTP 调用集合或创建入口时遗漏或传入空 `project_id`；服务返回既有 `400` validation 错误契约，不返回跨 Project 数据。
3. MCP 调用从已选择的 Project scope 取得 ID，并将其显式传递给 application usecase；MCP 不依赖“未传参数即全量查询”的后门。
4. 用户按 Application、Service、Version 等资源 ID 操作时，请求仍带 `?project_id=`；服务以资源 ID 和该 Project 联合查询，缺失 scope 或目标不属于当前 Project 均不能操作该资源。

## Acceptance

- [ ] 规格阶段完成项目归属资源与其 HTTP、MCP、application、repository 入口清单，明确所有项目资源 HTTP 入口均接收 `project_id`，并列出每个直接列或关联过滤条件。
- [ ] 所有项目归属 HTTP API，包括详情、修改、删除、运行和嵌套资源操作，均显式接收并校验 query string 的非空 Project scope；不存在未传 scope 即返回跨 Project 全量数据或访问任意资源 ID 的分支。
- [ ] 所有项目资源前端 API 方法均要求调用方传入当前 Project，且没有 interceptor、router state 或后端切面自动注入。
- [ ] application usecase 和 repository 的项目归属查询不再以 `*string` 表达“可选 Project 过滤”；接口签名反映必填租户作用域。
- [ ] 显式 Project scope 在权限校验、查询过滤和创建归属中使用同一值，且不允许调用方通过不一致的 ID 越过成员校验。
- [ ] 任务完成后，现有 `NULL project_id` 数据以一条 `UPDATE` 语句归入默认 Project；运行时代码不保留 default fallback。
- [ ] 缺失或非法 `project_id` 保持既有 `{code, error, requestId}` 错误格式和 `400` status；401、403、404、405 与 5xx 行为不回归。
- [ ] 覆盖正常 Project 作用域、缺失 scope、非成员 scope、跨 Project 资源 ID 访问、前端 query 传递和 Project 切换竞态的最小有效测试。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- Project 是租户类作用域；对于项目归属资源，缺失 Project 不是“不过滤”，而是无效请求。
- 前端当前 Project 的唯一来源是 `projectStore.activeProjectId`；每个项目资源 HTTP 请求都用 `?project_id=` 显式传递该值，资源 ID 不是 scope 的替代物。
- 后端 scope 必须来自 HTTP query string 或 MCP 已选 Project 的显式普通参数；不从资源查询、请求 context 或其他旁路推导。
- MCP 的连接级选中 Project 是其入站 scope 载体；不改变 MCP tool schema 的前提下，内部仍显式传递该值。
- `project_id` 是请求的未可信声明。只有成员资格/权限校验与 SQL 联合过滤共同成立时，才构成 Project 隔离；前端 store 和 query string 不承担授权职责。
- Project-owned ID 操作在请求 scope 存在且成员合法时，以 `id + project_id` 找不到目标返回 `404`；请求的 Project 不存在或 actor 非成员维持既有 `404`/`403` 语义，避免用资源归属暴露其他 Project 的对象存在性。
- 历史空归属数据在任务完成后用 `UPDATE ... WHERE project_id IS NULL` 归入默认 Project；不设置运行时默认 Project 或兼容分支。
- 本任务移除的是可选 Project scope 的语义，不是为每个 `xx_id` 设计字段专属解析方法。
- 采用严格模式 / strict：本任务涉及授权边界、跨层接口与大量 SQL 查询。
- 本任务独立于 `20260915-http-transport-boundaries`，不将租户语义变更混入包收敛实现。

## Risk

- 直接 API 客户端或内部调用可能依赖无 `project_id` 的全量查询；收紧后会得到 validation 错误，必须以调用方审计和测试明确改造范围。
- 既有空归属数据在任务完成后统一更新到默认 Project，不参与运行时 fallback。
- 当前 API client 和 HTTP handler 中有大量仅收资源 ID 的调用点；逐项加入显式 scope 会扩大变更面，规格必须确保前端参数、usecase 签名和 SQL 条件同时收敛，不能留下一段无 scope 路径。
- Project 切换期间在途请求可能将旧 scope 的响应写入当前页面；实现必须比较请求发起时 scope 或取消旧请求。

## User review notes

- 2026-09-16：用户指出 Project 本质为租户类概念，要求显式传递该字段。
- 2026-09-16：用户要求将该方向拆为独立 Intent 并列入进行中任务，不修改 HTTP transport 收敛任务的范围。
- 2026-09-16：审计确认 Application/Repository 的空 scope 会返回跨 Project 全量数据；MCP 已选中 scope 的用户体验与内部显式传参可以同时成立。
- 2026-09-16：用户明确要求扩大到前端 Pinia 当前 Project 与所有后端过滤入口；HTTP 使用 query string 显式传递，不引入前端或后端 scope 切面，也不以资源归属旁路推导 scope。
- 2026-09-16：用户确认历史空归属数据迁入默认 Project，并要求以主流实践复核整体租户边界决策。结论：逐 API 显式 scope 可行，前提是服务端成员校验、SQL 联合过滤和前端切换竞态保护同时落地。
- 2026-09-16：用户明确历史数据处理仅为任务完成后的 `UPDATE <table> SET project_id = <default-project-id> WHERE project_id IS NULL`；不将其扩展为迁移设计。
- 2026-09-16：用户确认整体租户边界决策无误，授权继续完成严格流程的 Spec、Plan 和 Implementation。
