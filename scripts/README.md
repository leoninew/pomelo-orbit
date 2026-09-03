# 脚本工具

本页只索引面向开发者直接调用的工具；测试脚本和发布包脚本不单独列出。具体用法以脚本帮助、配套文档或 `skills/` 中的专项技能为准，与代码冲突时以代码为准。

`scripts/pyproject.toml` 仅管理脚本依赖和开发工具，不将本目录构建或安装为 Python 包。使用 uv 执行检查：

```bash
uv --directory scripts run ruff format --check .
uv --directory scripts run ruff check .
uv --directory scripts run mypy
uv --directory scripts run pytest
```