# Gateway REST API 访问上下文验证
最后修改时间: 2026-09-18 15:28:00

Review status: Accepted

Mode: standard

## Intent alignment

- Gateway REST API 现在明确区分容器端点 `http://traefik:8080` 与部署宿主机端点 `http://127.0.0.1:8080`，两者随 Gateway 定义持久化、初始化和接管包迁移。
- RouteManager 在唯一请求入口按实际执行上下文选择一次 endpoint：容器化 local Orbit 使用容器端点；宿主机 local Orbit 和 SSH Environment 使用宿主机端点。请求失败不会切换 endpoint 或重试 GET/PUT。
- Gateway 初始化继续把容器 8080 以 `local` 模式绑定到 `127.0.0.1:8080`；Service 没有新增端口 overlay。
- 已删除固定生成的 `traefik-dashboard.<base_domain>` 业务 Route；Traefik dashboard API/UI 与普通 HTTP/TCP Route 上游语义均未改变。
- 接管包升级为 v2，运行时 `network_name` 不再导出；没有为旧包增加业务层兼容、URL 猜测或导入期修复。

`WaitUntilReady` 保留部署后的就绪等待轮询，但每次都只访问已选定的同一 endpoint；它不属于 endpoint fallback 或 Route GET/PUT 重试。

## Plan alignment

| 计划项 | 结果 |
| --- | --- |
| 双 REST endpoint 的模型、配置、SQLC、三方言迁移 | 已完成 |
| 初始化、HTTP/proto/MCP/Web 传递与展示 | 已完成 |
| 按执行上下文单次选择 endpoint | 已完成 |
| 删除历史系统 dashboard Route，保留用户 Route 与 dashboard API/UI | 已完成 |
| Handover v2 与运行时字段隔离 | 已完成 |
| 主流程自动化测试 | 已完成 |

## Actual diff summary

- `GatewayConfig`、Gateway Repository、SQL 查询和 schema 新增 `rest_api_host_url`；`000047` 为历史 Gateway 写入 loopback endpoint，并仅删除结构精确匹配的系统 Route。
- Gateway factory、完整定义导入导出与删除不再拥有 dashboard Route；`GatewayDefinition` 只包含 Application、Config 和 RuntimeService。
- Traefik RouteManager 的 router/service 读取、readiness 与 REST snapshot PUT 共享同一地址选择路径。
- Traefik router/service 读取与同步预览在实际 REST 调用失败时，返回包含执行位置和已选 endpoint 的 `service_unavailable`；curl/SSH 原始错误仍只保留在错误链和服务端日志。配置校验、目标解析和 Traefik 响应解码不再被误归类为不可用。
- Project Initialization、Gateway HTTP/proto/MCP 以及 Web 表单支持两个 endpoint；UI 文案明确容器与宿主机地址。
- PostgreSQL 开发库在首次执行 `000047` 时发现 `route.enabled` 与 `route.https_enabled` 为整数；迁移条件已从 `false` 修正为 `0`，随后成功迁移。
- 生成的 proto、SQLC 文件与相关测试夹具已同步更新。

## Expected vs actual files

计划涉及的模型、Gateway/Route/Initialization/Handover 应用层、HTTP/MCP/proto、SQL schema/migration、Web Gateway 表单和相关测试均已变更，实际范围与 Plan 一致。

`web/src/views/route/RoutePage.vue` 同时存在表单重置移动到数据读取成功后的改动；它不改变本任务的 endpoint、dashboard Route 或 handover 行为，建议在提交时单独审阅或拆分。

## Acceptance checklist

- [x] 三种执行位置具有确定且可持久化的 REST 地址语义。
- [x] 所有 Traefik REST 调用复用同一访问选择路径。
- [x] local 宿主机与 SSH 使用同一 host endpoint；容器化 local 使用 container endpoint。
- [x] Handover 不猜测、不改写 REST URL，v2 不导出运行时网络字段。
- [x] 普通 Route 上游仍按用户提供的 URL 发布，未增加限制。
- [x] 固定 dashboard Route 不再创建、恢复、导出或删除；历史系统记录通过迁移清理。
- [x] PostgreSQL 开发库已实测迁移到 `47 clean`，host endpoint 列存在，匹配的系统 dashboard Route 数量为 0。
- [x] HTTP `service_unavailable` 响应展示实际执行位置和已选 endpoint；不会返回 curl/SSH 原始错误。

## Test results

| 命令或操作 | 结果 |
| --- | --- |
| `task proto` | 通过 |
| `go test ./internal/infrastructure/external/traefik` | 通过；覆盖容器 local、宿主机 local、SSH、失败不 fallback 与 SSH snapshot PUT |
| Traefik 错误上下文定向测试 | 通过；覆盖三种执行位置、HTTP 503 响应消息，以及配置/解码错误不误报为不可用 |
| Gateway、Project Initialization、Handover、迁移与 HTTP handler 定向 Go 测试 | 通过 |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 通过 |
| Gateway Web Vitest 定向测试 | 2 files、4 tests 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `go test ./internal/infrastructure/database` | 通过 |
| `task check` 与 `./bin/golangci-lint run ./cmd/... ./internal/... ./sql` | 通过，静态分析 0 issues |
| 错误上下文变更后的 `task check` | TypeScript 与 ESLint 通过；Prettier 因已有 `web/src/views/environment/EnvironmentPage.vue` 格式问题退出，本次未改动该文件 |
| 开发 PostgreSQL `000047` 迁移 | 通过，最终 `version=47, dirty=false` |

未启动开发服务器，也未操作浏览器。远程 SSH 目标的真实 Traefik API 未在本轮重新调用；其地址选择由单元测试覆盖。

## Risks and incomplete items

- 已导出的 v1 接管包按设计被拒绝；需要重新从支持 v2 的版本导出。
- 调整 Gateway REST endpoint 仍是高级配置，错误地址会直接返回失败，不会被系统猜测或修复。
- Gateway API 原始响应体的敏感日志审计按用户决定不纳入本任务，需要时应单独定义日志保留和脱敏策略。

## Conclusion

本任务的已接受 Plan 已完成并通过自动化验证与开发 PostgreSQL 实测迁移。Traefik REST 请求失败现在以安全、可读的执行位置和 endpoint 返回，不改变 endpoint 选择、重试策略、dashboard Route 删除和接管数据迁移交付。
