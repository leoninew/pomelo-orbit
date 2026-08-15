# 部署记录与流水线运行记录删除 - 验证
最后修改时间: 2026-08-15 13:33:34

Review status: Accepted

## Requirement alignment

按 [需求](../requirement/20260815-execution-record-deletion.md) 核对：

- 部署记录与 PipelineRun 均提供单数 DELETE API，入口通过当前用户身份和项目成员关系校验。
- 仅 `ran_to_completion`、`faulted`、`canceled` 可删除；前端仅为终态展示删除入口，后端保留状态校验。
- 部署仅删除本次 `.log` 文件；PipelineRun 仅删除 `data/pipeline/runs/<run-id>/`，并在事务内删除 Artifact、Stage Run、Version Binding 和 Run。
- 关联服务、日志文件或运行目录缺失时写 warning 后继续删库；其他文件系统错误会阻止数据库删除。
- 列表和详情页均使用既有破坏性确认弹窗；列表删除后重载并处理末页，详情删除后返回对应列表。

## Actual Diff

暂存区包含 36 个文件、958 行新增、6 行删除，涵盖 HTTP 路由与处理器、应用服务、Repository/SQLC、运行工作区、前端 API 与四个页面、双语文案及专用测试。

`git diff --cached --check` 通过。与本功能无关的进行中改动未暂存；混合文件仅纳入记录删除对应 hunk。

## Expected vs Actual

| 预期范围 | 实际结果 |
| --- | --- |
| 部署记录删除及日志清理 | 已实现 |
| PipelineRun 删除及运行目录清理 | 已实现 |
| 内部关联记录事务删除 | 已实现 |
| 列表与详情页确认交互 | 已实现 |
| 缺失目标 warning 与错误阻断 | 已实现并覆盖测试 |
| Version 与仓库工作区保留 | 已实现并覆盖工作区边界测试 |

## Acceptance Checklist

- [x] 两个 DELETE API 均执行成员鉴权。
- [x] 仅三种终态允许删除。
- [x] 缺失文件或目录不阻断删除，其他清理错误阻断数据库删除。
- [x] PipelineRun 不触及 Repository 工作区或已生成 Version。
- [x] 列表与详情页均有删除确认交互和成功后的跳转/刷新。

## Validation Results

| 命令 | 结果 |
| --- | --- |
| `git diff --cached --check` | 通过 |
| `yarn --cwd web lint` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 通过，11 个测试文件、65 个用例 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |

实现阶段已执行 `yarn --cwd web lint:fix`、`go fmt ./cmd/... ./internal/...`；验证阶段未运行会写入工作区的格式化命令。

## Scope and Risk

light 模式无独立 Spec/Plan，按 Requirement 核对。验证命令在包含其他进行中工作的当前工作区执行，但所有相关改动均已单独暂存，且工作区未被还原、重置或覆盖。

没有已知未完成项。

## Conclusion

暂存的记录删除变更满足需求与验收标准，验证通过。
