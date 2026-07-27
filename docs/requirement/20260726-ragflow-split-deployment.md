# RAGFlow 拆分应用部署需求
最后修改时间: 2026-07-27 10:29:04

## Review status

Accepted

## Background

RAGFlow 源码位于 `D:\SourceCodes\opensource\ragflow`。其 `docker/docker-compose.yml` 通过 `docker-compose-base.yml` 组织运行时服务；默认 CPU 配置选择一个向量引擎，并依赖 MySQL、Redis、MinIO 和该向量引擎。

本需求要使用 Pomelo Orbit 的 Application / Version / Deployment 模型及其本地 stdio MCP Server 部署 RAGFlow。依赖服务不能与 RAGFlow 放在同一个 Application 或 Version 中，必须各自成为独立的 Application 和 Version。

本需求是任务二「RAGFlow 拆分部署」。任务一「既有实现的功能补充」仍单独记录在 `docs/requirement/20260726-orbit-deployment-capability-gaps.md`；本需求消费其 Version 与 Credential 契约，并同时拥有为该部署编排所需的 Gateway MCP、受管 `traefik` 网络预检和固定 HTTP Probe。

后续用户授权在同一 `ragflow` Application 新增一个聚合 Version，用于将五个组件作为一个 Compose 项目运行。它是既有拆分 Version 的补充，不替换、更新或删除拆分 Version；聚合 Version 中的所有 logical mount 必须按组件命名空间隔离。

## Goal

1. 在同一 Project、Docker context 和实例键中部署独立的 RAGFlow、MySQL、Redis、MinIO 和一个选定的向量数据库 Application。
2. 每个 Application 维护独立的 Version、组件规范、持久化目录、健康检查和部署记录。
3. 仅通过 Orbit HTTP 生命周期工具执行创建、发布、部署、等待和读取；MCP Docker 工具仅用于受管范围内的只读诊断与一致性验证。
4. 使用标准 Application 的共享 `traefik` Docker 网络和稳定别名，使 RAGFlow 连接各依赖，而不依赖原 Compose 的同项目服务名。
5. 在实施前完成方案审查，并记录环境移除、任务一能力的前置依赖、凭据策略、目标部署上下文及回滚边界。
6. 通过 MCP 创建或读取受管 Gateway，并只从已部署的 `kind=gateway` target 派生、验证固定的 bridge `traefik` Docker 网络。
7. 在 RAGFlow 部署后，以已由受管 Compose `ps` 返回的组件容器执行固定 HTTP Probe；不接受手写 readiness proof 或任意 Docker 命令。
8. 聚合 Version 的 MySQL、Redis、MinIO、Elasticsearch 和 RAGFlow 日志目录互不共享，防止不同服务将不兼容的数据写入同一宿主目录。

## Non-goal

1. 本阶段不创建实际 Orbit Application、Version、Service 或 Deployment，也不启动或停止实际 RAGFlow 容器；允许实现本需求明确限定的 MCP Gateway、运行时预检和 HTTP Probe 代码及测试。
2. 不直接导入或执行 RAGFlow 原始 Compose 文件，不使用 Docker CLI 生命周期写命令。
3. 不部署 RAGFlow 的可选 DeepDoc、TEI、NATS、Kibana、Sandbox 或未选中的向量引擎；若后续启用，另行评估为独立 Application。
4. 不在 Version、过程文档、日志或 MCP 请求摘要中写入实际密码、JWT 或其他凭据。
5. 不新增任意 Docker CLI、任意 network name、任意 Compose YAML、任意镜像命令或任意宿主机挂载的 MCP 接口；初始化器不自动创建、更新或删除 Gateway。

## User scenarios

1. 运维人员通过 MCP 创建并发布五个独立 Application Version，按依赖顺序部署，并能读取每次 Deployment 的状态、命令与日志。
2. RAGFlow 容器通过共享网络上的稳定别名访问 MySQL、Redis、MinIO 和向量数据库；数据分别保存在 Orbit 受管工作目录中。
3. 运维人员可用 `runtime_compose_config`、`runtime_compose_ps`、`runtime_compose_logs` 和 `verify_deployment` 验证每个受管 Application 的实际状态与渲染结果。
4. 运维人员通过 MCP 创建并部署 Gateway 后，初始化器以该 Gateway target 验证固定 `traefik` 网络存在且为 bridge 网络；缺失或类型不符时 fail-fast。
5. RAGFlow Deployment 完成后，初始化器对 `ragflow-cpu` 的固定端点运行受限 HTTP Probe，并仅把成功/失败、端点和 resource ID 写入无秘密运行记录。

## Acceptance

1. 选定向量数据库后，五个 Application 的应用 code、组件、镜像、命令、环境变量、挂载、健康检查、资源及 Expose 在 Spec 中完整列出，且不包含秘密值。
2. RAGFlow 的连接配置使用 Orbit 生成的跨应用别名 `<app-code>-<component>`，不使用原 Compose 的 `mysql`、`redis`、`minio` 或 `es01` 等内部服务名。
3. 有状态组件使用受管的逻辑目录挂载；其数据目录、初始化文件和文件更新策略可审计且可备份。
4. 部署顺序先满足共享网络和依赖健康，再部署 RAGFlow；每次部署均有终态等待和受管运行时核验。
5. 实施前明确记录并接受任务一交付的能力契约，以及跨 Application 启动依赖与凭据注入方案。
6. 所有写操作在用户接受 Plan 后才执行。
7. Gateway MCP 创建/读取只调用既有 Orbit HTTP `/api/gateway`，响应不包含 Credential value。
8. `runtime_doctor` 只有在提供 Gateway Application / instance 时才验证固定 `traefik` 网络，拒绝非 Gateway target，且只报告从该 target 派生的网络状态。
9. HTTP Probe 仅对目标 Compose `ps` 返回的 service container 执行固定 `curl`；非法 port/path、Docker 或 curl 失败均不回显 stdout / stderr。
10. 单元测试覆盖 Gateway MCP 映射、非 Gateway 拒绝、缺失网络、Probe argv 约束、失败脱敏和初始化器的 Gateway 参数及依赖顺序。
11. 聚合 Version 保留拆分 Version，并将 logical mount 分别编码为 `mysql/data`、`redis/data`、`minio/data`、`es01/data` 和 `ragflow-cpu/logs`。

## Open questions

1. 向量数据库选 Elasticsearch（与当前 RAGFlow 默认配置一致）、OpenSearch、Infinity、OceanBase 或 SeekDB 中的哪一种？本需求只能选择一种。
2. 使用哪个 Orbit Project 和 instance key？RAGFlow 对外访问是经现有 Gateway 的 public HTTP，还是仅映射为本机 local port？
3. 部署时的数据库、Redis、MinIO 和向量数据库凭据由谁生成、保存和轮换？当前 `runtime_config` 会被持久化到 Deployment options，不能假定它适合秘密值。
4. 是否将 Docker daemon 重启后的自动恢复视为本次上线必需条件？当前 VersionComponent 模型没有 `restart: unless-stopped` 字段。
5. 若选择 Elasticsearch，任务一补齐 `tmpfs: /tmp` 与 `ulimits.memlock` 后再部署；若选择其他向量引擎，需在其 Spec 中明确是否仍依赖任务一能力。
6. RAGFlow 是否需要启用 Go / hybrid API、NATS、TEI、DeepDoc、GPU 或 Sandbox？默认 CPU + Python 配置不需要它们。

## Decisions

1. 流程模式采用严格模式 / `strict`：Requirement -> Spec -> Plan -> Implementation -> Verification。
2. 标准 Application 共享外部 Docker 网络 `traefik`；组件运行别名为 `<app-code>-<component>`。跨 Application 依赖通过该别名连接，不能使用 `depends_on`。
3. RAGFlow 采用官方镜像内置的 `entrypoint.sh` 和 `service_conf.yaml.template`，不挂载本地源码副本；Orbit 当前逻辑文件挂载创建模式为 `0600`，不能安全覆盖一个必须可执行的 entrypoint。
4. 原 Compose 的 `include`、`profiles` 和 `env_file` 不直接导入。每个目标服务以独立 Application Version 的结构化字段表达。
5. Gateway 的 `traefik` 网络由其 Orbit Compose 定义创建；MCP 不直接执行 `docker network create`，`traefik` 也不是调用方可传入的参数。
6. HTTP Probe 固定使用从目标 Compose `ps` 派生的 container ID、`curl -fsS --max-time 10 --output /dev/null` 和 `http://127.0.0.1:<port><path>`；不运行 shell。
7. 聚合 Version 复用拆分 Fixture 的镜像、环境与 Credential 引用，但在创建时重写 logical mount source 为 `<component>/<source>`；参考 Compose 使用对应的 `./data/<component>` bind mount。

## Risk

1. 当前平台将所有标准 Application 连到共享 `traefik` 网络；该网络须先由受管 Gateway 创建且必须在 Docker context 中可见，否则所有标准应用部署失败。
2. `depends_on` 只能引用同一 Version 的组件，不能等待其他 Application 健康；部署编排和后续连通性检查必须承担该责任。
3. Task 1 未完成或其能力契约未被接受时，任务二不能实施任何依赖该能力的服务。
4. `source_type=volume` 目前不会在渲染结果中声明顶层 Compose volume；有状态服务应暂以逻辑目录挂载实现，除非平台补齐 named-volume 声明。
5. 默认示例配置包含开发用默认凭据，不能用于实际部署。
6. RAGFlow 依赖启动时建表和可选模型迁移；必须在 Spec 中明确初始化幂等性及失败恢复方式。
7. Gateway 尚未发布或部署、镜像中缺少 curl、或候选端点非成功都会阻断 RAGFlow 就绪；这是预期的 fail-fast 行为，诊断仅使用既有脱敏日志。
8. 同一 Application 的组件共享物理 Service 目录；若聚合 Version 使用裸 `data` 或 `logs` source，多个组件会共用该目录并可能损坏有状态服务数据。

## User review notes

用户于 2026-07-26 要求进入 Task 2 的 Spec / 规格阶段。环境移除任务由独立 agent 负责；未决的向量引擎、目标 Project、实例键、入口和凭据实际值作为 Spec review 项，不授权实施。

用户于 2026-07-26 要求将 `20260726-runtime-preflight-probe` 合并入本任务；其 Gateway MCP、受管 `traefik` 预检与固定 HTTP Probe 的需求、约束和验收项自此由本需求拥有。

用户于 2026-07-27 要求保留既有拆分 Version，并新增聚合 RAGFlow Version；随后确认共享 `data` 目录会导致 MySQL 启动失败，要求对聚合布局使用组件独立目录并完成受管部署验证。
