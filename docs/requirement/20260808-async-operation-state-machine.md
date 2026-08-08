# 异步操作状态机收敛需求
最后修改时间: 2026-08-08 13:43:45

Review status: Accepted

## Background

现行 CD 模型规定 `Service` 是运行态 SoT，`Deployment` 是异步操作流水；Pipeline Run 与 Pipeline Stage Run 也使用同一组工作状态。当前只列出了状态集合，没有把转换、取消、自动重试和记录关联定义为一个受约束的状态机。

现有实现存在以下不一致：

- 业务操作使用 `waiting_to_run`、`running`、`ran_to_completion`、`faulted`、`canceled`，底层 `background_task` 使用另一套 `pending`、`running`、`succeeded`、`failed`、`canceled`；
- 队列自动重试可让已经标记为 `faulted` 的 Deployment 再次执行并覆盖终态；
- 运行中取消会先写入 `canceled`，但外部命令或 Pipeline Stage 可能继续执行；
- Pipeline 上游失败后尚未进入执行的下游 Stage 被记录为 `canceled`，混淆了“主动取消”和“尚未运行”；
- `Service.deploying` 承担了异步操作进行态，和 Deployment 状态重复，导致操作失败或取消后服务可能滞留在不可操作状态。

## Goal

1. 为 Deployment、Pipeline Run 和 Pipeline Stage Run 定义明确、可验证的状态机与状态所有权。
2. 将用户可见异步操作的主状态固定为：`waiting_to_run`、`running`、`ran_to_completion`、`faulted`、`canceled`。
3. 保证 `ran_to_completion`、`faulted` 和 `canceled` 为不可逆终态；任何状态更新都必须验证期望源状态。
4. 将 Pipeline Run 的 Retry 定义为命令而非状态或回边：它创建一条新的 Run 记录，并以 `waiting_to_run` 进入完整状态机；原 Run 保持终态并可追溯。
5. 将 Service 运行态与操作进行态分离，界面和并发守卫由关联的非终态操作决定，而不是由 `Service.deploying` 推断。
6. 让状态记录能准确表达实际效果、失败原因、取消结果和重试谱系，供 HTTP、MCP、Web 与审计读取。

## Non-goal

- 不改变 Application、Version、Component、Service 的业务配置模型或 Docker Compose 渲染语义。
- 不在本需求中增加暂停、定时调度、优先级或多节点分布式编排能力。
- 不把基础设施 lease 恢复或消息投递细节直接暴露为业务操作状态。
- 不修改已执行迁移文件；所需持久化结构变更须通过新迁移完成。

## User scenarios

1. 用户发起 Deploy、Restart 或 Stop 后，立即看到对应操作处于 `waiting_to_run`；worker 领取后变为 `running`，最终进入一个终态。
2. 用户可对任意状态的 Pipeline Run 执行 Retry。原 Run 状态不变；新 Run 保存父 Run 关联，从 `waiting_to_run` 独立执行；新 Run 仍受 Repository 活跃执行并发守卫约束。
3. 用户取消尚未执行的操作。该操作不会开始外部副作用，并以 `canceled` 终结。
4. 用户取消正在执行的操作。系统立即将操作记录为 `canceled`，worker 随后停止外部命令和子任务；`canceled` 表示取消已被接受，不承诺外部副作用已经回滚或停止完成。
5. Pipeline 的上游 Stage 失败后，所有尚未进入 `running` 的下游 Stage 保持 `waiting_to_run`；它们不得被写为 `canceled` 或额外的 `skipped` 状态。
6. 同一个 Service 存在非终态操作时，Web、HTTP 与 MCP 一致地拒绝或引导取消冲突操作；Gateway 的单活跃约束也使用同一判定。

## Cancel scope

- Deployment：现有取消入口覆盖 Deploy、Restart 和 Stop；三类操作都需要区分排队取消与运行中取消。Stop 进入 Docker `down` 后可能已经产生部分副作用，取消语义须单独定义其收敛方式。
- Pipeline Run：现有取消入口覆盖整个 Run 的排队和运行中状态；运行中取消立即终结 Run，并把停止信号传递到正在执行的 Stage/容器。
- Pipeline Stage Run：当前没有独立取消入口。本需求不擅自增加 Stage 级用户取消；Run 取消只处理实际已开始的 Stage。没有开始的下游 Stage 按已确认规则保留 `waiting_to_run`。
- `background_task`：当前没有用户取消入口；它是基础设施调度记录，不是另一个用户可见的业务取消对象。

## Current retry configuration

- `worker.max_attempts` 是新建 background task 的默认重试预算，开发配置固定为 `1`。
- 入队必须将该值持久化到 `background_task.max_attempts`；worker 仅按任务记录中的 `attempts` 与 `max_attempts` 决定失败后回到 `pending` 或进入 `failed`，不得在执行时读取当前配置。
- 运行中修改配置只影响后续新建任务，不能改写排队或运行中任务的预算。业务任务默认预算为 `1`；Pipeline Run Retry 仍是独立的显式用户命令，不受 task 自动重投影响。

## Acceptance

- [ ] 活文档给出对象级状态集合、允许转换、终态语义、取消语义、Pipeline Run Retry 语义和并发规则。
- [ ] Deployment 与 Pipeline Run 的主状态只使用已确认的五个值，且状态更新由条件写入保障合法转换。
- [ ] Pipeline Run Retry 不修改原 Run 的终态；新 Run 具备 `retry_of` 父 Run 关联，并从 `waiting_to_run` 进入状态机。
- [ ] Pipeline Run Retry 不以来源 Run 状态作为前置条件；只受新 Run 的独立并发守卫约束。
- [ ] background task 的失败重投只依据持久化的 `attempts/max_attempts`；运行时配置变化不影响既有任务。显式的 Pipeline Run Retry 仍创建新 Run。
- [ ] 业务执行超时一律进入 `faulted`，错误原因明确为 timeout；默认预算为 `1` 时对应 background task 进入 `failed`。超时不能被记录为 `canceled`。
- [ ] Deploy/Restart 的成功继续以 `docker compose up -d` 返回零退出码判定；Healthcheck 不改变 Deployment 主状态。
- [ ] 运行中取消立即记录为 `canceled`；worker 使用该终态停止外部执行，且后续执行不得覆盖该终态。
- [ ] Service 不再使用 `deploying` 表示操作进行态；活跃操作查询成为唯一的并发和 UI 操作守卫依据。
- [ ] Pipeline 上游失败后，尚未进入 `running` 的下游 Stage 仍为 `waiting_to_run`；只有 Cancel API 时实际处于 `running` 的 Stage 才随 Run 迁移为 `canceled`。
- [ ] HTTP、MCP、Proto、Web 显示与错误行为，以及 Go 单元测试和相关集成测试，覆盖所有允许和拒绝的转换。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 用户已确认：Pipeline Run Retry 不是状态，不得表示为 `running -> waiting_to_run` 等状态回边。
- 用户已确认：Pipeline Run Retry 后必须以新的 Run 记录重新进入状态机，原 Run 保持终态。
- 当前不为 Deployment 设计、增加或预留 Retry / `retry_of`；它不是本需求范围。
- 用户已确认：Pipeline 上游失败时，未进入运行状态的下游 Stage 保持 `waiting_to_run`，不得写为 `canceled` 或 `skipped`。
- 用户已确认：运行中取消立即写 `canceled`；该终态同时是 worker 停止外部执行的持久化信号。
- 用户已确认：`worker.max_attempts` 必须在任务创建时写入 `background_task`；worker 只按任务持久化预算决定后续重投，不能运行时读取配置。
- 用户已确认：历史 `Service.deploying` 记录在主业务完成后通过独立离线脚本处理，不在业务代码中兼容、迁移或回写。
- 用户已确认：业务执行超时是失败，主操作状态为 `faulted`，不是 `canceled`。
- 用户已确认：Deploy/Restart 不等待 Healthcheck，Compose 命令零退出码即为业务成功。
- 用户已确认：Pipeline Run Retry 不区分来源 Run 状态；来源 Run 状态不构成重试前置条件。
- 主操作状态机的终态不可逆；状态变化不允许仅按记录 ID 无条件更新。
- `Service` 只表达运行态，不表达 Deployment、Restart 或 Stop 的操作进行态。

## Risk

- 状态收敛同时影响数据库查询、队列 worker、Deployment、Pipeline Run、Gateway 约束、HTTP/Proto/MCP 和 Web 操作入口，必须按统一规格实施，不能局部修补。
- `canceled` 先于外部进程实际退出可见；Docker/Pipeline runner 必须可靠观察该终态并停止执行，避免用户看见已取消但外部工作长时间继续。
- 现有 `Service.deploying` 历史记录需要在上线前离线清理；业务代码不承担历史状态回写。

## User review notes

- 2026-08-08：用户要求使用严格模式收敛状态与任务记录。
- 2026-08-08：用户明确 Retry 是命令，不是状态；Retry 创建新记录并重新进入状态机。
- 2026-08-08：用户明确下游未进入运行状态的 Stage 应保持 `waiting_to_run`。
- 2026-08-08：用户确认 Deployment 不存在 Retry；Retry 需求仅限 Pipeline Run。
- 2026-08-08：用户要求开始 Spec，Requirement 视为接受。
- 2026-08-08：用户确认取消立即写 `canceled`，取消观察复用 `worker.poll_interval`；历史 `Service.deploying` 离线处理，不写入业务代码。
- 2026-08-08：用户先前要求业务任务默认不重试，开发配置固定 `max_attempts=1`。
- 2026-08-08：用户确认超时后业务操作应为 `faulted`，并记录 timeout 原因。
- 2026-08-08：用户确认部署仍以 Compose 命令退出码判定成功，不将 Healthcheck 纳入主状态。
- 2026-08-08：用户确认 Pipeline Run Retry 不区分来源 Run 状态。
- 2026-08-08：用户要求先完成主业务，再用独立脚本处理历史数据。
- 2026-08-08：用户更正 `max_attempts` 不得移除；配置值在入队时冻结并持久化，worker 后续只读取任务记录。
- 2026-08-08：用户确认本开发任务只占用 `000031`；`max_attempts` 变更收回该迁移，不保留补偿版本。
