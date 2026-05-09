# frontend-rekaui 待办

更新日期：2026-05-09

## 范围

当前文档只记录 `frontend-rekaui` 的真实进度和待办事项。风格一致性规范已抽取到 `docs/style.md`。

## 当前状态

- 表格体系已基本收敛：当前扫描到的页面表格均使用 `app-table-list` 或 `app-table-detail`。
- 弹窗和抽屉已基本收敛：页面层手写 `DialogRoot/DialogContent` 已清掉，统一通过 `AppDialog` / `AppDrawer` 使用 Reka UI。
- Toast 已统一到 `AppToaster` / Reka Toast。
- CI/CD 主要列表页和详情页已完成第一轮共享样式收口，包含表格、链接、按钮、输入框、错误态、tip、surface、section header。

## 真实待办

- 继续观察窄屏下 `app-toolbar-simple` 页面是否需要也切到横向滚动；当前优先保证桌面和主要列表页不换行。
- 后续可把状态徽标颜色进一步抽成共享状态类，但这不是当前阻塞项。
- 后续若 `ComboboxSelect` 需要支持更复杂对象值，再单独扩展；当前应用详情已通过 `service_name` 字符串值接入共享组件。
- 确认 `StageEdit.vue` 是否仍有实际入口；若无引用，后续清理该遗留组件，避免维护两套构建阶段编辑体验。

## 2026-05-09 本轮处理

- 新增 `docs/style.md`，承载 `frontend-rekaui` 风格一致性规范。
- `docs/todo.md` 收敛为真实待办和进度记录，不再承载长期风格规范。
- 统一脚本抽屉风格：
  - `BuildStage.vue` 脚本抽屉宽度统一到 `w-[min(960px,100vw)]`。
  - `StageEdit.vue` 脚本抽屉主按钮文案从“确定”统一为“保存”。
- 统一文件抽屉风格：
  - `ApplicationDetail.vue` 配置文件查看/编辑抽屉宽度统一到 `w-[min(960px,100vw)]`。
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
- 建立模态窗风格标准，并完成第一轮收口：
  - 新增 `app-field-hint`，字段级说明从零散 `text-xs text-muted-foreground` 收敛为共享类。
  - 新建/添加/编辑类弹窗的主按钮文案按场景区分为“创建 / 添加 / 保存”。
  - 构建、路由、凭据、变量、Webhook 等表单弹窗的字段块间距和 label 写法统一到 `space-y-1.5` / `app-field-label block`。
  - 移除重复默认宽度和临时窄宽度，复杂 textarea 弹窗使用 `600px` 范式。
- 统一取消模态窗自动 focus 行为：
  - `AppDialog` / `AppDrawer` 统一阻止打开和关闭时的自动 focus。
  - 移除弹窗内 `ComboboxSelect` 的局部 `open-on-focus=false` 特例。
  - `ComboboxSelect` 移除 `openOnFocus` prop，默认不再 focus 展开。

## 验证记录

- 2026-05-09 本轮已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- frontend-rekaui docs/todo.md docs/style.md`
- 2026-05-09 模态窗第一轮已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- docs/todo.md frontend-rekaui`
- 2026-05-09 focus 行为收口已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- docs/todo.md frontend-rekaui`
