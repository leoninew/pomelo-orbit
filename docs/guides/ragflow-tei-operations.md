# RAGFlow 内嵌 BGE-M3 TEI 运维手册
最后修改时间: 2026-07-31 15:53:31

本文记录集成式 RAGFlow 内嵌 Text Embeddings Inference (TEI) 的部署选择、模型准备和验收方法。集成式 Application 包含 CPU/GPU 两个完整 Version；拆分式部署使用独立的 `deploy-ragflow-split-orbit` skill。运行时部署仍只能通过 Pomelo Orbit 管理。本轮只进行 CPU 运行验收，GPU Version 仅交付静态规格与 Compose 渲染，不声称完成 GPU 容器测试。

## 方案摘要

使用一个 `ragflow` Application，维护两个包含 `mysql`、`redis`、`minio`、`es01`、`tei`、`ragflow-cpu` 的完整 Version：

- `ragflow-integrated-cpu`：通用开发和无 NVIDIA GPU 的 Host。
- `ragflow-integrated-gpu`：具备 NVIDIA Driver、NVIDIA Container Toolkit 和相应 GPU compute capability 的 Host。

TEI 是同一 Service 内部组件，不发布独立宿主机端口。RAGFlow 保留唯一的 local HTTP expose：`127.0.0.1:9380 -> ragflow-cpu:80`。RAGFlow 的 HuggingFace Embedding Provider 使用模型 `BAAI/bge-m3` 和基础地址 `http://tei:80`；不要填旧独立应用地址，也不要在地址后附加 `/embed`。

## BGE-M3 简介

`BAAI/bge-m3` 是支持 TEI 的多语言嵌入模型，Hub revision 为 `5617a9f61b028005a4858fdac845db406aefb181`。其稠密向量维度为 1024，最大序列长度为 8192 token；模型还具备稀疏和多向量能力，但 RAGFlow 的 HuggingFace Embedding 集成使用 TEI 的嵌入接口。

固定 revision 的完整下载包含 30 个文件、约 4.6 GiB。缓存中至少应存在 `config.json`、`pytorch_model.bin`、tokenizer 文件、`onnx/` 和 `onnx/model.onnx_data`；仅有目录或 `.cache` 不是可用模型。

## TEI 镜像选择

截至 2026-07-31，官方 TEI 最新正式发布为 `v1.9.3`。基线必须使用不可变 digest，不能使用 `latest` 或 `cpu-latest`。

| 场景 | 官方标签 | 已验证上游 digest | 说明 |
| --- | --- | --- | --- |
| CPU x86_64 | `cpu-1.9.3` | `sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07` | CPU Version 的基线。 |
| 通用 CUDA | `cuda-1.9.3` | `sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a` | GPU Version 的可移植初始选择。 |
| Ampere SM 8.0 | `1.9.3` | `sha256:536efce2a0dc0acd0336d683ef1b81fcf900f76c8fb850e25a6d48a54a97df91` | A100、A30 等。 |
| Ampere SM 8.6 | `86-1.9` | `sha256:a7d82dfef16c3bf1a95e93f5b226f358312512dbb0d585b48c3cf886f9d470a9` | A10、A40 等。 |
| Ada SM 8.9 | `89-1.9` | `sha256:e47e625ced2385d3dbfdee79ba0380204578e0b27ef1a926783f9b3486aaf109` | RTX 4000 系列等。 |
| Hopper SM 9.0 | `hopper-1.9` | `sha256:e3009cd99f63dbabd29346412e2d65224e4ab179e7559b0065f26e38fb7334c4` | H100 等。 |

通用 CUDA Version 是跨 GPU Host 的初始 Variant。追求性能时，应在 GPU Host 读取 compute capability 后先同步修改固定镜像契约、Compose 覆盖和导出校验，再原地更新 `ragflow-integrated-gpu` 并重新导出基线；当前集成式交付不得新增临时或第三个 Version。TEI 官方资料指出 compute capability 低于 7.5 的 NVIDIA GPU 不受支持；目标 Host 的 NVIDIA 驱动需要兼容 CUDA 12.2 及以上。

## Compose 参考

`scripts/ragflow-bundled/docker-compose.yml` 与 `scripts/ragflow-split/docker-compose.ragflow.yml` 是 CPU 参考。对应的 `*.gpu.yml` 是仅覆盖 `tei` 的 GPU 参考：固定 CUDA digest 并声明 `nvidia`、全部 GPU、`gpu` capability。Docker Compose 的规范化输出将 `count: all` 表示为 `count: -1`，两者等价。

这些文件只用于审阅和 `docker compose ... config` 预览；实际创建、更新、部署和停止仍通过 Pomelo Delivery MCP（`pomelo_delivery`）完成。

## 模型缓存准备

目标路径为：

```text
data/deployment/ragflow/default/tei/cache/bge-m3
```

TEI 挂载其父目录 `tei/cache` 到 `/data`，启动参数为：

```text
--model-id /data/bge-m3 --json-output
```

同一目标目录只选一种下载方式，完成后再启动 Service。

已有独立 TEI 的 Git LFS 缓存通过校验时，优先使用前置脚本复制到 RAGFlow 的新管理路径，而不是重新下载或移动原目录：

```powershell
python skills/_ragflow/prepare_ragflow_tei.py stage-model --source-dir data/deployment/tei-bge-m3/default/tei/cache/bge-m3
```

该命令要求目标目录为空，复制时忽略 Git 元数据和下载缓存，创建逐文件 SHA-256 manifest，并保留原始 Git LFS checkout 不变。

### 压缩备份与反复测试

在 Git LFS 工作树通过 `git lfs fsck` 和文件预检后，使用前置脚本创建 `tar.gz` 归档及逐文件 SHA-256 manifest。默认归档路径为项目根 `data/backup/bge-m3-<revision>.tar.gz`；归档只包含供 TEI 使用的模型工作树，不包含重复的 `.git` 或 `.cache` 内容。权重文件本身通常已压缩，归档未必显著减小体积，必须记录脚本报告的实际大小和耗时。

恢复只允许写入空目标目录，恢复后使用 manifest 和文件清单重新校验。恢复目录不含 Git 元数据，因此用归档 SHA-256 代替 `git lfs fsck`；原始 Git LFS 工作树继续用 `git lfs fsck` 验证。脚本不得自动覆盖非空缓存、移动或删除原始独立缓存。

### Git LFS：默认方式

Git LFS 可续传，并可用 `git lfs fsck` 校验 LFS 对象，适合作为默认方案。

```powershell
git lfs install
git lfs clone https://huggingface.co/BAAI/bge-m3 data/deployment/ragflow/default/tei/cache/bge-m3
git -C data/deployment/ragflow/default/tei/cache/bge-m3 lfs fsck
```

当目录已经是有效 Git checkout 时，使用 `git lfs pull` 恢复中断下载。不要删除或覆盖非空缓存来“修复”它；先执行预检并处理结果。

### Hugging Face CLI：等价后备

Hugging Face CLI 适合固定 revision 下载和 dry-run 盘点：

```powershell
hf download BAAI/bge-m3 --revision 5617a9f61b028005a4858fdac845db406aefb181 --local-dir data/deployment/ragflow/default/tei/cache/bge-m3 --dry-run
```

实际下载只应在空目标目录中执行。`--dry-run` 也可能创建 `.cache` 和空目录，不能以“目录存在”判定就绪。已安装 CLI 为 `1.9.2`，当前 PyPI 的 `huggingface_hub` 为 `1.26.0`；前置脚本应检测 `hf download --dry-run` 能力，而非假设机器安装的确切版本。

### 路径选择比较

| 方式 | 可靠性 | 效率 | 结论 |
| --- | --- | --- | --- |
| Git LFS 预下载 | 可续传，可 `fsck` 校验 | 首次约 4.6 GiB，后续复用本地缓存 | 默认。 |
| HF CLI 预下载 | 可固定 revision、支持 dry-run | 同样需完整下载；无 LFS 对象校验 | 后备。 |
| TEI 启动时远程下载 | 启动依赖外网，失败难以区分 | 首次启动长且不可预测 | 不作为基线。 |
| Docker/registry mirror | 依赖第三方可用性和内容一致性 | 可能减少拉取延迟 | 仅人工切换的故障后备。 |

## 上游与备用镜像站

基线拉取源为 `ghcr.io/huggingface/text-embeddings-inference`。2026-07-31 的 manifest 探针中，上游 `cpu-1.9.3` 返回 200，观测延迟约 547 ms。

本机配置了 `docker.1ms.run`、`hub.rat.dev`、`dockerproxy.net` 和 `proxy.vvvv.ee`。针对 GHCR TEI manifest 的结果：前两者返回 401，`proxy.vvvv.ee` 返回 403；`dockerproxy.net` 的 HEAD 返回 200，但 GET 返回 HTML 而不是 OCI manifest，未通过内容校验。因此当前没有可自动采用的备用站点。

使用任一备用站前，必须验证目标镜像 manifest 可解析且 digest 与上游一致；不能把镜像站域名写入 SQL 基线或固定 Version。

## 已完成验证

| 验证 | 结果 |
| --- | --- |
| BGE-M3 Git LFS 缓存 | `git lfs fsck` 通过。 |
| Hub revision | `git ls-remote` 返回 `5617a9f61b028005a4858fdac845db406aefb181`。 |
| HF CLI dry-run | 成功列出 30 个文件、约 4.6 GiB。 |
| 当前独立 TEI `/health` | HTTP 200。 |
| 当前独立 TEI `/embed` | HTTP 200，返回 1024 维向量，验证请求约 2.58 秒。 |
| 官方 CPU/CUDA 镜像 | 官方 GHCR manifest 可解析，digest 见上表。 |
| GPU 运行 | 未验证：本机没有 NVIDIA GPU。 |

## GPU Host 前置条件（本轮不运行验收）

1. 用 `nvidia-smi` 确认 GPU 型号、Driver 和 compute capability。
2. 安装并验证 Docker NVIDIA Container Toolkit；Docker Compose 渲染必须含 NVIDIA GPU device reservation，不能仅设置 CPU/内存资源。
3. 选择对应计算能力的 TEI 镜像，或先使用 `cuda-1.9.3` 验证再创建优化 Version。
4. 本轮不通过 Orbit 部署 GPU Version，也不执行 GPU `tei` 健康或嵌入请求；最终 SQL 和交付说明必须将该 Version 标记为 `GPU runtime 未验证`。
5. 后续 GPU 验收通过后，再从 RAGFlow 配置 HuggingFace Embedding Provider，并在同一知识库重新解析或重建索引。更换嵌入模型后，旧向量不得与新向量混用。

## RAGFlow 初始化与恢复

RAGFlow 的模型供应商配置保存在 RAGFlow 自己的 MySQL 中，不属于 Orbit Application/Version/Service SQL。控制面基线恢复后，使用 RAGFlow UI 表单完成以下初始化：

1. 配置 HuggingFace Embedding 实例：`tei-bge-m3`、模型名 `BAAI/bge-m3`、基础地址 `http://tei:80`、Max tokens `8192`。
2. 执行连接校验并设为默认嵌入模型。
3. 创建或重建知识库索引。

本轮不调用 RAGFlow API 自动初始化：该路径需要先注册用户并获得 API key，不是 Orbit 或模型缓存的前置能力。

SQL 基线应只记录停机状态的 Application、CPU/GPU Version、Gateway、Service 和 local RAGFlow expose；运行时配置、Deployment 历史和“running”状态由 Orbit 在恢复后创建。

## 前置准备脚本

实施后统一使用 `skills/_ragflow/prepare_ragflow_tei.py` 准备 TEI 运行条件。该脚本会以 `argparse` 子命令和清晰退出码呈现所有可选路径，供人和自动化 Agent 使用：

```powershell
# 只读：检查工具、缓存、镜像 digest 和指定 Host 的 CPU/GPU 条件
python skills/_ragflow/prepare_ragflow_tei.py check --profile cpu
python skills/_ragflow/prepare_ragflow_tei.py check --profile gpu

# 显式执行：按 Git LFS 或 HF CLI 准备固定 revision 的模型
python skills/_ragflow/prepare_ragflow_tei.py prepare-model --source git-lfs
python skills/_ragflow/prepare_ragflow_tei.py prepare-model --source hf-cli

# 显式压缩备份已校验模型，并恢复到空缓存目录供下一轮测试
python skills/_ragflow/prepare_ragflow_tei.py backup-model
python skills/_ragflow/prepare_ragflow_tei.py restore-model --archive data/backup/bge-m3-<revision>.tar.gz

# 显式拉取已验证的 TEI image digest，并检查备用镜像站
python skills/_ragflow/prepare_ragflow_tei.py prepare-image --profile cpu
python skills/_ragflow/prepare_ragflow_tei.py prepare-image --profile gpu

# 按 profile 执行所有经明确授权的准备步骤
python skills/_ragflow/prepare_ragflow_tei.py prepare --profile cpu
```

脚本不会启动或停止容器，不执行 Docker Compose，也不会创建或修改 Orbit Application、Version、Service 或 Deployment。通过 `check` 后，集成式使用 `deploy-ragflow-integrated-orbit`，显式拆分式使用 `deploy-ragflow-split-orbit` 执行 Orbit 生命周期操作。
