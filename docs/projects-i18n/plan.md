# /projects i18n 覆盖率改进 Plan

## Status

Draft

## Requirement / Spec status

- Requirement: `docs/projects-i18n/requirement.md`，Status = Draft
- Spec: `docs/projects-i18n/spec.md`，Status = Draft

用户明确要求进入 Plan，因此继续推进。由于前置文档未 Approved，本 plan 将相关不确定性作为 assumptions / risks 记录。

## Assumptions

- 本次只改 `/projects` 列表页，不改 `/projects/:id` 详情页。
- 使用现有 `vue-i18n` Composition API，不新增依赖。
- 页面专用翻译 key 放在现有 `project` namespace。
- 不修改共享组件实现；`AppEmptyState` 通过已有 `message` prop 传入翻译文案。
- 保留废弃项目确认弹窗中项目名加粗的视觉效果，因此使用 prefix/suffix 两段翻译包裹 `<strong>`。

## Files to change

1. `frontend/src/views/ProjectPage.vue`
2. `frontend/src/components/SearchControl.vue`
3. `frontend/src/components/ListPagination.vue`
4. `frontend/src/i18n/locales/zh-CN.ts`
5. `frontend/src/i18n/locales/en-US.ts`

不修改：

- `frontend/src/views/ProjectDetail.vue`
- store、API、router、后端代码

## Implementation steps

### Step 1: 补充中文 locale key

在 `frontend/src/i18n/locales/zh-CN.ts` 的 `project` namespace 下补充 `/projects` 页面需要的 key：

- `toolbar`
- `searchPlaceholder`
- `create`
- `createTitle`
- `editTitle`
- `name`
- `code`
- `active`
- `deprecated`
- `deprecate`
- `deprecateTitle`
- `deprecateConfirmPrefix`
- `deprecateConfirmSuffix`
- `codeHint`
- `nameRequired`
- `codeInvalid`
- `loadFailed`
- `created`
- `updated`
- `deprecatedToast`

保留现有 key：

- `currentProject`
- `projectManagement`
- `noProjects`

### Step 2: 补充英文 locale key

在 `frontend/src/i18n/locales/en-US.ts` 的 `project` namespace 下补充同名 key，确保 `zh-CN.ts` 与 `en-US.ts` key 对齐。

### Step 3: 改造 `ProjectPage.vue` 模板文案

- 引入并使用 `useI18n`。
- 替换 toolbar aria-label：`t('project.toolbar')`。
- 替换 SearchControl placeholder：`t('project.searchPlaceholder')`。
- 替换按钮、表头、状态 badge、dialog title、form label、hint、footer button 文案。
- `AppEmptyState` 改为传入 `:message="t('project.noProjects')"`。
- 废弃确认弹窗使用：
  - `t('project.deprecateConfirmPrefix')`
  - `<strong>{{ deprecatingProject?.name }}</strong>`
  - `t('project.deprecateConfirmSuffix')`

### Step 4: 改造 `ProjectPage.vue` script 文案

- `validate()` 中错误信息改为 `t(...)`。
- `error || '加载失败'` 改为 `error || t('project.loadFailed')`。
- success toast 改为：
  - `t('project.updated')`
  - `t('project.created')`
  - `t('project.deprecatedToast')`

### Step 5: 修复直接相关搜索组件文案

`SearchControl.vue` 是 `/projects` 页面直接使用组件，其搜索按钮、清空按钮 aria-label 和默认 placeholder 属于页面渲染链路上的用户可见文案：

- 使用 `useI18n()` 读取 `common.search`。
- 新增并使用 `common.clearSearch`。
- 默认 placeholder 改为 `placeholder || t('common.search')`，保留调用方传入自定义 placeholder 的能力。

### Step 6: 修复直接相关分页组件文案

`ListPagination.vue` 是 `/projects` 页面直接使用组件，其页大小选项属于页面渲染链路上的用户可见文案：

- 使用 `useI18n()` 读取 `common.perPage`。
- 新增并使用 `common.perPage`，通过 `{ size }` 插值生成页大小标签。

### Step 7: 自检 i18n 覆盖率

检查 `ProjectPage.vue` 中是否仍存在用户可见硬编码中文。

允许保留：

- 示例 placeholder 值 `Default Project` / `default`，因为这是表单示例值，不是中文硬编码；如实现时认为应国际化，也可改为 locale key。
- 数据字段本身，如 `project.name` / `project.code`。

## Verification plan

按项目要求在前端目录执行：

```bash
yarn lint --fix && yarn typecheck
```

补充人工/静态检查：

- 确认 `ProjectPage.vue` 用户可见中文文案已替换为 `t(...)`。
- 确认新增 key 在 `zh-CN.ts` 和 `en-US.ts` 中同名存在。
- 确认没有修改 API/store/router 行为。

## Blockers

无已知技术 blocker。

## Risks

- Requirement 和 Spec 尚未 Approved，后续如果用户改变范围，可能需要回退或扩展本 plan。
- 如果用户希望 placeholder 示例值也完全国际化，需要额外补充 key。
- `ProjectPage.vue` 当前使用 `ToolbarRoot`，页面层直接使用 Reka primitive 是既有代码；本计划不改该结构，避免超范围重构。

## Rollback

如实现后不符合预期，回滚以下 3 个文件的改动即可：

- `frontend/src/views/ProjectPage.vue`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`

## User review notes

- 待用户 review。
