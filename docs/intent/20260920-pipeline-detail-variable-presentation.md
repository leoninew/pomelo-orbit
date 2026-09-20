# 流水线详情的阶段与变量展示
最后修改时间: 2026-09-20 22:11:47

Review status: Accepted

## Background

Pipeline 详情当前将内置上下文、全局配置和按 Stage 提取的变量放入同一张变量表。构建阶段列表需要继续承载名称、执行镜像、版本和依赖等自身信息；制品是 Stage 的显式输出声明，不应被变量展示替代或从构建命令推断。镜像构建模板中的 `image_name`、`repository_dockerfile`、`working_dir` 等变量，也必须能明确关联到其使用的构建阶段。

现有 `GET /api/pipeline/:pipeline_id` 的 `variable_declarations` 会返回阶段变量的 `stage_id`、`stage_name`，但没有稳定的变量类型字段，也不返回变量引用的 Stage 字段位置。`source` 混合了持久化来源和运行期内部语义，前端不能用它推断类型。详情查询也不会读取绑定 Repository 的当前自定义变量，Repository 变量只在创建 Run 时加入解析。若继续由前端重解析脚本、制品 Liquid 或拼接 Repository 接口，会形成不可靠的展示逻辑。

Application Pipeline 固定绑定 Repository 身份。每次创建 Run 时，服务端读取该 Repository 当前的地址、编码、默认分支和自定义变量；`repository_ref` 可由 Trigger 临时覆盖；最终有效值保存到 Run 快照。这个运行时语义不应因详情页展示调整而改变。

## Goal

- 让用户在 Pipeline 详情中快速识别 Stage 的依赖、制品输出和变量输入。
- 将只读运行时上下文、可配置的全局变量、按 Stage 覆盖的变量明确分开。
- 对多个镜像制品、共享同一 Repository 而目录不同的前后端 Stage，保持紧凑且可比较的展示。
- 让配置页面展示配置值和解析来源，让 Run 详情展示该次 Run 的已冻结实际值。
- 将 Application Pipeline 绑定 Repository 的当前自定义变量直接列入全局变量；它们仍只能在 Repository 管理。
- 使所有变量查询接口按稳定类型、作用域、阶段绑定和引用位置输出公共变量读模型；各变量页面复用同一种分类与关联展示，前端不解析 Liquid 或内部 `source` 字符串来分类和关联。

## Non-goal

- 不新增 Project 级变量，也不改变 Repository 变量在每次创建 Run 时读取当前值的规则。
- 不把 `repository_ref` 改为持久化的 Pipeline 配置；它仍是 Trigger 前的单次运行输入。
- 不从 Shell 或 Docker 命令语义自动推断制品。制品继续由 Stage 的显式 `ArtifactConfig` 声明。
- 不改变变量解析优先级、Liquid 语法、Stage DAG 或制品收集协议。
- 不将 Repository 当前值复制为 Pipeline 持久化配置，也不在 Pipeline 页提供 Repository 变量编辑入口。
- 不将详情页的计划使用关系伪装为 Run 的实际值；实际值和实际制品仍以 Run 快照为准。

## User Scenarios

1. 模板维护者查看镜像构建模板，能看出 Docker build Stage 使用哪些参数，并知道它会收集一个或多个 Docker 镜像制品。
2. 应用流水线维护者在“全局变量”中看到当前绑定 Repository 的变量和全局 Pipeline 配置；Repository 变量明确为只读、每次 Run 读取当前值。
3. 应用流水线维护者在“阶段变量”中按 Stage 查看前端、后端分别使用的变量、Liquid default、全局继承和阶段覆盖，不会误以为阶段覆盖会作用于另一个 Stage。
4. 运行者触发流水线时仅临时修改 `repository_ref`；详情页不会把 `runtime_datetime` 或 Repository 当前值伪装成可持久化配置。
5. 事故追溯时，用户进入 Run 详情查看已冻结的变量和制品，而非用当前 Pipeline 页面反推历史值。

## Recommended Presentation

### Stage Area

- 保留列表与 DAG 两种视图以及现有的 Stage 信息列。变量管理不替代或挤占 Stage 的名称、执行镜像、版本、依赖和操作。
- 制品继续作为每个 Stage 的显式 `ArtifactConfig` 声明展示和维护；Docker 制品在 Application Pipeline 中可显示 Component 映射。制品声明并非变量的附属信息。
- Stage 名称继续链接到来源阶段详情；来源阶段详情负责展示和维护脚本、显式制品声明及其原始 Liquid 表达式。
- Stage 变量统一在独立的变量管理卡片中查看，按 Stage 分组或筛选，不把所有变量塞入构建阶段主表。
- 多个镜像制品保持为同一 Stage 下的多个显式声明；建议使用稳定的制品名称，例如 `frontend-image`、`backend-image`，动态镜像地址只放在 `reference`。

### Variable Area

- 变量管理使用一个卡片和两个 Tab：“全局变量”和“阶段变量”。Tab 是作用域切换，不以嵌套卡片重复承载同一数据。
- “全局变量”直接列出当前绑定 Repository 的自定义变量、全局 Pipeline 配置、`repository_ref` 和系统上下文。Repository 变量为只读且标识“创建 Run 时读取当前值”；`repository_ref` 标识“默认分支，可在运行前修改”；时间戳标识“启动时生成”。
- “阶段变量”按 Stage 分组或按选中的 Stage 筛选。每行显示变量类型、绑定阶段、引用位置、Stage Liquid default、继承的全局配置和该 Stage 覆盖。
- 类型与来源使用面向用户的语义名称，不直接显示 `pipeline`、`pipeline_stage`、`runtime`、`system` 等内部枚举值。

### Variable Classification And Read Model

所有变量查询接口的变量列表应是服务端组装的读模型，而非持久化 `VariableDeclaration` 的直接映射。变量分类以管理归属和生命周期为准，采用以下稳定的 `kind`：

| `kind` | 语义 | 默认作用域与操作 |
|---|---|---|
| `repository_variable` | 绑定 Repository 的自定义变量 | `global`；在 Pipeline 中只读，每次 Run 读取当前值 |
| `pipeline_variable` | Pipeline 持久化配置 | `global`；可在 Pipeline 中管理 |
| `stage_variable` | 当前 Stage 脚本或制品声明的 Liquid 输入 | `stage`；可显示同名全局继承和该 Stage 覆盖 |
| `runtime_input` | 仅本次 Trigger 可提交的输入，如 `repository_ref` | `global`；不作为 Pipeline 持久化配置 |
| `system_context` | 服务端由 Repository 身份或运行环境提供的只读上下文，如 `repository_code`、`repository_url` | `global`；不可配置 |
| `system_generated` | Run 创建时生成的只读值，如 `runtime_datetime` | `global`；不可配置 |

每个条目还必须独立表达：

- `scope`：`global` 或 `stage`，禁止由空 `stage_id` 让前端猜测。
- `stage_binding`：阶段作用域条目的 `{ stage_id, stage_name }`；全局条目为空。
- `references[]`：变量被使用的 `{ stage_id, stage_name, field }`。`field` 至少区分 `script`、`artifact_reference`、`artifact_name`、`artifact_command`，必要时包含制品稳定名或序号。
- 面向配置展示的全局配置、阶段覆盖、Liquid default、当前条目的值来源和可编辑性。敏感值继续遵守既有脱敏规则。

同名全局变量被多个 Stage 使用时，在全局 Tab 保持一个全局条目，并通过 `references[]` 列出使用阶段；每个 Stage 在阶段 Tab 有自己的 `stage_variable` 条目。阶段覆盖是该 Stage 变量的配置状态，而不是靠前端将同名全局行复制或重新解析得出。

### Template, Application, and Run

- Template Pipeline 展示变量声明、Stage Liquid default 和制品声明，不展示不存在的具体 Repository 当前值。
- Application Pipeline 展示可编辑的配置值与将于 Run 创建时解析的上下文；Repository 自定义变量仍在 Repository 详情维护，不在这里复制编辑入口。
- Pipeline Run 详情展示 `variables_snapshot` 和已收集 Artifact 的实际引用、镜像 ID 与 source commit，作为唯一的历史事实来源。

## Acceptance

- 构建阶段列表保留 Stage 原有信息；制品仍作为独立的显式声明可识别，变量区不会取代其展示。
- Application Pipeline 的全局变量列表包含当前绑定 Repository 的自定义变量，并以只读、动态读取语义显示。
- `image_name`、`repository_dockerfile`、`working_dir` 等 Stage 变量与 `docker build` Stage 及具体字段明确关联；前端和后端使用同名变量时可分别配置和比较。
- Repository、Pipeline、Pipeline Snapshot 和 Pipeline Run 的变量条目均显式提供 `kind`、`scope`、`stage_binding`、`references`；前端不重解析脚本、制品或内部 `source` 来补齐这些信息。
- 用户不会在 Pipeline 详情把 `repository_code`、`repository_url`、`runtime_datetime` 当作可编辑配置，也能理解 `repository_ref` 仅对本次 Trigger 可改。
- 当前 Repository 修改后，新 Run 使用更新值；已创建 Run 在详情中仍显示创建时冻结的变量和制品。
- 详情页在桌面和移动宽度下保持 Stage、变量名和值可扫描，不依赖冗长的页面说明文字。

## Open Questions

暂无需要用户确认的未决事项。

## Decisions

- `repository_ref` 保持为 Trigger 前的运行时输入，不转为持久化变量。
- Repository 变量不是 Project 变量；其当前值在每次创建 Run 时参与解析，并由 Run 快照保证历史可追溯。
- 制品采用显式声明和 collector，不引入对构建脚本的语义推断。
- Application Pipeline 的详情变量读模型直接包含绑定 Repository 的当前自定义变量，归入全局变量，但不提供 Repository 变量编辑或 Pipeline 覆盖。
- Template Pipeline 未绑定具体 Repository，因此不输出某个 Repository 的当前自定义变量；其变量视图只说明可引用的配置和系统上下文结构。
- `kind`、`scope`、`stage_binding` 和 `references` 是所有变量响应的公开语义契约；内部 `source` 不再承担前端分类或关联责任。

## Risk

- Repository 变量在详情读取与 Run 创建之间可能改变，因此详情只能表示“当前值/当前声明”；历史值必须继续从 Run 快照读取。
- 对 Secret 的详情输出必须保持脱敏，不能因合并 Repository 变量而暴露原值。
- 将动态表达式用于制品稳定名称会使 Component 映射和历史检索难以识别，展示与校验都应优先稳定制品名。
