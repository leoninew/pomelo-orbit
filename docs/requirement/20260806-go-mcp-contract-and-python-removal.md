# Go MCP 契约补齐与 Python MCP 移除
最后修改时间: 2026-08-06 13:04:00

Review status: Accepted

流程模式: 标准 / standard

## Background

持续部署的 MCP 已由 Go `internal/api/mcp/delivery` 直接适配 application usecase，并通过认证后的 Streamable HTTP `/mcp` 供持续部署对话使用。当前仓库仍保留两套独立 Python MCP 工程：`mcp/src/delivery-mcp/` 与 `mcp/src/pipeline-mcp/`；`.codex/config.toml` 和部分 skill、操作文档仍指向 Python delivery stdio 入口。

工具名称对照表明 Python delivery 与 Go delivery 各注册 48 个同名工具。问题不在于把 Python 的 HTTP 客户端、Docker 封装或编排逻辑搬到 Go，而在于基于现有 Go application 能力，让 MCP schema、工具说明和对话编排准确表达领域规则。

最近的对话部署显示该缺口：用户明确请求 `controlled_file` 和 `gateway_http`，模型读取到已有但不匹配的 `file`、`host` 配置后没有纠正，仍发起部署，并在工具调用轮次耗尽后以失败结束。Go 应用层已支持并校验 `controlled_file`，但 MCP 没有向模型清楚说明其 source type、相对路径、空内容、`mode` 等契约，也缺少变更后的复核和单次部署约束。

## Goal

1. 直接移除整个 `mcp/` 目录中的所有 Python MCP 实现与其 Python/uv 项目工件；`pipeline-mcp` 不迁移、不补 Go 对应能力、不保留兼容入口。
2. 保持 Go delivery MCP 作为持续部署 MCP 的唯一实现，继续直接调用现有 application usecase，不回退到本项目 HTTP API 回环或 Python 适配层。
3. 补齐 Go MCP 面向模型的领域契约，使挂载、端点等现有能力可被稳定发现和正确调用，优先解决 `controlled_file` 与 `gateway_http`。
4. 改进持续部署对话的变更编排：对于已有但不符合用户期望的资源，先纠偏、回读确认，再创建且只创建一次部署，并把部署终态或部分已执行结果明确反馈给用户。
5. 将 Codex 注册、AGENTS、活操作指南和 RAGFlow skill 更新为当前 Go MCP transport 与认证边界，不再引用 Python stdio 命令。

## Non-goal

1. 不将 `pipeline-mcp` 的 11 个只读 CI 工具改写为 Go，也不把它们并入 `pomelo_delivery`。
2. 不复刻 Python delivery 的 HTTP client、JWT 缓存、Docker runtime 实现、工具包装形状或旧的 stdio 启动方式。
3. 不改变 Application、Version、Service、Gateway、Deployment 的领域模型、数据库 schema、HTTP API、异步 worker 或 Docker Compose 生命周期语义。
4. 不删除与 MCP 无关的 `scripts/*.py`。
5. 不修改 `docs/archive/**` 的历史记录。

## User scenarios

1. 用户通过持续部署对话要求创建或修复组件配置，包含受控文件挂载和 Gateway HTTP 暴露；模型能使用 MCP 的真实字段契约完成配置并说明结果。
2. 目标 Application、Version 或 Service 已存在但配置不符合请求时，对话先读取差异、更新对应声明或 overlay、回读验证，再部署一次并等待终态。
3. 用户在 Codex 中使用 `pomelo_delivery` 时，注册不依赖已删除的 Python/uv 项目，并以 Go `/mcp` 所要求的认证方式连接。
4. 仓库不再包含可运行的 Python delivery 或 pipeline MCP 工程、其测试、锁文件、启动命令或活文档引用。

## Acceptance

1. `mcp/` 目录及其中的 Python packaging、测试、锁文件、启动说明均从仓库移除；不保留 Python MCP 兼容入口。
2. Go delivery MCP 保留当前 48 个持续部署工具及其 application-usecase 边界；删除 Python 后不以工具删减替代能力补齐。
3. `orbit_update_version_component_mounts` 的 MCP schema 或工具说明明确区分 `directory`、`file`、`named_volume`、`controlled_file`，并说明受控文件的相对 source、非 host path、内容上限、四位八进制 `mode` 与空内容形态。调用 `controlled_file` 能正确映射到既有 Go application input。
4. 与组件写入有关的端点、挂载及其他离散领域值向模型暴露的约束与现有 Go 校验一致；不得只依赖模型猜测或后台 validation error 发现规则。
5. 对话在发现已有状态与用户目标不一致时，执行最小必要更新并回读断言；同一用户请求不重复创建 Deployment；若轮次或模型调用失败，响应保留已创建资源和部署的可识别结果，而非把成功副作用笼统报告为不可用。
6. `.codex/config.toml`、`AGENTS.md`、活 `docs/guides/` 与 `skills/` 不再引用 Python MCP 命令或已删除目录，并准确说明 Go MCP 的实际连接和认证方式。
7. 为 Go MCP 工具 schema、受控文件映射与校验、既有配置纠偏、单次部署编排和 Python MCP 引用清理提供针对性自动化检查。

## Open questions

暂无需要用户确认的未决事项。浏览器登录触发方式、凭据持久化格式与 token 刷新策略属于 Plan 阶段的实现设计，必须满足现有认证边界且不得向日志或仓库写入凭据。

## Decisions

1. 这是新的任务记录，不修改正在暂存的 `20260805-go-delivery-mcp` 过程文档。
2. `pipeline-mcp` 直接移除，无须盘点或设计其替代能力。
3. Go application usecase 和当前 Go MCP 是唯一的业务与工具实现依据；Python 行为不构成兼容性目标。
4. `controlled_file` 是 Go MCP 的可用性与表达能力缺口，不是要求新增第二套文件物化机制。
5. 活文档和 skill 应贴近实际 Go transport；历史 archive 仅保留为历史证据。
6. Codex 使用 Go MCP 时，通过浏览器打开 Orbit 登录页完成认证，并在本地记录认证凭据，供 Go MCP 的认证连接使用。
7. `mcp/` 是 Python MCP 实现目录，直接整体移除；其中的 pipeline MCP 无须保留、迁移或替代。

## Risk

1. 当前 `.codex/config.toml` 仍启动 Python delivery MCP。删除 `mcp/` 前必须完成以浏览器登录和本地凭据为基础的 Go MCP 注册切换，否则外部 `pomelo_delivery` 调用会失效。
2. Go MCP 的通用 proto schema 对条件字段和枚举的表达有限；若只增加宽泛文字说明，模型仍可能误用字段，需要同时定义可测试的工具契约与对话复核规则。
3. 对话工具调用是有副作用的多轮过程，单纯提高轮次上限会放大重复部署风险，不能替代显式的阶段状态和结果收敛。
4. 浏览器认证和本地凭据持久化必须避免密码、JWT 或其他敏感值写入仓库、过程文档和应用日志，并需要处理凭据失效后的重新登录。

## User review notes

1. 用户要求以新的 SpecFlow 标准 / standard 任务记录推进。
2. 用户明确要求移除所有 Python MCP 实现。
3. 用户明确 `pipeline-mcp` 直接移除，无须关注或迁移。
4. 用户强调实现方向不是复刻 Python MCP，而是基于现有 Go 能力补齐 MCP；`controlled_file` 是优先缺口之一。
5. 用户决定 Codex 使用 MCP 时打开浏览器登录 Orbit 并记录本地凭据。
6. 用户确认 Python 实现位于 `mcp/`，该目录直接移除。
7. 用户明确要求进入 Plan / 计划阶段，Requirement 视为接受。
