# Orbit 服务级 JSONL 导入导出适配
最后修改时间: 2026-08-22 21:19:38

Review status: Accepted

Mode: standard

## Background

当前 `scripts/database_transfer.py` 同时包含全库 SQLite/MySQL 数据传输和 Orbit 服务部署闭包选择。前者是可独立复用的数据库能力，后者依赖 Orbit 的 `project`、`application`、`version`、`service`、`route` 等模型，二者不应继续同处于一个项目和脚本。

dbtalk 是全库 JSONL 数据传输、格式文档、导入导出转换和通用 skill 的所有者。Orbit 只维护服务闭包选择、服务级导入导出适配和 Orbit 专用 skill，并基于 dbtalk 的稳定传输契约完成底层数据读写。

## Goal

1. 将 Orbit 的服务部署闭包导出为 dbtalk 定义的 JSONL 表块格式，保留既有项目、应用、版本谱系、组件、Gateway、服务和路由的选择范围及依赖顺序。
2. 从 dbtalk JSONL 导入一个服务部署闭包，并以 Orbit 的业务约束验证其表集、单个服务和版本谱系，不在 Orbit 复制通用 SQLite/MySQL 连接、全库枚举、类型转换或导入策略。
3. 提供 Orbit 专用 skill，指导用户使用服务级导出/导入并调用 dbtalk 的通用数据传输能力；通用全库迁移不在该 skill 中实现或说明。
4. 在 Orbit 文档中说明服务级边界、对 dbtalk 的依赖和目标 schema 初始化前提，避免保留第二份 JSONL 格式规范或通用数据库传输文档。
5. 以 `scripts/pyproject.toml` 和 uv 管理脚本的运行时与开发依赖，并以 pytest 作为脚本测试入口，不将 `scripts/` 演变为可安装 Python 包。

## Non-goal

- 不在 Orbit 维护全库 `export`、`import`、`convert` 或 SQLite/MySQL 方言适配的独立实现。
- 不在 Orbit 定义第二份 JSONL 格式、时区、主键覆盖或表级事务契约；这些都由 dbtalk 的通用能力拥有。
- 不从服务传输文件创建或修改 schema，也不迁移索引、视图、触发器、存储过程或权限。
- 不改变服务部署闭包已有的领域关系和选择范围，除非为适配 dbtalk 格式所必需。
- 不为脚本工具引入 `src/` 布局、构建后端、Python 包安装或命令行 entry point。

## User scenarios

1. 管理员导出 `sub2api-default` 服务；文件仅包含该服务所需的项目、应用、版本谱系、组件、Gateway 配置、服务环境和路由，不包含其他服务的数据。
2. 管理员先通过目标 Orbit 的迁移初始化 schema，再导入服务 JSONL；适配层拒绝不符合 Orbit 服务闭包表集、服务数量或依赖关系的文件。
3. 管理员使用 Orbit service-transfer skill 完成服务迁移；skill 清楚区分服务级操作和 dbtalk 的全库数据传输。

## Acceptance

- [ ] Orbit 服务导出/导入只读写 dbtalk 定义的 JSONL 格式，并不再产生或消费 Orbit 自己的全库 SQL 载体。
- [ ] 服务导出仍包含既有部署闭包表集，且只选择请求的服务及其项目、应用、版本谱系、组件、Gateway 配置、服务环境和路由依赖。
- [ ] 服务导入对表集、单个服务、应用归属、版本谱系和引用完整性执行 Orbit 领域校验；数据写入、冲突策略、主键预检、时区和事务遵循 dbtalk 契约。
- [ ] Orbit 代码不再包含全库表枚举、SQLite/MySQL 连接、方言 SQL 渲染、全库导入/导出或 JSONL 值类型适配实现。
- [ ] Orbit 文档和 service-transfer skill 明确依赖 dbtalk，并仅说明 Orbit 的服务选择规则、调用方式与前置 schema 初始化。
- [ ] `scripts/` 通过不打包的 uv 项目管理 `cryptography`、`ulid-py`、pytest、Ruff 和 MyPy；pytest 能收集并执行现有脚本测试。

## Open questions

已解决：Orbit 与 dbtalk 使用 CLI 进程边界，服务闭包逻辑保留在
`scripts/database_transfer.py`，Orbit skill 保持 `transfer-database-data` 名称并只描述服务级操作。

## Decisions

- dbtalk 是全库数据传输格式与行为的唯一 SoT；Orbit 通过稳定接口消费，而不复制实现。
- Orbit 保有服务部署闭包的领域选择和验证，因为该规则不能泛化到其他应用。
- 服务导入前仍由目标 Orbit 应用迁移初始化 schema；服务传输不携带 DDL 或其他数据库对象。
- dbtalk 的 `--mode insert|upsert` 由服务导入显式要求；标准目标经 Orbit 迁移初始化后含有种子 Project，服务迁移使用 `upsert`。
- `scripts/` 的 Ruff 只启用基础语法和未定义名称检查；格式、类型和 pytest 由局部 uv 环境统一运行，不借此重构既有运维脚本。

## Risk

- dbtalk 版本、接口或 JSONL 格式不一致会阻断 Orbit 服务迁移；计划阶段需要确定兼容性和错误诊断方式。
- 移除 Orbit 的全库实现时必须保留或迁移已有服务闭包测试，避免把领域关系错误交给通用传输层处理。
- 服务导入仍要求目标数据库 schema 与 Orbit 版本匹配；dbtalk 只能传输数据，不能修复 schema 漂移。

## User review notes

- 用户确认 dbtalk 拥有通用数据库传输的文档、导出、导入、类型转换与 skill；Orbit 只保留服务级能力。
- 用户确认 Orbit 与 dbtalk 使用 CLI 进程边界，而不是跨仓库 Python library API。
- 用户确认保留 `scripts/manage.py backup` 的远程目录备份能力；服务数据导出导入与该运维能力保持独立。
- 用户要求进入 Plan / 计划阶段。
- 用户确认 `scripts/` 使用 uv 管理依赖和工具，测试统一使用 pytest；不把目录改造成严格 Python 项目。
