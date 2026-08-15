# 应用流水线变量管理与直接运行实施计划
最后修改时间: 2026-08-15 13:57:20

Review status: Accepted

Mode: standard

## Implementation steps

1. 调整 Pipeline 变量规则：详情列表保留只读内置项，并合并 Pipeline 自身与阶段声明；校验保存变量名、来源和重复项；运行解析改为按最终值完整性校验，并由服务端注入默认分支。
2. 调整 Pipeline detail 用例和测试：Application Pipeline 返回包含只读内置项的详情变量列表；Template Pipeline 保留其既有变量配置交互，不提供阶段覆盖；覆盖和删除回退只通过现有 `variable_declarations` 更新接口完成。
3. 删除运行时 Preview API：移除 Proto 请求/响应、HTTP 路由/Handler、DTO、前端 API 与所有调用；手动 Trigger 改为空请求。
4. 调整 Pipeline Run 用例和测试：直接 Trigger 不接收变量，Retry 继续在内部使用原 Run 快照构造 overrides；未解析出有效值时拒绝创建 Run。
5. 改造 Pipeline 详情页：Template Pipeline 保留原有变量配置，Application Pipeline 支持新增、编辑、删除和覆盖阶段变量；删除应用流水线运行弹窗、表单和 preview 状态，运行按钮直接触发。
6. 更新 CI Pipeline 活文档，生成 Proto，并运行前端 lint/typecheck 与 Go fmt/vet/test。

## Files to change

- `internal/application/pipeline/rule/pipelinevariable/runtime.go` 与测试
- `internal/application/pipeline/usecase/pipeline.go` 与测试
- `internal/application/pipeline_run/{dto,usecase}` 与测试
- `internal/api/http/{handler,pipeline_run routes}`
- `proto/orbit/v1/pipeline_run/pipeline_run.proto` 及生成物
- `web/src/api/pipeline_run/pipeline_run.ts`
- `web/src/views/pipeline/PipelineDetail.vue`
- `docs/guides/ci-pipeline-vars-design.md`、`docs/guides/ci-pipeline-design.md`

## Verification plan

- 变量规则单测覆盖内置变量只读显示、Pipeline 对阶段变量的覆盖、Repository 优先级、最终无值拒绝和默认分支注入。
- Pipeline Run 用例测试覆盖无请求变量的直接 Trigger 与 Retry 快照输入。
- 运行 `task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...` 与 `git diff --check`。

## Assumptions

- `pipeline_custom` 是 Pipeline 变量配置层的现有持久化来源标记；运行时解析结果仍使用 `pipeline`，但该结果不用于变量管理 UI。
- 历史 Run 可能含有手工提交的旧变量和非默认 `repository_ref`；Retry 仅在内部使用该历史快照，不对新的直接 Trigger 暴露该能力。

## Risk and rollback

- 该变更删除公开 Preview endpoint 和 Trigger 变量字段，前后端必须同版本发布。
- 如需回退，应整体回退本 feature 的 Proto、HTTP、usecase 与前端改动，不能只恢复前端运行弹窗。
