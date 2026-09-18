# 网关删除清理
最后修改时间: 2026-08-09 13:28:43

Flow mode: light

Review status: Accepted

## Background

删除已停止服务的网关时，网关的托管 Version 仍被该 Service 引用。当前通用 Application 删除逻辑会在清理 Service 之前检查 Version 引用，导致网关删除返回 500；手动先删除 Service 后才能成功。

## Goal

- 用户删除网关时，系统自动删除该网关的停止 Service、托管 Version、Application 和 GatewayConfig。
- 仍有非停止 Service 时拒绝删除，并明确指出需要先停止的 Service。
- 网关删除 API 和 Web 客户端不再使用 `remove_dir`，删除网关不显式清理工作目录。

## Non-goal

- 不改变标准 Application 的删除语义和引用保护。
- 不删除工作目录，不变更通用 Application 删除 API 的 `remove_dir` 行为。
- 不改变网关部署、停止或渲染行为。

## User scenarios

1. 用户删除一个仅有停止 Service 的网关，删除成功，网关及其内部部署载体不再可见。
2. 用户删除一个仍存在非停止 Service 的网关，收到验证错误，网关数据保持不变。
3. Web 删除网关时不发送 `remove_dir` 查询参数。

## Acceptance

- 网关专用删除在单个数据库事务中按 Service、Version、GatewayConfig、Application 的顺序清理。
- 网关删除前检查全部 Service；发现非停止 Service 时返回明确的验证提示，不进入删除事务。
- `DELETE /api/gateway/:gateway_id` 忽略且不再读取 `remove_dir`；Web API 不再声明或发送该参数。
- 相关 Go 单元测试通过，前端 lint 与类型检查通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- Gateway 的 Application 与 Version 是内部部署载体；删除 API 的用户语义由网关领域负责，不要求用户预先删除停止 Service。
- 保留通用 `DeleteApplication` 的引用保护，新增网关专用删除路径而非修改其语义。
- 网关工作目录由独立运维操作管理，不随删除网关清理。

## Risk

- 数据库事务不能包含文件系统副作用，因此本变更移除网关删除中的目录删除，避免数据库失败后目录已删除的部分失败状态。
- 删除流程基于正常领域数据关系，不额外引入异常数据分析或数据库约束兜底分支。

## Follow-up

- 其他删除路径中的重复引用预检、冲突分支和异常外部状态检查已登记至 `docs/decisions/ledger.md` Backlog，后续以独立任务审视，不纳入本次变更。
