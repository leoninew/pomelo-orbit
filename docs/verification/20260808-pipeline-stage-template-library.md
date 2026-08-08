# 可复用流水线阶段验证
最后修改时间: 2026-08-08 17:51:09

Review status: Draft

流程模式: 严格 / strict

## Requirement alignment

- 项目级 `PipelineStage(kind=template)` 阶段库、Template Pipeline 引用和 Application Pipeline 私有阶段已分离实现。
- 模板阶段与 Template Pipeline 引用保存无 `component_name` 的制品声明；Application Pipeline 在实例化时才添加 Docker 制品到 Component 的私有绑定。
- 阶段详情使用 Monaco 展示脚本，编辑使用抽屉；创建阶段仅要求名称、执行镜像和说明。
- SQLite/MySQL 独立种子 SQL 提供 `git clone`、`docker build` 模板及其模板流水线 DAG；开发库已执行 SQLite 脚本并验证幂等。

## Spec Alignment

- 实现与活文档 [CI Pipeline 设计](../guides/ci-pipeline-design.md) 和 [CD 领域模型](../product/cd-model.md) 一致。
- 既有 Requirement、Spec、Plan 仍含“模板阶段和引用不保存制品”的旧设计。按用户指令未回写，后续与旧过程文档一并归档。

## Plan Alignment

- 已完成 DDL/schema、SQLC、Repository、Usecase、HTTP/Proto、前端阶段库与 Pipeline 引入路径的切换，并生成 Go/TypeScript 代码。
- 计划中关于模板制品边界的旧描述不再适用；实际边界以本记录和活文档为准。

## Actual Diff Summary

- `pipeline_stage` 支持项目级模板和应用私有阶段；`pipeline_stage_reference` 保存 Template Pipeline 的冻结节点、制品与 DAG。
- 新增阶段库 API、单数路由、列表/详情页、CI 导航、模板更新预览和应用路径。
- 模板阶段详情统一使用 Monaco 内框；阶段脚本、制品声明和复制操作按历史构建阶段交互恢复。
- 新增 `scripts/seed-pipeline-stage-templates.{sqlite,mysql}.sql`，以幂等方式写入两个构建模板及“镜像构建流水线”。
- 更新活文档 `docs/product/overview.md`，说明项目级阶段库、引用快照与实例化绑定边界。

## Expected And Actual Files

- 预期的 Pipeline 模型、持久化、服务、HTTP/Proto、生成物、前端、迁移和测试文件均有对应改动。
- 本次验证额外补齐 `web/src/navigation.test.ts` 的“阶段”导航断言；修复创建阶段请求遗漏 `artifacts: []` 和添加制品按钮的事件参数类型。
- 工作区还存在 Credential、Role、User、Application 详情等并发会话改动；未将其纳入本功能的实现或验证结论。

## Acceptance Checklist

- [x] 模板阶段按 Project 隔离、版本化，并保存无组件映射的制品声明。
- [x] Template Pipeline 引用冻结模板定义、制品、DAG 与排序；实例化为 Application Pipeline 时复制定义并重映射依赖。
- [x] Application Pipeline 仅在实例化时配置 Docker 制品到 Component 的绑定与来源 Version 策略。
- [x] CI 导航、阶段列表、详情、抽屉脚本编辑、制品管理与模板更新接口已接入。
- [x] `git clone`、`docker build` 和“镜像构建流水线”可由独立 SQLite/MySQL 种子 SQL 重建。
- [x] 开发库验证模板、Pipeline、两条引用、制品 JSON 与 DAG 依赖；重复执行种子 SQL 不产生重复行。

## Test Results

| 检查 | 结果 |
|---|---|
| `task sqlc` | 通过 |
| `task proto` | 通过；无外部依赖可更新提示 |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test -count=1 ./cmd/... ./internal/...` | 通过 |
| `yarn --cwd web lint:fix` | 无错误；3 个并发文件有未使用导入警告 |
| `yarn --cwd web test` | 通过，10 个测试文件、63 个测试 |
| `yarn --cwd web typecheck` | 未通过：并发改动的 Credential/Role/User 文件各有一个未使用导入；本功能的两个类型错误已修复 |
| SQLite 迁移测试 | `TestMigrateUpSQLiteCreatesPipelineSchema` 通过 |
| SQLite 种子脚本 | 通过；重复执行返回 `0 row(s) affected` |
| `git diff --check` | 未通过：并发详情页改造引入的尾随空白；阶段详情受该改造触及，迁移、种子 SQL、活文档与本功能暂存差异均通过该检查 |

## Risks And Incomplete Items

- 前端全量 typecheck 仍被三个并发修改文件阻断，待其所属会话清理未使用导入后重新执行。
- 工作区全量 whitespace 检查仍被并发详情页改造阻断；不对其他会话的文件做格式化或暂存操作。
- 既有 Requirement、Spec、Plan 过程文档与当前制品边界不一致；按用户决定不回写，归档前不应继续作为实现依据。
- 未启动或重启开发服务器；未执行真实 Docker 构建或完整 Pipeline 运行。

## Conclusion

后端、迁移、种子 SQL 与前端自动化测试已通过。除并发文件导致的全局前端类型检查失败外，本功能的已知验证项通过；等待并发改动收敛及用户审阅后确认验证结论。
