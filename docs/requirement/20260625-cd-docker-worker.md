# CD Docker 操作单节点运行时修复需求
最后修改时间: 2026-06-25 09:42:04

Review status: Accepted

## Background

远程部署环境中，`POST /api/cd/application/{app_id}/stop` 返回 500，日志显示 `Failed to stop application`。排查发现 API 容器 `pomelo-orbit` 未挂载 `/var/run/docker.sock`，但当前 `StopApplication` 在 API 请求线程内直接执行 `docker compose down`，因此无法连接 Docker daemon。

经过讨论，本轮不再继续拆分 API / worker 两个部署角色，也不设计 API -> worker HTTP、peer delegation 或内部查询协议。项目当前更适合采用单节点运行时：对外 API 节点本身同时启动后台 task worker loop，并挂载 Docker socket。外部请求进入 API 后只负责鉴权、校验、创建 deployment/task 并返回；后台消费方由同一进程内的 worker loop 处理，API handler 不关心谁消费。

## Goal

1. 将 `pomelo-orbit` API 节点改为单节点运行时：同一进程同时提供 HTTP API 和后台 task worker loop。
2. 移除远程部署中的独立 `pomelo-orbit-worker` 服务，避免 API/worker 双容器职责分裂。
3. 给 `pomelo-orbit` 服务挂载 `/var/run/docker.sock`，使 status/logs 和后台 task 都能访问 Docker daemon。
4. 将应用停止操作改为 task：API 层只负责鉴权、状态校验、创建 deployment、入队任务并返回 `deployment_id`。
5. worker loop 执行实际 `docker compose down`，并负责更新 deployment 与 application 状态。
6. `ApplicationStatus` 与 `ApplicationLogs` 保持同步 Docker 查询，不引入跨节点结果获取机制。

## Non-goal

1. 不实现 API -> worker HTTP。
2. 不实现 peer delegation、内部 token、worker 内部 HTTP 或跨节点 status/logs 查询。
3. 不在本轮设计多节点 task capability routing。
4. 不重构整个 CD 执行框架，只完成单节点运行时和 stop task 化所需改动。
5. 不修改已执行迁移文件。
6. 不执行 git commit / push / merge 等 git 写操作。

## User scenarios

1. 用户在应用列表或详情页点击停止应用后，接口立即返回 `deployment_id`，停止过程由同一 API 进程内的 worker loop 后台执行。
2. deploy / restart / stop 都以 task 方式执行，HTTP handler 不同步等待 Docker 变更型操作完成。
3. 用户查看应用状态时，API 节点直接执行 `docker compose ps --format json` 并返回。
4. 用户查看应用日志时，API 节点直接执行 `docker compose logs --tail N` 并返回。
5. 如果 stop Docker 操作失败，deployment 记录失败原因，application 状态不被错误标记为已停止。

## Acceptance

1. `App.Serve` 或等价启动路径同时启动 HTTP server 和 background worker loop。
2. `scripts/docker-compose.yml` 中不再部署独立 `pomelo-orbit-worker` 服务。
3. `scripts/docker-compose.yml` 中 `pomelo-orbit` 服务挂载 `/var/run/docker.sock:/var/run/docker.sock`。
4. `StopApplication` 不再在 API 请求处理中直接执行 `docker compose down`。
5. 新增 `cd.application.stop` task 类型，并注册对应 worker handler。
6. stop 成功后：deployment 标记为 `ran_to_completion`，application 状态更新为 `undeployed`。
7. stop 失败后：deployment 标记为 `faulted`，错误信息包含 Docker 命令输出或错误原因；application 状态保持原状，不被错误标记为已停止。
8. `remove_volumes` 参数能从 API 请求传入 stop task，并在 worker 执行 `docker compose down -v`。
9. `ApplicationStatus` 与 `ApplicationLogs` 不新增跨节点逻辑，继续依赖当前节点 Docker socket。
10. 路由相关 Docker reload 已审查：`route.go` 中 Traefik reload 仅在 `runtime.GOOS == windows` 时执行，当前远程 Linux 部署不触发，不纳入本轮实际改动。
11. 后端相关测试覆盖 stop task 入队、stop worker handler 调用、stop 执行成功/失败路径。
12. 代码变更后运行适用后端检查。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 采用单节点运行时：API 节点本身就是 worker。
2. 接受 API 节点挂载 Docker socket，以简化运行时架构和 status/logs 获取路径。
3. 砍掉独立 worker 容器部署；保留代码中的 worker loop 能力，并由 `serve` 启动。
4. stop 与 deploy/restart 对齐，改为后台 task 执行。
5. status/logs 保持当前同步接口语义。

## Risk

1. API 容器挂载 Docker socket 会扩大对外 API 容器的权限边界；本轮以简单稳定为优先，并依赖现有鉴权和受控 service 逻辑降低风险。
2. HTTP 服务与 worker loop 同进程运行，进程重启会同时影响 API 和任务消费；已有 task 表、lease 和 retry 机制负责恢复未完成任务。
3. stop 改异步后，接口语义从“停止完成后返回”变为“提交停止任务后返回”；目前前端使用 `deployment_id`，符合 deploy/restart 既有模式。
4. 远程 MCP `pomelo-orbit-ssh` 当前在本会话尝试时报 SSH key 路径错误；远程排查已通过 `python scripts/manage.py` 完成，后续如必须使用 MCP 需要先修正 MCP 配置。