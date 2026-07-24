# CD Environment / Expose / Service 实现计划
最后修改时间: 2026-07-22 08:35:55

Review status: Accepted

Flow mode: standard

## Requirement basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260721-cd-environment-expose-service.md` | **Accepted**（R1） |
| `docs/requirement/20260721-cd-application-version-cadence.md`（修订索引） | Accepted |
| `docs/requirement/20260721-cd-application-version-domain.md`（修订指针） | Accepted |
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | **Draft**（R2 / strict）；**不**在本 Plan 实现范围内 |

**路线位置**：cadence **R1** 的 Plan。R1 已实现并 Verification Accepted。  
**后继修订 R2** 将废止本 Plan 中 **EnvironmentBinding 用户 SoT**（P3/P4 域名手填、环境侧 component 对齐键作 CRUD），改为 Environment **IngressPolicy**；R2 Accepted 前本 Plan 仍描述**已交付**行为。

**需求硬约束（不得偏离）**：

1. Environment **归属 Project**：必有 `project_id` FK；**无** `application_id`；业务唯一 `(project_id, code)`。
2. **禁止跨 Project**：deploy 时 `application.project_id` **必须等于** `environment.project_id`；列表/创建均在当前项目上下文（与 Application 相同产品习惯）。
3. Version 承载 **ExposeSpec**（`http` \| `tcp` + `container_port` + `component_name`）；**无域名**。
4. Environment 承载域名/入口/证书/Binding；HTTP|TCP **同批**进模型与 Render 协议分支。
5. Service 唯一键 `(application_id, environment_id, instance_key)`；一 App 多 Service；且 app 与 env 必须同 project。
6. Traefik MVP：每 `(application_id, environment_id)` **至多一个** ingress target。
7. deploy 必选 `version_id` + `environment_id`；Render = Version × Environment。
8. **放弃历史兼容**；不修改已执行 migration；仅新增 `000009_*`。
9. 废止 `application_route` 长期 SoT；无 raw compose。

## Plan decisions（本 Plan 拍板）

| # | 决策 | 说明 |
|---|------|------|
| P1 | 表名 | `environment`、`expose`、`environment_binding`；物理名不用 `cd_` 前缀 |
| P2 | Expose 存储 | **独立表** `expose`，FK `version_id`；不嵌 Component JSON |
| P3 | Binding 对齐键 | 列：`component_name` + `protocol` + `container_port`（与 Expose 三元组等值匹配）；**不** FK expose.id（fork/替换 id 会变） |
| P4 | 对外 SoT | **Expose 为准**；Component.`ports_json` 仅保留为容器侧 publish 意图，**不**驱动 Traefik；Render 对对外端口以 Expose.container_port 为准写 labels |
| P5 | 默认环境 | **按项目** seed：migration 为**每个已有 Project** 插入 `code=local`、`name=Local`；**新建 Project** 时 usecase 同步创建本项目 `local` Environment |
| P6 | 工作目录 | `data/cd/{app_code}/{env_code}/{instance_key}/`（替换现 `data/cd/{app_code}/`） |
| P7 | compose project 名 | `{app_code}-{env_code}-{instance_key}`，避免多实例撞名 |
| P8 | ingress | `service.is_ingress` bool；同一 `(app_id, env_id)` 至多一条 `is_ingress=1`；deploy 选项 `attach_ingress`（默认：`instance_key=default` 为 true，否则 false） |
| P9 | Router 名 | `{app_code}-{env_code}-{instance_key}-{component}-{protocol}`（sanitize 为 traefik 安全字符） |
| P10 | 权限 | 新增 `environment:read`、`environment:write`；挂 admin（及与 `application:*` 对齐的角色）；**写/列必须带 `project_id` 且校验项目成员**；deploy 校验：项目成员 + app/env **同 project** + environment 可读 |
| P11 | 删除列/表 | `DROP application_route`；`ALTER application DROP route_managed`（sqlite/mysql 各自写法） |
| P12 | Deployment | 增 `environment_id`；options_json 可含 `instance_key`、`attach_ingress`、`force_recreate` 等 |
| P13 | API 风格 | 单数路径：`/api/cd/environment`（query **必填** `project_id` 列表/创建）；`/api/cd/version/:id/expose` 或 embed 在 version body；deploy body 必填 `environment_id` |
| P14 | 零对外 | Version **允许 0 条 expose**；有 expose 且 `attach_ingress=true` 时，每条 expose **必须**命中 EnvironmentBinding，否则校验失败 |
| P15 | TCP labels | Traefik TCP router/service labels 与 HTTP 分支并列实现；TCP binding：`entrypoint` 必填，`domains` 可空；可选 `sni_host`（可空） |
| P16 | HTTP binding | `domains_json` 至少 1 个域名；`entrypoint` 默认 `websecure`（可覆盖）；`tls_mode`：`none` \| `letsencrypt`（MVP；manual 证书可后开，字段预留） |
| P17 | 同项目部署 | `DeployApplication` 硬校验 `app.project_id == env.project_id`；跨项目返回业务错误（非 500） |
| P18 | 删除 Project | 级联或先禁删：Environment 有 Service 引用时不可删 env；删 Project 走既有级联策略（与 Application 一致，Plan 实现时对齐 project delete 行为） |

## Overview

```text
deploy(app, version_id, environment_id, instance_key?, attach_ingress?, options)
  ① assert app.project_id == env.project_id
  ② Service upsert (app, env, instance_key) → version_id, is_ingress?, status=deploying
  ③ Render(Version+Component+Expose, Environment+Binding) → compose
  ④ docker compose -p {project} -f ... up ...
  ⑤ Container 派生；Service running|faulted
```

| 实体 | 表 | 职责 |
|------|-----|------|
| Environment | `environment` | **Project 下**投放目标元数据 |
| EnvironmentBinding | `environment_binding` | 环境侧接入参数 |
| Expose | `expose` | Version 暴露规格 |
| Service | `service` 扩列 | 多实例运行绑定 + ingress 标记 |
| Deployment | `deployment` 扩列 | 记录 environment_id |
| ApplicationRoute | **DROP** | 废止 |

## Data model / migration `000009_cd_environment_expose`

### `environment`

| 列 | 说明 |
|----|------|
| id | PK |
| project_id | NOT NULL，FK → project（ON DELETE 对齐项目删除惯例） |
| code | `^[a-z][a-z0-9-]*$` |
| name | 显示名 |
| description | 可空 |
| created_at / updated_at | |
| UNIQUE | `(project_id, code)` |

**无** `application_id`。

### `expose`

| 列 | 说明 |
|----|------|
| id | PK |
| version_id | FK → version ON DELETE CASCADE |
| component_name | 必须存在于同 Version 的 component.name |
| protocol | `http` \| `tcp` |
| container_port | 1–65535 |
| path_prefix | 可空；仅 http 有意义，默认 `/` 语义在渲染时处理 |
| created_at / updated_at | |
| UNIQUE | `(version_id, component_name, protocol, container_port)` |

### `environment_binding`

| 列 | 说明 |
|----|------|
| id | PK |
| environment_id | FK → environment ON DELETE CASCADE |
| component_name | 匹配键 |
| protocol | `http` \| `tcp` |
| container_port | 匹配键 |
| domains_json | JSON 字符串数组；http 必填非空；tcp 可 `[]` |
| entrypoint | 必填（http 默认可在 API 层填 `websecure`；tcp 调用方必填） |
| tls_mode | `none` \| `letsencrypt`（及预留 `manual`） |
| sni_host | 可空；tcp 可选 |
| note | 可空 |
| created_at / updated_at | |
| UNIQUE | `(environment_id, component_name, protocol, container_port)` |

### `service` 变更

| 变更 | 说明 |
|------|------|
| DROP UNIQUE(application_id) | 允许多行 |
| ADD environment_id | NOT NULL，FK → environment |
| ADD instance_key | NOT NULL DEFAULT `'default'` |
| ADD is_ingress | NOT NULL DEFAULT 0/false |
| UNIQUE(application_id, environment_id, instance_key) | 业务唯一 |
| INDEX (environment_id) | |
| INDEX (application_id, environment_id) WHERE is_ingress — 若方言支持部分索引；否则应用层保证 | |

**既有 service 行迁移**（无兼容业务、但避免 migration 失败）：

1. 为**每个** `project` 插入 `environment(project_id, code='local', name='Local')`。
2. `UPDATE service SET environment_id = (SELECT e.id FROM environment e INNER JOIN application a ON a.project_id = e.project_id AND a.id = service.application_id WHERE e.code = 'local'), instance_key='default', is_ingress=1`。
3. 再加 NOT NULL / UNIQUE（sqlite 可能需 rebuild table；mysql 可分步 ALTER）。

### `deployment` 变更

| 列 | 说明 |
|----|------|
| environment_id | 可空 FK（历史行可空；新 deploy 必写） |

### DROP / DROP COLUMN

- `DROP TABLE application_route`
- 删除 `application.route_managed`（sqlite 3.35+ DROP COLUMN；mysql DROP COLUMN）

### 权限 seed

- `environment:read` / `environment:write`
- 绑定到与 `application:write` 相同角色集合（读 `000004`/`000006` 惯例复制）

### Down migration

对称 DROP 新表/列；**不**重建 `application_route`（无历史兼容）。

## API / Proto

### Environment

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/cd/environment` | 列表；**query 必填 `project_id`**；分页+search；项目成员 + environment:read |
| POST | `/api/cd/environment` | 创建；body/query 含 `project_id`；write + 项目成员 |
| GET | `/api/cd/environment/:env_id` | 详情含 bindings；校验成员属于该 env 的 project |
| PUT | `/api/cd/environment/:env_id` | 更新元数据（code 是否可改：MVP **code 创建后不可改**，仅 name/description） |
| DELETE | `/api/cd/environment/:env_id` | 无 Service 引用时可删 |
| PUT | `/api/cd/environment/:env_id/binding` | 全量替换 bindings（或 POST/PUT/DELETE 单项；**推荐全量替换**简化 MVP） |

响应字段含 `project_id`。

### Version / Expose

- `VersionCreateReq` / `VersionUpdateReq` / `VersionResp` **嵌入** `exposes: ExposeReq[]` / `ExposeResp[]`。
- 创建/更新 Version 时与 components **同事务**写入 expose（published 不可改）。
- Fork：复制 exposes。
- 校验：component_name ∈ components；protocol 枚举；port 范围；http path_prefix 可选。

### Deploy

```protobuf
message ApplicationDeployReq {
  string version_id = 1;
  string environment_id = 2;
  string instance_key = 3;      // 空则 default
  bool attach_ingress = 4;      // 见 P8 默认规则：服务端若未显式传，按 instance_key 默认
  bool force_recreate = 5;
}
```

- 服务端：**load app + env → assert same project_id**。
- 列表/详情 `ApplicationResp`：`service_status` 不足以表达多 Service → 增：
  - `services_summary` 可选列表，或
  - 详情页单独 `GET /api/cd/application/:id/service` 改为 **list** `GET .../service` 返回多条；`GET .../service/:service_id` 单条。
- **Breaking（无兼容）**：旧 `ServiceByApplication` 单条语义删除；调用方改 list / 按 env+instance 查。

### Project 创建联动

- `CreateProject`（或等价入口）成功后：在同一事务或紧随其后创建默认 Environment `local`（失败则回滚/报错，不留下无 env 的项目）。

### 删除旧 API

- 全部 `application_route` CRUD 路由与 handler 删除。
- Application create/update **去掉** `route_managed`。
- Import/Export：export 带 versions.exposes；**不再** export application routes；import 创建 exposes + 不写 route 表。Environment/bindings **不**进 application 包（环境属 Project；导入应用不创建/不覆盖 env）。

## Render

替换 `RenderCompose(app, version, components)` 签名为：

```text
RenderCompose(ctx, app, version, components, exposes, env, bindings, service) (composeYAML, error)
```

或打包 `RenderInput` 结构体。

逻辑：

1. 由 components 生成 `services`（同现网）。
2. 若 `service.is_ingress`：
   - 对每个 expose，查找 binding（三元组匹配）。
   - 找不到 → error（P14）。
   - `protocol=http` → `traefik.http.routers.*` / `services.*` labels（多 domain Host 合并规则对齐现 `injectDeploymentRouteLabels`）。
   - `protocol=tcp` → `traefik.tcp.routers.*` / `services.*` labels；rule 可用 `HostSNI(\`*\`)` 或 `HostSNI(\`{sni}\`)`。
3. Router/service 名用 P9。
4. 非 ingress Service：不写 traefik labels；仍可按 Component.ports_json 映射宿主端口（若有）。
5. Preview API：`POST /api/cd/version/:id/preview` body 增加 **必填** `environment_id` + 可选 `instance_key`；服务端同样校验 env 与 app 同 project。

## Usecase 变更要点

| 区域 | 变更 |
|------|------|
| `DeployApplication` | 入参 env + instance；**同 project 校验**；upsert service by 三元组；ingress 互斥 |
| `Stop/Restart` | 需定位 Service：默认 body/query 带 `environment_id`+`instance_key`，或 `service_id` |
| `DeleteApplication` | 删其下全部 Service；目录按 app 下各 env/instance 清理 |
| `ServiceByApplication` | → `ListServicesByApplication` / `ServiceByKey(app, env, instance)` |
| Version CRUD | 读写 expose；publish 校验 expose.component 存在 |
| Environment CRUD | 新 usecase `environment.go`；**一律 project 作用域** |
| Project create | 种子 `local` Environment |
| Workspace | `AppDir` → `ServiceDir(appCode, envCode, instanceKey)`；更新 port 接口与全部调用 |
| 冲突检测 | 同 Environment 内：http 同 domain（+path）绑定冲突；tcp 同 entrypoint+sni 冲突 — create/update binding 时校验（自然落在同一 Project） |

## Frontend

| 页面 | 变更 |
|------|------|
| 新 **Environment** 列表/详情 | **当前 Project** 下 CRUD env + bindings；API 始终带 `project_id` |
| Application 详情 | Version 表单增加 Expose 行（协议选择 http/tcp、端口、组件下拉） |
| 部署 | 必选 **本项目** Environment 列表 + Version；instance_key 输入（默认 default）；attach_ingress 开关；不展示他项目 env |
| 去掉 | Application 详情「路由配置」块、route_managed 开关 |
| 列表 | service 状态：展示多 service 摘要或「N 个实例」+ 主状态 |
| 导航 | CD 下增加 Environment 菜单项（随项目切换刷新列表） |
| i18n | zh/en 全量键 |

权限：菜单与 API 跟 `environment:read/write` + 项目上下文。

## Implementation steps

### Step 1 — Migration + models + repository

1. `sql/migration/sqlite|mysql/000009_cd_environment_expose.{up,down}.sql`（含按 project seed local + service backfill）
2. `model.Environment`（含 `ProjectID`）/ `Expose` / `EnvironmentBinding`；扩展 `Service`、`Deployment`；删除 `ApplicationRoute`、`Application.RouteManaged`
3. `repository.CDStore` / `DeploymentExecutionStore` 接口方法替换
4. sqlx cd repository 实现；删除 Routes/ApplicationRoute CRUD

### Step 2 — Proto + gen

1. 新 `environment.proto`（含 `project_id`）；改 `version.proto`（expose）、`application.proto`（deploy）、`application_bundle.proto`、`deployment.proto`
2. 去掉 route 相关 application_route 若仅 app 级；**保留** 项目级 `route.proto`（独立 Traefik 文件路由，与 application_route 不同）
3. `buf generate`；更新 Go mapper / TS gen

### Step 3 — Usecase：Environment + Expose + Service 键

1. Environment CRUD（project 作用域）+ binding 替换 + 冲突校验
2. Project 创建种子 local env
3. Version 创建/更新/fork/publish 带 expose
4. Service list/get-by-key；ingress 互斥事务；deploy 同 project 校验
5. 权限检查

### Step 4 — Render + deploy chain

1. `RenderCompose` 双输入 + http/tcp 分支
2. Deploy/Stop/Restart 按 service 键；workspace 路径与 compose project 名
3. Preview 带 environment_id + 同 project
4. 删除 application_route usecase/handler/routes
5. Import/Export 调整

### Step 5 — HTTP handlers + routes

1. `registerEnvironment`（list/create 强制 project_id）
2. 调整 application deploy/status/logs/service
3. 路由注册单数 path

### Step 6 — Frontend

1. API 模块 `web/src/api/cd/environment.ts`；改 application.ts
2. Environment 页 + 路由（当前项目）
3. ApplicationDetail Version/Expose/Deploy；删旧路由 UI
4. i18n；lint + typecheck

### Step 7 — Tests + 工程检查

1. 迁移版本断言 → 9
2. 集成测试：双实例 service 键、**跨 project deploy 拒绝**、同 project 多 app 共 env、http+tcp render labels、ingress 互斥、binding 缺失失败、新建 project 有 local env
3. 更新 compose_test / deployment_execution_test fakes
4. `go fmt/vet/test`；`yarn --cwd web lint:fix typecheck`

## Files to change（预期主集合）

| 区域 | 路径 |
|------|------|
| SQL | `sql/migration/sqlite|mysql/000009_*` |
| Model | `internal/model/cd.go` |
| Repo | `internal/repository/cd.go`、`internal/repository/impl/sqlx/cd/*` |
| Usecase | `internal/application/cd/usecase/*`（environment 新文件；compose_renderer；deployment_execution；service；version；application_extra）；**project create 联动** |
| Workspace | `internal/infrastructure/storage/local/cdworkspace/*`、`internal/application/cd/port/*` |
| Proto | `proto/orbit/v1/environment.proto`（新）、version/application/application_bundle/deployment |
| Handler/Routes | `internal/api/http/handler/cd/*`、`internal/api/http/routes/*` |
| Web | `web/src/api/cd/*`、`web/src/views/cd/*`、router、i18n |
| Tests | `internal/application/cd/usecase/*_test.go`、`internal/test/e2e/*`、bootstrap migration 版本 |

## Verification plan

| 检查 | 命令/方式 |
|------|-----------|
| 迁移 | 空库 migrate 至 9；含 service 重建路径；多 project 各有 local |
| 单元/集成 | `go test ./internal/application/cd/... ./internal/api/http/handler/cd/... ./internal/test/e2e/...` |
| 全量 | `go test ./cmd/... ./internal/...` |
| 前端 | `yarn --cwd web lint:fix` && `yarn --cwd web typecheck` |
| 手工 | 切换项目 → 仅见本项目 env；建 binding；version 含 http+tcp expose；deploy 到 local；跨项目 env id 应失败；改域名不改 version |

## Assumptions

1. 单机 Docker + Traefik 仍是唯一 runtime；Environment 元数据不表示多机 agent（字段可后续扩）。
2. 项目级 `Route`（文件路由）产品保留，与 application 级 expose **并存**；本 Plan 不改其语义。
3. 每项目 seed `local` 足够本机开发；生产可在同一项目下再加 env 行。
4. `attach_ingress` 未传时：仅 `instance_key=="default"` 为 true。
5. sqlite 改 UNIQUE 可能整表重建 `service`；可接受 downtime（开发期）。
6. Environment **不做**跨项目共享；若未来需要「物理集群共享」，另开需求，不在本 Plan 放宽 FK。

## Risks

| 风险 | 缓解 |
|------|------|
| workspace 路径变更导致旧目录残留 | 文档说明；可选一次性移目录脚本 **不做** 自动兼容 |
| Traefik TCP 入口未在集群配置 | 仍写 labels；运行手册要求 entrypoint 存在 |
| 权限过宽/过窄 | 项目成员 + environment permission；后续可加 env ACL 表 |
| 多 Service 列表 API 前端遗漏 | Step 6 显式列表与部署选择 |
| migration 在已有多脏数据上失败 | 无兼容；开发库可重置 |
| 历史「松散 Environment」心智残留 | 需求/domain/cadence/plan 同步更正；实现仅 project 作用域 API |

## Rollback

1. 代码回滚至 Plan 前 commit。
2. DB：执行 `000009` down（丢 environment/expose/binding 与 service 新列数据）。
3. **不**恢复 `application_route` 数据（已声明无历史兼容）。

## Out of scope（本 Plan 不做）

1. 同域名多实例 LB / 蓝绿 UI。
2. Environment 细粒度 ACL 表（仅 permission code + 项目成员）。
3. K8s / 远程 Docker host 字段落地。
4. manual 证书上传挂 EnvironmentBinding。
5. 修改已执行的 000001–000008。
6. 跨 Project 共享 Environment。
7. **环境接入策略取代 Binding**（域名模板、去掉环境页选 component）→ **R2**：`docs/requirement/20260722-cd-environment-ingress-policy.md`。

## Follow-on / 后继

| 文档 | 说明 |
|------|------|
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | R2：IngressPolicy 取代 per-component Binding 用户路径 |
| `docs/verification/20260721-cd-environment-expose-service.md` | R1 验收（有条件通过）；产品反馈驱动 R2 |

## Open questions

**无。** 实现向细节已在 Plan decisions 拍板。  
R2 的开放问题见 ingress-policy Requirement（Q1–Q7），不回写本 Plan。

## User review notes

- 2026-07-21：Requirement Accepted 后用户要求「开始 Plan」。
- 2026-07-21：用户更正——Environment **归属 Project**（非松散全局）；Plan 同步：`project_id`、`(project_id,code)` 唯一、同项目 deploy、列表/UI 随当前项目、按项目 seed `local`。
- 2026-07-21：用户 **接受本 Plan**，进入 Implementation。
- 2026-07-22：索引挂入 R2 后继；本 Plan 范围不扩展。
