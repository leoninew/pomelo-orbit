# Environment 工作目录归属计划
最后修改时间: 2026-09-09 20:06:07

Review status: Accepted

Mode: standard

## Basis

- Requirement: [Environment 工作目录归属需求](../requirement/20260909-environment-workspace-ownership.md)（`Accepted`）。
- 无独立 Spec；本任务采用 standard 模式，直接从 Requirement 进入 Plan。
- 当前暂存的 `20260902-remote-ssh-deployment-environments` 已建立 `local | ssh` 显式 target、共享 runtime dispatcher 与 `environment.workspace_root` 列。本任务只改变工作目录的所有权和读取路径。
- 并行的 [Environment 分层边界收敛计划](20260909-environment-architecture-boundaries.md) 负责 View、Prober、HTTP/MCP mapper 和 runtime 工厂等分层整理；本计划不重复实现其边界调整。

## Delivery boundary

将 `workspace_root` 提升为 `Environment` 的公共、可持久化 target 配置。local 与 ssh 都由 Environment 页面和同等 MCP 更新入口显式提交并从库存读取；本机 runtime、Probe、Compose/运行时查询、Gateway network、Route/Traefik 文件同步均从解析出的 Environment target 工作目录派生 `<workspace_root>/<service_code>`。

新 Project 仍在同一事务中创建 active local Environment，但其 `workspace_root` 初始为空，直到成员在 Environment 页面保存有效目录前不是可 Probe、可部署的目标。不存在 YAML 默认值、空值 fallback 或自动移动既有 Compose 树。部署执行日志保留在控制面独立的日志根，不再位于任一 Environment 的工作目录树。

本期明确不做：将已存在的 migration `000041` 合并进 `000040`、修改已执行 migration、使用 dbtalk 同步开发库 schema/migration 版本或对开发库做就地迁移记录整理。现有 `workspace_root` 列足以承载此功能，不新增 schema migration；存量空 local Environment 需要成员在页面或 MCP 保存目录后才能通过 Probe。

## Implementation steps

1. **收敛公共 Environment 工作目录模型与库存映射。**
   - 在 `model.Environment` 增加公共 `WorkspaceRoot`，从 `EnvironmentSSHTarget` 移除该字段；SSH target 仅保留 platform、host、port、username、credential binding 和 host key。
   - repository / SQLC 映射将 `environment.workspace_root` 无条件读写公共字段；不再因 target 是 local 写入 `NULL`，切换 local/ssh 时保留本次输入的目录而非把目录附着到 SSH 子对象。
   - 保持既有 `target_type`、SSH nullable columns、deployment snapshot 和 gateway binding schema 不变；不编辑 `000040_*`、`000041_*` 或任何其他已执行 migration。
   - Project bootstrap 创建的 local Environment 保持空 `WorkspaceRoot`；不由 config、seed、handler 或 MCP 自动填充值。

2. **扩展 Environment 更新、校验和 freshness 语义。**
   - `environmentdto.UpdateInput`、Proto `ProjectEnvironmentUpdateReq` 与 MCP `orbit_update_project_environment` 将 local / ssh 的工作目录作为各自 target payload 的必填配置；请求仍使用 `local.workspace_root`、`ssh.workspace_root`，不引入兼容字段。
   - 更新 `applyUpdate`：local target 仅接受 local 工作目录且清除 SSH 专属配置；ssh target 仅接受完整 SSH 配置和 SSH 工作目录。切换 target 时不沿用另一 target 的目录，也不依据 hostname 推断类型。
   - 将 `validWorkspaceRoot` 按明确平台校验：local 使用控制面平台的本地绝对路径规则，SSH 保持 Linux absolute / `~`、Windows drive absolute / `~` 规则。空值不能保存为 active 可用 target；为支持新建 Project 的待配置 local record，bootstrap 记录可存在空值，但 `Update`、`Probe`、`TargetResolver` 和 runtime 必须拒绝它并返回指向 Environment 配置的 validation diagnostic。
   - 目录变更纳入 `environmentTargetChanged`，使 `target_revision` 递增、Probe freshness 失效；沿用已有 deployment target type/revision snapshot 校验，不增加 workspace snapshot 列。

3. **让 local/SSH Probe 与 runtime 从 target 取目录。**
   - local runner 删除构造时的 `workspaceRoot` 状态；为每次 `ServiceDir`、environment-root query、materialize、sync 和 Probe 从 `target.Environment.WorkspaceRoot` 获取、清理并校验目录。SSH runner 同样读取公共字段。
   - `newDeploymentRuntime` 只注入 Docker daemon path resolver 和 local/SSH runner，不再接收 `cfg.Workspace.Deployment`。HTTP 与 Worker 继续复用同一工厂。
   - local Probe 在有效目录下创建/检查工作根并验证 Docker / Compose；空或不可写目录先产生安全诊断，再不执行 Docker 命令。SSH bootstrap 和 Probe 继续使用所保存的 SSH workspace，不改变一次性认证或 host-key 规则。
   - 通过共同 target runtime 保持 Compose lifecycle、runtime query/log、Gateway network、Traefik REST Route publish 的 local/SSH 目录语义一致；不增加 local/SSH fallback。

4. **解耦控制面部署日志根与 YAML 工作目录。**
   - 在 config 引入明确的 `logging.deployment_root`，独立于 `workspace.pipeline` 和 Environment 工作目录；`executionlog.NewDeploymentStore`、HTTP 与 Worker 都从该日志根初始化。
   - 移除 `WorkspaceConfig.Deployment`、YAML `workspace.deployment`、环境变量绑定、路径规范化/重叠校验和容器启动挂载验证；保留 `workspace.pipeline` 的 CI 职责不变。
   - 日志目录以 `<deployment_log_root>/<service_code>/<deployment_id>.log` 组织，删除部署记录时只删除该控制面日志文件；不迁移或删除任何旧 Compose/workspace 文件。
   - 容器运行时只对 `workspace.pipeline` 和新的 `logging.deployment_root` 做对应的目录 / Docker-daemon 可见性检查；local Environment workspace 的 Docker-visible requirement 在 Environment Probe 中按已保存路径验证。

5. **更新共享 View、HTTP/MCP 与 Web Environment 表单。**
   - 在并行分层整改产生的 `environmentdto.View` 上，local 与 SSH target 都映射库存 `workspace_root`；local 的 platform/host/username 仍来自组合根控制面快照，不可编辑也不持久化。
   - HTTP 和 MCP 更新/读取使用同一 target-specific 契约：local detail 显示已保存目录并在编辑 dialog 提供必填输入；ssh 继续将目录与 host/user/platform 同时编辑。新建 Project 的空目录在 UI 中显示未设置，保存后才允许 Probe。
   - 将前端表单重置、target 切换、提交、local/SSH 校验和 i18n 对齐：仅使用产品占位值，不将占位值视为已保存默认值；切换 SSH platform 时不改写 local 已保存路径。
   - 不暴露 SSH credential、私钥、bootstrap password/private key/passphrase；不在本期重做 View、Prober 或 mapper 的分层工作，只在其已收敛接口上补齐公共工作目录。

6. **校准活文档与回归测试。**
   - 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`、`docs/guides/volume-mounting.md`、`docs/guides/docker-deployment.md`，删除 local 工作目录来自 `workspace.deployment` 或固定 YAML 的描述，说明工作目录由 Environment 保存、日志根独立。
   - 只在必要处更新仍以 `<workspace.deployment>` 表示部署目标目录的活操作指引；不以本期重写 archive 文档或无关的历史 SpecFlow 文档。
   - 添加模型/usecase/repository、local/SSH runtime、Probe、HTTP/MCP、Web 表单与 config/log-root 的聚焦测试，覆盖空目录拒绝、目录改动失效 Probe/任务快照、local 写入正确目录与日志隔离。

## Files to change

| Area | Main paths |
| --- | --- |
| Environment model and persistence | `internal/model/environment.go`, `internal/application/environment/dto/environment.go`, `internal/application/environment/usecase/service.go`, `target.go`, `probe.go`, `view.go`, `internal/repository/impl/sqlc/environment/repository.go`, `sql/query/environment/environment.sql`, generated SQLC models/query code |
| Target contract | `proto/orbit/v1/environment/environment.proto`, generated Go/TypeScript Proto, HTTP Environment handler/routes, `internal/api/mcp/delivery/{environment_tools.go,mapper.go,types.go}`, related tests |
| Runtime and probe | `internal/infrastructure/runner/local/runtime.go`, `internal/infrastructure/runner/ssh/runtime.go`, `internal/infrastructure/runner/environment/prober.go`, `internal/infrastructure/runner/target/runtime.go`, `internal/bootstrap/deployment_runtime.go`, `http.go`, `worker.go` |
| Config and logs | `internal/config/config.go` and tests, `configs/config.yaml`, `internal/bootstrap/app.go` and tests, `internal/infrastructure/storage/local/executionlog/store.go`, bootstrap log-store callers |
| Web | `web/src/views/environment/EnvironmentPage.vue`, `web/src/api/project/environment.ts`, generated environment proto, `web/src/utils/deploymentEnvironment.ts`, locale files and focused tests |
| Living docs | `docs/product/cd-model.md`, `docs/architecture/cd-runtime.md`, `docs/guides/deployment.md`, `docs/guides/volume-mounting.md`, `docs/guides/docker-deployment.md`, directly affected operational guides |

Do not change: `sql/migration/**/000040_*`, `sql/migration/**/000041_*`, development database data or schema-migration records, `workspace.pipeline` behavior, SSH authentication semantics, local/SSH target discrimination, or archive documents.

## Verification plan

Implementation finishes before entering Verification. The Verification stage will:

1. Run focused Go tests for Environment validation/revision/probe, SQLC repository mapping, local and SSH runtime service-root resolution, deployment target snapshot failure after root changes, and execution-log storage.
2. Run focused HTTP/MCP tests proving local and SSH read/update payloads contain only applicable fields, persist their supplied root, never derive it from config, and keep sensitive SSH material absent.
3. Run focused config/bootstrap tests proving `workspace.deployment` is no longer loaded, normalized, mount-validated or passed to runners, while deployment logs use the independent control-plane root.
4. Run focused Web tests plus `yarn --cwd web lint:fix` and `yarn --cwd web typecheck` for local root form validation, empty new-Project state, SSH regression and target switch behavior.
5. Run `task sqlc`, `task proto`, `task check`, `go test ./cmd/... ./internal/...`, and `git diff --check HEAD`.
6. Inspect the final diff to confirm no migration merge, no dbtalk invocation/output, and no development database migration-record changes were introduced.

## Blockers

无。现有 local Environment 的空 `workspace_root` 不阻止实现，但在成员保存有效目录并重新 Probe 前不能部署，这是已接受的产品行为。

## Assumptions

- 当前 schema 已有 nullable `environment.workspace_root`，因此公共字段归属变化不需要新的 DDL；所有数据库形态中的既有空 local 记录按待配置处理。
- `logging.deployment_root` 作为独立进程运维配置，默认值仅决定控制面部署日志位置，不会出现在 Environment API、MCP 或页面，也不会被 target runtime 使用。
- Docker daemon 对 local 用户所选目录的可见性无法由 YAML 静态保证，Probe 需要以该目录实际执行验证。
- 分层边界任务会先或同时提供 View / runtime factory 收敛；若共享文件冲突，以“workspace root 来自库存，不得回退 YAML”为优先约束。

## Risks

- 去除 `workspace.deployment` 后，未保存 local root 的 Project 不能部署；页面、MCP diagnostic 和测试必须明确引导先配置 Environment。
- 用户选择的 local path 可能不是 Orbit 容器和 Docker daemon 均可见的 bind mount；Probe 必须在路径创建和 Compose prerequisites 中安全失败，不能回退到旧目录。
- 将 workspace 字段从 SSH 子对象提升到 Environment 公共字段会影响 model fixtures、SQLC mappings、Proto、HTTP/MCP mapper 和所有 runtime test target；遗漏通常会造成编译失败或错误的空目录。
- deployment logs 与旧 local workspace 脱钩后，历史 log 文件不会迁移；删除历史 deployment 时只按新日志根处理缺失文件，不能清理用户工作目录。
- 该计划与分层边界收敛共享 Environment DTO/usecase/bootstrap 文件，必须在合并时保持各自范围，不回退对方已建立的 View、Prober 或 runtime 工厂。

## Rollback

回退本任务的代码与配置变更即可。不要通过恢复 `workspace.deployment` fallback、自动为 Environment 注入旧目录或移动用户已有 Compose 目录来回退。对已保存的 Environment 路径维持数据原样；日志根按运维配置独立处理。

## User review notes

- 用户确认 local 与 ssh 的工作目录都属于 Environment，由环境页面/MCP 保存后再供 Probe 和部署读取。
- 用户否定以 YAML 作为 local 默认或运行时路径来源，也不接受空值 fallback。
- 用户要求剔除将 migration `000041` 合并进 `000040` 与同步开发库迁移记录，本计划和后续验证均不执行该整理工作。
