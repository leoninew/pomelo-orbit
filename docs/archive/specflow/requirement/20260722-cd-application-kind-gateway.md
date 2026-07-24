# CD Application kind：gateway vs standard
最后修改时间: 2026-07-22 13:06:52

Review status: Accepted

Flow mode: strict

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`（**修订 R3**）

前置：**R2 Verification Accepted** — `docs/verification/20260722-cd-environment-ingress-policy.md`。

Spec: `docs/spec/20260722-cd-application-kind-gateway.md`（**Accepted**）  
Plan: `docs/plan/20260722-cd-application-kind-gateway.md`（**Accepted**）  
Verification: `docs/verification/20260722-cd-application-kind-gateway.md`（**Accepted**）

**路线位置**：

```text
P0–P3 Version
  -> R1 Environment/Expose/Service（已交付）
  -> R2 Environment IngressPolicy（standard 如何被入口暴露；已交付 + Verification Accepted）
  -> R3 本文件 Application.kind（网关应用 vs 普通应用如何渲染）
  -> 扩展 E1–E5：consumer 网、路径、host/K8s、挂载细节 等
```

Related：

| 文档 | 关系 |
|------|------|
| `docs/requirement/20260721-cd-application-version-cadence.md` | 总索引；本需求 = **修订 R3** |
| `docs/requirement/20260721-cd-application-version-domain.md` | Application 身份；本需求 **扩展** kind 字段 |
| `docs/requirement/20260721-cd-application-version-render.md` | Render 入口；本需求 **分支** 渲染策略 |
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | R2：standard 应用 labels 合路；**不**定义网关本体 |
| `docs/verification/20260722-cd-environment-ingress-policy.md` | R2 已验收；R3 不得回归 standard 路径 |
| `docs/guides/routing-and-certificates.md` | 现网 Traefik 运维参考（非 SoT） |
| 真实参考（非需求正文） | `data/cd/traefik/docker-compose.yml`（gateway 形态示例） |

## Terminology / 规范用词

| 中文 | 英文 | 说明 |
|------|------|------|
| 应用类型 | **Application.kind** | 应用级枚举；决定 **Render 策略** |
| 普通应用 | **standard** | 默认；业务工作负载；可经 IngressPolicy 暴露 |
| 网关应用 | **gateway** | 平台入口身份；渲染为入口基础设施，非普通业务 compose |

## Background / 背景

1. 现网 Traefik **也通过本项目 CD 部署**，但 compose 与普通应用不同：创建共享网络、挂载静态/动态配置与证书、docker.sock、宿主机端口等（见 `data/cd/traefik/docker-compose.yml` 作**参考**，非唯一真理）。
2. R1/R2 解决的是 **standard 应用如何声明 Expose 并被入口发现（labels × IngressPolicy）**，**未**区分「应用本身就是入口」。
3. 大方向共识（讨论 2026-07-22）：
   - K8s 上网关是集群级 Controller/Gateway API，与业务生命周期解耦。
   - Compose 上 **Traefik 不是唯一实现**；可演进为多适配器。
   - Traefik **可以**宿主机进程或 host 网络运行；规则与 bridge+external 不同 → 属扩展。
4. 小方向共识：**在 Application 级**区分 `kind=gateway` / `kind=standard`（默认），**决定渲染方式**；不用 `code==traefik` 字符串魔法。

## Goal / 目标

1. Application 增加 **`kind`**：`standard` \| `gateway`；默认 `standard`。
2. **Render 按 kind 分支**：
   - `standard`：现有（R2）路径 — Component → compose services；有 Expose 时 Expose × Environment.IngressPolicy → labels。
   - `gateway`：网关渲染路径 — 产出满足入口角色的 compose（网络创建、端口、平台数据目录挂载等契约由 Spec 钉死）；**不是**再套一层「当普通业务 + 手填全量 YAML」的默认真相。
3. kind 为 **应用身份**，跨 Version 稳定；换 Version 升级网关实现，**不**改 kind。
4. 文档与 cadence 可追踪；实现前经 Spec + Plan（strict）。

## Non-goal / 非目标

1. **不**在本期实现 K8s Ingress/Gateway API 控制器部署。
2. **不**在本期实现宿主机二进制网关、多厂商适配器切换 UI。
3. **不**把「所有 standard 自动 `networks.traefik.external`」焊死为本需求必交付（列为 **扩展 E1**）。
4. **不**用 Environment 或 Version 替代 Application.kind 表达「是不是网关」。
5. **开发期** schema **就地**改现有 migration 文件；**不**为 R3 单独加增量 migration。
6. **不**恢复 Binding 用户 SoT；与 R2 一致。
7. **不**将 E5（挂载细节 + Component 全规格补齐）作为 R3 必交付；挂在既有 **E5**，不追加新扩展号。

## 与 R2 的边界

| | standard | gateway |
|--|----------|---------|
| IngressPolicy | 消费：推导 Host/entrypoint/TLS **labels** | 可选：dashboard 等次要暴露；**主责不是**「挂别人的入口」 |
| 网络角色（Compose/Traefik 参考） | 将来可 join 平台网（external） | 创建/声明平台网（bridge + name）等 **提供者** 语义 |
| Expose | 决定哪些服务写 labels（非全容器） | 可有（如 dashboard）；契约由 Spec 定 |
| 本期必交付 | R2 合路保留；去掉 is_ingress 门闩 | R3 主干：kind + Render 分支 |

## User scenarios / 用户场景

### 场景 1：创建普通业务应用（默认）

用户创建 Application，不选特殊类型（或显式 `standard`）：

- 行为与 R2 一致：Version / Expose / Deploy / IngressPolicy labels  
- 渲染**不**走网关模板，即使 `code` 恰好叫 `traefik`

### 场景 2：创建并部署网关应用

用户创建 Application 时指定 `kind=gateway`：

1. 维护 Version（**仍用** Version/Component 管线，Q2）  
2. 选择 Environment + Deploy  
3. 系统按 **gateway 渲染策略** 产出入口角色 compose  
4. 用户**不**靠手改产物 yaml；**不**勾选 attach_ingress

### 场景 3：升级网关实现

用户发布新 Version：`kind` 保持 `gateway`，仅 Version 变化驱动渲染差异。

### 场景 4：列表识别

用户在应用列表能区分 standard / gateway（展示；筛选 Plan 定）。

## Functional requirements（方向级，Spec 细化）

### FR1 — kind 字段

1. `application.kind` ∈ {`standard`, `gateway`}；缺省 / 迁移回填 = `standard`。  
2. 创建时可指定；**创建后不可修改**（Q1）。  
3. API / UI 创建与展示可见 kind。

### FR2 — Render 分支

1. `Render` **必须**读 `app.kind` 选择策略。  
2. `standard`：不得因 code 名特殊走网关模板；labels 仅针对 **Expose**。  
3. `gateway`：走网关策略；参考现网 traefik **结构意图**；Component 全规格补齐与挂载细节见 **E5**。  
4. Preview 与 Deploy **同一**分支函数。

### FR3 — 平台抽象

1. 产品概念是 **Gateway Application**，实现适配器本期默认 **Traefik-on-Compose**。  
2. 禁止将「唯一网关实现 = Traefik」写进领域不可替换硬约束；本期可不建 provider 字段。

### FR4 — 废止 Service 入口开关

1. 不使用 `attach_ingress` / `Service.is_ingress` 作为产品 SoT。  
2. Service 身份来自 `Application.kind`。  
3. standard 是否写 labels 由 **Version.Expose** 决定。

## 扩展（本需求不交付，路线预留）

| 扩展 | 说明 |
|------|------|
| E1 consumer 网络 | standard 有 Expose 时自动注入平台网 `external: true` 并 join。**已实现**（exposed 组件 join `traefik` external；无 Expose 不注入） |
| E2 路径布局 | 网关工作区扁平 vs `{env}/{instance}` 统一 |
| E3 宿主机/host 网络网关 | 另一部署模式 |
| E4 K8s 网关 | 另开 |
| **E5 gateway 挂载与 Component 规格** | **不另开任务**。在 E5 内交付：① 以现网 `data/cd.bak/traefik/docker-compose.yml`（或等价参考）为目标，**补齐** gateway Version 的 **Component 全规格**（已有字段：`ports` / `command` / `mounts` / `networks` / env 等），使 Preview/Deploy 产物在规格层可趋近参考 compose；② 平台侧挂载解析与契约（逻辑路径→宿主机路径、docker.sock、acme、配置目录等）。R3 **不**交付 E5。 |
| **E6 网关领域** | **另开扩展**。**Gateway domain** 双形态：`managed`（容器 — 配置面向用户、部署借 Version）\| `external`（宿主机自部署 — 只登记 rest）。Requirement Draft：`docs/requirement/20260722-cd-gateway-domain-config-e6.md`。R3/E5 **不**交付 E6。 |

## Acceptance / 验收（需求级）

1. cadence 索引含 **R3**；domain/render 有指针。  
2. Spec Accepted 后：`kind` 有单一 SoT；Render 分支可测；无 `code==traefik` 魔法主路径。  
3. standard 行为相对 R2 **不回归**（除 is_ingress 门闩改为 Expose 驱动 + 默认 kind）。  
4. gateway Preview/Deploy 与 standard 可区分。  
5. `go test` + 若改前端则 lint/typecheck。

## Open questions

**全部闭合**（见 Decisions）。

| # | 结论 |
|---|------|
| Q1 | **kind 创建后不可修改** |
| Q2 | **gateway 仍用 Version/Component 存规格**；Render 解释不同 |
| Q3 | 每 Project **不限制** gateway 数量；可建可不部署 |
| Q4 | **废止** attach_ingress / is_ingress 产品语义；Service 取 Application.kind；standard 暴露看 Expose |
| Q5 | 挂载细节 + **补 Component 全规格** = **E5**（同一后续任务，不另开）；非 R3 必交付 |

## Decisions（已钉）

1. **层级 = Application**。  
2. 枚举 **`standard`（默认）\| `gateway`**。  
3. **决定渲染方式**；现网 yaml 为参考。  
4. E1–E5 不阻塞 R3 主干。  
5. R2 IngressPolicy 合路对 standard **保留**。  
6. **Q3** 不限制数量。  
7. 只支持 Traefik + standard 按 label 渲染（有 Expose 时）。  
8. **Q4** 废止 attach_ingress / is_ingress 产品 SoT。  
9. **Q5** → E5。  
10. **Q1** kind **不可修改**。  
11. **Q2** gateway **使用** Version/Component 统一管线。

## Risk / 风险

1. 删除 is_ingress/attach 与已交付代码冲突 → Spec/Plan 迁移与测试改写，无双轨。  
2. E5 遗忘 → cadence 已挂。  
3. 误标 gateway 后无法改 kind → 创建 UI 必须清晰（Q1）。  
4. 禁止 `code==traefik` 魔法主路径。

## User review notes

- 2026-07-22：开 R3 Draft；R2 闭合后进入评审。  
- 2026-07-22：钉 Q3；解释 Q4/Q5。  
- 2026-07-22：闭合 Q4（kind + Expose，废止 is_ingress/attach）；闭合 Q5→E5。  
- 2026-07-22：对照 bak traefik compose 与 Preview 差距——**补 Component 全规格** 并入既有 **E5**，不追加 E6/新任务。  
- 2026-07-22：用户验收 → **Verification Accepted**（`docs/verification/20260722-cd-application-kind-gateway.md`）。  
- 2026-07-22：用户钉 **Q1 不可修改**、**Q2 是**；**Accept Requirement** → 进入 Spec。
