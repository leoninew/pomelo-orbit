# 服务部署目录确认与修改验收
最后修改时间: 2026-10-04 01:22:01

Review status: Accepted

Mode: standard

## Basis and scope

- 依据 [Accepted Intent](../intent/20261003-service-deployment-directory.md) 与 [Accepted Plan](../plan/20261003-service-deployment-directory.md)，以及用户要求“开始验证，不跑集成测试”、运行物理目录占用时阻止部署，并最终确认“集成测试无问题，完成 SpecFlow 验收”。
- 当前为 Verification / 验证；标准模式不另建 Spec。
- 初轮核对相对 HEAD 的全部 122 个实现文件；继续核对用户的弹窗简化、警告文案/样式调整及本阶段 F-01/F-02 修正。本阶段保留用户改动和暂存区，没有执行暂存、提交或推送。
- 本代理只运行静态检查、显式筛选的 Go 单元测试和前端组件测试；没有运行数据库迁移测试、HTTP/SSH 集成测试、E2E 或浏览器流程。Go 测试文件参与编译不代表其测试被执行。集成测试由用户完成并确认无问题。
- 两个临时单元诊断复现了下文的问题；诊断文件已移除，失败结果作为本轮证据保留在本文。
- 未启动开发服务器。本任务没有由代理启动的服务器需要关闭，用户已有服务器未操作。

## Intent alignment

普通服务和 Gateway 共用完整部署目录字段；首次根据 Environment 的工作区与服务编码拼接，当前修订下回填已确认目录。普通服务将版本、目录、目标修订放进一次部署请求。Gateway 按所属应用类型识别，保留 profile 版本选择与特殊部署流程。

Service 的确认目录与运行目录分开，Deployment 保存目录快照。Worker 在文件准备完成后、发出 Compose 命令之前绑定运行目录；准备失败保留旧运行位置，命令失败或取消保留本次已绑定位置。生命周期、状态、日志与运行时工具使用运行目录。Compose project 仍由服务编码确定。

local、Linux SSH、Windows SSH 使用各自目标路径语义；DooD 逐项映射相对挂载源，未映射路径拒绝。工作区外目录有显式文件范围，Gateway 发布也使用其服务运行目录。目录变化只警告，不要求停服，不自动移动数据。

主要产品行为与意图对齐。初轮发现的两项目录校验和占用保护遗漏已修复；运行数据占用依据 Docker 的实际运行容器及 bind mount，不依赖可变的待部署版本或库存状态。

## Spec alignment

不适用：standard 模式依据 Intent 与 Plan 核对。

## Plan alignment

| 计划步骤 | 实际核对 |
| --- | --- |
| Gateway 类型与目录持久化 | 创建逻辑写入 gateway；新 000050 迁移按 gateway_config 真实关联修正类型；模型、SQLC、Repository、Proto 提供确认/运行目录及修订 |
| 一次部署请求与快照 | HTTP/MCP 输入扩展；版本切换复用窄端口；目录和目标修订进入请求、任务快照与配置 hash |
| 显式运行位置 | Runtime 使用 ServiceLocation；local/SSH/target、日志、查询和工具调用已适配 |
| 失败与取消处理 | 准备前后目录绑定有用例覆盖，取消协调重新读取实际绑定；不自动 down、迁移或回退目录 |
| Gateway 发布目录 | 文件范围、目录依赖、容器挂载核对及 publication hash 纳入运行位置；目录变化使旧 preview 过期 |
| 共享控件与全部入口 | Service 列表卡片/表格、详情按 application_kind 禁用 Gateway 部署；Gateway 复用路径控件；停止过的服务仍显示风险警告且可提交 |
| 活文档与检查 | 产品模型、运行时、决策、部署与挂载指南已更新；最终固定检查及筛选测试通过，两项初轮诊断缺陷已修复 |

实际物理目录占用保护已补齐：本次部署根目录及相对挂载源与其他服务运行容器的实际 bind source 比较，覆盖嵌套挂载与父子目录。Docker 查询或解析失败时阻止文件准备，本服务自身的挂载不阻止再次部署。停止容器不构成运行数据占用；原有根目录归属保护继续适用。

## Actual diff summary

进入验证时，相对 HEAD 的实现 diff 为 122 个文件、3,106 行新增、533 行删除，不含本验收文档。122 个文件均已由用户暂存，本阶段未改变该暂存状态。收尾时用户删除两个停止弹窗中的服务名称行及不再使用的 computed，实际工作区 diff 变为 3,106 行新增、536 行删除；这些改动已核对并补跑前端检查。

上述数字为初轮验证快照。后续增量包括用户简化部署弹窗说明、调整普通服务风险文案并使用共享 app-field-warning，以及本阶段修改 directory_ownership.go、service_directory.go/service_directory_test.go 和新增 directory_ownership_test.go。文档同步更新 Intent、Plan、挂载指南及本文；未改变用户暂存区。

- 持久化：新增 SQLite/MySQL/PostgreSQL 000050 迁移、Service/Deployment 目录字段、查询与 Repository 映射。
- 后端：请求版本选择、任务快照、运行目录绑定、local/SSH 路径与文件边界、DooD 逐项映射、Gateway 类型及发布位置。
- 前端：共享目录组件/composable、普通服务与 Gateway 部署弹窗、Gateway 部署入口禁用及中英文风险文案。
- 协议：Proto/SQLC 的 Go 与 TypeScript 生成代码；共享 Service 模型引起多包生成物变化。
- 文档：Intent/Plan、CD 产品模型、运行时、C-04 决策及部署/挂载指南。

| 预期与实际文件差异 | 原因与结论 |
| --- | --- |
| physical_data_root.go 未修改 | 复用既有 Docker 路径 resolver；逐项挂载映射新增在部署渲染调用中 |
| GatewayConfig 未新增目录数据库字段 | 由受管 Service 投影目录；运行时内部字段不序列化、不重复持久化 |
| sqlc.yaml 与 sql/schema/mysql.sql 修改 | 声明新可空目录类型并维护生成依据，属于持久化变更 |
| 多个 SQLC 包 models.go 变化 | 共享 schema 生成结果，不是额外业务范围 |
| route_restart_e2e_test.go、windows_ssh_e2e_test.go、factory_integration_test.go、session_test.go 修改 | 适配新接口；本轮没有执行其中的测试 |
| project_scope_test.go 多一处 return | Implementation 为消除 staticcheck SA5011 补齐 t.Fatal 后控制流，属极小附带修正 |
| 先前文本截断与列表列宽任务 | 当前 diff 基线已有相关共享组件；本次验收聚焦部署目录，不重新验收此前界面任务 |
| 本阶段新增文件 | 本 Verification 文档及稳定的 directory_ownership_test.go；临时诊断文件已移除 |
| 验证期间的用户调整 | ServicePage/ServiceDetail 删除停止弹窗服务名称行及无用 computed，保留未暂存状态；部署目录逻辑未变化 |
| 验收收尾的用户调整 | 部署弹窗删除重复说明、普通服务风险文案缩短、共享警告样式；保留用户改动并补跑组件测试 |
| Verification 修正 | 运行容器实际挂载占用检查、Windows 根目录拒绝及稳定回归，均属于原定目录保护范围 |

## Acceptance checklist

“通过”仅指下述单元证据或明确注明的代码核对，不等同于真实环境部署验收。

| Intent 验收项 | 结果与依据 |
| --- | --- |
| 普通列表/详情同时选择版本与目录 | 通过：组件测试实际提交版本、编辑后的目录及环境修订，没有预先 PUT 版本 |
| 默认布局与完整路径覆盖 | 通过：helper 测试默认值和匹配修订回填；用例测试工作区外目录冻结与执行 |
| 确认路径持久化、环境变化重新确认 | 代码核对及用例/helper 测试通过；失效修订拒绝，Project 变化和旧异步响应清理有覆盖 |
| Deployment 记录版本/目录/目标、排队后不换位置 | 通过：快照测试随后改变服务版本和目录，Worker 仍使用本次快照 |
| 相对挂载、受控文件、绝对 host-path、named volume | 代码核对通过；逐项相对源映射有测试，显式 host-path/named volume 分支保留，部署日志位置独立 |
| local/Linux SSH/Windows SSH 目标路径与命令 | 单元与代码核对通过；F-02 已修复并补充盘符根目录回归；集成验收依据用户确认 |
| DooD 逻辑文件与物理挂载源、未映射拒绝 | 映射正向及拒绝未映射路径单元通过；F-01 已修复，实际运行挂载冲突及 Docker Desktop 路径比较回归通过 |
| 生命周期、状态、日志、工具使用运行目录 | 代码核对通过；重启冻结运行目录及日志目标变更测试通过，Compose project 继续使用 service code |
| 路径、权限、占用与失败后定位 | 单元通过：归属/符号链接、canonical scope、失败/取消绑定、运行物理源重叠阻止及检查失败阻止文件准备；F-01/F-02 已解决 |
| 不自动搬迁/删除，不影响 CI 与日志目录 | 代码核对通过，无迁移数据、清理旧目录或修改 CI 目录的新流程 |
| 工作区外目录及 Gateway 文件边界 | 单元与代码核对通过：显式 FileScope；Gateway 自定义运行位置和 session 目录变化测试通过 |
| Gateway kind=gateway，真实关联迁移、普通 Traefik 保留 standard | 创建/三种迁移 SQL/查询投影代码核对通过；普通 Traefik 服务组件测试通过；本轮未执行数据库迁移测试 |
| 卡片、表格、详情均禁用 Gateway 部署 | 通过：三种可见入口组件测试，处理器也检查类型，共享 Tooltip 提供入口说明 |
| Gateway 一致路径控件，profile/重建/就绪保留 | 组件与 fake-runtime 用例通过；专属提交不自由选择版本，强制重建和就绪依赖保留 |
| 已部署含 stopped 修改路径只警告、恢复撤警告 | 通过：字段与页面组件测试；警告不进入提交禁用条件，恢复规范化原目录后消失 |

## Test results

本代理执行目录均为仓库根目录，没有运行集成测试。用户最终确认集成测试无问题；未提供具体命令或平台明细，本文不推断逐项平台覆盖。

| 检查 | 结果 |
| --- | --- |
| task check | 通过：Vue 类型、ESLint、Prettier、Go lint 配置/格式/lint，Go 为 0 issues |
| 收尾 yarn --cwd web typecheck / lint / format | 用户停止弹窗调整后通过；使用只读入口，无自动修复 |
| 4 个针对性 Vitest 文件 | 通过：14 项；服务入口、Gateway 部署提交、目录 helper 与警告组件 |
| Go 显式 allowlist | 最终通过：51 项顶层单元测试，8 个包，无失败/跳过；count=1，未使用缓存 |
| 初轮 TestVerificationRejectsSharedNestedDooDMount | 当时失败，证实 F-01；稳定占用回归在修正前复现，修正后通过 |
| 初轮 TestVerificationRejectsWindowsDriveRoot | 当时失败，证实 F-02；现在 TestServiceDirectoriesUseTargetPlatform 包含对应回归且通过 |
| 运行挂载占用回归 | 新增 6 项顶层单元，覆盖相同/父子源、多容器、停止后释放、named volume、本服务重部署、local/Linux/Windows/DooD 路径、查询失败及阻止文件准备 |
| git diff HEAD --check | 通过 |
| task sqlc / task proto | Implementation 已运行成功；本轮源定义未改，未重复生成 |
| lint:fix / typecheck / 完整 Go 测试入口 | Implementation 有通过记录；完整 Go 结果属于此前证据，本轮按用户限制改为单元 allowlist，不将此前集成结果计为本轮执行 |
| 数据库迁移、HTTP/SSH 集成、E2E、Docker/浏览器 | 本代理未执行；用户确认集成测试无问题 |

前端实际命令：

```sh
yarn --cwd web test src/views/service/serviceDeployment.test.ts src/components/DeploymentDirectoryField.test.ts src/composables/useDeploymentDirectory.test.ts src/views/gateway/gatewayNavigation.test.ts
```

最终 Go 实际命令（JSON 输出保存在本机 /tmp/orbit-running-directory-ownership-unit.jsonl）：

```sh
TMPDIR=/private/tmp go test ./cmd/... ./internal/... -count=1 -json -run '^(TestDeploymentRequestFreezesVersionDirectoryAndRevision|TestDeploymentRejectsStaleEnvironmentBeforeChangingConfiguration|TestRestartFreezesRuntimeDirectoryInsteadOfPendingDirectory|TestCanceledDeploymentKeepsTheDirectoryBoundAtCancellation|TestDeploymentDirectoryBindingFollowsPreparation|TestComposeMapsEachRelativeBindSource|TestDirectoryOwnershipChecksNestedDockerMountSources|TestServiceDirectoriesUseTargetPlatform|TestRuntimeStagesLocalWorkspaceAndResolvesDockerPath|TestRuntimeExpandsHomeWorkspaceRootAtUseTime|TestRuntimeSyncFilesReplacesSnapshotAndRemovesStagedFiles|TestRuntimeRejectsNonLocalTargetsAndMissingResolver|TestCustomWorkspaceOwnershipProtectsExistingCompose|TestCustomWorkspaceRejectsSymlinkEscapes|TestCommitWorkspaceFileReplacesExistingWindowsFile|TestCommitWorkspaceFileRestoresWindowsFileWhenReplacementFails|TestCommitWorkspaceFileKeepsAtomicRenameOnLinux|TestNormalizeSFTPHomePathForWindowsAndLinux|TestServiceDirUsesTargetPlatformPaths|TestRemoteCanonicalPathsKeepFileOperationsInServiceScope|TestSFTPPathResolverExpandsRemoteHomeForMountSource|TestRemoteCommandUsesWindowsDockerDesktopCLI|TestRemoteCommandQuotesLinuxArguments|TestRemoteCommandResolvesSSHHomeWorkspace|TestRemoteCommandUsesWindowsCurlExecutable|TestRemoteQueryDiagnosticKeepsSuccessfulStdoutClean|TestPowerShellEncodedCommandUsesUTF16LE|TestRuntimeDispatchesOnlyByExplicitTargetType|TestResolveDockerDaemonPathLeavesNativePathUnchanged|TestResolveDockerDaemonPathRequiresAbsoluteOrbitPathInContainer|TestResolveMountedContainerPathUsesLongestDestination|TestResolveMountedContainerPathRejectsUnmappedPath|TestGatewayPublicationTracksCustomRuntimeDirectory|TestHistoricalTargetRevisionAllowsExplicitRepublish|TestGatewayForDeploymentDoesNotRequireGatewayForInternalTCPEndpoint|TestGatewayForDeploymentFindsCarrierEvenWhenNetworkWasRequestedDisabled|TestGatewayForDeploymentSkipsGatewayConfigWhenTraefikNetworkDisabled|TestSelectGatewayVersionForDeploymentUsesRequiredCoordinator|TestGatewayDeploymentOverwritesStaticConfigAndRecreatesContainer|TestDeploymentAndRestartDoNotRequireRoutePublication|TestGatewayDeploymentAndRestartStillRequireGatewayReadiness|TestApplicationStatusReturnsNoContainersBeforeFirstDeployment|TestApplicationLogStreamWaitsBeforeFirstDeployment|TestComposePreviewsDoNotRequireConfiguredProjectEnvironment|TestContainerLogSubscriptionRejectsChangedTarget|TestDirectoryOwnershipRejectsRunningSharedMounts|TestDirectoryOwnershipAllowsUnusedDataDirectories|TestDirectoryOwnershipUsesTargetMountPaths|TestDirectoryOwnershipBlocksWhenMountInspectionFails|TestDirectoryOwnershipAllowsOwnRunningMounts|TestDeploymentRejectsOccupiedMountBeforePreparation)$'
```

TMPDIR 使用规范化的 macOS 临时路径；没有修改项目或全局环境配置。未启用 E2E 开关。

| Go 执行包 | 通过的顶层单元数 |
| --- | ---: |
| internal/application/deployment/usecase | 21 |
| internal/application/gateway/usecase | 3 |
| internal/common/workspacepath | 1 |
| internal/infrastructure/external/traefik | 2 |
| internal/infrastructure/runner/local | 6 |
| internal/infrastructure/runner/ssh | 13 |
| internal/infrastructure/runner/target | 1 |
| internal/infrastructure/storage/local | 4 |

## Findings

### F-01：嵌套 DooD 挂载源之间的占用校验缺失（P1，已修复）

修正位置：internal/application/deployment/usecase/directory_ownership.go:74。

修正前，当前服务的根目录及相对挂载源只与其他服务的根目录物理映射比较，其他服务的嵌套挂载源未参与。以下合法的 Orbit 挂载布局使两个逻辑上独立的服务目录共享数据：

| Orbit 可见路径 | Docker daemon 物理路径 |
| --- | --- |
| /app/a | /host/a |
| /app/a/data | /host/shared |
| /app/b | /host/b |
| /app/b/data | /host/shared |

当服务 A 已在 /app/a 运行，服务 B 在 /app/b 部署并使用相对挂载 ./data 时，原检查只将 /host/b、/host/shared 与 /host/a 比较，返回成功。仅检查两个独立的服务根目录也不会发现此问题，必须核对实际运行挂载。

临时单元诊断使用上述路径映射替身，调用真实 checkDirectoryOwnership，预期拒绝但实际成功。测试失败输出为 shared nested DooD data source was accepted。该诊断没有连接 Docker、SSH 或数据库。

按用户要求修复为：通过其他服务 Compose project label 查询目标 Docker 的运行容器，inspect 实际 bind mount 并复核 Running。相同或父子物理源阻止部署，并指出占用服务；查询/解析失败同样阻止文件准备。本服务自己的挂载跳过，停止容器与 named volume 不构成此项运行数据占用；原有根目录归属保护继续有效。

复用现有目标 QueryAtEnvironmentRoot 与 Worker session，local/Linux SSH/Windows SSH 保持原执行方式。Windows 分隔符、大小写和 Docker Desktop 宿主机盘符路径统一比较；DooD 在 Docker Desktop 容器中的物理路径同样处理。稳定回归证明冲突发生时不 StageWorkspace、不 Run、不改变旧运行目录。

### F-02：Windows 盘符根目录在规范化后逃过校验（P2，已修复）

位置：internal/common/workspacepath/service_directory.go:45。

修正前，NormalizeServiceDirectory("D:/", "windows") 先通过绝对路径检查，随后 path.Clean 将其变为 D:。之后的根目录判断仍使用要求冒号后有斜杠的正则，不能匹配 D:，最终无错误返回。

临时单元诊断预期拒绝根目录，实际得到 D:，失败输出为 Windows drive root accepted as "D:"。该诊断直接调用真实路径函数，没有远程操作。

原影响：HTTP/MCP 调用可接受并保存该值，Worker 再次按绝对路径校验 D: 时失败；前端已拒绝盘符根目录，导致前后端校验不一致。

已按清理后的 Windows 盘符根目录形式拒绝，TestServiceDirectoriesUseTargetPlatform 覆盖正斜杠、反斜杠及 ./ 形式，最终单元检查通过。

## Missed or expanded scope

- 本阶段没有扩展业务实现范围。验证发现的 F-01/F-02 均位于原定目录保护范围，已修复并同步文档。
- 按最新用户要求，固定 Go 测试入口保留编译范围，执行限定为具名单元；数据库迁移和已有集成/E2E 适配只做代码核对。
- Gateway 类型纠正、新字段及三库新迁移属于已接受计划，没有修改先前已执行的迁移。
- 文本截断、服务列表操作列及自定义路径列宽属于先前任务，不纳入本次部署目录验收。

## Risks and incomplete items

- F-01/F-02 已修复，最终固定检查与单元回归没有已知未修复失败。
- 用户确认集成测试无问题，并明确要求完成验收。本代理未执行真实 SFTP、Docker/Desktop/DooD 或浏览器流程；实际平台覆盖以用户执行结果为准，不将单元替身视为实机证据。
- 本轮不执行迁移测试。SQLite 在 Implementation 有运行记录；MySQL/PostgreSQL 仅核对 SQL，三库真实升级缺口保留。
- 新配置 hash 不兼容旧 hash；按已更新的指南，在升级前完成或取消活动任务，升级后重新提交，不改写历史成功 hash。
- 已部署服务改目录会改变相对数据挂载；Gateway 的路由、证书、ACME 数据也可能缺失。按用户决定只警告，数据和路由需用户另行处理，部署不自动迁移或发布。
- 运行挂载检查反映查询时的容器状态，没有引入跨服务部署锁；本次不承诺阻止其他并发操作在检查后新占用同一源。
- 本代理未使用浏览器或启动开发服务器，用户已有服务器未操作。

## Conclusion

固定静态检查、14 项前端组件测试与 51 项筛选 Go 单元测试通过；两项初轮诊断问题均已修复。用户确认集成测试无问题，并要求完成 SpecFlow 验收，验收结论为通过。

Review status 已按用户明确验收要求标记为 Accepted。临时诊断文件已清理，稳定回归保留；本阶段保留用户改动与暂存状态，没有执行 Git 暂存、提交或推送。
