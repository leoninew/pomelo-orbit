# CI Phase 2 任务清单

## 1. 领域层扩展

### 1.1 新增实体
- [ ] 扩展 `backend/src/pomelo_orbit/domain/ci/entities.py`
  - [ ] Artifact 实体
  - [ ] Project 增加 webhook_secret 和 branch_filter 字段

### 1.2 新增值对象
- [ ] 扩展 `backend/src/pomelo_orbit/domain/ci/value_objects.py`
  - [ ] ArtifactType 枚举
  - [ ] StepDefinition 增加 depends_on 和 artifacts 字段
  - [ ] RetryPolicy 枚举（always_rerun, skip_if_success）

## 2. 基础设施层扩展

### 2.1 数据库迁移
- [ ] 创建迁移脚本 `backend/migrations/v0.6.1__ci_phase2.sql`
  - [ ] artifacts 表
  - [ ] projects 表增加 webhook_secret, branch_filter
  - [ ] pipeline_runs 表增加 retry_of

### 2.2 Repository 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/infrastructure/ci/repositories.py`
  - [ ] ArtifactRepository
  - [ ] ProjectRepository 增加 webhook 相关查询
  - [ ] PipelineRunRepository 增加 retry 相关查询

### 2.3 ORM 模型扩展
- [ ] 扩展 `backend/src/pomelo_orbit/infrastructure/ci/models.py`
  - [ ] ArtifactModel
  - [ ] ProjectModel 增加字段
  - [ ] PipelineRunModel 增加字段

### 2.4 Mapper 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/infrastructure/ci/mappers.py`
  - [ ] ArtifactMapper

### 2.5 依赖图实现
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/dependency_graph.py`
  - [ ] DependencyGraph 类
  - [ ] build() 方法
  - [ ] get_ready_jobs() 方法
  - [ ] mark_completed() 方法
  - [ ] topological_sort() 方法

### 2.6 Pipeline 执行器扩展
- [ ] 扩展 `backend/src/pomelo_orbit/infrastructure/ci/executor_impl.py`
  - [ ] 并行执行逻辑（asyncio.gather）
  - [ ] 依赖图构建和调度
  - [ ] Fail-fast 实现
  - [ ] 制品记录逻辑
  - [ ] 重试跳过逻辑

### 2.7 Webhook 签名验证
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/webhook_verifier.py`
  - [ ] verify_github_signature()
  - [ ] verify_gitlab_signature()

## 3. 应用层扩展

### 3.1 Webhook Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/webhook_service.py`
  - [ ] handle_webhook()
    - [ ] 验证签名
    - [ ] 解析 payload
    - [ ] 匹配 Project
    - [ ] 触发 PipelineRun
    - [ ] 记录 WebhookEvent
  - [ ] get_webhook_url()
  - [ ] list_webhook_events()

### 3.2 Pipeline Service 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/application/ci/pipeline_service.py`
  - [ ] retry_pipeline()
    - [ ] 创建新 PipelineRun（携带 retry_of）
    - [ ] 复用原 pipeline 和变量
    - [ ] 提交后台任务
  - [ ] execute_pipeline() 扩展
    - [ ] 重试时判断是否跳过 Job
    - [ ] 记录制品

### 3.3 Artifact Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/artifact_service.py`
  - [ ] list_artifacts()
  - [ ] get_artifact()
  - [ ] download_artifact()（文件类型）

### 3.4 Project Service 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/application/ci/project_service.py`
  - [ ] create_project() 生成 webhook_secret
  - [ ] get_webhook_config()

## 4. API 层扩展

### 4.1 Webhook API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/webhooks.py`
  - [ ] POST /api/v1/webhooks/git
  - [ ] GET /api/v1/projects/{id}/webhook-events

### 4.2 Artifact API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/artifacts.py`
  - [ ] GET /api/v1/runs/{id}/artifacts
  - [ ] GET /api/v1/artifacts/{id}/download

### 4.3 Run API 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/api/v1/ci/runs.py`
  - [ ] POST /api/v1/runs/{id}/retry

### 4.4 Project API 扩展
- [ ] 扩展 `backend/src/pomelo_orbit/api/v1/ci/projects.py`
  - [ ] GET /api/v1/projects/{id}/webhook-url

## 5. 测试

### 5.1 单元测试
- [ ] tests/domain/ci/test_entities.py 扩展
  - [ ] Artifact 实体测试
- [ ] tests/infrastructure/ci/test_dependency_graph.py
  - [ ] 依赖图构建测试
  - [ ] 拓扑排序测试
  - [ ] 循环依赖检测测试
- [ ] tests/infrastructure/ci/test_webhook_verifier.py
  - [ ] GitHub 签名验证测试
  - [ ] GitLab 签名验证测试
- [ ] tests/infrastructure/ci/test_executor_impl.py 扩展
  - [ ] 并行执行测试
  - [ ] Fail-fast 测试
  - [ ] 重试跳过测试
- [ ] tests/application/ci/test_webhook_service.py
  - [ ] Webhook 处理测试
  - [ ] Project 匹配测试
- [ ] tests/application/ci/test_pipeline_service.py 扩展
  - [ ] 重试逻辑测试
  - [ ] 制品记录测试

### 5.2 集成测试
- [ ] tests/integration/ci/test_webhook_integration.py
  - [ ] 端到端 webhook 触发测试
  - [ ] 签名验证集成测试
- [ ] tests/integration/ci/test_parallel_execution.py
  - [ ] 并行执行集成测试
  - [ ] 依赖关系正确性测试
- [ ] tests/integration/ci/test_retry.py
  - [ ] 重试流程集成测试
  - [ ] 跳过逻辑集成测试
- [ ] tests/integration/ci/test_artifacts.py
  - [ ] 制品记录集成测试
  - [ ] 制品下载集成测试

## 6. 文档

- [ ] API 文档更新（OpenAPI/Swagger）
  - [ ] Webhook 端点
  - [ ] Artifact 端点
  - [ ] Retry 端点
- [ ] 用户指南更新
  - [ ] 如何配置 webhook
  - [ ] 如何使用并行执行
  - [ ] 如何重试失败的 pipeline
  - [ ] 如何查看和下载制品
- [ ] Pipeline YAML 语法文档
  - [ ] depends_on 用法
  - [ ] artifacts 声明
  - [ ] retry_policy 配置

## 7. 部署

- [ ] 数据库迁移脚本执行
- [ ] 更新示例 PipelineTemplate（包含并行和制品）
- [ ] 配置 webhook 接收端点（确保可从外网访问）

## 实施顺序建议

### 第一阶段：并行执行（1-2 天）
1. 依赖图实现（2.5）
2. 执行器扩展（2.6 并行部分）
3. 单元测试（5.1 依赖图）
4. 集成测试（5.2 并行执行）

### 第二阶段：Webhook 触发（1-2 天）
1. 数据库迁移（2.1 webhook 部分）
2. Webhook 签名验证（2.7）
3. Webhook Service（3.1）
4. Webhook API（4.1）
5. 单元测试（5.1 webhook）
6. 集成测试（5.2 webhook）

### 第三阶段：制品记录（1 天）
1. 数据库迁移（2.1 artifacts 表）
2. Artifact 实体和 Repository（1.1, 2.2, 2.3, 2.4）
3. 执行器扩展（2.6 制品部分）
4. Artifact Service（3.3）
5. Artifact API（4.2）
6. 单元测试（5.1 制品）
7. 集成测试（5.2 制品）

### 第四阶段：重试机制（1-2 天）
1. 数据库迁移（2.1 retry_of）
2. 执行器扩展（2.6 重试部分）
3. Pipeline Service 扩展（3.2 retry）
4. Run API 扩展（4.3）
5. 单元测试（5.1 重试）
6. 集成测试（5.2 重试）

### 第五阶段：文档和部署（1 天）
1. API 文档更新（6）
2. 用户指南更新（6）
3. 部署准备（7）

总计：5-8 天
