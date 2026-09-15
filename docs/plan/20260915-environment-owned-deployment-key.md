# 部署密钥归 Environment 实施计划
最后修改时间: 2026-09-15 22:23:06

Review status: Accepted

Mode: standard

## Implementation steps

### 1. Schema 与离线迁移

- `000042_environment_credential` 已执行，禁止修改该迁移，也不新增离线数据迁移。
- 保持现有 `environment_credential` 表与 `environment.ssh_credential_id` / `ssh_credential_revision` binding；新增受管密钥覆盖 SQL，用同一 `id` 更新公钥、加密私钥和 revision。
- `000042` 遗留的 `environment.ssh_credential_id -> repository_credential` 引用不在状态读取时自动修复。Windows 初始化命令准备动作在当前 Project 内检测该失效 binding，创建新的 `environment_credential` 并覆盖 binding。
- `task sqlc` 重新生成新增更新查询的仓储代码；不改三库 schema 或迁移文件。

### 2. Environment 密钥读写

- Environment 侧负责创建密钥对、以全局 JWT `secret_key` 加密私钥、按 binding 解密私钥；不写 `repository_credential`。
- `POST /api/project/:project_id/initialization/environment/windows-command` 接收当前 Windows SSH form。它在现有 HTTP 写请求事务中创建缺失 Environment/credential，或直接覆盖已有 Environment target 与当前项目密钥；失效的 `000042` binding 创建新 credential 并重新绑定。仅返回新 `public_key` 和当前初始化状态。
- Wizard 和环境页都调用该 POST，再以返回公钥生成 PowerShell。生成命令不执行远程 I/O，也不要求 SSH 已连通；用户在远端执行后显式检查连接，再运行 Probe。
- Probe、TargetResolver、Linux bootstrap 只从 `environment_credential` 解密私钥。
- 删 `CreateDeploymentSSHCredential`、`CredentialTypeDeploymentSSHPrivateKey`、`deployment-ssh` 名称逻辑。
- HTTP/MCP/日志仍不输出私钥；不保留独立公钥读取接口或密钥轮换 command。

### 3. 仓库凭据改名

- Go：`internal/application/credential`、HTTP handler/routes、sqlc 包、proto `orbit.v1.credential` 改为 `repository_credential` 语义。
- HTTP：`/api/credential` → `/api/repository-credential`（单数）。无旧路径。
- 前端：`web/src/api/credential`、`views/credential`、router `/credential`、navigation 同步改名。
- `validCredentialType` 只保留仓库类型。
- `task proto` 后确认 generated 干净。

### 4. 部署快照

- Deployment 继续快照 `ssh_credential_id` + `ssh_credential_revision`，语义改为 `environment_credential`。
- `applyDeploymentTargetSnapshot` / `verifyDeploymentTargetSnapshot` 对照新表 revision。
- 不写运行时兼容；迁移前未跑完的任务按目标不一致失败。

### 5. 文档与测试

- 更新活文档中「受管 deployment-ssh 凭据」表述为 Environment 关联的 `environment_credential`。
- 不改服务传输表集合（本阶段不实现环境导出）。
- 测试见 verification plan。

## Files to change

- `sql/query/environment_credential/`、`internal/repository/impl/sqlc/environment_credential/`
- `internal/application/environment/**`、`internal/application/project_initialization/**`
- `internal/api/http/**`、`proto/orbit/v1/project_initialization/**`、`web/src/**` 初始化 API、store 与 Windows 命令入口
- 相关 Go/Web 测试；`docs/product/cd-model.md`、`docs/guides/deployment.md` 中部署密钥表述

## Verification plan

1. Windows command preparation：无 pair 时创建 Environment/credential；完整 pair 时在原 credential 上覆盖密钥并增加 revision；`000042` 失效 binding 时创建新 credential 后重新绑定。
2. 状态读取：遇到 legacy binding 仍能进入 Wizard，不自动生成 credential；准备成功后状态为 `needs_probe`。
3. HTTP/proto/Web：只有携带 Windows form 的 POST；响应只含公钥与状态；Wizard 生成命令后仍要求显式 SSH 检查才能进入下一步。
4. Probe/TargetResolver 用新表私钥；响应/MCP/日志不含 PEM。
5. `task proto`、`task sqlc`、`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。

## Blockers

无。运行时准备命令需要有效的全局 `jwt.secret_key` 才能生成并加密部署私钥。

## Assumptions

- 绑定列继续使用 `ssh_credential_id` / `ssh_credential_revision`，只指向 `environment_credential`。
- 历史 `repository_credential` 载荷不再作为 Environment 运行时 fallback；用户生成 Windows 初始化命令时以新密钥修复。
- 本阶段不改 `database_transfer.py` 的服务闭包表集合。

## Risks

- 准备动作必须运行在现有 HTTP 写请求事务中，避免 credential 更新和 Environment binding 半成功。
- 用户尚未执行远端命令时 SSH 检查会失败；这是预期流程，生成命令不能以连接成功为前提。
- proto/前端改名漏改会打到已删除的 `/api/credential`。

## Rollback

重新生成 Windows 初始化命令会以新 key 和当前 form 覆盖受管 Environment binding；不提供独立密钥轮换或旧 `repository_credential` 回退路径。

## User review notes

- 用户要求进入计划 / Plan。
- 用户确认公钥、私钥分列存储。
- 用户确认生成 Windows 初始化命令是 Environment 与项目级密钥的持久化准备动作：Environment/credential 缺失或 binding 失效时修复，完整 pair 时覆盖；不在未保存 Environment 前读取 credential，也不添加独立轮换功能。
