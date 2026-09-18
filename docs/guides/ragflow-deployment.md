# RAGFlow 部署手册

最后修改时间: 2026-09-10 18:30:00

本文是当前 RAGFlow 部署的唯一运维依据。不提供部署脚本或 Codex Skill；所有配置、预览、部署、停止和回滚均通过 Pomelo Orbit 完成。

## 部署范围

当前运行形态为一个 `ragflow-integrated/default` Service。MySQL、MinIO、Redis 已拆分为独立 Service；Elasticsearch、TEI、MinerU 与 RAGFlow 保留为该 Service 内部 Component。

不要重新建立历史上的一体化或拆分式拓扑。它们的存储归属和跨 Service 地址均与当前部署不兼容。

## 当前拓扑

| 角色 | 所属 | 内部地址 | 持久化位置 |
| --- | --- | --- | --- |
| MySQL | `mysql-default` / `mysql` | `mysql-mysql:3306` | `<workspace_root>/mysql-default/mysql` |
| MinIO | `minio-default` / `minio` | `minio-minio:9000` | `<workspace_root>/minio-default/minio` |
| Redis | `redis-default` / `redis` | `redis-redis:6379` | `<workspace_root>/redis-default/redis` |
| Elasticsearch | RAGFlow `es01` Component | `es01:9200` | `<workspace_root>/ragflow-integrated-default/es01` |
| TEI | RAGFlow `tei` Component | `tei:80` | BGE-M3 模型缓存挂载至 `/data/bge-m3` |
| MinerU | RAGFlow `mineru` Component | `mineru:8000` | 使用镜像中的本地模型源 |
| RAGFlow | RAGFlow `ragflow` Component | `ragflow:80` | `<workspace_root>/ragflow-integrated-default/ragflow/logs` |

外置数据 Service 与 RAGFlow Service 通过受管 `traefik` Docker 网络通信。Component 名称只在所属 Compose 项目中可解析；跨 Service 连接必须使用表中的别名地址。

## RAGFlow Component

选定的 RAGFlow Version 必须仅包含下列 Component：

| Component | 镜像与职责 | 依赖 |
| --- | --- | --- |
| `es01` | Elasticsearch 8.11.3 | 无 |
| `tei` | 使用本地 BGE-M3 缓存的 GPU TEI | 无 |
| `mineru` | `mineru-gpu:latest`，在 8000 端口提供 API | 无 |
| `ragflow` | `infiniflow/ragflow:v0.26.4` | 健康的 `es01`、`tei`、`mineru` |

`mineru` 是内部 HTTP Component。其镜像启动入口必须保留为 `mineru-api`；这是可执行文件名，不是 Orbit Component 名称。除非明确需要向外暴露 API，不为它配置网关 Endpoint。

`tei` 与 `mineru` 都请求 NVIDIA GPU。部署前确认宿主机的 GPU 运行时与显存能同时满足两者。

## 环境变量

Version 只声明 `${KEY}` 占位符，具体值只保存在 RAGFlow Service 环境变量中。

| Key | 当前值或来源 | 使用方 |
| --- | --- | --- |
| `MYSQL_HOST` | `mysql-mysql` | RAGFlow |
| `MYSQL_PORT` | `3306` | RAGFlow |
| `MYSQL_DBNAME` | `ragflow_026` | RAGFlow |
| `MYSQL_USER` | 已创建的 MySQL 用户 | RAGFlow |
| `MYSQL_PASSWORD` | 密钥 | RAGFlow |
| `REDIS_HOST` | `redis-redis` | RAGFlow |
| `REDIS_PASSWORD` | 密钥 | RAGFlow |
| `MINIO_HOST` | `minio-minio` | RAGFlow |
| `MINIO_USER` | MinIO 访问用户 | RAGFlow |
| `MINIO_PASSWORD` | 密钥 | RAGFlow |
| `ES_HOST` | `es01` | RAGFlow |
| `ES_USER` | `elastic` | RAGFlow |
| `ELASTIC_PASSWORD` | Elasticsearch 与 RAGFlow 共用的密钥 | Elasticsearch、RAGFlow |
| `MINERU_APISERVER` | `http://mineru:8000` | RAGFlow MinerU provider |
| `MINERU_BACKEND` | `pipeline` | RAGFlow MinerU provider |
| `MINERU_API_MAX_CONCURRENT_REQUESTS` | `1` | MinerU |
| `MINERU_DEVICE_MODE` | `cuda` | MinerU |
| `MINERU_MODEL_SOURCE` | `local` | MinerU |

`ES_HOST`、`ES_USER` 是 RAGFlow 客户端的配置名称；`ELASTIC_PASSWORD` 是 Elasticsearch 服务端凭据，同时也是 RAGFlow 的连接密码。

RAGFlow Component 必须声明 `MINERU_APISERVER=${MINERU_APISERVER}` 与 `MINERU_BACKEND=${MINERU_BACKEND}`。MinerU Component 必须为三个 `MINERU_API_*` 运行时键声明对应占位符。

## 部署步骤

1. 确认 MySQL、MinIO、Redis 正在运行且健康，并确认 `traefik` 网络存在。
2. 确认 BGE-M3 缓存完整、NVIDIA Docker 运行时可用且 GPU 容量充足。
3. 核查 Version 的 Component 集合、依赖、挂载、镜像和占位符是否与本手册一致。
4. 在 Orbit 预览 RAGFlow Service，确认外置地址与 `MINERU_APISERVER=http://mineru:8000` 已渲染为正确值。
5. 通过 Orbit 部署该 Service。
6. 等待 `es01`、`tei`、`mineru`、`ragflow` 全部健康；从 RAGFlow 容器确认 `http://mineru:8000/openapi.json` 可访问。
7. 确认 RAGFlow 网关路由仍指向 `ragflow` Component 的 80 端口。

修改 Service 环境变量或 Version 后，必须重新部署才会生效。

## 数据迁移与备份

MySQL、MinIO、Redis 保存 RAGFlow 的业务状态，迁移前必须取得可验证的备份。

- MySQL：使用逻辑 dump 或经验证的冷数据目录复制；先还原到独立 MySQL Service，再将 RAGFlow 指向新实例。
- MinIO：复制对象目录前停止源 Service；启动目标前后核对文件数与总字节数。
- Redis：复制 `dump.rdb` 前停止源 Service，并核对目标文件。
- Elasticsearch：保留其 RAGFlow 所属数据目录，不能挂载到其他 Service。

同一数据目录不能同时由两个 MySQL、MinIO 或 Redis 实例使用。新部署经过正常业务观察前，保留原目录作为回滚依据。

## MinerU 与回滚

MinerU 现由 RAGFlow Service 管理。旧 `mineru-api-default` Service 已停止，仅保留为回滚配置。除非有明确测试目的和足够 GPU 容量，不要与 RAGFlow 内的 `mineru` 同时启动。

RAGFlow Version 失败时：

1. 通过 Orbit 停止或替换失败部署，不删除卷。
2. 将 RAGFlow Service 绑定到最近一次已验证的 Version。
3. 恢复匹配的 Service 环境变量，尤其是外置 Service 地址和 MinerU 配置。
4. 重新部署，并验证全部 Component 健康及 RAGFlow 到 MinerU 的连通性。

不要通过把旧数据目录挂到另一个运行中的 Service 来回滚。

## 运行检查

以 Orbit 的部署状态、Component 健康状态与作用域容器日志为准。

- RAGFlow 自身和全部依赖 Component 均健康，才视为服务可用。
- MinerU 必须在 RAGFlow 网络内的 8000 端口暴露 `/health` 与 `/openapi.json`。
- MinerU 报 `exec: --: invalid option` 时，检查 Service overlay 是否丢失 `entrypoint=mineru-api`。
- RAGFlow 无法使用 MinerU 时，优先检查 Component 名称 `mineru` 与 `MINERU_APISERVER` 是否一致。
- 健康检查失败、不断重启、缺少 GPU runtime 或跨 Service 别名无法解析，都应阻断继续部署。
