# Pomelo Orbit 编码指南

## 核心原则

### 1. 简洁至上

- 只编写解决问题所需的最少代码
- 避免过度设计和提前优化
- 不要添加"可能有用"的功能

### 2. 直接修改，不做兼容

- 不使用兼容层、别名或默认值处理
- 不写大量 if-else 分支
- 直接更新代码到目标状态

### 3. 命名规范

#### API 端点

使用单数形式，不用复数

```
✅ /api/application
✅ /api/credential
✅ /api/deploy-record
❌ /api/applications
❌ /api/credentials
❌ /api/deploys
```

#### API 文件命名

后端使用单数形式

```
✅ backend/src/pomelo_orbit/interfaces/api/application.py
✅ backend/src/pomelo_orbit/interfaces/api/credential.py
✅ backend/src/pomelo_orbit/interfaces/api/deploy_record.py
❌ backend/src/pomelo_orbit/interfaces/api/applications.py
```

前端使用单数形式

```
✅ frontend/src/api/application.ts
✅ frontend/src/api/credential.ts
✅ frontend/src/api/deploy-record.ts
❌ frontend/src/api/applications.ts
```

#### API 导出命名

使用单数形式

```typescript
✅ export const applicationApi = { ... }
✅ export const credentialApi = { ... }
✅ export const deployRecordApi = { ... }
❌ export const applicationsApi = { ... }
```

#### 数据库表名

使用单数形式

```
✅ application
✅ credential
✅ git_source
✅ deploy_record
❌ applications
❌ credentials
```

#### 字段命名

保持前后端一致，不使用别名。示例：统一使用 `extra_data`，不用 `metadata` 别名

```python
# ✅ 正确
class CredentialResp(BaseModel):
    extra_data: str | None

# ❌ 错误
class CredentialResp(BaseModel):
    metadata: str | None = Field(alias="extra_data")
```

#### Vue 组件命名

列表页使用单数 + Page 后缀

```
✅ ApplicationPage.vue
✅ CredentialPage.vue
✅ DeployRecordPage.vue
❌ ApplicationsPage.vue
❌ Applications.vue
```

详情页使用单数 + Detail 后缀

```
✅ ApplicationDetail.vue
✅ CredentialDetail.vue
✅ DeployRecordDetail.vue
❌ ApplicationsDetail.vue
```

其他页面直接使用功能名称

```
✅ Settings.vue
✅ Login.vue
```

#### 路由命名

路径使用复数或单数均可，保持语义清晰

```typescript
✅ /applications
✅ /credentials
✅ /deploy-records
```

路由名称使用单数形式

```typescript
✅ { name: 'Application', path: '/applications', component: ApplicationPage }
✅ { name: 'DeployRecords', path: '/deploy-records', component: DeployRecordPage }
```

#### DTO 文件组织

目录结构按领域分离

```
backend/src/pomelo_orbit/interfaces/api/dto/
├── __init__.py          # 统一导出
├── common.py            # 通用响应
├── auth.py              # 认证相关
├── application.py       # 应用相关
├── credential.py        # 凭据相关
├── deploy_record.py     # 部署记录
├── event.py             # 事件相关
└── gateway.py           # 网关配置
```

文件命名使用单数形式，无需 `_schemas` 后缀

## 架构设计

### 领域驱动设计 (DDD)

- 聚合根: ApplicationModel 是核心聚合根
- 实体: GitSourceModel, ImageSourceModel, ContainerConfigModel
- 值对象: 简单配置对象
- 通过聚合根管理所有相关实体

### 分层架构

```
interfaces/api/        # 接口层 - HTTP 适配
application/           # 应用层 - 业务编排
domain/                # 领域层 - 核心业务逻辑
infrastructure/        # 基础设施层 - 技术实现
```

### 依赖注入 (DI) 规范

**核心原则**：每层通过 `di.py` 管理自己的依赖，串联下级模块

#### DI 文件组织

```
backend/src/pomelo_orbit/
├── application/
│   └── di.py                    # 应用层 DI - 管理 Service
├── infrastructure/
│   ├── persistence/
│   │   └── di.py                # 数据库会话管理
│   ├── repositories/
│   │   └── di.py                # 仓储层 DI
│   ├── traefik/
│   │   └── di.py                # Traefik 管理器 DI
│   └── cert/
│       └── di.py                # 证书服务 DI
```

#### DI 实现示例

**基础设施层 - 数据库会话**

```python
# infrastructure/persistence/di.py
from sqlalchemy.orm import Session

def get_db() -> Generator[Session, None, None]:
    """获取数据库会话"""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()
```

**基础设施层 - 仓储**

```python
# infrastructure/repositories/di.py
from typing import Annotated
from fastapi import Depends
from sqlalchemy.orm import Session

from pomelo_orbit.domain.repositories import RouteRepository
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.repositories.route import RouteRepositoryImpl

def get_route_repository(db: Annotated[Session, Depends(get_db)]) -> RouteRepository:
    """获取路由仓储实例"""
    return RouteRepositoryImpl(db)
```

**基础设施层 - 外部服务**

```python
# infrastructure/traefik/di.py
from pomelo_orbit.infrastructure.traefik.manager import TraefikManager
from pomelo_orbit.infrastructure.config import get_settings, get_project_root

def get_traefik_manager() -> TraefikManager:
    """获取 TraefikManager 实例"""
    settings = get_settings()
    config_path = get_project_root() / settings.traefik.dynamic_route_dir
    cert_path = get_project_root() / settings.traefik.cert_dir
    return TraefikManager(config_path, settings.traefik.container_name, cert_path)
```

**应用层 - Service**

```python
# application/di.py
from typing import Annotated
from fastapi import Depends

from pomelo_orbit.application.route_service import RouteService
from pomelo_orbit.domain.repositories import RouteRepository
from pomelo_orbit.infrastructure.repositories.di import get_route_repository
from pomelo_orbit.infrastructure.traefik.di import get_traefik_manager
from pomelo_orbit.infrastructure.cert.di import get_mkcert_service

def get_route_service(
    route_repo: Annotated[RouteRepository, Depends(get_route_repository)],
    traefik_manager: Annotated[TraefikManager, Depends(get_traefik_manager)],
    mkcert_service: Annotated[MkcertService, Depends(get_mkcert_service)],
) -> RouteService:
    """获取路由服务实例"""
    return RouteService(
        route_repo=route_repo,
        traefik_manager=traefik_manager,
        mkcert_service=mkcert_service,
    )
```

**接口层 - 使用 Service**

```python
# interfaces/api/route.py
from typing import Annotated
from fastapi import APIRouter, Depends

from pomelo_orbit.application.di import get_route_service
from pomelo_orbit.application.route_service import RouteService

router = APIRouter(prefix="/route", tags=["route"])

@router.get("")
def list_routes(
    route_service: Annotated[RouteService, Depends(get_route_service)],
):
    """列出所有路由"""
    routes, total = route_service.list_routes(page=1, per_page=10)
    return {"items": routes, "total": total}
```

#### DI 规范要点

1. **统一命名**：所有 DI 函数使用 `get_xxx` 格式
2. **分层管理**：每层的 `di.py` 只管理本层和串联下级
3. **类型注解**：使用 `Annotated[Type, Depends(get_xxx)]` 声明依赖
4. **避免循环**：上层依赖下层，下层不依赖上层
5. **单一职责**：每个 DI 函数只负责一个依赖的创建

### 异常处理规范

**核心原则**：Service 层抛出业务异常，接口层不捕获，由全局异常处理器统一处理

- Service 层抛出 `BusinessError`（带正确的 status_code），不使用 `ValueError`、`RuntimeError` 等通用异常
- 接口层只负责 HTTP 适配，不写 try-except，不做业务逻辑判断
- 全局异常处理器统一捕获 `BusinessError`，转换为标准 HTTP 响应

```python
# ✅ 正确
class RouteService:
    def delete_route(self, route_id: str) -> None:
        route = self.route_repo.find_by_id(route_id)
        if not route:
            raise BusinessError(f"Route {route_id} not found", status_code=404)
        if route.enabled:
            raise BusinessError("Cannot delete enabled route. Please disable it first.", status_code=400)
        self.route_repo.delete(route)

@router.delete("/{route_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_route(route_id: str, route_service: Annotated[RouteService, Depends(get_route_service)]):
    route_service.delete_route(route_id)  # 不需要 try-except

# ❌ 错误
@router.get("/{id}")
def get_route(id: str):
    try:
        return route_service.get_route(id)
    except ValueError as e:
        raise HTTPException(status_code=404, detail=str(e))
```

### 日志规范

**核心原则**：日志以业务描述开头，参数以 `key=value` 格式跟在后面，搜索友好，不以变量起头

#### 级别选择

- `logger.info` — 关键业务步骤的正常结果（部署成功、路由下发等）
- `logger.warning` — 预期内的失败或降级行为（签名校验失败、自动检测回退等）
- `logger.error(..., exc_info=True)` — 意外异常，需要 traceback

```python
# ✅ 正确
logger.info(f"Deploy succeeded: app={application.code}, deployment={deployment.id}")
logger.warning(f"Traefik reload failed: container={container}, error={e.stderr}")
logger.error(f"Deploy failed: app={application.code}, deployment={deployment.id}, error={e}", exc_info=True)

# ❌ 错误 - 以变量起头
logger.info(f"[{app.code}] started")
logger.info(f"{deployment.id} queued")

# ❌ 错误 - warning 用了 exc_info（预期内的失败不需要 traceback）
logger.warning(f"Branch not allowed: branch={branch}", exc_info=True)

# ❌ 错误 - error 没有 exc_info（意外异常应附上 traceback）
logger.error(f"Deploy failed: {e}")
```

#### 不需要打日志的场景

- 每个文件写入/删除操作（过于细碎）
- 紧接着就 raise 的错误（调用方会处理）
- 读操作、查询操作（无副作用）

### 数据库设计

- 使用外键约束保证数据完整性
- 使用 `ON DELETE CASCADE` 自动清理关联数据
- 字段顺序：重要字段在前，辅助字段在后

## 技术栈规范

### 时间处理最佳实践

**核心原则**：存储和传输用 UTC，展示用本地时区

| 层级 | 时区 | 格式 | 说明 |
|------|------|------|------|
| **数据库** | UTC | naive datetime | 统一时区，避免混乱 |
| **后端内部** | UTC | datetime 对象 | 内部逻辑全部使用 UTC |
| **API 传输** | UTC | ISO 8601 (带 Z) | 如 `2024-03-07T08:15:30Z` |
| **前端展示** | 本地时区 | 格式化字符串 | 自动转换为用户本地时间 |
| **前端提交** | 转为 UTC | ISO 8601 | 提交前转换为 UTC |

**为什么这样做**：
- 避免夏令时、时区变更等问题
- 便于全球化部署和跨时区查询
- 数据一致性和可比较性
- 用户体验友好（自动显示本地时间）

### 后端 (Python/FastAPI)

#### 时间处理规范

**统一工具模块**: `backend/src/pomelo_orbit/infrastructure/time_utils.py`

所有时间操作必须使用统一的工具函数，禁止在各个文件中散落 `datetime.now()` 调用。

```python
# ✅ 正确 - 使用统一工具
from pomelo_orbit.infrastructure.time_utils import utc_now, to_iso8601

user.last_login_at = utc_now()  # 获取当前 UTC 时间
response_time = to_iso8601(dt)  # 转换为 ISO 8601 格式

# ❌ 错误 - 散落的 datetime 调用
from datetime import datetime, timezone
user.last_login_at = datetime.now(timezone.utc).replace(tzinfo=None)
```

**可用函数**:
- `utc_now()` - 获取当前 UTC 时间（naive datetime）
- `to_iso8601(dt)` - 转换为 ISO 8601 格式（API 响应）
- `from_iso8601(iso_string)` - 从 ISO 8601 解析（API 请求）
- `add_minutes/hours/days(dt, n)` - 时间计算

**时区处理原则**:
- 数据库存储：UTC naive datetime
- API 传输：ISO 8601 格式（带 Z 后缀）
- 内部逻辑：UTC datetime
- 前端展示：自动转换为用户本地时区

#### 路由定义

```python
router = APIRouter(prefix="/application", tags=["application"])

@router.get("", response_model=list[ApplicationResp])
def list_applications(): ...

@router.get("/{id}", response_model=ApplicationResp)
def get_application(id: str): ...
```

#### Schema 定义

```python
class ApplicationResp(BaseModel):
    id: str
    name: str
    created_at: datetime

    model_config = {"from_attributes": True}
```

#### 数据验证

使用 Pydantic Field 进行验证，使用正则表达式约束枚举值

```python
type: str = Field(..., pattern="^(github_token|docker_registry|ssh_key)$")
```

### 前端 (Vue 3/TypeScript)

#### 时间处理规范

**统一工具模块**: `frontend/src/utils/time.ts`

所有时间操作必须使用统一的工具函数，禁止在各个文件中散落 `dayjs()` 调用。

```typescript
// ✅ 正确 - 使用统一工具
import { formatTime, isAfterToday } from '@/utils/time';

const displayTime = formatTime(utcTime); // UTC → 本地时间
const isFuture = isAfterToday(utcTime);  // 判断是否在今天之后

// ❌ 错误 - 散落的 dayjs 调用
import dayjs from 'dayjs';
const displayTime = dayjs(time).format('YYYY-MM-DD HH:mm:ss');
```

**可用函数**:
- `formatTime(time, format?)` - UTC 时间转本地时间格式化
- `formatRelativeTime(time)` - 相对时间（如"3分钟前"）
- `isAfterToday(utcTime)` - 判断是否在今天之后
- `toUTC(localTime)` - 本地时间转 UTC（用于提交数据）
- `nowUTC()` - 获取当前 UTC 时间

#### 页面结构

移除顶级 div 包裹，直接使用 `<a-space>` 作为根元素

```vue
<!-- ✅ 正确 -->
<template>
  <a-space direction="vertical" size="large" style="width: 100%">
    <div class="page-header">
      <h2>页面标题</h2>
    </div>
    <a-table ... />
  </a-space>
</template>

<!-- ❌ 错误 -->
<template>
  <div>
    <a-space direction="vertical" size="large" style="width: 100%">
      ...
    </a-space>
  </div>
</template>
```

#### API 客户端

使用单数形式，显式声明返回类型

```typescript
export const applicationApi = {
  get(id: string): Promise<Application> {
    return request.get(`/api/application/${id}`)
  },
  create(data: ApplicationCreate): Promise<Application> {
    return request.post('/api/application', data)
  },
}
```

**重要**：不要使用泛型参数（如 `request.get<T>()`），`request` 拦截器已提取 `response.data`

#### 类型定义

- 与后端 Schema 保持一致
- 使用 TypeScript 严格类型检查

#### 表单验证规范

**核心原则**：使用 Ant Design Vue 的表单验证，禁止手动检查表单字段

所有创建和编辑表单必须使用标准的表单验证机制，不要在提交函数中手动检查字段值。

**方式1：使用 `@finish` 事件（推荐用于简单表单）**

表单验证通过后自动触发，无需手动调用 `validate()`

```vue
<template>
  <a-form
    :model="form"
    :rules="formRules"
    @finish="handleSubmit"
  >
    <a-form-item label="用户名" name="username">
      <a-input v-model:value="form.username" />
    </a-form-item>
    <a-form-item>
      <a-button type="primary" html-type="submit">提交</a-button>
    </a-form-item>
  </a-form>
</template>

<script setup lang="ts">
const form = reactive({
  username: '',
  password: '',
});

const formRules = {
  username: [{ required: true, message: '请输入用户名' }],
  password: [{ required: true, message: '请输入密码' }],
};

// @finish 事件只在验证通过后触发
async function handleSubmit() {
  // 直接处理业务逻辑，无需验证
  await api.submit(form);
}
</script>
```

**方式2：使用 `formRef.validate()`（用于模态窗表单）**

需要手动触发验证，适用于模态窗等场景

```vue
<template>
  <a-modal @ok="handleOk">
    <a-form
      ref="formRef"
      :model="form"
      :rules="formRules"
    >
      <a-form-item label="应用名称" name="name">
        <a-input v-model:value="form.name" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import type { FormInstance } from 'ant-design-vue';

const formRef = ref<FormInstance>();
const form = reactive({
  name: '',
});

const formRules = {
  name: [{ required: true, message: '请输入应用名称' }],
};

async function handleOk() {
  // 先验证表单
  try {
    await formRef.value?.validate();
  } catch {
    return; // 验证失败，直接返回
  }

  // 验证通过，处理业务逻辑
  await api.create(form);
}
</script>
```

**必须包含的元素**：
1. 表单 ref（方式2需要）：`ref="formRef"`
2. 表单模型：`:model="form"`
3. 验证规则：`:rules="formRules"`
4. 字段 name：`<a-form-item name="fieldName">`

**禁止的做法**：

```vue
<!-- ❌ 错误 - 手动检查字段 -->
<script setup lang="ts">
async function handleSubmit() {
  if (!form.name.trim()) {
    message.error('请输入名称');
    return;
  }
  if (!form.value) {
    message.error('请输入值');
    return;
  }
  await api.submit(form);
}
</script>

<!-- ❌ 错误 - 缺少验证规则 -->
<a-form :model="form">
  <a-form-item label="名称">
    <a-input v-model:value="form.name" />
  </a-form-item>
</a-form>

<!-- ❌ 错误 - 缺少 name 属性 -->
<a-form :model="form" :rules="formRules">
  <a-form-item label="名称">
    <a-input v-model:value="form.name" />
  </a-form-item>
</a-form>
```


#### 模态窗命名规范

模态窗变量统一使用 `showXxxModal` 格式命名：

```vue
<!-- ✅ 正确 -->
<script setup lang="ts">
const showApplicationCreate = ref(false);
const showBasicInfoModal = ref(false);
const showCredentialModal = ref(false);

function showApplicationCreateModal() {
  // 初始化表单
  showApplicationCreate.value = true;
}
</script>

<!-- ❌ 错误 -->
<script setup lang="ts">
const modalVisible = ref(false);
const visible = ref(false);
const isOpen = ref(false);
</script>
```

命名规则：
- 列表页创建模态窗：`showXxxCreate`（如 `showApplicationCreate`）
- 详情页编辑模态窗：`showXxxModal`（如 `showBasicInfoModal`）
- 如果页面有多个模态窗，使用具体功能命名（如 `showCredentialModal`、`showAddFileModal`）

#### Loading 处理规范

**列表页**：在 table 上使用 `:loading`

```vue
<template>
  <a-space direction="vertical" style="width: 100%">
    <div class="page-header">
      <h2>应用管理</h2>
    </div>
    <a-table
      :columns="columns"
      :data-source="applications"
      :loading="loading"
      :pagination="pagination"
    />
  </a-space>
</template>
```

**详情页**：在卡片粒度使用 `v-if` 和 `:loading`

```vue
<!-- ✅ 正确 -->
<template>
  <a-space direction="vertical" style="width: 100%">
    <div v-if="application" class="page-header">
      <h2>{{ application.name }}</h2>
    </div>

    <!-- 基本信息卡片 -->
    <a-card title="基本信息" :loading="basicInfoLoading">
      <template v-if="application" #extra>
        <a-button @click="handleEdit">编辑</a-button>
      </template>
      <a-descriptions v-if="application" :column="2" bordered>
        <a-descriptions-item label="名称">
          {{ application.name }}
        </a-descriptions-item>
      </a-descriptions>
    </a-card>

    <!-- 配置文件卡片 -->
    <a-card title="配置文件" :loading="fileListLoading">
      <a-table v-if="files.length > 0" :data-source="files" />
      <a-empty v-else />
    </a-card>
  </a-space>
</template>

<script setup lang="ts">
const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
const { loading: fileListLoading, execute: executeFileList } = useStatusAsync();
const application = ref<Application>();
const files = ref<ConfigFile[]>([]);
</script>

<!-- ❌ 错误 - 外层 v-if 导致看不到页面结构 -->
<template>
  <a-space v-if="application" direction="vertical" style="width: 100%">
    <a-card title="基本信息">
      ...
    </a-card>
  </a-space>
</template>

<!-- ❌ 错误 - 卡片上有 v-if 导致看不到 loading -->
<template>
  <a-card v-if="application" title="基本信息" :loading="loading">
    ...
  </a-card>
</template>
```

核心原则：
- 列表页：table 使用 `:loading`
- 详情页：卡片使用 `:loading`，内容使用 `v-if`
- 多个接口：各自使用独立的 loading 变量
- 数据加载时可以看到页面结构和卡片骨架

#### Detail 页面规范

Detail 页面在卡片粒度使用 v-if，不使用可选链。

```vue
<!-- ✅ 正确 -->
<template>
  <a-space direction="vertical" style="width: 100%">
    <div v-if="application" class="page-header">
      <h2>{{ application.name }}</h2>
    </div>
    <a-card title="基本信息" :loading="loading">
      <a-descriptions v-if="application" :column="2" bordered>
        <a-descriptions-item label="仓库">
          <span v-if="application.git_source">{{ application.git_source.repository_url }}</span>
          <span v-else>-</span>
        </a-descriptions-item>
      </a-descriptions>
    </a-card>
  </a-space>
</template>

<script setup lang="ts">
const application = ref<Application>(); // 非空类型

async function fetchApplication() {
  const data = await applicationApi.get(id);
  application.value = data; // 断言成功后赋值
}
</script>

<!-- ❌ 错误 -->
<template>
  <a-space direction="vertical" style="width: 100%">
    <h2>{{ application?.name || '加载中...' }}</h2>
    <a-descriptions-item label="仓库">
      {{ application?.git_source?.repository_url || '-' }}
    </a-descriptions-item>
  </a-space>
</template>

<script setup lang="ts">
const application = ref<Application | null>(null); // 可空类型
</script>
```

核心原则：
- 数据类型使用非空（`ref<T>()`），不用可空（`ref<T | null>(null)`）
- 在卡片粒度使用 `v-if` 条件渲染，不在字段级别使用 `?.`
- 对可能为 null 的字段使用显式 `v-if/v-else`
- 数据加载失败时跳转回列表页，不显示错误状态
## 质量保证

### 代码检查

- 后端: 使用 ruff 或 pylint
- 前端: 使用 ESLint + TypeScript

### 测试

- 编写单元测试覆盖核心逻辑
- API 测试覆盖所有端点
- 测试文件命名: `test_*.py` 或 `*.test.ts`

## 常见模式

### 分页查询

```python
@router.get("", response_model=PaginatedResp[ApplicationResp])
def list_applications(
    page: int = Query(1, ge=1),
    per_page: int = Query(10, ge=1, le=100),
    search: str | None = Query(None),
):
    # 实现分页和搜索逻辑
    ...
```

### 错误处理

后端异常处理见[架构设计 - 异常处理规范](#异常处理规范)。

#### 前端错误处理

所有 API 错误在拦截器（`frontend/src/utils/request.ts`）中统一处理，业务代码只需 catch Error 对象：

```typescript
// ✅ 正确
try {
  await execute(async () => {
    await api.sync();
    message.success('同步成功');
  });
} catch (error: unknown) {
  message.error(error instanceof Error ? error.message : '操作失败');
}

// ❌ 错误 - 不要在业务代码中处理响应格式
try {
  const res = await api.sync();
  if (res.error) { message.error(res.error.message); }
} catch (error: any) {
  if (error.response?.data?.detail) { message.error(error.response.data.detail); }
}
```

### 级联删除检查

```python
# 检查是否被其他资源使用
apps_using = db.query(ApplicationModel).filter(
    ApplicationModel.credential_id == credential_id
).count()
if apps_using > 0:
    raise HTTPException(
        status_code=status.HTTP_400_BAD_REQUEST,
        detail=f"Credential is used by {apps_using} application(s)",
    )
```

## 禁止事项

❌ 不要使用复数形式的端点和表名

❌ 不要添加兼容层、别名和默认值处理

❌ 不要在接口层捕获异常（交给全局处理器）

❌ 不要提前优化，不要添加未明确需要的功能

❌ 不要在各个文件中散落时间处理逻辑，必须使用统一的工具模块

❌ 不要在日志中以变量起头，不要对预期内的失败使用 `exc_info=True`

## 开发流程

1. 理解需求: 从产品和用户视角审视
2. 设计方案: 选择最简单直接的实现
3. 编写代码: 只写必需的代码
4. 测试验证: 运行 lint 和单元测试
5. 迭代改进: 根据反馈直接修改

## 迁移脚本规范

### 文件命名

```
v0.3.0__unified_application.sql
v0.3.1__init_data.sql
```

### 幂等性

- 使用 `IF NOT EXISTS` 或 `WHERE NOT EXISTS`
- 确保脚本可重复执行

### 初始化数据

- 管理员用户
- 默认配置
- 示例应用（如 pomelo-orbit 自身）

## 示例参考

### 完整的 CRUD API

参考 `backend/src/pomelo_orbit/interfaces/api/application.py`

### 前端列表页

参考 `frontend/src/views/ApplicationPage.vue`

### 前端详情页

参考 `frontend/src/views/ApplicationDetail.vue`

### DTO 定义

参考 `backend/src/pomelo_orbit/interfaces/api/dto/application.py`
