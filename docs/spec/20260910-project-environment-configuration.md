# Project / Environment 配置归属、初始化与 MCP 规格
最后修改时间: 2026-09-12 08:58:36

Review status: Accepted

Mode: strict

## Requirement basis

本规格落实已接受的 [Requirement](../requirement/20260910-project-environment-configuration.md)。目标是将部署配置和生命周期收敛到 Project、Environment、GatewayConfig、Version / Component，同时用 Web Wizard 统一部署资源初始化，并让 Codex stdio MCP 以 connection-local 的已选 Project 执行操作。

## Overview

Project 创建和空库 identity seed 只提供可进入系统的 Project 与 membership，不再创建部署资源。Project 打开时，Web 读取由持久化资源派生的初始化状态：缺 Environment、最新 Probe 不成功、缺 Gateway/default Service 时，项目级页面统一转到 Project Initialization Wizard；完成后 Gateway 停留在停止态，等待显式部署。

Wizard 的唯一业务入口为 `ProjectInitialization` application boundary。它使用仅供初始化的进程配置生成默认输入，持久化完成后不再读取该配置。该 boundary 组合现有 Environment、Probe 和 Gateway factory 行为，但不把外部 Probe 放入数据库事务，也不维护第二份初始化进度数据。

Windows SSH 目标增加一个辅助初始化命令流程。Web 使用当前表单的 Host、Port、SSH 用户和 Windows 工作目录生成本地主机 PowerShell；服务端按 Project 返回受管部署公钥，命令通过 SSH 在目标写入管理员或普通用户的 authorized keys、创建工作目录，并检查 WSL2 Docker Desktop distribution、Docker Engine、Linux containers、Docker Compose 和 daemon。该流程不要求 Environment 已保存或部署密钥认证已经成功；执行后由用户回到页面继续 SSH reachability test 和 Probe。

Codex stdio MCP 从未选择 Project 的状态启动。用户以 Project 名称声明目标，Codex 先列出可见 Project，以唯一 name 匹配，必要时请用户提供唯一 code，再以内部 `project_id` 调用 `orbit_select_project`。成功结果确认 Project、Environment 和 Gateway readiness；后续工具从当前 connection scope 取得 Project。Web Dialogue 在 client 创建时以页面请求的 Project 预先完成相同 scope 选择，仍不建立网页跨 Project 流程。

## Design decisions

### Project initialization lifecycle

`ProjectInitialization` 是 application 层新服务，不新增独立领域实体或状态表。初始化状态只由当前 Project 的资源派生：

| 状态 | 判定 | Web 行为 |
| --- | --- | --- |
| `needs_environment` | 不存在 Environment | 显示 Environment 初始化输入 |
| `needs_probe` | Environment 存在但不是 active 或最新 Probe 未成功 | 显示已保存的 target，允许修正并 Probe |
| `needs_gateway` | Environment active 且最新 Probe 成功，但缺 Gateway 或 default Service | 显示 Gateway 初始化确认 |
| `ready` | Environment active、最新 Probe 成功，且 Gateway/default Service 完整 | 允许进入正常项目页面 |

服务公开三类受限 command：

| Command | 行为 | 事务与失败语义 |
| --- | --- | --- |
| `SaveEnvironment` | 按 Wizard 确认的 `local` 或 `ssh` target 创建缺失 Environment，或更新已存在但未完成初始化的 Environment | 只写数据库；保存成功后 Project 进入或保持 `needs_probe` |
| `ProbeEnvironment` | 对已保存 target 执行既有 Environment Probe | 外部 I/O 在事务外运行；复用现有 target revision 条件更新，Probe 失败保留 Environment 和诊断供 Wizard 重试 |
| `CreateGateway` | 仅在最新 Probe 成功后以确认的初始化值创建 Gateway Application、GatewayConfig、四个 Version、Component、default Service、Route 与 Environment binding | Gateway bundle 保持现有工厂的单事务原子性；失败不留下半个 Gateway |

受管 `deployment-ssh` 凭据按 Project 唯一名称复用。Environment 保存或 Windows 命令流程在前次操作失败后再次执行时，不重复创建相同名称；若发现的是非受管同名凭据则返回冲突。

### Windows SSH helper command

- `GET /api/project/:project_id/initialization/environment/deployment-key` 只返回当前 Project 受管部署凭据的公钥；没有 Environment 时也可创建/复用该凭据。
- Web Wizard 和 Environment detail 共用 `WindowsSshInitializationDialog` 与命令生成器；Environment detail 只在已保存的 Windows SSH target 上显示入口。
- 命令是幂等的：authorized key 已存在时不重复写入，工作目录使用 `New-Item -Force`；远端检查失败以明确的 PowerShell 错误结束，便于随后 Probe 诊断。
- 命令生成不替代 SSH 测试或 Probe；测试按钮保持可用，执行命令后用户必须显式测试和探测。

HTTP 只暴露 `/api/project/:project_id/initialization` 下的状态和上述 command；现有通用 Environment/Gateway HTTP 创建入口不再作为初始化入口。Project create 仅创建 Project/membership，不调用 `BootstrapForProject`。`000032_seed_identity` 保留 Project/membership；`000037_seed_gateway`、`000040_seed_environment` 在 SQLite、MySQL、PostgreSQL 中删除 Gateway、Service、Route、Environment 业务 seed。这是用户明确要求对现有 seed 迁移的定向修改。

### Initialization source configuration

以集中 typed `ProjectInitializationConfig` 替代 `config.Traefik`、Gateway form 常量及 Gateway/Environment seed 中重复的可变初值。它只由 `ProjectInitialization` 读取，用于填充并呈现 Wizard 初始输入；用户确认后写入持久化对象，正常 runtime 不读取该配置，也不以它补齐数据库字段。`local_workspace_root` 默认 `~/.pomelo-orbit`：配置加载、HTTP/MCP 输出和库存都不把 `~` 翻译成绝对路径；local runtime 在 Probe/部署时把 `~` / `~/...` 展开为控制面用户主目录，SSH runtime 展开为远端登录用户主目录。

| 初始化来源数据 | Wizard 确认后的持久化归属 |
| --- | --- |
| 默认 local workspace root（`~/.pomelo-orbit`，原样展示/存储；使用时展开 `~`）；Wizard 第一步显式选择 `local | ssh` | Environment |
| Gateway image | 初始 Gateway Version / Component |
| REST URL、readiness timeout、base domain | GatewayConfig |
| default entrypoint、TLS、ACME profile/email/token | GatewayConfig |

Gateway product模板不属于该配置：Gateway Application name/code 固定为产品身份 `Traefik` / `traefik`，不把 Project code 拼进名称；default Service code 为 `<application-code>-default`（即 `traefik-default`）；Component 名称固定 `traefik`；初始 pull policy 固定 `missing`；网络名、端口、socket、mount、resolver YAML 和四个 profile Version 是 Version / Component 的产品拓扑。`missing` 只定义初始 Version 值，后续 Version 编辑继续使用普通 Component 的显式 `pull_policy`。

`traefik_component_name` 从 GatewayConfig model、数据库 schema、repository、proto、HTTP/MCP DTO、Web form 和 Gateway update 输入移除。Dashboard Route 使用固定产品 Component 名称或从绑定 Version 解析，不能再从 GatewayConfig 取得另一份名称。为现有三种数据库增加新迁移以删除该 column，SQLite 使用重建表的既有迁移模式；不保留读取别名或运行时 fallback。

### Web project entry and Wizard

Web 增加 Project Initialization Wizard route。项目选择或进入项目级 route 时，`projectStore.activeProjectId` 对应的初始化状态是唯一检查对象：

1. `ready` 继续原目标页面。
2. 其他状态重定向到 Wizard，不加载 Environment、Gateway、Application、Route、Deployment 或 Dialogue 的运行时数据。
3. Wizard 按派生状态恢复到适当步骤。第一步显式选择 `local | ssh`，并与环境页共用目标表单；使用同一 Project 已保存的数据和初始化来源默认值，不读取上一个 Project 的任何资源。
4. Gateway 创建成功后进入 Gateway/default Service 的待部署视图；不自动创建 Deployment。

Project 管理页、登录页和 Wizard 自身不被该 guard 循环重定向。切换 active Project 时重新读取该 Project 初始化状态，不从浏览器 `localStorage` 外推 Environment 或 Gateway。

### Gateway behavior after initialization

Gateway factory 只接收完整、经 Wizard 确认的 Gateway input，不再有 `applyCreateDefaults` 或 `CreateDefaults` 从进程配置补空值。HTTP `orbit_create_gateway` 对应的 Delivery MCP registration 删除；HTTP Gateway create 也不再是面向常规页面的独立资源创建流程。

`orbit_provision_gateway` 改为只解析当前 Project 已绑定 Gateway 的 `default` Service 并返回停止态 Service。Environment 或 Gateway/default Service 缺失时返回 `environment_not_ready` 或 `gateway_not_ready` 业务错误；不创建 Gateway、Version、Service 或使用初始化来源配置。

Gateway update 仅保留持久化 GatewayConfig 的可编辑属性：name、REST URL、readiness、base domain、entrypoint/TLS 和 ACME 字段。MCP `orbit_update_gateway` 与 HTTP API 覆盖完全相同的字段。Version / Component 拓扑，包括 image 和 pull policy，仍经普通 Version Component 工具修改。

### MCP Project scope

Delivery MCP Core 维护 `ProjectScope`，它只存在于一个 stdio MCP connection：

| Tool / path | Scope 语义 |
| --- | --- |
| `orbit_list_projects` | 无 scope；返回当前 actor 可见 Project 的 id、name、唯一 code 和 active 状态 |
| `orbit_select_project(project_id)` | 选择唯一入口；验证成员关系、Environment active、最新 Probe 成功、Gateway/default Service 完整后写入 scope，并返回 Project、Environment、Gateway/default Service 摘要 |
| `orbit_get_current_project` | 返回当前 scope；无选择时返回 `project_not_selected` |
| 所有项目级工具 | 不再接受 `project_id`，通过 `requireProjectScope` 取得当前 Project |
| Application、Version、Service、Gateway、Route、Deployment ID 工具 | 在既有归属查询后要求资源 Project 等于 current scope |

用户以名称声明是 Codex 的交互语义，不是持久化 MCP 参数。名称唯一匹配由 Codex 在 `orbit_list_projects` 结果中完成；同名时以 code 向用户澄清。`orbit_select_project` 始终接收内部 `project_id`，其成功响应是对用户声明目标的确认。connection 关闭后 scope 消失，不写入 `configs/config.yaml`、`.env`、数据库、浏览器状态或 Codex 用户偏好。

Web Dialogue 和 stdio 继续共享 Delivery Core 的资源工具、输入 DTO 与错误映射。Dialogue factory 改为接收当前 request `project_id`，在 MCP tools/list 前建立该 Project scope；页面对话不执行名称发现或跨 Project 选择。`orbit_select_project` 作为 stdio 的 bootstrap tool 不改变公共资源工具契约。

所有运行时 MCP 操作在 scope 建立后仍断言 Environment readiness；Gateway 专属操作额外断言 Gateway/default Service。MCP 不返回 Wizard 跳转或初始化说明。

## Interfaces

### HTTP and proto

- 新增 Project Initialization 状态、Environment 保存（`local | ssh`）、部署公钥读取、Probe、Linux SSH bootstrap、Gateway 创建的 proto request/response，并在 `/api/project/:project_id/initialization` 下映射单数 API 路由。
- 删除 Gateway create 的常规页面入口及 `traefik_component_name` 的 proto/HTTP 字段；更新 Gateway response、update request 和前端生成类型。
- Project create response 仅表示 Project 已创建，不暗示存在 Environment/Gateway。

### Config

- `configs/config.yaml`、`.env.example` 和 `internal/config` 将可变 Gateway 初值迁入 `project_initialization` typed section，并删除 `traefik.*` 运行时职责。
- `workspace.deployment` 与 `data/deployment` 不再出现在活配置或部署操作文档中；部署工作目录始终来自 `Environment.workspace_root`。

### MCP

- 新增 `orbit_select_project`、`orbit_get_current_project`；保留 `orbit_list_projects` 用于 Project name/code discovery。
- 删除 `orbit_create_gateway`；修改 `orbit_provision_gateway` 为 existing Gateway/default Service 的读取/准备操作。
- 删除所有项目级 MCP input 的 `project_id`，仅 selection tool 保留该内部精确 ID。
- Project scope 输出由 selection confirmation 和 `orbit_get_current_project` 表达，Version/Service/Gateway Service 领域 DTO 不额外添加 `project_id`。

## Affected components

| 区域 | 变更 |
| --- | --- |
| `internal/config`, `configs/config.yaml`, `.env.example` | 新增 typed 初始化来源配置，删除 Traefik 可变运行时初值和全局 deployment workspace 残留 |
| `internal/application/project`, `environment`, `gateway` | 去除 Project bootstrap；新增 `ProjectInitialization` boundary；Gateway factory 输入改为完整显式值；删除 component-name 配置字段与 provision 隐式创建 |
| `internal/model`, `repository`, `sql/schema`, `sql/migration` | 删除 `gateway_config.traefik_component_name`；更新三数据库 schema/repository；按用户决定收敛 Gateway/Environment seed |
| `proto/orbit`, `internal/api/http`, `web/src/api` | 初始化 API/proto；删除 Gateway create/component-name 契约；重新生成 Go/TypeScript 类型 |
| `web/src/stores`, `router`, `views/environment`, `views/gateway`, `components`, `utils` | 初始化状态 store/guard/Wizard；Windows SSH 命令 Dialog/生成器在 Wizard 与 Environment detail 复用；删除独立 Gateway 创建 UI 与 component-name 字段；已初始化详情继续使用 GatewayConfig 编辑 |
| `internal/api/mcp/delivery`, `internal/infrastructure/mcp/delivery`, `application/dialogue` | Project scope、select/current tools、scope-aware资源工具、stdio 与 Dialogue factory scope 注入 |
| `docs/product`, `docs/architecture`, `docs/guides`, `docs/decisions` | 更新 CD model/runtime、MCP direct operations 和决策账本，移除已废止的 seed/default/provision/project-id 表述 |

## Data migration and state transition

新库执行收敛后的 seed 后，默认 Project 没有 Environment/Gateway，首次打开即进入 Wizard。已存在数据库通过后续 schema migration 删除 GatewayConfig component-name column；不保留该字段的兼容读取。Project 在任何时刻可处于 Environment 已保存但 Probe 未成功或 Gateway 未创建状态，Wizard 由资源派生状态恢复，而运行时和 MCP 仅接受 `ready`。

本任务不支持多个 Project 共用一个部署宿主，也不为其建立 host、port、network 或 workspace 冲突处理。现有已部署资源的转换不引入双写、别名或全局默认值路径。

## Verification design

- 三数据库迁移测试确认 `000032` 仍产生可进入的 Project/membership，`000037`/`000040` 不产生部署资源，GatewayConfig schema 不再有 component-name column。
- Project 创建测试确认只创建 Project/membership；初始化 usecase 覆盖 `needs_environment`、Probe 失败重试、Gateway bundle 原子创建和 `ready` 派生状态。
- Gateway factory 测试确认 image 等值必须来自初始化 command，固定 `traefik` / `missing` 写入初始 Component，Gateway update 不再接受 component name；provision 不创建缺失 Gateway。
- Web 测试覆盖空库默认 Project 和新建 Project 进入 Wizard、初始化完成后返回原项目页面、切换到未初始化 Project 不加载旧项目数据。
- Web 测试覆盖 Windows 命令按当前 Host/Port/用户生成、PowerShell 转义、Docker/WSL 检查以及缺少公钥时不生成命令；初始化公钥接口覆盖保存前生成。
- 凭据测试覆盖保存失败后的受管 `deployment-ssh` 重试复用与非受管同名冲突。
- MCP server 测试覆盖名称发现、选择确认、未选择错误、未就绪选择错误、切换 scope、项目级 schema 移除 `project_id`、资源归属与 scope 一致，以及 Web Dialogue 固定请求 scope。
- 运行 `task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，并执行针对迁移和 MCP 的定向测试。

## Risks and alternatives

| 项目 | 结论 |
| --- | --- |
| 页面直接调用通用 Environment/Gateway API | 不采用；会保留多个部署资源初始化入口 |
| 一个包含 Probe 的长事务初始化 API | 不采用；外部 Probe 不应包进数据库长事务 |
| GatewayConfig 保存可编辑 component name | 不采用；已与 Version Component 发生实际漂移 |
| Codex 每次工具调用传 `project_id` | 不采用；用户声明并确认一次目标，后续使用 connection scope |
| Codex 从 Web `activeProjectId` 或全局配置读取 Project | 不采用；浏览器状态不是 stdio MCP 上下文 |
| 将未初始化 Project 作为 MCP 创建资源的入口 | 不采用；MCP 只返回 readiness 业务错误，Wizard 是唯一初始化入口 |

受影响的活文档当前仍描述 GatewayConfig component name、自动 Gateway provision 和 MCP `project_id` 参数；实现与验证阶段必须同步更正，不能将这些文档当作新行为依据。

## User review notes

- 用户接受 Requirement，并要求进入 Spec。
- 用户接受 `ProjectInitialization` application boundary，拒绝页面直接串通用初始化 API。
- 用户确认 Gateway Component 名称 `traefik` 和初始 pull policy `missing` 是固定产品模板，不是初始化来源或 GatewayConfig。
- 用户确认 Codex 用户以 Project 名称声明目标；模型解析后由 `orbit_select_project(project_id)` 确认，后续 MCP 工具使用当前 scope。
