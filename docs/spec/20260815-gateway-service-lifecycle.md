# Gateway Service Lifecycle
最后修改时间: 2026-08-15 12:04:38

Review status: Accepted

## Requirement basis

本规格依据 `docs/requirement/20260815-gateway-service-lifecycle.md`。它不重写 `20260731-gateway-runtime-policy-boundary`、`20260809-gateway-deletion` 或 `20260813-custom-tcp-routes` 的已接受过程记录；实现时以当前代码和活 SoT 为准，并把本任务的最终结论回写活文档。

## Target model

Gateway 不是另一套运行实体，而是一个带平台策略的资源组：

```text
Gateway
  = Application(kind=gateway)       // 长期身份
  + GatewayConfig                   // REST、域名、TLS 等平台策略
  + Version/Component               // 默认 Traefik 规格，后续按常规 Version 编辑
  + Service(instance_key=default)   // 创建时附带的初始运行/部署目标
  + Service(instance_key=...)       // 可选的其他候选实例
  + generic Deployment              // 异步操作记录
```

`Service` 拥有 Version binding、组件映射、运行态、工作目录、Deployment 和任务队列契约。Gateway 领域只拥有资源组创建和部署完成后的 Traefik 控制面同步；选定 Version 的容器规格仍归通用 Application/Version/Service 模型所有。

## Design

### 1. 原子创建资源组

`CreateGateway` 改为一个事务性的 factory，而不是连续调用多个 Repository 后尝试补偿删除。它按以下顺序建立数据，并在任一步失败时整体回滚：

1. 校验项目成员、Application name/code、GatewayConfig 和受管镜像/入口策略。
2. 创建 `Application(kind=gateway)` 与 `GatewayConfig`。
3. 从 `traefik.*`、`cert.letsencrypt.*` 和创建请求生成一个初始、`unpublished` 的 Traefik Version/Component 声明。
4. 创建 `instance_key=default`、`status=stopped` 的普通 Service，code 采用 `<application-code>-default` 规则。
5. 建立 Service Component mappings，使默认 Service 可直接进入通用 Deploy 命令。

事务内应复用 Service 的 code、实例唯一性、组件 mapping 和 Version 所属校验；不得在 Gateway usecase 中另造一份 Service 数据模型。实现可新增窄的 `GatewayBundleStore` 或在已有 context-aware SQLC Repository 上使用共享 transaction context，但不能以忽略错误的补偿删除替代事务。

创建响应和 Gateway 查询必须能让调用方稳定识别默认 Service。建议在 `GatewayView` / `GatewayResp` 中返回该 Service 的 ID、instance key、code 和状态，并可列出已有 Gateway Service instances；至少 HTTP、MCP 与 Web 不得通过“没有则创建”的方式推断默认实例。创建完成不发布、不部署，也不把该 Version 锁成 Gateway 私有配置；用户可转到对应 Application/Version 页面继续配置。

### 2. 统一部署命令

Gateway 详情页和 MCP 使用下列统一流程：

```text
resolve target Gateway Service (default when no instance is specified)
  -> generic DeployService(service_id)
     / RestartApplication(application_id, service_id)
     / StopApplication(application_id, service_id)
  -> generic Deployment + task
  -> worker Render + docker compose
  -> Gateway post-deploy route publication
```

Gateway 页面不得在 Deploy 点击处理器内创建 Service。部署对话框列出既有 Service 和其 Application 的 Version，并默认选中该 Service 当前绑定版本；仅在用户确认部署且版本不同的情况下，按普通 Service 更新规则重绑 Version 和组件覆盖，再调用通用 DeployService。Gateway Config 保存不改变 Version。页面可以选择已有 Gateway Service instance，但不能把 instance 选择实现为即时创建或更新 binding。

`orbit_provision_gateway` 保留为幂等资源准备入口：若资源组不存在则调用 factory，若已存在则按传入的 `instance_key` 解析 Service，未传入时使用 `default`。它为显式 Provision 的缺失实例创建 Service 时必须复用通用 Service 创建/组件 mapping 语义，并返回停止态 Service；不得发布 Version、调用部署命令或等待 Deployment。部署由用户或 MCP 随后的显式通用 `DeployService` 调用完成。

### 3. 薄 Gateway deployment policy

通用 Deployment command/execution 不新增 Gateway Deployment 类型，也不增加 Gateway 专用运行前预检。Gateway policy 仅在创建默认 Version 与 Compose 成功后介入：

| 时点 | Gateway policy | 通用 Service 责任 |
|---|---|---|
| command 前 | 不覆盖选定 Version；仅冻结部署后 Gateway REST 同步所需的 GatewayConfig | 权限、Service/Version 所属、同 Service 活跃 Deployment、通用同 Application 单运行实例、创建 Deployment/Task |
| plan/render | 无 Gateway 规格注入；只使用选定 Version 和 Service overlays 的通用有效计划 | 合成 EffectiveServicePlan、计算/校验快照、写 workspace、执行 Compose |
| compose 成功后 | 等待 Traefik REST API，发布完整 HTTP/TCP Route snapshot | 更新 Service/Deployment 状态、日志、取消与故障处理 |

移除 `EnsureGatewayRunning` 与 `HasActiveGatewayService` 的 command 预检。`ensureSingleRuntime` 作为通用规则同样适用于 `kind=gateway`，不再因 Application kind 跳过，因此同一 Gateway Application 的 `default`、`staging` 等实例不能并发运行。不同 Application 间固定端口、共享网络和同名容器等冲突不提前分类为 Gateway 业务错误，而由 Compose 的正常执行与日志反馈。标准 Service 同样不因 Gateway 状态被 API 预先阻止；外部 `traefik` 网络不存在等问题由常规 Compose 执行路径报告。

部署后 Route 发布失败必须继续沿用当前通用故障处理：Gateway Service 被标记为 `faulted`，Deployment 从 `running` 进入 `faulted`，并保留日志原因。动态 Route snapshot 在 Gateway 就绪后读取当时的完整启用 Route 集合，以恢复最新控制面；它不是 Gateway 静态 Render 输入的回写通道。

### 4. 初始默认与可编辑 Version

Gateway 创建需要提供一个可工作的初始 Traefik Component，但这只是创建默认值，不是长期的 Gateway Render 模板。Docker socket 挂载、`traefik.yml`、ACME 状态挂载、entrypoint、共享网络和 Dashboard 相关项与镜像、拉取策略一样，都写入初始 Version Component，受普通 Version 状态和编辑规则管理。

Application/Version 页面必须对 Gateway 保留常规 Version/Component 编辑能力。用户可在未发布 Version 中编辑镜像、拉取策略、端点、挂载、环境、资源和静态文件，并按普通流程发布 Version、更新 Service binding 后部署。Gateway deploy、restart、Route 保存/启停/删除和 worker 都不得调用 `CompileGatewayToVersion` 或以任何方式重写这些字段。

GatewayConfig 只保存 Gateway 领域字段：REST 地址、基础域名、默认 HTTP entrypoint 与 TLS 模式。它用于 Gateway 控制面同步及已有的通用 Route/label 推导，不是 Version 静态配置的镜像。修改 GatewayConfig 不创建、修改或选择 Version；修改 Traefik 进程静态配置必须显式编辑目标 Version。

初始 Version 的默认值按以下来源生成，避免以代码常量或 YAML 字面量取代已有配置：

| 参数类别 | 来源与处理 |
|---|---|
| Gateway 的 REST 地址、基础域名、默认入口、TLS 模式 | `GatewayConfig`；创建时采用已存在的 `traefik.rest_api_url`、`traefik.base_domain`，并保持当前 `web`/`none` 默认；保存后以 GatewayConfig 为准 |
| 镜像、镜像拉取策略 | `traefik.image`；拉取策略保持当前 `missing` 默认。二者在 Version Component 中均可由用户修改 |
| REST 就绪等待、ACME 持久化、Let's Encrypt | `traefik.rest_ready_timeout` 直接供部署后等待使用；证书/ACME 目录固定从 `workspace.deployment/traefik/data/certs` 派生，`cert.letsencrypt.enabled/email/challenge/dns_provider` 写入初始 Version 的静态配置。不得继续写死 `admin@localhost` 或固定 HTTP challenge |
| 共享网络 | 保持当前固定 `traefik` 网络。它同时被初始 Gateway Version 与需要 gateway endpoint 的通用 Compose Render 使用，因此不提供单个 Version 的私有覆盖 |
| Docker socket、provider endpoint、web/websecure/REST 的默认端口与监听地址、静态配置/ACME 容器路径、Dashboard/API 开关、日志等级 | 不新增进程配置。它们作为初始 Version 的 Traefik 模板内容保存，用户可在 Application/Version 中按普通能力修改；部署不得重新注入这些值 |
| Gateway code/name/component 等受管资源识别名称 | 代码识别常量；它们不是环境运行参数，也不进入 GatewayConfig |

初始 Version 中持久 ACME/证书目录固定从 `workspace.deployment/traefik/data/certs` 派生，具体的物理路径解析沿用现有 workspace/physical-data-root 机制。初始 Version 保存后，后续部署快照由通用 EffectiveServicePlan 覆盖其 Component/mount/endpoint 值；worker 不得重新读取进程配置来替换用户保存的 Version。

Route 的动态快照继续在 Gateway 就绪后读取最新启用 Route 集合。它不再 reconcile 或编译 Gateway TCP listener；需要新增静态 TCP entrypoint/宿主机端口时，用户先在目标 Gateway Version 中显式配置并部署，再启用相应 Route。缺失 entrypoint 的问题走 Traefik/Compose 或 Route 发布的普通错误路径，不引入 Gateway Version 自动修复。

### 5. Version 状态与 Service 实例

本需求不为 Gateway 创建额外的“发布后才可部署”例外。初始受管 Version 的状态应遵从通用 `DeployService` 当前语义；若后续产品决定只能部署 `published` Version，必须作为适用于所有 Service 的独立规则实施。`orbit_provision_gateway` 不得再静默发布 Gateway Version，以免 HTTP 和 MCP 行为分叉。

Gateway 使用普通 Service 的实例模型。每个 Environment 只允许一个 `instance_key=default` 的初始 Service，code 为 `<application-code>-default`。Application code 是产品身份 `traefik`，不把 Project code 拼进 Application code。

Service create/update 继续复用全局 code 唯一性和 `(application_id, instance_key)` 唯一性，Gateway 不建立平行的实例标识、Service code 或实例并发校验。通用同 Application 单运行实例规则负责阻止同 Application 的多个 Gateway Service 同时启动。

Compose project 与 Service 工作目录都使用 Service code；受管容器名和网络 alias 仍由 `<application-code>-<component-name>` 派生，并不含 instance key。停止操作执行 `docker compose down`，通用单运行实例规则防止同 Application 的实例同时创建同名 Traefik 容器。实例语义因此是顺序切换，不是并发副本或灰度运行。

## Affected components

- `internal/config` 与 `configs/config*.yaml`：接入既有 Traefik/证书配置，作为 Gateway 初始 Version 的默认值来源；不新增拉取策略、默认入口/TLS 或网络配置。
- `internal/application/gateway`: 资源组 factory、默认 Service 查询、GatewayConfig 更新语义与部署后 hook；删除/停用会重写 Version 的 Gateway compile 路径。
- `internal/application/service`: 默认/指定 Gateway Service 的创建、组件 mapping 复用和实例选择。
- `internal/application/deployment`: GatewayConfig 的部署后同步快照、通用 worker hook 调用点；删除 Gateway 专用前置检查，并让通用单运行实例规则覆盖 Gateway。有效部署计划继续完全来自 Version/Service。
- `internal/application/route` 与 `internal/infrastructure/external/traefik`: 保持 readiness 与完整动态 snapshot，删除 Route 修改时编译 Gateway Version 的路径。
- Repository/SQLC：为跨 Application/Gateway/Service 的原子创建提供事务边界，移除仅供 Gateway 专用 active 预检的查询/端口；不修改已执行迁移文件。
- HTTP/Proto/MCP/Web：Gateway 返回默认 Service、部署 UI 删除临时 Service 创建/改绑、文案更新、Provision 统一。
- 测试与活文档：覆盖 factory 原子性、入口一致性、Gateway Version 的常规可编辑性、部署不覆盖 Version、通用单运行实例、无 Gateway 预检时的 Compose 失败传播和 Route 发布失败分支。

## Evaluation

该规格与用户目标和现行领域模型一致：Gateway 的运行态、操作状态和部署执行全部归 Service/Deployment；`kind=gateway` 仅保留确有必要的平台策略。它还能消除 HTTP 与 MCP 的不同资源时序，以及 Gateway 配置写入和受管静态配置脱节的问题。

用户已确认 Gateway 采用普通 Service 实例模型。后续 Project 初始化规格将每个 Environment 收敛为唯一 `default` Service：Application code 固定 `traefik`，Service code 为 `traefik-default`。实例并发直接复用通用同 Application 单运行实例规则，不再使用 active Gateway 查询或按 `service_id` 的 Gateway 互斥逻辑。

新的审视表明，前一版“Gateway Render 注入受管静态物料”的设计会绕过 Application/Version 配置面，与领域模型冲突。最终方案是：已有配置仅生成初始 Version；拉取策略、默认入口/TLS 与共享网络保持当前默认，初始 Version 及其后续版本完全由常规 Application/Version/Service 流程配置和部署，Gateway 只保留创建 factory 与部署后 Route 同步。Route 不再隐式编译 Gateway Version。用户已确认不存在未决事项，本 Spec 已接受。

## Risks

1. Route 不再自动创建静态 TCP listener 后，用户必须先发布含所需 entrypoint/端口的 Gateway Version；否则失败将延后到部署或 Route 发布阶段。
2. 从“受管 Render 覆盖”迁移为“Version 规格 SoT”会影响 Gateway compile、Route TCP reconcile、页面文案和测试 fixture，必须删除而非保留两套写入路径。
3. 多个 Gateway Service 的 Service code、工作目录和 Compose project 可以独立，但受管容器名不含 instance key；通用单运行实例规则必须覆盖 Gateway，测试需覆盖同 Application 的实例互斥和停止后切换。
4. 去掉 Gateway 状态预检后，缺失外部网络或端口冲突会在 Compose 执行期失败；日志和 HTTP/MCP 错误需要能直接指出 Docker 原因。
5. Route snapshot 使用最新动态路由而 Version 静态规格使用已选 Version 的通用部署快照是有意的双时点模型，日志必须能说明两者来源，避免运维误判。
6. 证书/ACME 目录必须从 `workspace.deployment` 单一派生，并作为 Docker daemon 可见 host-path mount 写入初始 Version；不得演变为覆盖用户 Version 的第二套 SoT。

## Alternatives

1. 保留当前“Gateway 更新或 Route 变更时编译 Version”模型：会覆盖用户在 Application/Version 中保存的组件规格，导致 GatewayConfig 与 Version 成为同一静态配置的双重来源，因此不采纳。
2. 保留 Gateway 专用 active / readiness 前检：可以更早返回领域错误，但重复了 Service 运行态与 Compose 资源冲突语义，且会让 Gateway 部署与常规应用继续分叉。

## User review notes

- 用户要求 Gateway 创建时附带 Application、Version 和 Service，并确认 Gateway 部署应以 Service 部署为本体、仅在前后保留自身特定业务。
- 用户确认使用新的 `20260815-` Requirement，既有 Gateway 任务文档不修改。
- 用户补充：当前 Gateway 使用 `default` instance，Service code 为 `<application-code>-default`，但实例模型必须保留；实例并发改由通用同 Application 单运行实例规则处理。
- 用户确认切换到标准模式，且不保留未决事项；受管静态物料 Render snapshot 与通用 Version 状态边界按本 Spec 采纳。
- 用户要求 Gateway 尽量不使用特殊校验和逻辑：部署前和实例并发优先使用常规 Service 校验，端口/网络冲突走普通 Compose 错误路径。
- 用户要求：除受管资源的通用名称外，Gateway 运行参数优先复用已有配置；找出未覆盖项后补入配置，禁止在代码或受管静态 YAML 中写死。
- 用户纠正：Gateway 创建后仍可通过 Application、Version 和 Service 的常规入口继续配置；不能把 Gateway 做成一次性受管 Render 或在部署中覆盖 Version 规格，需据此重新审视本 Spec。
- 用户最终确认：不新增拉取策略、默认入口/TLS、共享网络配置，保持当前默认；已有镜像等配置仅作为创建默认，用户可在 Application/Version 覆盖。
