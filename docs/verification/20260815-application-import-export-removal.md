# 移除应用导入导出验证
最后修改时间: 2026-08-15 17:41:00

Review status: Accepted

## 需求对齐

按 `docs/requirement/20260815-application-import-export-removal.md` 核对。本任务采用轻量模式 / light；无独立 Spec 或 Plan，按已接受的 Requirement 验证。

- 应用列表删除 JSON 文件导入控件、导入对话框和相关表单逻辑。
- 应用详情删除导出命令与下载逻辑。
- HTTP 路由不再注册 `POST /api/application/import` 或 `GET /api/application/:app_id/export`。
- 应用层 DTO、用例、Proto 和 Go/TypeScript 生成文件均删除 `application_bundle`、`ApplicationImportReq` 与 `ApplicationExportResp`。
- 关联中英文文案同步删除；凭据等其他领域的导入导出能力未改动。

## 实际差异

实现范围为 15 个暂存文件、58 行新增和 730 行删除，符合需求边界：

| 范围 | 实际文件 |
| --- | --- |
| 过程文档 | `docs/requirement/20260815-application-import-export-removal.md` |
| HTTP 与应用层 | Application handler、router、DTO 与 import/export usecase 删除 |
| 协议 | `application_bundle.proto` 及 Go/TypeScript 生成文件删除 |
| Web | Application API、列表页、详情页与中英文 locale |

`internal/application/application/usecase/version.go`、SQL 迁移、流水线、路由、服务和脚本等其他工作区改动未纳入本次暂存范围。

## 验收清单

- [x] 应用列表不再包含导入控件、文件选择或导入对话框。
- [x] 应用详情不再包含导出操作或浏览器下载逻辑。
- [x] 两个导入导出 HTTP 路由已删除，handler 不再依赖 Service runtime usecase。
- [x] `application_bundle.proto`、相关请求/响应类型、映射和应用用例已删除。
- [x] 暂存区 `internal/`、`web/`、`proto/` 不再匹配旧符号或 API 路径。
- [x] 后端格式化、静态检查和测试，以及前端 lint、类型检查和测试均通过。

## 命令结果

| 命令 | 结果 |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `go test -count=1 ./internal/api/http/handler/application ./internal/application/application/usecase ./internal/api/http/routes` | 通过 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 11 个测试文件、65 个用例全部通过 |
| `git diff --cached --check` | 通过 |
| `git grep --cached -n -E 'ApplicationImportReq|ApplicationExportResp|ImportApplication|ExportApplication|application_bundle|/api/application/import|/api/application/.*/export' -- internal web proto` | 无匹配 |

## 范围偏差与风险

未启动开发服务器或执行浏览器手工操作；列表与详情页入口移除由暂存 diff、前端类型检查和现有测试验证。外部客户端若仍调用已删除 API 会得到 404，这是 Requirement 明确接受的破坏性契约变更。

验证命令在包含其他未暂存变更的工作区执行，暂存范围已通过 index 单独核对；测试结果不是隔离 worktree 的纯暂存快照。

## 结论

需求验收项均满足，代码检查和测试通过。本变更可交付；本验证记录尚未加入暂存区，等待用户决定是否纳入同一提交。
