# Environment 工作目录归属需求
最后修改时间: 2026-09-09 18:27:13

Review status: Accepted

Mode: standard

## Background

工作目录是 Environment 的配置项，和 target type、SSH host 一样：在环境管理页面新建/编辑时一起保存，Probe 与部署时从已保存的 Environment 取出使用。

当前只有 SSH 走这条路。local 的工作目录来自进程配置 `workspace.deployment`：环境页只读展示、不入库、local runtime 在启动时焊死 yaml。这不是 Environment 配置。

已接受的 `20260902-remote-ssh-deployment-environments` 曾规定 local 工作目录取 `workspace.deployment` 且不保存。本需求废止该条。CD 工作目录不属于 yaml，也不用 yaml 给 Environment 填默认值。

## Goal

1. 工作目录只通过环境管理页面（及同等 MCP 更新）随 Environment 一起保存；local 与 ssh 都是如此。
2. Probe、部署、runtime 只读取已保存的 Environment 工作目录；进程配置不是运行时来源，也不是默认值来源。
3. 环境页编辑 local 时，工作目录与 state、target type 一起提交；切到 ssh 时与 host/user 等一起提交。缺工作目录不能保存为可用目标。
4. 改工作目录抬 `target_revision`、清旧 Probe；已排队 deployment 因快照不一致失败。
5. 从 yaml 去掉 CD 工作目录这项职责。`workspace.deployment` 不再表示 Environment 工作根。控制面部署日志使用独立目录，不跟 Environment 工作目录绑在一起。

## Non-goal

- 不改 local / ssh 语义，不把 `127.0.0.1` 解释为 local。
- 不把 `workspace.pipeline`、Docker socket、Traefik、镜像仓库搬进 Environment。
- 不把 local 平台、主机名、当前用户当成可编辑配置；它们仍是控制面展示事实。
- 不在 yaml / Settings 页提供工作目录默认值，也不做「配置为空则回退 yaml」。
- 不自动搬迁已落地的 compose 文件。
- 不并入分层整改：DTO view、Prober 端口、Runtime 装配、errors/config/routes/backend.md 归 `20260909-environment-architecture-boundaries`。

## User scenarios

1. 成员打开环境管理页编辑 local：填写工作目录，与其它环境信息一次保存；之后 GET/MCP 读到的是刚才保存的值。
2. 成员把目标改成 ssh：在同一表单里填平台、host、user、工作目录并保存；部署用这份库存配置。
3. 成员改已有工作目录并保存：必须重新 Probe，新的 deploy 写到新路径下的 `<service_code>`。
4. 新建 Project 后进入环境页：在保存 Environment 时给出工作目录；未保存可用工作目录前，Probe/部署失败并提示补全。
5. 部署日志仍在控制面独立日志根，不出现在环境页，也不随工作目录改动而移动。

## Acceptance

- [ ] 环境页（HTTP）与 MCP 更新 Environment 时，local 与 ssh 都能提交并持久化 `workspace_root`。
- [ ] GET Environment 的 local/ssh 工作目录来自库存，不是 yaml 投影。
- [ ] 工作目录为空或非法时不能把 Environment 存成可部署目标；Probe/runtime 拒绝空工作目录。
- [ ] local 与 ssh 的 service 目录均为 `<已保存 workspace_root>/<service_code>`。
- [ ] 修改工作目录增加 `target_revision` 并清除 probe freshness。
- [ ] 代码路径中 local runtime / handler / MCP 不再读取 `workspace.deployment` 作为工作根或默认工作根。
- [ ] yaml 不再承担 CD 工作目录配置；活文档删除「local 工作目录 = workspace.deployment」。
- [ ] 部署日志根与 Environment 工作目录分离。
- [ ] `task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用标准模式 / standard。
- 工作目录是环境管理页的职责：新建/编辑时与环境信息一起保存，使用时从 Environment 读取。
- 不用 `workspace.deployment`、也不用任何 yaml 键给新 Environment 填工作目录。表单占位可用产品常量（与现网 SSH 占位 `~/.pomelo-orbit` 同类），用户仍须保存后才生效。
- Project 自动创建的 local Environment 在用户于环境页保存工作目录之前，不是可部署目标。
- 废止「local 工作目录取控制面 yaml 且不保存」。
- 协议仍用 `local.workspace_root` / `ssh.workspace_root`；列沿用 `environment.workspace_root`。
- 存量空 local 的一次性回填若需要保持旧目录能继续用，只允许迁移时把当时磁盘上的旧路径写入 Environment；回填后运行时不再读 yaml。
- 不自动搬家。
- 分层边界（view/Prober/组合根/既有分层）归 `20260909-environment-architecture-boundaries`；本需求只定义工作目录由环境页保存、使用时从 Environment 读取。

## Risk

- 新建 Project 后若用户未在环境页保存工作目录就去部署，会失败；需要页面和错误文案指回环境页。
- 存量库 local 的 `workspace_root` 为空，必须有一次性数据回填或强制用户打开环境页保存，否则现网 local 部署会中断。
- 容器内 local 路径必须是 Docker 可见路径；这由 Probe 检查用户保存的值，不靠 yaml。
- 日志若仍写在旧 `workspace.deployment` 树下，会和「工作目录已归环境页」的心智冲突，必须拆开。

## User review notes

- 用户确认：工作目录是 Environment 配置，不论本地还是远程。
- 用户否定用 yaml 作为默认路径或运行时来源；应在环境管理页面新建/编辑时随环境信息保存，使用时取出。
- 后续用户决定无兼容地重组 `000039`–`000042`，并就地同步开发库迁移记录；该项由独立变更完成。
