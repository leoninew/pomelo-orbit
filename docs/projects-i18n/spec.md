# /projects i18n 覆盖率改进 Spec

## Status

Draft

## Requirement basis

基于 `docs/projects-i18n/requirement.md` Draft 版本编写。

注意：Requirement 尚未标记为 `Approved`。本 spec 继续推进时采用以下 assumptions：

- 只处理 `/projects` 列表页，即 `frontend/src/views/ProjectPage.vue`。
- 不处理 `/projects/:id` 对应的 `ProjectDetail.vue`。
- 不改共享组件实现；如共享组件已有 prop 支持，则由页面传入翻译后的文案。

## Overview

`/projects` 页面当前直接写入中文用户可见文案。实现时在 `ProjectPage.vue` 中引入 `useI18n`，通过 `t(...)` 渲染页面文案；在 `zh-CN.ts` 和 `en-US.ts` 的现有 `project` namespace 下补充页面专用翻译 key，并尽量复用 `common` namespace 中已有通用 key。

目标是让语言切换后 `/projects` 页面主要文案跟随当前 locale，同时不改变页面数据流、交互、布局和 API 调用。

## Design decisions

### 1. i18n 调用方式

- 使用项目现有 `vue-i18n` Composition API：`const { t } = useI18n()`。
- 模板中直接使用 `t('...')`。
- script 中校验错误和 toast 使用 `t('...')`。

理由：项目已有页面（例如 `UserPage.vue`）采用该模式，保持一致。

### 2. locale key 组织

- 页面专用文案放到现有 `project` namespace。
- 通用文案优先复用 `common`：
  - `common.cancel`
  - `common.save`
  - `common.operation`
  - `common.status`
  - `common.createdAt`
  - `common.updatedAt`
- 如现有 `common` 语义不足，则新增到 `project`。

建议新增或补齐的 `project` key：

```ts
project: {
  currentProject: string,
  projectManagement: string,
  noProjects: string,
  toolbar: string,
  searchPlaceholder: string,
  create: string,
  edit: string,
  createTitle: string,
  editTitle: string,
  name: string,
  code: string,
  active: string,
  deprecated: string,
  deprecate: string,
  deprecateTitle: string,
  deprecateConfirm: string,
  codeHint: string,
  nameRequired: string,
  codeInvalid: string,
  loadFailed: string,
  created: string,
  updated: string,
  deprecatedToast: string,
}
```

其中 `deprecateConfirm` 需要插值项目名称，例如：

- zh-CN: `确定要废弃项目 {name} 吗？废弃后将无法切换到该项目。`
- en-US: `Deprecate project {name}? You will no longer be able to switch to it after deprecation.`

### 3. AppEmptyState 使用

`AppEmptyState.vue` 已支持 `message?: string`，默认使用 `common.noData`。`/projects` 页面应显式传入 `t('project.noProjects')`，不修改共享组件。

### 4. 表单校验

现有 `validate()` 仍保留同步校验逻辑，只把错误字符串替换为 `t('project.nameRequired')` 和 `t('project.codeInvalid')`。

不在本次改为 HTML5 validation，因为本需求是 i18n 覆盖率改进，避免改变现有提交行为。

### 5. toast 和 fallback error

- `toast.success('项目已更新')` → `toast.success(t('project.updated'))`
- `toast.success('项目已创建')` → `toast.success(t('project.created'))`
- `toast.success('项目已废弃')` → `toast.success(t('project.deprecatedToast'))`
- `error || '加载失败'` → `error || t('project.loadFailed')`

保留 `error` 原样展示，不翻译后端返回内容。

## Affected components

### 修改

- `frontend/src/views/ProjectPage.vue`
  - 引入 `useI18n`
  - 添加 `const { t } = useI18n()`
  - 替换页面硬编码用户可见文案
  - 对 `AppEmptyState` 传入项目空状态文案

- `frontend/src/i18n/locales/zh-CN.ts`
  - 在 `project` namespace 补充 `/projects` 页面中文文案

- `frontend/src/i18n/locales/en-US.ts`
  - 在 `project` namespace 补充对应英文文案

### 不修改

- `frontend/src/views/ProjectDetail.vue`
- `frontend/src/components/AppEmptyState.vue`
- 后端 API 和 store 实现

## Interfaces

不新增外部接口，不改变 API request/response，不改变路由。

页面内部新增的 i18n key 是前端 locale 资源接口的一部分，需要保证 `zh-CN.ts` 与 `en-US.ts` key 对齐。

## Technical questions

- Requirement 未 approved；当前按用户“进入 spec”的指令继续推进。
- 是否把 `project.projectManagement` 复用于创建标题以外的页面标题：当前页面没有显式标题，不新增标题。

## Risks

- 如果后续要求同时覆盖 `ProjectDetail.vue`，本 spec 范围需要扩大。
- 如果新增 key 命名与未来项目管理其他页面冲突，可能需要进一步细分 namespace；当前为最小变更，仍使用 `project`。
- `deprecateConfirm` 插值如果直接包含 `<strong>` 会复杂化模板；建议拆成纯文本插值，不保留项目名加粗，或在模板中使用分段文案。为保持现有视觉，实施时可使用 `deprecateConfirmPrefix` / `deprecateConfirmSuffix` 两个 key 包裹 `<strong>`。

## Alternatives

1. **新增 `projectPage` namespace**
   - 优点：页面级 key 更清晰。
   - 缺点：现有已有 `project` namespace，新增 namespace 对本次最小变更没有必要。

2. **修改共享 `AppEmptyState` 支持 i18n key prop**
   - 优点：调用方可传 key。
   - 缺点：需要改共享组件，不符合本次范围。

3. **一次性改造项目详情页**
   - 优点：项目模块 i18n 更完整。
   - 缺点：超出用户当前 `/projects` 范围。

## User review notes

- 待用户 review。
