# Gateway Service Lifecycle
最后修改时间: 2026-08-15 12:50:00

Review status: Accepted

## Basis

本计划实施已接受的 `docs/requirement/20260815-gateway-service-lifecycle.md` 与 `docs/spec/20260815-gateway-service-lifecycle.md`。Gateway 创建生成初始 Application、GatewayConfig、可编辑 Version/Component 和默认停止态 Service；后续配置与部署复用普通 Application/Version/Service/Deployment 模型。

## Implementation steps

1. 整理 Gateway 初始 Version 模板与既有配置接线。
   - 在 `internal/application/gateway` 将现有 Traefik Component 生成逻辑改为仅供 `CreateGateway` 建立初始 `unpublished` Version 使用。
   - 保持当前 `missing`、`web`、`none`、`traefik` 默认，不新增配置字段；接入已有 `traefik.image`、`rest_api_url`、`base_domain`、`cert_dir`、`rest_ready_timeout` 和 `cert.letsencrypt.*`。
   - 初始 Version 写入后不再由 Gateway usecase、worker 或 Route usecase 覆盖其 image、pull policy、endpoint、mount 或静态文件。

2. 实现跨 Application/Gateway/Service 的原子 Gateway factory。
   - 为 Application、GatewayConfig、Version、VersionComponent、默认 Service 和 Service Component mapping 建立一个共享事务边界。
   - 默认 Service 使用 `instance_key=default`、code `<application-code>-default`、`stopped` 状态，绑定初始 Version。
   - 扩展 Gateway 查询/响应，稳定返回默认 Service 及已有实例，不允许 Web、HTTP 或 MCP 通过“缺失则创建”推断运行绑定。

3. 移除会重写 Version 的 Gateway compile 与 Route TCP reconcile。
   - 删除或停用 `CompileGatewayToVersion` / `CompileTCPRouteListeners` 的生产调用，以及 Route 写操作对 Gateway Version 的隐式变更。
   - 保留 HTTP/TCP 动态 Route snapshot；静态 TCP entrypoint 与端口必须由用户在目标 Gateway Version 中显式配置并部署。
   - 一并移除不再需要的 port、repository 和测试依赖，避免保留新旧两条写入链路。

4. 将 Gateway 创建和 Provision 收敛为资源准备。
   - `CreateGateway` 只创建完整资源组，不发布、不部署。
   - `orbit_provision_gateway` 幂等地创建/解析指定实例的 Service，缺失实例通过通用 Service create/mapping 能力建立；它不得发布 Version、调用 DeployService 或等待 Deployment。
   - 相应简化 Gateway Provision DTO、MCP output、端口和测试，明确后续部署必须是显式通用 Service 命令。

5. 收敛通用 Deployment 行为。
   - 移除标准 Service 对 `EnsureGatewayRunning` 的前置依赖和 Gateway 自身 `HasActiveGatewayService` 预检及 SQL 查询。
   - 让 `ensureSingleRuntime` 覆盖 `kind=gateway`，复用同 Application 单运行实例规则；保留同 Service 活跃 Deployment 检查。
   - 让有效部署计划严格来自 Version + Service overlays。Gateway 只在 Compose 成功后等待 REST 就绪并发布完整动态 Route snapshot；就绪等待使用既有 `traefik.rest_ready_timeout`。

6. 对齐 HTTP、MCP、Proto 与 Web 交互。
   - Gateway 详情部署直接选取已有 Service ID 并调用通用部署接口，不在部署弹窗中创建或更新 Service binding。
   - 保留从 Gateway 进入 Application/Version 的工作流；Gateway Application 不得因 `kind=gateway` 被限制普通 Version/Component 编辑。
   - 更新 MCP 与 HTTP 响应以表达“资源已准备、部署需显式发起”的结果，移除过期的 Version published/Deployment wait 表述。

7. 补齐测试、活文档与回归检查。
   - 追加 factory 原子性、默认 Service、入口幂等性、Gateway Version 编辑后不被部署重写、Route 不编译 Version、通用单运行实例和 Compose 失败传播测试。
   - 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md` 及必要 how-to，使其不再声明 Gateway compile/单 active Gateway 专用策略或受管 Version 私有配置。
   - 执行 Go 格式化、vet、测试；涉及 Web 改动时执行 `yarn --cwd web lint:fix` 和 `yarn --cwd web typecheck`。

## Files to change

- `internal/application/gateway/usecase/{service,compile,provision}.go` 及 DTO、port、测试。
- `internal/application/service/usecase/service.go` 与 Gateway factory 所需的 Repository/SQLC 事务接口。
- `internal/application/deployment/usecase/{command,deployment_execution,effective_service_plan,compose_renderer,runtime_network}.go`、port、store、测试。
- `internal/application/route/{port,usecase}`、外部 Traefik adapter 及测试。
- `internal/api/{http,mcp}`、`proto/orbit/v1/gateway/gateway.proto`、生成代码和 `web/src/views/gateway/*` 等调用方。
- `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、相关 guides 与本任务过程文档。

## Verification plan

1. Gateway factory 成功后断言资源组和默认停止态 Service/mapping 完整；任一步故障时断言无残留数据。
2. 从 Application/Version 修改 Gateway Component 后，部署生成的 Compose 使用该 Version 值，且 Gateway/Route 操作不改写该 Version。
3. default 与 staging Service 并发部署同一 Gateway Application 时，由通用单运行实例规则阻止；停止后可切换实例。
4. 标准 Application 在 Gateway 未运行时可入队；Gateway/标准服务的端口、网络或容器冲突在 Compose 日志中失败，而非 command 预检。
5. Gateway Compose 成功后等待 REST 并发布完整 HTTP/TCP Route snapshot；失败将 Service/Deployment 标记为 `faulted`。
6. 运行 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`；若修改 Web，再运行项目约束的 lint/typecheck。

## Blockers

无。

## Assumptions

- 当前 `missing`、`web`、`none`、`traefik` 是保留的创建默认，不新增对应进程配置；用户可通过 Version 或 GatewayConfig 修改可编辑行为。
- Gateway Version 的可编辑性遵从现有标准 Application/Version 权限和状态规则，不增设 Gateway 特殊校验。

## Risks

1. Route 不再自动创建静态 TCP listener 后，用户必须先发布含所需 entrypoint/端口的 Gateway Version；否则失败将延后到部署或 Route 发布阶段。
2. 原子 factory 需要跨现有 SQLC repository 共享 transaction context，不能沿用当前补偿删除。
3. Compose 中共享网络的名称保持固定，变更该现状不属于本任务。

## Rollback

该任务不改已执行迁移。上线后若需要撤回，回滚应用代码与文档到前一发布版本；不新增数据兼容层或保留双生命周期。

## User review notes

- 用户确认不存在未决事项，并要求开始推进任务。
- 用户确认既有配置仅用于初始默认，不新增拉取策略、默认入口/TLS 或共享网络配置；Version/GatewayConfig 的后续编辑保持可用。
- 用户明确要求开始实现；本 Plan 视为 Accepted。
