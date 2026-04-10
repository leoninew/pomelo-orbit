# CI Phase 2 Webhook 实现进度总结

**文档编号：** 06  
**创建日期：** 2026-03-31  
**当前阶段：** Phase 2 - Webhook 基础设施层完成

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

**测试覆盖**
- ✅ 64 个单元测试（Phase 1）
- ✅ 测试覆盖率：75%
- ✅ 代码质量：通过 ruff + mypy 检查

---

### Phase 2 - Webhook 基础设施层 ✅ 刚完成

**领域层扩展**
- ✅ Project 实体增加字段：
  - `webhook_secret: str | None` - webhook 签名密钥
  - `branch_filter: str | None` - 分支过滤（如 "main,develop"）
- ✅ Project.create() 和 update() 方法支持 webhook 字段

**基础设施层扩展**
- ✅ 数据库迁移：`v0.6.1__ci_webhook.sql`
  - 增加 `projects.webhook_secret` 字段
  - 增加 `projects.branch_filter` 字段
- ✅ ORM 模型扩展：ProjectModel 增加 webhook 字段
- ✅ Mapper 扩展：ProjectMapper 支持 webhook 字段映射
- ✅ Repository 扩展：
  - `ProjectRepository.find_by_repository_url()` - 根据仓库 URL 查找项目
- ✅ Webhook 签名验证：`webhook_verifier.py`
  - `verify_github_signature()` - GitHub HMAC-SHA256 验证
  - `verify_gitlab_signature()` - GitLab token 验证
  - 验证失败时记录日志（便于调试）

**测试覆盖**
- ✅ 10 个 webhook 签名验证测试
- ✅ 5 个 ProjectMapper 测试（含 webhook 字段）
- ✅ 4 个 ProjectRepository 测试（含 find_by_repository_url）
- ✅ 4 个 Project 实体测试（含 webhook 字段）
- ✅ 总计：94 个 CI 单元测试全部通过

**文档**
- ✅ Phase 1 设计文档（requirements, design, tasks）
- ✅ Phase 2 设计文档（requirements, design, tasks）

**代码质量**
- ✅ 通过 ruff 检查（无警告）
- ✅ 通过 mypy 类型检查
- ✅ 修复 Code Review 中的 3 个 MUST 问题：
  - 删除不必要的 webhook_secret 索引
  - 为签名验证失败添加日志
  - 改进 find_by_repository_url 文档字符串

---

## 🎯 下一步计划：完成 Webhook 触发（应用层和 API 层）

### 需要实现的模块

#### 1. Webhook Payload 解析器（基础设施层）

**文件：** `backend/src/pomelo_orbit/infrastructure/ci/webhook_parser.py`

**功能：**
- 解析 GitHub webhook payload（push, release 事件）
- 解析 GitLab webhook payload（push, tag 事件）
- 提取关键信息：
  - `repository_url: str` - 仓库 URL
  - `branch: str | None` - 分支名（如 "main"）
  - `commit_sha: str` - commit SHA
  - `author: str` - 提交者
  - `event_type: str` - 事件类型（push/release/tag）

**接口设计：**
```python
@dataclass
class WebhookPayload:
    repository_url: str
    branch: str | None
    commit_sha: str
    author: str
    event_type: str

def parse_github_webhook(payload: dict) -> WebhookPayload:
    """解析 GitHub webhook payload"""
    
def parse_gitlab_webhook(payload: dict) -> WebhookPayload:
    """解析 GitLab webhook payload"""
```

**测试：**
- `tests/infrastructure/ci/test_webhook_parser.py`
- GitHub push 事件解析
- GitHub release 事件解析
- GitLab push 事件解析
- GitLab tag 事件解析
- 异常处理（缺少字段）

---

#### 2. Webhook Service（应用层）

**文件：** `backend/src/pomelo_orbit/application/ci/webhook_service.py`

**功能：**
- `handle_webhook()` - 核心处理逻辑
  1. 验证签名（调用 webhook_verifier）
  2. 解析 payload（调用 webhook_parser）
  3. 匹配 Project（调用 ProjectRepository.find_by_repository_url）
  4. 检查 branch_filter（如果配置）
  5. 触发 PipelineRun（调用 PipelineService.trigger_pipeline）
  6. 记录 WebhookEvent（复用现有实体）
- `get_webhook_url()` - 生成 webhook URL
- `list_webhook_events()` - 查询事件历史

**接口设计：**
```python
class WebhookService:
    def __init__(
        self,
        project_repo: ProjectRepository,
        pipeline_service: PipelineService,
        webhook_event_repo: WebhookEventRepository,
    ):
        ...
    
    async def handle_webhook(
        self,
        source: str,  # "github" | "gitlab"
        signature: str,
        payload: dict,
    ) -> str:
        """
        处理 webhook 请求
        
        Returns:
            webhook_event_id: WebhookEvent ID
            
        Raises:
            BusinessError: 签名验证失败（401）
        """
    
    def get_webhook_url(self, repository_id: str) -> dict:
        """
        获取 webhook 配置
        
        Returns:
            {
                "url": "https://api.example.com/api/v1/webhooks/git",
                "secret": "xxx",
                "events": ["push", "release"]
            }
        """
    
    def list_webhook_events(
        self,
        repository_id: str,
        page: int = 1,
        per_page: int = 20,
    ) -> PaginatedResp[WebhookEvent]:
        """查询 webhook 事件历史"""
```

**测试：**
- `tests/application/ci/test_webhook_service.py`
- 签名验证成功，触发 pipeline
- 签名验证失败，返回 401
- 匹配到多个 Project，都触发
- 无匹配 Project，记录 ignored
- branch_filter 过滤生效
- 异常处理（payload 解析失败）

---

#### 3. Webhook API（接口层）

**文件：** `backend/src/pomelo_orbit/api/v1/ci/webhooks.py`

**端点：**

**POST /api/v1/webhooks/git**
- 接收 GitHub/GitLab webhook
- Header: `X-Hub-Signature-256` (GitHub) 或 `X-Gitlab-Token` (GitLab)
- Body: webhook payload (JSON)
- Response: `200 OK` 或 `401 Unauthorized`

```python
@router.post("/webhooks/git", status_code=200)
async def receive_git_webhook(
    request: Request,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
):
    """接收 Git webhook"""
    # 1. 提取签名
    signature = request.headers.get("X-Hub-Signature-256") or request.headers.get("X-Gitlab-Token")
    
    # 2. 读取 payload
    payload = await request.json()
    
    # 3. 判断来源
    source = "github" if "X-Hub-Signature-256" in request.headers else "gitlab"
    
    # 4. 调用 service
    event_id = await webhook_service.handle_webhook(source, signature, payload)
    
    return {"event_id": event_id}
```

**GET /api/v1/projects/{id}/webhook-events**
- 查询项目的 webhook 事件历史
- Query: `page`, `per_page`
- Response: `PaginatedResp[WebhookEvent]`

---

#### 4. Project Service 扩展

**文件：** `backend/src/pomelo_orbit/application/ci/project_service.py`

**扩展：**
- `create_project()` - 生成 webhook_secret（使用 secrets.token_urlsafe）
- `get_webhook_config()` - 获取 webhook 配置

```python
def create_project(self, req: CreateProjectReq) -> Project:
    """创建项目"""
    # 生成 webhook_secret
    webhook_secret = secrets.token_urlsafe(32) if req.enable_webhook else None

    project = Project.create(
        name=req.name,
        repository_url=req.repository_url,
        pipeline_template_id=req.pipeline_template_id,
        git_credential_id=req.git_credential_id,
        webhook_secret=webhook_secret,
        branch_filter=req.branch_filter,
    )
    ...


def get_webhook_config(self, repository_id: str) -> dict:
    """获取 webhook 配置"""
    project = self.repository_repo.find_by_id(repository_id)
    if not project:
        raise BusinessError(f"Project {repository_id} not found", status_code=404)

    return {
        "url": f"{self.settings.api_base_url}/api/v1/webhooks/git",
        "secret": project.webhook_secret,
        "events": ["push", "release"],
    }
```

---

#### 5. 集成测试

**文件：** `tests/integration/ci/test_webhook_integration.py`

**测试场景：**
- 端到端：GitHub webhook → 验证签名 → 匹配 Project → 触发 PipelineRun
- 端到端：GitLab webhook → 验证签名 → 匹配 Project → 触发 PipelineRun
- 签名验证失败，返回 401
- 无匹配 Project，记录 ignored 事件
- branch_filter 过滤，不触发 pipeline
- 多个 Project 匹配同一仓库，都触发

---

## 📅 时间估算

| 任务 | 预计时间 |
|------|---------|
| Webhook Payload 解析器 + 测试 | 2 小时 |
| Webhook Service + 测试 | 3 小时 |
| Webhook API + 测试 | 1 小时 |
| Project Service 扩展 | 0.5 小时 |
| 集成测试 | 1.5 小时 |
| **总计** | **8 小时（1 天）** |

---

## 🔄 后续阶段规划

### 第三阶段：制品记录（1 天）
- Artifact 实体和 Repository
- 执行器扩展（制品记录逻辑）
- Artifact Service 和 API
- 单元测试和集成测试

### 第四阶段：重试机制（1-2 天）
- 数据库迁移（retry_of 字段）
- 执行器扩展（重试跳过逻辑）
- Pipeline Service 扩展
- 单元测试和集成测试

### 第五阶段：文档和部署（1 天）
- API 文档更新（OpenAPI/Swagger）
- 用户指南（如何配置 webhook）
- 部署配置和示例

**Phase 2 总计：** 5-8 天

---

## 📝 技术决策记录

### 1. Webhook 签名验证日志

**决策：** 在签名验证失败时记录 warning 日志

**理由：**
- 便于调试 webhook 配置问题
- 帮助识别恶意请求
- 不影响正常流程性能

**实现：**
```python
if not is_valid:
    logger.warning("GitHub signature verification failed: signature mismatch")
```

### 2. 删除 webhook_secret 索引

**决策：** 不为 webhook_secret 创建数据库索引

**理由：**
- webhook 匹配是通过 repository_url 查询，不是通过 webhook_secret
- webhook_secret 仅用于签名验证，不用于查询
- 减少不必要的索引维护开销

### 3. find_by_repository_url 返回列表

**决策：** find_by_repository_url() 返回 list[Project] 而不是单个 Project

**理由：**
- 多个项目可能使用同一个仓库 URL（如不同分支的项目）
- 支持一个 webhook 触发多个 pipeline 的场景
- 更灵活的配置方式

---

## ✅ 验收标准

### 当前阶段（Webhook 基础设施层）
- ✅ Project 实体支持 webhook 字段
- ✅ 数据库迁移脚本正确
- ✅ GitHub/GitLab 签名验证正确
- ✅ find_by_repository_url 查询正确
- ✅ 所有单元测试通过（94 个）
- ✅ 代码通过 ruff 和 mypy 检查

### 下一阶段（Webhook 应用层和 API 层）
- [ ] 能接收 GitHub webhook 并自动触发 PipelineRun
- [ ] 能接收 GitLab webhook 并自动触发 PipelineRun
- [ ] 能验证 webhook 签名，拒绝无效请求
- [ ] 能根据 branch_filter 过滤分支
- [ ] 能记录 webhook 事件到数据库
- [ ] 能查询 webhook 事件历史
- [ ] 所有单元测试和集成测试通过

---

## 📚 参考文档

- [Phase 1 需求文档](./specs/ci-phase1/requirements.md)
- [Phase 1 设计文档](./specs/ci-phase1/design.md)
- [Phase 2 需求文档](./specs/ci-phase2/requirements.md)
- [Phase 2 设计文档](./specs/ci-phase2/design.md)
- [Phase 2 任务清单](./specs/ci-phase2/tasks.md)
- [项目编码规范](../CLAUDE.md)

---

**最后更新：** 2026-03-31  
**更新人：** Kiro AI Assistant
