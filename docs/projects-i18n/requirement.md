# /projects i18n 覆盖率改进 Requirement

## Status

Draft

## Background

`frontend` 项目已经有 i18n 机制，包含 `frontend/src/i18n/locales/zh-CN.ts` 和 `frontend/src/i18n/locales/en-US.ts`。`/projects` 页面当前仍存在大量用户可见硬编码中文文案，例如工具栏 aria-label、搜索 placeholder、按钮、表头、状态标签、弹窗标题、表单标签、校验错误、提示文案和 toast。

这会导致语言切换时 `/projects` 页面无法完整跟随当前语言，影响国际化一致性。

## Goals

- 提高 `/projects` 页面用户可见文案的 i18n 覆盖率。
- 将 `/projects` 页面硬编码文案迁移到现有 locale 文件。
- 使用项目现有 i18n 调用方式替换硬编码文案。
- 保持 `/projects` 页面现有功能、布局和交互行为不变。

## Non-goals

- 不改造其他页面的 i18n 覆盖率。
- 不重构 i18n 架构。
- 不新增语言种类。
- 不引入新的 i18n 依赖。
- 不调整 UI 样式、布局或交互流程。
- 不处理非用户可见的变量名、类型名、内部常量名。

## Scope

### In scope

- `frontend/src/views/ProjectPage.vue`
- `/projects` 页面所需的 locale key 补充：
  - `frontend/src/i18n/locales/zh-CN.ts`
  - `frontend/src/i18n/locales/en-US.ts`
- 页面内用户可见文案，包括：
  - toolbar aria-label
  - search placeholder
  - create/edit/deprecate/save/cancel 等按钮与操作文案
  - table header
  - active/deprecated 状态标签
  - dialog title 和确认说明
  - form label、placeholder、hint、validation message
  - loading error fallback
  - create/update/deprecate success toast

### Out of scope

- `ProjectDetail.vue` 或 `/projects/:id` 详情页，除非后续用户明确纳入。
- 全局共享组件的默认文案改造，除非 `/projects` 页面必须显式传入翻译文案。
- 后端 API、数据模型和权限逻辑。

## User scenarios

1. 用户切换为中文后访问 `/projects`，页面所有主要操作文案显示为中文。
2. 用户切换为英文后访问 `/projects`，页面所有主要操作文案显示为英文。
3. 用户创建、编辑或废弃项目时，弹窗、校验提示和 toast 使用当前语言。
4. 用户搜索项目时，搜索输入框 placeholder 使用当前语言。

## Acceptance criteria

- `/projects` 页面中不再存在用户可见的硬编码中文文案。
- `/projects` 页面新增或使用的翻译 key 同时存在于 `zh-CN.ts` 和 `en-US.ts`。
- 语言切换后，页面标题、按钮、表头、状态、弹窗、表单提示、校验错误和 toast 能使用当前语言显示。
- 不改变项目列表加载、搜索、分页、创建、编辑、废弃和跳转行为。
- 前端变更后通过项目要求的检查：`yarn lint --fix && yarn typecheck`。

## Open questions

- 是否只处理 `/projects` 列表页，不处理 `/projects/:id` 详情页？当前假设：只处理列表页。
- `AppEmptyState` 默认文案是否需要在本次范围内显式传入 `/projects` 相关翻译？当前假设：如果组件支持传参且页面显示相关，则纳入；否则不改共享组件。

## Decisions

- 使用现有 i18n 机制，不新增依赖或新架构。
- locale key 优先放在现有 `project` 命名空间下；通用操作文案优先复用 `common` 命名空间。
- 当前阶段只产出 requirement 文档，不修改产品代码。

## User review notes

- 待用户 review。
