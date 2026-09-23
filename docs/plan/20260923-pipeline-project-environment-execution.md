# CI/CD 共用 Project Environment 与流水线远端运行计划
最后修改时间: 2026-09-23 15:45:00

Review status: Accepted

Mode: standard

## Intent basis

依据 `docs/intent/20260923-pipeline-project-environment-execution.md` 的修订目标：Environment 从持续部署的导航与产品语义中抽出，作为 CI/CD 共用的 Project 执行目标；系统设置承载其管理入口，CI 增加 SSH 目标执行。标准模式不额外创建 Spec。现行产品/指南仍只描述已实现能力，目标不提前写成现状。

## Delivery strategy

建议 **一个 feature、四个顺序 review 的实现批次**：① 共用环境归属、独立设置入口与前端就绪引导；② Run 目标契约及后端就绪/排队安全；③ Windows SSH 执行、远端日志/制品及其独立验证；④ Linux SSH 执行及其独立验证、跨平台回归。批次一能独立验收管理入口；批次二为 Windows 和 Linux 的共用目标约束；必须先完成批次三的实现和验证，再进入批次四，不把 Linux 当作 Windows 功能的预验证。

用户已确认各项产品边界：每 Project 一个 Environment；独立系统管理入口；local 与 SSH 的 CI/CD 操作都需最新 Probe，Web 未就绪返回初始化页面；远端日志和文件制品不回传；本地目录 Repository 沿用 local 语义而不增传输；Windows 先于 Linux。现有跨 Project Docker target 冲突校验不因本需求改变，同一 target 的多 Project 协同写入不纳入范围。

## Implementation steps

### 批次一：共用环境与就绪入口

1. Environment 保留 Project 1:1 归属，CI/CD 读取同一配置；现有 target 冲突校验保持原样。不把环境当 CD 独有，也不改造成全局资源或复制第二份配置。
2. 将环境管理页面作为系统管理的独立路由/菜单入口，移除持续部署菜单中的环境入口；调整 `getNavigationScope`、页面标题、项目详情快捷入口、i18n 和导航测试。URL 可保留 `/environment`，但路由不依附现有 `/settings` 页，也不自动继承 `SETTING_READ` 权限。
3. 在 CI 触发/重试和 CD 部署的 Web 动作前实时检查当前 Project 环境的保存状态、工作目录和最新 Probe；不就绪则导航到 `/project/:id/initialization`。现有向导仍处理 Environment → Probe → Gateway；`needs_gateway` 表示环境可用于 CI，而不是 CI 不就绪。CD 的 Gateway/default Service 额外条件仍单独检查。前端重定向不是后端安全边界。
4. 活文档区分 Environment 共用语义与 CD Gateway 的专有语义；系统设置入口与远端 CI 未实现前不写成已交付能力。

### 批次二：目标契约、后端就绪与排队安全

5. CI 触发/重试在后端验证所属 Environment 存在、工作目录有效、最新修订的 Probe 成功（含 local），SSH 另验证受管密钥和 host-key fingerprint；CD 保留或复用其现有后端就绪校验。无 Gateway 但 Environment 就绪的 CI 可运行；无 Gateway 的 CD 仍按原规则拒绝。MCP/API 绕过前端时亦不可跳过校验。
6. Run 持久化 Environment 身份、target type/revision 与仅 SSH 凭据身份/revision；为既有 Run 的读取和删除确定明确规则。用新增迁移与 SQLC 查询/模型实现，不修改已执行迁移，不保存私钥。
7. Worker 在外部副作用前校验 Run 目标快照，环境/凭据发生变更就标记任务失败并提示重试；Retry 创建新 Run，按当前目标重新解析。Stage 之间不静默切换目标。
8. 把 Workspace/ContainerRunner/日志存储的注入边界调整为以 Run 目标为准，保留 local Docker 行为和已有 Pipeline Snapshot、变量、DAG、Version fork 语义。local 本地目录 Repository 仍解析控制面路径并只读挂载 `/source`；SSH + 本地目录的组合保持不可运行、不增加传输，在触发前明确拒绝，不回退 local。

### 批次三：Windows SSH 执行与验证

9. 实现 Windows native OpenSSH + WSL2 Docker Desktop 的 Pipeline Runner，复用受管 SSH 密钥与 host-key pinning；Windows 路径、命令引用、Docker Linux containers 的宿主路径/挂载和 socket 用实测约束设计，不直接复用 CD Compose 接口或控制面 Docker CLI。
10. 远端 `<workspace_root>/pipeline` 管理源码工作区、Stage 日志与 Run 文件制品：SSH 增量日志读取、文件制品存在性/远端位置元数据、命令制品、目标 Docker daemon 的镜像 ID 检查、取消/超时后的容器清理与 Run 删除的远程文件清理。不回传日志/制品或增加文件下载接口。
11. 为 Windows 的正向 Stage/DAG、日志、各类制品、错误与取消路径补齐测试；使用 Windows native OpenSSH + WSL2 Docker Desktop 可访问目标做端到端验证，并记录前置条件与结果。Windows 行为通过后再开始 Linux 功能实现。

### 批次四：Linux SSH 执行与验证

12. 在已验证的 Run 目标/远端工作区契约上增加 Linux OpenSSH + Docker 的目标适配；独立处理 Linux shell、路径、Docker socket 与 bind mount，不把 Windows 路径规则写成跨平台常量。
13. 为 Linux 增加同等的执行、增量日志、制品、取消/超时与删除测试；使用可访问的 Linux 目标进行端到端验证，并做 local/Windows/Linux 跨平台回归。若缺目标环境，明确标注未验证，不能宣称最终验收完成。
14. 更新 Run 详情、错误提示及活文档，说明 Environment 共用配置、运行前 Probe、Windows/Linux 远端运行、远端日志/制品驻留与本地目录型 Repository 的边界。

## Files to change (expected)

- 环境/初始化与后端就绪：`internal/application/project_initialization/{dto,usecase}/`、`internal/application/environment/`、`internal/application/pipeline_run/usecase/`；复用现有 Project/Environment 服务，不创建新环境模型。
- 前端入口与就绪引导：`web/src/navigation.ts`、`web/src/navigation.test.ts`、`web/src/router/index.ts`、`web/src/views/environment/EnvironmentPage.vue`、`web/src/views/project/{ProjectDetail,ProjectInitializationPage}.vue`、`web/src/views/{pipeline,pipeline_run,deployment,service,gateway}/` 的运行/部署操作入口、`web/src/i18n/locales/` 与相关测试。
- CI 目标与运行：`internal/application/pipeline_run/port/port.go`、`internal/application/pipeline_run/usecase/{service,execution,executor}.go`、`internal/bootstrap/{http,worker,pipeline_workspace}.go`。
- 持久化：`internal/model/pipeline_run.go`、`internal/repository/impl/sqlc/pipeline_run/`、`sql/query/pipeline_run/`、`sql/migration/{mysql,postgres,sqlite}/` 的新增迁移和生成代码。
- Windows/Linux 目标适配：`internal/infrastructure/runner/pipeline/`、`internal/infrastructure/runner/ssh/` 的受管连接和平台命令/路径能力、目标感知的 workspace/log/file artifact 存储适配。
- 对外契约/活文档依最终设计：`proto/orbit/v1/pipeline_run/`、`internal/api/http/handler/pipeline_run/`、`web/src/views/pipeline_run/`，以及 `docs/product/{overview,cd-model}.md`、`docs/INDEX.md`、`docs/architecture/cd-runtime.md`、`docs/guides/ci-pipeline-design.md`。

## Verification plan

1. 批次一：检查独立系统管理入口/导航/项目切换与权限；local/SSH 环境不存在、Probe 过期或失败时，CI Run/Retry 和 CD Deploy 的 Web 动作均进入对应项目初始化页；Environment 就绪但 Gateway 缺失时 CI 可继续、CD 仍拒绝。
2. 批次二：后端和 MCP/API 不依赖前端即可拒绝 local/SSH 未就绪；覆盖目标/凭据修订改变、排队任务拒绝、Retry 重新绑定、local 本地目录 Repository 行为及 SSH + 本地目录组合的明确拒绝。
3. 批次三：Windows SSH Stage/DAG、工作区宿主路径/挂载、增量远端日志、远端文件制品、命令和镜像制品、删除及取消/超时清理；真实 Windows OpenSSH + WSL2 Docker Desktop 目标验证通过后再推进 Linux。
4. 批次四：Linux 独立执行与远端读写测试，真实 Linux OpenSSH + Docker 目标验证；local/Windows/Linux 全量关键路径回归并核对历史 Run 行为。
5. 修改 Go 后运行 `task check`、`go test ./cmd/... ./internal/...`；修改前端后运行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`；如改 Proto 执行仓库生成命令，辅以定向测试和 `git diff --check`。

## Blockers and assumptions

- Windows native OpenSSH + WSL2 Docker Desktop 可访问目标是第三批真实验收前置；Linux OpenSSH + Docker 可访问目标是第四批前置。两者缺失时可运行单元/集成模拟，但不宣称相应平台端到端验收通过。
- “本地目录 Repository 保持当前行为”限定在当前 local 执行路径；控制面绝对路径在 SSH 远端不具备可见性，本计划不增加传输，SSH 与此来源组合保持不可运行，并前移错误提示。
- 每 Project 一个 Environment；现有跨 Project 相同 Docker target 冲突校验保持，不设计多项目共享同一 target 的协同写入或新的全局环境池。
- 既有 Run 没有目标快照；新增字段需明确历史读取和清理规则，不能静默把历史 Run 指向当前可能已变化的 Environment。

## Risks

- 移到系统管理后若误套用 `/settings` 的全局权限，Project 成员可能失去本项目的环境管理能力；就绪引导若只写前端，会让 MCP/API 绕过 Probe。
- 当前 local CI 无 Probe 门槛；新增前置校验会改变行为。Probe 失败时必须阻止启动并让 Web 进入现有初始化页，避免绕过或无限重定向。
- Windows 的 SSH shell/路径、Docker Desktop/WSL2 宿主路径与 Docker socket 和 Linux 不同；Windows 优先实现不能把其路径假设固化为共享运行时。
- 远端日志/制品不可回传时，目标离线或被重配会影响查看与删除；Run 必须保留目标归属，权限和错误说明不能泄露 SSH 密钥。
- 本地目录来源无远程可见路径；把控制面 `/source` 提供给远端会导致错误或越权。命令引用、Git 凭据和 Docker socket 处理需要逐平台检验。

## Rollback

批次一的导航/就绪引导可独立回退而不删除 Environment；批次二以后只通过新增迁移变更 Run 持久化，已执行迁移不得改写。Windows 或 Linux 远端执行出现问题时停止对应新 SSH Run，保留 local Run 的行为与数据；日志/文件制品始终按原目标位置留存或清理，不把远端数据误删为控制面本地文件。

## User review notes

- 用户补充 Environment 是 CI/CD 共用能力，需从持续部署移至系统设置；据此增加可独立验收的“共用环境入口与就绪边界”批次，合计四批。
- 用户确认 Windows 实现/验证先于 Linux、远端日志与文件制品不回传、local 本地目录 Repository 保持原行为、local/SSH CI/CD 运行前做就绪检查并由 Web 引导至现有初始化页。
- Intent 与 Plan 均已接受；依用户“go on”进入 Implementation / 实现，按批次一至四的顺序推进。
- 用户要求继续：接受 Plan 并进入 Implementation / 实现，先实施共用环境与就绪入口批次。

## Implementation progress

- 批次一：环境入口迁至系统管理独立页面；CI Run/Retry 与 CD Deploy 的 Web 操作实时读取 Project 初始化状态，不就绪时引导至现有向导。CI 最新 Probe 成功但 Gateway 缺失时可返回原运行页；已有 Gateway 的过期 Probe 成功后返回原操作页。
- 批次二：触发/Retry 经同一 TargetResolver 检查 local/SSH 当前修订 Probe 和 SSH 受管凭据；新增 Run 环境/凭据目标快照及三数据库迁移，Worker 在副作用前复核并拒绝过期 Run，Retry 重新绑定。读取/删除按快照定位，无法安全定位的历史 Run 拒绝文件访问；运行资源按已校验的目标选择，SSH 不会回退本机。SSH + 本地目录或非 HTTPS Git 在触发前拒绝；在批次三的 Windows SSH 执行器交付前，有效 HTTPS SSH 目标同样明确拒绝触发，不排入注定失败的 Run。local 目录 Repository 沿用既有行为。
- 批次三：Windows native OpenSSH + WSL2 Docker Desktop 的 SSH Pipeline Runner 已实现；远端 `<workspace_root>/pipeline`、SFTP 增量日志、远端文件制品定位与删除、命令/镜像制品、取消后容器清理已接入 Run 目标。真实 Windows 目标的 Runner 烟测（含取消、容器清理、host-key 拒绝）及应用层 Stage/DAG、三类制品集成测试通过；使用目标已有的含 `sh` 镜像 `docker:29.4`，控制面不依赖 Docker CLI。首次烟测因目标镜像拉取凭据失败产生的单个测试目录已核对内容并清理，后续 E2E 要求显式指定目标已有镜像。`go test ./cmd/... ./internal/...`、Go lint（0 issues）、Go 格式及 `git diff HEAD --check` 通过。`task check` 的 Web typecheck/lint 通过，但在未修改的 `web/src/views/project/ProjectInitializationPage.test.ts` 既有 Prettier 差异处中止；未为本批次更改该无关文件。
- 批次四（Implementation）：Linux OpenSSH + Docker 已接入同一 Run 目标选择与远端工作区/日志/制品契约，Linux shell、绝对路径和 bind mount 独立校验；Windows 路径与 PowerShell 仍保持原分支。Run API/详情显示持久化的 Environment ID 与目标类型，不暴露凭据；活文档更新了远端驻留和目录权限要求。Linux 独立路径/命令单元测试、Linux Trigger/Retry 与 API 映射测试，以及 Windows/Linux 真实目标 Runner 与应用层 Stage/DAG E2E 均通过；Go 全量测试、Go lint、Web lint:fix/typecheck 和本次 Web 文件 Prettier 检查通过。真实 Linux Project 的原配置 `<workspace_root>/pipeline` 由 root 持有且权限 0755，SSH 登录用户无法写入：使用仅测试期间的 SSH 用户主目录隔离工作区完成 E2E，并按明确目录白名单清理；**原配置下的 Linux 运行验收仍待目录权限修复与复跑**，不将隔离工作区测试等同于原配置验收。`task check` 在 Web typecheck/lint 成功后仍被未修改的 `web/src/views/project/ProjectInitializationPage.test.ts` 既有 Prettier 差异拦住；本批次修改过的 Web 文件均单独通过 Prettier 检查。Proto 生成通过；`buf lint` 因仓库既有所有 `orbit.v1.*` package 缺少末尾版本段报错，本批次未扩范围重命名。
- 批次二检查：定向用例（触发/重试未就绪、目标/凭据漂移、SSH 源码约束与拒绝本机回退、历史访问、SQLC 往返、SQLite 新迁移与回滚）及 `go test ./cmd/... ./internal/...`、Go lint、`git diff HEAD --check` 通过；`task check` 的 Web typecheck/lint 通过，止于未修改的 `web/src/views/project/ProjectInitializationPage.test.ts` 既有 Prettier 差异，不扩范围修改。
