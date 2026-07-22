# CD E6：网关领域（Gateway domain）
最后修改时间: 2026-07-22 17:43:11

Review status: Draft

Flow mode: strict

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`  
前序：R3（`kind=gateway`）**Accepted**；E5（通用挂载 / rest 路由 / Component 全规格）Requirement/Spec/Plan **Accepted**（实现中）

**路线位置**：

```text
R3 Application.kind（已交付）
  -> E5 通用能力 + rest 动态路由（实现中）
  -> E6 本文件：网关领域 — 双运行形态 + 面向用户的领域配置
       · managed：容器 Traefik（平台配置 + 借 Version 部署）
       · external：宿主机 Traefik（用户自部署，只登记 rest 管理面）
  -> E5-F1..F7 / E1–E4 等
```

Related：

| 文档 | 关系 |
|------|------|
| `docs/requirement/20260722-cd-application-kind-gateway.md` | kind 身份（R3） |
| `docs/requirement/20260722-cd-gateway-component-mount-e5.md` | 通用 mounts / rest PUT / 部署管线 |
| `docs/spec/20260722-cd-gateway-component-mount-e5.md` | E5 后续任务表（挂 **E6**） |
| 形态参考（非 SoT） | `data/cd.bak/traefik` |

## Terminology

| 中文 | 英文 | 说明 |
|------|------|------|
| 网关领域 | **Gateway domain** | 平台「入口 / 路由控制面」的一等产品域；**不是** ordinary Application 详情里的 Version 表单子集 |
| 网关配置 | **Gateway config** | 面向用户的结构化配置（表单/API）；**产品 SoT**；**不是**用户编辑的 Version 行或 traefik.yml 文本 |
| 网关运行形态 | **Gateway run mode** | `managed` \| `external`（见下） |
| 托管网关 | **Managed gateway** | 基于**容器**部署的 Traefik：平台生成规格并**部署**；配置对人，部署**借** Version/Service/Deploy 能力实现 |
| 外置网关 | **External gateway** | **宿主机**（或他处）用户自运维的 Traefik：平台**不**部署该进程；用户将 **rest 管理面地址**登记给 Orbit |
| 管理面 | **Control plane** | Traefik `providers.rest` 等可写端点；Orbit 唯一写者 PUT 动态路由 |
| 数据面 | **Data plane** | 80/443 等入口流量路径 |
| 部署载体 | **Deploy vehicle** | managed 模式下：内部由 Version / Component / Service / Deployment **机制**执行 deploy；**用户不**把 Version 当网关配置编辑面 |

## Background / 背景

1. E5 把 mounts/env/content 做成全 kind 通用能力后，网关仍落在「手写 yml / env」上，业务规则不可理解。  
2. **用户决策（2026-07-22）**：承认网关特殊性，定义网关领域配置。  
3. **用户决策（同日 · 本修订）**：网关领域计划支持 **两种 Traefik 运行形态**：  
   - **容器托管**：平台负责配置与部署；**配置面向用户，不面向 Version**；部署**借助 Version 能力**落地。  
   - **宿主机外置**：用户自行部署 Traefik，只需把 **rest 地址**暴露给平台管理。  
4. 动态路由主路径仍为 E5 定案的 **`providers.rest` PUT**；E6 解决「入口从哪来、配置长什么样、谁部署进程」。

## Goal / 目标

1. 建立 **Gateway domain**：独立于「standard 应用 Version 编辑」的产品模型与 UX。  
2. 支持 **`managed`（容器）** 与 **`external`（宿主机/外置）** 两种 run mode，并统一到同一控制面写路径（rest）。  
3. **Managed**：用户只编辑 **Gateway config**（领域字段）；系统将配置 **编译** 为可部署规格，再经现有 **Version → Render → Deploy** 管线执行；用户无感 Version 作为网关配置表。  
4. **External**：用户只登记 **rest base URL**（及后续鉴权等最小连接信息）；平台**不**创建/运维该 Traefik 容器。  
5. 平台路由仍 Orbit 快照 + rest PUT；与 run mode 解耦。

## Non-goal / 非目标

1. 不回退 File/`dynamic_route_dir` 写平台路由。  
2. 不把整包 Traefik dynamic JSON 给用户编辑。  
3. 不在 E6 吞并 F1 证书、F2 rest 鉴权细节、F3 多 gateway 编排、F5 dashboard 产品化（可预留字段，交付边界 Spec 钉）。  
4. 不要求 external 模式支持平台改 Traefik **静态**配置（只消费已暴露的 rest）。  
5. 不恢复 `code==traefik` 魔法；与 Application.kind / Gateway 实体关系 Spec 钉。  
6. 本 Draft 不钉 DDL/列名（Spec）。

## 领域模型（核心）

### 双运行形态

```text
                    ┌─────────────────────────────────────┐
                    │         Gateway domain config         │
                    │   （面向用户 · 产品 SoT · 非 Version）  │
                    └─────────────────┬───────────────────┘
                                      │
              ┌───────────────────────┴───────────────────────┐
              │                                               │
              ▼                                               ▼
     run_mode = managed                              run_mode = external
     （容器 Traefik）                                  （宿主机 / 自部署 Traefik）
              │                                               │
              │  配置 → 编译部署规格                              │  仅登记
              │  → Version 载体（机制）                          │  control_plane.rest_url
              │  → Service / Deploy / Container                 │  （可选：探活）
              │                                               │
              └───────────────────────┬───────────────────────┘
                                      │
                                      ▼
                         平台 Route 快照 → PUT {rest}/api/providers/rest
```

| 形态 | 英文 | 谁部署 Traefik | 用户配置什么 | 平台部署什么 |
|------|------|----------------|--------------|--------------|
| **托管** | `managed` | **平台**（容器） | 入口/控制面/docker provider 等**领域字段** | Traefik 工作负载（经 Version 管线） |
| **外置** | `external` | **用户**（宿主机或任意位置） | **rest 管理面地址**（+ 连接约束） | **不**部署 Traefik；只写路由 |

### 配置 vs 部署（Managed 铁律）

| 层面 | SoT | 用户是否直接编辑 |
|------|-----|------------------|
| **网关配置** | Gateway config（领域） | **是** — 产品主路径 |
| **Version / Component** | 由配置 **编译生成** 或平台维护的内部载体 | **否** — 非网关配置 UX；运维可观测/调试另议 |
| **compose / yml 文本** | Render 产物 | **否** — 引擎侧 |
| **动态路由** | 平台 Route 快照 | **否** — 平台写 rest |

一句话：**配置面向用户，不是面向 Version；部署借助 Version 的能力实现。**

### External 铁律

| 项 | 规则 |
|----|------|
| 部署 | 平台 **不** 对 external Traefik 做 compose up/down |
| 必填 | 可达的 **rest base URL**（`api_url`） |
| 可选（后续） | 鉴权凭据（F2）、探活、展示用数据面端口说明 |
| 静态配置 | 用户自责启用 `providers.rest`（及网络可达）；平台文档契约 + 连接校验 |
| 与 managed 互斥 | 同一「当前入口」身份下二选一（与 E5 单 gateway 一致，多实例归 F3） |

## 用户场景

### 场景 1：托管网关（容器）

1. 用户进入 **网关** 产品面（非「像配 ordinary app 一样改 Version 挂载」）。  
2. 选择 **managed**，填写领域配置（镜像 tag、80/443、rest 启用策略、docker 网络意图、log level 等 — Spec 列字段）。  
3. 保存配置 → 系统编译内部部署规格。  
4. 用户点 **部署/升级** → 走 Service + Version 管线（用户不感知填 traefik.yml）。  
5. 平台对 rest 写动态路由；standard 应用 Expose 经 labels 进同一数据面网络语义。

### 场景 2：外置网关（宿主机）

1. 用户在宿主机（或外部）自行安装/运行 Traefik，静态配置启用 rest，管理面仅管理网可达。  
2. 在 Orbit **网关** 面选择 **external**，登记 **rest URL**。  
3. 平台校验连通（可选 GET 观测）后，**只** PUT 路由快照；**无** 容器部署步骤。  
4. 换 rest URL / 停用外置入口有明确产品语义（Spec）。

### 场景 3：形态切换

- managed ↔ external 是否允许、是否要求先 stop 旧载体：Spec 钉（默认倾向：**有运行入口时切换需显式步骤**，避免双写 rest）。

## 配置范围草图（面向用户字段 · 非最终 schema）

### 共有（两种形态）

| 主题 | 说明 |
|------|------|
| `run_mode` | `managed` \| `external` |
| 入口身份 | 与 Project/单 gateway 策略绑定（E5 D9） |
| 路由写目标 | rest base；managed 可部署后推导，external 用户填写 |

### Managed 专有

| 主题 | 例 |
|------|-----|
| 镜像 | Traefik image/tag |
| 数据面端口 | 80/443 发布意图 |
| 控制面 | rest 启用（默认开）、insecure/后续鉴权模式、管理口是否仅内部 |
| Docker provider | 启用、network、exposedByDefault；socket 经 special mount 编译进载体 |
| 网络 | 与 Environment 合并的 top-level network 意图（接 E5 D4） |
| 观测 | log level |
| 持久化附件 | acme 等仍可走通用 mount **机制**，产品字段可封装「启用 ACME 存储」而非让用户写路径文本 |

### External 专有

| 主题 | 例 |
|------|-----|
| `rest_url` | 必填；平台 PUT 目标 |
| 连通性 | 保存/启用时探测（实现 Plan） |
| 说明性元数据 | 可选备注、预期数据面地址（不驱动部署） |

## 与 Application.kind / Version / E5 的关系

| 概念 | E6 定位 |
|------|---------|
| `Application.kind=gateway` | 可保留为「此应用是网关工作负载身份」；**或** Gateway 实体与 Application 1:1 映射 — Spec 二选一钉死 |
| Version | **Managed 部署载体**：内部版本化部署规格；**不是**用户网关配置表 |
| Component mounts/env | E5 **机制**；由 Gateway config **编译**写入载体，用户不直接维护 rest/yml |
| rest PUT / 单 gateway | E5 **保留**；两种 run mode **共用**写路径 |
| 部署工作区平等 | Managed 仍无统一 deploy 物化；External **无** gateway 工作区要求 |

## Functional requirements（Draft）

### FR1 — 领域与形态

1. 系统支持 `run_mode ∈ {managed, external}`。  
2. 用户主配置面为 **Gateway config**，**不是** Version 编辑页。  
3. 同一时刻生效的入口控制面地址唯一（接 E5 单 gateway；多实例 F3）。

### FR2 — Managed

1. 用户仅维护领域字段；系统保证 rest 控制面契约被编译进部署规格。  
2. 部署/升级/停止 **复用** Version + Service + Deployment 能力。  
3. Preview 可展示「将部署的」合成结果（调试）；默认 UX 不要求编辑 yml。

### FR3 — External

1. 用户登记 rest URL 后即可成为路由写目标。  
2. **不**创建/启动/停止用户 Traefik 进程。  
3. rest 不可达时路由同步失败须明确错误（不静默成功）。

### FR4 — 路由

1. 两种形态下动态路由均为平台 snapshot → rest PUT。  
2. 不回流 File 写路由。

## Acceptance / 验收（节奏级 · Draft）

1. cadence 索引 E6；本文件 Accepted 后开 Spec。  
2. 产品能区分并完成 **managed** 与 **external** 两条路径。  
3. Managed：**配置 UX ≠ Version 表单**；部署可观测到 Version/Service 机制被使用。  
4. External：仅 rest 登记即可接收平台路由；无容器 deploy 依赖。  
5. 两种形态均不要求用户粘贴 `providers.rest` yml 作为主路径（external 静态侧用户自运维除外）。

## Open questions（待闭合）

| ID | 问题 | 倾向 |
|----|------|------|
| Q1 | Gateway config 实体挂在哪？（独立 Gateway / Application 扩展 / Project 单例） | Project 下至多一个当前入口 + 配置实体；与 Application 映射 Spec 定 |
| Q2 | Managed 的 Version 是否对用户隐藏、只读可见、还是「高级」可进 | 默认隐藏；高级只读 |
| Q3 | Managed 编译产物是否仍物化为真实 Version 行（可回放） | 是，便于 Deployment 审计 |
| Q4 | External 是否允许不建 `kind=gateway` Application | 倾向：external 可不建网关 App，只登记控制面；或轻量占位 — 待钉 |
| Q5 | managed→external 切换时已部署容器是否自动 stop | 倾向：显式确认后 stop |
| Q6 | rest 鉴权字段是否进 E6 最小集还是纯 F2 | 最小：URL；鉴权 F2 |

## Decisions（已钉 · 路线 / 本文件）

1. **承认网关领域**；配置一等、可理解。  
2. **双形态**：`managed`（容器，平台配置+部署）与 `external`（宿主机/自部署，只暴露 rest）。  
3. **Managed：配置面向用户，不是面向 Version；部署借助 Version 能力。**  
4. **External：用户自部署；平台只消费 rest 管理面。**  
5. 动态路由仍 rest PUT；E5 机制保留；E6 不静默并入 E5 实现。  
6. 任务号 **E6**；F1–F7 不吞并。

## Risk / 风险

1. Version 既是载体又曾是配置面 → UX 必须严格分流，否则双 SoT。  
2. External rest 无鉴权 → 安全（F2）；须网络约束。  
3. 形态切换双写/残留容器 → 需显式状态机。  
4. Managed 编译规则膨胀 → Spec 控制字段面，高级逃生口默认关。

## User review notes

- 2026-07-22：承认网关特殊性；开 E6。  
- 2026-07-22：**领域设计钉双形态** — 容器托管（配置对人、部署借 Version）与宿主机外置（自部署 + rest 地址）。  
