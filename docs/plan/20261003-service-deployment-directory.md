# 服务部署目录与 Gateway 入口实施计划
最后修改时间: 2026-10-04 01:22:01

Review status: Accepted

Mode: standard

## Intent basis

依据 [已接受的意图](../intent/20261003-service-deployment-directory.md)：部署时根据当前环境拼出完整目录，用户可确认、修改并选择环境工作区之外的路径。Gateway Application 正确标记为 `kind=gateway`，已有类型按真实关联纠正；服务入口依据所属应用类型禁用 Gateway 部署。普通服务与 Gateway 专属部署入口使用一致的路径控件。已部署服务改目录只显示风险警告，不要求停服、不增加确认步骤、不阻止有效提交。实现覆盖本地、Linux/Windows SSH 和 Orbit 容器化的 DooD。

## Current findings

- `internal/application/gateway/usecase/service.go` 的创建逻辑明确使用 `ApplicationKindStandard`；[决策 C-04](../decisions/ledger.md) 目前指定 Gateway 身份由 `gateway_config` 关联表达。用户已明确要求改变这一设计：纠正创建逻辑及已有真实 Gateway 类型，并让服务入口直接依据 `application_kind=gateway` 禁用部署；实现时同步改写 C-04，不保留类型与额外识别字段并行的方案。
- Application 和 Service 的响应已经提供 `kind` / `application_kind`，不需要新增 `is_gateway`。普通应用编辑只更新名称和编码，不会把 Gateway 改回 `standard`；需保留此类型行为。
- `SelectGatewayDeploymentVersion` 按已保存的 ACME profile 选择绑定版本。普通服务弹窗允许自由选择版本，但该选择随后可能被网关流程重新选择；应禁用错误入口，而非让其模拟普通服务部署。
- Gateway 页面目前也调用 `serviceApi.deploy`。按钮限制属于 Web 入口规则，不能在共享服务部署 API 中一概拒绝 Gateway，否则正确入口也无法工作。
- 部署、重启、停止、状态、容器日志及 MCP 运行时工具都通过 `Runtime` 按服务编码推导目录。只增加前端字段不能完成此需求。
- 本地运行时已区分 `ServiceDir` 与 `ComposeMountSourceDir`。DooD resolver 根据 Orbit 容器实际挂载进行最长路径匹配；当前渲染主要映射服务根目录后拼接相对子路径，嵌套挂载需要按具体挂载源检查。
- Gateway 的路由、证书、ACME 和发布记录同样依赖服务目录；本地和 SSH 文件读取目前限制在 Environment 根目录内，需要改为明确的服务目录范围。

## Implementation steps

### 1. 纠正 Gateway 类型并补齐目录持久化

- Gateway 创建逻辑使用现有的 `ApplicationKindGateway` 常量。通过新增迁移将 `gateway_config` 关联的现有 Application 类型统一纠正为 `gateway`；不按名称、code 或镜像推断，不修改已执行迁移。检查初始化、Gateway 列表与通用应用/服务读模型，确保类型一路正确返回。
- 服务入口只使用现有 `application_kind`；GatewayConfig 和 Environment 绑定继续决定 profile、配置及目标归属。检查现有按 `standard` 过滤的查询与界面：普通业务暴露列表可以排除 Gateway，Gateway 自身操作不能依赖 `standard`。保留普通应用编辑不修改类型的现状，不添加多余类型编辑入口。
- 在 Service 中区分 `deployment_directory` 与 `runtime_directory`。前者保存用户确认的目录用于再次部署与配置判断；后者保存已经准备好且用于生命周期命令的目录，用于重启、停止、查询和日志。新建且尚未部署的服务没有运行目录。目录的环境身份及 target revision 一并记录，不能把旧目标的目录当作当前目标目录。
- 未确认目录的服务使用当前 Environment 标准布局作为部署表单候选值；这属于新服务的产品默认行为。用户确认后的完整路径原样保存，`~` 仅在目标使用时展开。环境修订变化后重新生成、校验并确认候选目录。
- Deployment 增加正式的 `working_directory` 快照列，与现有版本、Environment identity/revision 共同确定执行输入。部署、重启和停止均记录其实际使用的目录；不把目录隐藏在命令选项 JSON 中。
- Service 读模型保留所属应用的正确 `application_kind`，并返回确认目录、运行目录及所属环境修订，供表单回填和改目录警告使用。Gateway 响应提供其受管 Service 的相同目录信息，不在 GatewayConfig 中重复持久化目录，也不增加与应用类型重复的识别字段。
- 新增 SQLite/MySQL/PostgreSQL 的对应编号迁移。按既有环境布局一次性初始化旧 Service 的确认目录；仅在目标修订匹配的运行或成功任务历史存在时初始化运行目录，单凭 Service 状态不足以确定目录。迁移只改变库存，不创建、移动或删除目标文件。新代码直接使用新字段，不保留旧目录推导的兼容分支。
- 旧 Deployment 无法可靠恢复的历史目录保持“未记录”，不伪造历史快照。旧排队任务只有在保存的目标修订与当前 Environment 一致时，才可在迁移中补齐原规则下可确定的目录；不能确定的任务在执行前明确失败，不猜测路径。升级窗口需核对现有排队和运行任务。
- 更新 SQL 查询、Repository 映射、领域模型、DTO 和 Proto，使用 `task sqlc`、`task proto` 生成代码；不修改已执行迁移。

### 2. 收敛部署请求与目录校验

- 扩展共享 `ServiceDeployReq`，接收确认后的完整 `deployment_directory`、当前环境 target revision，以及普通服务可提交的 `version_id`。普通服务列表/详情直接在一次部署请求中提交所选版本和目录，移除部署弹窗中“先 PUT 版本、再 POST 部署”的中间状态；独立的服务基本信息编辑能力继续保留。
- Gateway 页面提交目录及既有选项，不提供普通服务的自由版本选择。后端依旧按 GatewayConfig 的 ACME profile 选择版本，保留 Gateway enrichment、自动强制重建和管理 API 就绪检查。
- 用现有请求事务和应用层窄端口完成版本映射、确认目录保存、Deployment 创建与队列任务写入。Deployment 用例不直接依赖 Service 用例包；普通版本切换复用 Service 已有的组件映射规则，通过消费方定义的窄端口调用。MCP 直连用例时沿用当前事务装配约定。
- 提交时检查 Project 归属、成员权限、当前环境修订、既有 active deployment 约束及路径格式；用户打开弹窗后环境变化时返回明确冲突，要求重新确认。目录改变本身不增加状态限制，不自动先停止旧服务。
- 允许平台绝对路径或 `~` / `~/...`；不接受依赖未知进程工作目录的普通相对路径。Linux、Windows SSH 和 local 分别按执行目标平台校验，不能用控制面 Linux 的 `filepath.IsAbs` 判断远端 Windows 盘符路径。
- 远端认证、SFTP、目录可写和 Docker 路径映射等 I/O 放在 Worker 的目标修订校验之后执行，不在 HTTP 写事务中完成。环境 Probe 就绪不能代替本次目录检查。
- 对具体解析后的目录检查服务归属和占用；DooD 还需检查实际物理挂载源的占用，避免不同容器别名指向同一宿主机目录。归属依据为 Service 的确认目录、运行目录及运行容器的实际 bind mount；不能占用其他服务的目录，既有 Compose 按本次有效计划替换，运行目录为空不阻止部署。技术校验失败仍拒绝执行，与只提示的改目录风险警告分开。
- 物理数据目录占用依据当前目标 Docker 的运行容器：通过其他服务的 Compose project label 查询运行容器，inspect 实际 bind mount 与运行状态。将本次部署根目录、相对挂载的物理源与实际使用的源比较，包含嵌套源与父子目录重叠；不从其他服务当前可变的版本重建运行挂载。查询或解析失败时拒绝继续准备文件。停止容器不构成运行数据占用，本服务自己的挂载不阻止其再次部署；原有根目录归属约束继续适用。
- 配置 hash 纳入部署目录及其环境关联；任务 hash 使用本次快照实际选定的目录。Service 的待部署判断继续比较确认配置与最近成功任务，不把失败任务误判为已应用配置。
- 同步 MCP 部署输入、工具说明和输出映射，使其显式提交或确认当前目录并使用同一用例，不通过新别名保留旧新业务路径。

### 3. 运行时显式使用服务目录

- 扩展现有部署端口，以一个明确的服务位置值携带服务编码和目录，替代各操作只收到 service code 后自行推导路径的方式。保留 Environment target 的显式分派；Runtime 不查询数据库，也不根据 SSH host 或错误猜测 local。
- 部署使用 Deployment 的确认目录快照；重启、停止、状态、容器日志和运行时工具使用该服务在当前目标上的运行目录。重启不隐式切换到上一次尚未执行成功的待部署目录；相关任务冻结其选用目录。
- local 在 Orbit 可见目录写 Compose 和受控文件；SSH 使用远端 SFTP 目录。目标上的 `~` 展开和路径规范化集中在现有运行时实现中，得到的目录在本次操作内复用，文件准备、挂载源和命令工作目录保持一致。
- DooD 继续通过既有 resolver 获取 Docker daemon 可见的物理源，不要求用户填写宿主机路径；不允许把未映射的容器私有路径直接交给 daemon。对具体的相对 bind source 应用实际挂载映射，处理比服务根目录更深的独立挂载，而不是仅做字符串拼接。显式 absolute host-path mount 和 named volume 保留现有语义。
- Linux SSH 沿用 POSIX 引号及远端 shell；Windows SSH 沿用 SFTP 盘符处理、非交互 PowerShell、`Set-Location -LiteralPath` 与 `docker.exe`。Windows 远端路径不经过控制面 DooD resolver，不切入 WSL 执行 Compose。
- Compose project、容器身份、共享 `traefik` 网络和独立的部署日志目录继续使用现有稳定标识；自定义目录名不成为新的 Docker project。

### 4. 明确执行失败和目录切换时点

- 请求接受时保存确认目录和任务快照，生命周期操作仍定位旧的运行目录。Worker 在目标、目录、挂载源检查及新 Compose/受控文件准备完成之前，不切换运行目录。
- 文件准备成功且即将发出 Compose 生命周期命令时，以短事务将该任务目录绑定为 Service 的运行操作目录，并记录目标身份和修订；外部 Docker/SSH 命令不占用此事务。目录绑定失败时不发出命令。
- 此后的命令失败或取消可能已部分改变同一 Compose project；保留新运行操作目录，并按该目录和固定 project 查询、停止或清理，不退回“最近成功目录”而遗漏新资源。绑定前的失败或取消保留旧运行目录。
- 执行结束后的状态和版本更新不能覆盖已保存的目录绑定；取消协调与容器查询使用同一任务位置。禁止排队任务后续再读取可变确认目录来重新选路径。
- 不自动执行 `down`、迁移数据或删除旧目录。切换目录的有效请求可以在服务仍运行时执行，风险由明确警告告知用户。

### 5. Gateway 文件发布沿用服务运行目录

- Gateway 运行投影除 service code 外补齐其运行目录和目标绑定，RouteManager 用该位置解析 `gateway/dynamic`、`gateway/certs`、`gateway/acme` 与 `.orbit/route-publication`。
- WorkspaceFiles/SyncFiles 接收明确的服务范围，读写仍限制在该服务的受管目录内；支持 Environment 根目录之外的 Gateway，不简单删除现有范围校验。
- Route 的 publication fingerprint、请求内 session 依赖和实际容器挂载核验包含 Gateway 服务位置，目录变化使旧 preview 过期；避免将旧根目录的发布结果当作新目录已同步。
- 不自动复制旧路由、证书或 ACME 数据，不从业务 Route 发布草稿。用户需要在新目录另行显式同步 Route；Gateway 部署与 Route 同步继续遵循 C-17 的独立边界，不引入相互部署锁或自动恢复流程。

### 6. 统一目录控件和部署入口

- 在共享组件目录增加 `DeploymentDirectoryField.vue`，复用项目 input、字段错误和警告样式；统一完整路径输入、目标平台格式校验和风险警告。目录初始化及目标切换逻辑集中在小型共享 helper/composable，组件不自行连接 SSH 或调用 Docker。
- 普通 ServicePage 的卡片/表格和 ServiceDetail 根据 `application_kind === 'gateway'` 禁用 Gateway 的部署按钮；通过共享 Tooltip 提示从网关入口部署，点击处理器也检查同一条件，避免残留状态打开错误弹窗。保留普通服务按钮现有任务执行中与操作中禁用规则，不因服务容器正在运行而禁止部署或改目录。
- 普通服务部署弹窗保留版本选项，加入共享目录字段。首次打开按当前 Environment 配置拼接，已有确认目录在目标修订匹配时回填；修订不匹配时重新生成候选值并重新确认。切换 Project、加载失败或尚未取得目标信息时不能提交旧表单。
- GatewayDetail 部署弹窗复用同一目录字段和初始化规则，保留现有 force-recreate 交互及 Gateway profile 版本流程。前端从相同 Service 读模型取得目录基线，不另存 Gateway 目录。
- 判断“已有部署”使用运行目录和部署依据，不能仅以当前 `status=running` 判断；已停止但曾部署过的服务修改目录同样需要警告。比较规范化后的输入与原运行目录，恢复原目录后撤下警告。
- 警告说明相对数据挂载会改变、旧数据不会迁移，以及可能重建容器和中断服务。Gateway 补充路由、证书、ACME 数据及显式 Route 同步风险。警告不增加勾选、确认弹窗或提交禁用条件。
- 同步中英文文案及 API types。产品界面只展示路径、字段错误和实际部署风险，不暴露内部映射实现细节。

### 7. 同步活文档与完成实现检查

- 更新 CD 产品模型、运行时与工作目录指南，说明自动拼接、完整目录覆盖、目标路径语义、DooD 映射、Gateway 专属入口和警告但允许改目录的规则。同步修订决策 C-04：Application.kind 表达普通应用与 Gateway 的明确分类，GatewayConfig 保存配置，Environment 绑定校验归属。
- 补充能证明本次行为的单元与组件测试，并运行仓库要求的检查。按用户要求不新增或启用真实 SSH、Docker、浏览器集成流程，不启动、停止或重启开发服务器。
- 实现完成后报告实际 diff、检查结果与剩余风险，按标准模式停在 Implementation，等待用户要求进入 Verification；本轮 Plan 不实现产品代码。

## Files to change

| 范围 | 预期文件 |
| --- | --- |
| 持久化 | `sql/migration/{sqlite,mysql,postgres}/<next>_service_deployment_directory.{up,down}.sql`、`sql/query/service/service.sql`、`sql/query/deployment/deployment.sql`、`sql/query/gateway/gateway.sql`、对应 Repository 与 SQLC 生成物 |
| 模型与用例 | `internal/model/{service,deployment,gateway}.go`、`internal/application/service/{dto,usecase}/`、`internal/application/deployment/{dto,port,usecase}/`、`internal/application/gateway/{dto,port,usecase}/`、必要的应用类型投影检查与 bootstrap 端口装配 |
| 目标运行时 | `internal/infrastructure/runner/{local,ssh,target}/`、`internal/infrastructure/storage/local/physical_data_root.go`、必要的 `internal/common/workspacepath/` |
| Gateway 发布 | `internal/infrastructure/external/traefik/`、`internal/application/route/` 中服务位置相关依赖与 hash；不扩展 Route 发布业务流程 |
| API 与工具 | `proto/orbit/v1/{service,deployment,gateway}/`、HTTP service/deployment/gateway mapper、`internal/api/mcp/delivery/`、Go/TypeScript 生成物 |
| 前端 | `web/src/components/DeploymentDirectoryField.vue`、共享目录 helper/composable、`web/src/views/service/{ServicePage,ServiceDetail}.vue`、`web/src/views/gateway/GatewayDetail.vue`、相关 API 与中英文文案、对应组件/工具测试 |
| 活文档 | `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/decisions/ledger.md`、`docs/guides/deployment.md`、`docs/guides/volume-mounting.md` |

## Verification plan

1. 用例测试覆盖普通服务的版本与目录快照、Gateway profile 选择、环境修订过期、工作区外有效目录，以及准备前失败保留旧目录、开始执行后失败/取消保留本次操作目录。使用 fake store/runtime，断言命令实际收到的目录与后续操作的位置。
2. 路径单元测试覆盖 local、Linux SSH、Windows SSH、主目录展开、带空格路径和 DooD 映射；重点验证未映射路径拒绝以及嵌套挂载映射到实际源。沿用现有编码和引号测试，不运行真实远端。
3. 验证 Gateway 创建及既有库存迁移使用 `kind=gateway`、应用与服务响应正确投影类型；组件测试依据该类型检查卡片、表格与详情中的按钮禁用，并验证 Gateway 专属目录控件仍能提交。无 Gateway 关联的普通 Traefik 应用保持 `standard` 且可从服务入口部署；应用普通编辑保留原类型。
4. 前端测试覆盖环境默认路径、已确认路径回填、Project/环境变化处理、工作区外路径、已部署但停止状态下的改目录警告、恢复原目录，以及有警告仍可提交。测试实际提交值和可见状态，不断言无关样式或内部实现。
5. Route 文件边界与发布依赖测试覆盖环境根之外的 Gateway 文件操作、目录变更使 preview 过期，以及新目录未自动迁移或发布旧路由数据。以 fake target 文件接口验证，不运行浏览器或容器集成。
6. 生成及仓库固定检查：`task sqlc`、`task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`，并运行本次相关的 Vitest。对固定 Go 测试入口中已有的显式真实目标用例保持未启用，不设置 E2E 开关。
7. Verification 阶段核对 Intent/Plan 与实际 diff、目录切换时点、Gateway 身份及全部入口；记录真实 Linux/Windows SSH 与 DooD 未进行集成验证的缺口，不把单元检查等同于实机部署验证。

## Blockers

无已知需要用户决定的阻塞。真实 SSH/Docker 环境不作为本次实施和单元检查的依赖；升级旧库存与未知历史目录的处理需在实现时按实际记录核对。

## Assumptions

- 用户确认工作区外目录可用、Gateway 从专属入口部署、改目录只警告不阻止；路径无效、目标过期和他人目录占用等技术错误仍拒绝。
- 目录配置属于 Service；GatewayConfig 只通过受管 Service 读取位置，不建立第二套目录库存。
- 用户明确纠正现行类型设计，Gateway 使用 `kind=gateway`；通过真实关联纠正库存，不按 Traefik 名称推断，服务入口不使用额外 `is_gateway` 字段。
- 继续使用现有单节点 API/worker、请求事务与非交互 local/SSH runtime；不新增泛用远端命令执行或跨节点工作区管理。
- 既有未提交的文本截断和列表调整由此前任务产生，实施时保留这些变更，实际 diff 评审明确本任务增量。

## Risks

- 运行中的服务允许切换目录，会重建同一 Compose project 并改变相对挂载数据位置；用户接受通过警告处理，系统不代办停服、迁移或回滚。
- Gateway 改目录可能使路由、手工证书和 ACME 数据缺失，并导致入口暂不可用或重新申请证书；需要明确警告和后续显式同步。
- Docker 命令失败或取消可能只完成部分容器更新；运行操作目录代表可用于继续管理该 project 的新 Compose 工作区，不保证全部容器已经使用新数据源。
- 旧服务历史可能被清空，单靠数据库无法还原其所有历史执行事实。迁移应基于可确定证据处理，不伪造旧 Deployment 目录，也不扫描或迁移远端数据。
- Gateway 类型纠正会影响按 kind 的过滤以及包含应用类型的配置 hash；核对暴露列表和待部署状态，不重写历史成功任务的 hash 或将普通 Traefik 应用误改。
- DooD 的实际容器挂载配置不在 Environment revision 内；执行时必须重新映射与校验，不能把表单显示的目录当作宿主机物理路径保证。
- 工作区外目录放开后，所有文件接口都要保留明确的 Service 范围，尤其是 Gateway publication 与证书操作。
- 本期不进行集成测试，真实 Linux/Windows SSH、Docker Desktop 和 DooD 部署行为仍有实机验证缺口，应在交付中说明。

## Rollback

- 产品代码回退不能直接丢弃自定义运行目录：先按记录的位置处理当前任务与容器，再由用户决定数据处置和回到原目录的部署；不自动删除任何目标文件。
- 在自定义目录仍被使用时保留新目录库存，不直接执行 down migration。字段删除和旧版本恢复需在数据库备份及运行资源已妥善处理后由用户明确授权。
- Gateway 回到旧目录时同样由用户处理证书/ACME 数据，并通过现有显式 Route 同步恢复入口；部署不自动恢复旧发布记录。

## User review notes

- 用户允许目录位于 Environment 工作区之外。
- 用户要求排查 Gateway 为 `standard` 的原因，并在服务入口禁用 Gateway 部署按钮；网关入口加入交互一致的目录控件。
- 用户要求已部署服务修改目录只警告，不阻止。
- 用户明确要求开始 Plan，Intent 已标记为 Accepted。
- 用户进一步明确要求纠正 Gateway 的 `kind=standard`；计划改为修正创建逻辑、按真实关联迁移已有类型，并由服务入口直接依据 `application_kind` 禁用部署，移除额外 `is_gateway` 方案。
- 用户要求更新文档后开始实现，计划标记为 Accepted，进入 Implementation。

## Implementation notes

- 新迁移编号为 000050；旧任务目录只按可确定的当前目标快照回填，新的配置 hash 不兼容旧 hash，升级前处理活动任务，升级后重新提交。
- 确认目录保留规范化后的目标路径（包括 `~`），命令使用本次解析后的目录；运行目录在文件准备成功、Compose 命令前绑定，取消协调重新读取实际绑定。
- 相对挂载源逐项映射；local 与 SSH 受管文件检查实际路径和范围，SSH 部署复用现有运行时 session。
- 按用户要求仅做固定检查和针对本次行为的单元、组件验证，不启动开发服务器或启用真实 SSH/Docker/浏览器集成流程；完成后停在 Implementation。
- Verification 修正 F-01：按用户确认新增运行容器实际挂载占用检查，复用现有 QueryAtEnvironmentRoot 与目标 session；Windows 比较统一盘符、分隔符、大小写及 Docker Desktop 的宿主机盘符映射。
- Verification 修正 F-02：拒绝规范化后变为盘符相对路径的 Windows 根目录，补充盘符根目录回归。用户确认集成测试无问题并要求完成验收；本代理仅运行固定检查和单元、组件测试。
