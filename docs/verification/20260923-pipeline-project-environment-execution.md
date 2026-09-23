# CI/CD 共用 Project Environment 与流水线远端运行验证
最后修改时间: 2026-09-23 17:25:11

Review status: Draft

Mode: standard

## Verification scope

本次验证对照：

- `docs/intent/20260923-pipeline-project-environment-execution.md`
- `docs/plan/20260923-pipeline-project-environment-execution.md`
- 当前工作区相对 `HEAD` 的实现差异
- 开发库 `DBTALK_DSN_APP` 的实际 schema 与数据

本次未启动、停止或重启开发服务器。

## Intent alignment

实现仍围绕每个 Project 唯一的 CI/CD 共用 Environment：

- Environment 入口与项目资源归入独立的“项目”导航分组；系统管理保留用户、角色、登录历史、访问令牌和系统设置。
- Pipeline Run 使用所属 Project Environment 的目标快照；local 保持本地执行路径，SSH 按目标平台选择远端运行，不回退到控制面本地。
- SSH 日志、文件制品位置和清理由目标端 workspace 负责；控制面只读取增量日志和元数据，不增加下载接口。
- Pipeline 历史记录作为逻辑关联和冗余快照，不因历史 Run 阻止 Pipeline 或 Repository 删除；Repository 删除前仅校验当前 Pipeline 绑定。
- Deployment 记录继续保留 Project 与 rollback 自引用的必要物理关系；对 Application、Version、Service、Environment、Gateway 等历史引用保持逻辑引用。

结论：与 Intent 的产品边界和删除语义一致。

## Plan alignment

四批实现均已有代码或测试覆盖：

1. 共用 Environment 入口、项目切换和 CI/CD 就绪引导。
2. Run 目标/凭据快照、执行前复核、Retry 重新绑定，以及 SSH 源码约束。
3. Windows SSH runner、远端 workspace/log/artifact 适配和取消清理。
4. Linux SSH runner、POSIX 路径/命令适配及跨平台回归。

计划中要求的真实 Windows/Linux SSH 目标属于环境前置条件，本次未提供对应 E2E 环境，因此只能验证到测试代码和模拟/应用层路径，不能把平台真实端到端标记为完成。

## Actual diff summary

当前相对 `HEAD` 共 40 个文件变更，约 670 行新增、176 行删除，主要包括：

- Pipeline Run 的 Environment/target 快照、API 映射、Proto 与 SQLC 生成代码。
- Pipeline Run 的 local/SSH 目标解析、执行前校验、远端 workspace/log/artifact 和 Windows/Linux runner。
- Pipeline/Repository 删除约束：删除 Repository 前检查当前绑定 Pipeline，不查询历史 `pipeline_run` 作为阻止条件。
- Environment 页面、项目切换、导航分组、CI/CD 操作引导及中英文文案。
- CI/CD 产品、架构和指南文档更新。

未发现与本任务无关的已执行迁移文件被改写；新增目标字段通过迁移 `000048_pipeline_run_target` 提供 up/down 脚本。

## Expected vs actual files

### Expected and present

- `internal/application/pipeline_run/`
- `internal/infrastructure/runner/ssh/`
- `internal/api/http/handler/pipeline_run/`
- `internal/model/`、Proto、SQLC 生成代码与 Pipeline Run 查询
- `web/src/navigation.ts`、Environment/Project/Pipeline Run 视图、stores、i18n 和相关测试
- `sql/migration/{postgres,mysql,sqlite}/000048_pipeline_run_target.*`
- `docs/product/`、`docs/guides/` 与计划文档

### Additional but in scope

- `internal/application/repository/`、`sql/query/repository/`：落实 Repository 删除前的当前 Pipeline 绑定校验。
- `internal/application/repository/usecase/*_test.go`：覆盖绑定时拒绝、无绑定时允许删除和 project/repository 作用域。
- `internal/infrastructure/runner/ssh/pipeline_posix_test.go`：Linux 路径、命令和挂载约束测试。

### Not changed

- 未修改 `web/src/views/project/ProjectInitializationPage.test.ts`；该文件的既有 Prettier 差异导致 `task check` 的完整任务在 Web format 阶段停止。

## Acceptance checklist

| 验收项 | 结果 | 证据 |
|---|---|---|
| Environment 是 Project 下唯一共用能力，入口不再重复挂在持续部署 | 通过 | `web/src/navigation.test.ts`；前端全量测试 133/133 通过 |
| CI/CD 未就绪时有后端校验，Web 初始化引导保留 | 通过 | Pipeline Run 定向用例、Project Initialization 相关前端测试、Go 全量测试通过 |
| Environment 就绪但无 Gateway 时 CI 不被 Gateway 错误阻断 | 通过 | `ProjectInitializationPage.test.ts` 的 CI 返回路径用例通过；应用层测试通过 |
| Run 创建时冻结目标/凭据修订，执行前拒绝漂移，Retry 重新绑定 | 通过 | Pipeline Run service、API mapper、Windows/Linux 应用层定向测试通过 |
| local 目录 Repository 保持原行为，SSH + 本地目录不回退本机 | 通过 | 触发拒绝与 Repository source 相关定向测试通过 |
| Windows SSH 的 Stage/DAG、远端日志/制品、命令/镜像制品、取消清理 | 代码与模拟验收通过；真实 E2E 未执行 | `windows_remote_test.go`、SSH runner 测试通过；缺少 Windows Project ID 和目标已有镜像，E2E 明确 skip |
| Linux SSH 的独立路径、命令、Stage/DAG、日志/制品和跨平台回归 | 代码与模拟验收通过；真实 E2E 未执行 | `pipeline_posix_test.go`、Linux 应用层测试通过；缺少 Linux Project ID 和目标已有镜像，E2E 明确 skip |
| SSH 日志/文件制品不回传，历史 Run 按目标快照安全读取/清理 | 通过 | workspace/log/artifact 定向测试、API mapper 测试和 Go 全量测试通过 |
| Pipeline 历史记录不阻止 Pipeline/Repository 删除 | 通过 | Repository 删除仅调用 `RepositoryHasBoundPipelines`；开发库保留 1 条已断 Pipeline/Repository 逻辑引用历史 Run |
| Deployment 记录遵从历史逻辑引用规范 | 通过 | 开发库 138 条 Deployment；项目引用和 rollback 自引用均无断裂，Deployment 对其他资源无物理外键 |

## Test results

### Passed

- `go test -count=1 ./cmd/... ./internal/...`
- `go test -count=1 ./internal/application/pipeline_run/usecase ./internal/infrastructure/runner/ssh ./internal/api/http/handler/pipeline_run ./internal/application/repository/usecase`
- `yarn --cwd web test`：30 个文件、133 个测试通过
- `yarn --cwd web typecheck`
- `yarn --cwd web lint`
- `yarn --cwd web lint:fix`
- 修改过的 Web 文件单独执行 Prettier check：通过
- `golangci-lint config verify`
- `golangci-lint fmt --diff ./cmd/... ./internal/... ./sql`
- `golangci-lint run ./cmd/... ./internal/... ./sql`：0 issues
- `git diff HEAD --check`

### Partial / blocked

- `task check`：Web typecheck、lint 通过；Web format 在未修改的 `web/src/views/project/ProjectInitializationPage.test.ts` 发现既有格式差异后停止。未为本任务扩大范围修改该文件。
- `go test -count=1 -v ./internal/application/pipeline_run/usecase -run 'Test(Windows|Linux)PipelineApplicationRemoteEndToEnd$' ./internal/infrastructure/runner/ssh -run 'Test(Windows|Linux)PipelineRemoteEndToEnd$'`：测试进程通过，但 Windows/Linux 真实远端测试均因未设置对应 Project ID 和已有 shell image 而 skip。

## Database verification

通过 `dbtalk --dsn-env DBTALK_DSN_APP` 查询开发库 PostgreSQL 18.6：

- 表计数：`pipeline_run=11`、`deployment=138`、`pipeline=8`、`repository=4`、`environment=3`、`project=4`。
- `pipeline_run` 没有指向 `pipeline` 或 `repository` 的物理外键；其 `pipeline_id`、`repository_id` 是历史逻辑关联/冗余快照。
- 当前存在 1 条历史 Run `01M0A3HPJW1ENQ84NVPEHHJETT`，指向已删除 Pipeline `01M0A3FS37Y78D0R096GTFMNNV` 和 Repository `01M0A25N55CCE3G2913D2S9X57`；状态为 `faulted`。该记录按规范保留，不作为删除阻断条件。
- `deployment` 只发现 `deployment.project_id -> project.id` 与 `deployment.rollback_from_deployment_id -> deployment.id` 两个物理外键，删除规则为 `NO ACTION`；对 Application、Version、Service、Environment、Gateway 的列均为逻辑引用。
- Deployment 断裂引用检查：`deployment_missing_project=0`、`deployment_broken_rollback=0`。
- `pipeline_run` 的新增目标字段已在开发库实际存在，迁移 `000048_pipeline_run_target` 的 up/down 脚本与三种数据库目录一致。

## Scope deviations

- 真实 Windows/Linux OpenSSH + Docker 目标未提供，不能完成计划要求的两类平台端到端验收。
- 既有 `task check` 格式阻塞未修复，因为不属于本任务改动文件。
- 实现记录中已知真实 Linux Project 原配置的 `<workspace_root>/pipeline` 权限由 root 持有且 SSH 用户不可写；隔离测试工作区不能等同于原配置验收，仍需修复权限后复跑。

## Risks and incomplete items

1. 提供可访问的 Windows native OpenSSH + WSL2 Docker Desktop Project 和已有 shell image 后，重新运行 Windows Pipeline Application/Runner E2E。
2. 提供可访问的 Linux OpenSSH + Docker Project，并修复原配置 workspace 权限后，重新运行 Linux E2E；验证远端日志、制品和取消清理。
3. 若要使仓库级 `task check` 全绿，应单独处理 `web/src/views/project/ProjectInitializationPage.test.ts` 的既有 Prettier 差异，不将其混入本任务交付。

## Conclusion

代码级、模拟/应用层、数据库 schema/数据和现有 local 路径验证通过；导航、Environment 共用语义、Pipeline Run 目标快照、Repository 删除前当前绑定校验及 Deployment 历史引用规范均与 Intent/Plan 对齐。

本次验证不能标记为无条件完成：Windows/Linux 真实 SSH E2E 尚未执行，且仓库级 `task check` 受无关的既有 Web 格式差异阻塞。建议在补齐两个真实目标并复跑后，将本验证文档从 `Draft` 更新为 `Accepted`。
