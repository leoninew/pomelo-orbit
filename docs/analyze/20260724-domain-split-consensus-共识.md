# Pomelo Orbit 领域划分基线

日期：2026-07-24
状态：已确认的当前项目领域边界；后续 Proto、SQL migration 与 sqlc 拆分以本文为准。

## 1. 目的与范围

本文将 API Proto、数据库 Schema、Repository / sqlc 和当前实现中的结论收敛为一份项目领域地图。它回答四个问题：

1. 每个领域拥有哪类业务职责和实体；
2. 领域之间为何依赖、哪些资源不应合并；
3. Proto、SQL migration 和 sqlc query / 实现应如何落点；
4. 现有命名、消息归属和遗留契约需要如何纠正。

“领域”定义资源生命周期、写入职责和依赖方向，不等同于一个 migration 文件、一个 Proto 文件或一个 Go 包。各层必须遵循本文的**逻辑归属**，但可以为 FK、生成工具和查询模式采用不同的物理文件粒度。

`ci`、`cd` 是历史路由和迁移分组，不是领域名。`common` 是共享契约类型，不是业务领域。

本文不定义 HTTP handler、worker 装配、生成工具配置或 migration 版本号。未来的定时调度能力尚无当前业务模型，不在本轮泛化范围内。

## 2. 划分依据

| 依据 | 本项目中的应用 |
|---|---|
| 聚合与生命周期 | pipeline 定义、一次 pipeline 执行、应用版本规格、运行时 service、deployment 历史分别拥有不同状态变化，不能合并。 |
| 资源归属 | webhook 从属 repository；artifact 和 pipeline stage run 从属 pipeline run；version component / expose 从属 application version。 |
| 领域能力而非表位置 | gateway 是平台网络基础设施，即使 `gateway_config` 以 `application_id` 关联部署载体，仍属于 gateway。 |
| 上游租户边界 | project 拥有项目成员关系；交付领域读取 project / membership，project 不反向依赖交付领域。 |
| 查询不改变归属 | 子资源可以有独立 query 文件，例如 webhook、artifact、snapshot；这不使其成为顶层领域。 |
| 技术能力独立 | task 拥有队列执行状态，不拥有业务 payload；settings 管理进程配置，不构成关系型业务数据。 |

## 3. 领域地图

### 3.1 平台核心

| 领域 | 职责 | 逻辑实体 / 数据 |
|---|---|---|
| `auth` | 认证会话与安全审计：登录、Token、OAuth、CSRF、Turnstile、改密、登录审计。 | login_history、login_attempt；读取 user 身份和 role 授权结果。 |
| `user` | 用户账户、密码、状态和用户角色分配。 | user、user_role。 |
| `role` | RBAC 策略定义。 | role、permission、role_permission。 |
| `project` | 租户边界与项目成员管理。 | project、project_member。 |
| `settings` | 平台进程配置的展示、环境覆盖、重置和 secret 标记。 | 配置定义、环境变量覆盖；无业务表。 |
| `task` | 通用可靠队列：创建、领取、重试、完成、失败和查询。 | background_task。 |

核心关系：`user_role` 从属于 user 的角色分配；`role_permission` 从属于 role 的策略定义；`project_member` 从属于 project 的成员关系。auth 通过窄的账户身份读取与授权读取端口读取 user / role，不拥有账户资料或 RBAC 策略。

### 3.2 流水线与交付

| 领域 | 职责 | 逻辑实体 / 数据 |
|---|---|---|
| `credential` | 可复用的代码源和构建密钥。 | credential。 |
| `repository` | 代码仓库及其回调配置。 | repository、repository_webhook。 |
| `pipeline` | 流水线定义、可复用阶段库和不可变快照。 | pipeline_template、pipeline_stage、pipeline_template_stage、pipeline_snapshot。 |
| `pipeline_run` | 一次流水线执行、阶段执行和产物。 | pipeline_run、pipeline_stage_run、artifact。 |
| `application` | 应用定义和应用版本规格。 | application、version、version_component、version_expose；application 导入/导出传输。 |
| `environment` | project 级部署目标。 | environment。 |
| `service` | application × environment × version 的运行时绑定。 | service。 |
| `deployment` | 部署、重启、停止、回滚等操作历史。 | deployment。 |
| `gateway` | 持续部署平台的网络基础设施。 | gateway、gateway_config、gateway exposure 投影。 |
| `route` | project 级 L7 路由与证书意图。 | route、Traefik 只读观测契约。 |

关键边界：

- `pipeline_stage` 是可复用阶段定义；`pipeline_template_stage` 是模板中的编排引用；`pipeline_stage_run` 是某次 pipeline run 的阶段执行实例。
- application version 是 application 在某一时刻的规格。component / expose 从属于 version；Compose 的镜像、环境变量、挂载、网络等配置由 version_component 持久化。
- service 是运行时绑定，不是 version 字段；deployment 是操作历史，不是 service 的状态扩展。
- gateway 是顶层网络领域。它使用 kind=gateway 的 application 作为部署载体，并复用该 application 的普通 version、service、deployment 能力；Gateway version 没有专用模型或特殊生命周期。
- route 拥有期望的路由和证书规则；gateway 负责把网络配置发布到实际控制面。route 不从属于 application、version 或 gateway。

## 4. 依赖方向

箭头表示右侧领域依赖左侧领域的能力或资源；不要求每一层都产生直接 import 或 FK。

```text
role ──→ user ──→ project
role + user ──→ auth

project
  ├──→ credential ──→ repository
  ├──→ pipeline ──→ pipeline_run
  ├──→ repository ──→ pipeline_run
  ├──→ application（含 version）
  ├──→ environment
  ├──→ gateway
  └──→ route

application（含 version）+ environment ──→ service
service + application version + environment ──→ deployment
gateway ──→ application version / service / deployment
pipeline_run + deployment ──→ task

settings（独立的平台进程配置能力）
```

补充约束：

- credential 被 repository 引用；repository webhook 可以引用 pipeline template。
- pipeline snapshot 属于定义态，pipeline run 引用 snapshot；artifact 随 pipeline run 产生。
- task 只保存通用队列信封和执行状态。pipeline run、deployment 等生产领域生成并解释各自的任务命令和 payload。
- gateway 读取 application version / service / deployment 能力来运行自身；普通 application 在需要公网暴露时读取 gateway 提供的网络能力。双方不互相拥有生命周期。

## 5. Proto 指导

### 5.1 目录和 package

目标路径为 `proto/orbit/v1/<domain>/<entity>.proto`，package 为 `orbit.v1.<domain>`。同领域文件使用同一 package；跨领域类型使用显式 import。`common` 位于 `orbit.v1.common`。

| 领域 | Proto 文件目标 |
|---|---|
| `common` | `common/common.proto` |
| `auth` | `auth/auth.proto` |
| `user` | `user/user.proto` |
| `role` | `role/role.proto` |
| `project` | `project/project.proto` |
| `settings` | `settings/settings.proto` |
| `task` | `task/task.proto` |
| `credential` | `credential/credential.proto` |
| `repository` | `repository/repository.proto`、`repository/webhook.proto` |
| `pipeline` | `pipeline/template.proto`、`pipeline/pipeline_stage.proto`、`pipeline/snapshot.proto` |
| `pipeline_run` | `pipeline_run/pipeline_run.proto`、`pipeline_run/pipeline_stage_run.proto`、`pipeline_run/artifact.proto` |
| `application` | `application/application.proto`、`application/version.proto`、`application/application_bundle.proto` |
| `environment` | `environment/environment.proto` |
| `service` | `service/service.proto` |
| `deployment` | `deployment/deployment.proto` |
| `gateway` | `gateway/gateway.proto` |
| `route` | `route/route.proto`、`route/traefik.proto` |

### 5.2 消息归属

- `ServiceResp`、`ServiceListResp`、`ServicePaginatedResp` 迁出 version，定义在 `service/service.proto`。
- `PipelineRunExecuteTaskReq` 归 `pipeline_run`；`ApplicationDeployTaskReq`、`ApplicationRestartTaskReq`、`ApplicationStopTaskReq` 归 `deployment`。`TaskResp` 与通用 Task 创建/查询契约保留在 task。
- application bundle 仅是 application 的导入/导出 DTO，不形成 bundle 领域或 sqlc 领域。
- Traefik 只读消息归 route；Gateway API 消息归 gateway。
- `ArtifactConfig*` 保留在 pipeline_stage，因为它描述构建阶段配置，不是执行产物 artifact。

生成物采用 source-relative 路径：Go 位于 `internal/gen/proto/orbit/v1/<domain>/`，TypeScript 位于 `web/src/gen/proto/orbit/v1/<domain>/`。

## 6. SQL migration 与 Schema 指导

Schema 以领域归属命名表和后续 migration；migration 可以将同一 FK 拓扑中的表放在一起，但必须在变更说明中标明其逻辑领域。

| 领域 | 目标表 / 非表资源 | Schema 说明 |
|---|---|---|
| `auth` | login_history、login_attempt | 认证审计从 user 账户表中拆出。 |
| `user` | user、user_role | user_role 的写入语义属于用户角色分配。 |
| `role` | role、permission、role_permission | 权限策略与角色一起维护。 |
| `project` | project、project_member | project_member 由 project 拥有。 |
| `settings` | 无表 | 配置由环境覆盖 adapter 持久化。 |
| `task` | background_task | 独立的队列状态表。 |
| `credential` | credential | 依赖 project。 |
| `repository` | repository、repository_webhook | repository 依赖 credential / project；webhook 关联 repository 与 pipeline template。 |
| `pipeline` | pipeline_template、pipeline_stage、pipeline_template_stage、pipeline_snapshot | pipeline_stage 是原 build_stage 的规范名称。 |
| `pipeline_run` | pipeline_run、pipeline_stage_run、artifact | pipeline_stage_run 是原 stage_run 的规范名称。 |
| `application` | application、version、version_component、version_expose | version 依赖 application；component / expose 依赖 version。 |
| `environment` | environment | 仅依赖 project。 |
| `service` | service | 依赖 application、environment、version。 |
| `deployment` | deployment | 依赖 service、version、environment、project。 |
| `gateway` | gateway_config | 以 application_id 关联部署载体；逻辑归属仍为 gateway。 |
| `route` | route | 仅依赖 project。 |

推荐的建表依赖顺序为：

1. role、user、permission、user_role、role_permission、project、project_member、auth 审计表、background_task；
2. credential、pipeline_stage、pipeline_template、pipeline_template_stage、pipeline_snapshot、repository、repository_webhook；
3. pipeline_run、pipeline_stage_run、artifact；
4. application、environment、version、version_component、version_expose、gateway_config、service、deployment、route。

其中 route 可以在 project 后独立创建；gateway_config 可与 application 放在同一 migration 以满足 FK，但其领域标识保持 gateway。

## 7. sqlc 与 Repository 指导

sqlc query 文件和实现包按下表领域拆分，禁止继续用 `ci`、`cd` 作为目标业务包。一个领域可以拥有多个 query 文件，但 query 不应跨越领域写入职责。

| 领域 | query 文件 / 实现切片 | 说明 |
|---|---|---|
| `auth` | `auth.sql` / `sqlc/auth` | 管理 login_history、login_attempt；通过账户身份读取端口与授权读取端口读取 user / role。 |
| `user` | `user.sql` / `sqlc/user` | 管理 user、user_role；分配角色前读取 role 校验。 |
| `role` | `role.sql` / `sqlc/role` | 管理 role、permission、role_permission。 |
| `project` | `project.sql` / `sqlc/project` | 管理 project、project_member；读取 user 用于成员校验和展示。 |
| `task` | `background_task.sql` / `sqlc/task` | 管理 background_task 的队列状态机。 |
| `credential` | `credential.sql` / `sqlc/credential` | 管理 credential。 |
| `repository` | `repository.sql`、`webhook.sql` / `sqlc/repository` | webhook 可独立 query 文件。 |
| `pipeline` | `pipeline_template.sql`、`pipeline_stage.sql`、`pipeline_snapshot.sql` / `sqlc/pipeline` | 定义态查询与写入。 |
| `pipeline_run` | `pipeline_run.sql`、`artifact.sql` / `sqlc/pipeline_run` | 执行态查询与写入。 |
| `application` | `application.sql`、`version.sql` / `sqlc/application` | version/component/expose 是 application 子聚合。 |
| `environment` | `environment.sql` / `sqlc/environment` | 管理 environment。 |
| `service` | `service.sql` / `sqlc/service` | 管理运行时绑定。 |
| `deployment` | `deployment.sql` / `sqlc/deployment` | 管理操作历史与状态流转。 |
| `gateway` | `gateway.sql` / `sqlc/gateway` | 管理 gateway_config 及其 application 部署载体查询。 |
| `route` | `route.sql` / `sqlc/route` | 管理 route。 |
| `settings` | 无 sqlc | 使用环境覆盖 adapter。 |

跨域读取通过窄接口表达，而不是让一个 Store 变成总线：auth 读取账户身份与授权结果；project 读取 user；交付领域复用 project-read；gateway 读取 application version / service / deployment。数据库事务可以在应用服务层编排，但不改变实体归属。

## 8. 已确认的纠正

| 项目 | 当前问题 | 目标纠正 |
|---|---|---|
| build stage | `build_stage` 名称把可复用流水线阶段误限为构建阶段。 | 统一更名为 `pipeline_stage`；Proto、sqlc、目标表和文档使用新名称。对已有数据库采用保留数据的 rename migration。 |
| stage run | `stage_run` 未表达其从属的一次流水线执行。 | 统一更名为 `pipeline_stage_run`；归 pipeline_run。对已有数据库采用 rename migration。 |
| service 消息 | `ServiceResp*` 位于 version.proto，混淆规格与运行时。 | 迁入 service Proto，service 作为独立领域。 |
| gateway | schema 以 application_id 关联，容易被误判为 application 附属。 | gateway 保持顶层领域；gateway_config 由 gateway sqlc 切片管理，application 只提供部署载体。 |
| config_file.proto | 没有生产路由、handler、use case、model、repository、query 或表。 | 删除 Proto 源文件及对应 Go / TypeScript 生成物；部署生成的 docker-compose.yml 是 deployment 物化产物。 |
| service_config.proto | 旧 Compose 配置模型已被 version_component 取代，且无生产调用方。 | 删除 Proto 源文件及对应生成物；不建立 service_config sqlc 领域。 |
| Task 请求消息 | pipeline / deployment 命令错放在 task.proto。 | 迁至 pipeline_run 与 deployment；task 只保留通用队列契约。 |
| Traefik 消息 | Traefik 是路由控制面的只读观测，不是 application 配置。 | `traefik.proto` 归 route；无独立 Schema / sqlc 表。 |
| CI/CD 名称 | 以技术流程名作为业务包会重新形成粗粒度双桶。 | 只保留为历史路由或 migration 分组；Proto、sqlc 采用资源领域名。 |

## 9. 实施检查清单

每个领域拆分变更必须同时检查：

1. Proto 路径、package、import、消息归属和生成物路径；
2. migration 中的目标表、FK 拓扑、已有数据 rename / 兼容策略；
3. sqlc query 文件、生成包、Repository 接口和跨域读取端口；
4. application use case、HTTP handler、worker 对旧路径和旧类型的引用；
5. Go、TypeScript 生成和全量测试。

## 10. 来源

- `20260724-api-proto-domain-split-proto视角.md`
- `20260724-ci-cd-schema-domain-split-数据库视角.md`
- `20260724-repository-sqlc-domain-split-仓储视角.md`
- 当前 Proto、Schema、Repository、use case 与 workspace 实现的核对结果。
