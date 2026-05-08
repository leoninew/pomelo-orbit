# Reka UI 风格一致性推进

更新日期：2026-05-08

## 目标

以 `frontend-rekaui` 为范围，逐轮收敛页面元素的组件范式和视觉语言。每一轮只处理一组明确目标，避免复刻旧 `frontend`，重点补齐内容缺失、统一交互骨架、压缩样式分叉。

## 基础约束

- 遵从 `CODE_STANDARDS.md`。
- 开发服务器由用户管理，不主动启动或停止。
- 每次代码变更后执行 `yarn lint --fix` 和 `yarn typecheck`。
- 分页默认保持用户调整后的 `pageSize: 10`。
- 使用 Reka UI 组件和本仓库共享组件优先，不新增无必要抽象。
- 列表页表格使用 `app-table-list`，详情页表格使用 `app-table-detail`。

## 已完成

- 表格体系：
  - 列表页表格统一为 `app-table-list`。
  - 详情页表格统一为 `app-table-detail`。
  - 列表页表格上下留白已收紧，介于紧凑型和旧宽松样式之间。
  - 首页、登录历史、系统设置已对齐列表页表格风格。
- 持续部署：
  - 应用列表、部署记录列表完成搜索框、表格外壳、链接样式收敛。
  - 相关列表页分页保持 `10 条/页`。
- 持续集成：
  - 流水线运行详情补齐变量快照、制品内容。
  - 流水线运行详情阶段编排使用表格展示。
  - 流水线运行详情日志浮层迁移到 `AppDrawer`。
- 共享组件与样式：
  - 新增共享类：`app-surface`、`app-input`、`app-textarea`、`app-select-trigger`、`app-popover-content`、`app-option-item`、`app-link`、`app-tip`、`app-dialog-content`、`app-drawer-content`、`app-toast-root`。
  - `SearchControl`、`SelectControl`、`ComboboxSelect` 已接入共享样式。
  - `AppDialog` 统一 Dialog 外壳，补齐无描述场景的可访问性处理。
  - 新增 `AppDrawer`。
  - 新增 `AppToaster`，全局 toast 改为 Reka Toast。
- 系统设置：
  - 修改密码、重置确认弹窗迁移到 `AppDialog`。
  - 设置页搜索、表格外壳、输入框、按钮、提示样式完成第一轮统一。

## 已验证

- `yarn lint --fix`
- `yarn typecheck`
- `git diff --check`
- Playwright 截图验证：
  - 系统设置页与修改密码弹窗。
  - Reka Toast 错误通知。
  - 持续部署应用列表与部署记录列表。
  - 流水线运行详情与日志抽屉。
  - 窄屏系统设置页。

## 当前轮次

- 目标：继续处理高频手写弹窗和表单控件。
- 已完成：
  - `src/views/ci/CredentialPage.vue`
    - 凭据新建/编辑、删除确认、导入凭据迁移到 `AppDialog`。
    - 输入框、textarea、错误态、按钮、危险链接、表格外壳统一到共享样式。
  - `src/views/ci/RepositoryPage.vue`
    - 新建仓库迁移到 `AppDialog`。
    - 输入框、错误态、按钮、链接、表格外壳统一到共享样式。
    - 手写分页替换为 `ListPagination`，保留 `pageSize: 10`。
    - 列表表头与关键列补充单行展示，避免短标签和操作列换行撑高行高。
  - `src/views/ci/components/WebhookList.vue`
    - Webhook 新建/编辑、删除确认迁移到 `AppDialog`。
    - 表格外壳、标题区、按钮、链接、复制图标按钮、输入框、错误态统一到共享样式。
    - 继续沿用详情页表格 `app-table-detail`。
- 本轮验证：
  - `yarn lint --fix`
  - `yarn typecheck`
  - `git diff --check`
  - Playwright 截图验证：
    - 代码仓库列表与新建仓库弹窗。
    - 仓库详情 Webhook 区块与添加 Webhook 弹窗。
- 下一批优先页面：
  - `src/views/cd/RoutePage.vue`
  - `src/views/ci/BuildStagePage.vue`
  - `src/views/ci/RepositoryDetail.vue`

## 后续队列

- 模态窗：
  - 将仍然手写 `DialogRoot` / `DialogContent` 的页面逐步迁移到 `AppDialog`。
  - 删除确认弹窗统一宽度、标题、说明、footer 按钮。
- 表单控件：
  - 原始输入框迁移到 `app-input` / `app-textarea`。
  - 错误态统一使用 `app-input-error` 与 `app-field-error`。
  - 下拉框优先使用 `SelectControl` / `ComboboxSelect`。
- 链接和操作：
  - 主链接统一使用 `app-link`。
  - 危险操作统一使用危险链接或危险按钮样式。
- 抽屉：
  - Stage 脚本编辑、应用文件编辑等手写抽屉逐步迁移到 `AppDrawer`。
- Tip / Alert：
  - 页面内提示统一到 `app-tip` 或后续封装的 alert 组件。
- 卡片 / Surface：
  - 页面主容器外壳逐步统一到 `app-surface`。
  - 保持详情页信息块和表格块的轻量分隔，不做过度卡片化。
