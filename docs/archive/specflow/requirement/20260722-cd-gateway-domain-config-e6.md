# CD E6：网关领域（Gateway domain）
最后修改时间: 2026-07-22 18:47:16

Review status: Accepted

Flow mode: strict

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`  
前序：R3 **Accepted**；E5 Req/Spec/Plan **Accepted**、实现已提交、Verification **Draft**（`docs/verification/20260722-cd-gateway-component-mount-e5.md`）；E1 Verification **Accepted**。

**路线位置**：

```text
R3 Application.kind（已交付）
  -> E5 通用能力 + rest 动态路由（已实现；Verification Draft）
  -> E6 本文件：网关领域语义 + UI
       · 本期断言：仅容器托管 Traefik（managed）
       · 工作负载仍长在 Application(kind=gateway) + Version 之上
       · external / 用户自管 Traefik：未规划，挂后续
  -> E5-F1..F7 / E2–E4 等
```

Related：

| 文档 | 关系 |
|------|------|
| `docs/requirement/20260722-cd-application-kind-gateway.md` | kind 身份（R3） |
| `docs/requirement/20260722-cd-gateway-component-mount-e5.md` | 通用 mounts / rest PUT / 部署管线 |
| `docs/verification/20260722-cd-gateway-component-mount-e5.md` | E5 验证（Draft） |
| `docs/spec/20260722-cd-gateway-component-mount-e5.md` | E5 后续任务表（挂 **E6**） |
| 形态参考（非 SoT） | `data/cd.bak/traefik` |

## Terminology

| 中文 | 英文 | 说明 |
|------|------|------|
| 网关领域 | **Gateway domain** | 平台「入口 / 路由控制面」的一等产品语义与 UI；**不是**把工作负载拆出 Application 体系 |
| 网关配置 | **Gateway config** | 面向用户的网关领域配置（表单/API）；目标产品 SoT；由系统编译进 Version 载体 |
| 托管网关 | **Managed gateway** | **容器**部署的 Traefik；**本期唯一**支持的形态 |
| 外置网关 | **External gateway** | 用户自运维 Traefik + 仅登记 rest；**本期不做**（未规划） |
| 部署载体 | **Deploy vehicle** | `Application.kind=gateway` + **Version** + Service + Deployment；网关工作负载**必须**长在此之上 |
| 管理面 | **Control plane** | Traefik `providers.rest`；Orbit 唯一写者 PUT 动态路由 |
| 数据面 | **Data plane** | 80/443 等入口流量 |

## Background / 背景

1. E5 提供全 kind 共用的 mounts/env/content 与 rest 写路由后，仍缺**网关领域语义与 UI**（用户仍像配普通应用一样堆 Version）。  
2. **用户决策（2026-07-22）**：要 Gateway 语义和 UI；**容器托管 Traefik 仍长在 Application + Version 上**。  
3. **用户决策（同日）**：用户自行管理 Traefik（external）**尚未规划**；本期**断言全部为容器托管（managed）** 网关。  
4. **用户决策（同日）**：应用列表可按 `kind` 默认只展示常规（standard）应用；gateway 走网关面。  
5. **用户决策（同日）**：**不向后兼容、不迁移旧数据**。  
6. **用户决策（同日）**：Version **完整挂载能力保留**；在网关领域 UI 配齐之前，**必须能走 Version 能力**把 Traefik 配到可 rest（E5 路径不断档）。  
7. 动态路由主路径仍为 E5 **`providers.rest` PUT**。

## Goal / 目标

1. 建立 **Gateway domain** 产品语义与 UI（配置意图可读、可表单化）。  
2. **本期仅 managed**：容器 Traefik；工作负载 **Application(kind=gateway) + Version**，不另造脱离 Version 的运行时。  
3. Gateway 领域配置（到位后）**编译**进 Version/Component（含 mounts/env/content），再经既有 Deploy 管线落地。  
4. **在领域 UI 完备前**：用户/运维仍可用 **Version 完整挂载与 env** 实现 rest 可用的 gateway（与 E5 一致）。  
5. 应用列表默认 **standard**；gateway 不与普通应用列表混为主路径。  
6. 平台路由仍 Orbit 快照 + rest PUT。

## Non-goal / 非目标

1. **本期不做 external**（用户自管 Traefik / 只登记 rest）— 挂后续，不实现、不预留半成品切换向导。  
2. 不回退 File 写平台路由。  
3. 不把整包 Traefik dynamic JSON 当用户主编辑面。  
4. 不吞并 F1 PEM、F2 rest 鉴权、F3 多 gateway、F5 dashboard 产品化。  
5. **不做**旧数据迁移 / 兼容层 / 双轨读写。  
6. 不恢复 `code==traefik` 魔法。  
7. 本 Draft 不钉 DDL/列名（Spec）。

## 领域模型（核心）

### 本期唯一形态：managed（容器托管）

```text
[Gateway 语义 / UI]  （产品意图 · 领域配置）
        │  编译（领域 UI 到位后）
        ▼
Application (kind=gateway)  +  Version / Component
        │  mounts · env · content · ports …（E5 完整能力）
        ▼
Render → Deploy → Service / Container
        │
        ▼
平台 Route 快照 → PUT {rest}/api/providers/rest
```

| 项 | 规则 |
|----|------|
| 工作负载身份 | **必须** `Application.kind=gateway` |
| 规格与部署 | **必须** 经 **Version**（及 E5 挂载/env/content） |
| 领域 UI | 表达网关意图；**不**替代 Application/Version 存储与部署管线 |
| external | **断言不存在**（本期） |

### 配置 vs 部署（Managed 铁律）

| 层面 | SoT / 角色 | 用户编辑 |
|------|------------|----------|
| **Gateway 语义/UI** | 领域意图（目标主路径） | 是（领域表单） |
| **Version / Component** | **真实部署规格载体**；含完整挂载能力 | **允许且必须保留**（E5）；领域 UI 未覆盖前为主路径 |
| **compose / yml 文本** | Render 产物 | 否（引擎侧） |
| **动态路由** | 平台 Route 快照 | 否（平台 PUT） |

一句话：

- **网关要有领域语义和 UI**；  
- **容器 Traefik 仍长在 Application + Version 上**；  
- **Version 挂载能力完整保留**，领域 UI 配好之前靠 Version 实现 rest。

### External（挂账 · 非本期）

| 项 | 状态 |
|----|------|
| 用户自管 Traefik + 仅 rest URL | **未规划**；不进 E6 MVP |
| 与 managed 切换状态机 | **不做** |
| 文档 | 仅路线备注，不写 FR 验收 |

## 用户场景

### 场景 1：领域 UI 到位前 — Version 路径（必保）

1. 用户创建 `kind=gateway` Application。  
2. 在 **Version** 上用 E5 完整能力配置 mounts（含 content/seed|sync）、env、ports、command 等，使 Traefik 启用 `providers.rest` 且管理面可达。  
3. Deploy → 容器运行；平台对 rest PUT 动态路由。  
4. **不得**因 E6 文档存在而削弱或移除 Version 挂载/编辑能力。

### 场景 2：领域 UI 到位后 — Gateway 语义主路径

1. 用户进入 **网关** 产品面（列表默认不把 gateway 与 standard 混为同一主列表，见 Q2）。  
2. 编辑网关领域字段（镜像、端口意图、docker provider、网络、log 等 — Spec 列字段）。  
3. 系统将领域配置 **编译** 为该 Application 下的 **Version**（真实行）。  
4. 用户 Deploy/Stop 仍走 Application + Version + Service 管线。  
5. 需要时仍可进入 Version 查看/使用完整挂载能力（不设「永久锁死不可用 Version」）。

### 场景 3：应用列表

- 默认列表 **只展示常规应用（`kind=standard`）**。  
- gateway 应用在 **网关** 入口呈现（或 kind 筛选可见）；避免和业务应用抢同一默认列表心智。

## 配置范围草图（面向用户 · 非最终 schema）

### Managed（本期）

| 主题 | 例 |
|------|-----|
| 身份 | 绑定的 `kind=gateway` Application |
| 镜像 | Traefik image/tag |
| 数据面端口 | 80/443 发布意图 |
| 控制面 | rest 启用、管理口暴露方式 |
| Docker provider | network、exposedByDefault；socket → special mount 编译 |
| 网络 | top-level network 意图（E5 D4） |
| 观测 | log level |
| 持久化 | ACME 等可封装为领域开关，底层仍 mounts 机制 |

### External（非本期）

不列字段；后续单独立项。

## 与 Application.kind / Version / E5 的关系

| 概念 | E6 定位（已钉） |
|------|-----------------|
| `Application.kind=gateway` | 托管网关的**唯一**工作负载身份；**不**脱离 Application |
| Version | **必须**存在的部署规格载体；完整 E5 挂载/env/content **保留** |
| Gateway 语义/UI | 领域层；编译进 Version，**不**另起一套 deploy 引擎 |
| rest PUT / 单 gateway | E5 保留 |
| external | 非本期 |
| 旧数据 | **不迁移、不兼容** |

## MVP 边界（本需求拟交付）

| 纳入 E6 MVP | 不纳入 |
|-------------|--------|
| Gateway **语义 + UI**（managed） | external / 用户自管 Traefik |
| 工作负载 = Application(gateway) + Version | 形态切换向导 |
| 领域配置 → 编译 Version（目标路径） | F1 PEM / F2 鉴权 / F3 多 gateway / F5 dashboard |
| Version **完整挂载**不断档（过渡与长期能力） | 旧数据迁移 / 兼容层 |
| 列表默认 standard（kind 过滤） | 双轨 File+rest |

## Functional requirements

### FR1 — 领域与载体

1. 产品具备 **Gateway 语义与 UI**（与「普通应用 Version 堆料」区分）。  
2. 容器托管 Traefik **必须**建模为 `Application.kind=gateway` + **Version** 部署管线。  
3. 本期 **仅** managed；系统与文档 **断言**不存在 external 入口。  
4. 保存领域配置 ≠ 部署成功；须显式 Deploy。

### FR2 — Version 能力（硬约束）

1. Version / Component **完整挂载能力**（E5：logical/volume/special、content、content_mode、env、ports 等）**不得**因 E6 削弱或删除。  
2. **领域 UI 配置完备之前**，用户必须能仅凭 Version 能力部署可 rest 的 Traefik gateway。  
3. 领域 UI 到位后，默认引导走领域配置；Version 仍可作为完整规格面（查看/必要编辑 — Spec 钉「是否限制 gateway 的 Version 写」时不得违背本条「能力保留」）。

### FR3 — Managed 部署

1. 部署/升级/停止复用 Version + Service + Deployment。  
2. 领域配置编译产物为**真实 Version 行**（可回放、可挂 Deployment）。  
3. Preview 可展示合成结果；compose 文本非主编辑面。

### FR4 — 路由

1. 动态路由 = 平台 snapshot → rest PUT（E5）。  
2. 不回流 File 写路由。  
3. 生效 rest 目标与 gateway 部署/平台配置的关系 Spec 钉（本期无 external 分支）。

### FR5 — 列表与发现

1. 应用主列表默认仅 **`kind=standard`**（或等价「常规应用」过滤）。  
2. gateway 从 **网关** UI 进入为主路径。

### FR6 — 兼容与数据

1. **不向后兼容**；**不迁移**旧数据。  
2. 无双轨读写、无旧模型适配层。

## Acceptance / 验收（节奏级）

1. cadence 索引 E6；本文件 Accepted 后开 Spec。  
2. 存在 Gateway 语义/UI 方案边界（Spec）且 managed 路径清晰。  
3. 容器网关 **可证明**长在 Application + Version 上（非旁路进程）。  
4. Version 完整挂载路径在领域 UI 前后均可部署 rest-capable gateway。  
5. 默认应用列表不把 gateway 当作常规应用主列表项。  
6. 无 external 产品入口；无旧数据迁移脚本作为交付物。  
7. 与 E5 单 gateway / rest 唯一写者不冲突。

## Open questions

**无。**（Q1–Q6 已闭合，见 Decisions）

## Decisions（已钉）

1. **承认网关领域**；需要 **Gateway 语义和 UI**。  
2. **容器托管 Traefik 仍长在 `Application` + `Version` 之上**；不另造脱离 Version 的运行时。  
3. **本期断言全部为 managed（容器自托管 Traefik）**；**external / 用户自管 Traefik 未规划，不做**。  
4. **Q1**：领域语义/UI + 载体 = gateway Application + Version（配置意图在领域层，规格与部署在 Version）。  
5. **Q2**：列表/发现按 **`kind` 过滤**；**默认只展示常规（standard）应用**；gateway 走网关面。  
6. **Q3**：部署规格 **必须** 真实 **Version** 行（可回放）；领域编译目标即 Version。  
7. **Q4**：external **不需要**（本期）；不建「仅登记 rest、无 App」路径。  
8. **Q5**：**不向后兼容、不迁移旧数据**；无旧模型双轨。  
9. **Q6**：**Version 完整挂载能力保留**；领域 UI 配齐之前 **必须**能走 Version 实现 rest；领域配置是增强主路径，不是砍掉 Version。  
10. 动态路由仍 rest PUT；E5 机制保留；F1–F7 不吞并。  
11. 任务号 **E6**。

## Risk / 风险

1. 领域 UI 与 Version 表单并存 → 需 Spec 钉编译时机与「谁覆盖谁」，避免双 SoT 打架。  
2. 领域 UI 未完成前完全依赖 Version — 文档与验收必须显式保留 E5 路径。  
3. 无旧数据迁移 → 环境需按新语义重建 gateway（产品可接受）。  
4. 列表隐藏 gateway 后发现性依赖网关入口是否好找。  
5. rest 鉴权仍 F2；管理面网络暴露风险不变。

## User review notes

- 2026-07-22：承认网关特殊性；开 E6；曾讨论 managed/external 双形态。  
- 2026-07-22：E5 Verification Draft 后补 Q 提案表。  
- 2026-07-22：**用户闭合 Q1–Q6** —  
  - 要 Gateway 语义/UI；managed 仍 **Application+Version**；  
  - external 未规划，**断言仅自托管容器 Traefik**；  
  - Q2：`kind` 默认只展示常规应用；  
  - Q5：不兼容、不迁移旧数据；  
  - Q6：Version 完整挂载保留；领域 UI 前走 Version 实现 rest。  
- 2026-07-22：用户「推进任务」→ Requirement **Accepted**；进入 Spec。  
- 2026-07-22：Spec 审阅纠正 — **改表** `gateway_config`（1:1 Application）；禁止 application 脏列；导航 Gateway；**rest/域名从 Gateway**；Env 剥离网关配置；**移除全局** api_url/domain_suffix；「Intent」仅为草稿用词 → **Gateway config**。  
- 2026-07-22：**Environment.base_domain 移除，Gateway 负责**；Spec/Plan **Accepted**。  
- 2026-07-22：**Env 不留 domain_template**；Host=`{app_code}.{gateway.base_domain}`；**开始实现** E6-A/B。
