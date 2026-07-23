# CD 版本化主线收口：E5/E6 清理与文档对齐
最后修改时间: 2026-07-23 10:45:00

Review status: Accepted

Flow mode: light

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`  
前序：E5 / E6 主能力已落地；用户实机已能启动 managed Traefik 并部署其他应用。  
本任务：**工程清理 + 过程文档对齐**，不开新业务能力。

## Background

E5/E6 验证记录（`docs/verification/20260722-cd-gateway-component-mount-e5.md`、`docs/verification/20260722-cd-gateway-domain-config-e6.md`）将下列项记为 **Incomplete / 残留**，不构成主路径未交付，但会误导运维与回归：

1. 全局 `traefik.api_url` / `traefik.domain_suffix` 产品路径已删，**BindEnv 死键**与部分测试样例 yaml 仍写旧键。  
2. **单 active gateway** 部署拦截有实现，**专用单测偏弱**（E5 V9 / E6 V9）。  
3. cadence 与 Verification 状态文案滞后（E6-C 实际已做；实机已验收；Verification 仍 Draft）。  

用户决定：**先记录并执行清理**；自签/自定义 PEM（F1）、用户自管 Traefik（external）**不算**本任务及本主线必做后续。

## Goal

1. 去掉无效全局 Traefik 配置入口残留（BindEnv 死键 + 测试样例对齐），避免运维仍设置无效环境变量。  
2. 补强 **单 active gateway** 二次部署拦截的专用自动化测试。  
3. 将 E5/E6 Verification 与 cadence 索引更新为与现状一致（含：主路径可验收/已实机验收、E6 compile 已交付、清理项关闭）。  
4. 明确：**应用版本化主兑现线（P0→E6）在清理完成后视为闭环**；F2–F6 / E2–E4 / K8s 等仅作可选 backlog，不在本任务实现。

## Non-goal

1. 不实现 **F1** 自定义/自签 PEM 证书产品化。  
2. 不实现 **external** / 用户自管 Traefik。  
3. 不实现 F2 rest 鉴权、F3 多 gateway、F4 provider 演进、F5 dashboard、F6 healthcheck 表单。  
4. 不实现 E2 路径布局、E3 host 网络网关、E4 K8s。  
5. 不扩大 Gateway config 字段（ports/log_level/acme 等表结构增强）。  
6. 不强制补浏览器 E2E 或再跑一轮实机 Traefik（用户已验收启动与部署）。  
7. 不做旧数据迁移 / 兼容层。  
8. 本 light 任务**不**顺带重构无关模块。

## User scenarios

1. 运维查看配置绑定 / 环境变量文档时，**不再**看到有效的 `POMELO_ORBIT_TRAEFIK__API_URL` / `DOMAIN_SUFFIX` 类死入口；rest URL 与域名仅来自 Gateway。  
2. CI / 本地 `go test` 对「已有 active gateway 时再部署第二个 gateway」有明确失败断言。  
3. 读者打开 cadence / E5·E6 Verification，能读到：主能力已交付、实机可用、清理项状态清晰。

## Acceptance

1. `internal/config`（或等价）**不再** `BindEnv` 已删除的 `traefik.api_url` / `traefik.domain_suffix`（及同义死键）；无 mapstructure 字段接收的无效绑定一并删。  
2. 依赖「旧键仍出现在样例 yaml」的测试改为断言：**未知/历史键被忽略**或样例改为仅 `cert_dir`（与现行 `TraefikConfig` 一致）；不得重新引入产品 SoT。  
3. 存在**专用**单测覆盖：已有 active gateway Service 时，再次部署 gateway 被拒绝（或与现有 `HasActiveGatewayService` 语义一致的断言）；不只依赖代码审查。  
4. 更新：  
   - `docs/verification/20260722-cd-gateway-component-mount-e5.md`  
   - `docs/verification/20260722-cd-gateway-domain-config-e6.md`  
   - `docs/requirement/20260721-cd-application-version-cadence.md`  
   使状态与索引反映：实现完成、用户实机验收、本清理任务范围；Verification 在用户确认后可标 **Accepted**（实现阶段可先改 Incomplete/正文，Accept 以用户明示为准）。  
5. 项目检查通过：`task check`；测试：`task test`（或等价 `go test ./cmd/... ./internal/... ./sql`）。  
6. diff **不包含** F1/external/F2–F6/E2–E4 业务实现。  
7. **不做历史兼容**：活跃产品路径/示例/测试中的旧全局 Traefik env 与 `domain_suffix` 配置语义直接删除或改写；不保留双轨兼容说明。

## Open questions

**暂无需要用户确认的未决事项。**

已按用户口径钉死：

- 清理要做。  
- 自签证书、用户管理的 Traefik **不算**后续必做。  
- **不做历史兼容**；活跃测试/示例中的旧 `domain_suffix` 配置语义一并改掉。  
- 验证命令用 **`task check`**。

## Decisions

1. 任务号建议文件名：`20260723-cd-post-e6-cleanup`；**light** 流程：Requirement → Implementation → Verification。  
2. 范围 = E5/E6 Incomplete 中的**工程清理 + 文档对齐**，不是新扩展号（非 E7）。  
3. F1 / external **排除**；F2–F6 / E2–E4 / K8s **不**纳入本任务，仅 backlog 备注。  
4. 用户实机已验证「能启动 Traefik + 部署其他应用」→ 本任务**不**再以实机 E2E 为验收门槛。  
5. 与 cadence 关系：清理完成后，**应用版本化主兑现线视为闭环**；下一大任务需另开产品/需求，而非本 cadence 内隐式续项。  
6. **不做历史兼容**：产品路径、示例 env、运维指南、活跃测试中的旧全局 `api_url`/`domain_suffix` **直接删除或改写为 Gateway 语义**；不保留双轨说明作兼容。  
7. **已执行 migration 文件不改**（项目硬约束）；`000005` 种子中历史 liquid 文本可残留于库文件，运行时不再作为产品 SoT。  
8. 检查命令以项目 **`task check`**（及 `task test`）为准，不另造检查入口。

## Risk

1. 删除 BindEnv 后，若外部脚本仍导出旧环境变量，行为不变（本就无效），但运维可能困惑——应用文档/guides 已指向 Gateway，本任务以删死键为主。  
2. 单测若依赖完整 DB fixture，需与现有 usecase/repository 测试风格对齐，避免脆弱集成。  
3. 同步改 cadence / 两份 Verification 时勿改写已 Accepted 的历史决策正文语义，只更新状态、索引与 Incomplete。  
4. 误把 F2–F6 塞进本 PR 会扩大范围。

## Risk / 假设（实现向）

| 假设 | 说明 |
|------|------|
| A1 | 死键仅存在于 config BindEnv 与少量测试 yaml（以仓库检索为准） |
| A2 | `HasActiveGatewayService` 已可测，无需改产品拦截语义 |
| A3 | 前端无需改动 |

## User review notes

- 2026-07-23：用户确认 E6 主路径实机可启 Traefik 并部署其他应用；要求 **清理**。  
- 2026-07-23：用户确认排除自签证书、用户自管 Traefik 后讨论「是否还有后续」→ 主 cadence 无强制大任务。  
- 2026-07-23：`/specflow light` **记录清理任务**；本文件 Draft，待用户 Accept 后再实现。  
- 2026-07-23：用户「开始实现」→ Requirement **Accepted**；进入 Implementation。  
- 2026-07-23：用户要求用 **`task check`**；清理与更新 **不做历史兼容**（活跃路径直接改，不双轨）。
