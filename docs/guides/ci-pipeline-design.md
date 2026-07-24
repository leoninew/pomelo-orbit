# CI Pipeline 设计文档
最后修改时间: 2026-07-24 10:47:37

Doc role: living guide。与代码冲突时以代码为准。

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
- 模板编排好后，遍历所有 Stage 的 `script`/`env`/`artifacts` 自动提取占位符
- 内置变量（仓库 + 模板）只在 Stage 实际引用时才出现在变量列表中，`editable` 由 spec 决定（`repository_ref` 可编辑，其余只读）
- Stage 中引用的非内置变量自动补充到模板变量声明，标记为 `template_stage`，可编辑
- 支持从 `{{ VAR | default('x') }}` 提取默认值，存入 `default` 字段；用户覆盖值存入 `value` 字段
- `template_stage` 变量用户设置了值后，保存时升级为 `template_custom` 持久化
- 所有 `template_custom` 变量最终都必须有值；`secret` 仅用于脱敏展示
- 仓库自定义变量在触发时按名称合并，用户可在触发时临时覆盖非内置变量，但不回写模板

**版本管理**：模板可以重新编排、重新配置变量。每次修改递增 `version`，触发时检测到版本变更则创建新快照。

---

### 流水线快照（PipelineSnapshot）

模板某一版本的不可变副本。

触发运行时，如果模板自上次快照以来有变更（`template.updated_at > snapshot.created_at`），则创建新快照，将当前编排和变量声明冻结下来。

```
PipelineSnapshot {
  id
  template_id
  version                           // 单调递增，从 1 开始
  stages_snapshot                   // 冻结的编排（含 stage 内容 + depends_on）
  variables_snapshot                // 冻结的变量声明（只含 Stage 实际用到的变量）
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
  repository_name
  snapshot_id             // 本次运行基于哪个快照
  template_id
  template_name
  template_version        // 触发时快照的版本号
  trigger                 // manual | webhook
  trigger_ref             // 分支/commit
  variables_snapshot      // 本次运行实际变量（脱敏存储，含 source 标记）
  status                  // waiting_to_run | running | ran_to_completion | faulted | canceled
  error_message           // 失败原因（faulted 时填充）
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
内置变量（全局 + 仓库 + 模板，只读，不可覆盖）
    ↓
运行时临时覆盖（触发时用户提供，仅作用于本次运行，不能覆盖内置变量）
    ↓
仓库自定义变量（Repository.variable_overrides）
    ↓
模板声明默认值（variable_declarations[].value 或 .default）
    ↓
Stage 提取变量默认值（template_stage，最低优先级）
```

**说明**：运行时覆盖优先级高于仓库自定义变量，确保触发时传入的值不被仓库配置静默覆盖。

---

## 触发流程

### 手动触发

```
POST /api/ci/projects/{id}/trigger
  body: { template_id, trigger_ref?, variables? }
```

1. 选择模板 → 加载 `variable_declarations`（经 `resolve_template_variables` 计算的完整列表）
2. 内置变量展示但禁用输入（`repository_ref` 除外，可编辑）；`template_stage` 变量显示 default 值，可覆盖；`template_custom` 变量可输入
3. 校验所有 `template_custom` 变量均已有值；`template_stage` 有 default 时豁免
4. 提交 → 合并变量（优先级：内置 > 运行时覆盖 > 仓库自定义 > 模板声明 > stage default）→ 检测快照版本 → 创建 PipelineRun → 异步执行

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
CREATE TABLE pipeline_stage (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    env TEXT NOT NULL DEFAULT '{}',       -- JSON
    artifacts TEXT,                        -- JSON | NULL
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 模板：持有编排信息
CREATE TABLE pipeline_template (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',  -- JSON
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 编排：模板对 Stage 的引用 + 依赖 + 顺序
CREATE TABLE pipeline_template_stage (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES pipeline_template(id) ON DELETE CASCADE,
    stage_id TEXT NOT NULL REFERENCES pipeline_stage(id),
    depends_on TEXT NOT NULL DEFAULT '[]',  -- JSON array of stage_id
    sort_order INTEGER NOT NULL DEFAULT 0,
    UNIQUE (template_id, stage_id)
);

-- 快照：冻结编排 + 变量声明
CREATE TABLE pipeline_snapshot (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL REFERENCES pipeline_template(id),
    version INTEGER NOT NULL,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',      -- JSON，冻结的 Stage 定义（含 depends_on）
    variables_snapshot TEXT NOT NULL DEFAULT '[]',   -- JSON，冻结的变量声明
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (template_id, version)
);
```

---

## API

| 端点 | 用途 |
|------|------|
| `GET /api/ci/pipeline-stage` | Stage 列表 |
| `POST /api/ci/pipeline-stage` | 创建 Stage |
| `GET /api/ci/pipeline-stage/{id}` | Stage 详情 |
| `PUT /api/ci/pipeline-stage/{id}` | 更新 Stage |
| `DELETE /api/ci/pipeline-stage/{id}` | 删除 Stage |
| `GET /api/ci/template` | 模板列表 |
| `POST /api/ci/template` | 创建模板 |
| `GET /api/ci/template/{id}` | 模板详情（含编排） |
| `PUT /api/ci/template/{id}` | 更新模板信息及编排 |
| `POST /api/ci/template/resolve-variables` | 实时解析模板变量（前端预览用） |
| `GET /api/ci/snapshot/{id}` | 快照详情 |
| `GET /api/ci/repository` | 仓库列表 |
| `POST /api/ci/repository` | 创建仓库 |
| `GET /api/ci/repository/{id}` | 仓库详情 |
| `PUT /api/ci/repository/{id}` | 更新仓库 |
| `DELETE /api/ci/repository/{id}` | 删除仓库 |
| `GET /api/ci/repository/{id}/run` | 仓库的运行列表 |
| `POST /api/ci/repository/{id}/trigger` | 手动触发 |
| `GET /api/ci/repository/{id}/webhook` | Webhook 列表 |
| `POST /api/ci/repository/{id}/webhook` | 创建 Webhook |
| `PUT /api/ci/repository/{id}/webhook/{wid}` | 更新 Webhook |
| `DELETE /api/ci/repository/{id}/webhook/{wid}` | 删除 Webhook |
| `POST /api/ci/webhook/{webhook_id}` | Git 平台推送入口（公开） |
| `GET /api/ci/run` | 所有运行列表 |
| `GET /api/ci/run/{id}` | 运行详情（含 stage_runs） |
| `GET /api/ci/run/{id}/artifacts` | 运行的制品列表 |
| `GET /api/ci/run/{id}/stages/{stage_run_id}/log` | Stage 执行日志（增量） |
| `POST /api/ci/run/{id}/cancel` | 取消运行 |
| `POST /api/ci/run/{id}/retry` | 重试运行 |

---

## 工作目录结构

```
data/ci/{repository.code}/
├── workspace/              # 克隆目录（仓库共享，每次 run 覆盖）
└── runs/{run_id}/
    ├── artifacts/
    └── secrets/            # SSH key 等（执行后立即删除）
```

---

## 待设计

- 制品（Artifact）领域：制品存储、版本管理、跨 Run 引用
- Stage 复用：同一个 Stage 被多个模板编排
