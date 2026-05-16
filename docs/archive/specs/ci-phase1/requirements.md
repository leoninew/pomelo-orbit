# CI Phase 1 需求文档

## 目标

实现 CI 系统的核心可运行版本，支持手动触发 pipeline，执行串行 steps，查看日志和状态。

## 功能需求

### 1. 项目管理（Project）

- 创建项目：指定仓库地址、关联凭据、选择 pipeline 模板、配置项目级变量
- 查询项目列表：分页、搜索
- 查询项目详情：包含关联的凭据和模板信息
- 更新项目：修改变量、切换模板
- 删除项目：仅当无运行中的 PipelineRun 时允许

### 2. 凭据管理（Credential）

- 创建凭据：支持 git_ssh、github_token 两种类型，数据加密存储
- 查询凭据列表：仅显示名称和类型，不回显敏感数据
- 删除凭据：仅当无项目引用时允许
- 凭据注入：执行时解密并注入到容器环境，不落盘明文

### 3. Pipeline 模板管理（PipelineTemplate）

- 创建模板：YAML 内容、变量声明（名称、描述、是否必填、默认值）
- 查询模板列表：区分内置和用户自定义
- 查询模板详情：包含完整 YAML 和变量声明
- 更新模板：修改 YAML 和变量声明
- 删除模板：仅用户自定义模板可删除，且无项目引用时允许

### 4. Pipeline 执行（PipelineRun）

- 手动触发：指定项目、分支/tag/commit、可选的运行时临时变量
- 变量合并：全局变量 ← 项目变量 ← 运行时变量，就近优先
- 变量校验：所有 required 变量必须有值，否则拒绝触发
- 变量渲染：使用 Jinja2 渲染 pipeline YAML
- 快照保存：resolved_pipeline 和 variables_snapshot（secret 脱敏）
- 状态管理：waiting → running → success/failed
- 查询运行列表：按项目过滤、分页
- 查询运行详情：包含状态、快照、Job 树

### 5. Job 执行

- 串行执行：按 YAML 顺序依次执行 steps
- 状态机：waiting → running → success/failed/faulted
- 容器执行：docker run，挂载 workspace 和 artifacts 目录
- 日志收集：捕获容器 stdout/stderr，存储到数据库
- checkout action：使用 alpine/git 拉取代码，注入凭据，提取 outputs
- 工作目录：每个 PipelineRun 独立 workspace（data/ci/runs/{run_id}/workspace）
- 清理：Run 结束后清理 workspace

### 6. 日志查询

- 按 Job 查询日志：返回完整日志文本
- 日志持久化：存储在数据库或文件系统

## 非功能需求

### 安全

- 凭据加密：使用 Fernet 对称加密（复用现有基础设施）
- 凭据注入：SSH 私钥通过临时文件挂载，HTTPS token 通过环境变量，执行完销毁
- 变量脱敏：variables_snapshot 中 secret 类型变量值替换为 "***"

### 性能

- 容器执行：复用 Docker daemon，避免频繁启动
- 日志收集：流式写入，避免内存溢出

### 可维护性

- 执行器抽象：PipelineExecutor 接口，便于后续迁移 Temporal
- 清晰的状态机：参考现有 Deployment 状态机设计
- 完整的日志：记录执行过程中的关键事件

## 约束

### Phase 1 不支持

- Webhook 触发（Phase 2）
- 并行执行（Phase 2）
- 重试机制（Phase 2）
- 制品记录（Phase 2）
- 取消运行（Phase 3）
- 超时控制（Phase 3）
- 并发控制（Phase 3）
- 事件通知（Phase 3）
- SSE 实时推送（Phase 3）

### 技术约束

- Python 3.12+
- FastAPI
- SQLAlchemy
- Docker Engine API
- 复用现有的 Settings 表存储全局变量

## 验收标准

1. 能创建 Project，关联 Credential 和 PipelineTemplate
2. 能手动触发 PipelineRun，传入运行时变量
3. 能执行包含 checkout 的 pipeline，成功拉取代码
4. 能执行包含自定义容器的 step，看到日志输出
5. 能查询 PipelineRun 状态和 Job 日志
6. 凭据数据加密存储，不可回显
7. 变量正确合并和渲染，secret 变量在快照中脱敏
