# SQL 迁移合并：领域对齐基线 — Spec
最后修改时间: 2026-07-24 11:23:35

Review status: Accepted

## Requirement basis

- Requirement: `docs/requirement/20260724-sql-migration-squash.md`（Accepted）
- Domain baseline: `docs/analyze/20260724-domain-split-consensus-共识.md`
- 现状：粗拆 `000014`–`000019`（`ci_schema`/`cd_schema` 等）待替换

## Overview

仅重写 **`sql/migration/{sqlite,mysql}`**：

1. 废弃旧链与粗 CI/CD 双桶文件名；
2. 自 `000014` 按共识**每领域一文件**重建终态 schema + down；
3. migration 内表名：`pipeline_stage`、`pipeline_stage_run`；
4. `seed_system` + `seed_pipeline`；无 application/route 演示数据；
5. 双驱动同构。

**不改** Go / Proto / 前端 / 测试。migrate 运行时不变。

## Design decisions

### D1. 物理文件与版本号（30 的含义）

本轮 **17** 个连续版本，起点 `000014`：

`末版本 = 14 + (17 - 1) = 30`

| 含义 | 说明 |
|---|---|
| **不是**上限 | 项目从不限制「版本不能超过 30」 |
| **是**本轮末版本 | 本轮全部 up 成功后，`schema_migrations.version` 预期为 **30** |
| 之后 | 新功能迁移从 **`000031`** 继续递增 |

| Version | stem | Domain | Tables / content |
|---|---|---|---|
| 000014 | `task` | task | `background_task` |
| 000015 | `role` | role | `role`, `permission`, `role_permission` |
| 000016 | `user` | user | `user`, `user_role` |
| 000017 | `auth` | auth | `login_history`, `login_attempt` |
| 000018 | `project` | project | `project`, `project_member` |
| 000019 | `credential` | credential | `credential` |
| 000020 | `pipeline` | pipeline | `pipeline_template`, `pipeline_stage`, `pipeline_template_stage`, `pipeline_snapshot` |
| 000021 | `repository` | repository | `repository`, `repository_webhook` |
| 000022 | `pipeline_run` | pipeline_run | `pipeline_run`, `pipeline_stage_run`, `artifact` |
| 000023 | `application` | application | `application`, `version`, `version_component`, `version_expose` |
| 000024 | `environment` | environment | `environment` |
| 000025 | `gateway` | gateway | `gateway_config`（独立文件，不并入 application） |
| 000026 | `service` | service | `service` |
| 000027 | `deployment` | deployment | `deployment` |
| 000028 | `route` | route | `route` |
| 000029 | `seed_system` | seed | 系统种子 |
| 000030 | `seed_pipeline` | seed | pipeline 演示种子 |

文件头注释：逻辑领域、表列表、共识引用。

### D2. 表名（仅 migration）

| 旧 | 新 | 领域 |
|---|---|---|
| `build_stage` | `pipeline_stage` | pipeline |
| `stage_run` | `pipeline_stage_run` | pipeline_run |

本轮 **只**在 migration/seed SQL 中使用新名。Go / Proto / 测试中的旧标识**本轮不动**。

### D3. Schema 内容

以现粗拆终态 SQL 为列集合真相源，按领域剪切；应用 D2；删除旧中间表定义。不增业务列。

### D4. 种子

- **000029 seed_system**：admin、role、permission、default project、系统 environment（对齐现 seed_system 语义）。
- **000030 seed_pipeline**：原 CI demo，表名改为 `pipeline_stage` 等；无 application/route 演示。

### D5. Down

每版本对称 `.down.sql`；seed DELETE 在前；schema 逆序 DROP。

### D6. 范围边界（用户确认）

| 在范围 | 不在范围 |
|---|---|
| `sql/migration/sqlite/**` | 一切 `.go` / Proto / web |
| `sql/migration/mysql/**` | `migration_test.go` 等期望 version 更新 |
| 过程文档（requirement/spec/plan） | repository 内嵌 SQL 旧表名修复 |

### D7. 双驱动

同 version、同 stem；方言差异按项目惯例；列与 FK 语义一致。

## Affected components

| 路径 | 变更 |
|---|---|
| `sql/migration/sqlite/*` | 替换为 000014–000030 领域文件 + down |
| `sql/migration/mysql/*` | 同上 |

无其他代码树变更。

## Interfaces

- 对外 API：不变。
- 对内：DB 表名与 migration 版本集合变化；应用代码对齐**不在本 Spec 交付范围**。

## Schema notes（FK 序）

与共识 §6 一致；实现严格按 D1 序号创建，避免 FK 失败。`gateway` 在 `application` 之后；`service` 在 application/environment/version 之后；`route` 仅依赖 project，放在 000028。

## Technical questions

无。用户已确认：

1. 领域一文件；30 = 本轮末版本算术结果，非上限。
2. 本轮只改迁移脚本。
3. gateway 独立文件。

## Risks

| 风险 | 处理 |
|---|---|
| 迁移更名后 Go 测试失败 | **预期**；后续代码对齐任务处理；本轮不扩 scope |
| seed 与集成测假设不一致 | migration 尽量保留现 seed 语义；不修测试 |
| 误改非 migration 文件 | 实现时 diff 门禁仅 `sql/migration/**` |

## Alternatives considered

| 方案 | 结论 |
|---|---|
| 本轮连同 Go 表名引用一起改 | 否；用户限定只改迁移脚本 |
| 本轮暂不更名表，仅拆文件 | 否；共识更名 + 重建时机；更名落在 migration 内 |
| gateway 并 application 文件 | 否；用户确认独立 |
| 把 30 当成永久版本上限 | 否；仅本轮末版本 |

## Implementation outline（供 Plan）

1. 删除 sqlite/mysql 现有 `000014`–`000019` 粗拆。
2. 按 D1 写 000014–000028 schema up（sqlite 再 mysql）。
3. 写 000029–000030 seed up（新表名）。
4. 补全全部 down。
5. 自检：双目录文件清单对称；无 `build_stage`/`stage_run` 表定义；无 `ci_schema`/`cd_schema` 文件名。
6. **不**改 Go；**不**跑「修到绿」类全量测试作为本轮完成条件（可选：仅手工确认 migrate 工具能解析文件——若环境方便）。

## User review notes

- 用户反馈已吸收：澄清 version 30；范围仅 migration；gateway OK。
- Spec 仍为 **Draft**，接受后进 Plan。
