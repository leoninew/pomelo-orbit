# CI Phase 1 任务清单

## 1. 领域层实现

### 1.1 实体定义
- [ ] 创建 `backend/src/pomelo_orbit/domain/ci/entities.py`
  - [ ] Project 实体
  - [ ] Credential 实体
  - [ ] PipelineTemplate 实体
  - [ ] PipelineRun 实体
  - [ ] Job 实体
  - [ ] JobLog 实体

### 1.2 值对象定义
- [ ] 创建 `backend/src/pomelo_orbit/domain/ci/value_objects.py`
  - [ ] CredentialType 枚举
  - [ ] PipelineRunStatus 枚举
  - [ ] PipelineRunTrigger 枚举
  - [ ] JobStatus 枚举
  - [ ] VariableDeclaration 值对象
  - [ ] StepDefinition 值对象
  - [ ] PipelineDefinition 值对象

### 1.3 执行器接口
- [ ] 创建 `backend/src/pomelo_orbit/domain/ci/executor.py`
  - [ ] PipelineExecutor 抽象接口
  - [ ] ExecutionContext 值对象

## 2. 基础设施层实现

### 2.1 数据库迁移
- [ ] 创建迁移脚本 `alembic/versions/xxx_add_ci_tables.py`
  - [ ] projects 表
  - [ ] credentials 表
  - [ ] pipeline_templates 表
  - [ ] pipeline_runs 表
  - [ ] jobs 表
  - [ ] job_logs 表

### 2.2 Repository 实现
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/repositories.py`
  - [ ] ProjectRepository
  - [ ] CredentialRepository
  - [ ] PipelineTemplateRepository
  - [ ] PipelineRunRepository
  - [ ] JobRepository
  - [ ] JobLogRepository

### 2.3 容器执行器
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/container.py`
  - [ ] ContainerExecutor 类
  - [ ] docker run 封装
  - [ ] 日志捕获
  - [ ] 退出码获取

### 2.4 Checkout Action
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/actions/checkout.py`
  - [ ] CheckoutAction 类
  - [ ] 参数解析（depth, ref, sparse_checkout, submodules）
  - [ ] 凭据注入（SSH 私钥、HTTPS token）
  - [ ] git clone 执行
  - [ ] outputs 提取（commit_sha, commit_message, author, committed_at）

### 2.5 Pipeline 执行器实现
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/executor_impl.py`
  - [ ] AsyncioPipelineExecutor 类
  - [ ] 串行执行 steps
  - [ ] 递归处理嵌套 steps
  - [ ] Job 状态管理
  - [ ] 日志收集
  - [ ] 错误处理

## 3. 应用层实现

### 3.1 Project Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/project_service.py`
  - [ ] create_project()
  - [ ] list_projects()
  - [ ] get_project()
  - [ ] update_project()
  - [ ] delete_project()（检查无运行中的 run）

### 3.2 Credential Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/credential_service.py`
  - [ ] create_credential()（加密存储）
  - [ ] list_credentials()（不回显敏感数据）
  - [ ] delete_credential()（检查无项目引用）
  - [ ] decrypt_credential()（内部使用）

### 3.3 Template Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/template_service.py`
  - [ ] create_template()
  - [ ] list_templates()
  - [ ] get_template()
  - [ ] update_template()
  - [ ] delete_template()（检查无项目引用）

### 3.4 Pipeline Service
- [ ] 创建 `backend/src/pomelo_orbit/application/ci/pipeline_service.py`
  - [ ] trigger_pipeline()（手动触发）
    - [ ] 合并变量（全局 ← 项目 ← 运行时）
    - [ ] 校验 required 变量
    - [ ] 渲染 pipeline YAML（Jinja2）
    - [ ] 保存快照（脱敏 secret）
    - [ ] 创建 PipelineRun
    - [ ] 提交后台任务
  - [ ] execute_pipeline()（后台任务）
    - [ ] 创建 workspace 目录
    - [ ] 解析 PipelineDefinition
    - [ ] 调用 PipelineExecutor
    - [ ] 更新状态
    - [ ] 清理 workspace
  - [ ] list_runs()
  - [ ] get_run()
  - [ ] get_run_jobs()
  - [ ] get_job_logs()

## 4. API 层实现

### 4.1 Project API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/projects.py`
  - [ ] POST /api/v1/projects
  - [ ] GET /api/v1/projects
  - [ ] GET /api/v1/projects/{id}
  - [ ] PUT /api/v1/projects/{id}
  - [ ] DELETE /api/v1/projects/{id}

### 4.2 Credential API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/credentials.py`
  - [ ] POST /api/v1/credentials
  - [ ] GET /api/v1/credentials
  - [ ] DELETE /api/v1/credentials/{id}

### 4.3 Template API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/templates.py`
  - [ ] POST /api/v1/pipeline-templates
  - [ ] GET /api/v1/pipeline-templates
  - [ ] GET /api/v1/pipeline-templates/{id}
  - [ ] PUT /api/v1/pipeline-templates/{id}
  - [ ] DELETE /api/v1/pipeline-templates/{id}

### 4.4 Run API
- [ ] 创建 `backend/src/pomelo_orbit/api/v1/ci/runs.py`
  - [ ] POST /api/v1/projects/{id}/runs
  - [ ] GET /api/v1/projects/{id}/runs
  - [ ] GET /api/v1/runs/{id}
  - [ ] GET /api/v1/runs/{id}/jobs
  - [ ] GET /api/v1/jobs/{id}/logs

## 5. 工具和辅助

### 5.1 YAML 解析
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/parser.py`
  - [ ] parse_pipeline_yaml()
  - [ ] 校验 YAML 结构
  - [ ] 转换为 PipelineDefinition

### 5.2 变量处理
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/variables.py`
  - [ ] merge_variables()（三层合并）
  - [ ] validate_variables()（检查 required）
  - [ ] render_template()（Jinja2）
  - [ ] mask_secrets()（脱敏）

### 5.3 工作目录管理
- [ ] 创建 `backend/src/pomelo_orbit/infrastructure/ci/workspace.py`
  - [ ] create_workspace()
  - [ ] cleanup_workspace()
  - [ ] get_workspace_path()
  - [ ] get_artifacts_path()

## 6. 测试

### 6.1 单元测试
- [ ] domain/ci/entities_test.py
- [ ] domain/ci/value_objects_test.py
- [ ] infrastructure/ci/container_test.py
- [ ] infrastructure/ci/actions/checkout_test.py
- [ ] infrastructure/ci/executor_impl_test.py
- [ ] infrastructure/ci/parser_test.py
- [ ] infrastructure/ci/variables_test.py
- [ ] application/ci/project_service_test.py
- [ ] application/ci/credential_service_test.py
- [ ] application/ci/template_service_test.py
- [ ] application/ci/pipeline_service_test.py

### 6.2 集成测试
- [ ] 端到端测试：创建项目 → 触发 pipeline → 查看日志
- [ ] checkout action 集成测试（真实 git 仓库）
- [ ] 容器执行集成测试（真实 Docker）

## 7. 文档

- [ ] API 文档（OpenAPI/Swagger）
- [ ] 用户指南：如何创建项目和触发 pipeline
- [ ] 开发者指南：如何扩展 action

## 8. 部署

- [ ] 更新 docker-compose.yml（确保 Docker socket 挂载）
- [ ] 更新环境变量配置（全局变量）
- [ ] 数据库迁移脚本执行
- [ ] 创建内置 PipelineTemplate（示例模板）

## 实施顺序建议

1. 领域层（1.1 → 1.2 → 1.3）
2. 基础设施层数据库（2.1 → 2.2）
3. 基础设施层执行（2.3 → 2.4 → 2.5）
4. 工具层（5.1 → 5.2 → 5.3）
5. 应用层（3.1 → 3.2 → 3.3 → 3.4）
6. API 层（4.1 → 4.2 → 4.3 → 4.4）
7. 测试（6.1 → 6.2）
8. 文档和部署（7 → 8）
