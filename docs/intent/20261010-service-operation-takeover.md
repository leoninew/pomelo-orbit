# Service 容器操作与任务接替
最后修改时间: 2026-10-10 21:24:46

Review status: Accepted

Mode: standard

## Background

用户报告应用停止请求返回 `500/internal_error`，响应只有通用摘要，日志只有 `container log configuration is not ready`。初步排查发现该错误实际来自 Service 的 `runtime_directory` 为空，发生在创建停止任务之前，并非容器日志读取失败。

用户可以通过终端、Docker CLI 或其他系统启动、停止、重建容器；Orbit 保存的 `Service.status` 可能与实际状态相反。系统没有运行目录记录，也不能直接证明目标上没有容器。

当前部署、停止、重启调用 `ensureNoActiveDeployment`，拒绝同一 Service 有活动任务时的新操作；停止、重启另有库存状态检查。Service 与 Gateway 页面依据 `status`、`active_deployment` 禁用操作。用户要求不能因这些记录阻止用户在没有 Orbit 时本来可以进行的操作，也不能要求前一个部署结束后才执行下一个。

系统仍有义务管理自己发起的任务。同一 Service 后提交的有效操作优先，并尝试取消前一个任务。旧任务可能已失败、卡死或不存在；取消无法达成时容错，最多记录 warning，不阻塞新操作。配置化执行超时提供保底边界。

现有 Worker 每秒检查 Deployment 取消状态，本地命令使用 `exec.CommandContext`，SSH 在 Context 取消时关闭连接。可以扩展现有机制检查当前操作归属；数据库取消状态不代表 Docker daemon 已撤销已接收的命令。

现状依据：

- [CD 产品模型](../product/cd-model.md)、[CD 运行时](../architecture/cd-runtime.md)、[后端架构](../architecture/backend.md)、[部署指南](../guides/deployment.md)。
- `internal/application/deployment/usecase/command.go`、`service_location.go`：请求准入及目录依赖。
- `internal/application/deployment/usecase/service.go`、`deployment_execution.go`：取消、执行监控与结果回写。
- `internal/infrastructure/runner/{local,ssh}/runtime.go`：命令执行与共享文件准备。
- `web/src/views/service/{ServicePage,ServiceDetail}.vue`、`web/src/views/gateway/GatewayDetail.vue`：操作按钮与状态限制。
- `internal/api/http/transport/error.go`、`internal/api/http/middleware/logging.go`：错误响应与请求诊断。

## Goal

1. 部署、停止、重启不以库存运行状态或旧活动任务作为额外准入条件，前后端遵循同一规则；保留权限、参数、目标修订与真实资源边界校验。
2. 以 Service 为接替粒度，部署、停止、重启共享同一当前操作归属；后成功受理的操作取代先前操作，不等待先前操作正常结束或确认退出。
3. 尽力取消旧任务及其后续编排。已终态、失败或不存在的旧任务幂等跳过，保留原结果；取消失败、超时或退出未确认最多记录 warning，不影响新任务执行。
4. 所有本期 CD 生命周期任务使用独立的配置化执行超时，卡死任务不能无限占据执行等待。
5. 被接替任务不能覆盖新任务的运行目录、版本、Service 状态与当前任务结果；成功、失败、取消后观测和迟到返回均受操作归属保护。
6. 按操作真实依赖处理目录和 Compose 配置。缺少运行目录记录不统一解释为必须先部署；可确定的当前目标目录可用于执行，确实缺少必要输入时给出准确原因。
7. 失败响应保持既有错误契约，接替、取消和超时日志可关联 Service、前后 Deployment、阶段及内部原因；取消未确认不能记为物理停止成功。

## Non-goal

- 不处理多个 Project 共用宿主机、Docker daemon、目录或 Compose project 的协调；该配置造成的冲突由用户负责。
- 不按 Environment、宿主机或物理容器身份合并接替范围，Environment 继续仅承担既有目标分派与修订校验职责。
- 不增加心跳、租约、进程重启恢复、执行批次恢复或通用队列改造。
- 不考虑 CI；不扩展 Probe、Route 发布、终端会话的任务接替。
- 不限制用户在 Orbit 外操作容器，不承诺独占 Docker 或取消已发往 daemon 的动作。
- 不自动回滚容器、网络、文件或数据副作用，不自动删卷或迁移旧数据。
- 不增加任意外部容器接管、宿主扫描或运行目录恢复系统。
- 不引入兼容层，不修改已执行迁移，不管理开发服务器。

## User scenarios

1. 用户在 Orbit 外启动服务，页面仍显示停止；仍可发起停止。库存显示运行而容器已停止时，也可再次部署或重启，由实际执行反馈结果。
2. A 正在部署或拉取镜像，用户提交 B；B 成为同一 Service 的当前操作，系统尝试取消 A。随后提交 C 时，C 接替 B，执行顺序不取决于 Worker 领取顺序。
3. A 已失败或已完成，但页面或任务信息滞后；提交 B 不需要成功取消 A，也不覆盖 A 原有的失败原因。
4. A 看似运行中，实际已卡死或执行记录不存在；B 仍可接替，取消异常仅 warning，配置化超时限制旧执行等待。
5. A 在 B 开始或完成之后返回成功、失败或取消后观测结果；A 的结果不覆盖当前 Service 的目录、版本或状态。
6. A 取消时 Docker 已处理部分操作；B 继续执行自己的意图，系统报告实际错误，不自动回滚或删除持久化数据。
7. 不同 Service 的任务独立；即使关联同一 Environment，也不会互相取消。
8. 停止或重启缺少真正必要的目录或配置时，反馈具体依赖缺失，而不是日志配置错误或统一要求先部署。

## Acceptance

- [ ] 前后端不因 `Service.status`、`active_deployment` 或旧任务存在拒绝部署、停止、重启；其他必要校验保留。
- [ ] 接替粒度为 Service，三类操作互相接替；不同 Service 不合并，Gateway 使用其受管 Service 的同一规则。
- [ ] 同一 Service 后成功受理的操作优先；无效请求、创建失败或入队失败不取消有效旧任务。
- [ ] 新操作不等待旧部署正常完成或取消确认；旧任务已经失败、终态或不存在时幂等跳过，保留原结果。
- [ ] 旧任务取消失败、取消超时或停止未确认只记 warning，不使新操作失败，也不伪称旧进程已停止。
- [ ] CD 生命周期执行使用集中 typed config 中的正数超时，覆盖准备、命令与 Gateway 就绪等待；超时终止执行等待，并尝试取消所属 runtime。
- [ ] 已被接替的排队任务不执行，运行任务发现失去归属后取消 Context，不继续主动编排后续步骤。
- [ ] 旧任务的目录绑定、成功/失败回写、版本变更及取消后观测不能覆盖当前操作；当前任务结束后也不会重新赋予旧任务资格。
- [ ] 共享文件准备检查 Context 与操作归属，避免已知被接替任务继续发布；已经发出的外部 I/O 作为尽力取消边界明确记录。
- [ ] 目录缺失反馈区分真实依赖与库存未记录，不新增宿主扫描或自动恢复；删卷仍只由显式选项触发。
- [ ] 响应使用既有 `code`、安全摘要 `error`、`requestId`；日志关联请求、Service、前后 Deployment 和内部 cause，秘密不进入错误响应或日志。
- [ ] 验证接替、取消失败继续、超时、迟到回写和前端可操作性；不扩展心跳、租约、重启恢复或 CI 验收。

## Open questions

暂无需要用户确认的未决事项。当前操作标识、条件写入与超时配置的具体形式在 Plan 草稿中给出，属于待审阅的实施选择。

## Decisions

- 用户指定标准模式 / standard，流程为 Intent -> Plan -> Implementation -> Verification；用户要求开始计划，本 Intent 标记为 Accepted。
- 用户明确同一容器以后置操作为准；本系统以 Service 为业务粒度，按有效操作成功受理的顺序确定先后。
- 取消旧任务采用尽力策略：取消无法达成时容错，最多 warning，新操作继续。
- 配置化超时是卡死执行的保底，不通过心跳、租约或重启恢复识别存活。
- 不考虑跨 Project 共用宿主；不根据 Service 关联的 Environment 等信息扩展接替范围。
- 暂不考虑 CI，不把共享队列治理带入本任务。
- 保留真实授权、目标修订、必要参数与资源边界；实际查询用于执行反馈，不增加运行状态准入锁。
- 本任务沿用独立过程文档，不修改此前部署目录或环境终端任务的历史文档。

## Risk

- Docker daemon 已接收的命令可能在 CLI 或 SSH 连接关闭后继续。后置优先保证 Orbit 当前操作与回写归属；物理副作用取消仅尽力实现，不能保证任意不可取消命令的最终先后。
- 共享 Compose 和受控配置可能有已发出的 I/O；Context 与归属检查可停止后续步骤，无法撤回已经进入操作系统或远端的写入。
- 配置超时能够限制应用等待，Go 无法强杀任意不响应 Context 的 goroutine；实现需确保其迟到结果失效，并在 warning 中如实说明退出未确认。
- 用户在系统外操作仍可能与 Orbit 命令重叠；不同 Project 配置同一物理目标的冲突不由本任务协调。
- 当前工作区有此前未验证的错误诊断改动；“缺少运行目录必须先部署”的提示需调整，未使用 import 及相关检查需在 Implementation 收尾。终端全屏改动独立于本任务。

## User review notes

- 用户指出停止失败响应和日志没有有效信息，并指出库存状态可能与实际容器状态双向偏离。
- 用户要求允许前一次未完成时发起并执行下一次部署，否定等待前次结束的串行约束。
- 用户要求重入时取消旧任务，并考虑已失败、卡死或不存在的旧任务。
- 用户确认后置操作优先，取消失败最多 warning，并以配置化超时保底。
- 用户确认接替力度为 Service，排除跨 Project 共用宿主、心跳、重启恢复、租约和 CI。
- 用户要求开始 Plan；本轮更新 Intent 并创建 Plan 草稿，不进入产品实现。
