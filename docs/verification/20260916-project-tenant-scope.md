# Project 租户作用域显式传递验收
最后修改时间: 2026-09-16 16:49:18

Review status: Accepted

Mode: strict

## Intent alignment

本次实现将 Project 明确作为资源租户作用域。Web 前端从 Project store 取得当前 Project，并由每个 Project-owned API 方法显式传递 `projectId`；HTTP 使用 `?project_id=...`；后端 handler、usecase、repository 和 SQL 查询沿同一请求值完成成员校验与数据过滤。

实现没有引入前端 interceptor、后端 middleware/interceptor、context scope 或资源 ID 反推 Project 的旁路。MCP 保留 connection-local 的 selected Project，但仍将其作为普通参数显式传入 scoped usecase。

## Spec alignment

- Project-owned HTTP API 的 scope 统一使用必填 query `project_id`，缺失或空值走既有 400 校验错误。
- Project-owned usecase 和 repository 接收独立的非空 `projectId`，详情、修改、删除和运行操作均同时约束资源 ID 与 Project。
- 直接归属 Project 的表使用 `project_id` 过滤；Version、Component、Service、Gateway、Stage Log、Run Binding 等派生资源使用父资源的 `JOIN`/`EXISTS` 约束。
- Project scope 不使用 `sqlc.narg`，没有保留跨 Project 的全量分支。
- Environment 和 Project Initialization 已从旧 Project path scope 收敛到 query scope；旧路由没有兼容 alias。
- Dialogue proto 直接移除冲突的 `project_id` 字段，没有为本次删除新增 `reserved`。

## Plan alignment

计划中的前端 API、HTTP handler、application usecase、repository port、SQLC query、MCP delivery、Environment/Initialization 路由和回归测试均已完成。SQLC 与 proto 生成步骤已执行，生成代码未手工维护；随后已同步 repository 调用点并修复生成 API 的 `SSH`、`OAuth` 和 `Id` 字段契约。

## Actual diff summary

- 为 Project-owned Web API 增加显式 `projectId` 参数，并在请求配置中写入 `project_id` query。
- 将 scope 从 HTTP handler 传入 usecase、repository 和 SQLC 查询，覆盖列表、详情、写入、删除、运行时、日志与 SSE 路径。
- 将资源读取和写入改为同 Project 约束；跨 Project 目标按既有 not-found 语义处理。
- MCP selected Project 显式进入 scoped application boundary，移除无 scope 读取后再比较归属的路径。
- Environment/Initialization 改为 query scope，Dialogue 输入不再携带冲突的 Project 来源。
- 更新 SQLC 配置、SQL query 定义和生成输出，并同步调用点到生成的 `ProjectId`、`ApplicationId` 等字段。
- 补充和更新 handler、usecase、repository、MCP、前端请求与切换竞态相关测试。

## Expected vs actual changed files

预期变更区域为 `web/src/api/**`、Project-owned views/composables、HTTP handlers/routes、scoped application/repository、MCP delivery、SQL queries/generated SQLC、Dialogue proto 及相关测试。实际 diff 覆盖这些区域，并包含对应的过程文档和生成代码同步；未引入独立 scope 切面、兼容路由、兼容参数或运行时默认 Project。

## Acceptance criteria checklist

- [x] Project-owned HTTP 请求显式携带 `?project_id=<id>`。
- [x] 前端 API 方法显式接收 `projectId`，不依赖全局请求拦截器或隐式注入。
- [x] handler、usecase、repository 逐层显式传递同一 scope。
- [x] 直接资源和父资源派生查询均按 Project 过滤；不存在 `narg(project_id)` 的全量回退分支。
- [x] 缺失 scope、非成员 scope 和跨 Project 资源访问保持预期错误语义。
- [x] Environment/Initialization 使用 query scope，旧 Project path 路由没有兼容入口。
- [x] MCP selected Project 显式传入 scoped usecase，不通过资源反推 Project。
- [x] Dialogue proto 直接删除冲突字段，没有新增本次任务的 `reserved` 或兼容字段。
- [x] SQLC/proto 生成结果与手写调用点一致，Go 工程可编译。
- [x] 前端 lint、format、typecheck、测试和 Go 测试全部通过。

## Test results

| 命令 | 结果 |
|------|------|
| `task sqlc` | 通过；由用户确认已执行，生成 API 与 SQL 定义同步。 |
| `task proto` | 通过；由用户确认已执行，Dialogue 生成代码同步。 |
| `yarn --cwd web lint:fix` | 通过。 |
| `yarn --cwd web typecheck` | 通过。 |
| `task check` | 通过：Web typecheck、ESLint、Prettier 检查通过。 |
| `yarn --cwd web test` | 通过：25 个测试文件、114 个测试。 |
| `go test ./cmd/... ./internal/... ./sql` | 通过：全部 Go 包通过。 |
| `task test` | 通过：前端测试和上述 Go 测试均通过。 |
| `git diff --check` | 通过。 |

## Missed or expanded scope

本次未执行历史数据回填 SQL。此前核对的受影响表中 `NULL project_id` 为 0，因此没有产生实际回填任务；运行时代码也没有加入默认 Project 兜底。SQLC 生成字段和查询方法的命名同步属于生成契约修复，不改变业务 scope 语义。

没有启动、停止或重启开发服务器，也没有执行 Git 暂存、提交或推送。

## Risks and incomplete items

- 这是有意的 API 契约收紧；未携带 `project_id` 的旧外部调用不再兼容。
- Project path scope 到 query scope 是破坏性路由变更，旧路径没有 alias。
- 本次验证覆盖自动化测试和静态检查，未启动开发服务器执行浏览器端人工流程；前端请求契约由 typecheck、Vitest 和 handler/usecase 测试覆盖。

## Conclusion

Project 租户 scope 已按 Intent、Spec 和 Plan 完成实现并通过验收。scope 在前端、HTTP、application、repository、SQLC 和 MCP 边界均为显式传递，未保留隐式推导或兼容回退；测试、lint、format、typecheck 和 diff 检查全部通过。
