# Service 容器操作与任务接替验证
最后修改时间: 2026-10-10 23:11:11

Review status: Accepted

## Basis / 依据

- 依据已接受的 [Intent](../intent/20261010-service-operation-takeover.md) 与 [Plan](../plan/20261010-service-operation-takeover.md)，核对 Service 部署、停止、重启及 Gateway 受管 Service 的整体交付。
- 用户明确反馈“实测有效”，并要求开始验证文档，作为本次人工验收结论。本文记录已知实测与检查，不从这句话推断全部平台、异常场景或数据库认证均逐项实测。
- 当前为 Verification / 验证。沿用已有意图与计划，按当前 SpecFlow 文档契约形成报告；已有模式字段作为历史记录保留，不新建 Spec 或 task。
- 本阶段收集已执行的开发检查并核对当前 diff，没有新增回归测试、重新执行容器操作或管理开发服务器，也没有修改用户暂存区。

## Intent alignment / 意图对齐

部署、停止和重启已移除库存状态及旧活动任务的准入限制；前端与后端遵循同一规则。权限、归属、目标修订、目录和 Gateway 专属部署入口等必要校验继续生效。

接替以 Service 为粒度。成功受理的新操作在事务内创建 Deployment、写入 `current_deployment_id` 并入队；事务失败保留旧资格。Worker 只开始当前任务，通过指针变化取消旧 Context，旧任务取消失败或退出未确认最多 warning，新任务不等待旧 runtime 正常结束。

执行与收尾分别使用集中配置中的正数时限。Service 目录、状态、版本及 Deployment 开始和完成均受当前归属及任务状态保护；迟到结果不能覆盖后续操作。无运行目录记录时，停止和重启可以使用当前目标修订下已确认的目录。

边界与意图一致：不扩展 CI、心跳、租约、重启恢复、Environment 分组或跨 Project 宿主协调；不自动回滚 Docker 动作、删卷或搬迁数据。

## Plan alignment / 计划对齐

| 计划步骤 | 实际交付与核对 |
| --- | --- |
| 当前操作持久化 | 新增三种数据库的 000053 迁移；模型、Repository、窄端口和 SQLC 同步；当前指针参与条件更新，不进入配置 hash |
| 请求受理与接替 | 三种命令共享事务入口，HTTP 与 MCP 使用同一应用边界；创建、指针写入、入队原子提交；旧取消位于请求事务外 |
| 配置超时与迟到保护 | `execution_control.go` 统一执行监控、有界取消与结果提交；`deployment.execution_timeout=1h`、`cancel_timeout=5s` 集中校验并注入 |
| 文件准备取消 | local/SSH 文件锁等待与逐项发布检查 Context；Compose 命令及镜像拉取不持有该文件锁 |
| 生命周期目录定位 | 优先使用同修订运行目录；无运行记录时允许已确认目录；真正缺失依赖返回明确原因 |
| 前端操作入口 | Service 卡片、表格、详情与 Gateway 去除库存及活动任务限制，保留请求 busy 和 Gateway 专属入口 |
| 错误诊断与文档 | 原始错误进入请求日志，4xx 为 warning、5xx 为 error，错误响应沿用既有契约；产品、运行时、指南与决策同步 |

实施检查期间发现的 PostgreSQL 指针截断和 Compose 环境变量插值问题均已修正，纳入本次整体交付。

## Task deliveries and integration / 子任务交付与集成

不适用。本任务沿用同一会话中的整体意图与计划，没有建立 task、DAG 或执行分配记录。

## Actual diff summary / 实际差异摘要

进入本阶段时，相对 HEAD 的快照为 81 个文件、2,600 行新增、740 行删除，全部已由用户暂存；包含当前任务和此前独立的终端全屏改动。该数字不含本验证文档及本阶段的文档增量，不能据此把所有文件都归为任务接替功能。

- 持久化与查询：新增 nullable 当前任务指针、六个迁移文件、查询条件及对应 SQLC 生成物；共享 schema 导致多个包的 Service 模型同步生成。
- 应用与运行时：事务受理、统一执行控制、配置超时、条件回写、目录定位和 local/SSH 可取消文件准备。
- 前端：Service 与 Gateway 生命周期入口的准入调整，以及相关组件测试。
- 诊断与配置：错误日志分类、内部 cause、请求关联以及两项 CD duration。
- 修正：四处指针转换使用 `CHAR(26)`；Compose 环境变量在渲染边界将字面 `$` 写为 `$$`。
- 文档：既有活文档与 Intent/Plan；本阶段新增本文、更新 Plan 的阶段及实测记录，并统一 `cd-runtime.md` 的行尾以消除空白检查错误。

## Expected vs actual changed files / 预期与实际修改文件对比

| 范围 | 差异与结论 |
| --- | --- |
| 部署命令与执行用例 | 新增 `command_transaction.go`、`execution_control.go` 和 `operation_takeover_test.go`，与计划中的事务、执行边界及检查职责一致 |
| Repository、SQL、模型与 bootstrap | 属于当前指针与条件更新的完整依赖链，没有改动通用 Worker 协议 |
| 多包 SQLC `models.go` | 共享 schema 生成结果，不表示扩展这些包的业务行为 |
| 新迁移与 schema | SQLite/MySQL/PostgreSQL 各有 000053 up/down；没有修改已执行迁移 |
| Proto/API | 当前指针保持内部字段，没有增加公开准入字段，也没有生成新的 Proto 契约 |
| `compose_renderer.go` | 实测发现的字面值处理修正，作为追加缺陷修复记录；不改变数据库原值或初始化账号 |
| HTTP 错误与日志 | 工作区既有诊断增量，与本次原始停止错误直接相关，一并核对 |
| 终端抽屉与两份 locale | 此前独立的全屏功能，保留并纳入相关组件检查；应与任务接替区分提交范围 |
| 本阶段增量 | 验证文档、Plan 状态记录、运行时文档行尾；没有新增产品代码或实际环境写入 |

## Acceptance criteria checklist / 验收标准检查清单

“通过”的依据为代码核对、已执行自动检查和用户整体实测确认。fake runtime 与 SQLite 查询测试不代表真实 Docker daemon 动作已绝对撤销。

| Intent 验收项 | 结果与依据 |
| --- | --- |
| 库存状态、活动任务不拦截生命周期命令 | 通过：三种命令与前端入口 diff 一致；`TestDeployRestartAndStopReplaceEachOtherRegardlessOfStoredStatus` 及前端组件检查 |
| Service 粒度，三类操作互相接替，Gateway 复用 | 通过：当前指针挂在 Service；用例核对三类命令和不同 Service 隔离，Gateway 保留受管 Service 路径 |
| 后成功受理优先，失败受理保留旧任务 | 通过：事务入口及 `TestFailedAdmissionRollsBackCurrentOperationAndEnqueuedTask` |
| 新操作不等待旧 runtime 正常退出，终态或缺失旧记录不阻塞 | 通过：`TestNewOperationCompletesBeforeOldRuntimeExitsDespiteCancellationFailure`、`TestOldTerminalOrMissingOperationDoesNotBlockNewExecution` |
| 旧取消失败或超时最多 warning，新操作继续 | 通过：上述取消失败测试及 `TestCancellationTimeoutDoesNotBlockNewOperation`，检查 warning 与无额外 ERROR |
| 集中配置的正数超时覆盖整个执行等待 | 通过：配置加载与校验测试，`TestDeploymentExecutionTimeoutReleasesWaitAndPreservesTerminalResult`，代码核对准备、命令与 readiness 使用执行 Context |
| 过期排队任务不执行，失去归属时取消 Context | 通过：Begin SQL 条件、执行监控及当前资格检查 |
| 迟到目录、状态、版本与结果不能覆盖新操作 | 通过：`TestLifecycleWritesRequireCurrentAndExecutableOperation`、迟到成功/失败用例及取消后观测条件 |
| 共享文件准备响应取消，不把整个部署串行化 | 通过：local/SSH `TestCanceledWorkspaceDoesNotPublishWhileFileLockIsHeld` / `TestCanceledWorkspaceDoesNotConnectWhileFileLockIsHeld`，锁仅覆盖文件准备 |
| 目录依赖准确，不扫描宿主、不隐式删卷 | 通过：`TestLifecycleOperationsUseConfirmedDirectoryWithoutRuntimeRecord` 与目录错误测试；实际停止记录成功，删卷仍需显式选项 |
| 错误响应与关联诊断 | 通过：沿用 `code/error/requestId`；请求日志与目录错误测试核对安全摘要、内部 cause、状态及日志级别；未扩大为通用 secret 审计 |
| 验证接替、取消、超时、迟到回写与前端可操作性 | 通过：已有自动检查和用户“实测有效”确认；CI、心跳、租约与恢复仍不参与验收 |

## Test / command results / 测试与命令结果

项目工具链为 Go、SQLC、Task，以及 `web` 下的 Vue/TypeScript、Yarn、Vitest。收集本会话已完成的实现及修正检查；本阶段未修改产品代码，未重复运行已通过的相同测试。

| 检查或证据 | 实际结果与范围 |
| --- | --- |
| `task sqlc` | 通过；PostgreSQL 转换修正后重新生成，核对四个语句的参数类型及 Repository 绑定 |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 实现阶段通过 |
| `task check` | 实现及修正后通过；前端类型、lint、格式及 Go 格式/lint，0 issues |
| `go test ./cmd/... ./internal/...` | 实现及修正后通过，部分既有包使用缓存；包含 SQLite 持久化与 fake-runtime 用例，不据此宣称可选真实数据库 E2E 已运行 |
| 相关 Vitest | 3 个文件、26 个测试通过：`serviceDeployment.test.ts`、`gatewayNavigation.test.ts`、`EnvironmentTerminalDrawer.test.ts` |
| PostgreSQL 只读诊断 | 证实无长度 `CHAR` 截断为一个字符；`CHAR(26)` 返回完整 ID；后续停止成功且 Service 当前指针为完整新 ID |
| 实际 Compose 配置解析 | Compose v5.4.0；临时程序调用实际有效计划合并与渲染器，8 组虚构值的解析配置 hash 与原始字面值一致、无警告，重复渲染稳定；未启动容器，临时程序已删除 |
| `git diff HEAD --check` | 本阶段初次发现 `cd-runtime.md` 混合行尾；统一 LF 后通过 |
| Go `-race` | 未运行成功：CGO 关闭且环境无可用 gcc/clang，未改变工具链 |
| 用户人工实测 | 用户明确确认“实测有效”；未提供逐项命令和全部平台清单，不补造覆盖信息 |

### 实际运行记录

- `01M4K2PC5Y02P597GFH7ERRTXX`：历史 stop 被误取消，原因为 PostgreSQL 指针被截断，不能当作正常接替的成功证据。该记录保留原终态，没有重放或伪造成功。
- `01M4K3X8YXDPYPC1PPAKCVTSQS`：北京时间 22:36:41 开始的 stop，记录为 `ran_to_completion`；用户提供的日志包含容器和网络移除，以及旧配置的变量插值 warning。
- `01M4K4C7TF63M53ZEMHQM6EJWG`：北京时间 22:44:52.832 至 22:44:55.538，stop 成功，耗时 2,705 ms。只读核对 Service 为 `stopped`，当前指针等于该完整 ID；运行目录记录为空，操作使用已确认的同修订目录，验证了此生命周期路径。
- 上述最新 stop 在 Windows SSH 目标执行。停止直接读取远端已有 Compose，不重新渲染；旧文件的 warning 不等同于停止失败，也不能用此记录宣称新的 `$` 转义已重新发布到远端。
- 环境变量修正的自动证据为实际渲染器加 Compose 配置 hash 核对，涵盖单个 `$`、`${...}`、连续 `$$`、引号/反斜杠、多行、空值、普通值和共享环境变量解析。后续用户实测确认作为人工验收结论另行记录。

## Missed or expanded scope / 遗漏或扩大的范围

- 核对未发现原定实现范围遗漏。PostgreSQL 显式转换与 Compose 字面值转义是执行中暴露的必要修正，已记录于 Plan；未引入数据库兼容分支或改变用户配置值。
- 按用户要求，不增加 PostgreSQL 指针问题的专门回归。Compose 使用一次性检查收集实际解析证据，没有保留新增测试文件。
- 独立终端全屏改动存在于整体 diff 中，建议单独组织提交；本报告不重新定义该功能的验收标准。
- 本阶段仅整理验证记录和处理相关文档行尾，没有启动新生命周期任务、执行数据库迁移、修改历史业务记录或执行 Git 写操作。

## Risks / 风险

- Docker daemon 已接受的动作可能在 CLI 或 SSH 取消后继续；当前归属与条件回写保证业务结果不被旧任务覆盖，外部动作取消仍为尽力策略。
- Go 不能强杀任意不响应 Context 的 goroutine；应用等待有界，迟到结果受保护，退出未确认需如实 warning。
- 文件或远端 I/O 已进入系统调用时不能绝对撤销；文件锁与检查限制后续发布，不提供跨 Project 或物理宿主协调。
- 旧 Compose 文件要由后续部署或重启重新生成。修正环境变量不会自动改变已初始化 MySQL 数据中的账号密码，也不会自动删除持久化卷。
- 用户实测确认未附逐平台明细；MySQL 作为 Orbit 存储后端、Linux SSH、真实卡死命令的强制退出及 daemon 取消效果没有逐项实测证据。上述边界不改变已约定交付范围。

## Incomplete items / 未完成事项

当前授权范围内无已知未完成的实现项。`-race` 受环境限制，未补齐逐平台实测记录；作为验证边界保留，不据此新增任务、工具链安装或实际环境操作。独立终端改动的提交拆分由用户安排，代理没有改动暂存区。

## Conclusion / 结论

实现与已接受意图及计划对齐，必要开发检查通过，实测中发现的两个缺陷已修正。用户已明确确认“实测有效”，本次验证完成并记录为 Accepted。

该结论确认 Service 后置操作优先、尽力取消、配置超时和旧结果保护等本次业务承诺；不把业务取消状态解释为 Docker 物理动作已绝对撤销，不扩展到未约定的 CI 或恢复机制。未执行提交、推送或实际环境变更。
