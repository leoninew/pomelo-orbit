# CI Phase 6 内置模板初始化完成

**文档编号：** 09  
**创建日期：** 2026-04-01  
**当前阶段：** Phase 6 - 内置模板初始化完成

---

## 📊 Phase 6 完成内容

### 内置模板初始化 ✅

创建了 3 个内置 Pipeline 模板，通过数据库迁移自动初始化：

#### 1. Python FastAPI 构建

**ID:** `01KN3T0000000000000000TMPL`

**功能:**
- 使用 uv 管理依赖
- 并行执行 lint (ruff + mypy) 和单元测试 (pytest)
- 构建并推送 Docker 镜像到指定 Registry

**变量声明:**
- `PYTHON_VERSION` (可选，默认 3.12)
- `REGISTRY` (必填) - 镜像仓库地址
- `REGISTRY_USER` (必填) - 镜像仓库用户名
- `REGISTRY_PASSWORD` (必填) - 镜像仓库密码
- `IMAGE_NAME` (必填) - 镜像名称

**Pipeline 结构:**
```
checkout
  └─> test (并行)
       ├─> lint (ruff + mypy)
       └─> unit-test (pytest)
  └─> build
       ├─> registry-login
       └─> build-and-push (Docker)
```

#### 2. Node.js 构建

**ID:** `01KN3T0000000000000001TMPL`

**功能:**
- 使用 npm 管理依赖
- 并行执行 lint 和单元测试
- 构建并推送 Docker 镜像

**变量声明:**
- `NODE_VERSION` (可选，默认 20)
- `REGISTRY` (必填)
- `REGISTRY_USER` (必填)
- `REGISTRY_PASSWORD` (必填)
- `IMAGE_NAME` (必填)

**Pipeline 结构:**
```
checkout
  └─> test (并行)
       ├─> install (npm install)
       ├─> lint (依赖 install)
       └─> unit-test (依赖 install)
  └─> build
       ├─> registry-login
       └─> build-and-push (Docker)
```

#### 3. 简单构建

**ID:** `01KN3T0000000000000002TMPL`

**功能:**
- 最简单的 CI 模板
- 仅包含 checkout 和自定义构建步骤
- 适合快速上手或自定义构建流程

**变量声明:**
- `BUILD_IMAGE` (必填，默认 alpine:latest) - 构建镜像
- `BUILD_COMMAND` (必填，默认 echo 'Build completed') - 构建命令

**Pipeline 结构:**
```
checkout
  └─> build (自定义命令)
```

---

## 🔧 技术实现

### 迁移文件

**文件:** `backend/migrations/v0.6.4__ci_init_templates.json`

使用 JSON 格式迁移，通过 `INSERT` 操作批量插入 3 个模板：
- 所有模板标记为 `is_builtin=1`（内置模板）
- 使用固定 ULID 作为 ID（便于引用和测试）
- 变量声明存储为 JSON 字符串
- Pipeline 内容使用 YAML 格式（支持 Jinja2 变量）

### 迁移系统修复

在执行过程中修复了几个迁移问题：

1. **v0.6.1 (webhook)** - 列已存在，注释掉 ALTER TABLE 语句
2. **v0.6.3 (retry)** - SQLite 不支持 `ALTER TABLE ADD CONSTRAINT`，移除外键约束语句
3. 手动标记已生效的迁移到 `__migration_history` 表

---

## ✅ 验证结果

```bash
$ uv run python verify_templates.py
=== Pipeline Templates ===
  🔒 Python FastAPI 构建 (ID: 01KN3T0000000000000000TMPL)
  🔒 Node.js 构建 (ID: 01KN3T0000000000000001TMPL)
  🔒 简单构建 (ID: 01KN3T0000000000000002TMPL)

总计: 3 个模板
内置模板: 3 个
```

---

## 🎯 下一步计划

### Phase 7: UI 增强和文档（建议）

**前端增强:**
- 模板市场页面（展示内置模板，支持一键创建项目）
- Pipeline YAML 编辑器增强（语法高亮、自动补全、变量提示）
- Job 树可视化（依赖关系图）
- 实时日志推送（SSE）

**文档完善:**
- 用户指南（如何配置 Webhook、如何编写 Pipeline YAML）
- 内置模板使用说明
- API 文档更新（OpenAPI/Swagger）
- 部署配置示例

**测试增强:**
- 端到端测试（完整 Pipeline 执行流程）
- 内置模板测试（验证 YAML 语法和变量声明）

---

## 📚 相关文件

### 新增文件
- `backend/migrations/v0.6.4__ci_init_templates.json` - 模板初始化迁移
- `backend/fix_migrations.py` - 迁移修复脚本（临时）
- `backend/verify_templates.py` - 模板验证脚本（临时）

### 修改文件
- `backend/migrations/v0.6.1__ci_webhook.sql` - 注释掉重复的列添加
- `backend/migrations/v0.6.3__ci_retry.sql` - 移除 SQLite 不支持的外键约束

---

**最后更新：** 2026-04-01  
**更新人：** Kiro AI Assistant
