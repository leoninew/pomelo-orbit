# CI Pipeline 设计文档

**日期**: 2026-04-07
**版本**: v7.0

---

## 核心概念

### 流水线阶段（PipelineStage）

执行的最小单元。一个 Stage 描述"做什么"：用哪个镜像、跑哪些命令、产出什么制品。

**Stage 自身不含任何编排信息**——它不知道自己在哪个流水线里、前面是谁、后面是谁。`depends_on`、执行顺序、并行方式都是流水线模板对它的编排，不属于 Stage 本身。

```
PipelineStage {
  id
  name          // 唯一标识，如 "checkout"、"unit-test"、"build-image"
  image         // 执行镜像，如 "python:3.12-slim"
  script        // Shell 脚本，多行，按行顺序执行
  env           // Stage 级环境变量（可含占位符）
  artifacts     // 制品声明（可含占位符），约定产出路径和名称
  description
}
```

**变量占位符**：`script`、`env`、`artifacts` 中可以使用 `{{ VAR_NAME }}` 占位符。
Stage 自身可以分析出所需变量列表，但变量的管理（声明、默认值、是否必填）在模板层聚合，不在 Stage 层单独维护。

**制品**：Stage 的额外属性，基于变量占位，运行前就约定了制品在哪里、叫什么名字，运行后直接得到。制品领域将来单独设计。

---

### 流水线模板（PipelineTemplate）

将一组 Stage 编排起来，定义它们的依赖关系和执行顺序，配合变量声明，就可以运行。

```
PipelineTemplate {
  id
  name
  description
  orchestration: StageOrchestration[]   // 编排：stage + 依赖 + 顺序
  variable_declarations: VariableDeclaration[]  // 聚合所有 Stage 的变量，统一管理
}

StageOrchestration {
  stage_id      // 引用哪个 PipelineStage
  depends_on    // 依赖哪些 stage（按 stage_id 引用），决定串并行
  sort_order    // 在列表视图中的显示顺序
}
```

**编排决定执行方式**：
- `depends_on` 为空 → 可以立即执行（与其他无依赖的 Stage 并行）
- `depends_on: ["checkout"]` → 等 checkout 完成后才执行
- 同一层（依赖相同前置）的多个 Stage 并行执行

**变量管理**：
- 模板编排好后，遍历所有 Stage 的 `script`/`env`/`artifacts` 自动提取占位符，生成 `variable_declarations`
- `variable_declarations` 可以补充元数据（描述、是否必填、默认值、是否 secret）
- 项目（Project）已定义的变量按名称自动匹配填充，未匹配的 required 变量需用户配置
- 触发时用户可提供临时变量，优先级最高

**版本管理**：模板可以重新编排、重新配置变量。每次修改 `updated_at` 更新，触发时检测到版本变更则创建新快照。

---

### 流水线快照（PipelineSnapshot）

模板某一版本的不可变副本。

触发运行时，如果模板自上次快照以来有变更（`template.updated_at > snapshot.created_at`），则创建新快照，将当前编排和变量声明冻结下来。

```
PipelineSnapshot {
  id
  template_id
  version                           // 单调递增，从 1 开始
  orchestration_snapshot            // 冻结的编排（含 stage 内容 + depends_on）
  variable_declarations_snapshot    // 冻结的变量声明
  created_at
}
```

快照一旦创建不可修改，保证历史 Run 的可追溯性。

---

### 流水线运行记录（PipelineRun）

模板（通过快照）的一次执行过程和结果。

```
PipelineRun {
  id
  repository_id
  pipeline_snapshot_id    // 本次运行基于哪个快照
  trigger                 // manual | webhook
  trigger_ref             // 分支/commit
  variables_snapshot      // 本次运行合并后的变量（脱敏存储）
  status                  // waiting | running | success | failed | canceled
  retry_of                // 重试时指向原 run
  started_at
  finished_at
}
```

对用户有价值的展示：
- **流程编排**：哪些 Stage 并行、哪些串行，依赖关系可视化（DAG）
- **运行日志**：每个 StageRun 的状态、耗时、输出日志

```
StageRun {
  id
  pipeline_run_id
  name              // 对应 Stage 名称
  status            // waiting | running | success | failed | faulted | canceled
  started_at
  finished_at
  exit_code
  error_message
}
```

---

## 领域关系

```
PipelineStage（独立存储，可复用）
    ↑ 被编排引用
PipelineTemplate（编排 + 变量声明）
    ↓ 触发时按需创建快照
PipelineSnapshot（不可变副本）
    ↓ 每次运行基于一个快照
PipelineRun（执行过程 + 结果）
    ↓ 每个 Stage 产生一条记录
StageRun × N
```

**关键设计原则**：
- Stage 是可复用的执行单元，编排属性（`depends_on`、`sort_order`）属于模板，不属于 Stage
- 变量在模板层聚合管理，Stage 只负责声明占位符
- 快照保证历史可追溯，删除模板不影响历史 Run

---

## 变量优先级

高 → 低：

```
内置变量（自动注入，不可覆盖）
  REPOSITORY_URL, DEFAULT_BRANCH, GIT_CREDENTIAL_ID
  trigger_ref, trigger_type, repository_name
    ↓
运行时临时变量（触发时用户提供，locked 变量除外）
    ↓
项目级变量（Project.variable_overrides）
    ↓
全局变量（系统配置）
    ↓
声明默认值（variable_declarations[].default）
```

---

## 触发流程

### 手动触发

```
POST /api/ci/projects/{id}/trigger
  body: { template_id, trigger_ref?, variables? }
```

1. 选择模板 → 加载 `variable_declarations`
2. 项目变量自动匹配填充
3. 未匹配的 required 变量高亮待填写
4. 提交 → 合并变量 → 检测快照版本 → 创建 PipelineRun → 异步执行

### Webhook 触发

```
POST /api/ci/webhooks/{webhook_id}
  headers: X-Hub-Signature-256 | X-Gitlab-Token
```

1. 查找 `ProjectWebhook`（含 `repository_id` + `template_id`）
2. 验证签名
3. 检查 `branch_filter`（空/None 则拒绝所有分支，有值则 glob 匹配）
4. 触发对应项目 + 模板

---

## 数据库结构

```sql
-- Stage：独立存储，不含编排属性
CREATE TABLE pipeline_stages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    env TEXT NOT NULL DEFAULT '{}',       -- JSON
    artifacts TEXT,                        -- JSON | NULL
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 模板：持有编排信息
CREATE TABLE pipeline_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 编排：模板对 Stage 的引用 + 依赖 + 顺序
CREATE TABLE pipeline_template_stages (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES pipeline_templates(id) ON DELETE CASCADE,
    stage_id TEXT NOT NULL REFERENCES pipeline_stages(id),
    depends_on TEXT NOT NULL DEFAULT '[]',  -- JSON array of stage_id
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (template_id, stage_id)
);

-- 快照：冻结编排 + 变量声明
CREATE TABLE pipeline_snapshots (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES pipeline_templates(id),
    version INTEGER NOT NULL,
    orchestration_snapshot TEXT NOT NULL DEFAULT '[]',          -- JSON
    variable_declarations_snapshot TEXT NOT NULL DEFAULT '[]',  -- JSON
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (template_id, version)
);
```

---

## API

| 端点 | 用途 |
|------|------|
| `GET /api/ci/stages` | Stage 列表 |
| `POST /api/ci/stages` | 创建 Stage |
| `GET /api/ci/stages/{id}` | Stage 详情 |
| `PUT /api/ci/stages/{id}` | 更新 Stage |
| `DELETE /api/ci/stages/{id}` | 删除 Stage |
| `GET /api/ci/templates` | 模板列表 |
| `POST /api/ci/templates` | 创建模板 |
| `GET /api/ci/templates/{id}` | 模板详情（含编排） |
| `PUT /api/ci/templates/{id}` | 更新模板信息 |
| `DELETE /api/ci/templates/{id}` | 删除模板 |
| `GET /api/ci/templates/{id}/orchestration` | 获取编排 |
| `PUT /api/ci/templates/{id}/orchestration` | 更新编排（整体替换） |
| `GET /api/ci/templates/{id}/snapshots` | 快照列表 |
| `GET /api/ci/projects/{id}/webhooks` | 项目 Webhook 列表 |
| `POST /api/ci/projects/{id}/webhooks` | 创建 Webhook |
| `PUT /api/ci/projects/{id}/webhooks/{wid}` | 更新 Webhook |
| `DELETE /api/ci/projects/{id}/webhooks/{wid}` | 删除 Webhook |
| `POST /api/ci/webhooks/{webhook_id}` | Git 平台推送入口（公开） |
| `POST /api/ci/projects/{id}/trigger` | 手动触发 |
| `GET /api/ci/runs/{id}/stages` | StageRun 列表 |
| `GET /api/ci/stages/{stage_run_id}/log` | Stage 执行日志 |

---

## 工作目录结构

```
data/ci/{project.code}/
├── workspace/              # 克隆目录（项目共享，每次 run 覆盖）
└── runs/{run_id}/
    ├── artifacts/
    └── secrets/            # SSH key 等（执行后立即删除）
```

---

## 待设计

- 制品（Artifact）领域：制品存储、版本管理、跨 Run 引用
- Stage 复用：同一个 Stage 被多个模板编排
