# CD 应用版本化实现计划
最后修改时间: 2026-07-22 08:35:55

Review status: Accepted

Flow mode: standard

## Requirement basis

| ID | 文档 | 状态 |
|----|------|------|
| 节奏 | `docs/requirement/20260721-cd-application-version-cadence.md` | Accepted |
| P0 | `docs/requirement/20260721-cd-application-version-domain.md` | Accepted |
| P1 | `docs/requirement/20260721-cd-application-version-spec.md` | Accepted |
| P2 | `docs/requirement/20260721-cd-application-version-deploy.md` | Accepted |
| P3 | `docs/requirement/20260721-cd-application-version-render.md` | Accepted |

**硬约束（需求已钉死）**：

- Version = 业务规格；deploy **必选 Version**。
- 顺序：① Service upsert → ② Render(Version+Components) → ③ apply → ④ Container + component_name。
- **放弃历史兼容**：无旧部署路径、无双轨、无 raw compose。
- Environment / 域名路由 / K8s：本期不做。
- force recreate 等 = **部署操作选项**，挂 Deployment。
- 不修改已执行 migration；仅新增。

## Plan decisions（用户已拍板）

| # | 决策 |
|---|------|
| 1 | 运行绑定表物理名 **`service`**（不是 `cd_service`） |
| 2 | **Component 拆独立表** `component`，不嵌在 Version JSON 大字段里当唯一形态 |
| 3 | **Container 一期不落库**，runtime 派生，API 带 `component_name` |
| 4 | **DROP** `application_config_file`、`application_service`；删除 `application.status` 列 |

Go 侧：领域实体 `model.Service` / `model.Component` / `model.Version`；usecase 门面仍为包内 `cdsvc.Service`（与 model 不同限定符，可共存）。

## Overview

| 实体 | 存储 | 职责 |
|------|------|------|
| Application | `application` | 身份 |
| Version | `version` | 版本元数据 + 版本级 env 等 |
| Component | **`component` 表** | 规格单元；属某 Version |
| Service | **`service` 表** | 本机运行绑定与状态 |
| Deployment | `deployment` | 操作记录 |
| Container | **无表** | 派生观测 |

## Implementation steps

### Step 1 — 数据模型与迁移

新增 migration（sqlite + mysql）：

**`version`**

| 列 | 说明 |
|----|------|
| id | PK |
| application_id | FK |
| label | 与 application_id UNIQUE |
| status | `unpublished` \| `published` |
| env_json | 版本级全局 env（可空） |
| created_from_version_id | 可空 |
| note | 可空 |
| created_at / updated_at | |

**`component`**（拆表）

| 列 | 说明 |
|----|------|
| id | PK（代理键，便于引用） |
| version_id | FK → version，ON DELETE CASCADE |
| name | Version 内唯一；`UNIQUE(version_id, name)`；业务身份之一 |
| image | 必填 |
| command_json / args_json | 可空 |
| env_json | 可空；与 version.env 合并时 Component 覆盖同名键 |
| ports_json | 可空；容器侧端口意图 |
| mounts_json | 可空 |
| networks_json | 可空 |
| depends_on_json | 可空；存依赖的 **component name** 列表 |
| healthcheck_json | 可空 |
| resources_json | 可空 |
| pull_policy | 可空 |
| created_at / updated_at | |

> 领域复合身份仍是 `(version_id, name)`；表用代理 `id` + 唯一约束。  
> Container 引用用 **name**（component_name），与 compose service 名对齐。

**`service`**（运行绑定）

| 列 | 说明 |
|----|------|
| id | PK，stop/start 不变 |
| application_id | **UK**（本期一本机一条） |
| version_id | 意图 Version |
| last_successful_version_id | 可空 |
| status | deploying/running/stopped/faulted |
| created_at / updated_at | |

**`deployment` 扩展**

- `version_id`、`service_id`、`options_json`（force_recreate 等）

**删除（无兼容）**

- `DROP TABLE application_config_file`
- `DROP TABLE application_service`
- `application` 表 **DROP COLUMN status**（sqlite 按项目既有 migration 写法处理）
- **保留** `application_route`

sqlc query：`version.sql`、`component.sql`、`service.sql`、更新 deployment/application。

Seed / business_data：改为 Application + Version + Component 行，无 config_file。

### Step 2 — 领域模型与端口

1. `model.Version`、`model.Component`、`model.Service`、扩展 `model.Deployment`。
2. Repository：Version CRUD；Component 按 version 批量读写；Service GetByApplicationId / Upsert；Deployment 带新字段。
3. 常量：Version status、Service status、operation_type。
4. published Version：禁止改 Version 元数据中的可变规格字段，且禁止改其 Component 行（update/delete/insert 均拒）。

### Step 3 — ComposeRenderer（P1+P3）

1. 加载 `Version` + `[]Component`（≥1）。
2. 校验 name 唯一、depends_on 指向存在 name。
3. 渲染 `docker-compose.yml`（路径解析用 physical_app_dir）。
4. 预览 API 与 deploy **同一 Render 函数**。
5. 无 raw compose 编辑。

### Step 4 — 部署执行链（P2）

```text
Deploy(applicationId, versionId, options):
  load Version + Components
  ① upsert service(version_id, deploying); create deployment(...)
  ② Render(Version, Components) → appDir
  ③ compose up (+ options)
  ④ ps → component_name; running 则 last_successful_version_id；否则 faulted
```

- Restart：读 `service.version_id` + 其 Components。
- Stop：`service.status=stopped`，不删行。
- Container：不落库；API 派生。
- deploy **必选 version_id**。

### Step 5 — HTTP / Proto

| 能力 | 路径示意 |
|------|----------|
| Version CRUD / publish / fork | `/api/cd/version` |
| Component 随 Version 写（或子资源） | 创建/更新 Version 时事务写入 component 表；或 `/api/cd/version/{id}/component` |
| Preview | `/api/cd/version/{id}/preview` |
| Deploy | `/api/cd/application/{id}/deploy` body: version_id, force_recreate? |
| Service 查询 | `/api/cd/service` 或 application 详情内嵌 |
| Containers | 派生列表 |
| Deployment | 现有 + version_id / options |

删除 config_file / service_config API。

Fork Version：复制 version 行 + **复制全部 component 行**（新 id，新 version_id）。

### Step 6 — 前端

- Version 列表/编辑；Component 可多条编辑（拆表后仍是 Version 编辑 UX）。
- 部署必选 Version；操作选项 force recreate。
- 运行态读 `service`；Container 表含 component。
- 去掉 compose 文件树 SoT UI。
- lint:fix + typecheck。

### Step 7 — 测试与清理

- Renderer、publish 不可改、deploy 缺 version、service id 稳定、Component 唯一约束。
- 删 config_file 死代码；更新 migrator/e2e。

## Files to change（主集合）

- `sql/migration/sqlite|mysql/000008_*.up.sql`
- `sql/query/version.sql`, `component.sql`, `service.sql`, deployment/application 调整
- `internal/model/*`
- `internal/repository/**`, `impl/sqlx/cd/**`
- `internal/application/cd/usecase/deployment_execution*.go`
- `compose_renderer.go`, `version*.go`, component 写入逻辑
- HTTP / proto / worker payload
- `web/src/views/cd/**`, api, i18n

## Verification plan

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

1. Version + ≥1 Component → preview → deploy(version_id)。
2. publish 后 Component/Version 规格不可改；fork 复制 Component 行。
3. 成功：`service.status=running`，last_successful 更新；Container 带 component_name。
4. 失败：service 仍在 faulted。
5. stop 后 service.id 不变。
6. 无 config_file 入口；无 version_id 的 deploy 400。

## Assumptions

1. 表名小写：`service`、`component`、`version`（与现库 `application` 风格一致）。
2. Component 复杂字段用 JSON 列，不拆 mounts 子表。
3. Container 不落库。
4. DROP 旧表可接受。
5. 单数 API 路径风格不变。

## Risks

1. 改动面大（创建/导入/测试原绑 config_file）。
2. `model.Service` vs `cdsvc.Service` 命名需在代码中保持限定清晰。
3. SQLite drop column 写法需符合项目 migration 惯例。
4. Renderer 路径 Windows/Linux。

## Blockers

- 无。Plan 已 Accepted；进入 Implementation。

## Rollback

- Git / migration down；无退回 config_file 业务路径。

## Out of scope

- Environment、多机、K8s、对外访问改造、密钥库、Container 表、raw compose。

## Follow-on / 后续路线（本 Plan 之后）

本 Plan 仅 Version 底座。对外访问与 Environment 不在此实现，已拆为独立修订链：

| 顺序 | 文档 | 状态 |
|------|------|------|
| R1 | Requirement/Plan/Verification `20260721-cd-environment-expose-service` | **已交付** |
| R2 | Requirement `docs/requirement/20260722-cd-environment-ingress-policy.md` | **Draft**（strict）；修正 R1 的 per-component Binding 形态为环境接入策略 |
| — | K8s / 蓝绿等 | 未开 |

索引总表见 cadence：`docs/requirement/20260721-cd-application-version-cadence.md`。

## User review notes

- 2026-07-21：P0–P3 Accepted；进入 Plan。
- 2026-07-21：用户拍板——**表用 service**；**Component 拆表**；Container 不落库与 DROP 旧表 **接受**。
- 2026-07-21：用户 **接受 Plan**，进入 Implementation。
- 2026-07-22：挂入 R2 后续路线指针（ingress-policy），不改本 Plan 范围。
