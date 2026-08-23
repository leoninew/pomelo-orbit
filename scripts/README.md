# 脚本工具

本页只索引面向开发者直接调用的工具；测试脚本和发布包脚本不单独列出。具体用法以脚本帮助、配套文档或 `skills/` 中的专项技能为准，与代码冲突时以代码为准。

`scripts/pyproject.toml` 仅管理脚本依赖和开发工具，不将本目录构建或安装为 Python 包。使用 uv 执行检查：

```bash
uv --directory scripts run ruff format --check .
uv --directory scripts run ruff check .
uv --directory scripts run mypy
uv --directory scripts run pytest
```

## 本地开发工具

| 脚本 | 用途 |
| --- | --- |
| [`gen_ulid.py`](./gen_ulid.md) | 生成一个或多个 ULID。 |
| [`cert.py`](./cert.md) | 生成或检查本地开发证书。 |

## 发布支持

| 脚本 | 用途 |
| --- | --- |
| [`version-calc.py`](./version-calc.md) | 从 Git 历史计算并按需写入发布版本。 |
| [`release.py`](./release.py) | 通过 Claude、Codex 和 Grok 原生 CLI 发布本地 plugin；项目无 standalone skill。 |

## 远程运维

| 脚本 | 用途 |
| --- | --- |
| [`manage.py`](./manage.py) | 远程部署、SSH、Compose、文件复制、备份与 Docker 可回收空间清理入口。 |
| [`database_transfer.py`](./database_transfer.py) | 通过 dbtalk 导出或导入一个 Orbit Service 的部署闭包。 |
