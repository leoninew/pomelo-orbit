# Frontend 风格一致性规范
最后修改时间: 2026-09-30 16:33:56

Doc role: living guide。范围：`web/`。与代码冲突时以代码为准。

更新日期：2026-05-10

## 基础原则

- 范围限定为 `web/`（历史文案中的 `frontend` 指同一前端工程）。
- 遵从 [编码规范](../../CLAUDE.md)；开发服务器由用户管理，不主动启动、停止或重启。
- 优先使用 Reka UI、共享组件和共享样式类，不复刻旧实现。
- 页面层避免直接使用 Reka primitives 组合业务外壳；优先使用项目内共享组件。
- 字段差异是业务差异，不为统一风格强行补齐或删除字段。

## 相关文档

- [前端架构](./architecture.md)
- [Reka UI 使用指南](./reka-ui-guide.md)
- [编码规范](../../CLAUDE.md)

## 表格

- 列表页表格使用 `app-table-list`。
- 详情页和详情内嵌数据表使用 `app-table-detail`。
- 表头高度必须大于等于数据行高度。
- 分页保持 `pageSize: 10`；分页组件优先作为表格区域附属控件使用，不在页面里临时复刻。

## 工具条

- 简单搜索加右侧操作使用 `app-toolbar-simple`。
- 多筛选、多操作且可能横向拥挤的工具条使用 `app-toolbar-scroll` + `app-toolbar-row`。
- 工具条筛选下拉使用 `app-toolbar-select`。
- 搜索框使用 `SearchControl`；可搜索下拉使用 `ComboboxSelect`。
- 带下拉筛选的列表页，筛选值变更后应重置到第一页并立即查询。

## 表单控件

- 文本输入使用 `app-input`，多行文本使用 `app-textarea`。
- 普通下拉使用 `SelectControl`，可搜索选择使用 `ComboboxSelect`。
- `ComboboxSelect` 只处理选项显示、搜索输入、翻页入口与选中事件；远端资源查询和结果映射放在 `RepositorySelect`、`RepositoryCredentialSelect` 等业务组件中。
- 字段块使用 `space-y-1.5`，label 使用 `app-field-label block`。
- 错误态：控件加 `app-input-error`，错误文本使用 `app-field-error text-xs`。
- 字段级说明使用 `app-field-hint`；块状说明或导入摘要使用 `app-tip`。

## 模态窗

- 外壳统一使用 `AppDialog`；页面层不直接使用 Reka `DialogRoot/DialogContent`。
- 默认表单弹窗使用 `AppDialog` 默认宽度 `w-[min(520px,calc(100vw-32px))]`。
- 复杂 textarea 或长表单可显式放宽到 `600px` 或更宽。
- 确认类弹窗使用 `width-class="w-[min(420px,calc(100vw-32px))]"` 和 `body-class="hidden"`。
- 表单主体使用 `space-y-4`。
- Footer 按钮由 `AppDialog` 负责右对齐：
  - 取消用 `app-button`。
  - 普通提交用 `app-button-primary`。
  - 危险确认用 `app-button-destructive`。
- 列表页新建弹窗按钮文案用“创建”；详情页编辑弹窗用“保存”；导入弹窗用“导入”；删除/取消确认用对应危险操作文案。
- `AppDialog` 打开时使用 Reka UI 默认的 focus 管理，确保焦点进入弹窗；关闭时保留统一的 focus 策略。

## 抽屉

- 抽屉外壳统一使用 `AppDrawer`；页面层不直接组合 Reka Dialog/Drawer primitives。
- 表单型抽屉使用默认 body：`min-h-0 flex-1 overflow-y-auto px-6 py-4`。
- 编辑器、日志、文件预览类抽屉使用 `body-class="min-h-0 flex-1 overflow-hidden p-0"` 或 `p-4`，内部自行管理滚动区域。
- 脚本、日志、文件编辑等宽内容抽屉统一使用 `w-[min(960px,100vw)]`；业务确需更窄时再显式说明。
- 抽屉 footer 规则与 `AppDialog` 一致：取消用 `app-button`，保存/创建用 `app-button-primary`。
- 脚本编辑抽屉主按钮统一使用“保存”。
- `AppDrawer` 打开时使用 Reka UI 默认的 focus 管理，确保焦点进入抽屉；关闭时保留统一的 focus 策略。

## Tooltip

- 需要可见悬浮内容的提示统一使用共享 `AppTooltip`，业务组件不直接组合 Reka Tooltip primitives。
- 有补充说明或异常原因时，触发元素提供信息图标和键盘焦点；没有内容时不展示图标或提示浮层。
- Tooltip 通过 Portal 展示，层级高于模态窗及滚动容器，不被表格滚动区域裁切。
- 单行长文本使用共享 `AppTruncatedText`，由调用方约束宽度；仅实际截断时显示虚线下划线，悬停或键盘聚焦展示完整文本，并随容器尺寸和文本变化重新检测。
- 链接等已有交互元素使用 `AppTruncatedText` 的 `as-child` 插槽，让原元素直接触发 Tooltip，沿用原有跳转与键盘焦点。
- 导航、下拉选项等复合交互元素的内部标签使用 `as-child` 保留非聚焦 `span`，提供截断后的悬停提示，不新增内部键盘停靠点。

## Toast

- 全局 toast 统一走 `AppToaster` / Reka Toast。
- 页面内不要自建 toast 容器或临时提示组件。

## Focus 与弹出行为

- `AppDialog` / `AppDrawer` 打开时不要阻止 auto focus，否则触发按钮仍会在被 `aria-hidden` 的背景中保持焦点；关闭行为由共享组件统一管理。
- `ComboboxSelect` 默认只点击展开，不支持 focus 自动展开。

## 日志控件

- CI 阶段、CD 操作与容器、Service、Gateway、Route 统一使用 LogView；抽屉使用 LogDrawer/AppDrawer，宽度 `w-[min(960px,100vw)]`，正文无额外外边距，阅读区域自行滚动。
- 使用共享 Monaco 主题、12px 等宽字体和行号。查找、跟随图标使用 LogActions，放在抽屉标题栏或 CD 日志卡片标题栏，沿用 app-icon-button 和 title 提示；正文不另设状态/工具行，不显示“读取完成”，没有复制、导出、重复刷新 footer。
- 读取异常通过项目共享 Toast 提示，连续重连失败只提示一次；已有内容保留，日志控件不显示异常文案或异常图标。截断/缺口使用独立历史提示图标。查找框悬浮于日志内容，不增加顶部占位。跟随按钮可切换：跟随时为向下贴底图标，停止跟随时为暂停图标。上滚停止跟随，回到底部或点击跟随后恢复；接收过程持续。
- CD 详情操作日志使用 DetailInfoCard 内嵌 LogView，卡片右上角的 ScrollText 日志按钮打开容器日志的 LogDrawer。打开容器抽屉不隐藏或卸载卡片中的操作日志；两类日志各自保有状态与订阅。容器日志在抽屉打开时流式读取，部署进行中也可订阅，关闭后保留页面内缓存；stop 操作不显示容器日志按钮。
