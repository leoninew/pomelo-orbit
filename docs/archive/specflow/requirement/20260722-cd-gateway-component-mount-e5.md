# CD E5：gateway 挂载与 Component 全规格
最后修改时间: 2026-07-22 17:38:46

Review status: Accepted

Flow mode: strict

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`  
挂账来源：R3 扩展 **E5**（`docs/requirement/20260722-cd-application-kind-gateway.md`；R3 Verification **Accepted**）

**路线位置**：

```text
R3 Application.kind（已交付）
  -> E5 本文件：
       Version 表单（挂载/环境变量）+ RuntimeConfig
       + 平台路由改为 providers.rest（废除 dynamic 目录写路由）
       + gateway 与 standard 部署目录结构平等（无 kind=gateway 特殊工作区）
  -> E6 网关领域配置（Gateway domain config）— 见 cadence / `20260722-cd-gateway-domain-config-e6.md`
  -> E1 consumer 网；dashboard/访问能力延后
```

Related：

| 文档 | 关系 |
|------|------|
| `docs/requirement/20260722-cd-application-kind-gateway.md` | E5 挂账定义 |
| `docs/verification/20260722-cd-application-kind-gateway.md` | R3 已验收；本需求不得回退 kind 分支 |
| `docs/requirement/20260721-cd-application-version-spec.md` | 挂载 **逻辑路径** 规则（Accepted） |
| `docs/requirement/20260622-backend-go-cd-physical-path.md` | 物理路径解析既有能力参考 |
| 形态参考（非 SoT） | `data/cd.bak/traefik/docker-compose.yml` |
| 现状路由推送（将废弃） | `internal/infrastructure/external/traefik/route.go`（写文件到 `dynamic_route_dir`） |
| 探针 | `data/cd.bak/traefik`（rest 实测；非生产 SoT） |

## Terminology

| 中文 | 英文 | 说明 |
|------|------|------|
| 组件全规格 | **VersionComponent full spec** | name/image + ports / command / **表单化挂载** / **表单化环境变量** 等 **结构化字段**（非巨型自由 JSON） |
| 逻辑挂载源 | **logical mount source** | Version 内相对应用数据根（或命名卷）；禁止绝对路径作唯一真相 |
| 挂载文件正文 | **mount content** | 仅 **logical + 文件型** mount 可选字段；Deploy 物化时写入宿主机文件（≈ ConfigMap 正文） |
| 内容写入模式 | **content_mode** | `seed`（默认，仅文件不存在时写）\| `sync`（每次 Deploy 用 Version content 覆盖） |
| 物理挂载源 | **physical mount source** | Render 时解析后的宿主机路径 |
| 运行时配置 | **RuntimeConfig** | 部署时由 **Service = Version × Environment × 运行时推导** 产生；**不**写入 Version 本体 |
| 占位 | **placeholder** | VersionComponent 中声明的可替换项；部署时填必填/默认值 |
| 网关应用 | **gateway Application** | `kind=gateway` |
| REST 路由推送 | **providers.rest** | Traefik 可写配置通道：`PUT /api/providers/rest` **全量 JSON**；E5 **采用为平台动态路由主写入面** |
| 路由快照 | **route snapshot** | Orbit 侧维护的 rest 命名空间全量配置（结构化模型 → 序列化为 PUT body）；**不**塞进 Version 巨型 JSON |

## Background / 背景

1. R3 仅有 kind 分支与最小 Component（UI 多半只有 name/image）；gateway Preview 无法接近可运行入口。
2. 旧实现将 **`.env` / 部分配置**与 Version 或固定工作区耦合；平台向 **traefik dynamic 目录写路由文件**，迫使 gateway 挂载该目录 → 组件与平台写路径缠在一起，且 Windows file watch/HUP 脆弱。
3. 产品方向（用户钉死，含路由通道定案）：
   - Version 负责**规格**（表单化目录挂载、环境变量声明）。
   - **配置能力下沉到部署**：`Service ≈ Version + Environment + RuntimeConfig`。
   - top-level network = **`kind=gateway` × Environment** 合并。
   - dashboard / container 访问能力 **延后**。
   - 挂载解析与表单能力 **所有 kind 共用**。
   - **该建模的建模**；**禁止** Version 巨型 JSON 主 SoT。
   - **动态路由改由 `providers.rest` 管理**：**不再**基于目录挂载 / File provider 写盘管理平台路由；**不再**为 `kind=gateway` 做特殊部署目录结构。
4. Traefik 多版本实测（3.6.10 / 3.6.23 / 3.7.8）支撑 rest 可写；`api@internal` 仍只读。

## Traefik v3 配置通道调查（E5 输入 · 已闭环）

**实测环境**（2026-07-22）：

| 项 | 值 |
|----|-----|
| 探针 compose | `data/cd.bak/traefik/docker-compose.yml`（`TRAEFIK_IMAGE` 可覆盖，默认 `traefik:latest`） |
| 静态配置 | `data/cd.bak/traefik/data/traefik.yml`（`providers.rest.insecure: true`） |
| 管理口 | `http://127.0.0.1:8080` |

**版本矩阵**：

| 镜像 tag | 解析版本 | `providers.rest` | `api` 写 | `PUT /api/providers/rest` |
|----------|----------|------------------|----------|---------------------------|
| `traefik:3`（当时） | **3.6.10** | 有 | **405** | **200** 全量替换 |
| `traefik:v3.6` | **3.6.23** | 有 | **405** | **200** |
| `traefik:latest` | **3.7.8** | 有 | **405** | **200** |

### 1. 官方 API（`api@internal`）— 只读

GET `/api/http/routers` 等 **200**；POST/PUT/DELETE **405**。仅观测（`ListRouters` 继续可用）。

### 2. REST Provider — 可写（E5 采用）

- 启用：`--providers.rest` / `providers.rest.insecure`（insecure 时自动挂在 entrypoint `traefik`：`PathPrefix(\`/api/providers\`)`）。
- **仅 PUT** `/api/providers/rest`，body **JSON**（YAML **400**）；GET/POST/PATCH/DELETE **405**。
- **全量替换** rest 命名空间；**不** merge 单条。
- 清空：`{"http":{"routers":{},"services":{}}}`（`{}` 不清）；注意 `providersThrottleDuration`（默认 2s）后再观测。
- 资源名 `@rest`；可跨 provider 引用（如 `api@internal`）；与 Docker/File 隔离。
- 安全：insecure 无鉴权 → Spec 须钉管理面网络/鉴权（至少不裸奔公网）。

### 3. 其它通道（本需求不采用为平台路由主路径）

| 方式 | E5 态度 |
|------|---------|
| **File Provider** 写 `dynamic_route_dir` | **废除**作为平台动态路由管理手段 |
| HTTP Provider / KV | 非本期；rest 已满足「零写 gateway 数据盘」 |
| Docker labels | standard Expose 仍可用；**不**替代平台全局路由推送 |

### E5 决策含义（路由通道 · 已定案）

1. **平台动态路由主写入面 = `providers.rest`（PUT 全量 JSON）**。  
2. **废除**基于 **dynamic 目录挂载 + 写文件** 管理平台路由（含 `RouteManager.deployRouteFile` 路径语义）。  
3. **不**为 `kind=gateway` 保留特殊部署目录结构（无「gateway 工作区必须含 dynamic/certs/… 供平台写路由」）。  
4. gateway 与 standard **部署物化平等**：同一套工作区/挂载解析；gateway 仅保留 kind 语义差异（如 top-level network 合并、路由控制面可达性等，Spec 列清单）。  
5. 路由快照由 **Orbit 平台侧**结构化维护并 PUT；**禁止**塞进 Version 巨型 JSON。  
6. `api@internal` 仍只读观测。  
7. rest 虽官方文档未列 Supported Providers，但 v3.6–3.7 实测可用；采用时在 Spec 记录约束（全量快照、并发、鉴权、版本下限）。  
8. 探针 `data/cd.bak/traefik` 仅实验，非生产 SoT。

## Goal / 目标

1. **Version UI/API**：表单 + 结构化字段维护挂载、环境变量（及必要 ports/command）；**禁止** Version 巨型 JSON 主 SoT。  
2. **部署模型**：`Service = Version + Environment + RuntimeConfig`；废止 Version↔`.env` 耦合。  
3. **挂载契约（全 kind）**：逻辑路径存 Version；Render 解析物理路径；Preview/Deploy 同一函数。  
4. **top-level network**：`kind=gateway` + Environment **合并**渲染。  
5. **平等部署**：gateway 与 standard **同一部署目录/工作区策略**；**无** kind=gateway 特殊路由目录树。  
6. **平台路由**：改 **`providers.rest` PUT**；**不再**依赖 File provider 目录挂载写路由。  
7. **不兼容拟合**：识别失效旧用例，**删改**而非双轨。

## Non-goal / 非目标

1. E1 consumer 自动 join、E3 host、E4 K8s。  
2. dashboard / Service 内容器访问能力产品化（**延后**）。  
3. raw compose 编辑；`code==traefik` 魔法。  
4. 继续维护 File provider 平台路由写盘（**明确不做**）。  
5. 为旧 `.env` / 旧 dynamic 目录路由路径做兼容层。  
6. 多厂商网关适配器 UI。  
7. 用「整包 Traefik dynamic JSON」顶替 VersionComponent 结构化规格（路由快照属**平台**，非 Version 表单）。  
8. 本期切换 HTTP provider / KV（rest 已定主路径；日后若换通道不回流 File 写盘）。

## 问题分解（闭合后）

### A. Version 表单与建模

| 能力 | 决策 |
|------|------|
| 目录挂载 | **表单 + 结构化字段**（业务数据挂载；**不是**平台 dynamic 路由目录） |
| 文件挂载正文 | **表单可选 content + content_mode**；仅 logical 文件型；非独立 ConfigMap 实体；**非** kind=gateway 魔法生成 |
| 环境变量 | **表单 + 结构化字段**（静态 vs 占位；与 RuntimeConfig 分界） |
| ports / command 等 | 规格闭环；UI 粒度 Spec 定 |
| JSON 角色 | 传输可 JSON；**非**用户编辑巨型 Version blob |

### B. 部署与 RuntimeConfig

| 概念 | 归属 |
|------|------|
| Version | 静态规格 + 占位声明 |
| Environment | 环境策略（含 gateway 网络合并） |
| RuntimeConfig | **部署时**生成/填写 |
| 部署工作区 | **全 kind 同一策略**；**无** gateway 专用 dynamic/certs 平台写树 |

### C. Network

| 项 | 决策 |
|----|------|
| top-level `networks` | **`kind=gateway` + Environment 合并** |
| service 级 networks | Spec 钉 |

### D. 路由（定案）

| 项 | 决策 |
|----|------|
| 平台动态路由写入 | **`providers.rest` PUT 全量 JSON** |
| File / `dynamic_route_dir` 写路由 | **废除** |
| gateway 部署目录为路由服务 | **废除特殊结构** |
| 官方 `api@internal` | **只读**观测 |
| 路由数据归属 | Orbit **平台 route snapshot**（结构化）；非 Version 挂载、非 Version 巨型 JSON |
| gateway 静态配置要求 | gateway 应用/镜像侧须启用 rest（及管理面可达）；Spec 钉如何配置/校验（可模板，**不**写死 code=traefik） |

### E. 延后

- dashboard / `api@internal` 对外访问 / Service 内访问能力。

## User scenarios / 用户场景

### 场景 1：表单维护挂载与环境变量

用户编辑 unpublished Version：表单维护 **业务**挂载与环境变量 → 保存回读 → Preview 反映 volumes / environment。**不**要求配置 dynamic 路由目录。

### 场景 2：部署时补齐运行时配置

Deploy：按占位推导 RuntimeConfig；**不**回写污染 Version。

### 场景 3：gateway 网络自动合并

`kind=gateway` Deploy：top-level network 来自 kind × Environment；用户无需粘贴 bak networks 块。

### 场景 4：standard 共用挂载解析

`kind=standard` 同表单挂载 + 解析；**无** gateway 网络提供者语义。

### 场景 5：平台路由经 REST，部署目录无差别

平台推送/更新动态路由：Orbit 维护 route snapshot → **PUT** 到 gateway 的 rest 端点。  
gateway Deploy 的工作区/挂载 **不**因「要接平台路由」而出现特殊 dynamic 目录要求；与 standard **结构平等**。

### 场景 6：取消某路由 / 重建快照

平台按全量快照 PUT（或显式清空 rest 命名空间）；**不**再删改宿主机上某 yaml 文件。

## Functional requirements（方向级）

### FR1 — Version 表单：挂载与环境变量（结构化）

1. UI 表单编辑 **目录挂载**、**环境变量**（主入口非高级 JSON）。  
2. API 与字段一致；**禁止**单一巨大自由 JSON 作 Component SoT。  
3. 逻辑挂载源校验（对齐 P1）。  
4. 挂载表单**不包含**「平台 dynamic 路由目录」强制项。  
5. **文件型 logical mount** 可编辑可选 **content**（正文）与 **content_mode**（`seed`\|`sync`）；目录 / volume / special **禁止** content。  
6. 平台 **不**因 `kind=gateway` 自动生成 `traefik.yml`；正文由用户/示例 Version 的 mount content 提供。  
7. content 体积上限 Spec 钉；`acme.json` 类状态文件：无 content，保持空文件 seed、永不覆盖已有。

### FR2 — RuntimeConfig（部署面）

1. 部署路径：Version + Environment + **RuntimeConfig**。  
2. 占位 → 必填/默认/覆盖。  
3. 废止 Version 绑定 `.env` 主路径。  
4. Preview 可展示将用运行时键（密钥策略 Spec）。  
5. RuntimeConfig **不**塞回 Version 巨型 JSON。

### FR3 — 挂载解析（全 kind）

1. 全 kind 共用解析；Preview/Deploy 同一入口。  
2. 特殊源白名单（docker.sock 等）Spec 钉。  
3. 失败明确报错。  
4. **无**「仅 gateway 才解析/创建 dynamic 子树」分支。  
5. Deploy 物化：logical 目录 MkdirAll；logical 文件：无 content 则 touch 空文件；有 content 则按 content_mode 写入（seed 不覆盖 / sync 覆盖）。

### FR4 — top-level network 合并

1. `kind=gateway` 时合并 Environment（及平台约定）生成 top-level network。  
2. 网络名/驱动规则产品化（不与 domain_suffix 混淆）。  
3. standard 不注入提供者 top-level（E1 另论）。

### FR5 — 平台路由：providers.rest（废除 File 写路由）

1. `RouteManager`（或后继）以 **PUT `/api/providers/rest`** 推送 **全量** JSON 快照；观测仍用 `GET /api/http/*`。  
2. **删除/停用**向 `dynamic_route_dir` 写 yaml 并依赖 File watch 的平台路径。  
3. **删除** gateway 部署「必须挂载 dynamic 供平台写路由」的产品/实现/测试假设。  
4. **删除** kind=gateway 特殊部署目录结构（为路由服务的 data/dynamic/… 平台约定）。  
5. Orbit **自管 route snapshot**（并发、全量组装、失败重试 Spec）；body 由结构化模型生成，**不是** Version 自由 JSON。  
6. gateway 控制面地址/启用 rest 的配置来源 Spec 钉（Environment / 平台设置 / Service 元数据等）。  
7. 鉴权与网络隔离：禁止默认裸奔公网 rest 写入口（Spec）。

### FR6 — 不兼容拟合

1. 失效旧用例删除或改写（见下表）。  
2. 无双轨、无「File 与 rest 并行写路由」。

## 失效/待识别旧用例（初表，Spec/实现期补全）

| 旧假设 / 路径 | 处理 |
|---------------|------|
| compose `env_file: .env` 由 Version/工作区固定耦合 | **废弃**；改 RuntimeConfig |
| Version 或模板写死绝对路径挂载 | **拒绝**；逻辑路径 + 解析 |
| gateway Component **必须**挂载 `…/dynamic` 供平台写路由 | **废弃** |
| `RouteManager` 写 `dynamic_route_dir` YAML + File provider | **废弃**；改 rest PUT |
| `RouteManager` 路径写死 `data/cd/traefik/...` / code=traefik | **废弃**；不拟合唯一 traefik 应用 |
| kind=gateway 特殊部署目录树（为路由/平台写盘） | **废弃**；与 standard 平等 |
| Windows HUP / file watch 驱动的路由热更 | **废弃**（随 File 路由路径） |
| bak 对照要求 `api@internal` dashboard | **延后** |
| 测试/文档要求 Preview 含 env_file | **改写** |
| 用「整包 dynamic JSON」当 Version 编辑面 | **拒绝**；路由快照在平台 |

## Acceptance / 验收（需求级）

1. cadence 索引 E5；Q 表全部闭合或显式延后。  
2. Spec 钉：表单字段、占位、RuntimeConfig 存储、network 合并、挂载语法、**rest 快照模型与推送时机**、控制面地址、鉴权。  
3. 表单可完成挂载 + 环境变量 → 保存 → Preview 可见。  
4. 部署可完成 RuntimeConfig 推导；无 Version-.env 耦合。  
5. gateway top-level network 来自 kind×Environment 合并。  
6. 全 kind 共用挂载解析与**同一部署目录策略**；无 gateway 路由专用目录结构。  
7. 平台动态路由经 **rest PUT**；**无** File `dynamic_route_dir` 写路由路径。  
8. Version/Component **无**巨型自由 JSON SoT；路由快照不进 Version 表单。  
9. 失效旧用例已处理；`go test` + 前端门禁（改 UI 时）。

## Open questions / 待讨论

| # | 状态 | 结论 |
|---|------|------|
| Q1 UI 表单 | **闭合** | Version **基于表单**实现目录挂载、环境变量 |
| Q2 .env / 配置归属 | **闭合** | 配置归 **部署/RuntimeConfig**；占位→必填/默认 |
| Q3 top-level network | **闭合** | **`kind=gateway` + Environment 合并** |
| Q4 dashboard/访问 | **闭合（延后）** | 访问能力 **延后** |
| Q5/Q6 路由通道 | **闭合（定案）** | **采用 `providers.rest`**；**废除** File/目录挂载管理平台路由；**废除** kind=gateway 特殊部署目录结构 |
| Q7 兼容 | **闭合** | 失效用例改写/删除，不兼容拟合 |
| Q8 解析范围 | **闭合** | 挂载解析与表单 **全 kind 共用** |
| Q9 建模粒度 | **闭合** | 结构化字段 + 表单；禁止 Version 巨型 JSON |
| Q10 部署目录 | **闭合** | gateway **无**相对 standard 的路由专用目录特殊结构 |

### 仍待 Spec 细化（非产品方向分歧）

| 项 | 说明 |
|----|------|
| S1 | 挂载表单字段模型（逻辑目录 / 命名卷 / 白名单特殊源） |
| S2 | 环境变量：静态 vs 占位语法与校验 |
| S3 | RuntimeConfig 落库位置 |
| S4 | Environment 上哪些字段参与 gateway 网络合并 |
| S5 | ports/command 是否同一期表单必达 |
| S6 | **route snapshot** 结构、组装来源（Expose/Service/…）、PUT 触发点（deploy / expose 变更 / 启动对账） |
| S7 | 既有 `*_json` 列与结构化 API 边界 |
| S8 | rest 控制面 base URL / 鉴权 / insecure 约束；多 gateway 实例寻址 |
| S9 | 全量替换下的并发与「读-改-写」：Orbit 是否唯一写者、如何避免丢路由 |
| S10 | 迁移：清理旧 dynamic 文件、配置项 `dynamic_route_dir` 下线步骤 |

## Decisions（已钉）

1. E5 = R3 挂账落地，不另开 E6。  
2. Version 表单：**目录挂载 + 环境变量**（结构化，非巨型 JSON）。  
3. **RuntimeConfig** 属部署面；废止 Version↔`.env` 耦合。  
4. top-level network = **gateway kind × Environment** 合并。  
5. dashboard/访问能力 **延后**。  
6. **平台动态路由 = `providers.rest` PUT 全量 JSON**；**废除** File/目录挂载写路由；**废除** kind=gateway 特殊部署目录结构；组件部署平等。  
7. **不兼容拟合**；挂载解析 **全 kind 共用**。  
8. 无 code 魔法；compose 仍为渲染产物。  
9. **建模原则**：该建字段建字段；Version 不做巨大 JSON；**路由快照在平台侧结构化维护**。  
10. `api@internal` 仅观测，不作写入面。  
11. **网关领域配置不在 E5 静默扩大**：用户后续决策——网关静态契约不得仅依赖 opaque Version 文本 → 独立扩展 **E6**（`docs/requirement/20260722-cd-gateway-domain-config-e6.md`）。E5 仍交付通用挂载/rest 动态路由；**`providers.rest` 启用等控制面契约的一等建模归 E6**。

## Risk / 风险

1. RuntimeConfig 存储不当 → 密钥进 Version 或无法审计。  
2. rest **全量替换** + 多写者 → 路由丢失；须 Orbit **唯一写者**或严格快照锁（S9）。  
3. rest 为遗留能力，未来大版本可能移除 → 锁定支持版本下限；预留换 HTTP/KV 的接口边界（仍不回流 File 写盘）。  
4. 无鉴权 rest 写入口暴露 → 安全事故；Spec 必须钉网络与鉴权。  
5. 网络合并与多 Environment / 多 gateway 冲突 → Spec 钉唯一性与寻址（S8）。  
6. 表单范围膨胀 → S5 可切迭代。  
7. 删除 File 路径导致短期回归 → 用失效清单管理（S10）。  
8. 实现把路由 JSON 塞进 Version → 验收第 8 条卡住。

## Assumptions

1. E1/E3/E4/dashboard 不阻塞 E5 Accept。  
2. gateway 镜像/静态配置 **能**启用 `providers.rest` 且 Orbit 网络可达其管理面（具体交付形态 Spec）。  
3. 开发库 schema 策略 Plan 定。  
4. 本期 **不**保留 File 写路由并行通道。  
5. Traefik **≥ 3.6**（实测 3.6.10–3.7.8）作为 rest 前提；更低版本不承诺。

## User review notes

- 2026-07-22：开 E5 Requirement Draft；对照 bak 差距。  
- 2026-07-22：用户闭合方向——表单挂载/环境变量；RuntimeConfig；network=gateway×Environment；访问延后；不兼容拟合；全 kind 共用。  
- 2026-07-22：建模——禁止 Version 巨型 JSON。  
- 2026-07-22：本地实测 rest（3.6.10 / 3.6.23 / 3.7.8）：PUT 全量可用；`api` 写 405。  
- 2026-07-22：**用户定案**——  
  - **既然可用 rest 管理动态路由，就不再基于目录挂载管理路由**；  
  - **也不需要特殊处理 kind=gateway 的部署目录结构**；  
  - 平台路由主路径从 File/`dynamic_route_dir` **切换为 `providers.rest`**。  
- 调查附记：旧 `RouteManager` 写盘路径进入失效清单，实现期删除而非兼容。  
- 2026-07-22：用户 **Accept Requirement / 进入 Spec**；Traefik rest 知识写入 Obsidian `技术相关/middleware/traefix/Traefik v3 REST 动态路由管理.md`。  
- 2026-07-22：Spec **Accepted**——参考规格体例（不写项目背景）；自定义证书等进后续任务；**暂不支持多 gateway**（80 冲突 + labels 目标）；其余采纳。  
- 2026-07-22：用户决策纳入路线——承认网关特殊性、定义领域配置 → **E6** Draft（不并入本 E5 范围）。
