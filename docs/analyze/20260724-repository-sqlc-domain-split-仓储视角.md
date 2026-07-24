# Repository / sqlc 领域拆分（仓储视角）
最后修改时间: 2026-07-24 09:45:28

## 目的

记录从「数据访问实现」角度对仓储域的切分理解，用于 sqlx → 全量 sqlc 迁移时的包边界、query 文件边界与交付切片。  
不含业务背景与产品动机。

## 分层位置

```text
internal/repository/*.go          # 接口（跨域契约）
internal/repository/impl/sqlc/*   # 实现（按域分包）
sql/query/*.sql                   # 命名 SQL（按域分文件）
internal/gen/sqlc                 # 生成物（可按 query 文件拆多个 .sql.go）
```

连接与迁移不属业务域：

```text
internal/infrastructure/database  # *sql.DB Open / Migrate
internal/bootstrap                # 装配
```

## 域总表

| 域 ID | 名称 | 接口（现状） | 实现现状 | query 文件（现状/目标） |
|-------|------|--------------|----------|------------------------|
| conn | 连接与装配 | — | sqlx | — |
| task | 后台任务 | task repo（queue 侧） | 混 sqlc+sqlx | `background_task.sql` |
| role | 角色权限 | `RoleStore` | 混 sqlc+sqlx | `role.sql` |
| user | 用户与登录 | `UserStore` | 混 sqlc+sqlx | `user.sql` |
| project | 项目与成员 | `ProjectStore` | 混 sqlc+sqlx | `project.sql` |
| ci | CI | `CIStore` / `PipelineExecutionStore` | 全 sqlx | 多文件（见下） |
| cd | CD | `CDStore` / `DeploymentExecutionStore` | 全 sqlx | 多文件（见下） |

横切只读（CI/CD 内重复出现，宜收敛）：

| 横切 | 典型方法 | 归属建议 |
|------|----------|----------|
| project-read | `Project`, `IsProjectMember` | 复用 `project` query / 委托 `ProjectStore`，CI/CD 不复制实现 |

## 域内子切片

### conn

- Open SQLite / MySQL
- pragma / 连接池
- MigrateUp / ReadMigrationVersion
- bootstrap HTTP / worker 注入
- 测试 `sql.Open` 基建

### task

- enqueue
- claim（事务：选行 + 乐观更新）
- complete / fail
- find by id

### role

- by id / code / name
- list（page + search）
- create / update / delete
- permissions list
- set role permissions（事务）
- permission by codes（IN）

### user

- by id / username / email
- create / update / status / delete
- mark logged in
- login history save / list
- roles / permissions / role details
- list users
- set user roles（事务）
- roles by user ids（IN 或循环）

### project

- by id / code
- list by member / active by member
- create（事务：project + member + 默认 environment 等）
- update / deactivate
- members list / add / remove
- count repositories / applications
- is member

### ci 子域

| 子 ID | 实体簇 | 能力要点 |
|-------|--------|----------|
| ci.credential | credential | CRUD、list/search、exists、name、referenced by repo |
| ci.repository | repository | CRUD、list/search、by code、has running pipelines |
| ci.webhook | repository_webhook | list by repo、by id、CRUD |
| ci.build_stage | build_stage | CRUD、list/search、by name、by ids（IN）、referenced |
| ci.template | pipeline_template + stage 关联 | CRUD、list、stages、update-with-stages / duplicate 事务、referenced、delete |
| ci.snapshot | pipeline_snapshot | create、get、latest by template |
| ci.run | pipeline_run + stage_run | create、状态流转、list 多条件、list by repo、stage list/get/insert/update |
| ci.artifact | artifact | insert、list 多条件、list by run |
| ci.cross | project-read | 委托/复用 project |

### cd 子域

| 子 ID | 实体簇 | 能力要点 |
|-------|--------|----------|
| cd.application | application | CRUD、list（project/search/kind）、by name/code |
| cd.version | version | CRUD、list/page、runtime ref count、delete 事务 |
| cd.component | version_component | list by version、replace 事务、随 version 创建 |
| cd.expose | version_expose | list by version、replace 事务、随 version 创建 |
| cd.environment | environment | CRUD、list/search、by project+code、service count |
| cd.service | service | by key/id、upsert、status、after-deploy、list by app、list by project（join） |
| cd.gateway | gateway_config + gateway app | get/upsert、resolve active、list gateway apps、create app+config 事务、has active service |
| cd.route | route | CRUD、list/search、list all、list enabled、by domain |
| cd.deployment | deployment | create、running/complete/cancel、list 多条件 |
| cd.cross | project-read | 委托/复用 project |

## 交付切片映射（仓储迁移）

| 切片 | 覆盖域 | 依赖 |
|------|--------|------|
| T0 | 约定（全域适用） | — |
| T1 | conn | T0 |
| T2 | task | T0，宜与 T1 闭合编译链 |
| T3 | role | T1/T0 |
| T4 | user | T3（角色绑定） |
| T5 | project | T4 弱依赖（member 用户存在） |
| T6 | ci.* | T5（project-read） |
| T7 | cd.* | T5（project-read） |
| T8 | 清零（conn + 全域） | T2–T7 完成 |
| T9 | 验证 | T8 |

CI/CD 内部可再按子 ID 拆 PR，但 **T6 / T7 完成条件** 为对应大域零 sqlx、零生产手写 SQL。

## query 文件目标拆分

| 文件 | 域 |
|------|-----|
| `background_task.sql` | task |
| `role.sql` | role |
| `user.sql` | user |
| `project.sql` | project |
| `credential.sql` | ci.credential |
| `repository.sql` | ci.repository |
| `webhook.sql` | ci.webhook |
| `build_stage.sql` | ci.build_stage |
| `pipeline_template.sql` | ci.template |
| `pipeline_snapshot.sql` | ci.snapshot（新建） |
| `pipeline_run.sql` | ci.run（新建，可含 stage_run） |
| `artifact.sql` | ci.artifact（新建） |
| `application.sql` | cd.application（新建） |
| `version.sql` | cd.version + component + expose（可再拆文件） |
| `environment.sql` | cd.environment（新建） |
| `service.sql` | cd.service（新建） |
| `gateway.sql` | cd.gateway（新建） |
| `route.sql` | cd.route（新建） |
| `deployment.sql` | cd.deployment（新建） |

实现时可合并过碎文件，但 **逻辑域边界** 以上表为准。

## 实现包目标拆分

```text
internal/repository/impl/sqlc/
  dbmodel/          # 生成类型 ↔ model 转换（横切）
  task/
  role/
  user/
  project/
  ci/               # 或再分子目录 credential/repository/...
  cd/               # 或再分子目录 application/version/...
```

删除目标：

```text
internal/repository/impl/sqlx/
```

## 域间依赖方向

```text
conn
  ↑
task, role, user, project, ci, cd
  ↑
user → role          # 角色绑定读 role 数据
project → user       # member 关联用户（弱）
ci → project         # project-read
cd → project         # project-read
ci 内部: template → build_stage, run → snapshot/repository, artifact → run
cd 内部: version → application, component/expose → version,
         service → application+environment+version,
         deployment → application+version+service,
         gateway → application
```

禁止反向依赖（例如 project 实现依赖 ci/cd）。

## 非域（明确排除）

- HTTP handler / routes / proto
- application usecase 业务规则
- runner、workspace、envfile、execution log 等非 SQL 存储
- migration 文件内容与版本策略（另册）

## 与其他视角文档的关系

| 文档 | 视角 |
|------|------|
| `20260724-api-proto-domain-split-proto视角.md` | API/proto |
| `20260724-ci-cd-schema-domain-split-数据库视角.md` | schema 表 |
| **本文** | repository 实现 / sqlc query / 迁移交付切片 |

三份边界应大致可对齐，但允许粒度不同（例如 schema 一表、仓储一子域多表）。
