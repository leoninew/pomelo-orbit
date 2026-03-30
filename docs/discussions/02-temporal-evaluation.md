# [探索性讨论] Temporal 与 CI/CD 方案对比

> 本文档为探索性讨论记录, 记录的是关于可能方案的思考和交流, 并非最终确定的实施计划或规划。内容可能包含未经验证的假设和理想化的设计, 实际实施时需要进一步评估和调整。

## 一、工作流引擎对比

### Temporal vs 其他引擎

| 特性 | Temporal | Cadence | Argo Workflows | Tekton |
|------|----------|---------|----------------|--------|
| 定位 | 通用工作流引擎 | 通用工作流引擎 | K8s 原生工作流 | CI/CD 专用框架 |
| 持久化执行 | 原生支持 | 原生支持 | 依赖 K8s | 依赖 K8s |
| 自动重试 | 细粒度配置 | 细粒度配置 | 基础支持 | 基础支持 |
| 长时间运行 | 无超时限制 | 无超时限制 | 有超时 | 有超时 |
| 可观测性 | 完整 Web UI | Web UI | Argo UI | Tekton Dashboard |
| Python SDK | 官方支持 | 社区支持 | 通过 API | 通过 API |
| K8s 依赖 | 无 | 无 | 必需 | 必需 |
| 学习曲线 | 中等 | 中等 | 较陡 | 较陡 |
| 社区活跃度 | 高 | 中 | 高 | 高 |

### 选择 Temporal 的理由

1. Python 优先: 官方 Python SDK 成熟, 开发体验好
2. 无 K8s 依赖: 可在 Docker/Compose 环境运行
3. 持久化执行: 关键特性, 确保长时间构建任务可靠性
4. 灵活编排: 支持复杂的依赖关系和并行执行
5. 完整可观测性: Web UI 提供完整的执行轨迹

---

## 二、容器执行方案对比

| 方案 | 实现方式 | 优点 | 缺点 |
|------|---------|------|------|
| Docker SDK | 直接调用 Docker API | 简单直接、Python 原生 | 需要挂载 socket |
| Docker CLI | subprocess 调用 docker 命令 | 无需额外依赖 | 性能差、解析输出复杂 |
| Docker Compose | 通过 compose 管理多容器 | 声明式配置 | 编排能力有限 |
| Containerd | 直接调用 containerd API | 更轻量、更安全 | Python SDK 不成熟 |

### 推荐: Docker SDK for Python

```python
import docker
client = docker.from_env()
container = client.containers.run("alpine", "echo hello", detach=True)
```

优势:

- 官方支持, API 稳定
- 异步支持, 性能好
- 完整的功能覆盖
- 良好的错误处理

---

## 三、存储方案对比

| 方案 | 类型 | 适用场景 | 复杂度 |
|------|------|---------|--------|
| Docker Volume | 本地存储 | 缓存、临时制品 | 低 |
| MinIO 容器 | 对象存储 | 制品管理、长期存储 | 中 |
| 本地文件系统 | 文件存储 | 日志、配置 | 低 |
| SQLite | 数据库 | 元数据、状态 | 低 |

### 推荐组合

- 缓存: Docker Volume (快速、简单)
- 制品: MinIO 容器 (可扩展、S3 兼容)
- 日志: 本地文件 + Docker Volume
- 元数据: SQLite (Temporal 自带)

---

## 四、日志方案对比

| 方案 | 实现方式 | 实时性 | 持久化 | 复杂度 |
|------|---------|--------|--------|--------|
| Docker logs API | 流式读取容器日志 | 实时 | 容器删除后丢失 | 低 |
| 文件存储 | 写入日志文件 | 延迟 | 持久化 | 低 |
| Loki | 日志聚合系统 | 实时 | 持久化 | 高 |
| ELK Stack | Elasticsearch + Logstash + Kibana | 实时 | 持久化 | 高 |

### 推荐: Docker logs API + 文件存储

```python
for line in container.logs(stream=True):
    # 实时推送到 WebSocket
    await broadcast_log(line)

    # 写入日志文件
    await write_log_file(build_id, line)
```

---

## 五、Secret 管理方案对比

| 方案 | 安全性 | 易用性 | 适用场景 |
|------|--------|--------|---------|
| 环境变量 | 低 | 简单 | 开发环境 |
| Docker Secrets | 中 | 简单 | 单机部署 |
| HashiCorp Vault | 高 | 复杂 | 生产环境 |
| K8s Secrets | 中 | 简单 | K8s 环境 |

### 推荐: Docker Secrets + 环境变量

```yaml
services:
  worker:
    secrets:
      - db_password
      - api_key

secrets:
  db_password:
    file: ./secrets/db_password.txt
  api_key:
    file: ./secrets/api_key.txt
```

---

## 六、通知集成方案对比

| 方案 | 集成方式 | 实时性 | 功能丰富度 |
|------|---------|--------|-----------|
| Webhook | HTTP POST | 实时 | 中 |
| Slack API | Slack App | 实时 | 高 |
| 邮件 (SMTP) | SMTP 协议 | 延迟 | 中 |
| 企业微信 | Webhook | 实时 | 中 |

### 推荐: Webhook + Slack

```python
@activity.defn
async def send_notification(event: dict) -> None:
    await httpx.post(event["webhook_url"], json=event)

    if event.get("slack_channel"):
        await slack_client.chat_postMessage(
            channel=event["slack_channel"],
            text=format_message(event)
        )
```

---

## 七、性能优化策略

### 并发控制

```python
from temporalio.worker import Worker

worker = Worker(
    client,
    task_queue="ci-queue",
    max_concurrent_activities=10,
    max_concurrent_workflow_tasks=5
)
```

### 资源限制

```python
container = client.containers.run(
    image="alpine",
    command="sleep 1000",
    nano_cpus=1_000_000_000,
    mem_limit="512m",
    detach=True
)
```

### 缓存策略

多级缓存:

- L1: 内存缓存 (热数据)
- L2: Docker Volume (温数据)
- L3: MinIO (冷数据)

---

## 八、安全考虑

### Docker Socket 安全

```yaml
services:
  worker:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

### 网络隔离

```yaml
networks:
  ci-network:
    driver: bridge
    internal: true
```

### 镜像安全

```python
def verify_image(image: str) -> bool:
    # 检查镜像签名
    # 检查镜像来源
    # 检查镜像漏洞
    return True
```

---

## 九、监控和告警

### 指标收集

```python
from prometheus_client import Counter, Histogram

build_counter = Counter(
    'ci_builds_total',
    'Total number of builds',
    ['status', 'project']
)

build_duration = Histogram(
    'ci_build_duration_seconds',
    'Build duration in seconds',
    ['project']
)
```

### 告警规则

```yaml
- alert: BuildFailureRateHigh
  expr: rate(ci_builds_total{status="failed"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "构建失败率过高"
```

---

## 十、扩展性考虑

### 多节点支持 (未来)

```
┌─────────────────┐
│  Load Balancer  │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───┴───┐ ┌───┴───┐
│Worker1│ │Worker2│
└───────┘ └───────┘
```

### 插件系统

```python
class CIPlugin:
    def before_step(self, step: StepConfig) -> None:
        pass

    def after_step(self, step: StepConfig, result: dict) -> None:
        pass

    def on_error(self, step: StepConfig, error: Exception) -> None:
        pass
```

---

## 总结

在 Python + Docker/Compose 环境下, Temporal + Docker SDK 是最佳选择:

1. Temporal 提供: 强大的工作流编排能力, 无需 K8s 依赖
2. Docker SDK 提供: 简单直接的容器执行能力
3. Python 生态: 丰富的库支持, 快速开发

该方案平衡了功能性和复杂度, 适合中小型团队快速构建 CI 系统。
