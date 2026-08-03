# RAGFlow 部署工作流能力改进需求
最后修改时间: 2026-08-03 12:29:10

Review status: Accepted

流程模式: 标准 / standard

## Background

在一次明确要求的 RAGFlow split 部署中，`deploy-ragflow-split-orbit` skill 依次完成主机与 TEI 前置检查、Orbit 资源准备、Gateway 初始化、后端依赖服务启动、RAGFlow split Service 启动和运行时验证。过程表明，skill 的步骤边界基本存在，但 MCP 的网络可用性断言、Gateway 初始化配置和部署发现仍不足以让调用方在不依赖 MCP 外部写操作的前提下完成闭环。

当前观测到 Gateway、MySQL、Valkey、MinIO、Elasticsearch 和 TEI 已可用；`ragflow-split/ragflow-cpu` 在当前主机上因磁盘性能较低而长期执行 `init_database_tables`，因此保持 `starting`，且容器内 `80` 尚未监听。这是正常的长初始化，现有健康检查已能准确表达该状态；本需求不扩展健康检查或为此增加诊断能力，也不在本需求阶段修改运行中资源。

部署时同时发现，现有 `runtime_doctor` 只依据 `traefik` 网络存在且为 bridge 网络判断可用；Gateway 初始化则使用固定默认 payload，skill、MCP 与用户没有统一的配置覆盖入口。

## Goals

- 让 MCP 断言 `traefik` 网络可供 Gateway 与应用连接；网络已存在时直接复用，缺失时由 Gateway 初始化创建。
- 将 Gateway 视为带默认模板的普通 Application：skill、MCP 和用户可在发布前通过受校验的配置覆盖 Application、Version、Component 与 endpoint 默认值。
- 让调用方能够按 Application、Service 或 Deployment 范围盘点部署，完成选定拓扑启动前的运行态检查。
- 收敛 split 与 integrated 部署的配置权威来源，确保各自 skill、Compose 和 primitive 对运行时变量及依赖地址的声明一致。
- 为 TEI 和 GPU/CPU 选择增加有界、可机器读取的前置检查。
- 将 skill 的操作边界明确为只读取当前 Compose 与 primitive 定义，不读取、使用或引用 `docs/archive/` 或导出的 SQL/SQLite 数据库基线。

## Non-goals

- 不在本需求中修复 RAGFlow `init_database_tables` 的应用初始化问题，除非后续诊断证明该问题位于 Orbit 或 MCP 控制面。
- 不改变 RAGFlow split 的 Application 拆分、CPU/GPU Version 选择规则或用户已有的数据目录。
- 不通过 Docker Compose CLI 执行部署、停止、重启或网络生命周期操作。
- 不自动重启、删除、迁移或修改当前运行的 Gateway、Service、容器、卷和开发服务器。

## User scenarios

1. 操作者在部署前发现名为 `traefik` 的网络时，MCP 能报告其存在性、driver、scope 和可用结论；网络可用时直接复用，缺失时由 Gateway 初始化创建，不需要 MCP 外的 Docker 网络写操作。
2. 操作者初始化 Gateway 时，可在默认 Application、Version、Component 与 endpoint 模板上声明覆盖配置；网络存在时直接使用，缺失时初始化 Gateway 创建网络。
3. 操作者启动 split 拓扑前，可列出 `ragflow`、`ragflow-split` 及目标 Service 的活动 Deployment，并据此确保同一宿主机上只运行被选定的 RAGFlow Service。
4. 操作者查看 `ragflow-split/default` 的现有健康状态时，可区分正常的长初始化与不健康状态；当前数据库表初始化期间的 `starting` 状态无需扩展健康检查或新增诊断字段。
5. 操作者使用 skill 选择 CPU 或 GPU 时，TEI 预检在受控时限内返回缓存或镜像命中情况、GPU 检测、预检结论和下一步；镜像清单查询不会无限拖延工作流。

## Acceptance criteria

- `runtime_doctor` 或专用只读工具对 `traefik` 返回存在性、driver、scope 和可用结论；满足 Gateway 连接条件的现有网络直接复用，不要求由 Orbit 创建或标记为受管。
- 网络缺失时，`orbit_provision_gateway` 初始化并部署 Gateway 以创建可用网络；网络不满足连接条件时返回阻塞结论，不删除、重命名或协调现有网络。
- `orbit_provision_gateway` 对可用的现有 `traefik` 网络直接复用；网络缺失时按默认模板创建 Gateway Application、Version、Service 和网络。它返回创建或复用的资源 ID，但不引入阶段化续接协议。
- Gateway 初始化先创建默认 Version 模板，再合并显式配置覆盖。默认的 Traefik API endpoint 为 `http` local `127.0.0.1:8080 -> 8080`，但它和其他默认 Component/endpoint 字段均可在 draft Version 中通过 skill、MCP 或用户操作覆盖；后续供应或重新编译不得静默覆盖已声明的值。
- MCP 提供按 Application、Service 和 Deployment 查询 Deployment 的能力，并返回足以判断活动、终态、目标 Version 与创建时间的摘要。
- 既有健康检查继续作为组件启动状态的判断依据；当前因磁盘性能产生的数据库表初始化长启动不要求新增等待语义、端口检查或运行时诊断字段。
- split 与 integrated Compose、部署 primitive 和 skill 对 `MYSQL_ROOT_HOST`、`TEI_BASE_URL` 及其他运行时变量的定义具有单一、机器可读的权威来源；验证能检测两种拓扑中这三者的漂移。
- `prepare_ragflow_tei.py check` 为镜像清单与本地缓存检查设置明确超时，允许使用有限缓存，并输出机器可读取的检查项、耗时、缓存命中和阻塞原因。
- 更新后的 split skill 明确要求在任何 Orbit 生命周期写入前完成硬件检测和所选 profile 的 TEI 预检；检测到 NVIDIA 但 GPU 预检失败时仍然阻止部署。
- 更新后的 skill 明确禁止读取、使用或引用 `docs/archive/` 与导出的 SQL/SQLite 数据库基线；部署配置只以当前 Compose 和 primitive 定义为准。

## Open questions

- `traefik` 网络的最小可用断言应只要求 driver/scope，还是还应检查 Docker attachability；需要以 Gateway 的实际 Compose 网络声明确定。
- Gateway 初始化规格应覆盖哪些受校验字段，及已发布 Gateway 被重新初始化时应新建 draft Version 还是要求调用方显式创建 Version；需要在实现计划中确定。
- 当前主机的磁盘性能基线及数据库表初始化耗时是否需要作为独立运行手册信息记录；这不影响现有健康检查的语义。

## Decisions

- 本需求采用标准 / standard 流程；用户明确要求进入 Spec，因此按 Requirement -> Spec -> Plan -> Implementation -> Verification 推进。
- 本阶段只记录问题、边界和验收标准；用户已接管后续 RAGFlow 运行流程，不执行任何部署、重启或运行时修改。
- MCP 负责受控的部署生命周期与状态发现；skill 负责按当前配置编排 MCP 调用及前置检查，不承担 MCP 外部补救操作。`traefik` 是可复用的共享网络能力，不以 Orbit 归属作为前置条件。
- Gateway 的 Application kind 只提供可演进的默认模板，不构成配置特权；Version、Component 与 endpoint 的显式覆盖优先于默认值。
- 部署配置以 topology-specific 的当前 Compose 契约及其对应 primitive 为权威来源；历史归档和导出的数据库基线不是操作依据。
- 选择 split 或 integrated 拓扑前，必须通过 Orbit 停止另一 RAGFlow Service，且不删除其卷；RAGFlow 通过 Gateway HTTP endpoint 提供访问。

## Risks

- 现有 `traefik` 网络若不满足 Gateway 实际连接条件，部署会被前置检查阻止；工具不能尝试删除、重命名或改造该网络。
- 增加部署查询、阶段化供应和诊断字段可能影响现有 MCP 客户端的 schema 与响应解析，需要版本化或兼容策略。
- 当前主机的磁盘性能会延长数据库表初始化，导致 `starting` 状态持续更久；现有健康检查应继续如实反映该过程，不应由控制面强行缩短或改变其语义。
- 多份运行时变量定义若未被真正收敛，skill 的检查可能只发现静态差异，仍无法保证容器的最终有效配置。

## User review notes

- 2026-08-03：用户要求以标准 / standard 模式记录本次 split 部署中暴露的 skill 与 MCP 能力不足，并接管后续运行流程。
- 2026-08-03：用户确认 `ragflow-cpu` 卡在数据库表初始化由当前主机磁盘性能导致，现有健康检查已满足要求；不将健康检查、等待语义或该问题的扩展诊断列为改进项。
- 2026-08-03：用户明确要求开始 Spec；Requirement 视为已接受。
- 2026-08-03：用户要求 Gateway 默认配置可由 skill、MCP 或用户覆盖，不能继续以硬编码方式取代普通 Application/Version 配置能力。
- 2026-08-03：用户要求简化 Gateway 网络模型：现有可用 `traefik` 网络直接复用，缺失时初始化 Gateway；不再设计网络归属判定或网络协调操作。
- 2026-08-03：用户确认 Gateway 待办只聚焦初始化：复用可用网络或按默认模板创建，并允许配置覆盖；不实现阶段化续接或操作记录。
- 2026-08-03：用户要求将同一配置收敛能力扩展到 integrated；两种拓扑使用共享渲染器和独立契约，分别保留其数据目录、依赖关系和网络别名。
