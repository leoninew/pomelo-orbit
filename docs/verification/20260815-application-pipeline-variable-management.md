# 应用流水线变量管理与直接运行验证
最后修改时间: 2026-08-15 15:35:23

Review status: Accepted

Mode: standard

## 验证范围

验证对象为暂存区中“应用流水线变量管理与直接运行”相关的 20 个文件。当前暂存区还包含此前的面包屑和标题调整，本记录不将该组变更纳入本 feature 的验收结论。

验证依据：

- [Requirement](../requirement/20260815-application-pipeline-variable-management.md)
- [Spec](../spec/20260815-application-pipeline-variable-management.md)
- [Plan](../plan/20260815-application-pipeline-variable-management.md)

## Requirement 与 Spec 对齐

- Application Pipeline 详情变量列表已由运行时 Preview 改为持久化管理视图：包含 `pipeline_custom`、可覆盖的 `pipeline_stage` 和四项只读内置变量；Repository 变量不进入该列表。
- 变量解析顺序为内部 Retry overrides、Repository、Pipeline、Stage 默认值。Repository 保持优先级最高；直接 Trigger 没有 HTTP 用户变量输入。
- 直接 Trigger 的请求 Proto 为空消息，前端直接 POST 空对象；Preview 路由、DTO、Proto 类型、前端 API 与调用点均已删除。
- 解析后的非系统变量没有有效值时，会在创建 Snapshot 前返回 validation error。`repository_ref` 从 Repository 的 `default_branch` 注入，Retry 仍可在内部使用历史快照 overrides。
- Template Pipeline 仍走既有变量列表解析，不获得 Application Pipeline 的阶段变量覆盖操作。

## Plan 与实际改动对比

实际暂存改动为 20 个文件，合计 739 行新增、605 行删除：

- 变量规则和测试：`runtime.go` 将详情管理列表与运行时解析分开，并增加变量名、来源、重复项与最终值校验；`runtime_test.go` 覆盖内置只读、阶段回退、Repository 优先级、缺失最终值和 Retry overrides。
- Pipeline detail 用例：`pipeline.go` 依 Pipeline 类型分别返回 Application 管理列表和 Template 既有列表。
- Pipeline Run：移除 Preview DTO/路由/Handler 行为，Trigger 不再接收变量；新增 `direct_trigger_test.go` 验证缺失变量不会创建 Snapshot。
- Proto 与前端 API：Trigger 请求改为空消息，删除 Preview 生成类型和客户端调用。
- Pipeline 详情 UI：删除运行弹窗、表单、Preview 状态和 watchers；新增阶段变量“覆盖”入口，运行按钮直接创建 Run。
- 文档：更新 CI 变量和 Pipeline 设计指南，并新增本 feature 的 Requirement、Spec 和 Plan。

计划中列出的 `internal/application/pipeline/usecase/pipeline_test.go` 未随本 feature 暂存。该文件当前含有另一项尚未暂存的阶段模板工作，未为凑齐范围而混入；本 feature 的规则与直接 Trigger 行为由上述已暂存测试覆盖。规格中提及的 `internal/api/http/handler/pipeline/mapper.go` 无需修改，既有 mapper 可直接承载更新后的 usecase 输出。

## 验收清单

- [x] Application Pipeline 显示变量管理列表，Template Pipeline 保留原变量配置列表且不显示阶段覆盖操作。
- [x] 可新增、编辑、删除 `pipeline_custom`；后端验证变量名、内置变量、来源与重复名称。
- [x] 从阶段 script、制品 `reference`、`name`、`command` 提取的 `pipeline_stage` 变量可通过 UI 覆盖为 `pipeline_custom`。
- [x] 删除同名 Pipeline 配置后恢复阶段来源及其 Liquid `default`，有规则单测覆盖。
- [x] Repository 变量不出现在管理列表，且运行解析保持 Repository > Pipeline > Stage default，有规则单测覆盖。
- [x] `repository_ref` 固定取绑定 Repository 默认分支；四项内置变量在管理列表只读。
- [x] Application Pipeline 运行按钮直接 Trigger，不再打开变量弹窗，也没有 Preview API 调用。
- [x] 非系统变量最终无值时拒绝创建 Run，且缺失变量测试确认不会创建 Snapshot。
- [x] Run 变量快照仍由服务端解析后写入；执行和 Retry 继续消费快照。
- [~] Pipeline 配置保存后的版本递增沿用既有 `UpdatePipeline` 路径；本次未暂存该路径的专门回归测试，见风险项。

## 检查结果

| 命令 | 结果 |
|---|---|
| `git diff --cached --check` | 通过 |
| `git grep --cached -n -E 'PreviewPipelineRunVariables|PipelineRunVariablePreview|previewVariables|variable-preview|PipelineRunTriggerInput' -- internal web proto` | 无匹配，确认已移除过时 Preview 引用 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web lint` | 通过 |
| `yarn --cwd web test` | 11 个文件、65 项测试全部通过 |
| `go vet ./internal/application/pipeline/... ./internal/application/pipeline_run/... ./internal/api/http/handler/pipeline_run/...` | 通过 |
| `go test -count=1 ./internal/application/pipeline/rule/pipelinevariable ./internal/application/pipeline/usecase ./internal/application/pipeline_run/usecase` | 通过 |
| `go test ./internal/application/pipeline/... ./internal/application/pipeline_run/... ./internal/api/http/handler/pipeline_run/...` | 通过 |

## 范围偏差

- 无产品行为的扩展或遗漏。
- 为避免影响存在大量在途改动的工作树，未运行会写入文件的 `yarn --cwd web lint:fix`、`go fmt ./cmd/... ./internal/...` 与 `task proto`。生成的 Go/TypeScript Proto 已参与编译、类型检查和测试。
- 没有启动、停止或重启开发服务器；本次未进行浏览器端到端手工操作验证。

## 风险与未完成项

- 验证命令针对当前工作树执行，其中包含未暂存的其他功能改动；暂存差异范围已通过 index 单独核对，但运行结果并非隔离 worktree 的纯暂存快照。
- 缺少 Pipeline detail UI 的专门组件/端到端测试，阶段“覆盖”、保存后刷新与直接跳转 Run 详情依赖静态审查、类型检查和 API/usecase 测试确认。
- 缺少本 feature 专门验证“变量保存使 Pipeline 版本递增”的新增测试；该行为沿用既有更新用例路径。

## 结论

暂存的应用流水线变量管理与直接运行改动与已接受的 Requirement、Spec 和 Plan 一致，关键规则、直接 Trigger、移除 Preview 和前端编译测试均通过，验证结论为 Accepted。上述未运行的写入型检查和 UI 端到端覆盖属于后续加强项，不构成当前交付阻断。
