# Plan / 计划：服务受控文件内容与写入策略覆盖
最后修改时间: 2026-11-02 21:39:17

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

依据[已接受意图](../intent/20261102-service-controlled-file-overrides.md)。用户已授权实现及本轮整体审查，当前进入 Verification；由主会话复核两项最终交付和后续产品修订，结果登记于[验证报告](../verification/20261102-service-controlled-file-overrides.md)。

本轮产品方案覆盖服务受控文件通用能力；Gateway 调整分析直接在本计划完成，不能用“以后再看”替代。实际业务流程改造保持在通用能力之后，本文明确其目标和实施顺序，不将分析交付表述为业务代码已调整。

## Technical decisions / 技术设计与决策

### 1. 字段存在性与合并

继续以版本挂载的 `target` 标识服务覆盖。`source_type`、`target`、`shared` 来自版本，不扩展为服务任意增加挂载或改变类型。

| 字段 | 版本声明 | 服务覆盖 | 规则 |
| --- | --- | --- | --- |
| `source` | string | optional string，已有 | 未设置继承；显式值必须符合来源类型及路径规则 |
| `source_is_host_path` | bool | optional bool，已有 | 独立继承；与最终来源路径合并校验；受控文件只能 false |
| `content` | string | optional string，新增 | 仅受控文件；空字符串是显式清空，不能折叠为继承 |
| `mode` | string | optional string，新增 | 仅受控文件；显式值必须为 `0000`–`0777` 四位八进制，空串无效 |
| `ignore_if_exists` | bool | optional bool，新增 | 仅受控文件；false=覆盖，true=存在则保留 |
| `read_only` | bool | optional bool，新增 | 目录、普通文件等挂载支持；受控文件不支持，校验拒绝该设置 |

Go 使用指针、Proto 使用 optional、SQL 使用 NULL、TypeScript 使用 undefined 表达继承，不引入第二套状态枚举。Proto 沿用旧字段号，新增字段顺序追加；声明响应追加内容、权限和策略，已有只读值继续返回。

`state=override` 表示至少一个字段显式设置，其余字段继承。取消当前强制同时填写 source/source_is_host_path 的要求；合并后再执行完整校验。不接受所有字段均缺省的 override，全部继承应省略该条目。`state=deleted` 不允许携带任何覆盖值。

显式值与版本相同也保留；移除当前按来源相等丢弃挂载覆盖的归一化。不扩大修改其他环境变量、命令或资源的归一化规则。恢复单字段继承仅清除对应值；恢复整条挂载清除覆盖或删除状态。

保存服务覆盖和构建有效计划复用同一挂载合并/校验规则，放在 model 的挂载规则范围，保持 Proto、SQLC 和文件操作在各自边界。不为此次改动增加泛化覆盖框架。内容保持字面值，不 trim，不插值；256 KiB 按 UTF-8 字节校验。

版本切换沿用按组件名重映射和按 target 匹配：仍存在且兼容的覆盖保留；目标移除或类型变化使显式文件字段不再适用时，返回明确校验错误，用户先恢复继承或调整配置。不得静默丢掉自定义内容。

### 2. 持久化和接口

在 `service_component_mount` 新增 nullable `content`、`mode`、`ignore_if_exists`、`read_only`。SQLite/PostgreSQL 内容使用 TEXT；MySQL 实际迁移使用 LONGTEXT，避免 256 KiB 内容被 TEXT 上限截断。同步 SQLC schema 镜像与查询，以实际迁移为运行表结构依据。现有记录新增字段为 NULL，来源覆盖原值保留。按用户追加要求直接简化本次 `000056` 迁移，并就地调整已执行该迁移的本地库；不新增修补迁移。数据库保留主键、非空、既有外键和 state 枚举等简单约束，override/deleted 与字段值的组合关系、内容大小和权限格式由现有 model 规则负责；down 恢复旧版本结构。

按用户最新确认，受控文件不支持只读：`000056` 将既有受控文件版本声明的 `read_only` 清零，不影响目录或普通文件；版本保存、服务覆盖及部署校验拒绝受控文件只读。此数据调整不可由 down 恢复。已检查开发库 59 条受控文件均未启用只读，受控文件只读覆盖为零，无需对该库执行额外修改。用户明确不为迁移编写测试。

Repository 的替换写入沿用当前事务和项目范围约束；新增字段贯通读取与插入，不能用默认空串/false 替代 NULL。迁移 down 明确会删除这些新覆盖数据，不作为无损回退手段。

HTTP 沿用组件详情和整体 overlay 更新接口，返回版本声明与服务稀疏覆盖两部分。更新仍为完整替换而非 PATCH，未提交的旧字段覆盖被清除；调用方须保留无关 env/resources/endpoints/runtime 覆盖。

MCP 当前 `serviceComponentOutput` 未返回 mounts 等完整集合，且没有服务组件详情读工具。仅扩展输入会留下无法可靠回读和保留其他覆盖的问题，故本任务同时：

1. 新增 `orbit_get_service_component(service_id, component_id)`，复用现有 `GetServiceComponent` 应用方法和 Project/成员校验，返回版本声明与完整服务覆盖。
2. 完善服务组件输出中的 env、mounts、resources、endpoints，使写入结果及详情均可核对；不通过 HTTP handler 复用映射，不让应用依赖 Proto。
3. 更新 overlay 工具说明，注明字段存在性、空内容、false、全量替换、内容上限、保留模式初始化语义；调用前可通过新读工具保留无关字段。
4. 更新依赖接口、工具注册及对应协议测试；工具数量断言跟随真实新增，不保留历史固定数量。

### 3. 有效计划和文件执行

产品统一使用“覆盖已有文件 / 保留已有文件”，分别对应既有 `ignore_if_exists=false/true`。不增加第三项、策略字段、兼容别名或转换层；本轮契约和表结构无须改变，已迁移开发库保留现有内容及策略值。后续确需改动本次迁移时，按用户授权就地修改。本轮同时取消 Gateway 对该布尔值的强制覆盖，预览、计划 hash 与实际文件准备使用同一有效策略；其选版、内容补丁和重建逻辑不因此视为已完成整体治理。

合并结果仍使用 `VersionComponentMount` 作为有效挂载。内容、mode、只读和策略进入现有 EffectiveServicePlanHash 的 Mounts 序列化，不另造内容哈希或 Compose label。核对详情 pending_deploy、受理部署、worker 重建计划和 hash 校验的共同输入，证明新覆盖不漏检。

继续由 `remoteWorkspaceFromRender` 传递内容、权限及 IgnoreIfExists，local/SSH `StageWorkspace` 先准备文件再运行 Compose：

| 策略 | 不存在 | 已存在 |
| --- | --- | --- |
| 覆盖 | 创建父目录并初始化 | 写临时文件后完整替换，应用指定权限 |
| 保留 | 同样初始化 | 不修改内容和权限 |

路径为目录、无写权限、取消和传输失败沿用真实错误传播。保留模式只代表部署时不覆盖，不是远程文件锁，也不承诺跨进程原子 create-if-absent。多个文件不是事务，不新增整批文件回滚机制。

现有 mode 校验允许 `0000`，writer 却把数值 0 转为 `0644`。本任务修正这一直接影响完整表达的问题：`WorkspaceFile.Mode` 明确为实际权限，移除 writer 的零值兜底；核对全部生产调用方，Compose、Route 等调用显式传入原本要求的 mode。Linux 验证 `0000`；Windows 沿用平台可表达的权限行为，不承诺 POSIX 权限等价。

不自动启用强制重建：单文件 bind mount 的源文件被重命名替换后，普通 `up` 可能沿用旧挂载；用户通过现有部署选项选择重建。文件写入策略先执行，重建不能越过“保留”。相关说明放在部署/挂载操作文档，编辑成功仅表示声明已保存。

### 4. 前端编辑

沿用 `ServiceComponentDetail.vue` 挂载入口及项目 Dialog/Drawer、`MonacoEditor`。提取局部 `serviceComponentMount` 表单转换模块，承担声明+覆盖到编辑状态、字段恢复和稀疏请求生成，不另建全局表单框架。

沿用原挂载弹窗布局，展示类型、来源、目标；目录/文件保留只读及宿主路径选项。受控文件直接展开内容、权限及写入策略，不展示只读。弹窗不展示继承状态、字段来源或自定义开关；内部仍按字段存在性保存，用户修改即记录为覆盖，未操作字段保留原设置，不按与版本相等清除。挂载列表的重置恢复整条版本设置。

版本内容编辑和服务挂载编辑的写入策略均只有“覆盖已有文件”和“保留已有文件”两项，使用同一控件和文案，移除版本入口的“已存在时忽略”勾选。保留模式下内容仍可编辑，明确提示“已有文件不变，修改后的内容和权限仅在文件不存在时初始化”；不读取真实服务器文件。只读仅用于非受控文件挂载。

请求只提交显式覆盖字段，全继承条目省略；保持页面既有提交边界，取消弹窗不改变已保存值，失败保留草稿。保留无关覆盖，错误显示在对应字段。代码编辑统一 12px，其余文字沿用共享样式。

## Gateway analysis / Traefik 与 Gateway 配置和策略分析

### 当前覆盖顺序与证据

Gateway 的实际链路是：选择 ACME profile 对应版本 → 合并服务覆盖 → 注入 Gateway 配置快照 → `enrichGatewayPlan` → 计算/核对有效计划 hash → 文件物化 → Compose → readiness。预览也经过 enrichment，而服务声明编辑本身不展示这次运行时改写。

| 当前行为 | 代码依据 | 与用户配置的关系及风险 | 调整结论 |
| --- | --- | --- | --- |
| 生成四类普通版本和静态文件、三个目录挂载 | `gateway/usecase/initial.go` | 作为初始声明可被理解和编辑 | 保留业务初始化模板，默认值写入普通版本，不让部署时重复修复用户配置 |
| 按 GatewayConfig.AcmeProfile 在部署前切换服务版本 | `gateway/usecase/deployment.go:SelectGatewayDeploymentVersion` | 可覆盖服务已选择版本；此重映射当前未走普通服务完整覆盖校验 | 目标是显式业务配置动作选择版本并调用统一校验；部署使用已保存服务版本，不在部署入口暗中改选 |
| 替换 `providers.file`、移除 `providers.rest`、覆盖 resolver email | `deployment/usecase/gateway_enrichment.go`、`deployment/render/gateway.go` | 即使服务自定义 content，最终字面内容仍被改写；只删除 IgnoreIfExists 赋值无效 | 目标是初始化或显式应用 Gateway 配置时生成可查看的普通声明/服务覆盖，部署不再修改其内容 |
| 强制 `IgnoreIfExists=false` | `gateway_enrichment.go:applyGatewayPlan` | 服务选择保留仍会被写入，违背通用策略 | 本轮移除此强制赋值，按版本+服务的有效策略执行；保留旧文件可能导致新的 Gateway 配置不生效，接受并明确说明 |
| 重置/补齐 dynamic、certs、acme 的整个挂载项 | 同上 | source、read_only、删除选择会被覆盖；未来扩展也会丢失 | 不在部署时覆盖挂载；模板提供初始值，实际配置错误由发布校验或运行错误反馈 |
| 注入/覆盖 `CF_DNS_API_TOKEN` | 同上 | 显式服务环境覆盖失效；Gateway token 同时供 Cloudflare DNS 管理使用 | 部署凭据改为普通版本占位符+Service env 的显式配置；DNS 管理凭据保留其业务职责，不能简单删除 GatewayConfig token 字段 |
| Gateway 始终 force-recreate | `deployment/usecase/command.go`、`deployment_execution.go` | 无论用户选项都重建，当前目的之一是刷新被替换的静态文件挂载 | 目标是部署服从通用选项；业务操作若需要重建，明确传入部署参数，不在多处按 Gateway 身份 OR true |
| ACME 使用 `./gateway/acme:/letsencrypt` 目录挂载 | `gateway/usecase/initial.go` | Traefik 自行写 `/letsencrypt/acme.json`，当前没有部署模板覆盖该文件 | 保留这一初始目录声明；不为 ACME 新增专用受控文件豁免、不自动搬移已有证书状态 |
| Route 动态配置、上传证书与发布记录由 Route 发布生成并写入 | `infrastructure/external/traefik/publication*.go` | 是 Route 业务发布产物，不是版本受控文件的另一份 content | 保留显式 Route 发布流程及其回滚/所有权边界，不把所有动态文件改成受控文件或交给部署覆盖 |
| Route 发布校验固定 provider、目标及物理目录 | `publication.go:ValidateGateway` | 若只取消目录覆盖，用户自定义路径可能使发布失败 | 当前固定布局可以作为 Route 发布能力的可检查要求；配置可保存，但不满足该能力时明确失败，不能悄悄改回。支持任意布局需另改发布路径解析，非通用文件任务的隐含能力 |
| Windows/DooD 下 Route 发布后执行 HUP | `publication.go:reloadFileProvider` | 是已请求的动态配置发布操作，与部署 force-recreate 不同 | 不随静态文件策略一起删除；独立验证发布的实际加载结果 |

表中应用文件路径分别相对 `internal/application/`，Traefik 适配器路径相对 `internal/`。以上均来自当前实现，不以历史设计文档替代事实。

### 明确目标及调整范围

Gateway 应作为通用配置能力的使用者：模板负责初始配置，用户决定普通声明和服务覆盖，通用部署负责执行；Route 发布负责自己的动态产物。平台不根据 `traefik.yml`、`acme.json` 文件名替用户选择写入策略。

保留真实业务能力校验与用户自行配置并不冲突：例如 Route 无法向一个未启用 file provider 的网关发布，可以明确失败；不能通过部署静默修改用户文件让其“满足要求”。用户可保存不适用于该业务的配置并承担结果。

`acme.json` 如由用户自行配置为受控文件，与任何受控文件同样支持保留/覆盖及权限。目录挂载通常更适合应用自行重命名文件，但不因用途禁止用户选择单文件挂载。本任务不承诺具体 Traefik 版本的单文件写入方式。

GatewayConfig 的 profile/email/token 当前是独立配置来源，不能简单把 enrichment 删掉：邮件不会再注入、DNS token 不再进入容器、profile 选择可能与路由证书要求不一致。目标方案是将“生成并应用 Gateway 部署配置”变成显式业务动作，输出用户可查看的普通配置；不在每次部署执行隐式双向同步，也不提供长期新旧路径并存开关。

### 通用能力完成后的业务调整顺序

1. 初始化入口生成完整普通版本/服务配置：已有 provider、目录和 ACME 结构保留；需要的邮件写入生成内容，DNS token 通过已有 `${KEY}` Service env 契约传入，文件内容自身不做变量模板替换。
2. Gateway 配置更新明确区分其业务元数据和“应用到部署配置”的动作。后者输出版本选择、服务内容和环境差异，明确替换由该动作选择的字段；不把普通部署当作同步触发器，不盲目覆盖其他用户配置。
3. 对已有 Gateway 先读取其版本、服务覆盖、GatewayConfig 及当前生成结果，形成可审查的普通配置转换。仅用新建模板不能覆盖存量；转换不读取/重置 ACME 数据，不自动触发部署。
4. 同步移除部署受理、预览、worker 中的配置改写，以及按身份强制重建；以同一已保存声明构建有效计划和 hash，确保快照重验一致，删除不再有生产用途的补丁函数及旧断言。
5. 保留 Route 发布/证书产物及固定布局能力校验；部署错误配置时不补齐，Route 同步返回实际不满足条件的原因。任意路径支持与此改造分开，不默认获得该能力。
6. 通过静态文件“覆盖/保留”、邮件/token 更新、显式选版、重建选择、Route 发布及 ACME 数据保留的业务验证后，再称 Gateway 调整完成。

### 本任务交付边界与验收证据

A10 的分析由本章交付，整体 Verification 复核其与通用实现的实际差异。用户本轮确认两项统一写入能力，主会话取消 Gateway 强制覆盖例外，并补充从有效策略到 WorkspaceFile 的测试；其余 Gateway 产品改造按上述顺序展开，不将本次策略修正宣称为其内容、选版、目录和重建逻辑整体治理完成。

Gateway 后续验收至少包括：服务选择保留时既有静态文件不变；显式内容不被 YAML 补丁改写；source/read_only/删除选择不被补齐；profile/邮件/token 配置来源唯一且可查看；部署选项与实际命令一致；默认布局 Route 正常发布；自定义不满足发布条件时返回真实错误；重复部署及重建不改 ACME 状态；受理与 worker hash 一致。

## Implementation steps / 实现步骤

1. 接受计划后，T01 完成协议、增量迁移、Repository、覆盖合并、HTTP/MCP 读写及 writer 权限一致性。
2. T02 基于已核对并冻结的源协议并行完成服务编辑、稀疏请求及用户文档；生成物由 T01 独占，T02 typecheck 在生成完成后执行，整体集成须两项交付均接受。
3. 主会话对照 A01–A10 做跨端核对，报告实际差异与开发检查；按阶段授权进入整体 Verification。
4. 按用户本轮追加要求统一版本和服务的两项策略入口，取消 Gateway 强制覆盖例外，验证策略在重部署和强制重建中不被修改；其余 Gateway 业务调整继续按本章分析单独处理。

## Files to change / 需修改文件

- T01：`proto/orbit/v1/service/service.proto`、`internal/model/{service,mount}.go`、`internal/application/service/usecase/service.go`、`internal/application/deployment/usecase/{effective_service_plan,mount_env}.go` 及相关测试。
- T01：`sql/migration/{sqlite,mysql,postgres}/` 新迁移、`sql/schema/{sqlite,mysql}.sql`、`sql/query/service/service.sql`、`internal/repository/impl/sqlc/service/`。
- T01：`internal/api/http/handler/service/`、`internal/api/mcp/delivery/`、`internal/gen/proto/`、`internal/gen/sqlc/service/`、`web/src` 下对应生成 Proto；`internal/infrastructure/runner/{local,ssh}/` 和需要显式 mode 的调用方。
- T02：`web/src/views/service/ServiceComponentDetail.vue`、新增 `serviceComponentMount.ts` 与轻量测试、必要局部共享 UI、`web/src/i18n/locales/{zh-CN,en-US}.ts`、`docs/handbook/reference/compose.md`。
- 主会话：本功能意图、总计划及后续验证报告；本轮追加统一策略控件、版本内容入口、Gateway 强制覆盖修正及相应测试。Gateway 其他业务改造不在本轮范围。

## Tasks and coordination / 子任务与协调

### Coordination / 协调信息

- 方案修订：3。修订2在 17:23:30 固定 Proto 字段并按文件边界并行；修订3根据用户确认，受控文件不支持只读，目录和普通文件保留只读，由主会话直接完成并重新核对 T01/T02 的集成结果。
- 权威协调根：`D:/SourceCodes/mywork/pomelo-orbit`。
- 仲裁会话：当前创建本计划的主会话。
- 阶段授权：2026-11-02 用户“开始实现”，授权实现本计划并允许通过 codex-cli 向同目录会话“查看华为云机器上部署的容器应用”分配任务和读取进度。包括必要开发检查，不包括实际部署、Git 提交或 Gateway 业务策略改造。
- 协议入口：`C:/Users/wangm25/.codex/plugins/cache/specflow-local/specflow/0.2.1/skills/specflow/SKILL.md` 及同目录 `references/task-coordination.md`。
- 后续执行者只写指定 task 的接收、进展与交付记录；主会话维护本计划登记。默认由用户转交可复制摘要，无自动跨会话通知。

### Task definitions / 任务定义表

| ID | 任务文档 | 直接前置任务与要求 | 预期交付与验收 | 意图验收归属 |
| --- | --- | --- | --- | --- |
| T01 | [后端覆盖契约](../task/20261102-service-controlled-file-overrides/T01-backend-overrides.md) | 无；实施前方案与阶段需已接受/授权 | 完整可空字段读写及合并、MCP 回读、writer 权限一致性；必要后端检查通过且交付版本可用 | A01–A04、A06–A07，A09 后端部分 |
| T02 | [服务编辑交互](../task/20261102-service-controlled-file-overrides/T02-service-editor.md) | 无前置实现任务；输入为主会话本次核对接受的 source Proto diff 与本计划固定字段合同，生成物仍由 T01 提供 | UI 与稀疏请求一致，继承/恢复/保留文案明确，轻量单测和生成完成后的前端质量检查通过 | A05，A02–A03 交互部分，A09 前端部分 |

A08 由主会话组织整体运行边界核对，A10 由本计划 Gateway 分析及整体复核负责。没有未归属验收。

### Execution registry / 执行登记表

| ID | 执行状态 | 执行者 | 分配编号 | 工作基线 / 交付位置 | 仲裁结论 / 当前阻塞 |
| --- | --- | --- | --- | --- | --- |
| T01 | accepted | 会话 01a19ae7-d2b0-7241-a111-42e61e551fbf | T01-20261102-170337-01a19ae7 | 同目录 HEAD 09eda33e8d19ab6a255e8088db053829dd1c0d0d + 后端/迁移/生成物未提交 diff | 17:40 读取 T01 17:39:28 交付，置 submitted；17:41:42 主会话审查实际diff、测试及最终task check通过，接受实现交付；外部环境未验证单列 |
| T02 | accepted | 主会话 01a19b5a-d828-7fc1-abfb-aed164b1fb46 | T02-20261102-172330-main | 同 HEAD + 前端手写代码、语言包、使用文档未提交 diff | 17:39:07 交付置 submitted；17:41:42 复核稀疏序列化、局部表单、197测试和task check通过，接受实现交付 |

### DAG and parallelism / DAG 与并行安排

修订2的两个实现节点无直接依赖，可并行，均输入本计划固定契约，最后统一集成；无自依赖或环。两项在同一工作区按文件范围隔离：T01 独占后端、Proto、SQLC及生成TS，T02 独占前端手写代码、语言包与使用文档。生成完成前不执行依赖新DTO的typecheck；不并行运行全仓自动格式修复。必要契约修正由主会话仲裁并复核两端，不能单方修改。

主会话负责集成和总体验收。派发前核对有效授权、唯一分配、前置审查证据及可访问版本，不能把方案 Accepted 当作实现 accepted。

### Assignment history / 分配历史

- 2026-11-02 17:03:37：分配 T01 给同目录会话“查看华为云机器上部署的容器应用”（01a19ae7-d2b0-7241-a111-42e61e551fbf），编号 T01-20261102-170337-01a19ae7，方案修订 1。用户本轮明确授权实现与消息分配。可写范围为 T01 后端及生成代码、T01 的接收/进展/交付章节；总计划仅主会话写。共享代码基线为 09eda33e8d19ab6a255e8088db053829dd1c0d0d，无其他产品改动。主会话不同时修改 T01 范围。
- 2026-11-02 17:23:30：主会话接受当前 Proto 字段合同并修订依赖安排，T01 原编号及实现范围不变，通过 codex-cli 队列通知其优先生成和避免全仓修复；分配 T02 给主会话，编号 T02-20261102-172330-main，修订2，授权来自用户本轮“开始实现/继续”。实际基线为上述 HEAD + T01 源协议 diff。接收已写入 T02，统一检查由主会话在集成后运行。
- 2026-11-02 17:41:42：主会话读取两项完整交付并审查当前工作区；T01/T02 均从 submitted 接受为 accepted。未重新分配或提交Git。已有临时生产排障记录属于用户在另一会话的独立工作，不计入本功能交付。

## Implementation delivery / 实现交付

2026-11-02 17:41:42 完成本次授权实现及开发检查，保持 Implementation 阶段，尚未执行整体 Verification 或归档。

- 通用能力已贯通：版本声明/服务可空覆盖 → SQLite/MySQL/PostgreSQL 增量迁移及Repository → HTTP/MCP → 有效计划 → local/SSH文件写入；前端使用独立挂载编辑器及稀疏字段提交。MCP新增组件详情读工具。
- 显式同值、空内容、false、字段恢复继承均保留；`0000` 不再被writer替换成0644。内容不做变量替换，保留模式不存在时初始化、存在时不写入。
- 实际文件差异：T02增加局部Dialog以缩小主页面；SQLC按现有生成器更新所有包中的公共schema模型，故多个gen包出现同样4字段变更；新增数据库迁移升级测试。均与计划职责一致，没有Gateway策略产品改动。
- 主会话审查：核对model合并与两处调用、Proto/TS存在性、SQLC插入及NULL读取、SQLite旧记录升级、失败事务回滚、MCP完整回读、前端请求与取消草稿隔离；确认所有生产WorkspaceFile调用已有显式mode。
- 检查结果：`task proto`、`task sqlc`成功；T01定向及`go test -count=1 ./cmd/... ./internal/... ./sql`通过，随后补充的迁移/回滚/MCP改动定向检查通过；主会话最终`task check`通过（Go lint 0 issues）；前端`lint:fix`、`typecheck`和完整44文件197测试通过，无测试警告。首次前端lint错误已修复并复跑；`git diff --check`通过。
- 未运行：真实MySQL/PostgreSQL迁移、真实SSH/POSIX权限、隔离Docker重建及浏览器人工操作。SQLite临时数据库和文件测试不能代替这些运行环境证据；A08真实容器部分留待整体Verification。
- A10已在本计划给出Gateway逐项调整结论；其现存静态配置改写、强制覆盖及重建尚未移除，不把本次通用能力交付描述成Gateway业务改造。
- 无未完成的本轮实现项；未执行本功能生产迁移、实际部署、服务器启停或Git提交。

2026-11-02 19:25:12 追加交付：按用户要求简化本次挂载迁移的跨字段约束，组合规则继续由 model 与应用用例负责。本地 PostgreSQL 已在事务内删除复杂 CHECK，保留简单约束及全部 3 条挂载记录，迁移版本仍为 56。定向测试、完整 `go test -count=1 ./cmd/... ./internal/... ./sql`、`task check` 和 `git diff --check` 均通过。此证据为本地就地修改及 SQLite 迁移测试，不代表三种真实数据库迁移全量验证。

2026-11-02 20:22:02 交互调整：用户要求回到原挂载弹窗直接编辑，移除继承/自定义操作。按上述前端设计保留类型、来源、目标、只读与目录宿主路径布局，受控文件展开内容、权限和覆盖/保留策略；未改字段保持原状态。移除失效文案与旧切换 helper，更新轻量单测和操作手册。前端 lint:fix、typecheck、44 文件 198 项单测和差异检查通过，无测试警告。本次未修改后端或触发部署。

2026-11-02 20:43:58 两项策略交付：先完成意图、计划和手册修订，再统一版本/服务的共享策略控件，移除 Gateway 强制覆盖例外。保留模式的文件测试模拟应用修改并核对内容、权限、文件标识及修改时间不变；部署测试确认强制重建仍传递保留策略。定向后端、完整 Go 测试、前端 44 文件 198 项单测及最终 task check 通过，无前端测试警告；首次格式问题已修正。开发库已核对为迁移 56 且非 dirty，原字段可直接表达两项能力，无需修改表结构或加入兼容层，3 条覆盖及 Casdoor 内容保持不变。未部署、未启动/重启开发服务器；真实 Docker/SSH 和浏览器操作仍未验证。

2026-11-02 最新修订：受控文件取消只读，目录/文件保留只读；版本和服务两处界面、业务校验、部署校验及本次迁移同步调整。按用户要求移除此前新增的迁移专用测试；本次不扩展 Gateway 业务修改。

2026-11-02 21:37:09 总体验收：主会话复核修订3后的 T01/T02 工作区集成，受控文件和目录只读边界已重新检查；前后端检查及隔离 Linux/Docker 边界验证通过，详见验证报告。此前提交历史中迁移测试为过程记录，该专用测试已按用户要求删除。

## Verification plan / 验证计划

| 验证边界 | 主要证据 |
| --- | --- |
| 合并与校验单测 | 受控文件内容/权限/策略独立覆盖，目录和普通文件只读覆盖，空内容及 false，显式相等值，恢复继承，版本切换，非法类型/权限/大小 |
| 本地持久化集成 | 使用当前迁移临时 SQLite；NULL 与空/false 的往返、替换写入保留无关数据、失败回滚及 Project 隔离；迁移语法按已有工具检查 |
| HTTP/MCP 协议 | mapper 存在性、组件详情、MCP 新读工具、完整集合回读、全量替换保持无关覆盖；不以完整 DB 装配代替纯 mapper 单测 |
| 文件适配器 | 已有 local/SSH 边界扩展必要用例：覆盖、保留、初始化、目录冲突、mode `0000`；Linux 权限与 Windows 差异分开记录 |
| 部署 | 有效 hash 包含新字段、Compose ro/rw、WorkspaceFile 内容及策略传递；用户 ForceRecreate 进入现有 lifecycle，不自动改策略 |
| 前端轻量单测 | 编辑状态到稀疏请求、恢复继承、相等值仍自定义、空/false、UTF-8 大小和权限反馈；不挂整页串联 Router/Store/API |
| 容器运行边界 | 有可用隔离 Docker 测试环境时，复用既有显式启用条件验证单文件替换后强制重建读新内容及保留模式；未运行须写明，不冒充已证实 |
| Gateway 分析 | 对照本章逐项复核现状、目标和依赖，确保未把业务分析当成产品改造完成；无需连接真实 Gateway |

生成沿用 `task proto`、`task sqlc`；Windows 使用已安装工具及正确 PATH，不为生成升级工具或启动服务器。检查生成 diff，避免无关锁文件变化。

实施后按范围先跑定向测试，再执行项目要求：根目录 `task check`、`go test ./cmd/... ./internal/... ./sql`；`web/` 执行 `corepack yarn lint:fix`、`corepack yarn typecheck`、`corepack yarn test --reporter=default --silent=false --no-color`。统一单测入口为现有 `task test`；不使用技能中已与当前 Taskfile 不符的旧任务名。计时需要 `-count=1`。

以上为整体验证安排；本轮已运行的开发检查及未运行项见“实现交付”。未将开发检查自动登记为整体Verification通过。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- 空值/false/presence 丢失会造成错误覆盖，协议与数据库必须实际往返验证。
- 旧来源覆盖记录无法恢复历史用户当时是显式选择还是前端复制；保留已存覆盖事实，不猜测并清除。
- 没有整批文件事务或运行进程锁；平台不能承诺阻止应用同时修改文件。
- `0000` 可能使应用无法读取，这是有效用户配置；Linux 执行须准确，测试清理按文件所有者恢复权限。
- Windows 不承诺 POSIX mode；实际 SFTP/POSIX rename 能力与容器挂载生效需在对应环境验证。
- Gateway 业务策略在 T01/T02 期间仍存在，其界面声明和实际 enrichment 的差异必须在交付说明中列明，不夸大通用能力覆盖面。
- 暂无必须用户先决定才能完成本计划的未决问题。Gateway 显式配置动作的具体 UI 属于后续业务实施设计，不影响本次通用契约。

## Rollback / recovery / 回滚 / 恢复

- 产品变更按两个任务的实际 diff 分别回退，不重置用户工作区，不自动提交。
- 已部署新增数据库列时，旧程序虽可能忽略新字段，其部署行为可能回到版本内容；不能将二进制回退视为语义安全。回退前需保存服务覆盖并安排恢复，不自动执行 down 删除内容。
- 文件已覆盖后的业务数据恢复依赖部署前备份；本任务不宣称数据库回滚能够恢复服务器文件，也不触碰 ACME 状态。

## User review notes / 用户审查记录

- 2026-11-02：用户要求进入计划，并强调 Gateway 配置和策略必须在本任务分析清楚。
- 本计划形成通用能力实施方案与 Gateway 当前/目标/顺序/验收分析；两个 task 均 Draft、unassigned，产品代码尚未改动。
- 2026-11-02：用户开始实现，接受计划及两个 task；按上述分配开展 Implementation，前一条为计划交付时历史状态。
- 2026-11-02 17:41:42：完成实现、集成审查和开发检查，两项交付接受，待后续整体验证安排；无额外部署授权推断。
- 2026-11-02 19:19:27：用户要求简化数据库约束、业务层补齐校验，并授权就地修改本地库。主会话继续 T01 范围内调整本次迁移，保留现有业务校验与覆盖内容；不扩大到其他业务表。
- 2026-11-02 20:43:58：用户确认两项产品策略并授权先文档后实现、无兼容/适配、迁移及开发库继续就地修改；本轮由主会话完成共享产品入口及 Gateway 写入策略修正，不影响已接受的原 T01/T02 分配历史，不扩大为 Gateway 整体治理。
