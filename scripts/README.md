# 脚本工具

本目录包含本地开发、远程运维、发布支持与一次性迁移脚本。每个 Python 脚本都必须有同名 Markdown 参考文档；文档描述用途、前置条件、支持的用法和安全边界。与文档冲突时以脚本代码为准。

## 本地开发工具

| 脚本 | 用途 |
| --- | --- |
| [`gen_ulid.py`](./gen_ulid.md) | 生成一个或多个 ULID。 |
| [`run.py`](./run.md) | 同时启动后端和前端开发服务。 |
| [`cert.py`](./cert.md) | 生成或检查本地开发证书。 |
| [`prepare_ragflow_tei.py`](./prepare_ragflow_tei.md) | 准备和验证 RAGFlow TEI 模型缓存与镜像。 |

## 远程运维

| 脚本 | 用途 |
| --- | --- |
| [`manage.py`](./manage.md) | 远程部署、SSH、Compose、文件复制与备份入口。 |
| [`clean_remote_docker.py`](./clean_remote_docker.md) | 检查或清理远程 Docker 可回收空间。 |
| [`reconcile_compose_proxies.py`](./reconcile_compose_proxies.md) | 协调手工 CD Compose 项目的代理配置。 |

## 一次性迁移与生成工具

| 脚本 | 用途 |
| --- | --- |
| [`fix_frontend_import_format.py`](./fix_frontend_import_format.md) | 修正指定前端 Proto 导入格式。 |
| [`gen_sqlc_domain_repos.py`](./gen_sqlc_domain_repos.md) | 生成部分 SQLC 领域仓储实现。 |
| [`move_platform_api_domains.py`](./move_platform_api_domains.md) | 迁移前端 API 模块到领域目录。 |
| [`rewrite_frontend_proto_imports.py`](./rewrite_frontend_proto_imports.md) | 迁移前端 Proto 导入到领域路径。 |
| [`rewrite_proto_domain_imports.py`](./rewrite_proto_domain_imports.md) | 迁移后端 Proto 导入到领域包。 |

## 测试与发布支持

| 脚本 | 用途 |
| --- | --- |
| [`test_reconcile_compose_proxies.py`](./test_reconcile_compose_proxies.md) | Compose 代理协调单元测试。 |
| [`test_prepare_ragflow_tei.py`](./test_prepare_ragflow_tei.md) | RAGFlow TEI 缓存 staging 与 `tar.gz` 归档单元测试。 |
| [`test_export_ragflow_tei_baseline.py`](./test_export_ragflow_tei_baseline.md) | RAGFlow TEI SQLite 基线导出/导入单元测试。 |
| [`version-calc.py`](./version-calc.md) | 从 Git 历史计算并按需写入版本。 |
| [`export_ragflow_tei_baseline.py`](./export_ragflow_tei_baseline.md) | 导出 RAGFlow TEI SQLite 控制面候选基线。 |

## 可选 Python 检查

仓库目前未为 `scripts/` 配置强制执行的 Python 检查。若本机已安装相应工具，可手动执行不修改文件的检查：

```bash
python -m mypy scripts/
python -m ruff check scripts/
python -m ruff format --check scripts/
```

维护脚本时，若变更了脚本接口、前置条件或副作用，必须在同一变更中更新对应的同名 Markdown 文档。
