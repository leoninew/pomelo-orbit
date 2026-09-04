# 远程 SSH 部署环境规格
最后修改时间: 2026-09-04 08:02:27

Review status: Accepted

## Requirement basis

依据远程 SSH 部署环境需求。本规格保留 Project 作为租户边界，并定义 Project 1:1 Environment 1:1 Gateway 的 SSH-only CD 模型。

## Overview

Project 是 membership 和资源隔离边界。Project 解析其唯一 Environment，Environment 解析其唯一 Gateway。Project-scoped Application、Service 和 Route 创建 Deployment snapshot 后，SSH executor 在该 Environment 的远程 Docker Compose target 执行操作。

## Project as tenant boundary

Project 保留现有 project_member 授权语义。Application、Repository、Credential、Pipeline、PipelineRun、Deployment、Route、Gateway backing Application 与所有列表/详情/写入操作均维持 project_id scope。HTTP、MCP 和 Web active-project 继续显式传递或保存当前 Project；资源 ID 查找后必须校验 membership。

Project code 变为创建后不可变的基础设施 identity。它用于派生 Environment code、Compose namespace、remote workspace segment 和 Gateway network 名，避免修改代码导致已部署运行时漂移。Project 名称与 active 状态仍可更新；停用 Project 前必须先 disabled Environment。

## Environment relationship and lifecycle

Environment 增加 project_id NOT NULL UNIQUE 逻辑引用。一个 Environment 只能属于一个 Project，一个 Project 只能有一条 Environment 记录。Environment 没有独立 create/delete surface：

- POST /api/project 使用一个请求级事务创建 Project、deployment_ssh_private_key Credential 与 Environment。
- GET 和 PUT /api/project/:project_id/environment 读取或更新该唯一 Environment。
- POST /api/project/:project_id/environment/probe 运行显式 Probe。
- Environment 只可 active 或 disabled，不能删除，也不能迁移到另一 Project。

Environment code 等于 Project code，并由服务端填充；外部请求不能修改。字段继续保存 platform、host、port、username、workspace root、credential ID/revision、host-key fingerprint、target revision、Probe metadata 与可选 gateway_application_id。

Project create 的 Environment payload 必须完整，包含专用 deploy key 名称、private key material、可选 passphrase 和 SSH target。使用既有事务模式：先写 Project，再加密写 Credential，再写 Environment；任一步失败即回滚。private key 仅在此写入路径出现，不进入响应、日志或 snapshot。

## Gateway and Route relationship

Environment 的 gateway_application_id 是唯一且不可被第二个 Gateway 覆盖的逻辑引用。Gateway provision 在同一事务中校验 Project membership、Environment active/probed、无现有 binding，然后创建 Project-scoped backing Application/Version/Service，并绑定 Environment。Gateway 的 Application project_id 必须等于 Environment project_id。

Route 保留 Project scope。创建、更新、enabled list、preview/confirm 和 certificate materialization 均通过 Project 解析 Environment；managed target Application/Service 必须属于同 Project。Gateway deploy/restart 只发布该 Project Environment 的 route snapshot。全局 active Gateway 和全局 traefik network 被移除。

## Service, Deployment and runtime identity

Service 不保存重复的 environment_id，因为 Application 的 Project 已唯一确定 Environment。部署、运行时查询、compose preview、logs、inspect、network doctor 和 HTTP probe 先验证 Project membership，再解析 Environment。

Deployment 以正式列保存不可变的 project_id、environment_id、Environment target revision、deploy credential ID/revision、可选 Gateway Application ID 与 effective plan hash；`options_json` 只保存命令选项和 Gateway 配置快照。执行时缺少目标列、Environment disabled、Probe 过期或 revision 不一致均 fail closed，不从控制面本机或其他 Project 推断目标。

## SSH and platform contract

只使用 golang.org/x/crypto/ssh、private-key authentication、严格 host-key fingerprint 校验与 SFTP。Linux 使用受控 POSIX sh 和 curl；Windows 使用 `powershell.exe -NoProfile -NonInteractive` 调用宿主机 `docker.exe` 与 `curl.exe`，由 WSL2 Docker Desktop 提供 Linux container engine。禁止 password auth、PTY、forwarding、系统 ssh/scp、用户 command input、Cygwin/MSYS/Git Bash/WSL SSH server 与 Windows Containers。

registry login、CA、DNS、网络及多 registry 配置由目标宿主机维护，Executor 不保存或改变 Docker login 状态。

## API, MCP and Web

保留 Project API、Project selector、Project detail 和 membership 管理。Environment 在 Project detail 中呈现和配置，而不是作为跨 Project 的独立导航资源。当前 Project 切换后，资源页面继续用既有 project_id query contract，Environment 状态/平台可作为上下文展示。

MCP tools 保留 project_id 作为授权和目标 scope。部署、Gateway 和 Route 工具在服务器端根据 Project 解析唯一 Environment；不得接受任意 environment_id 绕过 tenant scope。

## Data and migration boundary

不修改已执行 migration。新增三数据库 migration 在 existing Project schema 上创建 environment.project_id 唯一逻辑引用及 Deployment snapshot 字段；不删除 Project 或任何既有 project_id。旧数据库升级必须先为每个 active Project 生成完整 Environment 配置或 fail closed，不允许无 target 的 Project 进入可部署状态。

## Risks

- Project create 的 atomic Environment bootstrap 是跨聚合事务，必须先覆盖 rollback 和 sensitive-data redaction。
- Project code immutability 是破坏性 API 行为，现有 code 更新请求必须被明确拒绝。
- 任何 Route/Gateway 跨 Project target 校验遗漏都会破坏租户隔离。
- 真正支持 Linux/Windows SSH 仍依赖可复现的 integration hosts。

## User review notes

- 用户明确将 Project 定义为类似租户的边界，而非删除的遗留领域。
- 用户明确一 Project 一 Environment、一 Environment 一 Gateway，Project 切换即环境切换。
