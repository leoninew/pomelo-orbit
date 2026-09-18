# 数据传输 Python 运行时与共享配置
最后修改时间: 2026-09-15 22:37:51

Review status: Accepted

Mode: standard

## Background

`scripts/database_transfer.py` 目前通过 `subprocess` 调用外部 `dbtalk` 二进制完成 SQLite/MySQL/PostgreSQL 的 JSONL 导入导出和目标库查询。该边界隐藏了 dbtalk 的具体诊断，并让 Orbit 需要维护 CLI 参数、临时文件排序与本机二进制安装的额外契约。

`pomelo-dbtalk` 已发布到 PyPI，当前版本为 `0.28.0`，Python import 名为 `dbtalk`。它提供 `dbtalk.database.transfer` 的 `TransferConnection`、`ExportOptions`、`ImportOptions`、`export_database()` 和 `import_database()`，以及 `create_client()` 等同步查询 API。该包要求 Python `>=3.12`。

Orbit 的 Go 服务已经有明确的配置模型：`POMELO_ORBIT_APP__ENV` 仅从进程环境选择环境；配置优先级为 `configs/config.yaml < configs/config.<env>.yaml < .env[.<env>] < OS env`；嵌套覆盖使用 `POMELO_ORBIT_` 前缀和 `__` 分隔符。Python 传输工具不能另设 `DBTALK_*` DSN 或复制一份数据库和 JWT 配置。

本任务参考 `D:/SourceCodes/mywork/k12-ai-publishing-os` 的 Dynaconf 模式：由进程环境先选择唯一 dotenv profile，Dynaconf 加载基础配置，再显式应用 dotenv 和 OS 环境覆盖。日志按 `python-logging-standard` 的简单 CLI 场景处理。

## Goal

1. 将 `scripts/` 收敛为 Python `>=3.12` 的 uv 运行时项目，以 `src/` layout 和 Click 根命令组织可扩展 CLI；声明 PyPI 依赖 `pomelo-dbtalk` 与 Dynaconf，并更新 `uv.lock`。
2. 删除 `database_transfer.py` 对 `dbtalk` 可执行文件、`subprocess`、`shutil` 和 `--dbtalk-command` 的依赖；导出、导入和目标库查询全部调用 `pomelo-dbtalk` Python API。
3. 新增 Python 配置加载边界，用 Dynaconf 读取与 Go 相同的 `configs/config.yaml`、可选 `configs/config.<env>.yaml`、`.env` 或 `.env.<env>`，并以相同的 `POMELO_ORBIT_*`/`__` 变量覆盖规则解析、结构化映射和启动时校验数据库、JWT 与日志配置。
4. Python 使用 Go 业务配置中的 `database.driver`、SQLite path、MySQL DSN、PostgreSQL DSN 和 `jwt.secret_key`；传给 `pomelo-dbtalk` 前在 Python 边界规范化为其所需 canonical DSN，不在 dotenv 中另存 `DBTALK_*` 镜像配置。
5. 保持服务闭包与 Environment（含 SSH 私钥跨实例重加密）的既有业务边界、输入校验和数据脱敏要求。
6. 为普通人类 CLI 诊断配置一次简单文本 `logging.basicConfig`。运行日志写 stderr，stdout 只输出稳定的成功摘要；日志不得包含 DSN userinfo、密码、Fernet key、私钥或 JSONL 内容。

## Non-goal

- 不改写 Go 服务的 Viper/godotenv 配置加载器，也不把 Go 业务迁移到 Python。
- 不改变 Orbit 服务传输闭包、Environment 导入冲突规则、Gateway 解绑或私钥重加密语义。
- 不为 Python 新建另一份 YAML、dotenv 命名空间、`DBTALK_*` DSN 或密钥配置。
- 不引入 JSON Lines 日志、文件轮转、日志 YAML、常驻 worker 或 HTTP 服务。
- 不支持遗留的 `--dbtalk-command` 兼容参数或外部 dbtalk 二进制 fallback。

## User Scenarios

1. 开发者设置 `POMELO_ORBIT_APP__ENV=development` 后执行传输工具；Python 与 Go 都读取 `configs/config.yaml`、可选 `configs/config.development.yaml` 和 `.env.development`，OS 环境仍拥有最高优先级。
2. 开发者的 Go 配置使用 `database.driver=postgres` 与现有 PostgreSQL URL；Python 解析同一配置，并仅在调用 `pomelo-dbtalk` 前转换为 canonical `postgresql+psycopg://` DSN。
3. 环境备份恢复在没有 `dbtalk` 二进制位于 `PATH` 时成功；脚本直接使用锁定的 PyPI Python 包完成预检和数据写入。
4. 导入失败时，CLI 输出安全的失败摘要和 stderr 诊断，不输出数据库凭据、Fernet key、SSH 私钥或传输文件内容。

## Acceptance

- [ ] `scripts/pyproject.toml` 的 Python 约束为 `>=3.12`，锁定 Click、`pomelo-dbtalk` 与 Dynaconf；`uv --directory scripts run pomelo-orbit-cli` 不依赖本机或 PATH 中的 dbtalk 二进制。
- [ ] Python 配置加载器与 Go 使用同一环境选择器、文件命名、优先级、`POMELO_ORBIT_` 前缀和 `__` 嵌套映射；缺失可选 profile 文件的行为也一致。
- [ ] Python 配置只读取 Go 已有的数据库/JWT 字段；`.env`、`.env.<env>` 与示例文档中没有独立的 `DBTALK_*` DSN 镜像值。
- [ ] 所有服务与环境传输数据库操作调用 `pomelo-dbtalk` 的 Python API；代码和帮助文本不再提及 `--dbtalk-command`、dbtalk 二进制安装或 PATH 前置条件。
- [ ] SQLite、MySQL、PostgreSQL 的已支持配置均能映射为 `pomelo-dbtalk` 所需连接；driver 与 DSN dialect 不一致时在写入前失败。
- [ ] 环境导入继续以目标实例 `jwt.secret_key` 重新加密 SSH 私钥，拒绝已有 Environment 的目标 Project，并且不暴露私钥或密钥材料。
- [ ] CLI 日志只在入口初始化一次，采用参数化文本消息；stdout 不混入运行日志，stderr 和错误消息不含敏感值。
- [ ] 测试覆盖配置优先级、DSN 规范化、Python API 调用与错误映射，以及 SQLite 和真实 PostgreSQL 的服务/SSH Environment 导入主流程；不 mock 掉 `pomelo-dbtalk` 的实际导入行为。

## Open Questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard；本轮只记录 Intent，不进入 Plan 或 Implementation。
- Python 配置遵循参考项目“进程环境选择 profile -> Dynaconf 基础配置 -> dotenv 覆盖 -> OS 环境覆盖”的显式加载方式，但环境选择器与业务键名保持 Orbit 的现行 Go 契约。
- 使用 PyPI 发行名 `pomelo-dbtalk>=0.28.0,<0.29`，Python 导入路径为 `dbtalk`；不依赖本地 `D:/SourceCodes/mywork/pomelo-dbtalk` 源码或 CLI 二进制。
- 使用简单常规文本日志：入口一次性配置，运行诊断走 stderr，完成摘要走 stdout；不引入复杂日志配置。
- 采用 `python-click-cli` 模板的 `src/` 包布局和 `[project.scripts]` 入口。命令通过 `uv --directory scripts run pomelo-orbit-cli` 运行，只安装在 scripts 项目环境内，不要求全局安装。
- 首批迁移 `database-transfer` 命令及其 `export`、`import`、`export-environment`、`import-environment` 子命令；其他现有独立脚本留待后续按命令逐项迁移。旧 `scripts/database_transfer.py` 不保留兼容入口。
- Python CLI 只通过 `load_settings()` 加载配置；根 Click 命令在任何子命令执行前校验共享数据库连接、Fernet key 和日志等级，业务命令不延迟校验或直接读取环境变量。

## Risk

- `pomelo-dbtalk` 要求 Python 3.12，而现有 scripts 项目声明 3.10；运行环境和 CI 必须同步升级。
- Go 接受的 MySQL/PostgreSQL DSN 形式与 `pomelo-dbtalk` 的 canonical SQLAlchemy DSN 不完全相同，规范化规则必须集中并对三种 driver 测试，不能把转换结果回写配置文件。
- 直接 API 调用会暴露包版本和 Python API 的兼容性，须通过明确版本范围、锁文件和真实数据库集成测试管理，而不是回退到二进制调用。
- 环境 JSONL 含临时明文 SSH 私钥；日志、异常和测试夹具必须继续避免输出其内容。

## User Review Notes

- 用户要求使用 `specflow` 标准流程记录该任务。
- 用户确认 PyPI 依赖名为 `pomelo-dbtalk`。
- 用户要求传输工具不再调用二进制，而是 import PyPI 包，并作为 uv 项目运行。
- 用户要求引入 Dynaconf 与 `.env.<env>` 配置系统，和 Go 业务共用配置。
- 用户指定参考 `D:/SourceCodes/mywork/k12-ai-publishing-os` 的配置实现。
- 用户要求 Python 日志基于 `python-logging-standard` 保持简单。
- 用户要求参考 `D:/SourceCodes/mywork/best-practices/templates/python-click-cli` 引入 CLI 目录结构，并将 `database_transfer.py` 作为首批命令迁入。
- 用户指定 `pomelo-dbtalk` 使用 `0.28.0` minor 版本线。
- 用户要求将配置结构化映射至 dataclass，并在 CLI 初始化时集中校验。
