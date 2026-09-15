# 数据传输 Python 运行时与共享配置计划
最后修改时间: 2026-09-15 22:37:51

Review status: Accepted

Mode: standard

## Intent basis

本计划实施已接受的 `docs/intent/20260915-python-data-transfer-runtime.md`。Python 传输工具从独立脚本收敛为 `scripts` uv 项目内的标准 Click CLI，首批命令为 `database-transfer`。它继续拥有 Orbit 服务闭包和 Environment 跨实例重加密的领域规则，但所有数据库 I/O 改由 PyPI `pomelo-dbtalk` API 完成，并读取 Go 应用的同一份配置。

## Target layout

```text
scripts/
├── pyproject.toml
├── uv.lock
├── src/pomelo_orbit_cli/
│   ├── __init__.py
│   ├── __main__.py
│   ├── cli.py
│   ├── context.py
│   ├── logging_config.py
│   ├── settings.py
│   └── commands/
│       ├── __init__.py
│       └── database_transfer.py
└── tests/
    ├── test_database_transfer.py
    └── test_manage.py
```

`pomelo-orbit-cli` 是 `[project.scripts]` 提供的项目环境入口。它不要求全局安装，调用形态为：

```bash
uv --directory scripts run pomelo-orbit-cli database-transfer export ...
```

旧 `scripts/database_transfer.py` 和 `scripts/test_database_transfer.py` 会迁入上述包结构，不保留直接执行或兼容 wrapper。其余独立脚本不在本轮迁移范围内。

## Implementation steps

1. 将 `scripts` 转为可安装的 Python `>=3.12` src-layout 项目：增加 `uv_build`、Click、Dynaconf、`python-dotenv` 和 `pomelo-dbtalk>=0.28.0,<0.29`，声明 `pomelo-orbit-cli = "pomelo_orbit_cli.cli:main"`，并重建 `uv.lock`。调整 Ruff、Mypy 和 Pytest 的扫描路径到 `src/` 与 `tests/`。
2. 创建 Click 根命令、`-h/--help`、`--version` 和 `-v/--verbose`，通过冻结的 `AppContext` 在整个调用树中共享已加载的配置与日志级别。根命令只负责生命周期初始化和子命令注册；`database_transfer.py` 在 `commands/` 中注册 `database-transfer` group 及当前四个业务子命令。
3. 创建集中式配置加载器。它先从进程环境读取 `POMELO_ORBIT_APP__ENV`，定位仓库根目录，按 `configs/config.yaml`、可选 `configs/config.<env>.yaml`、可选 `.env` 或 `.env.<env>`、OS 环境的顺序加载；只接受白名单内的 `POMELO_ORBIT_` 前缀及 `__` 分隔覆盖。配置对象以 frozen dataclass 建模数据库驱动、规范化连接、Fernet 与日志等级；根 Click 命令初始化时集中校验全部字段，业务命令不再延迟校验。
4. 在配置边界集中规范化连接。SQLite 相对路径相对于 Orbit 仓库根目录解析为 dbtalk 使用的 SQLite DSN；MySQL 和 PostgreSQL 仅接受与 Go driver 一致的 DSN，并在调用 API 前转换为 `pomelo-dbtalk` 所需 canonical SQLAlchemy DSN。校验失败只陈述字段和原因，不回显 DSN。
5. 将现有 JSONL 解析、闭包选择、输入校验、Environment 目标重写和 SSH 私钥重加密迁入 `commands/database_transfer.py`；将外部 `dbtalk` 命令构造、`subprocess`、`shutil`、`--source`、`--target`、`--dsn`、`--dsn-env`、`--dbtalk-command` 和临时导入文件排序职责替换为 `pomelo-dbtalk` 的 transfer/query API。命令改为使用当前配置选择源/目标数据库，并保持安全的 stdout 成功摘要。
6. 新增简单的文本日志配置：入口只调用一次 `logging.basicConfig`，运行诊断写入 stderr，命令结果只写 stdout。为 dbtalk API 异常建立安全映射，不输出 DSN userinfo、密码、Fernet key、SSH 私钥、密文或 JSONL 内容。
7. 将数据传输测试迁入 `scripts/tests/`：使用 `CliRunner` 测试命令树、选项帮助、成功摘要和错误映射；为配置优先级、可选 profile、环境覆盖与 DSN 规范化建立单元测试；以 API 调用替换二进制命令 mock。补充 SQLite 和有显式 PostgreSQL e2e 配置时运行的服务与 SSH Environment 传输主流程。更新服务迁移指南和 `transfer-database-data` skill 为新的 CLI 入口。

## Files to change

- `scripts/pyproject.toml`、`scripts/uv.lock`
- `scripts/src/pomelo_orbit_cli/**`
- `scripts/tests/**`
- 删除 `scripts/database_transfer.py`、`scripts/test_database_transfer.py`
- `scripts/README.md`
- `docs/guides/service-data-transfer.md`
- `skills/transfer-database-data/SKILL.md`

## Verification plan

1. 运行 `uv --directory scripts lock --check`、`uv --directory scripts run ruff format --check .`、`uv --directory scripts run ruff check .`、`uv --directory scripts run mypy` 和 `uv --directory scripts run pytest`。
2. 运行 `uv --directory scripts run pomelo-orbit-cli --help` 与 `uv --directory scripts run pomelo-orbit-cli database-transfer --help`，确认 Click 命令树、帮助和入口安装在项目环境内。
3. 以临时 Orbit 根目录分别验证 base YAML、profile YAML、dotenv 和 OS 环境的优先级，以及 SQLite、MySQL、PostgreSQL DSN 的安全规范化。
4. 运行 SQLite 传输主流程；在 `BACKEND_GO_POSTGRES_E2E_CONFIG` 或 `BACKEND_GO_POSTGRES_E2E_DSN` 已配置时，运行 PostgreSQL 导入主流程。确认未配置时测试明确 skip，不伪造成功。
5. 对照数据传输指南和 skill 验证所有示例不再提及 `database_transfer.py`、`--dbtalk-command` 或需要 `dbtalk` 在 PATH。

## Assumptions

- 采用 `pomelo-dbtalk>=0.28.0,<0.29`，由提交的锁文件固定实际版本。
- Python 配置在根 Click 命令初始化时验证数据库连接、JWT Fernet key 和日志等级；所有首批命令共享同一份已验证 Settings。
- 继续沿用 Go PostgreSQL e2e 的 `BACKEND_GO_POSTGRES_E2E_CONFIG` / `BACKEND_GO_POSTGRES_E2E_DSN` 约定，避免为 Python 新建平行测试连接变量。

## Risks

- `pomelo-dbtalk` 的已发布 API 可能与 Intent 记录的符号不同，实施前需从锁定包的实际接口确认并据此调整调用层，不回退到二进制模式。
- 将当前命令的显式 DSN 参数移除后，文档与 automation 必须先具备有效的共享 Orbit 配置；这是刻意收敛，不提供新旧调用并存。
- 真实 PostgreSQL 流程依赖外部实例；没有显式 e2e 配置时只能 skip 并在 Verification 中记录。

## Rollback

本轮处于活跃开发期，不增加运行时兼容 wrapper。若发布前验证失败，回退整个迁移提交；不要重新引入 `database_transfer.py` 与新 CLI 的双入口。

## User review notes

- 用户明确要求按 `python-click-cli` 模板引入 CLI 目录，`database_transfer.py` 是首批迁入的命令。
- 用户已要求进入 Plan，本计划仅定义实现与验证边界，不开始产品代码修改。
