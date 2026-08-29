# RAGFlow 部署工作流能力改进规格
最后修改时间: 2026-08-03 12:29:10

Review status: Accepted

流程模式: 标准 / standard

## Requirement basis

本规格落实 [RAGFlow Split 部署工作流能力改进需求](../requirement/20260803-ragflow-split-deployment-workflow.md)。需求已接受；用户已接管当前 RAGFlow 运行流程，因此本规格不包含对运行中资源的操作。

## Overview

实现分为四个边界清晰的部分：MCP 网络可用性断言、可配置的 Gateway 初始化、Deployment 盘点，以及由机器可读契约驱动的 split/integrated 配置和 TEI 预检。所有部署生命周期写入继续经 `pomelo_delivery` MCP 执行；skill 只编排 MCP 与本地只读前置检查。

既有 `runtime_compose_ps` 健康摘要和 RAGFlow/TEI health check 保持不变。数据库表初始化受主机磁盘性能影响而较长时，`starting` 仍是其正确状态，不增加新的等待工具、诊断字段或健康判定。

## Affected components

| 范围 | 责任 |
| --- | --- |
| `mcp/src/delivery-mcp/src/pomelo_delivery_mcp/docker_runtime.py` | 网络可用性断言。 |
| `mcp/src/delivery-mcp/src/pomelo_delivery_mcp/tools/runtime.py` | 暴露网络可用性的 MCP 只读工具。 |
| `mcp/src/delivery-mcp/src/pomelo_delivery_mcp/orbit_client.py` | 调用现有 Deployment 列表 HTTP 接口。 |
| `mcp/src/delivery-mcp/src/pomelo_delivery_mcp/tools/orbit.py` | Deployment 列表工具与 Gateway 初始化。 |
| `internal/application/gateway/usecase/` | Gateway 默认模板的编译、显式覆盖合并与 draft Version 创建。 |
| `mcp/src/delivery-mcp/tests/` | 网络可用性、Gateway 初始化配置和 Deployment 查询测试。 |
| `scripts/prepare_ragflow_tei.py` | 有界的 registry 检查和已验证清单缓存读取。 |
| `scripts/ragflow-split/`、`scripts/ragflow-integrated/` | 各拓扑的机器可读部署契约、由契约生成的 Compose。 |
| `scripts/render_ragflow_deployment_contract.py` | 共享的契约渲染与漂移检查入口。 |
| `skills/deploy-ragflow-split-orbit/`、`skills/deploy-ragflow-integrated-orbit/` | 使用新 MCP 读取/写入边界与契约化前置检查的编排。 |

## Design decisions

### Network availability

`runtime_doctor(network_name="traefik")` 只断言网络是否可供 Gateway 与应用连接。结果包含 `exists`、driver、scope 和 `usable`：网络存在且满足 Gateway Compose 的连接条件时为 `usable=true`，不检查 Docker labels、连接容器或 Orbit Gateway 归属；网络缺失时为 `exists=false`；属性不兼容时为 `usable=false` 并返回阻塞原因。

`orbit_provision_gateway` 在网络可用时直接复用该网络，并初始化或复用 Gateway Application、Version 和 Service；网络缺失时初始化并部署 Gateway，使其创建 `traefik` 网络。网络属性不兼容时供应流程返回 `outcome="blocked"`，不删除、重命名、迁移或协调现有网络。不存在 `orbit_reconcile_gateway_network` 工具。

### Gateway initialization

`orbit_provision_gateway` 先断言 `traefik` 网络。网络可用时直接复用；网络缺失时按默认模板创建 Gateway Application、Version、Service 并部署，以创建网络；网络属性不兼容时返回阻塞结论，不创建或修改任何网络。

初始化结果返回 Gateway、Version、Service 和 Deployment 的资源 ID，以及网络存在性和可用结论。它不定义 `operation_id`、阶段状态、续接协议或新的持久化操作模型。重复调用按当前 Application、Version、Service 状态复用既有资源，不主动创建重复 Gateway。

### Gateway defaults and overrides

Gateway 是 `kind=gateway` 的普通 Application，默认模板仅用于初始化，不是运行时配置的唯一来源。默认模板创建 `traefik` Component、`web:80`、`websecure:443` 和 `api:8080` endpoint；`api` 默认值为 `protocol=http`、`mode=local`、`bind_address=127.0.0.1`、`listen_port=8080`，以支持 Dashboard。

`orbit_provision_gateway` 新增可选 `gateway_spec`。其结构使用现有 Gateway、Version、Component 和 endpoint 的领域字段，而不是接收未校验的 Traefik YAML：

```json
{
  "gateway": {
    "name": "...",
    "rest_api_url": "...",
    "base_domain": "...",
    "default_entrypoint": "...",
    "tls_mode": "..."
  },
  "component": {
    "image": "...",
    "pull_policy": "..."
  },
  "endpoints": [
    {
      "name": "api",
      "protocol": "http",
      "container_port": 8080,
      "mode": "local",
      "bind_address": "127.0.0.1",
      "listen_port": 8080
    }
  ]
}
```

省略字段使用默认模板；提供字段按 Application 配置、Component 基本字段和 endpoint 名称逐项覆盖。MCP 保留通用 `orbit_update_version_component_*` 工具，使用户或 skill 能在 draft Version 中使用相同的受校验模型覆盖初始化结果。编译 Gateway 静态配置时，默认值只补齐缺失字段，不能重置显式 endpoint 或 Component 覆盖。

对于已发布且需要不同配置的 Gateway，供应流程创建一个新的 draft Version，以当前受管 Component 为基础合并 `gateway_spec`，再由既有 preview/publish/deploy 路径发布。它不修改已发布 Version。skill 将 Gateway 初始化规格写入其 primitive，并在创建或修复 Gateway 时传给 `orbit_provision_gateway`；用户可改用低层 MCP 工具完成同等覆盖。

### Deployment discovery

Orbit 后端已经提供按 Project/Application 过滤的 Deployment 列表接口。`OrbitClient` 新增对应读方法，MCP 新增 `orbit_list_deployments(project_id, application_id, service_id=None, status=None, page=1, per_page=50)`。

工具必须验证 `service_id` 属于所选 Application，再在 Application 范围内过滤并返回分页摘要。每项仅包含 `id`、`project_id`、`application_id`、`service_id`、`version_id`、`operation_type`、`status`、`created_at`、`started_at` 和 `finished_at`。skill 用它盘点 `ragflow-integrated/default` 与 `ragflow-split/default` 的活动部署，并在选择拓扑后通过 Orbit 停止另一 RAGFlow Service。

### Deployment contract authority

新增 `scripts/ragflow-split/deployment-contract.json` 与 `scripts/ragflow-integrated/deployment-contract.json`，分别作为 split 与 integrated 拓扑的机器可读权威定义。契约包含 Application/Version/Component 名称、镜像、命令、挂载、网络别名、固定环境变量、运行时变量键名、health check、端点、CPU/GPU 差异、模型目录和手工 Provider 初始化项。共享渲染器保证两种契约使用相同的校验和输出规则，但不合并其数据目录、依赖关系或别名。

新增标准库 Python 渲染器，以契约生成全部 `scripts/ragflow-split/docker-compose.*.yml`、`scripts/ragflow-bundled/docker-compose*.yml` 以及两份 deployment primitive。渲染器提供显式 `--write` 和只读 `--check`；后者比较受控生成结果与提交文件，发现漂移即非零退出。两份 skill 不再自行声明组件环境或地址，只引用各自契约生成的 primitive。

`MYSQL_ROOT_HOST` 作为两种 MySQL 容器环境显式写入契约和生成的 Compose，值为 `%`。`TEI_BASE_URL` 不属于 RAGFlow 容器环境：它是手工 HuggingFace Provider 初始化的地址，因此在契约的 `manual_provider` 部分定义为 `http://tei:80`，并从 Connectivity primitive 的容器环境表中移除。运行时变量仅声明键名和应用范围，不将值写入契约。

### TEI preflight bounds

`prepare_ragflow_tei.py check` 新增 `--registry-timeout-seconds`（默认 30，范围 1..60）和 `--manifest-cache-ttl-seconds`（默认 300，范围 0..3600）。registry token 与 manifest 请求共享同一单调时钟 deadline；任何网络请求不超过剩余时间。

`prepare-image` 在清单验证成功后写入包含 image digest、验证时间和结果的缓存。`check` 只读取该缓存：在 registry 超时或不可达时，只有未过期且 digest 完全匹配的成功缓存可将 `image_manifest` 记为通过；其他情况失败。每个 JSON 结果项补充 `elapsed_seconds` 与 `source`（`registry` 或 `cache`）。

### Skill workflow

更新 split skill 的顺序如下：

1. 读取当前部署契约生成的 primitive，运行 `--check` 验证生成物无漂移。
2. 检测 NVIDIA，执行所选 profile 的两个模型目录 TEI 预检；检测到 NVIDIA 而 GPU 预检失败即停止。
3. 读取 Project、Application、Version、Service 与 `orbit_list_deployments`，再调用网络可用性检查。
4. 网络可用时直接复用；网络缺失时使用 `orbit_provision_gateway` 初始化，网络不兼容时停止并报告阻塞，不以 MCP 外部 Docker 写操作补救。
5. 每次 MCP 生命周期写入前重新读取当前 MCP schema。Gateway 使用 `orbit_provision_gateway` 初始化或复用，后端依赖和 selected RAGFlow Service 继续使用既有 preview/deploy/health 验证流程。

skill 明确禁止把 `docs/archive/` 或导出的 SQL/SQLite 数据库基线作为配置、资源 ID、运行状态或部署步骤的来源。当前 Compose 和部署契约是唯一操作依据。

## Technical questions

- `gateway_spec` 应否覆盖 TCP entrypoint、挂载和静态 provider 选项，还是先限于已有 GatewayConfig、Component 基本字段与 endpoint；需要以普通 Application 的校验边界决定。
- 后端 Deployment 列表 API 的最大 `per_page`、排序和 status 枚举需要在 MCP client 测试中固定，避免 client 过滤遗漏最新活动 Deployment。
- `traefik` 网络的最小可用断言应只要求 driver/scope，还是还应检查 Docker attachability；需要以 Gateway 的实际 Compose 网络声明确定。
- Compose 生成器的文件格式和稳定排序需要先确定，避免无意义的整文件差异。

## Risks

- 已存在的 `traefik` 网络可能不满足 Gateway 实际连接条件；前置检查必须阻止部署，但不能尝试修改该共享网络。
- Gateway 默认模板和显式覆盖的合并顺序若不固定，会在修复流程中改变用户已发布的意图；必须在 Gateway usecase 和 MCP 契约测试中锁定。
- 契约生成会使两种 Compose 文件成为派生物，迁移时需一次性验证所有 CPU/GPU 与后端服务文件，避免遗漏局部手工更改。
- registry 缓存只能缩短瞬时网络故障，不能替代对镜像 digest 的正常验证；缓存过期或 digest 不同必须阻止部署。

## Alternatives

- 判断或协调 `traefik` 网络的 Orbit 归属：不采用。共享网络的可用性，而非创建者，是 Gateway 初始化的前置条件。
- 继续把 Gateway endpoint 和 Component 配置写死在编译逻辑：不采用。默认值只负责初始化，不能绕过普通 Version 的覆盖路径。
- 在 `orbit_wait_deployment` 中增加组件 health 等待：不采用。当前 health check 已能正确表达慢磁盘造成的表初始化过程。
- 继续让任一拓扑的 skill、Compose 和 Markdown primitive 分别维护配置：不采用。无法可靠检测连接地址和环境声明的漂移。

## User review notes

- 2026-08-03：用户明确要求在标准流程中进入 Spec；本规格不包含任何当前运行资源的操作。
- 2026-08-03：用户确认现有健康检查满足数据库表初始化慢的场景，本规格保持该边界。
- 2026-08-03：用户要求 Gateway 作为有默认模板的普通 Application，skill、MCP 与用户均可覆盖初始化配置；本规格增加 `gateway_spec` 和 draft Version 合并设计。
- 2026-08-03：用户要求将网络设计收敛为可用性断言：复用现有可用 `traefik` 网络，缺失时由 Gateway 初始化创建，不实现网络归属或协调功能。
- 2026-08-03：用户确认 Gateway 初始化不需要阶段化续接。规格仅保留网络复用/缺失创建和默认模板配置覆盖。
- 2026-08-03：用户要求开始实现配置契约和一致性检查；Spec 视为已接受。
- 2026-08-03：用户要求将配置契约实现扩展到 integrated。使用共享渲染器和 topology-specific 契约，保持 split 与 integrated 的配置和数据边界独立。
