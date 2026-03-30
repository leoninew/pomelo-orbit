# [探索性讨论] Pomelo Orbit 能力评估与补齐方向

> 本文档为探索性讨论记录, 记录的是关于可能方案的思考和交流, 并非最终确定的实施计划或规划。内容可能包含未经验证的假设和理想化的设计, 实际实施时需要进一步评估和调整。

## 一、现有能力总结

Pomelo Orbit 目前是一个部署管理系统, 而非完整的 CI 系统。它提供了应用生命周期管理、Docker Compose 部署、GitHub Webhook 集成和完整的 Web UI。

## 二、能力评估矩阵

| 能力 | 状态 | 完成度 | 说明 |
|------|------|--------|------|
| 事件驱动执行 | 部分实现 | 60% | 已有 GitHub webhook 接收和签名验证, 但只支持单阶段部署 |
| YAML Pipeline 配置 | 未实现 | 0% | 设计文档已规划 DSL, 但代码未实现 |
| 容器化 Step 执行 | 部分实现 | 40% | 已有 Docker Compose 部署, 但缺少 step 级别的隔离执行 |
| 工作流编排 (Temporal) | 未实现 | 0% | 设计文档已规划, 但未引入 Temporal SDK |
| 缓存机制 | 未实现 | 0% | 无缓存 key 生成、Volume 管理逻辑 |
| Secret 管理 | 部分实现 | 50% | 已有 JWT 和 Fernet 加密, 但缺少 Docker Secrets 集成 |
| 日志收集 | 部分实现 | 70% | 已有部署日志收集和增量读取, 但缺少实时流式传输 |
| 制品管理 | 未实现 | 0% | 无制品存储、上传下载 API |
| Web UI | 完全实现 | 100% | Vue 3 + Ant Design Vue, 功能完整 |
| 通知集成 | 未实现 | 0% | 无 Slack/邮件/Webhook 通知 |

## 三、已实现能力详情

### 事件驱动执行 (60%)

已实现:

- GitHub webhook 接收 (interfaces/api/webhook.py:35-170)
- Webhook 签名验证 (infrastructure/security.py:95-118)
- Push/release/ping 事件解析 (application/webhook_parser.py)
- Webhook 事件数据库记录 (infrastructure/persistence/models.py:138-167)
- 基于分支的过滤 (interfaces/api/webhook.py:121-126)
- 自动部署触发 (interfaces/api/webhook.py:128-163)

未实现:

- 多阶段 pipeline 执行
- 并行 step 执行
- 基于依赖的 stage 排序
- GitLab/Bitbucket 支持

### 容器化 Step 执行 (40%)

已实现:

- Docker Compose 部署 (infrastructure/docker/manager.py)
- 容器生命周期管理 (部署、停止、重启)
- 通过 subprocess 执行 Docker 命令 (infrastructure/docker/manager.py:149-186)
- 镜像拉取策略 (domain/value_objects.py:25-30)
- 环境变量文件支持
- Jinja2 模板渲染

未实现:

- 独立 step 容器执行
- Workspace Volume 挂载
- Step 级别资源限制 (CPU、内存)
- Step 级别超时配置
- Step 依赖解析

### Secret 管理 (50%)

已实现:

- JWT secret key 管理 (infrastructure/security.py:39-50)
- Fernet 加密敏感值 (infrastructure/security.py:83-93)
- Webhook secret 验证 (infrastructure/security.py:95-118)
- 环境变量配置 (config.defaults.yaml:28-35)
- UI 中的 secret 脱敏显示

未实现:

- Docker Secrets 集成
- 每个 pipeline 的 secret 管理
- Secret 轮换
- HashiCorp Vault 集成
- Secret 版本控制

### 日志收集 (70%)

已实现:

- 部署日志收集到文件 (infrastructure/docker/manager.py:288-293)
- 增量日志读取 (infrastructure/docker/manager.py:379-393)
- 日志 API 端点 (interfaces/api/deployment.py:63-79)
- Docker compose 日志获取 (infrastructure/docker/manager.py:233-237)
- 请求日志中间件 (infrastructure/logging.py)

未实现:

- 实时日志流 (WebSocket)
- 跨 step 日志聚合
- 日志搜索功能
- 日志保留策略 (除 config.defaults.yaml:38-42 的清理配置)

### Web UI (100%)

已实现:

- Vue 3 + Ant Design Vue 前端
- 仪表盘统计 (views/Home.vue)
- 应用管理 (CRUD、部署、停止、重启)
- 部署历史和详情
- Webhook 事件查看器
- 路由配置管理
- Traefik HTTP routers 查看
- 登录历史
- 系统设置
- Monaco Editor 文件编辑器
- 实时部署状态更新
- JWT 认证

---

## 四、需要补齐的能力

### 1. YAML Pipeline 配置 (优先级: 高)

需要实现:

- Pipeline DSL 解析器
- Pydantic schema 验证 (PipelineConfig、StageConfig、StepConfig)
- 配置文件版本控制
- 模板变量替换

涉及文件:

- 新增: application/pipeline_parser.py
- 新增: domain/pipeline.py
- 修改: interfaces/api/webhook.py (集成 pipeline 执行)

### 2. 工作流编排 - Temporal (优先级: 高)

需要实现:

- 引入 temporalio SDK
- Temporal Worker 实现
- Pipeline 工作流定义
- Step Activity 定义
- 并行执行和依赖管理

涉及文件:

- 修改: pyproject.toml (添加依赖)
- 新增: workflows/pipeline.py
- 新增: workflows/activities.py
- 新增: worker.py
- 修改: docker-compose.yml (添加 Temporal 服务)

### 3. Step 级别容器执行 (优先级: 高)

需要实现:

- Docker SDK 集成 (替代 subprocess)
- 独立容器创建和执行
- Workspace Volume 管理
- 资源限制 (CPU、内存)
- 超时控制

涉及文件:

- 新增: infrastructure/docker/step_executor.py
- 修改: infrastructure/docker/manager.py

### 4. 缓存机制 (优先级: 中)

需要实现:

- 缓存 key 生成逻辑
- Docker Volume 缓存管理
- 缓存命中/未命中处理
- 缓存清理策略

涉及文件:

- 新增: infrastructure/cache/manager.py
- 新增: infrastructure/cache/key_generator.py

### 5. 制品管理 (优先级: 中)

需要实现:

- 本地/Docker Volume 制品存储
- MinIO 集成 (可选)
- 制品上传下载 API
- 制品元数据跟踪

涉及文件:

- 新增: infrastructure/artifact/store.py
- 新增: interfaces/api/artifact.py

### 6. 通知集成 (优先级: 中)

需要实现:

- Webhook 通知
- Slack 集成
- 邮件通知
- 通知配置管理

涉及文件:

- 新增: infrastructure/notification/
- 新增: interfaces/api/notification.py

### 7. Secret 管理增强 (优先级: 低)

需要实现:

- Docker Secrets 集成
- 每个 pipeline 的 secret 配置
- Secret 版本控制

涉及文件:

- 修改: infrastructure/security.py
- 新增: infrastructure/secret/

### 8. 日志系统增强 (优先级: 低)

需要实现:

- WebSocket 实时日志流
- 跨 step 日志聚合
- 日志搜索
- 日志保留策略

涉及文件:

- 新增: interfaces/websocket/log_stream.py
- 修改: infrastructure/docker/manager.py

---

## 五、实施建议

### Phase 1: 核心 CI 能力 (2-3 周)

1. YAML Pipeline 配置解析器
2. Temporal 工作流集成
3. Step 级别容器执行
4. 基础缓存机制

### Phase 2: 增强能力 (1-2 周)

1. 制品管理
2. 通知集成
3. Secret 管理增强

### Phase 3: 完善能力 (1 周)

1. 日志系统增强 (WebSocket 实时流)
2. 性能优化
3. 监控告警

---

## 六、技术栈补充

当前依赖 (pyproject.toml):

- fastapi
- uvicorn
- sqlalchemy
- docker
- pyyaml
- jinja2
- httpx
- pydantic

需要新增:

- temporalio (Temporal Python SDK)
- aiofiles (异步文件操作)
- websockets (实时日志流)
- minio (可选, 制品存储)
