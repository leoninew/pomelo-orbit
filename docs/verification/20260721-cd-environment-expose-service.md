# CD Environment / Expose / Service 验证记录
最后修改时间: 2026-07-22 08:35:55

Review status: Accepted

Flow mode: standard

路线位置：cadence **修订 R1** 的 Verification。  
后继 **R2**（产品反馈：Binding 形态）：`docs/requirement/20260722-cd-environment-ingress-policy.md`（Draft / strict）。

## Requirement alignment

| 需求要点 | 结论 |
|----------|------|
| Environment 归属 Project（`project_id`，唯一 `(project_id, code)`，无 `application_id`） | **符合** — model/migration/API 均为 project 作用域 |
| 禁止跨 Project deploy（app.project_id == env.project_id） | **符合** — usecase 校验返回业务错误 |
| Version 承载 ExposeSpec（http\|tcp，无域名） | **符合** — `expose` 表 + Version CRUD 嵌入 exposes |
| Environment 承载 Binding（域名/入口/证书参数） | **符合** — `environment_binding` + 全量替换 API |
| Service 唯一键 `(app, env, instance_key)`；一 App 多 Service | **符合** — schema + list API + 前端 Services 表 |
| Traefik MVP：每 (app, env) 至多一个 ingress（`is_ingress`） | **符合** — deploy 写 `IsIngress`；Render 仅 ingress 写 labels |
| deploy 必选 version_id + environment_id；Render = Version × Environment | **符合** |
| 废止 `application_route` 长期 SoT；无历史兼容 | **符合** — DROP 表/列；删除 proto 与运行路径；**不**重建数据 |
| 按 Project seed `local`；新建 Project 同步 local | **符合** — migration seed + project create 联动 |
| 工作目录 / compose project 名含 env + instance | **符合** — `data/cd/{app}/{env}/{instance}`、`{app}-{env}-{instance}` |

## Spec alignment

不适用（standard 模式无独立 Spec 文档）。

## Plan alignment

| Plan Step | 结论 |
|-----------|------|
| Step 1 Migration + models + repository | **完成** — `000009_*` sqlite/mysql；sqlx CD 仓库扩展 |
| Step 2 Proto + gen | **完成** — `environment.proto` / `version.proto`；去掉 `application_route.proto` |
| Step 3 Environment/Expose/Service usecase | **完成** |
| Step 4 Render + deploy chain | **完成** — `compose_renderer` http/tcp 分支 |
| Step 5 HTTP handlers + routes | **完成** — 单数路径 `/api/cd/environment` |
| Step 6 Frontend | **完成** — Environment 页、Detail Expose/Deploy/Services、i18n/nav |
| Step 7 Tests + 工程检查 | **完成**（见 Test results） |

Plan 预期外但合理的清理：删除已无表可对的 sqlc 查询（`application.sql` / `route.sql` / `config_file.sql` / `service_config.sql`）及其生成物，避免 `sqlc generate` 与 000008/000009 后 schema 冲突。

## Actual diff summary

工作树相对 `develop` 基线（约 109 files，+9949 / -3909，量级随未提交状态浮动）核心交付：

1. **数据**：`000009_cd_environment_expose`；Environment / Expose / Binding；Service 多实例列；DROP `application_route` / `route_managed`。
2. **领域**：Environment CRUD + binding 替换；Version exposes；Deploy/Preview 双输入；同 Project 硬校验；ingress 与 workspace 路径改造。
3. **API**：`/api/cd/environment`；deploy/stop/restart/preview/service list 适配；废止 app 级 route API。
4. **前端**：Environment 管理页；ApplicationDetail Expose + Deploy 对话框 + 多实例 Stop/Restart 目标选择；列表 service_count；i18n zh/en。
5. **文档**：requirement/plan（本 feature）及同日 version 系列过程文档。
6. **工具链清理**：删除 `application_route` proto 与 sqlc 死查询。

同 diff 中亦包含 **Version 底座**（`000008`、version proto/usecase）相关改动，作为本 feature 的前置实现一并在本分支交付，而非无关杂项。

## Expected vs actual changed files

| Plan 预期主集合 | 实际 |
|-----------------|------|
| `sql/migration/.../000009_*` | 有；另含 `000008_*`（Version 底座） |
| `internal/model/cd.go`、sqlx cd repo、usecase | 有 |
| workspace + port | 有 |
| proto environment/version/application/bundle/deployment | 有；另删除 `application_route.proto` |
| handler/routes | 有 |
| web api/views/router/i18n | 有 |
| tests / migration 版本 9 | 有 |
| sqlc 清理 | Plan 未单列，实现期补齐 |

未发现与 Plan 冲突的「双轨兼容层」或跨 Project Environment API。

## Acceptance criteria checklist

| # | 验收项 | 结果 |
|---|--------|------|
| 1 | Environment → Project N:1；deploy 强制同 project | **通过**（代码路径 + 校验文案） |
| 2 | Expose 含 http/tcp；Render 合路 | **通过**（compose_renderer + compose_test） |
| 3 | Service 多实例键 + Traefik MVP 单 ingress | **通过** |
| 4 | 废止 application_route SoT | **通过** |
| 5 | seed local / 新建 Project 有 local | **通过**（migration + project create + 集成测试断言） |
| 6 | 前端 Environment + Deploy 选 env/instance | **通过**（typecheck；无浏览器 E2E） |
| 7 | 迁移版本 9 | **通过**（`assertGolangMigrateVersion(..., 9)`） |

手工场景（切换项目只见本项目 env、真实 Traefik TCP 入口是否已配）**未**在本机 UI/运行时复现，记入 risks。

## Test results

| 命令 | 结果 |
|------|------|
| `go test ./internal/application/cd/usecase/... ./internal/api/http/handler/cd/... ./internal/infrastructure/database/... ./internal/test/e2e/... ./internal/bootstrap/... -count=1` | **通过** |
| `go test ./cmd/... ./internal/... -count=1` | **通过** |
| `yarn --cwd web lint:fix` | **通过**（0 error；既有 import/dayjs warning） |
| `yarn --cwd web typecheck` | **通过** |

实现期另曾运行：`go fmt` / `go vet` 通过。

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| 扩展 | 残留清理：删除 application_route proto/sqlc 死查询；Detail 多实例 Stop/Restart 目标选择 |
| 扩展 | 同 diff 含 Version 系列 docs + `000008` 底座（本 feature 依赖，合理合并交付） |
| 未做 / 明确 out of scope | 同域名多实例 LB、蓝绿 UI、Environment ACL 表、K8s 多机、manual 证书上传 |
| 未做 | 浏览器端到端手工验收、真实 Docker/Traefik 联调截图 |

## Risks

1. **workspace 路径变更**：旧 `data/cd/{app}/` 目录不会自动迁移；需运维理解或重建实例。
2. **Traefik TCP entrypoint**：labels 会写，但集群未配置对应 entrypoint 时运行侧不通。
3. **列表页 Stop**：应用列表一键 stop 在多实例时依赖后端 `resolveServiceTarget`（单实例可走、多实例需 environment/service 指定）；详情页已补选择 UI。
4. **过程文档混入**：version 系列 requirement/plan 与本 feature 同批未提交；拆 commit 时需注意边界。
5. **无 UI E2E**：验收清单中的「切换项目 / 配 binding / 跨 env id」依赖人工或后续浏览器测试。

## Incomplete items

1. 未做真实环境手工点验（产品侧建议仍走一遍 Plan 手工表）。
2. 应用列表页多实例 stop 未做目标选择 UI（详情页已覆盖）。
3. 创建向导仍为「空白应用」主路径；旧「从镜像生成 compose+route」产品能力未在本迭代重做（i18n 已去掉 routeDomain 死键）。
4. **环境 Binding 产品形态**（手填域名 / 环境侧 component）经 2026-07-22 反馈判定不当 → **不在本 Verification 内修复**；转入 **R2** `docs/requirement/20260722-cd-environment-ingress-policy.md`。

## Conclusion

**验证结论：通过（有条件）**。

相对 Accepted requirement / plan，核心领域约束与实现步骤已落地；自动化测试与前端 typecheck 全绿。条件为：

1. 交付前建议人工走通：Project local env → Version expose → Environment binding → Deploy → 多实例 ingress 行为。
2. 若需拆分提交，建议将「Version 底座 + Environment/Expose/Service」与「过程文档」按评审习惯分组，避免单 commit 语义过载。

用户确认进入下一阶段后，将 Review status 更新为 **Accepted**（2026-07-21）。

## Follow-on / 后继路线

| 文档 | 说明 |
|------|------|
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | R2：IngressPolicy 取代 per-component Binding |
| `docs/requirement/20260721-cd-application-version-cadence.md` | 总索引 P0–P3 → R1 → R2 |

## User review notes

- 2026-07-21：用户「继续下一阶段」→ Verification Accepted；SpecFlow 本 feature（R1）闭环。
- 条件仍成立：建议上线前人工走通 local env → expose → binding → deploy；拆 commit 时注意 Version 底座与过程文档边界。
- 2026-07-22：挂入 R2 后继；R1 验收结论不变。
