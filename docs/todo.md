# Reka UI 风格一致性推进

更新日期：2026-05-09

## 范围

当前文档只记录 `frontend-rekaui` 的真实进度和剩余观察项。历史旧队列已清理，不再保留已过期的 P0/P1 摘录。

## 固定约束

- 遵从 `CODE_STANDARDS.md`。
- 开发服务器由用户管理，不主动启动、停止或重启。
- 每次代码变更后执行 `yarn lint --fix` 和 `yarn typecheck`。
- 分页保持 `pageSize: 10`。
- 列表页表格使用 `app-table-list`，详情页表格使用 `app-table-detail`。
- 优先使用 Reka UI、共享组件和共享样式类，不复刻旧 `frontend`。

## 当前状态

- 表格体系已基本收敛：当前扫描到的页面表格均使用 `app-table-list` 或 `app-table-detail`。
- 弹窗和抽屉已基本收敛：页面层手写 `DialogRoot/DialogContent` 已清掉，统一通过 `AppDialog` / `AppDrawer` 使用 Reka UI。
- Toast 已统一到 `AppToaster` / Reka Toast。
- CI/CD 主要列表页和详情页已完成第一轮共享样式收口，包含表格、链接、按钮、输入框、错误态、tip、surface、section header。
- `ComboboxSelect` 已支持 `openOnFocus`，Dialog 首控件场景可关闭自动展开。

## 2026-05-09 本轮处理

- 修复 `DeploymentDetail.vue` 应用详情链接，前端路由从错误的 `/cd/application/:id` 改为 `/cd/applications/:id`。
- 修复 `PipelineRunDetail.vue` Stage 日志抽屉状态：
  - 区分加载中、等待日志输出、无日志、加载失败。
  - 加载失败提供重试入口。
  - 避免终态无日志时一直显示“加载日志中...”。
- 收口 `ApplicationDetail.vue` 应用路由弹窗的服务选择：
  - 页面内手写 Reka Combobox 替换为共享 `ComboboxSelect`。
  - `ComboboxSelect` 增加 `invalid` 错误态入口。
- 收口 CI/CD 列表工具条样式：
  - 新增 `app-toolbar-simple`、`app-toolbar-scroll`、`app-toolbar-row`。
  - 多筛选/多操作工具条使用横向滚动范式，避免换行撑高。
  - 简单搜索+操作工具条使用简单范式，减少逐页 class 分叉。

## 剩余观察项

- 继续观察窄屏下 `app-toolbar-simple` 页面是否需要也切到横向滚动；当前优先保证桌面和主要列表页不换行。
- 后续可把状态徽标颜色进一步抽成共享状态类，但这不是当前阻塞项。
- 后续若 `ComboboxSelect` 需要支持更复杂对象值，再单独扩展；当前应用详情已通过 `service_name` 字符串值接入共享组件。

## 验证记录

- 2026-05-09 本轮已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- frontend-rekaui docs/todo.md`
