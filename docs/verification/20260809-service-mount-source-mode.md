# 服务挂载源路径模式验证
最后修改时间: 2026-08-09 17:04:50

Review status: Accepted

## Requirement alignment

与 [`20260809-service-mount-source-mode.md`](../requirement/20260809-service-mount-source-mode.md) 对齐：Version 挂载编辑页不展示宿主机路径模式，Service 覆盖同时保存源路径和该模式，保存错误留在当前模态内；有效运行配置错误以页面 Toast 展示而不归属到环境变量；没有新增数据库状态组合约束或异常兜底。

## Spec / Plan alignment

不适用。该任务采用 light 流程，依据 Requirement 实现。

## Actual diff summary

- Service mount overlay 协议、模型、SQLC 查询和 repository 增加可空 `source_is_host_path`。
- `000026_service` 的 SQLite/MySQL 原始建表定义直接包含可空 `source_is_host_path`；不新增后续迁移或历史数据回填。
- 服务业务校验根据目录/文件挂载的路径模式接受绝对宿主机路径，或拒绝错误的绝对逻辑路径；错误包含组件名和目标挂载。
- Version 挂载模态删除宿主机路径开关，保存失败显示在模态内；Service 组件页的挂载列表与 Version 使用相同的类型、源路径、目标路径、只读和操作列，并使用独立编辑模态。模态只编辑源路径和宿主机路径模式，沿用页面顶部的统一持久化命令。
- Service 详情页在首次加载及保存基本信息、环境变量后，根据响应内 `effective_error` 显示 8 秒 Toast；环境变量卡片不再接收或展示该通用错误。
- 更新产品模型说明、映射和业务校验测试；迁移版本保持为 30。

## Expected vs actual files

- 预期：Service 协议、模型、用例、repository、SQL、前端 Service/Version 页面、测试和活文档。
- 实际：上述范围均有修改；生成的 proto 与 SQLC 文件由 `task proto`、`task sqlc` 更新。
- 工作区还有先前已暂存的 Version preview 相关 `effective_service_plan.go` 与测试改动；本次未还原或拆分这些已有改动。

## Acceptance

- [x] 服务页目录/文件挂载可保存宿主机绝对路径和来源模式。
- [x] 未选择宿主机模式的绝对路径由服务业务校验明确拒绝。
- [x] Version 挂载编辑页不展示宿主机路径开关，异步错误留在当前模态。
- [x] Service 挂载不再在表格单元格内联编辑；列表和编辑模态展示类型、源路径、目标路径、只读字段，并提供源路径字段级必填反馈。
- [x] 原始建表定义不增加状态组合 `CHECK` 约束，也不为缺失声明写默认模式。
- [x] HTTP 映射和有效部署计划保留服务覆盖的路径模式。
- [x] 有效运行配置错误以页面 Toast 表达，不再显示在环境变量区域；Toast 显式展示 8 秒。

## Test results

- `yarn --cwd web lint:fix` 通过。
- `yarn --cwd web typecheck` 通过。
- `yarn --cwd web test` 通过，10 个测试文件、63 个测试。
- 本轮再次执行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 与 `yarn --cwd web test`，均通过（10 个测试文件、63 个测试）。
- `task proto`、`task sqlc` 通过。
- `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...` 通过。
- 本轮执行 `go test ./internal/bootstrap -run TestMigrateAndMigrationVersion -count=1` 通过：从空 SQLite 数据库迁移至版本 30，并确认 `service_component_mount.source_is_host_path` 存在。
- 本轮完整复验 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 和 `yarn --cwd web test` 均通过；前端为 10 个测试文件、63 个测试。
- `git diff --check` 通过。

## Risks and incomplete items

- 未进行浏览器运行态验收：项目约束要求不由 Agent 启动或停止开发服务器。
- 本任务不提供已存在数据库的列补充或历史数据回填；按当前开发期约束，验证范围为从空库执行完整迁移。

## Conclusion

实现和静态、单元、集成迁移验证均通过；可由用户在已运行的开发环境中验收 Service 组件挂载编辑流程。
