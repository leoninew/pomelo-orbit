# 自定义路由同步优化与环境目标恢复验证
最后修改时间: 2026-10-03 13:04:00

Flow mode: light
Review status: Draft
Current stage: Verification

## Basis

- [Intent](../intent/20261003-route-sync-operation-cost.md)：Accepted，以 13:04:00 合并后的范围和决策为依据，包含操作成本优化与环境目标切换恢复。
- 用户最新要求完成本项验证，明确不进行集成测试。本次验收采用代码审查、静态检查和单元/前端组件测试；集成测试、真实 SSH/Docker、实机性能及实际浏览器验收均排除在本次完成条件之外。
- 已核对当前暂存变更与逐条读取接口；最新阶段预算已补做配置、匹配预算单元测试及静态检查。本文合并两项工作的验证结果，统一证据来源和完成边界。
- 未创建 Spec/Plan，轻量模式按 Intent 核对。

## Intent alignment

| 目标 | 实际实现 |
| --- | --- |
| 单条仅操作对应 Route | 应用层先读取业务记录与 code；selected 不枚举发布目录。`InspectPublication` 仅读取当前记录、对应 YAML/PEM 及 pending 必需的 previous |
| Project 使用相同单条流程 | 枚举业务 ID 与发布目录 ID 并集后逐条检查；保留已删除 Route 撤销，忽略没有 state.json 且没有业务记录的空目录 |
| 请求内连接复用 | `RuntimeSessions` 通过 target runtime 分流；SSH/SFTP 绑定目标与凭据修订，退出关闭，阶段取消或传输失效时淘汰，恢复可以重新连接 |
| 降低加载核验操作量 | 仅查询候选及撤下资源的协议；HTTP/TCP 每轮各两个 API，协议切换仍观察旧协议资源缺席 |
| 无变化和 pending 优化 | 当前 revision、文件、证书一致时先核对实际加载；匹配后不写文件或重载。pending 恢复所得基线等于期望时复用运行核验 |
| 逐条反馈、最终刷新 | Web 仍循环单条确认；saved 立即更新草稿/启停，finished 在整轮完成或失败后刷新一次，详情仅重读 Route |
| 独立预算与可用诊断 | 删除后端总预算与 Axios 总超时；阶段记录起止、耗时和结果，SSH/SFTP 错误保留 context 原因，日志带 project_code/route_code |
| Docker 失败原因可见 | Docker 查询失败或 Gateway 未运行采用独立 unavailable 编码，通用 HTTP 层映射 503；模态与 Toast 共用本地化原因，原始诊断只入日志 |
| 历史 TargetRevision 可读取 | `readPublicationRecord` 校验 Route/Project/Gateway 身份，允许来源 revision 不同；selected 与 Project 使用同一逐条读取接口，Inspect/preview 不写文件 |
| 显式重新发布使用当前目标 | `PublishRoute` 构建候选记录时写当前 `<environment_id>:<target_revision>`；`publicationUnchanged` 要求 revision 一致，历史记录即使内容相同仍执行提交和加载核验 |
| 当前目标变化仍使预览过期 | session 绑定目标、工作区及 Gateway；`resolveTarget` 检查当前 Environment identity/revision，依赖 hash 覆盖预览与确认之间的变化 |
| 切换与探测不自动迁移或发布 | 未修改 Environment update/probe；恢复只操作当前目标工作区，不连接旧目标搬运或删除文件 |

## Spec and plan alignment

不适用。按已接受 Intent、最新决策及当前活文档核对。

## Actual diff summary

- Route port/usecase 改为显式会话、发布 ID 枚举和逐条发布检查；保留业务保存短事务及发布前当前项的修订复核。
- Traefik adapter 增加无变化判断、pending 核验复用、协议维度的加载/缺席检查和可读阶段日志；目标 YAML 存在检查仍阻止覆盖他人文件。
- 随逐条接口替换移除历史 target revision 的读取硬性拒绝；显式发布写当前 revision，当前目标或工作区变化仍报告 `route_sync_preview_expired`。
- deployment port 与 local/SSH/target runtime 增加请求会话能力；共享 SSH/SFTP，取消与连接失效后淘汰，不跨请求缓存。
- config/bootstrap/example 切换为独立阶段预算；`operation.StartStage` 保留底层错误与截止原因。
- Web 合并最终刷新，保留逐条结果、已保存草稿和失败重试；预览/确认请求取消 Axios 总超时，Docker/Gateway 错误映射到统一提示。
- 更新路由指南和 CD 运行时，明确单条与批量操作数量、逐条读取、条件执行及历史 revision 的显式恢复流程；本轮合并两份 Intent 和两份 Verification，未新增产品代码。

## Expected versus actual files

| 预期职责 | 实际范围 | 结论 |
| --- | --- | --- |
| Route 应用编排 | `internal/application/route/{port,usecase}/` | 符合；不新增批次续跑或持久化索引 |
| 发布适配器 | `internal/infrastructure/external/traefik/{publication,publication_files,route,session}.go` 及测试 | 符合；移除全局缓存与跨 Route 状态/证书扫描是最新接受的范围 |
| 连接生命周期 | `internal/application/deployment/port/port.go`、`internal/infrastructure/runner/{local,ssh,target}/` | 符合；复用能力保持在 runtime 边界 |
| 预算与阶段日志 | `internal/{config,bootstrap,common/operation}/`、`configs/config.yaml`、`.env.example` | 已有超时修复作为本次基础保留 |
| 页面反馈与请求 | `web/src/{api/route,components,views/route,i18n}/` 及测试 | 符合；沿用共享模态与页面模式 |
| 活文档与过程文档 | CD runtime、routing guide、合并后的 Intent/Verification | 已对齐逐条读取、操作数量及目标恢复 |
| Environment、数据库、部署、远端服务 | 无 Environment update/probe 或 schema/迁移修改；E2E fixture 随 port/timeouts 更新 | 未扩展为自动恢复、旧目标迁移或服务管理 |

## Acceptance checklist

| 验收项 | 结果与证据 |
| --- | --- |
| selected 不列目录、不读无关记录/文件 | 通过：本轮 `TestSelectedPublicationReadsOnlySelectedRecordAndFiles`；无关记录损坏仍可成功。应用层业务范围与 code 传递已审查，既有 `TestSelectedSyncDoesNotCarryAnotherRoutesUnsyncedEdit` 结果保留 |
| Project 保留删除项撤销 | 代码核对通过：业务与发布 ID 并集逐条处理，缺业务记录时使用已发布快照撤销。既有 `TestProjectPreviewWithdrawsDeletedPublishedRoute`、`TestProjectPreviewSkipsPublicationDirectoriesWithoutRecords` 通过，本轮不重跑 usecase 集成测试 |
| 当前项发布前复核，不覆盖他人编码 | 通过：当前记录读取计数、目标文件冲突、缺失文件使预览过期及外部修改保护测试 |
| 会话复用与退出关闭 | 代码核对通过：会话复用 SSH/SFTP 并绑定目标/凭据，defer 关闭。既有本地协议 fixture 已验证连接数量、退出关闭及凭据 revision 拒绝，本轮不重跑协议集成测试 |
| 取消后恢复与坏连接淘汰 | 代码核对通过：context 取消及 SSH/SFTP Wait 淘汰连接，恢复可重连。既有 command/SFTP startup 取消和失效传输回归通过，本轮不重跑协议集成测试 |
| 普通 HTTP 两个 API，协议切换旧资源核验 | 通过：selected 请求数量断言、`TestProtocolSwitchVerifiesNewAndRetiredResources` |
| no-op 不写文件且仍确认运行结果 | 通过：无写入/无 HUP；配置未加载时重载，API 不可用时报告失败，HTTPS 证书结果仍为 unverified |
| pending 相同期望不二次发布 | 通过：`TestPendingMatchingDesiredPublicationDoesNotPublishAgain`，仅写 confirmed 记录，只有一轮重载及两个 HTTP API |
| 释放编码可复用且不误删 | 通过：批量改名后编码复用、`TestUnchangedRenamePreservesOldCodeReusedByAnotherProtocol` 及撤销/改名回归 |
| 独立预算与失败后继续 | 通过：本轮文件预算不截断重载、匹配在重载后开始、恢复独立限时回归。批量继续逻辑已审查，既有单项超时后处理下一项的 usecase 结果保留 |
| 前端逐条反馈与一次最终刷新 | 通过：同步组件/页面测试，包含批量无中间刷新、失败仍刷新和详情不额外 GET Gateway |
| 可读阶段日志与 Docker 原因 | 通过：`TestRoutePublicationLogsReadableCodesAtEveryStage`、Docker 查询/未运行测试及模态/Toast 本地化测试 |
| 历史记录不阻断 selected 或 Project 读取 | 通过：`TestHistoricalTargetRevisionAllowsExplicitRepublish`；Project 枚举后调用相同 `InspectPublication`，既有 usecase 覆盖合法 project/空 route_ids |
| 历史记录读取不修改文件 | 通过：历史 revision 单元测试断言 writes=0 且原 state.json 原样保留；预览流程只有读取与校验操作 |
| 显式重发写当前 revision，历史记录不走 no-op | 通过：SSH revision 0 切换 local revision 2，同内容仍 committed，记录写入 `environment-1:2` |
| 请求内及预览/确认之间的目标变化拒绝发布 | 通过：历史 revision 单元测试覆盖会话打开后 revision 再变的过期错误；既有 `TestTargetChangeInvalidatesFirstPublicationPreview` 覆盖依赖变化后不发布 |
| 切换/Probe 不自动发布或迁移 | 代码核对通过：未改 Environment usecase；显式恢复仅操作当前目标，当前项归属、外部修改、证书版本和恢复材料校验保留 |

## Test results

操作成本验证使用内存 fake runtime 的 14 个指定 Traefik 发布单元测试，使用 `-run` 完整名称选择和 `-count=1`，不使用测试结果缓存；全部通过，包内测试耗时 1.106s。范围包括 selected 读取、批量编码复用、no-op、pending、协议切换、可读日志、SFTP 错误保留、文件/重载/匹配/恢复预算、证书损坏及恢复材料保护。历史 revision 回归使用同一内存工作区与可变 target resolver，已在先前全量测试中通过。上述模拟证据不代表实机结果。

最新预算调整后补做配置测试、匹配预算单元测试和 `task check`，均通过；本轮合并文档不重复测试。完整 Go 测试与前端组件检查来自用户排除集成测试之前的既有结果；本轮未重跑完整 Go 测试、usecase 数据库集成测试、本地 SSH/SFTP 协议 fixture 或 E2E，未操作真实 SSH、Docker 或数据库。

| 命令/检查 | 证据来源 | 结果 |
| --- | --- | --- |
| 14 个指定 Traefik 发布单元测试，`go test` 使用 `-run` 和 `-count=1` | 操作成本验证 | 通过，exit 0；内存 fake runtime，不执行集成测试 |
| `go test ./internal/config -count=1` | 最新预算调整后 | 通过，0.958s；包括 API 上限大于匹配上限的合法配置 |
| `go test ./internal/infrastructure/external/traefik -run '^TestConfigurationMatchBudgetStartsAfterReload$' -count=1` | 最新预算调整后 | 通过，0.934s；匹配预算在重载结束后开始 |
| `go test ./cmd/... ./internal/...` | 既有 | 通过，exit 0；部分包使用缓存，包含历史 revision、目标过期、usecase 和协议 fixture，本轮不重跑 |
| `task check` | 最新预算调整后 | 通过；Web typecheck/lint/format、Go 配置/格式/静态检查，0 issues |
| `yarn --cwd web lint:fix` | 既有 | 通过；未产生额外代码 diff |
| `yarn --cwd web typecheck` | 既有 | 通过，包含在 task check 中 |
| `yarn --cwd web test src/components/RouteSyncDialog.test.ts src/views/route/routeSync.test.ts src/api/route/route.test.ts` | 既有 | 3 个文件、25 项通过 |
| Go race detector | 工具链限制 | 需要 CGO，缺少 gcc；未运行成功，不作为本次完成条件 |
| Environment 和迁移 diff 核对 | 合并验证 | 无此次修改；未新增自动恢复或旧目标迁移 |
| 集成测试、实机性能、实际浏览器验收 | 用户指定排除 | 本轮不执行，不作为本次完成条件；模拟证据不代表实机结果 |

## Operation cost and budgets

详情单条仍有一次 preview 和一次 confirm；列表整批一次 preview、B 次 confirm，以保留前端逐条反馈。正常单条最多读取自身 state.json 三次：预览、确认准备、发布前复核；selected 发布目录枚举为零。Project 预览枚举一次 ID 后逐条读取。Web 每轮最终刷新一次，详情只 GET Route。

普通 HTTP/TCP 一轮加载匹配使用对应协议两个 API；改名核验旧资源，协议变化最多四个 API。无变化且已加载时没有文件提交、HUP 或收尾；pending 已恢复且期望相同时不再进行第二轮提交和加载。Gateway 校验仍是每个 preview/confirm 请求一次，SSH exec 仍逐条启动远端命令，未承诺所有操作变成单次远端请求。

当前独立阶段预算已按用户指定校准，配置、示例环境变量和活文档一致：

| 阶段 | 上限 |
| --- | --- |
| Gateway 锁等待 | 30s |
| 状态准备 | 10s |
| 文件发布及收尾 | 5s |
| 单次 Traefik API 请求 | 10s |
| 重载 | 5s |
| 配置匹配 | 5s |
| pending/失败恢复 | 20s |

API 请求若处于匹配阶段内，同时受父 context 剩余预算限制，使用更早的截止时间；配置允许 API 上限大于匹配上限。没有请求总预算，阶段耗时继续记录。优化前状态准备约 18-43s 的日志属于旧操作基线；最新 Intent 记录用户提供的优化后 9 条 Project 预览为 7.274s，是状态 10s 的校准依据。更多路由共同使用状态预算，证书写入、协议切换和失败恢复仍需后续实际日志校准；本次未自行实测 P95/P99，也不承诺固定提速倍数。

## Scope changes and risks

- 最新 Intent 已明确取消全局发布元数据缓存和跨 Route 状态/证书冲突扫描；旧接口及对应旧行为没有恢复。当前 Route 的文件、证书版本、恢复材料和目标文件冲突检查仍有效。
- 同域名多 Route 共享 Traefik TLS store；当前不再通过全局发布扫描防止不同 Route 的证书配置冲突，不能由按文件独立发布推导出证书隔离。
- confirmed 记录可以包含上一版快照；当前读取不会检查其旧文件，no-op 也不把历史编码当作必须缺席的资源。只有当前 pending/本次变更使用必要的上一版材料。
- 连接复用扩大了生命周期，自动化验证覆盖取消、坏连接和退出；race 检测仍有工具链限制。
- 数据库保存和远端发布不是跨系统事务；业务已保存但发布失败仍保留 saved 状态，失败恢复有自己的有限预算。
- 进程内 Gateway 锁不承诺多 Orbit 实例或外部人工操作间的互斥；当前范围保留此边界。
- 历史 revision 的来源语义不放宽当前项文件和归属校验；新目标仍必须存在合法 Gateway provider/mount，Probe 成功不能证明 Gateway 就绪。
- 本修复不保证 SSH 与 local 指向同一物理目录，不迁移旧目标状态；旧目标文件由用户管理。目标 revision 校验在解析节点发生，不构成跨系统原子事务。
- 状态准备预算涵盖整个 Project 范围；路由数量增长或网络变慢可能触发阶段超时，失败保留恢复材料。

## Incomplete items

本次用户指定的验证范围内无未完成项。

集成测试及真实 Windows/Linux SSH、Docker Desktop/DooD、性能与实际浏览器验证按要求排除；race detector 保留缺少 C 编译器的工具链限制。这些边界不阻止本次验证完成，也不被报告为已实测。

## Conclusion

两项验证已按用户限定范围完成。实现符合合并 Intent 的逐条读取、请求内连接复用、必要协议核验、no-op/pending 优化、逐条反馈及最终刷新要求；历史目标 revision 不再永久阻断同步，显式重发写当前 revision，当前目标变化仍使预览过期。发布单元测试、历史 revision 既有回归、最新预算检查及既有前端检查通过，未发现需要补回的代码或修复的偏差。集成测试按要求排除，验证结论不包含实机目标切换与性能保证；Review status 保持 Draft，待用户审阅。
