# /projects i18n 覆盖率改进 Verification

## Status

Accepted

## Verification scope

本次验证覆盖 `/projects` 列表页及其直接渲染链路中的共享组件：

- `frontend/src/views/ProjectPage.vue`
- `frontend/src/components/SearchControl.vue`
- `frontend/src/components/ListPagination.vue`
- `frontend/src/components/AppEmptyState.vue`
- `frontend/src/components/AppDialog.vue`
- `frontend/src/components/AppBadge.vue`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`

## Requirement alignment

### 通过

- `/projects` 页面自身用户可见文案已替换为 i18n key。
- `/projects` 直接使用的 `SearchControl.vue` 搜索按钮、清空按钮 aria-label、默认 placeholder 已改为 i18n。
- `/projects` 直接使用的 `ListPagination.vue` 页大小选项已改为 i18n。
- `zh-CN.ts` 和 `en-US.ts` 已补齐同名 key。
- 未改造 `/projects/:id` 详情页，符合当前 scope assumption。

### 过程发现

初始实现只扫描了 `ProjectPage.vue`，遗漏 `SearchControl.vue` 中的搜索按钮文案。后续 Verification 扩大到 `/projects` 渲染链路后，又发现 `ListPagination.vue` 的页大小选项 `${size} 条/页`。两处均已修复。

## Spec alignment

### 通过

- 使用现有 `vue-i18n` Composition API：`useI18n()`。
- 页面专用文案放在 `project` namespace。
- 通用文案补充在 `common` namespace：
  - `common.clearSearch`
  - `common.perPage`
- 保留 API、store、router、页面数据流和交互行为不变。

### 与原 spec/plan 的偏差

原 plan 限定只修改 3 个文件，后因直接相关组件存在用户可见硬编码文案，范围扩展为 5 个文件：

- `frontend/src/views/ProjectPage.vue`
- `frontend/src/components/SearchControl.vue`
- `frontend/src/components/ListPagination.vue`
- `frontend/src/i18n/locales/zh-CN.ts`
- `frontend/src/i18n/locales/en-US.ts`

`docs/projects-i18n/plan.md` 已同步更新该范围。

## Plan alignment

最终实现与更新后的 plan 一致：

- `ProjectPage.vue` 硬编码中文已替换为 `t(...)`。
- `SearchControl.vue` 直接可见文案已使用 `common.search` / `common.clearSearch`。
- `ListPagination.vue` 页大小标签已使用 `common.perPage`。
- Locale key 在中英文文件中保持一致。

## Test results

### Static i18n checks

对 `/projects` 渲染链路关键文件执行中文字符扫描：

- `ProjectPage.vue`：无匹配
- `SearchControl.vue`：无匹配
- `AppEmptyState.vue`：无匹配
- `AppDialog.vue`：无匹配
- `ListPagination.vue`：仅剩中文注释，无用户可见文案

### Frontend checks

执行命令：

```bash
yarn --cwd frontend lint --fix && yarn --cwd frontend typecheck
```

结果：通过。

## Risks

- `Requirement`、`Spec`、`Plan` 文档仍是 Draft，未由用户明确标记为 Approved。
- 本次只覆盖 `/projects` 列表页及其直接渲染链路；其他页面或非 `/projects` 使用的共享组件中文文案不在本次验收范围。
- `ListPagination.vue` 中仍有中文注释，但不影响用户界面 i18n 覆盖率。

## Incomplete items

无本范围内未完成项。

## Conclusion

通过。当前实现满足 `/projects` 列表页及其直接渲染链路的 i18n 覆盖率改进目标，并通过前端 lint 与 typecheck。
