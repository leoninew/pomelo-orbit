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
| `auth` | 认证会话与安全审计：登录、Token、OAuth、CSRF、Turnstile、改密、登录审计。 | login_history；读取 user 身份和 role 授权结果。 |
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
| `auth` | login_history | 认证审计从 user 账户表中拆出。 |
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
| `auth` | `auth.sql` / `sqlc/auth` | 管理 login_history；通过账户身份读取端口与授权读取端口读取 user / role。 |
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

## 10. 跨层物理文件组织与命名

本节定义 model、application、Repository、infrastructure、入站 adapter 和 bootstrap 的共同物理组织规则。它服务于本文的领域边界，但不要求 HTTP URL、Proto 文件、SQL query、Go 文件或 use case 一一对应。

基本原则是：**目录和 package 提供第一层语义，文件名只表达该作用域内的第二层主题。**Go 文件不是全局命名空间，也不是领域边界本身；不应要求单个文件名脱离路径后仍能完整说明其层次、领域和职责。

### 10.1 分层目录与领域文件

| 层 | 推荐组织 | 文件命名规则 |
|---|---|---|
| `model` | 保持业务模型，不依赖 HTTP、Proto、sqlc 或外部 SDK 类型。 | 在集中 `model` package 中用 `<domain>.go` 或 `<entity>.go`；禁止用 `ci.go`、`cd.go` 充当跨领域桶。 |
| `application/<domain>` | 领域 use case、DTO 和 port 分别位于 `usecase/`、`dto/`、`port/`。 | 用 `<entity>.go`、`<capability>.go` 或动作名；`service.go` 仅用于定义该领域主 `Service` 类型及其构造。 |
| `repository` | 根 package 定义领域 port；`impl/sqlc/<domain>` 隔离 sqlc 实现和 DB Row 转换。 | 根 package 用 `<domain>.go`；领域实现目录内可用 `repository.go` 表示该实现的主 Repository。 |
| `infrastructure` | 按外部系统或技术实现组织，例如 `<provider>`、`runner`、`storage`。 | 用 provider 或能力名，不以业务领域命名来掩盖外部系统适配。 |
| 入站 adapter | HTTP handler、routes、worker handler 等按 transport 和领域分组。 | `handler/<domain>/` 内使用 `handler.go`、`<entity>.go`、`<entity>_mapper.go`；`routes/<domain>.go` 只绑定 URL 与 handler。 |
| `bootstrap` | 组合根，负责创建实现并注入 port。 | 使用 transport 或组合目标命名，例如 `http.go`、`worker.go`；不承载业务规则。 |

在一个扁平的多领域目录中，选择 `<domain>.go`，不选择 `<domain>_<layer>.go`：父目录已经给出层语义，后缀重复信息。例如 `repository/<domain>.go` 优于 `<domain>_repository.go`，`model/<domain>.go` 优于 `<domain>_model.go`。当领域拥有多个实体或行为时，先进入 `<layer>/<domain>/`，再以 `<entity>.go` 或 `<action>.go` 划分。

结构性文件名如 `handler.go`、`service.go`、`repository.go` 只适合位于已同时限定层和领域的 package 中，且文件确实定义该 package 的主类型、构造或共享行为；它们不应在承载多个领域的扁平目录中替代领域名。若领域名与层名相同且路径仍容易误读，可以使用真实业务限定词，例如 `vcs_repository.go`，而不是机械追加 `_handler`、`_service` 或 `_repository` 后缀。

```text
internal/
  model/
    <domain>.go
  application/<domain>/
    dto/<entity>.go
    port/<capability>.go
    usecase/service.go
    usecase/<entity>.go
  repository/
    <domain>.go
    impl/sqlc/<domain>/repository.go
  api/http/
    handler/<domain>/handler.go
    handler/<domain>/<entity>.go
    routes/<domain>.go
  bootstrap/
    http.go
    worker.go
```

同名文件由路径和 package 消除歧义。例如 `routes/route.go` 是 `route` 领域的路由声明，`handler/route/route.go` 是该领域 endpoint 实现，`model/route.go` 是领域模型；不应仅因同名而改为冗余的 `route_handler.go`。

### 10.2 聚合、注册与装配文件

聚合式文件不能使用一个业务领域名，也不应假装是领域。按实际职责命名：

| 名称 | 适用职责 | 不应承担的职责 |
|---|---|---|
| `registry.go` | 定义 registry、静态映射或集合，并集中协调已注册项。HTTP Router 聚合、worker handler registry 等均可使用。 | 领域业务规则、具体 endpoint 或 task 实现。 |
| `register.go` | 仅提供 `registerAll` 或同类总注册函数。 | 依赖创建、进程启动和业务规则。 |
| `wire.go` | 组合根中的依赖构造、接口实现注入或生成的 Wire 装配代码。 | 路由声明、任务注册和业务规则。 |
| `server.go` | HTTP server 生命周期、fallback/static serving 等 server 职责。 | application service、Repository 或外部系统实现的实例化。 |
| `index.go` | 仅用于确有必要的生成代码或兼容性入口。Go 不赋予它特殊语义。 | 作为无职责的包概览或杂项收集点。 |

现有 `routes.go` 同时定义 `Router` 并聚合 `register<Domain>` 调用，语义上符合 `registry.go`。由于其 package 已命名为 `routes`，继续使用 `routes.go` 也可接受；若后续拆出纯总注册函数，采用 `registry.go` + `register.go` 会更精确。依赖实例化应保留在 `bootstrap`，而不应迁入 `routes`、领域 use case 或 Repository。

### 10.3 禁止的模糊名称

不要创建 `common.go` 作为跨领域代码的默认落点。它没有可验证的职责边界，容易演变为杂物箱。共享代码必须根据实际技术职责进入明确的包或文件，例如 `response`、`binding`、`codec`、`middleware`、`security`、`errors` 或 `<capability>.go`。业务领域之间的读取、写入、事件或外部能力共享，应以 application port 表达，不应放入 `common`。

`common` package 如确有必要，只能容纳无业务语义、向内稳定且没有上层依赖的基础能力；它不能依赖 domain model、Proto、sqlc、HTTP 或外部 provider。领域专属的辅助函数必须留在所属领域 package 中。

`authz` 这类跨 handler 的请求认证/授权辅助能力也不是业务领域；应位于明确的 HTTP 技术职责包，例如 `security`，若实现为 Gin middleware 则位于 `middleware`。

### 10.4 协议 DTO 与 application 用例契约

Proto 生成类型是 transport contract，不是 application DTO。它们属于 `internal/gen/proto/...`，只能由 HTTP、gRPC、worker task 等入站/出站 adapter 导入；application、model、Repository 和 infrastructure 不得依赖生成的 Proto 类型。这样 API 字段兼容、`oneof`、wrapper、field presence、JSON 表示和分页编码不会渗透到业务用例。

application 可以且应当定义自己的用例输入/输出类型，但不应机械逐字段复制所有 Proto message。为避免“DTO”一词混淆，按契约角色组织：

| 类型 | 所属层 | 职责 |
|---|---|---|
| `*v1.<Entity>CreateReq`、`*v1.<Entity>Resp` | Proto / transport | 版本化的对外请求、响应和任务消息格式。 |
| `<Entity>CreateInput`、`<Entity>UpdateInput` | application command | 用例需要的业务输入，可由 HTTP、worker、CLI 等多个 adapter 共享。 |
| `<Entity>ListQuery` | application query | 用例需要的查询条件，不包含 HTTP URL、Proto wrapper 等协议细节。 |
| `<Entity>View` | application view | 跨聚合投影、组合视图或脱敏后的用例输出；简单聚合无需强制定义。 |
| `model.<Entity>` | model | 聚合的业务状态，不承载协议或持久化细节。 |
| sqlc Row / generated query 参数 | Repository 实现 | 持久化实现细节，仅限 `impl/sqlc`。 |

标准调用方向为：

```text
Proto request / worker task message
  -> transport mapper
  -> application Command or Query
  -> use case
  -> model or application View
  -> transport mapper
  -> Proto response
```

mapper 是刻意保留的反腐层，而非应消除的重复：它执行协议解码、路径/Query/Body 合并、字段形状转换和协议默认值；application use case 是业务校验、授权、状态转换和跨领域编排的唯一位置。不得在两层重复业务规则。简单聚合读取或写入可直接返回 `model.<Entity>`，再由 adapter 映射为 Proto response；只有投影、组合或脱敏需要时才定义 application View。

仅有单一 gRPC/HTTP 入口、没有 worker/CLI、业务简单且 API 生命周期完全等同于用例时，直接把 Proto request 传入 service 是可接受的简化。它不是本项目的默认规则：本项目同时存在 HTTP 与 worker task 入站方式，并要求长期 API 兼容和领域拆分。因此每个领域应拥有自身 application Command/Query/View，旧 `ci/dto`、`cd/dto` 不得继续作为跨领域 DTO 桶。

### 10.5 外部 API 与 SPA 地址

HTTP path 和 SPA 页面地址同样必须表达资源领域；`ci`、`cd` 是历史技术流程分组，不得继续作为 `/api` 下或浏览器页面地址中的父 segment。`/api` 是 transport 前缀，不是领域；`pipeline`、`pipeline_run`、`application`、`deployment` 等才是资源能力边界。

本次地址迁移采用硬切换：不注册旧 `/api/ci/**`、`/api/cd/**` 后端别名，不保留 Vue Router 旧 route、redirect 或 rewrite，也不保留导航入口。旧 API 必须未注册并返回 404；旧 SPA path 不应解析到任何领域页面。数据库 migration 标签和已持久化 task value 的兼容策略不构成保留 HTTP 或 SPA URL 的理由。

路径只因领域归属而调整，不顺带统一单复数、引入 API version 或改变 DTO。API 与 SPA 不要求逐字相同：前者保持现有 transport 资源路径，后者可以用既有 collection / detail 约定；两者都不得重建 `ci` / `cd` 粗粒度父层。

## 11. 当前实施评估（2026-07-24）

本节核对当前 workspace 的实现与本文领域地图。评估时工作树正处于迁移过程中，存在未提交变更；因此这里记录的是当前实现状态，而不是对目标领域边界的修订。

结论：Proto、sqlc、Repository、HTTP handler、worker、bootstrap 和 application 已完成领域切换。生产代码不再依赖 `internal/application/ci` 或 `internal/application/cd` 业务包，application family 只保留 application / version 规格。任务 06 已完成：前端状态键、运维数据路径、规则 package、文案和一次性迁移脚本均已完成硬切换，不保留历史 `ci` / `cd` 兼容面。

### 11.1 已符合的组织实践

- HTTP handler 已按 `internal/api/http/handler/<domain>/` 内聚，领域内以 `handler.go` 放置构造和共享错误处理，以实体或子资源文件放置 endpoint 与 mapper。
- `internal/api/http/routes/<domain>.go` 分别声明各领域路由；`routes.go` 作为复数的聚合和注册入口，不与 `route` 领域混淆，且没有引入无边界的 `common.go`。
- 旧 `/api/ci`、`/api/cd` URL 与 `/ci/**`、`/cd/**` SPA 页面地址不保留兼容面：不注册 alias、redirect 或 rewrite；历史地址不改变内部用例、DTO、端口和 Repository 的逻辑归属。
- `pipeline_stage`、`pipeline_stage_run` 已成为 Proto、SQL、Repository 和 application 层的业务主名称；`application/gateway`、`application/deployment`、`application/service` 均已具备独立 DTO、port、use case 和定向测试。

### 11.2 收尾任务状态

| 任务 | 状态 | 已核对结果 | 剩余收口 |
|---|---|---|---|
| 01 Pipeline 术语 | 完成 | `PipelineStage`、`PipelineStageRun` 已取代旧主模型与 DTO；模型已拆为 credential、repository、pipeline、pipeline_run 等领域文件，变量规则 package 已命名为 `pipelinevariable`。 | 无。 |
| 02 Gateway use case | 完成 | gateway 已拥有 DTO、port、use case 和测试；仅通过 port 协调 deployment，不依赖 HTTP、Proto、sqlc implementation 或 infrastructure。 | 无。 |
| 03 Deployment use case | 完成 | command、execution、DTO、port 和 worker 已迁入 deployment；Gateway 规则经 `DeploymentCoordinator` port 调用，不复制 Gateway 编译规则，且 deployment service 不再携带未使用的 `config.Config`。 | 无。 |
| 04 Service use case | 完成 | service 已拥有 project list、单项读取、application-scoped list、primary service 与 target resolver，生产实现无越界 import。 | 无。 |
| 05 Application Family 总切换 | 完成 | HTTP handler、worker、queue dispatch 和 bootstrap 均注入 gateway、deployment、service 的独立 use case；application 仅保留 application / version 规格。 | 无。 |
| 06 历史命名与边界审计 | 完成 | 旧内部业务包、模型双桶、`authz`、任务类型、HTTP / SPA route、Proto 向内层穿透、前端状态键、运维路径和一次性迁移脚本均已清理。 | 无。 |

### 11.3 已完成的收口

| 项目 | 完成结果 |
|---|---|
| 运维数据路径 | `scripts/cert.py` 使用调用方传入的证书目录；备份脚本仅排除调用方显式传入的 workspace 路径。 |
| 前端状态 | `web/src/constants/application.ts` 提供 `applicationVersionsApplicationIdKey`，键为 `pomelo_orbit_application_versions_application_id:<projectId>`；版本页只读写新键。 |
| Pipeline 变量规则 | `civariable` 已迁为 `pipelinevariable`，pipeline、pipeline_run 和 repository use case 均使用新 import 与 package 名。 |
| Application 最小依赖 | deployment command / execution service 及测试辅助构造器均已移除未使用的 `config.Config` 依赖。 |
| 文案与迁移工具 | deployment 注释、repository credential 注释和首页文案已采用领域语言；九个引用旧树的一次性迁移脚本已删除。 |

### 11.4 验证状态

- `go vet ./...` 与 `go test ./...` 已通过；HTTP、worker、queue dispatch、bootstrap 和 e2e 均完成当前工作树构建。
- `yarn --cwd web typecheck`、`yarn --cwd web test`、`yarn --cwd web lint` 和 `yarn --cwd web build` 已通过；lint 为零 error、零 warning。生成的 `web/src/gen/proto/**` 由 ESLint 精确忽略，手写 `src` 代码仍在 lint 范围内。
- production HTTP / SPA path、task type、旧 Go 业务 package、Proto 向 application / model / Repository / infrastructure 的 import、`common.go`、旧 local storage key、旧运维目录及本节原列的历史命名均已通过静态扫描；历史备份文件不作为运行时兼容实现。
- 领域拆分完成的最低验证门槛是：全量 Go 测试通过；HTTP handler、worker、bootstrap 和 use case 均调用正确的领域服务；生产内部业务包、前端状态键和运维数据路径不再使用 `ci` / `cd` 双桶；Gateway 规则仅由 gateway 领域拥有；历史 HTTP / SPA path、task type、数据目录和协议字段不保留兼容层。

## 12. 来源

- `20260724-api-proto-domain-split-proto视角.md`
- `20260724-ci-cd-schema-domain-split-数据库视角.md`
- `20260724-repository-sqlc-domain-split-仓储视角.md`
- 当前 Proto、Schema、Repository、use case 与 workspace 实现的核对结果。
