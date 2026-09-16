# MCP 部署体验改进需求
最后修改时间: 2026-07-29 14:20:23

Review status: Accepted

流程模式: 标准 / standard

## Background

通过 `pomelo_orbit` MCP 部署 RAGFlow 时，部署本身可完成，但控制面存在若干需要调用方理解内部实现的断点：Gateway 创建后仍需手工补齐 Service、发布 Version、创建 Deployment；没有受管目标时无法直接确认 `traefik` 外部网络；会话暴露的 MCP schema 可能滞后于工作区源码；运行时诊断默认返回过多 Compose 与 inspect 细节。

这些问题增加了 agent 和人工操作者的部署判断成本，并使部署技能需要承担本应由 MCP 工具完成的编排与输出筛选工作。

## Goals

- 将 Gateway 的创建到可用网络之间的生命周期收敛为可发现、可重复执行的 MCP 工作流。
- 允许在尚无受管应用目标时，通过 MCP 直接检查指定外部 Docker 网络的可用性。
- 使 MCP Server 实际注册的工具 schema 与当前源码、客户端调用契约可验证地保持一致。
- 为运行时状态和部署验证提供默认精简摘要，并保留按需获取诊断详情的能力。

## Implementation ownership

| 改进项 | MCP 实现 | Go 实现 |
| --- | --- | --- |
| Gateway 部署闭环 | 需要。MCP 编排现有的 Gateway、Service、Version 发布、Deployment 与等待 API，并负责返回摘要。 | 初版不必新增。现有 Go Gateway 用例已负责 Gateway 配置与 Version 编译；若需要跨调用原子幂等、并发去重或单个 HTTP API，则新增 Go 用例/API。 |
| 无目标的外部网络检查 | 需要。在 Python `DockerRuntime` 增加受限的指定网络只读检查，并由 `runtime_doctor` 或专用工具暴露。 | 不需要。该检查依赖 MCP 所在机器的 Docker context，而非 Orbit 控制面。 |
| MCP schema 与源码同步 | 需要。处理 stdio Server 的启动/重载、工具注册检查及 MCP 集成测试。 | 通常不需要。Go 仅维持 HTTP 接口契约与分组组件保存端点；若 API 契约变化，补充 Go handler/usecase 测试。 |
| 运行时与验证默认精简输出 | 需要。在 MCP runtime/verification 工具中定义 summary 与显式详情请求。 | 不需要。完整 Docker/Compose 数据由 MCP 本地诊断层收集和整形。 |

结论：本轮以 MCP 为主要实现面。Go 的必做项是确认并测试既有 Gateway、部署与分组组件 API 契约；只有把 Gateway 供应流程升级为控制面原子业务操作时，才扩展 Go 生产实现。

## Non-goals

- 本轮不处理 MCP 响应、部署记录或容器 inspect 中的敏感信息脱敏问题。
- 不改变 Orbit Application、Version、Service、Deployment 的领域模型和 RAGFlow bundled 五组件拓扑。
- 不改用 Docker Compose CLI 直接执行生命周期操作。
- 不在本需求中重构无关的 MCP 工具或 Web UI。

## User scenarios

1. 操作者部署一个需要 `traefik` 网络的标准应用时，可先读取网络状态；若网络缺失，可通过一个明确的 MCP Gateway 工作流创建并部署受管 Gateway，得到可供后续部署使用的结果与资源 ID。
2. 操作者调用 MCP 时，工具清单应反映当前服务所运行版本的真实 schema；组件保存接口改为分组接口后，旧的完整组件保存工具不应继续作为可用工具暴露。
3. 操作者检查运行状态时，默认只看到每个组件的服务名、状态、健康、重启次数和宿主机端口；需要分析故障时，才显式请求日志、Compose 或 inspect 详情。
4. 操作者完成部署验证时，默认得到结论、差异、稳定性及关键资源状态，而不是完整渲染 Compose 和全部容器 inspect 数据。

## Acceptance

- Gateway MCP 工作流能够在一次明确调用中完成创建或复用、Service 准备、Version 发布、Deployment 创建与终态等待，并返回 Gateway Application、Service、Version、Deployment ID 及网络可用结论。
- 对已可用 Gateway 的重复调用具有明确、文档化的复用或冲突语义，不会静默创建重复的同类 Gateway。
- `runtime_doctor` 或新增只读网络检查工具可在无 Application/Service target 时检查 `traefik` 等指定外部网络，并返回存在性、driver、scope 与可操作的问题信息。
- MCP 的已注册工具表有可执行的同步检查，能检测源码声明、Server 实际 schema 与测试期望之间的不一致。
- `runtime_compose_ps` 默认响应不包含完整标签、挂载、命令和 inspect 数据；调用方可通过明确参数或专用工具取得相应详情。
- `verify_deployment` 默认响应不包含完整 Compose 与全部 inspect evidence，但仍包含 `conclusion`、differences、组件健康、端口约束和稳定性结论；详细 evidence 必须显式请求。
- 已有 MCP 部署、停止、重启与范围限定诊断路径保持可用，相关测试覆盖新增行为与兼容边界。

## Open questions

- MCP Server 的源码/schema 同步应采用开发期自动重启、启动时版本指纹，还是 CI/测试防护；以当前 stdio Server 的启动模型和部署方式决定。
- 精简输出的详情请求采用统一 `detail` 参数，还是保持摘要工具与详情工具分离；以现有 MCP schema 可读性和调用安全性决定。

## Decisions

- 本需求采用标准 / standard 流程，按 Requirement -> Plan -> Implementation -> Verification 推进；无需单独创建 Spec。
- 敏感信息脱敏问题已识别，但按本次明确指令延期处理，不作为本轮验收阻塞项。
- Gateway 与外部网络属于部署前置能力，优先在 MCP 服务端完成，而不是继续把固定编排放入部署 skill 或调用方提示词。
- 新增 `orbit_provision_gateway` 作为高层确保型工作流，负责复用或创建、Service 准备、Version 发布、部署、等待和网络结论；保留 `orbit_create_gateway` 作为只创建 Gateway 配置与受管应用的低层工具。

## Risks

- Gateway 默认端口可能与开发机既有服务冲突，工作流需要把冲突作为可诊断结果而非误报为网络就绪。
- 改变 MCP 默认输出会影响依赖当前完整字段的调用方，需要明确 opt-in 详情路径和迁移策略。
- Server schema 与源码不同步的根因可能位于进程生命周期、打包缓存或 MCP client 会话缓存，修复需要覆盖本地开发与测试环境。

## User review notes

- 2026-07-29：记录 RAGFlow 部署后的 MCP 使用体验问题；敏感信息相关问题暂缓处理。
- 2026-07-29：接受需求，并采纳新增 `orbit_provision_gateway`、保留 `orbit_create_gateway` 的工具边界。
