# 路由与证书
最后修改时间: 2026-10-03 12:59:03

Doc role: living guide

Orbit 通过 Gateway 的 Traefik File provider 管理自定义 Route，每条 Route 使用路由编码命名的独立 YAML，例如 `route-mineru.yaml`，router/service 名分别为 `route-mineru-route`、`route-mineru-service`。Route 创建、编辑、删除先写入业务数据；列表页和详情页的启停先保留在前端草稿中。详情页的证书弹窗只保存 HTTP/HTTPS、证书方式及验证方式，不触发发布。同步预览显示发布、撤销、跳过清单；确认前核对选定业务配置、受管文件和 Gateway/Environment 依赖修订，发布后查询 Traefik API 核对加载结果。

详情页仅同步当前 Route，列表页显式选择 Project 范围。Project 预览取业务 Route 与已知发布记录的并集，确认按冻结 ID 顺序逐条处理：启用的 Route 发布或覆盖自己的文件；禁用及已删除的已知 Route 撤销自己的文件；禁用且从未发布的 Route 直接跳过。范围外路由、未知文件、Docker label 与其他 provider 资源保留。预览不比较 Traefik 增删改，也不提交运行时 hash。

自定义路由列表和详情页共用同步模态窗。编码、操作、协议三列统一为紧凑的 `7rem` 宽度，规则列使用剩余宽度，状态列维持原比例；两个入口采用相同的逐项失败 Toast、状态 tip 和重试行为。协议补充说明与失败状态旁均显示信息图标，悬停或键盘聚焦时通过共享 `AppTooltip` 展示原因；提示浮层通过 Portal 位于模态窗上方，不依赖浏览器原生 `title`。

Web 在同步模态窗的原预览表中保留名称、操作、协议与规则，追加状态列；协议列单行展示 HTTP/HTTPS，证书方式（例如 `Let's Encrypt · DNS-01`）作为协议文本的 tip 展示，无补充说明时不设置 tip，规则列只展示匹配条件与目标。API 返回独立的 `rule.protocol`、`rule.match` 和 `rule.target`，协议由入口配置决定，不从上游 URL 推断。确认、逐条反馈、失败重试与完成结果都留在同一模态窗和表格中。前端按预览顺序逐条确认，每次只提交该 Route 的预览修订和启停草稿。每条响应立即更新对应记录，单条失败后继续处理后续项；状态依次为待处理、处理中、完成或失败，失败时立即弹出包含路由名称和原因的 Toast，状态列的 tip 保留相同原因。Toast 位于模态窗上方，不被遮罩遮挡。每条响应同时更新页面已保存的草稿，已保存的业务修改不会因外部发布失败回滚；未保存的启停草稿仍保留。预览失败同样弹出原因 Toast，并在模态窗内保留错误信息。预览或同步失败后，底部主按钮切换为可点击的「重新预览」；重新预览只重试未完成项，并保留成功记录，预览成功后仍需点击「同步」确认，不自动发布。同步执行期间禁止关闭模态窗，完成后由用户关闭；本轮不增加关闭页面或弹窗后的任务续跑能力。

API/MCP 保留显式批量确认能力。确认响应的汇总编码为 `route_sync_completed` 或 `route_sync_incomplete`，逐项报告业务保存、文件提交、配置匹配、证书验证、恢复和清理状态；Web 只展示一个状态字段，不展开内部阶段。预览同时返回整批及每项独立的 `business_hash`、`publication_hash`，单条确认使用对应项的修订，期间业务或共享依赖变化须重新预览，不能自动发布未确认的新内容。

### 单条与批量的执行操作

优化依据为 2026-10-03 11:22–11:24 的单条 minio-api 日志。父阶段包含子阶段，不能重复相加：

| 请求/阶段 | 开始 → 结束 | 耗时 | 结果 |
|-----------|-------------|------|------|
| 预览全量扫描 | 11:22:54.417 → 11:23:16.586 | 22.169s | 完成，读取六条记录与实际文件 |
| 预览 Gateway 校验 | 11:23:16.592 → 11:23:23.367 | 6.774s | 完成 |
| 预览请求 | 11:22:54.417 → 11:23:23.368 | 28.951s | HTTP 200 |
| 确认状态准备 | 11:23:29.329 → 11:23:51.984 | 22.655s | 扫描 16.991s + Gateway 校验 5.656s |
| 确认发布前第二轮扫描 | 11:23:51.984 → 11:24:07.964 | 15.980s | 完成，重复全量读取 |
| 上次 pending 恢复 | 11:24:07.964 → 11:24:24.932 | 16.967s | 完成，含重载 3.687s、匹配 6.205s |
| 文件发布 | 11:24:24.932 → 11:24:28.501 | 3.569s | 完成 |
| 重载 | 11:24:28.501 → 11:24:32.295 | 3.793s | 完成 |
| 配置匹配 | 11:24:32.295 → 11:24:38.532 | 6.236s | 四个 API 串行查询 |
| 收尾 | 11:24:38.532 → 11:24:40.621 | 2.088s | 完成 |

确认请求到最后收尾约 71.293s，所贴日志没有最终 HTTP 完成行，不能作为精确请求总耗时。三轮扫描合计 55.140s，占预览加确认已知执行时间约 55%；pending 表示上次中断的遗留状态，本轮所贴阶段均未出错。单次文件操作 1–2 秒包含 SSH/SFTP 建连、通道、远端进程及实际文件读取，不能把全部时间都归为网络或握手。

列表批量和详情单条采用同一发布流程。列表先预览整批，再循环逐条确认；详情预览并确认当前一条。以下按正常成功、无额外轮询计算：B 为前端确认条目数，W 为进入文件发布适配器的条目数（W ≤ B）。禁用且从未发布的项仍确认业务保存，但跳过文件发布。

| 操作 | 详情单条 | 列表批量 | 做什么、为什么需要 |
|------|----------|----------|--------------------|
| `POST /api/route/sync/preview` | 1 次 | 1 次 | 鉴权、冻结范围及每项业务/发布修订，不写文件 |
| `POST /api/route/sync/confirm` | 1 次 | B 次 | 顺序确认，前端立即反馈每项结果，失败后继续 |
| `gateway_lock` / `resolve_target` | 预览、确认各一次 | B+1 次 | 串行化同一 Gateway，绑定当前目标与受管凭据 |
| `inspect_publication` | 当前项 2 次 | 预览 B 次、确认 B 次 | 先读取业务记录和 code，再逐条检查对应发布记录与文件，不读取其他 Route |
| 发布记录目录枚举 | selected 为 0 次 | Project 预览 1 次 | 全量范围合并业务记录与发布目录中的 ID，保留已删除 Route 的撤销；确认仍逐条处理 |
| `read_publication_record` | 最多 3 次，均为当前项 | 2B+W 次 | 预览、确认准备和发布前复核只读取对应 state.json，与范围外发布记录数量无关 |
| 实际 YAML/PEM 检查 | 仅当前项与必要 previous | 预览范围及每次确认项 | 比较活动文件和证书版本，检测缺失、外部修改、中断及恢复材料；不检查范围外的实际文件 |
| `validate_gateway` | 2 次 | B+1 次 | 查 Compose 容器、容器静态配置和实际挂载；核对 provider、入口与 resolver |
| 短数据库事务 | 1 次 | B 次 | 复核业务修订，有修改才保存；远端操作在事务外执行 |
| `verify_publication` | 最多 1 次 | W 次 | 发布入口重读当前记录与实际文件，拒绝预览之后的变化 |
| `pending_recovery` | 条件执行 | 各项条件执行 | 按已持久化候选/previous 文件确认或恢复上次中断，不能用新草稿重建 |
| `file_publication` | 最多 1 轮 | 最多 W 轮 | 备份旧 YAML、写 pending、准备不可变证书、替换活动 YAML 或撤下旧路径 |
| `reload` | 发生文件变化且目标需要时 | 各项条件执行 | Windows bind mount 必要时定位 Gateway 并发送 HUP；Linux 依赖 watcher |
| `configuration_match` | 普通 HTTP 每轮 2 个 API | 普通 HTTP 每项每轮 2 个 API | 只查询涉及协议的 router/service；TCP 同样两个，协议切换最多四个，并核对旧资源消失 |
| `finalize_publication` | 文件变化时 1 轮 | 各项条件执行 | 写 confirmed，清理该 Route 不再需要的旧证书版本 |
| SSH/SFTP 关闭 | 每个请求退出时 | 每个请求退出时 | 请求内复用连接；不跨预览/确认或不同确认请求共享 |
| 最终页面刷新 | 1 次 GET Route | 1 次 GET Route 列表 | saved 逐项更新本地草稿/启停，finished 在成功或失败结束后只刷新一次；详情不重查未变化的 Gateway |

API/MCP 在一个请求内显式批量确认时，整批共享一次状态准备、Gateway 校验和连接作用域；状态准备逐条读取所选 Route，每项发布前仍只复核自己的记录和文件，并独立执行发布、重载、核验及报告结果。Web 保持逐条请求，跨请求重新读取当前项和校验 Gateway，用于发现批次期间目标及共享依赖变化。

已确认 YAML、证书和当前目标修订均未变化时，报告 `file_commit=unchanged`：检查运行配置，已匹配则不写 backup/pending/活动文件、不发送 HUP、不做收尾清理；配置未加载才重载并再次匹配。API 不可用或外部文件变化仍失败。pending 确认/恢复已经核验运行配置，若所得基线等于本次期望，就复用核验结果，避免第二轮发布、重载与匹配。已撤销编码被其他路由合法复用时，不要求该资源消失，也不会删除它。

一次请求中的 SSH/SFTP 操作复用相同连接与文件客户端，SSH exec 仍为每条命令开启独立通道及远端进程。阶段取消或连接失效会关闭并淘汰连接；恢复在自己的有限预算内建立新连接。正常退出关闭连接，不保留跨请求连接池。

Environment 切换目标类型、工作目录、SSH 地址/平台/凭据后，当前工作区内的历史 `TargetRevision` 只表示发布来源，不阻断预览。切换或 Probe 成功不修改文件/记录，也不自动同步；显式确认重新发布所选路由并写入当前 revision，不使用旧 revision 的无变化捷径。预览与确认之间或请求执行期间目标变化仍使预览过期；Project/Gateway 归属、外部修改、当前 Route 证书完整性与恢复校验仍有效。旧目标文件不由新目标同步迁移或删除。

Traefik 直接重启从持久化文件恢复，Orbit 可以离线。Service 部署/重启与自定义 Route 同步独立：不检查发布记录、YAML/PEM 或已发布上游引用，不因 pending、发布失败或文件缺失阻止部署。Gateway 仅检查 Traefik 管理 API 就绪，不核验自定义 Route；业务 Service 不等待 Gateway API。移除或改名 Component/endpoint、改变网络或 Gateway entrypoint/resolver 后，若需要更新自定义路由，应另行显式同步。Component `endpoint.mode=gateway` 仍是随部署生效的 Docker label 路由，默认 entrypoint/TLS 由 GatewayConfig 控制。

前端 `/routes` 以自定义 Route 为主列表，提供创建、编辑、批量启停草稿和同步入口；Route 详情页提供单条启停草稿、独立证书配置和同步。已启用 Route 修改证书后先保存，页面提示待同步；预览同时展示协议与证书方式。未启用 Route 的证书配置同样直接保存，不提示待同步，启用并同步后才发布。未启用 Route 的创建、编辑和删除也不单独提示待同步。`/route/traefik` 单独展示 Traefik 当前路由，可从主列表进入并返回。

Route 编码为最多 32 字符的小写 DNS 单标签，以字母开头，只允许 `a-z`、`0-9`、`-`，末尾不能是连字符。`external_domain` 可为 `sub.example.com` 等多级子域名，每级标签符合 ASCII 主机名规则，末级标签不能纯数字，最长 220 字符。Gateway 配置 `external_domain` 后，创建和编辑 Route 时修改编码会在现有域名控件中生成 `编码.external_domain`；手工填写的其他域名不会被后续编码修改覆盖。未配置外网域名时，创建表单域名控件留空。保存的是域名控件中的完整域名；外网域名配置本身不创建 DNS 记录或对外路由，也不批量修改已保存 Route。

Gateway 详情页在独立的「Traefik API」卡片中展示和编辑容器 API URL、宿主机 API URL 及就绪超时；基本信息单独编辑名称和域名。移除的是 REST provider 发布入口，管理 API 继续用于就绪检查、运行时列表和发布/恢复后的加载核验。本地容器中的 Orbit 使用容器地址；本机 Orbit 和 SSH 目标使用宿主机地址，SSH 请求在远端执行。

自定义 Route 同步不设整次请求的总耗时上限，Web 的预览和确认请求也不设 Axios 总超时；后端为每个阶段单独创建有限的 context。当前配置如下：

| 后端配置 | 时长 | 作用 |
|----------|------|------|
| `route.gateway_lock_timeout` | `30s` | 等待当前 Gateway 的同步锁 |
| `route.state_load_timeout` | `10s` | 准备预览/确认状态；发布前再次检查受管记录 |
| `route.file_publication_timeout` | `5s` | 写入备份、发布记录、证书及活动文件；确认后记录和清理另起同等预算 |
| `route.api_request_timeout` | `10s` | 每次 Traefik API 查询，包含 SSH 连接与远端命令启动 |
| `route.reload_timeout` | `5s` | 文件提交后必要的 SIGHUP 重载 |
| `route.configuration_match_timeout` | `5s` | 重载结束后核对 router/service，内部 API 请求受剩余时间限制 |
| `route.recovery_timeout` | `20s` | 文件恢复、重载与匹配，以及上次 pending 的确认或恢复 |

阶段从进入时开始计时，前序阶段不扣减后续预算；API/MCP 批量请求的单项超时不会耗尽其他项的预算。调用方取消时停止正常处理，尚未开始的项明确报告跳过；已经需要恢复的操作使用脱离调用方取消、单独限时的恢复 context。恢复内部的重载和匹配仍受恢复剩余时间约束；恢复未完成则保留 pending 记录与材料，不报告成功。启动时校验各阶段时长为正，不要求单次 API 上限小于匹配上限：嵌套 context 自动采用较早的截止时间，匹配阶段内每次 API 查询实际最多使用该阶段剩余的 5 秒预算。

2026-10-03 12:38–12:40 的优化后日志共有 155 个阶段起止配对，无超时；状态准备约 4.2–7.7 秒，其中 Project 预览读取 9 条记录耗时 7.274 秒；实际文件提交最长 1.729 秒，收尾最长 0.770 秒，重载约 2.19 秒，单次 API 约 0.85–1.27 秒，HTTP 两接口匹配约 1.73–2.17 秒。用户据此明确采用上表预算。状态准备 `10s` 限制整个范围，更多路由需共同使用该预算；新日志未覆盖证书批量写入、协议切换与失败恢复，优化前 pending 恢复约 17 秒仅作为历史参考。日志用 `phase`、`timeout_ms`、`elapsed_ms`、`status` 标明阶段结果，并逐条记录 `state.json` 读取时间。阶段取消引起的 SSH/SFTP 底层错误同时保留 context 的超时原因，避免只看到 `connection lost` 或 channel-open 错误而无法定位截止来源。

配置支持 `.env.example` 中对应的 `POMELO_ORBIT_ROUTE__*` 环境变量覆盖；修改后需重启 Orbit 后端才生效。这与 Gateway 页面中的 Traefik API 就绪超时不同，后者只用于 Gateway 启动就绪检查。

Gateway Service 工作目录的结构如下：

```text
<Environment.workspace_root>/deployment/<gateway-service-code>/
  gateway/
    dynamic/route-<route-code>.yaml
    certs/route-<route-id>/<sha256-revision>/
      cert.pem
      key.pem
    acme/acme.json
  .orbit/route-publication/<route-id>/
    state.json
    previous.yaml
```

证书目录和 `.orbit` 归属记录使用稳定 ID。修改路由编码后显式同步会写入新编码文件、撤下旧文件，并核对旧 router/service 消失；失败恢复旧路径。撤销使用已发布编码，未同步的编码修改不会使其他路由文件被删除。目标 YAML 已存在且不属于当前 Route 时，发布报告冲突。已确认改名或撤销后释放的编码不再由历史记录占用，可供其他 Route 复用。

同步日志使用 `project_code`、`route_code`，从业务记录取得编码后再读取对应发布记录。单条同步不会读取其他 Route 的 state.json、YAML 或 PEM；同域名多 Route 的证书配置也不再通过全局发布状态扫描核验。业务已删除时先记录发布记录路径，成功读取后补充已发布 code。

删除 `gateway/dynamic` 后，重新部署 Gateway 可以重新准备空目录，不会因自定义路由文件缺失报告部署失败，也不会自动发布路由。重新预览并确认 Route 同步可重建选中 YAML 及其父目录；只有用户确认的内容被发布。外部修改的现存文件仍在路由同步中报告冲突，不自动覆盖；预览后删除文件会使该预览过期。

Gateway 部署统一应用 `providers.file.directory=/etc/traefik/dynamic`、`watch=true`，移除 REST provider，动态配置与手工证书目录只读挂载，ACME 目录读写挂载；静态文件覆盖后自动重建 Gateway 容器。发布记录与旧 YAML 位于未挂载、未监视的 `.orbit` 下。路由文件、PEM 与记录使用现有 `WorkspaceFile/SyncFiles`，文件 Mode 为 `0600`；完整临时文件写好后替换活动文件。清理只处理该 Route 的旧证书版本，保留当前和必要的上一版本。

SSH 文件由远端 runtime 写入，沿用远端身份和权限。Orbit 容器内运行 local target 时，文件写 logical path，Docker 挂载使用既有 resolver 得到的 daemon 可见 physical path。正常发布检查运行容器的 provider、entrypoint/resolver 和实际挂载；权限失败不会被当作成功。Windows 替换窗口与 ACL 平台差异仍为已知限制。现存受管 REST Gateway 通过线下告知重新部署后显式同步，不转换旧 REST 快照或新增迁移 UI。

Windows 工作目录 bind mount 到 Docker Desktop Linux 容器时，可能出现“容器可读新文件，但 File provider watcher 不刷新”。Orbit 在文件提交后向当前 Gateway 容器发送 SIGHUP，让 File provider 重新读取配置，再核对加载结果；撤销、失败恢复和中断恢复使用相同机制，不重启 Gateway。SSH 按远端平台判断，local 按实际 daemon mount source 判断，因此容器中的 Orbit 也能识别 Windows bind mount 的 DooD 场景。Linux 原生目录与 SSH Linux 继续使用 watcher，暂不主动发送 HUP；真实 SSH Linux 热更新留待后续测试。

重载失败使用逐项编码 `route_sync_reload_failed`，不会被当作同步成功。文件恢复后仍须重载并匹配才报告已恢复；失败则保留恢复材料供下次预览和重试。SIGHUP 重读现存文件集合，保留其他 Route 和未知文件，不读取未同步的业务编辑。

预览及确认准备阶段无法查询 Gateway 的 Docker 容器时，返回 HTTP 503 与 `route_sync_gateway_unavailable`，提示检查目标主机的 Docker、Compose 和 Gateway 部署；Docker 查询成功但没有运行中的 Gateway 时，返回 `route_sync_gateway_not_running`，提示先启动或部署 Gateway。SSH 终端能登录只代表远程连接可用，不能证明 Docker daemon 或 Gateway 容器可用。预览错误与确认错误共用原因编码翻译，模态和 Toast 展示同一原因；命令退出码及最多 4 KiB 的诊断输出保留在服务端 cause 中，不进入前端。

文件权限失败使用逐项编码 `route_sync_publish_permission_denied`；发布阶段超时且没有更具体原因编码时使用 `route_sync_publish_timeout`；API 查询失败或超时导致配置无法确认使用 `route_sync_configuration_unavailable`；已观察到配置但与提交内容不符使用 `route_sync_configuration_mismatch`；上次中断的发布无法确认或恢复且没有更具体原因编码时使用 `route_sync_pending_recovery_failed`。其他文件发布失败使用对应原因编码或 `route_sync_publish_failed`，不再笼统提示“发布未完成，请重新预览后重试”。弹窗与 Toast 展示相同的本地化原因，自动恢复失败时追加说明待恢复记录已保留，不显示请求/操作 ID 或原始错误。重新预览只刷新待发布计划，不会修复 API 连接、加载超时或文件权限；应先根据具体原因处理再确认重试。配置匹配与实际 TLS 证书验证在 API 中分开报告：router/service API 匹配不能证明新 PEM 已生效，当前正常同步的 HTTPS 证书结果为 `unverified`。界面“完成”表示配置同步及必要操作完成，不表示证书已生效。实际 TLS 验证使用正确 SNI 和预期证书 fingerprint；未验证证书不自动回滚已匹配的配置。

HTTP Route 可使用手工 PEM、mkcert 或 Let's Encrypt。HTTP-01 需要 Gateway `http` 或 `http-dns` profile；DNS-01 需要 `dns` 或 `http-dns` profile 和 Gateway 中保存的 Cloudflare token。DNS-01 仍要求可注册的真实域名。

配置 DNS-01：

1. 在 Cloudflare 创建仅包含目标 zone `Zone:Read`、`DNS:Edit` 的 API token。
2. 在 Gateway 编辑页选择 `dns` 或 `http-dns`，填写 email 与 token。
3. 保存并重新部署 Gateway。
4. 在 Route Let's Encrypt 对话框选择 DNS-01。

TCP Route 先要求选中 Gateway Version 已声明对应 TCP endpoint 和 host port，再创建 Route。GatewayConfig 不提供 listener 或端口编辑。
