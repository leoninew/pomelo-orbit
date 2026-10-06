# 组件运行身份、DooD 工作区及 Orbit 停机迁移
最后修改时间: 2026-10-06 17:16:29

Review status: Accepted

Mode: standard

## Background

用户检查 Tencent SSH 环境的部署 `01M47YFFXYT0E0S9EXMG0WTPT1` 时，发现错误 `deployment directory overlaps service mihomo-default`。该次部署确认目录为 `/opt/pomelo-orbit`，同一目标修订下 `mihomo-default` 的确认目录为 `/opt/pomelo-orbit/data/deployment/mihomo-default`。目录检查因父子关系拒绝部署，尚未进入实际 Docker 挂载占用检查。

用户希望评估将 Orbit 自身迁到普通服务目录 `/opt/pomelo-orbit/data/deployment/pomelo-orbit-default` 的业务能力与部署步骤，并明确接受停机。用户要求区分现有业务无法表达的能力与可通过部署准备、配置或人工切换完成的事项。

2026-10-06 的 SSH 与库存只读核查结果：

| 项目 | 核查时的现状 |
| --- | --- |
| 远端入口 | `ssh tencent`，Linux SSH 环境 |
| 当前编排 | `/opt/pomelo-orbit/docker-compose.yml`，手工 Compose project 为 `pomelo-orbit` |
| 当前容器 | `pomelo-orbit`；镜像自身未指定运行用户，Compose 指定 `user: 1000:1000`、`group_add: ["988"]` |
| Docker socket | `/var/run/docker.sock` 属于 `0:988`，权限为 `srw-rw----` |
| 持久化挂载 | `/opt/pomelo-orbit/data -> /app/data`、`/opt/pomelo-orbit/logs -> /app/logs`、`/opt/pomelo-orbit/.env -> /app/.env`，另有 Docker socket 挂载 |
| 数据库 | PostgreSQL，由现有配置连接；共享 data 同时包含其他托管服务工作区 |
| 域名入口 | 手工 Docker labels 提供 `orbit.preflite.cn` 与 `orbit-api.preflite.cn` 的 HTTPS 入口 |
| 库存服务 | `pomelo-orbit-default`，确认目录为 `/opt/pomelo-orbit`，运行目录为空 |
| 版本规格 | 核查的 Orbit 版本没有挂载声明；最新构建版本也没有环境变量与端点声明 |
| Gateway | 库存运行目录为空；发布自定义 Route 前需重新部署到已确认目录并完成运行绑定 |
| 回退基线 | 当前 Compose 文件指定的镜像与实际运行容器不同，应以已运行成功的镜像准备回退配置 |

当前 Version Component 和 Service Component 覆盖模型没有 `user`、`group_add`，Compose renderer 也不输出这些字段。这是保持现有 `1000:1000 + 附加组 988` 运行方式时确认的业务能力缺口。

绝对宿主机挂载、环境变量、HTTP 端点、自定义 Route、目录确认与运行绑定已有业务支持。Orbit 启动时读取工作目录下的 `.env`，本场景可继续挂载到 `/app/.env`；不要求为了此次迁移新增通用 Compose `env_file` 能力。

现状依据：

- [CD 产品模型](../product/cd-model.md)、[CD 运行时](../architecture/cd-runtime.md)、[工作目录与挂载](../guides/volume-mounting.md)、[Docker 部署指南](../guides/docker-deployment.md)。
- `internal/model/application.go`、`internal/model/service.go`：Version 声明与 Service 覆盖模型。
- `internal/application/deployment/usecase/effective_service_plan.go`、`runtime_fields.go`、`compose_renderer.go`：有效计划与 Compose 渲染。
- `internal/application/deployment/usecase/compose_command.go`、`deployment_execution.go`、`directory_ownership.go`：Compose 身份、目录绑定与归属保护。
- `internal/infrastructure/runner/ssh/runtime.go`：SSH 文件准备和已有 Compose 的替换。
- `internal/config/config.go`：启动时读取 `.env`。
- `internal/infrastructure/external/traefik/publication.go`：Route 发布需要有效的 Gateway 运行目录。

## Goal

1. 在组件业务模型中支持 Docker Compose 的 `user` 与 `group_add`，可表达运行用户、主组及多个附加组，覆盖本场景的 `1000:1000` 和 `["988"]`。
2. Version Component 保存可复用的声明，Service Component 可按目标运行环境覆盖；继承、覆盖及恢复继承沿用现有运行字段的设计边界。
3. 字段贯通持久化、业务 DTO、现有 HTTP/MCP 数据契约、定义导入导出、Web 配置、有效计划和 Compose 渲染。涉及的具体接口与文件在 Plan 阶段确定。
4. 未配置这些字段时，不向 Compose 注入运行用户或附加组，不改变镜像自身的默认运行身份。
5. 新字段参与现有配置 hash 与部署一致性校验，使预览、实际部署及后续重启使用一致的有效配置。
6. 提供 Tencent 上 Orbit 迁入普通 Service 目录的部署说明，允许停机，由外部控制面或宿主机完成切换，继续使用现有 PostgreSQL、共享 data、logs 与密钥配置。
7. 部署说明明确版本规格补齐、Gateway 初始化、旧实例停止、新目录部署、域名路由发布、验收与回退步骤。以上事项使用现有能力与明确的部署准备，不扩展为自动接管或自动搬迁产品功能。
8. 评估与验证显式覆盖 DooD：区分控制面容器内 logical path、Docker daemon 的 physical path 和 SSH 远端路径，核对启动、CI/CD、日志与 Route 文件的工作区映射。
9. 支持受管 Orbit 在 DooD 中继续部署、重启共享工作区内的其他服务：Orbit 作为普通应用，通过通用的显式共享目录挂载表达工作区；依据实际运行容器的共享声明与挂载判断占用，与当前执行 worker 身份无关。继续保护 Service 根目录和未声明共享的实际运行数据。

## Non-goal

- 不执行远端迁移；实际停机与迁移操作按用户后续指令执行。
- 不实现不停机升级、自动接管手工 Compose、旧 project 自动识别或自动清理。
- 不把当前 API/worker 容器替换自身的更新流程作为交付目标；本次迁移从外部控制面或宿主机执行。
- 不移动整棵 `/opt/pomelo-orbit` 到它的子目录，不将包含其他服务的共享 data 作为 Orbit 私有数据搬迁。
- 不自动迁移 PostgreSQL、轮换 JWT/凭据加密密钥，或搬迁其他服务、Gateway 证书及 ACME 数据。
- 不允许 Orbit 保持父目录部署，不取消通用目录归属或实际运行挂载占用检查；本次选择普通兄弟服务目录，通过通用共享挂载声明表达工作区，不识别 Orbit 或当前控制面角色。
- 不新增通用 Compose 原文、任意 labels、`env_file`、运行身份推断或根据宿主机自动填充 Docker GID 的机制。
- 不安装或修改目标宿主机用户、用户组、Docker socket 权限，不把容器用户与 SSH 登录用户混为一个概念。
- 不修改已执行迁移，不增加兼容层、别名或新旧逻辑并行。
- 此前服务详情展示 `runtime_directory` 的独立改动不纳入本任务的产品实现与验收范围。

## User scenarios

1. 用户在 Version 的组件运行配置中设置 `user=1000:1000` 与 `group_add=["988"]`，保存后能再次读取，并在 Compose 预览中看到同样的运行身份。
2. 同一 Version 用于 Docker socket GID 不同的目标时，用户通过 Service 覆盖附加组；恢复继承后重新使用 Version 声明。
3. 用户未配置运行用户和附加组时，容器沿用镜像定义；此次能力新增不改变现有服务的运行身份。
4. 用户为 Orbit 准备完整版本规格，停止 Tencent 上手工运行的旧实例，再从外部 Orbit 控制面以 `pomelo-orbit-default` 部署到普通服务目录。
5. 新实例保留原来的绝对 data、logs、配置与 socket 挂载，经现有 Route 发布两个 HTTPS 域名，人工检查登录、数据库连接与 Docker 操作。
6. 迁移失败时，用户按说明停止新 project、撤销新路由发布，再启动使用已知成功镜像的旧 Compose；回退不会删除共享数据或重置密钥。
7. Orbit 受管后继续保留 DooD 共享工作区挂载，由其自身 worker 在控制面在线时部署和重启其他 Service；其他业务真实运行挂载冲突仍被保护。

## Acceptance

- [ ] Version 的组件运行配置可保存、读取、编辑和清除运行用户及附加组；组件详情将两字段独立为运行身份区域，通过单独接口保存，基本信息保存不改变身份。配置展示与项目现有表单、字段错误和字体规范一致。
- [ ] Service 可覆盖运行用户与附加组并恢复 Version 继承；有效配置和前端展示能区分声明与覆盖，具体表示方式由 Plan 明确。
- [ ] 以 `1000:1000`、`["988"]` 为主要正向案例，预览与部署 Compose 一致输出 `user` 和 `group_add`。
- [ ] 未设置运行用户或附加组时，Compose 省略相应字段，不设置额外的 root、UID/GID 或 Docker GID 默认值。
- [ ] 定义导入导出、版本 fork、已有 CI 版本派生和重启路径保留新字段，配置 hash 与验证使用有效值。
- [ ] 支持配置已能被当前 Compose 正确表达的用户/组标识；采用项目已有方式在写入时校验格式，约束细节在 Plan 确定，不新增宿主机账号探测前置条件。
- [ ] 针对新字段的持久化、声明/覆盖合并与渲染补充最小有效测试，并运行仓库规定的前后端检查；不以静态或替身测试声称已完成真实 Tencent 验收。
- [ ] 部署说明列出四项绝对宿主机挂载、现有 `.env` 加载、HTTP 端点、发布版本与外部控制面前提，说明这些是配置准备而非新业务能力。
- [ ] 部署说明明确 Gateway 当前缺少运行绑定的处理依据，备份 Compose 后确认原目录并重部署，保留其证书与 ACME 数据，并说明 Route 发布的前置条件。
- [ ] 部署说明使用空目标目录由业务生成 Compose，显式停止旧 project，记录新旧容器/project 身份差异，不依赖旧 project 自动识别。
- [ ] 部署说明覆盖两个域名 HTTPS、登录、PostgreSQL 连接、Docker 操作及服务生命周期的验收，以及可执行的停机回退顺序。
- [ ] 对 DooD 核对实际容器身份识别、启动工作区检查、相对挂载的逐项物理映射和 CI/CD/Route 路径；说明共享工作区的运行挂载占用保护对其他服务部署的影响，不能以迁移成功代替这些检查。
- [ ] 受管 Orbit 的 worker 可在保留显式共享工作区挂载时部署、重启其他 Service，外部 worker 遵循同一规则；未声明共享的实际 bind 及任何 Service 根目录冲突继续拒绝。验证涵盖 local DooD 与 SSH；不新增 DooD 容器识别或路径映射实现。
- [ ] 本次过程文档与部署说明不包含实际密钥、DSN、JWT、DNS token 或其他凭据值。

## Deployment scope

这是意图阶段确认的迁移范围与顺序依据，不表示已经执行；完整命令、前置检查和回退细节在 Plan 中明确。

1. 准备当前配置、PostgreSQL 与已知成功镜像的回退材料。
2. 创建可编辑的 Orbit 版本，补齐运行身份、宿主机挂载、环境与 HTTP 端点，并发布该版本。
3. 核对已有 Gateway 的实际目录和 project，准备两个域名的 Route 和证书/ACME 回退材料；其实际重部署需避开 Orbit 原确认目录及共享工作区的运行占用，具体顺序见 Plan。
4. 从外部控制面或宿主机进入停机窗口，停止旧 `pomelo-orbit` project，停用旧启动入口。
5. 以 `pomelo-orbit-default` 部署到 `/opt/pomelo-orbit/data/deployment/pomelo-orbit-default`，目标为空，由业务生成受管文件；共享 data、logs 与现有配置继续使用绝对 host-path 挂载。
6. 首次部署终态后由外部或内部 worker 完成 Gateway 运行绑定，再显式同步 Route；退出临时 worker 后核验新 Orbit 的在线部署能力。外部执行端保留回退用途，失败时停止新实例、撤销新发布并恢复旧实例。

## Open questions

暂无需要用户确认的未决事项。Plan 核查发现的 DooD 共享工作区占用问题已由用户明确选择“纳入本次计划，明确支持 DooD”，本任务包含对应业务判定与验收；具体实现设计见 Plan。

实际停机迁移尚未被要求执行，当前交付范围是运行身份、共享挂载能力及迁移步骤；不将接受停机理解为已经授权本轮停止远端服务。

## Decisions

- 用户要求先排查部署失败，并解释目录父子关系保护的意义。
- 用户提出将 Orbit 自身迁入普通服务目录，明确可以接受停机。
- 用户收敛评估口径为业务无法支撑的能力与具体部署步骤；配置补齐和人工切换不作为必须新增的产品能力。
- 保持现有非 root 运行身份所需的 `user`、`group_add` 是已确认的能力缺口；当前镜像默认 root 可运行，不作为此次能力实现的替代交付。
- 挂载、`.env`、端点、Route、目录绑定和人工验收已有支撑。Plan 阶段补充 DooD 核查后发现，共享工作区的运行挂载占用判定会影响 Orbit 受管后的持续部署能力；用户明确将其纳入本次产品改造。
- 用户以 `specflow:specflow` 指定标准模式记录本任务；新建独立 Intent 草稿，未将此前的目录任务文档改写为本次任务。

## Risk

- Docker socket GID 是目标宿主机事实，988 只作为此次 Tencent 场景的核查值；其他目标须自行配置对应组，不能硬编码为产品默认。
- 组名解析依赖镜像内账号数据库，数字 UID/GID 的格式正确也不能保证文件权限；真实验收需检查进程身份、附加组与 Docker 操作。
- 用户接受停机，但新旧实例不能在同一数据库和域名入口上未经核对同时运行；部署步骤需明确切换与回退顺序。
- 当前配置文件与运行镜像不一致，直接备份 Compose 不等于准备了可启动的回退版本。
- Gateway 库存与实际目录状态尚未通过此次迁移初始化；现有 Route 发布能力不能绕过缺失的运行目录绑定。
- 新服务目录改变相对路径的解析位置，误将共享 data 改为新目录下的空 data 会改变数据库/工作区来源；迁移说明需明确保留绝对挂载。
- DooD 不能仅靠绝对 host-path 声明验收：启动配置仍使用控制面容器内路径，local 相对源须映射到 daemon 路径，SSH 源须使用远端路径。新容器需要能读取 Docker socket 并 inspect 自身，以通过工作区映射检查。
- Orbit 受管后，其 `/opt/pomelo-orbit/data -> /app/data` 运行挂载覆盖同一目标上的其他服务目录；现有目录保护会拒绝这些服务的部署或需要准备文件的重启。此前“没有其他业务缺口”的结论不覆盖这一持续运维场景，不能沿用为 DooD 已获完整支撑的结论。
- 意图形成时仅完成只读核查；Verification 须区分代码检查与实际部署证据，产品实现不表示已修改库存或进行停机迁移。

## User review notes

- 用户：“查看 ssh tencent /opt/pomelo-orbit，如果想将它转移到普通目录比如 /opt/pomelo-orbit/data/deployment/pomelo-orbit-default，业务是否有缺口”。
- 用户：“你评估的内容里，有些可以通过部署步骤添加；还有我们可以接受停机。应该评估的是业务上无法支撑的缺口，以及部署步骤”。
- 用户：“看起来业务缺口有 user/group，还有吗？”。
- 用户：“$specflow:specflow 标准 记录这次任务”。
- 用户：“开始计划”；视为接受 Intent，进入标准模式 / standard 的 Plan 阶段。
- 用户：“我们有 dood 场景，不要评估遗漏”；Plan 核查加入 DooD，并将发现的共享工作区占用问题回写到本 Intent。
- 用户选择：“纳入本次计划，明确支持 DooD（推荐）”；确认新增业务范围，Intent 继续保持 `Accepted`，Plan 继续形成 `Draft`。
- 用户明确 Orbit 作为普通应用，通过目录挂载支持 DooD；共享占用规则采用通用声明，不采用当前控制面容器身份豁免。用户“开始实现”，Plan 标记 Accepted，进入 Implementation。
- 实现期间仓库并行提交 2dbffa194 移除了部署 owner 标记机制，Gateway 运行绑定准备改为备份后确认原目录并正常重部署。
- 用户要求“开始验证”，进入标准模式 / standard 的 Verification；验证期间要求组件详情的运行用户和附加组从基本信息拆出，保存调用独立接口，纳入当前实现和验证。
- 用户要求运行身份编辑简化为一个表单、两个文本框；使用紧凑弹窗、纵向单行输入，附加组以逗号分隔，保留独立保存接口。
- 用户再次要求“开始验证，验证中的问题要修复”；本轮修复既有凭据更新时间测试的时钟精度依赖，补充 local DooD、SSH 共享目录及 CI 派生版本回归测试。
- 用户明确“不要操作我的浏览器”；后续验证仅使用代码、自动化测试和静态检查，页面实际效果保留人工验收。
