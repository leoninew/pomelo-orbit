# Orbit 服务级 JSONL 导入导出适配计划
最后修改时间: 2026-08-22 21:18:41

Review status: Accepted

## Basis

本计划实施已接受的 `docs/requirement/20260819-jsonl-database-transfer.md`。dbtalk 拥有全库 SQLite/MySQL JSONL 导出、导入、类型转换、时区、主键冲突策略、事务和通用 skill；Orbit 仅拥有服务部署闭包的领域选择、领域校验和服务级 skill。

跨项目边界是已安装的 `dbtalk database` CLI。Orbit 不 import dbtalk Python 包，不实现 SQLite/MySQL 连接、全库枚举、SQL 方言、通用值类型编码或数据库写入。dbtalk 已发布并可在本机调用是本计划实施的前置条件。

## CLI Integration Contract

Orbit 服务适配脚本使用以下 dbtalk 调用：

```text
dbtalk database export --source sqlite|mysql --output <temporary-service.jsonl> --dsn <DSN>|--dsn-env <ENV_NAME> --include-table <service-table>... [--tz <IANA timezone>]
dbtalk database import --target sqlite|mysql --input <service.jsonl> --mode insert|upsert --dsn <DSN>|--dsn-env <ENV_NAME> [--tz <IANA timezone>]
```

- 服务适配脚本暴露并原样转发 dbtalk 的 `--dsn` 或 `--dsn-env`、`--tz`，不读取或复制 dbtalk 配置；二者必须且只能提供一个。
- 服务导出让 dbtalk 只写入服务闭包所需表的临时 JSONL，再按 Orbit 领域关系裁剪最终服务 JSONL；这是以 CLI 为边界且不让 Orbit 连接数据库的实现方式。临时文件必须在成功或失败后清理，且被视为敏感数据。
- 服务导入先在 Orbit 内验证 JSONL 的服务闭包，再调用 dbtalk import。`--mode insert|upsert` 必须显式指定；目标经正常迁移初始化后已有种子 Project，标准服务迁移使用 `upsert`。具体主键与事务行为完全遵从 dbtalk 契约。服务文件不包含 `schema_migrations`，因此导入不传表排除项。
- 适配脚本通过 `subprocess.run` 传递参数，不使用 shell 拼接；捕获非零退出码并以不含密码的上下文失败。标准输出只报告最终文件、服务 code、表数和行数。

## Implementation Steps

1. 确认并固定对 dbtalk JSONL v1 的最小消费面。
   - 在实现开始前验证已安装 `dbtalk database --help`，并确认其 JSONL v1 的 `header`、`table`、`row`、`end`、列顺序和主键字段。
   - Orbit 只解析这些公开 JSONL 记录以完成服务领域筛选和验证；不复制 JSON scalar/tag 的编码或 MySQL/SQLite temporal 转换。
   - 在脚本启动时检查 `dbtalk` 可执行文件；找不到 CLI 或版本/格式不兼容时 fail fast，提示先安装 dbtalk。

2. 将现有服务闭包领域逻辑抽成 JSONL 选择与验证模块。
   - 保留 `SERVICE_TABLES`、`VERSION_COMPONENT_CHILD_TABLES`、`SERVICE_COMPONENT_CHILD_TABLES` 和既有闭包规则：按 service code 选出其 project、application、完整 version lineage、version component 与所有 child tables、gateway config、service env/component overrides 和 route。
   - 用表块的列名将 `row.values` 临时视为行映射来推导 ID 集合；写回时保留 dbtalk 表块 header 和原始 values/tag，不改写通用值。
   - 输出表块严格按与 dbtalk 目标外键拓扑一致的 `SERVICE_TABLES` 顺序，每块精确更新 `end.rows`；拒绝缺失所需表、非唯一 service code、缺失 application/project/version、跨 application 版本谱系、循环谱系或缺失引用。
   - 服务导入对同一闭包约束进行全文件验证：表集合与顺序准确、service 表恰好一行、所有引用可闭合；非服务文件或混入其他服务的数据在调用 dbtalk 前失败。

3. 替换旧的 Orbit 全库传输脚本与测试边界。
   - 从 `scripts/database_transfer.py` 移除全库 `export`、`import`、`convert`、SQL render、Base64 archive、SQLite/MySQL connection、值编码、时区和通用导入代码。
   - 将服务闭包适配保留在通用数据操作入口 `scripts/database_transfer.py`，为后续其他业务数据操作保留扩展空间；不保留旧 SQL 文件格式或 compatibility wrapper。
   - 以新的服务 JSONL 行为替换 `scripts/test_database_transfer.py`，测试只保留服务选择、JSONL 闭包验证、dbtalk CLI 参数构造、子进程错误、临时文件清理和不输出敏感数据；通用 SQLite/MySQL/类型/SQL 测试迁移或归属 dbtalk。
   - `scripts/database_transfer.py` 和 `scripts/test_database_transfer.py` 的通用时间格式问题不单独延续为 Orbit 功能；由 dbtalk 实现和测试承接，避免两套实现分叉。

4. 迁移 Orbit 文档和 skill。
   - 收敛现有 `skills/transfer-database-data/`，移除它对完整 SQLite/MySQL 导入导出和 SQL 方言转换的错误归属。
   - 保持 skill 名称为 `transfer-database-data`，仅描述 service code 的闭包范围、目标 Orbit schema 需先运行迁移、dbtalk CLI 前置条件、插入/覆盖模式和导入后的 Orbit 领域验证；全库迁移转向 dbtalk `dbtalk-database` skill。
   - 新增或更新 Orbit `docs/guides/` 的服务数据迁移指南，更新 `docs/INDEX.md`；只链接/引用 dbtalk 格式和通用命令，不复制 JSONL/时区/事务完整规范。

5. 建立脚本工具环境，不将 `scripts/` 打包。
   - 新增 `scripts/pyproject.toml`，将 `cryptography`、`ulid-py` 作为运行时依赖，pytest、Ruff 与 MyPy 作为开发依赖，并设置 `tool.uv.package = false`；Ruff 仅启用基础语法和未定义名称检查。
   - 生成并提交 `scripts/uv.lock`；忽略 `scripts/.venv/`，不添加构建后端、`src/` 布局、Python 包安装或命令行 entry point。
   - pytest 直接收集既有 `unittest.TestCase` 测试；不为更换测试运行器进行无行为价值的测试重写。
   - 为根 `Taskfile.yml` 添加 `scripts:check`，统一运行格式、lint、类型检查与 pytest。

6. 对服务域、CLI 集成和脚本工具环境完成定向验证。
   - 使用临时 JSONL fixtures 覆盖单个服务的完整闭包、排除其他 service、版本 lineage 顺序、全部 component child tables、gateway config、route 与 `end.rows`。
   - 覆盖服务输入的错误表集、多个 service、跨 application lineage、缺失引用、循环 lineage，断言 dbtalk CLI 不会被调用。
   - mock `subprocess.run` 覆盖 export/import 的完整参数转发、导出 include 表、必填 mode、`insert`/`upsert`、`--tz`、非零退出和临时文件清理；断言密码不出现在报告文本。
   - 运行 `uv --directory scripts run ruff format --check .`、`uv --directory scripts run ruff check .`、`uv --directory scripts run mypy` 和 `uv --directory scripts run pytest`，并执行项目要求的 `task check` 与 `go test ./cmd/... ./internal/...`；不因本任务启动开发服务器。

## Files To Change

- 收敛 `scripts/database_transfer.py` 与 `scripts/test_database_transfer.py` 为服务级适配入口及其测试，并为后续业务数据操作保留脚本命名空间。
- 更新 `skills/transfer-database-data/` 的服务级边界及 `agents/openai.yaml`。
- 新增或更新 `docs/guides/service-data-transfer.md`、`docs/INDEX.md`、README 中需要的运维入口。
- 新增 `scripts/pyproject.toml`、`scripts/uv.lock`，更新 `scripts/.gitignore`、`scripts/README.md` 与根 `Taskfile.yml`。
- `docs/requirement/20260819-jsonl-database-transfer.md`、本 Plan，后续 Verification 文档。

## Verification Plan

1. dbtalk 发布后，从 Orbit 工作目录运行 `dbtalk database --help`，验证脚本不会依赖本地源码 import。
2. 以含目标和无关数据的 JSONL fixture 导出一个服务闭包，断言表集合、每表行集、版本顺序、行数和无关 service 排除。
3. 对服务 JSONL 注入每种领域错误，断言适配在调用 CLI 前失败，且不留下最终输出或临时服务 JSONL 文件。
4. mock CLI 集成，断言 SQLite/MySQL 连接参数、导出闭包表 include 项、必填 `--mode` 与 `--tz` 原样传递；断言失败信息不泄露密码。
5. 运行 pytest、Ruff、MyPy、`task check` 与 `go test ./cmd/... ./internal/...`，检查 Orbit 文档/skill 不再声称拥有全库数据传输，且 `scripts/` 不作为 Python 包安装。

## Blockers

- 无。dbtalk 已交付并发布 JSONL export/import CLI；Orbit 以已安装的
  `dbtalk` 可执行文件调用该能力。

## Assumptions

- 服务闭包数据量可接受“先全库导出至临时 JSONL，再裁剪”的实现方式；不在此任务为服务选择发明第二种数据库查询接口。
- 目标 Orbit schema 已由 `go run ./cmd/migrate` 或等价的应用迁移初始化；服务导入不携带 DDL。
- dbtalk JSONL 格式版本只在 header 中声明一次，且后续兼容性变更以明确版本升级处理，不提供隐式旧 SQL 格式兼容。

## Risks

1. 临时全库 JSONL 会短暂包含无关敏感数据；适配脚本必须受控创建、始终清理且不得打印内容。
2. CLI 是跨项目边界，dbtalk 未发布、PATH 解析到旧版本或格式不兼容会阻断服务迁移；应 fail fast 并提供安装诊断。
3. 表级事务允许导入局部成功，Orbit 服务校验只能保证输入闭包完整，不能把 dbtalk 的通用事务契约改成服务级原子性。
4. 移除旧脚本是硬切换，已有旧 `.sql` transfer 文件不再可由 Orbit 导入；符合当前无兼容层约束，文档必须提前说明。

## Rollback

本任务不修改已执行的数据库迁移。代码与 skill 可通过回退到上一 Orbit 发布恢复；已由 dbtalk 导入的目标数据只能由操作者从备份恢复或使用显式的后续迁移操作处理，适配脚本不自动删除数据。

## User Review Notes

- 用户确认采用 CLI 作为 Orbit 与 dbtalk 的接口边界。
- 用户确认保留 `scripts/manage.py backup` 的远程目录备份能力；本计划仅改变服务数据导出导入适配。
- 用户确认脚本依赖由 uv 管理，pytest 为测试入口；避免引入严格 Python 项目结构。
