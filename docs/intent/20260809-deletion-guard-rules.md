# 删除路径防御规则整理
最后修改时间: 2026-08-09 13:58:22

Flow mode: light

Review status: Accepted

## Background

网关删除修复暴露出若干删除路径同时存在业务层检查、仓储层引用兜底和外部状态解析。它们的职责不一致：有些检查是用户必须先完成的业务操作，有些只是在重复预防本应不会发生的调用，或尝试解释 Docker 的异常状态。

## Goal

- 删除操作只保留真实的业务前置条件，并通过明确的验证提示引导用户处理。
- 移除通用 Application 删除中与用例层重复的 Version 运行时引用预检。
- 将 Credential、Repository 删除的既有阻止条件统一为验证错误，而非专门的冲突分支。
- 移除 Gateway 部署前对 Docker network inspect 输出、数量和 Compose 标签的异常状态分析。
- 保留 Pipeline DAG 节点删除的依赖完整性检查，并让提示明确指出仍依赖该节点。

## Non-goal

- 不改变 Credential 被 Repository 使用、运行中的 Pipeline 使用 Repository、或 DAG 节点被依赖时不得删除的业务约束。
- 不新增数据库约束错误转换、异常数据恢复、通用删除框架或兼容分支。
- 不修改已执行迁移、Gateway 删除语义、Docker Compose 生命周期或前端。

## User scenarios

1. 用户删除普通 Application 时，存在 Service 或 Version 会立即收到对应的明确验证提示；仓储层不再重复检查 Version 运行时引用。
2. 用户删除仍被 Repository 使用的 Credential，或删除仍有运行 Pipeline 的 Repository 时，收到可执行的验证提示。
3. 用户删除仍被其他节点依赖的 Pipeline 节点时，提示中包含被删除节点的名称。
4. 部署 Gateway 时不再额外解析 Docker 网络的异常状态；网络创建与冲突由 Compose 命令的正常执行路径处理。

## Acceptance

- 通用 `DeleteApplication` 不再调用 `CountVersionRuntimeRefs` 或返回 `ErrReferenced`；用例层对 Service/Version 的检查保持不变。
- Credential 与 Repository 的删除前置条件返回 `KindValidation`，消息清楚说明引用对象或需要停止的 Pipeline。
- Pipeline 节点删除仍会因依赖关系被拒绝，且消息包含目标节点名称。
- Gateway 部署路径不再调用 `preflightGatewayNetwork`；相关预检实现和测试移除。
- 覆盖上述删除/部署分支的 Go 测试通过；后端格式、静态检查与测试通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 业务规则由所属用例显式检查并返回验证错误；Repository 不为无法通过公开用例到达的状态增加重复引用防线。
- `409 conflict` 保留给需要表达并发或状态冲突的产品语义；本次删除前置条件使用 `400 validation`。
- Docker 网络所有权不是应用数据模型的一部分，不在 Gateway 部署前额外推断和分类。

## Risk

- 移除网络预检后，已有不兼容 Docker 网络将在 Compose 执行时失败，返回路径由既有部署任务日志提供；本次不解析或重写 Docker 错误。
- 通用 Application 仓储删除也用于创建失败后的补偿清理；它会在单一事务中清理该 Application 自己的 Service 和 Version。公开删除路径仍先执行所属用例的业务检查。
