# 部署记录与流水线运行记录删除
最后修改时间: 2026-08-15 13:07:34

Review status: Accepted

## Background

部署记录和流水线运行记录持续积累。用户需要从列表和详情页清理已结束的记录，同时回收仅属于该次执行的日志和文件。

## Goal

- 部署记录列表与详情页支持删除已结束记录。
- PipelineRun 列表与详情页支持删除已结束记录。
- 删除时清理关联的执行日志与运行专属文件，并清理运行内部的数据库关联记录。

## Non-goal

- 不允许删除 `waiting_to_run` 或 `running` 的执行记录。
- 不删除部署 Service 共享工作目录、Compose 配置或 Repository 工作区。
- 不删除 PipelineRun 已生成 Version；Version 保留已持久化的镜像和制品展示快照。

## User scenarios

1. 用户在部署记录列表或详情页删除已完成、失败或取消的部署，系统删除记录及对应日志文件。
2. 用户在 PipelineRun 列表或详情页删除终态运行，系统删除 Run、阶段运行、运行绑定、制品记录及该 Run 专属目录。
3. 用户删除历史记录时，关联日志文件、文件目录或关联服务已不存在，系统写 warning 日志并继续删除数据库记录。

## Acceptance

- `DELETE /api/deployment/:deployment_id` 与 `DELETE /api/pipeline-run/:run_id` 均执行项目成员鉴权。
- 两类接口只允许 `ran_to_completion`、`faulted`、`canceled` 状态。
- 缺失文件或目录不阻断记录删除；其他文件系统错误返回失败且保留数据库记录。
- 删除入口使用现有破坏性确认弹窗，成功后刷新列表或返回对应列表页。
- 删除 PipelineRun 时只清理 `data/pipeline/runs/<run-id>/`，不触碰仓库工作区。

## Risk

- PipelineRun 的 Artifact ID 可能被已生成 Version 作为历史引用保存；删除 Artifact 后 Version 继续使用已持久化的镜像、SHA 和 source commit 快照，不再依赖 Artifact 记录。
