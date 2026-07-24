# CD Environment IngressPolicy 验证记录
最后修改时间: 2026-07-22 10:39:11

Review status: Accepted

Flow mode: strict

路线位置：cadence **修订 R2** 的 Verification。  
后继 **R3**（Application.kind）：`docs/requirement/20260722-cd-application-kind-gateway.md`（Requirement **Draft**；**未**进入 Spec/实现，本验证不覆盖）。

## Requirement alignment

| 需求要点 | 结论 |
|----------|------|
| Environment 接入策略 SoT（域名推导 / 默认 entrypoint / TLS） | **符合** — `environment` 列：`base_domain`, `domain_template`, `default_entrypoint`, `tcp_entrypoint`, `tls_mode` |
| 域名由策略推导，默认不手填 per-expose 域名 | **符合** — `deriveHost` 默认 `{app_code}.{base_domain}`；无 Binding 域名表单 |
| Environment **禁止** component 用户配置维度 | **符合** — Environment 页/API 无 component 字段；component 仅 Version Expose |
| Render 合路：`Version.Expose × Environment.IngressPolicy × Application` | **符合** — `RenderCompose` / `injectExposeLabels`；Deploy/Preview 共用 |
| 缺策略失败，错误指向补全环境接入策略 | **符合** — `validatePolicyForExposes` 文案 `environment ingress policy incomplete: …` |
| 废止 EnvironmentBinding 用户 SoT；无双轨 | **符合** — DROP `environment_binding`；删除 Binding CRUD API/UI |
| 同 app 维度可区分域名（模板含 app_code） | **符合** — 默认模板含 `{app_code}`；可选 `{env_code}` |
| 无 HTTP expose / 不写 labels 时不强制完整域名策略 | **符合** — Spec D-Validate：仅 ingress + HTTP 时强制 base_domain；TCP `tls_mode=none` 可用 `HostSNI(*)` |
| 自动化 go test + yarn lint/typecheck | **符合**（见 Test results） |

## Spec alignment

| Spec 决策 | 结论 |
|-----------|------|
| D-Store DROP `environment_binding` | **符合** — migration 000010 up |
| D-Policy 策略内嵌 Environment | **符合** — 无独立 policy 表 |
| D-Template 默认 Host `{app_code}.{base_domain}` | **符合** — `deriveHost` |
| D-MultiHTTP 共用主 Host + path_prefix；path 冲突失败 | **符合** — `validateHTTPPathConflicts` + 测试 |
| D-TCP entrypoint 回落；TLS 时要求 host | **符合** — `buildTraefikLabels` TCP 分支 |
| D-Validate 仅需写 HTTP labels 时强制策略 | **符合** |
| D-Seed `local.test` / `web` / `tls_mode=none` | **符合** — migration 默认 + usecase create 默认 |
| D-Override 应用级覆盖不做 | **符合** — 无覆盖 API |
| D-API 删除 Binding；CRUD 带策略字段 | **符合** — `environment.proto` 无 Binding 消息；handler/routes 无 binding 路由 |
| D-UI 去掉绑定配置；表单编策略 | **符合** — `EnvironmentPage.vue` 列表/表单为策略字段 |
| D-Migration 仅新增 000010 | **符合** — 未改 000001–000009 |

## Plan alignment

| Plan 步骤 | 结论 |
|-----------|------|
| 1 Migration 000010 sqlite + mysql | **完成** |
| 2 Model / Repo 扩字段；删 Binding；seed 默认策略 | **完成** |
| 3 Render 去掉 Bindings；deriveHost + labels | **完成** |
| 4 Usecase CRUD 归一化；Deploy/Preview 不读 binding | **完成** |
| 5 Proto / handler / routes + gen | **完成** |
| 6 Frontend EnvironmentPage + api + i18n | **完成** |
| 7 Tests；migrate version = 10；go/yarn 检查 | **完成** |

## Actual diff summary

相对当前分支基线（`feature/cd-redesign`，R1 已提交 `6e410a6`）的 R2 交付核心：

1. **数据**：`sql/migration/*/000010_cd_environment_ingress_policy.*` — ALTER environment 策略列、回填、DROP `environment_binding`。
2. **领域/渲染**：`model.Environment` 策略字段；删除 `EnvironmentBinding` 模型与替换 API；`compose_renderer` 策略推导 Host/path/TCP/TLS。
3. **API**：`environment.proto` 策略字段；删除 Binding 消息与 `PUT …/binding` 路由。
4. **前端**：Environment 列表展示 base_domain/entrypoint/tls；表单编辑策略；移除绑定模态与绑定数列。
5. **测试**：migration/bootstrap/e2e 期望版本 **10**；compose 测试覆盖 Host 推导、缺策略、path 冲突。
6. **文档**：R2 requirement/spec/plan；cadence 索引 R2；同批工作树另含 **R3 Requirement Draft**（仅文档，无产品代码）。

## Expected vs actual changed files

| Plan 预期 | 实际 |
|-----------|------|
| `000010_*` sqlite/mysql | 有 |
| model / sqlx repo / project seed | 有 |
| compose_renderer + deploy render 路径 | 有 |
| environment usecase/handler/mapper/routes | 有；删 binding |
| proto + gen (go/ts) | 有 |
| EnvironmentPage + api + i18n | 有 |
| migration_test / compose_test / e2e version 10 | 有 |
| R2 requirement/spec/plan/verification | 有 |

**同工作树但非 R2 产品范围**：

- `docs/requirement/20260722-cd-application-kind-gateway.md` 及 cadence/domain/render 上的 **R3 指针**（Draft 路线文档，无 `application.kind` 实现）。

未发现恢复 Binding 双轨或 `code==traefik` 魔法分支。

## Acceptance criteria checklist

| # | 验收项（Requirement） | 结果 |
|---|----------------------|------|
| 1 | Environment 编辑接入策略；无 component / 默认 per-expose 域名行 | **通过** |
| 2 | Version 含 Expose + Env 含策略即可 Deploy 生成 labels，无需 Binding 行 | **通过**（代码路径 + 单测） |
| 3 | 同项目两应用同名 component 时 Environment 无 redis 冲突配置；域名可按 app 区分 | **通过**（无环境级 component；Host 含 app_code） |
| 4 | 无 Expose 可 Deploy；有 HTTP+ingress 时才强制域名策略 | **通过** |
| 5 | 旧 Binding API/UI 非用户路径；策略推导与失败有测 | **通过** |
| 6 | `go test` + `yarn lint:fix` + `typecheck` | **通过**（lint 仅既有 warning，0 error） |

## Test results

| 命令 | 结果 |
|------|------|
| `go fmt ./cmd/... ./internal/...` | 通过（无额外 diff） |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过（含 `cd/usecase`、`database` migration v10、`bootstrap`、`e2e`） |
| `yarn --cwd web lint:fix` | 通过（0 errors；既有 gen/import warning） |
| `yarn --cwd web typecheck` | 通过 |

未做：真实 Docker compose up + Traefik 联调（与 R1 验证同级：单元/迁移覆盖 labels 合成，运行时联调依赖本机环境）。

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| R3 文档 | 工作树含 Application.kind Requirement Draft 与索引指针；**不在**本 R2 验收交付物内，属下一阶段 |
| consumer external 网络注入 | Requirement Non-goal / R3 E1；未实现 |
| 应用级域名覆盖 | Spec Non-goal；未实现 |
| 浏览器 E2E 手工点检 Environment 页 | 未在本机浏览器跑通；UI 以代码与 typecheck 为准 |

## Risks

1. 开发库若仍有旧 binding 数据，migration DROP 表后数据不可恢复（符合「无历史兼容」）。
2. Down migration 重建空 `environment_binding` 且丢策略列数据，仅开发回滚用。
3. 未做 Traefik 真机联调：labels 文案与现网是否完全一致需部署时观察。
4. 工作树同时含 R3 Draft 文档：提交时建议与 R2 实现拆分或在 message 中明确「含 R3 路线文档」。

## Incomplete items

1. 无 `application.kind`（R3）— **有意延后**。
2. 无 Verification 级浏览器/UI 录屏。
3. cadence 子索引在本验证 Accepted 后应回写 R2 Verification 状态（见 cadence 同步）。

## Conclusion

**R2 IngressPolicy 验证通过（Accepted）。**  
实现与 Requirement / Spec / Plan 对齐：环境级策略取代 per-component Binding；Render 按策略推导 Host 与 labels；migration 10；自动化检查通过。  

**下一阶段任务（未开始）**：R3 `Application.kind` = `gateway` \| `standard` — 当前仅有 Requirement Draft，需用户 Accept 后再进 Spec/Plan/实现。
