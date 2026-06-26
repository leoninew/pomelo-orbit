# CD 应用创建方式选择验证
最后修改时间: 2026-06-27 00:07:52

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260626-cd-application-create-methods.md` 核对，本次实现覆盖需求中的核心目标：

1. 应用列表页主按钮文案调整为“创建应用”，点击后跳转独立创建页面 `/cd/applications/create`，不在应用列表页停留，也不使用创建模态窗。
2. 独立创建页面中使用 Reka UI Stepper 展示多步骤创建流程。
3. 第一步展示 4 种创建方式：从镜像创建、从 docker-compose 创建、导入应用包、高级空白应用。
4. “从镜像创建”完整可提交，支持应用名称、应用代码、镜像、容器端口、环境变量、多条数据目录和可选域名/路由。
5. 环境变量生成 `.env`，并在 `docker-compose.yml` 中通过 `env_file: .env` 显式引用。
6. 数据目录支持多条 volume 映射，并写入 `docker-compose.yml`。
7. 填写域名时通过现有接口启用路由托管，并为 service `app` 创建或更新路由。
8. 前端串联现有接口，不新增 backend-go 聚合接口；如果应用已创建但后续补齐失败，重新提交会继续使用同一个 `createdApplication` 补齐配置文件和路由。
9. “高级空白应用”在独立创建页面内复用原字段和原创建接口，只创建应用基础信息，不生成配置文件。
10. “从 docker-compose 创建”和“导入应用包”本次不落地完整提交流程，保留后续支持或现有导入入口说明。

需求中的 Non-goal 保持一致：未实现模板库、未重构数据模型、未修改迁移文件、未处理旧 Python backend、未实现部署预检或 `.env` 专用编辑器、未移除高级用户手写配置文件能力。

## Spec alignment / 规格对齐

不适用。当前流程为标准模式 / standard，本次没有单独创建 spec 文档，按 requirement / 需求和 plan / 计划核对。

## Plan alignment / 计划对齐

依据 `docs/plan/20260626-cd-application-create-methods.md` 核对，本次实现与计划主体一致：

- 新增独立创建页面 `frontend/src/views/cd/ApplicationCreatePage.vue`。
- 新增 `frontend/src/components/ApplicationCreateWizard.vue`，封装 Stepper、创建方式选择、从镜像表单、高级空白表单、确认页和失败恢复 UI。
- 新增 `frontend/src/utils/applicationCompose.ts` 生成 compose/env。
- 新增 `frontend/src/utils/applicationCompose.test.ts` 覆盖生成规则。
- 修改 `frontend/src/router/index.ts` 注册 `/cd/applications/create`。
- 修改 `frontend/src/views/cd/ApplicationPage.vue`，主按钮跳转独立创建页面，保留现有导入能力。
- 修改中英文 i18n 文案。
- 未修改 backend-go、迁移文件或旧 Python backend。

计划中“从镜像创建”的 service 固定为 `app`、`ports` 使用同端口映射、环境变量写 `.env` 并通过 `env_file` 引用、多 volume 输出、失败后继续补齐同一个已创建应用等设计均已在实现中体现。

## Actual diff summary / 实际差异摘要

跟踪文件 diff 统计：

```text
docs/frontend/reka-ui-llms.txt            | 124 ------------------------------
frontend/src/i18n/locales/en-US.ts        |  55 ++++++++++++-
frontend/src/i18n/locales/zh-CN.ts        |  51 +++++++++++-
frontend/src/router/index.ts              |   6 ++
frontend/src/views/cd/ApplicationPage.vue |  75 +-----------------
5 files changed, 113 insertions(+), 198 deletions(-)
```

未跟踪文件：

```text
docs/frontend/reka-llms.txt
docs/plan/20260626-cd-application-create-methods.md
docs/requirement/20260626-cd-application-create-methods.md
frontend/src/components/ApplicationCreateWizard.vue
frontend/src/utils/applicationCompose.test.ts
frontend/src/utils/applicationCompose.ts
frontend/src/views/cd/ApplicationCreatePage.vue
```

实际功能改动包括：

- `frontend/src/router/index.ts`
  - 新增 `/cd/applications/create` 路由，组件为 `ApplicationCreatePage.vue`。
- `frontend/src/views/cd/ApplicationPage.vue`
  - “创建应用”按钮改为跳转独立创建页面。
  - 移除列表页空白创建模态窗相关逻辑。
  - 保留当前 JSON 导入按钮和导入模态窗。
- `frontend/src/views/cd/ApplicationCreatePage.vue`
  - 新增独立创建页面，展示标题、说明、返回按钮和创建向导。
- `frontend/src/components/ApplicationCreateWizard.vue`
  - 使用 Reka UI Stepper 呈现创建流程。
  - 展示 4 种创建方式。
  - 完整实现“从镜像创建”。
  - 实现高级空白应用原语义创建。
  - 实现失败后继续补齐同一个已创建应用，并提供跳转详情页手动修复。
- `frontend/src/utils/applicationCompose.ts`
  - 实现 `docker-compose.yml` 和 `.env` 生成逻辑。
- `frontend/src/utils/applicationCompose.test.ts`
  - 覆盖 compose/env 生成逻辑。
- `frontend/src/i18n/locales/zh-CN.ts`、`frontend/src/i18n/locales/en-US.ts`
  - 更新创建入口文案和创建向导文案。

## Expected vs actual changed files / 预期与实际改动对比

| 预期范围 | 实际文件 | 结论 |
| --- | --- | --- |
| 独立创建页面 | `frontend/src/views/cd/ApplicationCreatePage.vue` | 符合 |
| 创建向导组件 | `frontend/src/components/ApplicationCreateWizard.vue` | 符合 |
| compose/env 生成工具 | `frontend/src/utils/applicationCompose.ts` | 符合 |
| compose/env 单元测试 | `frontend/src/utils/applicationCompose.test.ts` | 符合 |
| 应用列表页入口跳转 | `frontend/src/views/cd/ApplicationPage.vue` | 符合 |
| 前端路由注册 | `frontend/src/router/index.ts` | 符合 |
| 中英文文案 | `frontend/src/i18n/locales/zh-CN.ts`, `frontend/src/i18n/locales/en-US.ts` | 符合 |
| backend-go | 无修改 | 符合计划 |
| 数据库迁移 | 无修改 | 符合计划 |
| 旧 Python backend | 无修改 | 符合计划 |
| Reka 文档文件 | `docs/frontend/reka-ui-llms.txt` 删除、`docs/frontend/reka-llms.txt` 新增 | 范围外，需要提交前确认是否为预期重命名 |

## Acceptance criteria checklist / 验收清单

- [x] 应用列表页主创建入口跳转到独立创建应用页面。
- [x] 独立创建页面第一步展示 4 种创建方式。
- [x] 创建流程使用 Reka UI Stepper 组件实现步骤展示。
- [x] “从镜像创建”可以填写应用名称、应用代码、镜像、容器端口，并生成 `docker-compose.yml`。
- [x] “从镜像创建”填写环境变量时，系统生成 `.env`，并在 `docker-compose.yml` 中通过 `env_file: .env` 显式引用。
- [x] “从镜像创建”支持多条数据目录映射，并写入 service volumes。
- [x] “从镜像创建”填写域名/路由时，创建后的应用启用路由托管，并创建或更新 service `app` 的路由。
- [x] 前端串联创建应用、配置文件和路由接口时，已处理应用创建成功后重新提交继续补齐同一个应用的场景。
- [x] “高级空白应用”在独立创建页面内使用原字段表单和原创建接口，不生成配置文件。
- [x] “从 docker-compose 创建”“导入应用包”展示后续支持或现有入口说明，不进入半成品提交流程。
- [x] 前端变更后运行 `yarn lint --fix && yarn typecheck` 并通过。
- [x] 新增 compose/env 生成单元测试并通过。
- [x] 不修改 backend-go、旧 Python backend 或已执行迁移文件。

## Command results / 命令结果

### applicationCompose 单元测试

命令：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn test -- applicationCompose
```

结果：通过。

```text
Test Files  1 passed (1)
Tests       5 passed (5)
```

### 前端 lint 与 typecheck

命令：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn lint --fix && yarn typecheck
```

结果：通过。

```text
$ eslint . --cache --fix
Done in 0.85s.
$ vue-tsc --noEmit
Done in 4.96s.
```

## Missed or expanded scope / 范围偏差

- 未做浏览器手工验证；当前验证覆盖静态 diff、单元测试、lint 和 typecheck。
- 未运行 backend-go 测试，因为本次实现没有修改 backend-go。
- 工作区存在 `docs/frontend/reka-ui-llms.txt` 删除和 `docs/frontend/reka-llms.txt` 新增。该变化不属于本次应用创建功能本身，且可能是用户提供 Reka 文档路径造成的文件名差异；提交前需要确认是否作为文档重命名一起提交，或单独处理。

## Risks / 风险

1. 前端串联接口不具备事务性。应用创建成功后如果配置文件或路由补齐失败，当前页面状态可以继续补齐；但浏览器刷新后状态丢失，需要从详情页手动修复。
2. 生成 compose 使用同端口主机映射，可能与宿主机已有端口冲突；这是为了遵从当前 backend-go 通过 `ports` 解析默认端口的最小实现。
3. `.env` 只实现基础 key/value 生成和 key 校验，不支持完整 dotenv 编辑体验。
4. “从 docker-compose 创建”和“导入应用包”当前只展示，不完整落地，后续需要继续拆分实现。
5. `docs/frontend/reka-ui-llms.txt` / `docs/frontend/reka-llms.txt` 的文件名变化如果混入本次提交，可能扩大提交边界。

## Incomplete items / 未完成项

无阻塞本功能交付的未完成项。

提交前建议先处理或确认：

- `docs/frontend/reka-ui-llms.txt` 删除与 `docs/frontend/reka-llms.txt` 新增是否为预期重命名。

## Conclusion / 结论

本次标准模式 / standard 的 Verification 通过。实现与已接受的 requirement 和 plan 中定义的目标、非目标和验收标准一致；前端单元测试、lint 和 typecheck 均通过；未发现阻塞功能交付的问题。

需要注意的是，当前工作区包含一个与应用创建功能无直接关系的 Reka 文档文件名变化，提交前应单独确认或拆分处理。
