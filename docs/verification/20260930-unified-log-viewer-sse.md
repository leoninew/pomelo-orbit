# CI/CD 日志统一与 SSE 流式传递验收
最后修改时间: 2026-09-30 16:33:56

Review status: Draft

Flow mode: standard

## Basis and scope

- 依据 [Accepted Intent](../intent/20260930-unified-log-viewer-sse.md) 与 [Accepted Plan](../plan/20260930-unified-log-viewer-sse.md)，以及用户在实施过程中确认的界面调整。
- 用户已明确要求“进行验证”；当前为 Verification。标准模式不另建 Spec。
- 验证对象为相对 HEAD 的本次日志改动，包含用户已暂存的内容、本阶段的退避与本地环境修复、部署详情展示修正。本阶段没有执行 Git 暂存、提交或推送。
- 用户明确自行测试界面，本阶段不打开浏览器，也不启动、停止或重启开发服务器。

## Intent alignment

六类 Web 日志入口均已迁移到共享日志阅读组件与认证 Fetch + SSE：CI 阶段、CD 操作、部署详情容器、Service、Gateway、Route。Route 仍读取关联 Gateway 的日志。

文件日志按字节游标补读与追加；容器日志采用 Compose follow、时间范围与重叠恢复。部署容器订阅没有任务完成条件，stop 操作仍不展示容器日志。关闭视图停止接收并保留页面缓存；离页与 Project 切换释放订阅。

共享操作放在抽屉标题/部署操作日志卡片标题栏，移除正文中的状态行和“读取完成”文案。部署操作日志始终展示在卡片中，右上角 ScrollText 按钮打开容器日志抽屉；两种来源分别订阅，打开抽屉不卸载卡片控件。搜索采用 Monaco 悬浮查找框；跟随与停止跟随采用不同图标，停止跟随不停止网络接收。读取异常接入现有 Toast，日志组件不展示异常文案或异常图标。

## Spec alignment

不适用：standard 模式按 Intent 与 Plan 核对。

## Plan alignment

| 计划范围 | 实际核对 |
| --- | --- |
| 公共事件、游标和窄读取端口 | 新增 application/logstream 与 common/log_stream.proto；应用层不引用 Gin、SFTP 或生成 DTO |
| 四个单数资源 SSE 入口 | CI stage、deployment operation/container、application runtime 均有路由与 handler，使用当前用户和 Project 归属校验 |
| local/SSH 文件读取 | 每批 32 KiB；缺少文件等待；SSH 复用连接并短时打开文件，沿用 Windows 共享冲突处理 |
| 文件终态输出读取 | 普通 CI 阶段关闭 writer 后记录终态；终态与无新增均满三秒后最终读取 EOF；取消观测上限一秒 |
| 容器持续输出 | 专用 Runtime.Stream；Compose follow 与 Docker 时间戳；未就绪等待、来源重建、成员与目标复核；部署结束不结束容器订阅 |
| 客户端接收与缓存 | Bearer Fetch、事件增量解析、持续 UTF-8 解码、100 ms 批量追加、四个关闭资源 LRU、50,000 行/10 MiB 单资源边界 |
| 统一阅读与全部入口 | LogView/LogDrawer/LogActions；运行时抽屉是资源适配层；页面详情刷新独立于日志接收 |
| 清理与活文档 | 移除旧 Web 快照路由/API/DTO/ContainerLogView；保留 MCP 实际使用的短读服务；更新 CI/CD 与前端活文档 |

本次核对包括新增路由关闭响应正文捕获与 writer 的 Unwrap；改动仅支持 SSE 传输及保留 HTTP 请求元信息，没有改造日志框架。

## Actual diff summary

相对 HEAD 的改动为 84 个文件、3,870 行新增、1,368 行删除；包含生成代码、Intent/Plan、活文档及部署详情回归测试，不包含本验收文档。统计截至本次收尾修正。

- 后端：流式协议、通用文件读取机制、CI/CD 订阅用例、HTTP SSE transport、local/SSH/target 适配和对应测试。
- 前端：统一日志 API、SSE parser、缓存/订阅 composable、日志组件和全部六类入口；删除重复轮询与展示实现。
- 协议：新增日志流事件，删除无调用 Web 日志快照响应，更新 Go/TS 生成文件。
- 文档：意图、计划、CI 指南、CD 运行时、前端架构/样式和决策账本。

| 预期与实际文件差异 | 说明 |
| --- | --- |
| bootstrap 未修改 | 现有注入对象已经实现扩展端口，不需要新增组装 |
| SSH pipeline.go 未修改 | 文件持续读取新增在 log_reader.go；现有 runtime 已解析工作区和平台路径 |
| runtimeContainerLogs.ts 未修改 | 当前资源类型仍适用；生命周期改造位于抽屉与页面缓存 |
| 新增 LogActions.vue | 落实用户要求：操作移到标题/页签栏，正文删除中间一行 |
| CI/CD execution 与 executor 修改 | 限制取消观测间隔、关闭写入器后进入普通终态，支持最后输出读取 |
| 现有测试替身修改 | 扩展新读取/Stream 端口，没有旧接口运行时回退 |

依赖声明/锁文件、数据库迁移、任务队列、SSH 终端和部署对话没有变更。

## Acceptance checklist

“通过”指代码核对或下列自动化证据；未运行的平台与界面项单独保留，不能据此推断实机验收通过。

| 验收项 | 结果与依据 |
| --- | --- |
| 六类入口共享组件与公共阅读操作 | 代码核对通过；CI/CD 使用 LogView，三类运行时入口使用资源适配抽屉 |
| 主题、12px 等宽字体、行号、抽屉宽度 | 配置核对通过；沿用 MonacoEditor/AppDrawer；明暗主题与窄屏效果待用户确认 |
| 带 Bearer 的 Fetch + SSE、增量事件接收 | API/parser 测试通过，覆盖任意网络字节边界、多行 data、心跳注释与中文 |
| 执行过程中读取文件与容器日志、缺少来源等待 | 文件流运行态追加测试通过；容器等待/出现的编排测试通过；代码没有部署终态订阅门槛 |
| 文件历史回放、游标续读、最后输出 | 文件流测试覆盖 offset 补读、int64 精度、来源绑定和取消后延迟写入；真实平台取消时序仍需联测 |
| 容器持续输出、初始范围、重叠恢复与重建 | 可控 runner 测试覆盖命令参数、来源重建和部署结束不停流；计数去重测试保留真实重复行 |
| 异常走 Toast、保留已加载内容 | composable 测试覆盖 403 停止重试、503 连续失败 Toast 去重与已有内容保留；组件无异常展示 |
| 跟随、停止跟随与搜索 | 代码核对通过：按钮切换，滚动离底停止跟随，查找框不增加顶部空间；真实 Monaco 交互由用户测试 |
| 移除日志复制操作 | 组件无复制按钮，日志 Monaco 上下文菜单关闭；通用编辑器其他用途不变 |
| stop 无容器日志、有操作日志 | 前端资源与容器日志按钮均排除 stop，后端容器入口也校验；操作入口无该限制 |
| local/Linux SSH/Windows SSH 来源与授权 | 本地环境无 SSH 信息的容器日志回归通过；成员撤销、目标修订与 SSH 凭据修订变化均有覆盖；真实 SSH 平台项未完成 |
| 操作日志卡片与容器日志抽屉 | 新增页面回归通过：部署运行中打开容器抽屉，操作日志控件身份和持续输出保留；关闭仅取消容器订阅，重开恢复容器内容与游标 |
| 关闭、离页、Project 切换与旧事件隔离 | composable/抽屉测试覆盖主动取消、跨资源旧事件拒绝与组件卸载后恢复；任务执行不使用订阅 context |
| 页面内保留内容、游标、UTF-8 与阅读状态 | 测试覆盖 close/reopen、抽屉卸载、follow 状态和 UTF-8 半字符续读；阅读位置配置核对，视觉效果待用户确认 |
| 文件完成、容器不停流、确定性错误停止、短暂故障重连 | 对应文件/容器/composable 测试通过；本阶段补充 ready 后失败的退避回归 |
| 缓存边界与 UTF-8 | 缓冲测试覆盖行淘汰和超长中文单行字节边界；实际 10 MiB/50,000 行负载与 Monaco 性能未实测 |
| 仓库质量门禁 | 最终 task check、前端 171 项测试、Go 测试通过，diff 无空白错误 |

## Test results

执行目录均为仓库根目录；package scripts 由 --cwd web 进入前端。

| 命令 | 结果 |
| --- | --- |
| task check | 本地环境修复后通过：前端类型/lint/Prettier、Go 配置/格式/lint；Go lint 为 0 issues；最后页面修正另跑下列前端检查 |
| go test ./cmd/... ./internal/... | 本地环境修复后通过；没有将此结果视为远程平台通过；之后没有修改 Go |
| yarn --cwd web test | 前轮通过：39 个测试文件、171 项测试；最后页面修正采用下列针对性回归，没有运行覆盖率 |
| yarn --cwd web test src/composables/useLogStream.test.ts | 修复前 1 项失败，复现 ready 后持续失败每次约一秒重试；修复后 4 项通过 |
| yarn --cwd web lint:fix | 最后页面修正后通过，无错误或警告 |
| yarn --cwd web typecheck | 最后页面修正后通过 |
| yarn --cwd web test src/views/deployment/DeploymentDetail.test.ts | 通过：一条页面行为回归，覆盖卡片保留、运行中容器输出、关闭清理与重开续读 |
| 日志 API/parser/buffer/composable/运行时抽屉针对性测试 | 五个文件、12 项测试通过；复用既有覆盖，未增加覆盖率要求 |
| git diff HEAD --check | 通过 |
| go test -v ./internal/infrastructure/runner/ssh -run 'Test(Windows\|Linux)PipelineRemoteEndToEnd$' | 命令成功，但 Windows/Linux 两项均 SKIP；未配置测试 Project ID 和已可用镜像 |
| task proto | Implementation 阶段已生成并通过；本阶段未修改 proto，未重复生成 |
| Go race | 未运行：当前 CGO_ENABLED=0，且本机 PATH 没有 gcc；没有为此安装工具或改变环境 |
| 浏览器、真实 Docker/SSH 与生产代理联测 | 本阶段未运行：用户自行测试界面；远程 E2E 配置未提供，生产代理未联测 |

## Verification correction

发现并修复一项重连问题：服务端先发送 ready、再返回读取错误时，客户端把 ready 当成恢复成功，不断把退避计数归零，持续失败时反复约一秒重试。

新增可控计时回归，先复现失败，再验证修正。现在收到 chunk 或正常 waiting 才重置退避；重试间隔按 1/2/4/8/15 秒延长，抖动后仍限制在 15 秒内。持续失败仅提示一次 Toast，输出缓存不清空。修改限于 useLogStream 与测试，并同步 Plan/前端架构。

服务详情容器日志 500 的根因为 local Environment 合法地不含 SSH 配置，容器日志游标范围和目标比较却访问了 SSH 凭据字段。修复为按明确目标类型读取凭据；本地环境持续输出及 local/SSH 修订变化的回归通过，固定 Go 检查通过。

修复后仍出现旧错误时，只读核对发现 Air 已构建新二进制，但旧后端子进程仍占用 9021，新进程因端口占用退出。用户反馈自行重启后恢复；没有操作开发服务器。此现象属于热加载进程替换失败，不能把构建成功当成运行中的服务已更新。

部署详情按用户纠正收敛为操作日志卡片与容器日志抽屉：删除日志页签及操作日志的放大抽屉，右上角使用容器日志按钮。卡片控件始终挂载，容器订阅仅随抽屉开关；页面回归验证两种来源互不替换，并在部署进行中流式读取。

## Risks and incomplete items

- 桌面/窄屏、明暗主题、真实 Monaco 搜索悬浮、滚动跟随与阅读位置由用户自行验收，本阶段没有浏览器通过结论。
- Linux/Windows SSH 真实持续读取、SFTP 写入共享和读取子进程退出未验收。既有远程 E2E 被跳过；可控 runner 的退出断言不能证明真实 OpenSSH 子进程清理。
- 文件终态读取使用三秒无新增窗口与最终 EOF 复查，取消观测最多一秒。延迟末尾写入的回归通过，但这仍是有界静默策略，没有显式写入完成标记；不据此承诺所有平台的任意延迟输出均可补齐。
- 容器恢复仅补读 Docker 仍保留的输出；来源重建/缓存截断可提示缺口，不保证恢复已删除历史。
- 生产代理的缓冲/超时、十五秒心跳和十秒写入截止的长连接行为未做实机联测；真实 HTTP 测试已验证 ready/chunk Flush 和客户端关闭取消。
- 有界缓冲算法有自动化覆盖；没有真实 Monaco 大日志性能、四资源 LRU 压力或所有动态目标修订路径的专门集成测试。

## Conclusion

代码核对与可运行的自动化检查通过，验证发现的重连问题已修复，当前没有这些检查范围内的已知未修复失败。

整体人工验收仍待确认：界面由用户自行测试，真实平台与生产代理缺口按上节保留。Review status 保持 Draft，不把用户要求“进行验证”视为其已确认全部验收项。没有新增依赖或改变原定功能范围。
