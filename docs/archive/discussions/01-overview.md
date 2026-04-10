# [探索性讨论] 轻量级云原生 CI 系统设计

> 本文档为探索性讨论记录, 记录的是关于可能方案的思考和交流, 并非最终确定的实施计划或规划。内容可能包含未经验证的假设和理想化的设计, 实际实施时需要进一步评估和调整。

环境约束: Python + Docker + Docker Compose (不支持 K8s)

## 一、核心能力需求

### 基础 CI 能力

- 事件驱动: Git webhook 触发 pipeline 执行
- 声明式配置: YAML 定义 pipeline (类似 GitHub Actions)
- 容器化执行: 每个 step 在独立容器中运行

### 核心功能

| 功能 | 说明 |
|------|------|
| Pipeline 定义 | stages、steps、依赖关系、并行执行 |
| 缓存机制 | 依赖缓存 (npm/pip/maven)、Docker layer 缓存 |
| Secret 管理 | Docker Secrets 或环境变量 |
| 通知集成 | Slack/邮件/Webhook |
| 日志收集 | 实时日志流 + 持久化存储 |
| 制品管理 | 本地卷或 MinIO 容器 |

### 云原生特性

- 可观测性: 完整的执行轨迹和审计日志
- 高可用: 持久化执行, 故障自动恢复
- GitOps: 配置即代码, pipeline 定义存放在仓库

---

## 二、技术选型

### 核心组件

| 组件 | 选型 | 说明 |
|------|------|------|
| 工作流引擎 | Temporal | 持久化执行、自动重试、完整可观测性 |
| 容器运行时 | Docker SDK for Python | 直接调用 Docker API |
| API 服务 | FastAPI | 轻量级, 异步支持 |
| 配置 DSL | YAML | 声明式 pipeline 定义 |
| 存储 | Docker Volume | 本地卷管理缓存和制品 |
| 日志 | Docker logs API + 文件 | 实时日志流 |

### Python 依赖

```txt
temporalio>=1.7.0        # Temporal Python SDK
docker>=7.0.0            # Docker SDK for Python
pyyaml>=6.0              # YAML 解析
fastapi>=0.109.0         # API 服务
uvicorn>=0.27.0          # ASGI 服务器
aiofiles>=23.2.1         # 异步文件操作
```

---

## 三、架构设计

### 整体架构

```
┌─────────────────────────────────────────────────┐
│            Temporal Server (Docker)              │
│  工作流编排、状态管理、重试机制                    │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│         Python Worker (Docker)                   │
│  解析 .ci.yaml → 编排 step → 调用 Docker API     │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│              Docker Engine                       │
│  每个 step 作为独立容器运行                        │
└─────────────────────────────────────────────────┘
                       │
┌─────────────────────────────────────────────────┐
│            存储层 (制品/缓存/日志)                │
│  Docker Volume (制品) + Volume (缓存) + 文件 (日志)│
└─────────────────────────────────────────────────┘
```

### 关键设计决策

1. 执行模式: Temporal 工作流 → Docker API → 容器执行
2. 状态管理: Temporal 持久化 + 本地缓存 (减少查询)
3. 并行执行: 利用 Temporal 的 child workflow 实现 step 并行
4. 失败处理: Temporal 重试 + 用户自定义重试策略
5. 资源隔离: 每个 step 独立容器, 避免环境污染

---

## 四、配置 DSL 设计

### 示例 .ci.yaml

```yaml
version: v1
stages:
  - name: build
    steps:
      - name: compile
        image: golang:1.21
        commands:
          - go build -o app
        cache:
          - path: /go/pkg/mod
            key: go-mod-${{ hashFiles('go.sum') }}

  - name: test
    depends_on: [build]
    steps:
      - name: unit-test
        image: golang:1.21
        commands:
          - go test ./...

  - name: deploy
    depends_on: [test]
    steps:
      - name: push-image
        image: docker:latest
        commands:
          - docker build -t app:${{ github.sha }} .
          - docker push app:${{ github.sha }}
```

### 配置 Schema

```python
from pydantic import BaseModel
from typing import List, Optional, Dict

class CacheConfig(BaseModel):
    path: str
    key: str

class StepConfig(BaseModel):
    name: str
    image: str
    commands: List[str]
    env: Optional[Dict[str, str]] = None
    cache: Optional[List[CacheConfig]] = None
    timeout: Optional[int] = 3600  # 默认 1 小时

class StageConfig(BaseModel):
    name: str
    steps: List[StepConfig]
    depends_on: Optional[List[str]] = None

class PipelineConfig(BaseModel):
    version: str
    stages: List[StageConfig]
```

---

## 五、Docker Compose 配置

```yaml
services:
  temporal:
    image: temporalio/auto-setup:1.25
    ports:
      - "7233:7233"
      - "8233:8233"
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

  worker:
    build:
      context: ./worker
      dockerfile: Dockerfile
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal:7233
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - workspace:/workspace
      - cache:/cache

  api:
    build:
      context: ./api
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    depends_on:
      - temporal
    environment:
      - TEMPORAL_HOST=temporal:7233

volumes:
  temporal-data:
  workspace:
  cache:
```

---

## 六、核心代码结构

### 项目目录

```
ci-system/
├── docker-compose.yml
├── worker/
│   ├── Dockerfile
│   ├── requirements.txt
│   ├── worker.py
│   ├── workflows.py
│   └── activities.py
├── api/
│   ├── Dockerfile
│   ├── requirements.txt
│   └── main.py
└── examples/
    └── .ci.yaml
```

### 工作流定义

```python
from temporalio import workflow
from datetime import timedelta

@workflow.defn
class CIPipeline:
    @workflow.run
    async def run(self, config: dict) -> dict:
        results = {}
        for stage in config["stages"]:
            stage_result = await workflow.execute_activity(
                run_stage,
                stage,
                start_to_close_timeout=timedelta(hours=1),
                retry_policy=RetryPolicy(maximum_attempts=3)
            )
            results[stage["name"]] = stage_result

            if not self.check_dependencies(stage, results):
                raise Exception(f"Stage {stage['name']} dependencies failed")
        return results
```

### 容器执行 Activity

```python
from temporalio import activity
import docker

@activity.defn
async def run_step(step: dict) -> dict:
    client = docker.from_env()

    container = client.containers.run(
        image=step["image"],
        command=step.get("commands"),
        volumes={
            "/workspace": {"bind": "/workspace", "mode": "rw"}
        },
        environment=step.get("env", {}),
        detach=True
    )

    logs = []
    for line in container.logs(stream=True):
        logs.append(line.decode())

    exit_code = container.wait()["StatusCode"]
    container.remove()

    return {
        "exit_code": exit_code,
        "logs": "".join(logs)
    }
```

### 缓存机制

```python
@activity.defn
async def run_step_with_cache(step: dict) -> dict:
    cache_key = generate_cache_key(step)
    cache_volume = f"cache-{cache_key}"

    client = docker.from_env()
    try:
        client.volumes.get(cache_volume)
        cache_hit = True
    except docker.errors.NotFound:
        client.volumes.create(cache_volume)
        cache_hit = False

    volumes = {
        cache_volume: {"bind": step["cache"]["path"], "mode": "rw"}
    }

    result = await run_step(step, volumes=volumes)
    return {**result, "cache_hit": cache_hit}
```

### Worker 启动

```python
import asyncio
from temporalio.client import Client
from temporalio.worker import Worker

async def main():
    client = await Client.connect("temporal:7233")

    worker = Worker(
        client,
        task_queue="ci-queue",
        workflows=[CIPipeline],
        activities=[run_step, run_step_with_cache]
    )

    await worker.run()

if __name__ == "__main__":
    asyncio.run(main())
```

---

## 七、功能实现对比

| 功能 | 实现方式 | 复杂度 |
|------|---------|--------|
| 工作流编排 | Temporal + Python SDK | 中 |
| 容器执行 | Docker SDK for Python | 低 |
| 缓存机制 | Docker Volume | 低 |
| 日志收集 | Docker logs API + 文件存储 | 中 |
| Secret 管理 | Docker Secrets 或环境变量 | 低 |
| 制品存储 | 本地卷或 MinIO 容器 | 中 |
| Web UI | Temporal UI + 简单 FastAPI | 低 |

---

## 八、Temporal 的优势与限制

### 优势

1. 持久化执行: 工作流在 worker 崩溃、服务器重启时自动恢复
2. 自动重试: 在 activity 和 workflow 级别可配置重试策略
3. 长时间运行支持: 可等待数小时而不会超时
4. 完整可观测性: 通过 Temporal Web UI 查看完整执行轨迹
5. 版本控制: 工作流版本化, 支持安全升级
6. 语言灵活: Python SDK 成熟可用

### 限制

1. 无原生容器支持: 需要通过 Docker API 集成
2. 无原生制品管理: 需要集成外部存储
3. 无原生缓存: 需要实现自定义缓存逻辑
4. 无原生 Secret: 需要集成 Vault/KMS
5. 基础设施开销: 需要运行 Temporal Server
6. 学习曲线: 需要理解工作流确定性和重放机制

---

## 九、限制和注意事项

1. 资源限制: 单机 Docker 有资源上限, 不适合大规模构建
2. 网络隔离: 容器间通信需要自定义 Docker 网络
3. 存储管理: 缓存和制品占用本地磁盘, 需定期清理
4. 并发控制: 需要限制并行构建数量, 避免资源耗尽
5. 安全性: 挂载 Docker socket 有安全风险, 生产环境需额外隔离

---

## 十、MVP 实现路径

### Phase 1: 基础版 (单机)

- Temporal + Docker API 集成
- 基本的 step 串行执行
- 基础日志收集
- 简单 Web UI

### Phase 2: 增强版

- 支持并行执行
- 实现缓存机制
- 添加 Secret 管理
- 制品管理

### Phase 3: 完善版

- Web UI 增强
- 通知集成
- 性能优化
- 监控告警

---

## 十一、与现有方案对比

| 方案 | 优势 | 劣势 |
|------|------|------|
| Temporal + Docker | 强大的编排能力、持久化执行、Python 生态 | 需要运行 Temporal Server、单机限制 |
| Argo Workflows | 原生 K8s 支持、容器原生 | 仅限 K8s 环境 |
| Tekton | 专为 CI/CD 设计、K8s 原生 | 学习曲线陡峭、依赖 K8s |
| Dagger | 容器原生、本地开发友好 | 依赖 BuildKit、Go 生态 |
| GitHub Actions | 托管服务、生态丰富 | 供应商锁定、自定义受限 |

---

## 总结

在 Python + Docker/Compose 环境约束下, Temporal + Docker API 是构建轻量级云原生 CI 系统的最佳选择:

1. Temporal 提供: 工作流编排、持久化执行、自动重试、完整可观测性
2. Docker 提供: 容器执行、资源隔离、环境一致性
3. Python 提供: 快速开发、丰富生态、易于维护

该方案适合中小型团队的 CI/CD 需求, 可在单机环境快速部署, 后续可扩展到多机环境。
