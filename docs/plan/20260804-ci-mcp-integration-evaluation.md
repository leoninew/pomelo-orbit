# 持续集成 MCP 集成评估计划
最后修改时间: 2026-08-04 15:09:46

Review status: Accepted

流程模式: 标准 / standard

## 需求依据与范围

- [持续集成 MCP 集成评估](../requirement/20260804-ci-mcp-integration-evaluation.md) 已于 2026-08-04 接受。
- 本计划只定义后续实现与验证路径；本轮不写入 MCP、应用 API、数据库、任务队列或部署环境。
- 当前 `pomelo_delivery` 仍专属 Orbit Application、Version、Service、Gateway 生命周期和受管运行态诊断。CI MCP 必须是独立 Server，不能复用 `orbit_*` 的资源模型或把 CI 成功隐式转换为部署。

## 现状与能力盘点

1. CI HTTP 层已暴露 PipelineStage、PipelineTemplate、PipelineSnapshot、PipelineRun、制品和阶段日志的操作；触发、取消和重试由 `pipeline_run` 应用服务处理，执行经后台 task 派发。
2. 现有 CI 设计依赖模板版本/快照、变量优先级与项目成员授权，因此 MCP 只能适配现有 API，不能直连数据库或手工构造运行状态。
3. 当前可用 `pomelo_delivery` 工具覆盖 `orbit_list_*`、`orbit_get_*`、生命周期写操作、`runtime_*` 诊断和 `verify_deployment`，不含 CI 工具或资源。现有 MCP resource 也没有 CI 数据源。
4. GitHub Actions 的 `go-verify.yml` 与 `docker-publish.yml` 验证本仓库并发布镜像；它们不等价于产品的 PipelineRun，后续 MCP 不纳入其运行控制。

## 推荐集成设计

在 `mcp/src/pipeline-mcp/` 完成独立的 stdio MCP 项目，注册名、启动命令、包目录和鉴权注入点以实施时的项目布局和工具 schema 为准。它持有一个薄 HTTP client，调用现有 CI HTTP API；Go 应用服务继续负责授权、输入校验、变量解析、快照创建、状态迁移和后台派发。

工具按最小权限分两批交付：

| 批次 | 候选工具 | 权限与输出边界 |
| --- | --- | --- |
| 只读 | `pipeline_list_stages`、`pipeline_get_stage`、`pipeline_list_templates`、`pipeline_get_template`、`pipeline_resolve_template_variables`、`pipeline_list_runs`、`pipeline_get_run`、`pipeline_list_artifacts`、`pipeline_get_stage_log` | 只返回调用者有权查看的投影；变量、脚本、日志按字段和字节数脱敏/截断，日志维持 offset 分页。 |
| 受控写入 | `pipeline_create_stage`、`pipeline_update_stage`、`pipeline_create_template`、`pipeline_update_template`、`pipeline_trigger_run`、`pipeline_cancel_run`、`pipeline_retry_run` | 每次动作独立且显式；需认证、项目范围、输入校验、审计请求 ID 和明确状态结果。删除工具延后，待恢复策略和引用约束评审后再决定。 |

跨 MCP 的行为保持显式：读取 CI 成功后，只有用户明确要求，才可在单独的 `pomelo_delivery` 调用中执行部署。CI MCP 不调用 `orbit_deploy`，Delivery MCP 也不接受 PipelineRun ID 作为部署授权。

## 实施进度

2026-08-04，用户授权进入 Implementation / 实现阶段后，已在 `mcp/src/pipeline-mcp/` 实现首期独立只读 Server：`pipeline_list_stages`、`pipeline_get_stage`、`pipeline_list_templates`、`pipeline_get_template`、`pipeline_get_snapshot`、`pipeline_list_runs`、`pipeline_list_repository_runs`、`pipeline_get_run`、`pipeline_list_artifacts`、`pipeline_list_run_artifacts` 和 `pipeline_get_stage_log`。HTTP client 仅公开固定的 `GET` 路由，要求外部传入短期用户 JWT，不接受用户名/密码，也不写入 token cache。

阶段/模板脚本、变量 default/value、运行错误文本和制品物理路径默认不出现在 MCP 响应中。阶段日志在返回前对常见凭据模式做遮蔽，并以 UTF-8 字节窗口截断，返回可继续读取的 offset。

本首期未实现 `pipeline_resolve_template_variables`：现有 API 是 `POST /api/pipeline/template/resolve-variables`，虽然其意图是无持久化的预览，但不满足本轮“仅固定 GET 查询”的最小权限边界。它连同全部写工具、Codex 自动注册和任何 CI->CD 自动串联均延后到后续独立需求或评审。

## 实施步骤

### 1. 固化现有 CI HTTP 契约

1. 以 `internal/api/http/routes/pipeline.go`、`pipeline_run.go`、对应 handler、Proto DTO 和应用服务为准，列出每个候选 MCP 工具的请求、响应、授权、错误与状态前提。
2. 补齐只在 Web 层可用但 MCP 必需的只读投影时，先新增最小 HTTP API/DTO，再由 MCP client 调用；不得让 MCP 越过应用服务读取 repository、任务或日志文件。
3. 为触发请求设计幂等键或等价的请求去重契约，并确定取消/重试在终态、并发和网络超时下的可预期响应。若现有 API 不具备该能力，先更新需求/计划再改 Go 代码。

### 2. 建立独立 MCP 的认证与传输边界

1. 在 `mcp/src/pipeline-mcp/` 确认现有预留工程的真实布局；添加独立的配置、HTTP client、server 注册和测试入口，但不改变 `pomelo_delivery` 的命令或工具表。
2. 使用短期、用户范围的身份上下文调用 CI API；禁止将管理员凭据、数据库路径、Docker socket 或运行时目录暴露给 MCP。
3. 为每次工具调用注入可关联的 request ID，并把 HTTP 错误统一映射为稳定 MCP 错误；不向客户端回显服务端堆栈、认证材料或未脱敏响应。

### 3. 先实现只读工具

1. 先实现阶段、模板、变量预览、运行、制品和阶段日志的只读工具，输出显式的项目/仓库/模板/运行 ID 和分页/offset 信息。
2. 对阶段和模板详情建立最小视图：默认隐藏敏感变量值与不必要的执行细节；是否可返回脚本文本需按现有产品授权模型单独确认。
3. 日志使用已有 `offset` 与 `is_complete`，设置最大读取窗口和截断标识；工具不追踪、轮询或流式阻塞等待运行结束。

### 4. 增量开放受控写工具

1. 在只读 schema、认证和审计验证通过后，再逐项加入阶段/模板创建更新、运行触发、取消和重试。
2. 每个写工具只做其名称表达的动作；触发运行不等待完成、不读取日志、不部署，取消/重试不触发其他运行，配置更新不隐式修改关联对象。
3. 删除阶段/模板和批量操作不纳入首版。需要时另立需求，明确引用检查、恢复、审计与并发策略。

### 5. 更新文档、注册与操作约定

1. 更新 MCP README、操作指南与 Codex 注册说明，明确两个 Server 的用途、启动方式、工具前缀、会话重启和 CI/CD 的显式交接。
2. 将 API/MCP schema 测试加入 pipeline MCP 的默认检查；工具或 schema 变更后以新 stdio 会话重新发现工具，不假设热加载。
3. 不改变 GitHub Actions 的 Go 验证与镜像发布职责；必要时只为 MCP 项目增加其自身的静态检查/测试工作流。

## 预期涉及文件

| 路径 | 后续可能改动 |
| --- | --- |
| `mcp/src/pipeline-mcp/` | 独立 CI MCP server、配置、HTTP client、工具注册、测试与打包元数据；实施前先确认实际预留目录的内容。 |
| `internal/api/http/routes/pipeline.go`、`internal/api/http/routes/pipeline_run.go` | 仅在现有受鉴权 API 缺少 MCP 必需的最小投影或幂等契约时调整。 |
| `internal/api/http/handler/pipeline/`、`internal/api/http/handler/pipeline_run/` | 与新增/收敛 HTTP 契约配套的绑定、授权和错误响应。 |
| `internal/application/pipeline/`、`internal/application/pipeline_run/` | 仅补充服务层授权、幂等或安全投影规则；不将 MCP 逻辑放入领域层。 |
| `proto/orbit/v1/`、生成代码 | 仅在 HTTP/Proto 契约确需扩展时修改 source 并按项目工具链重新生成。 |
| `docs/guides/`、MCP README、Codex 配置 | 双 MCP 注册、最小权限边界、会话刷新和使用说明。 |

本评估不修改上述非文档文件。实施开始前必须重新检查当时的工作树、MCP tool schema 和项目指引；`docs/archive/` 不作为操作依据。

## 验证计划

1. 对照路由、handler、DTO 和应用服务，为每个 MCP tool 建立 HTTP 契约测试，覆盖项目成员授权、资源不存在、无权访问、输入校验和错误映射。
2. 对只读工具验证项目范围隔离、变量/脚本/日志脱敏、日志 offset 连续性、最大窗口、终态 `is_complete` 和制品元数据访问控制。
3. 对触发、取消、重试验证幂等键、重复请求、并发请求、终态运行、后台派发失败和网络超时后的可重试语义；断言工具不会越权启动部署。
4. 对模板/阶段写工具验证引用约束、变量解析、快照不可变性和服务层授权继续生效；不以 MCP 侧重复业务校验替代后端。
5. 运行 pipeline MCP 项目的格式、lint、类型检查和测试，以及受影响的 Go `fmt`、`vet`、`test` 与生成代码校验。具体命令以实施时项目入口为准。
6. 在隔离的测试项目中，以新 stdio 会话核对工具注册表、参数 schema、默认响应脱敏和审计 request ID；不连接生产数据库、不打印凭据、不执行部署。
7. 仅在用户明确授权的环境中进行一次端到端 CI MCP 试运行；部署验证须另行取得用户明确授权并使用 `pomelo_delivery` 的既有受控流程。

## 回滚与风险

1. 新 MCP 仅在注册后对客户端可见；出现问题时撤回其注册并重启 stdio 会话。既有 CI HTTP API、`pomelo_delivery` 和 GitHub Actions 不应受影响。
2. 已触发的 PipelineRun 不因 MCP 回滚而自动取消或删除；后续操作须由用户明确指示并走既有 CI 服务路径。
3. 最高风险是服务身份绕过用户权限或响应泄露变量、脚本、日志和凭据引用。先完成身份、字段投影和审计设计，再暴露任何写工具。
4. 并发触发和客户端重试可能重复消耗 CI 资源；未确认幂等语义前，首版写工具不得上线。
5. CI/CD 合并工具会扩大事故范围；保留两个 MCP Server 和两次显式用户动作是有意的安全边界。

## 审查记录

1. 2026-08-04：根据已接受 Requirement 创建并接受本计划；不进入 Implementation。
2. 2026-08-04：确认现有 `pomelo_delivery` 只提供交付控制面和运行诊断，不包含 CI MCP 工具或资源。
3. 2026-08-04：确认后续实现以独立 pipeline MCP、只读优先、受控写入和显式 CI->CD 交接为基线。
4. 2026-08-04：完成首期固定 GET 的 CI 查询 MCP；变量预览因现有 POST 契约而延期，未注册 Codex Server，未实现写工具或部署交接。
