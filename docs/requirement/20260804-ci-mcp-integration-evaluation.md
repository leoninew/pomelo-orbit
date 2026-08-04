# 持续集成 MCP 集成评估
最后修改时间: 2026-08-04 15:26:00

Review status: Accepted

流程模式: 标准 / standard

## 背景与范围

Pomelo Orbit 已具备完整的持续集成领域能力：流水线阶段与模板、不可变快照、运行触发、异步执行、取消/重试、阶段日志和制品查询。当前这些能力经受鉴权的 HTTP 路由提供给 Web 前端和调用方；本次评估判断是否以及如何向 Codex 暴露独立的 CI MCP。

本评估范围包括现有 CI 领域和 HTTP API、当前会话实际可用的 MCP 工具与资源，以及独立 CI MCP 和既有 `pomelo_delivery` 的职责边界。结论与后续计划仅描述建议，不进行 MCP、HTTP API、数据库、流水线或部署的写入。

## 当前现状

1. CI 领域以可复用 `PipelineStage`、编排 `PipelineTemplate`、不可变 `PipelineSnapshot` 和可追溯 `PipelineRun` 为核心；运行以后台任务异步执行，已有单元和集成测试覆盖触发、执行、日志及制品读取。
2. 当前 HTTP API 已提供模板和阶段的列出、创建、读取、更新、删除、复制与变量解析；还提供按仓库触发运行、列出/读取运行、读取制品与增量日志、取消和重试。实际路由位于 `internal/api/http/routes/pipeline.go` 和 `internal/api/http/routes/pipeline_run.go`。
3. 仓库自身 GitHub Actions 包含 Go 代码生成、格式化、lint、vet、测试及镜像构建发布工作流。这些是仓库验证/发布自动化，不是 Pomelo Orbit 面向用户的 PipelineRun 控制接口。
4. 本次会话发现的 `pomelo_delivery` MCP 暴露 `orbit_*` 控制面工具和 `runtime_*`、`verify_deployment` 运行态只读工具。MCP resources 仅包含终端服务信息与文件传输状态，不包含 CI 的项目、模板、运行或日志资源。

## MCP 能力与集成边界

| 边界 | 归属 | 结论 |
| --- | --- | --- |
| Application、Version、Service、Gateway 生命周期 | `pomelo_delivery` 的 `orbit_*` | 保持现状；其写操作只完成用户明确要求的单一交付动作。 |
| 受管 Docker Compose 状态、日志、网络、探针与部署验证 | `pomelo_delivery` 的 `runtime_*` 与 `verify_deployment` | 保持受管 target 和只读诊断限制；不能作为 CI 运行的替代日志或执行接口。 |
| CI 阶段、模板、快照、运行、日志、制品 | 本项目 CI HTTP API，后续独立 `pipeline-mcp` | 需要独立 MCP 契约；不得通过 `orbit_*` 直接读取或修改 CI 数据。 |
| CI 运行触发、取消、重试 | 后续独立 `pipeline-mcp` 的显式写工具 | 必须逐项显式调用、沿用服务层授权和校验；不得隐式部署、停止服务或读取凭据。 |
| CI 到 CD 的衔接 | 调用方编排 | CI 成功不自动触发 `orbit_deploy`；部署仍需用户明确请求，且在独立 MCP 调用中完成。 |

`pomelo_delivery` 的 Server instructions 明确其面向 Orbit 生命周期写入，运行态工具只读且目标必须受 Orbit 管理。因此，把 CI 工具加入该 Server 会混淆权限、资源标识和审计语义；当前 `mcp/src/pipeline-mcp/` 的独立预留方向与此结论一致。

## 目标

1. 给出可实施的独立 CI MCP 集成方案，而不改变当前 CI 或部署行为。
2. 先让代理可安全检索项目内可见的 CI 定义和运行状态，再逐步开放具备明确副作用的动作。
3. 保持 CI 的项目成员授权、变量校验、快照不可变性、后台派发和日志 offset 语义由既有 Go 应用服务统一执行。
4. 明确 CI 与 CD 的人工确认边界，防止“成功构建”被扩展为未经授权的生产部署。

## 非目标

1. 不注册新的 MCP Server、工具、资源或 OAuth/令牌方案；用户后续授权的首期实现仅限独立的固定 GET 查询 MCP。
2. 不改动 `pomelo_delivery` 的工具表、Orbit HTTP API 或 Docker Compose 生命周期边界。
3. 不将 GitHub Actions workflow 映射为用户 CI PipelineRun，也不读取 GitHub Actions 的外部运行记录。
4. 不执行流水线、部署、停止、重试、取消或任何外部写操作。

## 推荐方案

采用独立的 `pomelo_pipeline` MCP Server（实现项目为 `mcp/src/pipeline-mcp/`，命令名、注册名和包布局须在实施时以当前工程为准），通过最薄的 CI HTTP client 调用既有受鉴权 API；不访问 SQLite、任务队列目录、运行日志文件或 Docker CLI。

第一阶段只读工具应覆盖：项目可见的阶段和模板列表/详情、模板变量预览、运行列表/详情、制品元数据和带 `offset` 的阶段日志。第二阶段才引入显式写工具：创建/更新模板或阶段、触发运行、取消运行和重试运行。每个工具均传递用户身份/授权上下文，由现有应用服务进行项目成员校验、参数校验与状态转换；工具不自行重建这些规则。

写工具的结果必须返回稳定的资源标识、当前状态和简短动作摘要，默认不返回变量值、仓库凭据、完整脚本、运行环境或原始日志。日志读取应复用现有 `offset` 与 `is_complete` 契约，并限制单次结果大小。跨 MCP 的 CI->CD 工作流必须由调用方显式串联：先读取并确认 CI 成功，再由用户单独授权 `pomelo_delivery` 部署。

## 验收标准

1. 方案明确区分 CI MCP、`pomelo_delivery` 和 GitHub Actions 的职责，并禁止 CI 成功后自动部署。
2. 工具清单覆盖至少模板/阶段、运行、日志和制品的只读需求，并将写动作拆分为显式的触发、取消、重试和配置变更。
3. 每个候选工具都有对应 HTTP API/应用服务来源，避免绕过授权、变量校验、快照或任务派发。
4. 计划包含 schema、鉴权、脱敏、日志分页、错误传播、幂等性与 CI/CD 边界的验证。
5. 本次交付仅新增本需求和计划文档，不修改生产代码或执行外部写入。

## 风险与假设

1. 假设独立 pipeline MCP 可以获得与 Web API 等价的短期用户身份上下文；若不能，必须先设计代理身份、项目范围与审计模型，不能以管理员令牌替代。
2. 模板脚本和运行变量可能含有路径、令牌引用或业务敏感内容，默认 MCP 响应需要字段级脱敏和长度限制，尤其是日志与变量。
3. 触发、取消和重试都改变运行状态；网络重试可能造成重复触发。因此需要 API 侧幂等键或请求标识与可重放结果的设计验证。
4. CI 执行容器与 Orbit 受管部署容器共享宿主机时，MCP 必须避免提供任意 Docker 访问能力或扩大挂载、网络和凭据可见范围。
5. stdio MCP 工具表会被客户端缓存；实现/注册变更后需建立新会话，并通过工具清单与 schema 测试确认生效。

## 决策与审查记录

1. 2026-08-04：按用户授权的 Requirement -> Plan 流程，本需求直接接受并进入 Plan。
2. 2026-08-04：选择独立 `pipeline-mcp`，不扩展 `pomelo_delivery`。
3. 2026-08-04：选择“只读优先、显式写入、CI/CD 不自动串联”的渐进交付策略。
4. 暂无需要用户确认的未决事项；涉及身份传递和幂等键的具体契约在实施前必须依据当时的 API 与 MCP schema 复核。
5. 2026-08-04：用户授权进入 Implementation / 实现阶段，首期范围调整为实现独立只读 CI MCP；不改变本需求中的部署隔离、无写操作、无自动注册和无 CI->CD 自动串联约束。
