# CI / CD Schema 领域拆分

日期：2026-07-24

范围：`sql/migration/*/{ci,cd}_schema` 表集合。不含业务背景、产品叙事与验收条款。

## 1. 表清单

### CI（10）

| 表 | 引用（FK 出边） |
|---|---|
| `credential` | `project` |
| `pipeline_template` | `project` |
| `build_stage` | `project` |
| `pipeline_template_stage` | `pipeline_template`, `build_stage` |
| `pipeline_snapshot` | `pipeline_template`, `project` |
| `repository` | `credential`, `project` |
| `repository_webhook` | `repository`, `pipeline_template` |
| `pipeline_run` | `repository`, `pipeline_snapshot`, `pipeline_run`(retry), `project` |
| `stage_run` | `pipeline_run` |
| `artifact` | `pipeline_run`, `project` |

### CD（9）

| 表 | 引用（FK 出边） |
|---|---|
| `application` | `project` |
| `version` | `application`, `version`(created_from) |
| `version_component` | `version` |
| `version_expose` | `version` |
| `environment` | `project` |
| `gateway_config` | `application` |
| `service` | `application`, `environment`, `version`×2 |
| `deployment` | `version`, `service`, `environment`, `deployment`(rollback), `project` |
| `route` | `project` |

## 2. 依赖图（仅 schema 内）

### CI

```
credential
pipeline_template ──┬── pipeline_template_stage ── build_stage
                    └── pipeline_snapshot
repository ── credential
repository_webhook ── repository
                   └── pipeline_template
pipeline_run ── repository
             └── pipeline_snapshot
stage_run ── pipeline_run
artifact  ── pipeline_run
```

拓扑序（创建）：

1. `credential` / `pipeline_template` / `build_stage`（可并行）
2. `pipeline_template_stage` / `pipeline_snapshot`
3. `repository`
4. `repository_webhook`
5. `pipeline_run`
6. `stage_run` / `artifact`

### CD

```
application ── gateway_config
            └── version ── version_component
                        └── version_expose
environment          (仅 project)
service ── application + environment + version
deployment ── service / version / environment
route                (仅 project)
```

拓扑序（创建）：

1. `application` / `environment` / `route`（可并行）
2. `gateway_config` / `version`
3. `version_component` / `version_expose`
4. `service`
5. `deployment`

## 3. 领域切分原则

1. **聚合根 / 生命周期**：定义态与执行态分离；规格与运行时绑定分离。
2. **FK 跨文件单向向下**：子表、junction 跟父聚合；禁止环。
3. **1:1 扩展表跟主实体**：例 `gateway_config` → `application`。
4. **弱关联独立实体单独成域**：例 `credential`、`route`（仅 `project`）。
5. **单文件表数**：优先 1～5 表；避免一表一文件（down 噪声）与全库一文件（diff 难）。

## 4. 推荐领域

### CI（4）

| 域 | 表 | 说明 |
|---|---|---|
| `ci_credential` | `credential` | 密钥根；被 repository 引用 |
| `ci_pipeline` | `pipeline_template`, `build_stage`, `pipeline_template_stage`, `pipeline_snapshot` | 定义目录 + 不可变快照 |
| `ci_repository` | `repository`, `repository_webhook` | 源绑定；webhook 跨依赖 template |
| `ci_run` | `pipeline_run`, `stage_run`, `artifact` | 执行态 + 产物 |

域序：`credential` → `pipeline` → `repository` → `run`

### CD（6）

| 域 | 表 | 说明 |
|---|---|---|
| `cd_application` | `application`, `gateway_config` | 应用根 + 1:1 扩展 |
| `cd_version` | `version`, `version_component`, `version_expose` | 版本规格一体 |
| `cd_environment` | `environment` | project 级目标；与 application 无 FK |
| `cd_service` | `service` | app × env × version 运行时绑定 |
| `cd_deployment` | `deployment` | 操作历史 |
| `cd_route` | `route` | project 级独立实体；无 FK 到 app/version |

域序：`application` ∥ `environment` ∥ `route` → `version` → `service` → `deployment`  
（`route` 可置于 environment 邻近或链尾，不影响其余域）

## 5. 备选粒度

| 方案 | CI | CD | 备注 |
|---|---|---|---|
| 现状 | 1 文件 | 1 文件 | 粗 |
| **推荐** | 4 | 6 | 对齐 use case / FK |
| 略合 | 4 | 5 | `service`+`deployment` → `cd_runtime` |
| 极细 | ≈表数 | ≈表数 | 过碎 |

## 6. 明确不采用的切法

| 切法 | 原因 |
|---|---|
| `pipeline_snapshot` 并入 `ci_run` | snapshot 属定义版本化，非 run 子表 |
| `artifact` 单独文件 | 表过少，收益低 |
| `credential` 并入 `repository` | 密钥是可复用能力，边界应独立 |
| `version_expose` 与 `route` 同域 | 规格暴露 vs 独立 L7 路由，无表级 FK |
| `environment` 并入 `cd_application` | 仅 project 依赖，可先于 application 存在 |
| `gateway_config` 独立文件 | 1:1 扩展，跟 application 即可 |

## 7. 建议文件名（续历史版本号）

在既有 `core_infra` / `auth_schema` 之后示例：

```
000016_ci_credential
000017_ci_pipeline
000018_ci_repository
000019_ci_run
000020_cd_application
000021_cd_version
000022_cd_environment
000023_cd_service
000024_cd_deployment
000025_cd_route
000026_seed_system
000027_seed_ci
```

若采用「略合」：`000023_cd_runtime`（含 service+deployment），其后 route/seed 顺延减 1。

## 8. 与代码包的对应（校验用）

| 域 | 主要代码触点 |
|---|---|
| `ci_credential` | `application/ci/.../credential` |
| `ci_pipeline` | `template`, `build_stage`, `snapshot` |
| `ci_repository` | `repository` |
| `ci_run` | `pipeline_run`, `execution`, `artifact` |
| `cd_application` | application / gateway use case |
| `cd_version` | `version` |
| `cd_environment` | `environment` |
| `cd_service` | `service` |
| `cd_deployment` | `deployment_execution` |
| `cd_route` | `route` |

非约束条件，仅用于核对切分是否与模块边界一致。
