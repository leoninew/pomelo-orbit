# SQL 迁移合并：领域对齐基线 — Verification
最后修改时间: 2026-07-24 11:55:56

## Requirement alignment

| 需求项 | 结果 |
|---|---|
| 废弃 000001–000013 | 符合：旧链文件已从 sqlite/mysql 移除 |
| 新链自 000014 续号 | 符合：000014–000030 |
| 按领域命名文件，无 ci_schema/cd_schema | 符合 |
| 表更名 pipeline_stage / pipeline_stage_run | 符合（migration 内） |
| 系统种子 + pipeline 演示种子；无 application/route 演示 | 符合（SQLite 单测覆盖） |
| 双驱动同构 | 符合：文件名对称 |
| 对称 down | 符合 |
| 仅 migration（后扩展：迁移单测 version 断言） | 基本符合：交付含 migration + 迁移相关测试 + 过程文档；未改业务 repository SQL / Proto |

## Spec alignment

| 设计点 | 结果 |
|---|---|
| D1 领域一文件 014–030 | 符合 |
| D2 表更名仅 migration | 符合 |
| D5 down 对称 | 符合 |
| D6/D7 双驱动 | 符合 |
| gateway 独立文件 | 符合 `000025_gateway` |
| Plan 文档 | 未单独写 Plan（用户确认后直接实现）；以 Spec outline 为准 |

## Plan alignment

不适用（standard 流程下未产出独立 plan 文件；按 Spec implementation outline 实施）。

## Actual diff summary

暂存区核心变更：

1. `sql/migration/{sqlite,mysql}`：删除历史 001–013 增量；新增领域基线 014–030（up+down）。
2. 表规范名：`build_stage`→`pipeline_stage`，`stage_run`→`pipeline_stage_run`。
3. 种子：`seed_system` / `seed_pipeline`；无 CD 演示数据。
4. 单测：期望 version=30；补充领域表 / 种子 / 旧表不存在断言。
5. 过程文档：requirement、spec、domain consensus analyze。

工作区另有 **未暂存** Proto 领域拆分（proto / gen / web gen），与本交付无关。

## Expected vs actual changed files

| 预期 | 实际 |
|---|---|
| `sql/migration/**` | 有 |
| 过程文档 requirement/spec/analyze | 有 |
| 迁移 version 单测（实现后为验证需要追加） | 有：`migration_test.go` / `app_test.go` / `mysql_e2e_test.go` |
| 业务 Go SQL 旧表名对齐 | 无（Non-goal） |
| Proto/sqlc 领域搬迁 | 无（本提交不包含；工作区另有未暂存改动） |

## Acceptance checklist

1. [x] 无 001–013 旧链文件  
2. [x] 014–030 连续；sqlite/mysql 对齐  
3. [x] 领域文件覆盖共识表归属；gateway 独立  
4. [x] `pipeline_stage` / `pipeline_stage_run` 存在；旧名不存在  
5. [x] 系统 + pipeline 种子；无 application 演示（SQLite 断言）  
6. [x] 各版本有 down  
7. [x] SQLite MigrateUp → version=30 dirty=false  

## Test results

| 命令 | 结果 |
|---|---|
| `go test ./internal/infrastructure/database/... -count=1` | **PASS**（version=30、schema/种子断言） |
| `go test ./internal/bootstrap/... -count=1` | **FAIL（setup）** — 工作区未暂存 Proto 拆分导致 `internal/gen/proto/orbit/v1` 包路径断裂；**非本迁移 diff 引入**。此前在 Proto 未破坏时 bootstrap 迁移断言曾通过。 |
| `go vet`（database/bootstrap） | bootstrap 同因 setup failed |
| MySQL e2e | 未跑（无 e2e 环境配置） |
| 全量 `go test ./cmd/... ./internal/...` | 未跑（受 Proto 工作区干扰 + 范围外） |

## Missed or expanded scope

| 项 | 说明 |
|---|---|
| 扩展 | 原 Non-goal 写「不改测试」；验证阶段为对齐 version=30 与基线断言，**有意更新**迁移相关单测。 |
| 未做 | Go 业务 SQL 仍可能引用 `build_stage`/`stage_run`；业务集成测未在本轮修绿。 |
| 未做 | 独立 Plan 文档。 |

## Risks

1. 重建库后，业务代码若仍查旧表名会运行失败 → 需后续代码对齐任务。  
2. 工作区 Proto 拆分未完成会阻断 bootstrap/全量编译 → 提交时勿与本迁移混提。  
3. MySQL 仅文件对称，e2e 未执行。  

## Incomplete items

1. Go 侧旧表名 SQL 对齐。  
2. MySQL e2e 实跑。  
3. 工作区无关 Proto 变更需另提交。  

## Conclusion

**迁移基线交付可接受（Accepted）。**  
SQLite 迁移单元测试证明新链 up 到 30、领域 schema 与种子符合预期。提交应仅包含 migration + 迁移测试 + 本 feature 过程文档；勿夹带 Proto 拆分。
