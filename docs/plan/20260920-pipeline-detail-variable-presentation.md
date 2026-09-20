# 流水线详情的阶段与变量展示实施计划
最后修改时间: 2026-09-20 22:11:47

Review status: Accepted

Mode: standard

## Intent Basis

本计划落实 [Intent](../intent/20260920-pipeline-detail-variable-presentation.md)。目标是让 Repository、Pipeline、Pipeline Snapshot 和 Pipeline Run 统一以服务端结构化读模型输出变量类型、作用域、阶段绑定与引用位置，Application Pipeline 将绑定 Repository 的当前自定义变量直接纳入全局变量。构建阶段和制品继续按既有独立结构展示与维护。

## Design Boundary

1. `VariableDeclaration` 继续是 Repository、Pipeline 持久化配置和 Run/Snapshot 历史声明的通用存储结构；不为展示直接扩展或重定义其 `source`。
2. 在 `common.proto` 定义公共 `VariableResp` 读模型，Repository、Pipeline、Pipeline Snapshot 和 Pipeline Run 的响应统一使用 `variables`。`RepositoryUpdateReq`、`PipelineCreateReq`、`PipelineUpdateReq` 中的 `variable_declarations` 写入请求保持为持久化配置输入，不将只读变量、引用或派生状态回传保存。
3. 读模型使用稳定字符串值而不是让前端解释内部 `source`：`kind` 为 `repository_variable`、`pipeline_variable`、`stage_variable`、`runtime_input`、`system_context`、`system_generated`；`scope` 为 `global` 或 `stage`。
4. 每个 Stage 变量条目显式输出 `stage_binding` 和 `references[]`；每个引用位置携带来源 Stage、字段类型、制品标识/序号及该位置的 Liquid default。避免将同名变量在不同字段或不同 Stage 的多个 default 压缩为一个不明确值。
5. 读模型为 Stage 变量直接表达全局 Pipeline 配置、Stage 覆盖、Repository 命中和 Liquid default 的配置/取值来源。前端只渲染这些字段，不通过变量同名、`source` 或文本重解析自行拼接。
6. Application Pipeline 在详情读取时加载已绑定 Repository 的当前变量。Template Pipeline 没有绑定 Repository，不返回某个 Repository 的变量；无绑定 Repository 的 Application Pipeline 保留可引用的运行时/系统变量定义，但没有 Repository 自定义条目。
7. Secret 仍沿用现有详情输出与前端脱敏规则；本变更不改变加密、权限和运行时变量优先级。Repository 在详情后发生修改时，新 Run 仍读取新值，历史 Run 仍读取快照。

## Response Shape

在 `proto/orbit/v1/common/common.proto` 定义公共变量响应消息，生成 Go/TypeScript 类型。精确字段名可在实现中按项目 Proto 命名规则校准，但语义必须包含：

| 内容 | 用途 |
|---|---|
| `VariableResp.name`、`description`、`secret`、`editable` | 变量身份和展示/操作元数据 |
| `kind`、`scope` | 稳定分类和 Tab 归属 |
| `stage_binding` | 阶段变量的 `{ stage_id, stage_name }`，全局变量为空 |
| `references[]` | 使用变量的 Stage、字段类型 `script` / `artifact_reference` / `artifact_name` / `artifact_command`、制品名称或索引，以及该位置的 Liquid default |
| `configuration` | Repository 或全局 Pipeline 等主配置的 `value`、`default`、`description`、`secret` 与可编辑状态 |
| `global_configuration`、`stage_override` | Stage 变量在页面中直接展示的全局 Pipeline 配置和本 Stage 覆盖，不依赖前端 join |
| `value_source` | 当前配置视图中该 Stage 变量的优先级来源，例如 Repository、Stage 覆盖、全局 Pipeline、Liquid default 或缺失；不表示某次 Run 的冻结实际值 |

所有 `variables` 响应字段始终返回非 `nil` 空列表。读模型以当前资源或冻结资源为上下文组装：Repository 变量只有全局配置；Pipeline 变量包含当前 Stage 引用；Snapshot/Run 基于其冻结 Stage 和变量声明重建引用与分类。所有页面使用同一展示组件，资源页面只选择适合其场景的 Tab、操作和时间语义。

## Implementation Steps

1. 在公共应用 DTO 中定义变量读模型、配置片段、阶段绑定和引用位置；所有资源详情 DTO 使用它，避免 `map[string]any` 穿透至 HTTP mapper。
2. 在 `pipelinevariable` 规则中抽取可复用的结构化 Stage 引用收集器。它解析 `script` 与每个 Artifact 的 `reference`、`name`、`command`，保留 Stage、字段、制品和本地 Liquid default；既有运行时变量提取改为复用该收集结果，保证运行行为不变。
3. 在变量读模型组装器中，按资源上下文将阶段引用、Repository/Pipeline 配置、阶段覆盖、内置运行时/系统变量和冻结 Run/Snapshot 值组合为公共条目。复用既有名称校验和运行优先级规则，但不调用或模拟一次具体 Run 的实际值解析。
4. 调整 Pipeline、Pipeline Snapshot 和 Pipeline Run 查询用例：Application Pipeline 有绑定 Repository 时通过现有 `RepositoryStore` 读取当前 Repository；Snapshot/Run 使用冻结 Stage 和声明重建展示关系；Template Pipeline 不传 Repository。已绑定 Repository 缺失或读取失败应作为领域错误返回，不静默降级。
5. 修改 `common.proto` 及各资源响应 Proto 为 `variables` 和公共嵌套消息；运行 `task proto` 生成 Go 与 TypeScript。更新 Repository、Pipeline、Pipeline Run 的 HTTP mapper 统一映射读模型。
6. 新建公共变量展示组件，供 Repository、Pipeline、Snapshot 和 Run 页面复用。Pipeline 页面保留“全局变量”“阶段变量”Tab 与 Pipeline 配置操作；其他页面复用同样的分类、绑定和引用展示，并按各自只读/可编辑边界控制操作。删除对响应 `source`、空 `stage_id` 和 `stage_defaults` 的分类推断。
7. 保持构建阶段表的现有列、DAG 和 Stage 操作。制品继续读取 `stage_nodes[*].artifacts`，不由变量表或构建命令推断；必要的制品展示调整限于明确的制品信息，不将 Stage 主表转为变量载体。
8. 更新所有变量 mapper、工具与页面测试；删除仅为旧 `VariableDeclarationResp.source` 设计的详情断言。

## Files To Change

- `proto/orbit/v1/common/common.proto`、`proto/orbit/v1/{repository,pipeline,pipeline_run}/...`、生成的 `internal/gen/proto/...` 和 `web/src/gen/proto/...`
- `internal/application/{repository,pipeline,pipeline_run}/dto/...` 与公共变量读模型 DTO
- `internal/application/pipeline/rule/pipelinevariable/runtime.go` 及必要的新读模型辅助文件
- `internal/application/pipeline/usecase/pipeline.go`
- `internal/api/http/handler/{repository,pipeline,pipeline_run}/...`
- `internal/application/pipeline/rule/pipelinevariable/runtime_test.go`
- `internal/application/pipeline/usecase/pipeline_test.go`
- `internal/api/http/handler/pipeline/mapper_test.go`
- `web/src/utils/variableDeclaration.ts`、`web/src/utils/variableDeclaration.test.ts`
- `web/src/views/{repository,pipeline,pipeline_run}/...`
- 公共变量展示组件及其必要的局部测试

不修改数据库迁移、变量写入 API 和运行变量合并顺序。

## Verification Plan

1. 变量规则单测：覆盖六类 `kind`、全局/阶段 `scope`、Repository 当前变量、同名全局变量的多 Stage 引用、所有脚本/制品字段、每个引用位置独立 default、全局继承、阶段覆盖、Template Pipeline 无 Repository 变量。
2. Pipeline、Snapshot、Run usecase 单测：覆盖 Application Pipeline 读取绑定 Repository 的当前变量、冻结资源重建关联、缺失绑定 Repository 的错误路径及 Template Pipeline 的无 Repository 读模型。
3. HTTP mapper 单测：覆盖所有资源 `variables` 的结构化 JSON/Proto 映射、空数组、`stage_binding`、`references`、配置片段和 Secret 元数据。
4. 前端测试：覆盖四个页面的相同分类/阶段展示、Pipeline 的全局/阶段 Tab，以及保存全局配置、创建/编辑/删除 Stage 覆盖的请求仍只包含可写 Pipeline 配置。
5. 执行 `task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`，并按失败定位运行相关的 Go/Vitest 测试。最后运行 `git diff --check`。

## Assumptions And Risks

- 现有资源响应既被列表又被详情使用；列表继续接收结构化 `variables`，Application Pipeline 列表可能因此增加一次 Repository 查询。实现时应复用当前详情加载边界，避免在前端追加 N+1 请求；如发现列表性能成为实际问题，后续单独拆分 summary 响应，不在本变更引入兼容字段。
- `value_source` 是“按当前配置可得的优先级来源”，不是预执行的最终渲染值；动态嵌套 Liquid 和 Run 时间戳的最终值仍以 Run 快照为准。
- 现有前端文件已有用户进行中的修改。实现时以其当前工作树状态为基础，只替换旧详情变量读法，不能还原或覆盖无关变更。

## Rollback

本变更不涉及持久化迁移。若需要回退，整体回退公共 Proto、读模型、全部资源 mapper 和前端页面适配；不能只恢复某个页面对 `source` 的推断，否则会重新形成不一致的变量语义。

## User Review Notes

- 已确认 Repository 变量直接列入全局变量。
- 已确认变量分类与每个阶段的变量展示由后端结构化输出，不接受前端重解析或猜测。
- 已确认构建阶段保留原有信息；制品是独立的显式声明，不是变量的附属定义。
- 已确认变量读模型必须用于所有变量接口和页面，而非仅 Pipeline 详情。
