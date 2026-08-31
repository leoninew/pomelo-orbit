# Component Policy Governance Verification
最后修改时间: 2026-08-31 12:35:19

Review status: Accepted

## Requirement alignment

按 Requirement 核对 `pull_policy` 和 `restart_policy` 的枚举、必填校验、Service overlay、Compose 输出及 seed 原位修正。无新增 spec/plan，按 light 模式直接验证实现。

## Spec/Plan

不适用（light 模式）。

## Actual diff summary

- 应用层增加 Version Component 重启策略非空及四值校验，扩展 Service overlay 校验。
- Compose 渲染支持 `on-failure`、`always`、`unless-stopped`，`no` 不输出 `restart`。
- 前端创建/编辑表单默认和校验覆盖两种策略集合。
- MCP 创建工具继续使用生成的 `applicationv1.VersionComponentReq`，mapper 直接传递策略字段；创建和更新工具要求策略字段，应用层负责非空及枚举校验。
- 三种数据库的既有 Gateway seed 直接写入 `unless-stopped`；未新增 `000038` 或其他迁移。
- 本地开发 SQLite 中 5 条 `version_component.restart_policy IS NULL` 记录已原位更新为 `unless-stopped`；Service overlay 的可空继承数据未修改。
- `/api/version/:version_id` 的 `versionResponse -> versionComponentResponse` 已映射 `RestartPolicy`。本次缺失响应来自上述本地存量 `NULL`，而非 HTTP/Proto 响应字段遗漏。
- 补充后端、MCP、前端测试及 MCP 操作指南。

## Expected vs actual

预期是新增数据不再产生空值或非法策略，前端创建提供默认、MCP 创建显式提交策略，运行时和 Service overlay 接受完整枚举，历史 seed 通过原位修改修正。实际 diff 与该范围一致，没有新增数据库约束或迁移。

## Acceptance checklist

- [x] `pull_policy` 应用层限定为 `missing`、`always`、`never`。
- [x] `restart_policy` 应用层限定为 `no`、`on-failure`、`always`、`unless-stopped`，且不可为空。
- [x] 前端创建默认 `missing` / `unless-stopped`，字段级校验生效。
- [x] MCP 创建和 basic 更新要求显式 `pull_policy` 和 `restart_policy`，由应用层执行常规校验。
- [x] Service overlay 保留 `nil` 继承并支持四种重启策略。
- [x] Compose 输出策略符合约定。
- [x] Gateway seed 三种方言已原位更新。
- [x] `/api/version/:version_id` 的 Component 响应映射包含 `restart_policy`；本地存量数据无空值。

## Test results

- `go test ./internal/api/mcp/delivery ./internal/application/application/usecase ./internal/application/deployment/usecase ./internal/application/service/usecase`：通过。
- `go test ./internal/api/mcp/delivery -run 'TestCreateVersionComponentPassesPolicies|TestToolListIncludesDeliverySurfaceAndFlatCollectionSchemas' -count=1`：通过。
- `go test ./cmd/... ./internal/...`：通过。
- `task check`：通过（前端 typecheck、lint、format，golangci-lint）。
- `yarn lint:fix`：通过。
- `yarn typecheck`：通过。
- `yarn test`：通过（16 个文件、78 个测试）。
- `go test ./internal/api/http/handler/application ./internal/api/http/codec`：通过。
- `dbtalk database query`：本地 SQLite 中 `version_component.restart_policy` 分布为 `unless-stopped=34`，缺失数量为 0。
- `git diff --check`：通过。

## Missed or expanded scope

- 未修改已执行迁移，也未新增数据库约束，符合用户明确范围；本地开发 SQLite 的存量 Version Component 已完成原位数据修复。
- MCP 请求类型恢复为仓库已有的 protobuf 请求类型；删除了自定义 `mcpVersionComponentReq` 和 `UnmarshalJSON` 方案。

## Risks

运行中数据库的既有 Service overlay CHECK 约束未通过迁移调整；本次只修复了本地开发 SQLite 的 Version Component 空值，部署环境仍需确认其存量数据和策略约束。

## Incomplete items

暂无。

## Conclusion

实现已按 Requirement 完成，全量检查通过，可交付。数据库约束保持未新增，历史 seed 已原位修正。
