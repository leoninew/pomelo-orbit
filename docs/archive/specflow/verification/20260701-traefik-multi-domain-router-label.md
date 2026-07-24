# Traefik 多域名 Router Label 验证
最后修改时间: 2026-07-01 13:21:56

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260701-traefik-multi-domain-router-label.md` 核对，本次实现与轻量模式 / light requirement 对齐：

1. 支持在应用详情中为同一个 service 添加多条域名路由；前端仍复用现有单条路由表单和多条 `application_route` 记录能力。
2. 生成 Docker Compose labels 时，同一 service、同一 container port 的多条路由合并为一个 Traefik router rule，格式为 `Host(`domain1`) || Host(`domain2`)`。
3. 不同 service 仍按各自 service name 生成独立 router / service labels。
4. 同一 service 配置不同 container port 时返回错误，不静默选择其中一个端口。
5. 前端和后端均增加 domain/host 合法性校验，拒绝空值、空格、反引号和不符合域名格式的值。
6. 预览和实际部署共用 `renderApplicationConfigFile` 路由 label 注入逻辑，避免预览与部署输出不一致。
7. 无路由时仍保留清理托管 labels 的既有行为。
8. 已补充后端单元测试覆盖多域名合并、不同端口拒绝、非法 domain 拒绝以及预览/部署一致性。

Requirement 中的 Non-goal 保持一致：未新增多域名输入框，未调整 `application_route` 表结构，未改造 Traefik runtime routers 只读页面，未实现跨 service 的路由合并。

## Spec alignment / 规格对齐

本功能采用轻量模式 / light，没有单独创建 spec 文档。验证按 requirement / 需求核对。

## Plan alignment / 计划对齐

本功能采用轻量模式 / light，没有单独创建 plan 文档。实现范围按 requirement / 需求和用户“直接实现”要求执行。

## Actual diff summary / 实际差异摘要

验证时已暂存的相关变更如下：

```text
A  docs/requirement/20260701-traefik-multi-domain-router-label.md
M  internal/service/cd/application_extra.go
M  internal/service/cd/compose_test.go
M  internal/service/cd/deployment_execution.go
M  internal/service/cd/deployment_execution_test.go
M  web/src/i18n/locales/en-US.ts
M  web/src/i18n/locales/zh-CN.ts
M  web/src/views/cd/ApplicationDetail.vue
```

`git diff --cached --stat` 摘要：

```text
.../20260701-traefik-multi-domain-router-label.md  |  60 +++++++++++
internal/service/cd/application_extra.go           | 113 ++++++++++++++++-----
internal/service/cd/compose_test.go                |  56 ++++++++++
internal/service/cd/deployment_execution.go        |  22 +---
internal/service/cd/deployment_execution_test.go   |  34 ++++++-
web/src/i18n/locales/en-US.ts                      |   1 +
web/src/i18n/locales/zh-CN.ts                      |   1 +
web/src/views/cd/ApplicationDetail.vue             |  17 +++-
8 files changed, 254 insertions(+), 50 deletions(-)
```

实际改动内容：

- `docs/requirement/20260701-traefik-multi-domain-router-label.md`
  - 新增轻量模式 requirement，记录多域名合并规则、非目标、验收标准和风险假设。
- `internal/service/cd/application_extra.go`
  - 新增 application route domain 校验与小写归一化。
  - 新增 `renderApplicationConfigFile`，让 compose 预览和部署写盘共用模板渲染、service config 应用和 route label 注入逻辑。
  - 将 route label 注入从逐条覆盖改为按 service 分组后合并 Host 条件。
  - 同一 service 不同 port 返回 validation error；重复 domain 去重。
- `internal/service/cd/deployment_execution.go`
  - 部署写文件路径改为复用 `renderApplicationConfigFile`，保证预览与实际部署一致。
- `internal/service/cd/compose_test.go`
  - 覆盖同 service 多域名合并、同 service 不同 port 拒绝、非法 domain 拒绝和 domain 小写归一化。
- `internal/service/cd/deployment_execution_test.go`
  - 覆盖 application compose 预览和部署写出的 `docker-compose.yml` route labels 一致。
- `web/src/views/cd/ApplicationDetail.vue`
  - 前端 route 表单增加 domain 格式校验，提交前 trim + lowercase。
- `web/src/i18n/locales/en-US.ts`, `web/src/i18n/locales/zh-CN.ts`
  - 增加 domain 校验错误文案。

## Expected vs actual changed files / 预期与实际改动对比

| 预期范围 | 实际文件 | 结论 |
| --- | --- | --- |
| Requirement 文档 | `docs/requirement/20260701-traefik-multi-domain-router-label.md` | 符合 |
| 后端路由 label 注入与 domain 校验 | `internal/service/cd/application_extra.go` | 符合 |
| 部署写盘复用预览渲染逻辑 | `internal/service/cd/deployment_execution.go` | 符合 |
| 后端单元测试 | `internal/service/cd/compose_test.go`, `internal/service/cd/deployment_execution_test.go` | 符合 |
| 前端 domain 校验 | `web/src/views/cd/ApplicationDetail.vue` | 符合 |
| 前端校验文案 | `web/src/i18n/locales/en-US.ts`, `web/src/i18n/locales/zh-CN.ts` | 符合 |
| 数据库结构 / API contract | 无迁移、无 API 字段变更 | 符合 Non-goal |
| Traefik runtime routers 只读页面 | 无相关改动 | 符合 Non-goal |

## Acceptance criteria checklist / 验收清单

- [x] 同一 service、同一端口的多条路由生成一条合并后的 router rule。
- [x] 不同 service 仍分别生成各自 labels。
- [x] 同一 service 多域名不会因为循环赋值导致只保留最后一条。
- [x] 同一 service 若配置了不同 container port，不静默选择其中一个端口，直接返回错误。
- [x] 前端和后端都校验 host/domain 合法性，至少拒绝包含空格的值；本次同时拒绝反引号和明显非法域名。
- [x] 预览逻辑和部署生成逻辑共用同一套路由 label 注入逻辑。
- [x] 现有“无路由时清理托管 labels”的行为保持不变。
- [x] 补充后端单元测试覆盖多域名合并行为和非法 host 拒绝行为。

## Test results / 命令结果

已按项目约束运行后端检查：

```text
go fmt ./cmd/... ./internal/...
结果：通过
输出摘要：无输出。
```

```text
go vet ./cmd/... ./internal/...
结果：通过
输出摘要：无输出。
```

```text
go test ./cmd/... ./internal/...
结果：通过
输出摘要：cmd/internal packages 均通过或无测试文件；多处结果来自缓存。
```

已按项目约束运行前端检查：

```text
yarn --cwd web lint:fix
结果：通过
输出摘要：eslint . --fix --cache，Done in 0.96s.
```

```text
yarn --cwd web typecheck
结果：通过
输出摘要：vue-tsc --noEmit，Done in 4.49s.
```

补充检查：

```text
git diff --check --cached
结果：通过
输出摘要：无输出。
```

```text
git diff --check -- docs/requirement/20260701-traefik-multi-domain-router-label.md internal/service/cd/application_extra.go internal/service/cd/compose_test.go internal/service/cd/deployment_execution.go internal/service/cd/deployment_execution_test.go web/src/views/cd/ApplicationDetail.vue web/src/i18n/locales/en-US.ts web/src/i18n/locales/zh-CN.ts
结果：通过
输出摘要：无输出。
```

## Missed or expanded scope / 范围偏差

- 未做真实 Docker / Traefik 环境手工验证；本次验证覆盖生成 compose labels 的 Go 单元测试、预览/部署一致性测试、前端 typecheck 和 lint。
- 当前工作区仍存在其他未暂存变更，不属于本次 Traefik 多域名 router label 验证范围，例如 `.gitignore`、`Taskfile.yml`、manage.py scp、部署详情布局补充、MonacoEditor、CI 页面、`ApplicationCreateWizard.vue` 删除等。
- 本验证文档是验证阶段新增文件，创建后未自动暂存；如需要和本次 Traefik 变更一起提交，应由用户确认后再暂存完整文件。

## Risks / 风险

- 当前 router/service label 命名仍以 service name 为唯一键，因此同一 service 不同 port 被显式拒绝；若未来需要同一 service 多端口路由，需要新增 router/service 命名策略。
- 当前 domain 校验不支持 wildcard domain、IPv6 literal、带 scheme/path 的输入；这符合本次“host/domain”字段语义，但如未来需要更多 Host rule 形式需另开需求。
- 表结构不限制重复路由记录；实现层已对同 service 重复 domain 去重，避免重复 Host 条件，但数据库层仍允许重复记录存在。

## Incomplete items / 未完成项

- 无阻塞交付的未完成项。
- 可选后续优化：如产品需要支持 wildcard domains 或同 service 多端口路由，应扩展需求并设计新的 label 命名策略。

## Conclusion / 结论

本次 Traefik 多域名 Router Label 实现与 `docs/requirement/20260701-traefik-multi-domain-router-label.md` 对齐；后端 Go fmt/vet/test、前端 lint/typecheck 和 diff whitespace 检查均通过。除未做真实 Traefik 环境手工验证、当前工作区存在其他无关未暂存改动外，没有发现阻塞交付的问题。当前可以进入用户验收或提交准备阶段。
