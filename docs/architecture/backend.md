# 后端架构（现行）
最后修改时间: 2026-10-14

Doc role: living SoT  
权威：与代码冲突时以代码为准。  
代码锚点：`cmd/server`、`internal/bootstrap`、`internal/api/http`、`internal/application`、`internal/repository`、`internal/infrastructure`。

## 技术栈

| 层 | 选型 |
|----|------|
| 语言 | Go（见 `go.mod`） |
| HTTP | Gin |
| API 契约 | Protobuf（`proto/orbit`）+ 生成代码 `internal/gen/proto`；HTTP 映射在 handler |
| 配置 | Viper + YAML + ENV；固定 overwrite.env 应用覆盖；启动校验（`internal/config`） |
| DB | SQLite / MySQL / PostgreSQL；**golang-migrate** embed（`sql/migration`） |
| SQL | **sqlc** + `database/sql`；实现位于 `internal/repository/impl/sqlc`，通用 SQL 错误处理位于 `sqlcommon` |
| 队列 | DB 后台任务 + 同进程 worker（单节点；API 与 worker 可同启） |
| 模板 | Liquid（CI/部分渲染路径） |
| 前端静态 | `web/dist` 等由部署配置决定 |

**已废止文档中的栈**：Python / FastAPI / SQLAlchemy / Alembic — 见归档 `docs/archive/designs/architecture.md`。

## 分层与依赖方向

```text
cmd/server, cmd/migrate
  → bootstrap（组装 HTTP、MCP、worker、DB 与迁移）

api/http、api/mcp、queue/worker/handler  # 入站适配
  → application/*（usecase、dto、port）
  → model + repository（接口）+ application/*/port

repository/impl/sqlc + infrastructure  # 出站适配，由 bootstrap 注入
  → 内层接口与 model
  → 所属技术依赖（SQLC、database/sql、文件、SDK 等）

gen/proto 由 API 映射；gen/sqlc 由持久化实现消费，不进入业务规则。
```

约定：

- **API 路由单数**，例如 `/api/ci/repository`（`CLAUDE.md`）。  
- 不在业务层散落读环境变量；配置集中 `internal/config`。  
- 生成代码隔离在 `internal/gen`。

## 测试与架构治理

具体实践见项目内 [backend-architecture-governance skill](../../skills/backend-architecture-governance/SKILL.md)。单元测试执行真实规则和用例，只替换明确端口；临时 SQLite 的查询、约束、回滚和历史保留测试属于本地持久化集成验证，Gin/MCP 的内存协议验证及真实本地进程验证也按实际边界分类。默认 Go 测试包含这些验证，不等于全部是纯单测。

架构治理应检查实际及传递依赖、装配和资源生命周期，不能仅因测试通过或包目录正确就认定职责归位。前端的测试限制不适用于后端真实的取消、租约、事务和资源释放承诺；外部数据库、Docker/Traefik 与远端 SSH 联调按既有显式条件启用。

日常使用 `task test:go:unit -- -count=1` 验证 application/model/common/queue 核心规则和用例；已分类的集成/外部测试使用 `!unit` 约束，普通 `go test` 和 `task test:go:all -- -count=1` 仍包含全部本地验证。单测耗时应以独立运行的实际用例计时评估，区分编译、进程启动和外部资源成本。

Service 状态检查由独立 `StatusCheckService` 承担，bootstrap 注入权限、状态存储、目标解析和容器清单端口。Docker 命令、JSON 与 Compose labels 在 `infrastructure/runner/docker` 适配器解析；业务层只接收结构化事实。状态确认的条件 SQL 在 Repository 验证，业务测试不依赖数据库迁移或完整部署服务。

部署目录占用检查通过 `MountInventory` 获取其他 Service 正在使用的 bind mount 及共享声明。Docker ps/inspect、运行状态过滤和 label 解码由同一 Docker 适配器承担；目录重叠、共享子目录及 Service 根目录保护仍由用例判断。

容器日志用例依赖 `ContainerLogs`，由 Docker 适配器承担容器 ID 发现和日志跟随命令。应用保留成员/目标修订复核、日志源变化、游标与重叠续读规则，以及取消后的流关闭。日志发现和跟随在环境根目录执行，不要求 Service workspace 或 Compose 文件。

通用运行时工具使用 `RuntimeInspector`，取消后状态观察只依赖其中的 `ContainerObserver`。Docker 适配器承担 ps/config/logs/inspect/probe/version 命令与 JSON 解码；应用消费容器状态、健康度、重启次数和网络名事实，负责归属、组件身份及稳定性规则。原始诊断数据仅用于对外 evidence，不在业务层按 Docker JSON 字段解析。网络短名必须解析到已确认归属的完整网络名后再查询。Compose 配置解析由适配器返回服务名/镜像事实，应用比较预期服务、容器和镜像差异，缺失镜像也应报告漂移。Compose 暴露/网络策略先解析为结构化事实，挂载计划与卷字符串编码分离，env 占位符仅在有效计划阶段解析。Gateway enrichment 在 hash/预览/执行渲染之前完成，成功时发布独立切片，失败不修改共享声明；deployment/render 只编辑 YAML，不选择 profile/token/挂载策略。渲染作为应用输出映射保留，不增加仅为隔离格式而设的端口。

PipelineRun 的 `Service` 负责命令、查询和日志读取，`ExecutionService` 负责 worker 执行及终态收尾，两者使用独立构造入口。命令服务不接收凭据仓储、解密密钥、容器执行器、版本生成器或本地源码读取器；日志和清理只获取文件资源。

Deployment 同样使用独立 `Service` 与 `ExecutionService`。HTTP/MCP 命令服务通过 `WorkspaceReader` 读取目录，通过事实端口查询运行时和容器日志；部署日志只提供读取/删除能力，不持有执行 store、readiness、挂载观察或执行超时。worker 注入 `ExecutionStore`、`ExecutionWorkspace`、`ServiceLifecycle`、观察端口和日志 writer，依据冻结快照执行，不接收 Gateway 选择协调器。`ExecutionWorkspace` 仅保留目录解析、挂载路径及 workspace 准备能力。`ServiceLifecycle` 接收目标、目录和部署/停止选项，由 Docker 适配器生成并执行命令；命令服务通过 `LifecycleCommands` 保存同源命令描述，描述不参与执行。应用拥有 session 和日志 writer 的生命周期，适配器借用 context 与 writer。Compose 渲染为显式输入函数，命令预览和执行共用。测试通过各自构造入口组装，不能借同一 Service 混用命令与执行依赖。

环境终端的 `TerminalService.MonitorSession` 负责周期性成员/目标复核和空闲判断，复用输入前的 `ValidateSession` 规则。HTTP handler 通过 `TerminalApplication` 调用应用能力，记录输入/输出活动，消费终止原因并发送控制消息、关闭会话；不自行轮询或判断空闲。连接、最长会话及消息写入的传输期限仍由 handler 管理。周期规则以端口替身和虚拟时间验证，WebSocket 测试验证协议与资源关闭，不等待真实业务轮询周期。

### 事务与执行失败边界

- HTTP 写 UoW 位于 `internal/api/http/middleware/transaction.go`，bootstrap 装配，提交后才发送成功响应；通用 `infrastructure/database/tx` 仅负责 DB/Tx context 和短事务，不依赖 Gin。
- MCP 的纯数据库写工具通过 `addWriteTool` 使用同一组合根注入的事务能力。stdio 和部署对话内存 MCP 共用此装配；部署命令自行管理应用短事务，Probe、日志、外部观察与等待不进入工具外层事务。
- `/api/setting/config` 使用覆盖文件自身的 revision、锁和恢复机制，不建立请求 SQL 事务。mkcert 路径在事务外生成证书，保存时再次校验成员权限，并用 Project、domain、HTTP protocol 和原更新时间条件更新证书字段；路由变化返回冲突。
- PipelineRun 的配置超时覆盖目标解析、工作区准备和 stage 执行。Begin 之后的读取/执行失败必须收敛 run；run 和 stage 终态写入使用脱离原取消、最多 5 秒的清理 context，存储失败继续传播。
- Gateway readiness 使用总超时 context 覆盖目标解析、单次 API 请求与重试等待，单次请求预算不能延长总预算；超时后不再发起探测，返回错误保留 deadline/cancel 原因及最后一次请求诊断。纯端口重试测试使用虚拟时间，不改变生产轮询间隔。
- Worker 通过任务存储端口领取与写回结果，不持有 `*sql.DB` 或注入 DB context。Repository 使用组合根注入的数据库，领取使用独立短事务，应用内部原子阶段使用事务端口，handler 不包外层长事务。Worker 将实际投递次数传入流水线用例。重投递遇到遗留 `running` 时将其记为中断失败，释放活动运行占用，用户可显式重试创建新 run；不重放可能已产生副作用的 stage。首次重复调用拒绝活跃 run，已取消/已完成记录保持终态。

## 模块边界（application）

| 包 | 职责 |
|----|------|
| `application/application` | 应用与版本 |
| `application/auth` | 认证 |
| `application/credential` | 凭证 |
| `application/deployment` | 部署与 compose 执行 |
| `application/dialogue` | 部署对话 |
| `application/environment` | 部署环境目标 |
| `application/gateway` | 网关 |
| `application/pipeline` | 流水线定义 |
| `application/pipeline_run` | 流水线运行与产物 |
| `application/project` | 项目 |
| `application/repository` | 仓库 |
| `application/role` | 角色 |
| `application/route` | 平台路由 |
| `application/service` | Service 运行绑定、服务编码与运行时覆盖 |
| `application/settings` | 系统设置 |
| `application/user` | 用户 |

## 数据与迁移

- 数据库配置只使用 `database.url` / `POMELO_ORBIT_DATABASE__URL`。连接和迁移从 URL scheme 判断驱动：`sqlite`、`mysql`、`postgres` / `postgresql`；启动时集中校验 URL，系统设置将其标记为 secret。默认 `sqlite:///data/db/pomelo-orbit.db` 相对进程工作目录；Unix 绝对路径使用 `sqlite:////data/pomelo-orbit.db`，Windows 使用 `sqlite:///D:/data/pomelo-orbit.db`，内存库使用 `sqlite:///:memory:`。MySQL 使用 `mysql://user:password@host:3306/database?parseTime=true`，PostgreSQL 使用 `postgres://user:password@host:5432/database?sslmode=disable`；用户名和密码中的 URL 特殊字符须百分号编码。
- Schema 与 seed 均为普通编号迁移，统一位于 `sql/migration/{sqlite,mysql,postgres}/`，由 `MigrateUp` 按版本顺序执行和跟踪；不设独立 data migration 加载流程。
- 默认**不修改已执行的迁移文件**（项目约束）。Repository/Repository Credential 全局共享任务按用户明确要求就地修订了既有迁移，这仅定义新库从头运行的终态；已执行这些版本的数据库不会自动变化，须另行审计并转换，或确认后重建。
- 终态 CD 表见 `*_cd_schema*` 类迁移（application/version/component/expose/environment/gateway_config/service/deployment/route 等）；**无** 旧表 `application_config_file`、`application_service`、`application_route`、`environment_binding` 作为现行 schema。

## 运行形态

配置优先级为默认 YAML < profile YAML < 启动 dotenv < 启动进程 ENV < `<工作目录>/overwrite.env`。配置字段由 typed Config 的 mapstructure 标签自动发现，覆盖、设置页和 ENV 映射使用同一字段定义；不修改 OS ENV。HTTP/worker/MCP/migrate 均调用 config.Load，保存只写覆盖文件，各进程重启后分别生效。设置接口为 `/api/setting/config`，显示当前快照、启动基线、下次值及服务端待重启状态，批量更新和重置经过完整配置校验。

覆盖文件保持 inode 原地写入，跨进程锁和 prepared/committed ENV 恢复记录位于 `<工作目录>/data/config`，启动与接口读取都先恢复中断事务。容器文件挂载及 data 目录必须持久化，见 Docker 与卷挂载指南。

- `App.Serve`：普通迁移（含 seed）→ HTTP → **同进程** background worker（见 `internal/bootstrap/app.go`）。
- `App.RunWorker`：普通迁移（含 seed）→ 可单独跑 worker。
- `App.RunMCP`：本地 stdio MCP；工具发现无认证副作用，每次 `tools/call` 校验统一配置加载后的 MCP PAT，并固定 session actor。Grok 通过仓库 `.grok/config.toml` 启动该进程，Codex 通过 `.codex/config.toml`。它直连 application usecase，不暴露 HTTP `/mcp`，也不使用浏览器 grant、loopback callback、本地 token 文件或 Web JWT。
- Deployment Dialogue：HTTP 入站认证后的 actor 通过每 turn 的内存 MCP session 调用与 stdio 相同的 Core；不转发浏览器 Authorization。
- 单节点挂载 Docker socket 执行 CI/CD 容器操作；不做 API→远程 worker 协议主路径。

## 相关

- [CD 运行时](./cd-runtime.md)  
- [产品 CD 模型](../product/cd-model.md)  
- [决策账本](../decisions/ledger.md)  
