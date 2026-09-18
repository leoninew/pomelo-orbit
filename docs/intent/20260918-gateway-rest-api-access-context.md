# Gateway REST API 访问上下文
最后修改时间: 2026-09-18 14:54:44

Review status: Accepted

Mode: standard

## Background

`GatewayConfig.rest_api_url` 当前作为 Gateway 定义的一部分持久化并随 Project CD 接管包原样导入导出，但字段只表达 URL，没有表达由哪个网络命名空间访问。

远程 Tencent 环境已验证：运行中的 `pomelo-orbit` 容器和 `traefik-traefik` 容器同属 Docker `traefik` 网络，后者具有 `traefik` DNS alias。源 Orbit 的 Gateway 配置为 `http://traefik:8080`，其 `/api/route/traefik` 请求成功。Gateway 初始 Version 同时将 Traefik API 的 8080 端口以 `127.0.0.1:8080` 发布到部署宿主机。

接管后的 Project 使用 SSH Environment 时，RouteManager 通过 SSH Runtime 在远程宿主机执行 `curl <rest_api_url>/api/...`。宿主机不使用 Docker embedded DNS，不能解析 `traefik`，请求以 curl exit status 6 失败。相同 URL 因控制面执行位置从 Docker 网络变为 SSH 宿主机而失效；包内容没有被改写，也不是 Traefik 路由数据本身错误。

需要区分三种执行位置，而非只区分 local / SSH target：

1. Orbit 作为二进制或宿主机服务安装在部署宿主机上，并管理 local Environment。REST 请求直接由部署宿主机发出。
2. 另一台 Orbit 通过 SSH 管理该部署宿主机。REST 请求由 SSH session 在同一部署宿主机发出。
3. Orbit 运行在 Docker 容器中，并管理 local Environment。REST 请求由 Orbit 容器所在的 Docker 网络发出。

前两种场景处于同一部署宿主机网络命名空间，应该使用相同的 REST 可达性契约；SSH 只改变命令如何到达宿主机，不能成为另一套 URL 语义。第三种场景才需要处理容器网络与宿主机网络之间的明确边界。

当前实现的 `ListRouters`、`ListServices`、Gateway readiness、Route snapshot 发布均复用这一调用路径。现有接管任务规定 Environment 和 Gateway 定义原样迁移，导入不部署、不探测、不发布，因此不能在导入路径按主机名或 URL 内容猜测并修复该地址。

## Goal

1. 为 Gateway REST API 明确可迁移的地址语义与访问执行上下文，消除 Docker 网络 URL 和部署宿主机 URL 被同一字段隐式混用的模型缺口。
2. 使 Route 列表、同步预览、确认同步、Gateway 部署后的 readiness 和 REST snapshot 发布，在 local 与 SSH Environment 下使用同一套明确、可验证的访问契约。
3. 让 Project 接管保留源配置数据，不因导入位置或失败结果对 `rest_api_url` 进行隐式 URL 改写、hostname 猜测或 runtime 回退。
4. 保持普通 HTTP Route 的自定义上游自由：用户可配置任意合法的 HTTP(S) 目标地址；本任务只处理 Orbit 到受管 Traefik 管理 API 的访问，不限制业务路由上游。

## Non-goal

- 不修改 Route 自定义 `target_url` 的可用范围，不限制外部域名、国家、IP 或协议以外的业务选择。
- 不通过硬编码 `localhost:9020`、`localhost:8080` 或 `traefik` 修复单个环境。
- 不在接管导入过程中执行 Docker、SSH 探测、Traefik publish 或对已有容器做网络变更。
- 不基于 URL 字符串、Docker alias、连接失败或 hostname 自动猜测目标网络并重写数据。
- 不将源 Orbit 作为目标 Orbit 的运行时依赖，也不为两套控制面建立 fallback。
- 不在本任务中扩展到用户自管 external Traefik、多 Gateway 或 Kubernetes。
- 不在本任务中新增 Gateway API 原始响应体的敏感日志审计；该项需要独立界定日志保留和脱敏策略。

## User scenarios

1. 用户将 Orbit 安装在部署宿主机上并管理 local Environment。Gateway REST API 通过宿主机可达地址访问，路由读取和发布成功。
2. 用户从另一台 Orbit 通过 SSH 管理同一部署宿主机。该请求与场景 1 使用相同的宿主机 REST 可达性契约，不能因 SSH 模式而要求不同 URL 或出现 Docker DNS 失败。
3. 用户在 Docker 网络内运行 Orbit 并管理 local Environment。Gateway REST API 的网络地址可被声明的容器执行上下文访问，路由读取和发布成功。
4. 用户变更 Environment target 或接管到不同执行拓扑时，系统不会悄悄把 Gateway REST 地址替换成猜测值；不满足已声明访问契约时，应给出能定位网络上下文的错误或在显式配置步骤中要求用户决定。
5. 用户创建普通 HTTP Route 并填写外部上游，例如政府网站，系统按 URL 作为 Traefik 业务上游保存和发布，不将其当作 Gateway 管理地址解析或限制。

## Acceptance

- [ ] Gateway REST API 的 URL 语义、访问执行位置及其与 Environment target 的关系被明确建模并写入活文档。
- [ ] 任何调用 Traefik REST API 的路径使用同一访问抽象，不能分别在 Route 列表、同步、部署后发布中采用不同网络假设。
- [ ] Docker network alias 与宿主机 loopback 地址不再由未声明的控制面进程位置决定是否可用。
- [ ] Orbit 运行在部署宿主机与外部 Orbit 通过 SSH 管理该宿主机时，使用同一个 Gateway REST API 访问契约，并覆盖路由读取和 snapshot 发布主流程。
- [ ] Project handover 保持包中业务定义原样；不会为 `rest_api_url` 添加 URL 内容启发式、导入补丁或 silent fallback。
- [ ] local 和 SSH 两种 target 均有覆盖主流程的自动化验证；覆盖 Docker 网络地址和宿主机地址的边界，以及接管后首次路由读取或发布。
- [x] 用户可读到实际失败的访问上下文、执行位置和已配置地址；错误不再只呈现泛化的“Traefik is unavailable”。

## Open questions

1. `rest_api_url` 的规范所有者与可见性是什么：继续作为 GatewayConfig 的完整配置，还是将“如何从 Environment 访问管理 API”的部分拆为 Environment-owned runtime access 配置？
2. 规范访问上下文选择部署宿主机、Gateway Docker 网络，还是显式支持两种模式？无论选择什么，宿主机 Orbit 与 SSH 到该宿主机必须共用同一语义；容器化 Orbit 的 local target 是需要单独纳入的另一执行上下文。
3. 若目标 Environment 的访问上下文与包内地址不兼容，导入后应在何处要求用户显式配置：接管表单、Project Initialization，还是 Gateway 配置编辑流程？

## Decisions

- 采用标准模式 / standard；本轮先完成 Intent，不进入 Plan 或实现。
- 此任务与 `20260917-project-cd-handover-transfer` 相邻但独立：接管任务的“Environment 与 Gateway 原样迁移”仍是事实依据，本任务解决该数据在不同执行拓扑下的管理 API 可达性契约。
- `http://localhost:9020/routes` 不作为当前 RouteManager 的调用路径或修复目标；当前实现调用 `<rest_api_url>/api/http|tcp/...`。
- curl exit status 6 视为 DNS 解析失败诊断，不等同于 HTTP 响应、Traefik router 状态或业务 Route 上游失败。
- “Orbit 在宿主机运行”与“通过 SSH 在该宿主机执行”是同一 REST 网络位置；实现不得因 target_type 不同而给这两个场景赋予不同的 URL 解释。
- local Environment 且 Orbit 运行在容器中时，只调用 container endpoint；local Environment 的宿主机 Orbit 与 SSH Environment 时，只在目标宿主机调用 host endpoint。请求失败不能改变 endpoint 或触发业务请求重试。
- REST 调用失败向用户返回执行位置和已配置 endpoint，但不返回 curl/SSH 原始错误；敏感响应日志审计不纳入本任务。

## Risk

- 将地址作用域仅隐藏在 URL 文本中会继续导致“源环境可用、接管后不可用”的不可预期故障，并影响路由读取、同步与部署成功后的配置发布。
- 将 host URL、Docker network URL 或外部 URL 强行归为单一隐式约定，会使另一种已支持运行拓扑失效；必须先确认规范执行模型。
- 为解决单例而在导入路径特殊处理 `traefik` 或 `localhost`，会破坏接管包原样恢复的承诺，并形成不可维护的 hostname heuristic。
- Gateway API 的完整响应日志当前可能包含敏感配置，继续保留会扩大密钥泄露面；无论是否纳入本任务，都需要显式处置。

## User review notes

- 用户要求对该问题使用 SpecFlow 标准模式开始分析。
- 用户强调接管的预期是“能用就能导出，能导出就能导入”，不接受通过导入期猜测、修复或重解释业务数据来掩盖模型问题。
- 用户明确指出普通 Route 的上游由用户决定；不得将 Gateway 管理 API 网络问题扩展为对业务上游的限制。
- 用户要求纳入宿主机已安装 Orbit 发起 REST 调用的场景，并与远程 SSH 管理该宿主机的场景一并分析。
