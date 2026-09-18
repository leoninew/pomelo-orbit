# SQL 迁移合并：领域对齐基线
最后修改时间: 2026-07-24 11:23:35

Review status: Accepted

## Background

`sql/migration/{sqlite,mysql}` 使用 golang-migrate。历史链 `000001`–`000013` 已过长：终态 schema 拆散在增量 ALTER 中，CD 种子与终态表不一致。

项目已确认：

1. 允许并要求**重建数据库**；废弃旧链文件 `000001`–`000013`。
2. **版本号从历史最大版本之后递增**（`000014` 起），不从 `1` 重开。后续功能迁移继续 `000031`、`000032`… **没有**「最大只能到 30」的上限；30 仅是本轮文件个数推算出来的**本轮末版本号**。
3. 领域边界以 `docs/analyze/20260724-domain-split-consensus-共识.md` 为准：`ci` / `cd` **不是领域名**；migration 物理文件按领域命名与归属落点。
4. 工作区已有一版粗拆基线 `000014`–`000019`（`core_infra` / `auth_schema` / `ci_schema` / `cd_schema` / `seed_*`），与共识不符，需在本需求范围内**按领域重拆**，并在 migration 中完成共识表名纠正。

迁移执行机制（golang-migrate + embed + `MigrateUp`）保持不变。

## Goal

1. 废弃旧链 `000001`–`000013`；新链从 `000014` 续号，sqlite 与 mysql **文件名、版本号、语义对齐**。
2. 以**终态 schema** 重建迁移，不再保留 CD 演进中间态 SQL；无旧表（如 `application_config_file`、`application_service`、`application_route`、`environment_binding`）。
3. **按共识领域**拆分 migration 文件；文件名使用领域名，禁止 `ci_schema` / `cd_schema` 作为目标命名。
4. **表规范更名**（仅 migration 内；重建库直接建新表名）：
   - `build_stage` → `pipeline_stage`
   - `stage_run` → `pipeline_stage_run`
5. 种子策略：
   - **保留**系统种子：admin、default project、admin 角色、运行所需 permission 绑定；default environment 等系统级环境种子。
   - **保留** pipeline 演示种子：`repository` / `pipeline_template` / `pipeline_stage` / `pipeline_template_stage` 等。
   - **删除** application / route 等交付演示种子。
6. **先 up 后 down**：up 齐后补各文件 `.down.sql`。
7. **本轮只改 `sql/migration/**`**（sqlite + mysql）。

## Non-goal

1. 不改 golang-migrate 接入、embed、启动迁移入口。
2. 不做旧库兼容或数据迁移；库删除后重建。
3. **不改任何 Go / Proto / 前端 / 测试代码**（含 `migration_test.go`、repository SQL、期望 version 断言）。表名与版本号在应用侧的对齐属**后续任务**。
4. 不做 Proto / sqlc / 包路径领域拆分。
5. 不引入兼容层、双表、视图别名。
6. 不执行 git 写操作（除非用户另行授权）。

## User scenarios

1. **重建库后跑迁移脚本**
   - 删除旧库，执行 migrate up。
   - 自 `000014` 执行到本轮末版本（按文件表为 `000030`），得到终态 schema + 系统种子 + pipeline 演示种子。
   - `schema_migrations.version` 等于本轮末版本；`dirty=false`。

2. **按领域阅读 migration**
   - 文件名对应该领域；`gateway` 独立文件；注释标明逻辑领域。

3. **与应用代码的关系（本轮后）**
   - 迁移脚本已用新表名；Go 内嵌 SQL 仍可能引用旧表名 → **本轮不修**；在后续「代码对齐」任务中处理。在此之前，依赖旧表名的测试/进程可能失败，属预期分界。

## Acceptance

1. `sql/migration/{sqlite,mysql}/` 无 `000001`–`000013`；无 `ci_schema` / `cd_schema` 目标文件名。
2. 新链自 `000014` 连续编号至本轮末版本；双驱动文件名与语义对齐。
3. 物理文件覆盖共识表归属（见 Decisions）；`gateway` 独立文件。
4. 存在表名 `pipeline_stage`、`pipeline_stage_run`；不存在 `build_stage`、`stage_run` 及旧 CD 中间表。
5. 种子符合 Goal；无 application/route 演示行。
6. 各版本具备对称 `.down.sql`。
7. **本轮 diff 仅限** `sql/migration/**`（及本需求/Spec/Plan 过程文档若需要）；无业务/测试代码改动。

## Open questions

无阻塞。已确认：

1. 「30」不是上限，见 Decisions §版本号说明。
2. 本轮只改迁移脚本。
3. `gateway` 独立文件。

## Decisions

1. **流程**：SpecFlow **standard / 标准模式**；含 Spec 文档。
2. **领域基线**：`docs/analyze/20260724-domain-split-consensus-共识.md`。
3. **版本号说明**：
   - 从历史最大版本 **之后** 递增：本轮第一步 = `000014`。
   - 本轮规划 **17** 个版本文件（15 schema 域 + 2 seed）→ 末版本号 = 14+16 = **30**。
   - **30 不是项目硬限制**；只是「本轮落地后 `schema_migrations.version` 的预期值」。之后新迁移从 **31** 起继续。
4. **重建库**；无兼容脚本。
5. **种子**：pipeline 演示保留；application/route 演示删除。
6. **双驱动**：sqlite 与 mysql 同构。
7. **表更名**：仅在 migration SQL 中使用新表名；**本轮不改 Go**。
8. **范围**：**仅** `sql/migration/{sqlite,mysql}/**`。
9. **物理文件序**（版本号 = 014 起连续编号；下列「30」为算术结果）：

| Version | stem | 领域 | 表 / 内容 |
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
| 000025 | `gateway` | gateway | `gateway_config` |
| 000026 | `service` | service | `service` |
| 000027 | `deployment` | deployment | `deployment` |
| 000028 | `route` | route | `route` |
| 000029 | `seed_system` | seed | 系统种子 |
| 000030 | `seed_pipeline` | seed | pipeline 演示种子 |

10. **down**：先 up 后 down。
11. **废止** light 六文件 / version=19 / 仅 sqlite 等旧决策。

## Risk

1. **脚本与代码短期不一致**：migration 已更名，Go 仍用 `build_stage`/`stage_run` → 本轮后相关测试/进程会失败，直至后续代码对齐任务。
2. **权限/environment 种子遗漏**：仅 migration 侧尽量对齐现 seed；集成测失败不在本轮修代码。
3. **FK/列遗漏**：按现粗拆终态 SQL 剪切校验。
4. **旧库未删**：必须重建。
5. **误改 Go**：范围门禁仅 `sql/migration/**`。

## User review notes

- light → 按共识修正；standard + Spec。
- 用户确认：(1) 领域一文件 OK；**澄清 30 非上限，是本轮末版本算术结果**；(2) **本轮只改迁移脚本**；(3) gateway 独立 OK。
