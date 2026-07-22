# CD 应用版本化领域模型兑现节奏
最后修改时间: 2026-07-22 10:42:53

Review status: Accepted

Flow mode: standard

## Terminology / 规范用词（Accepted）

| 中文 | 英文 | 说明 |
|------|------|------|
| 应用 | **Application** | 长期身份 |
| 应用类型 | **Application.kind** | `standard`（默认）\| `gateway`；决定 Render 策略（见 **R3**） |
| 版本 | **Version** | 业务数据：结构化静态规格；**不是** compose |
| 环境 | **Environment** | **Project 下业务数据**；有 `project_id`（见修订文档） |
| 部署 | **Deployment** | 唯一操作记录（deploy/stop/restart） |
| 服务 | **Service** | 业务数据：运行绑定（app+env+instance）；stop 不删 |
| 容器 | **Container** | Component 物化实例 |
| 组件 | **Component** | Version 内规格单元 |
| 暴露规格 | **ExposeSpec** | Version 内 http\|tcp + 端口；无域名 |

## Background / 背景

CD 应用管理职责过载。目标：Application 身份 + Version 规格 + Service 运行绑定 + Deployment 操作流水 + Container 观测。

部署顺序：① Service → ② Render(Version) → ③ apply → ④ Container。

## Goal / 目标

分批把「可版本化应用」落到可验收边界；P0/P1 先闭合再谈 P2 实现闭环。

## Non-goal / 非目标

1. 本文件不定 DDL/API/UI。
2. 不一次做多环境/K8s/TCP Ingress。
3. 本阶段默认不改产品代码，直至需求 Accepted 并进入 Plan/实现。
4. 不修改已执行迁移文件。

## Delivery cadence / 兑现节奏

| 批次 | 文档 | 意图 | 需求闭合 |
|------|------|------|----------|
| **P0** | [domain](./20260721-cd-application-version-domain.md) | 领域对象、ER、生命周期、部署顺序 | **Open questions = 0** |
| **P1** | [spec](./20260721-cd-application-version-spec.md) | Version 最小 schema | **Open questions = 0** |
| **P2** | [deploy](./20260721-cd-application-version-deploy.md) | 本机 deploy 闭环 | **Accepted** |
| **P3** | [render](./20260721-cd-application-version-render.md) | Renderer / raw | **Accepted** |
| **修订 R1** | [environment-expose-service](./20260721-cd-environment-expose-service.md) | Env 实体、Expose、多 Service、Traefik MVP | **Accepted**；已实现 + Verification Accepted |
| **修订 R2** | [environment-ingress-policy](./20260722-cd-environment-ingress-policy.md) | 环境**接入策略**取代 per-component Binding | Requirement/Spec/Plan/Verification **Accepted** |
| **修订 R3** | [application-kind-gateway](./20260722-cd-application-kind-gateway.md) | Application.**kind**=`standard`\|`gateway`，决定 Render 分支 | Requirement **Draft**（strict） |
| 后续 | consumer 自动 join 平台网 / 宿主机网关 / K8s Gateway 等 | 见 R3 扩展 E1–E4；另开或并入 R3 Spec | — |

```text
P0 领域
  -> P1 Spec
  -> P2 本机 deploy
  -> P3 Renderer
  -> R1 Environment / Expose / Service（已交付）
  -> R2 Environment IngressPolicy（standard 如何被入口暴露）
  -> R3 Application.kind gateway|standard（入口应用如何渲染；当前 Draft）
  -> 扩展 E1–E4 / K8s 等
```

**注意**：不再采用「P2 先用 compose 全文当 Version 存储」；与 P0/P1 **Version=业务结构化规格** 冲突，已废止。

## User scenarios / 目标态

1. 维护多 Version；未发布可改，已发布只读。
2. deploy 指定 Version：先 Service，再按 Version 渲染并 apply。
3. 失败在 Service/Container/Deployment 分层可见。
4. stop/start 同一 Service.id。

## Acceptance / 节奏级

1. P0/P1 需求文案 **无 Open questions**，决策自洽。
2. P2 能按 P0 顺序部署本机并写 Deployment/Service/Container。
3. P3 明确 render 输入为 Version。
4. Redis TCP / K8s 不阻塞 P0–P2。

## Open questions / 待讨论（节奏级）

**P0/P1：无。**

P2/P3 实现向细节（不阻塞 P0/P1 业务语义）：

| 项 | 说明 |
|----|------|
| Container 是否落库 | 领域有 Container；存储派生 vs 落库属 Plan |
| 预览 API 与 deploy 是否强制同一 render 函数 | 原则已要求一致；实现细节 Plan |

以下**不是**待讨论业务歧义（已否决/已钉死）：

| 项 | 结论 |
|----|------|
| 未指定 Version 能否部署 | **不能**。Version 是部署前置，deploy 必选 Version |
| 旧「部署当前文件」兼容 | **不做兼容**；大版本一切换，以 Version 为准 |
| force recreate 等 | **部署操作选项**，挂 Deployment/本次操作，**不进 Version** |
| raw compose / overlay 优先级 | **无此产品概念**；功能完成后只有 Version → 渲染产物，无用户维护的 raw compose yaml |
| 「暴露子系统」 | 仅指现状 **域名/端口/路由（如 application_route → Traefik）** 是否纳入本版本化需求；**本期不做**，用词改称「对外访问/路由」，不另造子系统品牌 |

## Decisions / 决策

1. 用词与 P0 一致；Component 定名。
2. Deployment 唯一；deploy/stop/restart 共用。
3. ~~Environment 概念不实现~~ → **修订**：Environment 为 **Project 下业务实体**（`project_id`）；不做跨项目共享。详见 `20260721-cd-environment-expose-service.md`。
4. Version 结构化业务数据；compose **仅引擎侧渲染产物**，非用户编辑物、非 Version 本体。
5. 部署顺序：Service → Render(Version×Environment) → apply → Container。
6. **deploy 必选 Version**；Environment 落地后 **必选 environment_id**。
7. Version unpublished/published；published 不可改不可回退。
8. Service 状态主入口；last_successful_version_id；失败 faulted；唯一键 `(app, env, instance_key)`。
9. Health：runtime health 或 running。
10. application.status 不双写；Service 为运行真相。
11. **放弃历史兼容**；不做旧部署/双轨/兼容层；**大版本直接切换**到 Version 模型。
12. force recreate / remove volumes 等 = **部署选项**，属操作参数。
13. **无 raw compose 产品路径**；无 overlay 优先级问题。
14. ~~对外访问本期不进~~ → **修订 R1**：ExposeSpec 在 Version（http|tcp 同批）；环境侧接入在 Environment；Traefik MVP 每 (app,env) 至多一个 ingress target。
15. **P0、P1 Requirement 已 Accepted**（2026-07-21）。
16. **Environment/Expose/Service 基数修订 R1 Accepted**（2026-07-21）：`docs/requirement/20260721-cd-environment-expose-service.md`。
17. **环境接入策略修订 R2（已交付）**（2026-07-22）：域名由 Environment **IngressPolicy** 推导，**不**手填 per-component Binding；Environment **禁止**以 component 为用户配置维度。详见 `docs/requirement/20260722-cd-environment-ingress-policy.md`；Verification `docs/verification/20260722-cd-environment-ingress-policy.md`。与 R1 的 D4 EnvironmentBinding 用户 SoT **冲突时以 R2 为准**。
18. **应用类型修订 R3（Draft）**（2026-07-22）：Application.**kind** = `standard`（默认）\| `gateway`，**决定 Render 方式**；网关应用与普通应用分支渲染。详见 `docs/requirement/20260722-cd-application-kind-gateway.md`。不把 `code==traefik` 当主路径；K8s/宿主机网关/consumer external 网为扩展。

## Risk / 风险

1. 实现偏离 P0/P1 会回潮 Application 详情堆料。
2. 英文撞名（Service / Deployment）需在 API/文档限定语境。
3. 上线需一次性数据迁移生成 Version（无兼容入口兜底）。

## Sub-requirements index

| ID | 文档 | 状态 |
|----|------|------|
| P0 | domain | **Accepted**（Service/Environment 基数见 R1） |
| P1 | spec | **Accepted** |
| P2 | deploy | **Accepted** |
| P3 | render | **Accepted** |
| 修订 R1 | `20260721-cd-environment-expose-service.md` | **Accepted**；Plan Accepted；Verification Accepted |
| 修订 R2 | `20260722-cd-environment-ingress-policy.md` | Requirement **Accepted**；接在 R1 之后 |
| Spec R2 | `docs/spec/20260722-cd-environment-ingress-policy.md` | **Accepted**（含 ER） |
| Plan R2 | `docs/plan/20260722-cd-environment-ingress-policy.md` | **Accepted**；已实现 |
| Verification R2 | `docs/verification/20260722-cd-environment-ingress-policy.md` | **Accepted** |
| 修订 R3 | `20260722-cd-application-kind-gateway.md` | Requirement **Draft**（strict）；Application.kind |
| Spec R3 | — | **未建**（R3 Requirement Accepted 后） |
| Plan | `docs/plan/20260721-cd-application-version.md` | Implemented（Version 闭环） |
| Plan | `docs/plan/20260721-cd-environment-expose-service.md` | **Accepted**；已实现 |
| Verification | `docs/verification/20260721-cd-environment-expose-service.md` | **Accepted**（有条件） |

## User review notes

- 2026-07-21：多轮讨论收敛术语、Service、Version、部署顺序、ER。
- 2026-07-21：用户要求 P0/P1 需求分析到无遗留/不明确项 → **闭合 Open questions**，废止「compose 当 Version 存储」节奏表述。
- 2026-07-21：接受 Expose http|tcp 同批、一 App 多 Service + Traefik MVP 单接入 → 修订文档 Accepted。
- 2026-07-21：**更正** Environment 归属 Project（否决松散/跨项目共享）；Plan Draft 同步。
- 2026-07-21：澄清——Version 为部署前置；不做旧部署兼容；force recreate 属部署选项；无 raw compose；「暴露」= 域名/路由，本期不做。
- 2026-07-21：用户 **接受 P0/P1**；**可放弃历史兼容**。
- 2026-07-21：用户 **接受 P2/P3**，进入 **Plan**。
- 2026-07-22：产品反馈 Binding 手填域名 / 环境选 component 不当 → 新开 **R2** `20260722-cd-environment-ingress-policy.md` 挂入本节奏索引（Draft / strict）。
- 2026-07-22：R2 Requirement Accepted；Spec Draft：`docs/spec/20260722-cd-environment-ingress-policy.md`（含 ER）。
- 2026-07-22：用户「推进实现」→ Spec/Plan Accepted；migration 000010 + 产品代码落地；待 Verification。
- 2026-07-22：确认 Application.**kind**=gateway|standard 决定渲染；路线无覆盖 → 开 **R3** `20260722-cd-application-kind-gateway.md`（Draft）。
- 2026-07-22：用户要求先验证当前任务 → **R2 Verification Accepted**；R3 仍 Draft，未开 Spec/实现。
- 2026-07-22：用户要求展示提交建议后进入 R3 → 进入 **R3 Requirement** 评审（Draft 补场景/风险）；未 Accept、未写 Spec。
