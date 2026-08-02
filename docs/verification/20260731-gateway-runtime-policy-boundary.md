# Gateway Runtime Policy Boundary
最后修改时间: 2026-08-02 13:06:01

Review status: Draft

## Requirement Alignment

实现遵循 `docs/requirement/20260731-gateway-runtime-policy-boundary.md`：VersionComponent 保留镜像使用方式与 Endpoint 契约；ServiceComponent 只保存声明项的稀疏运行时覆盖；GatewayConfig 只承载平台策略。`VersionComponentPort`、`ServiceExpose`、`runtime_config_json` 已从生产模型、SQL 查询、Proto/HTTP/MCP 和前端契约中移除，没有兼容读取路径。

Gateway 的共享网络由 Gateway 生成的 Compose 顶层键 `traefik`、物理名称 `traefik` 和 `bridge` 驱动；标准 Service 以 external 网络消费者加入。Gateway `api` Endpoint 与其他已声明 Endpoint 一样保留 Version 默认值或 Service overlay 形成的 Effective Endpoint，不再被计划归一为 `internal`。

## Spec Alignment

`EffectiveServicePlan` 将 Version 声明、Service 稀疏 overlay 与 Gateway 平台策略合并，并由 Preview、渲染和部署共同使用。其 fingerprint 写入成功 Deployment，Service 读取基于最新成功部署的 fingerprint 计算 `pending_deploy`。

`service_component_{env,mount,resource,endpoint}` 实现 `override` / `deleted` 两态：缺行继承，覆盖只保存不同值，tombstone 显式压制声明默认值。写入与合成均拒绝未声明的 env、mount、resource 或 Endpoint。Service 详情按 Runtime、Connectivity、Mounts、Advanced 展示声明、overlay、effective 值和来源状态，并保留显式部署语义。

## Plan Alignment

计划中的数据库、SQLC、Proto、HTTP、MCP、前端和脚本改动均已落地。Endpoint/ServiceComponent 表与 deployment fingerprint 已折叠进 SQLite/MySQL 的原始 DDL；开发数据库从空库重建到 schema v30，不提供 `000031` 或旧端口、Expose、运行时 JSON 的兼容升级。

验证阶段额外补充了四项小修正：删除不再使用的 Gateway entrypoint 校验、将 Compose kind 分支改为显式 `switch`、把 SQLite schema 回归测试扩展为 `foreign_key_check` 与 `integrity_check`，以及移除 Gateway `api` 的名称特判。最后一项确保 Endpoint mode 由声明和 overlay 合成，不被计划或渲染改写。

## Actual Diff Summary

预期范围是 Gateway 平台策略、统一 Endpoint 表达、ServiceComponent 运行时覆盖、部署 fingerprint 和对应的 API/MCP/UI/基线导入迁移。实际改动覆盖该范围：

- `sql/migration/{sqlite,mysql}/{000023_application,000025_gateway,000026_service,000027_deployment}.*.sql`、`sql/query/**`、`internal/model/**`、SQLC 生成代码和仓储实现建立新表与读写边界。
- `internal/application/deployment/usecase/**`、Gateway/Service/Application usecase 与 HTTP/Proto 采用 EffectiveServicePlan 和 Endpoint 契约。
- `mcp/src/delivery-mcp/**` 与 RAGFlow baseline 改用新模型。
- `web/src/views/service/ServiceComponentDetail.vue`、Service 详情与 API/Proto 生成 TypeScript 提供组件化运行时配置界面。
- 过程文档位于 requirement、spec、plan 和本 verification 目录。

当前暂存批次包含领域实现与过程文档；后续修订一并纳入该批次：Gateway `api` 的 endpoint mode 不再由名称或 Application kind 限制，EffectiveServicePlan 与 Compose renderer 均保留其合成结果。未暂存的日志、配置、任务脚本和通用前端整理维持在独立批次。

## Acceptance Checklist

| 验收标准 | 结论 | 证据 |
| --- | --- | --- |
| Gateway/标准 Application 网络所有权明确且 `traefik` 稳定 | 通过（自动化） | `TestRenderGatewayComposeUsesStableTraefikNetworkKey`；Compose renderer 使用稳定键/名称 |
| 渲染器不再以单一布尔分支混合行为 | 通过（代码与测试） | Endpoint、平台网络和 Dashboard 由独立合成/注入步骤处理 |
| Gateway/应用重复默认来源移除 | 通过（DDL 覆盖） | 原始 DDL 删除 `gateway_config.image` 与 `application.image_pull_policy` 的重复来源 |
| 宿主机绑定只有 Effective Endpoint 真相 | 通过（单元测试） | 覆盖继承、值覆盖、tombstone、未声明 overlay 拒绝及 Gateway `api` mode 保留 |
| REST Provider 与 Gateway 受控静态配置保留 | 通过（代码/测试） | Gateway 编译、MCP 及全量 Go/MCP 测试通过 |
| 跨 SQLite/MySQL、SQLC、HTTP/Proto、MCP、前端一致 | 通过（生成及自动化） | `task sqlc`、`task proto`、Go/MCP/Web 检查通过 |
| Traefik 3.6 经 Orbit 部署并验证入口与公开路由 | 未执行 | 本任务约束禁止远程部署或 Docker lifecycle 操作 |
| Service 配置支持修改与显式删除 | 通过（代码/测试） | sparse overlay/tombstone 合成、Service detail API 与 UI 构建通过 |
| Service/Version 配置分块及三值状态 | 通过（代码/构建） | `ServiceComponentDetail.vue`、API/Proto 契约、前端 typecheck/build |
| 配置保存不自动重启，Preview/Deploy 复用同一 plan | 通过（单元测试） | fingerprint 持久化、`pending_deploy` 比较与部署用例测试 |

## Test Results

- `task sqlc`: passed.
- `task proto`: passed；Buf 报告无需要更新的依赖。
- `task check`: passed；包含 Web typecheck/lint/format 与 Go format/lint。
- `task test`: passed；Web 8 files / 49 tests，Go `./cmd/... ./internal/... ./sql` 全部通过。
- `yarn --cwd web build`: passed。
- `uv run pytest`: passed，67 passed、1 deselected。
- `uv run ruff check src tests`、`uv run mypy`: passed。
- `python -m py_compile`（业务备份迁移、RAGFlow baseline 脚本）与 `python scripts/test_export_ragflow_tei_baseline.py`: passed，3 tests。
- SQLite schema 回归：从空库执行至 v30，确认 ServiceComponent/env/Endpoint schema、`foreign_key_check` / `integrity_check`；最新 schema 亦可导入 RAGFlow baseline。
- `git diff --check`: passed。旧契约扫描未发现 `VersionComponentPort`、`ServiceExpose` 或 `runtime_config_json` 的生产使用；`runtimeConfig` 的剩余命名仅用于 Web 启动配置或通用函数名。

## Scope Deviations

无产品范围扩展。验证发现并清理了三处 Go 静态检查问题，并提升 SQLite 回归测试的完整性检查；这是实现质量修正，不新增运行时能力。

## Risks And Incomplete Items

1. 未提供可用的 MySQL 集成数据库，因此 MySQL 最终 DDL 的真实建库和约束未执行；仅完成 SQL/生成代码审阅与 SQLite schema 回归。
2. 未连接 `pomelo-orbit-ssh`，也未运行 Orbit MCP 部署或 Docker Compose lifecycle。因此无法确认正式网络已带有 `com.docker.compose.network=traefik`、Traefik 3.6 可重建、80/443 可监听、REST Provider 与公开路由可端到端访问。
3. 本轮没有浏览器人工交互验收；前端通过 typecheck、lint、单测和生产构建，但仍应在部署前人工检查 ServiceComponent 编辑/删除/待部署提示。
4. 开发数据库按当前原始 DDL 重建；不支持通过历史 schema 或数据转换回退。

## Conclusion

代码级、生成级和 SQLite schema 验证通过，且验证中发现的静态检查问题已修复。由于 MySQL 与真实 Traefik 3.6 部署验证尚未执行，当前验证记录保持 `Draft`；在受控环境完成这两项后，方可将本验证阶段标记为 `Accepted`。
