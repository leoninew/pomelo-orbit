# CI Phase 2 设计文档

## 架构变更

Phase 2 在 Phase 1 基础上增加：
- Webhook 接收和处理模块
- 并行执行调度器
- Artifact 实体和存储
- 重试逻辑

## 新增实体

### Artifact（制品）

```python
class ArtifactType(str, Enum):
    DOCKER_IMAGE = "docker_image"
    FILE = "file"

class Artifact:
    id: str  # ULID
    pipeline_run_id: str
    job_id: str
    type: ArtifactType
    name: str  # 镜像 tag 或文件名
    path: str | None  # file 类型时的宿主机路径
    created_at: datetime
```

### WebhookEvent（Webhook 事件）

复用现有的 `WebhookEvent` 实体，扩展用于 CI：

```python
# 已有字段
class WebhookEvent:
    id: str
    source: WebhookSource  # github
    event_type: WebhookEventType  # push, release
    status: WebhookEventStatus  # received, matched, ignored, error
    repository_url: str | None
    branch: str | None
    sender: str | None
    payload: str | None
    signature_valid: bool | None
    matched_application_id: str | None  # 改为 matched_project_id
    triggered_deployment_id: str | None  # 改为 triggered_pipeline_run_id
    error_message: str | None
    received_at: datetime
    processed_at: datetime | None
```

## 核心流程

### 1. Webhook 触发流程

```
GitHub/GitLab 发送 webhook
  ↓
API: POST /api/v1/webhooks/git
  ↓
验证签名（HMAC-SHA256）
  ├─ 签名无效 → 返回 401，记录事件（signature_valid=false）
  └─ 签名有效 → 继续
  ↓
解析 payload
  ├─ 提取 repository_url, branch, commit_sha, author
  └─ 记录 WebhookEvent（status=received）
  ↓
匹配 Project
  ├─ 按 repository_url 查询 Project
  ├─ 检查分支过滤（如果配置）
  ├─ 无匹配 → 更新事件（status=ignored）
  └─ 有匹配 → 更新事件（status=matched, matched_project_id）
  ↓
触发 PipelineRun
  ├─ trigger=webhook
  ├─ trigger_ref=branch 或 commit_sha
  ├─ 合并变量（全局 ← 项目 ← webhook 内置变量）
  ├─ 渲染 pipeline
  ├─ 创建 PipelineRun
  └─ 更新事件（triggered_pipeline_run_id）
  ↓
返回 200 OK
```

### 2. 并行执行流程

```
PipelineExecutor.execute(run, definition)
  ↓
构建依赖图（DAG）
  ├─ 遍历所有 steps
  ├─ 提取 depends_on 关系
  └─ 构建 Job 依赖图
  ↓
拓扑排序
  ├─ 找出所有无依赖的 Job（入度为 0）
  └─ 按层级分组
  ↓
按层级并行执行
  ↓
  对于每一层：
    ├─ 使用 asyncio.gather 并行执行该层所有 Job
    ├─ 等待该层所有 Job 完成
    ├─ 检查是否有 Job 失败
    │   ├─ 有失败 → Fail-fast，标记未开始的 Job 为 canceled
    │   └─ 无失败 → 继续下一层
    └─ 更新依赖图（移除已完成的 Job）
  ↓
所有层完成 → 返回成功/失败
```

### 3. 制品记录流程

```
Job 执行完成
  ↓
检查 step 是否声明 artifacts
  ├─ 无声明 → 跳过
  └─ 有声明 → 继续
  ↓
检查 Job 状态
  ├─ 失败/故障 → 不记录制品
  └─ 成功 → 继续
  ↓
遍历 artifacts 声明
  ↓
  对于每个 artifact：
    ├─ 渲染变量（如 {{ trigger_ref }}）
    ├─ 判断类型（docker_image | file）
    ├─ docker_image：记录镜像 tag
    ├─ file：
    │   ├─ 检查文件是否存在于 artifacts 目录
    │   ├─ 记录文件路径
    │   └─ 保留文件（不清理）
    └─ 创建 Artifact 记录
  ↓
返回制品列表
```

### 4. 重试流程

```
用户请求重试
  ↓
API: POST /api/v1/runs/{id}/retry
  ↓
查询原 PipelineRun
  ├─ 不存在 → 返回 404
  └─ 存在 → 继续
  ↓
创建新 PipelineRun
  ├─ retry_of = 原 Run ID
  ├─ 复用 resolved_pipeline
  ├─ 复用 variables_snapshot
  ├─ trigger = 原 trigger
  └─ trigger_ref = 原 trigger_ref
  ↓
执行 Pipeline
  ↓
  对于每个 step：
    ├─ 检查 retry_policy
    │   ├─ always_rerun → 执行
    │   └─ skip_if_success → 检查上次状态
    ├─ 查询原 Run 中对应 Job 的状态
    ├─ 检查依赖链是否有重新运行的 Job
    │   ├─ 有 → 当前 Job 也重新运行
    │   └─ 无 → 检查上次状态
    ├─ 上次成功 → 标记为 skipped，跳过执行
    └─ 上次失败/不存在 → 执行
  ↓
返回新 PipelineRun ID
```

## 数据库变更

### 新增表

```sql
-- 制品表
CREATE TABLE artifacts (
    id VARCHAR(26) PRIMARY KEY,  -- ULID
    pipeline_run_id VARCHAR(26) NOT NULL REFERENCES pipeline_runs(id),
    job_id VARCHAR(26) NOT NULL REFERENCES jobs(id),
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    path TEXT,
    created_at TIMESTAMP NOT NULL,
    INDEX idx_run (pipeline_run_id),
    INDEX idx_job (job_id)
);
```

### 修改表

```sql
-- Project 表增加字段
ALTER TABLE projects ADD COLUMN webhook_secret VARCHAR(255);
ALTER TABLE projects ADD COLUMN branch_filter VARCHAR(255);  -- 可选，如 "main,develop"

-- PipelineRun 表增加字段
ALTER TABLE pipeline_runs ADD COLUMN retry_of VARCHAR(26) REFERENCES pipeline_runs(id);

-- Job 表增加字段（已有，无需修改）
-- depends_on 存储在 StepDefinition 中，不需要数据库字段

-- WebhookEvent 表字段重命名（如果需要）
-- matched_application_id → matched_project_id
-- triggered_deployment_id → triggered_pipeline_run_id
```

## API 设计

### Webhook

- `POST /api/v1/webhooks/git` - 接收 Git webhook
  - Header: `X-Hub-Signature-256` (GitHub) 或 `X-Gitlab-Token` (GitLab)
  - Body: webhook payload (JSON)
  - Response: `200 OK` 或 `401 Unauthorized`

### Artifact

- `GET /api/v1/runs/{id}/artifacts` - 查询 Run 的制品列表
- `GET /api/v1/artifacts/{id}/download` - 下载文件制品

### Retry

- `POST /api/v1/runs/{id}/retry` - 重试 PipelineRun
  - Body: `{ "runtime_variables": {...} }`（可选）
  - Response: `{ "run_id": "..." }`

### Project 扩展

- `GET /api/v1/projects/{id}/webhook-url` - 获取 webhook URL 和 secret

## 技术实现

### 并行执行

使用 asyncio.gather 实现：

```python
async def execute_layer(jobs: list[Job]) -> list[JobResult]:
    tasks = [execute_job(job) for job in jobs]
    results = await asyncio.gather(*tasks, return_exceptions=True)
    return results
```

### 依赖图构建

使用邻接表表示：

```python
class DependencyGraph:
    def __init__(self, steps: list[StepDefinition]):
        self.graph: dict[str, list[str]] = {}  # job_name -> [依赖的 job_name]
        self.in_degree: dict[str, int] = {}    # job_name -> 入度
        
    def build(self):
        for step in steps:
            self.graph[step.name] = step.depends_on or []
            self.in_degree[step.name] = len(step.depends_on or [])
    
    def get_ready_jobs(self) -> list[str]:
        return [name for name, degree in self.in_degree.items() if degree == 0]
    
    def mark_completed(self, job_name: str):
        for name in self.graph:
            if job_name in self.graph[name]:
                self.in_degree[name] -= 1
```

### Webhook 签名验证

GitHub 使用 HMAC-SHA256：

```python
import hmac
import hashlib

def verify_github_signature(payload: bytes, signature: str, secret: str) -> bool:
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### 重试跳过判断

```python
def should_skip(
    step: StepDefinition,
    original_run: PipelineRun,
    rerun_jobs: set[str]
) -> bool:
    if step.retry_policy == "always_rerun":
        return False
    
    # 查询原 Run 中对应 Job
    original_job = find_job(original_run, step.name)
    if not original_job or original_job.status != JobStatus.SUCCESS:
        return False
    
    # 检查依赖链
    for dep in step.depends_on or []:
        if dep in rerun_jobs:
            return False
    
    return True
```

## 目录结构

```
backend/src/pomelo_orbit/
├── domain/
│   └── ci/
│       ├── entities.py          # 增加 Artifact
│       └── value_objects.py     # 增加 ArtifactType
├── application/
│   └── ci/
│       ├── webhook_service.py   # 新增
│       └── pipeline_service.py  # 扩展 retry 方法
├── infrastructure/
│   └── ci/
│       ├── repositories.py      # 增加 ArtifactRepository
│       ├── executor_impl.py     # 扩展并行执行
│       └── dependency_graph.py  # 新增
└── api/
    └── v1/
        └── ci/
            ├── webhooks.py      # 新增
            ├── artifacts.py     # 新增
            └── runs.py          # 扩展 retry 端点
```

## 测试策略

### 单元测试
- 依赖图构建和拓扑排序
- Webhook 签名验证
- 重试跳过逻辑
- 制品记录逻辑

### 集成测试
- 端到端 webhook 触发
- 并行执行多个 Job
- 重试并跳过成功 Job
- 制品记录和下载

### 性能测试
- 并行执行性能提升
- 大量 Job 的依赖图构建性能
