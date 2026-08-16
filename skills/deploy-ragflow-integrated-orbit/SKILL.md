---
name: deploy-ragflow-integrated-orbit
description: "使用 Orbit 作为首选控制面，初始化、修复、预览或部署仅集成式 RAGFlow 拓扑：一个 ragflow-integrated Application 及 CPU/GPU 六组件 Version。用于默认一体化 RAGFlow 部署；不初始化拆分式或后端 Application。 Initialize, repair, preview, or deploy only the integrated RAGFlow topology with Orbit as the preferred control plane: one ragflow-integrated Application with CPU and GPU six-component Versions. Use for the default all-in-one deployment; do not initialize split or backing Applications."
---

# 通过 Orbit 部署集成式 RAGFlow / Deploy Integrated RAGFlow Through Orbit

对一体化拓扑使用本 skill，并以[集成式 primitive](references/ragflow-integrated-components.md) 为准。该 primitive 由本目录的 `deployment-contract.json` 生成，是组件配置、runtime key 归属、alias、endpoint、模型缓存检查和 Provider 初始化的权威依据。

Use this skill for the all-in-one topology and follow [the integrated primitive](references/ragflow-integrated-components.md). Generated from this directory's `deployment-contract.json`, the primitive is authoritative for component configuration, runtime-key ownership, aliases, endpoints, model-cache checks, and Provider initialization.

在发现、预检或写入前，阅读并遵循[共享 Orbit RAGFlow 部署策略](../_ragflow/orbit-deployment-policy.md)。

Read and follow the [shared Orbit RAGFlow deployment policy](../_ragflow/orbit-deployment-policy.md) before discovery, preflight, or writes.

## 前置条件 / Preconditions

1. 确认请求为集成式拓扑。用户未明确要求拆分式时，默认选择集成式。 / Confirm the request is for the integrated topology. Default to integrated unless the user explicitly requests split deployment.
2. 只检查集成式生成产物： / Check only the integrated generated artifacts:

   ```powershell
   python skills/deploy-ragflow-integrated-orbit/render_contract.py --check
   ```

   如有漂移，必须先修正并重新生成 contract，否则禁止写入。 / Contract drift blocks writes until corrected and regenerated.
3. 对集成式 primitive 列出的每个模型缓存路径执行共享 NVIDIA/TEI 预检。 / Apply the shared NVIDIA/TEI preflight to every model-cache path in the integrated primitive.
4. 列出 Project、Application、Version、Service 和活动 Deployment。仅在强制单活动路由时检查 `ragflow-integrated` 和 `ragflow-split/default`。 / List Projects, Applications, Versions, Services, and active Deployments. Inspect `ragflow-integrated` and `ragflow-split/default` only as needed for single-active routing.

## 初始化集成式库存 / Initialize Integrated Inventory

1. 仅创建或修复 `ragflow-integrated`、完整六组件 Version `ragflow-integrated-cpu` 与 `ragflow-integrated-gpu`，以及 `default` Service。新 Service 初始化后保持停止。 / Create or repair only `ragflow-integrated`, both complete six-component Versions, and its `default` Service. Keep new Services stopped after initialization.
2. 严格应用 primitive 的组件声明，包括 TEI CPU/GPU 镜像差异、GPU device request、mount、alias、environment、health gate 和 endpoint。 / Apply the primitive exactly, including TEI CPU/GPU image differences, GPU device requests, mounts, aliases, environment, health gates, and endpoints.
3. 对全新且空的集成式数据边界，在内存中创建一份 `IntegratedRuntimeConfig`，仅应用于 `ragflow-integrated/default`。数据非空时，保留匹配的已授权值，或停止并请求明确重置决策。 / For a new empty boundary, create one in-memory `IntegratedRuntimeConfig` and apply it only to `ragflow-integrated/default`. For non-empty data, retain matching authorized values or stop for an explicit reset decision.
4. 本流程不创建、修复、部署或初始化 `ragflow-split` 及任何拆分式后端 Application。 / Do not create, repair, deploy, or initialize `ragflow-split` or split backing Applications.

## 部署集成式拓扑 / Deploy Integrated Topology

1. 启动前，如 `ragflow-split/default` 正在运行，优先通过 Orbit 以 `remove_volumes=false` 停止。Orbit 无法执行时，只能使用经明确授权的共享策略降级方案。不更改其 Version、后端 Service、数据或缓存。 / Before startup, prefer Orbit to stop a running `ragflow-split/default` with `remove_volumes=false`. If Orbit cannot do so, use only an explicitly approved shared-policy fallback. Do not alter its Versions, backing Services, data, or cache.
2. 仅当检测到 GPU 且预检成功时选择 `ragflow-integrated-gpu`；否则选择 `ragflow-integrated-cpu`。 / Select `ragflow-integrated-gpu` only after GPU detection and successful preflight; otherwise select `ragflow-integrated-cpu`.
3. 仅预览和部署 `ragflow-integrated/default`。等待 primitive 定义的所有组件健康，执行 HTTP probe，再运行 `verify_deployment`。 / Preview and deploy only `ragflow-integrated/default`. Wait for all primitive-defined Components to become healthy, run the HTTP probe, then run `verify_deployment`.

## 完成与安全 / Completion and Safety

- 组件必须保持在 primitive 的 endpoint contract 内；不添加 local、host 或 public route。 / Keep Components within the primitive endpoint contract; do not add local, host, or public routes.
- 不将集成式存储作为拆分式数据的迁移目标。 / Do not use integrated storage as a migration target for split data.
- 健康确认后，在 RAGFlow UI 中手动配置 primitive 定义的 HuggingFace embedding provider。 / After health is confirmed, configure the primitive-defined HuggingFace embedding provider manually in the RAGFlow UI.
