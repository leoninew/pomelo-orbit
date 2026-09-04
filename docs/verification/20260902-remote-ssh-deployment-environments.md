# 远程 SSH 部署环境验证记录
最后修改时间: 2026-09-04 21:57:05

Review status: Draft

Mode: strict

## Requirement alignment

- Project 继续作为 membership、资源隔离和部署目标切换边界；Project 1:1 Environment 1:1 Gateway 已由领域模型、应用服务、HTTP、MCP 和 Web active-project 流程共同约束。
- Environment 在 Project 创建事务中与专用 deployment SSH Credential 一起创建；Environment 只有 active/disabled 状态，没有独立 create/delete/rebind surface。
- Service、Gateway、Route、Deployment 和 runtime operations 从 Project 解析唯一 Environment，外部 API/MCP 不接受可漂移的 `environment_id`。
- 支持边界收敛为 Linux OpenSSH + Linux containers，以及 Windows native OpenSSH + WSL2 Docker Desktop Linux containers；部署、运行时查询和 Traefik REST 均使用 SSH/SFTP。
- registry 登录、CA、DNS、网络和多 registry 配置仍由目标宿主机负责；不引入控制面本机 Docker fallback 或兼容双路径。

## Spec alignment

- Environment 使用 `project_id` 唯一逻辑引用；Gateway 使用 Environment 的唯一 logical binding，跨聚合关系由应用层校验同一 Project 归属。
- Deployment 以正式字段保存 Project、Environment、target revision、deploy credential revision 与 Gateway Application snapshot；执行前校验目标是否仍可用，发生漂移时 fail closed。
- SSH runtime 使用 `golang.org/x/crypto/ssh`、私钥认证、host-key fingerprint 校验与 SFTP。Windows adapter 通过 noninteractive PowerShell 调用宿主机 `docker.exe` 和 `curl.exe`，并验证 Docker Server OS 为 Linux。
- Project detail 页面复用既有 Project scope、共享表单组件和 active-project store，在 Project 内编辑、禁用和 Probe Environment；没有新增跨 Project Environment 导航。
- MCP 增加按 Project scope 的 Environment get/update/probe 工具；private key 与 passphrase 只可写入，响应和日志均不回显。

## Plan alignment

- 已恢复和保持 Project resource/membership scope，未将资源全局化。
- 已完成 Environment、Gateway、Route、Deployment snapshot 与远程 SSH runtime 的主线实现，并移除控制面本机 deployment runner 和 deployment workspace adapter。
- 已新增 MySQL/PostgreSQL/SQLite `000039_environment_foundation` migration、SQLC 查询与 v37 到 v38 的 MySQL/PostgreSQL 演练脚本。
- 已更新 CD runtime、部署、MCP 直接操作和卷挂载活文档，并新增 Windows OpenSSH 本地集成环境配置脚本。

## Actual diff summary

- 后端：新增 Environment model/repository/usecase/HTTP route/handler/proto/SQLC，实现 Project create 的 Project + Credential + Environment 请求事务、Probe 以及 Project-derived target resolution。
- 远端运行时：新增 SSH environment probe 和 runtime adapter；部署、Compose stage、runtime query、Gateway/Route snapshot 与 Traefik REST 从控制面本机执行切换为目标宿主机 SSH/SFTP 执行。
- 隔离和快照：Gateway、Route、Service 与 Deployment 均按 Project 校验；Deployment 固化 remote target identity/revision，target 变更或 Probe 不新鲜时拒绝执行。
- 前端和 MCP：Project 页面增加 Environment 配置/Probe 卡片与 API；MCP 增加 Project-scoped Environment tools，runtime 输出携带派生 Project identity。
- 平台工具和测试：新增 Windows OpenSSH 配置脚本、MySQL/PostgreSQL v37 到 v38 迁移脚本、SSH unit/integration tests 及 Windows native OpenSSH + WSL2 Docker Desktop live-host E2E。

## Expected vs actual changed files

| Expected scope | Actual files and result |
| --- | --- |
| Project/Environment domain and transaction | `internal/model/environment.go`、`internal/application/environment/**`、`internal/repository/environment.go`、`internal/infrastructure/database/tx/request.go`、Project/Credential usecases and tests; completed. |
| HTTP/Proto/SQL/Schema migration | Environment HTTP handler/routes, `proto/orbit/v1/environment/**`, `sql/migration/*/000039_environment_foundation.*`, `sql/query/environment/environment.sql`, SQLC generated outputs; completed. |
| SSH-only deployment/runtime and Traefik | `internal/infrastructure/runner/ssh/**`, deployment/gateway/route usecases, Traefik adapter; completed. Deleted control-plane-local runner/workspace implementation is intentional. |
| Web and MCP | `web/src/views/project/**`, Project environment API/store/i18n, MCP delivery tools/server/types; completed. |
| Operations and migration drill assets | `scripts/setup-windows-openssh.ps1`, `scripts/migrate_v37_to_v38.mysql.sql`, `scripts/migrate_v37_to_v38.postgres.sql`, relevant active documentation; completed. |
| Unrelated workspace configuration | `.codex/config.toml` is modified but excluded from this feature verification and any later feature commit. |

## Acceptance checklist

- [x] Project remains the resource-isolation and membership boundary; active-project scope remains in HTTP, MCP and Web paths.
- [x] Each Project has one non-deletable Environment using a unique logical `project_id` relation; Project code is immutable and provides the stable derived infrastructure identity.
- [x] Project creation writes Project, encrypted deployment SSH Credential and Environment atomically; rollback/redaction coverage is present.
- [x] Gateway is uniquely bound to its Project Environment; Gateway backing application/service and Route managed targets are checked for the same Project.
- [x] Deployment snapshots store Environment/credential/gateway identity and revisions; runtime operation targets are derived rather than supplied by callers.
- [x] CD, runtime and Traefik REST implementation uses SSH/SFTP only, with no local executor fallback; Windows real-host Probe, Compose execution and Traefik router query passed.
- [x] Required automated quality gates passed.
- [ ] Linux OpenSSH + Linux containers has not yet been exercised against a real target.
- [ ] Traefik REST readiness and full recoverable `providers.rest` snapshot PUT have not yet been exercised on a provisioned Gateway.

## Validation results

| Command | Result |
| --- | --- |
| `yarn --cwd web lint:fix` | passed |
| `yarn --cwd web typecheck` | passed |
| `yarn --cwd web test` | passed: 18 files, 91 tests |
| `task check` | passed: 0 issues |
| `go test ./cmd/... ./internal/...` | passed |
| `$env:POMELO_ORBIT_WINDOWS_SSH_E2E = '1'; go test -count=1 -run '^TestWindowsSSH' -v ./internal/test/e2e` | passed: Project/Environment HTTP transaction + Probe, Docker JSON query, Traefik router query, Compose stage/run/cleanup |
| `git diff HEAD --check` | passed |

Windows live-host E2E covered the real path `TargetResolver -> RouteManager -> SSH Runtime -> PowerShell curl -> JSON decode`. The PowerShell adapter suppresses progress/CLIXML output, and the SSH runtime separates stdout from stderr so Traefik JSON is not corrupted or truncated.

## Scope deviation

- No requirement or spec scope was removed.
- MCP Environment tools and the Windows setup script expand the planned operational surface only to make the Project-scoped Environment configurable and testable; they preserve the specified authorization and platform boundaries.
- Full Traefik snapshot publication is intentionally not executed against the current target because it replaces `providers.rest`; it requires a recoverable test window and snapshot restore plan.

## Risks

- Linux support is implemented and unit-tested but lacks live-host evidence; a Linux-specific shell/SSH/Docker variance may still exist.
- Full Traefik REST PUT validation can alter active routing. It must be exercised only with a known snapshot backup and a prepared restore path.
- Existing deployments created before `000039` need the supplied migration/drill flow and a subsequently configured Environment; incomplete target configuration remains fail closed by design.

## Incomplete items

1. Provision or select a Linux OpenSSH + Docker host, configure one Project Environment, and run the same Probe/Compose/runtime/Traefik E2E coverage.
2. In a recoverable Gateway test window, record the current `providers.rest` snapshot, verify Traefik readiness, PUT a controlled full snapshot, verify the rendered routers, then restore the recorded snapshot.

## Conclusion

Implementation and automated verification are complete for the verified Windows target and all repository quality gates. This record remains `Draft` because the two real-environment validation items above are intentionally outstanding. `.codex/config.toml` remains outside this feature's verified change set.