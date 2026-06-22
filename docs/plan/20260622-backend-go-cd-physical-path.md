# backend-go CD Docker Compose 物理路径修复计划
最后修改时间: 2026-06-22 11:41:13

Review status: Accepted

## Requirement basis

本计划依据：

- `docs/requirement/20260622-backend-go-cd-physical-path.md`，状态 `Accepted`。
- 旧 backend 对等实现：`backend/src/pomelo_orbit/infrastructure/cd/docker/manager.py`。
- CI 已有修复：`backend-go/internal/service/ci/workspace.go` 与提交 `0bbe6393418003705d47ee7aa48e8bea6701cbf3`。

目标是修复 backend-go CD Docker Compose 模板中 `app.physical_dir` / `app.physical_app_dir` 使用 logical path 的问题，并让 CI/CD 复用同一个 physical data root 解析实现。

## Implementation steps

### 1. 抽取 physical data root resolver

新增中性包：

- `backend-go/internal/runtimepath/physical_data_root.go`
- `backend-go/internal/runtimepath/physical_data_root_test.go`

将 `backend-go/internal/service/ci/workspace.go` 中与“当前进程 data root 对应宿主机路径解析”相关的逻辑抽出：

- `defaultPhysicalDataRoot`
- `currentContainerId`
- `containerIdFromCgroup`
- `normalizeContainerId`
- `currentContainerMountSource`
- `cleanContainerPath`
- Docker inspect mount DTO

公开最小接口：

```go
package runtimepath

func ResolvePhysicalDataRoot(ctx context.Context, logicalDataRoot string) (string, error)
```

行为要求：

- 先将 `logicalDataRoot` 清理并解析为绝对路径。
- 非容器模式返回绝对 data root。
- 容器模式通过当前容器 mount metadata 找到 host source。
- 解析失败返回明确错误，不 fallback 到容器内路径。

错误信息中的业务前缀从 CI 专用文案改为通用文案，例如 `resolve physical data root`。

### 2. 让 CI workspace 使用共享 resolver

修改：

- `backend-go/internal/service/ci/workspace.go`
- `backend-go/internal/service/ci/workspace_test.go`

保留 CI workspace 的领域职责：

- logical workspace/artifacts/stage log path 仍由 `CIWorkspace` 管理。
- Docker mount path 通过 `runtimepath.ResolvePhysicalDataRoot` 获取。
- 测试中的注入 resolver 能力继续保留，便于覆盖物理路径场景。

删除 CI 包中重复的容器检测和 Docker inspect 实现，避免 CI/CD 漂移。

### 3. 新增 CD workspace/path 组件

新增：

- `backend-go/internal/service/cd/workspace.go`
- `backend-go/internal/service/cd/workspace_test.go`

建议结构：

```go
type physicalDataRootResolver func(ctx context.Context, logicalDataRoot string) (string, error)

type Workspace struct {
    logicalDataRoot string
    resolver physicalDataRootResolver
    physicalOnce sync.Once
    physicalDataRoot string
    physicalErr error
}

func NewWorkspace(dataRoot string) *Workspace
func (w *Workspace) AppDir(appCode string) string
func (w *Workspace) DeploymentLogPath(appCode, deploymentId string) string
func (w *Workspace) PhysicalDataRoot(ctx context.Context) (string, error)
func (w *Workspace) PhysicalDir(ctx context.Context) (string, error)
func (w *Workspace) PhysicalAppDir(ctx context.Context, appCode string) (string, error)
```

语义：

- `AppDir` / `DeploymentLogPath` 使用 logical data root，供 backend-go 自己读写文件。
- `PhysicalDir` / `PhysicalAppDir` 使用 physical data root，供 Docker Compose 模板上下文使用。
- 返回给模板的 physical path 使用 `filepath.ToSlash`，避免 Windows 反斜杠。

### 4. Service 使用 CD workspace

修改：

- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/application_extra.go`
- 可能涉及 `backend-go/internal/service/cd/route.go`，仅当替换 logical path helper 时必要。

调整 `Service`：

- 增加 `workspace *Workspace` 字段。
- `NewWithRunner` / `NewExecutionService` 初始化 `NewWorkspace(cfg.DataRoot())`。
- 保留 `dataRoot` 字段或删除由实际改动决定；若只剩少量使用，可用 workspace 统一替代。

部署写文件与日志：

- `writeAndDeploy` 中 `appDir` 改为 `s.workspace.AppDir(app.Code)`。
- `deploymentLog` 使用 `s.workspace.DeploymentLogPath(...)`。
- restart、logs、status 等命令工作目录继续使用 logical `AppDir`。

模板渲染：

- 将 `renderApplicationTemplate(content, appCode, cfg)` 改为 service 方法，例如：

```go
func (s Service) renderApplicationTemplate(ctx context.Context, content string, appCode string) (string, error)
```

- 方法内部通过 workspace 获取 `PhysicalDir(ctx)` 和 `PhysicalAppDir(ctx, appCode)`。
- preview、compose service 解析、实际部署都调用同一 service 方法。
- physical path 解析失败直接返回错误；部署路径中应触发 faulted 状态，不继续写出错误 compose。

### 5. 更新测试

重点测试文件：

- `backend-go/internal/runtimepath/physical_data_root_test.go`
- `backend-go/internal/service/ci/workspace_test.go`
- `backend-go/internal/service/cd/workspace_test.go`
- `backend-go/internal/service/cd/deployment_execution_test.go`
- 可能扩展 `backend-go/internal/service/cd/compose_test.go` 或 `application_extra` 相关测试。

测试要求：

1. `runtimepath.ResolvePhysicalDataRoot` 本地模式把相对 data root 解析为绝对路径。
2. CI workspace 现有 physical mount 测试继续通过。
3. CD workspace：
   - logical app dir 仍为 logical data root 下 `cd/<app>`。
   - physical app dir 使用 resolver 返回的 physical data root。
   - physical path 输出为 slash 格式。
4. CD deploy liquid 渲染：
   - 不再包含裸 `data\cd\...`。
   - 包含 resolver 返回的 physical app dir。
5. resolver 失败时：
   - `ExecuteApplicationDeploy` 返回错误。
   - application status 标记 `deploy_failed`。
   - deployment status 标记 `faulted`。
6. `ApplicationComposePreview` / `composeServices` 使用与 deploy 一致的模板上下文。

### 6. 不修改运行时产物和已执行迁移

不修改：

- `backend-go/data/cd/traefik/docker-compose.yml`
- `backend-go/data/cd/**` 运行时产物
- 已执行迁移文件内容，除非实现过程中证明模板内容本身有错

修复后通过重新部署 Traefik 刷新生成文件。

## Files to change

本需求预计修改/新增：

- `backend-go/internal/runtimepath/physical_data_root.go`（新增）
- `backend-go/internal/runtimepath/physical_data_root_test.go`（新增）
- `backend-go/internal/service/ci/workspace.go`
- `backend-go/internal/service/ci/workspace_test.go`
- `backend-go/internal/service/cd/workspace.go`（新增）
- `backend-go/internal/service/cd/workspace_test.go`（新增）
- `backend-go/internal/service/cd/service.go`
- `backend-go/internal/service/cd/deployment_execution.go`
- `backend-go/internal/service/cd/application_extra.go`
- `backend-go/internal/service/cd/deployment_execution_test.go`
- 视实际引用情况小幅调整 `backend-go/internal/service/cd/route.go` 或相关测试
- `docs/plan/20260622-backend-go-cd-physical-path.md`

当前工作区已有上一项 runtime config 的未提交改动，最终汇报应明确区分：

- 既有改动：`backend-go/config.defaults.yaml`、`backend-go/internal/config/config.go`、`backend-go/internal/config/config_test.go`、`justfile`、`scripts/dev.py`、相关 requirement/verification。
- 本需求新增/修改：以上 CD physical path 相关文件与本 requirement/plan。

## Verification plan

实现后运行：

```bash
cd backend-go && go fmt ./...
cd backend-go && go test ./...
cd backend-go && go vet ./...
```

静态检查：

- 搜索 `data\\cd` / `data/cd` 在生成 compose 测试输出中的裸相对路径，确认测试不再固化错误行为。
- 搜索 `currentContainerId` / `currentContainerMountSource`，确认实现只在 `runtimepath` 中存在，CI/CD 没有重复实现。
- 搜索 `renderApplicationTemplate`，确认 preview、compose service 解析、deploy 使用同一模板上下文。

人工验证建议：

- 用户重新触发 Traefik 部署，确认新生成的 `docker-compose.yml` volume source 不再是 `data\cd\...`。
- 若 backend-go 在容器内运行，确认容器有 Docker socket 权限且 data 目录是可解析的 bind mount。

## Rollback

如实现后 CD 部署路径异常：

- 回退本需求相关代码改动即可恢复旧行为。
- 不涉及数据库迁移变更。
- 不涉及运行时产物强制修改；旧生成文件可由重新部署覆盖。

## Assumptions

- `cfg.DataRoot()` 表示 backend-go 自身读写的 logical data root。
- Docker Compose 模板变量 `app.physical_dir` / `app.physical_app_dir` 应表示 Docker 可使用的 physical host path。
- backend-go 容器化运行时，如果要部署 Docker 子容器，当前容器必须能访问 Docker socket，并且 data root 必须是可通过 Docker inspect 解析的 bind mount。

## Risks

- 抽取 resolver 触及 CI 代码，需要保证 CI 现有测试全量通过，避免回归。
- 容器模式依赖 Docker inspect 当前容器 mount metadata，权限或挂载不满足时会显式失败。
- Windows Docker Desktop 对路径格式敏感，模板输出必须统一 slash，避免反斜杠被 compose 误判。
- 当前工作区存在其他未提交改动，实施和汇报时必须明确边界，避免误认为都是本需求变更。

## User review notes

- 用户已明确要求“开始实现”，按 SpecFlow standard 规则视为接受 Requirement，并先补 Plan。
- 计划完成后需等待用户确认或再次要求开始实现，才能进入 Implementation 阶段。