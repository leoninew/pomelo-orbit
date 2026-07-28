# 部署运行时能力 Web 支持需求
最后修改时间: 2026-07-26 21:36:24

## Review status

Accepted

## Flow mode

标准模式 / `standard`

## Background

为支持部署运行时配置，项目已补齐 `runtime_env` Credential 引用、`restart_policy`、`tmpfs_json`、`ulimits_json`、Gateway-aware `runtime_doctor`、`verify_deployment` 与固定 HTTP Probe。RAGFlow 的五 Application 拆分部署已用于验证这些能力在真实 Docker context 中可以完成创建、发布、部署和验证；RAGFlow 不是本项目的产品目标。

Web 仍停留在通用的单 Application / Version / Deployment 管理层。它不能创建正确的 `runtime_env` Credential、不能将 Credential `data_key` 绑定至 Version Component，也不能编辑或审查新增的高级 Component 运行时字段。新增的平台能力尚未形成完整的 Web 配置、查看和日常运维路径。

本任务将这些通用部署运行时能力展示到项目既有的 Web 管理界面。RAGFlow 仅作为验收样本，帮助确认界面能正确表达真实的有状态 Application；本任务不增加任何 RAGFlow 业务功能、专用模型或编排流程。

## Goal

1. 让 Web 能以结构化表单首次创建、后续更新、查看和导出持久化的 `runtime_env` Credential，并让 Version Component 以 Credential 元数据和 `data_key` 建立运行时环境变量引用。
2. 让 Web 完整表达并显示通用 Component 的受管运行时配置：command / args、healthcheck、resources、restart policy、tmpfs 与 ulimits；已有 Version 必须可审阅，不能只透传未显示字段。
3. 在既有 Application、Version、Gateway、Service 与 Deployment 页面中展示新增能力及其实际配置、关联关系和部署结果，包括 Gateway target、instance key、Expose、Deployment 终态、错误和日志；Service 详情新增运行时环境变量卡片。
4. 复用既有 Deployment 和 Service 记录作为部署结果的界面依据，不为预检、运行时探测或一致性检查新增独立业务资源。
5. 保持通用运行时配置能力可复用于所有有状态 Application；RAGFlow 只用于验证这一通用能力。

## Non-goal

1. 不让浏览器直接调用 stdio MCP、Docker CLI 或任意 Docker runtime 命令，也不为本任务新增浏览器触发的预检、探测或验证接口。
2. 不重新实现或替换 `scripts/ragflow_initialize.py` 的 fixture 校验、MCP 编排、journal 和恢复语义；该脚本及其 RAGFlow 业务编排不属于本任务。
3. 不接受任意 Compose YAML、任意 Component 字段透传、任意网络名、任意宿主机路径或任意 HTTP Probe 参数。
4. 不在 Version、Deployment 或前端日志中持久化 Credential value。Service 卡片和 Credential 详情可在请求时读取并显示当前持久化值，但不创建第二份值存储。
5. 不在本任务实现 RAGFlow 备份、恢复、模型管理、DeepDoc、TEI、NATS、GPU 或任何 RAGFlow 专用页面或流程。
6. 不从 RAGFlow fixture 批量创建 Application、Version、Credential 或 Deployment；该职责继续属于既有初始化器和 MCP 编排。

## User scenarios

1. 运维人员在已选 Project 下首次创建或轮换 `runtime_env` Credential，以键值方式输入新的值；该值持久化到 Credential，页面可查看、编辑和导出值及可引用键名，而不是依赖手工 JSON 或猜测 data key。
2. 运维人员编辑未发布 Version 时，为 Component 选择同 Project 的 `runtime_env` Credential 与 data key，将其映射到目标环境变量。后续从 Service 或 Application 发起部署时，平台自动读取已保存的 Credential 并复用引用，不要求重复输入值。
3. 运维人员在 Version 详情审查任意 Component 的镜像、命令、挂载、健康检查、资源、restart、tmpfs、ulimits、非秘密环境变量及秘密引用，并能在支持的编辑状态修改它们。
4. 运维人员在 Service 详情的运行时环境变量卡片中查看当前绑定 Version 的 Component、目标环境变量、Credential、data key 和当前持久化值；并在已有 Gateway、Application、Version、Service 与 Deployment 详情中查看入口方式、资源约束、实例、Deployment 终态、错误和日志，且可在相关资源之间跳转。

## Acceptance

1. Credential 页面支持后端已接受的 `runtime_env` 类型，并以结构化 key/value 编辑器校验运行时环境变量键；创建、更新、列表、详情和导出的 UI 文案与行为清晰区分该类型，详情与导出允许读取已保存的值。
2. 对 `runtime_env`，Web 提供实际值与可引用键名的查看、编辑和导出路径；值在首次配置或轮换时持久化到 Credential，后续部署自动复用，Deployment 表单不得要求再次填写同一值。
3. Version Component 编辑和详情覆盖 `secret_env_refs`、command / args、healthcheck、resources、restart policy、tmpfs 与 ulimits；所有操作使用现有或新增的结构化 HTTP 契约，并保留后端验证。
4. 对任意已有 Application，既有页面可正确展示 Version 的高级运行时字段、Expose、Gateway 关联和 Deployment 结果；前端不从 fixture 生成或批量创建资源。
5. Service 详情增加运行时环境变量卡片。卡片按当前 `service.version_id` 解析 Component 的 `secret_env_refs` 并显示 Component、env key、Credential、data key 和当前 Credential value；无引用时显示空状态。
6. 卡片表示“当前持久化配置，供下一次部署使用”，不宣称读取正在运行容器的环境变量；Credential 值轮换后、重新部署前应明确提示该语义。
7. Web 不直接访问 MCP stdio 或 Docker；部署结果仅基于既有 Deployment / Service HTTP 读模型展示。
8. 对既有 Application / Version 的创建、预览、发布、部署和 Credential 使用方式保持兼容；高级字段为空时不改变既有渲染结果。
9. 前端覆盖新增表单校验、秘密引用选择、运行时字段序列化、Credential 值展示/导出、Deployment / Service 展示与资源跳转；后端覆盖 Web 契约、授权和运行时配置解析。

## Open questions

暂无需要用户确认的未决事项。

## Capability coverage

| 能力 | Web 处理 | RAGFlow MCP 部署关系 |
|---|---|---|
| `runtime_env` Credential 和持久化复用 | Credential 页面创建、查看、编辑、导出；Service 卡片展示当前持久化配置 | 必需 |
| `secret_env_refs` | Version Component 绑定、详情与 Service 卡片展示 | 必需 |
| command / args、healthcheck、resources、`restart_policy`、`tmpfs_json`、`ulimits_json` | Version 高级配置编辑与详情展示 | 必需 |
| Gateway 配置、Expose、Service / Deployment 状态与日志 | 复用既有 Gateway、Application、Version、Service 和 Deployment 页面 | 必需 |
| `orbit_list_gateways`、`orbit_create_gateway`、`orbit_get_gateway` | 既有 Gateway HTTP 页面已覆盖其业务对象管理 | 支撑部署，但不是新增 Web 工作流 |
| `runtime_doctor`、`verify_deployment`、`runtime_http_probe` | 保持 MCP 专用，不新增 Web 调用、独立记录或业务页面 | 用于部署验收，不阻塞 Web 展示 |

## Decisions

1. 本任务采用标准模式 / `standard`：Requirement -> Plan -> Implementation -> Verification；不单独创建 Spec。
2. 通用 Version Component 运行时配置属于 Application / Version / Credential 产品能力；RAGFlow 只作为验收样本，不新增 RAGFlow 专用编排或资源创建向导。
3. Deployment 和 Service 是部署结果的业务记录；前端复用其状态、错误和日志，不为预检、运行时探测或一致性检查新增独立业务资源。
4. 浏览器不得成为 MCP 或 Docker 的调用方；本任务不新增由页面触发的运行时验证能力。
5. `runtime_env` 是持久化 Credential 配置，不是每次 Deployment 提交的临时参数。Version 保存引用，后续 Service / Application 部署读取当前 Credential value 并注入运行环境；值轮换后重新部署生效。
6. `runtime_env` 值允许在 Credential 详情与导出中读取，并在 Service 详情的运行时环境变量卡片中展示。卡片读取当前持久化配置，不读取容器进程环境。

## Risk

1. Web 直接复刻初始化器或从 fixture 创建资源，会导致资源重复创建、恢复语义不一致或运行时真实状态漂移。
2. Credential 值轮换后、重新部署前，Service 卡片的当前持久化配置会与运行中容器的旧值不同；卡片必须明确其“下次部署配置”语义。
3. 当前 Application、Version、路由和前端导航文件存在并行开发修改。实施前必须与相关 agent 协调文件归属，避免覆盖尚未提交的调整。
4. Elasticsearch 的资源、tmpfs 和 ulimit 在不同 Docker context 上行为不同；RAGFlow 已暴露这一类风险，但 UI 只能表达配置和现有 Deployment 结果，不能替代后端 Preview 与 Deployment。

## User review notes

用户于 2026-07-26 在完成 RAGFlow 部署验证后的 Web 缺口评估后，要求以标准模式记录本任务。用户进一步澄清：RAGFlow 只是验证本项目新增部署能力的样本，不是本任务的业务目标。本 Requirement 为新任务，不修改既有 `20260726-ragflow-split-deployment` 或 `20260726-orbit-deployment-capability-gaps` 过程文档。

用户于 2026-07-26 接受本 Requirement 及其能力覆盖边界：Web 展示部署配置与业务记录，MCP 保留运行时诊断和验收职责。
