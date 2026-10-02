# Route 文件发布、重启恢复与独立同步
最后修改时间: 2026-10-02 14:03:00

Review status: Accepted

Flow mode: standard

Current stage: Implementation

## Background

Orbit 当前通过 Traefik `providers.rest` 发布自定义路由。REST provider 只保存内存状态；Traefik 因容器自动重启、宿主机重启或外部操作重新启动后，已发布路由丢失，直到重新同步。

现有实现把完整 JSON 写入 Gateway Service 的 `.orbit/traefik-rest.json`，随后执行 REST PUT。落盘文件没有接入 Traefik 启动流程。Gateway 部署及 Orbit 发起的重启会在 worker 中再次发布，但无法覆盖 Traefik 自行重启。

用户已选择切换 File provider，并补充三项要求：不同路由拆开，发布互不影响；支持 SSH 远程环境及文件权限；支持 Orbit 容器运行和 Docker-outside-of-Docker (DooD)。本任务不处理 TCP listener 校验。

代码依据：

- `internal/infrastructure/external/traefik/route.go`：路由快照、证书文件与 Traefik API 适配。
- `internal/application/route/usecase/sync.go`、`sync_change.go`：预览、确认、hash 校验和数据库提交。
- `internal/application/gateway/usecase/initial.go`：新建 Gateway Version 的初始声明。
- `internal/application/deployment/usecase/gateway_enrichment.go`：Gateway 渲染及现有拓扑边界。
- `internal/infrastructure/runner/local/runtime.go`、`runner/ssh/runtime.go`、`runner/target/runtime.go`：目标端文件写入、替换、权限与 local/SSH 分流。
- `internal/infrastructure/storage/local/physical_data_root.go`：Orbit 容器路径到 Docker daemon 路径的解析。
- 活文档：`docs/architecture/cd-runtime.md`、`docs/guides/volume-mounting.md`、`docs/guides/deployment.md`、`docs/decisions/ledger.md`。

## Goal

- 由 File provider 持久化已发布路由；Traefik 重启后直接加载文件恢复服务，不依赖 Orbit 在线或人工同步。
- 按路由拆分发布资源，更新、禁用、删除或重试一条路由，不重写或清理其他路由的配置与证书。
- 延续保存配置与发布配置的分离。Traefik 自行重启只恢复已发布状态，不读取业务数据库，也不发布未同步的编辑或启停草稿。
- 服务重新部署与容器重建后保留已发布的自定义路由；发布资源不能因 Compose 重新生成而丢失，也不能借服务部署隐式发布未同步的 Route 修改。
- 保留同步预览、确认、过期检查、错误反馈及重试能力，使其与发布范围一致。
- 同步预览简化为发布/撤销/跳过清单，不比较 Traefik 运行时增删改或提交运行时 hash；确认仍核对业务配置、受管文件及 Gateway/Environment 依赖，发布后验证加载结果。
- 复用显式 Environment target runtime，覆盖 local、SSH 和 Orbit 容器运行场景。
- 明确文件所有者、读写权限、挂载路径和发布结果，避免文件写入成功但 Traefik 未加载时误报成功。
- Windows 文件提交后通过目标 runtime 向当前 Gateway 容器发送 SIGHUP，再核对加载结果；覆盖发布、撤销、失败恢复及中断恢复。local 模式分别处理 Windows/Linux 与 DooD 的实际挂载来源。

## Non-goal

- 不修复或扩展 TCP listener、端口和转发规则。已有协议渲染仅随 provider 切换保持原有行为。
- 不改变 Docker labels 的发现与发布方式。
- 不保留 REST 与 File 两套自定义路由发布逻辑，不增加发布方式开关或旧路径回退。
- 不转换旧 REST 快照或旧证书目录，不回填历史发布状态，不增加迁移 UI 或外部 Traefik 接管能力。现存受管 Gateway（包括迁移脚本初始化结果）在用户重新部署时直接应用当前 File provider 与受管挂载规则；重新部署和显式同步一次通过线下传达。
- 不投入极边缘场景或操作系统限制的专项修复与验证。Windows 文件替换的短暂缺失窗口沿用现有 target runtime，不新增 Windows 专用替换或 ACL 能力；正常热更新补回 SIGHUP 重载。
- Linux 继续依赖 File provider watcher，本次不增加 Linux 主动重载；后续通过真实远程 SSH Linux 环境验证是否存在文件通知问题。
- 不将 SSH 目标目录作为 Orbit 本机 Docker 的挂载来源；SSH 失败不回退 local。
- 不为完成发布而自动获取 root、使用 `chmod 777`、递归改写整个工作区权限或放宽私钥权限。
- 不主动部署、启动、停止或重启用户现有服务；后续运行验证可使用独立测试容器。
- 不扩展为任意用户自管 Traefik 动态文件的编辑器，也不修改已执行迁移。

## User scenarios

1. 用户确认同步路由 A 后，直接重启 Traefik。A 自动恢复；此时 Orbit 可以离线。
2. 用户编辑 A 但尚未同步，或在前端切换启停草稿。Traefik 重启后仍使用此前已发布的配置。
3. 用户发布 A，路由 B 继续使用 B 的已发布配置；B 尚未同步的编辑不随 A 发布。
4. A 发布、禁用、删除或重试时，只操作 A 拥有的配置和证书文件。B 的文件、权限和转发结果不变化。
5. 两条不同路由并发发布时互不覆盖。同一路由的过期预览、竞争更新和失败重试有明确结果。
6. SSH 环境中的文件由远端执行身份通过目标 runtime 写入，远端 Traefik 从远端 Docker 可见的目录读取；本机工作区不参与该环境的发布。
7. Orbit 运行在容器中并连接宿主机 Docker socket 时，文件写入 Orbit 可见的 logical path，Gateway bind mount 使用 Docker daemon 可见的 physical path。
8. 发布目录不可写、临时文件不可创建或提交、Traefik 无法遍历目录或读取配置/证书时，用户获得可定位的失败信息，不能收到无依据的成功提示。
9. 用户重新部署业务 Service 或 Gateway，容器被重建。已有自定义路由仍保留其已发布配置；未同步的路由编辑不随这次服务部署发布。

## Acceptance

### 持久化与发布范围

- 自定义路由由 File provider 加载；Traefik 管理 API 保留用于就绪检查、运行时查询及发布结果核验，不再作为路由发布入口。
- Gateway 详情的基本信息与 Traefik API 分为独立卡片及编辑入口；API 卡片保留容器/宿主机地址和就绪超时，名称统一为 Traefik API，各表单保存对应字段。
- 动态文件及 router/service 使用路由编码命名，例如 `route-mineru.yaml`、`route-mineru-route` 与 `route-mineru-service`；内部归属记录和证书版本目录仍使用稳定 ID。修改编码后发布须撤下旧编码文件和资源，失败恢复旧路径；撤销依据已发布编码，不遗留旧资源或影响其他路由。
- 单条发布仅更新选定路由的配置、证书和发布状态；无其他路由的隐式发布或全目录清理。
- 成功发布后，直接重启 Traefik 仍能访问原路由；修改、禁用、删除后再重启也不恢复旧资源。
- Traefik 自行重启不会把未发布的业务修改变成已发布配置。
- 业务 Service 与 Gateway 重新部署、容器重建后，不丢失已发布路由。File provider 的持久化目录与挂载须在重新部署后保留；部署流程不得隐式发布未同步的 Route 修改。
- 发布通过现有 target writer 提交完整 YAML，不将传输中的临时文件作为活动配置。失败时按该路由恢复上一份可用配置，并保持其他路由的文件不变；Windows 现有替换的短暂缺失窗口作为已知限制，不承诺所有平台均无瞬时切换。
- 预览与确认检查选定发布范围；不同路由的独立操作不能相互覆盖。批量操作的成功、失败和重试范围明确。
- Web 按预览冻结的顺序在前端逐条确认，每条返回后立即更新对应记录，单条失败后继续下一条，保留已成功项；API/MCP 的显式批量调用保留汇总编码和逐项结果。
- 文件已落盘、Traefik 已加载、数据库已保存是不同状态，接口保留各阶段诊断。界面只展示一个状态字段，执行时为待处理/处理中，返回后为完成/失败；失败原因通过状态提示查看，不显示内部阶段和请求/操作 ID。完成以逐项结果编码为依据，不能只凭文件落盘判断。
- 配置匹配与实际 TLS 证书验证在 API 结果中分别报告，界面采用单一完成状态，不表示证书已生效。路由 API 的规则匹配不能证明新 PEM 已生效；每次同步不强制要求 TLS 握手，证据不足时证书结果明确为未验证。
- 新建 Gateway 直接声明 File provider，允许先部署空受管目录，再显式预览并确认 Route 同步。受管动态目录丢失后，Gateway 部署可重新准备空目录；已发布文件缺失仍在部署后核验中报告，用户重新预览并确认同步可重建选定文件。部署不从业务草稿自动补建，外部改写仍报告冲突。
- 受管 Gateway 部署覆盖有效配置中的 REST/File provider 和 dynamic/certs/acme 挂载，写入静态受控文件并自动重建容器，使现存 REST Version 无需手工修改 YAML 即可部署；不写回 Version、不新增 UI，不自动发布业务 Route。
- 全量同步仅便利地解析当前业务及已知发布 ID；启用 Route 覆盖自己的文件，已删除且存在发布记录的 Route 撤销，禁用且从未发布的 Route 跳过。未知文件保留，不以运行时差异决定发布范围。

### 环境、路径与权限

- local 原生进程、local 容器 DooD、SSH Linux、SSH Windows 的路径处理沿用项目既有支持范围。
- 在 Orbit 容器内写入的 logical path 与 Docker bind mount 的 physical path 指向同一持久化目录；不通过字符串猜测宿主路径。
- Orbit 容器运行时，SSH 文件仍写入 SSH 目标，远端 Compose 仍使用远端路径，不套用 Orbit 本机的容器路径映射。
- 路由声明所需文件权限，写入及权限实现由现有 target runtime 负责；正常发布路径验证文件可写及 Traefik 可读取，实际权限失败明确报告。沿用 Linux Mode 与 Windows 既有 ACL，不扩展 UID/GID 或 ACL 管理。
- 权限失败的诊断包含操作、目标环境和必要路径上下文，不包含证书私钥、SSH 凭证或 token。
- Gateway 只读挂载动态配置目录，发布进程在对应可写侧提交文件；目录整体挂载使文件替换后的更新可被发现。
- Windows 原生目录、Windows bind mount 的 local DooD 与 SSH Windows 提交后通过同一 Traefik 适配器重载；local Linux 原生目录、Linux DooD 原生目录及 SSH Linux 不主动发送 HUP。信号失败明确报告，文件恢复后须重新加载并匹配才能报告已恢复。
- 临时文件和备份文件不被 File provider 当作配置加载；正常提交、失败清理及重试不留下可被加载的中间态。

### 验证范围

- 增加直接重启 Traefik 的最小有效回归，不能仅验证 Orbit 发起的部署/重启 hook。
- 验证两条已发布路由：单条更新、禁用、删除、失败和重试不影响另一条。
- 对实际支持的 local/SSH 正常发布路径与实际权限失败补充最小验证；真实远端或容器环境未验证时明确标注。验证 Windows 正常发布的 HUP 重载，不安排文件替换窗口或 ACL 的专项验证。
- DooD 验证使用不同的 Orbit 容器路径和宿主机路径，证明实际挂载来源正确，不能只测试路径同名的场景。
- 实现后按仓库要求执行 `task check` 和 `go test ./cmd/... ./internal/...`；若修改前端，还执行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。

## Decisions

- 用户已选择 File provider，保留现有预览后确认同步的交互原则。
- 用户要求标准模式，先记录，再评审；现已接受 Intent 并要求写入 Plan，当前阶段为 Plan。
- 用户明确要求不同路由拆开，发布互不影响。
- 用户已确认“每条路由独立文件，并支持单条同步”；详情页单条同步不得隐式发布其他路由。
- 用户明确要求考虑 SSH 远程环境、文件权限、Orbit 容器运行和 DooD。
- 用户明确排除 TCP 问题。
- 路由、证书、临时文件和删除操作须有明确的路由归属，不再采用无发布范围的全目录清理。
- 用户最初采纳按 Route ID 分文件的结构，随后要求动态文件名及内容中的资源名使用路由编码；内部归属记录、证书目录继续按 Route ID 和证书版本组织。动态配置与手工证书目录只读挂载，ACME 存储由 Traefik 读写管理。具体结构写入同名 Plan。
- 用户最初明确“不处理历史遗留环境”，随后明确现存受管 Gateway 在部署时直接覆盖受管配置，通过线下传达重新部署及显式同步步骤；不转换历史发布数据、不保留 REST 发布逻辑或专用迁移流程。
- 用户选择配置匹配与证书验证分别报告；实际证书验证不作为每次配置同步的强制前提。
- 用户明确批量“顺序处理，跳过失败，有未完成的情况报告在响应里携带编码前端报告”；失败恢复只作用于对应 Route，不回滚整个批量。
- 复用既有 controlled_file、WorkspaceFile/SyncFiles 与 target writer。Version 保留镜像、端口、Docker provider 和 resolver 等声明；enrichment 在有效部署计划中覆盖 Route 所需 provider 和目录挂载，不写回库存。
- 路由业务只声明文件内容、路径及所需 Mode；临时文件权限时序和替换细节归通用 writer，不设为路由专属业务职责或验收条件。
- 用户支持复用现有 writer 的“完整写入临时文件，再替换活动文件”方式；不改为路由专属的直接覆盖内容逻辑。
- 用户已选择本次改为发布/撤销清单，移除 Traefik 增删改差异、`matched` 预览字段与 `traefik_hash`，保留配置及证书独立报告。
- 用户明确极边缘场景及操作系统限制只记为已知问题，不投入精力。Windows 沿用现有替换与权限机制；随后用户要求补回已有 SIGHUP 重载能力，Linux 主动重载暂不实现，后续通过远程 SSH 实测。
- 用户要求同步执行结果只展示一个完成状态字段，失败时在状态提示中展示原因；内部阶段、证书观测结果和诊断 ID 保留在 API 中。确认前仍展示发布/撤销/跳过操作清单。
- 用户要求 Web 在前端按预览顺序循环调用单条确认，每条响应立即更新对应记录，失败继续下一条；不等待整批响应才显示结果。本轮不扩展页面关闭、模态窗关闭后的任务续跑或恢复能力。
- 用户要求 HTTP/HTTPS 在同步清单中独立成列；预览规则直接返回结构化的入口协议，不从匹配表达式或上游 URL 解析协议。
- 用户要求 Traefik 路由文件名及 router/service 使用编码，提升可读性；发布适配器统一处理改名、撤旧、归属冲突及失败/中断恢复，不改变业务 ID。
- 用户要求 `Let's Encrypt · DNS-01` 等证书方式归入协议列，规则列只包含匹配条件与上游目标。
- 用户发现删除 `gateway/dynamic` 后部署及同步均被拦截；允许 Gateway 准备空目录并通过重新预览、确认同步重建选定文件，保留部署后缺失诊断和外部改写保护。
- 用户要求同步模态窗扩大；沿用共享 AppDialog，将最大宽度扩大至 960px、表格最大可见高度扩大至 384px，保留小屏边距与滚动约束。

## Open questions

- 暂无需要用户选择的方向性问题。批量响应具体字段与证书观测方式在 Plan 中落实；平台文件写入复用现有实现，已知 OS 限制不再作为待收敛问题。
- 正常文件写入和读取以仓库当前 Gateway 镜像及执行身份为基准；任意 rootless/userns、UID/GID 或 Windows ACL 组合不在本次扩展范围。

以上技术验证事项不表示已经接受当前代码草稿或整个 Plan。

## Risks and assumptions

- File provider 合并文件后仍共享 Traefik 的 router/service 命名空间和 TLS store。分文件不等于所有错误天然隔离；名称冲突、坏配置、证书冲突和首次启动读取失败需要在发布流程中控制。
- 现有证书同步使用整个 cert 目录的 `.pem` 清理；直接复用会删除未包含在单条请求中的其他路由证书。
- 挂载整个目录，使正常文件替换可由 File provider 观察。Windows 的文件通知可能无法传入容器，通过 SIGHUP 补充正常加载；SSH Windows 替换的短暂缺失窗口仍为已知限制，不专项处理。
- 文件写入成功不保证 Version 声明及实际挂载符合要求，也不保证运行时配置匹配；发布结果必须有明确依据。
- 目录 mode、umask、现有所有者、Docker UID/GID 映射及 Windows ACL 都可能影响访问。不能把一次 SSH Probe 成功等同于发布目录及容器读取权限就绪。
- 数据库提交与文件提交没有跨系统原子事务；失败重试及每条路由已发布状态应能解释这个边界。
- local 原子写入草稿修改了共享 workspace 文件写入实现，影响面大于 Route；Plan 应评估收窄范围或补齐共享行为验证。
- Gateway deploy/restart 现有 hook 会重新发布已保存的配置，需要改为检查已发布文件及运行时匹配，避免部署触发未同步业务编辑。
- 用户曾讨论 Docker label 替代方案，现已采纳 File provider 目录结构。普通 Docker 容器的 label 变更须重建承载容器；附着业务容器会耦合路由同步与 Service 部署，同一容器上的多条路由共同受到重建影响。本任务继续采用 File provider。

## Review findings

1. 根因成立。现有 REST provider 没有启动时加载已落盘快照的机制；切换 File provider 可以消除自行重启后的内存状态丢失。
2. 现有单文件全量发布草稿不符合新增隔离要求。应调整为按路由拥有文件及发布范围，不能以拆文件但仍读取并重写全部业务路由代替隔离。
3. 证书清理边界必须调整。配置文件拆分后仍全目录 prune 证书，会破坏其他路由。
4. 原评审要求 Gateway enrichment 仅校验 Version provider/mount；用户于 2026-10-02 明确部署直接覆盖受管部分，更新该边界，其他 Version 拓扑继续保留。
5. 当前草稿只有文件写入成功，没有 Traefik 加载结果依据；旧 Gateway 未重新部署或权限不可读时可能误报发布成功。
6. 原有 local/SSH target runtime 和 DooD resolver 可以复用；正常发布仍需证明目录映射正确及文件可读写。Windows 热更新补回原有 SIGHUP 重载，替换与 ACL 平台限制不纳入专项修复。

评审结论：用户已采纳目录结构并要求写入 Plan，Intent 标记为 Accepted。发布隔离、Gateway 配置所有权与发布成功边界写入 [Plan 草稿](../plan/20261001-route-file-provider-persistence.md)；此前代码草稿仍须按 Plan 调整，不视为已接受实现。

## User review notes

- 2026-10-01：用户指出 Traefik 重启后 REST 路由全部失效，需要人工同步。
- 2026-10-01：用户选择“改用 File provider”。
- 2026-10-01：用户要求 `$specflow:specflow` 标准模式，“先记录，再评审”。
- 2026-10-01：用户补充路由隔离、SSH 远程环境、文件权限、Orbit 容器运行及 DooD 要求。
- 2026-10-01：用户确认“每条路由独立文件，并支持单条同步”。
- 2026-10-01：用户讨论内存恢复、其他 provider 与 Docker label 的可行性，尚未改选发布方式；补充服务发布时必须保留自定义路由，避免容器重建后丢失配置。
- 2026-10-01：用户采纳展示的 File provider 目录结构，要求“写入 plan”；按标准模式将 Intent 标记为 Accepted，进入 Plan，Plan 保持 Draft 待评审。
- 2026-10-01：用户提醒已有受控文件及权限机制，并要求在对话中解释、讨论不确定事项；Plan 按实际 writer 能力记录权限和平台边界。
- 2026-10-01：用户明确不处理历史遗留环境，选择配置匹配与证书验证分别报告，并确定批量顺序执行、失败后继续、响应编码由前端报告。同步更新 Intent 与 Plan，不进入实现。
- 2026-10-01：用户质疑临时私钥权限时序是否属于路由职责；明确其归通用 writer。用户进一步明确极边缘场景及操作系统限制作为已知问题、不投入精力，移除 Windows 专用替换、watcher/ACL 专项验证及相关交付阻塞项。
- 2026-10-01：用户支持“写临时文件再替换”，并询问剩余不明确事项；核对后暂无需要用户再选择的方向性问题，批量预算、证书观测入口及正常 SSH/DooD 发布证据在实现中落实。该确认针对文件提交方式，Plan 仍为 Draft。
- 2026-10-01：用户要求“开始实施”；Plan 接受，开始按该计划调整现有草稿，当前阶段为 Implementation。
- 2026-10-01：用户指出独立文件使原全量增删改对比不再必要，并选择“本次改为发布/撤销清单”；按该决策修订 Plan 和契约，不再比较 Traefik 运行时快照。
- 2026-10-02：实现结果和风险已汇报；用户要求“补验证文档”，进入 Verification，沿用已完成的检查与真实 Linux DooD 证据，未覆盖环境如实记录。
- 2026-10-02：用户澄清预置指迁移脚本初始化的 Orbit Gateway，并明确“部署的时候直接覆盖”，要求重新部署通过线下传达、不增加 UI；修订有效部署计划的 provider/mount 归属，回到 Implementation 补齐该行为，不修改已执行迁移或转换历史路由快照。
- 2026-10-02：历史调查确认切换 REST 的 e264e6cc2 删除了 Windows SIGHUP 重载；用户要求补回该能力，提醒 local 同时支持 Windows/Linux，并要求变更集中、保持架构边界。Linux 主动重载暂不实现，远程 SSH Linux 留待后续实测。
- 2026-10-02：用户要求只展示一个完成状态字段，失败时在状态上 tip 展示原因；据此简化结果界面，保留确认前操作清单及 API 诊断字段，不改变发布和恢复语义。
- 2026-10-02：用户要求前端逐条循环处理，每处理一条立即反馈并更新对应记录；修订当前 Plan 后继续 Implementation。确认仍使用原预览的单条修订，不自动预览并发布尚未确认的新内容。

- 2026-10-02：用户确认保留 API 地址及就绪超时，采纳界面改称 Traefik API，并要求从基本信息拆出独立卡片；在当前 Implementation 中调整共享文案、详情展示及分区编辑。

## Existing workspace draft

进入 SpecFlow 前已经写入的 Go 改动保留为未评审草稿，包含单文件发布、provider 识别、新建 Gateway 模板、渲染时安装 provider/挂载、local 文件替换和测试草稿。

进入 SpecFlow 后，原草稿曾暂停修改与测试，未作为已接受实现。用户要求开始实施后，草稿按已接受 Plan 调整并运行必要检查；此前原有 REST 路由测试不作为本任务的验证结果。用户随后要求补 Verification 文档，又明确部署直接覆盖受管配置；当前处于补充 Implementation，验收文档保留 Draft。
