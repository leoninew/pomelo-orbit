# 远程 SSH 部署环境计划
最后修改时间: 2026-09-04 08:02:27

Review status: Accepted

Mode: strict

## Basis

- Requirement: 远程 SSH 部署环境需求
- Spec: 远程 SSH 部署环境规格
- Current evidence: Project/member、active-project store 与 Project-scoped SQL/usecase/HTTP/MCP 保持资源隔离；CD、runtime query 和 Traefik REST 已切换为显式 Project Environment SSH target，剩余工作以完整验证和真实目标集成为主。

## Delivery boundary

Project 是轻量租户边界，必须保留。交付目标是 Project 1:1 Environment 1:1 Gateway，并把所有部署和 Gateway/Route 操作迁移到该 Project 的 SSH target。不会发布全局资源、跨 Project Environment selector、local executor、共享 Gateway 或兼容双路径。

## Implementation steps

1. 恢复并冻结 Project scope
   - 恢复 Project model/repository/usecase/HTTP/MCP/Web/Proto/SQLC 的既有 contract，撤回全局化 resource、删除 membership 与移除 active-project 的未提交变更。
   - 保持 Application、Repository、Credential、Pipeline、PipelineRun、Deployment、Route 和 Gateway backing Application 的 project_id、列表过滤、ownership validation 与 membership checks。
   - 使 Project code 创建后不可变；Project deprecate 前要求其 Environment 已 disabled。

2. 以 Project 作为 Environment 唯一归属
   - 新增 environment.project_id 非空唯一逻辑引用；Environment code 从 Project code 派生且不可改。
   - 将 Environment create 收敛到 Project create 事务，创建 Project、deployment SSH Credential 与 Environment；移除独立 Environment create，禁止 Environment delete 和跨 Project rebinding。
   - 在现有 Project handler/route/detail 页中增加 Environment get/update/disable/probe；复用现有 Project membership guard、AppDialog、表单控件与 active-project store。

3. 补齐 Deployment/Gateway/Route identity
   - Deployment 以正式列固化 Project 与解析后的 Environment/credential revision、Gateway Application identity snapshot；Service/Route/Gateway 一律由 Project 推导 Environment，不接受可漂移的 Environment selector。
   - Gateway provision 使用请求级事务创建 Project-scoped backing Application/Service 并唯一绑定 Environment；Route managed target、enabled list、sync preview/confirm 和 certificate snapshot 均校验同 Project。
   - 将 global active Gateway 和 global traefik network 替换为以 Project/Environment code 派生的 remote namespace。

4. 实现 SSH executor
   - 以 deployment port 隔离 Probe、remote file set、Compose/runtime operations、Traefik REST publish 与 log stream。
   - 使用 x/crypto/ssh + SFTP、严格 host key、Linux POSIX adapter 与 Windows PowerShell adapter；删除对控制面 Docker workspace、Docker socket 和 remote localhost 访问的业务依赖。
   - 实现 staging、lock、atomic commit、relative mount materialization、cancel close-session + bounded Compose ps reconciliation。

5. 接入 Pipeline image contract 和 Web/MCP
   - 强制 linux/amd64 digest Component image，保持 registry 为宿主机责任。
   - MCP 继续接受 project_id 并在服务端解析唯一 Environment；按 HTTP 相同 membership 权限保护。
   - Project detail 和顶栏显示当前 Environment 状态；Service/Deployment/Route/Gateway 页面展示派生的 Environment，不新建跨 Project selector。

6. 迁移、测试与文档
   - 新增 SQLite/MySQL/PostgreSQL migration，保持 Project schema，增加 Environment/Gateway/Deployment snapshot 字段；旧库无完整 Environment 配置时 fail closed。
   - 覆盖 Project create rollback、Environment uniqueness/disable/probe、Gateway uniqueness、跨 Project route target rejection、credential redaction、Linux/Windows executor contracts。
   - 更新活 SoT 和运维指南后，运行 task check、go test ./cmd/... ./internal/...、yarn --cwd web lint:fix、yarn --cwd web typecheck。

## Verification plan

| Scope | Evidence |
| --- | --- |
| Tenant boundary | Project membership 继续控制每个资源；任何跨 Project resource/route/gateway 访问均被拒绝。 |
| Project/Environment | Project 恰有一个 Environment；Project create 事务失败不留下 Project/Credential/Environment 部分数据；Environment 不可删除。 |
| Gateway | 一个 Environment 只能 provision 一个同 Project Gateway；Gateway/Route snapshot 不跨 Project。 |
| Snapshot | Deployment 固化 Project、Environment target revision、deploy-key revision 与 Gateway revision。 |
| SSH | Linux 与 Windows target 覆盖 Probe、stage、deploy/restart/stop、logs、inspect、network、HTTP probe、cancel 对账和 SSH 内 Traefik REST。 |
| Quality gates | task check、go test ./cmd/... ./internal/...、yarn --cwd web lint:fix、yarn --cwd web typecheck。 |

## Risks

- 先前的全局化未提交改动必须完整回退，避免遗留无 Project scope 的路径。
- 需要真实 Linux OpenSSH+Docker 与 Windows native OpenSSH+WSL2 Docker Desktop target，缺失时不能宣称平台支持完成。
- Project code 不能再修改，现有 UI/API 需要显式反映此限制。

## User review notes

- 用户最新决定：Project 具有类似租户的意义，不能移除。
- 用户确认：一 Project 一 Environment，一 Environment 一 Gateway，Project 切换即 Environment 切换。
