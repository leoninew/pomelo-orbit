# CI 制品关联应用版本验证
最后修改时间: 2026-08-05 17:37:49

Review status: Draft

流程模式: 标准 / standard

## Requirement Alignment

- 构建阶段绑定 Application、`latest`/`fixed` 选择、来源 Version 冻结及删除引用约束已实现并有单元测试覆盖。
- 制品以 `artifact` 的 `value`、`value_format`、`image_ref`、`local_image_sha256` 与 `source_artifact_id` 保存；未引入 registry、部署动作或 tag 覆盖拦截。
- 构建成功后创建未发布 Version，目标 Component 更新镜像并保存 `artifact_id`；关联关系使用资源 ID。

## Spec Alignment

- SQLite 基线 schema 及 sqlc 查询使用合并后的 Artifact 模型；不存在 `artifact_value`、`artifact_input`、`container_image_artifact` 或 `version_component_artifact` 的运行时表和查询。
- `source_artifact_id` 自关联投影 `source_commit_sha`，Component 通过 `version_component.artifact_id` 关联 Artifact。
- `build-<runtime_datetime>`、单一上游 `git_object_id`、Repository 仅阻止 `running` Run、以及 Version 引用计数的实现和测试均在预期范围内。

## Plan Alignment

- 基线 SQLite/MySQL migration、seed、sqlc、领域模型、仓储、执行器、HTTP/Proto 与 Web 视图均在计划范围内变更。
- 开发库通过单独的 [迁移脚本](../../scripts/migrate-artifact-schema.sql) 原地迁移，不新增基线增量 migration。

## Actual Diff Summary

- PipelineStage/PipelineSnapshot 增加并冻结 BuildVersionBinding；触发和重试时解析并保存来源 Version。
- 制品 collector 使用完整 Artifact 写入；Docker image 记录 image ref、local SHA 与来源 commit Artifact。
- fork 在同一事务中创建未发布 Version、替换目标 Component 镜像、写入 Component `artifact_id` 并回写 Run binding。
- API、Proto、前端页面展示构建绑定、Artifact 详情和 Component 制品来源。
- 开发库已合并制品表：当前保留 5 条 command、3 条 docker_image 制品和 3 条 Component Artifact 关联。

## Expected And Actual Files

计划中的 SQL migration/query、Go model/repository/usecase、Proto/generated DTO、HTTP handler 与 Pipeline/Application/Artifact Web 页面均有对应变更。额外变更包括：

- `web/src/views/application/ApplicationPage.vue` 的 Application 删除交互和列表列布局修复；两者不属于制品与 Version 关联核心范围。
- `scripts/migrate-artifact-schema.sql`，用于已有 SQLite 开发库的一次性就地迁移。

## Acceptance Checklist

- [x] BuildVersionBinding 支持 `latest` 与 `fixed`，并校验来源 Application/Component。
- [x] `latest` 使用 `id DESC`，来源在 Run 创建时冻结。
- [x] command、file、docker_image collector 采用当前 schema 的完整 Artifact 载荷。
- [x] Docker Artifact 保存 tag、local image SHA、commit 来源及 Component 关联。
- [x] 成功的绑定构建 fork 未发布 Version，并仅替换目标 Component。
- [x] 不包含 registry、部署、MCP 写入或 tag 覆盖限制。
- [x] SQLite 基线 migration、当前开发库外键与 sqlc 再生通过。

## Test Results

| 命令或检查 | 结果 |
| --- | --- |
| `task check` | 通过：Vue typecheck、ESLint、Prettier、Go format 和 golangci-lint。 |
| `task test` | 通过：Vitest 10 文件、62 项；Go `./cmd/... ./internal/... ./sql`。 |
| `task sqlc` | 通过。 |
| 关键 Go 测试 | 通过：DAG source commit 约束、git object ID collector、build label、PipelineRun 执行、Application 引用删除、SQLite migration。 |
| 开发库结构检查 | 通过：仅保留 `artifact` 和 `version_component`；`PRAGMA foreign_key_check` 无结果。 |

## Scope Deviations

- Application 删除交互和 `/applications` 列布局修复为本轮工作区中的独立 UI 改动，不影响核心制品/版本行为。

## Risks And Incomplete Items

- 未在可用 MySQL 实例上执行基线 up/down migration；当前迁移自动测试覆盖 SQLite。
- 未执行真实 Docker build/inspect 的端到端运行；执行器相关单元测试使用测试 runner。
- 浏览器访问本地应用要求登录，未完成登录后手工验证 Pipeline Stage 绑定、Artifact 详情与 Component 来源页面。

## Conclusion

自动验证和开发库结构验证通过。核心需求与已接受规格、计划一致；上述三项环境相关的手工/集成验证仍待补齐。
