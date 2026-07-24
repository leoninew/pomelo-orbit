# CD Docker 操作单节点运行时修复验证
最后修改时间: 2026-06-25 10:49:31

Review status: Accepted

## Requirement alignment

依据 `docs/requirement/20260625-cd-docker-worker.md` 核对，当前实现与轻量需求保持一致：

1. `App.Serve` 已同时启动 HTTP server 与 background worker loop。
2. `scripts/docker-compose.yml` 已移除独立 `pomelo-orbit-worker` 服务。
3. `scripts/docker-compose.yml` 中 `pomelo-orbit` 已挂载 `/var/run/docker.sock:/var/run/docker.sock`。
4. `StopApplication` 已从同步执行 `docker compose down` 改为创建 deployment 后入队 `cd.application.stop` task。
5. 已新增 `cd.application.stop` task 类型，并在 worker router 中注册 stop handler。
6. stop worker 执行成功时更新 application 为 `undeployed`，deployment 为 `ran_to_completion`。
7. stop worker 执行失败时只标记 deployment 为 `faulted`，不把 application 错误标记为已停止。
8. `remove_volumes` 会从 API stop 请求进入 task payload，并由 worker 转换为 `docker compose down -v`。
9. `ApplicationStatus` 与 `ApplicationLogs` 未引入跨节点逻辑，继续由当前节点同步执行 Docker 查询。
10. 路由相关 Docker reload 已按 requirement 记录为远程 Linux 路径不触发，本轮未修改。
11. 已补充 stop task 入队、worker handler 调用、stop 成功/失败路径测试。

## Spec alignment

不适用。当前流程为轻量模式 / light，未创建独立 Spec 文档，按 requirement / 需求核对。

## Plan alignment

不适用。当前流程为轻量模式 / light，未创建独立 Plan 文档，按 requirement / 需求核对。

## Actual diff summary

### Runtime / deployment

- `backend-go/internal/app/app.go`：`Serve` 启动 HTTP server 的同时启动 background worker loop，并在任一组件失败时协调 shutdown。
- `backend-go/cmd/backend-go/main.go`：默认 command 从 `worker` 调整为 `serve`。
- `Dockerfile`：更新默认运行说明，表达 API server 与 background worker 同进程运行。
- `scripts/docker-compose.yml`：删除独立 `pomelo-orbit-worker` 服务；给 `pomelo-orbit` 增加 Docker socket 挂载。

### Stop task

- `backend-go/internal/status/status.go`：新增 `TaskTypeCDApplicationStop`。
- `backend-go/internal/bootstrap/bootstrap.go`：注册 CD stop worker handler。
- `backend-go/internal/service/cd/application_extra.go`：`StopApplication` 改为创建 deployment 后入队 stop task，并立即返回 `deployment_id`。
- `backend-go/internal/service/cd/deployment_execution.go`：新增 `ExecuteApplicationStop`，执行 `docker compose -f docker-compose.yml down`，按 `remove_volumes` 追加 `-v`，并更新 deployment/application 状态。
- `backend-go/internal/worker/handler/cd/handler.go`：CD handler 支持 deploy/restart/stop 三种 operation，并解析 `remove_volumes`。
- `backend-go/internal/transport/http/handler/task/handler.go`：补充 background stop task enqueue route。

### Tests

- `backend-go/internal/service/cd/deployment_execution_test.go`：覆盖 stop 成功、stop runner 失败。
- `backend-go/internal/worker/handler/cd/handler_test.go`：覆盖 stop payload 分发及 `remove_volumes`。
- `backend-go/internal/transport/http/application_routes_test.go`：覆盖 stop API 入队、deployment 等待执行、application 状态保持 deployed。
- `backend-go/internal/transport/http/server_test.go`：覆盖 background stop task route。

## Expected vs actual changed files

| 文件 | 预期 | 实际 |
| --- | --- | --- |
| `docs/requirement/20260625-cd-docker-worker.md` | 记录最终需求决策 | 已创建并标记 Accepted |
| `docs/verification/20260625-cd-docker-worker.md` | 记录验证结果 | 已创建 |
| `Dockerfile` | 更新单节点运行说明 | 已修改 |
| `scripts/docker-compose.yml` | 单服务挂 docker.sock，删除 worker 服务 | 已修改 |
| `backend-go/cmd/backend-go/main.go` | 默认启动 serve | 已修改 |
| `backend-go/internal/app/app.go` | serve 同时启动 worker loop | 已修改 |
| `backend-go/internal/bootstrap/bootstrap.go` | 注册 stop task handler | 已修改 |
| `backend-go/internal/status/status.go` | 新增 stop task type | 已修改 |
| `backend-go/internal/service/cd/application_extra.go` | stop API 改入队 | 已修改 |
| `backend-go/internal/service/cd/deployment_execution.go` | 新增 stop task 执行 | 已修改 |
| `backend-go/internal/worker/handler/cd/handler.go` | handler 支持 stop | 已修改 |
| `backend-go/internal/transport/http/handler/task/handler.go` | background stop enqueue route | 已修改 |
| 相关 Go test 文件 | 覆盖 stop 入队和执行路径 | 已修改 |

## Acceptance criteria checklist

- [x] `App.Serve` 同时启动 HTTP server 和 background worker loop。
- [x] `scripts/docker-compose.yml` 不再部署独立 `pomelo-orbit-worker` 服务。
- [x] `scripts/docker-compose.yml` 中 `pomelo-orbit` 挂载 Docker socket。
- [x] `StopApplication` 不再同步执行 `docker compose down`。
- [x] 新增 `cd.application.stop` task 类型，并注册对应 worker handler。
- [x] stop 成功后 deployment 为 `ran_to_completion`，application 为 `undeployed`。
- [x] stop 失败后 deployment 为 `faulted`，application 状态保持原状。
- [x] `remove_volumes` 能从 API 请求进入 stop task，并使 worker 执行 `down -v`。
- [x] `ApplicationStatus` 与 `ApplicationLogs` 未引入跨节点逻辑，继续依赖当前节点 Docker socket。
- [x] 路由相关 Docker reload 已审查，本轮未扩大范围。
- [x] 后端相关测试覆盖 stop task 入队、worker handler 调用、stop 执行成功/失败路径。
- [x] 已运行适用项目检查。

## Command results

### Focused backend verification

命令：

```bash
go -C backend-go test ./...
```

结果：通过。

### Diff whitespace check

命令：

```bash
git diff --check
```

结果：通过，无 whitespace error。

### Full project check

命令：

```bash
just check
```

结果：通过。该命令执行了：

- `cd frontend && yarn lint:fix`
- `cd frontend && yarn format`
- `cd frontend && yarn typecheck`
- `cd backend-go && go fmt ./...`
- `cd backend-go && go vet ./...`
- `cd backend-go && go test ./...`

所有步骤均通过。

## Missed or expanded scope

- 未实现 API -> worker HTTP、peer delegation、内部 token 或跨节点 status/logs 查询；这符合已接受的 Non-goal。
- 未修改 `ApplicationStatus` / `ApplicationLogs` 的执行方式；这符合已接受的单节点运行时决策。
- 未执行远程部署验证或真实 Docker stop 操作；当前验证范围为本地代码、测试与配置 diff。远程上线前仍需按部署流程重建并更新远程 compose。
- 未执行 git commit / push，符合项目 Git 操作限制。

## Risks

1. API 容器挂载 Docker socket 会扩大对外 API 容器权限边界；这是本轮已接受的架构取舍。
2. HTTP server 与 worker loop 同进程运行，进程重启会同时影响 API 与任务消费；已有 task 表、lease、retry 机制用于恢复未完成任务。
3. stop API 语义变为“提交停止任务后返回”，不再等待 Docker stop 完成；这与 deploy/restart 的 task 模式一致。
4. 当前验证未覆盖远程容器实际停止 FileBrowser 的端到端行为。

## Incomplete items

暂无代码层未完成项。远程部署和真实 stop 验证属于上线/运维后续动作。

## Conclusion

验证通过。当前实现满足已接受的轻量 requirement：项目改为单节点运行时，API 节点同时消费后台任务，独立 worker 容器从部署配置中移除；stop 改为后台 task 执行，status/logs 保持当前节点同步 Docker 查询。