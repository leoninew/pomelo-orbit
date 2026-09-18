# Environment 分层边界收敛需求
最后修改时间: 2026-09-09 17:47:28

Review status: Accepted

Mode: standard

## Background

对照后端分层约束审视 Environment / SSH 工作时，原先把展示字段、出站端口、组合根和一批既有分层问题写进同一需求。之后两件事改变了范围：

1. 实现已前进：`model.EnvironmentSSHTarget` 不再挂 install command；Linux 初始化改为 `environment/port.Bootstrapper` + 一次性认证，不再把脚本塞进领域对象。
2. 工作目录产品归属已拆到 `20260909-environment-workspace-ownership`：local/ssh 工作目录都是环境管理页保存的 Environment 配置，不是 yaml，也不是 handler 里的控制面投影。

本需求只保留**分层与装配**问题，不再定义工作目录从哪来、是否可编辑。

当前代码仍在的分层缺口：

1. Environment usecase 仍返回 `model.Environment`；HTTP handler 在构造时读 `runtime.GOOS` / hostname / 当前用户，并用 `workspace.deployment` 填 local 展示；MCP 的 local 输出只有工作目录。展示拼装散在入站适配器。
2. `TargetResolver`、`Bootstrapper` 已在 `environment/port`，Probe 仍是 usecase 包内未导出的 `environmentProber`。
3. `bootstrap/http.go` 与 `bootstrap/worker.go` 各自 `New` local/SSH Runtime。
4. 既有分层：`common/errors` 含 HTTP status；`settings` / `gateway` 仍接收 `config.Config`；`routes.Dependencies` 持有 `*sql.DB`；`docs/architecture/backend.md` 仍写 `application/cd`、`application/ci`。

## Goal

1. Environment 的 GET / Update / Probe / Initialize 返回 `environment/dto` view，不把协议展示塞进 `model.Environment`。
2. HTTP 与 MCP 都消费该 view。local 的平台、主机名、当前用户是控制面展示事实，由组合根注入，不入库、不当运行时路由。`workspace_root` 字段如何写入库存由 workspace 需求定义；本需求只禁止 adapter 再从 yaml 或 OS 私自拼装工作目录。
3. Probe 成为与 `Runtime` / `TargetResolver` / `Bootstrapper` 同级的出站端口，由 bootstrap 注入。
4. HTTP 与 Worker 共用同一个 Deployment Runtime 装配函数。
5. 不改变 local/ssh 显式分发、私钥不进协议/日志、Probe 不进请求级 UoW、不按 hostname 推断类型。
6. `common/errors` 只保留 Kind / Code / Message 与 cause；HTTP status 映射在 `api/http/response`。MCP 在入站适配器内分类。
7. `application/gateway`、`application/settings` 不接收 `config.Config`。
8. `routes.Dependencies` 不持有 `*sql.DB`；tx middleware 由 bootstrap 装配，Probe 仍排除。
9. `docs/architecture/backend.md` 改为现行 `application/<domain>`。

## Non-goal

- 不定义、不迁移、不编辑 Environment 工作目录；那是 `20260909-environment-workspace-ownership`。
- 不恢复 install command，不改 Initialize 的一次性认证语义。
- 不改 local / ssh 产品语义。
- 不把 local 平台/主机/用户持久化或当运行时路由依据。
- 不把 SQLC / Gin / proto 引入 application / model。
- 不拆除全部 model `db` tag，不清空 `common/constant`、`common/runtimeconfig`。
- 不改已验收 HTTP 错误 JSON 形状。
- 不把 Probe 放回请求事务，不引入兼容层。

## User scenarios

1. 维护者读 Environment：HTTP 与 MCP 对同一对象给出同一套 view 字段（含 local 展示用的平台/主机/用户）。工作目录以库存为准，不在本需求里改产品来源。
2. 维护者 Probe / Initialize：usecase 编排；SSH/local 探测与 Linux bootstrap 走注入的端口。
3. Worker 部署：与 HTTP 同一套 Runtime 装配。
4. 错误与配置注入、活文档模块表符合上方 Goal 6–9。

## Acceptance

- [ ] Environment usecase 对读/改/Probe/Initialize 返回 `environment/dto` view。
- [ ] HTTP handler 与 MCP mapper 消费该 view，不再在 handler 内读 OS 拼 local 展示，也不再把 `workspace.deployment` 当作 view 的工作目录来源。
- [ ] `environment/port` 提供 Prober；usecase 只依赖该端口。
- [ ] `bootstrap/http.go` 与 `bootstrap/worker.go` 通过同一函数得到 `deployment/port.Runtime`。
- [ ] 私钥仍不出现在协议；Probe 仍排除请求级 UoW。
- [ ] `common/errors` 不再导入 `net/http`，不再暴露 `StatusCode` / `NewForHTTPStatus` / 带 HTTP status 的 `Classify`。
- [ ] `api/http/response` 集中 Kind → HTTP status；错误 JSON 形状不变。
- [ ] gateway / settings 构造函数不再接收 `config.Config`。
- [ ] `routes.Dependencies` 不再包含 `Database *sql.DB`；原 mutating API 的 UoW 与 Probe 排除仍在。
- [ ] `docs/architecture/backend.md` 不再出现 `application/cd` 或 `application/ci`。
- [ ] `task check` 与 `go test ./cmd/... ./internal/...` 覆盖 view、Prober、Runtime 装配、错误映射和配置注入回归。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard。
- install command 已从 model 移除，本需求不再包含该项。
- 工作目录产品归属归 `20260909-environment-workspace-ownership`；本需求只约束 adapter 不要从 yaml/OS 拼装工作目录。
- 用户此前要求既有分层问题（errors / config.Config / routes DB / backend.md）留在本需求，不另立。
- Environment 读路径改返回 DTO view。
- local 平台/主机/用户仍是启动时注入的控制面快照。
- Probe 端口化不改变「外部 I/O 不进短事务」。
- HTTP 错误 body 只搬家不改形状。

## Risk

- workspace 需求会改 Environment 更新契约和 local runtime 取路径方式；本需求改 view/handler 时必须接那份库存字段，不能再写回 yaml 投影。
- Runtime 装配只共享 runner 构造，不要误合并 command/execution service。
- MCP 目前用 `apperror.Classify` 取 status，映射外移后要在 MCP 适配器接住。
- settings 改快照类型时可能漏键。
- 拿掉 `routes` 的 `*sql.DB` 时须保住 Probe 排除名单。

## User review notes

- 用户要求重新评估本需求：实现已更新，且 workspace 任务已拆分，需要重新分配。
- 用户确认工作目录由环境管理页保存、使用时从 Environment 读取，不属于本分层需求。
- 用户要求进入计划 / Plan，需求按已接受处理。
