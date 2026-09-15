# 部署密钥归 Environment
最后修改时间: 2026-09-15 14:48:59

Review status: Accepted

Mode: standard

## Background

SSH Environment 的部署密钥目前存在 `credential` 表，类型 `deployment_ssh_private_key`，名称 `deployment-ssh`。Environment 和 Deployment 用 `ssh_credential_id` / `revision` 指向它。同一张表还存 Git 仓库用的 `gitea_token`、`github_token` 等。凭据页、导出导入、名称冲突把两类东西搅在一起。

部署密钥是 SSH 环境用的钥匙，不是用户在凭据页维护的仓库登录信息。Fernet 密钥已经在全局配置 `jwt.secret_key` 里，用来加密落库字段；不需要、也不应该给每个 Project 或 Environment 再生成一把 Fernet。

本阶段拆存储：新建 `environment_credential` 存 SSH 环境密钥（密文）；原 `credential` 改名为 `repository_credential`，只服务仓库绑定。数据搬家用离线迁移，应用里不写新旧表双读或兼容分支。不实现环境导出导入命令。

## Goal

1. 新增 `environment_credential`：公钥、私钥分列。`public_key` 明文；`encrypted_private_key` 为 SSH 私钥经全局 Fernet 加密后的密文；另有 revision。加密用现有 `jwt.secret_key`。表里不存 Fernet 密钥。
2. SSH Environment 通过普通关联指向 `environment_credential`（现有 `ssh_credential_id` 一类绑定即可）。local Environment 不关联。不额外加「每个 Project 只能有一行」的唯一约束或校正逻辑。
3. Probe、TargetResolver、Linux bootstrap、Wizard/环境页公钥接口只读 `environment_credential`，不再读通用凭据表，不再走 `CreateDeploymentSSHCredential`。
4. 原表 `credential` 改名为 `repository_credential`。仓库 `git_credential_id` 指向新表。HTTP/前端/proto/Go 标识一并改名。应用不保留旧表名、旧路径、旧类型的读取分支。
5. `repository_credential` 只保留仓库凭据类型（`git_ssh`、`github_token`、`gitee_token`、`gitea_token`、`registry_token`）。去掉 `deployment_ssh_private_key` 类型和 `deployment-ssh` 受管名称。
6. HTTP/MCP/页面/日志不返回环境私钥；公钥可以返回。仓库凭据页不再出现部署密钥。
7. Deployment 快照继续记录所绑定环境密钥的 id 与 revision，指向 `environment_credential`。
8. 三库用离线迁移：把部署密钥行拷到 `environment_credential`（密文原样拷即可，同一实例 Fernet 不变）、改绑定、删除旧部署密钥行、表改名、修正仓库 FK。
9. Windows 初始化命令在 Environment 保存前仍能取公钥：需要时创建 `environment_credential` 并在保存 SSH Environment 时关联。不写回 `repository_credential`。
10. 为后续环境导出导入分开表，本阶段不实现传输命令。

## Non-goal

- 不实现环境 JSONL 导出/导入。
- 不给 Project/Environment 生成或保存 Fernet 密钥。
- 不在 `environment_credential` 上增加 Project 级唯一、数量上限或其它「防止误用」的额外校正。
- 不把仓库凭据迁入 Environment 或 `environment_credential`。
- 不让 HTTP/MCP/页面返回环境私钥。
- 不改变 local/SSH 目标模型、host+port 独占、Probe 新鲜度、Wizard 三步或 Gateway 创建。
- 不在应用层写新旧表双读、双写、类型别名或占位符运行时修复。数据问题离线迁移处理。

## User scenarios

1. 用户在 Wizard 选 Windows SSH，点「生成初始化命令」。后端创建或读取关联的 `environment_credential`，返回公钥。凭据页没有这条记录。
2. 用户「下一步」保存 SSH Environment，关联该密钥行。Probe 和部署用这一行解密出的 SSH 私钥。
3. 用户管理 Git 仓库凭据：走 `repository_credential` 与 `/api/repository-credential`。该入口不能创建部署 SSH 密钥。
4. 已排队 Deployment 在关联密钥 revision 变化后失败，要求重新部署。
5. 升级已有库：离线迁移后 SSH Environment 仍用原来的 SSH 钥匙（密文未改）；仓库凭据在 `repository_credential`。

## Acceptance

- [x] 存在 `environment_credential`：`public_key` 明文、`encrypted_private_key` 为全局 Fernet 加密的私钥、以及 revision。不存 Fernet 密钥。
- [x] SSH Environment 关联 `environment_credential`；local 不关联。关联按普通业务外键处理，不为 Project 再加唯一校正。
- [x] Probe、TargetResolver、bootstrap、部署公钥接口只使用 `environment_credential`。
- [x] 取公钥在需要时创建或读取关联行，不写 `repository_credential`。
- [x] `credential` 改为 `repository_credential`；仓库 FK、Go/HTTP/前端同步改名。proto 包名仍为 `orbit.v1.credential`（仓库凭据语义）。
- [x] API 为 `/api/repository-credential`（单数）；前端路由同步。无旧路径。
- [x] 仓库凭据类型不含 `deployment_ssh_private_key`。
- [x] Deployment 快照指向 `environment_credential` 的 id+revision。
- [x] `000042` 只做 DDL（建表、改名、改仓库 FK）。已有部署密钥行不写进迁移脚本或应用；离线处理。应用无兼容分支。
- [x] HTTP/MCP/日志无私钥 PEM；凭据 UI 无部署密钥。
- [x] 测试覆盖新表读写、仓库凭据改名、Probe/TargetResolver 用 `environment_credential`。不断言已删除的 `credential` 表或 `deployment_ssh_private_key` 类型。
- [x] `task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 通过。

## 表职责

| 表 | 职责 |
| --- | --- |
| `environment_credential` | SSH 环境密钥。`public_key` 明文；`encrypted_private_key` 用全局 Fernet 加密。 |
| `repository_credential` | 用户维护的仓库/registry 登录。 |
| `environment` | 部署目标。SSH 时关联一条 `environment_credential`。 |
| `deployment` | 快照环境与当时绑定的环境密钥 id+revision。 |

全局 Fernet 仍在 `jwt.secret_key`。`environment_credential` 与 `repository_credential` 的密文都用它加密。跨 Orbit 传输时才用源密钥解密、目标密钥再加密；同实例迁表不必解密。

## Follow-on：环境导出导入还要连带做什么

1. 改 `database_transfer.py`：导出 `environment` + 关联的 `environment_credential`。源实例用全局 Fernet 解开 SSH 私钥写入文件；导入到另一套 Orbit 的目标 Project 时，用 **那一套** 的 `jwt.secret_key` 再加密后写入其 `environment_credential`。
2. 不导出 `repository_credential`、Project 行、Gateway 绑定、磁盘 workspace、容器。
3. 从 Orbit-A 的 Project-1 导入到 Orbit-B 的 Project-2 是预期路径。
4. 导入前只断言 **目标 Project 没有 Environment**。有则拒绝。不另写一套「实例上 host+port 是否占用」的传输规则。
5. 写入目标时走与 Wizard/保存环境相同的应用逻辑。现有 `EnvironmentByTarget` 若仍存在，是保存路径上的字符串匹配（其它 Project 的 `target_type` + `host` + `port` 列值），不是探测远程机是否已被占用。
6. 只导 SSH。同一把 SSH 私钥导入后远程不必重跑初始化命令。
7. 仓库凭据继续用改名后的 repository_credential 导出/导入，与环境传输分开。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard。
- 本阶段只拆表和改名，不实现环境导出导入。
- Fernet 密钥保持全局配置，不为 Project/Environment 各生成一把。
- `environment_credential` 是普通关联表，不为「每项目一行」加唯一约束或校正逻辑。
- 原 `credential` 改名为 `repository_credential`；HTTP `/api/repository-credential`。应用不写兼容层；旧数据离线迁移，密文原样拷到新表（同实例 Fernet 不变）。
- HTTP/MCP/UI 对环境密钥只给公钥。
- `environment_credential` 公钥、私钥分列；取公钥读 `public_key`，不解密私钥。

## Risk

- 三库离线迁移步骤多（建表、拷行、改名、改 FK），SQLite 常要重建表。不做应用内兜底，迁移失败就是失败。
- 前端/proto/SQLC/Go 改名漏改会编译失败或打到旧路径。
- 离线把旧 JSON 密文拆成公钥列 + 仅私钥密文时，migrate 必须能读到 `jwt.secret_key`。解不开的占位符行不迁，绑定清空。

## User review notes

- 用户确认先做密钥归属，导出导入后做。
- 用户接受 `environment_credential`，原表改名为 `repository_credential`。
- 用户确认 Fernet 在全局配置，不必按项目/环境生成；表里只放被它加密的 SSH 私钥。
- 用户要求不要写「一个 Project 最多一行」这类过量校正，按普通业务关联处理。
- 用户要求不写业务兼容逻辑，数据离线处理。
- 用户确认跨 Orbit、跨项目导入环境是后续合理业务。
- 用户确认导入只应断言目标 Project 没有 Environment；不另发明「host+port 已被占用」的传输判断。保存环境若仍有 `EnvironmentByTarget`，那是写入路径上的列值匹配，不是传输阶段的独立规则。
- 用户确认 `environment_credential` 公钥、私钥分开存。
