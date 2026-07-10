# CD 部署详情状态轮询修复
最后修改时间: 2026-07-10 10:20:18

Review status: Accepted

## Background

停止应用接口异步创建 deployment 并提交后台任务。若部署详情页首次读取发生在任务执行前，会得到 `waiting_to_run`。当前详情页将 deployment 状态刷新绑定到容器日志轮询，而 `stop` 部署不支持容器日志，因此页面不会继续刷新状态，可能长期显示首次读取到的非终态状态。

## Goal

在部署详情页中，对所有未终态 deployment（包括 `stop`）定期请求 deployment 详情，直到状态进入 `ran_to_completion`、`faulted` 或 `canceled`。

## Non-goal

- 不改变后端 deployment、background task 或 Docker 执行逻辑。
- 不为 `stop` 部署请求或展示容器日志。
- 不修改部署状态机、取消语义或队列租约机制。

## User scenarios

- 用户提交停止操作后进入部署详情页，首次状态为 `waiting_to_run`；任务完成后页面自动更新为终态。
- 用户查看 deploy 或 restart 的详情时，页面继续刷新状态；容器日志行为保持既有能力。
- 页面卸载后停止所有轮询，避免遗留请求。

## Acceptance

- `waiting_to_run` 或 `running` 的 stop deployment 每隔固定时间刷新详情，直到进入终态。
- deploy/restart 的状态轮询不依赖是否请求容器日志。
- 终态 deployment 不启动状态轮询。
- stop deployment 不触发容器日志请求。
- 组件卸载时中止状态和日志轮询。

## Open questions

不适用。

## Decisions

- 复用现有 2 秒刷新间隔。
- 将状态轮询与容器日志轮询分离；日志轮询仅适用于非 stop 操作。

## Risk

轮询会增加少量详情请求；仅限未终态 deployment，进入终态或页面卸载后停止。