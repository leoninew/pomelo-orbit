# CI Phase 3 制品记录实现进度总结

**文档编号：** 07  
**创建日期：** 2026-03-31  
**当前阶段：** Phase 3 - 制品记录完成

---

## 📊 整体进度概览

### Phase 1 - 基础功能 ✅ 已完成

**领域层**
- ✅ 6 个核心实体：Project, Credential, PipelineTemplate, PipelineRun, Job, JobLog
- ✅ 值对象和枚举：CredentialType, JobStatus, PipelineRunStatus, PipelineDefinition, StepDefinition 等
- ✅ 执行器接口：PipelineExecutor, ExecutionContext

**基础设施层**
- ✅ 数据库迁移：`v0.6.0__ci_tables.sql`（6 张表）
- ✅ 6 个 Repository 实现（Project, Credential, Template, Run, Job, JobLog）
- ✅ 容器执行器：ContainerExecutor（Docker 封装）
- ✅ Checkout Action：支持 SSH/HTTPS 凭据注入
- ✅ 依赖图：DependencyGraph（拓扑排序、循环检测）
- ✅ 并行执行器：PipelineExecutorImpl（asyncio.gather）
- ✅ YAML 解析器：PipelineParser
- ✅ 变量处理：merge、validate、render、mask
- ✅ 工作目录管理：workspace 创建和清理

**应用层 + 接口层**
- ✅ PipelineService：Project / Credential / Template / Run CRUD
- ✅ CI API：`/api/v1/ci/...` 全套端点
- ✅ DI 注入：ci_di.py

**测试覆盖**
- ✅ 64 个单元测试（Phase 1）
- ✅ 代码质量：通过 ruff + mypy 检查

---

### Phase 2 - Webhook 触发 ✅ 已完成

**领域层扩展**
- ✅ Project 实体增加 `webhook_secret`、`branch_filter` 字段

**基础设施层扩展**
- ✅ 数据库迁移：`v0.6.1__ci_webhook.sql`
- ✅ ORM / Mapper / Repository 扩展（含 `find_by_repository_url`）
- ✅ Webhook 签名验证：`webhook_verifier.py`（GitHub HMAC-SHA256 / GitLab token）
- ✅ Webhook Payload 解析器：`webhook_payload_parser.py`（GitHub / GitLab push、tag、release）

**应用层 + 接口层**
- ✅ `CIWebhookService`：签名验证 → 匹配 Project → branch_filter → 触发 PipelineRun
- ✅ `PipelineService.trigger_pipeline`：变量合并、校验、快照、后台异步执行
- ✅ `_execute_run`：独立 session 后台任务，避免请求 session 关闭问题
- ✅ `POST /api/v1/ci/webhooks/git`：统一接收 GitHub / GitLab webhook
- ✅ 模块级 `_background_tasks` 集合防止 asyncio task 被 GC

**测试覆盖**
- ✅ 11 个 webhook payload 解析器测试
- ✅ 总计 156 个单元测试全部通过
- ✅ 代码质量：通过 ruff + mypy 检查

---

### Phase 3 - 制品记录 ✅ 刚完成

**领域层**
- ✅ 新增 `Artifact` 实体：`pipeline_run_id`、`job_name`、`type`（docker_image | file）、`name`、`path`、`created_at`
- ✅ `Artifact.create()` 工厂方法，参数使用 `artifact_type` 避免遮蔽内置 `type`

**基础设施层**
- ✅ 数据库迁移：`v0.6.2__ci_artifacts.sql`（含 `ON DELETE CASCADE`）
- ✅ `ArtifactModel`：ORM 模型
- ✅ `ArtifactMapper`：ORM ↔ 领域实体转换
- ✅ `ArtifactRepository`：含 `find_by_run(pipeline_run_id)` 查询

**执行器扩展**
- ✅ `PipelineExecutorImpl._save_artifacts`：job 成功后解析 step.artifacts 声明写入制品记录
- ✅ 写入前校验：
  - 类型合法性（仅允许 `docker_image` / `file`，未知类型 warning + skip）
  - name 非空（空 name warning + skip）
- ✅ file 类型：path 自动拼接 `artifacts_path` 前缀
- ✅ docker_image 类型：path 为 None，仅记录镜像 tag

**应用层**
- ✅ `PipelineService.list_artifacts(run_id) -> list[Artifact]`：先校验 run 存在，再查询制品

**接口层**
- ✅ `ArtifactResp` DTO
- ✅ `GET /api/v1/ci/runs/{run_id}/artifacts`：列出 run 的所有制品，需认证

**测试覆盖**
- ✅ 156 个单元测试全部通过
- ✅ 代码质量：通过 ruff + mypy 检查

---

## 🔄 后续阶段规划

### 第四阶段：重试机制（1-2 天）

- 数据库迁移：`pipeline_runs` 增加 `retry_of` 字段（引用原 Run ID）
- 领域层：`PipelineRun` 实体增加 `retry_of` 字段
- 执行器扩展：重试时根据 `retry_policy` 决定哪些 job 可跳过（`skip_if_success`）
- `PipelineService.retry_pipeline(run_id)`：复用原 Run 的 pipeline 快照和变量快照
- API：`POST /api/v1/ci/runs/{run_id}/retry`

### 第五阶段：前端 CI 页面（2-3 天）

- Project / Credential / Template 管理页面
- PipelineRun 列表和详情页（含 Job 状态树、日志查看）
- Artifact 列表展示
- Webhook 配置展示（URL + secret）

### 第六阶段：文档和部署（1 天）

- API 文档更新（OpenAPI/Swagger）
- 用户指南（如何配置 webhook、如何编写 pipeline YAML）
- 部署配置示例

---

## 📝 关键技术决策记录

### 1. 后台任务 session 隔离

`_execute_run` 在独立 session 中运行，不复用 HTTP 请求的 session。原因：请求结束后 session 被 `get_db()` 的 finally 关闭，后台 task 继续使用会报 `Session is closed`。

### 2. asyncio task 生命周期管理

使用模块级 `_background_tasks: set[asyncio.Task]` 持有 task 引用，防止 task 在 service 对象被 GC 时一起消失。task 完成后通过 `done_callback` 自动从集合中移除。

### 3. 制品类型校验在执行器层

类型合法性校验放在 `_save_artifacts` 而不是 YAML 解析器，原因：YAML 解析时无法知道平台支持哪些类型，执行时校验更接近实际写入点，错误信息也更精确。

### 4. Webhook 签名验证失败策略

多 Project 匹配同一仓库时，某个 Project 签名验证失败只 `continue` 跳过，不中断其他 Project 的处理。原因：不同 Project 可能配置了不同的 webhook_secret，一个失败不应影响其他。

---

**最后更新：** 2026-03-31  
**更新人：** Kiro AI Assistant
