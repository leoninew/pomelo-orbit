# API Proto 领域拆分

日期：2026-07-24

范围：`proto/orbit/v1` 契约源布局与领域边界。不含业务背景、产品叙事、验收条款与实现步骤。

目标路径形态：`proto/<product>/v1/<领域>/<实体>.proto`（本仓库 product = `orbit`）。

## 1. 输入清单

### 1.1 现有扁平实体文件（25）

| 文件 | message 数 | 现有 import 入边 |
|---|---:|---|
| `common.proto` | 6 | ← `repository`, `template`, `snapshot`, `pipeline_run` |
| `auth.proto` | 10 | — |
| `user.proto` | 9 | — |
| `role.proto` | 6 | — |
| `project.proto` | 7 | — |
| `settings.proto` | 4 | — |
| `task.proto` | 6 | — |
| `repository.proto` | 4 | → `common` |
| `webhook.proto` | 5 | — |
| `credential.proto` | 7 | — |
| `build_stage.proto` | 8 | — |
| `template.proto` | 10 | → `build_stage`, `common` |
| `snapshot.proto` | 2 | → `build_stage`, `common` |
| `pipeline_run.proto` | 8 | → `artifact`, `common` |
| `artifact.proto` | 2 | — |
| `application.proto` | 10 | — |
| `application_bundle.proto` | 2 | → `version` |
| `version.proto` | 14 | — |
| `config_file.proto` | 5 | — |
| `service_config.proto` | 7 | — |
| `deployment.proto` | 5 | — |
| `environment.proto` | 4 | — |
| `gateway.proto` | 5 | — |
| `route.proto` | 12 | — |
| `traefik.proto` | 3 | — |

### 1.2 现有 proto import 图

```text
common
  ↑
  ├── repository
  ├── template ──→ build_stage
  ├── snapshot ──→ build_stage
  └── pipeline_run ──→ artifact

application_bundle ──→ version
```

其余文件无交叉 import。

### 1.3 划分约束

1. 领域名不用 `ci` / `cd`。
2. 目录名 = protobuf package 后缀建议：`orbit.v1.<domain>`。
3. 实体文件尽量保持现有 basename；message 搬家仅在语义错放时记录。
4. 跨领域依赖应少、宜单向；共享类型进 `common`。
5. 本分析只定契约领域，不定义 application / repository 包结构。

## 2. 语义簇（划分依据）

| 簇 | 实体 | 判定 |
|---|---|---|
| 共享 | `common` | 被多文件 import；无独立资源生命周期 |
| 身份会话 | `auth` | 登录/会话/令牌；非用户主数据 CRUD |
| 账户 | `user` | 用户 CRUD / enable-disable / 角色绑定请求 |
| 授权 | `role` + permission | 角色与权限清单 |
| 租户边界 | `project` + member | 资源归属根 |
| 系统配置 | `settings` | 全局配置项 |
| 后台任务 | `task` | 队列任务创建与查询；payload 跨能力但 API 面独立 |
| 代码源 | `repository` | 仓库资源 |
| 钩子 | `webhook` | 挂在 repository 下；可并入 repository 域 |
| 密钥 | `credential` | 独立 CRUD + import/export |
| 构建步骤库 | `build_stage` | 可复用阶段定义；被 template/snapshot 引用 |
| 流水线定义 | `template`, `snapshot` | 编排与不可变快照 |
| 流水线执行 | `pipeline_run` | 一次 run / stage log |
| 构建产物 | `artifact` | run 产出列表；可并入 pipeline_run 域 |
| 应用定义 | `application`, `application_bundle`, `config_file`, `service_config` | 应用及其配置/导入导出 |
| 版本规格 | `version` 中 component/expose/preview 等 | 版本规格；文件最大 |
| 运行时绑定 | `version.proto` 内 `ServiceResp*` | 语义是 runtime service，不是 version 字段 |
| 发布记录 | `deployment` | 部署动作与日志 |
| 环境 | `environment` | 部署目标环境 |
| 网关 | `gateway` | 独立 CRUD + exposure |
| 平台路由 | `route` | 项目路由与证书操作 |
| 入口观测 | `traefik` | Traefik router/config 只读面；宜挂 route |

## 3. 不应作为契约领域名

| 名称 | 原因 |
|---|---|
| `ci` | URL/前端导航遗留；跨 repository…artifact 多个聚合 |
| `cd` | 同上；跨 application…route 多个聚合 |
| `dto` / `api` | 技术分层名，非业务能力 |
| `delivery` / `runtime` | ci/cd 换皮，粒度仍过粗 |

## 4. 局部合并 / 拆出判定

| 议题 | 选项 | 分析结论 |
|---|---|---|
| `webhook` | 独立域 vs ⊂ `repository` | 资源从属 repository；API 嵌套在 repository 下。**默认可 ⊂ repository** |
| `artifact` | 独立域 vs ⊂ `pipeline_run` | message 少且被 run import。**默认可 ⊂ pipeline_run** |
| `snapshot` | 独立域 vs ⊂ `pipeline`/`template` | 只读附属 template。**默认可与 template 同域 `pipeline`** |
| `version` | 独立域 vs ⊂ `application` | 文件大、生命周期相对独立；目录独立更清晰，也可作 `application/version.proto` |
| `ServiceResp*` | 留在 `version` vs 迁出 | **语义错放，应迁出**到 `service` 实体文件（域可独立或暂放 `application`） |
| `config_file` / `service_config` / `bundle` | 独立 vs ⊂ `application` | 仅服务 application 配置面。**⊂ application** |
| `traefik` | 独立域 vs ⊂ `route` / `gateway` | 观测 API 与 route handler 相邻。**⊂ route 作为实体文件** |
| `gateway` | ⊂ `application` vs 独立 | 独立路由与 exposure 模型。**独立域** |
| `DeploymentActionResp` | 在 `application` vs `deployment` | 现为 deploy 即时回执；可留 application 或迁 deployment，非阻塞 |
| `user` / `role` | 合并 `identity` vs 分开 | 现网资源与 handler 已分。**分开** |
| `task` | 打散到各域 vs 独立 | 统一 background 面。**独立** |

## 5. 方案对照

### 5.1 方案 A：细领域

领域目录（约 18–20）：

```text
common
auth
user
role
project
settings
task
repository
webhook
credential
build_stage
pipeline          # template + snapshot
pipeline_run
artifact
application       # application + bundle + config_file + service_config
version
service           # 自 version 迁出 Service*
deployment
environment
gateway
route             # route + traefik
```

- 优点：路径与聚合一一对应，后续模块对齐成本低。
- 缺点：目录多；handler 多包 import。

### 5.2 方案 B：中等（默认推荐）

领域目录（17 个实体落点 / 约 14–16 个领域目录）：

```text
common
auth
user
role
project
settings
task
repository        # repository + webhook
credential
build_stage
pipeline          # template + snapshot
pipeline_run      # pipeline_run + artifact
application       # application + bundle + version + config_file + service_config
                  # + service.proto（runtime，从 version 迁出 message）
deployment
environment
gateway
route             # route + traefik
```

相对方案 A 的合并：

| 合并 | 内容 |
|---|---|
| `repository` | + `webhook` |
| `pipeline_run` | + `artifact` |
| `application` | + `version` + runtime `service` 文件 |
| `route` | + `traefik` |

- 优点：目录数可控；仍避开 ci/cd；依赖图清晰。
- 缺点：`application` 目录文件偏多（用多实体文件消化）。

### 5.3 方案 C：粗领域

```text
common
identity          # auth + user + role
project
settings
task
delivery          # repository…artifact
runtime           # application…route
```

- 不推荐：粒度回到双桶，领域路径信息量低。

## 6. 推荐落表（方案 B）

| 领域目录 | package 建议 | 实体文件 |
|---|---|---|
| `common` | `orbit.v1.common` | `common.proto` |
| `auth` | `orbit.v1.auth` | `auth.proto` |
| `user` | `orbit.v1.user` | `user.proto` |
| `role` | `orbit.v1.role` | `role.proto` |
| `project` | `orbit.v1.project` | `project.proto` |
| `settings` | `orbit.v1.settings` | `settings.proto` |
| `task` | `orbit.v1.task` | `task.proto` |
| `repository` | `orbit.v1.repository` | `repository.proto`, `webhook.proto` |
| `credential` | `orbit.v1.credential` | `credential.proto` |
| `build_stage` | `orbit.v1.build_stage` | `build_stage.proto` |
| `pipeline` | `orbit.v1.pipeline` | `template.proto`, `snapshot.proto` |
| `pipeline_run` | `orbit.v1.pipeline_run` | `pipeline_run.proto`, `artifact.proto` |
| `application` | `orbit.v1.application` | `application.proto`, `application_bundle.proto`, `version.proto`, `config_file.proto`, `service_config.proto`, `service.proto` |
| `deployment` | `orbit.v1.deployment` | `deployment.proto` |
| `environment` | `orbit.v1.environment` | `environment.proto` |
| `gateway` | `orbit.v1.gateway` | `gateway.proto` |
| `route` | `orbit.v1.route` | `route.proto`, `traefik.proto` |

### 6.1 目标树

```text
proto/orbit/v1/
  common/common.proto
  auth/auth.proto
  user/user.proto
  role/role.proto
  project/project.proto
  settings/settings.proto
  task/task.proto
  repository/
    repository.proto
    webhook.proto
  credential/credential.proto
  build_stage/build_stage.proto
  pipeline/
    template.proto
    snapshot.proto
  pipeline_run/
    pipeline_run.proto
    artifact.proto
  application/
    application.proto
    application_bundle.proto
    version.proto
    config_file.proto
    service_config.proto
    service.proto          # ServiceResp / ServiceListResp / ServicePaginatedResp
  deployment/deployment.proto
  environment/environment.proto
  gateway/gateway.proto
  route/
    route.proto
    traefik.proto
```

### 6.2 跨域 import 目标形态

```text
orbit/v1/common/common.proto
  ↑
  ├── repository/repository.proto
  ├── pipeline/template.proto ──→ build_stage/build_stage.proto
  ├── pipeline/snapshot.proto ──→ build_stage/build_stage.proto
  └── pipeline_run/pipeline_run.proto ──→ pipeline_run/artifact.proto

application/application_bundle.proto ──→ application/version.proto
```

同域内 import 使用同目录相对路径规则（buf：`orbit/v1/<domain>/<entity>.proto`）。

## 7. 方案 A 增量（若选细领域）

在方案 B 上拆出：

| 从 | 到 |
|---|---|
| `repository/webhook.proto` | `webhook/webhook.proto` |
| `pipeline_run/artifact.proto` | `artifact/artifact.proto` |
| `application/version.proto` | `version/version.proto` |
| `application/service.proto` | `service/service.proto` |

其余不变。

## 8. 与 HTTP 路由前缀的关系

| 现状 path 前缀 | 契约领域 |
|---|---|
| `/api/auth` | `auth` |
| `/api/user` | `user` |
| `/api/role` | `role` |
| `/api/project` | `project` |
| `/api/settings` | `settings` |
| `/api/background` | `task` |
| `/api/ci/repository`, `/api/ci/webhook` | `repository` |
| `/api/ci/credential` | `credential` |
| `/api/ci/build-stage` | `build_stage` |
| `/api/ci/template`, `/api/ci/snapshot` | `pipeline` |
| `/api/ci/run`, `/api/ci/artifact` | `pipeline_run` |
| `/api/cd/application`, `/api/cd/version` | `application` |
| `/api/cd/service` | `application`（`service.proto`）或细方案 `service` |
| `/api/cd/deployment` | `deployment` |
| `/api/cd/environment` | `environment` |
| `/api/cd/gateway` | `gateway` |
| `/api/cd/route`, `/api/cd/traefik-route` | `route` |

说明：path 中的 `ci`/`cd` 不映射为领域名；领域以资源能力为准。

## 9. 生成物路径推论

在 `paths=source_relative` 与现有 out 根不变时：

| 侧 | 路径 |
|---|---|
| Go | `internal/gen/proto/orbit/v1/<domain>/*.pb.go` |
| TS | `web/src/gen/proto/orbit/v1/<domain>/*.ts` |

Go import 由扁平 `orbit/v1` 变为按领域多包（例如 `.../orbit/v1/application`、`.../orbit/v1/route`）。

## 10. 已知错放与最小纠正

| 项 | 现状 | 纠正 |
|---|---|---|
| `ServiceResp` / `ServiceListResp` / `ServicePaginatedResp` | 位于 `version.proto` | 迁至 `service.proto`（域按所选方案） |
| `Traefik*Resp` | proto 已有；application dto 手写重复 | 契约以 proto 为准；手写重复删除（实现阶段） |
| `ArtifactConfig*` | 在 `build_stage.proto` | 保留（构建配置，不是产物 `artifact`） |

## 11. 开放点（仅结构）

1. 采用方案 A 或 B（本文默认 B）。
2. `version` / `service` 是否在 B 中进一步提升为顶层域（接近 A）。
3. `pipeline` 目录内文件名保留 `template.proto` 或改名为 `pipeline_template.proto`（package 已是 `pipeline` 时 basename 可保留）。
4. `DeploymentActionResp` 归属 `application` 或 `deployment`。

## 12. 结论摘要

1. 契约领域应按资源聚合划分，禁止 `ci`/`cd` 作为领域目录。
2. 现有 25 个扁平文件可收入 **方案 B：约 16 个领域目录**；或 **方案 A：约 18–20 个**。
3. 依赖边少，领域拆分的主要工作是目录/package 迁移与生成路径变化，而非重解耦合。
4. 唯一建议的 message 级纠正：`Service*` 离开 `version`。
5. `traefik` 作为 `route` 域实体文件；`gateway` 独立域；`webhook`/`artifact` 在 B 中并入宿主域。
