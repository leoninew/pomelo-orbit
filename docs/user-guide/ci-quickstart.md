# CI 系统快速上手指南

欢迎使用 Pomelo Orbit CI 系统！本指南将帮助你在 10 分钟内创建第一个 CI 项目并成功运行。

---

## 📚 核心概念

### 1. Credential（凭据）
用于访问 Git 仓库和镜像仓库的认证信息，支持三种类型：
- **Git SSH**: SSH 私钥，用于通过 SSH 协议克隆仓库
- **Git Token**: Personal Access Token，用于通过 HTTPS 克隆仓库
- **Registry Token**: 镜像仓库的用户名和密码

### 2. Pipeline Template（模板）
定义 CI 流程的 YAML 配置，包含：
- **Steps**: 执行步骤（串行或并行）
- **Variables**: 可配置的变量（如镜像版本、仓库地址）
- **Artifacts**: 构建产物（Docker 镜像、文件）

### 3. Project（项目）
关联代码仓库和 Pipeline 模板，配置：
- 仓库地址
- Git 凭据
- 变量覆盖
- Webhook 配置

### 4. Pipeline Run（执行实例）
每次触发（手动或 Webhook）创建一个 Run，包含：
- 执行状态（waiting / running / success / failed）
- Jobs 列表
- 日志和制品

---

## 🚀 5 分钟快速开始

### 步骤 1: 创建 Git 凭据

1. 进入 **凭据管理** 页面（`/ci/credentials`）
2. 点击 **创建凭据**
3. 选择凭据类型：
   - **Git SSH**: 粘贴你的 SSH 私钥
   - **Git Token**: 粘贴 GitHub/GitLab Personal Access Token
4. 输入凭据名称（如 "GitHub SSH"）
5. 点击 **确定**

**示例 - GitHub Personal Access Token:**
```
Settings → Developer settings → Personal access tokens → Generate new token
权限: repo (Full control of private repositories)
```

### 步骤 2: 选择模板创建项目

1. 进入 **模板市场** 页面（`/ci/template-market`）
2. 浏览内置模板：
   - **Python FastAPI 构建**: 使用 uv + pytest + ruff + Docker
   - **Node.js 构建**: 使用 npm + lint + test + Docker
   - **简单构建**: 最简单的模板，适合快速上手
3. 点击模板卡片上的 **创建项目** 按钮
4. 填写项目信息：
   - **项目名称**: 如 "my-api"
   - **仓库地址**: `https://github.com/user/repo.git`
   - **Git 凭据**: 选择步骤 1 创建的凭据
   - **分支过滤**: 留空（所有分支）或填写 `main,develop`
5. 配置模板变量（根据模板要求）：
   - `REGISTRY`: 镜像仓库地址（如 `docker.io`）
   - `REGISTRY_USER`: 镜像仓库用户名
   - `REGISTRY_PASSWORD`: 镜像仓库密码
   - `IMAGE_NAME`: 镜像名称（如 `myuser/myapp`）
6. 点击 **确定**

### 步骤 3: 手动触发 Pipeline

1. 进入项目详情页面
2. 点击 **触发 Pipeline** 按钮
3. 输入触发参数：
   - **Ref**: 分支名、Tag 或 Commit SHA（如 `main`）
   - **运行时变量**: 可选，覆盖项目变量
4. 点击 **确定**

### 步骤 4: 查看执行结果

1. 自动跳转到 **Run 详情** 页面
2. 查看 Run 状态和 Jobs 列表
3. 点击 Job 名称查看实时日志
4. 等待执行完成（状态变为 `success` 或 `failed`）
5. 查看制品列表（如 Docker 镜像）

---

## 🔗 配置 Webhook 自动触发

### GitHub Webhook

1. 进入项目详情页面，复制 **Webhook URL** 和 **Secret**
2. 打开 GitHub 仓库设置：`Settings → Webhooks → Add webhook`
3. 填写配置：
   - **Payload URL**: 粘贴 Webhook URL
   - **Content type**: `application/json`
   - **Secret**: 粘贴 Webhook Secret
   - **Events**: 选择 `Just the push event` 或自定义
4. 点击 **Add webhook**
5. 推送代码到仓库，自动触发 Pipeline

### GitLab Webhook

1. 进入项目详情页面，复制 **Webhook URL** 和 **Secret**
2. 打开 GitLab 项目设置：`Settings → Webhooks`
3. 填写配置：
   - **URL**: 粘贴 Webhook URL
   - **Secret token**: 粘贴 Webhook Secret
   - **Trigger**: 勾选 `Push events`、`Tag push events`
4. 点击 **Add webhook**
5. 推送代码到仓库，自动触发 Pipeline

---

## 📝 编写 Pipeline YAML

### 基本结构

```yaml
version: v1
timeout: 3600  # 可选，默认 1 小时

steps:
  - name: checkout
    uses: checkout  # 内置 action，拉取代码
    with:
      depth: 1
    retry_policy: always_rerun

  - name: build
    depends_on: [checkout]  # 依赖 checkout 完成
    image: alpine:latest
    commands:
      - echo "Building..."
      - ./build.sh
```

### 串行执行

默认按顺序执行：

```yaml
steps:
  - name: step1
    image: alpine
    commands: [echo "Step 1"]

  - name: step2
    image: alpine
    commands: [echo "Step 2"]

  - name: step3
    image: alpine
    commands: [echo "Step 3"]
```

### 并行执行

使用 `depends_on` 实现并行：

```yaml
steps:
  - name: checkout
    uses: checkout

  - name: lint
    depends_on: [checkout]
    image: python:3.12
    commands: [ruff check .]

  - name: test
    depends_on: [checkout]  # lint 和 test 并行
    image: python:3.12
    commands: [pytest tests/]

  - name: build
    depends_on: [lint, test]  # 等待 lint 和 test 都完成
    image: docker:latest
    commands: [docker build -t myapp .]
```

### 嵌套步骤（编排节点）

```yaml
steps:
  - name: test
    steps:
      - name: unit-test
        image: python:3.12
        commands: [pytest tests/unit]

      - name: integration-test
        image: python:3.12
        commands: [pytest tests/integration]
```

### 使用变量

```yaml
steps:
  - name: build
    image: python:{{ PYTHON_VERSION }}  # 使用变量
    commands:
      - docker build -t {{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }} .
      - docker push {{ REGISTRY }}/{{ IMAGE_NAME }}:{{ trigger_ref }}
```

**内置变量:**
- `trigger_ref`: 触发的分支/Tag/Commit SHA
- `trigger_type`: `webhook` 或 `manual`
- `project_name`: 项目名称
- `run_id`: 本次 Run ID

### 声明制品

```yaml
steps:
  - name: build
    image: docker:latest
    commands:
      - docker build -t myapp:latest .
      - docker push myapp:latest
    artifacts:
      - type: docker_image
        name: "myapp:latest"
```

---

## 🔧 变量系统

### 三层变量优先级

```
全局变量（Settings）
  ↓ 被覆盖
项目级变量（Project.variable_overrides）
  ↓ 被覆盖
运行时临时变量（手动触发时传入）
```

### 在模板中声明变量

```yaml
# 在模板的 variable_declarations 中声明
[
  {
    "name": "PYTHON_VERSION",
    "description": "Python 版本",
    "required": false,
    "default": "3.12"
  },
  {
    "name": "REGISTRY",
    "description": "镜像仓库地址",
    "required": true
  }
]
```

### 在项目中配置变量

进入项目详情页面 → **变量配置** 卡片 → 添加或编辑变量

---

## 🔄 重试机制

### 重试策略

每个 Step 可以配置重试策略：

```yaml
steps:
  - name: checkout
    uses: checkout
    retry_policy: always_rerun  # 总是重新执行

  - name: build
    image: alpine
    commands: [./build.sh]
    retry_policy: skip_if_success  # 默认，上次成功则跳过
```

### 手动重试

1. 进入 Run 详情页面
2. 点击 **重试** 按钮
3. 系统创建新的 Run，引用原 Run ID
4. 根据重试策略决定哪些 Step 跳过

---

## ❓ 常见问题

### Q1: 如何查看 Job 日志？

进入 Run 详情页面 → 点击 Job 名称 → 查看日志抽屉

### Q2: Pipeline 执行失败怎么办？

1. 查看失败的 Job 日志，定位错误原因
2. 修复代码或配置后重新触发
3. 或点击 **重试** 按钮

### Q3: 如何使用私有镜像？

第一版暂不支持私有镜像拉取，请使用公开镜像。后续版本会支持配置 Registry 凭据。

### Q4: 如何配置并发控制？

暂不支持项目级并发控制，所有 Run 按触发顺序排队执行。后续版本会支持配置并发策略。

### Q5: 制品文件存储在哪里？

制品文件存储在服务器的 `data/ci/runs/{run_id}/artifacts/` 目录，可通过 API 下载。

### Q6: 如何取消正在运行的 Pipeline？

暂不支持取消功能，后续版本会添加。

---

## 📖 进阶主题

### 自定义模板

1. 进入 **模板市场** 页面
2. 点击 **自定义模板** 按钮
3. 填写模板名称、描述和 Pipeline YAML
4. 点击 **确定**
5. 在项目中选择自定义模板

### Monorepo 支持

使用 `sparse_checkout` 参数只拉取子目录：

```yaml
steps:
  - name: checkout
    uses: checkout
    with:
      sparse_checkout: src/backend  # 只拉取 backend 目录
```

### 条件执行

使用 Jinja2 条件语法：

```yaml
steps:
  - name: deploy
    image: alpine
    commands:
      - {% if trigger_ref == 'main' %}./deploy-prod.sh{% else %}./deploy-dev.sh{% endif %}
```

---

## 🆘 获取帮助

- **文档**: `docs/discussions/05-ci-design.md` - CI 系统设计文档
- **API 文档**: `/api/docs` - OpenAPI/Swagger 文档
- **问题反馈**: 提交 Issue 到项目仓库

---

**最后更新:** 2026-04-01  
**版本:** v0.6.4