# backend-go CI 运行时变量修复需求
最后修改时间: 2026-06-18 15:33:37

Review status: Accepted

## Background

`3657f12` 将 backend-go 模板引擎从 Pongo2/Jinja 迁移到 Liquid，并按后续裁定移除了初始化流水线模板中的模板内 `default` fallback。当前 backend-go 的 CI 运行时变量链路没有完整对齐 Python backend：触发运行时只保存/传递用户输入变量，执行阶段可能直接使用任务 payload 中的空变量 map，导致 `working_dir`、`repository_dockerfile` 等模板变量无法从默认值或声明中获得最终运行值。

Python `backend` 已有较完整实现，核心语义包括：创建 Run 前解析完整变量声明、构建运行时 merged variables、校验缺失变量、保存脱敏后的完整 `VariableDeclaration[]` 快照，并用该 merged variables 执行。

## Goal

修复 backend-go CI 运行时变量处理，使其行为与 Python backend 的核心语义保持一致：

- 手动触发、webhook 触发、retry 执行路径都能获得完整运行时变量。
- `PipelineRun.VariablesSnapshot` 保存完整变量声明列表，而不是仅保存用户本次传入的覆盖变量 map。
- 默认值来源遵循 `value` 优先、无 `value` 时使用 `default` 的规则。
- 内置变量由系统注入，不允许被 runtime overrides、仓库变量或模板变量覆盖。
- 种子内置流水线模板在移除模板内 `default` fallback 后仍具备明确变量来源。

## Non-goal

- 不恢复 Jinja/Pongo2 语法或 `.jinja` 支持。
- 不恢复模板内隐式 fallback 写法作为业务默认值来源。
- 不重构整个 CI 模块为 Python backend 的完整 DDD 结构。
- 不扩大前端交互范围；本次仅同步修复 `TriggerModal` 变量初始化使用 `value ?? default` 的行为。
- 不处理已部署数据库中的历史数据迁移策略；本次按用户要求就地修改当前初始化迁移脚本与种子数据行为。

## User scenarios

1. 用户手动触发“通用容器镜像流水线”，没有显式输入 `working_dir` 时，流水线应使用声明中的默认值而不是渲染失败。
2. 用户通过 webhook 触发流水线时，应获得相同的变量合并语义，并包含 `commit_sha`、`author`、`event_type` 等 runtime 变量。
3. 用户 retry 一个已完成或失败的 PipelineRun 时，应基于原 run/snapshot/repository 重新获得可执行变量，而不是因为任务 payload 不含变量而丢失默认值。
4. 用户尝试用 runtime variables 覆盖 `repository_id` 等内置变量时，系统应继续使用系统注入值。

## Acceptance

- backend-go 创建 PipelineRun 时能构建完整 runtime variable map，至少覆盖：
  - repository 内置变量：`repository_id`、`repository_name`、`repository_code`、`repository_url`、`repository_ref`
  - template 内置变量：`template_id`、`template_name`、`template_version`、`runtime_datetime`
  - runtime overrides
  - repository `variable_overrides`
  - template declarations 的 `value/default`
  - stage declarations 的 `value/default`
- 合并优先级与 Python backend 一致：系统内置变量最高，runtime overrides 不能覆盖内置变量，其后是仓库自定义、模板声明、stage default。
- `PipelineRun.VariablesSnapshot` 对新 run 保存完整 `VariableDeclaration[]`，且执行阶段可从该快照恢复变量 map。
- worker/task payload 不再作为运行变量的权威来源；至少不能因 payload 中 `variables:{}` 导致默认值丢失。
- 当前初始化的 “通用容器镜像流水线” 不依赖模板内 `default` filter 也能获得 `working_dir` 和 `repository_dockerfile` 默认值。
- 新增或更新 backend-go 单元测试覆盖：默认值合并、内置变量不可覆盖、手动触发或执行路径使用完整变量、retry/webhook 至少覆盖关键路径中的一种。
- frontend `TriggerModal` 初始化变量输入时使用 `value ?? default`，确保有默认值但无显式 value 的变量能在触发弹窗中显示默认值。
- backend-go 相关测试通过；若修改 frontend，则执行项目要求的前端 lint/typecheck。

## Open questions

- 无。

## Decisions

- 以 Python `backend` 的 `VariableResolver.build_runtime_variables` 与 `PipelineRunService.create_run` 语义作为 backend-go 对齐基准。
- 不通过恢复 Liquid `default` filter 到初始化模板来解决业务默认值缺失；默认值应来自变量声明/快照/运行时合并。
- 用户确认本次就地修改 backend-go 初始化迁移脚本，接受已执行 migration checksum 风险需在交付说明中明确提示。
- 用户确认本次纳入 frontend `TriggerModal` 的 `value ?? default` 初始化行为。
- 用户采纳 backend-go secret 变量脱敏按 Python `mask_secrets` 等价最小行为实现。

## Risk

- backend-go 当前 `pipelineRunVariables` 兼容旧 map 格式；改为新 run 保存完整 `VariableDeclaration[]` 时，需要保留读取旧格式的能力，否则历史 run 展示或 retry 可能受影响。
- 如果 snapshot 生成逻辑已经在别处固定版本，修改变量快照生成可能影响模板版本/快照复用语义，需要避免扩大范围。
- 种子 SQL 修改属于已执行 migration 文件修改，项目当前已有类似变更历史，但仍有 checksum 风险；本次应在实现结果中明确提示。
