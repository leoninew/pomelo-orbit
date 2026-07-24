# CD Application kind 实现计划
最后修改时间: 2026-07-22 13:06:52

Review status: Accepted

Flow mode: strict

## Requirement basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260722-cd-application-kind-gateway.md` | **Accepted** |
| `docs/spec/20260722-cd-application-kind-gateway.md` | **Accepted**（进入 Plan） |
| `docs/requirement/20260721-cd-application-version-cadence.md` | Accepted；本 Plan = **修订 R3** |
| `docs/verification/20260722-cd-environment-ingress-policy.md` | R2 已交付；standard labels 底座 |

**硬约束（不得偏离 Spec）**：

1. `application.kind`：`standard` \| `gateway`；默认 `standard`；**仅创建可写**（Update 契约无 kind）。  
2. Render **按 `app.kind` 分支**；禁止 `code==…` 魔法。  
3. gateway **无**额外 Component/端口数量校验；与 standard 共用物化路径。  
4. **删除** `attach_ingress`、`service.is_ingress`（无双轨）。  
5. labels 条件 = **存在 Expose**（不再看 IsIngress）。  
6. 不硬编码 Docker 网络名；`domain_suffix` 仅指**域名后缀**。  
7. **container_name** = `app_code + "_" + component_name`；**service 键 / depends_on** = Component.name（不映射）。  
8. app.code / component.name 早期规则 `^[a-z][a-z0-9-]*$`（无 `_`）；Render **不**做复杂 sanitize。  
9. **开发期 schema 就地改**：`kind` 写入 `000002` 建表；`is_ingress` 从 `000009` 去掉；**不**新增 `000011`；migrate 断言版本仍为 **10**。  
10. E1/E5 不在本期实现（E5 含 gateway 挂载 + **补齐 Component 全规格**，不另开任务）。

## Plan decisions

| # | 决策 | 说明 |
|---|------|------|
| P1 | 列名 | `application.kind` TEXT/VARCHAR NOT NULL DEFAULT `'standard'` |
| P2 | 合法值 | 创建 normalize：空→`standard`；仅允许 `standard`/`gateway` |
| P3 | Update | DTO/proto **无** kind 字段；SQL UPDATE 不写 kind |
| P4 | Render 分支 | `RenderCompose` 入口 `switch` kind；standard/gateway 可共用 Component 循环；gateway 独立函数便于后续 E1/E5 |
| P4b | 容器名 | 每个 service：`container_name: {app_code}_{component_name}` 纯拼接；**service 键**仍 `component.Name`；`depends_on` 原样输出组件名列表 |
| P4c | 命名校验 | Create app：保持 `^[a-z][a-z0-9-]*$`；保存 Version/Component：同规则校验 component.name；禁止 `_`；失败在 usecase，不在 Render 洗字符串 |
| P5 | labels | `if len(exposes) > 0 { injectExposeLabels(...) }`；按 component_name 找 **service 键**；删除 IsIngress 条件 |
| P6 | Deploy | 创建/更新 Service **不写** is_ingress；删除 `ClearOtherIngressServices`（或等价）调用 |
| P7 | Options JSON | `DeployOptionsJSON` 去掉 `attach_ingress`；历史 options 中该键忽略 |
| P8 | Proto | `application.proto`：Create/Resp +kind；Deploy 删 attach；`version.proto`（及 Service 消息）删 attach/is_ingress |
| P9 | UI | `ApplicationFormFields` 创建 kind；详情只读 badge/文本；Deploy 去 checkbox；Services 去 is_ingress |
| P10 | 测试 | migrate=**10**；compose：有 Expose 写 labels、无 Expose 不写；kind 分支函数被调用（gateway≠走 code 特判）；创建 kind 持久化；Update 无法改 kind |
| P11 | sqlc | `internal/gen/sqlc/models.go` 等与就地 schema 一致 |

## Implementation steps

### Step 1 — Schema 就地修改（开发期，无增量 migration）

文件：

- `sql/migration/sqlite|mysql/000002_orbit_schema.up.sql`：`application.kind` NOT NULL DEFAULT `'standard'`  
- `sql/migration/sqlite|mysql/000009_cd_environment_expose.up.sql`：service 建表/改表 **不含** `is_ingress`  
- mysql `000009` down：去掉 DROP `is_ingress`  

不新增 `000011_*`。

### Step 2 — Model / Repository

- `internal/model/cd.go`：`Application.Kind`；`Service` 去掉 `IsIngress`。  
- `internal/repository/impl/sqlx/cd/repository.go`：  
  - application columns + Create/Update/Get/List 读写 `kind`（Update **SET 列表不含 kind**）。  
  - service columns 去掉 `is_ingress`；Create/Update Service 去掉相关参数。  
  - 删除 `ClearOtherIngress…` 类方法（若仅服务于 is_ingress）。  
- 其它引用 `IsIngress` 的 repo 调用一并删。

### Step 3 — DTO / Usecase

- `internal/application/cd/dto/cd.go`：  
  - `ApplicationCreateInput.Kind`  
  - `ApplicationUpdateInput` 无 Kind  
  - Deploy/Preview 输入去掉 `AttachIngress`  
  - `DeployOptionsJSON` 去掉 `AttachIngress`  
- `CreateApplication`：normalize kind。  
- `UpdateApplication`：不碰 kind。  
- `deployment_execution` / preview / version preview：  
  - 不再读 attach_ingress / 写 IsIngress  
  - `RenderInput.Service` 无 IsIngress 依赖  
- `compose_renderer.go`：  
  - 分支 kind  
  - labels 改为 Expose 驱动  
  - 写 `container_name`；**不**改 service 键；**不**映射 depends_on  
  - gateway 独立函数（可委托共用渲染）  
- Component 保存路径：name 规则与 app.code 对齐

### Step 4 — Proto + gen

- `proto/orbit/v1/application.proto`：Create/Resp + kind；Deploy 删 `attach_ingress`。  
- `proto/orbit/v1/version.proto`：Preview/Service 视图删 `attach_ingress` / `is_ingress`。  
- 其它 bundle 若嵌套 Service 字段同步。  
- 运行 `task proto`（或项目既定生成命令）。

### Step 5 — HTTP handler / mapper

- Create 映射 kind；Resp 输出 kind。  
- Deploy/Preview mapper 去掉 attach。  
- Service list mapper 去掉 is_ingress。  
- 路由无需新 path（仍单数 resource）。

### Step 6 — Frontend

- `ApplicationFormFields.vue` / 创建流：kind 选择（默认 standard）。  
- 列表/详情：展示 kind（只读）。  
- `ApplicationDetail.vue`：Deploy 去掉 attach_ingress；Services 去掉 is_ingress 展示。  
- `web/src/api/cd/*` 与 gen 类型对齐。  
- i18n zh-CN / en-US。

### Step 7 — Tests & checks

- `migration_test` / `bootstrap` / `e2e`：version **10**。  
- `compose_test`：  
  - Expose + policy → Host labels（不依赖 IsIngress）  
  - 无 Expose → 无 traefik labels  
  - `container_name: demo_web`；service 键仍为 `web`；depends_on 仍为 `web`  
  - 非法 component 名（含 `_` 或大写）保存失败  
  - `kind=gateway` 走 gateway 路径（**禁止** code==traefik）  
- deploy/execution 测试去掉 attach/is_ingress。  
- `go fmt` / `go vet` / `go test ./cmd/... ./internal/...`  
- `yarn --cwd web lint:fix` + `typecheck`

## Files to change（预期主集合）

| 区域 | 路径 |
|------|------|
| SQL | `000002_orbit_schema`（+kind）、`000009_cd_environment_expose`（-is_ingress） |
| Model/Repo | `internal/model/cd.go`, `internal/repository/impl/sqlx/cd/repository.go`, `internal/repository/cd.go`（若接口签名变） |
| Usecase | `compose_renderer.go`, `compose_test.go`, `deployment_execution*.go`, `service.go`, `version.go`, `application_extra.go`, dto |
| Proto/Gen | `proto/orbit/v1/application.proto`, `version.proto`, gen go/ts |
| HTTP | handler/mapper application |
| Web | ApplicationFormFields, ApplicationDetail, i18n, api |
| Tests | migration_test, bootstrap app_test, e2e, integration tests |

## Verification plan

| 项 | 方式 |
|----|------|
| kind 持久化 | 单测或集成：Create gateway → Get kind=gateway |
| kind 不可改 | Update 无字段；DB 中 kind 保持 |
| labels / Expose | compose_test |
| 无 is_ingress / attach | 全仓 grep 产品路径为零（测试夹具同步删） |
| migrate 10 | migration_test |
| 工程门禁 | go test + yarn lint/typecheck |

## Blockers

- 无外部依赖。  
- sqlite `service` 表重建需对照 000009/000010 当前列清单，避免漏列。

## Assumptions

1. R3 实现期 gateway 与 standard 渲染体可高度重合；以可测分支为准。  
2. 设置项 `traefik__domain_suffix` **不**在本 Plan 改动。  
3. 不实现 E1/E5。

## Risks

1. 漏改测试夹具中的 IsIngress 导致编译失败——Step 7 全量 test 兜住。  
2. sqlite 重建 service 漏索引——对照现网 schema。  
3. 前端 Deploy 默认 attach 删除后行为变化（有 Expose 即写 labels）——符合 Spec，需在 UI 文案上可选提示。

## Rollback

- 开发库重建 / 重跑 migrate（无增量 down）。  
- 产品无兼容层、不恢复 attach 双轨。

## Out of scope

- E1 consumer 网络、E5（gateway 挂载细节 + Component 全规格补齐）、K8s、host 网络。  
- 应用级域名覆盖。  
- 生产增量 migration（本期开发库就地改现有文件）。

## User review notes

- 2026-07-22：用户「开始 plan」→ Spec Accepted 后创建本 Plan Draft，待 Accept 后实现。  
- 2026-07-22：澄清 domain_suffix=域名后缀。  
- 2026-07-22：纠正——**仅 container_name** 用 `app_code_component`；depends_on/service 键用组件名；早期命名规则；Spec 占位 compose 示例。  
- 2026-07-22：用户「开始实现」→ Plan **Accepted**；Implementation 已完成。  
- 2026-07-22：用户要求 **开发期 schema 就地改**，删除 `000011`；`kind` 并入 `000002`，`is_ingress` 从 `000009` 去除；migrate 仍为 **10**。  
- 2026-07-22：Verification **Accepted**。
