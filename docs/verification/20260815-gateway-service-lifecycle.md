# Gateway Service Lifecycle
最后修改时间: 2026-08-15 16:08:42

Review status: Accepted

## Verification basis

- Requirement: `docs/requirement/20260815-gateway-service-lifecycle.md`
- Spec: `docs/spec/20260815-gateway-service-lifecycle.md`
- Plan: `docs/plan/20260815-gateway-service-lifecycle.md`
- Verification scope: 已暂存的 `20260815-gateway-service-lifecycle` 变更；应用导入导出删除、Gateway REST 网络选项、流水线变量管理和本地目录仓库排障仍处于未暂存工作区，不纳入本任务结论。

## Requirement and spec alignment

- Gateway factory 在一个事务内创建 Application、GatewayConfig、初始 unpublished Version/Traefik Component、`default` Service 和组件 mapping。默认 Service 为 `stopped`，code 采用 `<application-code>-default`。
- Gateway 部署页面只选择已有 Service 并调用通用部署接口；Provision 只准备/解析 Service，不发布 Version 或创建 Deployment。
- 删除 Gateway compile 与 Route TCP reconcile 写入链路；保存或发布 Route 不再隐式修改 Gateway Version。
- 通用部署命令不再依赖 Gateway 运行态或全局 active Gateway 预检；同 Application 的 Gateway Service 由通用单运行实例约束控制。
- Gateway 仍在通用 Compose 成功后通过既有 Route publisher 等待 REST 就绪并发布全量快照；Route 发布失败沿通用 Deployment faulted 路径处理。
- HTTP、MCP、Proto 和 Web Gateway 页面均返回/使用稳定的默认 Service 标识。

## Actual diff summary

- 新增 Gateway 初始资源组 factory 与 SQLite 集成测试，移除旧的 compile、mount、exposure 生产写入路径。
- Gateway 创建、查询、Provision、MCP 输出和详情页收敛到普通 Service 实例模型。
- 删除 `HasActiveGatewayService` SQL/仓储/部署前检，并使 `ensureSingleRuntime` 覆盖 Gateway。
- Route snapshot 保留 REST readiness 和动态发布，去除静态 Gateway Version 反向编译。
- 更新 CD 领域模型、运行时和操作文档以反映 Version/Service 为规格与运行态 SoT。

## Acceptance checklist

- [x] Gateway factory 成功创建默认停止态 Service、初始 Version/Component 与 Service component mapping。
  `TestCreateGatewayCreatesAtomicDefaultServiceBundle`。
- [x] GatewayConfig 写入失败不会留下 Gateway Application。
  `TestCreateGatewayRollsBackWhenGatewayConfigWriteFails`。
- [x] Provision 对 `default` 幂等，对 `staging` 创建停止态普通 Service，且不发布 Version。
  `TestProvisionGatewayOnlyPreparesStoppedServices`。
- [x] Gateway `default` 运行时会阻止同 Application 的 `staging` 启动。
  `TestEnsureSingleRuntimeAppliesToGatewayServices`。
- [x] Route snapshot 不再编译 Gateway listener，仍等待 REST 并发布动态快照。
  `TestPublishSnapshotDoesNotCompileGatewayListeners`、`TestPublishRouteSnapshotDoesNotCompileGatewayListeners`。
- [x] Gateway 页面不再在部署弹窗创建或改绑 Service；Proto/HTTP/MCP 均暴露默认 Service。
  已检查暂存实现与 Web 类型检查。
- [x] 初始 Traefik Component 使用既有配置来源生成可编辑 Version 默认值。
  `TestBuildInitialGatewayComponentExposesDashboardApiAndConfiguredCertDirectory`。
- [x] 删除 Gateway 专用部署前检后，通用部署执行仍保留 Compose 后 Route 发布失败时的 Service/Deployment faulted 处理。
  已检查 `deployment_execution.go` 的通用执行路径。

## Test results

| Command | Result |
| --- | --- |
| `go fmt ./cmd/... ./internal/...` | Passed |
| `go vet ./cmd/... ./internal/...` | Passed |
| `go test ./cmd/... ./internal/...` | Passed |
| `go test -count=1 ./internal/application/gateway/usecase ./internal/application/deployment/usecase ./internal/application/route/usecase` | Passed |
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `git diff --cached --check` | Passed |

## Scope and residual risk

- 未启动开发服务器，也未执行真实 Docker Compose、Traefik REST 或浏览器端到端场景；仓库约束要求由用户管理开发服务器。Compose 冲突和 REST 实际连通性仍需在具备运行环境时验证。
- 全量检查在包含其他未暂存改动的工作区执行；Gateway 生命周期的核心包另以 `-count=1` 重新执行。提交拆分后应保持本任务暂存区边界，避免混入未暂存任务。
- 回滚覆盖了 GatewayConfig 写入失败；其他资源写入失败继续由相同事务边界处理，但未为每一种 Repository 失败分别注入测试。

## Conclusion

自动化验证通过，暂存范围与已接受 Requirement、Spec 和 Plan 一致。本任务可进入提交审阅；真实 Compose/Traefik 环境验证保留为部署前的运行环境检查。
