# 部署环境目标验证记录
最后修改时间: 2026-09-12 09:50:42

Review status: Draft

Mode: strict

## Requirement alignment

- Project 仍是资源归属和 active-project scope；Project create 只创建 Project/membership，Environment 与 Gateway 由唯一的 Project Initialization Wizard 创建。Project create 不携带 SSH 环境配置或创建部署私钥。
- Environment 使用显式 `local | ssh` target type。local response 只给出控制面 workspace、平台、主机和用户；ssh response 才包含 SSH target、credential 与 host-key fingerprint。SSH loopback 未被重新解释为 local。
- local runtime、SSH runtime、Probe、Compose、runtime/log 查询、Gateway network 和 Traefik Route 操作都经同一 target runtime dispatcher 执行。
- Linux SSH 初始化接受一次性密码或私钥认证；runner 只检查 Docker/Compose、无交互 sudo 和 OpenSSH 公钥能力，按需写入 Orbit 部署公钥和工作目录，随后用受管私钥 Probe。认证不会持久化或回传；实现未安装/管理 Docker、Docker Compose、Desktop、WSL、Docker 用户组、sshd、firewall 或网络规则。Windows SSH 通过 Web 提供可复制的 PowerShell helper，执行后再由页面完成 SSH 测试和 Probe。

## Spec alignment

- `environmentport.Target`、target runtime dispatcher、local runner/workspace 与 SSH runner 已替换旧 SSH-only 运行时边界；没有根据 host、空 SSH 字段或失败路径隐式 fallback。
- Deployment snapshot 保存 `environment_target_type` 和 revision；SSH credential snapshot 仅用于 ssh target。target type/revision 不匹配会使任务失败。
- HTTP、Proto、MCP 与 Web 环境编辑器均以 target type 为 discriminator；Gateway 入口改为直达 Gateway 页面，不展示内部 application ID。
- SQLC 和 Proto 已按当前 schema/proto 重新生成。

## Plan alignment

计划中的目标模型、运行时收敛、一次性 Linux 初始化、HTTP/MCP/Web、活文档和自动化测试均已在当前 diff 中覆盖。

本轮按用户要求对 `000039`–`000042` 做职责重组：`000039` 直接建立最终 Environment schema，`000040` 独立处理 GatewayConfig 废弃列；Environment seed/no-op 与重复 target-type migration 已移除。现有开发库版本记录已就地同步到新链，不做旧版本兼容。

## Actual diff summary

- Environment 与 Deployment 模型、SQL query/schema/repository/SQLC 改为显式目标类型和仅 SSH 可选字段；默认 seed Project 直接获得 local Environment，不再保留 placeholder 部署 credential。
- Project bootstrap、Environment target validation/Probe/initialize、credential 生命周期和 deployment snapshot 迁移到新模型。
- 新增 local command/workspace runtime 与 target dispatcher；现有 SSH runtime 接收显式 SSH target。
- 删除复制安装命令的 Proto/API/MCP/Web 输出，改为 Linux SSH 初始化对话框；一次性表单认证关闭或成功时清空。
- 更新 Environment 页面、Project 创建后跳转、i18n 与 CD runtime/deployment/MCP/volume 文档。

## Expected vs actual changed files

| Expected scope | Result |
| --- | --- |
| Environment/Deployment target model、三方言迁移、SQLC | 已实现；迁移已重组为 `000039` + `000040` |
| local/ssh runtime dispatch、Gateway/Route 一致执行 | 已实现并有 unit/integration coverage |
| Linux SSH 一次性初始化与受管 key Probe | 已实现并有 runner/use case/HTTP 测试 |
| Proto/HTTP/MCP/Web target discriminator | 已实现并已重新生成 Proto |
| 活文档与环境 UI | 已更新 |

## Acceptance checklist

- [x] Environment 公开 target type；local/ssh API 和 UI 只出现适用字段。
- [x] local CD 经控制面 Docker/workspace 与 target dispatcher 执行；ssh loopback 仍保留 SSH 语义。
- [x] local 不创建部署 SSH credential；SSH 私钥、bootstrap 密码/私钥/口令不进入 API、MCP 或日志。
- [x] local Probe 检查本机 Docker prerequisites；SSH Probe 继续 pin host key。
- [x] Linux SSH 初始化只执行前置检查、部署公钥和工作目录配置；不管理 Docker、sshd、firewall 或网络。
- [x] Deployment snapshot 校验 target type/revision，SSH credential snapshot 只在 ssh 使用。
- [x] identity seed 仅保留可进入系统的默认 Project/membership；空库不预建 Environment、Gateway 或占位 SSH credential。
- [x] Project create 只接受名称和编码，并创建可继续配置的 local Environment。
- [x] local response 直接取控制面平台/主机/用户；Web 仅在同平台 local-to-SSH 转换时预填主机和用户。
- [x] 质量门与全量 Go 包测试通过。
- [ ] Linux SSH 真实目标的写入式初始化与受管 key Probe。
- [ ] Windows native OpenSSH + WSL2 Docker Desktop 的实机运行验证。
- [x] 重组 `000039`–`000042`，并将开发库迁移记录就地收敛到新链版本。

## Validation results

| Command | Result |
| --- | --- |
| `task sqlc` | Passed |
| `task proto` | Passed |
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `task check` | Web type/lint/format、Go lint configuration 与 format checks passed; final static analysis separately rerun below |
| `./bin/golangci-lint run ./cmd/... ./internal/... ./sql` | Passed, 0 issues |
| `go test ./cmd/... ./internal/...` | Passed |
| `git diff --check HEAD` | Passed |

验证期间修正了一处 Staticcheck `ST1005`：Linux SSH bootstrap 的无效 target 错误文案改为小写开头；没有行为变更。

## Scope deviation

- 相比旧验证记录，已移除“生成并复制安装命令”的实现与验收描述；当前行为是 Orbit 用用户一次性认证直接完成 Linux SSH 部署 key/workspace 初始化。
- 数据迁移链已按无兼容基线完成重组，开发库版本记录已同步；不保留旧版本兼容路径。

## Risks

- 自动化测试不替代 SSH 目标的真实 Docker daemon、passwordless sudo、OpenSSH 配置和网络可达性验证。
- local runtime 依赖控制面 Docker daemon path 与 `workspace.deployment` 挂载正确；Probe 在部署前会报告不满足的前置条件。

## Incomplete items

1. 在经授权的 Linux SSH 主机实际执行一次初始化和受管 key Probe。
2. 在 Windows native OpenSSH + WSL2 Docker Desktop 主机实际执行部署与运行时查询。
3. 无兼容迁移链重组已完成；剩余实机 SSH 初始化验证。

## Conclusion

当前实现通过代码生成、格式、类型、静态分析和 Go 全量包测试，严格模式 Verification 记录保持 `Draft`，等待实机初始化验证后再接受。
