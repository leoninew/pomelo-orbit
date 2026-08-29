# RAGFlow 内嵌 TEI CPU/GPU 版本需求
最后修改时间: 2026-07-31 13:01:30

Review status: Accepted

## Background

当前开发库已有独立的 `bge-m3` Application，RAGFlow 通过外部 Traefik 别名调用其 TEI 端点。业务需要将 `BAAI/bge-m3` 的 Text Embeddings Inference (TEI) 合并到 `ragflow` Application 中，并以应用 Version 区分 CPU 与 NVIDIA GPU 部署变体。

现有本地 Git LFS 模型缓存已通过 `git lfs fsck`，但位于独立 Application 的挂载路径。RAGFlow 迁移后不能继续依赖独立 Application 的网络别名或模型目录。

## Goal

1. 以“模型预下载 + 本地挂载”为默认部署方式：在部署前通过 Git LFS 将 `BAAI/bge-m3` 下载到 RAGFlow 管理目录，TEI 只加载 `/data/bge-m3`，不在容器启动时下载模型；完整缓存必须能显式压缩备份、校验并恢复到空目录，以支持反复测试。
2. 在同一个 `ragflow` Application 中提供两个完整的六组件 Version：`mysql`、`redis`、`minio`、`es01`、`tei`、`ragflow-cpu`。
3. CPU Version 使用经验证的 TEI CPU 镜像摘要；GPU Version 使用相同 TEI 发布线的 NVIDIA CUDA 镜像摘要，并显式请求 NVIDIA GPU 设备。
4. `default` Service 可以从停止状态切换 CPU/GPU Version；两种 Version 只公开 RAGFlow 的本地 HTTP 入口 `127.0.0.1:9380 -> 80`，TEI 仅供同一 Service 内部访问。
5. RAGFlow 中 HuggingFace Embedding Provider 使用基础地址 `http://tei:80`，模型名 `BAAI/bge-m3`，由 RAGFlow 自身初始化步骤配置。
6. 在 CPU Version 与 Gateway 成功部署、验证，且 CPU/GPU Version 的静态数据和 Compose 渲染均通过校验后，生成可重建的 SQLite 基线 SQL，覆盖 Project、Application、Gateway、Version、Component、Service 和 Expose 的结构化数据；RAGFlow Service 的运行时密码在导出时随机生成，不复制现有实例；GPU Version 必须显式标为“未进行运行验证”。
7. 交付一份简明的运维手册，包含本次外部版本调研、模型与镜像选择、缓存准备、CPU/GPU 选择、验证结果、故障处理和恢复步骤；对话中取得的版本、摘要、容量和验证结论必须进入手册。
8. 提供一个可直接被人或 LLM 阅读和调用的 Python 前置脚本，统一准备模型、CLI 和 TEI 镜像；合并后的 RAGFlow+TEI Orbit 技能必须引用该脚本，而不是重复维护下载和环境准备命令。

## Non-goal

- 不运行或声称验证 GPU 容器；本轮不具备 NVIDIA GPU 测试条件，GPU 运行验收不属于本轮交付。
- 不让 TEI 使用远程 `BAAI/bge-m3` model ID 自动下载模型。
- 不将 TEI 单独作为一个长期运行的 `bge-m3` Application 保留在最终拓扑中。
- 不通过 Docker Compose 生命周期命令管理 Orbit 部署。
- 不把 RAGFlow 内部 MySQL 的模型供应商数据误认为 Orbit 控制面 SQL 的一部分。
- 不通过 RAGFlow API 自动初始化模型供应商；该路径需要先注册用户并取得 API key，超出本轮范围。

## User scenarios

### CPU 开发机

开发者在 `data/deployment/ragflow/default/tei/cache/bge-m3` 完成并校验 Git LFS 模型下载后，选择 CPU Version 部署。TEI 从本地挂载加载模型，RAGFlow 通过 `http://tei:80` 使用嵌入能力。

### GPU 部署机

部署机具备 NVIDIA Driver、Docker NVIDIA Container Toolkit 与兼容 CUDA 环境。开发者选择 GPU Version；渲染的 Compose 必须含 NVIDIA GPU device reservation，TEI 使用 GPU 镜像和相同本地模型挂载。

### 可重建开发环境

开发者迁移空 SQLite 数据库后导入基线 SQL，使用 Orbit 设置该环境的运行时配置并选择 CPU 或 GPU Version 部署。Gateway 由受管 Gateway 流程重建，不依赖旧 Deployment 记录伪造运行态。

## Acceptance

- Git LFS 缓存预检要求 `config.json`、tokenizer 文件、主权重与 `onnx/` 目录，且 `git lfs fsck` 通过。
- 已验证的模型缓存可创建到项目根 `data/backup/bge-m3-<revision>.tar.gz` 的带 SHA-256 manifest 的 `tar.gz` 压缩备份；从备份恢复到空目录后，文件清单和哈希必须通过，TEI 可将其用作本地模型目录。
- CPU/GPU Version 组件清单完全一致，仅 `tei` 镜像、GPU 设备请求与必要资源参数不同。
- `ragflow-cpu` 对 `tei` 使用 `service_healthy` 依赖；TEI 健康检查检查 `http://127.0.0.1:80/health`，允许模型加载期。
- RAGFlow Service 只保留 `ragflow-cpu:80` 的 local expose；TEI 不发布宿主机端口或 public 路由。
- GPU Version 只有在目标 Host 声明 NVIDIA 能力时才可部署；缺失能力时给出明确校验失败，不能静默以 CPU 方式运行。
- 控制面 SQL 可在已迁移的空 SQLite 数据库上重建两个停机状态的 Version/Service 基线，不携带旧 Deployment、运行态状态或环境特定运行时配置值；其中 RAGFlow Service 含新生成的五项运行时变量以支持首次启动。
- RAGFlow 模型供应商配置有独立的、可重复执行的初始化说明或自动化步骤，且不会把 `/embed` 附加到基础 URL。

## Decisions

- 采用 Git LFS 作为默认模型获取方式；Hugging Face CLI 仅作为等价的手动下载后备，不是 TEI 运行时依赖。
- 镜像镜像站可作为目标环境的 pull-through 配置，但不写入拓扑或 SQL 基线；基线使用上游镜像不可变摘要。
- 使用一个 `ragflow` Application、两个完整 Version 和一个可切换的 `default` Service。该决定取代现有 RAGFlow 技能中“五组件、单 Version”的限制。
- `scripts/ragflow-split/` 仅作为显式 split 部署的替代参考；其 RAGFlow Compose 同样内嵌 TEI，不再保留独立的 `tei-bge-m3` Compose。
- CPU 使用当前已验证的 TEI CPU 镜像摘要作为初始基线；GPU 使用与其同一 TEI 发布线的固定 CUDA 镜像摘要，但本轮只验证其数据和 Compose 渲染，不进行 GPU 运行验收。
- 需要给 Version Component 增加明确的 GPU device request 表达和 Compose 渲染支持；现有 CPU/内存资源字段不足以表达该需求。
- 新增 `scripts/prepare_ragflow_tei.py` 作为部署前唯一准备入口。脚本使用 Python 标准库和 `argparse`，提供清晰的 `--help`、非零退出码和可机器读取的结果，使人和 LLM 都能识别路径、可选下载途径、已完成项与阻塞项。
- 脚本至少提供 `check`、`prepare-model`、`prepare-image`、`backup-model`、`restore-model` 和组合的 `prepare` 命令；默认 Git LFS、可选 HF CLI，按固定 model revision 和 TEI image digest 工作。它检查 CLI、缓存、备份 manifest、GPU Host 前置条件、上游/备用镜像 manifest，且不能把目录存在误报为模型就绪。
- 工具安装、完整模型下载、镜像拉取和显式清理必须是用户选择的写操作；脚本不得自动删除非空模型缓存、不得启动/停止容器、不得调用 Docker Compose。
- `skills/deploy-ragflow-orbit` 与 `skills/deploy-bge-m3-orbit` 的职责最终合并到 `skills/deploy-ragflow-tei-orbit`。该技能在任何 Orbit 生命周期写入前调用脚本的只读 `check`，在模型/镜像未就绪时明确指向对应的脚本命令。

## Open questions

- GPU CUDA/Driver 的真实兼容性与镜像优化留待具备 NVIDIA Host 的后续任务，本轮不作为阻塞或验收项。
- 现有独立 Git LFS 缓存将复制到 RAGFlow 管理目录并在通过校验后压缩备份；不自动移动或删除原缓存。

## Risk

- 当前 Orbit API、领域模型和 Compose 渲染器不支持 GPU device reservation；仅新增 GPU Version 数据会在 GPU Host 上退化为无 GPU 的容器。
- RAGFlow 与 TEI 合并后，旧地址 `http://tei-bge-m3-tei:80` 失效，必须改为同一 Compose Service 名称 `http://tei:80`。
- 控制面 SQL 无法导出 RAGFlow 内部 MySQL 的模型供应商配置；缺少独立初始化会导致容器健康但嵌入模型未配置。
- GPU 运行时兼容性未经验证；交付中的 GPU Version 只能作为未来 NVIDIA Host 的受控部署规格，不能被标示为已运行验收。

## Delivery manual requirements

手册必须至少包含以下已经或即将获得的证据，且区分“本机已验证”、“本轮 GPU 运行验证跳过”和“后续 GPU Host 待验证”：

- TEI 官方最新发布版本、CPU/CUDA/计算能力专用镜像的不可变摘要，以及为什么不使用 `latest` 或 `cpu-latest`。
- BGE-M3 的 Hub revision、模型能力（1024 维、8192 token、多语言）和完整缓存的预期大小。
- Git LFS、Hugging Face CLI、容器自动下载、registry mirror 四种路径的前置条件、验证方式、可靠性和效率比较。
- Git LFS 缓存完整性检查、Hub 元数据 dry-run、TEI `/health` 与 `/embed` 验证命令及预期结果。
- 模型压缩备份的格式、manifest、归档大小/耗时、恢复到空目录后的校验和复测步骤。
- NVIDIA GPU 的 compute capability 到 TEI GPU 镜像标签的选择表、Docker NVIDIA Container Toolkit 前置条件，以及本机无法验证 GPU 的限制。
- CPU/GPU Version 的切换、RAGFlow HuggingFace Provider 初始化、SQLite 基线导入和 Gateway 重建流程。

## Research evidence

- 2026-07-31：官方 GitHub Release 显示 TEI 当前正式版为 `v1.9.3`。上游 GHCR manifest 已验证 CPU `cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07`、通用 Ampere `1.9.3@sha256:536efce2a0dc0acd0336d683ef1b81fcf900f76c8fb850e25a6d48a54a97df91`、CUDA `cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a` 及计算能力专用 `86-1.9`、`89-1.9`、`hopper-1.9` 标签可解析。
- 2026-07-31：Hub API 显示 `BAAI/bge-m3` revision 为 `5617a9f61b028005a4858fdac845db406aefb181`，标记为 TEI compatible；HF CLI dry-run 显示完整下载为 30 个文件、约 4.6 GiB。
- 2026-07-31：本机 Git LFS 为 `3.7.1`（也是当前正式版）；现有 BGE-M3 checkout 的 `git lfs fsck` 通过。安装的 Hugging Face CLI 为 `1.9.2`，PyPI 当前 `huggingface_hub` 为 `1.26.0`；前置脚本必须检查 CLI 是否具备 `hf download --dry-run`，而非假设特定本地版本。
- 2026-07-31：现有 TEI CPU 实例已完成本机 HTTP 验证：`/health` 返回 200；`/embed` 返回 1024 维向量，单次验证耗时约 2.58 秒。这验证了当前 BGE-M3 Git LFS 缓存和 TEI API 的可用性，不代表 TEI 1.9.3 或 GPU Variant 已完成运行验证。
- 2026-07-31：HF CLI dry-run 成功列出固定 revision 的完整 30 文件、约 4.6 GiB 下载计划，但同时创建了不含权重的目录骨架；该测试产物已清理。目录存在不是缓存就绪依据。
- 2026-07-31：上游 GHCR `cpu-1.9.3` manifest 返回 200，观测延迟约 547 ms。Docker 配置的四个备用站中，`docker.1ms.run`、`hub.rat.dev` 返回 401，`proxy.vvvv.ee` 返回 403；`dockerproxy.net` 对 HEAD 返回 200 但 GET 返回 HTML 而非 OCI manifest，未通过内容校验。当前没有可自动采用的 GHCR 备用镜像站；镜像站必须在每次使用前验证 manifest 与上游 digest。

## User review notes

- 2026-07-31：用户确认即使当前机器没有 NVIDIA GPU，仍必须提供可在 GPU 机器部署的 Version；应用现有 Version 机制应承担 CPU/GPU 变体选择。
- 2026-07-31：用户将流程切换为标准模式 / standard，并明确要求保留当前 Spec / 规格审查节点后再进入计划。
- 2026-07-31：用户确认本轮跳过 GPU 运行测试；模型缓存需要 `tar.gz` 压缩备份以支持反复测试；RAGFlow Provider 固定使用表单初始化，API 注册和 API key 流程不纳入本轮。
