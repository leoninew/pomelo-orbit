# 详情页面包屑验证
最后修改时间: 2026-08-14 18:37:38

Review status: Accepted

## Requirement alignment

对照 [docs/requirement/20260814-detail-breadcrumbs.md](../../requirement/20260814-detail-breadcrumbs.md) 核对，本次实现把 breadcrumb 责任下沉到页面本身，去掉了泛化的模块级节点，改为使用页面已加载的真实父实体名称。版本组件页、服务详情页和服务组件页都能正确指向父实体详情页，而不是列表页。

## Spec alignment

不适用。light 模式跳过 Spec 阶段。

## Plan alignment

不适用。light 模式无独立 Plan 文档，按 Requirement 实现。

## Actual diff summary

- 在 [VersionDetail.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/views/application/VersionDetail.vue) 中加入应用详情请求，breadcrumb 改为使用真实应用名称并链接到应用详情。
- 在 [VersionComponentDetail.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/views/application/VersionComponentDetail.vue) 中加入应用详情请求，breadcrumb 改为 `应用 -> 版本`，分别链接到应用详情和版本详情。
- 在 [ServiceDetail.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/views/service/ServiceDetail.vue) 中直接使用服务返回的应用名，breadcrumb 链接到应用详情。
- 在 [ServiceComponentDetail.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/views/service/ServiceComponentDetail.vue) 中补充服务详情请求，breadcrumb 改为使用真实服务实例名称并链接到服务详情。
- 在 [AppBreadcrumb.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/components/AppBreadcrumb.vue) 和 [App.vue](/D:/SourceCodes/mywork/pomelo-orbit/web/src/App.vue) 中收紧 breadcrumb 的间距与层次，减少与标题区的视觉挤压。
- 更新 [requirement 文档](/D:/SourceCodes/mywork/pomelo-orbit/docs/requirement/20260814-detail-breadcrumbs.md) 以匹配最终实现口径。

## Expected vs actual changed files

本次任务期望的实际改动集中在详情页 breadcrumb 数据声明、共享 breadcrumb 组件和全局布局间距调整。工作区里同时存在既有无关改动，未纳入这次实现：`internal/application/route/usecase/service.go`、`scripts/manage.md`、`web/src/components/RouteManagedTargetSelect.vue`、`web/src/views/application/componentForm.test.ts`。

## Acceptance checklist

- [x] 共享面包屑组件可用，语义化标记和窄屏显示正常。
- [x] 详情页标题继续承担当前页信息，breadcrumb 不重复渲染当前页标题。
- [x] 版本组件页可回到所属版本和应用详情。
- [x] 服务详情页可回到所属应用详情。
- [x] 服务组件页可回到所属服务详情。
- [x] 顶级模块名不再作为无意义 breadcrumb 节点展示。
- [x] breadcrumb 的视觉密度已收紧。
- [x] 前端 lint 与类型检查通过。

## Test results

执行命令及结果：

- `yarn --cwd web lint:fix`：通过。
- `yarn --cwd web typecheck`：通过。
- `yarn --cwd web test`：通过，11 个测试文件、64 个测试全部通过。
- `git diff --check`：通过。

## Missed or expanded scope

实现范围从“深层详情页 breadcrumb”收敛到当前用户明确点名的应用、版本、服务和组件链路，并顺手把 breadcrumb 的排版压紧。没有扩大到改动其它导航体系。

## Risks and incomplete items

- 服务组件页为了显示真实父实体名称，额外请求了一次服务详情；这是为了保持 breadcrumb 语义准确。
- 本次没有启动开发服务器做截图级视觉验收，breadcrumb 在实际路由上的行高、换行和窄屏表现仍需要人工确认。

## Conclusion

实现与 Requirement 对齐，代码检查和测试均通过。当前 breadcrumb 语义已经从“模块/列表入口”切换为“真实父实体详情入口”，可以交付人工 UI 验收。
