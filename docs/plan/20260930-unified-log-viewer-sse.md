# CI/CD 日志统一与 SSE 流式传递实施计划
最后修改时间: 2026-09-30 16:33:56

Review status: Accepted

Flow mode: standard

## Intent basis

依据 [已接受的意图](../intent/20260930-unified-log-viewer-sse.md)。用户已确认采用 Fetch + SSE，要求所有范围内日志在产生过程中持续读取，同时统一功能、样式与组件；部署中的容器日志不能等待任务完成才读取，日志控件移除复制能力。

已采纳的工程默认方案：持续接收与滚动跟随分离、关闭视图停止订阅但保留页面内阅读状态、重开补读、自动退避重连、有界展示缓存、任务文件日志与容器日志分别处理结束条件。本次不新增导出、日志存储平台或消息中间件；终端 WebSocket 与部署对话功能保持各自用途。

用户已接受计划、要求开始实施并进入验收；当前为 Verification，standard 模式不另建 Spec。

## Architecture and boundaries

| 层 | 职责 |
| --- | --- |
| `web/src/components/LogView.vue`、`LogDrawer.vue`、`LogActions.vue` | 共享展示、标题/页签栏工具、主题、查找、跟随与历史边界提示；不承载异常展示或控制远端命令 |
| `web/src/composables/useLogStream.ts` 与页面内缓存 | 接收状态、游标、重连、批量更新、缓存与视图生命周期 |
| Web API 与 SSE 工具 | 依据资源构造请求，沿用 Bearer 与现有错误归一化，解码流事件 |
| `internal/api/http/handler/` 与 `transport/` | 请求认证、参数映射、协议转换、SSE 编码、Flush 和 HTTP/流内错误响应 |
| `application/pipeline_run`、`application/deployment` | Project/资源授权、执行目标与日志来源定位、任务状态、容器等待和重建处理 |
| `application/logstream` | 仅承载跨 CI/CD 复用的事件 DTO、窄读取会话端口与文件持续读取机制；不集中管理 Project 或部署业务 |
| `infrastructure/storage/local`、`runner/local`、`runner/ssh`、`runner/target` | 有界文件/SFTP 读取、持续命令输出、连接取消与按显式目标分派 |
| `bootstrap` | 注入上述实现；不承载日志业务规则 |

依赖仍为 HTTP/前端适配 -> application -> ports/model；application 不引用 Gin、SFTP、具体 runner 或生成 DTO。生成协议仅在 HTTP 与前端边界使用。已有文件日志继续作为历史依据，读取消费者不接入任务写入路径，不依赖 API 与 worker 共享内存事件总线。

## Interfaces and defaults

### HTTP 入口

统一为带 Bearer 认证的 GET SSE 请求，路径使用单数资源：

| 资源 | 路径 | 定位参数 |
| --- | --- | --- |
| CI 阶段文件日志 | `/api/pipeline-run/:run_id/stage/:stage_run_id/log/stream` | `project_id`、可选 `cursor` |
| CD 操作文件日志 | `/api/deployment/:deployment_id/log/stream` | `project_id`、可选 `cursor` |
| 部署详情容器日志 | `/api/deployment/:deployment_id/container-log/stream` | `project_id`、可选 `cursor`；时间范围由部署记录决定 |
| Service/Gateway/Route 容器日志 | `/api/application/:app_id/log/stream` | `project_id`、显式 `service_id`、可选 `component` 与 `cursor`；首次默认最近 200 行 |

- 沿用现有 CurrentUser 与 Project 成员授权，并校验 Stage 属于 Run、Service 属于 Application、Component 属于该运行绑定。浏览器不传日志路径、宿主命令或 SSH 身份。
- 请求开始时捕获资源与目标修订。CI 沿用 Run 的目标快照，部署详情容器日志核对 Deployment 的目标快照，持续运行的容器日志绑定当前 Service/Environment。CD 操作文件位于控制面，读取历史文件不额外要求远端在线。
- 首次鉴权、参数与资源校验失败使用现有 HTTP 错误契约；流开始后使用 `error` 事件。GET 已由现有事务中间件排除，无需重排路由或改变事务机制。
- 所有 Web 调用迁入 SSE 后，删除无调用的旧日志轮询 API 方法、HTTP handler/路由和专用响应 DTO。MCP 实际使用的 `DeploymentLog`、`RuntimeComposeLogs` 等按需读取应用服务保留；它们是独立入站用途，不保留旧 Web 轮询路径作为兜底。

### SSE 事件契约

新增 `proto/orbit/v1/common/log_stream.proto`，集中定义日志流事件；application 使用独立 DTO，由 handler 映射后按现有 proto JSON 编码。

| 类型 | 含义 |
| --- | --- |
| `ready` | 已完成授权并建立订阅，返回来源标识和当前资源状态；此时可以尚无日志输出 |
| `chunk` | 原始日志字节与接收后的续读游标 |
| `waiting` | 等待任务日志文件、Compose 配置或容器输出可用；订阅保持，后续自动开始读取 |
| `source_changed` | 容器重建导致输出来源变化；保留已显示历史并重建该来源的解码/续读状态 |
| `gap` | 来源已丢失或重建、重连后不能保证补齐历史，明确提示恢复边界 |
| `complete` | 文件日志终态，最后输出已传递；包含任务最终状态，业务失败不等于日志读取失败 |
| `error` | 读取/授权错误，包含现有错误分类、HTTP 状态、可显示信息、request ID 和可重试标志 |

事件包含 `type`、`source_id`、`cursor`、`data_base64`、`status`、`message`、`code`、`request_id`、`http_status`、`retryable`，按事件类型校验所需字段。日志数据用显式 Base64 字符串保留原始字节，前端对同一来源使用持续的 TextDecoder，避免文件字节游标与跨块 UTF-8 混淆；当前 TS 生成配置只输出类型，不把 proto bytes 的 JSON 字符串误用为 Uint8Array。

SSE 使用标准 `data:` JSON 事件与注释心跳，LF/CRLF、跨网络块、多行 data 和连续事件均正确解析。`cursor` 是与已授权资源、来源及修订绑定的续读位置，不是文件路径或授权凭证；由日志源验证。文件内部使用 int64 字节位置，传输游标使用字符串，避免旧 int32 及 JavaScript 数字精度约束。

响应使用 `text/event-stream`、`Cache-Control: no-cache`、`X-Accel-Buffering: no`，首次握手、事件与心跳均及时 Flush。常规请求日志仅对新增日志 SSE 路由关闭响应正文捕获，保留请求元信息；不改造日志框架或记录持续输出副本。

### 初始工程参数

| 参数 | 初始值与约束 |
| --- | --- |
| 文件读取批次 | 每批最多 32 KiB；没有新增字节时等待 250 ms，继续使用同一读取会话 |
| 资源/权限复核 | 每 2 秒复核成员资格及相关目标/凭据修订；确定性失效终止，不切到新目标 |
| SSE 心跳 | 15 秒；前端 45 秒没有任何流活动则重新连接 |
| 连接建立等待 | 15 秒；握手成功后使用活动检测，不设固定总请求时限 |
| 自动重连 | 1、2、4、8 秒后增加到最多 15 秒，加入小幅抖动后仍限制在 15 秒内；收到输出或正常 waiting 后重置退避，ready 握手不单独重置 |
| 前端展示更新 | 最多每 100 ms 合并一批，任务结束时立即提交最后一批 |
| 单资源展示缓存 | 最多 50,000 行或 10 MiB UTF-8 内容，以先达到者为准；按行淘汰，超长单行也受字节边界约束 |
| 页面关闭资源缓存 | 最多保留 4 个最近资源的有界缓冲，采用 LRU；离页、换 Project 或目标修订变化清理 |
| SSE 写入 | 单次写入有截止时间，慢连接中断后可续读；不让读取消费者拖住任务写日志 |

数值集中为具名常量，不引入业务层环境变量或新的配置兼容键。文件终态后使用三秒无新增窗口并最终复查 EOF；执行器取消观测间隔限制在一秒内，普通 Stage 完成前关闭写入器。只有相关大日志或平台验证暴露具体风险时调整这些参数并记录依据。

## Implementation steps

### 1. 建立共同契约与窄读取端口

- 创建公共应用事件 DTO、可取消且可关闭的读取会话端口；文件会话支持按 offset/limit 有界读取，临时 EOF 不代表任务结束。
- 在现有 pipeline/deployment 日志端口中接入持续读取会话，保留写入和 MCP 实际需要的短读职责；生产适配器和已有测试替身同步实现，不使用类型断言回退旧接口。
- 定义 SSE 响应、错误分类与资源游标验证，完成 proto 生成。通用 transport helper 只处理协议编码、写入与 Flush，不包含任务状态或目标选择。
- 记录任务状态与连接状态的区分：失败/取消任务仍能完整读日志并正常 `complete`；读取异常统一通过 Toast 提示，短暂故障自动重连。

### 2. CI 与 CD 文件日志流

- 在 `pipeline_run/usecase` 和 `deployment/usecase` 增加资源级日志订阅用例，复用现有成员校验、Stage/Run 归属、Service 定位及目标快照校验。
- local 文件持续会话按限额增量读取；文件尚未创建时等待并发出 `waiting`，不在读取时创建文件或目录。
- SSH CI 订阅期间复用 SSH/SFTP 连接；Linux 可持续读文件，Windows 复用 SFTP 但短时打开文件，沿用现有共享冲突处理，避免长期文件句柄阻挡任务追加写入。
- 任务终态后继续读取到最后 EOF 再发送 `complete`，空文件也正常完成；任务取消可能早于执行器完全退出，需核对实际写入/退出次序并验证最后输出，不以首次终态快照直接结束。
- 请求取消、页面关闭、权限/目标变化、读取故障和服务进程退出均关闭读取会话；文件历史保持现有归属，SSH 不回退本地。

### 3. local/SSH 容器持续输出

- 在 `deployment/port` 增加专用持续输出能力，由 target dispatcher 分派到 local/SSH。复用现有 Compose 参数和平台命令构造，使用非 PTY 的 `docker compose logs --follow --timestamps --no-color`，不复用会缓存完整结果的 `Query` 或记录 Running 命令的部署 `Run`。
- stdout/stderr 通过有界读取直接交给订阅，保留 Compose 来源前缀；命令诊断与正常日志字节分别处理，禁止 `bytes.Buffer` 无界收集整次跟随输出。
- 订阅从本次部署开始时间或首次最近 200 行进入持续输出。缺少配置/容器时保持 `waiting`，按当前资源重新定位；普通等待不作为读取失败重试。
- 通过现有容器观测能力跟踪容器 ID 变化与跟随进程退出，重建后刷新来源继续订阅。订阅的结束规则绑定视图和来源有效性，不绑定一次 Deployment 的终态；stopped/faulted Service 的已有容器历史仍可读取。
- 容器恢复采用来源标识、Docker 时间戳及短重叠窗口补读，使用有界记录计数去重以保留相同文本的真实重复输出。无法补齐时提示缺口，不将容器恢复承诺为文件日志同等的精确字节回放。
- 在真实 local/Linux SSH/Windows SSH 上核验关闭读取会释放日志读取进程/session/client，且不停止容器或部署任务；如 Windows OpenSSH 默认关闭不足以结束读取子进程，针对该日志读取命令补充自身清理。

### 4. 前端 SSE 接收与页面内缓存

- 新增日志 API 和 `web/src/utils/event-stream.ts`，参考现有部署对话的增量解码与错误处理；保留对话业务代码，不迁移其固定超时或专属消息解析。
- Fetch 带现有 Bearer、Accept 与 AbortSignal，使用现有 `ApiError`、`toApiError` 和未授权处理；支持注释心跳并限制事件缓冲，区分用户主动 abort 与网络异常。
- `useLogStream` 管理连接、游标、来源解码器、重连和有界缓冲。仅在完整事件已接收并提交缓冲后推进游标；同一来源跨重连保留解码器中的未完成字符，来源切换时显式处理边界。
- 短暂网络/服务不可用自动退避重连；401/403、资源不存在、目标变化及契约不匹配停止重试。读取异常由订阅层接入项目共享 Toast，连续失败去重，不在日志控件显示异常；流内 401 也沿用现有登录失效处理。
- 页面持有按 Project/日志类型/资源/Component 定位的缓存，避免 Gateway 关闭后清空 target、组件卸载导致丢失缓冲。复开恢复内容、游标、跟随状态与阅读位置；旧 generation 的事件不能进入新视图。
- 日志订阅与 Run/Deployment 详情状态刷新分开，继续保留业务详情必需的状态刷新，移除日志自身的 HTTP 轮询；页级“暂停详情刷新”不影响日志接收。

### 5. 统一阅读控件与全部入口

- 将 `ContainerLogView.vue` 收敛为通用 `LogView.vue`；新增 `LogDrawer.vue`，复用 `AppDrawer`。`RuntimeContainerLogsDrawer` 作为资源适配层接入共享组件，移除其旧定时刷新、完成任务停刷和重复 footer 操作。
- 共用 LogActions，把查找和跟随放在抽屉标题栏或 CD 日志卡片标题栏；正文移除独立状态/工具行，不显示“读取完成”。日志控件不承载异常展示，历史截断/缺口使用信息图标提示；查找框悬浮于日志内容、不增加顶部占位。按钮使用项目图标与 Tooltip，不新增复制、导出或暂停网络接收的操作。复制移除只作用于日志控件，避免影响脚本/文件编辑器。
- 沿用现有 Monaco 主题、12px 等宽字体、行号和自动布局。首次挂载或重开恢复有界快照；平时用 Monaco model 增量编辑追加并裁剪头部，避免每个 chunk 用 `modelValue` 替换完整文档。需要日志专属行为时放在日志组件内，不改变通用编辑器其他用途。
- 监听用户滚动，离开底部暂停跟随，回到底部或主动恢复后继续跟随；跟随按钮可切换，跟随/停止跟随使用向下贴底/暂停图标。裁剪时尽量保留可见行位置，被淘汰的阅读位置明确显示截断状态。
- `PipelineRunDetail` 阶段抽屉与 `DeploymentDetail` 的操作日志卡片、容器日志抽屉都使用共享阅读能力。部署详情操作日志收进 DetailInfoCard，卡片右上角日志按钮打开容器日志的 LogDrawer；操作日志始终保留在卡片中，容器抽屉开关不影响其组件或订阅。容器日志在抽屉打开时读取并在关闭后保留内容、游标、跟随状态和阅读位置；部署进行中即可订阅，不等待操作完成，保留停止操作不展示容器日志规则。
- Service/Gateway/Route 共用运行时日志适配，Route 继续读取关联 Gateway；统一中英文文案、抽屉宽度、间距与错误样式。已显示日志不因错误而消失。

### 6. 清理与活文档校准

- 删除被替代的日志轮询、独立状态/滚动逻辑和无调用 Web 快照接口/协议，旧实现不作为运行时兜底并存。
- 将 `RuntimeContainerLogsDrawer.test.ts` 中“关联部署完成即停止刷新”的断言更新为当前承诺：容器日志继续读取；保持授权、取消和资源切换等成熟覆盖。
- 更新 CI 指南、CD 运行时、前端架构/样式指南和决策账本中的日志阅读与流式规则，记录本次跟随输出行为；旧过程文档保持历史记录。
- 固定检查属于实施质量门禁，Implementation 完成后先汇报 diff、已运行检查与剩余实机风险，等待用户要求进入 Verification 再创建验收文档。

## Files to change

| 范围 | 预期文件 |
| --- | --- |
| 公共契约 | 新增 `proto/orbit/v1/common/log_stream.proto`；按调用清理 `proto/orbit/v1/pipeline_run/pipeline_run.proto`、`deployment/deployment.proto`、`application/application.proto` 中旧日志响应；生成 `internal/gen/proto/`、`web/src/gen/proto/` |
| 公共流式机制 | 新增 `internal/application/logstream/{dto,port,usecase}/` 与 `internal/api/http/transport/log_stream.go` 及对应测试；只抽取实际复用部分 |
| CI/CD 用例 | `internal/application/pipeline_run/port/port.go`、`usecase/service.go` 与新增 `log_stream.go`；`internal/application/deployment/port/port.go`、`usecase/service.go`、`runtime_query.go` 与新增 `log_stream.go`；必要时核对执行器末尾写入顺序 |
| 数据源 | `internal/infrastructure/storage/local/executionlog/`、`internal/infrastructure/runner/ssh/pipeline.go`；local/ssh/target 下新增容器日志持续读取适配与测试 |
| HTTP 与组装 | `internal/api/http/handler/pipeline_run/`、`handler/deployment/`、`routes/pipeline_run.go`、`routes/deployment.go`、`routes/application.go`、`internal/bootstrap/http.go`；`middleware/logging.go` 仅追加 SSE 路由正文捕获例外 |
| 前端接收 | 新增 `web/src/utils/event-stream.ts`、`web/src/composables/useLogStream.ts`、日志缓冲辅助模块与对应测试；更新 `api/pipeline_run/`、`api/deployment/`、`api/application/`，按实际复用决定是否集中日志 API |
| 前端组件 | 新增 `LogView.vue`、`LogDrawer.vue`；移除被替代的 `ContainerLogView.vue`；改造 `RuntimeContainerLogsDrawer.vue`、`runtimeContainerLogs.ts` 及测试 |
| 页面与文案 | `PipelineRunDetail.vue`、`DeploymentDetail.vue`、`ServiceDetail.vue`、`GatewayDetail.vue`、`RouteDetail.vue`；`web/src/i18n/locales/zh-CN.ts`、`en-US.ts` |
| 活文档 | `docs/guides/ci-pipeline-design.md`、`docs/architecture/cd-runtime.md`、`docs/frontend/architecture.md`、`docs/frontend/style-guide.md`、`docs/decisions/ledger.md` |

预计不修改数据库迁移、任务队列、SSH 终端、部署对话或项目依赖声明。若接口扩展影响现有测试替身，仅按新端口机械更新；任何超出上述边界的必要变更先说明原因并同步计划。

## Verification plan

### 固定检查

实施时按仓库入口运行：

```text
task proto
yarn --cwd web lint:fix
yarn --cwd web typecheck
task check
go test ./cmd/... ./internal/...
```

通过 `yarn --cwd web test <相关测试文件>` 运行新增 SSE 客户端、日志缓冲、共享组件和改造抽屉测试；测试路径以实际新增文件为准。

### 最小有效行为验证

1. **文件流正向流程**：任务尚未完成时收到输出，文件创建前保持等待，历史补读后增量追加，跨块中文正确，重连按游标恢复，终态/取消后的最后输出被读完。
2. **真实 HTTP SSE**：首次事件及时到达、心跳 Flush、主动断开取消读取；流前授权错误符合 HTTP 契约，流内读取错误具有 request ID 与恢复分类，已有内容保留。
3. **容器持续输出**：部署进行中、容器未出现、出现后输出、重建后继续、部署结束不停流；关闭订阅不停止任务或容器。使用可控 runner 验证主要编排，再在可用环境做平台集成。
4. **授权与目标边界**：非成员拒绝、资源归属正确；活跃订阅遇成员撤销/目标或凭据修订变化终止，SSH 不读控制面替代文件。
5. **阅读与生命周期**：所有入口复用共享控件；上滚仍接收输出、恢复跟随、关闭/重开保留内容与游标、Gateway target 清空后仍恢复、Project 切换无旧事件污染；错误不遮住历史。
6. **性能边界**：以超过缓存上限的日志验证有界缓冲、批量模型追加、截断提示和重开恢复；不以每条消息重建完整 Monaco 文档。对剩余具体并发风险补充 race 检查，不无目的扩大测试矩阵。
7. **浏览器与平台验收**：使用用户已运行的服务检查桌面/窄屏、明暗主题、抽屉和内嵌页；验证真实 local、Linux SSH、Windows SSH 的持续读取和释放进程。按规则不主动启动、停止或重启开发服务器。

Verification 阶段对照 Accepted Intent、本计划与实际 diff 核验；无浏览器会话或实机目标时明确记录验证缺口，不能以替身测试冒充平台验收。

## Blockers

无已知代码级阻塞。真实 Linux/Windows SSH 目标与可用浏览器会话是对应集成验收的外部条件；可以先完成实现和可控测试，不将其作为当前需要用户决定的业务问题。

## Assumptions

- 当前日志写入和任务状态存储保持既有位置与语义；读取不引入新存储，也不依赖 worker 与 API 同进程才可回放。
- Web 日志快照接口的当前调用者均在本仓库，迁移后清理无调用接口；MCP 按需读取应用服务仍有实际调用。
- CI 容器输出按既有 UTF-8 字节内容读取，SSE 保留原始字节；Windows 路径与文件共享规则沿用既有 SSH 适配。
- 继续使用当前依赖与工具链，预计不新增依赖；确有必要时选择主流维护版本，说明引入理由并更新锁文件。

## Risks

- 文件任务取消状态与执行器实际退出存在时间差；必须验证最后输出读取，避免终态先到导致漏日志。
- Windows SFTP 的共享锁及 OpenSSH 关闭行为需要平台验证；持续会话不能阻挡写入，也不能留下日志读取子进程。
- 容器日志恢复没有文件 offset 的精确性，时间重叠去重需区分重复回放与真实重复行；Docker 已删除的历史不可恢复。
- 目标修订与权限复核使用有界间隔，旧订阅不能悄悄接到新目标；前端游标和缓存需与来源绑定。
- 文件大历史回放、页面多资源缓存及 Monaco 更新可放大内存成本，参数需由实际大日志验证支撑。
- 生产代理可能缓冲或中止 SSE；本地 handler 测试不能代替生产链路验证，部署环境需核验心跳与及时输出。

## Rollback

本次不更改持久化日志格式、数据库或任务执行输入。需要回退时，通过正常回退本次代码与生成契约变更恢复原日志查看实现；关闭服务时释放 SSE 与读取连接，既有日志文件和 CI/CD 任务历史仍可读取。运行时不引入新旧日志实现切换开关或轮询兜底。

## User review notes

- 用户明确要求 Fetch + SSE、全部日志执行中流式读取、移除复制，统一功能/样式/组件。
- 用户要求依据主流实践和项目现状作出工程选择，已明确“采纳建议，开始计划”；Intent 已更新为 Accepted。
- 用户明确要求“开始实施”，本计划更新为 Accepted，进入标准模式 / standard 的实施 / Implementation。
- 实施中发现 Deployment StartedAt 在任务启动时更新，因此容器游标的资源范围绑定 Deployment ID/目标修订，不把可变 StartedAt 纳入来源身份；StartedAt 继续作为首次读取的时间范围。
- 为使取消后的最后输出读取窗口覆盖执行器观测时机，CI/CD 取消状态轮询上限调整为一秒；不改变队列轮询配置。
- 用户要求进入 Verification，并明确自行进行界面测试、不再打开浏览器。验证中复现了 ready 后读取失败导致退避反复归零的问题，已修复并新增回归测试；异常仍统一通过 Toast 展示。
- 验收期间用户要求部署记录详情的操作日志采用卡片，从卡片右上角日志按钮打开容器日志抽屉。用户明确纠正：抽屉展示容器日志，按钮不是操作日志的放大操作；打开时操作日志必须保留在卡片中，按钮采用日志图标。
