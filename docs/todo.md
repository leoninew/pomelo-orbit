# Pomelo Orbit 开发待办

更新日期：2026-05-10

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

#### 高优先级

**CD 模块 API 接口重构：移除 get/update/delete 的 project_id 参数**

**背景：**
CI 模块已完成接口重构（2026-05-15），移除了 get/update/delete 操作中冗余的 project_id 参数。该重构基于以下原则：
- 资源通过 ID 已经唯一确定，无需额外传递 project_id
- project_id 可以从资源实体本身获取（`resource.project_id`）
- 简化 API 接口，减少参数冗余
- 为后续统一添加项目级权限断言做准备

**重构原则：**
1. **保留 project_id 的场景**：
   - `list` 操作：需要 project_id 限定查询范围
   - `create` 操作：需要 project_id 指定资源归属

2. **移除 project_id 的场景**：
   - `get` 操作：通过 resource_id 获取
   - `update` 操作：通过 resource_id 更新
   - `delete` 操作：通过 resource_id 删除
   - `duplicate` 操作：通过 resource_id 复制（从原资源获取 project_id）
   - 其他基于 ID 的操作（如 `retry`、`cancel`）

3. **实现层次**：
   - Domain Repository：添加 `find_by_id(resource_id)` 方法（不验证 project_id）
   - Infrastructure Repository：实现 `find_by_id()` 方法
   - Application Service：修改方法签名，移除 project_id 参数，使用 `resource.project_id` 处理关联操作
   - API Layer：移除 `Depends(get_current_project)` 依赖（对于 get/update/delete 端点）
   - Frontend：移除 API 调用中的 projectId 参数

**待重构的 CD 资源：**
- [ ] **Application**（应用）
  - [ ] Domain: `ApplicationRepository.find_by_id()`
  - [ ] Infrastructure: 实现 `find_by_id()`
  - [ ] Service: `get_application()`, `update_application()`, `delete_application()`
  - [ ] API: `GET/PUT/DELETE /api/cd/application/{id}`
  - [ ] Frontend: `applicationApi.get/update/delete`

- [ ] **Deployment**（部署）
  - [ ] Domain: `DeploymentRepository.find_by_id()`
  - [ ] Infrastructure: 实现 `find_by_id()`
  - [ ] Service: `get_deployment()`, `update_deployment()`, `delete_deployment()`, `cancel_deployment()`
  - [ ] API: `GET/PUT/DELETE/POST /api/cd/deployment/{id}/*`
  - [ ] Frontend: `deploymentApi.get/update/delete/cancel`

- [ ] **Route**（路由）
  - [ ] Domain: `RouteRepository.find_by_id()`
  - [ ] Infrastructure: 实现 `find_by_id()`
  - [ ] Service: `get_route()`, `update_route()`, `delete_route()`
  - [ ] API: `GET/PUT/DELETE /api/cd/route/{id}`
  - [ ] Frontend: `routeApi.get/update/delete`

- [ ] **TraefikRoute**（Traefik 路由）
  - [ ] Domain: `TraefikRouteRepository.find_by_id()`
  - [ ] Infrastructure: 实现 `find_by_id()`
  - [ ] Service: `get_traefik_route()`, `update_traefik_route()`, `delete_traefik_route()`
  - [ ] API: `GET/PUT/DELETE /api/cd/traefik-route/{id}`
  - [ ] Frontend: `traefikRouteApi.get/update/delete`

**实施步骤：**
1. 按资源逐个重构（Application → Deployment → Route → TraefikRoute）
2. 每个资源按层次从下往上修改（Domain → Infrastructure → Service → API → Frontend）
3. 修改完成后运行 `make lint` 确保类型检查通过
4. 更新方法文档字符串（"通过 ID 获取/更新/删除"）
5. 提交时使用统一的 commit message 格式：`refactor(cd): remove project_id from {resource} get/update/delete operations`

**预期收益：**
- API 接口更简洁，参数更少
- 前后端代码更清晰，减少冗余传参
- 为统一添加项目级权限断言奠定基础
- 与 CI 模块保持一致的架构风格

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
