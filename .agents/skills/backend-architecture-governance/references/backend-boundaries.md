# Pomelo Orbit 后端边界

路径相对任务工作区的项目根目录。以下区分当前落点和职责目标；使用时核对当前代码，不将历史问题当作已修复，也不要求复制另一项目的目录和框架。

## 职责与依赖

| 当前位置 | 应承担的职责 | 要检查的关系 |
| --- | --- | --- |
| `cmd/server`、`cmd/migrate` | 配置、启动、信号和退出 | 业务编排不放在进程入口；本项目 worker/MCP 由已有启动路径承载，不要求新增 `cmd/worker`。 |
| `internal/bootstrap` | 组合根、具体实现选择、资源生命周期 | 注入应用端口与 Repository；不承载独立业务规则。HTTP 与 stdio MCP 复用应用能力。 |
| `internal/api/http` | Gin 路由、中间件、鉴权、参数与响应映射 | Gin/HTTP 类型留在此边界；handler 调用用例，不直接执行 SQL 或 Compose。 |
| `internal/api/mcp/delivery` | MCP 工具契约、session actor 与 Project scope | 工具调用同一应用能力，不借 HTTP 入口或浏览器身份定位依赖。 |
| `internal/queue/worker/handler` | 任务载荷解析和执行入口 | 经注入的应用能力执行业务；queue 基础机制不拥有部署或流水线规则。 |
| `internal/application/<business>/{usecase,dto,port}` 及已有 `rule` 包 | 用例编排、业务输入输出和所需能力端口 | 依赖 `model`、`repository` 接口及明确应用契约；不依赖 API、bootstrap、Repository 实现、SQLC Row 或具体 runner。 |
| `internal/model` | 业务数据、值与规则 | 不用 Proto、Gin、数据库 Row、SDK response 表达规则；无需为了通用目录示例新建 `domain/`。 |
| `internal/repository` | 持久化接口、分页及错误契约 | 业务层消费接口，SQL 和事务实现不向此层泄漏。 |
| `internal/repository/impl/sqlc`、`sqlcommon` | 查询、Row 映射、技术错误及复合持久化 | 实现 Repository 接口；生成类型留在实现层，跨业务用例编排留在 application。 |
| `internal/infrastructure` | 数据库、文件、Docker/SSH、外部 API 和应用端口适配 | 可以依赖内层端口，不反向组装业务用例；资源和技术协议由适配器拥有。 |
| `internal/gen/proto`、`internal/gen/sqlc` | 生成协议对象与查询代码 | Proto 主要由 API 映射，SQLC 由持久化实现消费；走项目生成入口，不手改生成物。 |
| `internal/common` | 无业务语义的通用能力 | 共享消费者并不自动意味着应放 common；消费 `VersionComponent` 等业务对象的规则要核对实际所有者。 |

具体类型用于基础设施之间协作或 bootstrap 装配不是反向依赖。设置管理本就消费集中 typed config，项目初始化也接收已校验的有限默认配置；不要仅凭 `config` import 报错。禁止的是用例散落加载 ENV/Viper、创建资源或定位具体实现。

## 事务与生命周期

- `internal/infrastructure/database/tx/{context,run,runner}.go` 提供 context 中的 DB/Tx、复用已有事务和短事务实现；application 通过所属 `port.TransactionRunner` 使用，不直接消费 `*sql.Tx`。
- HTTP 写事务在 bootstrap 中装配。成功响应须在提交后写出；失败、panic 或提交失败须保持回滚和错误契约，流式/长操作的排除路径沿用当前配置。
- HTTP 事务中间件和协议测试位于 `internal/api/http/middleware/transaction{,_test}.go`；通用 DB/Tx 包不传递依赖 Gin。移动或调整时连同装配引用和响应缓冲语义一起核对，不恢复旧包别名。
- `internal/queue/worker/worker.go` 仅依赖任务存储、handler 和生命周期 context，不持有 SQL DB 或注入 DB context。Task Repository 的 claim 使用独立短事务；handler 的 Docker、文件和网络操作不包外层事务，应用的多步写入使用已注入的短事务端口，Complete/Fail 由任务存储负责。
- MCP 绕过 HTTP 事务时，应逐项确认多写步骤的原子性：检查实际工具 wrapper、应用事务端口与 context，不能从路由注释推断工具已拥有事务。Core 的 actor middleware 仅做鉴权，数据库写工具通过 `addWriteTool` 使用 bootstrap 注入的事务端口，提交后才返回成功结果。部署命令使用应用自身的短事务，Probe、观察、日志和等待工具不包外层事务。不要为跨入口复用依赖 Gin middleware，也不要一律将单仓储复合写入上提为新 UoW。
- HTTP 写事务的排除需要核对注册顺序、实际路径和用例调用链。非 DB 配置文件保存应沿用文件存储自己的原子性；调用外部进程的用例先完成外部阶段，再以必要短写边界保存并复核事实，不能只看应用包未导入基础设施就认定事务范围正确。
- PipelineRun 在 Begin 后拥有失败收敛责任。整个执行使用配置的总超时，终态写入使用独立有界清理 context；worker 传入真实投递次数，重投递发现遗留 running 时记录中断失败，不重放外部操作，不将活跃的首次重复执行当作成功。条件更新须保留用户取消和已完成状态。
- bootstrap 持有进程数据库与共享能力；执行和流会话明确自己的取消、关闭责任。客户端断开只取消所属读取/session，不能顺带停止后台部署或容器。
- 服务状态检查使用独立 `StatusCheckService`，只注入 Project 权限、`StatusCheckStore`、目标 resolver 与 `ContainerInventory`。Docker CLI、JSON 和 Compose label 解析属于 `infrastructure/runner/docker`；应用层消费结构化容器事实，负责状态规则和确认快照。测试应走该构造入口，不能借用部署接管夹具补写内部字段。
- 目录占用用例通过 `MountInventory` 获取运行 bind mount 的 Source/Shared 事实，Docker 查询、状态过滤和 label 解码由 Inventory 适配器验证。目录重叠与共享子目录规则直接测事实；不在用例测试里拼 Docker JSON。
- PipelineRun 命令/查询使用 `Service`，worker 使用 `ExecutionService`。检查构造参数是否实际需要：命令服务不接收执行密钥、凭据仓储、容器 runner、版本生成器或本地源码读取器；测试不能利用执行服务顺带触发命令来绕过正常装配。
- Deployment 命令与执行也使用独立类型和构造入口；命令注入 `WorkspaceReader`、`RuntimeInspector`、`ContainerLogs`、`LifecycleCommands` 和部署日志读取/删除端口，执行仅注入所需的 `ExecutionWorkspace`、`ServiceLifecycle`、`ContainerObserver` 和日志 writer。worker 使用冻结的 Gateway 快照，不组装选择协调器。共享 Compose 渲染接收显式输入，不依附空 Service。
- 容器日志的 Docker 标签过滤、JSON ID 解码和 follow 命令由 `infrastructure/runner/docker/container_logs.go` 承担；应用替身提供容器 ID 和记录续读请求，保留权限撤销、目标修订、源变化及取消关闭断言。
- 运行时查询及取消后观察通过 `RuntimeInspector`/`ContainerObserver` 获取事实；Docker ps/inspect 等协议留在 `infrastructure/runner/docker/runtime_inspection.go`。稳定性规则直接读取状态、健康度和重启次数，原始诊断仅透传为 evidence。归属校验通过的完整网络名必须与实际查询对象一致，不能校验带前缀的名字后查询另一个短名对象。Compose 配置解析在适配器输出服务/镜像事实，应用比较差异，不能因实际镜像缺失而跳过比较。Compose 暴露/网络策略由 usecase 先解析，挂载计划返回结构化路径、target、只读与落盘事实，格式输出只编码。Gateway profile/token/挂载规则在计划准备阶段处理，YAML 节点编辑由 deployment/render 承担；渲染不再次修改计划。Compose 是应用输出映射，不为搬目录增加渲染端口。
- 部署/停止用例调用 `ServiceLifecycle`，不拼装 CLI；Docker 适配器将选项映射为命令并透传目标、目录、context、输出与错误。`LifecycleCommands` 的持久化描述和执行共用参数构建，不解析描述执行。应用保留拉取策略、Gateway 强制重建、移除卷及资源关闭责任；测试分别验证业务选项与适配器协议。
- 本项目 `App.Serve`/`RunWorker` 启动会执行迁移与 seed。不要复制其他项目“启动不迁移”的约束；已执行迁移的修改仍受 AGENTS.md 限制。
- 环境终端通过 `TerminalService.MonitorSession` 复用成员/目标/凭据验证并执行周期复核与空闲规则；HTTP 消费终止原因，负责 WebSocket 消息和 session 关闭。规则测试使用虚拟时间和存储事实，协议测试直接注入终止原因验证无输入时的通知与释放，不能为了验证两秒轮询而启动整条 WebSocket 链路。

## 代表测试与分类

按测试实际连接的边界分类，不根据文件名、`t.Parallel()`、行数或耗时直接删除。

| 案例 | 应证明的行为 | 测试边界 |
| --- | --- | --- |
| [service_code_test.go](../../../../internal/application/service/usecase/service_code_test.go) | 用例使用提交的 Service code 和必要业务校验 | 真实用例 + Repository/应用替身；不需要 SQLite 或迁移。 |
| [Route publication_test.go](../../../../internal/application/route/usecase/publication_test.go) | 发布范围、修订、失败后继续、安全错误映射及 ACME 准入 | 真实用例 + 存储事实/发布端口替身；锁等待用虚拟时间。条件写入、实际保存与失败后的持久化状态继续在集成测试验证。 |
| [model/component_identity_test.go](../../../../internal/model/component_identity_test.go) | 组件身份规则 | 直接验证业务值与规则。 |
| [environment/environment_test.go](../../../../internal/api/http/handler/environment/environment_test.go) | 当前 DTO 与 Proto 的目标字段映射 | 直接调用 mapper，避免数据库和完整服务装配。 |
| [MCP route_tools_test.go](../../../../internal/api/mcp/delivery/route_tools_test.go) | 工具参数、actor/Project scope 与结果映射 | 内存 MCP 连接 + 应用替身，属于协议契约验证。 |
| [MCP transaction_integration_test.go](../../../../internal/api/mcp/delivery/transaction_integration_test.go) | MCP 组件集合更新失败不丢原值，成功提交后可见 | 内存 MCP + 真实用例/SQLC/临时 SQLite；一个必要的跨入口原子性验证，不能由应用替身证明回滚。 |
| [user/service_test.go](../../../../internal/application/user/usecase/service_test.go) | 密码长度、必填字段和真实密码处理 | 真实用例 + 存储事实替身，不需要迁移。 |
| [deployment/service_status_check_test.go](../../../../internal/application/deployment/usecase/service_status_check_test.go) | 预览、操作者隔离和确认时传播条件写入冲突 | 正常构造独立用例 + 结构化事实端口；不依赖 Docker 字符串或 SQLite。 |
| [deployment/container_log_stream_test.go](../../../../internal/application/deployment/usecase/container_log_stream_test.go) | 等待容器、重建后续读、权限撤销及目标变化后关闭 | 命令服务正常构造 + 容器 ID/日志请求替身；等待编排使用虚拟时间，不断言 CLI。Docker 参数及输出解码在适配器测试验证。 |
| [deployment/runtime_tools_test.go](../../../../internal/application/deployment/usecase/runtime_tools_test.go) | 容器/网络归属、组件身份、探测准入及重启计数增长 | 正常构造 + 结构化观察替身；稳定性窗口用虚拟时间。JSON 数组/记录流、命令参数和诊断数据契约在 Docker 适配器验证。 |
| [service/status_check_test.go](../../../../internal/repository/impl/sqlc/service/status_check_test.go) | 状态确认不能覆盖已变化基线或活动部署 | 真实 SQLite 条件 SQL，一次迁移、每个案例独立事务回滚；不重复测试外部观察。 |
| [deployment/execution_control_test.go](../../../../internal/application/deployment/usecase/execution_control_test.go) | 取消不等待无响应端口、新操作不等待旧运行退出、超时后忽略晚到结果 | 真实执行编排 + 事实/阻塞端口替身，`testing/synctest` 提供虚拟时间，不消耗真实 DB 清理预算。 |
| [service/repository_test.go](../../../../internal/repository/impl/sqlc/service/repository_test.go) | 删除保留 Deployment 历史、查询与隔离 | 临时 SQLite + 真实查询，属于本地持久化集成验证。 |
| [deployment/operation_takeover_test.go](../../../../internal/application/deployment/usecase/operation_takeover_test.go) | 新操作接管、旧结果不覆盖当前记录及失败准入回滚 | 正常构造命令服务 + 真实 DB/queue，直接验证 Repository 生命周期条件写入；取消、超时和晚退出的编排独立测试。 |
| [worker/worker_test.go](../../../../internal/queue/worker/worker_test.go) | 领取事实传递、成功/失败分派及结果保存错误传播 | 真实 Worker/Router + 任务存储记录替身，不依赖 SQL。领取与重试预算的 SQL 证据留在 Task Repository。 |

例如密码长度、名称校验等规则若通过完整迁移才能测试，优先直接调用真实用例并替换存储端口；唯一约束、事务回滚和历史保留则需另保留持久化证据。不要让假 Repository 自己实现权限、状态转换或回滚逻辑，再断言它做对了。

## 默认检查与外部联调

当前 `Taskfile.yml` 的 Go 测试范围为 `./cmd/... ./internal/... ./sql`。其中包含本地 SQLite、文件、内存协议及真实本地终端进程验证，所以默认测试通过不等于“全部都是单测”。测试环境目录和数据库应由 `t.TempDir()`、内存库或现有 helper 管理，不操作用户工作数据。

日常核心入口为 `task test:go:unit -- -count=1`，范围为 application/model/common/queue，`!unit` 文件保留在完整入口中。`task test:go:all -- -count=1` 与普通 `go test` 执行完整验证。该分类用于快速反馈，不替代完整检查；核心入口的测试依赖不得携带迁移、具体 SQLC Repository 或 Gin。

外部验证已有显式启用条件：

- `internal/test/e2e`：MySQL/PostgreSQL 配置或 DSN、Windows OpenSSH。
- `internal/application/pipeline_run/usecase/windows_remote_e2e_test.go` 与 `internal/infrastructure/runner/ssh/pipeline_windows_e2e_test.go`：指定 Project 和可用镜像后执行 Windows/Linux 远端 SSH 流程。
- `internal/infrastructure/external/traefik/route_restart_e2e_test.go`：Docker 和真实 Traefik 重启。
- `internal/infrastructure/runner/local/terminal_test.go`、`internal/common/envfile/store_test.go` 中的特殊场景：隔离容器 Docker socket 或可写文件 bind mount。

默认检查沿用这些条件并报告 skip。显式外部联调需按任务授权和隔离目标执行；不使用用户现有数据库、Project 或宿主机来补足默认检查。SQLite 结果不能替代 MySQL/PostgreSQL 方言或真实 SSH/Docker 的证据。
