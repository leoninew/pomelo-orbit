# Pomelo Orbit 开发待办

更新日期：2026-05-18

## 当前状态

### Frontend 完成情况

- ✅ 表格体系已基本收敛：当前扫描到的页面表格均使用 `app-table-list` 或 `app-table-detail`
- ✅ 弹窗和抽屉已基本收敛：页面层手写 `DialogRoot/DialogContent` 已清掉，统一通过 `AppDialog` / `AppDrawer` 使用 Reka UI
- ✅ Toast 已统一到 `AppToaster` / Reka Toast
- ✅ CI/CD 主要列表页和详情页已完成第一轮共享样式收口，包含表格、链接、按钮、输入框、错误态、tip、surface、section header
- ✅ 页面迁移验证已完成：15 个页面（8 个 CI + 7 个 CD）
- ✅ 确认对话框规范化修复已完成：15 个文件
- ✅ Monaco Editor 集成已完成：BuildStageDetail.vue、ApplicationDetail.vue、DeploymentDetail.vue

---

## 待办事项

### Frontend

#### 中优先级

**功能实现**
- **流水线模板详情页阶段编排拓扑排序**：当前已移除手动序号列，需要根据依赖关系自动排序
- **变量声明表格重置功能**：当前操作列文案已从"删除"改为"重置"，但实际行为仍是删除，需要实现真正的重置到默认值逻辑

**表格对齐统一**
部分表格标题左右对齐不一致，需要统一规范：
- 列表类表格（`app-table-list`）：标题和内容都左对齐，操作列右对齐
- 详情类表格（`app-table-detail`）：标题和内容都左对齐，操作列右对齐

#### 低优先级

**优化项**
- 状态徽标颜色可进一步抽成共享状态类（非阻塞项）
- `ComboboxSelect` 若需支持更复杂对象值，再单独扩展（当前已满足需求）

### Backend & General Features

#### 权限系统待办

- **修复 token 过期后受保护页面放行问题**：`authStore.fetchUser()` 失败时当前返回 `undefined`，路由守卫需要在用户信息获取失败后跳转登录页
- **保护系统角色和最后管理员权限路径**：角色更新/删除前需要防止删除 admin 角色或清空最后一个 `role:write` 持有人
- **优化角色列表权限查询**：角色列表响应当前逐个查询 `permission_codes`，后续改为批量查询组装，避免 N+1 查询

#### 架构说明

**CD 模块与 CI 模块的架构差异**

CD 模块（Application、Deployment、Route）从设计之初就采用了简化的架构：
- 领域实体**不包含** `project_id` 字段
- API 端点从未使用 `project_id` 参数或 `get_current_project` 依赖
- 所有 get/update/delete 操作已经是"只通过 ID 访问"的模式

这与 CI 模块不同：
- CI 模块实体包含 `project_id` 字段（Repository、Credential、BuildStage 等）
- CI 模块已完成重构（2026-05-15），移除了 get/update/delete 操作中冗余的 project_id 参数

**注**：CD 模块的数据库模型虽然有 `project_id` 字段（nullable），但这是为未来多租户功能预留的，当前领域层和应用层完全未使用。

#### 待实现功能

- **持续部署 port 变量化**：CD 模块中 port 应该用变量占位
- **制品（Artifact）领域**：制品存储、版本管理、跨 Run 引用
- **触发弹窗重构**：去掉独立的 trigger_ref 输入框，改为完整变量列表（`repository_ref` 可编辑）
- **流水线模板"保存并运行"**：流水线模板详情页"运行"按钮有未保存变更时已拦截，后续考虑支持"保存并运行"快捷操作

#### 已知技术债

- **流水线异步任务状态管理**：`backend/src/pomelo_orbit/infrastructure/ci/executor_impl.py` 中流水线异步任务状态管理复杂，cancel/retry 逻辑与 asyncio task 生命周期耦合较深，需要梳理

---

## 页面迁移验证记录

### 验证完成情况
已完成 15 个页面的功能完整性对比验证（8 个 CI 模块 + 7 个 CD 模块）

### CI 模块（8/8 完成）

| 页面 | 状态 | 备注 |
|------|------|------|
| PipelineTemplateDetail.vue | ✅ | ⚠️ 拖拽排序功能已移除 |
| PipelineSnapshotDetail.vue | ✅ | 功能完整，新增多个改进 |
| PipelineRunPage.vue | ✅ | ⚠️ 重试/取消按钮已移除（有意） |
| PipelineRunDetail.vue | ✅ | 功能完整，日志查看改进 |
| CredentialPage.vue | ✅ | 新增搜索功能 |
| CredentialDetail.vue | ✅ | 按钮样式改进 |
| BuildStagePage.vue | ✅ | 新增搜索功能 |
| BuildStageDetail.vue | ✅ | 已集成 Monaco Editor |

### CD 模块（7/7 完成）

| 页面 | 状态 | 备注 |
|------|------|------|
| ArtifactPage.vue | ✅ | 新增搜索功能 |
| TraefikRoutePage.vue | ✅ | 新增搜索功能，错误处理改进 |
| RoutePage.vue | ✅ | 新增搜索、created_at 列 |
| RouteDetail.vue | ✅ | HTTPS 配置 UI 改进 |
| DeploymentPage.vue | ✅ | 新增应用筛选、错误列 |
| DeploymentDetail.vue | ✅ | 日志状态机改进，智能返回按钮，已集成 Monaco Editor |
| ApplicationPage.vue | ✅ | 分页大小选择器，导入功能增强 |
| ApplicationDetail.vue | ✅ | 服务镜像表格增强，路由选择改进，已集成 Monaco Editor |

### 主要改进点
- 所有列表页新增搜索功能
- 统一使用共享组件（AppDialog、AppDrawer、ComboboxSelect）
- 表格样式统一（app-table-list、app-table-detail）
- 部署/停止操作跳转到详情页（更好的 UX）
- 日志查看状态机更完善

---

## 历史变更记录

### 2026-05-15
- 移除 CD 模块 API 重构任务（经分析，CD 模块从设计之初就不包含 project_id，无需重构）
- 添加 CD 模块与 CI 模块架构差异说明

### 2026-05-10
- 完成 15 个页面的迁移验证（8 个 CI + 7 个 CD）
- 完成所有确认对话框的规范化修复（15 个文件）
- 完成 Monaco Editor 集成（BuildStageDetail.vue、ApplicationDetail.vue、DeploymentDetail.vue）
- 文件重命名：BuildStage.vue → BuildStageDetail.vue
- 重构 todo.md 文档结构
- 合并 `docs/discussions/tood.md` 到 `docs/todo.md`
- 完成文档重组：建立 development/、frontend/、backend/ 目录结构

### 2026-05-09
- 新增 `docs/style.md`，承载风格一致性规范
- 统一脚本抽屉风格（960px 宽度）
- 统一文件抽屉风格（960px 宽度）
- 修复 `DeploymentDetail.vue` 应用详情链接路由
- 修复 `PipelineRunDetail.vue` Stage 日志抽屉状态
- 收口 `ApplicationDetail.vue` 应用路由弹窗的服务选择
- 收口 CI/CD 列表工具条样式（新增 `app-toolbar-*` 类）
- 建立模态窗风格标准并完成第一轮收口
- 统一取消模态窗自动 focus 行为
- 删除遗留组件 `StageEdit.vue`
- 移除流水线模板详情页阶段编排表格的序号列
- 变量声明表格操作列文案从"删除"改为"重置"
- 变量声明表格无值时显示从斜体"未设置"改为普通"—"
