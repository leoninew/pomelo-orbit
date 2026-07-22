# CD Environment 接入策略（取代 per-component Binding）
最后修改时间: 2026-07-22 08:35:55

Review status: Draft

Flow mode: strict

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`（**修订 R2**）

**路线位置**（已挂入 cadence 与 R1 文档链，非孤立稿）：

```text
P0–P3 Version
  -> R1 Environment/Expose/Service（20260721-cd-environment-expose-service；已交付）
  -> R2 本文件 IngressPolicy（当前 Draft / strict）
  -> K8s / 其它
```

Related（保留不废止的底座）:

| 文档 | 关系 |
|------|------|
| `docs/requirement/20260721-cd-application-version-cadence.md` | 总索引；本需求为 **修订 R2** |
| `docs/requirement/20260721-cd-environment-expose-service.md` | **R1**；本需求 **部分修订** D4 EnvironmentBinding 用户 SoT |
| `docs/plan/20260721-cd-environment-expose-service.md` | R1 Plan 已实现；本需求修正 Binding CRUD / 域名手填路径 |
| `docs/verification/20260721-cd-environment-expose-service.md` | R1 有条件通过；产品反馈驱动本 R2 |
| `docs/requirement/20260721-cd-application-version-domain.md` 等 P0–P3 | 头注已挂 R2 指针；正文 Accepted 不改写 |

不推翻：

- Environment 归属 Project（`project_id`）；deploy 同项目校验
- Version.ExposeSpec：`protocol` + `container_port` + Version 内 `component_name`；**无域名**
- Service 唯一键 `(application_id, environment_id, instance_key)`
- Traefik MVP：每 `(application_id, environment_id)` 至多一个 ingress target
- deploy 必选 `version_id` + `environment_id`；Render 生成 compose / Traefik labels
- 无历史兼容层 / 无 raw compose 产品路径

## Terminology / 规范用词

| 中文 | 英文 | 说明 |
|------|------|------|
| 暴露规格 | **ExposeSpec** | Version 内：协议 + 容器端口 + 所属 component（Version 内部名）；**无域名** |
| 环境 | **Environment** | Project 下投放目标；承载**接入策略**，不承载「按 component 接线表」 |
| 环境接入策略 | **IngressPolicy**（工作名） | Environment 上：域名推导规则、默认 entrypoint、默认 TLS 等；**全局于该环境** |
| 环境接入绑定 | **EnvironmentBinding** | **本期拟废止或降级**为用户不可见的中间产物（若保留表则不得再作为 Environment 页 SoT） |
| 接入目标 | **ingress target** | 某 (Application, Environment) 上写入 Traefik labels 的 Service |

## Background / 背景

### 已落地能力

`20260721-cd-environment-expose-service` 已交付：

- Project 级 Environment、Version Expose、多 Service、deploy 合路写 Traefik labels
- 前端 Environment「绑定配置」：按 **component + protocol + port** 手填 **domains / entrypoint / tls**

### 产品反馈（2026-07-22）

合成 labels 的目标理解正确，但当前 **Binding 产品形态错误**：

1. **域名手填不当**  
   旧实现中，域名/接入参数来自**环境侧配置或约定**，部署时生成 labels，用户**不必**为每个暴露口填域名表单。  
   现实现把域名拆到 Binding 行，增加无效劳动，偏离「环境配置 → 部署合成」心智。

2. **在 Environment 上选 / 填 component 不当**  
   - `component.name` 属于 **某 Application 的某 Version 内部**命名空间  
   - 不同 App、不同 Version 可同名（例如都叫 `redis`）  
   - Environment 是 **Project 级**，在环境上维护「绑哪个 component」是**层级错误**  
   - 对齐键 `component+protocol+port` 只在 **Deploy 已选定 Version** 时有意义，不应固化为 Environment CRUD 的用户 SoT

### 问题本质

| 正确 | 错误（现状） |
|------|----------------|
| Deploy 时：`Version.Expose × Environment.策略 → labels` | 环境页预填 per-component Binding 表 |
| Environment 回答「本环境默认怎么接入」 | Environment 回答「某个叫 redis 的组件绑哪个域名」 |
| component 只在 Version 编辑 Expose 时出现 | Environment UI 要求选择 component |

## Goal / 目标

1. 将 Environment 的对外接入从 **per-component Binding 表** 收敛为 **环境级 IngressPolicy（接入策略）**。
2. 域名（HTTP）由策略/约定在 **Render / Deploy** 时推导，用户**默认不填** per-expose 域名表单。
3. Environment 管理 UI **不得**要求用户选择或填写 Version 的 component 名。
4. 保留 Deploy 生成 Traefik 可识别 labels 的能力；合路输入变为：  
   `Render(Version + Exposes, Environment + IngressPolicy, Application, Service/ingress 选项)`。
5. 明确废止或降级当前 `EnvironmentBinding` 作为用户 SoT 的产品路径（含 API 形态在 Spec 阶段定）。
6. 与既有 MVP 单 ingress 约束兼容。

## Non-goal / 非目标

1. 不做 K8s / 远程多机 agent。
2. 不做同域名多实例 LB / 蓝绿产品流（仍受 Traefik MVP 单 ingress 约束，除非另开需求）。
3. 不做跨 Project 共享 Environment。
4. 不恢复 `application_route` / raw compose 作为产品 SoT。
5. 本期不要求实现「每应用覆盖域名」的完整覆盖层（可列为 Open / 后续）；若做，也**不得**做成环境级 component 表。
6. 不在本 Requirement 阶段改产品代码（strict：先 Spec → Plan → Implementation）。

## User scenarios / 用户场景

### 场景 1：配置本机 local 环境接入策略

用户打开 Environment 页，编辑 `local`（或任意环境）：

- 配置 **接入策略**（如基础域名 / 域名模板、默认 entrypoint、默认 TLS）
- **不出现** component 选择，**不出现**「为每个端口填域名」的绑定表（默认路径）

### 场景 2：Version 声明暴露，直接部署

1. 用户在应用详情为 Version 配置 Component + Expose（如 `nginx` / http / 80）  
2. 用户 Deploy：选择 Version + Environment（如 local），可选 instance / attach_ingress  
3. 系统按 **该 Version 的 Expose 列表** × **该 Environment 的 IngressPolicy** 生成 labels  
4. 用户**无需**事先在 Environment 上为 `nginx` 建 Binding 行

### 场景 3：同名 component 多应用

项目内 App A、App B 的 Version 均有 component `redis`：

- 环境策略对两者统一适用（域名模板含 `app_code` 等变量区分）  
- Environment **不**维护「redis → 某域名」的全局行

### 场景 4：无对外暴露的内部部署

Version 无 Expose（或均不需要 ingress）：

- Deploy 可不写 Traefik 对外 labels（或仅内部网络）  
- 不要求用户配置域名

## Functional requirements / 功能需求

### FR1 — Environment 接入策略（SoT）

Environment 必须能表达至少：

| 能力 | 说明 |
|------|------|
| 域名推导 | HTTP 主机名由策略生成，而非用户 per-expose 手填（默认路径） |
| 默认 entrypoint | 写入 Traefik entrypoints 的默认值 |
| 默认 TLS 策略 | 如 none / letsencrypt 等（枚举 Spec 定） |
| （可选）是否启用自动 HTTP 接入 | 有 Expose 且 attach_ingress 时是否写 labels |

字段级命名、模板变量集合在 **Spec** 钉死；Requirement 只钉业务意图。

### FR2 — 域名推导原则

1. 默认路径：**不**要求用户在 Environment 绑定表中填写域名列表。  
2. 推导输入至少可包含：`application.code`、`environment.code`、策略中的 base/模板；是否包含 `component_name` / `protocol` / `port` 由 Spec 决定（MVP 可倾向「每 app+env 一个主域名」以贴合单 ingress）。  
3. 推导结果必须满足现有域名校验规则（Render 侧不得 silent 写非法 Host）。

### FR3 — component 边界

1. `component_name` **仅**允许出现在：  
   - Version 的 Component / Expose 规格  
   - Deploy 时读取的 **该次 Version** 快照  
2. Environment CRUD、IngressPolicy、环境列表 UI **禁止**以 component 为用户输入维度。  
3. 禁止用「环境级 component 下拉」对齐多 Version。

### FR4 — Render / Deploy 合路

```text
① 解析 Service（app + env + instance，ingress 规则不变）
② 加载 Version + Exposes + Components
③ 加载 Environment + IngressPolicy
④ 对需要对外接入的 Expose 生成 Traefik labels（策略 × expose × app 上下文）
⑤ compose up / apply
```

缺策略导致无法生成合法 HTTP Host 时：**校验失败**，不 silent skip（与旧「缺 binding 失败」同严格度；错误文案指向「补全环境接入策略」，而非「补 binding 行」）。

### FR5 — 废止 / 降级当前 Binding 产品路径

1. 前端 Environment「绑定配置」按 component 编辑 domains 的路径 **移除或替换**为「接入策略」编辑。  
2. 后端若仍有 `environment_binding` 表：  
   - 要么迁移为策略字段落在 `environment`（或独立 policy 列）  
   - 要么仅作 Render 内部缓存且 **不对用户暴露 CRUD**  
3. 无历史兼容：开发期可 migration 删除/重建 Binding 相关结构；**不**双轨。

### FR6 — Version Expose 编辑（已部分改进，本需求不回退）

- Expose 的 component 仍从 **本 Version 的 components** 选择（Reka SelectControl）——这是 Version 内部引用，**保留**。  
- 与 Environment 侧禁止选 component **不冲突**。

## Acceptance / 验收标准

1. Environment 编辑页可配置接入策略；**无** component 选择、**无**默认的 per-expose 域名行编辑（默认路径）。  
2. 用户完成「Version 含 Expose + Environment 含策略」后即可 Deploy 生成 Traefik labels，**无需**预先创建 Binding 行。  
3. 同项目两应用均含 component 名 `redis` 时，Deploy 各自成功且域名可区分（模板含 app 维度），Environment 上**无** redis 冲突配置。  
4. Version 无 Expose 时可 Deploy（不强制域名策略完整，或策略仅在有 HTTP expose 且需要 ingress 时校验——Spec 二选一钉死）。  
5. 旧 Binding API/UI 不再作为文档化用户路径；测试覆盖策略推导与失败文案。  
6. 自动化：`go test` 相关包 + `yarn lint:fix` + `typecheck` 通过。

## Open questions / 待决（需用户或 Spec 拍板）

| # | 问题 | 影响 | 建议默认（可改） |
|---|------|------|------------------|
| Q1 | HTTP 主域名模板具体形态？ | 实现与文档 | `{app_code}.{base_domain}`；MVP 每 app+env 一个主 Host |
| Q2 | 多 HTTP Expose（多 component）时 Host/path 如何分配？ | labels 规则 | MVP：仅 ingress Service；多 expose 共用主域名 + `path_prefix`（若有）；否则仅第一个 http expose 或失败策略在 Spec 定 |
| Q3 | TCP 暴露是否需要环境级额外策略字段（entrypoints 列表）？ | TCP labels | 保留 env 默认 entrypoint；TCP 无域名；SNI 是否模板化另定 |
| Q4 | `environment_binding` 表删除 vs 内化？ | migration | **倾向删除表**，策略列落 `environment` |
| Q5 | 应用级域名覆盖（例外）是否本期？ | 范围 | **本期不做**；记 Non-goal |
| Q6 | `local` seed 环境默认 `base_domain` / entrypoint 取值？ | 开箱体验 | 如 `localhost` 或 `local.test` + `web`；Spec 钉死 |
| Q7 | attach_ingress=false 时是否仍校验策略？ | 校验 | 不写 labels 则可不要求域名策略完整 |

## Decisions / 决策（讨论已收敛，待 Accept）

1. **目标不变**：Deploy 时生成 Traefik 认识的 labels。  
2. **域名**：默认由 **Environment 接入策略**推导，**不**要求 Binding 表单手填。  
3. **component**：不得作为 Environment 用户配置维度；仅 Version 内部 + Deploy 读 Version。  
4. **EnvironmentBinding 用户 SoT**：**废止**；改为 IngressPolicy。  
5. **无双轨 / 无历史兼容**。  
6. **严格模式**：Requirement → Spec → Plan → Implementation → Verification；本阶段只记需求。

## Risk / 风险

1. 与已 Accepted 的 `20260721` D4 冲突 → 必须以**本修订文档 Accepted** 为准，并在 cadence 索引回链。  
2. 已写入 DB 的 binding 行在开发库需清理；无生产迁移承诺。  
3. 模板推导错误导致 Host 冲突（同 env 多 app）→ 模板必须含 `app_code`（或等价）。  
4. 前端已交付的 Bindings 模态需删除/替换，避免半残 UI。

## Spec / Plan gate

本需求 **Accepted** 后进入 **Spec（严格模式）**，Spec 至少覆盖：

1. IngressPolicy 字段与校验  
2. 域名模板变量与 MVP 多 expose 规则  
3. migration：策略落库；Binding 表处理  
4. Render 伪代码与错误码/文案  
5. API / Proto 变更面  
6. 前端 Environment 页替换点  

**在 Spec + Plan Accepted 前，不改产品代码（除用户另行授权的无关热修）。**

## User review notes

- 2026-07-22：用户确认 labels 目标正确；指出（1）域名旧为环境配置勿手填 Binding；（2）Environment 选 component 因跨 Version 同名而不妥。  
- 2026-07-22：用户要求 **strict / 严格模式** 先记录需求，不进入实现。  
- 2026-07-22：用户要求检查并挂入 cadence / P0–P3 / R1 plan·requirement·verification 路线 → 已插入 **R2** 索引与双向指针。
