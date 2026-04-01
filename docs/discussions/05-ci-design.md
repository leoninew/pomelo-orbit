# [探索性讨论] CI 系统设计

> 本文档整理自设计讨论，记录 CI 功能的核心设计决策，供后续 spec 参考。

---

## 一、边界划定

本次开发的是持续集成（CI），不是部署（CD）。

- CI 的职责：监听 Webhook 或手动触发，解析 pipeline 配置，执行构建，输出制品
- CD 的职责：将制品组合成应用并部署，是后续独立的设计

现有的 Application / Deployment 概念属于 CD 侧，CI 侧引入全新的实体，两者完全解耦。

---

## 二、核心实体

### Project（项目）

对应一个代码仓库，是 CI 的最小粒度。

```
Project
  repository_url          # 仓库地址（https:// 或 git@ 格式）
  pipeline_template_id    # 引用的 PipelineTemplate
  git_credential_id       # 拉代码用的凭据
  variable_overrides      # 项目级变量，覆盖全局变量或模板默认值
  webhook_secret          # 接收 Webhook 鉴权用
  concurrency_policy      # 并发策略（见并发控制章节）
```

### Credential（凭据）

独立实体，可被多个 Project 复用。

```
Credential
  name              # 用户起的名字，如 "公司 GitLab"
  type              # git_ssh | git_token | registry_token
  data              # 加密存储（Fernet 对称加密，复用现有基础设施）
```

UI 上只显示凭据名称，不回显内容。凭据注入不落盘明文：
- SSH 私钥：写入临时文件挂载到容器，执行完销毁
- HTTPS token：通过 GIT_ASKPASS 或 URL 内嵌方式注入容器环境变量

### PipelineTemplate（模板）

可复用的 pipeline 定义，独立于 Project 存在。

```
PipelineTemplate
  name                    # 如 "uv + FastAPI 后端构建"
  description
  content                 # pipeline yaml 内容
  variable_declarations   # 声明模板需要哪些变量（名称、描述、是否必填、默认值）
  is_builtin              # 平台内置 vs 用户自定义
```

### PipelineRun（执行实例）

每次触发创建一条记录。

```
PipelineRun
  project_id
  trigger                 # webhook | manual
  trigger_ref             # branch / tag / commit sha
  resolved_pipeline       # 快照：本次实际使用的 pipeline yaml
  variables_snapshot      # 快照：本次变量值（secret 脱敏）
  status                  # waiting | running | canceling | success | failed | canceled
  retry_of                # 重试时引用原 Run ID
  jobs[]                  # 执行树（见 Job 模型）
```

### Artifact（制品）

```
Artifact
  pipeline_run_id
  job_name
  type              # docker_image | file
  name              # 镜像 tag 或文件名
  path              # file 类型时的宿主机路径，docker_image 为空
  created_at
```

### ProjectNotification（通知订阅）

```
ProjectNotification
  project_id
  event              # run.success | run.failed | run.canceled | *
  channel            # webhook | email
  target             # webhook URL 或邮件地址
  enabled
```

---

## 三、Pipeline 配置来源

所有 pipeline 定义集中在平台管理，通过 PipelineTemplate 引用。不支持从仓库读取 `.ci.yaml`，避免鸡和蛋的问题（读取 `.ci.yaml` 本身需要先 checkout，而 checkout 又依赖 pipeline 定义）。

---

## 四、变量系统

### 三层变量，就近优先

```
全局变量（Global Variables，复用现有 Settings）
  ↓ 被覆盖
项目级变量（Project.variable_overrides）
  ↓ 被覆盖
运行时临时变量（手动触发时传入，不持久化）
```

### 变量声明（模板侧）

```yaml
variables:
  - name: REGISTRY
    description: 镜像仓库地址
    required: true
  - name: IMAGE_NAME
    required: true
  - name: PYTHON_VERSION
    required: false
    default: "3.12"
```

### 内置变量

平台执行时自动注入：

| 变量 | 说明 |
|------|------|
| `trigger_ref` | branch / tag / commit sha |
| `trigger_type` | webhook \| manual |
| `project_name` | 项目名称 |
| `run_id` | 本次 PipelineRun ID |

### 校验逻辑

触发时校验所有 required 变量都有值，否则拒绝执行。Webhook 触发校验失败则事件标记为 error。

### 变量占位符

使用 Jinja2 语法，与项目现有渲染机制一致，支持条件和过滤器。

---

## 五、Job 模型（统一 stage/step）

stage 和 step 本质相同，统一为 **Job**，支持嵌套：

- 有 `steps` 子列表 = 编排节点，自身不执行容器，等所有 step 完成才算完成
- 有 `image` / `commands` 或 `uses` = 叶子节点，实际执行
- 默认串行，`depends_on` 打破串行实现并行
- 任意层级规则一致，实现上只需一个递归的 Job 类

### Job 状态机

参考现有 Deployment 任务状态机设计，Job 状态定义如下：

| 状态 | 值 | 含义 |
|------|-----|------|
| 待运行 | `waiting` | 已创建，等待依赖完成或调度 |
| 运行中 | `running` | 正在执行（叶子 job）或子 job 执行中（编排 job） |
| 成功 | `success` | 执行成功 |
| 失败 | `failed` | 执行失败（容器非零退出） |
| 故障 | `faulted` | 执行异常（超时、容器启动失败等） |
| 跳过 | `skipped` | 重试时因 skip_if_success 策略跳过 |
| 取消 | `canceled` | 用户取消或 fail-fast 触发 |

#### 叶子 Job 状态转换

```
waiting ──调度──► running ──┬── 容器退出码 0 ──► success
                           ├── 容器退出码非 0 ──► failed
                           ├── 超时/启动失败 ──► faulted
                           └── 用户取消 ──► canceled

waiting ──重试跳过──► skipped
waiting ──用户取消──► canceled
running ──用户取消──► canceled
```

#### 编排 Job 状态聚合规则

编排 job（有 `steps` 子列表）的状态从 steps 聚合：

1. **初始状态**：`waiting`
2. **进入 running**：任一 step 进入 `running`
3. **终态判定**（所有 step 都到达终态后）：
   - 所有 step 都是 `success` → `success`
   - 所有 step 都是 `skipped` → `skipped`
   - 任一 step 是 `failed` 或 `faulted` → `failed`（取最严重的状态）
   - 任一 step 是 `canceled` 且无 failed/faulted → `canceled`
   - 混合 `success` 和 `skipped` → `success`

4. **Fail-fast 行为**：
   - 任一 step 进入 `failed` 或 `faulted` 时，立即停止调度新 step
   - 已在 `running` 的 step 继续执行完成（不主动中断）
   - 未开始的 step 标记为 `canceled`

5. **取消传播**：
   - 编排 job 被取消时，递归取消所有 steps
   - `waiting` step → `canceled`
   - `running` step → 发送 SIGTERM，超时后 SIGKILL，最终 `canceled`

### 串行与并行规则

- 无 `depends_on`：按 yaml 顺序串行执行
- 有 `depends_on`：等指定 job 完成后执行，与其他无依赖关系的 job 并行

### 示例

```yaml
version: v1
timeout: 3600

steps:
  - name: checkout
    uses: checkout
    with:
      depth: 1
      sparse_checkout: src/backend   # monorepo 可选
    outputs:
      - commit_sha
    retry_policy: always_rerun

  - name: test
    depends_on: [checkout]
    timeout: 600
    steps:
      - name: unit-test
        image: python:{{ PYTHON_VERSION }}
        commands: [uv run pytest tests/unit]

      - name: lint
        image: python:{{ PYTHON_VERSION }}
        commands: [uv run ruff check .]

      - name: coverage
        depends_on: [unit-test]      # 与 lint 并行，等 unit-test 完成
        image: python:{{ PYTHON_VERSION }}
        commands: [uv run pytest --cov]

  - name: build
    depends_on: [test]
    steps:
      - name: registry-login
        image: docker:latest
        volumes:
          - docker-config:/root/.docker
        commands:
          - docker login {{ REGISTRY }} -u {{ REGISTRY_USER }} -p {{ REGISTRY_PASSWORD }}

      - name: build-and-push
        depends_on: [registry-login]
        image: docker:latest
        volumes:
          - /var/run/docker.sock:/var/run/docker.sock
          - docker-config:/root/.docker
        commands:
          - docker build -t {{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }} .
          - docker push {{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }}
        artifacts:
          - type: docker_image
            name: "{{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }}"
```

### 超时继承

pipeline → 父 job → 子 job，就近优先，未配置则继承上层。超时触发时容器收到 SIGTERM，超时后强制 kill，job 标记为 faulted，触发 fail-fast。

### 内置 Action：checkout

平台提供 `uses: checkout` 内置 action，负责拉取代码到 `/workspace`。

#### 接口规范

```yaml
- name: checkout
  uses: checkout
  with:
    depth: 1                          # 可选，默认 1（浅克隆），0 表示完整历史
    ref: ""                           # 可选，默认使用 trigger_ref
    sparse_checkout: ""               # 可选，monorepo 场景指定子目录（如 "src/backend"）
    submodules: false                 # 可选，是否拉取子模块
  outputs:
    - commit_sha                      # 实际拉取的 commit SHA
    - commit_message                  # commit message 首行
    - author                          # 作者名
    - committed_at                    # 提交时间（ISO 8601）
  retry_policy: always_rerun          # checkout 总是重新执行
```

#### 凭据注入

根据 Project 关联的 Credential 类型：

- **SSH 私钥**：
  - 写入临时文件 `/tmp/ssh_key_{run_id}`（权限 600）
  - 挂载到容器 `/root/.ssh/id_rsa`
  - 设置环境变量 `GIT_SSH_COMMAND="ssh -i /root/.ssh/id_rsa -o StrictHostKeyChecking=no"`
  - 容器销毁后删除临时文件

- **HTTPS Token**：
  - 方式 1（推荐）：URL 内嵌 `https://{token}@github.com/user/repo.git`
  - 方式 2：设置 `GIT_ASKPASS` 脚本返回 token
  - 环境变量不落盘，仅传递给容器

#### 执行容器

使用 `alpine/git` 镜像，挂载 `/workspace`，执行 git clone/fetch 命令。

#### Outputs 传递

checkout 完成后，平台解析 git log 提取 outputs，存储在 Job 记录中，后续 step 可通过变量引用（如 `{{ steps.checkout.outputs.commit_sha }}`）。

---

## 六、制品

制品必须在叶子 job 中显式声明，平台不做自动扫描。

两种类型：
- `docker_image`：存在 registry，记录镜像 tag
- `file`：存在文件系统，平台保留文件并提供下载接口

job 执行成功后平台写入 Artifact 记录，失败则不记录。制品流转到 CD 是后续独立设计。

---

## 七、工作目录

每次 PipelineRun 有独立 workspace：

```
data/ci/runs/{run_id}/
├── workspace/     # 挂载到容器 /workspace
└── artifacts/     # 挂载到容器 /artifacts
```

run 结束后 workspace 清理，artifacts 按保留策略保存。

---

## 八、执行层设计

执行层封装为独立的 `PipelineExecutor` 接口，schema 不感知底层实现：

- 初版：asyncio 实现，递归处理 Job 树，`depends_on` 用 `asyncio.gather` 实现并行
- 迁移 Temporal 时：只换 `PipelineExecutor` 实现，其余不变

迁移映射直接：编排 job → child workflow，叶子 job → activity，`depends_on` → workflow 顺序编排。

---

## 九、重试机制

重试创建新的 PipelineRun，携带 `retry_of` 引用原 Run：
- pipeline 定义和变量快照复用原 Run 的快照
- 平台查 `retry_of` 的 Run 决定哪些 job 可跳过

### 重试策略

```yaml
- name: checkout
  retry_policy: always_rerun      # 总是重跑

- name: build
  retry_policy: skip_if_success   # 默认
```

### 跳过传播规则

`skip_if_success`：上次成功，**且依赖链上没有任何 job 重新运行**，才跳过。

后续增强：通过 job outputs（如 `commit_sha`）实现更精确的跳过条件。

---

## 十、并发控制

全局配置 `max_concurrent_runs`，超出时排队。

项目级并发策略：

```
concurrency_policy:
  key: "{{ project_name }}/{{ trigger_ref }}"   # 同分支串行，不同分支并行
  on_conflict: cancel_running                    # | queue | skip
```

---

## 十一、取消运行

```
waiting  ──取消──► canceled
running  ──取消──► canceling ──► canceled
```

canceling 状态：停止调度新 job，向运行中容器发 SIGTERM，超时强制 kill，所有未完成 job 标记 canceled。

---

## 十二、前端可视化与日志流

Job 树天然适合渲染流程图：编排 job 是节点组，`depends_on` 是有向边，叶子 job 是可点击节点。

实时进度通过 SSE 推送 job 级别状态变更，日志每个叶子 job 独立一个 SSE 连接。SSE 成本低，FastAPI 原生支持，满足单向推送需求。

---

## 十三、事件通知

pipeline 状态变更（success / failed / canceled）触发通知，订阅配置挂在 Project 上，支持 webhook（POST JSON）和 email。

---

## 十四、TODO

- 制品文件存储：本地文件系统先做，接口抽象一层，后续可换 MinIO
- 私有镜像拉取（step image 字段）：第一版只支持公开镜像
- 精确跳过（job outputs + skip_conditions）：后续增强
- 并发 cache 写冲突：第一版不处理，后续按 cache key 隔离


---

现在审视离实施还差多少。

**设计已经完整的部分**

核心模型全部清晰：Project、Credential、PipelineTemplate、PipelineRun、Job、Artifact、Notification 都有定义。执行流程从触发到制品输出有完整路径。变量系统、重试、并发、取消、超时都有设计。

**实施前还需要确认的**

有两个地方设计上还有模糊：

1. **Job 状态机** 没有完整定义。PipelineRun 有状态，但每个 Job 自己的状态（waiting / running / success / failed / skipped / canceled）需要明确，特别是编排节点（有子 jobs 的）的状态如何从子 job 聚合。

2. **`uses: checkout` 的具体行为** 没有完全定义。平台内置 action 的接口规范（支持哪些 `with` 参数、outputs 格式）需要在实施前确定，否则 pipeline yaml 解析器写不了。

**实施顺序建议**

可以分三步走，每步都能独立交付可用的东西：

第一步（核心可跑）：Project + Credential CRUD、PipelineTemplate CRUD、手动触发 PipelineRun、Job 串行执行、日志收集、基础状态展示。

第二步（完整 CI）：Webhook 触发、并行执行、checkout 内置 action、制品记录、重试。

第三步（生产就绪）：并发控制、取消、超时、通知、前端流程图可视化。

你觉得这个顺序合理吗，还是有优先级要调整的？