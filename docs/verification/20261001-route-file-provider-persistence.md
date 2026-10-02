# Route 文件发布、重启恢复与独立同步验收
最后修改时间: 2026-10-02 13:19:32

Review status: Draft

Flow mode: standard

Current stage: Implementation

## Basis

- [Intent](../intent/20261001-route-file-provider-persistence.md)：Accepted。
- [Plan](../plan/20261001-route-file-provider-persistence.md)：Accepted，包含用户在实施中确认的“发布/撤销清单”调整。
- 用户已看过实现结果及真实环境覆盖边界，于 2026-10-02 要求“补验证文档”。本文记录最终 diff、已执行检查和验收缺口，保持 Draft 待审阅。
- 本文最初仅补录已执行验证；用户随后明确部署直接覆盖受管配置、不新增 UI，已修订 Intent/Plan 并回到 Implementation 补齐共享部署行为。下述既有实机证据不视为已完成本次 REST 首次切换实机验收；未重新部署用户服务。
- 用户随后要求补回 Windows SIGHUP 重载，并强调本地 Windows/Linux、SSH 与 DooD 分流及架构边界；本轮仍在 Implementation，补录必要检查及 Windows 独立实机回归，Linux 主动重载不实现，远程 SSH 待后续实测。
- 用户最新要求执行结果只显示完成/失败，并在失败状态上提示原因；本轮按此要求调整前端，补录相关自动化检查，API 的完整诊断与恢复语义保留。

## Intent alignment

| 目标 | 实际行为与证据 |
| --- | --- |
| Traefik 重启后恢复 | File provider 从持久化 YAML 和 PEM 加载；独立 Traefik 3.6 E2E 绕过 Orbit 发布 hook 直接重启，HTTP 访问及 TLS leaf fingerprint 检查通过 |
| 路由发布互不影响 | 每条 Route 拥有编码命名的 YAML，证书版本及内部归属记录使用稳定 ID；改名撤下旧文件，单条 A 不携带 B 的未同步编辑，A 更新/撤销时 B 文件不改写；未知文件保留 |
| 保存与发布分离 | 普通编辑/证书先保存业务数据，启停可为前端草稿；Gateway 重建只核验持久化发布状态，不读取编辑中的 Route 重发 |
| 全量同步简化 | Project 范围便利地解析业务记录与已知发布记录的并集；预览发布/撤销/跳过清单，确认冻结 ID 顺序；不比较 Traefik 增删改或提交运行时 hash |
| 批量失败可报告和重试 | A/B/C 顺序执行，B 失败后 C 仍执行；汇总编码及逐项保存、文件、配置、证书、恢复/清理状态交给 Web/MCP；重试只取未完成 ID |
| SSH 与权限机制 | 复用显式 local/SSH target、WorkspaceFile/SyncFiles、Mode 和 IgnoreIfExists；没有 SSH 失败回退 local、权限放宽或 Route 专属 SSH writer。真实远端与实际权限失败证据尚缺 |
| Orbit 容器与 DooD | Linux 隔离运行器挂载宿主 Docker socket 和持久化卷，logical path 与 daemon physical source 不同，真实挂载、发布及重启测试通过 |
| 范围边界 | 现存受管 Gateway 部署覆盖 provider/mount，不转换旧 REST 快照或保留 REST 发布分支；未新增迁移 UI、TCP listener 行为、数据库迁移、UID/GID 或 Windows ACL 管理 |

## Spec alignment

标准模式未创建独立 Spec，按已接受 Intent、Plan 和用户后续明确选择核对。

## Plan alignment

| 计划内容 | 实现及核验 |
| --- | --- |
| 独立动态文件 | `gateway/dynamic/route-<code>.yaml`；router/service 由编码派生，编码修改在显式同步时变更资源并撤下旧路径，失败恢复旧路径；内部归属仍使用 ID |
| 不可变证书版本 | `gateway/certs/route-<id>/<sha256-revision>/{cert.pem,key.pem}`；先写版本，再提交 YAML；已有版本验证内容后复用，清理限定该 Route 并保留当前/上一版本 |
| 发布记录与恢复 | `.orbit/route-publication/<id>/{state.json,previous.yaml}` 不挂载给 Traefik；记录限定 Project、Environment target revision、Gateway Application 和所在 Service 目录；不保存私钥正文 |
| 中断恢复 | pending 候选或上一版通过 YAML/PEM fingerprint 核对，确认或恢复后才接收新发布；未知外部修改保留，不能自动以业务草稿重建 |
| 显式范围与过期核对 | `selected`/`project`；`business_hash` 覆盖选定业务、草稿及 Gateway，`publication_hash` 覆盖选定发布记录、实际文件和运行依赖；确认再复核冻结输入 |
| 发布/撤销/跳过 | 启用 Route 发布；禁用或已删除的已知发布 Route 撤销；禁用且从未发布的 Route 保存必要草稿后跳过文件操作，不创建记录、不删除未知文件 |
| 文件提交与权限 | 复用 WorkspaceFile/SyncFiles；受管文件 Mode 为 `0600`；local 完整临时文件关闭后替换活动文件，SSH 沿用既有 writer；不进行目录递归权限整改 |
| Gateway 配置归属 | 四种初始 Version 使用 File provider；按用户最新决定，enrichment 在有效计划中覆盖 provider 和受管目录挂载，静态文件强制覆写，Gateway 自动 force-recreate。其他声明保留，不写回 Version；运行前仍核对实际配置和挂载 |
| 部署保护 | API/worker 管理器共用按 Project/Gateway 的进程锁；部署前保护已发布上游、网络、entrypoint/resolver；Gateway 部署/重启后等待 readiness 再核验已发布状态 |
| 批量预算与事务 | 单条短事务复核/保存后执行外部 I/O；批量预算 105 秒，配置匹配及恢复各最多 8 秒；失败继续，未执行项独立报告，不回滚成功项 |
| 配置与证书结果 | 文件写入与 router/service 匹配分别报告；正常 HTTPS 同步证书结果为 `unverified`，不强制 TLS 握手；E2E 独立通过正确 SNI 检查实际证书 |
| 前端反馈 | 详情只选当前 ID，列表显式 Project 预览；Web 使用每项原修订顺序调用单条确认，响应立即更新记录及已保存草稿。原表格保留名称、操作、协议、规则、状态，证书方式归协议列；只有一个处理状态字段与失败提示，成功记录保留，重试不携带成功项 |

预算数值和超时分支已核对代码；本轮未真实等待整批 105 秒或模拟生产网络断开。进程锁测试验证跨管理器共享，不代表多个 Orbit 实例共享写入能力。

## Actual diff summary

- 自定义 Route 发布适配器移除 REST PUT、聚合快照与整个 PEM 目录 prune，增加独立 YAML、证书版本、发布元数据、配置匹配及单条恢复。
- 同步用例与 DTO/proto 改为显式发布清单和类型化逐项结果；调整 HTTP/MCP 映射并生成 Go/TypeScript 契约。
- Gateway 初始 Version 使用 File provider；enrichment 统一覆盖有效计划中的受管 provider/mount，command 与 worker 自动重建 Gateway；保护已发布依赖，worker hook 从数据库重发改为检查持久化状态。
- local/SSH/target runtime 增加受限工作区文件读取、列举和精确删除；local 共享 writer 改为完整临时文件提交，并覆盖其正常替换与清理行为。
- 前端同步弹窗、列表、详情和中英文文案匹配新契约；删除仅测试旧运行时差异的 `sync_test.go`，补充当前发布行为的测试。
- 更新产品、架构、决策和路由/证书/挂载/部署指南。补验收文档前 diff 为 56 个文件；本次再新增本文，仅修改三个过程文档。

## Expected versus actual files

| 计划范围 | 实际文件或目录 | 对齐情况 |
| --- | --- | --- |
| Route 应用契约与用例 | `internal/application/route/{dto,port,usecase}/` | 符合计划；新增 `dependency.go`，移除旧差异测试及不再调用的批量目标解析 helper |
| Traefik 发布 | `internal/infrastructure/external/traefik/{route,publication}.go` 及对应测试、`route_restart_e2e_test.go` | 按允许的包内拆分职责；保持现有 API 查询能力 |
| Gateway 拓扑与部署 | `internal/model/gateway.go`、`internal/application/gateway/usecase/initial.go`、`internal/application/deployment/{port,usecase}/` | 按用户最新决定覆盖有效计划中的受管 provider/mount，command 自动 force-recreate；其他拓扑和库存不重写 |
| 文件 runtime | `internal/infrastructure/runner/{local,ssh,target}/files.go`、local `runtime.go` 及测试 | 符合计划；没有 SSH 私钥临时权限修复或 Windows 专用替换 |
| API 与生成代码 | `proto/orbit/v1/route/route.proto`、HTTP route handler、MCP delivery、对应 Go/TS 生成目录 | 符合新清单/结果契约；旧 proto 字段号保留为 reserved，不保留旧行为 |
| Web 同步 | `RouteSyncDialog.vue`、`RoutePage.vue`、`RouteDetail.vue`、locale 及测试 | 符合计划；沿用共享 AppDialog/Actions/表格样式 |
| 组合根与 Web API client | `internal/bootstrap/`、`web/src/api/route/route.ts` 未修改 | 现有 RouteManager/runtime 注入和远端请求超时可直接复用，生成类型自动传递新契约，无需额外接入修改 |
| 活文档 | `docs/{product,architecture,decisions}/` 及五个相关 guide | 符合计划并反映用户后续确认的预览简化 |
| 过程文档 | 本任务 Intent、Plan、Verification | 同名文件保留，当前阶段更新为 Verification；验收状态仍为 Draft |

## Acceptance checklist

| 验收项 | 结果与证据 |
| --- | --- |
| File provider 实际加载、直接重启恢复 | 通过：`TestFileProviderRestoresRoutesAfterTraefikRestart`，真实 Traefik 3.6 HTTP 访问及 TLS fingerprint |
| A 不携带 B 未同步编辑 | 通过：`TestSelectedSyncDoesNotCarryAnotherRoutesUnsyncedEdit` |
| A 更新/撤销不重写 B 配置、证书或权限 | 自动化通过：`TestRoutePublicationUpdatesAndWithdrawsOnlyOwnedFiles`；真实 E2E 核对 B YAML 原样保留及 TLS 证书 |
| 未知动态/证书文件保留 | 通过：上述 adapter 测试增加未知 YAML/PEM 保留断言；Project 清单仅发布业务及已知 ID |
| 禁用和已删除 Route 撤销 | 通过：禁用后直接重启不恢复 A；`TestProjectPreviewWithdrawsDeletedPublishedRoute` 验证数据库删除后仍可撤销 |
| 禁用且从未发布时跳过 | 通过：`TestDisabledUnpublishedRouteSavesDraftWithoutPublishing`，业务草稿保存而文件/记录无操作 |
| 证书版本复用、腐坏检查及冲突 | 通过：`TestCertificateRevisionIsReusedWithoutRewritingAndRejectsCorruption`、`TestPublishedManualCertificateConflictsWithCandidateACME` |
| 手工证书轮换及重建恢复 | 通过：真实 E2E 轮换后和 Gateway force-recreate 后分别检查 TLS leaf fingerprint |
| 发布失败和中断恢复 | 自动化通过：单条权限错误注入恢复、pending 候选确认、未知外部修改保留、缺失候选/旧证书处理；无真实远端目录权限失败证据 |
| 预览修订与范围核对 | 通过：已保存证书变化使预览过期，首次发布的 target 依赖变化被拒绝；代码核对哈希只取选定发布记录，不取运行时差异 |
| 批量 A/B/C 失败继续 | 通过：`TestBatchSyncContinuesAfterFailureInFrozenOrder`，B 权限失败而 A/C 保留成功 |
| Gateway 与上游部署保护 | 通过：`TestGatewayRecreationWaitsThenVerifiesWithoutReadingEditableRoutes`、上游/网络移除和 resolver 移除测试，真实 E2E Gateway 重建保留文件 |
| Gateway 部署覆盖受管配置 | 自动化通过：`TestEnrichGatewayPlanReplacesManagedProviderAndMounts`、`TestGatewayDeploymentOverwritesStaticConfigAndRecreatesContainer`；覆盖 REST 移除、File 配置/挂载替换、IgnoreIfExists=false、其他声明保留、重复 enrichment hash 稳定和普通业务 Service 行为 |
| API/worker 协调 | 通过：`TestGatewayPublicationLockIsSharedAcrossManagers` 和部署调用点代码核对 |
| 详情单条、列表 Project、失败重试及草稿保留 | 通过：RouteSyncDialog 和 routeSync 的 16 项测试，包含结构化协议、证书列、逐条反馈与列表/详情未保存草稿 |
| 配置成功与证书未验证分开 | 通过：adapter 保留独立结果；`keeps the original table open through synchronization and completion` 验证证书未验证不改变配置完成状态，不宣称每次同步完成 TLS 验证 |
| 单一状态与失败提示 | 通过：预览、执行与完成保持同一张五列表，只有一个处理状态字段；每条响应立即更新状态，权限、重载与清理失败显示本地化提示，不显示内部阶段或请求/操作 ID |
| 整批预览独立确认 | 通过：`TestProjectPreviewCanBeConfirmedOneRouteAtATime`，同一 Project 清单的发布/撤销/跳过逐项确认，原修订在前项发布后仍有效；空草稿统一规范化 |
| 原预览修订与继续执行 | 通过：`TestItemRevisionRejectsEditedRouteWithoutInvalidatingOtherItems`；前端确认提交每项原修订，预览过期/网络失败后继续，保留失败草稿，显式重试不重复成功项 |
| local 正常替换、target 分流和 DooD 路径 | 通过：local runtime/target 测试与真实 Linux DooD E2E，logical/physical 路径不同 |
| 真实 SSH Linux/Windows 发布与权限 | 未验证：没有实际远端目标；target/Mode 测试不能代替实机证据 |
| Windows 正常文件热更新 | 已补回目标端 SIGHUP 重载；真实 Windows bind mount E2E 通过，发布前后容器启动时间不变。替换窗口/ACL 专项仍不整改 |
| 编码命名、改名、撤销和复用 | 通过：ID 与编码不同的路由命名、稳定证书归属、改名撤旧、三阶段失败恢复及重试、双路径 pending 恢复、编码占用/未知文件保护、未同步改名撤销与编码复用回归；Windows E2E 实测改名后原文件消失 |
| 删除动态目录后重建 | 通过：Gateway 预检允许缺失已确认 YAML，仍拒绝外部改写；显式同步重建或撤销缺失文件，预览后删除使修订过期。Windows E2E 删除完整动态目录、staging 重建空目录、Gateway 重建后显式发布并校验 TLS；部署不会将缺失文件报告为已恢复 |

## Test results

以下结果来自同一任务的实现阶段，均已在上一轮汇报；本次文档补录不将缓存结果或模拟测试称为新实机验收。

用户最新决定后的 Implementation 补充已再次执行 `task check`（0 issues）和 `go test ./cmd/... ./internal/...`，均通过。新增回归覆盖有效计划配置/挂载替换、重复 enrichment hash 稳定、静态文件覆写、Gateway 重建命令及普通 Service 行为；部分无变更包使用 Go 缓存，未进行新的实机测试或前端修改。

| 命令与环境 | 结果 |
| --- | --- |
| 根目录 `task proto` | 通过，生成最终发布清单与结果契约；本次未重复生成未变化的契约 |
| 根目录 `yarn --cwd web lint:fix` | 通过 |
| 根目录 `yarn --cwd web typecheck` | 通过，包含在最终 `task check` 中 |
| 根目录 `task check` | 最终完整通过：Web typecheck/lint/format、Go lint 配置/格式/静态分析，0 issues |
| 根目录 `go test ./cmd/... ./internal/...` | 通过，部分结果使用 Go 缓存；追加未知文件保留断言后相关包再次通过 |
| `go test ./internal/infrastructure/external/traefik ./internal/application/route/usecase` | 最后追加断言后通过 |
| `yarn --cwd web test src/components/RouteSyncDialog.test.ts src/components/RouteCertificateDialog.test.ts src/views/route/routeSync.test.ts src/views/route/routeDomain.test.ts src/views/gateway/gatewayConfigForm.test.ts src/views/gateway/gatewayNavigation.test.ts` | 6 个文件、18 项通过 |
| Linux DooD 隔离运行器，`POMELO_ORBIT_TRAEFIK_E2E=1`，`TestFileProviderRestoresRoutesAfterTraefikRestart` | 最终通过，21.44 秒；真实 Traefik 3.6、HTTP 请求和 TLS 握手，非 mock |
| Windows 原生进程，同一真实 Docker E2E，补 HUP 前重跑 | 当时失败，10.875 秒；首次发布后仅观察到 internal provider，配置匹配超时并恢复；测试实例已清理 |
| Windows Docker Desktop 最小隔离 watcher 探针，Traefik 3.6.23 | 启动读取成功；宿主替换后容器可读新文件，10 秒内 API 仍为旧规则；仅重启隔离容器后 API 显示新规则。证明当前环境的文件可见性与热更新通知不同；探针及容器已清理 |
| Windows 原生进程，补 HUP 后的 `TestFileProviderRestoresRoutesAfterTraefikRestart`，`POMELO_ORBIT_TRAEFIK_E2E=1` | 通过，21.10 秒；真实路由创建/更新/撤销、HTTP 请求、TLS fingerprint 及证书轮换、直接重启和 force-recreate 后恢复，发布前后容器启动时间不变。隔离容器/目录已清理 |
| 本轮根目录固定检查与同步前端测试 | `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check` 最终通过（0 issues），`go test ./cmd/... ./internal/...` 通过；两个同步前端测试文件 16 项通过。模态窗扩大后再次运行 lint:fix 与 typecheck，通过 |
| 本轮 Windows 原生独立 Traefik 3.6 E2E | 通过，27.41 秒；ID 不同于编码的改名撤旧、HTTP/TLS 热更新、证书轮换、直接重启/重建恢复、删除完整动态目录后的空目录 staging 和显式恢复发布；TLS fingerprint 核验通过，测试实例及目录已清理 |
| 根目录 `git diff --check HEAD` | 本次补文档前后均通过，检查受 Git 跟踪的 diff |
| 新增验收文档行尾空白与 Intent/Plan 相对链接 | 行尾空白检查通过，链接目标均存在 |

实现中已修复检查发现的格式、错误字符串大小写和测试 fixture 必需字段问题。真实 Linux Traefik 测试发现空协议段 YAML 及 TLS API 默认 options 的差异，修正后 E2E 通过。此前 Windows 原生试运行未通过，未将其计为成功结果。

Linux DooD 运行器使用挂载 Docker socket 的独立 Linux 容器，工作区卷挂载为 `/workspace`，daemon source 由现有 resolver 得到。Traefik 测试使用独立容器名、网络与临时端口；测试后容器、网络、临时卷、测试镜像及 Linux 编译产物均已清理。

## Scope changes and risks

- 用户明确选择发布/撤销清单后，移除旧运行时差异及 hash 属于已接受范围调整；原差异测试随旧行为删除，测试收敛到当前发布承诺。
- local 共享 writer 的完整提交影响 StageWorkspace/SyncFiles，超过单条 Route 适配器的范围；该调整已在 Plan 记录，并有正常替换/临时文件清理及全量 Go 回归。SSH writer 未进行权限时序或替换算法整改。
- File provider 仍共享 TLS SNI store。手工/ACME 冲突有发布前检查，不能据分文件承诺同域不同路径能使用不同证书。
- 数据库、文件与 Traefik 没有跨系统原子事务；文件提交失败后业务修改可能已保存，恢复和清理也可能未完成。接口逐项报告，pending 记录用于后续核对。
- 部署/发布协调以单节点进程为前提，不承诺多个 Orbit 实例或外部人工写入的跨进程锁。
- 正常同步没有可靠 TLS 入口观测时证书结果为 `unverified`；配置匹配不等于证书生效。真实手工证书 TLS 证据来自 E2E，未覆盖生产 ACME 签发。
- Windows 短暂文件缺失、ACL，以及任意 rootless/userns/UID/GID 组合按用户确认保留为已知限制，不扩展专项整改；用户后续授权的正常发布 SIGHUP 重载已实现并取得 Windows 实机证据。
- 用户最新要求现存受管 Gateway 在部署时直接覆盖受管配置并自动重建，通过线下传达操作，不增加 UI。这替代了此前仅校验 Version 拓扑的约束，不转换旧 REST 发布快照；首次重建到显式同步完成之间可能出现路由不可用窗口。

## Incomplete items

1. 真实 SSH Linux/Windows 的文件发布、Traefik 加载和远端权限失败未实测；目前只有既有 runtime 回归、target 分流与错误注入证据。
2. Windows 原生发布已通过 HUP 实机回归；Windows bind mount 的 DooD 与 SSH Windows 目前有平台分流和加载/恢复的模拟证据，尚无本轮对应真实运行证据。Linux 原生目录不主动重载，真实 SSH Linux 留待后续验证。
3. 当前 Web 验收来自组件测试，未进行真实浏览器桌面/移动视觉和交互验收；未主动启动开发服务器。
4. 未进行生产 ACME 签发、整批耗尽预算、真实进程崩溃/宿主断电或多 Orbit 实例测试；中断恢复采用持久化 pending fixture 验证。相关边界不得扩大为已验证承诺。
5. 本次已补齐受管 Gateway 直接部署覆盖配置与自动重建，并通过相关自动化测试；尚未实测已运行 REST 实例的首次切换。此前 Linux DooD E2E 从 File provider 配置开始，不能作为这次首次切换的实机证据。

## Gateway onboarding review

用户明确“预置”指迁移脚本初始化出来的 Orbit Gateway，不包括外部自管实例；随后决定部署时直接覆盖配置，通过线下传达重新部署要求，不新增 UI。Intent/Plan 已按该决定修订，当前回到 Implementation 完成补充。

| 场景 | 当前行为 | 操作与边界 |
| --- | --- | --- |
| 新建 Gateway | 初始化生成 File provider，部署统一应用受管规则 | 先部署，再显式同步路由 |
| 迁移脚本初始化的 Gateway，尚未部署 | 部署在有效计划中覆盖受管 provider/mount，其他库存声明不修改 | 正常部署，不按预置 ID 特判，不修改已执行迁移 |
| 导入的 Gateway 定义，尚未部署 | 导入保留声明，部署使用同一 enrichment 覆盖受管部分 | 正常部署，不增加导入专用兼容路径 |
| Orbit 管理的 Gateway，已按 REST 配置部署 | 部署移除 REST、写入 File 配置与挂载并自动 force-recreate | 线下告知重新部署一次，再显式同步一次；不转换旧 REST 路由快照 |

- Gateway 页面、普通 Service 部署入口、API/worker 和 Compose 渲染使用相同 enrichment；Route 同步仍只发布确认范围，不隐式重建 Gateway，不新增配置更新按钮、预览或迁移确认参数。
- 静态配置仍由 Version 的 controlled_file 承载，受管 provider 和目录挂载由有效计划覆盖；保留用户镜像、端口、其他 provider 与 resolver，静态文件 IgnoreIfExists=false，Gateway 部署自动强制重建。
- 首次切换不恢复旧 REST 快照；线下说明重新部署、显式预览并确认同步的顺序及维护窗口。已有 File 发布目录在重新部署时保留，不从业务草稿自动补建缺失文件。
- 用户已排除外部自管 Traefik，本次不增加外部实例接入、任意路径绑定或未知容器接管能力。

## Conclusion

最终实现与已接受的 File provider、独立发布、清单预览和部署保留方向对齐；仓库固定检查及相关自动化测试通过，Linux DooD 的直接重启、手工证书轮换及 Gateway 重建有真实证据。

远端实机、Windows DooD 和真实浏览器覆盖缺口仍存在；Windows 原生 Docker 热更新已通过本轮 HUP E2E。本文保持 Draft，不将覆盖缺口标为通过，也不表示用户已接受最终验收。用户最新决定触发 Implementation 补充；未执行 Git 暂存/提交操作、未启动开发服务器或操作用户服务。

用户补充的预置及已部署 REST Gateway 配置更新已收敛为正常部署直接覆盖，不再依赖手工修改 YAML。自动化覆盖配置更新和重建命令；首次 REST 实机切换仍未验证，不承诺保留旧内存路由或无中断切换。

## Runtime diagnosis

- 2026-10-02 用户在重新部署后对 ragflow、mineru、temporal 批量同步，预检通过，但三项均在配置匹配阶段超时，首次候选文件撤下并恢复。该失败不是预检中的 provider/mount 缺失；响应 `route_sync_incomplete` 与逐项失败一致。
- 只读检查现有 Traefik 3.6.23：File provider 已启动，watch=true，dynamic/certs 只读与 acme 读写挂载正确；动态目录源为 Windows `D:/SourceCodes/mywork/pomelo-orbit/data/deployment/traefik-default/gateway/dynamic`。回滚后动态目录为空，API 没有 `@file` 资源，日志没有候选配置语法错误。
- 独立 Docker 测试复现首次发布超时。另用与业务数据隔离的 Windows 临时目录和单个测试路由验证：启动加载 `Host(initial.test)`；宿主临时文件替换后容器读取到 `Host(updated.test)`，等待 10 秒 API 仍是 initial；重启该测试容器后 API 加载 updated。Traefik debug 日志也仅在启动/重启收到 File provider 配置。
- 初始诊断结论限于当前 Windows/Docker Desktop 工作目录的实测通知行为，不能推广为所有 Windows 环境必然失败。后续历史调查找到原生 SIGHUP 重读配置机制，用户授权补回后已通过 Windows E2E，替代了仅将该热更新失败记作已知限制的处理。
- 初始诊断未改产品代码或用户运行中的容器；后续 HUP 补充只验证隔离实例，不增加同步时重启整个 Gateway 的隐式行为。此前失败的候选已回滚，需通过现有入口显式同步。

## Windows HUP implementation

- 产品改动集中在 `internal/infrastructure/external/traefik/publication.go`：共用容器定位及提交后的加载方法，复用现有 Runtime。业务层、组合根、public port、文件 writer、前端与数据库结构均不因 HUP 增加逻辑。
- SSH 使用目标声明的平台；local 使用现有 resolver 返回的实际 mount source 识别 Windows 文件系统，覆盖 Windows 原生与容器内 Orbit 的 Windows bind mount。Linux 原生 local、Linux 原生卷的 DooD 和 SSH Linux 保持 watcher；不根据控制面运行平台决定远端行为。
- 正常发布/撤销、失败恢复及 pending 候选确认/恢复复用“必要时发送 SIGHUP，再匹配配置”；合计使用已有 8 秒加载预算。信号失败编码为 `route_sync_reload_failed`，恢复须重载并匹配，失败保留 pending 材料并允许仅重试对应 Route。
- `TestFileProviderReloadUsesTargetFilesystem` 覆盖 7 个实际平台/挂载分支；Windows 更新/撤销用 API 快照仅在信号成功后刷新，证明文件写入与配置匹配分离，且保留 B 与未知文件。信号失败、恢复失败和重试，以及 pending 候选确认和证书恢复均有回归。
- E2E 修正 Docker 重启后随机宿主端口变化及端口尚未发布时的观测时机；重载前后核对容器 StartedAt，实际 Windows HTTP/TLS 热更新及直接重启、force-recreate 后恢复均通过，耗时 21.10 秒。
- 本轮最终 `task check` 通过（Go lint 0 issues），`go test ./cmd/... ./internal/...` 通过，相关发布/路由包测试通过。E2E 前两次因旧测试的端口刷新时机失败，修正后通过；不将前两次称为发布机制失败或通过的验收结果。
- Windows 原生实机与模拟 target 分流各自记录；没有本轮真实 SSH 或 Windows DooD 证据，不扩展 Linux HUP 或声称 Linux 远端文件通知已验证。

## User review notes

- 2026-10-02：用户在实现结果汇报后要求“补验证文档”；创建本文并同步 Intent/Plan 当前阶段为 Verification，保留实际未验证项供审阅。
- 2026-10-02：用户指出不能要求手工修改 YAML，并要求考虑预置和已部署 REST Traefik；补记入口覆盖、配置更新及外部实例接入边界，产品代码与已接受 Plan 尚未调整。
- 2026-10-02：用户澄清仅考虑 Orbit 已创建或已部署的 Gateway，预置指迁移脚本初始化结果；收敛补充讨论到现存受管 Version 的配置更新与 REST 切换，排除外部实例接入。
- 2026-10-02：用户决定“部署的时候直接覆盖”，不增加 UI，通过线下传达重新部署一次；修订 Intent/Plan 并回到 Implementation，补齐有效计划覆盖、静态文件覆写和自动重建，保留历史数据不转换的边界。
- 2026-10-02：用户提供批量同步超时日志及结果截图；只读检查及最小隔离实测确认当前 Windows/Docker Desktop bind mount 未触发 File provider 热更新，记录实际限制，未修改产品代码或操作用户服务。
- 2026-10-02：用户要求补回 Windows HUP，Linux 暂不实现主动重载并留待远程 SSH 实测；提醒 local 同时支持 Windows/Linux，要求变更组织集中。按此范围完成 Traefik 适配器补充与必要检查，更新本文实际证据，保持 Implementation 和 Draft。
- 2026-10-02：用户要求只展示一个完成状态字段，失败原因在状态上 tip 展示；简化结果界面，保留 API/MCP 诊断数据、确认前操作清单、失败重试及草稿保存边界，补录本轮检查，保持 Implementation 和 Draft。
- 2026-10-02：用户要求前端循环逐条处理，每条响应立即调整记录，本轮不考虑页面/弹窗关闭后的任务管理；补录单条修订、逐行反馈、失败继续和仅重试失败项的实现及检查，保持 Implementation 和 Draft。

## Sync status presentation

- 本轮只修改 `RouteSyncDialog.vue`、对应组件/页面测试、中英文文案及相关文档。结果表只显示路由名称和完成/失败；失败原因沿用共享 `AppBadge` 的 `title` 提示。API 契约、发布/恢复流程和文件 writer 未改变。
- `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 均通过；`yarn --cwd web test src/components/RouteSyncDialog.test.ts src/views/route/routeSync.test.ts` 两个文件、11 项通过，覆盖状态提示、失败重试、列表/详情未保存草稿保留。旧页面断言曾依赖已移除的汇总段落，按新展示承诺修正后通过。
- 结果界面不显示配置/证书内部阶段、业务保存、文件提交、恢复/清理状态或请求/操作 ID；API 仍保留这些数据用于诊断与重试。发布、撤销、跳过继续在确认前的清单中展示。
- 本轮未进行真实浏览器视觉验收，未启动开发服务器或操作用户服务；此前环境验证边界仍适用。

## Sequential Web confirmations

- Web 预览一次，确认按原清单顺序逐条请求，每条响应立即更新记录和父页面已保存草稿；单个状态字段展示待处理/处理中/完成/失败，失败原因在状态提示中展示。成功记录在只重试失败项后保留；HTTP 预检错误或网络失败不伪造后端提交状态。
- 单条修订由同一 `syncPlan` 拆分计算，通过 DTO/proto/HTTP 映射返回；每条确认仍核对业务、自己的已发布文件和共享 Gateway/Environment 依赖。后端保留现有发布、恢复、锁和 API/MCP 批量能力，没有修改文件 writer 或 Traefik 平台重载分支。
- `task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`（0 issues）与 `go test ./cmd/... ./internal/...` 通过，部分 Go 包使用缓存。两个前端测试文件 14 项通过；延迟响应测试在 B/C 未返回前检查 A 已完成，失败后继续及逐条保存事件，并验证重试保留 A/C 成功记录。
- 相关 Route/HTTP/MCP 回归通过，新增后端测试覆盖 Project 清单的三种操作拆分确认、空草稿规范化，以及预览后编辑只拒绝对应 Route。检查中修复了重用单条计划后遗留的未使用 import；最终固定检查通过。
- 本轮未启动开发服务器、未运行新的真实浏览器或 SSH/Traefik 实机测试，没有操作用户服务；页面/弹窗关闭后的任务续跑按用户要求不扩展。

## Original preview table feedback

- 按用户要求在原预览表追加状态列，保留名称、操作、规则与证书方式。确认、顺序执行、失败重新预览和完成后始终保留同一模态窗、表格及已有行；成功项不因重试移除，失败项的新预览只更新原行内容。
- 同步执行期间禁用关闭操作并阻止 Escape 关闭，完成后由用户关闭；未增加后台任务或关闭后的续跑能力。
- `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 和两个同步相关测试文件均通过（14 项测试）。延迟响应与重试测试核对表格、行和模态窗 DOM 身份，验证原规则/操作仍显示、失败提示和最新预览修订提交。
- 本轮未启动开发服务器、未操作用户服务或进行真实浏览器视觉验收；后端契约未因本次表格调整改变。

## Acceptance feedback resolved

| 验收中发现的问题 | 当前处理与验证 |
| --- | --- |
| Windows 发布文件可读，但 watcher 未刷新，三条 Route 配置匹配超时 | 历史调查找到原有 SIGHUP；Windows local/SSH/DooD 在适配器内提交后重载再匹配，失败恢复使用同一流程。Windows 原生 Docker E2E 通过；Linux 仍使用 watcher，真实 SSH 留待后续 |
| 结果界面暴露多个内部阶段和操作 ID，用户无法直接判断是否完成 | 同一原预览表只保留一个状态字段，失败状态提示原因；API 保留诊断。逐行状态、权限/重载/清理提示、失败重试和未保存草稿回归通过 |
| 后端整批循环导致前端等待很久 | 每项返回独立预览修订，Web 顺序调用单条确认，每条响应立即更新原行和父页面草稿；失败继续，成功项不重复。延迟响应及重试保留 DOM 的组件测试通过 |
| HTTP/HTTPS 混在规则字符串中，证书方式仍位于规则列 | API 结构化返回 `protocol/match/target`；协议列展示入口 HTTP/HTTPS 及 `PEM`、`mkcert`、`Let's Encrypt · DNS-01/HTTP-01`。入口与上游协议相反的后端/前端回归通过，DNS 证书说明归协议列的断言通过 |
| Traefik 文件名、router/service 使用数据库 ID，难以阅读 | 动态文件和资源名使用编码，证书和内部归属仍使用 ID。改名撤旧、未同步改名撤销、三阶段失败恢复、pending 双路径恢复、编码占用和复用均有回归；真实 Windows E2E 验证旧文件消失 |
| 删除 `gateway/dynamic` 后，部署和同步均未重建 | 原因是缺失文件触发部署依赖预检和发布前完整性拒绝。Gateway 现允许准备空目录，部署后仍报告未恢复文件；重新预览并确认同步重建选定 YAML。失败恢复空基线，不使用过期 backup；外部改写仍拒绝。真实 Windows E2E 删除完整目录后完成 staging、Gateway 重建、显式同步及 TLS 核验 |
| 同步模态窗空间不足 | 最大宽度由 760px 扩至 960px，表格可见高度由 256px 扩至 384px；保留小屏宽度和共享弹窗高度上限。前端 lint/typecheck 通过，真实浏览器视觉验收仍未执行 |

本轮最终 `task check` 为 0 issues，完整 Go 回归通过，两个同步前端测试文件 16 项通过。第一次仓库检查发现新增错误字符串首字母大写（ST1005），修正后完整检查通过；该检查失败不记为通过结果。Windows E2E 耗时 27.41 秒，覆盖正常发布、改名、撤销、证书轮换、直接重启、force-recreate，以及删除动态目录后的恢复。

真实 SSH Linux/Windows、Windows DooD、真实远端权限失败及 Web 桌面/移动视觉验收仍未覆盖；模拟 target 分流与本机 Docker 证据不代替这些验收。现有文件被外部改写时仍保留冲突保护；本次不转换此前 ID 命名产物，不自动清理未知文件。未启动开发服务器或操作用户服务。用户要求检查问题已记录并提交代码，本文保持 Draft，不据提交行为宣称这些未验证项通过。

## Other staged changes

用户当前暂存区另包含远程目录备份脚本、测试及 Docker 部署指南，属于本次提交的附带变更，不纳入 Route 功能验收。`python -m unittest scripts.tests.test_manage` 执行 4 项，其中 3 项通过，1 项因当前 Windows 环境没有 POSIX `sh` 而跳过；实际归档内容测试不记为通过，未执行真实远端备份。
