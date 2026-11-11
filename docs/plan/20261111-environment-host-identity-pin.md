# Plan / 计划：环境主机身份以稳定指纹固定
最后修改时间: 2026-11-11 17:54:54

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

关联 [意图](../intent/20261111-environment-host-identity-pin.md)。新增列只表达策略开关，复用 `host_key_fingerprint` 保存当前策略的固定值；验收以 A01–A09 为准。用户于 2026-11-11 要求“开始实施”，授权本计划范围的实现及必要开发检查。

实施基线为工作树及 `HEAD 1498b88a`，最新迁移为 000056。意图与计划已接受，进入 Implementation；方案接受不表示实现或整体 Verification 已完成。

## Technical decisions / 技术设计与决策

### 选项与默认行为

- `environment` 新增 `host_identity_policy`，取值 `negotiated` / `stable`，非空默认 `negotiated`。`negotiated` 即当前行为，语义不变。
- 两种策略共用 `host_key_fingerprint`；该列只保存当前策略的固定值，不新增第二份指纹或身份类型字段。
- 策略放在既有 SSH target 模型中，本机环境不展示或消费。持久化本机记录时策略使用 `negotiated`；从 local 新建 SSH 目标也按下述创建规则设置策略，不继承已移除 SSH 目标的状态。
- 普通编辑切换任一方向的策略都计入 `environmentTargetChanged` 与 `environmentIdentityChanged`：清空指纹、递增 `target_revision`；重复提交同一策略不触发变化。`SaveInitialization`、`UpdateForUser`、`PrepareSSHEnvironment` 共用该规则。
- 保存环境和生成初始化命令只写配置及当前负责的凭据，不建立 SSH 连接、不轮换现有凭据。旧任务和会话在现有复核点按修订失效，不新增后台中断机制。
- 指纹为空时沿用现有首次记录流程：Probe 成功后记录；终端、命令和转发在同次连接确认后记录，不要求 Docker Probe 成功。部署和 CI 的目标解析仍要求已有固定值。

### 相关列

| 列 | 变化 | 用途 |
| --- | --- | --- |
| `host_identity_policy` | 新增，非空，默认 `negotiated` | 选择现有行为或 `stable` |
| `host_key_fingerprint` | 复用 | 当前策略下唯一的固定指纹 |
| `target_revision` | 复用 | 策略切换作为身份变更递增修订 |
| `last_probe_revision` | 复用 | 保留既有探测记录与目标修订的关联 |
| `last_probe_status`、`last_probe_at`、`last_probe_diagnostic` | 复用 | 保留既有探测诊断；策略切换不伪造新探测结果 |

### 输入、默认值与快照契约

- HTTP 的共享 `EnvironmentSSHTargetReq` 增加 `optional string host_identity_policy`，应用输入保留有无值的信息；初始化向导复用该消息，无需另建接口或修改其请求结构。响应在已有 `ssh` 对象内输出完整策略。
- 新建 SSH 目标且字段未提供时，在应用创建边界设置 `negotiated`；已有 SSH 目标更新未提供字段时保留当前值，防止只改工作区的请求意外切换策略。显式空串或未知值在外部输入边界按现有 validation 错误契约拒绝。
- MCP 更新使用相同的可选输入与应用规则。前端新建表单默认 `negotiated`，编辑从响应恢复已有策略，提交与 dirty 判断包含该字段。
- 数据库默认只负责迁移和列省略写入；SQLC 显式 INSERT 时由应用或本机映射提供确定值。不在 runner、仓储读取或已有内部目标中以空串兜底到默认策略，也不对内部持久化数据重复规范化。
- `TargetDefinition.SSH` 同时携带完整策略和指纹。`configurationFromEnvironment` / `environmentFromConfiguration` 及交接恢复成对传递二者；这是快照恢复，保留包内修订、探测状态与指纹，不执行普通编辑的清空与递增步骤。
- 项目交接当前 `FormatVersion = 2`；本次新增必需的完整快照字段，按既有版本机制升至下一版（当前为 3），同步编解码测试和手册。仅接受当前包格式，不为旧包增加策略推断或兼容转换。未选择覆盖环境的导入仍保留目标环境。

### 稳定身份的表示与校验

- `negotiated`：完全沿用现有实现，`strictHostKeyCallback` 比较 `ssh.FingerprintSHA256(key)`，包含证书 blob。
- `stable`：普通主机 key 存 `ssh.FingerprintSHA256(key)`；CA 签发的主机证书存 `ssh.FingerprintSHA256(cert.Key)`，即证书内公钥指纹。证书 blob 随重签变化，内公钥随 host key 变化。
- 稳定模式信任的是 key，不是 CA；同一 key 的证书与裸公钥是同一身份。不新增 CA 信任、证书主机名或有效期检查，首次记录仍按现有 TOFU 流程执行。
- 在 `internal/infrastructure/runner/ssh` 内集中一个按策略提取指纹的函数，由 strict/probe 两个回调共用。首次观察、后续比对与同连接 rekey 都使用同一算法，不修改 SSH 算法协商优先级。
- **两条实际握手入口都传入策略**：`EnvironmentProber.Probe` 和 `Runtime.dialSSHWithHostKey`。后者承接部署、CI、SFTP、终端、命令、转发；只修改 Probe 将造成探测与执行语义分裂。
- 应用层只处理策略变更、权限、修订、指纹记录和目标传递，不引入 `ssh.PublicKey` / `ssh.Certificate`，也不实现密码学比对。`TargetResolver` 继续检查指纹存在性与凭据，实际握手由 runner 执行。
- 会话接口继续返回一个指纹字符串，无需给终端、命令、转发另加身份对象或接口。现有条件写入保留环境 ID、目标修订、凭据 ID/修订及空值或同值约束，旧修订不能写入新策略的指纹。

### 迁移

- 新增三库迁移，实施时根据当时最高版本选择下一编号，不修改已执行迁移。
- 只新增 `host_identity_policy`，默认 `negotiated`，约束为 `negotiated | stable`；存量 `host_key_fingerprint` 原样保留。
- 同步现有 SQLite/MySQL schema 快照及 SQLC 的查询/模型生成；PostgreSQL 沿用本项目迁移来源。使用 `task sqlc` 和 `task proto`，不手改生成代码。
- down 删除策略列。已启用稳定模式的环境降级前应切回默认策略并按原流程重新记录指纹，不能把稳定指纹直接解释成证书整体指纹。

### 接口与前端

- HTTP 与 MCP 在已有环境对象中读写策略，不新增路由或 MCP 工具。初始化状态响应和“生成 SSH 初始化命令”的请求、响应映射均需传递策略。
- CLI 通过原有 Access/Terminal/Command/Forward 用例取得带策略的目标；不新增 CLI 策略参数、编辑命令或强制改变列表输出。
- 共享 `EnvironmentTargetFields` 在 SSH 初始化和编辑时提供相同选项，复用 `SelectControl`；`environmentForm.ts` 的默认值、回填、请求序列化与 dirty 判断同步。local 表单不出现该选项。
- 环境详情展示当前策略和指纹。说明稳定模式按公钥身份识别，切换后需要由现有流程重新记录；不增加二次确认或 re-pin 交互。
- 用户在实施后明确调整共享 SSH 表单：主机身份校验与类型同行，标题后用问号 Tooltip 展示简短说明，移除下方说明段落；中英文同步。
- 更新活手册中的环境策略、首次记录与降级说明，并同步项目交接格式版本说明；不修改归档材料。

## Implementation steps / 实现步骤

1. 同步策略列、schema、model 与 Repository 映射，通过 `task sqlc` 更新查询生成物；补存量升级与持久化验证。
2. 完成应用输入缺省规则及三个普通保存入口的身份变更逻辑，保留凭据和短事务边界。
3. 在 runner 提取共享指纹算法，并让 Probe/Runtime 两条握手入口传入策略；用回调与本地握手测试验证。
4. 为完整配置与项目交接增加策略，升级包格式；验证导出恢复及不覆盖目标环境的导入行为。
5. 更新共享 proto、HTTP/MCP 映射，执行 `task proto`；更新共享表单、dirty 判断、详情、i18n 和活手册。
6. 按下表补充必要测试并执行项目检查；将实际修改、命令结果、未运行项和偏差写回本计划供实现交付 review，不提前宣称完成整体 Verification。

## Files to change / 需修改文件

| 范围 | 预期文件与职责 |
| --- | --- |
| SSH 算法与握手 | `internal/infrastructure/runner/ssh/environment_probe.go`、`runtime.go` 及必要的同包 helper/测试；终端、命令、转发、pipeline 入口以复用和验证为主，不预设需要修改其生产接口 |
| 应用与模型 | `internal/model/environment.go`、环境 DTO、`usecase/service.go`、`ssh_initialization.go`、`view.go`、`configuration.go`；检查 target/terminal/command/forward 透传，避免无目的改动 |
| 持久化 | `internal/repository/impl/sqlc/environment/repository.go`、`sql/query/environment/environment.sql`、两个现有 schema、三库新增 up/down、SQLC 生成代码和迁移测试 |
| HTTP 与 MCP | 共享 environment proto 与生成代码、HTTP environment/project_initialization 的请求和响应 mapper、`internal/api/mcp/delivery/environment_tools.go` 与 `mapper.go` |
| 项目交接 | `internal/application/project_handover/dto/package.go`、编解码及导出恢复测试，必要的应用映射 |
| 前端 | `EnvironmentPage.vue`、`EnvironmentTargetFields.vue`、`environmentForm.ts` 及轻量测试、生成类型、现有语言包 |
| 活文档 | `docs/handbook/guides/environments.md`、现有升级及项目交接说明 |

表列的是预期范围；实际 diff 应以调用关系为准。`project_initialization.proto` 已复用共享 SSH 消息，不为了形式完整重复定义字段；CLI 生产接口与部署/流水线快照表不预设变更。

## Tasks and coordination / 子任务与协调

不适用（单任务）。

## Verification plan / 验证计划

| 验证边界 | 必要证据 | 意图验收 |
| --- | --- | --- |
| 纯 runner 回调 | 动态生成同一 key 的两张证书：默认模式拒绝变化，stable 通过；普通公钥结果相同，不同 key 拒绝；首次观察与 rekey 使用同一规则 | A03、A04、A08 |
| 应用单测，使用端口替身 | 新建默认、更新省略保留、相同策略不变、双向切换清空并递增、非法外部取值拒绝；三个保存入口复用规则；空指纹部署仍拒绝，已有会话/任务修订复核保留 | A01、A05、A07、A08 |
| 持久化/迁移集成 | 升级前已有行默认 negotiated 且原指纹不变，新增/更新/读取正确；旧修订指纹写入被拒绝且不改变 Probe/Gateway；三库 up/down 与约束一致 | A01、A02、A06、A08 |
| 本地 SSH 协议/组件验证 | 用现有测试服务器检查 `EnvironmentProber.Probe` 与 `Runtime.dialSSHWithHostKey` 均收到策略；不依赖真实远端或 Docker，不能把此类测试计为应用单测 | A03、A04、A05、A08 |
| 契约与快照 | HTTP 初始化/编辑 mapper 与 MCP 输入输出保留策略；交接新版本往返保留策略、指纹和修订，拒绝旧版本；不覆盖环境时保留目标配置 | A07、A08 |
| 前端轻量单测 | 在 `environmentForm.test.ts` 验证默认值、stable 回填、初始化与编辑序列化、仅改策略被判为 dirty；必要时补共享控件基本操作，不串联整页或终端流程 | A07、A08 |
| 项目质量检查 | 使用以下项目现有命令，记录实际通过、失败、跳过与未运行项 | A09 |

实施时从改动相关测试开始；最终在根目录运行 `task check`、`go test ./cmd/... ./internal/... ./sql`，在 `web/` 运行 `corepack pnpm lint:fix`、`corepack pnpm typecheck` 和 `corepack pnpm test src/components/environment/environmentForm.test.ts`（若新增其他轻量测试则一并运行）。不新增前端集成、E2E 或性能测试。

SQLite 在本地持久化测试执行；MySQL/PostgreSQL 沿用 `internal/test/e2e` 的显式配置与隔离测试目标，未启用时记录未实测，不宣称三库运行验证通过。无需为本任务修改用户真实项目配置或启动开发服务器。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- 两条握手入口或初始化、编辑、交接任一映射漏传策略都会造成行为不一致；runner 集中算法，应用集中普通编辑的生命周期，快照恢复独立遵循现有契约。
- 迁移默认值或读写分支设错，会让既有环境在升级后行为变化。
- 稳定身份只覆盖同一 key 的证书变化；网关更换底层 key 仍拒绝连接，属于预期行为。
- 策略切换清空固定值，意味着用户选择重新建立首次信任；不能宣称系统已证明新观测值与旧主机相同。旧修订任务和会话在现有复核点失效，界面给出简短说明即可。
- 包格式升级后旧交接包不作为本次支持的输入；已安装实例升级通过数据库迁移保留默认策略，不能将包格式升级误解为存量数据库需要重新导入。

## Rollback / recovery / 回滚 / 恢复

- 降级前将稳定模式环境切回 `negotiated` 并重新记录对应指纹，再回退代码与结构；不能保证切换后直接恢复连接。
- 迁移仅新增一列；按项目现有各数据库迁移机制执行，MySQL DDL 不作整体事务回滚保证。保留升级前备份供失败恢复。
- 新版交接包不交给旧版本解释；需要降级时使用相应版本备份或切回默认策略后导出的合适资料，不自动改写包版本号。

## User review notes / 用户审查记录

- 2026-11-11：按“pin 稳定身份”方向建立意图与计划。
- 2026-11-11：按用户决策改为环境级选项（默认现有行为）+ 新列稳定身份，其他语义不变。
- 2026-11-11：收敛为一个策略列与一个复用指纹列；移除额外存储、保存时 SSH 连接及不必要的用户待定事项。暂无需要用户确认的未决事项。
- 2026-11-11：整体审查已修正 runtime 建连遗漏、应用/SSH 适配边界、更新省略字段误重置风险、快照恢复与普通编辑混用、交接格式版本及验证范围缺口。本轮仅修订这两份文档，未修改产品或数据库；无新增用户待决策项。
- 2026-11-11：用户要求开始实施；本计划与意图 Accepted，当前由主会话直接实施，无 task 分配。

## Implementation progress / 实现记录

- 2026-11-11 17:22:34：开始实现；已有未跟踪目录 `.openidea/` 不属于本次范围。按存储与应用、SSH 校验、接口与前端、开发检查顺序推进。

### 实际交付

- 新增三库 `000057_environment_host_identity_policy` up/down，存量默认 `negotiated` 并保留原指纹；同步两个 schema、SQL 查询和 `task sqlc` 生成物。模型和仓储读写完整策略，本机记录写入 `negotiated`。
- 三个普通保存入口复用已有 target/identity 变更判定。省略更新字段保留当前策略；双向切换清空 pin、递增修订，相同策略不变；保存保留现有凭据且不调用 SSH。外部空串/未知策略拒绝。
- Probe 与 Runtime 均使用 runner 内同一个指纹提取函数；stable 取证书内公钥，默认保留完整协商对象。部署、CI、SFTP、终端、命令和转发沿现有目标传递，无额外生产接口。
- 配置快照、HTTP 初始化与环境 mapper、MCP 输入输出、proto 和生成类型均携带策略。项目交接格式升为 3，恢复策略、pin、修订及探测状态；旧包拒绝，不覆盖环境的现有流程保留。
- 共享 SSH 表单增加选择、默认/回填/请求/dirty 判断，详情展示当前策略，中英文文案及环境、升级、项目交接手册同步。
- 补充回调单测、本地 SSH 协议验证、三个保存入口的策略生命周期、SQLite 历史升级/约束/down、持久化旧修订拒写、HTTP/MCP 透传、交接往返/旧版本拒绝及前端轻量表单测试；同步受影响的既有 SSH 测试夹具。

### 范围核对与偏差

- 无设计或业务范围偏差。`ssh_initialization.go`、CLI、部署/流水线生产入口及快照表直接复用共享模型和既有修订逻辑，无需修改。
- 调用关系核对发现 `scripts/merge_orbit_databases.py` 会按目标字段合并探测记录，已将策略纳入比较，并补原有脚本测试，避免相同修订号但策略不同的两份数据复用探测状态。这是预期文件表之外的必要同步。
- 只维护活手册和本计划；未修改用户真实项目配置、远端主机、已执行迁移或归档资料，未启动/重启服务。实际交付为基线 `1498b88a` 上当前工作树 diff，未创建分支或提交。

### 开发检查

| 命令或边界 | 实际结果 |
| --- | --- |
| `task sqlc`、`task proto` | 通过；proto 的无配置依赖提示不影响生成 |
| 相关 Go package 检查 | 通过；首轮暴露旧 SSH 夹具缺失策略，已显式同步合法值，无运行时空串兜底 |
| `go test -count=1 ./cmd/... ./internal/... ./sql` | 通过，包含 SQLite 持久化/迁移和本地 SSH 协议测试 |
| `go test -tags=unit ./internal/application/... ./internal/model/... ./internal/common/... ./internal/queue/...` | 通过，共享夹具在核心单测入口下仍有效 |
| `corepack pnpm lint:fix`、改动文件 Prettier、Go fmt | 通过；只产生本次改动文件的格式调整 |
| `corepack pnpm typecheck` | 通过 |
| `corepack pnpm test --reporter=default --silent=false --no-color` | 43 个文件、194 个测试全部通过，未出现 warning/stderr |
| `python -m unittest discover -s scripts -p test_merge_orbit_databases.py` | 18 个测试通过 |
| `task check` | 全部通过，Go lint 为 `0 issues` |
| `git diff --check` | 通过 |
| MySQL/PostgreSQL 及本机 Windows OpenSSH 外部联调 | 按既有显式开关跳过；另外以 `go test -v ./internal/test/e2e -run 'Test(MySQL\|PostgreSQL)MigrationE2E\|TestWindowsSSH'` 核实跳过原因，未启用外部目标 |

上述为 Implementation 内必要开发检查，未进入整体 Verification，也未创建验证报告。稳定模式仍是公钥 pin 与原有 TOFU；未证明旧故障项目的历史身份连续性，也未自动为该项目切换策略。MySQL/PostgreSQL DDL 和真实代理证书联调尚无实测证据，部署时需按实际目标核对。

2026-11-11 17:41:45：Implementation 已完成，必要开发检查通过，无实现未完成项或新增用户待决策项；整体 Verification 尚未开始。

## Overall verification / 整体验证

- 2026-11-11：用户明确要求通过 `best-practices:dev-check` 与 `specflow:specflow` 完成验证，授权进入整体 Verification。
- 输入为 `HEAD 1498b88a` 上的当前工作树，包含实施后按用户要求调整的同行布局、问号 Tooltip 与简短中英文说明；现有暂存状态由用户维护，本轮不改动索引。
- 按 A01–A09 核对代码、迁移、接口、快照与测试证据，并运行现有只读质量入口。外部数据库与真实 SSH 联调继续遵循显式开关。
- 2026-11-11 17:54:54：整体验证完成，见[验证报告](../verification/20261111-environment-host-identity-pin.md)。`task check` 为 0 issues，`task test`、全量 `go test -count=1 ./cmd/... ./internal/... ./sql`、前端显式日志测试（194 个）与脚本单测（18 个）均通过；外部 MySQL/PostgreSQL 和 Windows OpenSSH 用例按既有开关跳过。交接包导出/导入与 MCP 已支持策略，无实现遗漏；验证阶段仅补过程记录，保留用户暂存状态。
