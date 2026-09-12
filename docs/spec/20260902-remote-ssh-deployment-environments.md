# 部署环境目标规格
最后修改时间: 2026-09-12 09:50:42

Review status: Accepted

## Requirement basis

依据“部署环境目标需求”。Project 1:1 Environment 1:1 Gateway 不变；Environment 的执行目标从 SSH-only 扩展为 explicit `local | ssh`。

## Overview

Environment 是目标的鉴别联合：公共 identity/state/revision/Gateway binding 与 target-specific configuration 分离。应用层只解析明确的 `environmentport.Target`；基础设施组合根把它交给 local 或 SSH runtime。调用者不会也不应按 hostname、空字段或运行环境猜测执行位置。

```text
Project -> Environment(target_type)
  local -> Environment.workspace_root/deployment + Docker daemon
  ssh   -> Environment.workspace_root/deployment + SFTP + Docker daemon
                 -> Gateway / Traefik REST
```

## Target model

Environment 公共字段：`id`、`project_id`、`code`、`state`、`target_type`、`target_revision`、Probe 状态、Gateway binding 与审计时间。

- `local`：运行时由 Project Environment 的 `workspace_root/deployment` 与本机 Docker CLI 定义。
- `ssh`：`platform`、`host`、`port`、`username`、`workspace_root`、SSH credential identity/revision、host-key fingerprint。只支持 Linux OpenSSH，或 Windows native OpenSSH + WSL2 Docker Desktop Linux containers。

Environment target type 可显式更新。任何 type 或适用 target configuration 改变均清空 Probe、递增 revision；SSH 仅修改 `workspace_root` 时保留既有部署 credential 与 pinned host-key fingerprint，不重新要求一次性 bootstrap 认证。local 不产生 SSH credential；ssh 产生受管部署密钥，首次成功 Probe 写入 host-key fingerprint。

Linux `POST /api/project/:id/environment/initialize` 接收 bootstrap SSH username 与恰好一种一次性认证（密码或私钥，私钥口令可选）。它们仅用于本次 SSH session，绝不持久化、返回或记录。runner 首先验证配置部署 user 的 Docker daemon/Compose，再要求 bootstrap user 的 `sudo -n`、验证现有 OpenSSH 配置支持公钥认证，最后仅按需写入部署公钥和工作目录。它不安装、启停或配置 Docker/Compose/Docker Desktop/WSL，不修改 Docker 用户组、sshd、firewall 或 Docker 网络。操作成功后调用原有 Probe，以受管私钥验证连接并 pin host key。Windows 不使用该 Linux bootstrap endpoint；Windows 通过 Web 生成的 PowerShell helper 写入受管公钥、创建工作目录并检查 WSL2/Docker Desktop/Linux containers/Compose，执行后再由页面测试和 Probe。

Project creation 不接受 Environment payload。Project service 先创建 Project，再以同一 request transaction 调用 Environment bootstrap，固定写入 active local Environment；因此新 Project 的 Environment 页面始终有可继续编辑的记录。仅当用户在该页切为 ssh 时，Environment service 才创建部署 SSH credential。

## Runtime boundary

`environmentport.Target` 保存 Environment 与仅 SSH 所需的 transient private key。`deploymentport.Runtime` 的所有 workspace、Compose、query、file sync 操作都接收该 Target。

组合根注入一个 explicit dispatcher：

- local runtime 复用历史本机执行语义，materialize `Environment.workspace_root/deployment/<service-code>`，使用本机进程运行 Docker Compose，并在该目录查询 Docker。
- SSH runtime 继续用 SFTP materialize remote workspace，并在 pinned SSH target 运行受控命令。
- dispatcher 仅按 `target_type` 路由；缺少适用 runtime 是配置错误，不尝试另一种运行时。

环境 Probe 使用相同 target discriminator：local 执行本机 Docker/Compose prerequisites；ssh 使用私钥、host key verification 和原有 Docker prerequisites。Gateway network、Gateway REST、Route preview/publish 与 deployment/runtime query 都使用相同 Runtime，因此目标语义一致。

## Deployment snapshot

Deployment 正式保存 `environment_id`、`environment_target_type`、`environment_target_revision`、Gateway application identity。SSH deployment 另外保存 SSH credential id/revision；local deployment 这两个字段为空。worker 重新解析当前 Target 后同时校验 type、revision 与 SSH-specific snapshot，任一不一致都失败。

## API, MCP and Web

Environment request/response 以 `target_type` 为 discriminator：

- Project create request 仅有 name 和 code；创建成功后 Web 设置 active project 并进入 `/environment`。
- local update request 仅有 target type 与 state；response 返回 local target、控制面 workspace、运行平台、主机名、当前用户和 Probe 状态。编辑当前 local target 切到 ssh 时，Web 只在所选 SSH 平台与该平台相同的情况下带入主机名和当前用户。
- ssh update request 在 SSH target object 内提交 platform/host/port/username/workspace root；response 只返回这些配置与 fingerprint。Linux initialization request 额外提交一次性连接认证，成功 response 是完成受管 key Probe 的 Environment；MCP 不接受或输出这类认证。

Web 编辑器切换 type 时仅显示对应表单。active Linux SSH target 显示“初始化部署主机”表单，用户选择密码或私钥并提交；对话框关闭或成功后立即清空一次性认证。Windows SSH target 显示可复制的 PowerShell helper，local 不显示 SSH 初始化入口；helper 执行后仍须完成页面 SSH 测试和 Probe。Gateway 导航固定为当前 Project 的 `/gateway`，由其唯一 Environment binding 解析详情，不展示内部 application id。

## Data and migration boundary

重组后的 `000039` 直接建立最终 Environment schema：SSH-specific columns 可空，并包含 `target_type` 与 Deployment `environment_target_type`。`000041` 仅迁移 develop(38) 已存在的完整 Traefik bundle 关系，不推断 Environment target；空库不创建 Environment 或 placeholder deployment SSH credential。

## Affected components

- Environment model、repository/SQLC、project bootstrap、credential creation、Probe、HTTP/MCP/proto 与 Web form。
- Environment target port、deployment runtime port、execution/query/log paths、Route manager 和 bootstrap composition。
- Local runtime/workspace/command runner、SSH runtime type signatures及其测试。
- MySQL/PostgreSQL/SQLite migration、default seed 与 generated SQLC/proto output。

## Risks

- local runtime 恢复时必须保留容器化控制面所需的 Docker daemon path resolver 与 workspace mount validation。
- 从 ssh 切到 local 会废弃旧排队任务和 SSH deployment credential；这符合不可变 snapshot contract。
- 不把任何 local 成功路径当作 SSH integration 的替代验证。
- Docker prerequisites 是外部依赖；初始化将拒绝未就绪目标，而不是尝试自动修复 Windows Docker Desktop/WSL 或 Linux Docker 安装。

## Alternatives

- 将 host 为 `127.0.0.1` 的 SSH Environment 转为 local：拒绝，丢失认证和 host-key 语义。
- 单个 SSH runtime 内按空 host/credential 切 local：拒绝，字段隐式、故障难诊断且与产品模型冲突。
- 保留 SSH-only response 并为 local 填哑值：拒绝，表单与目标状态会继续误导用户。

## User review notes

- local 与 ssh 是复用模型的两个显式类型，SSH loopback 不特殊处理。
- 历史本机实现现在恢复，不等单独提交后再回看历史。
