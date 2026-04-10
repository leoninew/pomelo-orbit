# CI Pipeline 变量设计

**日期**: 2026-04-10
**状态**: 已落地

## 目标

统一模板编辑、仓库配置、触发执行三段流程的变量规则：

- 所有变量最终都必须有值
- `secret` 只用于脱敏展示和日志脱敏
- 内置变量（全局 + 仓库 + 模板）不可覆盖
- Stage 引用的非内置变量自动补充到模板变量声明，支持从 `default(...)` 提取默认值
- 快照只存储 Stage 中实际用到的变量，减少冗余

---

## 变量来源分类

| source | 含义 | 可编辑 |
|--------|------|--------|
| `global` | 全局内置变量，来自配置 `ci.global_variables` | 否 |
| `repository` | 仓库内置变量，运行时注入 | 否 |
| `repository_custom` | 仓库自定义变量 | 是（仓库详情页） |
| `template` | 模板内置变量，运行时注入 | 否 |
| `template_stage` | Stage 脚本中提取的变量 | 是（模板详情页） |
| `template_custom` | 模板自定义变量 | 是（模板详情页） |

---

## 内置变量

### 全局内置变量

来自配置文件 `ci.global_variables`，注入所有 pipeline run：

```yaml
ci:
  global_variables:
    PROJECT_NAME: "Pomelo Orbit"
```

### 仓库内置变量

代码中固定定义，运行时从仓库实体取值：

| 变量名 | 值来源 |
|--------|--------|
| `repository_id` | 仓库 ID |
| `repository_name` | 仓库名称 |
| `repository_url` | 仓库地址 |
| `repository_ref` | 本次触发的 ref（分支/tag） |

仓库详情页展示时，`repository_ref` 显示为 `default_branch`。

### 模板内置变量

代码中固定定义，运行时从模板/快照取值：

| 变量名 | 值来源 |
|--------|--------|
| `template_id` | 模板 ID |
| `template_name` | 模板名称 |
| `template_version` | 快照版本号 |

---

## 变量优先级（运行时合并）

高 → 低：

```
内置变量（全局 + 仓库 + 模板）  ← 不可覆盖
    ↓
仓库自定义变量（repository_custom）
    ↓
模板声明默认值（template_custom / template_stage）
    ↓
运行时临时覆盖（触发时传入，不能覆盖内置变量）
```

---

## 各场景变量列表规则

### 仓库详情页

展示：仓库内置变量 + 仓库自定义变量

- 内置变量只读，`repository_ref` 显示 `default_branch`
- 自定义变量可增删改

### 模板详情页 / 编排预览

展示：所有内置变量（仓库 + 模板）+ Stage 提取变量 + 自定义变量

规则：
- 无 Stage 时：显示所有内置变量 + 所有自定义变量
- 有 Stage 时：只显示 Stage 中实际引用的变量（内置或自定义）
- 内置变量只读，`source=template`
- Stage 提取的新变量标记为 `template_stage`
- 已有自定义声明的变量保留元数据（description、secret、value）
- 支持从 `{{ VAR | default('x') }}` 提取默认值

前端通过 `POST /api/ci/template/resolve-variables` 实时获取变量列表。

### 触发执行

1. 按优先级合并所有来源的变量
2. 校验所有模板自定义变量都有值
3. 内置变量不接受用户覆盖
4. 合并结果脱敏后存入 `PipelineRun.variables_snapshot`

---

## 快照变量存储

`PipelineSnapshot.variables_snapshot` 只存储 Stage 中实际用到的变量，不存储未引用的内置变量。

`PipelineRun.variables_snapshot` 存储本次运行的实际变量值（脱敏后），source 反映真实来源：

- 仓库内置变量 → `source=repository`
- 仓库自定义变量 → `source=repository_custom`
- 模板内置变量 → `source=template`
- 模板自定义/stage 变量 → 保持原 source

---

## 变量模型

```python
class VariableDeclaration(BaseModel):
    name: str
    description: str = ""
    value: Any = None
    secret: bool = False
    source: VariableSource = VariableSource.TEMPLATE_CUSTOM
```

不再使用 `required`、`locked`、`builtin` 字段。是否只读由 `source` 决定。

---

## 兼容策略

- 历史模板 JSON 里的 `required`、`locked` 等旧字段读取时忽略
- 历史仓库变量里如果残留内置变量名（如 `repository_id`），服务层保存前自动过滤
- 历史模板变量里如果残留内置变量名，服务层保存前自动过滤
