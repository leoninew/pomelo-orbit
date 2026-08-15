# Gateway REST Deployment Boundary
最后修改时间: 2026-08-15 12:08:44

Review status: Accepted

## Background

普通 `kind=standard` 应用当前会将所有组件接入共享的外部 `traefik` 网络，以支持跨应用服务通信。组件端点的 `mode=gateway` 则在此基础上生成 Gateway 入口所需的 Docker provider labels 和静态配置；它不是 `Application(kind=gateway)`，也不在普通应用部署时向 Traefik REST 发布 Route。

此前未提交的实现将 `EnsureGatewayRunning` 放入普通 Service 的部署与重启命令，导致 Gateway 未运行时拒绝入队。这把入口可达性错误地作为应用进程启动的前提条件。

## Goal

普通 Service 的 Deploy/Restart 仅按通用 Service 规则入队。Gateway 是否运行不阻断部署；Gateway 停止时，面向 Gateway 的入口暂不可达，但应用仍可运行。用户可在部署和 Compose 预览时取消加入共享 `traefik` 网络。

## Non-goal

- 不改变 Gateway 部署后 Route 快照发布，或 Route 修改时的 Traefik REST 发布。
- 不改变多个 Gateway 都未运行时的配置选择规则；当前实现仅在存在唯一已配置 Gateway 时回退使用其配置，多个候选仍应显式消除歧义。
- 不为 Gateway Application 提供脱离其运行网络的选项。

## User scenarios

1. 默认部署或预览标准 Service 时，Compose 保留共享 `traefik` 外部网络和跨应用别名。
2. 用户取消加入网络时，标准 Service 的部署与预览都不渲染该网络、别名、Gateway labels 或 Gateway 配置依赖；应用仍保留 Compose 默认网络。
3. 唯一已配置 Gateway 已停止时，仍加入网络的普通应用可以部署；面向 Gateway 的入口不会在此期间可访问。
4. Gateway 自身 Compose 成功后，系统仍等待 Traefik REST 就绪并发布完整 Route 快照。
5. 用户创建、更新、启停或同步 Route 时，系统仍只对实际 Traefik REST 发布路径校验运行中的 Gateway 并处理请求失败。

## Acceptance

1. `DeployService` 和 `RestartApplication` 不再调用或暴露 `EnsureGatewayRunning`。
2. 标准 Service 的 Deploy、Service Preview 和 Version Preview 公开 `join_traefik_network` 选项，默认值为 `true`；其选择进入部署 options 与有效计划哈希。
3. `join_traefik_network=false` 时，标准 Service 不渲染或校验 `traefik` 网络，也不生成 Gateway labels 或读取 Gateway 配置。
4. `Application(kind=gateway)` 始终加入其运行网络；成功部署后的 `PublishSnapshot` 与 Route REST 发布路径保持现有 readiness/发布语义。
5. 不保留只服务于普通部署 Gateway 运行态预检的接口、实现或测试。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. `mode=gateway` 是组件端点的 Gateway 入口模式，不是 Gateway Application 身份，也不代表普通应用要调用 Traefik REST。
2. Gateway 运行态只属于实际 REST 请求的前置条件，不属于普通 Service 进程部署的前置条件。
3. 标准 Service 默认接入共享网络；取消选项会停用该次部署或预览的共享网络与 Gateway ingress 投影，而不是仅移除网络后保留不可达 labels。
4. 普通应用带 Gateway 入口端点仅在加入共享网络时需要 Gateway 配置；这与 Gateway 是否正在运行是两个独立问题。

## Risk

默认加入网络时，外部 `traefik` 网络不存在仍会在 Compose 执行期失败；取消该选项可部署为隔离的 Compose 默认网络。Gateway 停止时，加入网络的应用可启动但入口暂不可达。这些都不应由命令阶段的 Gateway 运行态预检替代。

## User review notes

- 用户确认普通应用部署不应关心 Traefik Gateway 是否运行；最多入口不通。
- 用户澄清：`mode=gateway` 表示面向 Gateway 的应用端点，不等同于 Gateway Application。
- 用户确认共享 `traefik` 网络具有跨应用服务通信用途；部署与预览必须提供默认勾选的加入选项，取消后不渲染或校验该网络。
- 用户要求采用轻量模式记录此任务并直接开始实现；Requirement 视为 Accepted。
