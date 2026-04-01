# CI Phase 2 需求文档

## 目标

在 Phase 1 基础上，实现完整的 CI 功能：Webhook 触发、并行执行、制品记录、重试机制。

## 功能需求

### 1. Webhook 触发

#### 1.1 Webhook 接收
- 接收 GitHub/GitLab webhook 请求
- 验证 webhook 签名（使用 Project.webhook_secret）
- 解析 webhook payload（push、release 事件）
- 记录 webhook 事件到数据库

#### 1.2 自动触发
- 根据 webhook 事件自动触发 PipelineRun
- 匹配 Project.repository_url 和 webhook 来源
- 检查分支是否匹配（如果 Project 配置了分支过滤）
- 提取 commit SHA、branch、author 等信息

#### 1.3 Webhook 管理
- Project 创建时生成 webhook_secret
- 提供 webhook URL 供用户配置到 Git 平台
- 查询 webhook 事件历史

### 2. 并行执行

#### 2.1 depends_on 支持
- 解析 step 的 depends_on 字段
- 构建 Job 依赖图（DAG）
- 使用 asyncio.gather 实现并行执行
- 等待所有依赖 Job 完成后才开始执行

#### 2.2 并行调度
- 无依赖的 Job 可以并行执行
- 有依赖的 Job 等待依赖完成
- 支持嵌套 steps 的并行执行

#### 2.3 Fail-fast
- 任一 Job 失败时，停止调度新 Job
- 已运行的 Job 继续执行完成
- 未开始的 Job 标记为 canceled

### 3. 制品记录

#### 3.1 Artifact 实体
- 记录 docker_image 类型制品（镜像 tag）
- 记录 file 类型制品（文件路径）
- 关联到 PipelineRun 和 Job

#### 3.2 制品声明
- 在 step 中通过 artifacts 字段声明
- 支持变量替换（如 `{{ trigger_ref }}`）
- Job 成功后才记录制品

#### 3.3 制品查询
- 按 PipelineRun 查询制品列表
- 按 Project 查询历史制品
- 提供文件制品下载接口

### 4. 重试机制

#### 4.1 重试触发
- 手动触发重试（指定原 PipelineRun ID）
- 创建新的 PipelineRun，携带 retry_of 引用
- 复用原 Run 的 pipeline 定义和变量快照

#### 4.2 重试策略
- always_rerun：总是重新执行（如 checkout）
- skip_if_success：上次成功且依赖链无变化则跳过（默认）
- Job 状态标记为 skipped

#### 4.3 跳过传播
- 如果依赖链上有任何 Job 重新运行，则当前 Job 也重新运行
- 只有所有依赖都跳过，当前 Job 才能跳过

### 5. 变量系统完整实现

#### 5.1 三层变量合并
- 全局变量（Settings 表）
- 项目级变量（Project.variable_overrides）
- 运行时临时变量（trigger 时传入）
- 就近优先原则

#### 5.2 变量校验
- 检查所有 required 变量都有值
- 触发时校验失败则拒绝执行

#### 5.3 变量渲染
- 使用 Jinja2 渲染 pipeline YAML
- 支持条件、过滤器等 Jinja2 语法
- 内置变量：trigger_ref、trigger_type、project_name、run_id

#### 5.4 变量脱敏
- variables_snapshot 中 secret 类型变量值替换为 "***"
- 日志中不输出 secret 变量值

## 非功能需求

### 性能
- 并行执行提升构建速度
- 重试时跳过成功 Job 节省时间

### 可靠性
- Webhook 签名验证防止伪造
- Fail-fast 避免浪费资源
- 制品记录确保可追溯

### 可维护性
- 清晰的依赖图构建逻辑
- 完整的 webhook 事件日志
- 重试逻辑易于理解和调试

## 约束

### Phase 2 不支持
- 取消运行（Phase 3）
- 超时控制（Phase 3）
- 并发控制（Phase 3）
- 事件通知（Phase 3）
- SSE 实时推送（Phase 3）

### 技术约束
- 并行执行使用 asyncio.gather
- Webhook 签名使用 HMAC-SHA256
- 制品文件存储在本地文件系统

## 验收标准

1. 能接收 GitHub webhook 并自动触发 PipelineRun
2. 能验证 webhook 签名，拒绝无效请求
3. 能并行执行无依赖的 Job（如 lint 和 test 并行）
4. 能正确处理 depends_on 依赖关系
5. 能记录 docker_image 和 file 类型制品
6. 能下载文件类型制品
7. 能重试失败的 PipelineRun
8. 重试时能跳过成功的 Job（skip_if_success）
9. 变量系统三层合并正确，secret 变量正确脱敏
10. Fail-fast 正确工作，失败时停止调度新 Job
