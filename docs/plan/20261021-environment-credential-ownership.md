# Plan / 计划：环境凭据以环境为归属主表
最后修改时间: 2026-10-21 14:13:03

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

- [同名意图](../intent/20261021-environment-credential-ownership.md)已因用户于 2026-10-21 要求“开始计划”而接受。本计划覆盖 A01–A08。
- 用户于 2026-10-21 要求“开始实现”，本计划据此接受；随后要求“完成 SpecFlow 验证”，授权已覆盖实现、必要开发检查和整体验证，业务数据库迁移不在本次执行范围。
- 调查基线：`ef610ac4`。工作树包含本任务意图文档；以实施开始时的最新迁移序号和实际代码再次核对基线。
- 遵循 [AGENTS.md](../../AGENTS.md)、[后端架构治理 skill](../../.agents/skills/backend-architecture-governance/SKILL.md)、[产品模型](../archive/cd-model.md)和[CD 运行时](../archive/architecture/cd-runtime.md)。

## Technical decisions / 技术设计与决策

### 1. 持久化关系

`environment_credential` 是 Environment 的可选子记录，保存当前使用的受管 SSH 密钥。

| 对象 | 最终约束与职责 |
| --- | --- |
| `environment` | 保留环境目标、工作区、主机指纹、目标修订和 Probe/Gateway 字段；删除 `ssh_credential_id`、`ssh_credential_revision` 两个存储列 |
| `environment_credential` | 保留 `id`、`public_key`、`encrypted_private_key`、`revision`、`created_at`；以非空 `environment_id` 替换 `project_id` |
| 当前凭据唯一性 | `UNIQUE(environment_id)`，一环境至多一份当前凭据；`revision >= 1` |
| 主从完整性 | `environment_id` 引用 `environment(id)`，使用物理外键和 `ON DELETE RESTRICT`，子记录必须有实际存在的环境 |
| Project / Gateway 关系 | 沿用环境对 Project、Gateway 的逻辑关联；本次物理外键仅约束同一环境内的从属密钥 |
| Deployment / PipelineRun | 保留已有凭据 ID、修订快照；继续作为执行输入，不加指向当前凭据行的物理外键 |

选择物理外键的依据：旧凭据表已经对 `project_id` 使用物理外键，本次把受约束的归属改为真正的主表 Environment。删除环境仍需显式处理从属记录，不增加环境删除 API 或级联删除业务。

local 环境与未初始化 SSH 环境没有凭据子行；凭据所属环境不能通过更新子行而迁移到另一个环境。项目归属从 Environment 获取。

### 2. 环境读取与凭据访问

沿用 `EnvironmentStore` 返回的环境聚合读取形态。`EnvironmentById`、`EnvironmentByProjectId`、`EnvironmentByTarget` 按 `environment_credential.environment_id = environment.id` 做左连接，只投影凭据 ID 和修订，不读取密钥内容；local 不投影 SSH 凭据。

`model.EnvironmentSSHTarget.CredentialId/CredentialRevision` 保留为明确标注的只读派生信息，来源只有子表。环境 INSERT/UPDATE 参数中移除这两个字段；不允许通过写 Environment 修改凭据。这样 PipelineRun 的目录定位、任务快照、日志和会话复核仍能一次读取目标及其当前凭据身份，不把凭据仓储和解密能力注入这些业务。

凭据仓储调整为按环境获取、创建和删除：

- 新增 `EnvironmentCredentialByEnvironment(ctx, environmentId)`；读取密钥的业务以环境 ID 定位子行。
- 删除使用明确的环境归属条件；凭据 ID 保留作身份与快照核对。交接替换以删除后创建新 ID 完成；不保留无生产调用方的原地更新接口，也不提供转移归属的写入口。
- 移除没有生产调用方的 `LatestEnvironmentCredentialByProject` 及其 SQL；原按凭据 ID 读取接口在调用方迁移完后删除，不保留备用路径。
- `matchesEnvironmentCredential` 核对 `credential.EnvironmentId == environment.Id`，并比较环境读取快照中的凭据 ID、修订；两次读取之间发生变化时返回冲突或现有无效凭据诊断，不能混合旧目标和新密钥。
- SQLC 使用 `sql/schema/mysql.sql` 作为 schema 输入，需同步 schema 并运行 `task sqlc`；各包生成的共享 model 变化属于预期生成结果。

这两个派生字段仅服务运行时读取，数据库中不保留反向绑定，也不保留双读或双写。

### 3. 保存、初始化与类型切换

纯数据库写入继续参与现有 HTTP 写 UoW、MCP `addWriteTool` 或调用方短事务。Repository 使用当前 context 中的事务；不新增 Repository 事务回调，不把 SSH、Probe 或会话放入长事务。

| 操作 | 凭据行为 | 修订与指纹行为 |
| --- | --- | --- |
| 首次保存 local / SSH 配置 | 只创建 Environment | 新环境 `target_revision=1`；SSH 可无凭据 |
| 首次生成 SSH 初始化命令 | 先创建或保存 Environment，再查询或创建其凭据，最后重读聚合 | 新环境维持修订 1；已有环境新增密钥时目标修订增加一次并清空指纹 |
| 重复生成命令 | 复用该环境的有效凭据、公钥和创建时间 | 无实际变化时不增加修订、不重置指纹 |
| 修改 SSH host / port / username / platform | 复用同一环境的密钥 | 目标修订增加一次，清空指纹 |
| 仅修改工作区 | 保留密钥 | 目标修订增加一次，保留指纹 |
| SSH 切换为 local | 同一事务删除该环境的凭据子行并保存 local 配置 | 清空 SSH 数据，目标修订增加一次 |
| local 切回 SSH | 先保存未初始化 SSH 配置；显式生成命令时新建密钥 | 不找回已删除密钥，不选取历史记录 |

`PrepareSSHEnvironment` 重排当前“创建凭据后保存环境”的顺序。先完成输入校验，再保存主表，随后创建从表；创建失败、唯一约束冲突或后续写入失败必须由原 UoW 整体回滚。环境配置和凭据同时变化时，目标修订只增加一次。普通保存、读取和 Probe 不生成密钥。

唯一约束冲突不能通过覆盖已有子行来解决；沿现有错误契约传播失败，由后续正常请求读取已提交记录。密钥创建、更新和删除规则在 environment 应用用例内维护，HTTP/MCP 只映射输入输出。

### 4. 指纹条件更新与运行时身份

`RecordEnvironmentHostKey` 的单条条件 UPDATE 保留环境 ID、目标类型、目标修订和已有指纹条件；原环境列上的凭据比较改为 `EXISTS` 子查询，校验子表 `environment_id`、凭据 ID、revision。

该写入在一条 SQL 中检查当前凭据，不能用“先查凭据，再无条件写指纹”替换。继续保留 MySQL 相同值可能返回零影响行时的重读核对，并从新的聚合查询获得当前凭据身份。重复 SQL 参数使用显式 `sqlc.arg(...)` 并检查生成绑定。

Probe、TargetResolver、TerminalService、CommandService、ForwardService 和交接导出统一通过新归属读取密钥。Deployment/PipelineRun 快照、Web 终端票据、CLI forward 记录和容器日志来源身份继续消费当前凭据 ID、revision；前端或外部 API 不新增凭据身份字段。

环境变化、凭据替换或删除后，排队任务与会话仍按既有规则拒绝旧目标或结束连接。Probe 成败不作为执行资格；首次受管连接固定指纹和后续严格校验的行为保持。

### 5. 项目交接

`TargetDefinition` 已剥离 Project 和存储身份，交接包无需加入 `environment_id`。导出按当前环境读取凭据；导入时先确定或创建目标 Environment，再将包内密钥保存为该环境的子记录。

- `SaveTargetDefinitionForUser/Handover` 先验证完整配置，再保存主表，最后写入或移除子行；参与调用方整体事务。
- 恢复 SSH 凭据时为目标环境创建新的凭据 ID；若目标已有凭据，在同一事务中删除旧行后创建新行。保留输入的密钥内容和凭据 revision，使用目标实例加密配置加密落库。
- 采用新 ID 是为了让替换密钥的操作具有新的执行身份，即使交接包里的 revision 恰好与旧值相同，旧任务或连接仍不能继续使用旧凭据。
- 普通生成初始化命令仍复用现有 ID；此处的新 ID 只用于显式完整配置恢复。
- local 恢复删除已有 SSH 子行；未选择覆盖环境时不改目标环境和凭据。
- 保留交接包当前格式及其 SSH credential revision 字段，由子表读取生成；继续校验包内修订一致性，导出私钥仍使用加密字段。
- 保留现有恢复目标修订、Probe 记录与 Gateway 关联的规则；此次不扩展交接能力。

### 6. 数据迁移

当前最高序号为 `000053`，预计新增三库 `000054_environment_credential_ownership.up/down.sql`；实施前重新确认序号。旧迁移文件不改写。

有效绑定的唯一映射来源是旧 `environment.ssh_credential_id`，并校验 target_type 为 SSH、原凭据 Project 归属正确、凭据修订匹配。不能按 `project_id` 或创建时间挑选一份凭据。

升级策略：

1. 以旧 schema 做预检查，区分未初始化 SSH、有效绑定、未绑定旧凭据和异常绑定。预检查报告仅含 ID、数量及问题类型，不输出密钥内容。
2. 正常未初始化 SSH 没有子行；有效绑定按原 ID、公钥、密文、revision、创建时间复制，并写入对应 `environment_id`。环境修订、指纹、Probe/Gateway 数据原样保留，避免让有效排队任务仅因结构迁移失效。
3. 没有任何当前环境绑定的旧凭据不进入新当前凭据表。升级前数据库备份保存这些旧行；Deployment/PipelineRun 的历史 ID、修订快照原样保留，它们不依赖旧私钥继续存在。指南明确这项数据收敛及 down 无法恢复这些旧行的限制。
4. 对非空绑定缺少凭据、跨 Project、修订不一致或重复绑定等异常，在破坏性 DDL 前失败；不把错误绑定当作未初始化状态，不自动重置密钥或指纹。实际异常修复单独基于数据事实处理。
5. 建立新子表及约束，复制有效行后去掉环境旧列。SQLite 重建所需表时明确事务及外键检查顺序；MySQL 拆掉涉及旧修订列的复合 CHECK，保留 `target_revision >= 1`；PostgreSQL 同步处理关联约束。失败时不继续启动应用。
6. 同步 SQLC schema；凭据表声明放在环境主表之后，以反映新的物理依赖。

新增简短迁移指南，给出预检查 SQL、备份要求、迁移后有效行数/归属/约束核对及恢复方式。实现和测试使用临时或隔离数据库；本次计划不读取、备份、清理或迁移用户现有业务库。

### 7. 工作区迁移工具与文档

`scripts/migrate_workspace.py` 的 `inventory`、`select_environment` 当前直接读取 `e.*`，`connection_for` 按旧凭据 ID 查询并检查 Project 归属；Go helper 同样比较 `Credential.ProjectId`。它们会在删列后失效，必须一并调整。

- 脚本库存查询按环境左连接凭据，取得当前 ID/revision 元数据；连接查询按 `environment_id` 取密钥。
- Go helper 的凭据输入使用 `environment_id`，核对所属环境及库存中的 ID/revision。
- 工作区迁移的 PipelineRun 快照比较仍使用实际当前凭据元数据；库存摘要纳入这些元数据，以便密钥变化后原预演失效。
- 更新脚本现有轻量测试，不运行真实目录搬迁、容器重建或远端 SSH。
- 同步产品模型、后端/运行时文档及相关指南中的主从关系和生命周期描述；历史 SpecFlow 文档仅引用，不改写其结论。

## Implementation steps / 实现步骤

1. **存储及迁移**：新增三库 up/down、更新 SQLC schema 和查询，生成代码，调整环境和凭据 Repository。审查迁移 SQL 的数据保留和异常处理规则；Repository 测试验证当前关系的唯一约束、外键及指纹条件更新，历史数据迁移留作发布前专项检查。
2. **环境用例**：调整模型归属、初始化创建顺序、SSH/local 切换与凭据获取；修订更新一次，读取从表重建运行时身份。同步替身，保持其仅提供存储事实。
3. **执行与交接**：核对 Probe、CI/CD、票据、终端、exec、forward、日志及交接链路；更新完整配置恢复的主从写入与凭据 ID 语义，验证事务回滚和旧身份失效。
4. **运维工具与活文档**：调整工作区迁移 Python/Go helper，补充本次 schema 迁移指南并同步模型说明。
5. **集成检查**：运行固定检查和最小有效验证，逐项对照 A01–A08；记录实际 diff、未运行的外部检查、范围偏差和遗留风险，交付实施结果。

各步骤在同一任务内顺序实施；不能在应用仍按旧列读取时单独交付 schema 升级供部署。

## Files to change / 需修改文件

| 范围 | 预期文件 / 模块 |
| --- | --- |
| Schema / SQLC | `sql/migration/{sqlite,mysql,postgres}/000054_environment_credential_ownership.{up,down}.sql`（序号待实施前核对）、`sql/schema/mysql.sql`、`sql/query/environment/`、`sql/query/environment_credential/`、`internal/gen/sqlc/**` 自动生成结果 |
| 模型 / 仓储 | `internal/model/environment.go`、`environment_credential.go`，`internal/repository/environment_credential.go` 及两个 SQLC Repository；环境 Store 接口只在实际签名需要变化时调整 |
| 环境用例 | `internal/application/environment/usecase/{service,ssh_initialization,ssh_key,configuration,probe,target,terminal}.go` 及相应测试；command/forward 的读取路径按实际改动核对 |
| 相关业务 | `internal/application/project_handover/`、`deployment/usecase/`、`pipeline_run/usecase/` 的相关测试与必要实现；不进行无关模块重构 |
| 持久化验证 | 两个环境 Repository 测试、环境应用集成测试及受新约束影响的现有 fixture；不新增数据迁移测试 |
| 运维脚本 | `scripts/migrate_workspace.py`、`scripts/workspace_migration_runtime.go`、`scripts/test_migrate_workspace.py` |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/backend.md`、`docs/architecture/cd-runtime.md`、`docs/decisions/ledger.md`、相关部署/工作区/CLI 指南；新增 `docs/guides/environment-credential-migration.md` 并登记 `docs/INDEX.md` |
| 过程记录 | 本计划与后续同名 verification；前端、Proto 和公开接口形状预计无需修改 |

共享生成代码可能让环境以外的 SQLC package model 一起变化，审查时区分必要生成结果和无关手工编辑。现有历史升级测试仍使用当时的旧字段；当前 schema 的业务夹具必须先创建主表，再创建子表。

## Tasks and coordination / 子任务与协调

不适用。保持单任务、主会话顺序实施，统一负责集成和验收；不创建 task 或分配记录。存储投影、生命周期和运行时身份是同一交付结果，不安排平行修改共享契约。

## Verification plan / 验证计划

| 验收 | 最小有效证据 |
| --- | --- |
| A01 | 当前 schema 的真实 SQLite Repository 测试：按环境取凭据；无主表不能插入、同环境不能重复；跨环境不会读到另一份密钥；环境写入不能改变子表身份 |
| A02 | 真实用例验证首次创建及重复复用；在现有短事务中让凭据写入失败，证明新环境与子行一起回滚；已有环境修改失败恢复原目标与凭据 |
| A03 | 保存未初始化 SSH/local 不建密钥；SSH→local 删除子行；切回 SSH 后显式初始化新建密钥；工作区与 SSH 身份变更沿用各自修订/指纹规则 |
| A04 | Probe/目标解析读取新归属；真实 SQL 验证指纹写入拒绝旧凭据；复用现有 Deployment/PipelineRun、terminal/command/forward 和日志测试验证旧身份失效，不重新搭建整条端到端链路 |
| A05 | 交接恢复先有主表；目标归属正确、加密内容可解密；替换密钥产生新 ID，即使包 revision 相同也使旧身份失效；失败回滚；响应及导出保持原密钥暴露边界 |
| A06 | 审查三库迁移 SQL 的映射、异常检查及回退规则；实际数据升级与回退按迁移指南在发布前专项验证，不纳入本次单元测试覆盖 |
| A07 | 检索当前 SQL、代码与脚本，确认不再读写环境旧列或凭据 `project_id`；允许任务快照、读取投影、交接字段和历史迁移保留同名语义；文档引用有效 |
| A08 | 以下项目命令结果与未运行原因完整记录；最终 diff 与上述范围一致 |

检查命令：

```text
task sqlc
task test:go:unit -- -count=1
task check
go test -count=1 ./cmd/... ./internal/... ./sql
python -m unittest discover -s scripts -p test_migrate_workspace.py
go build -o <临时目录>/workspace_migration_runtime.exe scripts/workspace_migration_runtime.go
```

最后一项验证带 `//go:build ignore` 的运维 helper，避免默认 Go 扫描漏检；输出到临时目录。完整 Go 命令覆盖项目固定检查范围并纳入迁移嵌入包，与 `task test:go:all -- -count=1` 范围一致，无需通过后重复跑同一全量入口。仅修改文档的计划阶段只检查结构、引用与 diff。

持久化/事务测试按现有分类使用本地 SQLite 集成边界，核心规则测试用真实用例和存储事实替身。单元测试不覆盖数据迁移，本次不新增历史升级夹具及 up/down 测试。不能让 fake 自己重演归属或回滚规则。若修改前端，另运行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。

MySQL/PostgreSQL 按既有 `internal/test/e2e` 显式配置在隔离库验证升级/回退和查询。MySQL 使用 `BACKEND_GO_E2E_CONFIG`，PostgreSQL 使用 `BACKEND_GO_POSTGRES_E2E_CONFIG` / `BACKEND_GO_POSTGRES_E2E_DSN`。未提供隔离环境时记录未运行；SQLite 通过不能算三库运行验证通过。计划阶段不启动数据库或开发服务器。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- 尚未读取任何实际业务库，未绑定历史凭据数量和异常绑定情况未知；这不妨碍按明确迁移规则实现，实际升级前必须以预检查结果决定数据修复范围。
- 未绑定旧凭据不迁入新表，完整恢复旧记录依赖升级前备份。历史任务 ID/revision 留存不代表旧私钥仍在线可取。
- 新物理外键使现有“先凭据后环境”的 fixture 和恢复流程必须改序；不能靠测试关闭外键规避真实约束。
- `RecordHostKey` 的凭据条件移到子查询后，必须有真实 SQL 证据；仅调整单测替身不足以验证旧身份无法写入。
- MySQL DDL 存在隐式提交，SQLite 当前迁移驱动设置 `NoTxWrap: true`。不能假定一次失败会自动回滚整份迁移；新脚本需明确可事务部分和恢复边界。
- 工作区搬迁工具不是正常应用编译范围，必须单独验证；使用旧 schema 库存生成的未完成搬迁批次应在 schema 升级前完成或撤销并重新预演，不增加双 schema 解释器。
- 暂无需要用户额外决定才能完成本计划的未决事项。以上物理外键、类型切换删除子行及未绑定旧行收敛策略均列为本计划的审查内容。

## Rollback / recovery / 回滚 / 恢复

- 应用和 schema 作为同一版本切换；升级期间停写且结束活动 worker/CLI 连接，完成数据库与加密配置备份，再执行迁移。这是发布操作安排，本计划阶段不执行。
- down 从当前子表恢复环境的凭据 ID/revision，并通过 Environment 恢复凭据 `project_id`；先解除新外键依赖，再还原旧结构，保留当前有效密钥。
- down 不恢复升级时剔除的未绑定旧凭据。需要完整恢复迁移前数据时，还原数据库备份及对应应用版本；不得通过旧应用读取新 schema。
- SQLite 失败检查事务与外键状态；MySQL/PostgreSQL 检查已执行 DDL 和 migration dirty 状态，按备份恢复，不直接强行标记迁移成功。
- 事务内初始化/恢复失败由既有 UoW 回滚，不发布半完成环境；有效密钥原样迁移无需目标宿主机重新安装公钥。

## User review notes / 用户审查记录

- 2026-10-21：用户要求“开始计划”；已将同名意图置为 Accepted，编写本 Draft。
- 2026-10-21：依据代码确认额外影响包含工作区迁移 Python/Go helper、PipelineRun 文件定位以及主机指纹条件写入，已纳入实施和验证范围。
- 2026-10-21：用户要求“开始实现”，计划置为 Accepted，主会话开始顺序实施。

## Implementation record / 实施记录

- 2026-10-21 13:35:23：接收实施，工作基线为 `ef610ac4`，仅有本任务过程文档未跟踪变更；按单任务完成存储、用例、工具与开发检查。

### 实际交付

- 存储：新增 SQLite/MySQL/PostgreSQL 的 000054 up/down，约束为 `environment_id NOT NULL UNIQUE` + `ON DELETE RESTRICT`；有效旧绑定保留原密钥 ID、密文、修订、时间以及环境状态和任务快照。异常绑定提前拒绝，未绑定旧密钥不进入当前表。PostgreSQL 显式区分升级/回退主键约束名，避免重命名表后索引名冲突。
- 仓储与用例：Environment 左连接派生凭据身份，写入移除凭据字段；密钥按环境读取。初始化先写环境后写凭据并重读；SSH→local 删除子行，切回 SSH 需显式生成新密钥。交接恢复使用新凭据 ID，保持包内密钥/revision，旧会话不再有效；指纹写入通过 EXISTS 校验当前子表身份。
- 工具与文档：工作区 Python/Go helper 使用新的环境归属；同步当前测试夹具、SQLC、MySQL/SQLite schema 快照及活文档，新增[迁移指南](../archive/guides/environment-credential-migration.md)并登记索引。公开 API、交接格式、前端及历史迁移不变。
- 最小持久化证据：真实 SQLite 外键/唯一约束、无父环境插入拒绝、环境写入不能覆盖凭据、按环境隔离读取；首次创建与重复复用、类型切换、相同 revision 的交接替换使旧会话失效；新建、已有未初始化环境及交接替换三种凭据写失败均由调用方事务恢复原数据。
- 迁移 SQL 已完成映射、异常检查及回退规则审查。初次实施曾运行版本 53→54→53→54 的测试；用户随后明确单元测试不覆盖数据迁移，已移除本次新增的迁移测试及旧数据夹具，不再将其作为持续测试交付。实际升级验证按迁移指南执行。

### 开发检查结果

| 命令 / 范围 | 结果 |
| --- | --- |
| `task sqlc` | 通过，生成查询及所有共享 model |
| `task test:go:unit -- -count=1` | 通过 |
| `task check` | 通过：前端 typecheck/lint/format，Go 格式及 lint（0 issues） |
| `go test -count=1 ./cmd/... ./internal/... ./sql` | 通过，包含本地持久化、事务、迁移、协议和进程测试 |
| `python -m unittest discover -s scripts -p test_migrate_workspace.py` | 28 项，23 通过、5 按既有外部条件跳过；预期停机提示 warning 不影响结果 |
| `go build -o <系统临时目录>/orbit-credential-ownership-runtime.exe scripts/workspace_migration_runtime.go` | 通过 |
| `git diff --check` | 通过 |
| MySQL/PostgreSQL 隔离库、Cygwin 外部用例 | 未配置既有显式启用变量，未执行；三库迁移文件齐全不代表三库运行验证通过 |

### 实施偏差与剩余边界

- 计划原拟保留凭据原地更新接口；交接统一 delete/create 新 ID 后已无生产调用方，按仓库治理约定一并删除该接口、SQL 和 fake 方法。读取、创建、按环境删除已覆盖全部现有用例。
- 检索时发现仍保留的 SQLite schema 快照含旧归属，增加同步该文件；SQLC 输入仍为 MySQL schema，无双 schema 兼容逻辑。
- 交付位于当前工作树，未提交 Git；没有运行实际业务库升级，也没有启动或停止开发服务器。业务库异常数量及未绑定旧密钥规模仍需发布前预检查；down 不恢复未绑定旧密钥。
- Implementation 及必要开发检查完成；用户随后授权整体 Verification，验证结论与证据见[同名验证报告](../verification/20261021-environment-credential-ownership.md)。

- 2026-10-21：按用户反馈移除本次新增的数据迁移测试；删除后重新运行 task check（0 issues）及 go test -count=1 ./cmd/... ./internal/... ./sql，均通过。
- 2026-10-21：按用户“完成 SpecFlow 验证”的授权完成意图、最新计划与实际 diff 核对；本轮重生成 SQLC 无漂移，复用最近通过的固定检查，整体验证通过。数据迁移仅作 SQL 审查，实际库升级/回退及外部环境运行验证留作发布前专项检查。
