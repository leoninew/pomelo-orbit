# API 盘点验证
最后修改时间: 2026-07-01 20:04:35

## Review status

Draft

## Requirement alignment

基于 `docs/requirement/20260701-api-inventory.md` 核对：

- 已按当前代码库后端 HTTP API 的 Req / Resp 规范化方向实施；实际 diff 不只是静态盘点文档，还包含后端接口契约和前端调用类型同步修改。
- 已补充命名 response / request DTO，并将多个直接数组响应改为 `{ items: [...] }` 形态。
- 已将大量匿名 `map[string]string{"detail": ...}` 错误响应收敛为 `transportresponse.ErrorResp[string]` 及 `transportresponse.Error` helper。
- 已同步前端 API 封装和视图消费方式，使前端读取 `ListResp<T>.items`，并补充通用 `ListResp<T>` / `ErrorResp<T>` 类型。
- 与原始 requirement 的 Non-goal 存在范围扩展：requirement 原先写明“不修改产品代码、路由行为、请求/响应结构或鉴权逻辑”，但后续 spec 和实现阶段已根据用户要求进入“修改 Req/Resp 契约”方向，因此本验证按已接受的 spec 作为主要实现依据。

## Spec alignment

基于 `docs/spec/20260701-api-inventory.md` 核对：

- 直接数组响应：已新增通用 `transportresponse.ListResp[T]`，并在角色权限、项目列表/成员、CI webhook/artifact/template variable、CD config file/route/compose service/service config 等接口中使用 `{ items: [...] }` 包装。
- 匿名成功响应：已新增或迁移多个 handler 层 DTO，例如 `HealthResp`、`ApplicationDeployResp`、`DeploymentLogsResp`、`DeploymentContainerLogsResp`、`ApplicationComposePreviewResp`、`ApplicationFileContentResp`、`DeploymentActionResp`、`ApplicationStatusResp`、`ApplicationLogsResp`、`MessageResp` 等。
- handler DTO 与 service/repository/model 隔离：已新增 `handler/*/dto.go`，并对 settings、task、cd traefik route、service config、pipeline snapshot、pipeline run variable 等路径建立 HTTP 层 DTO / 转换函数。
- 请求体命名：已新增或迁移 `*Req`，例如 auth、project、role、settings、task、CD/CI DTO；特殊 raw webhook、multipart 证书、SSE 不强行套普通 JSON Req/Resp。
- 错误响应：已新增 `transportresponse.ErrorResp[T]` 和 `transportresponse.Error`，非测试 HTTP 代码中大量错误路径已改为统一 helper。
- 前端对齐：已新增 `web/src/types/common.ts` 中的 `ListResp<T>` / `ErrorResp<T>`，并更新 API 返回类型和视图读取方式。
- 特殊接口：SSE 前端 `streamLogs` 封装和相关 feature flag 已移除；后端 SSE 是否仍保持特殊协议例外需在后续人工 review 中确认是否符合产品预期。

## Plan alignment

不适用。用户此前明确要求无须 Plan，按标准模式直接开始实现；本次验证按 accepted requirement + accepted spec 核对。

## Actual diff summary

当前暂存 diff 概览：

- 新增/更新 SpecFlow 文档：`docs/requirement/20260701-api-inventory.md`、`docs/spec/20260701-api-inventory.md`、`docs/verification/20260701-api-inventory.md`。
- 后端新增通用 HTTP DTO / response helper：`internal/transport/http/dto.go`、`internal/transport/http/response/response.go`。
- 后端按 handler 拆分 DTO：`internal/transport/http/handler/{auth,authz,cd,ci,project,role,settings,task,user}/dto.go`。
- 后端 handler 改为返回命名 `Resp`、`ListResp<T>`、`ErrorResp<T>`，并增加 service/model 到 HTTP DTO 的转换。
- 后端测试更新预期 JSON shape，覆盖 list response、错误 response、action response 和 DTO shape 变化。
- 前端 API/types/views/stores 更新为消费 `{ items }`、命名 response 和通用错误响应。
- `scripts/docker-compose.yml` 也在当前暂存 diff 中被修改，内容涉及 Traefik host/port 暴露配置；该文件未列入 api-inventory spec affected components，属于本次验证发现的范围外改动。

## Expected vs actual changed files

### 预期内

- `internal/transport/http/**/*.go`
- `internal/transport/http/**/*_test.go`
- `web/src/api/**/*.ts`
- `web/src/types/**/*.ts`
- `web/src/utils/request.ts`
- 受 API shape 影响的 `web/src/views/**/*.vue` / store 文件
- `docs/requirement/20260701-api-inventory.md`
- `docs/spec/20260701-api-inventory.md`
- `docs/verification/20260701-api-inventory.md`

### 范围外或需人工确认

- `scripts/docker-compose.yml`：修改了端口暴露和 Traefik host rule，未在 requirement/spec 中列为本次 API Req/Resp 契约规范化范围。
- `web/src/config.ts`、`web/src/env.d.ts`：移除 SSE feature flag，与前端 API 调用清理有关，但需要确认是否属于本次契约对齐范围，或是否应拆到部署/运行时配置相关变更。

## Acceptance criteria checklist

- [x] 后端接口请求/响应形态已有集中盘点和 checklist，见 spec 文档。
- [x] 直接数组响应已按 `{ items: [] }` 方向实施，并同步前端读取方式。
- [x] 多个匿名 map 成功响应已改为命名 `Resp`。
- [x] 错误响应已收敛为通用 `ErrorResp[T]` / `transportresponse.Error` helper。
- [x] 直接暴露 service/repository/model 的部分接口已新增 handler DTO 和转换。
- [x] 前端 API 类型、调用封装和视图消费已同步主要 JSON shape 变化。
- [x] 已运行后端格式化、vet、测试。
- [x] 已运行前端 lint 自动修复和 typecheck。
- [ ] 当前 diff 中存在 `scripts/docker-compose.yml` 范围外改动，建议提交前人工确认是否保留在同一提交。

## Command results

| Command | Result | Notes |
| --- | --- | --- |
| `go fmt ./cmd/... ./internal/...` | Passed | 无输出；执行后无 unstaged diff。 |
| `go vet ./cmd/... ./internal/...` | Passed | 无输出。 |
| `go test ./cmd/... ./internal/...` | Passed | 所有 Go package 通过；部分 package 无 test files。 |
| `yarn --cwd web lint:fix` | Passed | `eslint . --fix --cache` 通过；仅有 package license warning。 |
| `yarn --cwd web typecheck` | Passed | `vue-tsc --noEmit` 通过；仅有 package license warning。 |
| `git diff --cached --check` | Passed | 未发现暂存 diff whitespace error。 |

## Missed or expanded scope

- Expanded：本次实现从“API 盘点文档”扩大为“后端 API Req/Resp 契约规范化 + 前端契约同步”。该扩大范围来自 accepted spec 和用户后续要求，但与最初 requirement non-goal 文字存在历史冲突。
- Expanded：`scripts/docker-compose.yml` 的 Traefik host/port 修改不属于 API Req/Resp 规范化，建议提交前决定是否拆分。
- Potentially missed：未做运行时浏览器手工验证，未启动/停止开发服务器，符合项目约束“不主动管理开发服务器”。
- Potentially missed：未逐个调用所有 HTTP API，只通过单元/路由测试、Go vet/test 和前端 typecheck 验证静态与测试覆盖。

## Risks

- 直接数组响应改为 `{ items }` 是破坏性契约变化；虽然前端已同步，但外部调用方如存在仍需迁移。
- 错误响应统一为 `{ detail: string }` 的泛型结构后，前端统一错误处理已适配，但依赖旧匿名 map 的调用方需确认。
- 部分 DTO 转换由内部 model/service view 映射到 handler DTO，字段遗漏风险需要 code review 继续关注。
- SSE 相关前端封装被移除，若 UI 仍期望实时日志，需要确认当前轮询/日志读取策略是否满足产品体验。
- `scripts/docker-compose.yml` 的部署配置改动可能影响线上暴露域名和端口，不应在未确认时混入纯 API 契约提交。

## Incomplete items

- 人工确认是否保留或拆分 `scripts/docker-compose.yml` 改动。
- 人工确认移除 SSE feature flag / `streamLogs` 封装是否符合当前产品要求。
- 如存在外部 API 调用方，需要另行通知或补充迁移说明；当前验证只覆盖仓库内前端和后端测试。

## Conclusion

本轮 API Req/Resp 契约规范化在静态检查、后端测试和前端 typecheck/lint 层面通过。主要验收目标已满足，但当前暂存 diff 包含至少一个范围外部署配置文件变更，建议提交前先确认是否拆分；同时对 SSE 前端封装移除做一次产品行为确认。