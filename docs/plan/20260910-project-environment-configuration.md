# Project / Environment 配置归属、初始化与 MCP 实施计划
最后修改时间: 2026-09-12 08:58:36

Review status: Accepted

Mode: strict

## 输入与实施边界

本计划实施已接受的 [Requirement](../requirement/20260910-project-environment-configuration.md) 和 [Spec](../spec/20260910-project-environment-configuration.md)。实施顺序先收敛数据与应用边界，再替换 HTTP/Web/MCP 调用入口，避免短暂保留第二条初始化或运行时配置来源。

仅 Web Project Initialization Wizard 可以初始化部署资源。Codex stdio MCP 和 Web Deployment Dialogue 只使用已经就绪的 Project。Project、Environment、Gateway、Version / Component 的现有持久化资源是部署运行时唯一来源；`ProjectInitializationConfig` 仅提供 Wizard 的初始表单值，不能参与运行时 fallback。

## 实施步骤

### 1. 收敛初始化来源配置与 GatewayConfig 数据契约

**目标**：以一份 typed `ProjectInitializationConfig` 替换 `TraefikConfig`、Gateway 表单常量和业务 seed 中重复的可变初值；删除 GatewayConfig 中与 Version Component 重复的 component name。

**修改范围**：

- 在 `internal/config/config.go` 定义并加载 `ProjectInitializationConfig`，包含 Wizard 所需的 Environment 默认 target/workspace root 与 Gateway image、REST URL、readiness、domain、entrypoint、TLS、ACME 初始值；更新环境变量绑定、规范化和集中校验。`local_workspace_root` 保持 `~` / `~/...` 或平台绝对路径原样，不在 Load 时翻译成用户主目录。
- 更新 `configs/config.yaml`、`.env.example`、开发环境覆盖配置及 `internal/config/config_test.go`，删除 `traefik.*` 的运行时默认职责，保留控制面 `workspace.pipeline` 和 `logging.deployment_root`。
- 从 `model.GatewayConfig`、Gateway DTO、repository interface、SQL query、SQLC repository、Gateway HTTP/MCP mapper 和 Web generated usage 中删除 `traefik_component_name`。Dashboard Route 的容器目标改为固定产品组件名 `traefik` 或其绑定 Version 的解析结果，不再读取 GatewayConfig。
- 重组 `000039`/`000040` schema migrations：`000039` 直接建立最终 local/ssh Environment schema，`000040` 独立删除 SQLite、MySQL、PostgreSQL 的 `gateway_config.traefik_component_name`；不保留 Environment seed/no-op migration。同步更新 schema source、SQLC 生成物和 migration 测试。

**完成条件**：运行时无法从进程配置补齐 GatewayConfig 或 Component；GatewayConfig 不再表达 component topology，所有生成 DTO 与数据库实现不再含该字段。

### 2. 收敛空库 seed 和 Project 创建生命周期

**目标**：默认 Project 仍使空库用户可进入系统，但它与所有新 Project 一样从未初始化状态开始。

**修改范围**：

- 保留三个数据库的 `000032_seed_identity` Project 与 membership 数据。
- 按用户明确决定，修改 SQLite、MySQL、PostgreSQL 的 `000037_seed_gateway` up/down SQL，删除 Gateway、Version、Component、default Service、Route 等业务 seed；Environment 不再有 seed migration。
- 在 `internal/application/project/usecase/service.go` 删除 `Environment.BootstrapForProject` 调用及其依赖注入，Project create 仅写入 Project/membership。
- 删除或收敛仅为 Project 自动 bootstrap 服务的 public/usecase 入口，保留能由初始化 boundary 复用的 target 校验、持久化与 Probe 基础能力。

**完成条件**：新库和新 Project 都没有预建 Environment/Gateway；identity seed 仍允许用户登录并进入默认 Project。

### 3. 建立 `ProjectInitialization` application boundary

**目标**：将 Environment 保存、Probe、Gateway bundle 创建组织为一个 Project 生命周期服务，而不是页面串调通用资源 API。

**修改范围**：

- 新建 `internal/application/project_initialization` 的 DTO、port、usecase/service 和定向测试；在 `internal/bootstrap` 的 application composition 中注入已有 Project、Environment、Gateway、Repository/UoW、Probe 与 `ProjectInitializationConfig` 依赖。
- 实现资源派生的 `Status(projectID)`：`needs_environment`、`needs_probe`、`needs_gateway`、`ready`。判定只读取当前 Project 的 Environment、latest Probe 结果、Gateway 与 default Service，不建立初始化状态表。
- 实现 `SaveEnvironment`：仅为当前未完成初始化的 Project 按 Wizard 确认的 `local` 或 `ssh` target 创建缺失 Environment 或更新已保存 target，并在保存后进入 `needs_probe`。复用已有 target validation 与 `workspace_root` 归属规则。Linux SSH 主机初始化走同一 initialization boundary 的 bootstrap command。
- 受管部署凭据按 Project/name 复用；保存失败后的重试不重复创建 `deployment-ssh`，发现非受管同名凭据时返回冲突。增加部署公钥读取 command，使 Windows 命令可在 Environment 保存前生成；已有 Environment 则复用绑定凭据。
- 实现 `ProbeEnvironment`：在事务外执行已有 Probe，保留 target revision 的条件更新语义；失败持久化诊断和 Environment，允许 Wizard 后续修改或重试。
- 实现 `CreateGateway`：仅接受完整、经 Wizard 确认的输入，先断言 latest Probe 成功，再调用 Gateway factory 在一个数据库事务内创建 Application、GatewayConfig、四个 Version、固定 `traefik` Component、default Service、Route 与 Environment binding。Application name/code 固定 `Traefik` / `traefik`；default Service code 为 `traefik-default`。初始 `pull_policy` 固定 `missing`，成功后保持待部署状态且不创建 Deployment。
- 将 Gateway factory 改为显式完整输入，删除 `GatewayCreateDefaults`、`CreateDefaults`、`applyCreateDefaults` 及对全局 `TraefikConfig` 的依赖。

**完成条件**：Probe 不处于数据库长事务中；失败不会留下半个 Gateway；重新进入 Wizard 可从已持久化资源恢复正确步骤。

### 4. 替换 HTTP / Proto 初始化入口并重新生成契约

**目标**：HTTP 向 Web 提供单一 Project 初始化 API，普通 Gateway create 不再是独立初始化入口。

**修改范围**：

- 新增 Project Initialization proto DTO 与 handler/route，提供状态读取、Environment 保存、部署公钥读取、Probe、Gateway 创建，路由统一位于 `/api/project/:project_id/initialization`。
- 调整 `proto/orbit/v1/environment/environment.proto`、`proto/orbit/v1/gateway/gateway.proto`、HTTP handler/mapper/route，移除 `traefik_component_name`，删除常规 Gateway create endpoint 的页面使用；保留初始化完成后必需的 Environment/Gateway 编辑与读取能力。
- 将 Wizard 的三类 command 映射到 `ProjectInitialization` 服务，禁止 handler 直接编排 Environment/Gateway 通用 service。
- 执行 `task proto` 和 `task sqlc`，提交 Go 与 TypeScript generated DTO/SQLC 结果，修正全部编译调用点。

**完成条件**：API 契约没有独立 Gateway 初始化、component-name 双写或运行时全局默认输入；Wizard 之外不能经 HTTP 创建部署资源。

### 5. 修正 Gateway 的已初始化行为

**目标**：Gateway 只管理已经存在的资源，配置编辑只影响 GatewayConfig 的合法运行属性。

**修改范围**：

- 在 `internal/application/gateway/usecase/provision.go` 使 `ProvisionGateway` 仅查找当前 Project 的既有 Gateway/default Service。Environment 未就绪或缺 Gateway/default Service 时返回明确的业务错误，不创建 Gateway、Version、Service 或 Route。
- 收窄 Gateway create 的公开 usecase 使用范围为 `ProjectInitialization` 内部调用；删除独立创建 UI/MCP 注册。
- 对齐 Gateway update 的 application DTO、HTTP 请求和 MCP 输入，完整覆盖 name、REST URL、readiness、base domain、entrypoint、TLS 与 ACME 字段；image 与 pull policy 继续通过普通 Version Component 编辑。
- 更新 Gateway factory、Dashboard Route、deployment/runtime 读取和相关测试，固定初始 Component name 为 `traefik`，避免任何 GatewayConfig 到 Component topology 的回读。

**完成条件**：`orbit_provision_gateway` 和 HTTP/MCP Gateway 操作均不会成为第二个初始化路径，Gateway update 不会制造配置漂移。

### 6. 实现 Web 初始化检查、路由 guard 与 Wizard

**目标**：打开或切换到未初始化 Project 时先完成统一 Wizard，任何运行时页面都不会加载前一个 Project 或全局默认值。

**修改范围**：

- 在 `web/src/api` 和 `web/src/stores` 增加 initialization status/command client 与状态管理，所有请求显式使用当前 `activeProjectId`。
- 在 `web/src/router/index.ts` 为项目级 route 和 Project 切换建立 readiness guard：`ready` 回到目标页，其他状态转至 Wizard。Project 管理、登录与 Wizard route 排除循环 guard。
- 新建 `web/src/views/project` 下的 Initialization Wizard 页面。第一步使用与环境页共用的 `EnvironmentTargetFields` 选择 `local | ssh`；Linux SSH 主机初始化复用 `EnvironmentBootstrapFields`。Windows SSH 提供当前表单参数驱动的 PowerShell 初始化命令，命令可在保存/测试前生成；Wizard 按后端 status 恢复 Environment 保存、Probe、Gateway 确认三个步骤，用初始化来源的服务端默认值填充初次表单，但保存后只展示当前 Project 的持久化结果。
- 修改 `EnvironmentPage.vue`、Gateway 页面、`gatewayConfigForm.ts` 和导航，移除独立 Gateway 创建流程及 component-name 输入；环境页继续用同一套目标表单编辑已保存 Environment，并在 Windows SSH 目标上展示同一初始化命令 Dialog。未就绪资源不显示空页或错误加载状态。现有 SSH bootstrap 只保留与已保存 Environment 相关的明确能力，不替代 Wizard 生命周期。
- 更新 Web i18n、类型、router/store/form 测试，覆盖空库默认 Project、新建 Project、切换 Project 和初始化完成后回跳原目标 route。

**完成条件**：未初始化 Project 只进入 Wizard；切换 Project 不复用旧 Environment/Gateway 数据；Gateway 创建后处于可显式部署的待部署视图。

### 7. 为 Delivery MCP 引入 connection-local Project scope

**目标**：Codex stdio MCP 必须先确认一个已就绪 Project，后续项目级工具自动使用该 connection 的 scope；Web Dialogue 注入请求 Project 的固定 scope。

**修改范围**：

- 在 `internal/api/mcp/delivery/server.go` 与拆分的 tool 文件中实现 `ProjectScope`、`requireProjectScope`、readiness assertion 和资源归属 assertion。scope 只挂在 MCP connection/core 生命周期内，绝不写入 config、数据库、浏览器状态或用户偏好；scope 的读写按现有 server 并发模型保护。
- 保留 `orbit_list_projects` 作为发现工具，确保每项返回 id、name、唯一 code 和 active 信息。新增 `orbit_select_project(project_id)`，校验 member 可见性、Environment active/latest Probe 成功、Gateway/default Service 存在后写入 scope，并返回 Project、Environment、Gateway/default Service 摘要。新增 `orbit_get_current_project`；无选择时返回 `project_not_selected`。
- 从除 list/select 外的项目级 MCP 输入删除 `project_id`。list/create Application、Gateway、Route、Service、Version、Deployment、runtime 和 Environment 相关工具均先取得 scope；接受 Application/Version/Service/Gateway/Route/Deployment ID 的工具在已有读取后校验资源的 Project 等于 scope。
- 移除 `orbit_create_gateway` 注册。将 `orbit_provision_gateway` 改为无 `project_id` 的已存在 Gateway/default Service 解析操作。将 `orbit_update_gateway` 扩展为与 HTTP update 等价的完整 GatewayConfig 字段集。
- 更新 `internal/bootstrap/app.go` 的 stdio composition，并修改 Dialogue port/usecase、MCP client 和 HTTP bootstrap，使 Web Deployment Dialogue 用请求中的 `project_id` 创建已预设且已验证的短生命周期 scope，不执行名称发现或项目切换。
- 增加 MCP server tests：名称发现输出、无选择报错、选择确认、未就绪拒绝、同一 connection 切换 Project、后续 schema 不含 `project_id`、ID 归属不匹配拒绝、provision 不创建资源，以及 Dialogue 固定 scope。

**完成条件**：Codex 只能通过“列出 Project -> name 唯一匹配或用户以 code 澄清 -> select internal id”建立 scope；connection 结束后 scope 自动丢失；Web Dialogue 始终固定在页面请求 Project。

### 8. 更新活文档和清理遗留配置叙述

**目标**：活文档只说明当前配置归属和初始化边界，不继续暗示默认 Gateway、全局 deployment workspace 或 MCP 初始化能力。

**修改范围**：

- 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/mcp-direct-operations.md`、部署/路由相关 guide 与 `docs/decisions/ledger.md`。
- 删除 `workspace.deployment`、`data/deployment`、`traefik.*` 运行时 fallback、自动 provision、MCP `project_id` 后续传参和业务 seed 的活文档表述。
- 明确 Environment 的 `workspace_root` 是 Service Compose、Gateway certificates 与 Route snapshot 的部署目标目录；`workspace.pipeline` 和 `logging.deployment_root` 仍为控制面配置。

**完成条件**：文档、配置样例、HTTP/MCP 操作说明和代码契约一致，不把归档材料作为实现依据。

## 验证计划

1. 数据与配置：运行 config unit tests；运行 SQLite、MySQL、PostgreSQL migration tests，断言 identity seed 存在而部署业务 seed 不存在，并确认新 migration 删除 component-name column。
2. 应用服务：为 `ProjectInitialization` 覆盖四个派生状态、Environment 保存、Probe 失败重试、Probe 成功后原子 Gateway bundle 创建和 ready；为 Project create 断言不再创建 Environment。
3. Gateway：覆盖完整显式输入、固定 `traefik`/`missing` 初始 Component、无 defaults、Gateway update 字段完整性、Dashboard target 和 `ProvisionGateway` 缺失资源不创建行为。
4. HTTP / Proto：验证初始化路由、输入映射、错误状态和删除的 Gateway/component-name 契约；运行 `task proto`、`task sqlc` 后确认 generated files 干净。
5. MCP：覆盖 Project 名称 discovery 的输入输出、手动 code 澄清后的 select、scope 生命周期、未选择/未就绪/归属不符的业务错误，以及 Web Dialogue 的固定 request scope。
6. Web：覆盖 router guard、Wizard steps、Project 切换、完成后跳回，以及 Environment/Gateway 页面只读取 ready Project 数据。
7. Windows SSH：覆盖命令生成、当前表单参数、PowerShell quoting、远端 WSL2/Docker/Compose 检查和保存前公钥读取；覆盖受管部署凭据重试复用。
8. 最终执行项目固定检查：`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`，并按失败位置补充定向 Go/Web/MCP/migration 测试。

## 实施顺序与依赖

1. 先完成步骤 1-3，再执行 migration/SQLC/proto 生成，确保新 application boundary 有单一数据契约。
2. 先完成 HTTP 路由和后端 usecase tests，再实现 Web guard/Wizard，避免前端临时拼接通用 API。
3. MCP scope 在 Gateway 初始化入口移除后接入，随后完成 Dialogue 的固定 scope 注入，避免 stdio 与 Web 对话出现不同的 readiness 标准。
4. 文档在实际接口和测试落定后更新，最后运行全量检查。

## 风险、假设与回滚

| 项目 | 处理方式 |
| --- | --- |
| 重组 `000039`–`000042` | 按无兼容基线直接收敛最终 Environment schema，并将 GatewayConfig 清理保留为独立 migration；现有数据库由运维就地同步迁移版本记录。 |
| 外部 Probe 失败 | Environment 与诊断保留，Wizard 重试，不将 Probe 放进 Gateway bundle 事务。 |
| 初始化配置误作 runtime fallback | 以依赖注入边界和定向测试禁止 Gateway/Deployment/Route 从 `ProjectInitializationConfig` 读取。 |
| MCP scope 漏检 | 每个项目级工具集中调用 scope/readiness/ownership helper，并用 schema 与跨 Project ID 测试锁定。 |
| Windows SSH 尚未具备部署密钥 | 命令生成只依赖受认证 Project 公钥和当前表单，不依赖先保存或先 Probe；命令执行后仍由页面测试与 Probe 验证。 |
| 初始化失败遗留部署凭据 | `deployment-ssh` 按 Project/name 查找并复用受管记录；非受管同名记录继续返回冲突。 |
| 回滚 | 不保留运行时兼容层。需要回退时先执行新 migration 的 down 再回退二进制；该过程恢复列结构，不承诺恢复已删除的旧字段值。 |

## Blockers

暂无。现有 `task proto`、`task sqlc`、Go test 与 Web lint/typecheck 入口已明确；实施时若生成工具或三数据库测试环境不可用，记录具体命令和失败原因，不以手工编辑 generated files 替代生成。

## User review notes

- 用户要求以严格模式记录此跨 Project/Environment/Gateway/MCP 配置收敛任务。
- 用户确认保留 identity Project/membership seed，仅删除 Gateway、Environment 及关联部署资源 seed。
- 用户确认 Web Wizard 是唯一初始化边界，MCP 只断言 Project 已就绪，不返回页面跳转语义。
- 用户确认 Codex 用户通常以 Project name 声明目标；同名时以唯一 code 澄清，随后由 `orbit_select_project(project_id)` 确认并在当前 connection 内保存 scope。
- 用户明确不支持也不为多个 Project 复用同一部署宿主增加兼容或防御逻辑。
