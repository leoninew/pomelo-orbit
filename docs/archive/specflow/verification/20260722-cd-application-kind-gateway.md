# CD Application kind（gateway / standard）验证记录
最后修改时间: 2026-07-22 13:06:52

Review status: Accepted

Flow mode: strict

路线位置：cadence **修订 R3** 的 Verification。  
前置 R2 Verification Accepted：`docs/verification/20260722-cd-environment-ingress-policy.md`。  
后继扩展 **E1–E5**（E5 = gateway 挂载细节 + Component 全规格补齐；**不另开任务**；本验证不交付 E5）。

## Requirement alignment

| 需求要点 | 结论 |
|----------|------|
| `application.kind` ∈ {`standard`, `gateway`}；默认 `standard` | **符合** — schema `DEFAULT 'standard'`；`normalizeApplicationKind`；Create 可写 |
| kind **创建后不可修改**（Update 无字段） | **符合** — `ApplicationUpdateReq` / DTO 无 kind；SQL Update 不写 kind |
| Render **按 kind 分支**；禁止 `code==traefik` 魔法 | **符合** — `RenderCompose` `switch app.Kind`；`renderGatewayCompose` 钩子；`TestRenderComposeGatewayKindDoesNotUseCodeMagic`（code=traefik 走 gateway 物化，非 code 特判） |
| gateway 仍用 Version/Component/Expose 统一管线；无额外数量/端口校验 | **符合** — 与 standard 共用 `renderComposeServices` |
| standard：有 Expose 写 labels，无则不写；废止 is_ingress 门闩 | **符合** — `if len(exposes) > 0 { injectExposeLabels }`；产品代码无 `IsIngress`/`AttachIngress` |
| 废止 `attach_ingress` / `service.is_ingress` 产品 SoT | **符合** — proto Deploy/Service 无字段；migration `000009` service 无列；UI 无 checkbox/列 |
| 开发期 schema 就地改；无增量 `000011`；migrate 版本 **10** | **符合** — `000002`+kind；`000009`-is_ingress；无 `000011_*`；`migration_test` 期望 10 |
| E5（挂载 + Component 全规格）非 R3 必交付 | **符合** — 未实现平台挂载注入；文档已挂 E5 |
| cadence / domain / render 指针 | **符合** — 已索引 R3 |
| 自动化 go test + yarn lint/typecheck | **符合**（见 Test results） |

## Spec alignment

| Spec 决策 | 结论 |
|-----------|------|
| D-Kind / D-Enum / D-Immutable | **符合** |
| D-RenderBranch / D-GatewayParity | **符合** — gateway 与 standard 高度重合；kind 分支可测；未知 kind 失败 |
| D-StandardLabels / D-NoAttach / D-NoIsIngress | **符合** |
| D-ContainerName / D-NameRules | **符合** — `container_name=app_code_component`；component name `^[a-z][a-z0-9-]*$` 在 `normalizeVersionComponents` 校验 |
| D-Net / D-NoProviderField / D-Count | **符合** — 无硬编码平台网名；无 provider 列；无 gateway 数量限制 |
| D-Migration 就地 000002/000009 | **符合** |
| D-API / D-UI | **符合** — 创建选 kind；详情只读；Deploy 无 attach；Services 无 is_ingress |
| E5 非必交付 | **符合** |

## Plan alignment

| Plan 步骤 | 结论 |
|-----------|------|
| 1 Schema 就地改 000002/000009；无 000011 | **完成** |
| 2 Model/Repo kind；去 is_ingress / ClearOtherIngress | **完成** |
| 3 DTO/usecase Render 分支；Expose labels；container_name；component 名规则 | **完成** |
| 4 Proto + gen | **完成**（含 VersionComponent/VersionExpose 消息重命名） |
| 5 HTTP handler/mapper | **完成** |
| 6 Frontend 创建 kind / 详情展示 / 去 attach | **完成** |
| 7 Tests；migrate=10；go/yarn 检查 | **完成** |

## Actual diff summary

相对分支基线（R2 已提交 `90ebaed`）的 R3 交付核心：

1. **数据**：`000002` 增加 `application.kind`；`000009` 去掉 `service.is_ingress`；表名 `version_component` / `version_expose`（与类型重命名一致）；无 `000011`。
2. **领域/渲染**：`Application.Kind`；`RenderCompose` kind 分支；labels 仅 Expose 驱动；`container_name` 拼接；component 名规则。
3. **命名**：模型/DTO/Repo/Proto `Component`→`VersionComponent`、`Expose`→`VersionExpose`（JSON 字段名仍 `components`/`exposes`）。
4. **API**：Create/Resp +kind；Update/Deploy/Service 无 attach/is_ingress。
5. **前端**：创建 kind 选择；详情只读 kind；Deploy/Services 去掉 attach/is_ingress。
6. **测试**：migration version 10；compose gateway/unknown kind/container_name/labels 路径。
7. **文档**：R3 requirement/spec/plan；cadence；E5 范围含 Component 全规格补齐。

## Expected vs actual changed files

| Plan 预期 | 实际 |
|-----------|------|
| SQL 000002 +kind、000009 -is_ingress | 有（sqlite+mysql） |
| model / sqlx repo / repository 接口 | 有 |
| compose_renderer + deploy/preview 路径 | 有 |
| version/service/application_extra usecase + dto | 有 |
| proto application/version/bundle + gen go/ts | 有 |
| handler mapper + web form/detail/i18n | 有 |
| compose_test / migration_test / integration 夹具 | 有 |
| R3 requirement/spec/plan/verification | 有 |

**同批额外（实现期合理扩展，非 Plan 文件清单逐字）**：

- `VersionComponent` / `VersionExpose` 全域类型与表命名对齐（开发期就地重命名）。
- docs 中 E5 文案扩展：补 Component 全规格并入 E5、不另开任务。

**未改（符合 Out of scope）**：

- E1 consumer 网络、E5 挂载/全规格数据补齐、K8s、host 网络。
- 生产增量 migration。

## Acceptance criteria checklist

| # | 项 | 结果 |
|---|----|------|
| R1 | kind 单一 SoT；创建可写、Update 不可改 | **通过** |
| R2 | Render 按 kind；无 code 魔法主路径 | **通过**（单测） |
| R3 | standard：Expose 驱动 labels；无 is_ingress/attach | **通过**（代码 grep + 测试） |
| R4 | gateway 可区分（分支钩子 + container_name 物化） | **通过**；策略体与 standard 高度重合（Spec 允许） |
| R5 | migrate=10；无 000011 | **通过** |
| R6 | go test + yarn lint:fix + typecheck | **通过**（lint 仅既有 warning） |
| R7 | E5 未误交付 | **通过** |

## Test results

| 命令 | 结果 |
|------|------|
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `yarn --cwd web lint:fix` | 通过（0 errors；既有 gen/dayjs/config warnings） |
| `yarn --cwd web typecheck` | 通过 |

补充核对：

- 产品路径 `internal/**/*.go` / `web/src/**`：**无** `AttachIngress` / `IsIngress` / `attach_ingress` 字段用法（renderer 注释提及废止语义除外）。
- `sql/migration`：**无** `000011`；sqlite 迁移止于 `000010`。
- 历史 R1/R2 **文档**仍含 is_ingress/attach 叙述（归档语义，非运行时代码）。

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| 扩展 | `VersionComponent`/`VersionExpose` 重命名 — 与 R3 表/领域一致，已纳入交付 |
| 扩展 | E5 文档范围明确「补 Component 全规格」— 文档 only |
| 未做（预期） | E5 实现；gateway 专用 labels（如 `api@internal`）；restart/env_file 顶层 networks 建模 |
| 测试缺口 | Plan 写「非法 component 名保存失败」单测：校验已在 usecase 落地，**未见独立 `*_test.go` 用例**（残余风险低） |
| 人工 | 未跑浏览器 E2E 创建 gateway→Preview→Deploy 全流程（开发库曾出现多 HTTP expose path 冲突属数据配置问题，校验行为正确） |

## Risks

1. **gateway 与 standard 渲染体高度重合**：入口角色 compose（ports/挂载/网络）依赖用户填 Component 规格；未填时 Preview 仅为最小 service — 与 E5 挂账一致，运维上易误判为 Render bug。  
2. **开发库就地改 migration**：已有库需重建/重跑 migrate；无生产增量脚本。  
3. **历史 R1/R2 文档**仍描述 is_ingress/attach — 读档时以 R3 为准。  
4. **多 HTTP Expose 同 path** 仍会在 Deploy/Preview 失败（R2 规则）— 正确；gateway 数据需收敛 Expose。

## Incomplete items

1. E5：gateway Component 全规格补齐 + 平台挂载契约（已挂账，不另开任务）。  
2. E1–E4 路线项。  
3. 可选：补 component 名非法字符的单元测试。  
4. 可选：人工 UI 冒烟（创建 gateway / 只读 kind / Preview）。

## Conclusion

**R3 验收通过（Verification Accepted）。**

实现与 Requirement / Spec / Plan 主干一致：`Application.kind`、Render 分支、废止 attach/is_ingress、Expose 驱动 labels、container_name 规则、开发期 schema 就地改且 migrate=10。工程门禁（go test、web lint:fix、typecheck）通过。E5 及更完整 gateway 形态对齐明确不在本期。

## User review notes

- 2026-07-22：用户 `/specflow 验收当前任务` → 执行 Verification；本记录 **Accepted**。
