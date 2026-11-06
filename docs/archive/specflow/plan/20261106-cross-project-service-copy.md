# Plan / 计划：跨项目复制服务
最后修改时间: 2026-11-06 22:23:46

## Review status / 审查状态

Accepted

后端方案、版本范围和两个 task 的局部安排已接受；T01/T02 实施及任务交付审查已完成。用户于 2026-11-06 要求“开始实现”；授权按 T01 → T02 实施及其必要开发检查，不包含 Git 提交/推送、数据库资源操作或实际部署。

用户随后要求“完成验证”，授权主会话完成整体 Verification；现已接受交付并归档，结论与限制见[验证记录](../verification/20261106-cross-project-service-copy.md)。后文实施阶段的待验证表述保留为当时记录，以文末最终验证结论为准。

## Intent basis / 意图依据

- [总意图](../intent/20261106-cross-project-service-copy.md)，验收 A01–A08。
- 原样复制应用、服务配置、环境值与组件覆盖；弹窗版本范围默认“最新版本”，另可选择“全部版本”。最新模式只复制最新版本及组件并绑定该版本；全量模式复制所有版本并绑定源服务当前版本的对应副本。冲突拒绝，由用户之后编辑并部署。
- 调查基线：`develop`，`557763af`；产品工作树无修改，仅有本功能过程文档。

## Technical decisions / 技术设计与决策

### 复制职责与现有能力

- 在 `internal/application/service/usecase/copy.go` 增加独立 `CopyService`，通过 `NewCopyService(service Service, application applicationCopier, environment copyEnvironmentReader)` 装配。它拥有源服务读取、目标权限和编码检查、应用复制编排、目标服务及覆盖写入；沿用现有 Service 的存储与权限能力，不改变普通 Service 构造器。
- 在应用领域增加 `CopyApplication(ctx, userId, sourceProjectId, targetProjectId, applicationId, copyAllVersions)`，返回 `applicationdto.ApplicationCopyResult`：新应用、源版本 ID 到目标版本 ID 的映射、源版本组件 ID 到目标组件 ID 的映射。服务领域通过只包含此方法的局部接口调用，不直接写应用仓库。
- 应用复制复用 `ApplicationDefinitionForUser` 的完整读取、现有 Repository 的创建方法及 context 事务；不调用会合成初始版本的 `CreateApplication`。
- 不直接使用 `CreateApplicationFromDefinition` / `CreateServiceFromDefinition` 作为复制写入路径：前者会清除组件产物快照并重新处理字段，后者经 `remapServiceComponents` / `normalizeOverlay` 会清掉部分与版本相同的显式覆盖。复制只改变身份和归属，不能重新解释源配置。
- 复制方法直接操作已定义的模型和持久化契约，不新增 normalize、默认值补齐、通用克隆框架或兼容分支。现有导入、编辑与版本切换行为保持各自职责，不顺带重构。

### 字段与引用

| 数据 | 复制规则 |
| --- | --- |
| Application | 新 ID、目标 ProjectId、新建时间；Name、Code、Kind 原样保留 |
| Version | 按所选范围复制，生成新 ID；ApplicationId 和集合内 CreatedFromVersionId 映射到副本，Label、Status、Note、ComponentSummary 保留 |
| VersionComponent | 为所选版本复制组件，生成新 ID 和 VersionId；镜像、命令、身份、环境声明、挂载及文件内容、依赖、健康检查、资源、端点和其他声明原样保留 |
| 组件产物来源快照 | 保留已有 ArtifactId 和 Artifact 元数据作为历史来源信息；不复制 CI Run、Artifact 记录或镜像文件，不凭历史 ID 读取源项目运行资源 |
| Service | 新 ID、目标 ProjectId、新 ApplicationId；默认模式绑定最新版本副本，全量模式绑定源服务当前版本副本；保留 Code，DeploymentDirectory 使用目标目录确认值，Status 为 `stopped` |
| ServiceEnv / ServiceComponent | 环境键值及稀疏覆盖原样保留，服务组件生成新 ID、ServiceId 并映射 SourceVersionComponentId；子记录 ID 由现有 Repository 生成 |
| 部署事实 | CurrentDeploymentId 为空，RuntimeDirectory 为空，DirectoryTargetRevision 保存目标环境确认修订，RuntimeTargetRevision 为 `0`；不复制历史部署状态与源环境目录确认结果 |

部署目录使用目标环境工作区下的 `deployment/服务编码` 作为初值，复制弹窗允许编辑。前端复用 `useDeploymentDirectory` 和 `DeploymentDirectoryField`；后端读取目标环境元数据、检查目标修订并复用 `NormalizeServiceDirectory` 按平台校验。创建服务后使用已有 `UpdateServiceDeploymentDirectory` 保存确认路径及目标修订。挂载来源路径等其余业务配置仍原样复制；不探测运行环境或读取 SSH 私钥。

先依据版本范围选择待复制版本，为所选版本建立 ID 映射，并按来源版本先于派生版本的顺序创建。全量模式在集合内映射 `CreatedFromVersionId`。最新模式只创建最新项；若来源版本未包含在副本中，将目标 `CreatedFromVersionId` 置空，使其成为目标应用的起始版本，不指向源项目。最新版本采用现有版本列表的 ID 降序首项。复制结构体后只修改身份、归属和新建元数据；不修改共享子对象，不为未改动的配置建立递归深拷贝体系。

最新模式按组件名称映射源服务覆盖；若源服务组件集合与最新版本不一致，返回冲突并提示选择全部版本，避免丢弃覆盖或保留无效引用。全量模式直接映射源版本组件 ID。

### API 与事务

- 路由：`POST /api/service/:service_id/copy?project_id=<源项目ID>`。
- 新增 Proto `ServiceCopyReq { string target_project_id = 1; bool copy_all_versions = 2; string deployment_directory = 3; int64 environment_target_revision = 4; }` 和 `ServiceCopyResp { string id = 1; }`。`copy_all_versions` 默认 false 对应最新模式，true 对应全量模式。成功返回 `201` 与新服务 ID；请求、响应不携带环境值或组件内容，避免 HTTP body 日志记录凭据。
- 应用入口：`CopyService.Copy(ctx, userId, sourceProjectId, serviceId, input servicedto.ServiceCopyInput) (model.Service, error)`。HTTP 只完成认证、解码、调用与响应映射；单独 `CopyHandler` 沿用现有 authenticator 和统一错误 writer，不改变其他 Service handler 的构造签名。
- 外部边界检查源项目、源服务、目标项目标识必填，源目标项目不同；源服务读取与目标写入分别核对成员权限。目标需有环境配置以计算和确认目录，不要求 Probe 成功或容器可运行。
- 应用复制先核对目标权限及应用名称、编码，随后检查目标服务编码；任何冲突由同一事务回滚；三种冲突均返回 `KindConflict` / HTTP `409`。缺失输入或同项目为 `400`，权限不足为 `403`，项目或源服务不存在为 `404`，存储失败使用现有安全 `500` 契约。
- 保留现有数据库唯一索引作为最终约束，不自动重试或改名；数据库写失败沿现有错误边界回滚，不新增 SQL 驱动错误解析体系。
- 路由注册在现有 `MutatingUnitOfWork` 覆盖范围，复用请求 context 中的一次数据库事务；成功响应由中间件在提交后输出。Copy 用例要求调用方提供写事务，当前只开放 HTTP 入口，不新增 MCP、CLI 或异步任务。
- 不增加迁移或 SQL 查询：应用、版本、组件和服务的现有创建方法已支持必要写入。失败通过事务回滚恢复，不增加补偿删除。

### 前端交互

- 入口已由用户确认在 `ServiceDetail.vue` 右上操作区，与预览、部署等按钮并列；源服务由详情上下文确定。
- 在现有页面内加入 `AppDialog`、`ComboboxSelect`、`AppDialogActions`、版本范围 SelectControl 和少量局部状态，不另建 store、composable、表单模型或专用弹窗文件。
- 项目选项使用 `projectStore.activeProjects` 并排除源项目；项目列表尚未加载时沿用 `fetchProjects`。不调用目标环境探测或部署就绪检查，最终权限和唯一性以后端为准。
- 版本范围使用已有共享 `SelectControl`，只提供“最新版本 / 全部版本”两个值，首次打开默认最新版本。选择项目并成功加载目录前禁用确认；目录错误显示在目录字段下方；异步冲突、权限及请求失败显示在弹窗提交错误区，保留选择供用户调整。沿用现有 `app-field-error` 与控件 API，业务文字不改成 12px。
- 使用现有 `operating` 状态阻止重复提交，请求期间禁用取消与确认并阻止弹窗关闭；关闭或重开时重置局部选择和错误。
- 成功后通过已有 project store 切换目标项目，使用复制接口返回的新 ID 跳转 `/service/:id` 并关闭弹窗、提示成功；项目持久化沿用 setActiveProject，不另写 storage 或展示凭据。无目标项目时显示空选项提示并禁用确认。

## Implementation steps / 实施步骤

1. 主会话核对完成的意图与本计划；接受总计划时一并接受两个 task 方案。
2. 执行 T01：后端复制、Proto、HTTP 和装配，完成主要流程及失败边界检查，提交可核对的 diff 与接口交付。
3. 主会话审查并接受 T01 后执行 T02：接入生成类型、API、复制按钮、弹窗和双语文案。
4. 主会话汇总开发检查与任务交付；后续获 Verification 授权后逐项核对 A01–A08，记录真实验证结果。

## Files to change / 修改范围

- T01：应用和服务 usecase / DTO、HTTP service handler 与路由、`internal/bootstrap/http.go`、service Proto 及生成物、对应后端测试。
- T02：`web/src/api/service/service.ts`、经确认的详情页、`web/src/i18n/locales/zh-CN.ts` 和 `en-US.ts`。
- 具体写入范围、检查和不扩展边界见 task。T01 独占 Go/TypeScript Proto 生成物；T02 不改 Proto 或运行生成命令。

## Tasks and coordination / 子任务与协调

### Coordination / 协调信息

- 方案修订：3。
- 权威协调根：`D:/SourceCodes/mywork/pomelo-orbit`；本计划为分配、依赖与审查的唯一记录。
- 仲裁会话：当前主会话 `/root`。
- 阶段授权：2026-11-06 用户先要求拆分并制定计划，随后明确“开始实现”；总计划和两份 task 方案一并接受，授权当前会话按 DAG 顺序执行 T01 与 T02 及其必要开发检查。不授权子 agent、Git 提交/推送、数据库资源操作或部署。
- 验证授权：2026-11-06 用户明确要求“完成验证”，主会话对修订 3 的整体交付完成代码与检查核对；沿用现有 T01-002 / T02-003，不新增任务或执行者。

### Task definitions / 任务定义表

| ID | 任务文档 | 直接前置任务与要求 | 预期交付与验收 | 意图验收归属 |
| --- | --- | --- | --- | --- |
| T01 | [后端复制能力](../task/20261106-cross-project-service-copy/T01-backend-copy.md) | 无任务前置；总意图、计划与 task 方案需已接受并取得实施授权 | 原样复制用例、HTTP 接口、生成类型、权限/冲突/回滚证据；主会话核对 diff 与检查后接受 | A02–A07、A08 后端部分 |
| T02 | [前端入口与弹窗](../task/20261106-cross-project-service-copy/T02-frontend-copy.md) | T01 已接受的复制接口、Go/TypeScript 生成物及对应可访问版本 | 详情页复制入口、单项目选择弹窗、错误及成功反馈、前端检查 | A01、A08 前端部分，A04/A07 交互部分 |

### Execution registry / 执行登记表

| ID | 执行状态 | 执行者 | 分配编号 | 工作基线 / 交付位置 | 仲裁结论 / 当前阻塞 |
| --- | --- | --- | --- | --- | --- |
| T01 | accepted | `/root` | T01-002 | 基线 `557763af`，最终产品 diff 指纹见验证记录 | 2026-11-06 22:23:46 复核接受；后端契约、目标目录、配置保留和事务证据通过 |
| T02 | accepted | `/root` | T02-003 | 同一工作树及 T01-002 接口，最终产品 diff 指纹见验证记录 | 2026-11-06 22:23:46 复核接受；成功跳转与路由重新加载静态核对、前端检查通过 |

主会话 2026-11-06 21:17:29 接受 T01，并分配 T02-001 给当前会话；依赖交付为同一工作树的后端 diff 和生成类型。

### DAG and integration / DAG 与集成

`T01 → T02`。两任务串行，主会话负责验收归属、交付审查与最终集成；不启动执行会话。T01 交付后若协议变化，先修订总计划和受影响 task，再重新核对前置交付。A01–A08 的整体一致性由 Verification 汇总，不能以单个 task 自报完成代替接受。

### Assignment history / 分配历史

| 时间 | ID | 分配编号 | 执行者 | 动作与依据 |
| --- | --- | --- | --- | --- |
| 2026-11-06 20:25:27 | T01、T02 | 无 | 无 | 按用户要求建立方案，尚未分配 |
| 2026-11-06 20:51:00 | T01 | T01-001 | `/root` | 用户要求开始实现；主会话按已接受方案接收 T01，基线 `557763af3bea5f4570c64a95978b0c5f259f9e00` |

## Verification plan / 检查与验证安排

- T01：端口替身分别验证默认最新和全部版本两种范围、Service 版本绑定与版本来源关系、ID 映射、显式空值/覆盖、冲突与权限；真实临时 SQLite + 当前迁移验证副本可回读和中途失败整体回滚，明确归为集成验证，不以 fake 写入记录证明回滚。
- HTTP 协议检查只覆盖本接口请求与成功 ID 响应、必要错误分类及事务装配；复用既有统一错误和事务中间件测试，不重复整套状态码或构造完整服务器。
- T02：检查版本范围 SelectControl 默认值及切换后请求参数、必填项目反馈、失败保留选择、成功切换目标项目并展示新服务、请求锁定及状态清理；不挂载整页串联 Router/Store/API，不为简单赋值和薄 API wrapper 强加测试文件。按实际出现的独立业务逻辑决定轻量单测。
- 实施后的项目检查：根目录 `task check`、`go test ./cmd/... ./internal/...`、`task test`；前端修改后在 `web/` 运行 `corepack yarn lint:fix` 和 `corepack yarn typecheck`。已有检查通过后不无故重复全量执行。
- 无开发服务器启动或实际部署安排。若后续运行态 UI 验证没有现成且获授权的环境，明确记为未验证，不以静态检查代替。实施阶段执行这些产品检查并记录结果。

## Risks and recovery / 风险与恢复

- 原样配置可能含目标环境不可用的路径或连接地址；按用户要求保留，用户后续编辑，复制阶段不新增业务适用性校验。
- 防止已有定义工厂的规范化影响复制；以直接读取的持久化模型对照副本，而非仅断言数量。部署目录使用目标确认值及目标修订，源运行目录清空，避免错误宣称副本已在目标运行。
- 复制无外部副作用，数据库失败依赖 UoW 回滚；产品代码恢复只撤销本功能 diff，保留用户其他修改，不做资源删除或数据库重置。

## User review notes / 用户审查记录

- 2026-11-06：用户授权拆分与计划，建立 T01/T02 及依赖；未启动开发、子 agent、生成命令或测试。
- 2026-11-06：用户确认入口是服务详情页及版本范围默认最新、可选全部。
- 2026-11-06：用户要求开始实现，接受总计划和 task 方案并授权主会话按 DAG 实施及运行必要开发检查；不含 Git 写入、部署或外部数据库操作。

## Implementation delivery / 实施交付

- 2026-11-06 21:26:49：T01 和 T02 均已交付并经主会话接受，依据当前工作树 diff 与各 task 中的检查证据，无 Git 写入或外部资源操作。
- 最终检查：task check、task test、go test -count=1 ./cmd/... ./internal/... 通过；前端 lint:fix、typecheck、格式检查与显式日志单测通过，43 个文件/193 个用例，无警告。
- 用户纠正版本范围控件后，最终复用已有 SelectControl，默认最新、可选全部；没有新增前端组件或状态管理文件。
- 正式整体 Verification 尚未进行；浏览器交互和窄屏布局未实测。当前记录仅表示实现、开发检查和任务交付完成。

## Revision 2 / 目标目录调整（已接受）

用户明确要求目录按目标实际情况计算、展示并允许编辑，授权继续实现。此前关于原样保留 DeploymentDirectory、修订为 0、目标无需环境配置的规定由本节替代。

- 复用 useDeploymentDirectory 和 DeploymentDirectoryField：目标环境 workspace_root/deployment/服务编码作为初值，不传源目录；切换目标重新计算，提交保存用户编辑值。版本范围继续使用 SelectControl。
- ServiceCopyReq 新增 deployment_directory、environment_target_revision；既有 service DTO 增加 ServiceCopyInput，handler 映射输入。CopyService 注入已有环境用例的 EnvironmentForUser 窄接口，在写入前核对目标权限/环境修订并复用 NormalizeServiceDirectory 按平台校验。只读取环境配置，不执行探测、不读取 SSH 私钥。
- 副本 DeploymentDirectory 保存用户确认的目标路径，DirectoryTargetRevision 保存目标修订；RuntimeDirectory 仍为空、RuntimeTargetRevision 为 0，实际运行事实由后续部署建立。
- T01-002 由当前主会话执行，修改 DTO/Proto/复制用例/handler/bootstrap 和已有测试；T02 在新契约接受后接目录字段，不新增页面组件或状态封装。
- 检查聚焦 Windows 源复制到 Linux 目标的编辑路径持久化、平台不符及环境修订变化拒绝、目录载荷与既有默认目录算法；其余配置保留及事务测试继续有效。

- 2026-11-06 21:42:50：T01-002 定向测试（service/usecase、HTTP service handler、bootstrap）通过，主会话接受新增目录输入、目标环境读取及平台/修订校验。接收 T02-002，继续前端目录字段。

## Revision 2 delivery / 修订 2 交付

- 2026-11-06 22:03:56：主会话接受 T02-002；当前工作区的目标目录字段及 payload 与 T01-002 接口一致，T01/T02 修订 2 均完成。
- 前端复用已有目录 composable 和字段，选目标项目后重新计算初值，允许编辑并随复制保存。后端复用目标环境元数据读取和平台路径校验，保持 HTTP UoW、环境值和稀疏覆盖写入边界。
- 检查结果：task check 通过（Go lint 0 issues），task test 通过；此前针对同一产品代码运行的 go test -count=1 ./cmd/... ./internal/...、web lint:fix/typecheck 及显式日志单测均通过，前端 43 个文件/193 个用例，无警告。没有新增复杂 UI、性能或并发测试，也没有运行覆盖率。
- 测试仅更新现有主要流程与真实失败边界；SQLite 用于配置回读和跨领域事务回滚的持久化证据，保留 !unit 分类。
- 用户要求继续后保留现有暂存区，收尾文档及轻量测试修正未追加暂存；检查依据为当前工作区。浏览器交互与窄屏布局仍未实测，正式整体 Verification 尚未进行。

## Revision 3 / 成功跳转（已接受）

2026-11-06 22:11:01：按用户要求，T02-003 由主会话接收实施。复制成功后在既有 operating 回调内取得新服务 ID，调用 setActiveProject 后 router.push；详情页 serviceId 改为路由计算值并监听加载，避免复用页面仍请求源服务。仅前端及必要文档调整，T01 接口不变；只运行必要前端检查，不新增整页、性能、并发或覆盖率测试。

修订 3 交付：2026-11-06 22:15:11，T02-003 Accepted。复制成功使用返回 ID 导航至目标项目服务详情；serviceId 改为路由计算值，监听 ID 变化重新加载，后续操作使用当前 ID。web 的 lint:fix、typecheck、Prettier 及显式日志单测通过（43 文件/193 用例，无警告）。没有新增前端测试文件；后端未改动，不重复后端检查。浏览器跳转未实测；未修改现有暂存区。

## Final verification / 最终验证

2026-11-06 22:23:46：按用户“完成验证”的授权，完成整体 Verification，T01-002 和 T02-003 的集成交付接受；A01–A08 均有对应代码及检查证据，未发现需修复的产品偏差。本次只更新和归档过程文档，产品实现未改动。

- 本轮 `task check` 通过（Go lint 0 issues，包含前端类型、lint、格式），`task test` 通过，`go test -count=1 ./cmd/... ./internal/... ./sql` 通过。
- web 的 `corepack yarn lint:fix` 和显式日志单测通过，43 文件/193 用例，无警告；未新增测试或覆盖率配置。
- 核对成功使用返回的新服务 ID 切换目标项目并导航，详情随路由 ID 变化加载，后续 API 读取当前 ID；目录 composable 随目标项目变化重置，关闭重开不保留上次目标目录。
- 浏览器交互、实际跳转和窄屏布局未实测；未启动开发服务器、操作现有数据库或执行部署。
- 当前工作树相对基线的 21 个产品文件及其 diff 指纹、逐项验收和检查证据见[验证记录](../verification/20261106-cross-project-service-copy.md)。保留原暂存区，未暂存、提交或推送；同名意图、计划、task 和验证记录一并归档。
