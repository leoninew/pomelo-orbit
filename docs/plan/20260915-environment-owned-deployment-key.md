# 部署密钥归 Environment 实施计划
最后修改时间: 2026-09-15 15:00:00

Review status: Accepted

Mode: standard

## Implementation steps

### 1. Schema 与离线迁移

- 三库新增 `000042_environment_credential`。
- 建 `environment_credential`：`id`、`project_id`、`public_key`、`encrypted_private_key`、`revision`、`created_at`。不加 Project 级唯一约束。
- 从 `credential` 中 `type = deployment_ssh_private_key` 且非占位符的行，按原 `id` 插入新表：密文先拷到 `encrypted_private_key`，`public_key` 暂空。
- migrate 已加载 `jwt.secret_key`：离线解开旧 JSON payload（`private_key` / `public_key` / `passphrase`），写入明文 `public_key`，并把 `encrypted_private_key` 改成只加密私钥字符串。占位符行不迁，对应 Environment 绑定清空。
- `environment.ssh_credential_id` / `deployment.ssh_credential_id` 仍指向同一 id，改为关联 `environment_credential`。
- 删除已迁走的 `deployment_ssh_private_key` 行。
- 将剩余 `credential` 改名为 `repository_credential`，修正 `repository.git_credential_id` FK。
- 同步 `sql/schema/{sqlite,mysql,postgres}.sql`。SQLite 按现有重建表方式处理改名与 FK。
- `task sqlc`：新增 `sql/query/environment_credential`；`sql/query/credential` 改为打 `repository_credential`。

### 2. Environment 密钥读写

- 新增 Environment 侧仓储/usecase：创建密钥对、按 id 读公钥、解密私钥。生成逻辑从 `credentialsvc.generateDeploymentSSHKeypair` 挪到 Environment 域。
- Wizard/环境页 `GET .../deployment-key`：需要时插入 `environment_credential` 并返回 `public_key`；保存 SSH Environment 时绑定该 id+revision。不写 `repository_credential`。
- Probe、TargetResolver、Linux bootstrap 只从 `environment_credential` 解密私钥。
- 删 `CreateDeploymentSSHCredential`、`CredentialTypeDeploymentSSHPrivateKey`、`deployment-ssh` 名称逻辑。
- HTTP/MCP/日志仍不输出私钥；公钥接口只返回 `public_key`。

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

- `sql/schema/*.sql`，`sql/migration/{sqlite,mysql,postgres}/000042_*.sql`，`sql/query/environment/*.sql`，新建 `sql/query/environment_credential/`，`sql/query/credential/` → repository_credential，`sqlc.yaml`
- `internal/bootstrap` 迁移后的离线公钥回填（可读 config 的 Fernet）
- `internal/model`，`internal/application/environment/**`，删除/收缩 `internal/application/credential/usecase/deployment_ssh.go`
- `internal/application/deployment/usecase/remote_target.go` 及测试
- `internal/repository/impl/sqlc/**`，`internal/api/http/**`，`proto/orbit/v1/**`，`web/src/**` 凭据路由与 API
- 相关 Go/Web 测试；`docs/product/cd-model.md`、`docs/guides/deployment.md` 中部署密钥表述

## Verification plan

1. 迁移：空库、仅仓库凭据、已绑定 SSH 密钥、占位符行；断言新表分列、旧部署密钥行消失、仓库 FK 有效。
2. 公钥：无 Environment 时生成 `environment_credential` 并返回 `public_key`；再保存 SSH Environment 绑定同一 id；不写 `repository_credential`。
3. Probe/TargetResolver 用新表私钥；响应/MCP/日志不含 PEM。
4. 仓库凭据 CRUD 走新路径；拒绝 `deployment_ssh_private_key`。
5. 部署快照 revision 变化后拒绝执行。
6. `task proto`、`task sqlc`、`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`。

## Blockers

无。需要 `jwt.secret_key` 才能离线拆旧 JSON 密文；migrate 入口已 `config.Load()`。

## Assumptions

- 绑定列名可继续用 `ssh_credential_id`，只改指向的表。
- 旧 `encrypted_data` JSON 含 `public_key`/`private_key`；离线回填后应用只读写分列。
- 本阶段不改 `database_transfer.py` 的服务闭包表集合。

## Risks

- 三库改名 + 新表，SQLite 重建表易错；失败不靠应用兜底。
- 离线回填漏跑则 `public_key` 为空，Windows 命令会拿到空公钥。
- proto/前端改名漏改会打到已删除的 `/api/credential`。

## Rollback

执行 `000042` down：仓库 FK 改回、表改回 `credential`、把 `environment_credential` 行按原 id 写回 `deployment_ssh_private_key`（需把分列再封回 JSON 密文）、删新表。不承诺恢复占位符行。回退二进制。

## User review notes

- 用户要求进入计划 / Plan。
- 用户确认公钥、私钥分列存储。
