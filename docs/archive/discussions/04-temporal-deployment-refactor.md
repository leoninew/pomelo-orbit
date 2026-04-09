# [探索性讨论] 使用 Temporal 改进部署流程的可能性

> 本文档为探索性讨论记录, 记录的是关于可能方案的思考和交流, 并非最终确定的实施计划或规划。内容可能包含未经验证的假设和理想化的设计, 实际实施时需要进一步评估和调整。

## 一、当前部署架构的问题

### 现有实现

- 触发方式: FastAPI BackgroundTasks
- 并发控制: asyncio.Lock (进程内)
- 状态追踪: 数据库字段 (deployment.status, application.status)
- 日志收集: 文件系统

### 存在的问题

1. 无持久化: BackgroundTasks 在内存中运行, 服务器重启会丢失正在执行的部署
2. 无分布式锁: asyncio.Lock 只在单进程内有效, 多实例部署会产生竞态条件
3. 无重试机制: 瞬时故障 (网络、Docker 守护进程) 直接导致 FAULTED 状态
4. 无超时保护: 长时间运行的部署可能无限挂起
5. 锁字典泄漏: self._locks 无限增长, 不清理过期条目
6. 取消操作无效: cancel_deployment() 只更新数据库状态, 不实际停止运行中的子进程
7. 后台任务的 DB session 问题: BackgroundTask 使用的 domain entity 已脱离 DB session

## 二、Temporal 方案设计

### 整体架构

```
┌─────────────────────────────────────────────────┐
│              FastAPI (接口层)                    │
│  接收请求 → 创建 Deployment → 启动 Temporal 工作流│
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│            Temporal Server (编排层)              │
│  工作流编排、状态持久化、自动重试、超时控制        │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│           Temporal Worker (执行层)               │
│  执行 Activity: 写配置 → 运行 init → docker compose up │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│              Docker Engine                       │
│  容器生命周期管理                                  │
└─────────────────────────────────────────────────┘
```

### 核心组件

#### Temporal Workflow

```python
from temporalio import workflow
from datetime import timedelta

@workflow.defn
class DeployWorkflow:
    @workflow.run
    async def run(self, input: DeployInput) -> DeployResult:
        # 1. 验证应用状态
        await workflow.execute_activity(
            validate_deploy,
            input.app_id,
            start_to_close_timeout=timedelta(seconds=30)
        )

        # 2. 写入配置文件
        await workflow.execute_activity(
            write_config_files,
            WriteConfigInput(
                app_id=input.app_id,
                config_files=input.config_files
            ),
            start_to_close_timeout=timedelta(minutes=5)
        )

        # 3. 运行初始化脚本 (可选)
        if input.has_init_script:
            await workflow.execute_activity(
                run_init_script,
                input.app_id,
                start_to_close_timeout=timedelta(minutes=30)
            )

        # 4. Docker Compose 部署
        await workflow.execute_activity(
            compose_up,
            ComposeUpInput(
                app_id=input.app_id,
                pull_policy=input.pull_policy,
                env_file=input.env_file
            ),
            start_to_close_timeout=timedelta(hours=1),
            retry_policy=RetryPolicy(
                maximum_attempts=3,
                initial_interval=timedelta(seconds=10),
                backoff_coefficient=2.0
            )
        )

        return DeployResult(success=True)
```

#### Temporal Activity

```python
from temporalio import activity
from pomelo_orbit.infrastructure.docker.manager import DockerComposeManager

@activity.defn
async def validate_deploy(app_id: str) -> None:
    app_repo = get_application_repository()
    app = app_repo.find_by_id(app_id)
    if not app.can_deploy():
        raise ApplicationNotDeployableError(app_id)

@activity.defn
async def write_config_files(input: WriteConfigInput) -> None:
    manager = get_docker_manager()
    for config_file in input.config_files:
        manager._write_file(
            app_code=input.app_id,
            path=config_file.path,
            content=config_file.content
        )

@activity.defn
async def compose_up(input: ComposeUpInput) -> None:
    manager = get_docker_manager()
    manager._compose_up(
        app_code=input.app_id,
        pull_policy=input.pull_policy,
        env_file=input.env_file
    )
```

#### Workflow 启动

```python
# interfaces/api/application.py
from temporalio.client import Client

async def deploy_application(
    app_id: str,
    temporal_client: Client,
    ...
):
    app = app_service.get_application(app_id)
    deployment = app_service.create_deployment(app, "deploy", "manual")

    # 启动 Temporal 工作流
    handle = await temporal_client.start_workflow(
        "DeployWorkflow",
        DeployInput(app_id=app_id, ...),
        id=f"deploy-{deployment.id}",
        task_queue="deploy-queue"
    )

    return {"deployment_id": deployment.id, "workflow_id": handle.id}
```

### 并发控制

Temporal 原生支持:

- 同一应用的部署自动排队 (通过 workflow ID 唯一性)
- 不需要手动管理 asyncio.Lock
- 支持跨进程、跨机器的分布式锁

```python
# 使用 workflow ID 确保同一应用的部署串行执行
workflow_id = f"deploy-{app_id}"
await client.start_workflow(
    "DeployWorkflow",
    ...,
    id=workflow_id
)
```

### 重试机制

Temporal 原生支持:

- Activity 级别的自动重试
- 可配置重试次数、间隔、退避系数
- 区分可重试和不可重试错误

```python
# 配置重试策略
retry_policy=RetryPolicy(
    maximum_attempts=3,
    initial_interval=timedelta(seconds=10),
    backoff_coefficient=2.0,
    non_retryable_error_types=["ApplicationNotDeployableError"]
)
```

### 超时控制

Temporal 原生支持:

- Workflow 级别超时
- Activity 级别超时
- 心跳超时 (长时间运行的 Activity)

```python
# 配置超时
start_to_close_timeout=timedelta(hours=1),
heartbeat_timeout=timedelta(minutes=5)
```

### 取消操作

Temporal 原生支持:

- 取消正在运行的工作流
- 工作流收到取消信号后执行清理逻辑
- 状态持久化, 重启后仍可取消

```python
# 取消工作流
await client.get_workflow_handle(workflow_id).cancel()

# 工作流内处理取消
@workflow.defn
class DeployWorkflow:
    @workflow.run
    async def run(self, input: DeployInput) -> DeployResult:
        try:
            await workflow.execute_activity(...)
        except asyncio.CancelledError:
            # 执行清理逻辑
            await workflow.execute_activity(
                cleanup_deploy,
                input.app_id
            )
            raise
```

## 三、迁移策略

### Phase 1: 引入 Temporal 基础设施

1. 添加 temporalio 依赖到 pyproject.toml
2. 创建 docker-compose.temporal.yml (Temporal Server + UI)
3. 创建 Temporal Worker 进程
4. 定义基础工作流和 Activity

### Phase 2: 迁移部署流程

1. 将 ApplicationService.deploy() 拆分为多个 Activity
2. 创建 DeployWorkflow
3. 修改 API 端点, 使用 Temporal Client 启动工作流
4. 保留原有 BackgroundTasks 作为降级方案

### Phase 3: 迁移其他异步操作

1. 迁移 restart 操作
2. 迁移 stop 操作
3. 迁移 delete 操作
4. 移除 asyncio.Lock 和 BackgroundTasks

### Phase 4: 扩展能力

1. 添加 CI Pipeline 工作流
2. 添加通知工作流
3. 添加缓存管理工作流

## 四、代码变更概览

### 新增文件

- workflows/deploy.py: DeployWorkflow 定义
- workflows/activities.py: Activity 定义
- workflows/input.py: 输入数据结构
- worker.py: Temporal Worker 启动脚本
- docker-compose.temporal.yml: Temporal 服务配置

### 修改文件

- pyproject.toml: 添加 temporalio 依赖
- application/application_service.py: 移除 asyncio.Lock, 拆分 deploy() 为 Activity
- interfaces/api/application.py: 使用 Temporal Client 启动工作流
- interfaces/api/webhook.py: 使用 Temporal Client 启动工作流
- infrastructure/docker/manager.py: 适配 Activity 调用
- main.py: 添加 Temporal Client 生命周期管理

### 移除代码

- application/application_service.py: self._locks 字典和 get_lock() 方法
- interfaces/api/application.py: BackgroundTasks 依赖
- interfaces/api/webhook.py: BackgroundTasks 依赖

## 五、数据模型变更

### Deployment 实体扩展

```python
@dataclass
class Deployment:
    # 现有字段
    id: str
    application_id: str
    operation_type: str
    trigger_type: str
    status: str
    ...

    # 新增字段
    workflow_id: str | None = None  # Temporal 工作流 ID
    workflow_run_id: str | None = None  # Temporal 工作流运行 ID
```

### 新增数据库字段

```sql
ALTER TABLE deployment ADD COLUMN workflow_id VARCHAR(255);
ALTER TABLE deployment ADD COLUMN workflow_run_id VARCHAR(255);
```

## 六、配置变更

### config.defaults.yaml

```yaml
temporal:
  host: "localhost"
  port: 7233
  namespace: "default"
  task_queue: "deploy-queue"
```

### docker-compose.yml

```yaml
services:
  temporal:
    image: temporalio/auto-setup:1.25
    ports:
      - "7233:7233"
    environment:
      - DB=sqlite
      - SQLITE_FILE=/etc/temporal/temporal.db
    volumes:
      - temporal-data:/etc/temporal

  temporal-ui:
    image: temporalio/ui:2.21
    ports:
      - "8088:8088"
    depends_on:
      - temporal

volumes:
  temporal-data:
```

## 七、监控和可观测性

### Temporal Web UI

- 查看所有工作流执行历史
- 查看每个 Activity 的输入输出
- 查看重试记录和错误详情
- 手动取消或重试工作流

### 指标集成

```python
# Prometheus 指标
from prometheus_client import Counter, Histogram

deploy_counter = Counter(
    'deploy_total',
    'Total deployments',
    ['status', 'trigger_type']
)

deploy_duration = Histogram(
    'deploy_duration_seconds',
    'Deploy duration in seconds',
    ['app_id']
)
```

## 八、风险和缓解

| 风险 | 缓解措施 |
|------|---------|
| Temporal Server 额外运维 | 使用 Temporal Cloud 或单机部署 |
| 学习曲线 | 从简单工作流开始, 逐步迁移 |
| 数据库迁移 | 使用 Alembic 管理 schema 变更 |
| 降级方案 | 保留 BackgroundTasks 作为降级 |

## 九、预期收益

1. 可靠性: 部署不会因服务器重启而丢失
2. 可观测性: 完整的部署执行轨迹和审计日志
3. 重试能力: 瞬时故障自动恢复
4. 超时保护: 长时间部署不会无限挂起
5. 分布式支持: 多实例部署的并发控制
6. 可扩展性: 为后续 CI Pipeline 能力奠定基础
