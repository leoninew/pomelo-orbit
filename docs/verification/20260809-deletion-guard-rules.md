# 删除路径防御规则整理验证
最后修改时间: 2026-08-09 14:15:04

Flow mode: light

Review status: Accepted

## Requirement alignment

- 通用 Application 删除已移除 Version 运行时引用重复预检；公开删除用例仍在删除前分别检查 Service 与 Version。
- Credential 和 Repository 的删除前置条件均返回 `KindValidation`，并给出移除引用或取消/等待 Pipeline 的明确提示。
- Template 与 Application Pipeline 节点删除均直接查找依赖该节点的保留节点；提示同时包含目标和依赖节点名称。删除目标自身不参与该检查，避免其即将被删除的无关字段阻断操作。
- Gateway 部署不再调用 Docker network inspect 预检，预检实现与测试已移除。

## Spec and plan alignment

不适用。light 模式按 Requirement 直接实施。

## Actual diff summary

- 删除 `preflightGatewayNetwork` 的 Docker 输出、数量和 Compose 标签解析，以及其单元测试；`DeployService` 直接进入既有 Gateway 解析和部署创建路径。
- `DeleteApplication` 保持 Repository 内部的事务清理责任，移除只会与公开用例层规则重复的 `CountVersionRuntimeRefs` 分支。
- 以明确的 `400 validation` 替换 Credential/Repository 删除的专门 `409 conflict` 分支，并增加 Repository 删除提示的单元测试。
- Pipeline 节点删除不再将所有配置校验错误归因为依赖引用；直接依赖仍被明确拒绝并有单元测试。

## Expected and actual files

| 预期范围 | 实际文件 | 结果 |
|---|---|---|
| Requirement | `docs/requirement/20260809-deletion-guard-rules.md` | 一致 |
| Application 清理 | `internal/repository/impl/sqlc/application/repository.go`、`repository_test.go` | 一致 |
| Credential / Repository 提示 | 对应 usecase 与测试 | 一致 |
| Pipeline 依赖检查 | `stage_template.go`、`pipeline_test.go` | 一致 |
| Gateway 网络预检移除 | `command.go` 与已删除的预检实现/测试 | 一致 |

暂存区共 12 个文件，未发现与本任务无关的文件。

## Acceptance checklist

- [x] `DeleteApplication` 不再以 Version 运行时引用阻止其事务性清理。
- [x] Credential 与 Repository 删除前置条件使用 `KindValidation` 和可执行提示。
- [x] Pipeline 节点依赖完整性保持，且提示包含目标和依赖节点名称。
- [x] Gateway 部署不再存在 Docker network inspect 预检引用。
- [x] 后端格式化、静态检查和测试通过。

## Test results

| 命令 | 结果 |
|---|---|
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `git diff --check` | 通过 |
| `git diff --cached --check` | 通过 |

未启动开发服务器。

## Scope deviation

无。未修改迁移、前端、Gateway 删除语义、数据库错误转换或 Docker Compose 生命周期。

## Risk

- 已有不兼容 Docker 网络将在 Compose 执行时失败，诊断信息仍由既有 Deployment 任务日志提供；本次不额外解析或改写该错误。
- 未在实际 Docker 网络冲突环境中执行集成测试；该场景的预期变化正是取消预检并交由 Compose 执行路径处理。

## Incomplete items

无。

## Conclusion

实现与 Requirement 对齐，静态检查和全量后端测试均通过，可以进入提交审阅。
