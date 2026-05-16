# Pomelo Orbit V2 编码指南

> 本文档面向 AI 助手，提供核心编码规范和约束。详细架构设计请参考 `docs/` 目录。

## AI 助手工作规范

### 必须遵守的约束

1. **Git 操作**：❌ 禁止执行任何 git 写入操作（commit、push、merge），除非用户明确授权
2. **不确定时**：❌ 不要猜测 API 用法或组件属性，✅ 先问用户或查阅文档
3. **开发服务器**：❌ 不要自主启动/停止/重启开发服务器，✅ 由用户管理
4. **前端代码检查**：每次代码变更后必须执行 `yarn lint --fix && yarn typecheck`，有错误必须立即修复
5. **做得多错得多**：❌ 禁止加戏，如有想法，主动沟通

## 核心编码原则

### 1. 简洁至上
- 只编写解决问题所需的最少代码
- 避免过度设计和提前优化
- 不要添加"可能有用"的功能

### 2. 直接修改，不做兼容
项目处于活跃开发期，不需要维护向后兼容性：
- ❌ 不使用兼容层、别名或默认值处理
- ❌ 不写 if-else 分支处理新旧两种情况
- ✅ 直接更新代码到目标状态
- ✅ 一次性修改所有相关文件（数据库、模型、API、前端、测试）

### 3. 命名规范

**跨层一致性原则**：从数据库表名到前端组件名称保持统一

| 层级 | 命名规则 | 示例 |
|------|---------|------|
| 数据库表 | 单数 snake_case | `repository` |
| 实体类 | 单数 PascalCase | `Repository` |
| Repository | 单数 + Repository | `RepositoryRepository` |
| DTO | 单数 + Resp/Req | `RepositoryResp`, `RepositoryCreateReq` |
| API 路由 | 单数 | `/api/ci/repository` |
| API 文件 | 单数 snake_case | `repository.py` |
| 前端 API | 单数 + Api | `repositoryApi` |
| 前端组件 | PascalCase + Page/Detail | `RepositoryPage.vue`, `RepositoryDetail.vue` |
| 测试文件 | test_ + 单数 | `test_repository.py` |

**关键规则**：
- API 端点统一使用单数形式（`/api/ci/repository` ✅，`/api/ci/repositories` ❌）
- 外键引用使用单数表名（`ForeignKey("credential.id")` ✅）
- 变量命名使用统一前缀（`repository_id`, `repository_name`, `repository_url`）

## 关键约束（禁止事项）

### 后端

❌ **不要**给领域实体的业务字段设默认值（`deployment_type`、`status`、`auto_deploy` 等），业务代码必须显式传递

❌ **不要**用 `getattr` 防御性取值配置字段，✅ 直接访问 + `assert` 断言非空

❌ **不要**在接口层捕获异常，✅ 交给全局异常处理器

❌ **不要**在日志中以变量起头，✅ 以业务描述开头，参数用 `key=value` 格式

❌ **不要**对预期内的失败使用 `exc_info=True`，✅ 只在 `logger.error` 时使用

### 前端

❌ **不要**在各处散落 `dayjs()` 调用，✅ 使用 `frontend/src/utils/time.ts` 工具函数

❌ **不要**在业务代码中处理响应格式，✅ 拦截器已统一处理，只需 catch Error

❌ **不要**在提交函数中手动检查字段值，✅ 使用原生 HTML5 验证或框架验证

❌ **不要**页面层直接使用 Reka primitives，✅ 优先使用项目共享组件（`AppDialog`、`AppDrawer`、`ComboboxSelect` 等）

### 通用

❌ **不要**添加兼容层、别名和默认值处理

❌ **不要**提前优化，不要添加未明确需要的功能

❌ **不要**修改已执行的迁移文件

## 快速参考

### 时间处理

**原则**：存储和传输用 UTC，展示用本地时区

- 后端：使用 `pomelo_orbit.infrastructure.time_utils`（`utc_now()`, `to_iso8601()`）
- 前端：使用 `frontend/src/utils/time.ts`（`formatTime()`, `formatRelativeTime()`）
- 数据库：UTC naive datetime
- API 传输：ISO 8601 带 Z（如 `2024-03-07T08:15:30Z`）

### 异常处理

```python
# ✅ Service 层抛出业务异常
class RepositoryService:
    def delete(self, id: str) -> None:
        if not self.repo.find_by_id(id):
            raise BusinessError(f"Repository {id} not found", status_code=404)

# ✅ 接口层不捕获
@router.delete("/{id}", status_code=204)
def delete(id: str, svc: Annotated[RepositoryService, Depends(...)]):
    svc.delete(id)  # 不需要 try-except
```

### 日志规范

```python
# ✅ 正确
logger.info(f"Pipeline run completed: run_id={run_id}, status={status}")
logger.warning(f"Deployment failed: app_id={app_id}, reason=image_not_found")
logger.error(f"Container start failed: app_id={app_id}, error={e}", exc_info=True)

# ❌ 错误
logger.info(f"[{run_id}] completed")  # 以变量起头
logger.warning(f"Token expired", exc_info=True)  # warning 不需要 exc_info
logger.error(f"Failed: {e}")  # error 缺少 exc_info
```

### 前端错误处理

```typescript
// ✅ 正确
import { useToast } from '@/composables/useToast';
const toast = useToast();

try {
  await repositoryApi.create(form);
  toast.success('创建成功');
} catch (error: unknown) {
  toast.error(error instanceof Error ? error.message : '操作失败');
}
```

### 数据验证

```python
# ✅ 后端：Pydantic 声明式校验
class RepositoryCreateReq(BaseModel):
    name: str = Field(min_length=1)
    code: str = Field(min_length=1, pattern=r"^[a-z0-9_-]+$")
    repository_url: str = Field(min_length=1)
    git_credential_id: str | None = None
```

```vue
<!-- ✅ 前端：HTML5 验证 -->
<form @submit.prevent="handleSubmit">
  <input v-model="email" type="email" required />
  <button type="submit">提交</button>
</form>
```

### 分页查询

```python
@router.get("", response_model=PaginatedResp[RepositoryResp])
def list_repositories(
    page: int = Query(1, ge=1),
    per_page: int = Query(10, ge=1, le=100),
    search: str | None = Query(None),
):
    ...
```

## 详细文档

完整的架构设计、设计模式、最佳实践请参考：

- **后端架构**：`docs/backend/architecture.md` - DDD 分层、依赖注入、实体设计
- **前端架构**：`docs/frontend/architecture.md` - 技术栈、项目结构、组件规范
- **前端风格指南**：`docs/frontend/style-guide.md` - UI 组件使用规范
- **Reka UI 集成**：`docs/frontend/reka-ui-guide.md` - Reka UI 组件使用
- **开发工作流**：`docs/development/workflow.md` - Git 流程、代码审查
- **开发进度**：`docs/todo.md` - 当前任务和待办事项

## 质量保证

### 代码检查

**后端**：
```bash
ruff check --fix
mypy .
pytest
```

**前端**（强制）：
```bash
yarn lint --fix
yarn typecheck
```

### 测试

- 测试文件命名：`test_*.py`（后端）、`*.test.ts`（前端）
- 测试文件名使用单数形式，与被测试模块名称一致
- 后端测试使用内存 SQLite：`POMELO_ORBIT_DATABASE__SQLITE__PATH=:memory:`

## 迁移脚本规范

### 文件命名
```
v0.1.1__schema.sql
v0.1.2__init.sql
v0.1.3__add_sample.sql
```

### 幂等性要求
- 使用 `IF NOT EXISTS` 或 `WHERE NOT EXISTS`
- 已执行的迁移文件不可修改（checksum 校验）
