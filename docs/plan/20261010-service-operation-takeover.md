# Service 容器操作与任务接替实施计划
最后修改时间: 2026-10-10 23:14:45

Review status: Accepted

Mode: standard

## Intent basis

依据已接受的 [Intent](../intent/20261010-service-operation-takeover.md)。范围为 CD Service 的部署、停止、重启，Gateway 复用其受管 Service；同一 Service 后成功受理的操作优先，取消旧任务失败最多 warning，执行超时由独立配置兜底。

不处理跨 Project 共用宿主，不按 Environment 或物理容器分组，不增加心跳、租约、重启恢复或 CI 接替。继续使用当前数据库队列和 local/SSH runtime。

## Design decisions

### 当前操作归属

- Service 新增内部字段 `current_deployment_id`，表示最近成功受理的生命周期操作。它是持久化的操作资格，不代表进程存活，不加入 Compose 配置 hash，也不增加前端准入字段。
- 请求通过既有权限、输入、目标修订校验后，在现有请求事务中创建 Deployment、更新当前指针并入队；提交成功才生效。创建或入队失败回滚整个事务，旧任务仍有效。
- 同一 Service 的指针写入确定受理顺序；并发请求按数据库对该 Service 更新的顺序确定后置操作，不按时间戳、ULID 或 Worker 领取顺序猜测。
- 当前任务完成、失败或取消后保留指针，旧任务不能重新取得执行资格。指针沿用项目逻辑关联，不以历史 Deployment 的存在性决定资格，不因清理历史而恢复旧任务。
- 当前指针的可靠写入属于新任务受理的必要条件；旧任务取消记录更新与 runtime 退出属于尽力步骤，失败可容错。两者不能混为一个可忽略错误的事务。

### 接替与取消

- 沿用每秒以内的 `deploymentExecutionContext` 监控，增加 Service 当前指针校验。旧运行任务发现指针变化即取消自己的 Context，即使其 Deployment 取消状态更新失败也能响应接替。
- 新 Worker 开始操作前检查自己仍为当前任务。若已被更新的操作接替，幂等退出，不准备文件、不执行 Docker。
- 新任务在请求事务之外，对同一 Service 其他仍处于 waiting/running 的旧 Deployment 尝试标记取消。每次更新保留状态条件及当前归属条件，避免 B 的迟到取消步骤取消已接替 B 的 C。
- 已完成、失败、取消或不存在的旧任务视为无需再取消；不重写其原结果。查找旧记录、取消写入、退出确认失败或超时只记录 warning，新任务继续。
- 不等待旧任务正常完成、释放业务锁或确认远端进程退出。无需新增跨 HTTP/Worker 的进程内取消注册表；数据库当前指针和现有监控提供通知。
- 预期接替、旧任务取消异常以正常处理结果返回 CD handler，避免通用 Worker 将这些容错情况再升级为 ERROR。新任务自身实际执行失败仍正常报告失败。

### 配置化时间边界

计划新增 `DeploymentConfig`，通过现有 typed config 加载、校验、设置与 bootstrap 注入：

| 配置 | 计划基准值 | 用途 |
| --- | --- | --- |
| `deployment.execution_timeout` | `1h` | 从 CD Worker 开始处理起限制总执行时间，包含准备、命令与 Gateway readiness；不包含排队时间 |
| `deployment.cancel_timeout` | `5s` | 限制尽力取消的查询、写入和收尾等待；不要求新任务等待这段时间确认旧任务退出 |

- 两个 duration 必须为正数，在集中配置入口校验，不在业务代码读取环境变量或提供隐藏默认值。数值为本 Plan 的实施选择，可由配置调整。
- 不复用 CI 的 `pipeline_run.execution_timeout`，不改通用 Worker 的参数、租约或重试策略。
- CD 执行外层以结果、取消信号和 deadline 竞争结束等待。发现接替或达到超时后取消 Context，并在有限收尾时间内返回；不无限等待不响应取消的 runtime，从而释放当前 Worker slot。
- 内层迟到返回不能再次完成任务或更新 Service。被接替任务保留 canceled/接替原因；当前任务因自身执行超时变为 faulted，并记录阶段、配置时限和 deadline 原因。沿用现有状态枚举。
- local 沿用 `exec.CommandContext`，SSH 沿用 Context 关闭连接；停止未确认时仅 warning，不宣称进程或 daemon 动作必然已停止。资源的关闭归属与日志 writer 生命周期需避免和迟到执行并发冲突。

### 回写与外部步骤

- Service 运行目录绑定、部署版本与状态、停止结果、Gateway readiness 失败和取消后状态观测，统一使用带 `current_deployment_id = 本任务 ID` 的条件更新。
- 执行结果的 Service 更新同时要求本任务仍处于可执行状态，防止当前任务已因超时或手动取消结束后，自己的迟到结果继续改写 Service；取消后观测使用单独的受限条件更新。
- Deployment 的启动和成功/失败完成同时检查当前指针及原有状态条件；被接替任务只允许按原有非终态条件收敛为取消，不能被迟到成功或失败改写。
- 条件更新返回是否生效；零行表示已失去资格，按接替退出。数据库错误不能伪称正常成功；结果提交不占用外部 I/O 的长事务。
- 执行开始、发布共享配置前、绑定目录前、发出 Compose 命令前及后续 readiness 阶段检查 Context 和当前资格。数据库回写用 SQL 条件避免检查与写入之间的竞态。
- local/SSH `StageWorkspace` 在获取既有短文件锁后以及逐项写文件、发布最终 Compose 前检查 Context。锁等待响应取消，不能让等待旧文件准备重新形成整次部署串行。只复用短文件发布互斥，不新增覆盖镜像拉取或 Docker 命令的 Service 锁。
- 已发出的本地文件、SFTP 或 Docker 动作属于尽力取消边界；本任务不引入文件事务、宿主代理或物理目标锁来承诺绝对撤销。

## Implementation steps

### 1. 增加 Service 当前操作持久化与条件更新

- 新增 SQLite/MySQL/PostgreSQL 对应的下一编号迁移，增加 nullable `current_deployment_id`；不修改已执行迁移，不重建旧任务执行资格，也不引入恢复流程。
- 更新 Service 模型、列表投影和 Repository 映射。字段保持内部使用，无需为了接替新增 Proto/API 公开字段；配置 hash 继续只包含实际部署输入。
- 在窄 Repository/CommandStore/ExecutionStore 端口增加当前指针写入与条件更新方法；部署用例不直接依赖 SQLC 类型。
- SQLC 查询保护 Service 的状态/版本/目录绑定及 Deployment 的开始/完成；取消查询只更新同 Service 的过期非终态任务。沿用命名参数、现有事务上下文与 Repository 边界，执行 `task sqlc` 更新生成物。

### 2. 将请求准入改为有效新操作接替

- 从部署、停止、重启移除 `ensureNoActiveDeployment` 与库存运行状态准入检查；保留成员权限、Service/Application 归属、目标修订、必要参数、Gateway 专属版本选择与显式删卷选项。
- 三类操作共用最小的“创建任务、更新当前指针、入队”步骤，纳入已有 HTTP 写事务；核对 MCP 同类命令使用相同事务装配，避免成功改指针却未写入队列。
- HTTP 受理只做必要持久化，不在事务内等待旧 Worker、连接 SSH、运行 Docker 或做尽力取消。
- 新操作成功受理后，旧执行监控通过指针变化响应；新 Worker 领取后在独立有界操作中补充取消旧任务记录。提交期间未通过校验或事务失败的请求不接替旧任务。

### 3. 接入超时、尽力取消与迟到执行保护

- 在 `config.go`、配置 YAML 和配置测试中增加两项 CD duration，注入 `NewExecutionService` 与需要该边界的 CD handler；单独 Worker 和同进程 Worker 使用相同装配。
- 收敛部署、停止、重启的执行外层，所有阶段使用受监控且带 deadline 的 Context；当前部分阶段仍使用原始 `ctx`，一并调整。
- 以数据库当前归属、手动取消和 deadline 为取消来源。执行外层独立限制等待与收尾，处理不响应 Context 的 fake/runtime 返回；不改通用队列执行协议。
- 处理旧任务已终态、缺失、取消记录写入失败以及 runtime 退出未确认，确保新任务继续且日志等级最多 warning。
- 通过条件写入保护目录、版本、状态和完成结果；被接替取消不运行会覆盖新状态的旧 `reconcileCanceledService`。仍为当前任务的手动取消可在有界观测后条件更新。
- 收尾只操作任务自身 Context、连接和日志，不清理新任务文件、不发补偿性 `down`、不自动删除卷；迟到执行结果幂等丢弃。

### 4. 补齐共享配置准备的取消检查

- 在 local 与 SSH 文件准备中补上锁等待和逐项操作的 Context 检查；保留既有路径范围、符号链接、挂载映射和原子替换实现。
- 在调用 `StageWorkspace` 与 Compose 前校验本任务仍为当前操作；失去资格的任务不能主动进入下一阶段。
- 不扩展为新工作区布局、文件发布协议或跨进程租约，不让整个部署等待前一次部署结束。记录已发出写入与 daemon 命令无法绝对取消的风险。

### 5. 按实际依赖调整生命周期操作的目录定位

- 已有且属于当前目标修订的 `runtime_directory` 继续优先，重启和停止不因为用户刚确认新部署目录而隐式切换已绑定位置。
- 未记录运行目录时，生命周期操作可以使用当前目标修订下已明确确认的 `deployment_directory`，冻结到本次 Deployment，Worker 再检查所需目录及 Compose 文件；不根据库存状态推断容器存在与否。
- 没有可确定目录、目标修订不匹配或 Compose 实际缺失时，报告具体依赖及可操作的安全摘要；不猜旧环境位置，不增加宿主扫描，也不统一提示必须先部署。
- 只调整生命周期操作所需定位；其他确实依赖运行目录的查询准确报告自己的缺失，不借此新增运行时恢复机制。

### 6. 收敛前端操作限制

- ServicePage 卡片/表格、ServiceDetail、GatewayDetail 移除依据 `active_deployment` 或运行状态禁止部署、停止的逻辑，点击处理器同步调整；有重启入口时应用同一规则。
- 保留短 HTTP 请求期间的 busy、数据加载、权限、目录表单校验和 Gateway 必须从网关入口部署等既有规则；请求受理后恢复操作按钮，允许再提交。
- `active_deployment` 只用于任务展示，按当前指针对应 Deployment 是否非终态计算，避免取消记录更新失败的旧任务继续冒充当前任务；保留原字段和展示能力。
- 使用既有组件和中英文文案，不增加等待旧任务取消的确认弹窗或新的操作锁。

### 7. 收尾错误诊断并同步活文档

- 核对当前未提交的 logging/transport/目录错误增量，移除 `runtime_query.go` 中已失去用途的 import，完成此前未验证的改动。
- 真正的配置/依赖错误按既有错误分类返回安全原因与 requestId；原停止请求不能再被错误归为容器日志配置问题。5xx 保持安全摘要，内部 cause 在日志保留。
- 取消异常日志带 `project_id`、`service_id`、前后 `deployment_id`、阶段、耗时与 cause；记录取消请求和退出未确认，不用 canceled 状态宣称物理停止。
- 更新 CD 产品模型、运行时、部署指南与相关决策，说明库存状态、后置优先、Service 粒度、配置超时和外部副作用边界；保留此前终端全屏文档增量。
- 运行实现必需的固定检查与本次最小有效测试；实现结束汇报实际 diff 和风险，按标准模式停在 Implementation，等待用户要求进入 Verification。

## Files to change

| 范围 | 预期文件 |
| --- | --- |
| 持久化 | `sql/migration/{sqlite,mysql,postgres}/<next>_service_current_deployment.{up,down}.sql`、`sql/query/{service,deployment}/`、对应 SQLC 生成物、`internal/model/service.go`、`internal/repository/{service,deployment}.go`、`internal/repository/impl/sqlc/{service,deployment}/` |
| CD 用例与 worker | `internal/application/deployment/port/port.go`、`usecase/{command,service,stores,deployment_execution,service_location}.go`、必要的 CD worker handler 与新建的小型执行协调文件；相关测试 |
| 超时配置与装配 | `internal/config/{config.go,config_test.go}`、`configs/config.yaml`、必要的 profile 与配置测试 fixture、`internal/bootstrap/{http,worker}.go`；必要的 MCP 事务装配 |
| 目标运行时 | `internal/infrastructure/runner/{local,ssh}/runtime.go` 及现有文件锁辅助实现、相关单元测试 |
| 当前任务展示 | `internal/application/service/usecase/service.go`、Gateway 当前任务投影及 `HasActiveDeployment` 查询实现；现有响应契约无需新增字段 |
| 前端 | `web/src/views/service/{ServicePage,ServiceDetail}.vue`、`web/src/views/gateway/GatewayDetail.vue`、相关组件/工具测试及必要的中英文文案 |
| 错误诊断 | 当前已修改的 `internal/api/http/{middleware/logging*,transport/error.go}`、`internal/application/deployment/usecase/{runtime_query.go,deployment_directory_test.go}` |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`、`docs/decisions/ledger.md` 中直接相关条目 |

不计划修改 `internal/queue/worker`、Task 领取/租约查询、CI 用例或 CI 配置；终端全屏文件不是本任务新增变更。

## Verification plan

1. 请求用例与真实 SQLite Repository 测试验证同一 Service 的 A -> B -> C 受理和指针更新、部署/停止/重启互相接替、不同 Service 独立，以及入队失败回滚后 A 保持资格。不靠内存 fake 证明 SQL 条件原子性。
2. 使用可控 channel 的 fake runtime，让 A 停留在拉取/命令阶段，在 A 未正常结束时接受并开始 B；验证 B 无需等待 A 退出确认，且 B 被 C 接替后不能取消 C。
3. 旧任务已失败或完成、记录不存在、取消写入失败时，验证新任务仍执行、旧失败原因保留，异常只产生 warning。监控指针变化应独立于旧 Deployment 是否成功写为 canceled。
4. 使用短配置 duration 验证总执行超时覆盖准备、命令和 Gateway 等待；用不响应取消但可由测试最终释放的 fake，验证外层在有界时间返回、Worker slot 可释放，迟到成功/失败不改结果。测试结束释放所有 fake，避免遗留 goroutine。
5. 让 A 在 B 开始或结束后返回，验证目录绑定、Service 状态/版本、readiness 失败、取消后观测和 Deployment 完成的 SQL 条件均阻止旧写入；指针不会因 B 完成或历史清理恢复为 A。当前任务自身已超时/取消后，其迟到返回也不能写状态。
6. local/SSH runtime 单元测试验证 Context 已取消时不进入文件发布、等待短文件锁可退出、部分文件准备后取消不继续写 Compose，以及命令使用本任务 Context；沿用 Windows SSH 引号和 DooD 映射测试，不新增真实宿主测试依赖。
7. 目录测试验证运行目录优先、无运行记录但当前修订确认目录可冻结执行、确实缺失输入的准确错误；库存停止/运行均不构成额外拦截。保留目标修订与显式删卷行为。
8. 前端组件或现有共享操作 helper 测试验证 stopped/running/faulted 及活动任务下仍可操作、Gateway 专属入口限制保留、提交受理后可再次提交。检查卡片、表格、详情与 handler 条件一致。
9. HTTP 错误与日志测试验证原始停止失败的分类、可理解的安全摘要、requestId/cause 关联，以及接替取消 warning 不被额外升级为 ERROR。
10. 实现固定检查：`task sqlc`、`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，加本次相关 Vitest。仅在实际改变 Proto 契约时才运行 `task proto`。
11. Verification 阶段核对 Intent/Plan 与实际 diff、验收清单及未验证边界。本 Plan 阶段仅核对文档，不运行实现检查，不启动、停止或重启开发服务器。

## Blockers

无需要用户补充信息的阻塞。当前取消 runtime 和文件锁实现的细节需在 Implementation 按上述边界收敛，不扩展为存活检测或恢复工程。

## Assumptions

- “后置”指同一 Service 有效操作成功受理的顺序，校验失败的请求不改变当前任务。
- `execution_timeout=1h`、`cancel_timeout=5s` 是计划基准，运行时通过统一配置调整，保存配置沿用已有重启生效规则。
- 服务级当前指针是本期唯一接替依据；环境修订仍用于目标安全检查，不参与扩大取消范围。
- 新增字段不恢复升级前的活动执行；部署升级沿用现有指南处理活动任务后重新提交，不编写启动恢复分支。
- 前次终端全屏与错误诊断增量已经存在，本期只收尾直接相关诊断问题，验证结果需区分已有增量与本次新增行为。

## Risks

- 停止 Docker CLI 或关闭 SSH 连接可能无法撤销 daemon 已接受的操作；后置优先是 Orbit 操作资格和状态回写保证，现场执行可能仍有交叠。
- 本地文件或远端 I/O 已进入系统调用时不能绝对撤销；归属与 Context 检查阻止后续编排，有限文件锁只保护既有发布过程，不能保证任意卡死写入都已消失。
- 外层超时返回无法强杀不响应 Context 的 goroutine。需要有界等待、迟到写入保护和准确 warning；不将此限制扩展为进程监督系统。
- 当前操作持久化或入队本身失败时，新任务无法可靠受理，仍应返回真实错误；“取消失败可容错”不等于忽略新任务的持久化失败。
- 多 Project 共用物理目标或用户带外操作造成的资源冲突，由实际命令反馈；本任务不额外识别或协调。
- 单元测试能证明取消传播、时间边界和条件回写，不能证明真实 Docker/SSH 副作用已撤销；交付需保留这个验证边界。

## Rollback

- 代码回退前处理本期已受理操作，保留任务和目录记录；不自动撤回容器动作、删卷或删除远端文件。
- 新迁移提供结构回退脚本，但执行 down migration 需另行授权，并核对活动任务及数据库备份；本轮不执行迁移或 Git 写操作。
- 回退产品改动不得顺带撤销终端全屏或其他用户已有工作区变更。

## User review notes

- 用户要求开始计划，Intent 已标记 Accepted；随后明确要求开始实现，本 Plan 标记 Accepted，进入 Implementation。
- 已落实用户明确边界：Service 粒度、后置优先、取消失败 warning、配置超时；排除跨 Project 宿主协调、心跳、租约、重启恢复和 CI。
- 已按本 Plan 实现内部当前任务指针、数据库条件更新与两项 duration；实现范围没有新增业务边界或需要用户提供信息的阻塞。
- 用户明确反馈“实测有效”，要求开始 Verification；验收依据与结果记录在同名验证文档中。

## Implementation notes

实现阶段已完成，Review status 保持 Accepted。用户确认实测有效后进入 Verification / 验证，结果见 [验证文档](../verification/20261010-service-operation-takeover.md)；未执行暂存、提交或推送。

### 实际改动

- 新增三种数据库的 000053 迁移，持久化 Service 当前任务指针；更新模型、窄端口、SQLC 查询与生成物。共享 schema 使其他 SQLC 包的 Service 模型同步生成，不涉及 CI 用例或队列协议变更。
- 三类生命周期命令移除库存状态与旧活动任务准入限制；创建任务、设置指针和入队共用事务，HTTP 与 MCP 使用一致装配。入队失败保留旧操作资格。
- 执行外层以结果、接替/取消和配置 deadline 结束等待，旧记录取消与收尾均限时；运行目录、状态/版本与任务完成通过条件写入保护，最终结果在短事务内提交。
- local/SSH 文件锁等待及发布过程检查 Context；生命周期目录定位优先运行目录，无运行记录时允许当前修订已确认目录。Service 列表、卡片、详情与 Gateway 允许活动任务期间再次提交，保留 Gateway 专属部署入口。
- 错误诊断保留安全响应与内部 cause；取消 warning 包含任务、Service、阶段和时限，超时结果包含执行阶段。同步产品模型、运行时、部署指南与直接相关决策。
- 保留工作区已有的终端全屏改动；本次检查包含该组件回归测试，没有扩展终端行为。

### 实现检查

| 检查 | 结果 |
| --- | --- |
| `task sqlc` | 通过，命名参数与生成绑定已检查 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| Service、Gateway、终端抽屉相关 Vitest | 3 文件、26 测试通过 |
| `task check` | 通过；前端格式、lint、类型与 Go 格式/lint，0 issues |
| `go test ./cmd/... ./internal/...` | 通过 |
| 额外 Go `-race` | 无法运行：当前 CGO 关闭，环境无可用 gcc/clang；未改变工具链 |

真实 SQLite 测试覆盖部署/重启/停止互相接替、入队失败整体回滚、旧 runtime 未退出时新任务完成、取消写入失败与取消查询卡住、总执行超时、迟到成功/失败、条件回写、终态/缺失旧任务以及不同 Service 隔离。runtime 测试覆盖持有文件锁时取消不继续发布或连接；前端覆盖库存状态与活动任务下的操作入口和重复提交。

### 未验证边界与风险

- 未操作真实 Docker/SSH 目标，不能据单元测试宣称 daemon 已撤销动作、卡死 goroutine 已停止或物理副作用完全有序。
- SQLite 执行了完整迁移与真实条件查询；MySQL 新迁移及参数经过代码/生成检查，未连接真实实例运行。PostgreSQL 已连接开发实例诊断下述问题并确认修正后的类型转换，未运行真实 PostgreSQL 的完整操作流程。
- 未启动、停止或重启开发服务器，未执行实际环境数据库升级；部署时须按指南处理升级前活动任务。
- 正式验收与用户实测结论见同名验证文档；实现范围仍排除 CI、心跳、租约、重启恢复和跨 Project 宿主协调。

### PostgreSQL 指针截断修正

- 实际停止记录 `01M4K2PC5Y02P597GFH7ERRTXX` 被误判为已接替：PostgreSQL 将无长度的 `CAST(... AS CHAR)` 解释为 `CHAR(1)`，写入的 `current_deployment_id` 只有 `0`，与完整任务 ID 不相等，Worker 未执行停止命令便将记录收敛为 canceled。
- 将当前指针赋值、运行目录绑定、执行结果回写与取消后观测四处转换统一改为 `CHAR(26)`，重新执行 `task sqlc`；生成参数类型与 Repository 绑定保持一致，无需修改迁移。
- PostgreSQL 只读查询确认 `CHAR(26)` 返回完整任务 ID；诊断当时该 Service 的存量指针为 `0`，随后新停止记录 `01M4K4C7TF63M53ZEMHQM6EJWG` 成功完成，当前指针已写入该完整 ID。历史 canceled 记录保持原状；代理未重放旧停止任务、改写环境数据或操作容器。
- 按用户要求不新增此类回归测试。修正后 `task sqlc`、`task check` 与 `go test ./cmd/... ./internal/...` 均通过；本次 Verification 已收录这些检查结果。

### Compose 环境变量字面值修正

- 实际停止日志揭示环境变量值中的 `$` 被 Compose 当作插值，产生未设置变量警告并改变密码。YAML 字符串引号不能阻止 Compose 插值。
- 在 Compose 渲染边界将合并后的环境变量值中每个 `$` 转义为 `$$`；Service 占位符继续先解析，数据库、页面及有效计划中的原值不变，用户填写的连续美元符号按字面值保留。
- 使用临时程序调用实际有效计划合并与渲染器，再以 `docker compose config --hash` 与原始字面值的 `--no-interpolate` 配置比较：8 组虚构值涵盖单个 `$`、`${...}`、连续 `$$`、引号、反斜杠、多行、空值及共享环境变量解析，配置 hash 一致且无警告，重复渲染稳定。临时程序已删除，不新增持久回归测试，不启动或停止容器。
- `task check` 与 `go test ./cmd/... ./internal/...` 通过，仍处于标准模式的 Implementation 阶段。旧 Compose 文件由后续部署或重启重新生成；已初始化的 MySQL 账号密码不会因此自动修改。
