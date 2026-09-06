# 流水线阶段变量默认值规范
最后修改时间: 2026-09-06 22:46:22

Review status: Accepted

Mode: standard

## Background

流水线阶段基于 Go Liquid 模板引擎。当前运行时渲染支持 Liquid 变量和 `default` filter，但阶段变量默认值的识别、详情展示和多阶段作用域语义仍需明确收敛。历史阶段文本中可能存在 `${NAME:-default}` 写法；本任务不新增兼容解析，而是统一使用 Liquid 规范。

单阶段流水线可以通过补充一个全局 Pipeline 变量暂时运行，但多阶段流水线可能在不同阶段使用同名变量而需要不同默认值。例如前端阶段和后端阶段都使用 `working_dir`，但默认目录分别为 `frontend` 与 `backend`；把它们合并为一个全局默认值会使其中一个阶段错误。

本任务先确定唯一的 Liquid 默认值规范，再统一阶段变量识别、详情展示和运行时解析，使单阶段与多阶段流水线遵循同一规则。

## Goal

1. 为流水线阶段文本定义唯一、可验证的变量引用和默认值语法。
2. 让阶段脚本和阶段制品模板中的默认值被变量管理识别，并保留其所属阶段和出现位置。
3. 运行多阶段流水线时，在未配置全局覆盖值的情况下使用各阶段自身的默认值。
4. 保持 Repository、Application Pipeline 和 Stage 默认值的既有优先级，并将最终执行值保存到 Run 快照。
5. 对缺失、非法或不支持的表达式给出明确校验错误，不静默保留未解析文本。

## Non-goal

- 不同时支持多套阶段默认语法；不新增 `${NAME:-default}` 或其他非 Liquid 默认语法的兼容回退。
- 不改变 Repository 变量、Pipeline 变量、Snapshot、PipelineRun 或 Version 的持久化职责。
- 不把不同阶段的同名默认值强行持久化为一个 Pipeline 全局默认值。
- 不改变流水线构建镜像、Artifact 收集或成功后 fork Version 的业务流程。
- 不增加新的数据库表或迁移；阶段覆盖值复用现有 `pipeline.variable_declarations` JSON，并通过 `stage_id` 表达作用域。

## Default value specification

### 唯一语法

流水线阶段变量统一采用 Liquid：

```liquid
{{ NAME }}
{{ NAME | default: "default" }}
```

其中：

- `NAME` 沿用当前 Pipeline 变量命名规则：以字母开头，后续使用字母、数字或下划线。
- `{{ NAME }}` 表示必填变量；没有有效值时，所属阶段不能执行。
- `{{ NAME | default: "default" }}` 表示当 `NAME` 未设置或值为空字符串时使用字面量默认值。
- `default` 是该表达式当前位置的字面量；不进行嵌套变量展开、表达式求值或二次模板解析。
- 默认值按 Liquid 字面量规则书写，支持单引号或双引号；不隐式 trim。
- 只支持 Liquid `default` filter 作为流水线阶段默认值；不支持 `${NAME:-default}` 或其他非 Liquid 默认运算符。

### 适用范围

该规范适用于 Application Pipeline 阶段中的：

- `script`
- Artifact 的 `reference`
- Artifact 的 `name`
- Artifact 的 `command`

变量提取和运行时渲染必须使用同一套语法规则，不能由详情展示、Run 创建和 Stage 执行分别实现不一致的正则或 fallback 行为。

### 优先级与阶段作用域

同名变量的有效值按以下顺序处理：

```text
Repository 全局 value/default
  -> Application Pipeline 当前 Stage 的 value/default
  -> Application Pipeline 全局 value/default
  -> 当前表达式所在阶段的 default
```

阶段默认值属于变量的具体出现位置，而不是全局 Pipeline 配置。例如：

```liquid
# frontend stage
cd {{ working_dir | default: "frontend" }}

# backend stage
cd {{ working_dir | default: "backend" }}
```

在没有 Repository 或 Pipeline 配置时，前者解析为 `frontend`，后者解析为 `backend`。带 `stage_id` 的 Pipeline `working_dir` 只覆盖对应 Stage；不带 `stage_id` 的 Pipeline 配置是全局值，覆盖所有 Stage。

变量管理按 `(变量名, 来源 Stage)` 展示阶段条目；全局自定义变量另行展示。不能通过编辑一个 Stage 条目覆盖其他 Stage，也不能因为同名变量出现多次而丢弃来源信息。

如果一个变量的所有出现位置都有默认值，则未配置 Pipeline 值不应被视为缺失；如果某个出现位置使用 `{{ NAME }}` 且没有更高优先级值，运行创建或阶段解析必须报告变量名和阶段名称。

## User scenarios

### 单阶段默认值

阶段脚本使用：

```liquid
cd {{ working_dir | default: "." }}
docker build -f {{ repository_dockerfile | default: "Dockerfile" }} .
```

流水线详情应识别 `working_dir` 和 `repository_dockerfile`，并展示各自的阶段默认值。没有 Pipeline 配置时，运行使用 `.` 和 `Dockerfile`。

### 前后端多阶段默认值

前端和后端阶段都使用 `working_dir`，但默认值分别为 `web` 与 `webapi`。详情变量管理展示两个 `working_dir` 条目，并标明 `frontend`、`backend` 来源；运行时两个阶段分别使用自己的默认值。对前端条目设置值 `web-dist` 时，只覆盖前端；后端仍使用 `webapi`。

### Pipeline 覆盖

用户为 `repository_dockerfile` 保存 Pipeline 值 `Dockerfile.prod`。后续所有引用该变量的阶段都使用 `Dockerfile.prod`，并覆盖对应表达式的阶段默认值；删除该 Pipeline 配置后，各阶段恢复使用自己的默认值。

### 缺少必填变量

阶段使用 `{{ IMAGE_TAG }}`，Repository 和 Pipeline 均没有有效值。运行创建或执行前校验失败，错误至少包含 `IMAGE_TAG` 和所属阶段；系统不能把未解析的 `{{ IMAGE_TAG }}` 原样交给 shell 后继续执行。

### 制品模板变量

Artifact 的 `reference` 使用 `{{ repository_code }}:{{ runtime_datetime }}`，Artifact 的其他文本字段使用 `{{ IMAGE_NAME | default: "app" }}`。变量提取、运行时解析和 Artifact 收集必须得到与阶段脚本一致的结果。

## Acceptance

- [ ] 需求文档确定唯一默认语法为 Liquid `{{ NAME | default: "default" }}`，必填引用为同一语法族中的 `{{ NAME }}`；不实现 `${NAME:-default}` 或其他默认运算符兼容路径。
- [ ] 阶段 `script`、Artifact `reference`、`name`、`command` 中的 `{{ NAME }}` 与 `{{ NAME | default: "default" }}` 均能被统一识别。
- [ ] 流水线详情变量管理能识别 `repository_dockerfile`、`working_dir` 等阶段变量，并展示来源阶段及其默认值。
- [ ] 同名变量在多个阶段拥有不同默认值时，按来源 Stage 生成独立条目；详情展示不丢失信息。
- [ ] 多阶段运行在没有 Repository/Pipeline 覆盖时，分别使用各阶段默认值；不会要求用户填写一个无法同时满足不同 Stage 的全局值。
- [ ] Repository 全局值、Pipeline Stage 值和 Pipeline 全局值按作用域优先级覆盖阶段默认值；空值遵循 Liquid `default` 的 fallback 语义。
- [ ] Pipeline 阶段覆盖带有 `stage_id`，只对对应 Stage 生效；全局 Pipeline 值不带 `stage_id`。
- [ ] 没有默认值且没有高优先级有效值的 `{{ NAME }}` 在运行前或阶段解析时失败，并包含变量名和阶段上下文。
- [ ] 非法或不支持的表达式不会静默执行，返回可读的 validation/rendering error。
- [ ] 新增测试覆盖单阶段默认值、多阶段同名变量不同默认值、Pipeline 覆盖、空值 fallback、必填变量缺失及 Artifact 字段解析。

## Decisions

1. 采用 Liquid `{{ NAME | default: "default" }}` 作为唯一默认值规范。项目当前阶段渲染已经基于 Liquid，默认值由同一模板引擎解释。
2. `{{ NAME }}` 与 `{{ NAME | default: "default" }}` 属于同一 Liquid 变量规范；前者表达必填，后者表达可回退，不视为两套模板实现。
3. 阶段解析结果按 `(name, stage_id)` 保存；运行时同时维护全局和 Stage 作用域，详情与快照使用同一来源标识。
4. Pipeline 阶段覆盖值直接复用已有 JSON 配置并增加 `stage_id` 字段，不引入数据库迁移或第二套配置存储。
5. 不保留 `${NAME:-default}` 的阶段默认值兼容路径。当前阶段文本若需要默认值，必须显式改为 Liquid `default` filter 形式。

## Open questions

暂无需要用户确认的未决事项。Plan 阶段需要落实默认表达式解析器在模板渲染、变量提取和运行时校验之间的复用边界，但不改变本 Requirement 已确定的语法和语义。

## Risk

- 现有 Application Pipeline 阶段文本中可能存在 `${NAME:-default}` 写法；按本任务的“不做兼容”约束，这些文本需要由维护流程显式改写为 Liquid 形式，不能由运行时静默转换。
- 当前变量声明模型按变量名聚合，若直接复用单一 `default` 字段会重新引入多阶段默认值互相覆盖的问题；实现必须保留 occurrence/stage 维度，或采用等价的不可丢失表示。
- 阶段脚本由 shell 执行，但 Artifact 字段通常由 Orbit 在宿主侧解析；两者必须共用同一语法实现，否则可能出现脚本成功而 Artifact 引用无法收集的分裂行为。
- 默认值包含空格、引号或 `}` 时的语法边界需要在 Spec 阶段固化解析规则和错误信息，避免不同执行镜像的 shell 行为产生差异。

## User review notes

用户已确认 Liquid 唯一语法、空值 fallback 语义和多阶段同名变量的阶段级默认值展示方式，并授权进入 Implementation 与 Verification。
