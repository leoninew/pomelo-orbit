# Orbit 服务级 JSONL 导入导出适配计划
最后修改时间: 2026-08-19 18:21:46

Review status: Accepted

## Basis

本计划实施已接受的 `docs/requirement/20260819-jsonl-database-transfer.md`。Housekeeper 拥有全库 SQLite/MySQL JSONL 导出、导入、类型转换、时区、主键冲突策略、事务和通用 skill；Orbit 仅拥有服务部署闭包的领域选择、领域校验和服务级 skill。

跨项目边界是已安装的 `housekeeper database` CLI。Orbit 不 import Housekeeper Python 包，不实现 SQLite/MySQL 连接、全库枚举、SQL 方言、通用值类型编码或数据库写入。Housekeeper 计划完成并通过本机 `make release` 是本计划实施的前置条件。

## CLI Integration Contract

Orbit 服务适配脚本使用以下 Housekeeper 调用：

```text
housekeeper database export --source sqlite|mysql --output <temporary-full.jsonl> [connection] --exclude-table schema_migrations [--tz <IANA timezone>]
housekeeper database import --target sqlite|mysql --input <service.jsonl> --mode insert|upsert [connection] [--tz <IANA timezone>]
```

- 服务适配脚本暴露并原样转发 Housekeeper 的数据库连接参数与 `--tz`，不读取或复制 Housekeeper 配置。
- 服务导出先让 Housekeeper 写临时全库 JSONL，并固定排除 `schema_migrations`，再只按 Orbit 领域关系裁剪最终服务 JSONL；这是以 CLI 为边界且不让 Orbit 连接数据库的实现方式。临时文件必须在成功或失败后清理，且被视为敏感数据。
- 服务导入先在 Orbit 内验证 JSONL 的服务闭包，再调用 Housekeeper import。`--mode insert|upsert` 必须显式指定；目标经正常迁移初始化后已有种子 Project，标准服务迁移使用 `upsert`。具体主键与事务行为完全遵从 Housekeeper 契约。服务文件不包含 `schema_migrations`，因此导入不能传递该排除项。
- 适配脚本通过 `subprocess.run` 传递参数，不使用 shell 拼接；捕获非零退出码并以不含密码的上下文失败。标准输出只报告最终文件、服务 code、表数和行数。

## Implementation Steps

1. 确认并固定对 Housekeeper JSONL v1 的最小消费面。
   - 在实现开始前验证已安装 `housekeeper database --help`，并确认其 JSONL v1 的 `header`、`table`、`row`、`end`、列顺序和主键字段。
   - Orbit 只解析这些公开 JSONL 记录以完成服务领域筛选和验证；不复制 JSON scalar/tag 的编码或 MySQL/SQLite temporal 转换。
   - 在脚本启动时检查 `housekeeper` 可执行文件；找不到 CLI 或版本/格式不兼容时 fail fast，提示先按 Housekeeper 发布流程安装。

2. 将现有服务闭包领域逻辑抽成 JSONL 选择与验证模块。
   - 保留 `SERVICE_TABLES`、`VERSION_COMPONENT_CHILD_TABLES`、`SERVICE_COMPONENT_CHILD_TABLES` 和既有闭包规则：按 service code 选出其 project、application、完整 version lineage、version component 与所有 child tables、gateway config、service env/component overrides 和 route。
   - 用表块的列名将 `row.values` 临时视为行映射来推导 ID 集合；写回时保留 Housekeeper 表块 header 和原始 values/tag，不改写通用值。
   - 输出表块严格按 `SERVICE_TABLES` 顺序，每块精确更新 `end.rows`；拒绝缺失所需表、非唯一 service code、缺失 application/project/version、跨 application 版本谱系、循环谱系或缺失引用。
   - 服务导入对同一闭包约束进行全文件验证：表集合与顺序准确、service 表恰好一行、所有引用可闭合；非服务文件或混入其他服务的数据在调用 Housekeeper 前失败。

3. 替换旧的 Orbit 全库传输脚本与测试边界。
   - 从 `scripts/database_transfer.py` 移除全库 `export`、`import`、`convert`、SQL render、Base64 archive、SQLite/MySQL connection、值编码、时区和通用导入代码。
   - 将服务闭包适配保留在通用数据操作入口 `scripts/database_transfer.py`，为后续其他业务数据操作保留扩展空间；不保留旧 SQL 文件格式或 compatibility wrapper。
   - 以新的服务 JSONL 行为替换 `scripts/test_database_transfer.py`，测试只保留服务选择、JSONL 闭包验证、Housekeeper CLI 参数构造、子进程错误、临时文件清理和不输出敏感数据；通用 SQLite/MySQL/类型/SQL 测试迁移或归属 Housekeeper。
   - 当前未提交的 `scripts/database_transfer.py` 和 `scripts/test_database_transfer.py` 时间格式修复不单独延续为 Orbit 功能；其通用问题由 Housekeeper 实现和测试承接，避免两套实现分叉。

4. 迁移 Orbit 文档和 skill。
   - 收敛现有 `skills/transfer-database-data/`，移除它对完整 SQLite/MySQL 导入导出和 SQL 方言转换的错误归属。
   - 保持 skill 名称为 `transfer-database-data`，仅描述 service code 的闭包范围、目标 Orbit schema 需先运行迁移、Housekeeper CLI 前置条件、插入/覆盖模式和导入后的 Orbit 领域验证；全库迁移转向 Housekeeper `database-transfer` skill。
   - 新增或更新 Orbit `docs/guides/` 的服务数据迁移指南，更新 `docs/INDEX.md`；只链接/引用 Housekeeper 格式和通用命令，不复制 JSONL/时区/事务完整规范。

5. 对服务域和 CLI 集成完成定向验证。
   - 使用临时 JSONL fixtures 覆盖单个服务的完整闭包、排除其他 service、版本 lineage 顺序、全部 component child tables、gateway config、route 与 `end.rows`。
   - 覆盖服务输入的错误表集、多个 service、跨 application lineage、缺失引用、循环 lineage，断言 Housekeeper CLI 不会被调用。
   - mock `subprocess.run` 覆盖 export/import 的完整参数转发、导出排除 `schema_migrations`、必填 mode、`insert`/`upsert`、`--tz`、非零退出和临时全库文件清理；断言密码不出现在报告文本。
   - 运行 `python -m ruff format --check`、`python -m ruff check`、相关 `unittest`，并执行项目要求的 `task check` 与 `go test ./cmd/... ./internal/...`；不因本任务启动开发服务器。

## Files To Change

- 收敛 `scripts/database_transfer.py` 与 `scripts/test_database_transfer.py` 为服务级适配入口及其测试，并为后续业务数据操作保留脚本命名空间。
- 更新 `skills/transfer-database-data/` 的服务级边界及 `agents/openai.yaml`。
- 新增或更新 `docs/guides/service-data-transfer.md`、`docs/INDEX.md`、README 中需要的运维入口。
- `docs/requirement/20260819-jsonl-database-transfer.md`、本 Plan，后续 Verification 文档。

## Verification Plan

1. Housekeeper 发布后，从 Orbit 工作目录运行 `housekeeper database --help`，验证脚本不会依赖本地源码 import。
2. 以含目标和无关数据的 JSONL fixture 导出一个服务闭包，断言表集合、每表行集、版本顺序、行数和无关 service 排除。
3. 对服务 JSONL 注入每种领域错误，断言适配在调用 CLI 前失败，且不留下最终输出或临时全库文件。
4. mock CLI 集成，断言 SQLite/MySQL 连接参数、导出 `schema_migrations` 排除项、必填 `--mode` 与 `--tz` 原样传递；断言失败信息不泄露密码。
5. 运行 Python 定向检查以及 `task check`、`go test ./cmd/... ./internal/...`，检查 Orbit 文档/skill 不再声称拥有全库数据传输。

## Blockers

- 无。Housekeeper 已交付并发布 JSONL export/import CLI；Orbit 以已安装的
  `housekeeper` 可执行文件调用该能力。

## Assumptions

- 服务闭包数据量可接受“先全库导出至临时 JSONL，再裁剪”的实现方式；不在此任务为服务选择发明第二种数据库查询接口。
- 目标 Orbit schema 已由 `go run ./cmd/migrate` 或等价的应用迁移初始化；服务导入不携带 DDL。
- Housekeeper JSONL 格式版本只在 header 中声明一次，且后续兼容性变更以明确版本升级处理，不提供隐式旧 SQL 格式兼容。

## Risks

1. 临时全库 JSONL 会短暂包含无关敏感数据；适配脚本必须受控创建、始终清理且不得打印内容。
2. CLI 是跨项目边界，Housekeeper 未发布、PATH 解析到旧版本或格式不兼容会阻断服务迁移；应 fail fast 并提供安装诊断。
3. 表级事务允许导入局部成功，Orbit 服务校验只能保证输入闭包完整，不能把 Housekeeper 的通用事务契约改成服务级原子性。
4. 移除旧脚本是硬切换，已有旧 `.sql` transfer 文件不再可由 Orbit 导入；符合当前无兼容层约束，文档必须提前说明。

## Rollback

本任务不修改已执行的数据库迁移。代码与 skill 可通过回退到上一 Orbit 发布恢复；已由 Housekeeper 导入的目标数据只能由操作者从备份恢复或使用显式的后续迁移操作处理，适配脚本不自动删除数据。

## User Review Notes

- 用户确认采用 CLI 作为 Orbit 与 Housekeeper 的接口边界。
