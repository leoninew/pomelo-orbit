# 部署密钥归 Environment 验收
最后修改时间: 2026-09-15 17:51:27

Review status: Accepted

Mode: standard

## Requirement alignment

本次实现将 SSH 部署密钥从通用 `credential` 领域拆到 `environment_credential`，并将原表改名为 `repository_credential`。Environment 侧保存公钥、Fernet 加密后的私钥和 revision；全局 `jwt.secret_key` 仍是加密密钥，数据库不保存 Fernet 密钥。

SSH Environment、Probe、TargetResolver、Linux bootstrap、部署公钥接口和 Deployment 快照均改为使用 Environment 密钥。仓库凭据继续由 `repository_credential` 管理，HTTP/前端入口同步为 `/api/repository-credential`。HTTP、MCP、页面和日志不返回环境私钥。

## Spec alignment

不适用。当前采用 standard 模式，未单独建立 Spec 阶段文档；本验收按已接受 Requirement 和 Plan 核对。

## Plan alignment

- 三库新增 `000042_environment_credential`，同步 schema、SQL query 和 SQLC 输出；SQLite 的仓库表按现有重建方式修正仓库凭据外键。
- Environment application、repository 和 runner 已迁移到新表；保存 SSH Environment 时维护密钥 id/revision，公钥接口可在 Environment 保存前创建或复用密钥。
- 通用 credential 的部署密钥类型、受管名称和旧读取路径已删除；仓库凭据 Go、HTTP、前端和路由命名已同步。
- Deployment 继续记录环境密钥 id 与 revision；密钥 revision 不一致时按目标绑定失效处理。
- 活文档、需求文档和计划文档已同步 Environment 密钥归属边界。

## Actual diff summary

实际 diff 覆盖数据库迁移与 schema、SQLC 生成代码、Environment/Deployment/Repository application、HTTP 路由、前端凭据页面、测试和活文档。新增 Environment credential 的密钥生成、加密、读取、绑定和解密路径；删除 credential 域中部署 SSH 密钥逻辑。

验收前另补齐了本次触及手写 Go 文件中的内部缩写命名，将 `userID`、`projectID`、`credentialID`、`deploymentID` 等统一为 `userId`、`projectId`、`credentialId`、`deploymentId`。SQLC 生成字段仍保留生成器默认的 `ProjectID` 等形式，未在本次功能变更中进行全仓生成 API 重命名。

## Expected vs actual changed files

预期的 schema/migration、SQLC、Environment credential、Repository credential、Deployment snapshot、HTTP/前端、测试和文档变更均已出现。实际还包含 Windows SSH E2E 辅助流程和生成代码目录调整，均直接服务于保存前读取公钥和新表绑定场景；未发现与本功能无关的产品代码扩展。

## Acceptance criteria checklist

- [x] `environment_credential` 分列保存 `public_key`、`encrypted_private_key` 和 revision，不保存 Fernet 密钥。
- [x] SSH Environment 绑定 Environment credential；local Environment 不创建或绑定环境密钥。
- [x] Probe、TargetResolver、bootstrap、公钥接口只读取 Environment credential。
- [x] Environment 未保存时获取公钥可创建或复用密钥，后续保存复用同一 id/revision。
- [x] 原 `credential` 改为 `repository_credential`，仓库 FK、Go、HTTP、前端和路由同步完成。
- [x] `/api/repository-credential` 为单数路径，旧 `/api/credential` 路径和部署密钥类型不再保留。
- [x] Deployment 快照使用环境密钥 id/revision，revision 变化会阻止使用旧绑定。
- [x] `000042` 仅承担三库 DDL；旧数据拆分和密文转换不写入应用兼容分支。
- [x] HTTP、MCP、页面和日志不暴露环境私钥；仓库凭据页面不展示部署密钥。
- [x] 测试覆盖新表读写、保存前公钥、Probe/TargetResolver 读取新表和仓库凭据改名路径。

## Test results

验收确认：`task check` 和单元测试均已通过。本轮按用户要求不重复执行。

- `go test ./cmd/... ./internal/...`：通过。
- `task check`：通过，`0 issues`；包含 Web typecheck/lint/format 及 Go lint/config/fmt/run。
- SQLC v1.30.0 生成：通过。
- `git diff --check`：通过。

`task check` 已执行 Web lint 和 format 检查；未单独重复执行 `yarn --cwd web lint:fix`。

## Missed or expanded scope

本次没有实现环境 JSONL 导出/导入，符合 Requirement 和 Plan 的明确非目标。Windows SSH 实机初始化、三种数据库真实升级迁移和离线旧密文回填依赖外部主机、数据库和 `jwt.secret_key`，不在本次本地自动验收中执行。

缩写命名修复仅覆盖本次变更触及的手写 Go 标识符；全仓既有 `*ID` 及 SQLC 生成 API 的统一重命名属于独立重构，不作为本功能验收阻断项。

## Risks and incomplete items

- 三库升级必须先完成离线数据处理，再切换应用；无法依赖应用兼容分支兜底。
- Windows 初始化命令仍要求目标机具备管理员 PowerShell、OpenSSH、WSL2/Docker Desktop 和相应权限。
- `environment_credential` 的私钥解密依赖正确加载当前实例的 `jwt.secret_key`；跨实例传输仍需后续单独设计。

## Conclusion

实现与已接受 Requirement、Plan 及本次实际 diff 对齐。`task check` 和单元测试均已通过；剩余风险集中在真实三库升级、离线密文回填和 Windows 目标机实机验证。
