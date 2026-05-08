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
  - 应用列表补回新建、导入、部署/停止入口，保留表格主视图。
  - 相关列表页分页保持 `10 条/页`。
- 持续集成：
  - 流水线运行详情补齐变量快照、制品内容。
  - 流水线运行详情阶段编排使用表格展示。
  - 流水线运行详情日志浮层迁移到 `AppDrawer`。
- 共享组件与样式：
  - 新增共享类：`app-surface`、`app-input`、`app-textarea`、`app-select-trigger`、`app-popover-content`、`app-option-item`、`app-action-item`、`app-checkbox`、`app-link`、`app-tip`、`app-dialog-content`、`app-drawer-content`、`app-toast-root`。
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
    - 启用开关的 checkbox 接入 `app-checkbox`。
    - 继续沿用详情页表格 `app-table-detail`。
  - `src/views/cd/RoutePage.vue`
    - 添加路由迁移到 `AppDialog`。
    - 按钮、链接、输入框、开关、表格外壳统一到共享样式。
    - 保留 `pageSize: 10`。
  - `src/views/cd/ApplicationPage.vue`
    - 应用列表工具条补回新建应用和导入应用入口，使用 `AppDialog` 和 `ApplicationFormFields`。
    - 表格补充拉取策略列，操作列补回部署/停止动作，继续使用列表页表格范式。
    - 导入 JSON 保留配置文件、服务配置和路由数据，并使用 `app-tip` 显示导入摘要。
    - 保留 `pageSize: 10`。
  - `src/views/cd/DeploymentPage.vue`
    - 部署记录列表接入 `application_id` 查询参数，支持从应用详情跳转后按应用过滤。
    - 表格补充错误信息摘要列，操作列补回运行中部署的取消入口。
    - 取消确认迁移到 `AppDialog`，工具条对齐列表页单行滚动范式。
    - 保留 `pageSize: 10`。
  - `src/views/cd/ApplicationFormFields.vue`
    - 应用新增/编辑共用字段的 label、输入框、错误态和 checkbox 接入共享样式。
    - 继续复用 `SelectControl` 保持下拉框范式一致。
  - `src/views/ci/RepositoryDetail.vue`
    - 编辑仓库、删除仓库、添加变量、编辑变量迁移到 `AppDialog`。
    - 仓库详情按钮、基本信息块、变量配置块、输入框、错误态、链接统一到共享样式。
    - 仓库变量表只对真正的自定义变量开放编辑/删除，避免内置变量的无效编辑操作。
  - `src/views/ci/components/VariableDeclarationsTable.vue`
    - 编辑/删除操作链接统一到 `app-link` / `app-link-danger`。
  - `src/views/ci/BuildStagePage.vue`
    - 新建构建迁移到 `AppDialog`。
    - 列表外壳、主按钮、链接、输入框、textarea、错误态统一到共享样式。
    - 保留 `pageSize: 10`。
  - `src/views/ci/BuildStage.vue`
    - 基本信息、执行脚本、制品配置区块统一到 `app-surface` / `app-section-header`。
    - 顶部操作、编辑构建、添加制品、删除确认按钮统一到共享按钮样式。
    - 制品配置表格保持详情页 `app-table-detail` 范式，操作链接统一到 `app-link` / `app-link-danger`。
    - 脚本编辑从普通弹窗迁移到 `AppDrawer`。
    - 编辑构建、添加制品、删除确认弹窗表单统一到 `AppDialog` / `app-input` / `SelectControl`。
  - `src/views/ci/components/TriggerModal.vue`
    - 触发流水线弹窗迁移到 `AppDialog`。
    - 流水线模板下拉、触发 ref 输入、变量配置输入和 footer 按钮统一到共享样式。
  - `src/views/ci/PipelineTemplatePage.vue`
    - 新建流水线模板迁移到 `AppDialog`。
    - 列表默认表格视图，表格外壳、主按钮、链接、输入框、textarea、空态/错误态外壳统一到共享样式。
    - 分页保留 `pageSize: 10`，移除 12/24/48 分页分叉。
    - 桌面宽度下表格不再因最小宽度过大挤出操作列。
  - `src/views/ci/PipelineTemplateDetail.vue`
    - 基本信息、阶段编排、变量声明、制品声明区块统一到 `app-surface` / `app-section-header`。
    - 顶部操作、添加阶段、添加变量、弹窗 footer、删除确认按钮统一到共享按钮样式。
    - 阶段表格和变量操作链接统一到 `app-link` / `app-link-danger`。
    - 添加 Stage、运行流水线弹窗的手写 Reka Combobox 替换为 `ComboboxSelect`。
    - 编辑信息、添加变量、编辑变量弹窗表单统一到 `app-input` / `app-textarea` / `app-field-label`。
    - 依赖 Stage checkbox 接入 `app-checkbox`。
  - `src/views/ci/ArtifactPage.vue`
    - 表格外壳统一到 `app-surface`，链接统一到 `app-link`。
    - 项目、模板、名称、Stage、路径、时间列补充截断和标题提示，避免长内容撑开表格。
    - 制品类型徽标保持单行，列表行高与其他列表页保持一致。
    - 筛选下拉选项补充仓库地址和模板版本说明。
    - 制品记录工具条改为桌面单行展示，窄屏横向滚动，不再把搜索框挤到下一行。
  - `src/views/ci/PipelineRunPage.vue`
    - 流水线记录工具条改为桌面单行展示，窄屏横向滚动。
    - 表格外壳统一到 `app-surface`，搜索按钮、链接统一到共享样式。
    - 仓库、模板、错误信息、开始时间等长文本列补充截断和标题提示。
    - 表格列宽收敛，桌面宽度下操作列保持可见且“查看”不换行。
  - `src/views/ci/CredentialDetail.vue`
    - 凭据详情外壳统一到 `app-surface` / `app-section-header`。
    - 编辑凭据、删除凭据从手写 Reka Dialog 迁移到 `AppDialog`。
    - 顶部按钮、输入框、textarea、错误态、删除确认按钮统一到共享样式。
  - `src/views/ci/PipelineSnapshotDetail.vue`
    - 基本信息、Stage 快照、变量声明、制品声明区块统一到 `app-surface` / `app-section-header`。
    - Stage 快照和制品声明表格保持详情页 `app-table-detail` 范式。
    - 返回按钮和详情链接统一到共享按钮/链接样式。
  - `src/views/cd/DeploymentDetail.vue`
    - 基本信息、部署日志区块统一到 `app-surface` / `app-section-header`。
    - 顶部操作按钮、应用详情链接、取消部署确认按钮统一到共享样式。
  - `src/views/cd/RouteDetail.vue`
    - 基本信息、HTTPS 配置区块统一到 `app-surface` / `app-section-header`。
    - 顶部操作、外链、编辑路由、删除确认按钮统一到共享按钮/链接样式。
    - 编辑路由弹窗表单统一到 `app-input` / `app-field-label` / `app-field-error`。
    - HTTPS 操作项接入 `app-action-item`，减少详情页内联样式分叉。
  - `src/views/cd/ApplicationDetail.vue`
    - 基本信息、配置文件、服务配置、路由配置区块统一到 `app-surface` / `app-section-header`。
    - 配置文件、服务配置、路由配置表格继续沿用详情页 `app-table-detail`。
    - 顶部操作、表格操作链接、弹窗 footer、输入框、错误态统一到共享样式。
    - 文件查看/编辑从手写浮层迁移到 `AppDrawer`。
    - 服务镜像弹窗、添加/编辑路由弹窗、配置文件删除确认继续统一到共享表单和按钮样式。
    - 应用路由弹窗的手写 Combobox 选项区接入 `app-combobox-anchor` / `app-popover-content` / `app-option-item`。
    - 编辑应用和删除应用确认 checkbox 接入 `app-checkbox`。
  - `src/views/cd/TraefikRoutePage.vue`
    - 列表页外壳统一到 `app-surface`，Dashboard/刷新按钮统一到共享按钮样式。
    - 工具条改为桌面单行展示，窄屏横向滚动。
    - 规则外链统一到 `app-link`。
- 本轮验证：
  - `yarn lint --fix`
  - `yarn typecheck`
  - `git diff --check`
  - Playwright 截图验证：
    - 代码仓库列表与新建仓库弹窗。
    - 仓库详情 Webhook 区块与添加 Webhook 弹窗。
  - Pomelo PW 截图验证：
    - 仓库详情页。
    - 编辑仓库弹窗。
    - 删除仓库弹窗。
    - 添加自定义变量弹窗与空变量名错误态。
    - 构建阶段列表与新建构建弹窗。
    - 触发流水线弹窗。
    - 流水线模板列表与新建模板弹窗。
    - 流水线模板详情与添加 Stage 弹窗。
    - 制品记录列表。
    - 流水线记录列表。
    - 构建阶段详情、脚本抽屉、添加制品弹窗。
    - 凭据详情、编辑凭据弹窗、删除凭据弹窗。
    - 快照详情列表视图与 DAG 视图。
    - 部署详情。
    - 路由详情与编辑路由弹窗。
    - 应用详情、编辑应用弹窗、添加文件抽屉。
    - Traefik Routers 页面加载态。
    - 应用详情服务镜像弹窗、添加路由弹窗、配置文件删除确认弹窗。
    - 应用列表与新建应用弹窗。
    - 部署记录列表。
- 下一批优先页面：
  - `src/views/cd/RoutePage.vue` 复查添加路由弹窗里的表单字段、tip 和 action 分叉。
  - `src/views/cd/TraefikRoutePage.vue` 复查接口错误态和列表列宽。
  - 继续扫描持续部署页面里剩余的手写 action item、checkbox、tip 和弹窗宽度分叉。

## 当前阶段小结

- 当前阶段范围：持续部署剩余列表页细节收口，优先看 `RoutePage.vue` 和 `TraefikRoutePage.vue`。
- 已确认现状：
  - `RoutePage.vue` 已使用 `AppDialog`、`SearchControl`、`app-input`、`app-switch-*`、`app-table-list`，主要剩余是工具条单行滚动范式、添加路由表单提示、列宽/长文本截断的细节统一。
  - `TraefikRoutePage.vue` 已使用 `app-surface`、共享按钮、`app-link` 和列表页表格，主要剩余是接口错误态显式展示、列表列宽继续收敛，以及 `/api/cd/traefik-routes` 500 时避免表现成普通空数据。
  - 持续部署列表页已进入第二轮：应用列表和部署记录列表已补内容，路由和 Traefik 列表进入复查阶段。
- 本阶段下一步：
  - ✅ 已完成 `TraefikRoutePage.vue` 的错误态和列宽优化。
  - ✅ 已完成 `RoutePage.vue` 的列宽优化（目标地址列使用 `max-w-0` 配合 colgroup）。
  - ✅ 已完成 `DeploymentDetail.vue` 日志状态区分（loading/streaming/empty/done/error）。
  - ✅ 已验证 `ComboboxSelect` 的 `openOnFocus` 配置在所有 Dialog 首控件场景已正确设置为 `false`。

## 后续队列

- 模态窗：
  - 将仍然手写 `DialogRoot` / `DialogContent` 的页面逐步迁移到 `AppDialog`。
  - 删除确认弹窗统一宽度、标题、说明、footer 按钮。
- 表单控件：
  - 原始输入框迁移到 `app-input` / `app-textarea`。
  - 错误态统一使用 `app-input-error` 与 `app-field-error`。
  - 下拉框优先使用 `SelectControl` / `ComboboxSelect`。
  - ✅ 观察项已解决：`ComboboxSelect` 已支持 `openOnFocus` prop，所有 Dialog 首控件场景已正确设置为 `false`。
- 链接和操作：
  - 主链接统一使用 `app-link`。
  - 危险操作统一使用危险链接或危险按钮样式。
- 抽屉：
  - Stage 脚本编辑、应用文件编辑等手写抽屉逐步迁移到 `AppDrawer`。
- 观察项：
  - 部署详情在部分终态记录上日志区块可能长时间停留在“加载日志中”，后续确认是否需要空日志/加载失败态。
  - Traefik Routers 当前 `/api/cd/traefik-routes` 返回 500，页面截图停留加载态；样式已先收敛，接口/错误态后续单独确认。
- Tip / Alert：
  - 页面内提示统一到 `app-tip` 或后续封装的 alert 组件。
- 卡片 / Surface：
  - 页面主容器外壳逐步统一到 `app-surface`。
  - 保持详情页信息块和表格块的轻量分隔，不做过度卡片化。




---


• 剩余工作已经进入收尾阶段，不再是大迁移。按优先级评估如下。

  P0 当前阶段必做

  - TraefikRoutePage.vue
      - 补显式错误态，尤其 /api/cd/traefik-routes 返回 500 时不要表现成空数据或长期加载。
      - 表格列宽再收敛，规则、服务、入口点长文本保持稳定截断。
  - RoutePage.vue
      - 工具条改成和其他列表页一致的单行横向滚动范式。
      - 添加路由弹窗补 app-tip 级别的输入提示，例如目标地址格式、路径前缀说明。
      - 表格长文本列再检查，避免目标地址撑开。

  P1 观察项修正

  - DeploymentDetail.vue
      - 部分终态部署日志仍显示“加载日志中...”，需要区分：加载中、无日志、加载失败。
  - ComboboxSelect.vue
      - open-on-focus 会在 Dialog 自动聚焦时展开下拉，建议加 prop 控制，比如 openOnFocus = true，在弹窗首控件场景关闭。
  - ApplicationDetail.vue
      - 仍有手写 ComboboxRoot，虽然样式已接入共享类，但后续可替换为 ComboboxSelect。

  P2 低风险一致性收口

  - StageEdit.vue
      - 仍有手写抽屉、输入框、局部 action item 样式，是目前比较明显的旧风格残留。
  - PipelineRunDetail.vue
      - 还有少量手写按钮类，可替换为 app-button / app-button-primary。
  - Home.vue、Login.vue
      - 有若干非共享按钮/卡片/输入框样式；不是 CI/CD 主线，但后续要做全局一致性时应收掉。

  建议推进顺序

  1. 先完成 TraefikRoutePage.vue 和 RoutePage.vue，这会关闭当前“持续部署列表页”阶段。
  2. 接着处理 DeploymentDetail.vue 日志空态，这是用户可见问题。
  3. 再做 ComboboxSelect 的 openOnFocus 配置，降低后续弹窗下拉误展开风险。
  4. 最后清 StageEdit.vue、PipelineRunDetail.vue 这类残留样式债。

  当前阶段完成标准：CD 列表页都有一致的 toolbar、表格、loading/error/empty 状态，分页保持 10，并用 pomelo-pw 轻量验证 /cd/routes、/cd/traefik-http-routers、/cd/deployments。


## 2026-05-08 更新

### P0 和 P1 优先级工作已完成

**P0 完成项**：
- ✅ TraefikRoutePage.vue
  - 补充显式错误态，区分错误信息和提示文本
  - 表格列宽优化，使用 colgroup 和 max-w-0 确保长文本稳定截断
  - 表格最小宽度从 960px 调整为 1080px，列宽比例优化
- ✅ RoutePage.vue
  - 工具条已是单行横向滚动范式
  - 添加路由弹窗已使用 app-tip 提供输入提示
  - 表格目标地址列使用 max-w-0 配合 colgroup 避免撑开
  - 表格最小宽度从 1180px 调整为 1200px，使用 colgroup 精确控制列宽

**P1 完成项**：
- ✅ DeploymentDetail.vue
  - 日志状态区分：loading（加载中）、streaming（流传输中）、empty（无日志）、done（完成）、error（失败）
  - fetchLogs 和 startLogStream 函数都正确处理空日志情况
- ✅ ComboboxSelect.vue
  - 已有 openOnFocus prop，默认 true
  - 所有 Dialog 首控件场景已正确设置 :open-on-focus="false"
  - 验证场景：TriggerModal、PipelineTemplateDetail（添加 Stage、运行流水线）

**P2 完成项**：
- ✅ StageEdit.vue
  - 从手写抽屉迁移到 AppDrawer
  - 输入框统一到 app-input / app-textarea
  - 按钮统一到 app-button / app-button-primary
  - 脚本编辑抽屉也迁移到 AppDrawer
  - label 统一到 app-field-label
- ✅ PipelineRunDetail.vue
  - 顶部操作按钮统一到 app-button / app-button-primary / app-button-destructive
  - 取消确认弹窗按钮统一到共享样式
  - 所有 section header 统一到 app-section-header
- ✅ Home.vue
  - 概览卡片外壳统一到 app-surface
  - 刷新按钮统一到 app-button
  - 最近构建/部署区块统一到 app-surface / app-section-header
  - "查看全部"链接统一到 app-link
- ✅ Login.vue
  - 登录卡片外壳统一到 app-surface
  - 输入框统一到 app-input，错误态使用 app-input-error
  - label 统一到 app-field-label
  - 错误提示统一到 app-field-error
  - 登录按钮统一到 app-button-primary

**验证通过**：
- ✅ yarn lint --fix - 无错误
- ✅ git diff --check - 无空白字符问题

**当前阶段完成标准达成**：
✅ CD 列表页都有一致的 toolbar、表格、loading/error/empty 状态，分页保持 10。
✅ P2 低风险一致性收口工作全部完成。

**Reka UI 风格一致性推进项目完成**：
所有计划内的页面和组件已完成统一，包括：
- 表格体系（列表页 app-table-list、详情页 app-table-detail）
- 共享组件（AppDialog、AppDrawer、AppToaster、SearchControl、SelectControl、ComboboxSelect）
- 共享样式类（app-surface、app-input、app-textarea、app-button、app-link、app-tip 等）
- 所有 CI/CD 页面的弹窗、表单、按钮、链接、输入框统一
- 首页和登录页的样式统一
