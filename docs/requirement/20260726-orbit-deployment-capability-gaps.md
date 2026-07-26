# Orbit 部署能力补充需求
最后修改时间: 2026-07-26 16:12:57

## Review status

Accepted

## Background

这是任务一「既有实现的功能补充」。现有 Orbit Application / Version / Deployment 和本地 MCP 已能管理组件镜像、命令、环境变量、逻辑挂载、健康检查、资源、端口暴露、部署等待及受管 Docker 只读诊断。

RAGFlow 拆分部署计划识别出若干未被 VersionComponent 模型表达的 Compose 运行时字段。平台能力补充必须独立评审和交付，不将其混入 RAGFlow 的具体 Application 数据或部署操作。

## Goal

1. 为受管 Application Version 补齐经过确认的运行时字段及其完整链路：领域模型、HTTP 契约、MCP 映射、Compose 渲染、部署执行、预览、一致性验证和测试。
2. 确保新增字段有明确默认值、输入校验、持久化语义、向后兼容行为和安全边界。
3. 给任务二提供可审阅的能力契约，使 RAGFlow 资源建模不依赖未实现或被静默丢弃的 Compose 字段。

## Non-goal

1. 不在本任务创建 RAGFlow、MySQL、Redis、MinIO 或向量数据库 Application、Version、Service、Deployment。
2. 不实现通用 Compose 文件导入、任意 Docker 参数透传、任意 Docker CLI 生命周期命令或不受控宿主机路径挂载。
3. 不把生产秘密写入 Version、Deployment options、MCP 响应、日志或过程文档。
4. 不因为 RAGFlow 原 Compose 包含某字段就自动支持该字段；每项能力必须由需求和安全边界证明必要性。

## Candidate capability gaps

| 能力 | 当前状态 | RAGFlow 影响 | 初步处理 |
|---|---|---|---|
| `restart` policy | 未建模 | 有状态服务不能表达 `unless-stopped` | 若自动恢复为上线要求，则补齐受限枚举 |
| `tmpfs` | 未建模 | Elasticsearch 的 `/tmp` 需要可写临时空间 | 若选 Elasticsearch，则必须补齐受限结构 |
| `ulimits` | 未建模 | Elasticsearch / 部分向量引擎资源限制 | 按选定引擎定义白名单字段 |
| `extra_hosts` | 未建模 | RAGFlow 默认仅为可选 Prometheus 场景使用 | 默认不补，除非任务二启用该场景 |
| named volume 顶层声明 | 未渲染 | 无法安全使用 `source_type=volume` | 任务二先使用逻辑目录；是否补齐待定 |
| 秘密注入 | `runtime_config` 会持久化 | 不适合生产数据库密码 | 需单独设计凭据引用，不能以明文替代 |
| 跨 Application 编排 | 无领域依赖 / 健康等待 | `depends_on` 不能跨 Application | 任务二先以 MCP 顺序和验证编排；是否产品化待定 |

## User scenarios

1. 运维人员在 Version 中配置被平台批准的重启、临时目录或资源限制，并在 Preview 与实际部署中得到一致 Compose 结果。
2. 不认识或不在白名单中的运行时字段被 API 与 MCP 明确拒绝，而不是被忽略或透传。
3. RAGFlow 部署计划能引用已发布的能力契约，明确哪些字段可用、哪些依然不支持。

## Acceptance

1. 每个被接受的能力都有需求、规格、计划、实现和验证记录，并覆盖 Application Version CRUD、Preview、Deployment 与 MCP 工具。
2. 新字段的输入被集中校验；禁止任意宿主机路径、任意 Docker runtime 参数或秘密材料绕过边界。
3. 既有 Version 在未提供新字段时维持当前渲染和部署行为。
4. 任务二开始实施前，所有其实际需要的能力均已在本任务的 Verification 中验证，或被用户明确排除。

## Open questions

1. 自动恢复、`tmpfs`、`ulimits`、named volume、秘密引用、跨 Application 编排中，哪些是本任务必须交付，哪些仅记录为后续工作？
2. 向量数据库最终选择为何？它决定 `tmpfs`、`ulimits` 是否成为本任务的硬需求。
3. 秘密注入是否需要与现有 Credential 领域整合，还是本轮只限制任务二使用非秘密的本机开发配置？
4. 跨 Application 健康依赖是否必须产品化，还是接受由受管 MCP 部署 Runbook 顺序执行？

## Decisions

1. 流程模式采用严格模式 / `strict`：Requirement -> Spec -> Plan -> Implementation -> Verification。
2. 本任务只扩展受管、结构化且白名单化的能力；不接受自由格式 Compose 片段或 Docker 参数透传。
3. 任务一与任务二分别审查、分别接受 Plan、分别实施和验证；任务二不得反向修改任务一范围。

## Risk

1. 在 RAGFlow 资源已经创建后再修改平台模型，会造成重复部署、配置漂移和数据迁移风险，因此必须先完成任务一。
2. `tmpfs`、`ulimits` 与 Docker Desktop / Linux Docker 的支持差异需要在目标 context 验证。
3. 秘密引用若设计不完整，可能通过 Preview、Deployment options、日志或 MCP 响应泄漏。
4. 提升字段表达能力不能削弱 MCP 仅允许受管 Docker 读取、生命周期写入仅经 Orbit HTTP 的安全边界。

## User review notes

用户于 2026-07-26 接受本需求并要求进入 Spec / 规格阶段。Task 1 与 Task 2 继续独立审查。
