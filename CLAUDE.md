# Pomelo Orbit V2 编码指南

## 核心原则

### 1. 简洁至上

- 只编写解决问题所需的最少代码
- 避免过度设计和提前优化
- 不要添加"可能有用"的功能

### 2. 直接修改，不做兼容

**核心理念**：项目处于活跃开发期，不需要维护向后兼容性

- 不使用兼容层、别名或默认值处理
- 不写大量 if-else 分支处理新旧两种情况
- 直接更新代码到目标状态
- 一次性修改所有相关文件（数据库、模型、API、前端、测试）
- 不保留 deprecated API 或字段

**实践示例**：

```python
# ✅ 正确 - 直接修改
# 1. 数据库迁移：重命名表
ALTER TABLE projects RENAME TO repository;

# 2. 更新模型
class RepositoryModel(Base):
    __tablename__ = "repository"

# 3. 更新 API
@router.get("/repository")

# 4. 更新前端
const repositoryApi = { list: () => request.get('/api/ci/repository') }

# ❌ 错误 - 保留兼容性
# 不需要：
@router.get("/projects")  # 旧的 endpoint
@router.get("/repository")  # 新的 endpoint
def get_projects_compatible(): ...
```

### 3. 命名规范

**核心原则**：跨层一致性，从数据库表名到前端组件名称保持统一

#### 术语一致性原则

当一个业务概念在多个层级中出现时，所有层级使用相同的术语：

```
数据库表:      repository
实体类:        Repository
Repository:   RepositoryRepository
DTO:           RepositoryResp, RepositoryCreateReq
API 路由:      /api/ci/repository
路由标签:      tags=["repository"]
API 文件:      repository.py
前端类型:      Repository
前端 API:      repositoryApi
前端组件:      RepositoryPage.vue, RepositoryDetail.vue
测试文件:      test_repository.py
```

#### 反例（避免不一致命名）

```
❌ 数据库: projects
❌ 实体: Project
❌ API: /api/ci/repositories
❌ 前端: projectApi
❌ 组件: ProjectList.vue
```

#### API 端点

#### API 端点

**统一使用单数形式**，不使用复数

```
✅ /api/ci/credential
✅ /api/ci/repository
✅ /api/ci/template
✅ /api/ci/run
✅ /api/ci/build-stage
✅ /api/cd/application
✅ /api/cd/deployment
✅ /api/cd/route
✅ /api/cd/traefik-route

❌ /api/cd/applications
❌ /api/cd/deployments
❌ /api/cd/routes
```

#### 后端 API 文件命名

单数 snake_case

```
✅ credential.py
✅ repository.py
✅ template.py
✅ run.py
```

#### 路由定义规范

路由前缀也使用单数，与文件名保持一致：

```python
# ✅ 正确
router = APIRouter(prefix="/credential", tags=["credential"])
router = APIRouter(prefix="/repository", tags=["repository"])
router = APIRouter(prefix="/template", tags=["template"])

# ❌ 错误 - 使用复数
router = APIRouter(prefix="/credentials", tags=["credentials"])
router = APIRouter(prefix="/projects", tags=["projects"])
```

#### 前端 API 文件命名与导出

文件名单数 camelCase，导出对象单数 + `Api` 后缀

```typescript
// ✅ 正确
// 文件: credential.ts
export const credentialApi = { ... }

// 文件: repository.ts
export const repositoryApi = { ... }

// 文件: template.ts
export const pipelineTemplateApi = { ... }

// ❌ 错误 - 复数形式
// 文件: credentials.ts
export const credentialsApi = { ... }
```

#### DTO 文件命名

单数 snake_case

```
✅ dto/typing_content.py
✅ dto/typing_session.py
✅ dto/user.py
```

#### 数据库表名

使用单数形式

```
✅ user
✅ typing_content
✅ typing_session
```

#### 字段命名

保持前后端一致，不使用别名。

```python
# ✅ 正确
class TypingContentResp(BaseModel):
    content_type: str

# ❌ 错误
class TypingContentResp(BaseModel):
    type: str = Field(alias="content_type")
```

#### 变量命名一致性

相关变量使用统一前缀，保持命名一致性：

```python
# ✅ 正确 - Repository 相关变量统一前缀
repository_id: str
repository_name: str
repository_url: str
repository_repository_url: str  # 内置变量

# ✅ 正确 - Pipeline 相关变量统一前缀
pipeline_run_id: str
pipeline_snapshot_id: str
pipeline_template_id: str

# ❌ 错误 - 混用不同前缀
repository_id: str
project_name: str  # 应该是 repository_name
```

#### Vue 组件命名

PascalCase，路由页面使用 `Page`/`Detail` 后缀，子组件按功能命名

```
✅ ApplicationPage.vue       （列表页）
✅ ApplicationDetail.vue     （详情页）
✅ ApplicationFormFields.vue （子组件）
✅ TriggerModal.vue          （弹窗子组件）
✅ Login.vue
✅ Home.vue
```
```

## 架构设计

### DDD 分层架构

```
interfaces/api/        # 接口层 - HTTP 适配，DTO 校验
application/           # 应用层 - 业务编排（services/ + di.py）
domain/                # 领域层 - 核心业务逻辑（entities、value_objects、repositories、domain services）
infrastructure/        # 基础设施层 - 技术实现
```

**分层职责**：

- **接口层**：HTTP 请求/响应适配，DTO 校验，不包含业务逻辑
- **应用层**：业务流程编排，事务管理，调用领域服务和仓储
- **领域层**：核心业务规则，实体行为，领域服务（跨聚合的业务逻辑）
- **基础设施层**：数据库、外部服务、技术工具

**依赖规则**：

- 应用服务不得互相引用（同级引用）
- 应用服务只能依赖：仓储接口、领域服务、领域实体
- 跨聚合的业务逻辑应下沉到领域服务
- 领域服务不依赖应用服务

```python
# ✅ 正确 - 应用服务依赖仓储和领域服务
class PipelineRunService:
    def __init__(
        self,
        run_repo: PipelineRunRepository,
        repository_repo: RepositoryRepository,
        template_repo: PipelineTemplateRepository,
        snapshot_manager: SnapshotManager,  # 领域服务
        variable_resolver: VariableResolver,  # 领域服务
    ):
        ...

# ❌ 错误 - 应用服务之间互相引用
class PipelineRunService:
    def __init__(
        self,
        repository_service: RepositoryService,  # 同级应用服务
        template_service: TemplateService,      # 同级应用服务
    ):
        ...
```

### 依赖注入 (DI) 规范

**核心原则**：每个包用自己的 `di.py` 管理本层依赖，各层 `di.py` 串接，上层引用下层。子包简单时由父包 `di.py` 直接管理，复杂时子包自己有 `di.py`。

```
infrastructure/persistence/di.py   # get_db
infrastructure/di.py               # get_security_service, get_xxx_repository（引用 persistence/di）
application/di.py                  # get_xxx_service（引用 infrastructure/di）
interfaces/api/                    # 路由内联 Annotated[Type, Depends(get_xxx)]
```

#### DI 实现示例

**persistence 层**

```python
# infrastructure/persistence/di.py
def get_db() -> Generator[Session, None, None]:
    with SessionLocal() as session:
        yield session
```

**infrastructure 层**

```python
# infrastructure/di.py
def get_security_service(settings: Annotated[Dynaconf, Depends(get_settings)]) -> SecurityService:
    return SecurityService(settings)

def get_user_repository(db: Annotated[Session, Depends(get_db)]) -> Generator[UserRepositoryImpl, None, None]:
    yield UserRepositoryImpl(db)
```

**application 层**

```python
# application/di.py
def get_auth_service(
    user_repo: Annotated[UserRepositoryImpl, Depends(get_user_repository)],
    user_progress_repo: Annotated[UserProgressRepositoryImpl, Depends(get_user_progress_repository)],
) -> AuthService:
    return AuthService(user_repo, user_progress_repo)
```

**接口层**

```python
# interfaces/api/auth.py
@router.post("/auth/login")
def login(
    auth_service: Annotated[AuthService, Depends(get_auth_service)],
    data: LoginReq,
):
    return auth_service.login(data.username, data.password)
```

#### DI 规范要点

1. 所有 DI 函数使用 `get_xxx` 格式
2. 使用 `Annotated[Type, Depends(get_xxx)]` 声明依赖，不定义 `XxxDep` 别名
3. 上层依赖下层，下层不依赖上层
4. domain 层无 FastAPI 依赖，不需要 `di.py`
5. 每个 DI 函数只负责一个依赖的创建

### 实体默认值规范

**核心原则**：领域实体的业务字段不设默认值，业务代码必须显式传递

- 有业务语义的字段（`auth_source`、`email_verified`、`role` 等）不在 dataclass 中设默认值
- 只有纯技术性的可选字段（`phone`、`google_id` 等"无则为空"的字段）才允许默认值
- 这样可以在构造实体时暴露遗漏字段，而不是静默使用错误的默认值

```python
# ✅ 正确 - 业务字段无默认值，构造时必须显式传递
@dataclass
class User:
    id: str
    username: str
    password_hash: str
    role: str           # 无默认值，必须显式传 "student" / "teacher" / "admin"
    email: str
    auth_source: str    # 无默认值，必须显式传 "email" / "google"
    email_verified: bool
    phone: str | None = None   # 可选字段，无手机号时为 None

# ❌ 错误 - 业务字段设了默认值，构造时可能漏传
@dataclass
class User:
    auth_source: str = "email"     # 静默默认，Google 注册时可能漏传
    email_verified: bool = False   # 静默默认，邮箱注册后可能忘记设为 True
```

### 配置取值规范

**核心原则**：直接访问配置字段，用 `assert` 断言必填项非空，不用 `getattr` 做防御性取值

```python
# ✅ 正确 - 直接取值 + assert 断言
def get_resend_client(settings: ...) -> ResendClient:
    api_key: str = settings.resend.api_key
    from_email: str = settings.resend.from_email
    assert api_key, "TYPING_ISLAND_RESEND__API_KEY is not configured"
    assert from_email, "resend.from_email is not configured"
    return ResendClient(api_key=api_key, from_email=from_email)

# ❌ 错误 - getattr 防御性取值，掩盖配置缺失问题
def get_resend_client(settings: ...) -> ResendClient:
    resend = getattr(settings, "resend", None)
    api_key = getattr(resend, "api_key", "") if resend else ""
    return ResendClient(api_key=api_key, ...)
```

### 异常处理规范

**核心原则**：Service 层抛出业务异常，接口层不捕获，由全局异常处理器统一处理

- Service 层抛出 `BusinessError`（带正确的 status_code），不使用 `ValueError`、`RuntimeError` 等通用异常
- 接口层只负责 HTTP 适配，不写 try-except，不做业务逻辑判断
- 全局异常处理器统一捕获 `BusinessError`，转换为标准 HTTP 响应

```python
# ✅ 正确
class PipelineService:
    def delete_repository(self, repository_id: str) -> None:
        repository = self.repository_repo.find_by_id(repository_id)
        if not repository:
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        self.repository_repo.delete(repository)

@router.delete("/{repository_id}", status_code=204)
def delete_repository(repository_id: str, svc: Annotated[PipelineService, Depends(get_pipeline_service)]):
    svc.delete_repository(repository_id)  # 不需要 try-except

# ❌ 错误
@router.get("/{id}")
def get_repository(id: str):
    try:
        return svc.get_repository(id)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
```

### 日志规范

**核心原则**：日志以业务描述开头，参数以 `key=value` 格式跟在后面，搜索友好，不以变量起头

#### 级别选择

- `logger.info` — 关键业务步骤的正常结果
- `logger.warning` — 预期内的失败或降级行为
- `logger.error(..., exc_info=True)` — 意外异常，需要 traceback

```python
# ✅ 正确
logger.info(f"Session completed: user={user_id}, wpm={wpm}, accuracy={accuracy}")
logger.warning(f"Login failed: username={username}, reason=invalid_password")
logger.error(f"Session save failed: user={user_id}, error={e}", exc_info=True)

# ❌ 错误 - 以变量起头
logger.info(f"[{user_id}] completed session")

# ❌ 错误 - warning 用了 exc_info
logger.warning(f"Token expired: user={user_id}", exc_info=True)

# ❌ 错误 - error 没有 exc_info
logger.error(f"Save failed: {e}")
```

#### 不需要打日志的场景

- 每个文件写入/删除操作（过于细碎）
- 紧接着就 raise 的错误（调用方会处理）
- 读操作、查询操作（无副作用）

### 数据库设计

- 使用外键约束保证数据完整性
- 使用 `ON DELETE CASCADE` 自动清理关联数据
- 字段顺序：重要字段在前，辅助字段在后
- 外键引用使用**单数表名**，与 `__tablename__` 保持一致

```python
# ✅ 正确 - 外键引用与表名一致
class CredentialModel(Base):
    __tablename__ = "credential"

class RepositoryModel(Base):
    __tablename__ = "repository"
    git_credential_id: Mapped[str | None] = mapped_column(
        String(26), ForeignKey("credential.id"), ...  # 单数
    )

# ❌ 错误 - 外键引用使用复数
git_credential_id: Mapped[str | None] = mapped_column(
    String(26), ForeignKey("credentials.id"), ...  # 复数，与表名不一致
)
```

## 技术栈规范

### 时间处理最佳实践

**核心原则**：存储和传输用 UTC，展示用本地时区

| 层级 | 时区 | 格式 | 说明 |
|------|------|------|------|
| 数据库 | UTC | naive datetime | 统一时区，避免混乱 |
| 后端内部 | UTC | datetime 对象 | 内部逻辑全部使用 UTC |
| API 传输 | UTC | ISO 8601 (带 Z) | 如 `2024-03-07T08:15:30Z` |
| 前端展示 | 本地时区 | 格式化字符串 | 自动转换为用户本地时间 |
| 前端提交 | 转为 UTC | ISO 8601 | 提交前转换为 UTC |

**为什么这样做**：
- 避免夏令时、时区变更等问题
- 便于全球化部署和跨时区查询
- 数据一致性和可比较性
- 用户体验友好（自动显示本地时间）

### 后端 (Python/FastAPI)

#### 时间处理规范

统一使用 `backend/src/typing_island/infrastructure/time_utils.py`，禁止在各处散落 `datetime.now()` 调用。

```python
# ✅ 正确
from typing_island.infrastructure.time_utils import utc_now, to_iso8601

session.completed_at = utc_now()
response_time = to_iso8601(dt)

# ❌ 错误
from datetime import datetime, timezone
session.completed_at = datetime.now(timezone.utc).replace(tzinfo=None)
```

**可用函数**：
- `utc_now()` - 获取当前 UTC 时间（naive datetime）
- `to_iso8601(dt)` - 转换为 ISO 8601 格式（API 响应）
- `from_iso8601(iso_string)` - 从 ISO 8601 解析（API 请求）
- `add_minutes/hours/days(dt, n)` - 时间计算

#### 数据验证

**核心原则**：使用 Pydantic Field 进行声明式校验，不在业务逻辑中兜底处理

- 必填字段使用 `Field(min_length=1)` 确保非空
- 可选字段使用 `str | None = None`，不使用空字符串默认值
- 枚举值使用 `Literal` 类型约束
- 列表字段使用 `Field(default_factory=list)` 而不是 `= []`

```python
# ✅ 正确 - 声明式校验
class TriggerPipelineReq(BaseModel):
    template_id: str = Field(min_length=1)
    trigger_ref: str = Field(min_length=1)
    variables: dict[str, Any] = Field(default_factory=dict)

class RepositoryCreateReq(BaseModel):
    name: str = Field(min_length=1)
    code: str = Field(min_length=1, pattern=r"^[a-z0-9_-]+$")
    repository_url: str = Field(min_length=1)
    default_branch: str = Field(default="master", min_length=1)

# ❌ 错误 - 使用空字符串默认值 + 业务逻辑兜底
class TriggerPipelineReq(BaseModel):
    trigger_ref: str = ""  # 空字符串默认值

# 业务逻辑中兜底处理
effective_ref = trigger_ref or repository.default_branch  # 不要这样做
```

**校验规则**：
- `Field(min_length=1)` - 字符串非空
- `Field(pattern=r"^[a-z0-9_-]+$")` - 正则约束
- `Literal["value1", "value2"]` - 枚举值
- `Field(default_factory=dict)` - 可变默认值
- `Field(default_factory=list)` - 列表默认值

#### 路由定义

```python
@router.get("", response_model=PaginatedResp[TypingContentResp])
def list_contents(...): ...

@router.get("/{id}", response_model=TypingContentResp)
def get_content(id: str, ...): ...
```

#### 路由文档注释规范

路由函数可以添加简单的单行文档注释说明功能，不强制要求：

```python
# ✅ 允许 - 简单单行说明
@router.post("", status_code=201)
def create_application(...) -> ApplicationResp:
    """创建应用"""
    ...

# ✅ 允许 - 无文档注释（函数名已自解释）
@router.get("/{id}")
def get_application(id: str, ...) -> ApplicationResp:
    ...
```

**原则**：
- 文档注释应为简单的单行中文说明
- 如果函数名已清晰表达功能，可以省略文档注释
- 不需要详细说明参数和返回值（已有类型注解）

#### Schema 定义

```python
class TypingContentResp(BaseModel):
    id: str
    title: str
    created_at: datetime

    model_config = {"from_attributes": True}
```

### 前端 (Vue 3/TypeScript)

#### 请求规范

少量请求顺序 await 即可，无须 `Promise.all`；仅在请求数量较多或有明显延迟差异时才使用并发。

```typescript
// ✅ 少量请求 - 顺序执行，简洁清晰
const tplRes = await pipelineTemplateApi.list();
const credRes = await credentialApi.list();

// ✅ 大量无依赖请求 - 并发减少等待
const results = await Promise.all(ids.map(id => itemApi.get(id)));
```

#### 时间处理规范

统一使用 `frontend/src/utils/time.ts`，禁止在各处散落 `dayjs()` 调用。

```typescript
// ✅ 正确
import { formatTime } from '@/utils/time';
const displayTime = formatTime(utcTime);

// ❌ 错误
import dayjs from 'dayjs';
const displayTime = dayjs(time).format('YYYY-MM-DD HH:mm:ss');
```

**可用函数**：
- `formatTime(time, format?)` - UTC 时间转本地时间格式化
- `formatRelativeTime(time)` - 相对时间（如"3分钟前"）
- `isAfterToday(utcTime)` - 判断是否在今天之后
- `toUTC(localTime)` - 本地时间转 UTC（用于提交数据）
- `nowUTC()` - 获取当前 UTC 时间

#### 表单验证规范

**核心原则**：使用原生 HTML5 验证或框架验证，禁止在提交函数中手动检查字段值

**方式1：使用 `required` + `@submit.prevent`（推荐用于简单表单）**

```vue
<template>
  <form @submit.prevent="handleSubmit">
    <input v-model="email" type="email" required />
    <button type="submit">提交</button>
  </form>
</template>
```

**方式2：显式验证（用于复杂业务规则）**

```vue
<script setup lang="ts">
async function handleSubmit() {
  if (password.value !== confirmPassword.value) {
    error.value = '两次密码输入不一致';
    return;
  }
  await api.register(form);
}
</script>
```

#### Vue 组件规范

**Loading 处理**：

- 列表页：在数据容器上绑定 `:loading` 状态
- 详情页：在卡片/区块粒度使用 `:loading`，内容用 `v-if` 控制
- 多个接口：各自使用独立的 loading 变量，不共用一个

**Detail 页面**：

- 数据类型使用非空（`ref<T>()`），不用可空（`ref<T | null>(null)`）
- 在区块粒度使用 `v-if` 条件渲染，不在字段级别使用 `?.`
- `v-if` 保护块内部访问字段时使用 `!`（非空断言），不用 `?.`
- 页面标题等 `v-if` 保护块外部可使用 `?.` 或 `?? '默认值'`

```vue
<!-- ✅ 正确 -->
<script setup lang="ts">
const session = ref<TypingSession>()  // 非空类型
</script>
<template>
  <!-- v-if 外部：可用 ?. -->
  <h1>{{ session?.name ?? '详情' }}</h1>
  <!-- v-if 内部：用 ! 断言 -->
  <div v-if="session">{{ session.name }}</div>
</template>

<!-- ❌ 错误 -->
<script setup lang="ts">
const session = ref<TypingSession | null>(null)
</script>
<template>
  <span>{{ session?.wpm || '-' }}</span>
</template>
```

**模态窗命名**：变量统一使用 `is` 前缀 + 功能描述 + `DialogOpen` 或 `ModalOpen` 后缀

```typescript
// ✅ 正确
const isCreateDialogOpen = ref(false)
const isEditDialogOpen = ref(false)
const isDeleteDialogOpen = ref(false)
const isEditModalOpen = ref(false)

// ❌ 错误
const modalVisible = ref(false)
const visible = ref(false)
const showCreateModal = ref(false)
```

#### API 客户端

显式声明返回类型，不使用泛型参数（拦截器已提取 `response.data`）

```typescript
export const typingContentApi = {
  list(params: ListParams): Promise<PaginatedResp<TypingContent>> {
    return request.get('/api/typing-contents', { params });
  },
  get(id: string): Promise<TypingContent> {
    return request.get(`/api/typing-contents/${id}`);
  },
};
```

#### 前端错误处理

所有 API 错误在拦截器（`frontend/src/utils/request.ts`）中统一处理，业务代码只需 catch Error 对象：

```typescript
// ✅ 正确
try {
  await repositoryApi.create(form);
  message.success('创建成功');
} catch (error: unknown) {
  message.error(error instanceof Error ? error.message : '操作失败');
}

// ❌ 错误 - 不要在业务代码中处理响应格式
try {
  const res = await repositoryApi.create(form);
  if (res.error) { ... }
} catch (error: any) {
  if (error.response?.data?.detail) { ... }
}
```

## 质量保证

### 代码检查

- 后端：ruff + mypy（配置见 `backend/ruff.toml`、`backend/mypy.ini`）
- 前端：Biome（配置见 `frontend/biome.json`）

### 测试

- 测试文件命名：`test_*.py`（后端）、`*.test.ts`（前端）
- 测试文件名使用**单数形式**，与被测试的模块名称保持一致

```
✅ test_credential.py      (测试 credential 模块)
✅ test_repository.py       (测试 repository 模块)
✅ test_template.py         (测试 template 模块)
✅ test_pipeline_run.py     (测试 pipeline_run 模块)

❌ test_credentials.py      (复数形式)
❌ test_repositories.py     (复数形式)
❌ test_templates.py        (复数形式)
```

- 后端测试使用内存 SQLite（`TYPING_ISLAND_DATABASE__SQLITE__PATH=:memory:`）

## 迁移脚本规范

### 文件命名

```
v0.1.1__schema.sql
v0.1.2__init.sql
v0.1.3__add_sample.sql
```

### 支持格式

- `.sql` — DDL 和 DML
- `.json` — 数据操作（insert/update/delete），格式：

```json
[
  {"type": "insert", "table": "achievement", "data": [{"id": "...", "name": "..."}]},
  {"type": "update", "table": "user", "data": {"role": "admin"}, "where": {"id": "..."}}
]
```

### 幂等性

- 使用 `IF NOT EXISTS` 或 `WHERE NOT EXISTS`
- 已执行的迁移文件不可修改（checksum 校验）

## 禁止事项

❌ 不要添加兼容层、别名和默认值处理

❌ 不要给领域实体的业务字段设默认值（`role`、`auth_source`、`email_verified` 等），业务代码必须显式传递

❌ 不要用 `getattr` 防御性取值配置字段，直接访问 + `assert` 断言非空

❌ 不要在接口层捕获异常（交给全局处理器）

❌ 不要提前优化，不要添加未明确需要的功能

❌ 不要在各个文件中散落时间处理逻辑，必须使用统一的工具模块

❌ 不要在日志中以变量起头，不要对预期内的失败使用 `exc_info=True`

❌ 不要修改已执行的迁移文件

## 常见模式

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