# 文档整理：归档历史并沉淀当前产品 / 设计 / 技术 SoT
最后修改时间: 2026-07-24 11:00:18

Review status: Accepted

Flow mode: light

## Background

早期与后期产品/技术决策分散在 SpecFlow 过程文档（`docs/requirement|spec|plan|verification`）、`docs/guides`、`docs/designs` 及讨论/分析文中。部分结论已废止或被后续主线覆盖，但旧文仍为 `Accepted` 全文，Agent 与人类检索时易把历史决策当现行规范。

已确认方向：

1. **不合并** SpecFlow 四件套正文为「一篇超长过程文」；过程库是交付审计链。
2. **物理归档** 已提交的历史过程文档与过时活文档；归档目录含 README，明确 **无须采信 / 不得作为实现依据**。
3. **梳理并沉淀** 少量当前 **产品 / 设计 / 技术** 活文档作为开发默认 SoT。
4. 文码冲突时 **以代码为准** 回写活文档。
5. 新功能仍按 SpecFlow 写过程文档（写在主树 `docs/requirement|…`）；日常开发默认先读活文档。
6. **范围仅限 git HEAD 已提交的 docs**（`git ls-tree -r HEAD --name-only docs/`）。**未提交文档一律不在本任务处理之列**（不登记、不搬移、不消化、不校准）。
7. **`docs/INDEX.md` 独立存在**（与 `docs/README.md` 分工，不合并进 README）。

## Goal

1. 按主流分层建立活文档目录；**README + 独立 INDEX** 作为导航。
2. 将 **HEAD 已提交** 的过程库与需退场的活文档 **搬移** 到 `docs/archive/...`，并写清归档 README（无须采信）。
3. 从代码 +（归档前）最新主线过程文档 **抽取** 当前产品 / 架构 / 指南 SoT；冲突以代码为准。
4. HEAD 已提交的 `docs/analyze/` 材料 **本轮消化** 进活文档或决策账本，由新文档替代后归档原文。
5. 实现中维护 **已提交全量** 待处理清单；验收时无「HEAD 中有却未列出」的遗漏。
6. 文档-only；不改产品代码。

## Non-goal

1. 不改写归档过程文档的历史结论正文（不做伪历史）。
2. 禁止无备份硬删；搬移保留完整内容。
3. 不实现产品功能代码、API、迁移或前端功能变更。
4. **不处理任何未提交文档**（untracked / 仅工作区或仅 staged 未进 HEAD 的路径一律忽略）。
5. 不执行 git commit/push（除非用户另行授权）。
6. 不要求活文档一次达到「百科全书」完备：允许按域成文，但 **已提交清单必须逐项有处置结论**（归档 / 吸收进 SoT / 保留为现行 guide 并校准）。

## User scenarios

1. **开发者 / Agent** 读 `docs/README.md` + `docs/INDEX.md` → 活 SoT，不读 archive 做实现。  
2. **考古** 时进入 `docs/archive/`，README 已声明无须采信。  
3. **本任务** 仅对照 HEAD 已提交全量清单逐项关闭。

## Acceptance

1. **活文档结构**落地（见 Decisions D7），至少包含：  
   - 产品 SoT（领域/术语/能力边界，优先完整覆盖 CD；其他域可有骨架或明确「待补」）  
   - 技术/架构 SoT（与现行 Go 代码一致）  
   - 决策账本（有效 / 废止 / backlog）  
   - 现行 how-to：`docs/guides/` 中保留项已与代码对齐，或已归档  
2. **`docs/README.md`** 说明阅读顺序、活路径、archive 禁令、SpecFlow 新文档仍写主树过程目录。  
3. **`docs/INDEX.md` 独立文件**：主题 → 活文档 →（可选）归档索引链接；**不得**仅作为 README 内嵌章节代替。  
4. **`docs/archive/README.md`**（及子目录 README 若需要）明确：归档内容 **无须采信**、**不得作为实现/验收依据**；仅供历史追溯。  
5. **已提交文档全量清单**（下文，范围 = HEAD）每一项在实现结束时有终态：`archived` / `absorbed→活路径` / `kept-live-calibrated`。  
6. HEAD 已提交 `docs/analyze/application-management-usability.md` **消化**进本轮活文档后归档原文。  
7. 无产品代码 diff；verification 可对照清单关闭率；实现前后用 `git ls-tree -r HEAD --name-only docs/` 核对无漏项。

---

## 活文档目录：主流实践与本项目选型

### 业界常见做法（摘要）

| 实践 | 要点 | 代表 |
|------|------|------|
| **Diátaxis** | 四分：Tutorials / How-to / Explanation / Reference，避免「一篇文档既教又规范」 | 文档结构设计主流方法论 |
| **ADR** | `docs/adr/` 或 `docs/decisions/`：一条决策一篇，不可变历史 + 状态（Accepted/Superseded） | 工程决策可追溯 |
| **产品 vs 工程拆分** | `product`/`domain`（业务真相）与 `architecture`/`engineering`（系统如何支撑）分开 | 中大型 monorepo |
| **README 入口 + 薄索引** | 根或 `docs/README` 只做导航，正文进子目录 | GitHub 开源惯例 |
| **过程/RFC 与现行规范分离** | RFCs、设计评审、已交付变更记录进 archive 或 `history/`，现行规范单独维护 | 避免过程文当 SoT |

本仓库已有 SpecFlow 过程四目录 + `guides` + 少量 `designs`/`frontend`，与「过程库 / 活规范 / how-to」三分天然契合；**不宜**把一切塞进单一 `guides/`。

### 本项目采用（D7，已定）

```text
docs/
  README.md                 # 文档中心说明与阅读顺序
  INDEX.md                  # 独立主题索引（必建，不并入 README）
  product/                  # 产品 SoT（领域、术语、能力、生命周期）
  architecture/             # 技术/设计 SoT（栈、分层、数据、集成机制）
  decisions/                # 现行决策账本（有效/废止/backlog；非全量历史 ADR 复制）
  guides/                   # How-to（必须对齐 product/architecture + 代码）
  frontend/                 # 前端现行规范（校准后保留或局部归档）
  requirement|spec|plan|verification/   # 仅「进行中 / 新任务」过程文档
  archive/                  # 历史；README 声明无须采信
    README.md
    specflow/requirement|spec|plan|verification/   # 已提交过程库搬移落点
    guides/ ...             # 从主树退场的 guides 等
    designs/ ...
    analyze/ ...
    （既有 discussions/specs/... 保留）
```

说明：

- **product / architecture / decisions**：Explanation + 精简 Reference（现行真相）。  
- **guides / frontend**：How-to 与前端专项。  
- **SpecFlow 主树**：只留 active；HEAD 已提交历史 **整包搬入** `archive/specflow/`。  
- **README vs INDEX**：README 讲「怎么读 / 规则」；INDEX 讲「主题 → 文件」导航表。  
- 不强制上 Diátaxis 四个英文目录名，语义对齐即可。

---

## 已提交文档全量清单（实现时逐项关闭）

> **范围定义**：仅 `git ls-tree -r HEAD --name-only docs/`（当前分支最后一次提交中的路径）。  
> **明确排除**：一切未进入 HEAD 的文档（untracked、仅 staged、工作区草稿）——**不列表、不处置**。  
> 快照说明：清单与 2026-07-24 前后 HEAD 一致；**不含** `20260724-sql-migration-squash`（未进 HEAD）。  
> 实现开始时用 `git ls-tree -r HEAD --name-only docs/` 复核；HEAD 新增已提交项须追加。  
> **处置列** 为实现目标态；关闭时改「状态」列。

### 0. 入口

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/README.md` | 重写：阅读顺序、活路径、archive 禁令、指向 INDEX | 待处理 |
| `docs/INDEX.md` | **新建独立文件**：主题 → 活文档 → 归档索引 | 待处理 |

### 1. SpecFlow 过程库（已提交 → 物理归档）

**规则**：下列一律搬至  
`docs/archive/specflow/{requirement,spec,plan,verification}/`（保持文件名）。  
不改正文历史结论。归档后主树过程目录仅保留本任务自身等实现期文件（本任务 requirement 若尚未进 HEAD，本任务不强制搬它；进 HEAD 后的处置不在本轮未提交范围争论内——本轮只处理快照时已在 HEAD 的路径）。

#### 1.1 requirement（62，均在 HEAD）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/requirement/20260614-cloudflare-turnstile-captcha.md` | archive/specflow/requirement | 待处理 |
| `docs/requirement/20260614-startup-config-validation.md` | archive | 待处理 |
| `docs/requirement/20260615-backend-go-cd-application.md` | archive | 待处理 |
| `docs/requirement/20260615-backend-go-cd-deployment.md` | archive | 待处理 |
| `docs/requirement/20260615-backend-go-cd-route.md` | archive | 待处理 |
| `docs/requirement/20260615-ci-run-api-migration.md` | archive | 待处理 |
| `docs/requirement/20260616-backend-go-mainstream-template.md` | archive | 待处理 |
| `docs/requirement/20260618-backend-go-auth-sqlite-busy.md` | archive | 待处理 |
| `docs/requirement/20260618-backend-go-ci-docker-physical-mount.md` | archive | 待处理 |
| `docs/requirement/20260618-backend-go-ci-runtime-variables.md` | archive | 待处理 |
| `docs/requirement/20260618-backend-go-ci-snapshot-get-or-create.md` | archive | 待处理 |
| `docs/requirement/20260618-go-liquid-template.md` | archive | 待处理 |
| `docs/requirement/20260622-backend-go-cd-physical-path.md` | archive | 待处理 |
| `docs/requirement/20260622-backend-go-http-logging.md` | archive | 待处理 |
| `docs/requirement/20260622-backend-go-runtime-config.md` | archive | 待处理 |
| `docs/requirement/20260624-backend-go-config-loading.md` | archive | 待处理 |
| `docs/requirement/20260624-backend-go-docker-image.md` | archive | 待处理 |
| `docs/requirement/20260624-backend-go-env-config.md` | archive | 待处理 |
| `docs/requirement/20260624-backend-go-repository-variables.md` | archive | 待处理 |
| `docs/requirement/20260624-docker-image-roles.md` | archive | 待处理 |
| `docs/requirement/20260624-trixie-compose-apt.md` | archive | 待处理 |
| `docs/requirement/20260625-backend-go-logging.md` | archive | 待处理 |
| `docs/requirement/20260625-cd-docker-worker.md` | archive | 待处理 |
| `docs/requirement/20260625-incremental-log-domain-extraction.md` | archive | 待处理 |
| `docs/requirement/20260626-cd-application-contract-fixes.md` | archive | 待处理 |
| `docs/requirement/20260626-cd-application-create-methods.md` | archive | 待处理 |
| `docs/requirement/20260626-manage-scp.md` | archive | 待处理 |
| `docs/requirement/20260628-settings-plain-list.md` | archive | 待处理 |
| `docs/requirement/20260629-jwt-secret-validation.md` | archive | 待处理 |
| `docs/requirement/20260630-cd-deploy-force-recreate.md` | archive | 待处理 |
| `docs/requirement/20260630-cd-deployment-detail-container-logs.md` | archive | 待处理 |
| `docs/requirement/20260701-api-inventory.md` | archive | 待处理 |
| `docs/requirement/20260701-cloudflare-api-cors-baseurl.md` | archive | 待处理 |
| `docs/requirement/20260701-traefik-multi-domain-router-label.md` | archive | 待处理 |
| `docs/requirement/20260703-runtime-api-origin-refinement.md` | archive | 待处理 |
| `docs/requirement/20260705-api-proto-contracts.md` | archive | 待处理 |
| `docs/requirement/20260705-backend-go-golang-migrate.md` | archive | 待处理 |
| `docs/requirement/20260705-backend-go-sqlc.md` | archive | 待处理 |
| `docs/requirement/20260707-gin-http-transport-migration.md` | archive | 待处理 |
| `docs/requirement/20260707-http-assets-logging-filter.md` | archive | 待处理 |
| `docs/requirement/20260708-internal-layering-followup-split.md` | archive | 待处理 |
| `docs/requirement/20260708-internal-layering-record.md` | archive | 待处理 |
| `docs/requirement/20260709-internal-directory-structure-alignment.md` | archive | 待处理 |
| `docs/requirement/20260710-cd-deployment-status-polling.md` | archive | 待处理 |
| `docs/requirement/20260713-bootstrap-logger-lifecycle.md` | archive | 待处理 |
| `docs/requirement/20260713-web-1440-layout-adaptation.md` | archive | 待处理 |
| `docs/requirement/20260714-resource-route-registration.md` | archive | 待处理 |
| `docs/requirement/20260721-cd-application-version-cadence.md` | archive；精华吸收进 product + decisions | 待处理 |
| `docs/requirement/20260721-cd-application-version-deploy.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260721-cd-application-version-domain.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260721-cd-application-version-render.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260721-cd-application-version-spec.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260721-cd-deployment-failure-error-detail.md` | archive | 待处理 |
| `docs/requirement/20260721-cd-environment-expose-service.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260722-cd-application-kind-gateway.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260722-cd-gateway-component-mount-e5.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260723-cd-gateway-detail-deploy-stop.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260723-cd-gateway-entrypoint-ownership.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260723-cd-gateway-tcp-access.md` | archive；吸收 | 待处理 |
| `docs/requirement/20260723-cd-post-e6-cleanup.md` | archive；吸收 | 待处理 |

#### 1.2 spec（14）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/spec/20260614-cloudflare-turnstile-captcha.md` | archive/specflow/spec | 待处理 |
| `docs/spec/20260616-backend-go-mainstream-template.md` | archive | 待处理 |
| `docs/spec/20260618-go-liquid-template.md` | archive | 待处理 |
| `docs/spec/20260622-backend-go-http-logging.md` | archive | 待处理 |
| `docs/spec/20260624-backend-go-env-config.md` | archive | 待处理 |
| `docs/spec/20260630-cd-deployment-detail-container-logs.md` | archive | 待处理 |
| `docs/spec/20260701-api-inventory.md` | archive | 待处理 |
| `docs/spec/20260701-cloudflare-api-cors-baseurl.md` | archive | 待处理 |
| `docs/spec/20260705-api-proto-contracts.md` | archive；可吸收 API 约定进 architecture | 待处理 |
| `docs/spec/20260722-cd-application-kind-gateway.md` | archive；吸收 | 待处理 |
| `docs/spec/20260722-cd-environment-ingress-policy.md` | archive；吸收 | 待处理 |
| `docs/spec/20260722-cd-gateway-component-mount-e5.md` | archive；吸收 | 待处理 |
| `docs/spec/20260722-cd-gateway-domain-config-e6.md` | archive；吸收 | 待处理 |
| `docs/spec/20260723-cd-gateway-tcp-access.md` | archive；吸收 | 待处理 |

#### 1.3 plan（19）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/plan/20260614-cloudflare-turnstile-captcha.md` | archive/specflow/plan | 待处理 |
| `docs/plan/20260616-backend-go-mainstream-template.md` | archive | 待处理 |
| `docs/plan/20260618-go-liquid-template.md` | archive | 待处理 |
| `docs/plan/20260622-backend-go-cd-physical-path.md` | archive | 待处理 |
| `docs/plan/20260622-backend-go-http-logging.md` | archive | 待处理 |
| `docs/plan/20260624-backend-go-env-config.md` | archive | 待处理 |
| `docs/plan/20260625-incremental-log-domain-extraction.md` | archive | 待处理 |
| `docs/plan/20260626-cd-application-create-methods.md` | archive | 待处理 |
| `docs/plan/20260630-cd-deployment-detail-container-logs.md` | archive | 待处理 |
| `docs/plan/20260701-cloudflare-api-cors-baseurl.md` | archive | 待处理 |
| `docs/plan/20260705-api-proto-contracts.md` | archive | 待处理 |
| `docs/plan/20260705-backend-go-sqlc.md` | archive | 待处理 |
| `docs/plan/20260721-cd-application-version.md` | archive | 待处理 |
| `docs/plan/20260721-cd-environment-expose-service.md` | archive | 待处理 |
| `docs/plan/20260722-cd-application-kind-gateway.md` | archive | 待处理 |
| `docs/plan/20260722-cd-environment-ingress-policy.md` | archive | 待处理 |
| `docs/plan/20260722-cd-gateway-component-mount-e5.md` | archive | 待处理 |
| `docs/plan/20260722-cd-gateway-domain-config-e6.md` | archive | 待处理 |
| `docs/plan/20260723-cd-gateway-tcp-access.md` | archive | 待处理 |

#### 1.4 verification（50）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/verification/20260614-cloudflare-turnstile-captcha.md` | archive/specflow/verification | 待处理 |
| `docs/verification/20260614-startup-config-validation.md` | archive | 待处理 |
| `docs/verification/20260615-backend-go-cd-application.md` | archive | 待处理 |
| `docs/verification/20260615-backend-go-cd-deployment.md` | archive | 待处理 |
| `docs/verification/20260615-backend-go-cd-route.md` | archive | 待处理 |
| `docs/verification/20260615-ci-run-api-migration.md` | archive | 待处理 |
| `docs/verification/20260618-backend-go-auth-sqlite-busy.md` | archive | 待处理 |
| `docs/verification/20260618-backend-go-ci-docker-physical-mount.md` | archive | 待处理 |
| `docs/verification/20260618-backend-go-ci-runtime-variables.md` | archive | 待处理 |
| `docs/verification/20260618-go-liquid-template.md` | archive | 待处理 |
| `docs/verification/20260622-backend-go-cd-physical-path.md` | archive | 待处理 |
| `docs/verification/20260622-backend-go-http-logging.md` | archive | 待处理 |
| `docs/verification/20260622-backend-go-runtime-config.md` | archive | 待处理 |
| `docs/verification/20260624-backend-go-config-loading.md` | archive | 待处理 |
| `docs/verification/20260624-backend-go-docker-image.md` | archive | 待处理 |
| `docs/verification/20260624-backend-go-env-config.md` | archive | 待处理 |
| `docs/verification/20260624-backend-go-repository-variables.md` | archive | 待处理 |
| `docs/verification/20260624-docker-image-roles.md` | archive | 待处理 |
| `docs/verification/20260624-trixie-compose-apt.md` | archive | 待处理 |
| `docs/verification/20260625-cd-docker-worker.md` | archive | 待处理 |
| `docs/verification/20260625-incremental-log-domain-extraction.md` | archive | 待处理 |
| `docs/verification/20260626-cd-application-contract-fixes.md` | archive | 待处理 |
| `docs/verification/20260626-cd-application-create-methods.md` | archive | 待处理 |
| `docs/verification/20260628-settings-plain-list.md` | archive | 待处理 |
| `docs/verification/20260629-jwt-secret-validation.md` | archive | 待处理 |
| `docs/verification/20260630-cd-deploy-force-recreate.md` | archive | 待处理 |
| `docs/verification/20260630-cd-deployment-detail-container-logs.md` | archive | 待处理 |
| `docs/verification/20260701-api-inventory.md` | archive | 待处理 |
| `docs/verification/20260701-cloudflare-api-cors-baseurl.md` | archive | 待处理 |
| `docs/verification/20260701-traefik-multi-domain-router-label.md` | archive | 待处理 |
| `docs/verification/20260703-runtime-api-origin-refinement.md` | archive | 待处理 |
| `docs/verification/20260705-api-proto-contracts.md` | archive | 待处理 |
| `docs/verification/20260705-backend-go-golang-migrate.md` | archive | 待处理 |
| `docs/verification/20260705-backend-go-sqlc.md` | archive | 待处理 |
| `docs/verification/20260707-gin-http-transport-migration.md` | archive | 待处理 |
| `docs/verification/20260708-internal-layering-followup-split.md` | archive | 待处理 |
| `docs/verification/20260708-internal-layering-record.md` | archive | 待处理 |
| `docs/verification/20260709-internal-directory-structure-alignment.md` | archive | 待处理 |
| `docs/verification/20260710-cd-deployment-status-polling.md` | archive | 待处理 |
| `docs/verification/20260713-bootstrap-logger-lifecycle.md` | archive | 待处理 |
| `docs/verification/20260713-web-1440-layout-adaptation.md` | archive | 待处理 |
| `docs/verification/20260714-resource-route-registration.md` | archive | 待处理 |
| `docs/verification/20260721-cd-deployment-failure-error-detail.md` | archive | 待处理 |
| `docs/verification/20260721-cd-environment-expose-service.md` | archive | 待处理 |
| `docs/verification/20260722-cd-application-kind-gateway.md` | archive | 待处理 |
| `docs/verification/20260722-cd-consumer-platform-network-e1.md` | archive | 待处理 |
| `docs/verification/20260722-cd-environment-ingress-policy.md` | archive | 待处理 |
| `docs/verification/20260722-cd-gateway-component-mount-e5.md` | archive | 待处理 |
| `docs/verification/20260722-cd-gateway-domain-config-e6.md` | archive | 待处理 |
| `docs/verification/20260723-cd-gateway-detail-deploy-stop.md` | archive | 待处理 |

> 若实现时发现 verification 主树另有已提交未列文件，追加本表。当前 `git ls-files` 为 50 条如上。

### 2. guides（14）— 校准保留或归档

| 路径 | 处置（默认） | 状态 |
|------|----------------|------|
| `docs/guides/application-state-machine.md` | **归档**（与现行 Service/Deployment 模型冲突）；要点写入 product | 待处理 |
| `docs/guides/deployment.md` | 对照代码校准后 **kept-live** 或拆入 architecture | 待处理 |
| `docs/guides/docker-deployment.md` | 校准 / 归档 | 待处理 |
| `docs/guides/docker-label-routing.md` | 校准 / 归档 | 待处理 |
| `docs/guides/routing-and-certificates.md` | 校准（Gateway/Expose）后 kept-live | 待处理 |
| `docs/guides/certificate-management.md` | 校准 / 归档 | 待处理 |
| `docs/guides/server-deployment.md` | 校准 / 归档 | 待处理 |
| `docs/guides/volume-mounting.md` | 校准 / 归档 | 待处理 |
| `docs/guides/ci-pipeline-design.md` | 校准后 kept-live 或 architecture/ci | 待处理 |
| `docs/guides/ci-pipeline-vars-design.md` | 同上 | 待处理 |
| `docs/guides/permissions.md` | 校准后 kept-live | 待处理 |
| `docs/guides/google-oauth-china-network.md` | 校准后 kept-live（运维向） | 待处理 |
| `docs/guides/traefik-file-watch-windows-docker.md` | 校准后 kept-live（环境笔记） | 待处理 |
| `docs/guides/antdv-icons.md` | 校准后 kept-live 或并入 frontend | 待处理 |

### 3. designs（5）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/designs/architecture.md` | **归档**（Python/FastAPI 过时）；由 `architecture/` 新文替代 | 待处理 |
| `docs/designs/dashboard_redesign.html` | 归档 designs | 待处理 |
| `docs/designs/a09727de-27c9-4451-993f-b1f20c114ed6.png` | 归档 designs | 待处理 |
| `docs/designs/ChatGPT Image May 10, 2026, 09_07_35 PM.png` | 归档 designs | 待处理 |
| `docs/designs/ChatGPT Image May 10, 2026, 12_23_42 PM.png` | 归档 designs | 待处理 |

### 4. frontend（4）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/frontend/architecture.md` | 对照 `web/` 校准后 kept-live 或归档 | 待处理 |
| `docs/frontend/reka-ui-guide.md` | 校准后 kept-live | 待处理 |
| `docs/frontend/style-guide.md` | 校准后 kept-live | 待处理 |
| `docs/frontend/reka-llms.txt` | 工具材料：kept-live 或归档（实现时定） | 待处理 |

### 5. analyze（已提交 1）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/analyze/application-management-usability.md` | **本轮消化**进 product/decisions（可用性结论）；原文归档 `archive/analyze/` | 待处理 |

### 6. archive 既有（32）

| 范围 | 处置 | 状态 |
|------|------|------|
| `docs/archive/**` 现有 discussions/specs/designs 等 | **保留**；更新 `docs/archive/README.md` 统一声明无须采信 | 待处理 |
| 新搬入的 specflow/guides/designs/analyze | 写入子路径；根 README 索引 | 待处理 |

### 7. assets（4）

| 路径 | 处置 | 状态 |
|------|------|------|
| `docs/assets/*.png` | 若仅被归档文引用则随引用关系保留或迁 archive/assets；实现时按引用检查 | 待处理 |

### 8. 未提交文档（明确不在范围）

**不处理、不列表、不搬移、不消化。** 包括但不限于工作区中可能存在的：

- `docs/requirement/20260724-docs-current-sot-consolidation.md`（本文件，在 commit 进 HEAD 前）
- `docs/requirement/20260724-sql-migration-squash.md` 及任何仅 staged / untracked 过程文档
- `docs/requirement|plan/20260724-ci-cd-dto-file-split.md`、`docs/requirement/20260724-remove-sqlx-full-sqlc.md`
- `docs/analyze/20260724-*-domain-split-*.md` 等未进 HEAD 的分析稿

实现时 **禁止** 因「顺手」修改或归档上述路径。

---

## 待处理事项（实现中维护）

### A. 结构与入口

| ID | 事项 | 状态 |
|----|------|------|
| A1 | 创建 `docs/product/`、`docs/architecture/`、`docs/decisions/` | **完成** |
| A2 | 重写 `docs/README.md` | **完成** |
| A3 | **新建独立** `docs/INDEX.md`（主题索引） | **完成** |
| A4 | 建立 `docs/archive/specflow/{requirement,spec,plan,verification}/` | **完成** |
| A5 | 重写/强化 `docs/archive/README.md`：**无须采信、不得作实现依据** | **完成** |

### B. 活 SoT（以代码为准）

| ID | 事项 | 状态 |
|----|------|------|
| B1 | product：CD 领域模型与术语（自代码 + 归档前 cadence/E5/E6/TCP 吸收） | **完成** → `product/cd-model.md` |
| B2 | architecture：Go 分层、CD 渲染/网关、数据与迁移指针 | **完成** → `architecture/backend.md` + `cd-runtime.md` |
| B3 | decisions：有效 / 废止 / backlog 账本 | **完成** → `decisions/ledger.md` |
| B4 | 消化 `application-management-usability` 进 product/decisions | **完成**（product 可用性节 + 归档原文） |
| B5 | guides 逐项校准或归档（第二节清单） | **完成**（状态机归档；deployment 重写；其余 living banner） |
| B6 | frontend 逐项校准或归档 | **完成**（`web/` 路径校正；reka 链接修复） |

### C. 搬移（仅 HEAD 已提交）

| ID | 事项 | 状态 |
|----|------|------|
| C1 | 搬移 HEAD 中 requirement/spec/plan/verification 全量 | **完成**（62+14+19+50） |
| C2 | 搬移 designs 过时项与退场 guides/analyze | **完成** |
| C3 | 修复 README/INDEX/活文档内断链；archive 内不要求修全历史互链 | **完成**（入口级） |
| C4 | `git ls-tree -r HEAD --name-only docs/` 与清单对账 | **完成**（实现时按 HEAD 清单搬移） |

### D. 提交边界

| ID | 事项 | 状态 |
|----|------|------|
| D1 | 建议本任务文档变更与并行代码任务拆分 commit（屏幕说明） | **完成**（见实现汇报） |

### E. 项目约定（可选本批）

| ID | 事项 | 状态 |
|----|------|------|
| E1 | `CLAUDE.md` 增加文档阅读顺序与 archive 禁令 | **完成** |

---

## Open questions

**已全部闭合（2026-07-24）**：

| # | 问题 | 结论 |
|----|------|------|
| 1 | 活文档目录 | product / architecture / decisions + guides（D7） |
| 2 | 归档方式 | 物理搬移；archive README 无须采信 |
| 3 | 范围 | HEAD 已提交 docs 全量列出并处理 |
| 4 | 文码冲突 | 以代码为准 |
| 5 | analyze | HEAD 已提交 analyze 本轮消化后归档 |
| 6 | INDEX | **独立** `docs/INDEX.md`，不并入 README |
| 7 | 未提交文档 | **不在处理之列** |

暂无需要用户再确认的未决事项。

## Decisions

| # | 决策 | 说明 |
|---|------|------|
| D1 | 过程四件套不合并正文 | 审计链整包搬 archive |
| D2 | 活文档为开发默认 SoT | product/architecture/decisions/guides |
| D3 | **物理归档**已提交历史过程库 | `docs/archive/specflow/...` |
| D4 | archive README：**无须采信** | 不得作实现/验收依据 |
| D5 | light 模式 | 无独立 spec/plan 文件 |
| D6 | 文档-only | 不改产品代码 |
| D7 | 目录选型 | `product` + `architecture` + `decisions` + 保留校准后的 `guides`/`frontend` |
| D8 | 文码冲突 | **以代码为准** 写活文档 |
| D9 | 已提交全量清单 | 上表逐项关闭；范围 = HEAD |
| D10 | 已提交 analyze | 本轮消化后归档 |
| D11 | **INDEX 独立** | 必建 `docs/INDEX.md`，与 README 分工 |
| D12 | **未提交文档不在范围** | 不列表、不处置、不顺手改 |

## Risk

1. 搬移约 140+ 过程文件导致外链断裂 → README/INDEX 切断对 archive 的实现依赖；archive 内不要求修全历史互链。  
2. 抽取活文档时误抄过时结论 → **强制对照代码**。  
3. 工作区未提交文档与本任务并存 → **禁止触碰未提交路径**，避免误归档并行任务。  
4. 清单漏项 → 实现前后 `git ls-tree -r HEAD --name-only docs/` 对账。  
5. guides「校准」工作量大 → 明显冲突者优先归档，运维笔记可薄校准。

## User review notes

- 2026-07-24：用户确认「归档历史 + 梳理当前产品/设计/技术文档」思路。  
- 2026-07-24：用户答复：主流目录选型、物理搬移 + 无须采信、已提交全量处理、以代码为准、已提交 analyze 消化。  
- 2026-07-24：用户补充：**INDEX.md 独立**；**未提交文档都不在处理之列**（已写入 D11/D12，并移除未提交处置清单）。  
- 2026-07-24：用户「开始实现」→ Requirement Accepted；Implementation 执行归档 + 活 SoT 撰写。

## Implementation notes（2026-07-24）

实现摘要（停在 Implementation，待用户验收后再 Verification）：

| 类别 | 结果 |
|------|------|
| 活 SoT | `product/overview.md`、`product/cd-model.md`、`architecture/backend.md`、`architecture/cd-runtime.md`、`decisions/ledger.md` |
| 入口 | `docs/README.md`、独立 `docs/INDEX.md`、`archive/README.md`、`archive/specflow/README.md` |
| 过程库搬移 | HEAD requirement 62 / spec 14 / plan 19 / verification 50 → `archive/specflow/*` |
| 退场 | designs 全量、`guides/application-state-machine`、`analyze/application-management-usability` |
| 校准 | guides living banner；`deployment`/`routing` 对齐 CD；frontend `web/` |
| 约定 | `CLAUDE.md` 文档阅读顺序 |
| 未触碰 | 一切未进 HEAD 的 docs（含本文件同目录其他 untracked / staged 任务文） |

已提交清单各行处置原则：过程库 = `archived`（CD 主线精华已 absorbed 进 product/architecture/decisions）；guides = kept-live-calibrated 或 archived；designs/analyze = archived+absorbed。  

