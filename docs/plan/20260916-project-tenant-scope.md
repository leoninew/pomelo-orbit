# Project 租户作用域显式传递计划
最后修改时间: 2026-09-16 16:32:37

Review status: Accepted

Mode: strict

## Basis

依据已接受的 [Intent](../intent/20260916-project-tenant-scope.md) 和 [Spec](../spec/20260916-project-tenant-scope.md) 实施。用户已授权在 Plan 后直接进入 Implementation。本计划不修改 HTTP transport 契约，也不引入 scope middleware、interceptor、context 或 runtime default Project。

## Implementation steps

1. 建立 scoped API 改造基线。
   - 搜索每个 `web/src/api` Project-owned 方法及其 view/composable 调用点。
   - 将 `projectId: string` 变为显式必填首参；每个方法在自身 `params` 中写 `project_id`。
   - 保持 `projectApi`、auth、user、role、settings、background task 不变。

2. 收敛 CI 与 Application 链路。
   - 改造 Application、Version、Component、Repository、Repository Credential、Pipeline、Pipeline Stage、Snapshot、Pipeline Run、Artifact 的前端 API、handlers、usecase signatures 与 repository ports。
   - 将 Application/Repository/Pipeline Run/Artifact 的 optional scope、`*string` 和 SQL `narg(project_id)` 分支替换为必填 scope。
   - 对 Version、Component、Stage Log、Snapshot、Artifact by Run 使用父资源 `JOIN`/`EXISTS` scoped 查询，不先按 ID 读取再推导 Project。

3. 收敛 CD 与运行时链路。
   - 改造 Service、Deployment、Gateway、Route、Dialogue 的详情、写入、运行、日志和 SSE API，使全部显式接收 scope。
- 把 runtime target 的加载统一放在已经受 `projectId` 限制的 usecase/repository 查询之后。
   - 将 Environment 和 Project Initialization 路由从 `/api/project/:project_id/...` 直接改为 query-scoped 新路由，更新对应前端模块和 handler 测试。

4. 收敛 application/repository 数据访问边界。
- 所有 public scoped usecase 在入口 trim、校验 `projectId`，校验 actor membership/permission，再调用 scoped port。
   - 直接表的 CRUD 改为 `id + project_id`；派生表以明确 `JOIN`/`EXISTS` 限制父资源 Project。
   - 更新 `sql/query/**`，保持 SQLC 参数具名且 `project_id` 为 `sqlc.arg(project_id)`；运行 `task sqlc`，只提交由 SQL 定义产生的 `internal/gen/sqlc/**` 变更。

5. 收敛 MCP 入口。
   - 保留连接级 `orbit_select_project` 体验和 tool schema。
   - 将已选择 Project 显式传到所有 scoped application 调用；移除先无 scope 读取后 `assertResourceProject` 的授权路径。

6. 修复 Web Project 切换状态。
   - 在每个加载 Project-owned 数据的页面/组合函数捕获发起时的 `activeProjectId`；响应写状态前比较当前值，或取消旧请求。
   - Project 切换后重新加载或清空局部数据，避免旧 Project 的迟到响应覆盖当前页面。

7. 添加最小有效回归测试。
   - handler/application tests：缺失 scope `400`、非成员请求、同成员跨 Project ID `404`。
   - repository SQLC tests：直接归属和一个父关联资源的 scope 查询，列表叠加 search/pagination 的参数绑定。
   - frontend tests：请求 URL 的 `project_id` 与切换竞态保护。
   - MCP tests：连接选中 Project 作为普通 usecase 参数传入，外部 ID 不通过 unscoped read 获得。

## Files to change

| 范围 | 文件/目录 |
| --- | --- |
| Web API 与调用点 | `web/src/api/**`、`web/src/views/**`、相关 composables/tests、`web/src/stores/project.ts`（仅在切换保障需要时） |
| HTTP routes/handlers | `internal/api/http/routes/{application,repository,credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,environment,project_initialization}.go` 与同名 handler 目录 |
| Application | `internal/application/{application,repository,credential,pipeline,pipeline_run,service,deployment,gateway,route,dialogue,environment,project_initialization}/**` |
| Repository/SQL | `internal/repository/*.go`、`internal/repository/impl/sqlc/**`、`sql/query/**`、`internal/gen/sqlc/**` |
| MCP | `internal/api/mcp/delivery/**` |
| Process docs | 本任务的 `docs/intent`、`docs/spec`、`docs/plan`，Verification 仅在用户明确进入验证阶段后创建 |

## Verification plan

1. 执行 `task sqlc` 后检查生成接口不出现新的 `ColumnN` 或参数顺序异常。
2. 修改前端后执行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，并运行相关 Vitest。
3. 修改 Go 后端后执行 `task check` 与 `go test ./cmd/... ./internal/...`；对 scope 变更补充 focused package tests 以缩短定位路径。
4. 在用户要求进入 Verification 前，不创建 `docs/verification/**`；Implementation 阶段仍报告实际 diff、检查结果与已知风险。

## Risks and assumptions

- 本次是有意的 API 收紧：未传 `project_id` 的外部调用将收到 `400`，不保留兼容分支。Web 与 API 同版本交付。
- 路由从 Project path scope 改为 query scope 会是破坏性变更；不提供旧路由 alias。
- SQLC 只以 MySQL schema 生成，但 repository tests 实际覆盖 SQLite 路径；每次改动同时核对 SQL、生成代码与执行测试。
- 历史 `NULL project_id` 数据不进入产品代码。任务完成后再对受影响表执行 `UPDATE <table> SET project_id = <default-project-id> WHERE project_id IS NULL`。

## Rollback

代码层回滚以整体提交为单位，恢复旧 API 与查询行为；不以 runtime fallback 或双路径方式回滚。数据回填在执行前先确认默认 Project，且仅更新 `IS NULL` 行；它不依赖本次运行时代码的兼容逻辑。

## User review notes

- 2026-09-16：用户确认整体决策无误，要求继续推进；无问题时完成 Plan 后直接进入 Implementation。
