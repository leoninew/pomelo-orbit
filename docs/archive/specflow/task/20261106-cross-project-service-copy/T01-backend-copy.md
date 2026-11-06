# Task / 子任务 T01：后端跨项目复制能力
最后修改时间: 2026-11-06 22:23:46

## Review status / 审查状态

Accepted

局部方案修订 2 及 T01-002 产品交付均已由主会话审查接受。

## Parent and authority / 父任务与权威记录

- 总意图：[跨项目复制服务](../../intent/20261106-cross-project-service-copy.md)。
- 权威总计划：[计划](../../plan/20261106-cross-project-service-copy.md)；任务 ID：T01；方案修订：2。
- 权威协调根：`D:/SourceCodes/mywork/pomelo-orbit`；主会话 `/root` 唯一登记分配与审查。

## Goal and acceptance / 目标与验收

- 交付原样复制应用、所选范围版本、组件、环境值及服务覆盖的用例，目标引用独立、服务未部署；最新模式只复制并绑定最新版本，全量模式复制全部版本并绑定源服务当前 VersionId 的映射。
- 交付唯一 HTTP 复制入口和两端生成类型，完整覆盖权限、冲突、原子失败边界；对应 A02–A07、A08 后端部分。
- 下游使用的请求与响应固定为总计划中的 `ServiceCopyReq` / `ServiceCopyResp`，响应只有新服务 ID，不返回配置。

## Scope and files / 范围与文件

- `internal/application/application/usecase/copy.go`、现有 `dto/application.go`：应用复制及 `ApplicationCopyResult`。
- `internal/application/service/usecase/copy.go`：`CopyService`、局部应用复制接口和服务复制编排。
- `internal/api/http/handler/service/copy.go`、`internal/api/http/routes/service.go`、`routes.go`：专用薄 handler、依赖及路由。
- `internal/bootstrap/http.go`：在现有 application/service 实例之后构造 CopyService，注入 HTTP Dependencies。
- `proto/orbit/v1/service/service.proto`、`internal/gen/proto/orbit/v1/service/service.pb.go`、`web/src/gen/proto/orbit/v1/service/service.ts`。
- 上述领域对应的必要 `*_test.go`，以及已有 HTTP 事务装配测试。生成物由本任务统一维护；不改前端页面/API，不加 MCP/CLI，不改迁移。

## Inputs / 输入

- 无直接任务前置。当前调查基线 `557763af`，不是执行基线；派发前核对计划接受状态、实施授权和工作树。
- 依赖总计划固定的复制字段、冲突语义、接口及 HTTP UoW 约束；使用项目后端架构技能规定的职责与测试边界。

## Technical decisions / 局部技术设计

### 应用复制

1. `CopyApplication` 从源项目读取完整定义，检查目标成员权限及应用名称、编码唯一性。两种冲突在此返回 `KindConflict`，不改变普通应用创建的既有错误行为。
2. 复制 Application 结构体并更新 ID、ProjectId 和时间元数据，调用现有仓库创建；不调用会创建默认版本的交互用例。
3. 根据 `copyAllVersions` 选择完整版本列表或 ID 降序首项，为所选版本及其组件建立新 ID 映射。全量模式按父版本优先写入并映射集合内 `CreatedFromVersionId`；最新模式仅复制最新项，若父版本未纳入则清空其来源引用，避免跨项目指向。保留 Label、Status、Note、ComponentSummary。Service.VersionId 在最新模式映射到所复制的最新项，在全量模式映射到源服务当前 VersionId；不能根据 Label 猜测。
4. 版本组件声明和产物来源快照原样传给现有 `CreateVersionWithVersionComponents`。不重新解析命令、不做规范化或有效配置展开，不复制外部产物实体。
5. 返回新应用及版本/组件映射，映射完整性来自本用例创建过程，不为内部返回值重复校验、补默认值或猜测匹配关系。

最新模式按组件名称映射源服务覆盖；若源服务组件集合与最新版本不一致，返回冲突并提示选择全部版本，避免丢弃覆盖或保留无效引用。全量模式直接映射源版本组件 ID。

### 服务复制与接口

1. `NewCopyService(service Service, application applicationCopier, environment copyEnvironmentReader)` 注入已有服务、应用复制和环境元数据读取能力；`Copy` 校验外部标识必填及跨项目条件。
2. 经 `ServiceDefinitionForUser` 读取源服务、环境值和组件覆盖。通过 EnvironmentForUser 核对目标权限及环境修订，按目标平台校验目录；应用复制后检查服务编码，已有编码返回 `KindConflict`，读取错误按既有分类传播。
3. 调用应用复制取得映射。新 Service 携带目标归属、映射后应用/版本、原 Code、用户确认的目标 DeploymentDirectory、`stopped` 状态；部署 ID、实际运行目录、源环境目标修订、历史时间不带入副本。
4. 逐个复制 ServiceComponent 并更新 ID、ServiceId、SourceVersionComponentId 与时间元数据，保持覆盖内容和存在性不变。调用 `CreateServiceWithComponents` 与 `ReplaceServiceEnv` 写入；使用现有仓库生成挂载、端点等子行 ID。创建后以已有 `UpdateServiceDeploymentDirectory` 保存确认路径及目标修订；外部目录输入复用现有目标平台路径校验。
5. 返回新 Service 供 HTTP 提取 ID，不构建含环境值的 ServiceView，不触发 effective plan、目标解析或任何 runtime 操作。
6. `POST /api/service/:service_id/copy` 的源 project_id 来自 query，目标 ID、`copy_all_versions`、`deployment_directory`、`environment_target_revision` 来自 JSON；专用 `CopyHandler` 使用窄用例接口，沿用认证及 `transport.WriteError`，成功写 `201 ServiceCopyResp`。
7. 注册在写事务中间件之后，不增加事务绕过路径。调用方通过 context 提供外层 UoW；不在用例中写补偿删除或引入第二套事务端口。

## Implementation outline / 实施安排

1. 增加应用复制结果与用例，直接验证完整配置、多个版本和父子关系的保留。
2. 增加服务复制编排，验证覆盖存在性、权限、冲突和未部署元数据。
3. 增加 Proto 请求/响应，运行 `task proto`；只保留本次契约的生成 diff，不手改生成物，不升级工具。
4. 增加 handler、路由和 bootstrap 装配，验证请求、响应及事务覆盖。
5. 用临时持久化验证成功回读和中途失败回滚，完成项目要求的后端与静态检查，写交付证据。

## Checks and risks / 检查与风险

- 纯用例：使用现有 Repository/应用端口替身执行真实业务；以一个源服务绑定非最新版本的样本分别验证默认最新与全量模式的版本集合、服务目标版本、lineage、文件内容、Service 环境值及显式覆盖，不逐字段穷举理论输入。
- 真实失败：服务编码、应用名称、应用编码冲突，以及源/目标权限拒绝；断言拒绝复制，不用测试替身重演业务校验。
- 集成：临时 SQLite 使用当前迁移和真实 SQLC Repository，通过既有事务 runner 执行复制；在应用已创建后注入服务环境写入失败，确认目标新资源全数回滚、源和预先存在的目标资源不变。该测试按 `!unit` 分类，不称为纯单测。
- HTTP：局部 handler + 用例替身验证 query/body、`201 {id}` 与错误传播；在现有事务装配测试加入复制路由，复用提交失败与缓冲响应机制的已有证据。
- 命令：针对相关包运行必要测试，交付前运行 `task check`、`go test ./cmd/... ./internal/...`；不得启动服务或执行实际部署。失败、未运行项及修复后的结果均记录，测试输出不得打印环境值。
- 风险重点：误走 normalizeOverlay、版本父子顺序、配置切片被修改、部署目录配置遗漏、源环境目录确认修订或运行状态被复制、成功响应或日志泄露配置。

## Handoff basis / 交接依据

协议入口为已读取的 SpecFlow `SKILL.md` 与 `references/task-coordination.md`。

- 分配与授权：总计划“协调信息”“执行登记表”和授权记录；2026-11-06 用户要求开始实现，总计划及 task 方案 Accepted；任务范围为 T01 和必要检查。
- 工作区与基线：`D:/SourceCodes/mywork/pomelo-orbit`，分支 `develop`，基线 `557763af3bea5f4570c64a95978b0c5f259f9e00`；基线时仅有本功能的已记录 SpecFlow 文档未提交。
- 可写回报位置：本文“接收与进展”“交付与审查”；主会话读取此工作区文件。
- 回报渠道：当前主会话直接更新总计划登记并输出事实摘要。


## Reception and progress / 接收与进展

接收：2026-11-06 20:51:00，执行者 `/root`，编号 T01-001，方案修订 1。核对主计划及 task Accepted、用户实施授权、T01 无前置任务；基线为 `557763af3bea5f4570c64a95978b0c5f259f9e00`。开始后端实现。

| 时间 | 分配编号 / 执行者 | 接收、进展或阻塞 | 证据 / 回报位置 |
| --- | --- | --- | --- |
| 2026-11-06 20:25:27 | 无 / 无 | 局部方案建立 | 本文与总计划修订 1 |
| 2026-11-06 20:51:00 | T01-001 / `/root` | 核对 Accepted 方案与基线后接收 T01，开始实施 | 本文及总计划执行登记 |

## Delivery and review / 交付与审查

交付：2026-11-06 21:17:29，T01-001 / `/root`，当前工作树（基线 `557763af3bea5f4570c64a95978b0c5f259f9e00` 之上的后端 diff）。

- 已实现应用与服务复制、版本/组件映射、最新及全量范围、原样环境值与覆盖、停止状态及目录配置保留。
- 已交付 `POST /api/service/:service_id/copy`、`ServiceCopyReq` / `ServiceCopyResp`，运行 `task proto` 生成两端类型；成功响应只有 ID，写入使用现有 HTTP UoW。
- A02/A03/A06/A07：用例及 SQLite 集成测试验证独立引用、版本顺序与 lineage、占位引用和文件内容、显式空覆盖与 Service 环境值保留、源配置不变及未部署状态。最新组件集合不一致拒绝并提示全量。
- A04/A05：真实持久化验证服务编码及应用名称/编码冲突、源/目标成员权限拒绝；注入环境值写入失败后，新应用、版本、组件、服务整体回滚，已有目标服务不变。
- A08 后端：相关包定向测试、`go test -count=1 ./cmd/... ./internal/...`、Go 格式检查及 golangci-lint 均通过（0 issues）。HTTP 局部测试验证默认范围、全量参数、201 ID 响应、冲突、错误 JSON 及认证；既有事务装配测试包含复制路由。初次新增 HTTP 测试因 jwt 包别名错误编译失败，修正后通过。
- 未运行实际部署或外部资源操作；没有新增迁移或 MCP/CLI 入口。T02 完成后的全仓 `task check` 与 `task test` 均已通过。

主会话审查：Accepted。已核对复制路径未调用 normalizeOverlay、应用与服务领域写入边界、同一 context 事务、生成物及上述检查。放行 T02 使用当前工作树接口及 TypeScript 类型。

## Revision 2 / 目录调整

2026-11-06 21:37:42：用户授权按目标环境计算并编辑复制目录。局部方案 Accepted，具体契约与检查以总计划修订 2 为准；上轮交付的目录行为已被替代。当前主会话接收 T01-002，开始后端调整。

2026-11-06 21:42:50：T01-002 后端交付已接受：新增目录及目标修订输入，复用环境元数据读取与目录校验，保存目标目录/修订；原事务回滚及复制配置测试继续通过。

修订 2 最终审查：2026-11-06 22:03:56，T01-002 Accepted。外部目录输入的校验及跨应用编排在 usecase，handler 只映射 DTO；环境依赖只读取元数据，未引入 SSH/Docker 操作。单测验证目标平台和修订拒绝边界，既有 SQLite 集成测试验证 Windows 源到 Linux 目标的编辑目录保存、源配置不变及失败回滚。task check、task test 和同产品代码的完整 Go 测试通过。

## Overall verification / 整体复核

2026-11-06 22:23:46：主会话按用户验证授权复核 T01-002，维持 Accepted；本轮无产品修复、分配或方案变化。最终交付为基线 `557763af` 之上的当前工作树后端及生成物，产品 diff 指纹见[验证记录](../../verification/20261106-cross-project-service-copy.md)。

已核对应用与服务写入职责、目标权限和目录修订、版本及组件映射、原始配置与覆盖保留、停止状态及 HTTP UoW；`task check`（0 issues）、`task test` 和 `go test -count=1 ./cmd/... ./internal/... ./sql` 均通过。SQLite 回读与失败回滚属于本地集成验证；未操作已有数据库或运行容器。
