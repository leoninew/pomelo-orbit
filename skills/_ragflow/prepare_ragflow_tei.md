# `prepare_ragflow_tei.py` — 准备与校验 RAGFlow TEI 模型和镜像

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

准备并验证 RAGFlow TEI 使用的固定版本 `BAAI/bge-m3` 本地模型缓存，以及 CPU/GPU TEI 镜像。脚本提供人类可读输出、`--json` 机器可读输出和稳定的非零阻塞退出码。

默认模型缓存目录为：

```text
data/deployment/ragflow/default/tei/cache/bge-m3
```

## 先只读检查

```bash
python skills/_ragflow/prepare_ragflow_tei.py check --profile cpu
python skills/_ragflow/prepare_ragflow_tei.py check --profile gpu
```

`check` 验证 Git、Git LFS、Hugging Face CLI、模型缓存、镜像 registry manifest、Docker；GPU profile 额外验证 `nvidia-smi` 与 Docker NVIDIA runtime。可追加全局 `--json` 输出结构化结果。

## 准备命令

```bash
# 检查命令行依赖；仅在显式声明时安装/升级 huggingface_hub
python skills/_ragflow/prepare_ragflow_tei.py prepare-cli [--install-hf-cli]

# 下载固定 revision 的模型缓存
python skills/_ragflow/prepare_ragflow_tei.py prepare-model --source git-lfs
python skills/_ragflow/prepare_ragflow_tei.py prepare-model --source hf-cli

# 复用已通过 Git LFS 校验的独立缓存，复制到 RAGFlow 管理目录
python skills/_ragflow/prepare_ragflow_tei.py stage-model --source-dir data/deployment/tei-bge-m3/default/tei/cache/bge-m3

# 拉取固定 digest 的 CPU/GPU 镜像
python skills/_ragflow/prepare_ragflow_tei.py prepare-image --profile cpu
python skills/_ragflow/prepare_ragflow_tei.py prepare-image --profile gpu

# 创建默认位置 `data/backup/bge-m3-<revision>.tar.gz` 的模型归档，或从归档恢复
python skills/_ragflow/prepare_ragflow_tei.py backup-model
python skills/_ragflow/prepare_ragflow_tei.py restore-model --archive data/backup/bge-m3-<revision>.tar.gz

# 依次准备 CLI、模型和镜像，再运行检查
python skills/_ragflow/prepare_ragflow_tei.py prepare --profile cpu
```

`prepare-model` 可指定 `--model-dir`、`--source git-lfs|hf-cli` 与受限的 `--resume`。`stage-model` 先校验源缓存和 Git LFS，再复制模型文件与创建 SHA-256 manifest；源目录不会移动、修改或删除。`prepare-image`、`check` 和 `prepare` 支持 `--image` 覆盖镜像引用以进行探测或拉取。`backup-model` 默认写入 `data/backup/bge-m3-<revision>.tar.gz`，也支持 `--archive` 与显式 `--replace`；`restore-model` 仅恢复到空目标目录。

## 安全边界

`check` 是只读预检。其他子命令可能安装 Python 包、下载模型、拉取 Docker 镜像、创建归档、替换归档，或将经过 manifest 与哈希验证的归档恢复至空缓存目录。归档恢复拒绝符号链接、设备文件和路径逃逸成员；模型准备拒绝非空目标，除非对已有 Git LFS checkout 显式使用 `--resume`。

## 相关文档

RAGFlow + TEI 部署和运行验收请参阅 [`../../docs/guides/ragflow-tei-operations.md`](../../docs/guides/ragflow-tei-operations.md)。
