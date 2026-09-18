# Linux SSH 初始化命令与受管密钥流程实施计划
最后修改时间: 2026-09-17 22:16:18

Review status: Accepted

Mode: standard

## Intent basis

依据 [Intent](../intent/20260917-linux-ssh-initialization-command.md)，初始化命令是 Linux 和 Windows SSH Environment 首次生成受管密钥的唯一显式动作。该动作接收已完整填写并通过校验的当前 SSH target，原子保存 target 与 credential binding；再次展示命令时复用同一有效密钥。保存、编辑、裸 SSH 可达性检查和 Probe 不得产生或轮换密钥。

## Implementation steps

### 1. 收敛受管密钥生命周期

- 在 `internal/application/environment/usecase/ssh_key.go` 增加仅供初始化命令准备动作使用的“读取当前有效 binding，否则创建新 Environment credential”帮助方法。它只复用当前 Environment 绑定且项目、revision 均有效的 credential；缺失或失效 binding 才创建新密钥，不从 Project 的“latest credential”猜测或复用未绑定记录。
- 将 `SaveInitialization` 与 `UpdateForUser` 改为只保存 Environment target。SSH target 允许处于未初始化状态，即没有 credential binding；保存 target 不调用 `ensureGeneratedCredential`，不进行远程 SSH 连接。
- 调整 Environment 校验边界：持久化 target 配置允许未初始化 SSH target；初始化命令准备、Probe 成功、TargetResolver 与部署运行时继续要求完整且有效的 binding。
- 修改 `ProbeForUser`：移除 `ensureGeneratedCredential`。遇到未初始化 SSH target 时记录 target revision 对应的失败 Probe 状态，返回“请先生成并执行初始化命令”的诊断，不调用 runner，也不修改 credential。有效 credential 时保留当前 pin host key、条件写入 Probe 状态和底层 runner 诊断的行为。
- 保持 target 变更的 revision 和 host-key 清理语义。初始化命令以提交表单覆盖 target；host、port、平台或 SSH 用户变化时清除 host key 并递增 target revision，workspace 变化同样使 Probe 失效。有效 credential 保持原 id/revision。

### 2. 用统一的 SSH 初始化命令准备动作替换 Windows 专属路径

- 将 `PrepareWindowsEnvironment` 重构为平台无关的 `PrepareSSHEnvironment`（名称以实现时现有命名约定为准）。输入为完整 `SSHTargetInput`，仅接受 `linux` 或 `windows`。
- 该动作加载或创建当前 Project Environment，应用提交 target，执行 target 独占检查，并在同一 HTTP 写事务中创建缺失 credential、更新 binding 与写入 Environment。它不进行任何远程 I/O。
- 有效绑定存在时直接返回已有 `public_key`，不调用 `replaceEnvironmentCredential`、不递增 credential revision、不替换私钥。删除 `replaceWindowsCredential` 以及“每次准备都覆盖完整 key pair”的测试和语义。
- 将项目初始化 HTTP、proto、DTO、store 和前端 API 从 `windows-command` / `ProjectInitializationWindowsCommandResp` 收敛为通用 SSH 初始化命令请求与响应；删除旧 Windows 专属路径，不保留兼容路由。
- Environment detail 的初始化命令 API 同样使用通用准备动作，接收当前已校验 target 并返回 Environment view 与 public key。它与 Wizard 共享相同的领域服务，不复制密钥写入逻辑。

### 3. 移除 Linux 临时认证 bootstrap 与裸连通性测试

- 删除 `InitializeForUser`、`Bootstrapper` port、Linux bootstrap runner、临时密码/私钥 DTO、`/api/environment/initialize` 与 `/api/project-initialization/bootstrap`，以及对应 proto、handler、前端表单、页面状态和测试。应用不再接收一次性 SSH 密码或个人私钥。
- 删除初始化向导中的 `TestSSHReachability` 入口，以及 `Prober.TestSSH` 的无认证实现。现有实现把认证失败解释为“服务可达”，不满足受管部署连接的成功语义。
- Probe 保持为唯一的 SSH 检查：它必须使用 Environment 受管私钥完成公钥认证，随后执行平台 Docker/Compose 前置条件检查。保留当前握手原始错误透传至 `last_probe_diagnostic` 和结构化日志的改动，不记录私钥或口令。

### 4. 生成 Linux Bash 初始化命令

- 新增 `web/src/utils/linuxSshCommand.ts`，根据返回的 public key 生成 POSIX Bash 命令。命令使用严格 shell 语义并幂等地：创建 `$HOME/.ssh`、确保目录和 `authorized_keys` 权限、仅在该完整公钥尚不存在时追加一行。
- 命令只能操作执行用户的 `$HOME/.ssh/authorized_keys`，不接受任意用户、文件路径或远端命令输入；不安装、启停或修改 `sshd`、Docker、Compose、Docker 用户组或防火墙。
- 抽象现有 `WindowsSshInitializationDialog` 为平台无关的初始化命令 Dialog，命令内容由平台选择 Windows PowerShell 或 Linux Bash 生成器。页面不展示或持久化私钥。

### 5. 收敛 Wizard 与 Environment 页面工作流

- `ProjectInitializationPage.vue` 对 Linux 与 Windows SSH target 均在完整表单校验后提供初始化命令动作；该动作调用统一准备 API，持久化当前表单，更新 status，并展示相应脚本。
- 删除 Linux bootstrap 密码/私钥输入区和“先测试 SSH 服务”的步骤。用户执行命令后以现有 Probe 动作验证 Orbit 受管认证及运行时前置条件。
- `EnvironmentPage.vue` 使用同一通用准备 API 和命令 Dialog。编辑 SSH target 后，初始化命令基于当前已保存/提交的完整 target 重新展示；local target 不显示该入口。
- 更新 API、Pinia store、proto imports 与中英文文案，使“初始化命令”和“Probe”语义清楚区分，且未初始化的 Probe 诊断可直接恢复。

### 6. 更新活文档

- 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` 与 `docs/guides/deployment.md`：SSH Environment 可先保存为未初始化 target；受管密钥仅由显式初始化命令产生；Linux/Windows 均为“生成并执行命令 -> Probe”，但 Linux 命令只管理当前用户的公钥授权。
- 保留旧过程文档作为历史记录，不改写已接受的 Intent、Spec、Plan 或归档材料；现行文档以本 Intent 与代码完成后的行为为准。

## Files to change

- `internal/application/environment/usecase/service.go`
- `internal/application/environment/usecase/ssh_key.go`
- `internal/application/environment/usecase/probe.go`
- `internal/application/environment/usecase/windows_initialization.go`（重命名/收敛为通用准备动作）
- `internal/application/environment/port/prober.go`、`internal/application/environment/port/bootstrap.go`
- `internal/infrastructure/runner/ssh/environment_probe.go`、`internal/infrastructure/runner/ssh/environment_bootstrap.go`
- `internal/application/project_initialization/**`
- `internal/api/http/handler/environment/**`、`internal/api/http/routes/environment.go`
- `internal/api/http/handler/project_initialization/**`、`internal/api/http/routes/project_initialization.go`
- `proto/orbit/v1/environment/environment.proto`
- `proto/orbit/v1/project_initialization/project_initialization.proto`
- `web/src/api/project/environment.ts`、`web/src/api/project/initialization.ts`
- `web/src/stores/projectInitialization.ts`
- `web/src/views/environment/EnvironmentPage.vue`
- `web/src/views/project/ProjectInitializationPage.vue`
- `web/src/components/WindowsSshInitializationDialog.vue`（重命名/替换）
- `web/src/utils/windowsSshCommand.ts`、新增 `web/src/utils/linuxSshCommand.ts`
- `web/src/views/environment/EnvironmentBootstrapFields.vue`、`web/src/views/environment/environmentBootstrapForm.ts`（删除）
- 相关 Go、前端测试与 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`

## Verification plan

1. Go use-case/integration tests：保存或编辑 SSH target 不创建 credential；首次 Linux/Windows 初始化命令原子保存 form、创建 credential 并返回公钥；重复调用在相同 binding 下返回同一公钥且 revision 不变；target 改动清除 host key、递增 target revision 且复用 credential。
2. Probe/runner tests：未初始化 target 不触发 SSH runner 并记录明确诊断；认证失败时保留底层握手错误；成功路径使用受管私钥并照常记录 host key；不再存在无认证 `TestSSH` 或 bootstrap password/private-key 入口。
3. HTTP/proto tests：通用初始化命令 API 接收 Linux/Windows form，响应仅含 public key 与 Environment/status；删除旧 bootstrap 与 Windows 专属路由；响应、日志和测试断言不包含私钥或口令。
4. 前端 tests：Linux 命令为可执行的幂等 POSIX shell，Windows 保持 PowerShell；Wizard 与详情页从相同准备结果展示正确命令；local 不显示初始化入口；未初始化时 Probe 显示后端诊断。
5. 生成与检查：`task proto`、`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。

## Blockers

无。Linux 用户执行初始化命令的带外访问（云控制台、已有 SSH 登录等）是产品流程外部前置条件。

## Assumptions

- 受管 SSH 密钥继续每个 Environment/Project 一组，保存在既有 `environment_credential`，不修改已执行迁移。
- Environment detail 编辑器可在保存 target 后调用准备动作；Wizard 准备动作直接携带完整表单。两者最终调用同一领域方法。
- 已有完整有效 credential binding 可在平台、host、port、username 或 workspace 变更后继续复用；用户负责在新的目标/用户侧执行返回的命令。
- 有效 binding 的定义为 credential id、Project id、credential revision 与 Environment binding 一致；不以“项目最近一条 credential”作为替代。

## Risks

- 移除 bootstrap API 会改变现有调用方；仓库处于活跃开发期，按当前设计直接删除，不保留旧路由或 DTO 兼容层。
- Linux Bash 命令的 shell quoting 与幂等性错误会导致用户机器授权失败，必须以单元测试和 POSIX shell 语法检查覆盖。
- 在初始化命令未执行前 Probe 必然失败；必须让 API/UI 诊断与真实的网络、认证和 Docker 错误区分清晰。
- 复用同一密钥跨 target 是有意行为；target revision 与 host-key pinning 的清理必须可靠，避免旧 Probe/部署结果作用于新 target。

## Rollback

- 代码层面可回退本次变更；远端已追加的公钥不会自动删除，避免误删用户手工授权。
- 不提供运行时旧 bootstrap 或旧 Windows 路由 fallback；需要撤销时以完整代码版本回退并由用户重新生成当前初始化命令。

## User review notes

- 用户确认 Intent，要求进入计划阶段。
- 用户确认初始化命令必须在完整 Environment 表单之后生成，因平台决定 Bash 或 PowerShell。
- 用户确认初始化动作持久化当前 Environment target；页面刷新后回填表单，并复用已有有效密钥。
- 用户确认意图和计划已足够明确，要求进入实施阶段。
