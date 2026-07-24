# CD E6：网关领域 — 实现计划
最后修改时间: 2026-07-22 22:40:00

Review status: Accepted

Flow mode: strict

Implementation notes (2026-07-22):

- **E6-A/B**：已落地（gateway_config、Host/rest 改读 Gateway、Env 去 domain 列、全局 api_url/domain_suffix 移除）。
- **E6-C**：已落地 — `CompileGatewayToVersion` 于 Create/UpdateGateway 保存即 compile；托管 component `traefik`；docker.sock + traefik.yml(sync) + acme.json(seed)；按 target 合并挂载。
- **E6-D**：测试/lint 通过；Verification 文档待用户触发。

## Requirement / Spec basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | **Accepted** |
| `docs/spec/20260722-cd-gateway-domain-config-e6.md` | **Accepted**（本 Plan 进入时一并确认） |
| cadence | E6 扩展 |
| E5 Spec/实现 | rest PUT、挂载、单 gateway；本 Plan 改 **api_url / 域名来源** |

**硬约束（不得偏离 Spec）**：

1. **Gateway ≔ Application(kind=gateway)**；工作负载仍 **Version** 部署。  
2. **改表**：`gateway_config` 与 application **1:1**；**禁止** application 表堆网关专用列。  
3. **导航**增加 Gateway 入口；应用列表默认 `kind=standard`。  
4. **`rest_api_url`、域名（含原 `base_domain` / 全局 `domain_suffix`）从 Gateway config 读**。  
5. **Environment.base_domain 移除**；全局 `traefik.api_url` / `traefik.domain_suffix` / settings 等价项 **移除**。  
6. 仅 managed；无 external；**不迁移**旧数据；无双轨读全局+Gateway。  
7. Version **完整挂载**保留；配置可 compile 进 Version。  
8. API DTO **proto 生成**；路由单数 `/api/cd/gateway`。  
9. **不修改已执行的迁移文件**；新增 migration（期望版本 **11**）。  
10. F1–F7 / File 写路由：不做。

## Plan decisions

| # | 决策 | 说明 |
|---|------|------|
| P1 | 表名 | `gateway_config`；`application_id` **PK** + FK → `application(id)` ON DELETE CASCADE |
| P2 | 创建顺序 | 先 `application(kind=gateway)`，再 insert `gateway_config`；同事务 |
| P3 | `rest_api_url` | 必填（创建 gateway 时）；平台 PUT/GET **唯一**来源 |
| P4 | `base_domain` | 在 **gateway_config**；Host 推导 `{app_code}.{base_domain}` 等从 Gateway 取 |
| P5 | Host 推导 | **固定** `{app_code}.{base_domain}`；`base_domain` 仅来自 Gateway config；**无** domain_template |
| P6 | Env 删除列 | `environment.base_domain`、`environment.domain_template` **均删除**；API/UI/proto 同步删 |
| P7 | 全局配置 | 删除 `TraefikConfig.APIURL`、`DomainSuffix` 及 yaml/settings；`cert_dir` 暂留 data 布局（F1），**不**作 api 来源 |
| P8 | 当前 Gateway | 解析顺序：① 存在 active（running/deploying）gateway Service → 其 application 的 config；② 否则 Project 内 **唯一** gateway Application 的 config；③ 多个且无 active → 校验错误要求用户先部署或只保留一个 |
| P9 | List 过滤 | `GET application?kind=` 服务端过滤；主列表默认 `standard`；网关页 `gateway` |
| P10 | 导航 | 侧栏「网关」→ `/cd/gateway`（列表+详情） |
| P11 | compile | 保存 Gateway config 的部署意图字段 → upsert unpublished Version 的主 Component（E6-C）；A/B 可先只落库 config |
| P12 | 单测/检查 | migrate=11；无 base_domain 列；RouteManager 无 cfg.APIURL；compose Host 用 gateway base_domain |

### Host 推导

```text
host = lower(trim(app_code) + "." + trim(gateway_config.base_domain))
```

HTTP Expose 时若无 Gateway / `base_domain` 空 → **明确错误**。  
用户裁定：**Env 不留 domain_template**。

## Implementation steps

### E6-A — 表 + API 骨架 + 导航 + 列表过滤

#### A1. Migration 000011（sqlite + mysql）

- `CREATE TABLE gateway_config`：  
  - `application_id` PK  
  - `rest_api_url` TEXT NOT NULL  
  - `base_domain` TEXT NOT NULL  
  - 部署意图列（可先：`image` TEXT NULL；其余 C 组可二期或 `spec_json` TEXT 仅本表）  
  - `created_at` / `updated_at`  
- `environment`：**DROP COLUMN `base_domain`**（mysql 用 PROCEDURE/多步按项目既有风格；sqlite 按既有 rebuild 模式若有）  
- 更新 `migration_test` 期望版本 **11**  
- **不**写 data migration 回填

#### A2. Model / Repository

- `model.GatewayConfig`  
- repo：Get/Upsert/Delete by application_id；Create gateway 事务  
- Application list：`kind` 可选过滤  
- Environment model/repo/SQL：**去掉** BaseDomain

#### A3. Proto + gen

- 新 `proto/orbit/v1/gateway.proto`（或 application 旁）：Gateway CRUD 消息  
- `environment.proto`：Create/Update/Resp **删除** `base_domain`  
- `task proto`（项目既定命令）

#### A4. Usecase / HTTP

- `CreateGateway` / `UpdateGatewayConfig` / `GetGateway` / `ListGateways`  
- 路由：`/api/cd/gateway` 单数风格注册  
- Environment usecase：删除 defaultBaseDomain 与 base_domain 校验路径

#### A5. Frontend

- 侧栏菜单 **网关**  
- `GatewayPage` 列表 + 创建（code/name + rest_api_url + base_domain）  
- `GatewayDetail`：展示 config；链到 Version/Deploy（可先复用 ApplicationDetail 路由或 embed）  
- `ApplicationPage`：list 带 `kind=standard`  
- `EnvironmentPage`：去掉 base_domain 表单项与列  
- i18n

**A 验收**：能创建 gateway + config；主列表无 gateway；Env 无 base_domain；migrate=11。

---

### E6-B — 读路径切换：rest URL + 域名；去全局配置

#### B1. 配置删除

- `internal/config`：`TraefikConfig` 去掉 `APIURL`、`DomainSuffix`（保留或收敛 `CertDir`）  
- `configs/config.yaml` 删除对应键  
- settings：删除 `traefik__domain_suffix` 等  
- 全库检索替换编译错误

#### B2. RouteManager / 路由同步

- `ApplySnapshot` / `ListRouters`：`api_url` 参数改为 **调用方传入** 或 port 上 `ResolveGatewayRESTURL(ctx, projectId?)`  
- cd usecase `route` / Sync：先 **resolve current Gateway config**，空 URL → validation error  
- 单测：httptest + 显式 URL；**无**默认读 cfg

#### B3. Compose / Ingress Host

- `compose_renderer`：`resolveHTTPHost` 的 `base_domain` 来自 **RenderInput.Gateway**（或 `GatewayConfig`），不来自 `env.BaseDomain`  
- Deploy/Preview 加载 Gateway config 注入 RenderInput  
- 无 gateway / 无 base_domain 且需要 HTTP Host → 失败  
- gateway dashboard labels 域名：用 **本 gateway** 的 base_domain（不再 env）  
- 单测全面改夹具

#### B4. 其它引用

- template 示例 / seed `000005` 中 `config.domain_suffix`：改为文档说明或 gateway 变量（种子数据按「不迁移」可删改开发种子，**不**改已执行 migration 文件内容若已落地——**仅改未依赖的测试与 guides**；若 000005 已执行不可改文件，则运行时不再走该模板路径即可）  
- guides：deployment / routing 文档更新

**B 验收**：进程配置无 api_url；Sync 打到 gateway_config.rest_api_url；standard Expose Host 用 gateway base_domain。

---

### E6-C — 部署意图 + compile → Version

#### C1. Gateway config 字段补齐

- image、ports、docker_provider、docker_network、mount_docker_sock、log_level、acme_enabled 等（与 Spec D2.1 C 组对齐；可用列或 `spec_json` **仅 gateway_config**）

#### C2. Compile

- `CompileGatewayToVersion(app, config) (*versionId, error)`  
  - 确保 unpublished Version  
  - upsert 主 component（名如 `traefik` 或可配置 component_name）  
  - mounts：docker.sock special；acme logical seed；static 配置 **content** 生成 rest/docker provider 片段  
- UpdateGatewayConfig 成功路径末尾 compile（或显式「应用规格」按钮 — 推荐 **保存即 compile**）

#### C3. UI

- 网关详情：领域表单 +「Version 高级」入口保留 E5 挂载编辑  
- Deploy 仍选 environment + version

**C 验收**：不手写 yml，保存 config 后 Version 可 Preview/Deploy 出 rest-capable 规格（最小字段集）。

---

### E6-D — 回归与清理

- go test cd/usecase、traefik、config、handler  
- yarn lint:fix + typecheck  
- 确认无 `env.BaseDomain`、无 `cfg.Traefik.APIURL` 产品路径  
- cadence / guides 交叉引用  
- Verification 文档（实现后用户触发）

## Files to change（预期）

| 区域 | 路径（示意） |
|------|----------------|
| SQL | `sql/migration/*/000011_cd_gateway_config.*`；environment 去 base_domain |
| Model/Repo | `internal/model`、`repository`、sqlx cd |
| Config | `internal/config`、`configs/config.yaml`、settings |
| Usecase | gateway 新文件；environment；compose_renderer；route；deployment 注入 gateway |
| Traefik | `route.go` URL 注入 |
| Proto/gen | gateway.proto；environment.proto |
| HTTP | routes + handler gateway |
| Web | router、layout 菜单、GatewayPage/Detail、ApplicationPage、EnvironmentPage、i18n |
| Tests | migration_test、compose、route、environment、config |
| Docs | guides；cadence；本 feature 文档状态 |

## Verification plan

| # | 项 | 方式 |
|---|----|------|
| V1 | migrate version 11；有 gateway_config；environment 无 base_domain | migration_test + schema |
| V2 | 创建 gateway 写 config；list kind 过滤 | API/集成测 |
| V3 | 无全局 api_url；PUT 使用 rest_api_url | 单测 + 配置加载测 |
| V4 | Host 推导用 gateway.base_domain | compose 单测 |
| V5 | Env API/UI 无 base_domain | 编译 + 前端 |
| V6 | 导航 Gateway 存在；应用默认 standard | 手工/前端 |
| V7 | compile 后 mounts 校验通过（C） | 单测 |
| V8 | go test / vet；yarn typecheck；lint:fix | CI 本地 |
| V9 | 单 active gateway 仍拦截 | 既有 + 夹具 |

## Assumptions

1. Project 维度：list gateway 按 `project_id`；当前 Gateway 解析在 Sync 时若跨 project 路由则按 route.project_id 找 gateway（与现 Route 归属一致）。  
2. **Env 不保留 `domain_template`**（用户 2026-07-22 裁定）；Host 固定 `{app_code}.{gateway.base_domain}`。  
3. 开发库可清空；无生产迁移。  
4. cert_dir 短期可留配置或改 data 常量，不阻塞 api_url 移除。

## Risks

| 风险 | 缓解 |
|------|------|
| 无 gateway 时 standard 无法推 Host | 错误文案引导先建网关 |
| 多 gateway 无 active | P8 ③ 失败清晰 |
| sqlite DROP COLUMN | 沿用项目既有 migration 手法 |
| compile 与手改 Version 冲突 | upsert 键 + 文档 |
| 000005 种子仍含 domain_suffix 模板 | 不改已执行文件；运行时不用该路径 |

## Rollback

- 开发分支回退；migration 11 仅开发期。  
- 不恢复全局 api_url 双轨。

## Out of scope

- external 网关  
- F1–F7  
- 旧 Env base_domain 数据迁移  
- application 表网关列  

## User review notes

- 用户：**Environment.base_domain 移除，由 Gateway 负责**；实现步骤 E6-A→D **同意**；**开始 Plan** → 本文件 **Accepted**，Spec 同步 Accepted。  
- 用户：**Env 不留 domain_template**；Host 固定模式；**开始实现**。  

---

**当前：严格模式 / strict，计划 / Plan — Review status: Accepted**

下一步：用户确认「开始实现」后从 **E6-A** 落地（或本消息已授权 Plan 后等待显式实现指令）。  
按 SpecFlow：Plan Accepted 后默认停在 Plan，**除非**用户说开始实现。
