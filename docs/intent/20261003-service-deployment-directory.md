# 服务部署目录确认与修改
最后修改时间: 2026-10-04 01:22:01

Review status: Accepted

Mode: standard

## Background

当前服务列表和服务详情的部署弹窗支持选择版本。前端先更新 `Service.version_id`，再调用服务部署接口；部署目录由运行时固定推导为 `<Environment.workspace_root>/deployment/<service-code>`，用户无法在部署时确认或修改。

Worker 校验 Deployment 保存的环境修订和有效配置 hash，渲染 Compose、准备受控文件和相对挂载目录，再以服务编码作为 Compose project 执行部署。重启、停止、运行状态、容器日志和运行时工具同样按环境与服务编码推导目录。用户希望部署时同时确认版本和完整部署目录，目录应根据当前环境配置预先拼接，并允许修改。

部署目标包括本地、Linux SSH 和 Windows native OpenSSH + WSL2 Docker Desktop。Orbit 自身还可能运行在容器中，通过宿主机 Docker socket 执行 DooD；这时 Orbit 可写路径与 Docker daemon 可见的挂载源路径不同，不能将二者混用。

Gateway 创建用例明确将 Application 的 `kind` 写为 `standard`，与当前决策 C-04 的关联身份设计一致。用户在原因排查后明确要求纠正这一设计：Gateway Application 使用 `kind=gateway`，服务可以通过所属应用类型识别特殊服务并禁用普通部署入口。已有真实 Gateway 按 `gateway_config` 关联纠正库存类型，不按 Traefik 名称判断。Gateway 部署会按保存的 ACME profile 重新选择绑定版本，普通服务部署弹窗的自由版本选择不符合该语义；Gateway 入口提供交互一致的目录控件。

现状依据：

- [CD 产品模型](../product/cd-model.md)、[CD 运行时](../architecture/cd-runtime.md)、[工作目录与挂载](../guides/volume-mounting.md)。
- `web/src/views/service/ServicePage.vue`、`web/src/views/service/ServiceDetail.vue`：部署弹窗与版本选择。
- `internal/application/deployment/usecase/deployment_execution.go`：目录解析、渲染、文件准备与执行。
- `internal/infrastructure/runner/local/runtime.go`、`internal/infrastructure/runner/ssh/runtime.go`：目标运行时。
- `internal/infrastructure/storage/local/physical_data_root.go`：Orbit 容器路径到 Docker daemon 路径的映射。
- [决策账本 C-04](../decisions/ledger.md)、`internal/application/gateway/usecase/service.go`、`deployment.go`：Gateway 身份与 profile 版本选择。
- `web/src/views/gateway/GatewayDetail.vue`：Gateway 专属部署入口；目前仍调用共享服务部署 API。

## Goal

1. 普通服务列表和服务详情的部署弹窗同时提供版本与部署目录；根据当前 Environment 的 `workspace_root` 和服务编码拼出完整目录，用户可以直接确认或修改。
2. 确认的目录是服务完整部署根目录，包含 `docker-compose.yml`、相对挂载的数据目录和受控配置文件；用户输入的完整目录不再额外追加 `deployment/<service-code>`。
3. 确认后的目录可持久化并用于后续操作。本次部署使用的版本、目录及环境身份和修订具有明确的任务快照，后续编辑不能使排队任务落到另一目录或目标。
4. 部署、重启、停止、运行状态、容器日志和运行时工具统一使用相应服务的运行目录；Compose project 继续由服务编码标识。
5. 沿用本地、Linux SSH、Windows SSH 的执行方式和路径语义，并复用 DooD 的逻辑路径与物理挂载源映射。
6. 允许用户选择 `Environment.workspace_root` 之外的有效部署目录；文件操作范围由明确的服务目录及其归属约束。
7. Gateway 创建时正确写入 `kind=gateway`，已有真实 Gateway 的类型通过新增迁移纠正。服务列表和服务详情依据所属应用类型禁用其部署按钮；Gateway 页面提供与普通服务一致的路径控件，并保留其 profile、受管配置与就绪检查。
8. 已部署服务修改目录时显示风险警告，告知相对挂载位置变化、旧数据不自动迁移及可能的重建和中断；不因目录变化要求先停止、不增加风险确认步骤或阻止提交。

## Non-goal

- 不修改 Environment 工作区根目录来实现单个服务的目录选择；CI 的 `pipeline` 目录及其他服务目录不因此变化。
- 不自动迁移、复制或删除已有持久化数据和旧部署目录。
- 不改变 local/SSH 的目标分派、SSH 用户、受管凭据、host-key pinning 或 Docker 外部前置条件。
- 不把 Windows SSH 部署改为在 WSL 内执行 Compose，不增加其他 SSH 平台或 Docker 运行组合。
- 不引入新旧目录规则并行的兼容执行分支；已有记录的处理方式在计划阶段明确，不修改已执行迁移文件。
- 不依靠应用名称、编码或镜像名称推断 Gateway 身份，不将没有 Gateway 关联的普通 Traefik 应用改为 Gateway 类型。
- 不要求已部署服务先停止才能修改目录，不增加额外风险确认勾选或二次确认弹窗。

## User scenarios

1. 用户在本地服务的部署弹窗选择版本，看到根据当前环境自动拼好的完整目录，确认后使用该目录部署。
2. 用户修改 Linux SSH 服务的目录，例如 `/srv/services/api`；Orbit 通过 SFTP 准备文件，在同一远端目录执行 Compose。`~/...` 使用远端 SSH 登录用户的主目录。
3. 用户修改 Windows SSH 服务的目录，例如 `D:/services/api`；文件写入 Windows 宿主机，非交互 PowerShell 在该目录调用 `docker.exe`，保留现有空格和引号处理。
4. Orbit 容器挂载 `/srv/orbit:/app/data` 时，用户确认 `/app/data/deployment/api`；Orbit 在该容器路径写文件，Compose 的相对挂载源使用对应宿主机路径 `/srv/orbit/deployment/api/...`。
5. 用户再次部署、重启、停止或查看服务状态和容器日志时，操作使用已确认的目录。环境目标或工作区变化后，目录重新校验并由用户确认。
6. 用户尝试使用不可写、被其他服务占用或在 DooD 下无法映射的目录时，收到明确错误，不会在错误目标或挂载源上启动容器。
7. 用户选择环境工作区之外的有效目录，按该目录部署；文件操作仍只处理此服务受管范围。
8. 用户在服务列表或服务详情查看 Gateway Service 时，响应正确反映 `application_kind=gateway`，部署按钮禁用并可获知应从 Gateway 入口部署。未配置为网关的普通 Traefik 应用仍按 `standard` 处理。
9. 用户在 Gateway 部署弹窗确认或修改目录，控件、校验及警告与普通服务一致；Gateway 版本继续由已保存的 ACME profile 选择。
10. 已部署服务或 Gateway 的目录改变时，弹窗即时显示风险警告；用户仍可直接提交，系统不自动迁移数据或先停止旧容器。

## Acceptance

- [ ] 普通服务列表与服务详情的部署弹窗均提供根据当前环境拼接的完整目录，允许用户确认和修改，并保持原有版本选择能力。
- [ ] 未修改目录时沿用 `<Environment.workspace_root>/deployment/<service-code>` 的布局；修改后实际 Compose 工作目录与用户确认的完整目录一致。
- [ ] 确认后的目录被保存；后续部署和生命周期操作使用相应目录。已有目录与新环境配置不一致时，不能静默沿用失效路径。
- [ ] 每次 Deployment 明确记录本次版本和目录以及目标身份与修订；排队后的配置变更不会导致任务执行到其他目录或目标。
- [ ] 相对挂载和受控文件以确认的服务目录解析；绝对 host-path 挂载、named volume 和独立的控制面部署日志保持各自语义。
- [ ] 本地原生、Linux SSH、Windows SSH 和本地 DooD 使用各自正确的路径解析、文件准备和 Compose 执行方式；SSH 不经过本地 Docker 路径映射。
- [ ] DooD 的配置文件写入 Orbit 可见路径，Compose 挂载源使用 Docker daemon 可见路径；无法映射时在启动容器前失败。
- [ ] 重启、停止、运行状态、容器日志和运行时工具使用同一服务运行目录；Compose project 继续使用服务编码。
- [ ] 路径合法性、写权限和目录占用有明确校验和反馈；部署失败或取消后仍能定位本次实际执行目录以查询、停止和清理运行资源。
- [ ] 其他服务的运行容器已使用同一物理数据目录或其父子目录时，阻止本次部署；以目标 Docker 的实际 bind mount 为准，覆盖独立嵌套挂载，不按待部署版本或库存状态推测占用。
- [ ] 目录变更不会自动搬迁或删除数据，不影响 CI 工作区、其他服务或部署执行日志目录。
- [ ] 有效目录可以位于环境工作区之外；Gateway 的路由、证书、ACME 与发布记录操作沿用该 Gateway Service 的受管目录边界。
- [ ] 新 Gateway Application 使用 `kind=gateway`；已有真实 Gateway 按 `gateway_config` 关联通过新增迁移纠正，应用与服务响应一致反映正确类型。普通 Traefik 应用不被误改。
- [ ] 服务列表的卡片与表格，以及服务详情，均禁用 Gateway 的部署按钮，并提供 Gateway 部署入口的说明。
- [ ] Gateway 专属部署弹窗提供一致的目录控件，保留按保存的 ACME profile 选择版本、受管配置渲染、强制重建和 API 就绪检查。
- [ ] 已部署服务及 Gateway 修改目录时显示风险警告；恢复原目录后警告消失。警告不禁用提交，不要求停服、额外确认勾选或二次确认。

## Open questions

暂无需要用户确认的未决事项。具体接口、目录切换时点和旧库存处理方式由 Plan 明确。

## Decisions

- 用户指定标准模式 / standard。本任务新建独立 Intent，不修改此前以服务编码组织目录的任务文档。
- 用户明确：目录应根据环境配置拼接好，用户可以确认和修改。
- 用户明确要求同时考虑本地和远程 SSH 的 Windows/Linux 实现，以及 Orbit 自身容器化时的 DooD。
- 本地 DooD 的目录采用 Orbit 容器内可见路径语义，Docker 挂载源由既有 resolver 转换；SSH 的目录采用远端宿主机路径语义。
- “目录”是完整部署根目录；用户允许使用环境工作区之外的目录。
- Gateway 作为特殊服务从 Gateway 入口部署，该入口加入一致的目录控件；服务入口禁用 Gateway 部署。
- 用户明确纠正现行类型设计：Gateway Application 使用 `kind=gateway`，服务入口直接依据 `application_kind` 判断，不新增 `is_gateway` 识别字段。GatewayConfig 和 Environment 绑定继续承担网关配置及目标归属校验。
- 纠正 Gateway 创建逻辑，并按真实 Gateway 关联迁移已有类型；实现时同步修订 C-04 及活文档。仅名为 Traefik 的普通应用不自动成为 Gateway。
- 用户要求已部署服务修改目录只显示风险警告，不阻止提交、不强制先停止。Gateway 使用相同交互。
- 字段结构、接口是否合并、保存目录与实际运行目录的切换时点，以及已有库存处理方式留到 Plan 评估。
- 用户要求开始 Plan，当前 Intent 标记为 Accepted。
- 验证发现嵌套 DooD 挂载占用漏检后，用户明确要求：如果已有运行中的服务使用该物理数据目录，校验并阻止部署。原有根目录归属保护继续保留。
- 用户确认集成测试无问题，并明确要求完成 SpecFlow 验收；运行物理目录占用和 Windows 根目录校验的修正经单元复核后纳入交付。

## Risk

- 修改目录会改变所有 `./...` 挂载的持久化位置，而 Compose project 不变；直接部署可能重建同一组容器并挂载空目录。按用户决定提示风险但允许继续，不自动停止或迁移。
- 若在任务开始前提前切换服务运行目录，前置失败可能使后续停止或查询找不到旧资源；若仅记录最近成功目录，执行中失败或取消又可能遗失新目录。需要明确目录切换和失败恢复语义。
- DooD 的路径必须存在有效宿主机挂载映射；嵌套挂载可能使服务根目录与组件子目录对应不同宿主机位置，需要在计划中检查当前根目录映射后拼子路径的适用边界。
- 远端路径不能直接使用 Orbit 所在操作系统的路径规则；Windows 盘符、分隔符、大小写、空格与主目录展开应沿用目标平台语义。
- 当前部分文件读写限制在 Environment 工作区内。允许工作区外目录后需将边界收敛到已确认的 Service 目录及其归属，不能简单删除检查。
- Gateway 改目录后旧路由、手工证书与 ACME 数据不会自动进入新目录；警告需说明可能的路由中断，Route 仍由用户另行显式同步，部署不自动发布业务草稿。
- 环境 Probe 成功不代表任意自定义目录可写或能被 Docker 挂载；需要针对本次目录检查。
- 目前仅完成代码与活文档评估，尚未实现或在真实目标上验证；验证范围应与风险匹配，并遵守用户无需集成测试的约定。

## User review notes

- 用户要求了解持续部署中服务部署的逻辑，并评估部署时除版本外指定目录。
- 用户提醒注意本地与远程 SSH 的 Windows/Linux 实现。
- 用户追加要求关注 Orbit 运行在容器中的 DooD。
- 用户澄清：“这个地方要根据环境配置拼接好，用户可以确认和修改”。
- 用户要求通过 `specflow:specflow` 标准模式记录任务。
- 用户允许环境工作区之外的部署目录。
- 用户指出 Gateway 是特殊服务，要求排查 Traefik Application 为 `standard` 的原因，并在服务入口禁用其部署按钮；Gateway 入口加入交互一致的路径控件。
- 用户要求已部署服务改目录时给出风险警告，不阻止部署。
- 用户明确要求开始计划阶段，当前 Intent 标记为 Accepted。
- 用户进一步明确要求纠正 Gateway 的 `kind=standard`：应使用 `kind=gateway`，使服务入口可通过应用类型禁用 Gateway 部署。
