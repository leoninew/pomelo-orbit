# backend-go CI 快照获取或创建修复需求
最后修改时间: 2026-06-18 16:13:52

Review status: Accepted

## Background

从仓库详情页触发流水线时，请求体已经正确传入 `template_id`，例如：

```json
{"template_id":"01KP0K6W1YW73REFQE8YAVTPMN","trigger_ref":"master","variables":{}}
```

backend-go 当前触发路径会先读取 `pipeline_template`，再调用 `LatestPipelineSnapshot(templateId)` 获取该模板的最新 `pipeline_snapshot`。当模板存在但尚未创建 snapshot 时，后端直接返回 404：

```text
Pipeline snapshot for template 01KP0K6W1YW73REFQE8YAVTPMN not found
```

这与 Python backend 的行为不一致。Python backend 在创建 run 时会先解析模板完整变量声明，然后通过 `SnapshotManager.get_or_create_snapshot` 获取或创建 snapshot：如果最新 snapshot 版本等于模板版本则复用，否则基于当前模板和阶段创建新 snapshot。

## Goal

修正 backend-go CI 触发流水线时缺少 snapshot 即失败的行为，补齐最小 get-or-create snapshot 逻辑：

- 仓库详情手动触发流水线时，如果模板存在但没有 `pipeline_snapshot`，应自动基于当前模板创建 snapshot 并继续创建 run。
- webhook 触发流水线时使用相同 get-or-create snapshot 逻辑。
- 如果已有最新 snapshot 且版本等于当前模板版本，应复用该 snapshot。
- 如果已有 snapshot 但版本落后于当前模板版本，应基于当前模板创建新版本 snapshot。
- 新建 snapshot 应包含可执行阶段快照和完整变量声明快照，供 runtime variables 合并与执行阶段使用。

## Non-goal

- 不修改前端表单逻辑；当前问题不是前端未传 `template_id`。
- 不通过仅向 migration seed 插入固定 `pipeline_snapshot` 来解决根因。
- 不重构整个 CI snapshot/versioning 架构为 Python backend 的完整 DDD 实现。
- 不改变 retry 的核心语义；retry 应继续优先使用原 run 的 snapshot，避免重试时受当前模板变化影响。
- 不处理已存在数据库中历史 snapshot 的批量补数据策略。

## User scenarios

1. 用户在仓库详情页选择“通用容器镜像流水线”并触发；该模板存在但数据库没有 snapshot，后端应创建 snapshot 并成功创建 `PipelineRun`。
2. 用户通过 webhook 触发同一模板；如果 snapshot 不存在或版本落后，应自动创建当前模板版本的 snapshot。
3. 用户编辑模板导致模板版本增加；下一次触发应创建新版本 snapshot，而不是复用旧版本 snapshot。
4. 用户 retry 旧 run；系统应继续使用旧 run 关联的 snapshot，而不是因为模板已更新而改变 retry 的执行内容。

## Acceptance

- backend-go service 层提供最小 `getOrCreatePipelineSnapshot` 行为，对齐 Python `SnapshotManager.get_or_create_snapshot` 的核心语义。
- `TriggerRepository` 不再因模板存在但 snapshot 缺失而返回 `Pipeline snapshot for template ... not found`。
- `ReceiveRepositoryWebhook` 使用同样的 snapshot 获取或创建逻辑。
- 新建 snapshot 的 `stages_snapshot` 来自当前模板关联阶段和对应 build stages，按模板阶段顺序组织。
- 新建 snapshot 的 `variables_snapshot` 包含完整变量声明，包括内置变量、模板自定义变量、stage 中解析出的变量声明。
- 存在同版本 snapshot 时复用；模板版本变化时创建新 snapshot。
- backend-go 创建 snapshot 时，如果模板编排引用的 build stage 不存在，应返回错误，不静默创建不完整 snapshot。
- 如果创建 snapshot 时遇到 `(template_id, version)` 唯一冲突，应重新读取 latest snapshot 并复用同版本 snapshot，而不是直接把冲突暴露为触发失败。
- 新增或更新 backend-go 测试覆盖：无 snapshot 时手动触发可创建 run；已有同版本 snapshot 时复用；至少覆盖 webhook 或版本变化中的一个关键路径。
- backend-go 相关测试通过。

## Open questions

- 无。

## Decisions

- 采用 service 层 get-or-create snapshot 修复根因，不用前端绕过，也不只靠 migration seed snapshot。
- retry 路径保持使用原 run snapshot，不纳入本次 get-or-create 改动。
- 新建 snapshot 的变量声明复用当前 backend-go 已有变量解析能力，避免重复实现另一套变量提取规则。
- backend-go 创建 snapshot 时，如果模板编排引用的 build stage 不存在，应返回错误；这是有意比 Python backend 静默跳过缺失 stage 更严格，符合本项目 assertions-first 原则。
- Python backend 查 latest snapshot 时使用 `(project_id, template_id)`；backend-go 当前已有接口只按 `template_id` 查询。由于 template id 全局唯一，本次 light 修复不扩大为接口签名重构，但新建 snapshot 必须写入 `template.ProjectId`。

## Risk

- snapshot 创建需要从 `pipeline_template_stage` 和 `build_stage` 组装 `StageDefinition`，必须保持阶段顺序、依赖关系、脚本、镜像和 artifacts 一致。
- 并发触发同一模板且 snapshot 不存在时，可能同时尝试创建相同版本 snapshot；实现需要考虑唯一约束冲突后的复用或错误处理，避免用户偶发 500。
- backend-go 当前 latest snapshot 查询未带 `project_id` 条件，本次基于 template id 全局唯一假设不扩大修改范围；如后续需要更强多租户隔离，可单独收敛查询接口。
- 本次修复会继续叠加在当前未提交的 runtime variables 修复之上，验证时需要区分本次新增范围与前序变更范围。
