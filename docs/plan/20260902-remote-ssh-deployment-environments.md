# 部署环境目标计划
最后修改时间: 2026-09-09 17:04:54

Review status: Accepted

Mode: strict

## Basis

- Requirement: 部署环境目标需求
- Spec: 部署环境目标规格
- 保留 Project 1:1 Environment 1:1 Gateway、Project scope 与 Gateway shared `traefik` network。

## Delivery boundary

实现 explicit `local | ssh` target，恢复历史本机 Docker/workspace execution。删除 SSH-only 端口、模型和 UI 假设；不以 loopback、空 SSH 字段或失败重试做两类目标之间的 fallback。现有 SSH 功能继续存在，但仅属于 `ssh`。

## Implementation steps

1. 扩展 Environment/Deployment schema、SQL query、model、repository 与 generated SQLC：Environment target type 与 optional SSH fields；Deployment target type 与 optional SSH snapshot。新增三方言 migration，将既有 Environment 标记 ssh；default seed 收敛为 local，无 credential placeholder。
2. 以 `environmentport.Target` 替换 `SSHTarget`，按 target type 生成/校验 SSH credential、target revision 和 Probe；Project bootstrap 固定创建 active local Environment，SSH key 仅在 Environment 页面切为 ssh 后创建。
3. 将 `RemoteRuntime` 改为 target-neutral Runtime；恢复 local workspace/shell runner，新增 explicit runtime dispatcher，同时让 deployment、runtime query、container logs、Gateway/Route Traefik calls 共用。
4. 将 proto/HTTP/MCP DTO 变为 target discriminator 与 SSH nested target；Project create DTO 只保留名称和编码，Web 创建完成后切换 active Project 并进入 Environment 页面。local 不显示 SSH 表单/初始化入口；Linux SSH 以一次性密码或私钥直接初始化，先检查现成 Docker/Compose、`sudo -n` 与 OpenSSH 公钥能力，再按需写入部署 key 和工作目录，不管理 Docker/Compose/Desktop/WSL、Docker 用户组、sshd、firewall 或网络。
5. 更新产品模型、CD runtime、部署与 MCP guide，移除 SSH-only/local-fallback 描述。
6. 补充 local/ssh target resolution、snapshot validation、runtime dispatch、Probe、project bootstrap、seed 与 UI form tests；重新生成 SQLC/proto 并运行质量门。

## Files to change

- `sql/migration/*/000041_*`、`sql/migration/*/000040_seed_environment.up.sql`、`sql/query/environment/*`、`sql/query/deployment/*`、`sql/schema/*`。
- `internal/model/environment.go`、`internal/model/deployment.go`、Environment/Project/Deployment/Route ports and use cases、SQLC repositories。
- `internal/infrastructure/runner/local/*`、`internal/infrastructure/runner/ssh/*`、target runtime dispatcher、`internal/bootstrap/*`。
- `proto/orbit/v1/environment/environment.proto`、generated Go/TypeScript、HTTP/MCP mapper 和 `web/src/views/environment/EnvironmentPage.vue`、Project form/i18n.
- `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、relevant deployment/MCP guides。

## Verification plan

- Project creation transaction writes an active local Environment with no deployment SSH credential; Environment target validation rejects invalid/mixed type payloads and never reinterprets SSH loopback as local.
- Linux 自动初始化先检查 Docker/Compose、无交互 sudo 与 OpenSSH 公钥能力，再按需写入部署公钥和工作目录；一次性认证不进入持久化、响应、MCP 或日志，且操作不包含 Docker 安装/服务管理、Docker group、WSL、Docker Desktop、sshd、firewall 或网络管理。
- Local runtime writes/uses configured control-plane workspace and invokes Docker commands; SSH runtime receives only SSH targets.
- Local/SSH Probe and deployment snapshot checks cover type and revision behavior.
- Environment HTTP response exposes direct control-plane local platform/host/user values; the Web editor uses them only for a matching local-to-SSH platform transition.
- Gateway network and Route publish are exercised through common runtime selection.
- `task sqlc`、`task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`。

## Blockers

无。

## Assumptions

- 控制面部署环境的 `workspace.deployment` 与 Docker daemon path mount 配置正确；本机 Probe 会验证 Docker prerequisites。
- 用户将按说明重新部署受影响环境；不会添加业务逻辑兼容分支处理旧 SSH-only payload。

## Risks

- 三方言 migration 的 nullable/seed 更新与 SQLC type 映射需要同时验证。
- 本机 runner 运行在控制面容器时，物理 Docker daemon path resolver 不能退化为容器内路径。

## Rollback

回退本次提交和新的 migration；不通过重新引入 hostname heuristic 或 SSH fallback 回滚。

## User review notes

- 本机模式现在恢复，以便和历史本机实现直接比对。
- 用户明确要求拒绝兼容层、reserved proto 字段与业务侧隐式适配。
