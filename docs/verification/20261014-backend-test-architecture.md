# 后端测试与架构治理

日期：2026-10-14。范围：默认 Go 测试、生产与测试依赖、HTTP/MCP/worker 入口事务、执行失败恢复及测试漂移。首轮治理 B1–B9，后续按用户要求继续以测试耗时和装配问题治理 B10–B13；两阶段结果分别记录。

## 总进度与后续安排（持续更新）

当前到第十二轮，已登记 B1–B23，23 项均完成并通过最终全量验证。B14 的最后一批渲染边界已收尾，后续不默认继续拆 Compose。问题完成数只代表已登记事项，不能推导整个后端的完成率。

| 工作线 | 当前进度 | 验收与后续安排 |
| --- | --- | --- |
| 事务与失败收敛 | 已完成 B1、B3、B6–B9 | DB/Tx 去 Gin 依赖；MCP 写回滚、文件/证书短事务、取消与重投递收敛。保留对应持久化和失败传播证据。 |
| 应用职责与依赖 | 已完成 B10、B11、B17、B18、B20、B21 | 状态检查独立，命令/执行拆分，终端周期规则归应用，Worker 去 SQL 依赖；实际构造与核心依赖复核通过。 |
| 外部执行与格式边界 | 已完成 B14、B19 | Docker 观察/命令/解析归适配器；总预算覆盖 readiness 全阶段。渲染策略与格式分开，Gateway 计划原子准备、渲染不重复修改；Compose 仍是合理的应用输出映射。 |
| 测试必要性与漂移 | 已完成 B2、B4、B5、B12、B13、B15、B16、B22、B23 | Application 命名集成 19 文件/49 项、Deployment 接管 5 项、Repository 34 项已有保留依据。第十二轮复查迁移、文件、SSH 与终端：删 1 项重复种子验证；修复全局句柄计数误报。分类数不是全仓集成测试总数。 |
| 反馈成本 | 核心 357 项、12.85 秒墙钟、0.24 秒用例和；完整 820 通过/13 跳过、46.60 秒；阶段成本对照已完成 | 全量空跑仍 46.66 秒；代表二进制首次启动 0.50–1.07 秒、再次启动 0.016–0.026 秒。后续焦点为 Windows 构建/调度/新进程启动成本，不能继续归因于业务规则或靠删集成测试提速。 |

### 接下来的有界工作

1. **已完成：B14、B23 与全量回归**。固定质量检查、核心与完整测试、预期警告、旧调用/文档漂移、暂存全部变更。已失败的完整运行必须保留记录并在修复后重新执行，不能用定向通过代替。
2. **下一优先项：Windows 启动成本的系统归因**。现有对照已区分“执行测试”和“零用例入口”，并测出代表包构建及二进制首/复次启动。后续只读追踪进程创建、Go 调度/链接和系统安全扫描，找出可复现的等待来源后再决定优化；当前不宣称杀毒软件是原因，不调整全机安全设置，不靠降低测试并发或新增 skip 掩盖成本。
3. **外部证据与增量复审**。13 项显式外部/环境场景仍按原条件跳过，默认通过不能证明真实 MySQL/PostgreSQL、远端 SSH、Docker/Traefik 环境。后续仅在有对应变更或明确联调目标时执行。全局扫描和代表调用链复查不能证明每一行均无问题；没有新证据时不继续人为制造拆分任务。

日常改动先执行对应包的单测，再用核心入口检查业务边界；核心入口不替代项目固定的完整检查。每轮报告本轮完成项、未完成项、验收结果和下一项；所有修复、测试与文档按既有授权暂存，不创建提交。

## 依据与边界

- 当前 `AGENTS.md`、`Taskfile.yml`、[后端架构](../architecture/backend.md)、CD 产品模型与运行时约定。
- 用户指定的 K12 Delivery OS `architecture-governance`：采用按职责追踪、通过端口直接测试、治理实现与测试依赖的原则。
- `best-practices:backend-service-architecture`（0.1.0）：采用分层、依赖方向、生成代码隔离、组合根和适配器边界。
- 项目内 [backend-architecture-governance](../../.agents/skills/backend-architecture-governance/SKILL.md) 及其[项目边界](../../.agents/skills/backend-architecture-governance/references/backend-boundaries.md)。该 skill 自包含，无须另一项目或插件存在。

应用用例与 model 保持依赖内层端口；Gin/HTTP、MCP、SQLC 及外部协议留在对应适配边界。检查覆盖代表调用链和依赖扫描，不声称逐行审计全仓或验证外部环境。前端测试限制不扩展为后端禁止必要的 SQLite、协议、取消和事务验证；不以覆盖率或测试数量作为治理目标。

## 治理结果

| 编号 | 原风险 | 当前状态与证据 |
| --- | --- | --- |
| B1 / P2 | 通用 DB/Tx 包传递依赖 Gin | 已修正。HTTP 事务中间件及测试移至 [middleware/transaction.go](../../internal/api/http/middleware/transaction.go)，bootstrap 使用 `middleware.Transaction`。Service Repository 与 worker 的传递依赖不再含 Gin。 |
| B2 / P3 | 密码规则测试依赖完整迁移 | 已修正。[user/service_test.go](../../internal/application/user/usecase/service_test.go) 直接执行真实用例与密码处理，通过存储事实替身验证 36/37 字符边界和必填项；唯一约束、删除和历史保留仍由原集成测试验证。 |
| B3 / P2 | 取消规则的短预算限制真实持久化清理 | 已修正。[execution_control_test.go](../../internal/application/deployment/usecase/execution_control_test.go) 使用 `testing/synctest`、事实端口和明确同步点验证取消、超时、晚退出；[operation_takeover_test.go](../../internal/application/deployment/usecase/operation_takeover_test.go) 保留真实事务、准入与条件更新，移除执行器短超时配置。 |
| B4 / P3 | 普通 fixture 使用过时状态/类型 | 已修正。Application fixture 使用 `standard`，Version 使用 `unpublished`。Pipeline kind `application` 为合法现行值，保留。未改已执行迁移。 |
| B5 / P3 | 测试维持无生产用途的业务 helper | 已清理。核对无生产调用后删除 `internal/common/runtimeconfig` 及其测试，活文档不再将其列为现行实现落点或保留约束。 |
| B6 / P1 | MCP 多步写入失败可删除原配置 | 已修正。29 个纯数据库写工具通过 [addWriteTool](../../internal/api/mcp/delivery/server.go) 使用注入事务端口；stdio 与对话 MCP 复用 [bootstrap 装配](../../internal/bootstrap/http.go)。提交失败不暴露成功结果。 |
| B7 / P2 | 配置文件保存受无关 SQL 提交影响 | 已修正。PUT/DELETE `/api/setting/config` 排除 HTTP SQL 事务，保留文件存储自身的 revision、校验与恢复机制；[装配测试](../../internal/bootstrap/http_transaction_test.go) 验证排除配置。 |
| B8 / P1 | PipelineRun 读取失败滞留 running，重试伪成功 | 已修正。[ExecutePipelineRun](../../internal/application/pipeline_run/usecase/execution.go) 在 Begin 后负责错误收敛，使用独立有界清理 context；worker 传入投递次数，重投递对遗留 running 记录中断失败，不重放外部操作。 |
| B9 / P2 | mkcert 外部进程持有 SQL 请求事务 | 已修正。`/api/route/:route_id/mkcert` 排除请求事务；生成后复核成员权限，通过 [UpdateRouteCertificate](../../sql/query/route/route.sql) 按 Project、HTTP 协议、原 domain 和 updated_at 条件仅更新证书字段。变化时返回冲突。 |

### MCP 原子性

初审临时诊断：内存 MCP 调用真实 Application 用例与 SQLite Repository，trigger 拒绝插入新 env，工具报错后原 `OLD=old` 却已删除；相同调用在短事务内能保留原项。

长期验证 [transaction_integration_test.go](../../internal/api/mcp/delivery/transaction_integration_test.go) 保留这个必要的跨入口失败边界：新项失败时原集合、版本 summary 和更新时间不变；取消故障后完整替换成功。[transaction_test.go](../../internal/api/mcp/delivery/transaction_test.go) 验证提交失败不会泄露尚未提交的成功结果。协议测试中的无操作事务替身只用于工具契约，不能作为回滚证据。

部署/停止/重启沿用应用自身的短事务。Probe、观察、日志、等待和 Route sync 不增加外层事务。`orbit_provision_gateway` 当前只是读取并复用已有 Gateway，保留读取边界。actor、Project scope 和应用权限校验保留。

同步修正 MCP 文案漂移：创建 Application 返回初始 `unpublished` Version；发布后 Component 只读，Version 元数据仍可编辑，未被引用的 Version 可删除。

### PipelineRun 终态与重投递

初审诊断：Begin 成功后读取失败留下 running；第二次 Begin 返回 false，旧实现直接返回 nil，Repository 活动运行检查仍阻止后续触发。

现在整个执行（包括目标与工作区准备）使用配置的总超时。run/stage 终态写入使用脱离原取消、最长 5 秒的清理 context。读取错误和清理错误均继续传播；首次重复执行不会主动终止仍活跃的 run，终态记录也不会被覆盖。真实投递次数大于 1 且记录仍为 running 时，将其标记为中断失败，由用户重新触发，避免重放外部副作用。worker 租约大于执行预算的既有配置约束保持。

[execution_test.go](../../internal/application/pipeline_run/usecase/execution_test.go) 通过真实用例与事实端口覆盖初次/快照/最终读取失败、取消后的清理和清理错误传播；[execution_integration_test.go](../../internal/application/pipeline_run/usecase/execution_integration_test.go) 用真实 Begin/Complete SQL 验证清理不可用后重投递解除活动运行占用，并保留取消终态。

### 文件与证书边界

初审 Settings 诊断在真实 envfile 保存后注入 SQL Commit 失败，出现 HTTP 500 与文件已写入不一致。现在文件写路径不再依赖 SQL Begin/Commit；现有配置 revision、校验和恢复测试保留。

mkcert 外部阶段不持有请求事务；单条条件写入无需再引入应用 UoW。[certificate_integration_test.go](../../internal/application/route/usecase/certificate_integration_test.go) 使用生成器替身和真实条件 SQL，证明生成失败不保存、生成期间 Route 变化不保存旧证书且不覆盖新 domain。原有成功流程继续通过；没有调用真实 mkcert。

## 首轮验证记录（B1–B9）

在仓库根目录执行，默认不启用外部数据库、远端 SSH、Traefik 或特殊宿主资源联调条件：

```powershell
task sqlc
task check
go test -json -count=1 ./cmd/... ./internal/... ./sql
go list -deps ./internal/repository/impl/sqlc/service ./internal/queue/worker
git diff --check
```

| 阶段 | 顶层测试结果 | Package 结果 |
| --- | --- | --- |
| 初审首次全量 | 793 通过、13 跳过、1 失败 | 84 通过、1 失败、88 无测试文件 |
| 初审全量复跑 | 794 通过、13 跳过 | 85 通过、88 无测试文件 |
| 治理中首次全量 | 799 通过、13 跳过、1 失败 | 83 通过、1 失败、88 无测试文件 |
| 取消/晚退出编排拆分后全量 | 800 通过、13 跳过 | 84 通过、88 无测试文件 |
| 最终漂移修正后全量 | 800 通过、13 跳过、0 失败 | 84 通过、88 无测试文件 |

顶层测试数量不包含子测试，不代表覆盖率。无测试文件的 package skip 与用例主动 skip 分别统计；删除无生产用途的包导致通过 package 数减少。

B3 的首次初审失败为原 `TestCancellationTimeoutDoesNotBlockNewOperation`，单独连续 3 次复跑通过未被当作修复证据。迁移该用例后，治理全量又在 `TestNewOperationCompletesBeforeOldRuntimeExitsDespiteCancellationFailure/late_success` 报出 `context deadline exceeded`：真实 SQLite 清理仍受夹具的 200ms 预算限制。因此继续将晚退出与执行超时编排迁至虚拟时间端口测试；最后把剩余历史状态场景收敛为真实 SQL 生命周期验证，删除 SQLite 夹具中的短执行/取消预算。没有通过新增 skip、串行整个测试集或仅放大超时掩盖失败。

`task sqlc` 成功，仅更新 Route 查询生成物。最终 `task check` 通过，包含前端 typecheck/lint/format 及 Go 格式/lint，0 issues。传递依赖、变更文档相对引用与 `git diff --check` 检查通过。没有前端源码、依赖版本或已执行迁移变更。

原始 JSON 留在本机临时目录，未加入仓库：初审 `pomelo-orbit-backend-tests-20261014.jsonl`、`pomelo-orbit-backend-tests-retry-20261014.jsonl`；治理 `pomelo-orbit-backend-governance-20261014.jsonl`、`pomelo-orbit-backend-governance-final-20261014.jsonl`、`pomelo-orbit-backend-governance-final2-20261014.jsonl`。

13 个跳过用例对应 MySQL/PostgreSQL 联调、4 个 Windows OpenSSH 场景、4 个 Windows/Linux 远端 pipeline 场景、Traefik 重启、容器 Docker socket 和可写 bind mount。默认测试不证明这些外部环境已验证。

### 日志判断

拆分后与最终全量均有 6 条 WARN、5 条 ERROR 和 1 次 Gin panic recovery，均关联到主动注入的失败边界，没有全局静默日志：

| 用例/来源 | 预期原因 |
| --- | --- |
| `TestRouteSyncBoundsGatewayLockWait` | context 到期，验证网关锁等待超时和 deadline 传播；1 WARN。 |
| `TestSelectedPublicationReadsOnlySelectedRecordAndFiles` | 破坏发布记录，验证不能覆盖另一 Route 的目标文件；1 WARN。 |
| `TestUnchangedPublicationReloadsOnlyWhenLoadedConfigurationDiffers` | 注入 API 不可用，文件相同不能代替已加载配置观察；2 WARN。 |
| `TestHistoricalTargetRevisionAllowsExplicitRepublish`、`TestGatewayPublicationTracksCustomRuntimeDirectory` | 修改目标修订或 Gateway 目录，验证预览过期；各 1 WARN。 |
| `TestCurrentUserReturnsServiceUnavailableForUserLoadError` | 注入数据库不可用，断言 HTTP 503；1 ERROR。 |
| `TestBatchSyncContinuesAfterFailureInFrozenOrder`、`TestRouteSyncContinuesAfterItemTimeout` | 注入权限拒绝或单项超时，验证后续 Route 继续执行；各 1 ERROR。 |
| `TestRouteSyncPreservesSafeConfigurationFailureReason` | 注入带内部诊断的配置错误，客户端仅得到安全原因；1 ERROR。 |
| `TestRouteServiceReportsSavedStateWhenPublishFails` | 注入发布权限失败，验证已保存状态与发布状态区分；1 ERROR。 |
| `TestMiddlewarePanicDiscardsBufferedSuccessResponse` | handler 故意 panic，验证回滚和成功响应不泄漏；1 recovery。 |

## 漂移与交付

中间件移动、无用包删除、fixture 枚举和 MCP 描述已核对生产调用方、测试替身、活文档及 skill 引用。初审临时诊断测试已删除，由上述必要的长期验证替代；历史归档文档保留原记录。文档明确区分用例单测、协议测试和本地持久化集成验证。

本轮未启动开发服务器、未操作用户数据库或远端资源。最终按用户授权暂存全部变更，不创建提交。

## 后续治理：以测试暴露依赖和职责问题

用户进一步明确：测试清理是治理后端依赖和架构的入口，日常反馈不能长期依赖分钟级全量验证。本阶段在 B1–B9 之后继续治理，未将“测试通过”当作架构已全部收敛。

| 编号 | 当前问题与风险 | 治理与验证 |
| --- | --- | --- |
| B10 / P2 | 状态检查用例直接拼 Docker CLI、解析 JSON/Compose labels，测试跟随协议字符串，Runtime 端口太底层 | 已修正。新增 `ContainerInventory` 事实端口与 [Docker 适配器](../../internal/infrastructure/runner/docker/inventory.go)。适配器保留命令参数、精确 project/oneoff 解析和受限诊断输出；用例保留服务状态规则。 |
| B11 / P2 | 状态检查借用大型 Deployment Service 和接管 SQLite 夹具，再补写内部字段；命令、执行、状态快照装配耦合 | 已修正。独立 `StatusCheckService` 正常构造注入四项依赖，HTTP handler 与 bootstrap 明确装配；删除无调用的旧 `New` 构造器与未使用的 `store` 字段。现有命令/执行 Service 的其他职责没有宣称全部拆分完成。 |
| B12 / P3 | CSRF、旧密码、重复权限等简单规则仍启动完整迁移，Project 纯规则混在集成文件中 | 已修正。迁至真实用例 + 事实端口的单测，Project code 规则独立。真实登录历史、权限持久化、唯一约束等仍在完整验证。 |
| B13 / P2 | 日常核心规则与数据库、进程、协议混跑；部分端口测试用真实时间等待，接管测试将无需文件语义的 DB 落盘 | 已修正。新增核心单测入口；日志订阅和取消收尾使用虚拟时间。接管原子性保留真实 SQLite，改为独立内存库；状态条件写入迁至 Repository，用一次迁移和每场景独立事务验证实际 SQL。 |

### 入口与依赖证据

- `task test:go:unit -- -count=1`：application/model/common 的核心规则与用例，22 个已分类的集成/外部测试文件使用 `!unit` 条件。分类按实际连接边界，普通 `go test` 不排除它们。
- `task test:go:all -- -count=1`：完整 Go 范围 `./cmd/... ./internal/... ./sql`，继续执行本地数据库、协议和进程验证。原 `task test` 仍执行前端与完整 Go 测试。
- `go list -deps -test -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...`：核心测试依赖中无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。
- 状态检查单测直接调用真实构造入口和业务逻辑；替身只提供容器/数据库事实或记录写请求。条件更新的基线、活动部署保护由 [Repository 测试](../../internal/repository/impl/sqlc/service/status_check_test.go) 验证，未在假仓储重写 SQL 规则。
- 项目 skill 增补了“测试是治理入口”、成本拆分、虚拟时间及两种验证入口的原则。不能用缩小范围或 build tag 证明实现修复，因此本阶段同时检查核心入口和完整入口。

### 计时方法与结果

在当前 Windows 环境独立运行，使用 `-count=1` 排除测试结果缓存；保留 Go 已有编译缓存，不清理缓存或改工具链，不与 `task check` 同时运行。以下是本机实测，不作为跨机器 SLA。

| 范围 | 治理前 | 治理后 |
| --- | --- | --- |
| 完整 Go 范围墙钟 | 63.05 秒 | 55.88 秒；803 通过、13 跳过、0 失败 |
| 核心规则与用例墙钟 | 未有独立入口 | 11.25 秒；319 通过、无跳过或失败 |
| 完整运行中的 Deployment package | 44.68 秒 | 5.09 秒 |
| 接管三种操作的具体用例 | 8.31 秒 | 0.07 秒 |
| 容器日志跟随 / 权限撤销 / 取消后日志收尾 | 4.01 / 2.00 / 4.05 秒 | Go JSON 均记为 0.00 秒；虚拟时间仍验证原等待语义 |

全量顶层用例耗时之和从 64.66 秒降到 24.65 秒；核心用例之和为 0.22 秒。该和不等于墙钟或 CPU 时间，Go JSON 精度与多包调度也影响结果。完整测试仍有编译、启动和真实进程成本，没有宣称完整后端验证已降到几秒。完整运行有 85 个 package 通过、88 个无测试文件；新增 Docker 协议适配器的直接验证与迁移到 Repository 的条件更新验证使顶层用例从 800 增至 803。

最终 `task check` 通过（0 issues），两种测试入口、核心测试传递依赖与文档引用均检查通过。6 WARN、5 ERROR 和 1 次 recovery 与首轮的主动故障注入一致。完整检查没有使用 `unit` tag；真实持久化和协议验证均执行。

全量日志中的 package 时间含进程启动和初始化等成本。例如基线 HTTP auth 包首个测试前约等待 16 秒、断言为 0 秒；单独构建耗时 10.36 秒，执行空测试二进制耗时 0.86 秒，其中 Go inittrace 约 8ms。因此没有把这段等待错误归因为业务规则或数据库迁移，也未擅自更改机器安全软件、Go 并发参数或生产超时。

本阶段计时日志位于本机临时目录 `pomelo-orbit-backend-boundaries-{before,after,unit}.jsonl`。原有 13 个外部环境用例保持显式启用条件；没有新增 skip，也没有删除真实回滚或条件更新证据。

## 第三轮治理：运行挂载端口、执行职责与测试漂移

本轮承接继续治理授权，先按真实调用关系清点，再修改生产依赖和测试。保留真实数据库、协议和资源验证；没有新增 skip 或扩大 `!unit` 排除范围。

| 编号 | 发现与风险 | 本轮处理及状态 |
| --- | --- | --- |
| B14 / P2 | Deployment 目录归属用例直接拼 ps/inspect 命令并解码 Docker JSON，业务测试断言协议字符串；其他运行时工具也有同类边界问题 | 目录占用路径已修正：新增 `MountInventory` 与 `RunningBindMount`，由 Docker `Inventory` 返回 Source/Shared 事实。应用保留目录重叠、共享子目录、私有嵌套挂载及 Service 根目录保护。Docker 协议和查询失败在适配器验证。容器日志、通用 runtime tools 和 Compose 命令/渲染仍有技术协议落在应用层，属于剩余治理范围。 |
| B15 / P3 | `MaterializeLogicalMountSources` 和 `materializeFile` 没有生产调用，旧测试继续维护真实文件行为 | 已删除两个无用 helper 和单独的物化测试；混合测试只保留当前路径解析/物化声明规则。同步修正活跃 Route 文件发布计划，不改历史归档。真实 local/SSH writer 继续承担文件落盘。 |
| B16 / P2 | 两个 Route 规则和 envfile 编解码被集成标签排除；Gateway 探测校验依赖完整迁移 | 三个纯规则迁回无标签单测；Gateway 通过正常构造器、成员/环境/唯一性事实和事务替身验证探测准入。真实创建/回滚留在集成验证。ProvisionGateway 测试名及注释修正为复用现有未发布资源，删除已失效的“准备 stopped services”描述。已修正这些已确认案例，不宣称所有测试分类已穷尽。 |
| B17 / P2 | PipelineRun 命令和执行使用同一 Service，命令构造器注入执行器/密钥，测试可通过执行服务触发命令 | PipelineRun 已拆为 `Service` 和 `ExecutionService`，无业务 Service 嵌套或兼容别名。命令构造器从 17 项减至 12 项，移除凭据仓储、版本生成器、密钥、runner 和本地源码读取器；执行构造器移除 Project/Application 仓储。共享目标校验改为显式依赖的内部函数；命令日志/清理只接收文件资源，执行只保留所需日志 writer。共享仓库流程按两个正常构造器分别触发和执行。Deployment 命令与执行仍共用 Service，尚未完成同类拆分。 |
| B18 / P3 | Gateway 构造器仍接收从未使用的 Docker 路径解析器，端口和测试替身形成虚假依赖 | 已删除 Gateway 专用 `PhysicalPathResolver` 类型、构造参数、bootstrap 传参和无用测试 helper。实际 local runtime 和 repository source 的路径解析器继续承担其职责。 |

### 验证与计时

相关用例/适配器测试先验证行为，之后在 Windows 工作区独立计时两种范围，使用 `-count=1` 排除测试结果缓存，保留现有编译缓存。以下命令等价于两个 Task 的实际 Go 命令，额外使用 JSON 保存结果：

```powershell
go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...
go test -json -count=1 ./cmd/... ./internal/... ./sql
task check
```

| 范围 | 本轮结果 |
| --- | --- |
| 核心规则与用例 | 322 通过，0 失败/跳过；墙钟 12.23 秒，顶层用例耗时和 0.40 秒 |
| 完整 Go | 804 通过、13 跳过、0 失败；85 个 package 通过、88 个无测试文件；墙钟 66.20 秒，顶层用例耗时和 22.90 秒 |

相较上一轮核心 319、完整 803：四个已有案例进入核心入口，一个失效物化案例被删除，Docker 适配器增加两个顶层协议案例。按已核实的数据库口径，Gateway 校验迁出后从 130 个减至 129 个；这不是全集成/组件测试总数。

完整墙钟没有下降（上一轮 55.88 秒），不能将本轮依赖收敛宣称为全量提速。用例耗时和与编译、链接、进程启动、包调度共同影响的墙钟不是同一指标；本轮没有进一步归因墙钟增长，也没有改变 Go 并发参数、机器安全配置或生产超时。

完整运行仍有 6 WARN、5 ERROR 和一次 Gin recovery，均对应前述主动故障注入案例。外部环境 13 个 skip 沿用原条件，没有调用远端服务、用户数据库或启动开发服务器。原始日志：本机临时目录 `pomelo-orbit-backend-governance-round3-{unit,all}.jsonl`。

漂移复查覆盖已删除符号、构造调用方、执行接口、旧测试名称和活文档。特别删除了拆分后只被测试访问的执行资源 reader 字段；未为旧断言恢复已移除职责。项目检查及最终暂存状态以本轮收尾核验为准。

最终 task check 通过（0 issues），核心测试传递依赖未包含 Gin、SQLite、具体 Repository 或数据库迁移包；变更文档相对链接和 diff 空白检查通过。按既有授权暂存全部变更，不创建提交。

## 第四轮治理：Deployment 职责拆分与容器日志端口

本轮继续 B14、B17，以生产装配与依赖拆分为主，不修改测试分类、skip 条件或执行并发参数。

| 编号 | 本轮修正 | 当前状态 |
| --- | --- | --- |
| B17 / P2 | Deployment 拆为独立 `Service` 与 `ExecutionService`。worker 直接接收 `ExecutionStore`，移除 Project 权限仓储和未使用的 Gateway 协调器；命令端删除 Gateway 仓储、执行 store、readiness、挂载观察及执行超时字段。日志接口拆为读取/删除与 writer，命令 runtime 收窄为三个查询/路径方法。共享目标/Service 解析接收显式依赖，Compose 渲染成为函数。 | PipelineRun 与 Deployment 的命令/执行类型拆分均完成；这不代表所有运行时协议已归位。 |
| B14 / P2 | 新增 `ContainerLogs` 端口，Docker 适配器承担环境根目录容器 ID 查询、标签过滤/JSON 解码和 follow 命令。应用负责权限与目标复核、来源变化、等待、游标及重叠续读。 | 目录占用和 Web 容器日志命令边界已修正。通用 runtime tools/ApplicationStatus、取消后的状态观察、Compose up/down 命令及渲染仍含应用层协议处理，继续保留为未完成项。 |

### 测试与删除后的漂移

- 接管与容器日志 fixture 使用命令服务构造器；执行、目录占用和取消编排使用独立执行类型。共享流程分别创建命令和执行服务，不再通过同一对象注入两组依赖。
- 接管测试直接使用真实 Repository 验证生命周期条件写入，不再从命令服务访问执行 store。保留真实入队事务回滚、当前操作指针、旧结果保护与历史删除证据。
- 删除仓储手写转发层，改为既有 Repository 接口组合；删除未使用的 Gateway 查询端口、重复测试包装器与渲染测试中的空 Service。没有为旧测试恢复兼容别名或执行字段。
- 容器日志用例替身返回容器 ID、记录续读请求，断言等待、重建、权限撤销、目标变化和资源关闭；Docker CLI/JSON 断言放到适配器。保留无 Compose workspace 的停服日志读取语义和两秒重叠续读。
- 两个新增适配器顶层测试验证来源发现、查询/解码失败，以及环境根目录跟随命令、输出与错误传播；没有新增数据库或外部环境集成测试。既有持久化集成测试及其分类未删减。
- 同步更新后端架构、CD 运行时约定及项目 skill 的职责落点和代表测试，旧构造参数、执行字段和日志 helper 引用已复查。

### 验证与计时

先运行相关包测试，再完成 `task check`（0 issues），最后独立计时以下两种范围，使用 `-count=1` 排除测试结果缓存，保留现有编译缓存：

```powershell
go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...
go test -json -count=1 ./cmd/... ./internal/... ./sql
```

| 范围 | 顶层用例结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| 核心规则与用例 | 322 通过，0 失败/跳过 | 12.17 秒 | 0.27 秒 |
| 完整 Go | 806 通过、13 跳过、0 失败 | 70.50 秒 | 31.70 秒 |

完整运行 85 个 package 通过、88 个无测试文件。核心测试的传递依赖中无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。

全量墙钟较第三轮 66.20 秒增加，未实现全量提速；不能把职责拆分或秒级规则执行时间宣称为全量耗时已解决。耗时靠前的用例包括真实 Git 工作区验证和终端进程/资源验证；本轮没有据此删除它们，也未进一步断言全部等待成本的来源。

6 条 WARN、5 条 ERROR 和一次 Gin panic recovery 与前述故障注入用例一致，没有新增非预期警告。13 个外部环境 skip 保持原条件。原始日志位于本机临时目录 `pomelo-orbit-backend-governance-round4-{unit,all}.jsonl`。

最后检查文档相对链接、已删除符号、核心依赖和 diff 空白。按用户既有授权暂存全部变更，不创建提交。

## 第五轮治理：运行时观察、稳定性事实与网络归属

本轮继续 B14，将通用运行时查询和取消后的观察从 Docker 命令协议中拆出，没有更改测试入口、并发参数或 skip 条件。

### 实现与行为

- 新增 `RuntimeInspector` 与更窄的 `ContainerObserver`；Docker 适配器承担 ps/config/logs/inspect/probe/version 命令和 JSON 解码。命令侧只保留 `WorkspaceReader` 的目录存在/解析能力，执行侧使用四项能力的 `ExecutionRuntime`，不再接收查询、文件同步和日志跟随等无关方法。
- `ApplicationStatus`、Runtime 工具、取消观察及稳定性验证消费容器状态、健康度、重启次数和网络名事实。组件身份补充、归属校验、稳定性规则及取消条件回写仍在应用/Repository 边界。原始 inspect/ps 数据继续作为诊断 evidence 返回，响应形状不改为内部事实结构。
- 修复网络归属对象不一致：请求 `backend`、受管容器连接 `service-a_backend` 时，旧代码通过后者证明归属，却执行 `network inspect backend`，可能查询另一个无关网络。现在解析并查询已确认归属的 `service-a_backend`；完整名称匹配优先，未归属网络继续拒绝。用例测试断言实际查询名，适配器测试验证传递该名称。
- 将 `compose_ps.go` 及其四个协议测试迁到 `internal/infrastructure/runner/docker`；删除应用侧 ps/logs/config 命令 helper、`runRuntimeCommand`、无调用的 `outputOrError`，以及配置差异比较中未使用的 inspections 参数。
- 取消观察测试改用事实端口，删除临时目录和 JSON 夹具。收窄运行时后删除 workspace fake 的 Query/Stream/Sync 空实现及旧字段，部署独立性测试检查观察端口是否被调用。共享构造器与 HTTP/worker 装配同步更新。

定向测试最初发现共享执行夹具漏注入观察端口，取消收尾触发 nil panic；已修正实际夹具装配，随后定向、核心及全量测试均通过，没有增加 nil 兜底来掩盖缺失依赖。

### 验证与计时

新增四个应用顶层用例验证容器/网络归属、组件身份、探测准入和重启计数增长；稳定性等待用 `testing/synctest`。新增四个适配器顶层用例验证目标/目录/命令、诊断形状、结构化事实及命令/解码失败。协议测试从 application 迁出四项、业务用例新增四项，核心总数保持 322；全量新增八项。没有新增数据库或外部集成场景，既有真实持久化与进程证据保留。

相关包通过后，运行 `task check`（0 issues），再独立运行以下完整命令；不与 lint 同时计时，保留编译缓存，`-count=1` 排除测试结果缓存：

```powershell
go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...
go test -json -count=1 ./cmd/... ./internal/... ./sql
```

| 范围 | 顶层用例结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| 核心规则与用例 | 322 通过，0 失败/跳过 | 13.12 秒 | 0.19 秒 |
| 完整 Go | 814 通过、13 跳过、0 失败 | 55.73 秒 | 28.54 秒 |

完整运行 85 个 package 通过、88 个无测试文件。核心测试传递依赖仍无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。6 WARN、5 ERROR 和一次 Gin recovery 均对应先前列出的故障注入用例；外部环境 skip 条件未变。

本次完整墙钟低于第四轮 70.50 秒，但没有通过重复受控测量归因这次变化，不能据此宣称稳定提速。核心实际规则执行为亚秒级，完整验证仍承担编译、进程启动和真实资源验证成本。原始日志保存在本机临时目录 `pomelo-orbit-backend-governance-round5-{unit,all}.jsonl`。

### 剩余边界与收尾

B14 中已完成目录占用、容器日志、通用运行时查询及取消后观察的协议迁移。仍需治理 Compose up/down 命令、渲染和配置差异比较中的技术协议与业务规则混合；完整验证反馈成本仍需单独归因，未认定全部后端架构问题已消除。

后端架构与项目 skill 已更新；已删除符号、构造调用方、文档相对链接及 diff 空白检查通过。按既有授权暂存所有变更，不创建提交。

## 第六轮治理：部署生命周期与配置事实

本轮继续 B14，将部署/停止的 CLI 协议和配置解析移出应用用例，保留业务决策、执行身份、目标快照、取消与资源生命周期。未改变测试分类、并发参数或外部环境启用条件。

### 实现与测试漂移

- 新增 `ServiceLifecycle`，用例传递运行目标、目录、Compose project 和部署/停止选项；Docker 适配器生成 up/down 参数，借用调用方 context、session 与日志 writer，原样传播执行错误。拉取策略、Gateway 强制重建、移除卷与终态规则仍由应用负责。
- `LifecycleCommands` 为命令服务提供持久化诊断描述，与执行共用适配器参数构建；不解析命令描述执行。HTTP 和 worker 分别注入所需实现，命令侧无需获得执行能力。
- 准备端口收窄并更名为 `ExecutionWorkspace`，只保留目录解析、挂载路径及 workspace 准备。删除旧 CLI helper、无意义的 project 名恒等包装和 workspace fake 的通用 Run 空实现，拉取策略校验移至 `deployment_policy.go`。构造调用方和共享夹具同步迁移。
- `RuntimeInspector.ConfigurationServices` 解析 Compose YAML 并返回服务/镜像事实；应用直接比较事实并按服务名稳定输出差异。修正实际服务缺少 image 时漏报镜像差异的问题；缺少 services 段返回现有“无法解析”诊断，不再默认为空比较结果。对外诊断结构保持当前契约。
- 部署测试从断言 CLI 字符串改为断言保存、描述和执行使用同一组选项；取消和执行失败夹具改为注入生命周期能力。新增两个应用顶层用例验证配置差异及停止时保存的卷选项、目录和终态，新增两个适配器顶层用例验证生命周期命令/描述一致性、上下文和输出透传，以及配置解析。没有新增数据库、Docker、SSH 或跨层集成测试。

首次静态检查发现新增错误文本违反小写约定，已修正；定向测试、重新执行的 `task check` 和下述两个完整入口均通过。

### 验证与计时

`task check` 通过（0 issues），随后独立运行，保留编译缓存、使用 `-count=1` 排除测试结果缓存：

```powershell
go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...
go test -json -count=1 ./cmd/... ./internal/... ./sql
```

| 范围 | 顶层用例结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| 核心规则与用例 | 324 通过，0 失败/跳过 | 11.17 秒 | 0.22 秒 |
| 完整 Go | 818 通过、13 跳过、0 失败 | 50.01 秒 | 19.71 秒 |

完整运行 85 个 package 通过、88 个无测试文件。核心测试传递依赖无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。6 WARN、5 ERROR 和一次 Gin recovery 均对应首轮列出的故障注入用例；13 个 skip 沿用原启用条件。原始日志保存在本机临时目录 `pomelo-orbit-backend-governance-round6-{unit,all}.jsonl`。

完整墙钟低于上一轮，但未进行重复受控测量，不能将差值归因为稳定提速；日常核心规则执行仍为亚秒级，完整验证保留持久化、协议和进程成本。

### 剩余边界与收尾

B14 的生命周期命令和配置解析边界已完成；Compose 渲染、挂载/env 生成和 Gateway 静态配置改写仍混合业务规则与技术格式，尚未全部完成。后续应先区分规则输入与格式输出，不能将整套业务策略直接搬进 infrastructure。相关渲染测试是有效的 CPU 内存验证，不因代码仍待拆分就删除。

后端架构和项目 skill 边界参考同步更新，旧符号、调用方、文档引用和 diff 空白检查通过。按既有授权暂存全部变更，不创建提交。

## 第七轮治理：整体进度与真实超时边界

用户要求继续治理，同时能看到整体安排和进度。前几轮持续展开 B14，缺少各工作线的当前总览。本轮在本报告顶部增加持续更新的总进度、范围与验收顺序，文档索引提供直接入口，项目 skill 增加按整体风险和成本选取后续范围的要求。

全局复查读取上一轮计时、扫描 application/model/common 的生产依赖，以及 HTTP/MCP 入口和真实等待点。生产扫描未发现 application/model/common 直接依赖 infrastructure、API、bootstrap、SQLC、`database/sql`、`os/exec` 或 `net/http`；该扫描只证明直接依赖情况，不替代行为与生命周期审查。

| 编号 | 当前 / 风险 | 处理与验收 |
| --- | --- | --- |
| B19 / P2 | `RouteManager.WaitUntilReady` 只在请求返回后比较墙钟 deadline，单次请求仍用 3 秒预算；重试固定等待 500ms 后可能越过总预算再次探测，取消错误只保留上次请求诊断 | 改为总预算 context 贯穿目标解析、请求与重试，退出时保留 `context.DeadlineExceeded`/`context.Canceled` 和最后请求诊断。使用真实实现与窄 runtime 替身验证。 |
| B20 / P2 | `environment/terminal.go` HTTP handler 承担周期性成员/目标复核及空闲判断；`TestTerminalClosesAfterTargetChangesWithoutClientInput` 依赖真实 WebSocket 和 2 秒等待才验证规则 | 已确认，尚未实施。下一项迁移周期复核至应用能力，以虚拟时间验证规则；保留输入转发、控制消息和断开资源回收的必要协议/进程验证。不得简单将真实 socket 放进 synctest 或缩短生产间隔。 |

B19 的回归测试先在旧实现失败：750ms 总预算实际等待 3 秒；250ms 重试预算实际等待 500ms 并执行第二次请求；100ms 时调用方取消虽停止等待，但返回错误不支持 `errors.Is(err, context.Canceled)`。修复后分别在对应虚拟时间退出且均只发一次请求。原有“失败两次后成功”的验证改用虚拟时间，仍走实际重试路径，不删除场景或改变生产间隔。

同步修正共享 runtime 测试替身丢弃调用方 context 的漂移。新增一个适配器顶层单测，没有新增或删除集成验证，没有改 skip/tag/并发参数。终端、真实 shell/Git、持久化验证继续保留；集成测试必要性逐项复审和完整构建/启动成本归因按顶部安排继续，尚未完成。

验证：定向回归和 `task check` 通过（0 issues）；独立运行 `go test -json -count=1 ./cmd/... ./internal/... ./sql`，819 个顶层用例通过、13 个按原条件跳过、0 失败，85 个 package 通过、88 个无测试文件。墙钟 49.66 秒、顶层用例耗时和 28.18 秒；两项 readiness 验证均记为 0.00 秒。用例和高于上一轮，真实 Git/进程耗时有波动，不能将总墙钟的小幅差异宣称为稳定提速。本轮未改核心用例或其夹具，核心入口数据沿用第六轮，未重复运行。

日志仍为 6 WARN、5 ERROR 和一次 Gin recovery，均关联已有故障注入；新增回归在单独运行时产生的超时/不可用日志同样来自明确的注入。全量原始日志保存在本机临时目录 `pomelo-orbit-backend-governance-round7-all.jsonl`。文档链接和 diff 检查通过，全部变更暂存、不创建提交。下一项为 B20，Compose 仍在总安排中，不再以它作为连续治理的默认范围。

## 第八轮治理：终端周期规则与协议边界

本轮沿总安排处理 B20。`TerminalService.MonitorSession` 负责两秒周期的成员/目标/凭据复核和十五分钟空闲判断，复用现有 `ValidateSession`，每次校验仍有五秒期限。调用方取消时停止监测，不将正常取消误报为权限变化；应用不持有或关闭 WebSocket、PTY。

HTTP handler 通过消费方 `TerminalApplication` 接口注入现有应用服务，保留票据兑换、连接确认、输入前复核、消息编解码、输入/输出活动记录、会话关闭与日志。周期/空闲常量收为应用内部实现；连接、最长会话与写消息的传输期限仍由 handler 管理。生产装配继续使用原 `TerminalService`，没有兼容服务或测试专用执行分支。

规则验证迁入应用层虚拟时间：无客户端输入时发现目标变更、达到空闲期限、活动延长空闲期限、调用方取消。保留现有直接权限/目标/凭据校验测试。原本真实等待两秒的 HTTP 测试改为消费监测结果，验证 `access_or_target_changed` 与 `idle_timeout` 控制消息、正常关闭和底层 session 释放，不重演规则。真实本地 shell、WebSocket 收发、输入前权限撤销、连接取消和 SSH 指纹确认测试继续保留。

同步预查下一工作线的 Environment 集成场景：配置导入验证密钥加密落库、Probe 状态保存、local 不创建凭据和未初始化 SSH 绑定存为 NULL；目标警告验证跨 Project 查询与停用过滤。这些涉及真实存储和查询语义，保留持久化验证。该预查尚不代表其余 CRUD、Route、流水线及协议层复审完成；后续记录逐项处置，不以批量删除集成文件作为目标。

定向测试与 `task check` 通过（0 issues），随后分别独立运行两个入口，使用 `-count=1` 排除测试结果缓存：

| 范围 | 顶层结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| `go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...` | 325 通过，0 失败/跳过 | 10.59 秒 | 0.22 秒 |
| `go test -json -count=1 ./cmd/... ./internal/... ./sql` | 820 通过、13 跳过、0 失败 | 47.46 秒 | 20.61 秒 |

完整运行 85 个 package 通过、88 个无测试文件。终端周期规则与对应协议用例均记为 0.00 秒，核心测试传递依赖仍无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。6 WARN、5 ERROR 和一次 Gin recovery 对应原故障注入，没有新增非预期日志。原始日志为本机临时目录 `pomelo-orbit-backend-governance-round8-{unit,all}.jsonl`。总墙钟仍包含构建、启动及真实资源验证，本轮不据一次运行推导稳定提速幅度。

B20 完成，总览与架构/skill 参考同步更新，旧符号和调用方已复查。文档引用与 diff 检查后暂存所有变更，不创建提交。下一步按总安排逐项审查集成测试及其揭示的依赖问题；B14 与全量成本归因继续保留，整体治理目标未完成。

## 第九轮治理：Environment / Route 集成测试必要性

继续 B16 的测试分类与漂移复审。本批检查 Environment 的 6 个、Route 的 29 个原集成顶层案例，区分真实存储证据与仅借用数据库夹具的规则。Route 生产实现已有 Repository/外部端口，直接构造即可验证，本批不为了测试再增加生产接口或改动业务路径。

| 范围 | 处置 | 必要性与证据 |
| --- | --- | --- |
| Environment 初始化和配置导入，5 项 | 保留真实 SQLite | 初始化后读取、导入 SSH 凭据加密落库及 Probe 字段、交接保存、local 不创建凭据、未初始化 SSH binding 写为 NULL。`ReturnsLocalSeed` 重命名为实际的初始化后读取行为。 |
| Environment 目标共享警告，1 项 | 保留真实 SQLite | 验证跨 Project 查询、保存后的修订与目标，以及停用 Project 不再计入警告。不能由一个返回 bool 的假查询证明 SQL 语义。 |
| Route 锁预算、批量失败/超时后继续、安全错误映射，4 项 | 迁至核心单测 | 真实用例消费存储事实和发布端口替身；锁等待改用 `synctest`，无需迁移或真实 50ms 等待。 |
| Route 空发布目录、目标变更、选中范围、单项修订、运行时独立性，5 项 | 迁至核心单测 | 输入已存储 Route 与发布事实，执行真实预览/确认；其他 Route 的改变直接作为仓储事实，不额外执行创建/更新流程。运行时客户端增加调用计数，明确断言发布不查询运行时。 |
| Route DNS token 与 challenge 保存，1 项 | 迁至核心单测 | 真实准入规则与普通写请求记录，拒绝前不写；不需要数据库证明 Gateway token 检查。名称改为当前实际验证的 token 与 challenge 行为。 |
| 混合的证书保存/运行时视图案例 | 拆分，持久化部分保留 | 视图复制与错误映射单独成为核心单测；数据库案例只验证证书、Dashboard 与 Gateway 配置的保存读取。 |
| `ProjectSyncPreservesUnmanagedResources` | 删除重复且名实不符案例 | 给 runtime client 设置“非托管资源”，但被测预览/发布不消费该 client，原测试不能证明非托管资源保护；实际只重复显式发布流程，已由剩余保存/发布场景验证。 |
| `KeepsEnabledCustomTargetWhenAnotherRouteIsDisabled` | 删除重复且断言不足案例 | 未读取所谓被保护的 custom Route，只断言没有发布；与保留的 disable 持久化且不发布案例重复。 |
| Route 其余 17 项 | 保留真实 SQLite | 12 项验证 disable/rename、证书保存与显式发布、预览过期、保存成功但发布失败、HTTP/TCP 跨 Application/Version/Service 目标和 endpoint overlay；3 项验证逐项确认、数据库删除后的撤回、未发布草稿保存；另保留配置导入与 mkcert 条件写入各 1 项。 |

共享纯发布/客户端替身、预览确认 helper 和参数移入无 tag 的 `fixtures_test.go`，两种入口复用；新增事实夹具仅返回输入和记录普通写入，不模拟唯一约束、修订判断、提交/回滚。保留真实持久化断言的测试仍使用真实事务。生产构造器不变。

清理名称与 fixture 漂移：disable/rename 案例不再伪称验证非托管资源或已执行完整 sync，移除未参与调用的 client 数据；测试 Gateway 的 Application kind 改为当前 `gateway`。迁移后初次编译发现一个失去调用的 `ctx` 局部变量，删除后核心及普通 Route 定向测试均通过。

本批数量变化：Route 数据库顶层案例 29 → 17（10 迁移、2 删除）；混合案例拆出 1 个规则测试，核心入口净增 11 项，全量净减 1 项。Environment 6 项保留。没有新增 skip、改变 build tag 或删除持久化/回滚证据。

当前 application 中按 `*_integration_test.go` 命名的清单为 19 个文件、52 个顶层案例；其中本批保留的 Environment/Route 为 23 项，其余模块 29 项待继续复审。该口径不含其他文件名下的 Deployment/外部场景，也不含 Repository、HTTP/MCP、基础设施和进程验证，不能当作整个后端的集成测试总数。

`task check` 通过（0 issues）；定向核心/普通 Route 测试通过。随后独立运行以下两种入口，`-count=1` 排除测试结果缓存：

| 命令 | 顶层结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| `go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...` | 336 通过，0 失败/跳过 | 10.38 秒 | 0.22 秒 |
| `go test -json -count=1 ./cmd/... ./internal/... ./sql` | 819 通过、13 跳过、0 失败 | 50.44 秒 | 17.74 秒 |

85 个 package 通过、88 个无测试文件；核心测试依赖仍无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。6 WARN、5 ERROR 和一次 recovery 仍来自原故障注入；迁至核心入口的故障用例现在也会在核心运行中产生对应预期日志，不静默它们。完整墙钟未降低，本批不宣称全量提速。日志位于本机临时目录 `pomelo-orbit-backend-governance-round9-{unit,all}.jsonl`。

本批旧名称、共享 fixture、两种入口、文档引用和 diff 已核对，全部变更暂存而不提交。下一批是其余 CRUD/流水线 29 个命名集成案例及其他文件名下的实际集成场景；整体目标继续进行。
## 第十轮治理：其余应用层命名集成测试

继续按总安排检查其余 29 个顶层案例。本批迁移 1 项、合并 2 项，保留 26 项；与上一批 Environment/Route 的 23 项合计，目前 application 的 19 个命名集成文件保留 49 项。该数量仍不包含其他文件名、HTTP/MCP、Repository、基础设施及真实进程场景，不是全仓集成测试总数。

| 范围 | 本批处置 | 保留理由 / 调整依据 |
| --- | --- | --- |
| Auth 2、User 2、Role 1、Project 2 | 保留 7 项 SQLite | 密码/登录历史与 token hash 持久化、撤销/禁用、用户删除的关联清理且历史保留、角色权限关联与 FK 约束、Project 成员与唯一约束和停用查询。纯密码与权限规则此前已分离。 |
| Credential 4 | 迁出 1，保留 3 | Gitea 类型接受/未知类型拒绝改用真实构造器、成员事实和写入记录替身，继续执行真实加解密与导出，断言拒绝时不写。加密落库、唯一名称与引用保护、更新时间/搜索、历史非仓库类型的 SQL 过滤继续用真实 SQLite；清理未使用的 logger 构造。 |
| Gateway 5 | 合并 1，保留 4 | 命名案例与完整创建流程重复，将唯一的 name 断言并入创建资源束测试，统一从生产命名函数输入。保留创建/聚合查询、复用既有未发布 Service、profile/version/component 绑定、配置写失败的事务回滚。 |
| Settings 5 | 合并 1，保留 4 | 同值覆盖不需重启并入保存/刷新/重启/重置主流程。类型案例原本未断言非密钥类型结果，现补上保存后通过 config.Load 重新加载的实际值，验证字符串、密钥字面量、布尔、整数、列表和时长；拒绝小数整数后仍读取上次合法值。保留文件原子拒绝和 stale revision 保护。 |
| Pipeline 4、PipelineRun 2 | 保留 6 项 SQLite | 跨 Project 共享资源和权限查询、模板版本引用写入、删除前未终止 run 检查与历史保留、阶段变量清理、触发/执行后落库，以及清理失败后的重投递解除占用。修正普通 fixture 中过时的 RepositoryType `git` 为 `model.RepositoryTypeRemoteGit`。 |
| Application 1、Service 1 | 保留 2 项 SQLite | 导入/派生版本及组件身份、mount/env 的关系映射；删除运行中 Service 只删除配置且没有 runtime 操作。 |

Settings 当前消费 config.Runtime，是系统配置管理与实际解析链的连接点。Resolve 会读取 YAML/dotenv 并校验完整配置，不能把它当成单纯内存 DTO。本批保留文件集成验证，不增加仅为测试替换 Resolve 的接口或重新实现配置解析的 fake；同值判断复用主流程，避免重复加载整套夹具。Credential 直接复用现有 Repository 端口，无需生产新抽象。本批没有新增架构问题编号，也没有将既有 B14 标为完成。

整体后续仍为：其他文件名及协议/Repository/进程测试复审 → B14 剩余渲染边界 → 独立构建/启动成本归因及全局收口。命名文件复审完成只表示一个有界批次结束。

定向 Credential/Gateway/Settings/Pipeline 测试与 `task check` 通过（0 issues）。独立计时、使用 `-count=1`：

| 命令 | 顶层结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| `go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/...` | 337 通过，0 失败/跳过 | 11.00 秒 | 0.24 秒 |
| `go test -json -count=1 ./cmd/... ./internal/... ./sql` | 817 通过、13 跳过、0 失败 | 52.99 秒 | 23.03 秒 |

完整运行 85 个 package 通过、88 个无测试文件。核心传递依赖检查无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包。日志仍为 6 WARN、5 ERROR 和一次 recovery，逐项对应既有注入的超时、存储/发布失败、配置差异及 panic，不存在新增未解释警告。13 个外部场景保持原跳过条件。原始日志为本机临时目录 `pomelo-orbit-backend-governance-round10-{unit,all}.jsonl`。

本批核心净增 1，完整净减 2，来自迁移与合并，没有增删 tag/skip。完整墙钟仍在约 50 秒，不宣称全量提速；构建/启动归因仍是未完成工作。旧测试名称、常量共享、普通 fixture、文档链接与 diff 检查通过，全部变更按既有授权暂存，不创建提交。
## 第十一轮治理：Worker 依赖、Repository 夹具与剩余测试边界

本轮先沿实际边界复审其他文件名下的 Deployment 接管、队列、Repository、HTTP/MCP 与进程测试，发现并处理两项问题。

| 编号 | 当前 / 风险 | 治理结果 |
| --- | --- | --- |
| B21 / P2 | Worker 持有 `*sql.DB`，每次分派仅用于注入 DB context；成功/失败分派测试为此加载整库，掩盖了已有存储端口 | 移除 Worker 的 SQL 字段、构造参数与 context 注入，同步两个 bootstrap 入口。Repository 已持有构造注入的 DB，Task ClaimNext 自行管理短事务，执行应用使用已有事务端口。Worker 仅领取、调用 handler、记录结果；两项测试迁至端口事实/记录替身，补充 Complete/Fail 持久化失败传播。 |
| B22 / P2 | 普通 Application/Route 查询测试钉住旧迁移；Dialogue/Pipeline/PipelineRun 的 10 项测试复制局部业务 DDL，绕过当前必填字段、外键和种子数据，Dialogue 甚至保留生产已没有的 user 外键 | 全部改用当前迁移的独立内存 SQLite，补齐合法业务数据。Pipeline 只清理本测试库的种子模板以隔离列表断言；PipelineRun 使用实际必填字段、Artifact Project 和阶段所属 Pipeline。移除普通 Repository 测试中的 CREATE TABLE 与 MigrateTo。历史升级验证仍保留原版本。 |

Worker 新测试直接通过构造器和真实 Router 调用，验证 lifecycle context、task id/payload/投递次数、worker id/lease 传递，结果写入发生在 handler 之后，持久化失败继续传播原错误。替身不实现数据库领取、状态转换或重试算法。Task Repository 的领取/完成、失败终止、冻结重试预算 3 项真实 SQL 验证保留。核心入口扩展到 application/model/common/queue，增加既有队列协议映射及本轮纯测试共 16 项；完整入口范围不变。

切换当前迁移后的首次 Repository 验证有 7 项失败，原因是夹具缺失 pipeline.name、pipeline_run 必填字段、Artifact Project、stage 所属 Pipeline，以及全局列表混入真实种子模板；这些事实先前被手写 DDL 绕过。修正合法夹具与测试数据隔离后定向验证全部通过。未修改迁移文件或生产数据库，也未关闭外键来放行夹具。

### 本批必要性与处置

| 范围 | 保留/调整依据 |
| --- | --- |
| Deployment 接管 5 项 | 保留真实 SQLite/queue：失败准入和入队一起回滚、三种操作接管、过期结果/目录/取消不能覆盖当前操作、终态/缺失历史保留新生命周期。与已迁出的虚拟时间执行编排不同。 |
| Repository 34 项 | 保留真实 SQL：Application 4、Deployment 2、Dialogue 1、Environment 1、Pipeline 4、PipelineRun 5、Repository 3、Role 3、Route 1、Service 5、Task 3、User 2。分别验证参数绑定/分页/归属查询、历史引用/级联删除、快照映射、条件更新、显式调用方事务回滚。sqlcommon 另有 2 项纯错误转换，不计入此数。 |
| Role/User 测试命名 | 成功用例改名为关联映射保存/替换；失败用例明确为 `ParticipatesInCallerRollback`。现行 HTTP 写切面和仓储参与调用方事务的设计保留，不将外层回滚证据误报为仓储自身开事务，也不为其新增应用 UoW。 |
| HTTP 事务边界 | 3 项真实 SQLite probe 表验证成功提交、失败回滚及 panic；4 项 sqlmock 验证 begin/commit 失败及无需开事务的路径。probe 是通用事务测试数据，不复制业务 schema，保留。 |
| MCP 边界 | 保留 1 项真实应用/SQLite 集合写回滚，以及应用替身 + 内存 SDK 连接的工具契约、身份与 Project scope 验证。后者属于协议组件测试，不能替代实际持久化回滚。 |
| HTTP mapper、路由/鉴权、日志流与终端 | mapper 直接映射；Gin recorder 验证错误/路由/安全契约。真实 socket 用于日志 flush/断开取消和 WebSocket 收发/关闭，业务周期规则已分离；保留连接边界证据。 |
| Git Source | 删除 `TestSourceAcceptsRepositoryOutsideManagedWorkspace`：构造的 dataRoot 从未传入生产对象，实际只比较两个独立临时路径，重复有效仓库验证且再启动一套 Git 进程。保留真实 Git 修订解析/挂载路径和 linked worktree 拒绝两项。 |
| 本地终端、SSH 与显式远端流水线 | 本地终端保留 PTY 收发、进程树/句柄释放、Ctrl-C 等操作系统契约；SSH 连接与 SFTP 复用/恢复需真实本地协议。远端适配器验证 host key、日志/产物与取消清理，应用端到端验证 DAG/collector 编排，边界不同，保持原显式启用条件，不在默认检查中联调。 |

另修正两处普通 Deployment 历史 fixture 的 `succeeded` 为当前 `ran_to_completion`，删除 Service 搜索中已移除 instance 字段的空断言。应用命名集成 49 项仍保留；不能将其与不同层的数量相加后宣称全仓总数。基础设施文件/SSH/迁移场景尚需最终收口审计，B14 渲染边界与构建/启动成本也尚未完成。
最终验证：`task check` 通过（0 issues），定向 queue/bootstrap/Repository/Git 测试通过。随后独立运行、使用 `-count=1`：

| 命令 | 顶层结果 | 墙钟 | 用例耗时和 |
| --- | --- | --- | --- |
| `go test -json -count=1 -tags=unit ./internal/application/... ./internal/model/... ./internal/common/... ./internal/queue/...` | 353 通过，0 失败/跳过 | 13.13 秒 | 0.19 秒 |
| `go test -json -count=1 ./cmd/... ./internal/... ./sql` | 817 通过、13 跳过、0 失败 | 48.45 秒 | 26.17 秒 |

85 个 package 通过、88 个无测试文件。扩展后的核心测试依赖无 Gin、SQLite 驱动、具体 Repository 或数据库迁移包；Worker 的生产与测试传递依赖均无 `database/sql`。日志仍是 6 WARN、5 ERROR 和一次 recovery，逐项对应既有故障注入；没有新增 skip/tag，也没有禁用并发来改变结果。原始日志为本机临时目录 `pomelo-orbit-backend-governance-round11-{unit,all}.jsonl`。

本轮全量顶层总数不变（新增 1 项结果写入失败验证、删除 1 项无效 Git 范围测试），核心增加 16 项队列测试。墙钟及真实进程成本有波动，不据此次运行宣称稳定提速。架构文档、项目 skill 的当前入口与 Worker 边界已更新；旧构造调用、Repository 局部业务 DDL/历史版本钉住、相对链接及 diff 检查通过，全部变更暂存、不提交。下一项为 B14，最终还需成本归因与全局收口审计。

## 第十二轮治理：B14 收尾、资源断言与反馈成本归因

### 实现与测试

| 编号/范围 | 发现与处理 | 验收 |
| --- | --- | --- |
| B14 完成 | Compose 原先同时选择暴露/网络策略和编码 map/YAML；Gateway 在 hash 前和渲染中重复 enrichment。现在策略先返回结构化事实，挂载计划返回 source/target/readOnly 与落盘事实，env 占位符只在有效计划阶段处理。Gateway profile/token/受管挂载仍由用例选择，独立 render 包编辑 YAML；预览、命令、worker 在渲染前准备，渲染不再改写计划。 | 既有端口/路由/网络/挂载/美元字面值测试继续通过。新增共享声明所有权、失败不部分写入、预览准备及渲染不修改计划验证；修复 local IPv6 地址缺少方括号，并验证输出 `[::1]:8080:80`。 |
| B14 所有权 | enrichment 会修改调用方浅拷贝共享的 Mounts/Env；缺少 token 或 resolver 时可能已经改写部分值。 | 仅复制确实会修改的 Components/Mounts/Env 切片，准备完成后一次发布；失败保留输入。无须复制整个对象树或新增 DI 接口。 |
| B23 / P3 | 首次完整测试中 Windows 终端全进程句柄计数为 before=125、after=131，被旧断言认定为泄漏。该计数混入 Go runtime 和其他清理活动，不能证明终端责任。 | 改为检查该真实会话持有的 Process/Job 资源及输入/输出管道已关闭。Windows 可能复用句柄数值，因此查询具体对象类型；去掉 8 次重复启动和全进程容差。定向连续 5 次与最终完整运行通过。保留真实前台子进程终止、尾输出和断开释放测试。 |
| 迁移重复/漂移 | `DoesNotSeedDeploymentResources` 查询旧 Gateway ID 的不存在与全表为空断言完全重复；ULID 验证遍历多个不再有种子的业务表，断言空转。 | 删除重复顶层案例与失去用途的常量；ULID 验证只检查实际种子表，并纳入 Pipeline/Stage/Reference。历史迁移数据仍按升级目的保留，未修改任何迁移。 |
| 旧调用与夹具 | renderer 的 runtime env 参数、ignored bool、env 转接 helper 和仅测试调用的 wrappers 已无生产用途；Gateway fixture 错标 standard。 | 删除死路径，测试直接使用当前入口和结构化挂载事实，Gateway/普通应用夹具按真实 kind 区分；低层部署测试显式执行计划准备。 |

Compose 本身是应用输出，保留 usecase 中的输出投影；没有把业务策略搬到 infrastructure，也没有添加只有一个实现的 renderer port。格式节点编辑与业务选择分离后，B14 已达到本次职责治理边界，不继续以文件数量或目录纯度扩展工作。

### 基础设施保留依据

- 迁移相关 4 文件、19 项：其中 18 项执行 SQLite 升级/回退、种子或当前约束，1 项核对嵌入迁移清单。历史表/字段仅在验证转换时保留，不能按普通 fixture 漂移删除。PipelineRun target 的局部历史表用于单个增删列迁移，不是现行业务 Repository 的替代 schema。
- SSH session/terminal 6 项与 runtime 的 SFTP workspace 写入：验证真实本地 SSH/SFTP 的认证身份、连接复用、取消后恢复、断开/未消费输出释放和文件替换；纯命令编码、路径和失败回滚通过替身直接验证。没有增加真实等待；毫秒轮询只等待真实传输关闭。
- local storage 4 文件、16 项中：2 项真实 Git、1 项日志文件、1 项运行目录删除具有资源边界价值，其余为路径/映射规则（临时目录不等于完整集成）。Git 主流程与 linked worktree 拒绝保留；删除 run 目录不能波及源码，日志读取不能创建文件或丢追加数据。
- 原 application 命名集成 49 项、Deployment 接管 5 项、Repository 34 项维持既有保留理由。HTTP/MCP、SQL、文件、进程是不同边界，不能把包中所有测试都称作集成，也不能将这些局部清单相加冒充全仓统计。
- 全局复核应用生产 import、核心传递测试依赖、构造与渲染调用、旧 helper/fixture、真实等待及活文档引用。未发现新的跨层生产依赖；此结论限于扫描与已检查的代表调用链，不声称逐行审计全部后端。

### 独立计时与成本对照

不同时运行 lint 与计时测试；常规入口保持原并发和 skip 条件，`-count=1` 禁用测试结果缓存，未清空编译缓存。

| 范围 | 结果 | 墙钟 | 顶层用例耗时和 |
| --- | --- | --- | --- |
| 核心 application/model/common/queue（unit tag） | 357 通过、0 失败/跳过 | 12.85 秒 | 0.24 秒 |
| 首次完整 cmd/internal/sql | 819 通过、1 失败、13 跳过；暴露 B23 | 52.88 秒 | 不作为通过证据 |
| 修复后完整 cmd/internal/sql | 820 通过、0 失败、13 跳过 | 46.60 秒 | 19.94 秒 |
| 相同完整入口，加 `-run '^$'` | 85 个测试 package 启动成功，0 用例 | 46.66 秒 | 0 秒 |

最终完整运行 85 个 package 通过、89 个无测试文件；新增纯格式包带来一个无测试 package。核心传递依赖无 Gin、SQLite 驱动、具体 Repository、SQLC 或数据库迁移包。`task check` 通过、0 issues。日志仍为 6 WARN、5 ERROR 和 1 次 panic recovery，逐项属于注入的锁超时、发布/存储失败、配置差异及 recovery；没有新增未解释警告。

核心 JSON 显示 auth 首个用例前等待 11.133 秒，实际测试至收尾约 0.092 秒；environment 为 10.606/0.009 秒。这是 package 启动至首个 run 事件的间隔，不包含可单独归属的编译时间，不能把它当成业务测试耗时。

代表包 `go test -c` 构建到独立临时目录，再直接执行二进制：

| 包 | 构建/链接墙钟 | 新二进制空跑首次 | 再次空跑两次 | 直接执行测试 |
| --- | --- | --- | --- | --- |
| Auth（unit） | 0.828 秒 | 0.499 秒 | 0.018 / 0.026 秒 | 0.104 秒 |
| Deployment（unit） | 1.046 秒 | 0.675 秒 | 0.018 / 0.016 秒 | 0.062 秒 |
| Database | 1.830 秒 | 1.072 秒 | 0.018 / 0.016 秒 | 1.049 秒 |

构建时间使用已有编译缓存，包含 Go 命令开销，不是冷构建基准。空跑与全量墙钟相近、代表二进制首/复次启动差异明显，足以排除“全部约 50 秒都是业务规则/迁移执行”的解释；并发工作会重叠，不能直接相减计算某个系统组件的准确耗时。真实 Git 与终端仍贡献秒级测试工作，但目前不在全量反馈的主要关键路径。没有进程/文件扫描跟踪前，不将启动等待武断归因于 Defender 或其他特定程序。

原始日志位于本机临时目录 `pomelo-orbit-backend-governance-round12-{unit,all,all-final,empty}.jsonl`；代表构建实验位于 `pomelo-orbit-test-cost-42f8fe5a67124b75957881c09e3a51e3/results.json`。后续按顶部安排调查系统启动成本；本轮质量、行为、依赖及文档验证完成后暂存全部变更，不提交。
