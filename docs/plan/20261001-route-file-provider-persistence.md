# Route 文件发布、重启恢复与独立同步计划
最后修改时间: 2026-10-02 14:03:00

Review status: Accepted

Flow mode: standard

Current stage: Implementation

## Intent basis

- 依据 [已接受的 Intent](../intent/20261001-route-file-provider-persistence.md)。用户已采纳下述目录结构并要求编写 Plan。
- 自定义 Route 使用 Traefik File provider；普通 Component 的 Docker label 路由沿用现有行为。Traefik HTTP API 继续用于 readiness 与运行时查询。
- 详情页支持单条同步；批量同步必须显式确定范围。发布 A 不携带 B 的未同步编辑，也不重写 B 的配置、证书或权限。
- 支持现有 local、SSH Linux、SSH Windows，以及 Orbit 容器运行下的 DooD 路径边界。
- 业务数据库保存可编辑配置；重启与重新部署只恢复已发布配置。不扩展 TCP listener 功能，不保留 REST 发布回退。
- 不转换旧快照/证书、不回填历史发布状态、不增加首次切换 UI 或外部实例接管。现存受管 Gateway 在用户重新部署时直接应用当前 File provider 和受管挂载，通过线下传达重新部署及显式同步一次的要求。
- 批量按冻结顺序逐条执行，单条失败后继续；未完成情况通过汇总编码与逐项结果交给前端/MCP。配置匹配与证书验证分别报告。
- 复用现有 target writer 的文件与权限机制；极边缘场景和 OS 限制只记为已知问题，不投入专项修复或验证，也不作为交付阻塞项。
- 用户支持完整写入临时文件后再替换活动文件；沿用现有 writer，不增加 Route 专属直接覆盖内容实现。
- 用户已要求开始实施，本文作为已接受的实施依据；运行必要的实现检查，不部署或重启用户服务。
- 实施中用户选择将预览简化为发布/撤销/跳过清单；去掉 Traefik 运行时增删改对比与 hash，保留确认前修订核对及发布后加载验证。
- 用户补充要求恢复 Windows SIGHUP 重载，local 模式覆盖 Windows/Linux 和 DooD；Linux 保持 watcher，暂不增加主动重载，后续通过远程 SSH Linux 实测。

## Existing implementation

| 边界 | 现状与需要调整的行为 |
| --- | --- |
| Route 同步 | `sync_change.go` 从全部 enabled Route 构造计划；确认后再次读取全部 enabled Route。需要限定范围并发布确认时冻结的输入 |
| 发布适配器 | 原实现全量 REST PUT；工作区草稿写聚合 `routes.yaml` 并全目录 prune PEM。需要改为单条文件提交及按归属清理 |
| Gateway 生命周期 | deploy/restart hook 从数据库重新发布全部配置。需要改为检查持久化文件及其加载结果 |
| Gateway 拓扑 | 用户补充决定部署直接覆盖受管 File provider 与目录挂载；Version 的其他拓扑继续保留，不要求修改已保存 YAML |
| 受控文件与权限 | `controlled_file` 已声明 Content/Mode/IgnoreIfExists，并转换为 WorkspaceFile；local/SSH 的 StageWorkspace 与 SyncFiles 复用各自 writer。本任务直接复用这些字段与执行机制 |
| 文件提交 | SSH Linux 已有临时文件及 POSIX rename；SSH Windows 沿用现有备份、移动、恢复，目标文件短暂缺失属于已知限制；local 完整文件提交仍是工作区草稿，按正常发布需要评估和验证 |
| 前端与 MCP | 共用全量 preview/confirm 契约；单条页面只限定启停草稿，尚未限定发布范围 |

## Directory and mounts

采用用户已采纳的 Gateway 工作目录结构：

```text
<Environment.workspace_root>/deployment/<gateway-service-code>/
|-- docker-compose.yml
|-- gateway/
|   |-- dynamic/
|   |   |-- route-ragflow.yaml
|   |   `-- route-mineru.yaml
|   |-- certs/
|   |   |-- route-a/
|   |   |   `-- rev-001/
|   |   |       |-- cert.pem
|   |   |       `-- key.pem
|   |   `-- route-b/
|   |       `-- rev-003/
|   |           |-- cert.pem
|   |           `-- key.pem
|   `-- acme/
|       `-- acme.json
`-- .orbit/
    `-- route-publication/
        `-- <route-id>/
            |-- state.json
            `-- previous.yaml
```

- 动态文件为 `route-<route-code>.yaml`；证书目录仍为 `certs/route-<route-id>/<revision>/`，示例中的证书目录 `a/b` 表示稳定 Route ID。`.orbit` 的发布记录也使用 ID，编码改变不迁移证书或内部归属。
- `<revision>` 是发布时分配的不可变证书版本。示例中的数字仅用于展示，不要求全 Gateway 共享递增序号。
- 每份 YAML 包含该路由的 router、service、必要 middleware 和手工/mkcert 证书引用；纯 HTTP 路由不创建证书目录。
- router/service 名称统一由 Route 编码派生，内部引用一致，例如 `route-mineru-route`、`route-mineru-service`，运行时身份使用 `@file`。更改编码在下一次显式同步时发布新路径并撤下旧路径；先保留旧 YAML 及已发布编码，失败恢复旧路径并撤下候选路径。
- `.orbit/route-publication` 是计划补充的发布记录和回滚区，位于同一 Service 工作目录，不挂载给 Traefik，不存放未同步的业务草稿。
- 临时动态文件以 `.tmp` 结尾；备份位于监听目录外。证书临时文件也不作为 YAML 配置加载。

| 工作目录相对源路径 | Traefik 容器目标路径 | 挂载 |
| --- | --- | --- |
| `./gateway/dynamic` | `/etc/traefik/dynamic` | directory，read-only |
| `./gateway/certs` | `/etc/traefik/certs` | directory，read-only |
| `./gateway/acme` | `/letsencrypt` | directory，read-write |

Gateway Version 的静态配置声明：

```yaml
providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
```

Docker provider 等已有非 REST 声明保持由 Version 管理。新建 Gateway Version 直接采用 File provider，不再声明 `providers.rest`。证书引用示例：

```yaml
tls:
  certificates:
    - certFile: /etc/traefik/certs/route-a/rev-001/cert.pem
      keyFile: /etc/traefik/certs/route-a/rev-001/key.pem
```

挂载整个目录，使替换 YAML 文件后容器仍可观察到新文件。ACME 的 `acme.json` 由 Traefik 管理，不参与 Route 的证书版本清理；Linux 下保持 Traefik 要求的 `0600` 及可读写所有权，Windows 按实际 ACL 检查。

## Design decisions

### 发布状态与业务配置

1. Route 数据库记录是可编辑配置；目标端 YAML、被其引用的证书版本及发布记录共同描述已提交/已确认配置。Traefik API 用于观测 router/service 的配置匹配；实际 TLS 证书需独立验证。
2. `state.json` 记录 Project、Environment/target revision、Gateway Application、Route ID、已发布受管上游的 Service/Component/endpoint 引用、操作 ID、文件 fingerprint、证书版本和发布阶段，不保存私钥正文；Gateway Service 由记录所在 Service 目录限定。保存上一份可用 YAML 以支持单条回滚；部署依赖检查读取已发布引用，不读取未同步的目标编辑。
3. 提交前写入 pending 操作记录；提交后记录文件结果，再分别记录配置匹配与证书验证结果。中断后先比较目标文件与操作 fingerprint，解释或恢复这次操作，不从当前 Route 编辑数据自动重建。
4. 正常稳定状态下，动态文件是重启恢复来源；提交后但尚未确认的候选由发布记录区分，不能把它显示为已加载。
5. 不新增重复的数据库 Route 快照表。删除 Route 后，仍能从目标端受管文件/发布记录识别待撤销 Route ID；预览清单取业务 Route 与受管发布记录的并集。
6. 仅登记为当前 Project/Gateway 管理的资源可撤销。未知 File provider 文件、Docker label 资源与其他 provider 资源可以展示诊断，不纳入自动清理。
7. 以发布记录中的编码识别旧路径，以候选编码识别新路径；pending 修订覆盖两者的实际文件。启用路由及 pending 的候选/上一配置保留编码占用，拒绝其他 Route 占用的编码或无归属的目标文件。撤销使用已发布编码，即使业务编码已编辑。已确认撤销的编码允许其他 Route 复用，部署验证不会把复用资源判为旧 Route 未撤销。
8. 缺失文件与外部改写分开处理：删除后须重新预览，显式确认可重建选中 YAML，SyncFiles 负责创建父目录。若发布前旧文件已缺失，pending 记录该空基线，失败/中断恢复还原缺失状态而不使用过期 backup；文件确认后才完成发布。Gateway 的依赖预检允许已确认记录的 YAML 缺失，以便 staging 重建空目录；部署后核验仍报告未恢复的文件，不从业务草稿自动重发。

### 同步范围与并发

- 保留 `/api/route/sync/preview`、`/api/route/sync/confirm` 两个单数 API；修改 DTO/proto、前端与 MCP 的契约，统一显式范围。
- preview 区分 `selected` 与 `project` 范围：前者要求非空 Route ID 集合；后者由一次预览解析当前 Project 的受管 ID 集合。缺失范围不隐式等同于全量。
- preview 返回冻结、有确定顺序的 Route ID 清单和每项独立的 `business_hash`、`publication_hash`。Web 按该顺序循环，每次 confirm 只提交一个 ID、该项修订及对应启停草稿；每条返回后立即更新记录。整批修订继续供 API/MCP 的批量调用使用。确认后不重新枚举全部 enabled Route，不补入新建路由或范围外编辑。
- preview 的 `items` 表达 `publish`、`withdraw`、`skip` 及候选规则/目标、证书方式，不查询 Traefik 全量 router/service、不计算增删改、不返回 `matched` 或 `traefik_hash`。Traefik API 查询仅用于运行时列表、readiness 与发布后加载核对。
- Gateway 详情拆分基本信息与 Traefik API 卡片，各有独立编辑弹窗。基本信息保存名称及内外网域名；API 保存容器/宿主机地址与就绪超时，复用既有分区校验和部分更新接口。共享中英文名称改为 Traefik API，初始化页面使用同一字段文案；API 契约及目标端地址选择保持现有设计。
- 规则返回独立 `rule.protocol`（`http`、`https`、`tcp`）、`rule.match` 和 `rule.target`；入口协议由 Route 协议及 HTTPS 开关决定，匹配表达式不再拼 HTTP/HTTPS 前缀。同步表沿用当前预览到执行的同一清单，列为名称、操作、协议、规则、状态；协议列包含 HTTP/HTTPS 和独立证书方式行（例如 `Let's Encrypt · DNS-01`），规则列只保留匹配条件与目标，只有一个处理状态字段。不改变 TCP 校验或发布逻辑。
- `changes` 仅包含范围内的启停草稿。`business_hash` 包含选定 Route、证书配置及草稿；`publication_hash` 包含选定发布记录、实际 YAML/PEM fingerprint 及依赖修订。
- hash 同时覆盖必要的 Gateway/Environment 依赖修订，例如 entrypoint、resolver 和 target；B 的普通独立编辑/发布不使 A 的预览过期。
- 使用单节点既有运行模型中的按 Gateway 协调器，串行处理文件提交、加载观测与回滚；Gateway deploy/restart 与 Route 发布共用该协调边界。并发请求可以排队，不能同时覆盖共享加载结果。
- 不同 Route 的文件与证书归属独立；同一路由确认必须复核 fingerprint。对业务配置的并发编辑，在短事务内复核，只发布确认时冻结的候选，不在事务后重新读取成另一份配置。
- 外部文件写入和 Traefik 查询均在数据库事务外执行，沿用现有 application-owned TransactionRunner。

### 批量、禁用与删除

- 预览冻结全部选定 Route；Web 每次单条确认重新核对该项业务/文件修订及 Gateway/target 依赖，鉴权、范围或过期失败沿用 HTTP 错误契约，并在该行显示失败原因。不同 Route 发布不使其他项修订失效，同一项或共享依赖变化须重新预览。
- Web 按冻结的预览顺序逐条调用确认，不并行；每条响应立即更新状态和已保存草稿，失败仍继续下一条。后端保留单条短事务、外部发布、加载核对和失败恢复，以及 API/MCP 已有显式批量能力；批量请求内部仍顺序执行并保留成功项。
- 失败项尽力保留或恢复上一份可用配置。业务已保存、回滚失败、运行时结果不确定或清理待重试的情况分别记录；重试必须重新预览相应 ID，不能重放原批量来覆盖成功项。
- 返回每条 Route 的业务保存、文件提交、配置匹配、证书验证、恢复/清理和错误编码；批量未完成不能被 UI/MCP 转换成“全部同步成功”。
- 禁用通过撤下该 ID 归属的已发布编码 YAML 完成；先保存存在的旧文件，再提交撤销，确认该 Route 的运行时资源消失。业务编码尚未同步时仍撤销已发布路径，不操作新编码所指的其他文件。
- 禁用且从未发布的 Route 为 `skip`，保存必要的启停草稿后返回完成及文件/配置 `not_applicable`，不建立发布记录、不删除未知文件。这与预算不足的 `route_sync_skipped` 未完成项分开。
- 删除仍先修改业务数据，发布撤销继续走 preview/confirm。已删除但尚有受管文件的 ID 在批量预览中显示为待撤销，不因数据库记录消失失去归属。
- 清理仅作用于选定 Route 的受管版本。成功更新后保留当前及必要的上一份可用版本；更早版本在确认无活动/回滚引用后清理。撤销确认前不删除正在使用的证书。
- cleanup 失败单独报告并可重试，不撤销已确认的路由配置，也不误报为路由从未加载。

### 确认响应与前端报告

- 扩展专用 `RouteSyncConfirmResp`。逐项循环正常结束后返回 HTTP 200 的类型化处理结果；HTTP 200 表示结果已返回，业务是否全部完成由 `code` 和 `results` 决定。
- 汇总 `code` 使用 `route_sync_completed` 或 `route_sync_incomplete`。配置发布/撤销失败、跳过、结果不确定、恢复失败或清理未完成时，使用后者；包含每项 `route_id`、`operation_id`、阶段状态与独立错误编码，并返回 `request_id` 便于定位。专用 proto 响应沿用 `UseProtoNames` 的 snake_case；通用 HTTP 错误的 `requestId` 保持现有契约。
- 单条同步使用同一结果契约；失败项沿用 `route_sync_publish_failed`、`route_sync_publish_permission_denied` 等对应原因编码。整体预览过期等执行前错误仍使用 `WriteError` 的 `code/error/requestId` 响应。
- 证书验证结果独立于配置同步汇总：`matched`、`mismatched`、`unverified`、`not_applicable`。未验证或 TLS 观测失败不自动把已匹配的配置判为发布失败；不匹配在 API 中单独报告并提供诊断，不自动恢复 YAML。汇总完成仅表示配置同步及其必要操作完成，不能表达为“证书已生效”。
- `RouteSyncDialog` 在原预览表保留路由名称、操作、协议与规则，并追加一个状态列；确认、执行、失败重试与完成后都保留同一表格及行，不切换结果界面。预览及等待调用为“待处理”，当前调用为“处理中”，逐项编码为 `route_sync_completed` 时显示“完成”，其他未完成项显示“失败”。使用共享 `AppBadge`，失败原因通过 `title` 提示，以原因编码本地化，未知编码使用接口原因或通用失败文案；清理未完成也通过失败提示报告。不显示业务保存、文件提交、配置匹配、证书验证、恢复/清理阶段或请求/操作 ID。同步执行期间禁止关闭模态窗，完成后由用户关闭。
- 未完成时保留逐项状态并允许只对未完成项重新预览；根据 API 的业务保存结果刷新界面和保留未保存草稿。API/MCP 继续返回完整诊断，前端逻辑保留重试和实际保存边界。确认前的发布/撤销/跳过清单保持不变。
- 前端只使用用户已确认的每项修订，禁止逐条自动重新预览后发布。HTTP 执行前错误或网络失败只标记当前行失败并继续；不伪造文件/业务提交结果，未获知保存结果的草稿保留。重试仍需显式重新预览；成功项继续保留在结果列表中。
- 同步模态窗沿用共享 AppDialog，最大宽度 960px，表格最大可见高度 384px；保留 `100vw - 32px` 的小屏宽度约束、共享的 `90vh` 高度上限和表格滚动，不引入新弹窗组件。
- 现有通用 Axios 错误归一化只保留 `code/error/requestId`；因此逐项结果放在专用确认响应中，不扩展全局错误封装。字段通过 proto 生成并映射，前端按编码本地化文案。
- Route 确认请求采用现有远端操作超时配置；每次请求保留有限预算，Web 多条顺序调用不共享后端整批 105 秒预算。API/MCP 批量请求预算不足时，后续未执行项记录跳过编码并返回 `route_sync_incomplete`。本轮不增加页面/弹窗关闭后的续跑、取消或异步任务系统；网络失败不能假定当前项未执行，重新预览仍核对持久化发布记录。

### 单条发布顺序

1. 鉴权、校验范围和修订，读取该 Route 的当前受管文件与发布记录；预览及确认检查运行容器的 provider、entrypoint/resolver 与实际挂载。发布后的加载查询确认 API 可用及配置匹配。
2. 冻结所选 Route 的候选；用结构化 YAML 渲染，校验 resource 名称/引用、目标、证书与私钥匹配，以及和已发布配置的必要冲突。
3. 在短数据库事务内复核所选业务数据并保存启停草稿。后续失败沿用“业务可能已保存、发布可重试”的边界。
4. 保存旧 YAML 与 pending 发布记录；将证书写入该 Route 的新版本目录，完成权限及可读性检查。旧版本此时仍保留。
5. 将完整 YAML 交给现有 target writer，通过临时文件提交到候选编码的活动路径；编码改变时撤下上一发布路径并同时核对新资源与旧资源缺席。临时动态文件不使用可被 provider 加载的扩展名。复用各平台现有替换机制，Windows 短暂缺失窗口按已知限制处理；撤销时执行该 ID 归属的已发布路径精确删除。
6. Windows 通过目标 runtime 向当前 Gateway 容器发送 SIGHUP；Linux 保持 watcher。重载及配置匹配共用 8 秒预算，查询对应 `@file` router/service，核对规则、目标、entrypoint、TLS/resolver 和必要 middleware；撤销操作等待对应资源消失。
7. 记录配置匹配结果；有可靠 TLS 观测能力时独立记录证书验证，无能力时明确为未验证，不阻断配置同步。再按归属清理无引用的旧版本，保留必要的上一份版本与恢复材料。写入成功不能单独构成配置匹配，配置匹配也不能单独构成新证书生效。
8. 写入失败保留活动旧文件；提交后配置明确不匹配或配置观测超时，尝试恢复该 Route 的旧 YAML/旧证书引用并复核。首次发布失败撤下候选文件。回滚失败报告不确定状态并保留恢复材料。TLS 证书验证不匹配或无法验证单独报告，不自动触发配置回滚。
9. 进程中断后，以持久化 pending 记录和 YAML/PEM fingerprint 复核；候选证书完整才确认，恢复前核对旧证书与旧 YAML。恢复只处理对应操作，不覆盖后续发布，不读取未同步业务修改。

证书实际生效的观测边界：公开 router/service API 不能证明当前 TLS 握手使用了指定 PEM，证书单独更新时路由规则甚至不会变化。接口使用“配置匹配”与“证书验证”两个结果，不能把 API 匹配称为新 PEM 加载证明。TLS 观测应使用正确 SNI，确认实际连接到目标 Gateway 的入口，并核对预期证书 fingerprint；没有可靠入口、证书为 ACME 异步签发或缺少观测能力时如实报告未验证。每次同步不强制执行 TLS 握手，实际 TLS 验证仍是 HTTPS E2E 的验收证据。

TLS 冲突检查不能简单禁止同域名的不同路径路由：这些路由可以共用一致的证书，但 TLS 握手早于 HTTP 路径选择，不能按路径选不同证书。校验候选与其他路由的已发布证书配置，证书冲突时在文件提交前拒绝该项并继续下一项；不读取其他路由未同步的证书编辑。在 Gateway 协调范围内提交前复核共享依赖，避免并发发布绕过检查。

### Windows File provider 重载

- 实现集中在 Traefik 发布适配器，复用既有 Runtime 与 Gateway 容器定位；业务用例、前端、文件 writer 和 public port 不引入平台分支或新配置开关。
- SSH 使用目标声明的 Platform；local 使用 ComposeMountSourceDir 返回的实际 daemon mount source，区分 Windows 盘符/UNC/Docker Desktop Windows drive mount 与 Linux 原生目录。沿用现有 DooD resolver 得到实际路径，不根据 Orbit 容器内的 GOOS 判断宿主机，也不新增路径映射。
- 每次发布/撤销、失败文件恢复和 pending 候选确认/恢复，均共用“必要时重载，再等待配置匹配”的路径。容器 ID 通过该 Gateway Service 目录内的 docker compose ps -q traefik 精确取得，再经目标 runtime 执行 docker kill --signal=HUP <id>。
- SIGHUP 是 Traefik File provider 原生重读配置的机制，不重启或重建 Gateway。它重读当前文件集合，不生成全量业务快照、不覆盖其他 Route 文件、不删除未知文件或发布未同步编辑。
- 容器定位或信号发送失败使用 route_sync_reload_failed，由现有逐项确认响应报告。恢复文件后也必须完成重载与匹配才能报告 restored；恢复失败保留 pending 记录及恢复材料，后续仅重试该 Route。
- 重载及匹配合计最多 8 秒；恢复沿用独立 8 秒预算。Gateway 生命周期 VerifyPublished 仍是只读核验，不借重载重新发布业务配置。
- Linux 原生 local、Linux 原生卷的 DooD 及 SSH Linux 继续使用 watcher，不增加 Linux HUP；Linux 的文件通知可靠性后续用真实远程 SSH 验证，不将模拟测试称为实机证据。

### Gateway 新建与重新部署

- 新建 Gateway 的 `base/http/dns/http-dns` Version 初始声明 File provider、dynamic/certs 只读目录及 acme 读写目录。
- 正常流程先部署 Gateway，由有效计划创建空受管目录，再显式预览并确认 Route 发布。全新 Gateway 没有已发布 Route 时，空 dynamic/certs 目录有效，不作为部署或 readiness 失败。
- enrichment 在有效计划中结构化更新受控静态 YAML：移除 REST provider，覆盖 File provider 为固定目录且 watch=true，保留 Docker 等其他 provider、entrypoint 和 resolver。按 target 覆盖或补齐 dynamic/certs 只读和 acme 读写挂载，静态配置 IgnoreIfExists 强制为 false；不写回 Version/Service。
- API 计划/hash、worker 和 Compose 渲染共用 enrichment，保证迁移脚本初始化、普通创建和定义导入的受管 Gateway 行为一致，不按预置 ID 特判。Gateway 部署/Orbit 发起的 restart 自动使用 force-recreate，使静态受控文件替换与挂载变更生效；普通业务 Service 沿用原有选项。
- 不增加 UI 提示、更新按钮或确认参数；通过线下传达“现存 REST Gateway 重新部署一次，再通过现有入口显式同步路由”。首次切换不转换旧 REST 快照，重建到同步完成之间可能有路由不可用窗口，应按维护操作安排；部署不自动发布数据库草稿。
- 正常 Gateway deploy/restart 不再调用“从全部 enabled Route 构建快照”的 hook；改为等待 readiness 并检查已发布受管文件与运行时配置匹配。缺失已确认 YAML 不阻止 Gateway staging 准备空动态目录，但部署后核验仍报告缺失，不能用当前业务草稿填补；用户重新预览并确认 Route 同步后可重建。外部改写、pending 或证书损坏继续阻止依赖预检；没有发布记录的空目录正常。
- Workspace staging materialize 有效计划中的目录，不清空 Route 文件、证书版本、发布记录或 ACME 数据。普通业务 Service 的 Compose 渲染无需合并自定义路由。
- 受管上游沿用稳定容器名称/DNS，避免持久化可变容器 IP。若 Version 改动移除了已发布 Route 所依赖的 Component/endpoint，部署预检报告依赖冲突，要求显式处理相应路由。

### target、文件提交与权限

#### 复用现有受控文件机制

代码检查确认已有完整的文件声明到 target writer 链路：

| 已有能力 | 代码依据 | 本任务使用方式 |
| --- | --- | --- |
| 受控文件内容与权限声明 | `VersionComponentMount` 的 Content/Mode/IgnoreIfExists；`ComponentMount` proto 与 Version 内容编辑界面 | Gateway 静态 `traefik.yml` 继续使用现有 controlled_file，不新增权限表单或另一套字段 |
| mode 校验与转换 | `mount_env.go` 的 validateMountSpec/parseUnixFileMode/resolveMountSpecsForPaths | 保留四位八进制权限语义；受控挂载经已有转换得到 numeric FileMode |
| 统一待写文件结构 | `deployment_execution.go` 的 remoteWorkspaceFromRender 与 `deploymentport.WorkspaceFile` | 动态 YAML、PEM 与发布记录也使用 Path/Content/Mode/IgnoreIfExists，不另建文件权限 DTO |
| local 文件写入 | `runner/local/runtime.go` 的 StageWorkspace、SyncFiles、writeFile | Route 写入经现有 SyncFiles 进入同一个 writer；继续使用工作区边界与路径检查 |
| SSH 文件写入 | `runner/ssh/runtime.go` 的 StageWorkspace、SyncFiles、writeWorkspaceFile | 复用 SFTP、远端路径解析、Linux chmod 与平台文件提交 helper |
| DooD 路径区分 | ResolvedMount 的 LogicalSource/HostSource 和 PhysicalPathResolver | 文件写 logical path，目录挂载使用 daemon 可见 source；不新增路径映射规则 |

- Gateway 静态 YAML 继续通过 Version 的 controlled_file 承载，部署 enrichment 只覆盖受管 provider 部分并保证写入；Route 动态内容在同步确认时生成 WorkspaceFile，不写回 Version，也不在部署时从业务 Route 重新生成。
- `gateway/dynamic` 与 `gateway/certs` 仍声明为 directory mount。现有校验只允许 controlled_file 使用 Content/Mode/IgnoreIfExists；不能给 directory mount 填 Mode，也不能将热更新目录改为逐个受控文件 bind mount。
- 对每个实际父目录调用现有 `Runtime.SyncFiles`，并明确传 `pruneSuffix=""`：例如 dynamic 目录只写所选 YAML，证书版本目录只写该版本的 PEM。该接口要求文件是目录的直接子项，按目录分组即可复用，无需扩展为任意树写入。
- Route YAML、PEM 和内部发布记录都显式传 Mode，基线沿用现有 Route 写入的 `0600`。共享组读取若得到明确身份支持，仍通过同一 numeric Mode 表达，不能将 mode 为零时的 `0644` fallback 用于私钥。
- 活动 YAML 与会更新的发布记录使用 `IgnoreIfExists=false`。不可变证书版本已存在时先核对 fingerprint 再复用，不仅凭 IgnoreIfExists 跳过内容/权限检查。
- 真实部署的 local/SSH writer 遇到 IgnoreIfExists=true 时直接返回，不重新应用权限。`MaterializeLogicalMountSources/materializeFile` helper 的同名场景会 chmod，但当前仅在其测试中调用，不能据此判断 target runtime 行为，也不从 Route 用例调用它绕开 SSH。
- 文件读取/清单、精确撤销和限定归属的清理是现有 Runtime 尚缺的操作，按 Route 发布所需补充。写入、模式字段和路径解析复用现有实现，不增加平行的 Route 文件 writer，也不为 OS 限制扩展平台专项机制。

| 场景 | 写入与提交要求 |
| --- | --- |
| local 原生进程 | 使用 Environment ServiceDir 和既有 local writer；完整文件提交复用同目录临时文件机制，遵循当前平台替换能力 |
| local DooD | 写入 Orbit 可见 logical path；Compose source 由已有 PhysicalPathResolver 映射为 Docker daemon 可见 physical path；映射缺失即失败 |
| SSH Linux | 使用远端展开后的 ServiceDir、SFTP 写入及 POSIX rename；远端 Compose 使用远端目录，权限由远端执行身份处理 |
| SSH Windows | 使用远端路径、宿主机既有 ACL 和现有 SFTP 备份/替换/恢复机制；文件提交后向远端 Gateway 发送 HUP；目标文件短暂缺失为已知限制，不专项处理 |

- 补齐 target runtime 缺失的读取/清单、精确撤销和按归属清理能力，通过 route-owned port 使用；写入继续进入 WorkspaceFile/SyncFiles 的共享机制，不在业务层直接调用本地 OS 或 SFTP。
- 临时文件、权限应用与替换细节归 local writeFile 和 SSH writeWorkspaceFile/commitWorkspaceFile；Route 只声明所需内容/路径/Mode 并处理结果。StageWorkspace 与 SyncFiles 共用 writer，若为正常完整提交保留 local 草稿改动，只补充受影响共享行为的最小验证，不扩展为跨平台 writer 整改。
- 所有路径由可信 ServiceDir、Route ID 和发布 revision 构造并校验工作区边界；读取/删除请求不能携带任意路径。
- 同身份读取场景私钥使用 `0600`；明确共享组读取场景可使用 `0640`，目录与 YAML 按该组限制访问。检查父目录遍历、临时文件创建、替换/撤销权限和容器读取身份。
- SSH Linux 现有 writer 在写入正文后才对临时文件 chmod。该时序属于通用受控文件 writer 的已知问题，不由 Route 增加 SFTP/chmod 逻辑，本次不将其整改列为路由待办、专属验收或交付前提。
- local HEAD writer 的 os.WriteFile Mode 只决定新建文件权限，覆写旧文件不会自动更新权限；工作区临时文件草稿会显式 chmod，但尚未验证。不得以 helper 测试代替实际 writer 的权限证明。
- 当前 Gateway 镜像及其默认运行身份是基线，不扩展通用容器 UID/GID 模型。任意 rootless/userns 或用户自定义身份组合不自动视为已支持；身份不匹配时给出具体诊断。
- Windows writer 当前没有 ACL 声明或设置能力，依赖目标宿主机已有 ACL。本期沿用该机制，不新增 ACL 配置模型或专项检查。正常发布依据实际写入结果与 Traefik 配置匹配反馈报告失败，不以 Mode/chmod 宣称已配置 Windows ACL。
- 权限检查不读取并输出私钥正文；错误保留环境、操作、必要路径与请求 ID。沿用项目统一 HTTP 错误响应及 `route_sync_publish_permission_denied` 等现有业务错误码。
- 实际文件写入、替换或读取失败时，按单条失败记录与恢复，批量继续。已知 Windows 替换窗口或 watcher 限制不作为一概拒绝该平台发布的预检条件；不自动 sudo/chown，不递归改写工作区权限，不采用 `0777`。

## Implementation steps

1. 明确 Route-owned 发布 DTO、范围/hash/逐项结果、文件归属和发布阶段；扩展 proto 并同步生成 Go/TypeScript，调整 HTTP/MCP 契约。
2. 复用 WorkspaceFile/SyncFiles、受控文件权限字段及既有 local/SSH writer；补充路由所需读取/清单、精确撤销和限定清理，验证正常写入及 Traefik 配置匹配。Windows 正常加载补回 HUP；通用私钥临时权限时序、Windows 原生替换和 ACL 专项工作不纳入本次路由实现。
3. 将 Traefik adapter 改为按 Route ID 结构化渲染；实现证书版本写入、pending/旧文件保存、活动 YAML 提交及按归属清理，移除聚合发布和全 cert 目录 prune。
4. 改造 preview/confirm：显式范围、冻结顺序与候选、短事务复核、Gateway 协调、失败后继续、汇总编码/逐项结果、配置匹配观测及单条失败恢复；证书验证独立报告。
5. 更新新建 Gateway Version 声明；有效部署计划直接覆盖受管 provider/mount，保证静态文件覆写及 Gateway force-recreate，允许空目录部署；调整部署 hook，保持重启和重新部署只加载已发布文件。不转换历史发布数据、不增加 UI。
6. 更新列表、详情与同步弹窗：详情固定单条范围；列表显式 Project 批量预览，前端使用每项修订顺序调用单条确认并立即反馈。界面仅展示状态及失败原因提示，API/MCP 保留阶段诊断；每条响应同步已保存草稿，重试保留成功记录。
7. 根据真实变更更新活文档和 C-08/C-17 等相关决策，不提前把 Plan 草稿当作已交付行为。
8. Implementation 完成后汇报实际 diff、固定检查结果及未验证项；按标准模式等待人工验收和进入 Verification 的指令。

## Files to change

| 文件或目录 | 计划变更 |
| --- | --- |
| `internal/model/gateway.go` | 统一受管目录/provider 常量，保持 GatewayConfig 拓扑边界 |
| `internal/application/route/dto/route.go`、`port/port.go` | 显式范围、发布记录/文件端口、配置匹配/证书验证、汇总编码与逐项结果 |
| `internal/application/route/usecase/service.go`、`sync.go`、`sync_change.go` | 限定发布范围、冻结候选、删除撤销归属、协调及错误处理 |
| `internal/infrastructure/external/traefik/route.go` | 单路由 YAML、证书版本、文件清单/观测/回滚；可按现有包拆分职责文件 |
| `internal/application/gateway/usecase/initial.go` | 四种初始 Version 的 File provider 与目录挂载声明 |
| `internal/application/deployment/usecase/gateway_enrichment.go`、`command.go`、`deployment_execution.go` | 有效计划覆盖受管 provider/mount，静态配置覆写并自动重建 Gateway；部署后检查已发布文件，替换全量重发布 hook |
| `internal/application/deployment/port/port.go`、`internal/infrastructure/runner/{local,ssh,target}/` | 复用 WorkspaceFile/SyncFiles 与现有 writer，补充路由所需文件读取、精确撤销和限定清理；不加入 Windows 专用替换或权限整改 |
| `internal/bootstrap/app.go`、`deployment_runtime.go` | 注入 Route 发布文件能力和共用 Gateway 协调器，按实际依赖调整 |
| `proto/orbit/v1/route/route.proto`、`internal/api/http/handler/route/` | 请求范围/顺序、hash、确认响应编码与逐项结果映射 |
| `internal/api/mcp/delivery/route_tools.go` | 同步范围与结果；相关 service 接口一并调整 |
| `web/src/api/route/route.ts`、`views/route/RoutePage.vue`、`RouteDetail.vue` | 显式发布范围、已保存与已加载状态、部分结果/重试 |
| `web/src/components/RouteSyncDialog.vue`、`web/src/i18n/locales/{zh-CN,en-US}.ts` | 共用弹窗范围、逐项反馈与准确文案，沿用共享 UI |
| `internal/gen/proto/orbit/v1/route/`、`web/src/gen/proto/orbit/v1/route/` | 由项目 proto 任务生成，不手工修改 |
| 对应 unit/integration tests 与 `route_restart_e2e_test.go` | 验证下面列出的承诺行为；调整已有聚合发布草稿测试 |
| `docs/architecture/cd-runtime.md`、`docs/product/cd-model.md`、`docs/decisions/ledger.md` | 实现后同步 File provider、发布状态与部署语义 |
| `docs/guides/{routing-and-certificates,certificate-management,volume-mounting,docker-label-routing,deployment}.md` | 实现后说明路径、权限、新建 Gateway 部署与独立同步；按实际影响更新 |

本设计不要求数据库新增发布快照表或修改已执行迁移。`tcp.go` 的既有目标解析可以复用，不新增 listener 行为。

`internal/model/application.go`、`proto/orbit/v1/application/version.proto` 的受控文件权限字段，以及 `mount_env.go` 的校验/转换规则可直接复用，不因本任务新增字段或重做其表单。

## Verification plan

### 有效行为验证

| 场景 | 最小证据 |
| --- | --- |
| 两条路由独立发布 | A/B 均已发布；保存 B 的未同步编辑后仅发布 A；B 的 YAML/证书 fingerprint、权限与转发保持原状态 |
| 重启恢复 | 绕过 Orbit hook 直接重启独立 Traefik 测试容器；Orbit 无需参与恢复；HTTP 路由与 HTTPS 证书仍按已发布版本工作 |
| 更新/禁用/删除后重启 | A 更新或撤销后直接重启，不恢复旧配置；B 继续使用原配置 |
| 证书轮换与独立验证 | 先写新版本再替换 YAML，E2E 的实际 TLS fingerprint 符合预期；API 分别报告配置匹配和证书验证，不自动回滚配置；界面完成状态不宣称证书已生效；配置提交/匹配失败只恢复 A，B 的私钥文件不改动 |
| 同域名证书隔离 | 同域不同路径可使用一致证书；A 候选与 B 已发布证书冲突时，A 提交前失败，B 的证书与 TLS 结果保持原状态；B 未同步的证书编辑不参与 A 的发布 |
| 发布失败与中断 | 真实目录权限失败、提交/加载失败或 pending 操作中断时，保留旧文件与恢复材料；后续预览/重试只处理对应 ID |
| 范围与并发 | A 预览不因 B 的独立发布失效；同一路由过期被拒绝；发布与 Gateway 重建协调，业务事务不包含远端 I/O |
| 批量未完成 | 按预览顺序执行 A/B/C；B 失败仍尝试 C，A/C 成功保留；响应携带 `route_sync_incomplete` 及逐项状态/原因编码，前端与 MCP 报告未完成，重试只预览未完成项 |
| 新建与重新部署 | 新建 Gateway 允许空目录部署后显式同步；保存未同步修改后重建业务 Service/Gateway，持久化目录不被清空，加载此前已发布配置；声明缺失或已发布活动文件丢失有明确诊断 |
| DooD | 独立测试中 Orbit logical path 与宿主 physical path 不同，证明挂载同一目录；映射缺失时失败 |
| SSH Linux/Windows | 文件写到远端，local resolver 不参与；SSH Windows 发送 HUP，SSH Linux 保持 watcher；不安排 Windows 替换窗口或 ACL 专项验证，真实远端证据留待后续 |
| Windows 热更新 | 平台分流、发布/撤销后的 HUP、信号失败恢复/重试、pending 确认/恢复及其他路由/未知文件保留；复用独立 Docker E2E 验证实际 Windows bind mount 热更新和证书轮换 |
| 受控文件机制复用 | Route 经 WorkspaceFile/SyncFiles 传递正确 Mode 与 IgnoreIfExists，发布 A 不触碰 B 的文件权限；共享 writer 若有正常提交改动，仅覆盖其受影响行为，不将私钥临时权限时序设为路由验收 |
| 前端与接口 | 详情只传该 Route ID；列表 Project 预览，每次确认只传一项修订和草稿；第一条响应时立即更新状态，不等待整批；失败继续、仅重试失败项并保留成功记录；预览后编辑被拒绝，草稿保存边界正确；API 保留内部诊断 |

- 使用既有 opt-in Docker 重启 E2E 草稿作为起点，扩展为两条路由及 HTTPS，测试容器/目录独立命名并清理。
- Unit tests 验证路径、范围、归属与恢复决策；真实 Traefik 测试验证 watcher、加载和 TLS，不用“写入成功”断言替代。
- SSH 与 DooD 的 mock 测试不能代替真实目标证据。没有可用远端或对应容器条件时明确列为未验证，不宣称已覆盖。
- 仅测试当前支持身份和真实高风险边界，不穷举任意 UID、文件系统或已移除的 REST 发布行为。
- Windows 专用替换、ACL 平台差异及其他极边缘 OS 场景仅保留已知问题记录，不新增专项测试矩阵或验收要求。Windows 正常 HUP 加载与重启恢复通过现有独立 Traefik E2E 验证。

### 项目检查

按实现后的实际变更运行：

```text
task proto
task check
go test ./cmd/... ./internal/...
yarn --cwd web lint:fix
yarn --cwd web typecheck
yarn --cwd web test src/views/route src/components/RouteSyncDialog.test.ts src/components/RouteCertificateDialog.test.ts
```

`task proto` 仅在契约变更后用于生成；前端固定检查按仓库要求执行。真实重启 E2E 使用 `POMELO_ORBIT_TRAEFIK_E2E=1` 开关，在隔离测试容器运行；不操作用户服务。已执行结果记录在 Implementation notes，不表示进入正式 Verification。

## Blockers and assumptions

暂无需要用户进一步选择的方向性问题。批量等待预算、证书观测入口等细节按现有机制和实际 target 能力在实现中确定；下列运行前提及验证边界不要求新增用户审批，已确认的 OS 限制不转为整改任务。

- 当前单节点 API/worker 是协调与发布唯一写入者的前提；本计划不承诺多 Orbit 实例共同写同一目录。人工修改受管文件需通过 fingerprint 检测，而非静默覆盖。
- 当前 Version、目录写入身份与 Gateway 读取身份，在执行时必须可定位；当前不需要用户提前选择具体机器，但真实环境证据是后续验收条件。
- 现有 mode 控制文件权限，不控制 directory mode、owner/group 或 Windows ACL。复用该机制即可覆盖基线文件写入，但不能把它等同于所有挂载读取身份都已验证。
- SSH Windows 继续使用现有备份/移动/恢复；短暂文件缺失属于已知限制，不专项修复。用户补充授权的正常发布 HUP 重载纳入实现，远端实机仍未验证。
- TLS fingerprint 观测方式在实现中按实际 target 能力确定，不作为配置发布的强制前置条件。router/service API 不能替代证书生效证据；缺少可靠目标入口或观测能力时 API 结果必须为未验证，HTTPS E2E 仍需实际 TLS 证据。
- 顺序批量的观测/恢复耗时必须受预算约束，并与现有客户端超时配置配合；网络断开无法保证返回结果，依靠发布记录和重新预览定位已执行项，不因此扩展为异步任务系统。
- 工作区聚合 `routes.yaml` 草稿在实现中直接收敛为单条格式；本任务不实现它在历史目标上的转换或清理流程。

## Risks

- File provider 按文件合并仍共享资源命名空间和 TLS store；生成前检查冲突，提交后检查加载，失败只恢复本次 Route。同一 TLS 域名使用不同证书的冲突不能靠分文件隔离。
- ACME 存储与 TLS SNI 是 Gateway 共享资源；Route 文件归属独立不表示可以为同一个域名按不同 URL 路径分别选择证书。
- 文件替换保证完整提交不等于数据库、文件与 Traefik 的跨系统原子事务，也不保证一次发布绝无短暂配置切换。状态与恢复结果必须如实报告。
- remote 文件关闭/提交与宿主机突然掉电的耐久性取决于文件系统和协议能力，不能仅凭 rename 承诺所有崩溃场景的数据持久性。
- SSH Windows 替换活动文件时，旧文件已移走而新文件尚未放入的瞬间可能没有目标文件；该短暂窗口作为已知 OS 限制接受，文件完整提交后再发送 HUP，不新增替换算法。
- SSH Windows ACL、宿主机 UID/GID 映射、已有目录权限和容器 bind mount 可读性可能不同；宿主机检查通过仍需容器加载证据。
- 现有部署 snapshot 与显式 Route 发布的生命周期不同；替换 hook 时需检查所有 worker/内部调用点，避免遗漏的全量重发布触发未同步编辑。

## Rollback

- 普通单条配置发布失败：只恢复该 Route 的上一份 YAML、证书引用及发布记录，保留新版本供诊断/重试；复核其他路由文件未变化。证书验证不匹配或未验证独立报告，不自动恢复配置。
- 批量单条失败：记录并尽力恢复失败项后继续下一条，保留已成功项，不执行全 Gateway 回退；汇总未完成编码与逐项恢复/清理结果。清理失败和回滚失败分别报告。
- 正常重新部署不删除持久化目录；业务版本回滚也不能从旧 deployment snapshot 覆盖当前已发布 Route 文件。
- Plan 阶段不撤销或覆盖已有用户改动。本工作区此前产品草稿在 Implementation 时按本计划逐项调整，不使用整体 Git 回退。

## User review notes

- 2026-10-01：用户采纳按稳定 Route ID 分 YAML、按路由及版本分证书的结构，要求写入 Plan；Intent 标记为 Accepted。
- 2026-10-01：用户提醒已有受控文件及权限机制；已核对 controlled_file、WorkspaceFile、StageWorkspace/SyncFiles 与 local/SSH writer，并将 Plan 修正为直接复用现有模型及写入链路，仅补充发布所缺能力。目录权限、Windows ACL、IgnoreIfExists 与私钥临时文件权限时序的实际边界分别记录。
- 2026-10-01：用户明确“不处理历史遗留环境”；已移除首次切换、旧快照/证书转换及历史回填，采用新建 Gateway 空目录部署后显式同步的正常流程。
- 2026-10-01：用户选择“分别报告配置匹配与证书验证结果”；已移除每次同步必需 TLS 观测的前提和证书验证失败自动回滚配置的联动，保留独立报告与实际 HTTPS 验收证据。
- 2026-10-01：用户明确“顺序处理，跳过失败，有未完成的情况报告在响应里携带编码前端报告”；已规定冻结顺序、失败继续、保留成功项，以及专用确认响应的汇总编码和逐项结果。
- 2026-10-01：用户质疑私钥临时权限是否属于路由职责；将其归为通用 writer 已知问题，路由仅声明 Mode 并复用接口，不列专属业务处理、验收或交付前提。
- 2026-10-01：用户明确“极边缘场景及有操作系统限制的，属于已知问题，不投入精力”；移除 Windows 原生文件替换实现、watcher/ACL 专项验证及相关阻塞项，沿用现有 target runtime 并记录限制。
- 2026-10-01：用户支持复用现有“写临时文件再替换”，并询问剩余不明确事项。确认暂无需要用户再决定的方向性问题；正常发布验证、批量预算及证书观测能力在实现中落实。该支持针对文件提交方式，不据此将整个 Plan 标记为 Accepted 或进入实现。
- 2026-10-01：用户要求“开始实施”；Plan 标记为 Accepted，进入 Implementation，按现有代码完成接口字段和预算设计并运行仓库固定检查。正式 Verification 阶段仍等待实现后人工验收指令。
- 2026-10-01：用户选择本次改为发布/撤销清单；预览移除运行时差异与 hash，保留业务/文件/依赖修订、已删除 Route 撤销和未知文件保留。更新 API/MCP、生成契约、前端及相关测试。
- 2026-10-02：用户在实现结果汇报后要求“补验证文档”；进入 Verification，核对最终 diff 并补充 [验收记录](../verification/20261001-route-file-provider-persistence.md)。
- 2026-10-02：用户明确预置指迁移脚本初始化的受管 Gateway，决定部署直接覆盖受管 provider/mount，重新部署要求线下传达，不增加 UI；按此决定修订计划并回到 Implementation。
- 2026-10-02：用户要求补回切换 REST 时删除的 Windows HUP；本地必须区分 Windows/Linux，DooD 判断实际挂载来源，变更集中在 Traefik 适配器。Linux HUP 暂不实现，后续基于远程 SSH 实测。
- 2026-10-02：用户要求结果只展示一个完成状态，失败时在状态上 tip 展示原因；调整前端报告计划，API 内部阶段与诊断 ID 保留，发布/撤销/跳过预览、失败重试和草稿保留不改变。
- 2026-10-02：用户要求前端顺序循环，每条响应立即调整列表记录，提升长时间同步的反馈；按此要求补每项修订并改前端调度，不引入异步后台任务或页面/弹窗关闭后的管理能力。当前仍为 Implementation。
- 2026-10-02：用户要求把 HTTP/HTTPS 拆为独立列；扩展结构化规则的协议字段，直接供清单展示，保留当前同表逐条反馈及重试实现。当前仍为 Implementation。
- 2026-10-02：用户要求 Traefik 路由文件名及内容资源名改用编码，后续要求证书方式移入协议列；按最新决定统一文件改名、撤旧、失败/中断恢复及编码占用，稳定 ID 继续管理证书和内部归属。
- 2026-10-02：用户发现删除动态目录后无法重新部署和同步；调整已确认文件缺失的处理，允许 Gateway 准备空目录及显式确认重建选定 YAML，仍报告部署后的未恢复文件和外部改写，失败恢复缺失基线。
- 2026-10-02：用户要求扩大同步模态窗；最大宽度由 760px 扩至 960px，表格最大可见高度由 256px 扩至 384px，沿用共享弹窗的小屏边界和滚动。

## Implementation notes

- 2026-10-02：用户采纳 Traefik API 命名，并要求把 API 从基本信息拆出独立卡片；详情展示、编辑状态、校验和提交范围按两个分区组织，补充正常保存与字段校验回归。

- 已实现独立 YAML（后续按用户要求改为编码命名）、不可变证书版本、pending/previous 恢复、按 Route 清理、Gateway 声明及实际挂载核对；共用现有 target writer 与 DooD 路径 resolver，SSH 写入链路未新增权限策略。
- 已实现显式范围与发布/撤销/跳过清单；批量预算 105 秒，单次配置匹配/恢复最多 8 秒；失败继续并返回逐项结果。列表/详情保留未保存草稿，重试只处理未完成 ID。
- 已实现发布引用的部署保护和 API/worker 共用协调器；Gateway 重建核验持久化状态，不读取业务 Route 自动重发。发布记录限定 Project、Environment target revision、Gateway Application 和所在 Service 目录。
- `task proto`、`task check`（Go lint 0 issues）、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck` 均通过。Route 同步/域名、证书弹窗、Gateway 表单/导航 6 个前端测试文件共 18 项通过；追加的未知 YAML/证书文件保留断言及相关后端测试也通过。
- 范围隔离、A/B/C 失败继续、已删除撤销、禁用未发布跳过、依赖变化、权限失败/重试、证书复用/冲突/恢复、协调器共享、前端失败后草稿保留都有自动化覆盖。权限失败使用 adapter 错误注入，尚无真实远端目录权限失败证据。
- 此前 File provider 实现的真实 Traefik 3.6 Linux DooD 隔离测试通过（21.44 秒）：Orbit 路径与 daemon physical mount source 不同；直接重启、A 更新/撤销、B 文件保留、手工证书实际 TLS fingerprint、证书轮换及 Gateway force-recreate 均有证据。测试容器、网络、临时卷、测试镜像及编译产物已清理；本次部署覆盖补充未重跑该实机测试，不将其视为 REST 首次切换证据。
- 真实 SSH Linux/Windows 目标未提供，暂无远端运行证据；适配器测试验证 target 分流与 Mode，不能代替真实 SSH 验收。Windows 正常发布 HUP 已由用户补充授权，替换/ACL 与极边缘 OS 限制仍不专项处理。
- 此前 Windows 原生 E2E 未取得配置加载成功；修正通用 YAML 渲染及 TLS API 归一化后在 Linux DooD 重跑通过，未对 Windows 再次运行或专项定位。该限制不表示已验证 Windows watcher。
- 组合根沿用既有 RouteManager 注入，无需修改 bootstrap；跨 API/worker 管理器的协调由共享进程锁实现。未新增数据库迁移或兼容逻辑；当前 HTTPS 正常同步的证书验证为 `unverified`，实际 TLS 证据由上述 E2E 提供。
- 首次 Implementation 阶段未启动开发服务器、未操作用户服务、未创建 Verification 文档或执行 Git 写操作。用户随后要求补验收记录，又明确部署直接覆盖；当前阶段为补充 Implementation。
- 本次已实现共享 enrichment 覆盖受管 provider/mount、IgnoreIfExists=false、Gateway deploy/restart 自动 force-recreate；计划/hash、worker 和渲染一致，其他声明与库存不改写。自动化验证 REST/File 替换、其他声明保留、挂载替换/补齐、hash 稳定、实际 staging 和命令、普通 Service 不受影响。`task check`（0 issues）和 `go test ./cmd/... ./internal/...` 再次通过；未修改 UI、已执行迁移或部署用户服务。
- 本次 HUP 补充集中在 Traefik 发布适配器，复用既有 Runtime/容器定位；Windows 发布、撤销、恢复及 pending 共用重载再匹配，合计 8 秒加载预算。local Windows/Linux、Windows bind mount DooD、SSH 平台分流及失败重试回归通过；Linux 不增加 HUP。
- 最终 `task check`（0 issues）与 `go test ./cmd/... ./internal/...` 通过；Windows 原生独立 Traefik E2E 通过（21.10 秒），HTTP/TLS 热更新、证书轮换、直接重启与 force-recreate 后恢复，以及发布前后容器 StartedAt 不变均有证据。修正 E2E 重启后的随机端口刷新与等待时机；隔离实例已清理，没有操作用户服务。
- Windows DooD/SSH 的本轮证据限于模拟分流与发布/恢复；真实远程 SSH Linux/Windows 留待后续测试。此前“Windows watcher 不专项处理”仅保留为历史决策，正常 HUP 热更新由最新授权纳入实现。
- 同步结果已简化为路由名称与完成/失败状态，失败原因使用共享 `AppBadge` 的 `title` 提示，移除内部阶段和请求/操作 ID 的可见展示及无用翻译。后端/API 不变；`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 和两个相关前端测试文件的 11 项测试通过，覆盖原因提示、失败重试及列表/详情未保存草稿保留。
- 用户随后要求逐条反馈，已在每项预览中提供独立业务/发布修订，复用单条计划计算及现有确认用例；Web 顺序调用单条确认，立即显示待处理/处理中/完成/失败并按响应通知父页面更新草稿。原预览修订保留，不自动重新预览或发布编辑后的内容；成功记录在失败重试后保留。
- 本轮 `task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`（0 issues）和 `go test ./cmd/... ./internal/...` 通过；两个前端测试文件 14 项通过，覆盖延迟响应逐行反馈、权限/预览过期/网络失败后继续、原修订提交、每条保存反馈及只重试失败项。新增后端回归覆盖 Project 清单拆分确认的发布/撤销/跳过，以及编辑只使对应项过期。
- 本轮补充结构化协议和证书列、编码文件及资源名、改名/恢复/编码复用和缺失文件显式重建；文件转换集中在 Traefik 适配器，复用已有 target writer。`task check` 最终通过（0 issues）、全量 Go 回归通过、两个同步前端测试文件 16 项通过；模态窗扩大后再次执行前端 lint:fix 和 typecheck，通过。
- Windows 原生独立 Traefik 3.6 E2E 再次通过（27.41 秒），除既有 HTTP/TLS、直接重启、证书轮换和 force-recreate 外，新增 ID 不同于编码的改名撤旧，以及删除整个动态目录后 staging 创建空目录、重建 Gateway、显式同步恢复及 TLS 核验。测试只操作隔离实例，未操作用户服务；真实 SSH、Windows DooD 和浏览器视觉验收仍未补齐。
