# 配置加载最佳实践

## 优先级链

配置按以下顺序加载，后面的覆盖前面的：

```
config.defaults.yaml  →  config.yaml  →  .env  →  环境变量
```

| 层级 | 文件/来源 | 用途 |
|------|-----------|------|
| 1 | `backend/config.defaults.yaml` | 所有配置项的默认值，提交到版本库 |
| 2 | `backend/config.yaml` | 本地覆盖，不提交（.gitignore） |
| 3 | `backend/.env` | 敏感配置（密钥等），不提交 |
| 4 | 环境变量 | 生产/CI 注入，优先级最高 |

## 环境变量命名规则

前缀 `POMELO_ORBIT_`，嵌套 key 用双下划线 `__` 分隔：

```bash
# 对应 jwt.secret_key
POMELO_ORBIT_JWT__SECRET_KEY=xxx

# 对应 app.debug
POMELO_ORBIT_APP__DEBUG=true

# 对应 database.sqlite.path
POMELO_ORBIT_DATABASE__SQLITE__PATH=/data/db/app.db
```

## 新增配置项的步骤

1. 在 `config.defaults.yaml` 里加默认值（即使是空字符串）
2. 在 `backend/.env.example` 里加对应的环境变量示例
3. 敏感项（密钥、密码）只通过 `.env` 或环境变量提供，**不放进 yaml**

## 已知问题

**dynaconf 3.2.x `dotenv_encoding` 不生效**

dynaconf 内部的 `start_dotenv` 调用底层 `load_dotenv` 时未传 `encoding` 参数，
导致在 Windows 非 UTF-8 系统上读取 `.env` 时报 `UnicodeDecodeError`。

规避方式：启动时设置 `PYTHONUTF8=1`，Makefile 和 `pytest.ini` 里已配置：

```makefile
# Makefile
dev-backend:
    cd backend && PYTHONUTF8=1 uv run python -m pomelo_orbit.main --port 9001 --reload
```

```ini
# pytest.ini
[pytest]
env =
    PYTHONUTF8=1  # 若未配置，在 Windows 下运行测试会因 .env 编码问题失败
```

待 dynaconf 修复后可移除该环境变量。

## 测试中的注意事项

**不要用 `get_settings()` 单例测试配置加载**，`@lru_cache` 会让第一次调用的结果永久缓存，
后续测试无法覆盖。应直接构造 `Dynaconf` 实例：

```python
from dynaconf import Dynaconf

def make_settings(base_dir: Path) -> Dynaconf:
    dotenv = base_dir / ".env"
    s = Dynaconf(
        settings_files=[str(base_dir / "config.defaults.yaml")],
        load_dotenv=True,
        # 明确指定路径（即使文件不存在），阻止 dynaconf 向上搜索找到真实的 .env
        dotenv_path=str(dotenv) if dotenv.exists() else str(base_dir / ".env"),
        dotenv_override=False,
        envvar_prefix="POMELO_ORBIT",
        merge_enabled=True,
        encoding="utf-8",
    )
    s.as_dict()  # 强制触发加载，避免懒加载在环境变量还原后才读取
    return s
```

**`pytest.ini` 注入的环境变量会污染所有测试**，测试配置加载时避免使用
`pytest.ini` 里已注入的 key（如 `POMELO_ORBIT_JWT__SECRET_KEY`），
否则 yaml / `.env` 里的值会被覆盖，断言结果不符合预期。
