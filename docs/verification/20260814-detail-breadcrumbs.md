# 详情页面包屑验证
最后修改时间: 2026-08-14 18:38:00

Review status: Accepted

## Requirement alignment

对照 [docs/requirement/20260814-detail-breadcrumbs.md](../../requirement/20260814-detail-breadcrumbs.md) 核对，已覆盖持续部署与持续集成详情/编辑页面，并保留既有详情页操作区和返回按钮。需要父实体链路的页面自行声明 breadcrumb 数据，未修改 API 路由或后端响应，也未新增父实体请求。

## Spec alignment

不适用。light 模式跳过 Spec 阶段。

## Plan alignment

不适用。light 模式无独立 Plan 文档，按 Requirement 实现。

## Actual diff summary

- 新增共享 [AppBreadcrumb.vue](../../web/src/components/AppBreadcrumb.vue)，提供语义化 `nav`、`ol`、RouterLink 和窄屏横向滚动；当前页由详情页标题呈现。
- 在 [App.vue](../../web/src/App.vue) 主内容区统一渲染面包屑容器。
- 在 [useBreadcrumbs.ts](../../web/src/composables/useBreadcrumbs.ts) 提供页面级 breadcrumb 注入接口；有真实父实体链路的详情页自行声明路径。
- 在中英文 locale 中补充面包屑无障碍标签和节点文案。
- 新增需求阶段记录；未修改用户已有的其他页面、日志或脚本变更。

## Expected vs actual changed files

需求预期的产品代码范围为共享组件、布局、页面级 breadcrumb 接口和中英文文案；实际产品代码改动与该范围一致。工作区还存在以下既有无关变更，未纳入本次实现：`1.log`、`scripts/manage.md`、若干应用/部署/服务/项目页面及其相关文件。

## Acceptance checklist

- [x] 提供共享面包屑组件。
- [x] 使用语义化标记、祖先路径可访问链接；当前页由详情标题承担。
- [x] 持续部署详情/编辑页面接入。
- [x] 持续集成仓库、凭据、流水线、阶段、快照、运行、制品详情接入。
- [x] 深层页面展示祖先集合层级，祖先节点可跳转，且不重复渲染当前页标题。
- [x] 保留标题、状态、操作按钮和既有返回按钮。
- [x] 支持中英文文案。

## Test results

执行命令及结果：

- `yarn --cwd web test`：通过，11 个测试文件、64 个测试全部通过。
- `yarn --cwd web lint`：通过。
- `yarn --cwd web typecheck`：通过。
- `git diff --check`：通过。

## Missed or expanded scope

无功能性漏项。相较最初评估，用户明确要求补充持续集成详情页，因此将 CI 详情全部纳入；这是需求范围内的扩展。

## Risks and incomplete items

- 面包屑祖先节点使用资源集合名称（例如“应用 / 版本 / 组件”），不显示父实体实例名；这是 Requirement 中为避免请求瀑布以及与标题重复确定的约束。
- 本次未启动开发服务器，未执行浏览器视觉验收；需要人工在桌面和窄屏检查间距、横向滚动及各路由跳转。

## Conclusion

代码检查和自动化测试均通过，Acceptance checklist 全部满足。实现与 Requirement 对齐，可交付人工 UI 验收。
