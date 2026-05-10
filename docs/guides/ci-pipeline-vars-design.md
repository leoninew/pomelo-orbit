# CI Pipeline 变量设计

**日期**: 2026-04-11
**状态**: 已落地

## 目标

统一模板编辑、仓库配置、触发执行三段流程的变量规则：

- 所有变量最终都必须有值（`template_stage` 有 default 值时豁免强制校验）
- `secret` 只用于脱敏展示和日志脱敏
- 内置变量（全局 + 仓库 + 模板）由系统注入，不接受持久化覆盖
- Stage 引用的非内置变量自动补充到模板变量声明，支持从 `default(...)` 提取默认值
- 快照只存储 Stage 中实际用到的变量，减少冗余

---

## 变量来源分类

| source | 含义 | 可编辑 | 可持久化 |
|--------|------|--------|----------|
| `global` | 全局内置变量，来自配置 `ci.global_variables` | 否 | 否 |
| `repository` | 仓库内置变量，运行时注入 | 否 | 否 |
| `repository_custom` | 仓库自定义变量 | 是（仓库详情页） | 是 |
| `template` | 模板内置变量，运行时注入 | 否 | 否 |
| `template_stage` | Stage 脚本中提取的变量 | 是（模板详情页 + 触发弹窗） | 有值时升级为 `template_custom` |
| `template_custom` | 模板自定义变量 | 是（模板详情页） | 是 |

### template_stage 的持久化规则

`template_stage` 变量由 Stage 脚本自动提取，不直接持久化。当用户在模板详情页为其设置了值并保存时，该变量会被升级为 `template_custom` 存入数据库。下次加载模板时，`merge_declarations` 会识别到 `existing_map` 中已有该变量（source 已是 `template_custom`），保留用户设置的值，同时 Stage 脚本中的 `{{ working_dir | default('.') }}` 仍能正常提取并展示。

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
内置变量（全局 + 仓库 + 模板）  ← 系统注入，不接受覆盖
    ↓
运行时覆盖（触发时传入，不能覆盖内置变量）
    ↓
仓库自定义变量（repository_custom）
    ↓
模板自定义变量默认值（template_custom）
    ↓
Stage 提取变量默认值（template_stage）
```

**说明**：运行时覆盖优先级高于仓库自定义变量，确保用户在触发时传入的值能生效，不被仓库配置静默覆盖。

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
- Stage 提取的新变量标记为 `template_stage`，**可编辑**（用户可为其设置覆盖值）
- 已有自定义声明的变量保留元数据（description、secret、value）
- 支持从 `{{ VAR | default('x') }}` 提取默认值

前端通过 `POST /api/ci/template/resolve-variables` 实时获取变量列表。

### 触发执行（TriggerModal）

触发弹窗中展示模板的 `variable_declarations`（经 `resolve_template_variables` 计算后的完整列表）：

- `template` / `repository` 来源：内置变量，禁用输入
- `template_stage` 来源：Stage 变量，**可输入覆盖**，显示 `Stage` badge，default 值作为初始值
- `template_custom` / `repository_custom` 来源：自定义变量，可输入

触发时只发送用户修改过的变量（与初始值不同的），后端按优先级合并。

---

## 变量校验规则

`validate_variables` 对 `template.variable_declarations` 中的变量进行校验：

- `template_custom`：必须有值，否则报错
- `template_stage`：有 default 值时豁免（default 已在合并阶段填入），无 default 值时必须有运行时值
- 内置变量（`global`、`repository`、`template`）：由系统保证有值，不参与用户侧校验

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
    default: Any = None   # 系统/脚本提供的原始默认值，只读，用于展示和还原
    value: Any = None     # 用户的显式覆盖值，None 表示"未覆盖，运行时用 default"
    secret: bool = False
    source: VariableSource = VariableSource.TEMPLATE_CUSTOM
    editable: bool = True  # 展示层属性，由 VariableResolver 根据 source spec 设置
```

- `default`：来自 stage 脚本 `{{ VAR | default('x') }}` 提取，或内置变量的运行时注入值
- `value`：用户的显式覆盖，`None` 表示未覆盖，运行时取值逻辑为 `value ?? default`
- `editable`：由 `VariableResolver` 在返回时根据 source spec 设置，不参与持久化语义判断

是否只读由 `editable` 字段决定（由 spec 驱动），不使用 `required`、`locked`、`builtin` 等额外字段。

---

## 兼容策略

- 历史模板 JSON 里的 `required`、`locked` 等旧字段读取时忽略
- 历史仓库/模板变量里如果残留内置变量名，服务层保存前通过 `sanitize_variable_overrides` 自动过滤
- `template_stage` 变量有值时，`sanitize_variable_overrides` 自动升级为 `template_custom` 持久化
