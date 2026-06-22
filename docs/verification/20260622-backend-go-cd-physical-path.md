# backend-go CD Docker Compose 物理路径修复验证
最后修改时间: 2026-06-22 11:56:41

Review status: Accepted

## Requirement alignment

对照 `docs/requirement/20260622-backend-go-cd-physical-path.md`：

- 已修复 CD 模板上下文中 `app.physical_dir` / `app.physical_app_dir` 直接使用 logical `cfg.DataRoot()` 的问题。
- 已让 Docker Compose 模板中的 physical path 通过 shared resolver 解析为 Docker 可接受的宿主机路径。
- 已复用 CI 的 physical data root 解析思路，并抽取到中性包，避免 CI/CD 两套实现漂移。
- CD 自身读写应用目录、部署日志仍使用 logical data root。
- physical path 输出使用 slash 格式，避免 Windows 反斜杠进入 compose volume source。
- 容器模式下 resolver 失败时不 fallback 到容器内路径，部署返回错误并标记失败。

## Plan alignment

对照 `docs/plan/20260622-backend-go-cd-physical-path.md`：

- 已新增 `backend-go/internal/runtimepath/physical_data_root.go` 和测试。
- 已修改 CI workspace 使用 `runtimepath.ResolvePhysicalDataRoot`。
- 已新增 CD `Workspace`，集中管理 logical app dir、deployment log path、physical data root、physical app dir。
- 已修改 CD `Service` 及部署/预览/服务解析路径，统一通过 service template renderer 获取模板上下文。
- 已补充 CD workspace、template rendering、deploy failure、CI workspace 相关测试。
- 未修改运行时产物 `backend-go/data/cd/**`，未修改已执行迁移文件。

## Actual diff summary

本需求相关新增/修改文件：

- `backend-go/internal/runtimepath/physical_data_root.go`
- `backend-go/internal/runtimepath/physical_data_root_test.go`
- `backend-go/internal/service/ci/workspace.go`
- `backend-go/internal/service/ci/workspace_test.go`
- `backend-go/internal/service/cd/workspace.go`
- `backend-go/internal/service/cd/workspace_test.go`
- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/route.go`
- `backend-go/internal/service/cd/compose_test.go`
- `backend-go/internal/service/cd/deployment_execution_test.go`
- `docs/requirement/20260622-backend-go-cd-physical-path.md`
- `docs/plan/20260622-backend-go-cd-physical-path.md`
- `docs/verification/20260622-backend-go-cd-physical-path.md`

## Expected vs actual changed files

计划中预期修改/新增的 backend-go runtimepath、CI workspace、CD workspace、CD service、deployment execution、application extra、route 和相关测试均已覆盖。

静态 `git diff --name-only` 当前只列出本需求相关 Go 文件，因为上一项 runtime config 改动已不在当前 diff 列表中显示；本次验证未发现混入前端或旧 Python backend 修改。

## Acceptance checklist

- [x] CD 模板不再把 `cfg.DataRoot()` / `filepath.Join(cfg.DataRoot(), "cd", appCode)` 直接作为 `physical_dir` / `physical_app_dir`。
- [x] CI 和 CD 复用同一个 physical data root resolver：`runtimepath.ResolvePhysicalDataRoot`。
- [x] CD 有内聚 workspace 组件区分 logical path 和 physical path。
- [x] 本地相对 data root 解析为绝对 physical data root。
- [x] 渲染给 Docker Compose 的 physical path 使用 slash 格式。
- [x] 容器模式无法解析 data mount source 时 resolver 返回明确错误，不 fallback。
- [x] `renderApplicationCompose`、`composeServices`、实际部署均调用同一 `s.renderApplicationTemplate(ctx, ...)`。
- [x] 相关 Go 测试覆盖 physical path 渲染、resolver 失败部署 faulted、CI workspace 复用等场景。
- [x] Go 测试、vet 和 diff whitespace 检查通过。

## Command results

### `go -C "backend-go" fmt ./...`

结果：通过，无输出。

### `go -C "backend-go" test ./...`

结果：通过。

关键输出摘要：

```text
ok  	backend/internal/runtimepath	(cached)
ok  	backend/internal/service/cd	(cached)
ok  	backend/internal/service/ci	(cached)
ok  	backend/internal/transport/http	(cached)
ok  	backend/internal/worker/handler/cd	(cached)
ok  	backend/internal/worker/handler/ci	(cached)
```

### `go -C "backend-go" vet ./...`

结果：通过，无输出。

### `git diff --check`

结果：通过，无输出。

### 静态搜索

- `currentContainerId|currentContainerMountSource`：只存在于 `backend-go/internal/runtimepath/physical_data_root.go`。
- `renderApplicationTemplate(`：CD preview、compose service 解析、deploy 均调用 service 方法。
- CD 测试中搜索 `data\\cd|data/cd`：无匹配，测试不再固化裸相对 `data/cd` compose source。

## Missed or expanded scope

- 未修改前端；符合 non-goal。
- 未修改旧 Python backend；符合 non-goal。
- 未修改已执行迁移文件；符合 non-goal。
- 未手工修改 `backend-go/data/cd/**` 运行时产物；符合 non-goal。
- 未启动开发服务器或真实部署 Traefik；需要由用户按项目约束自行触发。

## Risks

- 容器化 backend-go 场景依赖 Docker socket 权限和 data root 的可解析 bind mount；如果环境不满足，现在会显式失败。这符合需求，但需要部署环境正确挂载 data 目录。
- 已存在的运行时 `backend-go/data/cd/traefik/docker-compose.yml` 不会被代码修改自动刷新，需要重新触发部署生成。

## Incomplete items

无代码层未完成项。

人工建议：重新触发 Traefik 部署后，检查新生成的 compose volume source 是否为 slash 格式的绝对宿主机路径。

## Conclusion

验证通过。本次实现符合 requirement 和 plan，修复了 CD Docker Compose 模板中 logical data path 被误当成 physical bind mount source 的问题，并与 CI 物理路径解析保持一致。
