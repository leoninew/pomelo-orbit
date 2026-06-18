# backend-go CI Docker 宿主机物理路径挂载验证
最后修改时间: 2026-06-18 17:04:52

Review status: Draft

## Requirement alignment

按 `docs/requirement/20260618-backend-go-ci-docker-physical-mount.md` 核对。

需求目标：backend-go CI 在启动 Docker stage 容器时，不再把配置中的相对 `dataRoot` 或容器内逻辑路径直接传给 Docker，而是像 Python backend 一样区分 logical data path 与 physical host path，并保持路径解析逻辑分层内聚。

核对结果：

- 已新增 `CIWorkspace` 作为 CI workspace/path 内聚组件，集中负责 logical path、physical data root 和 Docker stage mounts。
- `Executor` 不再直接拼接 Docker host path，而是通过 `workspace.DockerStageMounts(...)` 获取 mount 描述。
- `DockerRunner` 不再使用 `-v source:target:mode`，改为 `--mount type=bind,source=...,target=...`。
- `DockerRunner` 会拒绝相对 host path，避免再次把 `data\ci\...` 交给 Docker。
- 本地模式通过 `filepath.Abs(logicalDataRoot)` 将默认 `data` 解析为绝对 physical data root。
- 容器模式实现当前容器检测与 `docker inspect <container_id> --format {{json .Mounts}}`，用于解析容器内 data root 对应的宿主机 mount source。
- 若容器模式无法解析物理数据目录，返回明确错误，不回退到容器内路径。

## Spec alignment

不适用。light / 轻量模式未创建独立 spec 文档，按 requirement / 需求核对。

## Plan alignment

不适用。light / 轻量模式未创建独立 plan 文档，按 requirement / 需求核对。

## Actual diff summary

实际改动集中在 backend-go CI service 层和本次 requirement 文档：

```text
backend-go/internal/service/ci/execution.go
backend-go/internal/service/ci/execution_test.go
backend-go/internal/service/ci/executor.go
backend-go/internal/service/ci/pipeline_run.go
backend-go/internal/service/ci/repository.go
backend-go/internal/service/ci/runner.go
backend-go/internal/service/ci/workspace.go
backend-go/internal/service/ci/workspace_test.go
docs/requirement/20260618-backend-go-ci-docker-physical-mount.md
```

Diff 统计：

```text
9 files changed, 446 insertions(+), 43 deletions(-)
```

主要实现：

1. 新增 `CIWorkspace`：
   - `CreateRunDirectories`
   - `WorkspacePath`
   - `ArtifactsPath`
   - `StageLogPath`
   - `PhysicalDataRoot`
   - `DockerStageMounts`
2. 新增 physical data root 解析：
   - 本地模式：`filepath.Abs(logicalDataRoot)`。
   - 容器模式：从 `/proc/self/cgroup` 或 `/.dockerenv` + hostname 检测当前容器，再用 `docker inspect` 获取 mounts。
3. `Service` 持有 `workspace *CIWorkspace`，构造函数统一初始化 workspace。
4. `ExecutePipelineRun` 使用 `workspace.CreateRunDirectories`。
5. `Executor` 使用 `workspace.StageLogPath`、`workspace.ArtifactsPath`、`workspace.DockerStageMounts`。
6. `PipelineStageLog` 使用 `workspace.StageLogPath`。
7. `DockerRunner` 将 mount 参数生成拆成 `dockerRunArgs` / `dockerBindMountArg`，使用 `--mount` 并校验 host path 必须是绝对路径。
8. 新增 `workspace_test.go` 覆盖 path 解析和 runner 参数生成。
9. 扩展 `execution_test.go`，确认 stage runner 收到的是绝对 host path mounts。

## Expected vs actual changed files

| 预期 | 实际 | 结果 |
| --- | --- | --- |
| CI workspace/path 内聚组件 | 新增 `backend-go/internal/service/ci/workspace.go` | 符合 |
| Docker runner mount 参数生成 | 修改 `backend-go/internal/service/ci/runner.go` | 符合 |
| Executor 使用集中 path 组件 | 修改 `backend-go/internal/service/ci/executor.go` | 符合 |
| Pipeline execution 创建目录使用集中 path 组件 | 修改 `backend-go/internal/service/ci/execution.go` | 符合 |
| Stage log 读取路径同步 | 修改 `backend-go/internal/service/ci/pipeline_run.go` | 符合 |
| Service 构造注入 workspace | 修改 `backend-go/internal/service/ci/repository.go` | 符合 |
| 单元测试覆盖 path / mount | 新增 `workspace_test.go`，修改 `execution_test.go` | 符合 |
| 前端或其他业务模块 | 未修改 | 符合非目标 |

## Acceptance checklist

- [x] backend-go 不再把 `data\ci\...` 或 `data/ci/...` 相对路径传给 Docker bind mount。
  - `DockerRunner` 拒绝相对 host path。
  - `CIWorkspace` 本地模式输出绝对 physical path。
- [x] workspace 和 artifacts 的 Docker host path 由统一组件解析。
  - `CIWorkspace.DockerStageMounts` 统一生成。
- [x] 本地模式默认 `dataRoot=data` 解析为绝对物理路径。
  - `TestCIWorkspaceResolvesRelativeDataRootToAbsolutePhysicalMounts` 覆盖。
- [x] 容器模式尝试解析当前容器数据目录对应的宿主机 mount source。
  - `currentContainerId`、`currentContainerMountSource` 实现。
- [x] 容器模式无法解析时显式报错。
  - `currentContainerMountSource` 对 inspect 失败、mount 为空、未找到 mount 都返回明确错误。
- [x] Docker runner 使用不易受 Windows 盘符冒号影响的 bind mount 参数形式。
  - 使用 `--mount type=bind,source=...,target=...`。
- [x] 增加路径解析和 Docker mount 参数生成测试。
  - `workspace_test.go` 覆盖 relative data root、absolute data root、physical resolver、`--mount` 参数、相对 host path 拒绝。
- [x] 现有 CI trigger、snapshot、runtime variable 行为不回退。
  - 相关 Go 测试通过。

## Command results

### Targeted backend-go tests

命令：

```text
go -C backend-go test ./internal/service/ci ./internal/worker/handler/ci ./internal/transport/http
```

结果：

```text
ok  	backend/internal/service/ci	(cached)
ok  	backend/internal/worker/handler/ci	(cached)
ok  	backend/internal/transport/http	(cached)
```

### Full backend-go tests

命令：

```text
go -C backend-go test ./...
```

结果：

```text
?   	backend/cmd/backend-go	[no test files]
ok  	backend/internal/app	(cached)
?   	backend/internal/apperror	[no test files]
?   	backend/internal/bootstrap	[no test files]
ok  	backend/internal/config	(cached)
ok  	backend/internal/db	(cached)
ok  	backend/internal/logging	(cached)
?   	backend/internal/migrations	[no test files]
ok  	backend/internal/repository	(cached)
?   	backend/internal/repository/cd	[no test files]
ok  	backend/internal/repository/ci	(cached)
?   	backend/internal/repository/model	[no test files]
?   	backend/internal/repository/project	[no test files]
?   	backend/internal/repository/role	[no test files]
ok  	backend/internal/repository/task	(cached)
?   	backend/internal/repository/user	[no test files]
ok  	backend/internal/security	(cached)
ok  	backend/internal/service/auth	(cached)
ok  	backend/internal/service/cd	(cached)
ok  	backend/internal/service/ci	(cached)
?   	backend/internal/service/project	[no test files]
?   	backend/internal/service/role	[no test files]
?   	backend/internal/service/settings	[no test files]
?   	backend/internal/service/task	[no test files]
?   	backend/internal/service/user	[no test files]
?   	backend/internal/status	[no test files]
ok  	backend/internal/templatex	(cached)
ok  	backend/internal/transport/http	(cached)
?   	backend/internal/transport/http/handler/auth	[no test files]
?   	backend/internal/transport/http/handler/authz	[no test files]
?   	backend/internal/transport/http/handler/cd	[no test files]
?   	backend/internal/transport/http/handler/ci	[no test files]
?   	backend/internal/transport/http/handler/project	[no test files]
?   	backend/internal/transport/http/handler/role	[no test files]
?   	backend/internal/transport/http/handler/settings	[no test files]
?   	backend/internal/transport/http/handler/task	[no test files]
?   	backend/internal/transport/http/handler/user	[no test files]
ok  	backend/internal/transport/http/middleware	(cached)
?   	backend/internal/transport/http/response	[no test files]
ok  	backend/internal/worker	(cached)
ok  	backend/internal/worker/handler/cd	(cached)
ok  	backend/internal/worker/handler/ci	(cached)
```

## Missed or expanded scope

### 未验证项

- 未启动开发服务器。
- 未实际触发一次真实 Docker Desktop / Windows pipeline run。
- 未在 backend-go 容器化运行 + Docker socket 挂载环境中做集成验证。

这些属于运行环境集成验证，不在本轮自动执行范围内；但代码层已通过单元测试和 backend-go 全量测试。

### 范围扩展

无不合理扩展。本次变更没有修改前端、snapshot、runtime variable、pipeline orchestration 或 CD 执行业务逻辑。

## Risks

1. 容器模式依赖当前 backend-go 进程能执行 `docker inspect <container_id>`。
   - Docker socket 未挂载或权限不足时会显式失败。
2. 当前容器 ID 检测覆盖 `/proc/self/cgroup` 和 `/.dockerenv` + hostname；特殊容器运行时若两者都不可用，会走本地模式绝对路径。
3. 真实 Docker Desktop 对 Windows path 的接受行为仍建议通过一次实际 pipeline trigger 人工验收。
4. 测试结果中部分包为 `(cached)`，说明 Go 使用了测试缓存；由于相关文件已变更，`backend/internal/service/ci` 对应测试已覆盖新代码并通过。

## Incomplete items

- 无代码实现层面的未完成项。
- 建议用户在本地手动触发一次仓库详情 pipeline，确认 Docker Desktop 下 git clone stage 不再出现 `data\ci\... includes invalid characters`。

## Conclusion

本次实现符合 requirement：backend-go CI Docker mount 已从分散的相对路径拼接改为集中 workspace/path 组件解析，支持本地绝对 physical path 与容器模式 host mount source 解析，Docker runner 使用 `--mount type=bind` 并拒绝相对 host path。相关 backend-go 测试与全量 backend-go 测试均通过。
