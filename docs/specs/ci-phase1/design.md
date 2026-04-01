# CI Phase 1 设计文档

## 架构概览

```
API 层（FastAPI）
    ↓
Service 层（业务逻辑）
    ↓
Domain 层（实体 + 执行器接口）
    ↓
Infrastructure 层（数据库 + Docker 执行器实现）
```

## 领域模型

### 实体

#### Project（项目）

```python
class Project:
    id: UUID
    name: str
    repository_url: str  # https:// 或 git@
    pipeline_template_id: UUID
    git_credential_id: UUID
    variable_overrides: dict[str, Any]  # 项目级变量
    created_at: datetime
    updated_at: datetime
```

#### Credential（凭据）

```python
class CredentialType(str, Enum):
    GIT_SSH = "git_ssh"
    GIT_TOKEN = "git_token"

class Credential:
    id: UUID
    name: str
    type: CredentialType
    encrypted_data: str  # Fernet 加密后的 JSON
    created_at: datetime
```

#### PipelineTemplate（模板）

```python
class VariableDeclaration:
    name: str
    description: str
    required: bool
    default: Any | None
    secret: bool  # 是否敏感数据

class PipelineTemplate:
    id: UUID
    name: str
    description: str
    content: str  # YAML 文本
    variable_declarations: list[VariableDeclaration]
    is_builtin: bool
    created_at: datetime
    updated_at: datetime
```

#### PipelineRun（执行实例）

```python
class PipelineRunStatus(str, Enum):
    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"

class PipelineRunTrigger(str, Enum):
    MANUAL = "manual"

class PipelineRun:
    id: UUID
    project_id: UUID
    trigger: PipelineRunTrigger
    trigger_ref: str  # branch/tag/commit
    resolved_pipeline: str  # 渲染后的 YAML 快照
    variables_snapshot: dict[str, Any]  # secret 已脱敏
    status: PipelineRunStatus
    started_at: datetime | None
    finished_at: datetime | None
    created_at: datetime
```

#### Job（执行单元）

```python
class JobStatus(str, Enum):
    WAITING = "waiting"
    RUNNING = "running"
    SUCCESS = "success"
    FAILED = "failed"
    FAULTED = "faulted"

class Job:
    id: UUID
    pipeline_run_id: UUID
    name: str
    parent_job_id: UUID | None  # 嵌套 step 的父 job
    status: JobStatus
    started_at: datetime | None
    finished_at: datetime | None
    exit_code: int | None
    error_message: str | None
```

#### JobLog（日志）

```python
class JobLog:
    id: UUID
    job_id: UUID
    content: str  # 完整日志文本
    created_at: datetime
```

### 值对象

#### PipelineDefinition（解析后的 pipeline）

```python
class StepDefinition:
    name: str
    image: str | None
    commands: list[str] | None
    uses: str | None  # "checkout"
    with_: dict[str, Any] | None
    volumes: list[str] | None
    steps: list['StepDefinition'] | None  # 嵌套

class PipelineDefinition:
    version: str
    timeout: int | None
    steps: list[StepDefinition]
```

## 核心流程

### 1. 手动触发 Pipeline

```
用户请求
  ↓
API: POST /projects/{id}/runs
  ↓
Service: 创建 PipelineRun
  ├─ 查询 Project（含 template 和 credential）
  ├─ 合并变量（全局 ← 项目 ← 运行时）
  ├─ 校验 required 变量
  ├─ 渲染 pipeline YAML（Jinja2）
  ├─ 保存快照（resolved_pipeline + variables_snapshot 脱敏）
  ├─ 创建 PipelineRun（status=waiting）
  └─ 提交后台任务
  ↓
后台任务: 执行 Pipeline
  ├─ 更新 status=running
  ├─ 创建 workspace 目录
  ├─ 解析 PipelineDefinition
  ├─ 调用 PipelineExecutor.execute()
  ├─ 更新 status=success/failed
  └─ 清理 workspace
```

### 2. Pipeline 执行（串行）

```
PipelineExecutor.execute(run, definition)
  ↓
遍历 definition.steps（串行）
  ↓
  对于每个 step:
    ├─ 创建 Job 记录（status=waiting）
    ├─ 更新 status=running
    ├─ 判断类型：
    │   ├─ uses="checkout" → 执行 checkout action
    │   ├─ 有 image/commands → 执行容器
    │   └─ 有 steps → 递归执行子 steps
    ├─ 收集日志 → 创建 JobLog
    ├─ 更新 status=success/failed/faulted
    └─ 如果 failed/faulted → 中断，返回失败
  ↓
所有 step 成功 → 返回成功
```

### 3. Checkout Action 执行

```
CheckoutAction.execute(step, run, workspace_path)
  ↓
  ├─ 从 step.with_ 提取参数（depth, ref, sparse_checkout, submodules）
  ├─ 从 run.project 获取 repository_url 和 credential
  ├─ 解密 credential.encrypted_data
  ├─ 根据 credential.type 准备注入：
  │   ├─ git_ssh: 写临时私钥文件，设置 GIT_SSH_COMMAND
  │   └─ git_token: 构造 https://{token}@... URL
  ├─ 构造 docker run 命令：
  │   ├─ 镜像: alpine/git
  │   ├─ 挂载: workspace_path:/workspace
  │   ├─ 环境变量: GIT_SSH_COMMAND 或 GIT_ASKPASS
  │   └─ 命令: git clone --depth={depth} {url} /workspace
  ├─ 执行容器，收集日志
  ├─ 提取 outputs（git log --format=...）
  ├─ 清理临时文件
  └─ 返回 outputs
```

### 4. 容器执行

```
ContainerExecutor.run(image, commands, volumes, env, workspace_path)
  ↓
  ├─ 构造 docker run 参数：
  │   ├─ --rm（自动清理）
  │   ├─ -v {workspace_path}:/workspace
  │   ├─ -v {artifacts_path}:/artifacts
  │   ├─ -w /workspace（工作目录）
  │   ├─ -e {env}（环境变量）
  │   └─ {image} {commands}
  ├─ 执行 docker run，捕获 stdout/stderr
  ├─ 等待容器退出
  ├─ 获取退出码
  └─ 返回（exit_code, logs）
```

## 数据库设计

### 表结构

```sql
-- 项目表
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    repository_url TEXT NOT NULL,
    pipeline_template_id UUID NOT NULL REFERENCES pipeline_templates(id),
    git_credential_id UUID NOT NULL REFERENCES credentials(id),
    variable_overrides JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- 凭据表
CREATE TABLE credentials (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- 模板表
CREATE TABLE pipeline_templates (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    variable_declarations JSONB NOT NULL,
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- 运行表
CREATE TABLE pipeline_runs (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id),
    trigger VARCHAR(50) NOT NULL,
    trigger_ref VARCHAR(255) NOT NULL,
    resolved_pipeline TEXT NOT NULL,
    variables_snapshot JSONB NOT NULL,
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    INDEX idx_project_created (project_id, created_at DESC)
);

-- Job 表
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id),
    name VARCHAR(255) NOT NULL,
    parent_job_id UUID REFERENCES jobs(id),
    status VARCHAR(50) NOT NULL,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    exit_code INTEGER,
    error_message TEXT,
    INDEX idx_run_parent (pipeline_run_id, parent_job_id)
);

-- 日志表
CREATE TABLE job_logs (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id),
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    INDEX idx_job (job_id)
);
```

## API 设计

### Project

- `POST /api/v1/projects` - 创建项目
- `GET /api/v1/projects` - 查询项目列表
- `GET /api/v1/projects/{id}` - 查询项目详情
- `PUT /api/v1/projects/{id}` - 更新项目
- `DELETE /api/v1/projects/{id}` - 删除项目

### Credential

- `POST /api/v1/credentials` - 创建凭据
- `GET /api/v1/credentials` - 查询凭据列表
- `DELETE /api/v1/credentials/{id}` - 删除凭据

### PipelineTemplate

- `POST /api/v1/pipeline-templates` - 创建模板
- `GET /api/v1/pipeline-templates` - 查询模板列表
- `GET /api/v1/pipeline-templates/{id}` - 查询模板详情
- `PUT /api/v1/pipeline-templates/{id}` - 更新模板
- `DELETE /api/v1/pipeline-templates/{id}` - 删除模板

### PipelineRun

- `POST /api/v1/projects/{id}/runs` - 手动触发
- `GET /api/v1/projects/{id}/runs` - 查询项目的运行列表
- `GET /api/v1/runs/{id}` - 查询运行详情
- `GET /api/v1/runs/{id}/jobs` - 查询运行的 Job 树
- `GET /api/v1/jobs/{id}/logs` - 查询 Job 日志

## 技术选型

- YAML 解析：`pyyaml`
- 模板渲染：`jinja2`（复用现有）
- 加密：`cryptography.fernet`（复用现有）
- Docker 操作：`docker` Python SDK
- 后台任务：`asyncio.create_task`（Phase 1 简单实现，Phase 2+ 考虑 Celery/Temporal）

## 目录结构

```
backend/src/pomelo_orbit/
├── domain/
│   ├── ci/
│   │   ├── entities.py          # Project, Credential, PipelineTemplate, PipelineRun, Job
│   │   ├── value_objects.py     # PipelineDefinition, StepDefinition, JobStatus, etc.
│   │   └── executor.py          # PipelineExecutor 接口
├── application/
│   └── ci/
│       ├── project_service.py
│       ├── credential_service.py
│       ├── template_service.py
│       └── pipeline_service.py
├── infrastructure/
│   ├── ci/
│   │   ├── repositories.py      # SQLAlchemy repositories
│   │   ├── executor_impl.py     # AsyncioPipelineExecutor
│   │   ├── container.py         # ContainerExecutor (Docker SDK)
│   │   └── actions/
│   │       └── checkout.py      # CheckoutAction
│   └── encryption.py            # 复用现有 Fernet
└── api/
    └── v1/
        └── ci/
            ├── projects.py
            ├── credentials.py
            ├── templates.py
            └── runs.py
```
