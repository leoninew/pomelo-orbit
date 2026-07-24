# 后端架构

## 技术栈

### 核心框架
- **Python 3.11+** - 编程语言
- **FastAPI** - 现代高性能 Web 框架
- **Pydantic** - 数据验证和设置管理
- **SQLAlchemy 2.0** - ORM 框架
- **Alembic** - 数据库迁移工具

### 数据库
- **SQLite** - 开发和测试环境
- **PostgreSQL** - 生产环境（可选）

### 工具链
- **Ruff** - 快速 Python linter
- **Mypy** - 静态类型检查
- **Pytest** - 测试框架
- **Uvicorn** - ASGI 服务器

## 项目结构

```
backend/
├── src/
│   └── pomelo_orbit/
│       ├── interfaces/          # 接口层
│       │   └── api/            # HTTP API
│       │       ├── ci/         # CI 模块路由
│       │       ├── cd/         # CD 模块路由
│       │       └── auth.py     # 认证路由
│       ├── application/         # 应用层
│       │   ├── services/       # 应用服务
│       │   └── di.py           # 依赖注入
│       ├── domain/              # 领域层
│       │   ├── entities/       # 领域实体
│       │   ├── repositories/   # 仓储接口
│       │   └── services/       # 领域服务
│       ├── infrastructure/      # 基础设施层
│       │   ├── persistence/    # 数据持久化
│       │   │   ├── models/    # SQLAlchemy 模型
│       │   │   ├── repositories/ # 仓储实现
│       │   │   └── di.py      # 数据库依赖注入
│       │   ├── security/       # 安全相关
│       │   ├── docker/         # Docker 客户端
│       │   ├── k8s/            # Kubernetes 客户端
│       │   └── time_utils.py   # 时间工具
│       ├── dto/                 # 数据传输对象
│       │   ├── ci/
│       │   └── cd/
│       ├── main.py              # 应用入口
│       └── config.py            # 配置管理
├── tests/                       # 测试
├── alembic/                     # 数据库迁移
├── ruff.toml                    # Ruff 配置
├── mypy.ini                     # Mypy 配置
└── pyproject.toml               # 项目配置
```

## DDD 分层架构

### 1. 接口层 (Interfaces)

**职责**：
- HTTP 请求/响应适配
- DTO 校验
- 路由定义
- 不包含业务逻辑

**示例**：
```python
# interfaces/api/ci/repository.py
from fastapi import APIRouter, Depends
from typing import Annotated

router = APIRouter(prefix="/repository", tags=["repository"])

@router.get("", response_model=PaginatedResp[RepositoryResp])
def list_repositories(
    page: int = Query(1, ge=1),
    per_page: int = Query(10, ge=1, le=100),
    service: Annotated[RepositoryService, Depends(get_repository_service)],
):
    return service.list_repositories(page, per_page)

@router.post("", status_code=201, response_model=RepositoryResp)
def create_repository(
    data: RepositoryCreateReq,
    service: Annotated[RepositoryService, Depends(get_repository_service)],
):
    return service.create_repository(data)
```

### 2. 应用层 (Application)

**职责**：
- 业务流程编排
- 事务管理
- 调用领域服务和仓储
- 应用服务之间不得互相引用

**示例**：
```python
# application/services/repository_service.py
class RepositoryService:
    def __init__(
        self,
        repository_repo: RepositoryRepository,
        credential_repo: CredentialRepository,
    ):
        self.repository_repo = repository_repo
        self.credential_repo = credential_repo

    def create_repository(self, data: RepositoryCreateReq) -> Repository:
        # 验证凭证存在
        if data.git_credential_id:
            credential = self.credential_repo.find_by_id(data.git_credential_id)
            if not credential:
                raise BusinessError("Credential not found", status_code=404)

        # 创建实体
        repository = Repository(
            id=generate_id(),
            name=data.name,
            code=data.code,
            repository_url=data.repository_url,
            git_credential_id=data.git_credential_id,
            created_at=utc_now(),
        )

        # 持久化
        return self.repository_repo.save(repository)
```

### 3. 领域层 (Domain)

**职责**：
- 核心业务规则
- 实体行为
- 领域服务（跨聚合的业务逻辑）
- 仓储接口定义

**实体示例**：
```python
# domain/entities/repository.py
from dataclasses import dataclass
from datetime import datetime

@dataclass
class Repository:
    id: str
    name: str
    code: str
    repository_url: str
    default_branch: str
    git_credential_id: str | None
    created_at: datetime
    updated_at: datetime | None = None

    def update_credential(self, credential_id: str | None) -> None:
        """更新关联的凭证"""
        self.git_credential_id = credential_id
        self.updated_at = utc_now()
```

**仓储接口示例**：
```python
# domain/repositories/repository_repository.py
from abc import ABC, abstractmethod
from typing import List

class RepositoryRepository(ABC):
    @abstractmethod
    def find_by_id(self, id: str) -> Repository | None:
        pass

    @abstractmethod
    def find_by_code(self, code: str) -> Repository | None:
        pass

    @abstractmethod
    def list_all(self) -> List[Repository]:
        pass

    @abstractmethod
    def save(self, repository: Repository) -> Repository:
        pass

    @abstractmethod
    def delete(self, repository: Repository) -> None:
        pass
```

**领域服务示例**：
```python
# domain/services/variable_resolver.py
class VariableResolver:
    """变量解析领域服务"""

    def resolve(
        self,
        template: str,
        variables: dict[str, Any],
        built_in_vars: dict[str, Any],
    ) -> str:
        """解析模板中的变量"""
        # 领域逻辑：变量优先级、格式验证等
        pass
```

### 4. 基础设施层 (Infrastructure)

**职责**：
- 数据库访问
- 外部服务调用
- 技术工具实现

**仓储实现示例**：
```python
# infrastructure/persistence/repositories/repository_repository_impl.py
from sqlalchemy.orm import Session
from domain.repositories.repository_repository import RepositoryRepository
from infrastructure.persistence.models.repository_model import RepositoryModel

class RepositoryRepositoryImpl(RepositoryRepository):
    def __init__(self, db: Session):
        self.db = db

    def find_by_id(self, id: str) -> Repository | None:
        model = self.db.query(RepositoryModel).filter_by(id=id).first()
        return self._to_entity(model) if model else None

    def save(self, repository: Repository) -> Repository:
        model = self._to_model(repository)
        self.db.add(model)
        self.db.commit()
        self.db.refresh(model)
        return self._to_entity(model)

    def _to_entity(self, model: RepositoryModel) -> Repository:
        return Repository(
            id=model.id,
            name=model.name,
            code=model.code,
            repository_url=model.repository_url,
            default_branch=model.default_branch,
            git_credential_id=model.git_credential_id,
            created_at=model.created_at,
            updated_at=model.updated_at,
        )

    def _to_model(self, entity: Repository) -> RepositoryModel:
        return RepositoryModel(
            id=entity.id,
            name=entity.name,
            code=entity.code,
            repository_url=entity.repository_url,
            default_branch=entity.default_branch,
            git_credential_id=entity.git_credential_id,
            created_at=entity.created_at,
            updated_at=entity.updated_at,
        )
```

## 依赖注入 (DI)

### DI 层次结构

```
infrastructure/persistence/di.py   # get_db
infrastructure/di.py               # get_xxx_repository, get_xxx_client
application/di.py                  # get_xxx_service
interfaces/api/                    # 路由内联依赖注入
```

### 实现示例

**persistence 层**：
```python
# infrastructure/persistence/di.py
from sqlalchemy.orm import Session

def get_db() -> Generator[Session, None, None]:
    with SessionLocal() as session:
        yield session
```

**infrastructure 层**：
```python
# infrastructure/di.py
from typing import Annotated
from fastapi import Depends

def get_repository_repository(
    db: Annotated[Session, Depends(get_db)]
) -> Generator[RepositoryRepositoryImpl, None, None]:
    yield RepositoryRepositoryImpl(db)

def get_docker_client(
    settings: Annotated[Dynaconf, Depends(get_settings)]
) -> DockerClient:
    host = settings.docker.host
    assert host, "POMELO_ORBIT_DOCKER__HOST is not configured"
    return DockerClient(base_url=host)
```

**application 层**：
```python
# application/di.py
def get_repository_service(
    repository_repo: Annotated[RepositoryRepositoryImpl, Depends(get_repository_repository)],
    credential_repo: Annotated[CredentialRepositoryImpl, Depends(get_credential_repository)],
) -> RepositoryService:
    return RepositoryService(repository_repo, credential_repo)
```

## 核心设计原则

### 1. 实体默认值规范

**原则**：领域实体的业务字段不设默认值，业务代码必须显式传递

```python
# ✅ 正确
@dataclass
class Application:
    id: str
    name: str
    deployment_type: str    # 无默认值，必须显式传递
    status: str             # 无默认值，必须显式传递
    auto_deploy: bool       # 无默认值，必须显式传递
    description: str | None = None  # 可选字段

# ❌ 错误
@dataclass
class Application:
    deployment_type: str = "docker"  # 静默默认，可能漏传
    auto_deploy: bool = False        # 静默默认，可能忘记设置
```

### 2. 配置取值规范

**原则**：直接访问配置字段，用 `assert` 断言必填项非空

```python
# ✅ 正确
def get_docker_client(settings: Dynaconf) -> DockerClient:
    host: str = settings.docker.host
    assert host, "POMELO_ORBIT_DOCKER__HOST is not configured"
    return DockerClient(base_url=host)

# ❌ 错误
def get_docker_client(settings: Dynaconf) -> DockerClient:
    docker = getattr(settings, "docker", None)
    host = getattr(docker, "host", "") if docker else ""
    return DockerClient(base_url=host)
```

### 3. 异常处理规范

**原则**：Service 层抛出业务异常，接口层不捕获，由全局异常处理器统一处理

```python
# ✅ 正确
class RepositoryService:
    def delete_repository(self, repository_id: str) -> None:
        repository = self.repository_repo.find_by_id(repository_id)
        if not repository:
            raise BusinessError(f"Repository {repository_id} not found", status_code=404)
        self.repository_repo.delete(repository)

@router.delete("/{repository_id}", status_code=204)
def delete_repository(
    repository_id: str,
    service: Annotated[RepositoryService, Depends(get_repository_service)],
):
    service.delete_repository(repository_id)  # 不需要 try-except
```

### 4. 日志规范

**原则**：日志以业务描述开头，参数以 `key=value` 格式跟在后面

```python
# ✅ 正确
logger.info(f"Pipeline run completed: run_id={run_id}, status={status}, duration={duration}s")
logger.warning(f"Deployment failed: app_id={app_id}, reason=image_not_found")
logger.error(f"Container start failed: app_id={app_id}, error={e}", exc_info=True)

# ❌ 错误
logger.info(f"[{run_id}] completed pipeline")  # 以变量起头
logger.warning(f"Token expired: user={user_id}", exc_info=True)  # warning 不需要 exc_info
logger.error(f"Start failed: {e}")  # error 缺少 exc_info
```

## 数据库设计

### 表命名规范

- 使用单数形式
- 使用 snake_case
- 外键引用使用单数表名

```python
# ✅ 正确
class CredentialModel(Base):
    __tablename__ = "credential"

class RepositoryModel(Base):
    __tablename__ = "repository"
    git_credential_id: Mapped[str | None] = mapped_column(
        String(26), ForeignKey("credential.id")  # 单数
    )
```

### 外键约束

- 使用外键约束保证数据完整性
- 使用 `ON DELETE CASCADE` 自动清理关联数据

```python
git_credential_id: Mapped[str | None] = mapped_column(
    String(26),
    ForeignKey("credential.id", ondelete="CASCADE"),
    nullable=True,
)
```

## 时间处理

### 统一使用工具模块

```python
from pomelo_orbit.infrastructure.time_utils import utc_now, to_iso8601

# 获取当前 UTC 时间
run.started_at = utc_now()

# 转换为 ISO 8601 格式
response_time = to_iso8601(dt)
```

### 时间存储规范

| 层级 | 时区 | 格式 | 说明 |
|------|------|------|------|
| 数据库 | UTC | naive datetime | 统一时区，避免混乱 |
| 后端内部 | UTC | datetime 对象 | 内部逻辑全部使用 UTC |
| API 传输 | UTC | ISO 8601 (带 Z) | 如 `2024-03-07T08:15:30Z` |

## 数据验证

### Pydantic 声明式校验

```python
from pydantic import BaseModel, Field
from typing import Literal

class RepositoryCreateReq(BaseModel):
    name: str = Field(min_length=1)
    code: str = Field(min_length=1, pattern=r"^[a-z0-9_-]+$")
    repository_url: str = Field(min_length=1)
    default_branch: str = Field(default="master", min_length=1)
    git_credential_id: str | None = None

class RepositoryResp(BaseModel):
    id: str
    name: str
    code: str
    repository_url: str
    created_at: datetime

    model_config = {"from_attributes": True}
```

## 测试策略

### 单元测试

```python
# tests/test_repository_service.py
import pytest
from pomelo_orbit.application.services.repository_service import RepositoryService

def test_create_repository(repository_service: RepositoryService):
    data = RepositoryCreateReq(
        name="Test Repo",
        code="test-repo",
        repository_url="https://github.com/test/repo",
    )

    repository = repository_service.create_repository(data)

    assert repository.id is not None
    assert repository.name == "Test Repo"
    assert repository.code == "test-repo"
```

### 测试数据库

使用内存 SQLite：
```bash
POMELO_ORBIT_DATABASE__SQLITE__PATH=:memory: pytest
```

## 相关文档

- [编码规范](../../CLAUDE.md)
- [开发工作流](../development/workflow.md)
- [前端架构](../frontend/architecture.md)
