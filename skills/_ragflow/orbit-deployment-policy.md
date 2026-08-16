# Orbit RAGFlow 共享部署策略 / Shared Deployment Policy

使用任一 RAGFlow 部署 skill 前阅读本策略。 / Read this policy before using either RAGFlow deployment skill.

## 权威依据与流程 / Authority and Workflow

- 以所选 skill 的 `deployment-contract.json` 及生成 primitive 为操作权威。运行 renderer `--check`；contract 漂移时禁止写入。 / Use the selected skill's contract and generated primitive as operational authority. Run the renderer with `--check`; drift blocks writes.
- `docs/archive/` 仅作历史背景。不使用 `orbit_bootstrap_application`、已删除的 `scripts/ragflow_initialize.py` 或从归档文档重建的流程。 / Treat `docs/archive/` as historical context only. Do not use removed or reconstructed legacy workflows.
- 未指定拓扑时默认使用集成式 skill；仅在用户明确要求时使用拆分式 skill。 / Default an unspecified request to integrated; use split only when explicitly requested.
- 保持两种拓扑的库存与存储边界独立；不共享、挂载或迁移数据目录。 / Keep integrated and split inventory and storage boundaries independent; never share, mount, or migrate data directories.

## Pomelo Delivery 访问 / Access

- 使用 `pomelo_delivery` Codex MCP server；本地 stdio 命令为 `go run ./cmd/server mcp`，控制面工具保留 `orbit_*` 名称。 / Use the `pomelo_delivery` Codex MCP server; its local stdio command is `go run ./cmd/server mcp`, and control-plane tools retain `orbit_*` names.
- 首次使用时允许 Orbit 浏览器登录/授权；bearer credential 只能保存在本地用户配置目录。 / On first use, allow browser authorization; keep the bearer credential only in the local user configuration directory.
- 每次写入前立即检查实时 MCP tool definition；不从实现细节、归档文档或旧版工具推断参数。 / Inspect live MCP tool definitions immediately before each write; never infer parameters from implementation details, archives, or prior versions.
- 优先通过 `pomelo_delivery` 执行生命周期写入，使 Orbit 保持控制面所有权。 / Prefer `pomelo_delivery` lifecycle writes so Orbit remains the control-plane owner.
- 实时 MCP 不可用、不健康或无法安全表达所需操作时，先穷尽只读诊断。说明能力缺口、精确降级范围、状态所有权风险和恢复路径；使用 Docker Compose 或其他机制前取得用户明确授权。 / If MCP is unavailable, unhealthy, or insufficient, exhaust read-only diagnosis first. Explain the gap, exact scope, ownership risk, and recovery path; obtain explicit approval before fallback lifecycle operations.
- 获批降级时，仅使用所选 skill 的生成 Compose 参考，仅修改其拥有的拓扑，除非明确要求否则保留 volume，并在控制面恢复后对账可观察状态。不静默切换 Orbit/Compose 所有权。 / For approved fallback, use only generated references, limit changes to the owned topology, preserve volumes by default, reconcile state afterward, and never silently alternate ownership.

## 网关与单活动路由 / Gateway and Single-active Routing

- 同一时间只允许一个 RAGFlow Service 拥有 primitive 定义的 `ragflow-cpu:80` gateway route。启动一种拓扑前，优先通过 Orbit 以 `remove_volumes=false` 停止另一个；Orbit 无法执行时遵循降级策略。 / Only one RAGFlow Service may own the gateway route. Prefer Orbit to stop the other with `remove_volumes=false`; otherwise follow the approved fallback policy.
- 将 `127.0.0.1:8080` 的受管 Traefik API 视为本地控制面状态，而非公开 endpoint。 / Treat the managed Traefik API at `127.0.0.1:8080` as local control-plane state, not a public endpoint.
- 部署前运行 `runtime_doctor(network_name="traefik")`；仅当受管 Gateway 或网络不健康时调用 `orbit_provision_gateway`。 / Run `runtime_doctor(network_name="traefik")`; call `orbit_provision_gateway` only for an unhealthy managed Gateway or network.

## 硬件与 TEI 预检 / Hardware and TEI Preflight

- 以只读方式检测 NVIDIA，再对所选拓扑拥有的每个模型缓存路径运行 `skills/_ragflow/prepare_ragflow_tei.py check --profile <cpu|gpu> --model-dir <primitive-path>`。 / Detect NVIDIA read-only, then run the TEI check for every owned model-cache path.
- 仅当检测到 NVIDIA 且 GPU 预检成功时选择 GPU。如已检测到 GPU 但预检失败，阻止部署，不静默降级到 CPU。 / Select GPU only after detection and successful GPU preflight. A failed preflight on detected GPU blocks deployment; do not silently fall back.
- 启动前必须具备所选模型缓存与镜像，但库存初始化前无需。`orbit_deploy` 或任何启动/重启前重跑预检。 / Require the selected cache and image before startup, not inventory initialization. Rerun preflight before deploy, start, or restart.
- 安装工具、下载模型、拉取镜像、暂存缓存、恢复归档或覆盖非空数据前请求授权。 / Ask for authorization before installing, downloading, pulling, staging, restoring, or overwriting non-empty data.

## 密钥与报告 / Secrets and Reporting

- 将 runtime configuration 保留在内存中。除非所选 skill 明确要求已授权操作，否则不打印、导出、持久化、推断或轮换凭据。 / Keep runtime configuration in memory; do not print, export, persist, infer, or rotate credentials without an explicitly authorized operation.
- 报告前脱敏 tool result；不暴露原始 runtime configuration、deployment preview、deployment record、log 或 bearer credential。 / Sanitize tool results; do not expose raw runtime configuration, previews, records, logs, or bearer credentials.
