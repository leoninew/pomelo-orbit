# Project / Environment 配置归属、初始化与 MCP
最后修改时间: 2026-09-12 08:58:36

Review status: Accepted

Mode: strict

## Background

Project 是部署运行边界。一个已完成配置的 Project 具有其唯一的 Environment 和 Gateway；新建但尚未初始化的 Project 可以没有二者。Environment 已持久化 target type、local/SSH target 和 `workspace_root`；GatewayConfig 已持久化 REST URL、base domain、readiness、入口策略和 ACME 配置；Gateway Version / Component 已声明镜像、端口、mount 与 Traefik static topology。

目标空库和 Project 切换流程统一为 Project 初始化 Wizard：打开 Project 时检查它是否已有完整的 Environment、Gateway 和默认 Gateway Service。缺失任一资源即跳转 Wizard，由它在当前 Project 内完成 local 或 SSH Environment 配置、Probe 与 Gateway 创建；Gateway 最终只进入待部署状态，不自动部署。切换至另一个未初始化 Project 时走同一 Wizard，而不是复用前一个 Project、进程 YAML 或本地 host 的资源。

当前空库 seed 分散写入默认 Project 及一套默认部署资源：`000032_seed_identity` 写入用户进入系统所需的默认 Project 和 Project membership，`000037_seed_gateway` 写入完整 Gateway、Version、Service 和 Route，`000040_seed_environment` 写入绑定该 Gateway 的 Environment。默认 Project 是空库的进入点，应保留；部署资源则须由 Wizard 初始化。运行时另有 `project.Service.Create` 无条件调用 `Environment.BootstrapForProject`，而 Gateway 又可从 HTTP 或 MCP provision 创建。这些并列初始化路径造成默认值和数据来源分散。

Windows SSH 目标不能假定控制面已经能够使用部署公钥直接建立远程会话。用户需要在目标 Windows 主机上先执行一段由当前表单生成的 PowerShell 初始化命令，配置 OpenSSH authorized keys、Orbit 工作目录并检查 WSL2、Docker Desktop、Linux containers 和 Docker Compose；命令执行后再由页面测试 SSH 并运行完整 Probe。该命令必须能在 Environment 尚未保存或 SSH 公钥尚未可用时生成。

现有 `traefik.*`、Web `emptyGatewayConfigForm()` 和 seed 也分别保存 image、REST URL、base domain、readiness 与入口策略。它们应收敛为一份统一的 Project 初始化来源配置，供 Wizard 构造初始输入。该来源配置不是 Environment/Gateway 的运行时数据，也不是部署、Route 发布或运行时查询的 fallback；Wizard 提交后，运行时只使用当前 Project 持久化的 Environment、GatewayConfig 和 Gateway Version / Component。

MCP 有两条实际调用路径，不能将它们混为一种“当前 Project”模型：

1. Web Deployment Dialogue 由 HTTP 请求传入 `project_id`，并将该值写入对话会话和 system prompt；每个 turn 临时建立一个绑定当前用户的内存 MCP session。
2. Codex CLI 通过 `go run ./cmd/server mcp` 建立持续的 stdio MCP session；当前只有固定 actor，没有 Project scope。它需要由调用者先显式选择要管理的 Project，并在该 MCP connection 内保留选择结果。

Web 的 `projectStore.activeProjectId` 只保存在浏览器 `localStorage`，Codex stdio 进程不可读取，也不能把它作为 CLI 的全局 Project 来源。两条路径复用同一 Delivery MCP Core：Web Dialogue 在创建短生命周期 client 时注入页面请求的固定 Project scope；Codex CLI 从未选择状态开始，通过 MCP 工具在当前 connection 内切换 scope。两者均只使用当前 scope Project 的持久化资源。

## Goal

1. 将进程配置收敛为控制面运行配置和非运行时的 Project 初始化来源配置；移除已由 Environment、GatewayConfig、Gateway Version / Component 承担的部署运行配置职责。
2. 对 Gateway 建立唯一的配置来源：
   - Environment 保存部署 target 与工作目录；
   - GatewayConfig 保存网关运行属性；
   - Gateway Version / Component 保存初始和后续可编辑的 Compose 拓扑，包括镜像。
3. 将 seed、`traefik.*` 与前端硬编码的初始值收敛为一份统一的 Project 初始化来源配置。它只供 Wizard 产生和确认初始输入，不参与已初始化 Project 的运行时解析。
4. 将 Project 生命周期收敛为“未初始化 -> Wizard 配置并 Probe Environment -> Wizard 创建 Gateway、待部署”。Project 创建不再隐式创建 active local Environment；空库保留可进入系统的 seeded Project，但其部署资源和后续任一 Project 均由打开 Project 后的同一 Wizard 初始化。
5. Project 打开时检查初始化完整性，未完成则直接进入 Wizard；已配置 Project 的运行和部署只读取其持久化资源。环境页和网关页不读取、复制或展示另一 Project 的 Environment/Gateway。
6. Codex CLI MCP 不提供 Project、Environment 或 Gateway 初始化能力。用户以 Project 名称声明目标，Codex 通过 `orbit_list_projects` 解析后调用 `orbit_select_project(project_id)` 选择一个已就绪 Project；项目级运行时工具均使用该 connection 的当前 Project scope，并断言 Environment 存在、active 且最新 Probe 成功，Gateway 工具另要求既有 Gateway/default Service。切换要管理的 Project 必须先再次声明和选择。
7. 保留的 MCP Gateway 更新工具覆盖 HTTP Gateway API 已支持的全部 GatewayConfig 字段；Codex 的 `orbit_select_project` 成功响应明确确认所选 `project_id`、Project 摘要及其就绪资源，后续工具使用该已确认 scope。
8. 删除已废止的全局 CD workspace 配置说明，使运维文档只使用 `Environment.workspace_root` 表示 Service Compose、Gateway 证书与 Route snapshot 的目标目录。
9. Windows SSH 初始化命令必须使用当前表单的 Host、Port、SSH 用户和工作目录，在本地主机 PowerShell 中通过 SSH 执行，并展示可复制的分阶段检查结果。生成命令前可创建或复用当前 Project 的受管部署凭据，但不得展示私钥。

## Non-goal

- 不为多个 Project 共用同一个部署宿主增加限制、端口冲突检查、network 隔离或兼容分支。该用法不属于本任务支持的工作方式。
- 不将数据库、HTTP、日志、worker、CI pipeline workspace 等控制面运行配置迁入 Project 或 Environment。
- 不为旧 `.env` 的 `traefik.*` 或 `workspace.deployment` 保留兼容读取、别名或运行时 fallback。
- 不改变 local / SSH target 的显式分发、Probe、target revision 或 deployment snapshot 语义。
- 不将 Codex CLI 当前 Project 保存为进程配置、用户偏好、数据库字段或跨 MCP connection 的全局状态；它只存在于当前 stdio MCP connection。
- 不为 Web Dialogue 新增跨项目 scope guard、专用 MCP tool schema 或 session 状态；它只使用当前页面 Project 的既有调用上下文。
- 不在 Project 创建、HTTP/MCP 入口或进程启动时为缺失 Environment/Gateway 使用本地配置自动补齐；初始化仅由 Web Wizard 完成。
- 保留空库进入系统所需的默认 Project 和 Project membership seed；不保留默认 Gateway、Environment 或其关联 Service/Route 的部署业务 seed。
- 不处理配置字段的密钥展示、权限或脱敏问题。

## User scenarios

1. 空库中用户进入系统并打开 `000032_seed_identity` 保留的默认 Project。系统检查该 Project 未初始化并跳转 Wizard；Wizard 第一步选择 `local` 或 `ssh` 目标，从统一来源配置取得对应 Environment 与完整 Gateway 的初始输入，确认后只写入该 Project，创建 Gateway Version / Component 和 Service，并显示待部署。环境页继续使用同一套目标表单编辑已保存的 Environment。
2. 用户切换到一个尚未初始化的 Project 时，同一检查直接跳转 Wizard。前一个 Project 的 Environment、Gateway 和运行态均不参与该流程。
3. 用户在 Codex 中声明要管理的 Project 名称。Codex 先调用 `orbit_list_projects`，以名称匹配可见 Project；匹配唯一时调用 `orbit_select_project(project_id)`，选择一个 Environment 已就绪且具有 Gateway/default Service 的 Project。此后查询、更新、准备 Service 和部署均自动作用于当前选中 Project。若多个可见 Project 同名，Codex 展示 name/code 要求用户以 code 澄清；用户要管理另一个 Project 时，先再次声明和选择。CLI 不创建 Project、Environment 或 Gateway；选择或操作未就绪 Project 时，MCP 返回相应环境或 Gateway 未就绪业务错误。
4. Web Wizard 创建 Gateway 时提交统一来源配置产生且经用户确认的完整 Gateway 初始输入；该配置持久化到 GatewayConfig 与初始 Version / Component，之后不再读取来源配置。
5. 用户通过 Codex MCP 管理资源前先声明目标 Project；模型以 `orbit_select_project(project_id)` 提交该声明，MCP 成功响应确认 `project_id`、Project、Environment 与 Gateway 摘要。此后对 Application、Service、Version、Gateway、Route 或 Deployment 的操作均使用当前 scope，并要求资源归属一致；用户改为管理另一个 Project 时必须再次声明和选择。
6. 用户部署到当前 Project 的 Environment 时，工作目录始终来自已保存的 `Environment.workspace_root`，不是 `workspace.deployment` 或 `data/deployment`。

## Current assessment

### 配置归属

| 项目 | 当前实现 | 结论 |
| --- | --- | --- |
| `traefik.image` | `config.Traefik` 注入 Gateway Service，仅用于初始 Gateway Version 的 image | 移入统一初始化来源配置；Wizard 确认后写入 Gateway Version / Component，运行时不读取来源配置 |
| `traefik.rest_api_url` | 已持久化在 GatewayConfig；本地配置仍在 MCP 空输入和 provision 创建时补值 | 移入统一初始化来源配置；Wizard 确认后写入 GatewayConfig，Route/运行时只读库存值 |
| `traefik.base_domain` | 已持久化在 GatewayConfig；Route/域名派生使用库存值 | 移入统一初始化来源配置；归当前 Project 的 GatewayConfig |
| `traefik.rest_ready_timeout` | 已持久化在 GatewayConfig；Route publish / deploy readiness 使用库存值 | 移入统一初始化来源配置；归当前 Project 的 GatewayConfig |
| Gateway 初始 name/code、component、pull policy、ingress、ACME | SQL seed、Web form、Gateway factory 与 MCP schema 分散表达 | 统一初始化来源配置提供 Wizard 输入；固定产品拓扑保留 Gateway factory；确认值持久化到 GatewayConfig / Version / Component |
| `workspace.deployment` | 已从 Go Config 移除，但 `.env.example` 与运维/RAGFlow 文档仍引用 `data/deployment` | 删除残留；CD 工作目录归 Environment.workspace_root |
| `workspace.pipeline` | CI checkout、artifact 与 stage log 的控制面目录 | 保留为进程配置 |
| `logging.deployment_root` | 控制面 deployment execution log 目录 | 保留为进程配置 |

### Project 资源生命周期

| 阶段 | 当前实现 | 目标实现 |
| --- | --- | --- |
| 创建 Project | 无条件 `BootstrapForProject`，插入 active local Environment | 只创建 Project；Environment 和 Gateway 均不存在 |
| 空库本地初始化 | `000032`、`000037`、`000040` 分别 seed 默认 Project、Gateway、Environment | `000032` 继续 seed 可进入系统的默认 Project 和 membership；`000037`、`000040` 不再 seed 部署资源。打开该未初始化 Project 后进入同一 Wizard；它以统一来源配置创建 Environment、完成 Probe、创建 Gateway 和默认 Service；结果为待部署 |
| 切换到未配置 Project | 后端假定 Environment 存在，环境页会进入加载错误；Gateway 列表为空 | 项目打开即检查完整性并跳转 Wizard；不得读取任何其他 Project 或进程配置 |
| 已配置 Project | Environment 的 Gateway binding 指向唯一 Gateway Application | 保持；部署运行时从该 Project 的 Environment、GatewayConfig 和 Gateway Version / Component 读取 |

`CreateGateway` 已要求 Environment active 且最新 Probe 成功，适合作为 Wizard 的最后一步。应删除 Project create 对 `BootstrapForProject` 的隐式依赖，保留默认 Project/membership seed、删除默认 Gateway/Environment 及关联部署资源 seed，并将缺失资源定义为 Project 正常的未初始化状态。

### Gateway 创建入口

1. Web 的 `emptyGatewayConfigForm()` 写死 `traefik:3.6`、`http://localhost:8080`、`lvh.me` 和 `20`；它不读取后端 `traefik.*`，但与 seed、Gateway factory 形成第三套来源。
2. HTTP `GatewayCreateReq` 可以提交完整的 GatewayConfig 和初始 image。
3. `orbit_create_gateway` 只暴露 name、code、REST URL、base domain、image、pull policy、entrypoint 和 TLS mode；遗漏 `traefik_component_name`、`rest_ready_timeout_seconds`、`acme_profile`、`acme_email`、`dns_api_token`。
4. `orbit_update_gateway` 同样遗漏 component name、readiness 和 ACME 字段。
5. `CreateGateway` 的 `applyCreateDefaults` 会用 `config.Traefik` 填充空输入；`orbit_provision_gateway` 在 Gateway 不存在时直接通过这些默认值创建。两条路径都必须删除，provision 不再承担 Gateway 创建。
6. `traefik_component_name` 同时保存在 GatewayConfig 和初始 Version Component。`UpdateGateway` 可以只更新 GatewayConfig 的该值，初始 Version Component 名称并不随之修改；Dashboard Route 再使用 GatewayConfig 值计算目标容器。这是两份可变拓扑配置已经发生漂移的路径。

因此，必须先建立 Wizard 使用的统一初始化来源配置，再删除 seed、`traefik.*`、Web 硬编码和 MCP 的初始化入口；不能让任何已初始化 Project 在运行时回退读取来源配置。Gateway 固定产品拓扑不能同时作为 GatewayConfig 与 Version Component 的可编辑配置保存。

### MCP 调用模式与 Project 初始化

#### Codex CLI stdio MCP

当前实现：

- `orbit_list_projects` 提供 Project discovery；Environment 的 get/update/probe 全部显式接收 `project_id`。
- Application/Gateway/Route 的项目列表与创建工具显式接收 `project_id`；runtime 工具通过 Application 或 Gateway 归属解析 Project 和唯一 Environment。

目标边界：

- MCP 不注册 Project 创建、Environment 初始化或 Gateway 初始化能力。空库和未初始化 Project 必须回到 Web Wizard 完成配置。
- `orbit_list_projects` 返回 Project 的 name、唯一 code 与内部 id，供 Codex 将用户声明的名称解析为精确选择目标。Project name 不保证唯一；出现多个同名可见 Project 时，Codex 必须要求用户提供 code。`orbit_select_project(project_id)` 验证成员关系、Environment active、当前 Probe 成功以及 Gateway/default Service 已存在；成功后写入当前 stdio connection 的 selected Project，并返回 Project、Environment 与 Gateway 摘要。`orbit_get_current_project` 返回已选 Project；未选择时返回 `project_not_selected` 业务错误。
- 选择未完成初始化的 Project 必须返回对应的环境或 Gateway 未就绪业务错误，不从来源配置、seed 或其他 Project 补齐，也不表达 Web 页面跳转。
- 除 `orbit_list_projects` 与 `orbit_select_project` 外，项目级工具不再接收 `project_id`，而是从当前 selected Project scope 获取。按 Application、Service、Version、Gateway、Route 或 Deployment ID 操作的工具须确认其归属等于当前 scope。
- selected Project 只存于一个 stdio MCP session，连接关闭后丢失；不写入 `configs/config.yaml`、`.env`、数据库或 Codex 全局偏好。
- `orbit_create_gateway` 是 Gateway 初始化入口，与上述边界冲突，应从 Delivery MCP 移除。`orbit_provision_gateway` 仅复用已存在 Gateway 和其 default Service，不创建任何 Gateway。

结论：stdio 需要显式的 Project 切换协议，但不是全局 `use_project` 配置。`orbit_select_project` 是当前 MCP connection 的唯一 scope 切换入口；它使 Codex 每次只管理一个已初始化 Project，直到用户要求选择另一个 Project。

#### Web Deployment Dialogue

Web Dialogue 始终从当前页面 Project 发起，`DeploymentDialogueTurnReq.project_id` 已传入当前对话并与 conversation 绑定。Dialogue MCP factory 应将该值作为短生命周期 MCP Core 的固定 Project scope 传入；它不读取浏览器 `localStorage`，也不要求模型调用 `orbit_select_project`。

Project 打开时尚未完成初始化会先进入 Wizard，因此 Web Dialogue 与 Codex CLI 一样在 Environment 已就绪后使用现有 MCP 工具查询、更新、准备 Service 和部署。对话不是全局本地 Gateway 的第二个来源，也不参与 Wizard 的初始化写入。

需要适配的部分：

1. Codex CLI 的 Project 选择只保存于 stdio MCP session，不能由 Web 浏览器的 `activeProjectId` 投射或同步。Web Dialogue 以请求已传的 Project scope 创建 client，不暴露跨 Project 的选择流程。
2. HTTP/Web 必须提供 Wizard 所需的 Environment 创建、Probe 与 Gateway 创建；它们不注册为 MCP 初始化工具。
3. `orbit_create_gateway` 从 Delivery MCP 移除；`orbit_update_gateway` 仅操作 Wizard 已创建的 Gateway，且其 schema 必须覆盖 HTTP Gateway API 的全部现行配置字段。
4. `orbit_provision_gateway` 的“零个 Gateway 时自动创建”与配置唯一来源冲突。它只复用既有 Gateway；Environment 未就绪或 Gateway 不存在时返回对应业务错误。
5. `orbit_select_project(project_id)` 的成功响应必须确认由用户声明名称解析出的 `project_id`、Project name/code、Environment readiness 与 Gateway/default Service 摘要；`orbit_get_current_project` 返回同一 selected scope。Version、Service 与 Gateway Service 不为 MCP 展示额外扩展 `project_id`。
6. MCP 操作文档删除 Project/Gateway 初始化叙述，明确它只操作已初始化 Project；Web Dialogue 也复用这一边界。

## Acceptance

- [ ] 初始化来源配置是 Wizard 的唯一数据来源；它统一替代 `traefik.*`、Web form 常量和 Gateway/Environment 业务 seed，且不参与已初始化 Project 的运行时解析或 fallback。
- [ ] 新建 Gateway 的 REST URL、base domain、readiness、入口/证书配置均由 Wizard 确认输入并持久化到 GatewayConfig；初始 image 来自 Gateway Version / Component 创建输入。
- [ ] Gateway 初始 Component 名称固定为 `traefik`，初始 pull policy 固定为 `missing`；二者不属于初始化来源配置或 GatewayConfig。`traefik_component_name` 从 GatewayConfig、Gateway DTO 与更新入口移除，避免与 Version Component 漂移。
- [ ] `000032_seed_identity` 继续插入空库进入系统所需的默认 Project 和 Project membership；`000037_seed_gateway` 与 `000040_seed_environment` 不再插入默认 Gateway/Service/Route/Environment；三种数据库的 seed 与迁移测试同步收敛。
- [ ] 新建 Project 不再自动创建 active local Environment。打开 Project 时检查完整初始化状态，缺失 Environment、Gateway 或 default Service 则跳转同一 Wizard。
- [ ] Wizard 在当前 Project 完成 Environment 配置、Probe 和完整 Gateway/default Service 创建后返回待部署状态，不自动提交 Deployment；切换任一未初始化 Project 走同一流程。
- [ ] 未初始化 Project 的 Web 页面不显示或复用运行时资源，直接进入 Wizard；已初始化 Project 的运行和部署只读取其持久化资源。
- [ ] Delivery MCP 不提供 Project、Environment 或 Gateway 初始化能力，且不再注册 `orbit_create_gateway`；所有项目级运行时工具断言 Environment 已存在、active 且最新 Probe 成功。
- [ ] `orbit_update_gateway` 覆盖 HTTP Gateway API 的全部现行配置字段；`orbit_provision_gateway` 只复用已有 Gateway/default Service，Gateway 缺失时返回 Gateway 未就绪业务错误。
- [ ] Codex 用户以 Project 名称声明要管理的目标；模型由 `orbit_list_projects` 以 name 解析，匹配唯一后才使用内部 `project_id` 调用 `orbit_select_project`。同名时模型展示 name/code 要求用户用唯一 code 澄清。MCP 成功响应确认该 Project、Environment readiness 与 Gateway/default Service，之后由 `orbit_get_current_project` 可查询。Project scope 不持久化到本地进程配置、数据库、用户偏好或下一次 connection。
- [ ] 除 discovery/selection 工具外，Codex CLI 的项目级 MCP 工具不再接收 `project_id`，均使用当前 selected Project；资源 ID 工具验证其归属与当前 scope 一致。
- [ ] Web Deployment Dialogue 和 Codex CLI MCP 均只在 Environment 已就绪后执行运行时操作。Web Dialogue 由请求的 `project_id` 固定 scope，Codex CLI 由 connection-local selection 固定 scope，不产生第二套全局 Gateway 或 Project 状态。
- [ ] Version、Service 与 Gateway Service 的领域 DTO 不为 MCP scope 额外添加 `project_id`；MCP 以选择确认和当前 scope 查询表达 Project 上下文。
- [ ] `.env.example`、`configs/config.yaml` 和活运维文档不再将 `workspace.deployment` 或 `data/deployment` 表述为全局 CD workspace；示例使用 `Environment.workspace_root`。
- [ ] `workspace.pipeline`、`logging.deployment_root` 及其他控制面配置不被迁入 Project / Environment。
- [ ] 变更后运行项目约定的 Go、前端检查，并补充 Wizard redirect、seed 清理、已初始化 MCP 前置条件和 Gateway MCP contract 的最小测试。
- [ ] Windows SSH 初始化命令可在保存/测试前生成，使用当前 Host、Port、SSH 用户和工作目录，远端完成部署公钥、工作目录及 WSL2/Docker/Compose 检查；命令执行后可继续测试和 Probe。
- [ ] 环境保存失败后重试不会因遗留的受管 `deployment-ssh` 凭据触发同名冲突；非受管同名凭据仍返回冲突。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用严格模式 / strict。
- Web Project 切换是浏览器 `activeProjectId` 页面上下文；Codex CLI MCP 不读取或同步该浏览器状态。
- Codex 用户先以 Project 名称声明要管理的目标。模型通过 `orbit_list_projects` 匹配唯一 name 后，以内部 `project_id` 调用 `orbit_select_project`；同名时由用户以唯一 code 澄清。MCP 仅在确认该 `project_id` 的成员关系、Environment readiness、Gateway/default Service 后设置当前 scope 并返回确认结果；切换 Project 必须再次声明和选择。
- Codex 项目级 MCP 以 selected Project scope 作用域；资源 ID MCP 以持久化归属定位 Project 与 Environment，并要求与该 scope 一致。
- Web Dialogue 复用当前页面 Project 的资源生命周期，不建设额外的 project-scope guard 或独立 MCP tool surface。
- Wizard 通过 `ProjectInitialization` application boundary 按“保存 Environment -> Probe -> 创建 Gateway bundle”执行；它是部署资源初始化的唯一入口。Probe 失败保留未就绪 Environment 供同一 Wizard 重试，Gateway bundle 只在 Probe 成功后原子创建。
- Gateway Component 名称固定为 `traefik`，初始 pull policy 固定为 `missing`。二者不是 Project 初始化来源配置，不保存为 GatewayConfig，也不在 Gateway 表单或 MCP 更新入口出现；后续 Version Component 编辑继续使用通用显式规则。
- `orbit_provision_gateway` 仅准备已有 Gateway 的 Service；Gateway 不存在时返回 Gateway 未就绪业务错误，不接受 provision 内隐式创建、MCP Gateway 创建、页面跳转语义或全局默认配置补值。
- Project 初始化唯一入口是 Web Wizard。它检查当前 Project 的完整性、使用统一来源配置完成 Environment/Probe/Gateway 写入；Project 创建、seed、MCP 和进程启动均不初始化部署资源。
- 统一来源配置只生成 Wizard 初始输入；已初始化 Project 的运行时只使用 Environment、GatewayConfig 与 Gateway Version / Component 持久化值。
- Delivery MCP 不提供 Project/Environment/Gateway 初始化，移除 `orbit_create_gateway`；项目级运行时工具统一断言 Environment 已就绪，Gateway 工具另断言已有 Gateway/default Service。它和 Web Dialogue 不承担页面跳转语义。
- 不支持也不防御多个 Project 复用同一部署宿主；固定 network、80/443 与相关冲突不是本任务范围。
- 本地配置系统只保留控制面运行配置和 Wizard 的初始化来源配置；Environment、GatewayConfig 与 Gateway Version / Component 是部署运行配置的唯一归属。
- 不添加兼容层、别名或全局默认值 fallback。
- `project_initialization.environment.local_workspace_root` 与 Environment `workspace_root` 使用 `~` / `~/...` 或平台绝对路径，配置、API 和界面都原样展示和存储；local 使用时把 `~` 解释为控制面用户主目录，SSH 使用时解释为远端登录用户主目录。不以加载配置时翻译成绝对路径。
- Windows SSH 初始化命令属于 Web 辅助流程：公钥由受认证的 Project 初始化接口按 Project 生成/复用，命令只使用当前表单参数；用户执行命令后仍须通过页面 SSH 测试和 Environment Probe。

## Risk

- 删除默认部署资源 seed 会改变全新数据库的 Gateway、Environment 与关联资源数量，但空库仍保留可进入系统的默认 Project；迁移测试、集成测试和本地开发初始化路径必须改为经 Wizard/factory 建立部署前置资源，不得偷偷恢复 Gateway/Environment seed fixture。
- 统一来源配置、Wizard form、Gateway factory 与 HTTP DTO 同时变动，需用契约测试锁定“来源配置只用于初始化、运行时仅读持久化值”的边界。
- Project 创建从“总有 Environment”变为“可未初始化”后，现有 Environment、Gateway、Deployment 调用链中假定必有 Environment 的读取、HTTP 404 和页面加载状态需要逐一改为 Wizard redirect 或只在部署/运行时操作时要求存在。
- 删除 `data/deployment` 文档残留时，不能误删 `logging.deployment_root` 或 CI `workspace.pipeline` 的容器挂载说明。
- Codex selected Project 仅是 stdio connection 的 scope，不得演变为全局当前项目；Project 上下文由 `orbit_select_project` 的确认结果和 `orbit_get_current_project` 表达，不扩散到 Version/Service 领域 DTO。
- Windows 目标可能在部署公钥安装前无法完成密钥认证；命令生成不依赖先保存或先 Probe，避免把 Windows 初始化锁死在尚未建立的 SSH 密钥链路上。

## User review notes

- 用户要求按严格模式记录本任务，并在需求阶段评估 MCP 对 Project / Project 切换的适配。
- 用户澄清 MCP 有 Web Deployment Dialogue 与 Codex CLI stdio 两种调用方式；两者应按各自上下文评估，不可笼统视作同一种 session。
- 用户采纳 `orbit_provision_gateway` 仅复用已有 Gateway 的方案。
- 用户澄清：空库初始化会得到当前 Project 的本地 Gateway 待部署状态；切换到新 Project 时应显示其尚未配置 Environment/Gateway。网页对话固定在当前 Project，不需将跨 Project 调用防御作为本任务。
- 用户确认：使用 Project 初始化 Wizard 统一空库与 Project 切换流程；删除空库业务 seed，将其数据提取为统一初始化来源配置。该配置不是环境运行时数据。MCP 不提供 Project 创建能力，且只在 Wizard 已初始化完成的 Project 上工作。
- 用户澄清：MCP 对前置条件断言 Environment 已就绪，不返回或模拟跳转 Wizard 的页面行为。
- 用户明确：不支持两个 Project 切换到同一部署宿主，不为该错误用法增加兼容或防御。
- 用户要求聚焦业务配置归属；不讨论机密展示、权限或敏感信息问题。
- 用户要求工作目录使用 `~/.xxx` 形态：配置、前后端展示和库存都原样保存，只在使用时把 `~` 解释为对应用户主目录，而不是在加载配置时翻译成绝对路径。
- 用户澄清：`000032_seed_identity` 中让用户进入系统的 Project seed 不删除；仅删除 Gateway、Environment 及关联部署资源 seed。
- 用户确认：Codex CLI MCP 管理哪个 Project，先通过 MCP 显式切换到该 Project；后续 MCP 工具跟随当前选中的 Project。需要评估并提供查询当前 MCP Project 的能力。
- 用户采纳：Wizard 使用 `ProjectInitialization` application boundary；Gateway Component 名称 `traefik` 与初始 pull policy `missing` 为固定产品模板。Codex 用户先声明 Project 名称，模型解析后由 MCP 确认 `project_id`，再将后续操作作用于该 Project scope。
