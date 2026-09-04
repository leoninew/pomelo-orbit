# 远程 SSH 部署环境需求
最后修改时间: 2026-09-04 08:02:27

Review status: Accepted

Mode: strict

## Background

当前 Project 已承担资源归属、项目成员授权、HTTP/Proto/MCP scope 和 Web active-project 状态。它应保留为轻量租户边界，而不是删除后改用全局资源。切换 Project 必须同时切换到该 Project 的唯一部署 Environment。

CD 当前仍直接使用控制面本机 Docker daemon、workspace.deployment、全局 traefik network 与单 active Gateway。本期将 CD 收敛为 SSH 管理的远程 Docker Compose target，并让 Project、Environment 与 Gateway 的关系成为明确数据模型。

## Goal

1. 保留 Project、project_member、现有 Project scope 和 active-project Web 状态；Application、Repository、Credential、Pipeline、PipelineRun、Deployment、Route 与 Gateway 继续按 Project 隔离并复用 membership 授权。
2. 每个 Project 必须且只能拥有一个 Environment；Environment 只表示该 Project 的唯一 SSH Docker Compose target，切换 Project 即切换部署目标。
3. 每个 Environment 必须且只能绑定一个 Gateway；Gateway backing Application/Service 与 Environment 所属 Project 一致。
4. Service、Route 和 Gateway 从所属 Project 推导唯一 Environment，不增加可漂移的用户选择器；Deployment 保存 Environment、target revision、deploy-key revision 和 Gateway snapshot。
5. 仅支持 Linux OpenSSH + Linux containers，以及 Windows native OpenSSH + WSL2 Docker Desktop Linux containers；全部 CD、Gateway/Route 与 Traefik REST 操作经 SSH/SFTP 在目标端执行。
6. Environment 只支持 active 或 disabled，不支持删除；registry 登录、CA、DNS、网络和多个 registry 的访问由目标宿主机负责。

## Non-goal

- 不删除 Project、Project member、Project API、Project MCP scope、active-project selector 或既有 Project resource isolation。
- 不允许一个 Project 多个 Environment，也不允许一个 Environment 多个 Project 或多个 Gateway。
- 不支持 local executor、控制面 Docker daemon 部署、共享 Gateway、跨 Project Route publish、Cygwin/MSYS/Git Bash/WSL SSH server、Windows Containers 或 registry credential 管理。
- 不提供 Environment delete、运行时兼容或从 Project 推断到控制面本机部署的默认回退。

## User scenarios

1. 项目切换：成员在顶栏切换 Project，后续 Application、Service、Route、Gateway 和 Deployment 查询仍按该 Project 过滤，部署目标随之变为该 Project 的 Environment。
2. 创建项目：管理员通过同一事务创建 Project 与完整的初始 Environment。Environment 可先是 disabled，但必须包含合法 SSH target、host-key pin 和 deploy-key；不能出现无 Environment 的 Project。
3. 项目部署：成员部署某个 Project 的 Service 时，系统从 Application 的 project_id 解析唯一 Environment，验证其 active、Probe freshness 和 Gateway binding，并将相关 revision 固化到 Deployment。
4. 网关与路由：Environment provision 只为该 Project 创建一个 Gateway；Project 内的 Route 只发布到该 Gateway，managed target 必须属于同一 Project。
5. 租户隔离：成员无法通过资源 ID、Route target、Gateway 或 MCP 参数跨越 Project 访问或发布；管理员仍通过既有 membership/role 配置管理可访问项目。

## Acceptance

- [x] Project 是资源隔离与成员授权边界；所有现有 Project-scoped resource contract 和 active-project UI 保留并恢复一致。
- [x] Environment 具有 project_id 逻辑引用和唯一约束；每个 Project 恰好一个 Environment，Environment 不能删除。
- [x] Environment code 从不可变 Project code 派生，Project code 创建后不可修改；避免 network、workspace 与 remote identity 漂移。
- [x] Project 创建以事务性方式同时写入 Project、专用 deploy-key Credential 和 Environment；失败不得留下部分 Project 或凭据数据。
- [x] Gateway 与 Environment 是一对一关系；其 backing Application/Service、Route 和目标 Service 都必须属于 Environment 的 Project。
- [x] Service/Route/Gateway API 不要求用户传 environment_id；Deployment snapshot 保存实际 Environment identity/revision，所有 runtime operation 从 snapshot/Project 解析目标。
- [x] 所有 CD 和 Traefik REST 调用只通过 SSH executor；Linux/Windows 支持边界、host-managed registry、private-key redaction 和取消对账按已确认安全规则执行。
- [x] 后端通过 task check、go test ./cmd/... ./internal/...；前端通过 yarn --cwd web lint:fix、yarn --cwd web typecheck。

## Decisions

- Project 是轻量租户边界：拥有资源 scope、成员授权和唯一部署 Environment；不是全局资源的可选标签。
- Environment 的 project_id 与 Gateway binding 均采用逻辑外键；应用层在跨聚合操作中校验存在性与同 Project 归属。
- Project code 是稳定基础设施 identity，创建后不可修改；Environment code、remote workspace segment 和 Gateway network 名从它派生。
- Project 创建需要一次性提供 Environment 与 deploy key；部署 key 只加密存储，普通 detail/export/import/MCP/log/snapshot 不返回私钥。
- Environment disable 阻止新的 provision/deploy，但保留项目、服务、网关、路由与部署历史。

## Risks

- 将 Project 改成租户边界要求恢复所有先前误删的 project_id、membership 和 active-project contract；任何全局列表或缺失 scope 都是隔离漏洞。
- Project create 的跨聚合事务涉及 Project、Credential 与 Environment，必须复用现有 transaction runner，不能产生半初始化目标。
- Gateway/Route 需要同时校验 Project、Environment 和 backing runtime 归属，不能沿用全局 active Gateway。
- Windows native OpenSSH + WSL2 Docker Desktop 仍需真实 integration target 证明；不得使用本机或 Cygwin fallback 声称支持。

## User review notes

- 用户最新确认：Project 保留，并具有类似租户的资源与访问边界意义。
- 用户确认：一 Project 一 Environment，一 Environment 一 Gateway；切换 Project 即切换部署 Environment。
- 已确认的平台、registry 宿主机责任、Environment 不可删除、逻辑外键和 SSH 内 Traefik REST 规则继续有效。
