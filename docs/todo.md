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

## 模态窗风格标准

- 外壳统一使用 `AppDialog`；抽屉型编辑使用 `AppDrawer`。页面层不再直接使用 Reka `DialogRoot/DialogContent`。
- 默认表单弹窗使用 `AppDialog` 默认宽度 `w-[min(520px,calc(100vw-32px))]`；复杂 textarea 或长表单才显式放宽到 `600px` 或更宽。
- 确认类弹窗使用 `width-class="w-[min(420px,calc(100vw-32px))]"` 和 `body-class="hidden"`。
- 表单主体使用 `space-y-4`；字段块使用 `space-y-1.5`；label 使用 `app-field-label block`。
- 输入控件使用 `app-input` / `app-textarea`，下拉使用 `SelectControl`，可搜索选择使用 `ComboboxSelect`。
- 错误态：控件加 `app-input-error`，错误文本使用 `app-field-error text-xs`。
- 字段级说明使用 `app-field-hint`；块状说明或导入摘要使用 `app-tip`。
- Footer 按钮右对齐由 `AppDialog` 负责：取消用 `app-button`，普通提交用 `app-button-primary`，危险确认用 `app-button-destructive`。
- 列表页新建弹窗按钮文案用“创建”；详情页编辑弹窗按钮文案用“保存”；导入弹窗用“导入”；删除/取消确认用对应危险操作文案。
- 模态窗和抽屉统一取消自动 focus：`AppDialog` / `AppDrawer` 在 `open-auto-focus` 与 `close-auto-focus` 上 prevent；`ComboboxSelect` 不再支持 focus 时自动展开，只保留点击展开。
- 字段差异是业务差异，不为统一风格强行补齐或删除字段。

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
- 建立模态窗风格标准，并完成第一轮收口：
  - 新增 `app-field-hint`，字段级说明从零散 `text-xs text-muted-foreground` 收敛为共享类。
  - 新建/添加/编辑类弹窗的主按钮文案按场景区分为“创建 / 添加 / 保存”。
  - 构建、路由、凭据、变量、Webhook 等表单弹窗的字段块间距和 label 写法统一到 `space-y-1.5` / `app-field-label block`。
  - 移除重复默认宽度和临时窄宽度，复杂 textarea 弹窗使用 `600px` 范式。
- 统一取消模态窗自动 focus 行为：
  - `AppDialog` / `AppDrawer` 统一阻止打开和关闭时的自动 focus。
  - 移除弹窗内 `ComboboxSelect` 的局部 `open-on-focus=false` 特例。
  - `ComboboxSelect` 移除 `openOnFocus` prop，默认不再 focus 展开。

## 剩余观察项

- 继续观察窄屏下 `app-toolbar-simple` 页面是否需要也切到横向滚动；当前优先保证桌面和主要列表页不换行。
- 后续可把状态徽标颜色进一步抽成共享状态类，但这不是当前阻塞项。
- 后续若 `ComboboxSelect` 需要支持更复杂对象值，再单独扩展；当前应用详情已通过 `service_name` 字符串值接入共享组件。

## 验证记录

- 2026-05-09 本轮已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- frontend-rekaui docs/todo.md`
- 2026-05-09 模态窗第一轮已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- docs/todo.md frontend-rekaui`
- 2026-05-09 focus 行为收口已执行：
  - `cd frontend-rekaui && yarn lint --fix`
  - `cd frontend-rekaui && yarn typecheck`
  - `git diff --check -- docs/todo.md frontend-rekaui`
