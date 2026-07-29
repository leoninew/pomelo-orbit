# MCP 部署体验改进计划
最后修改时间: 2026-07-29 16:22:12

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

- [MCP 部署体验改进需求](../requirement/20260729-mcp-deployment-ergonomics.md) 已于 2026-07-29 接受。
- 本计划只实现 MCP 部署与诊断体验改进；敏感信息脱敏按已确认边界延期，不在本轮改变其行为。
- 保持通过 Orbit HTTP API 完成所有生命周期写操作；不使用 Docker Compose CLI 执行 `up`、`down`、`restart` 等生命周期写入。

## 实施步骤

### 1. 新增高层 Gateway 供应工具

修改 `mcp/src/pomelo_orbit_mcp/tools/orbit.py`，在保留 `orbit_create_gateway` 的前提下增加 `orbit_provision_gateway`。它是显式的、确保 Gateway 可运行的高层工作流；低层工具仍用于仅创建或单独更新 Gateway 配置。

1. 输入以 `project_id` 为必填，沿用现有 Traefik 默认配置；支持 `instance_key`（默认 `default`）、`runtime_config`（默认空对象）、`force_recreate` 和 `timeout_seconds`。不接受任意 Docker 命令、网络名或未定义的生命周期动作。
2. 按 `project_id` 和 Gateway `code` 查询既有 Gateway：零个时调用既有创建接口；一个时复用；多个时返回结构化冲突错误，要求调用方先消除歧义，绝不静默创建或任选一个。
3. 读取复用/新建 Gateway 的 Versions 和 Services，选择其当前可部署 Version；若 Version 未发布则发布。若同一 `instance_key` 的 Service 不存在，创建它；存在时复用。复用 Service 时绝不覆盖已有 runtime configuration；调用提供非空 `runtime_config` 时返回结构化冲突并指向已有的显式更新工具。
4. 通过既有部署、等待工具的底层 `OrbitClient` 调用创建 Deployment 并等待终态。失败终态、超时和 API 错误保留现有 MCP 错误/领域结果边界，不继续执行网络就绪检查。
5. 使用受控的只读网络检查确认 `traefik` 为存在的 bridge 网络，返回 `ready`、资源 ID、是否创建/复用以及简短的步骤结果。默认不返回 Gateway 配置、runtime configuration、完整 Deployment command 或 Docker inspect。

实现时提取不依赖 FastMCP 装饰器的私有编排函数，以便直接单元测试每个分支，并避免高层工具通过再次调用 MCP tool wrapper。

### 2. 开放受限的无目标外部网络读取

修改 `mcp/src/pomelo_orbit_mcp/docker_runtime.py` 与 `mcp/src/pomelo_orbit_mcp/tools/runtime.py`。

1. 在 `DockerRuntime` 增加单独的只读 external-network inspect 路径，只接收经过长度与字符规则校验的单个网络名，并执行固定的 `docker network inspect <name>`；不从用户输入拼接 shell，不枚举 Docker 网络，也不返回原始 inspect。
2. 扩展 `runtime_doctor` 以接受显式的 `network_name`，使其在没有 Application/Service target 时仍能报告 Docker context、Compose 可用性、数据目录和该网络的存在性、driver、scope。保留现有 Application target 与 Gateway target 参数组合及互斥校验。
3. 对 `network_name` 只允许 `traefik`，避免把面向固定部署前置条件的工具扩展为任意 Docker 资产浏览器。缺失、非 bridge、Docker 不可用或 context 不匹配分别产生可操作的 `issues`，且不得误报为 `healthy`。
4. 保留 `runtime_network_inspect` 的 target-derived 限制，作为已部署受管 Compose 项目的深度诊断路径；新能力不降低既有容器与网络访问边界。

### 3. 将运行时与验证输出改为摘要优先

修改 `mcp/src/pomelo_orbit_mcp/docker_runtime.py`、`mcp/src/pomelo_orbit_mcp/tools/runtime.py`、`mcp/src/pomelo_orbit_mcp/verification.py` 和 `mcp/src/pomelo_orbit_mcp/tools/verification.py`。

1. 为 `runtime_compose_ps` 增加 `detail: bool = false`。默认将每个 Compose 条目投影为稳定的摘要字段：`container_id`、`service`、`state`、`health`、`restart_count`、`ports`；完整 Compose JSON 仅在 `detail=true` 时返回。
2. 为 `verify_deployment` 增加 `detail: bool = false`。验证算法仍内部收集 Compose 与 inspect evidence，用于得出差异和稳定性结论；默认仅返回 `conclusion`、`differences`、组件健康摘要、端口/网络约束结果和稳定性摘要。仅 `detail=true` 返回完整 evidence。
3. 摘要构造采用集中 helper，并为 Docker/Compose 字段缺失或非预期形状设置确定的空值表示，避免不同工具各自猜测健康与重启次数。
4. `runtime_compose_logs`、`runtime_container_inspect`、`runtime_network_inspect` 和 `runtime_compose_config` 继续是明确请求详情的工具，本轮不移除。这样已依赖它们的诊断调用不受默认摘要策略影响。

### 4. 建立 MCP schema 与运行版本同步防线

修改 `mcp/src/pomelo_orbit_mcp/server.py`、`mcp/tests/test_server.py`、`.codex/config.toml` 及 MCP 使用文档。

1. 把工具表的预期名称和关键 input schema 断言集中为测试期望，覆盖新增 `orbit_provision_gateway`、`runtime_doctor(network_name)`、`runtime_compose_ps(detail)` 和 `verify_deployment(detail)`；同时明确完整组件保存工具不在注册表中。
2. 为 Server 提供可读取的构建/源码版本指纹，并在启动时记录到 MCP instructions 或健康描述，以便调用方能判定其连接的 stdio 进程版本。指纹不含本地绝对路径、配置或认证信息。
3. 在 `mcp/Makefile` 的 `check` 路径中运行 server-schema 测试。确认 `.codex/config.toml` 仍以当前工作树的 `uv --directory mcp run pomelo-orbit-mcp` 启动；文档明确新增/变更工具后需重启 MCP client 会话，因为 stdio server 不支持热加载。
4. 不假设客户端可强制重启进程；若当前会话工具清单和源码指纹不一致，调用方必须建立新会话后再进行写操作。

### 5. 更新部署 skill 与操作文档

修改 `skills/deploy-ragflow-orbit/SKILL.md`、`mcp/README.md` 和 `docs/guides/mcp-direct-operations.md`。

1. RAGFlow bundled 流程先以 `runtime_doctor(network_name="traefik")` 做无目标预检；缺失时使用一次 `orbit_provision_gateway`，再继续 RAGFlow 生命周期操作。
2. 文档区分 `orbit_create_gateway`（低层创建）与 `orbit_provision_gateway`（创建/复用到网络就绪），并说明后者隐含的步骤是该工具的明确契约，而非通用工具的隐式串联。
3. 记录摘要默认值和 `detail=true` 的获取方式，以及 MCP stdio server 的重启/版本同步要求。保持不展示 runtime configuration、凭据或原始运行时配置。

## 涉及文件

| 路径 | 计划改动 |
| --- | --- |
| `mcp/src/pomelo_orbit_mcp/tools/orbit.py` | Gateway 高层供应编排与摘要结果。 |
| `mcp/src/pomelo_orbit_mcp/orbit_client.py` | 仅在高层编排缺少现有 HTTP 读写调用时补充薄封装；不新增控制面业务 API。 |
| `mcp/src/pomelo_orbit_mcp/docker_runtime.py` | 固定外部网络只读检查与 Compose 摘要投影所需原始数据。 |
| `mcp/src/pomelo_orbit_mcp/tools/runtime.py` | `runtime_doctor(network_name)` 与 `runtime_compose_ps(detail)` MCP schema/输出。 |
| `mcp/src/pomelo_orbit_mcp/verification.py` | 验证 evidence 的摘要投影和稳定性/约束摘要。 |
| `mcp/src/pomelo_orbit_mcp/tools/verification.py` | `verify_deployment(detail)` schema 和默认响应。 |
| `mcp/src/pomelo_orbit_mcp/server.py` | Server 构建版本指纹和注册一致性支撑。 |
| `mcp/tests/test_server.py` | 完整工具表、关键 schema、无旧完整组件保存工具的回归断言。 |
| `mcp/tests/test_orbit_client.py` | Gateway 查询/创建/Service/Deployment 调用的 HTTP 契约测试（需要时）。 |
| `mcp/tests/test_docker_runtime.py` | 无目标网络存在、缺失、driver 非 bridge、名称校验与 Compose 摘要测试。 |
| `mcp/tests/test_runtime_tools.py` | 运行时工具参数组合、`network_name` 边界和默认/详情响应测试。 |
| `mcp/tests/test_verification.py` | 默认摘要、`detail=true` evidence、失败/漂移/稳定性边界测试。 |
| `mcp/Makefile`、`.codex/config.toml` | schema 检查和 stdio 启动一致性（仅在实现发现当前入口不足时修改）。 |
| `skills/deploy-ragflow-orbit/SKILL.md`、`mcp/README.md`、`docs/guides/mcp-direct-operations.md` | 新工作流、重启要求与摘要/详情契约说明。 |

不计划修改 Go 生产代码。`internal/application/gateway/usecase/service.go` 已在创建时编译 Gateway Version，`internal/application/gateway/usecase/deployment.go` 已约束标准应用须有运行中的 Gateway；本轮以 MCP 编排既有 `/api/gateway`、Service、Version、Deployment 接口为准。若实施中发现无法表达所需的幂等查询或出现跨调用并发竞态，停止在实现阶段并先补充需求/计划，再考虑新增 Go 原子用例与 HTTP API。

## 验证计划

1. 运行 `make -C mcp check`，覆盖锁文件、Ruff、格式、mypy 和默认 pytest。
2. 单元测试 Gateway provisioning 的新建、唯一复用、多个同 code 冲突、已发布/未发布 Version、已有/缺失 Service、失败终态和网络不就绪分支；断言失败后不继续后续写入。
3. 单元测试无 target `runtime_doctor(network_name="traefik")` 的健康、缺失、非 bridge、名称非法与 Docker 错误传播；确认 target-derived `runtime_network_inspect` 的原有限制不变。
4. 对 `runtime_compose_ps` 和 `verify_deployment` 分别断言默认响应不含原始 Compose/inspect/labels/挂载/命令，`detail=true` 才含完整诊断 evidence；覆盖字段缺失与停止容器。
5. 断言 `create_server().list_tools()` 与预期表完全一致，并校验关键参数默认值/可选性；在新 stdio 会话中确认已注册 schema 与测试结果一致。
6. 在不触发 Docker 生命周期写操作的前提下，按现有受控 Docker 集成测试约定运行 `make -C mcp test-docker`（仅当隔离 workspace、项目和显式环境变量齐备）。

## 兼容性与发布

- 新工具是增量 API；`orbit_create_gateway`、`orbit_deploy`、`orbit_wait_deployment` 及既有运行时详情工具保持可用。
- `runtime_compose_ps` 与 `verify_deployment` 的默认输出将收窄，这是有意的兼容性变更。所有旧字段通过同一工具的 `detail=true` 保持可访问；发布说明应列出该迁移方式。
- 新工具只会重用唯一匹配的 Gateway 和 instance-key Service，不更新已有 Gateway 配置、不改写非空 Service runtime configuration、不删除资源。
- 部署新 MCP 后必须重启 stdio server/client 会话；通过工具注册表与版本指纹确认生效后才使用新的写工具。

## 回滚

- MCP 工具或摘要契约异常时，回退到上一 MCP 包/工作树版本并重启 stdio server；Orbit 已创建的 Application、Version、Service、Deployment 不由回滚脚本自动删除。
- Gateway 供应在创建资源后失败时，返回已创建资源 ID 和失败步骤，供操作者用现有低层 MCP 工具诊断或显式清理；不执行隐式删除，避免破坏可能已被其他应用引用的 Gateway。
- 若默认摘要影响调用方，临时将调用切换为 `detail=true`，并用 `runtime_container_inspect`、`runtime_network_inspect` 或 `runtime_compose_config` 完成诊断；不恢复 Docker Compose 生命周期直连。

## 假设、风险与阻塞

- 假设现有 Gateway API 的 Version/Service 查询足以稳定识别唯一可部署对象；实施前用 API mock 固化该假设。
- 复用已经存在但状态异常的 Gateway 不能被高层工具静默“修复”，避免覆盖端口、TLS、入口点或运行时配置；工具应返回资源状态和可操作错误。
- Docker network 名限制为 `traefik` 是有意的最小权限边界。若后续产品支持多个 Gateway 网络，应形成独立需求后扩展白名单与授权模型。
- stdio client 缓存的 schema 无法由 Server 单方面刷新；本轮只提供可验证的一致性和明确的重启运行手册，不承诺热加载。
- 暂无阻塞项；本计划等待用户审阅和接受后才进入 Implementation / 实现。

## User review notes

- 2026-07-29：基于已接受需求创建计划草稿；新增高层 `orbit_provision_gateway`，保留 `orbit_create_gateway` 低层边界。
- 2026-07-29：用户要求开始 Implementation / 实现，计划自动接受。
- 2026-07-29：当前 Version Component 接口继续拆分；MCP 以现行 `basic`、`runtime`、`ports`、`env`、`mounts`、`dependencies`、`advanced` 路由和对应 JSON 字段为准，移除已删除的 `connectivity` 工具与 `networks` 输入字段。
