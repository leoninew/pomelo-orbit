---
name: deploy-ragflow-split-orbit
description: "使用 Orbit 作为首选控制面，初始化、修复、预览或部署仅拆分式 RAGFlow 拓扑：ragflow-split CPU/GPU Version 及 MySQL、Valkey、MinIO、Elasticsearch Application。仅在用户明确要求拆分式部署时使用；不初始化集成式 Application。 Initialize, repair, preview, or deploy only the split RAGFlow topology with Orbit as the preferred control plane: ragflow-split CPU/GPU Versions plus MySQL, Valkey, MinIO, and Elasticsearch Applications. Use only when split deployment is explicitly requested; do not initialize the integrated Application."
---

# 通过 Orbit 部署拆分式 RAGFlow / Deploy Split RAGFlow Through Orbit

仅对[拆分式 primitive](references/ragflow-split-primitives.md) 使用本 skill。该 primitive 由本目录的 `deployment-contract.json` 生成，是 Application 库存、组件配置、runtime key 归属、alias、endpoint、模型缓存检查和 Provider 初始化的权威依据。

Use this skill only for [the split primitive](references/ragflow-split-primitives.md). Generated from this directory's `deployment-contract.json`, the primitive is authoritative for Application inventory, component configuration, runtime-key ownership, aliases, endpoints, model-cache checks, and Provider initialization.

在发现、预检或写入前，阅读并遵循[共享 Orbit RAGFlow 部署策略](../_ragflow/orbit-deployment-policy.md)。

Read and follow the [shared Orbit RAGFlow deployment policy](../_ragflow/orbit-deployment-policy.md) before discovery, preflight, or writes.

## 前置条件 / Preconditions

1. 确认用户明确要求拆分式部署。 / Confirm that the user explicitly requests split deployment.
2. 只检查拆分式生成产物： / Check only the split generated artifacts:

   ```powershell
   python skills/deploy-ragflow-split-orbit/render_contract.py --check
   ```

   如有漂移，必须先修正并重新生成 contract，否则禁止写入。 / Contract drift blocks writes until corrected and regenerated.
3. 对拆分式 primitive 列出的每个模型缓存路径执行共享 NVIDIA/TEI 预检。 / Apply the shared NVIDIA/TEI preflight to every model-cache path in the split primitive.
4. 列出 Project、Application、Version、Service 和活动 Deployment。仅在强制单活动路由时检查拆分式 Application 和 `ragflow-integrated/default`。 / List Projects, Applications, Versions, Services, and active Deployments. Inspect split Applications and `ragflow-integrated/default` only as needed for single-active routing.

## 初始化拆分式库存 / Initialize Split Inventory

1. 仅创建或修复 `ragflow-split`、完整 Version `ragflow-split-cpu` 与 `ragflow-split-gpu`，以及 `default` Service。同时创建或修复 `ragflow-mysql`、`ragflow-redis`、`ragflow-minio` 和 `ragflow-elasticsearch`，每个都使用 primitive 定义的完整 Version 与 `default` Service。新 Service 保持停止。 / Create or repair only `ragflow-split`, both complete Versions, and its `default` Service, plus the four primitive-defined backing Applications with their complete Version and `default` Service. Keep new Services stopped.
2. 严格应用 primitive 声明，包括 RAGFlow/TEI CPU/GPU 差异、独立后端组件、mount、alias、environment、health gate 和 endpoint。 / Apply the primitive exactly, including RAGFlow/TEI CPU/GPU differences, independent backing Components, mounts, aliases, environment, health gates, and endpoints.
3. 仅当 Orbit 管理的 Application、镜像家族、health check、外部网络 hostname 和凭据 contract 兼容时，才复用后端 Service。保留正在运行且健康的兼容 Service；不静默替换数据库、缓存、对象存储或 Elasticsearch。 / Reuse a backing Service only when its Orbit-managed Application, image family, health check, external hostname, and credential contract are compatible. Preserve a compatible healthy running Service; never silently substitute backing systems.
4. 对全新且空的拆分式资源集，在内存中创建一份 `RagflowRuntimeConfig`，将必需 key 分配给 `ragflow-split/default` 及各新后端 Service。MySQL 或 Elasticsearch 数据非空时，保留匹配的已授权初始值，或停止并请求明确重置决策。 / For a new empty resource set, create one in-memory `RagflowRuntimeConfig` and distribute required keys to `ragflow-split/default` and new backing Services. For non-empty MySQL or Elasticsearch data, retain matching authorized bootstrap values or stop for an explicit reset decision.
5. 本流程不创建、修复、部署或初始化 `ragflow-integrated`。 / Do not create, repair, deploy, or initialize `ragflow-integrated`.

## 部署拆分式拓扑 / Deploy Split Topology

1. 启动前，如 `ragflow-integrated/default` 正在运行，优先通过 Orbit 以 `remove_volumes=false` 停止。Orbit 无法执行时，只能使用经明确授权的共享策略降级方案。不更改其 Version、数据或缓存。 / Before startup, prefer Orbit to stop a running `ragflow-integrated/default` with `remove_volumes=false`. If Orbit cannot do so, use only an explicitly approved shared-policy fallback. Do not alter its Versions, data, or cache.
2. 优先通过 Orbit 预览和部署每个拆分式后端 Service，等待组件健康后再启动 RAGFlow。控制面无法执行必需操作时，遵循共享降级策略。 / Prefer Orbit to preview and deploy each backing Service, then wait for health before starting RAGFlow. Apply the shared fallback policy when required.
3. 仅当检测到 GPU 且预检成功时选择 `ragflow-split-gpu`；否则选择 `ragflow-split-cpu`。 / Select `ragflow-split-gpu` only after GPU detection and successful preflight; otherwise select `ragflow-split-cpu`.
4. 仅预览和部署 `ragflow-split/default`。等待 primitive 定义的所有组件健康，执行 HTTP probe，再运行 `verify_deployment`。 / Preview and deploy only `ragflow-split/default`. Wait for all primitive-defined Components to become healthy, run the HTTP probe, then run `verify_deployment`.

## 完成与安全 / Completion and Safety

- 最终拓扑必须保持在 primitive 定义的拆分式库存内；不创建临时、重复、捆绑或模型 Application。 / Keep the final topology within the primitive-defined split inventory; do not create temporary, duplicate, bundled, or model Applications.
- 不超出 primitive endpoint contract 暴露组件，不添加 public route。 / Do not expose Components beyond the primitive endpoint contract or add public routes.
- 不迁移、挂载或复用 `ragflow-integrated` 数据目录。 / Do not migrate, mount, or reuse `ragflow-integrated` data directories.
- 健康确认后，在 RAGFlow UI 中手动配置 primitive 定义的 HuggingFace embedding provider。 / After health is confirmed, configure the primitive-defined HuggingFace embedding provider manually in the RAGFlow UI.
