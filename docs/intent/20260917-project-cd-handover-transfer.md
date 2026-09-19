# Project 级 CD 接管数据传输
最后修改时间: 2026-09-17 17:27:47

Review status: Accepted

Mode: standard

## Background

Orbit 已完成 Project / Environment 隔离。一个 Project 独占一个 Docker deployment target；Gateway、Application、Version、Service 和 Route 都以该 Project 为边界。旧控制面退出后，对一台服务器的接管单位必须是一个 Project 完整的 CD 配置聚合，而不是分别传输 Environment 或单个 Service。

接管前，源 Orbit 必须已升级至当前版本。此次不处理历史 Orbit schema、Compose 布局、路由模型或运行时语义的兼容转换。

当前 Python `database-transfer` 通过 dbtalk JSONL 表块按 Service 或独立 Environment 读写数据库。它不能表达 Project 唯一的 Gateway、Gateway Version bindings、dashboard Route、全量 Project Route 和完整 Environment，也没有当前用户身份、原子 Project 接管或业务界面。导入导出是 Orbit 的 Application 层用例，数据聚合查询、完整性校验和持久化属于领域仓储职责；不能在 Handler 或 Application 中堆叠表删除顺序、行循环和事务脚本。

Environment 包含 `target_type`、SSH host / user / credential、host-key fingerprint、`workspace_root`、target revision、Probe 状态和 Gateway binding；这些都是 Project 部署域的业务数据。接管时不预先假定已知 SSH 地址，因此 Environment 不能转换或拆分，必须随包原样恢复。

源与目标 Orbit 都可能通过 seed SQL 建立 Project，两边的 Project ID 可能恰好相同。Project ID 只是一处目标绑定值，不能作为可迁移身份或冲突处理依据。

## Goal

1. 以 Orbit 业务 API 和 Web UI 提供 Project 级 CD 接管导出与导入，作为按单个 Service 和独立 Environment 的 Python 导入导出脚本之外的新业务入口。
2. 使用单一、版本化的 Orbit JSON 传输包表达完整 CD 配置聚合，不继续使用 dbtalk JSONL、通用表块、通用 import mode 或脚本运行时。
3. 在 Project、Environment、Application / Version、Gateway、Service、Route 与 Pipeline 既有领域中补足正常业务能力；Handover Application 只按依赖顺序组合这些领域操作和目标 ID 映射，且不直接组织 SQL、表级删除或批量持久化。
4. 模式 A 创建新 Project：由业务服务生成目标 Project ID，以包中的 Project 业务字段创建 Project，并在同一原子导入中将当前认证用户加入 `project_member`。
5. 模式 B 选择当前用户已有成员资格的 Project：以包原样覆盖其 CD 配置聚合和 Environment；Project ID 保持目标 ID，既有成员关系不迁移也不删除。
6. 导入完成后，Service 保留包中的原始状态，Route 继续按现有规则禁用；导入不部署、不重启容器、不写远端 workspace，也不发布 Traefik REST snapshot。

## Non-goal

- 不支持未升级到当前 Orbit 版本的源实例，也不实现历史 schema 或旧 Compose 的转换器。
- 不迁移源 Project 的 `project_member`、用户、角色、认证、系统设置或其他控制面身份数据。
- 不迁移 Repository、repository credential、Pipeline、Pipeline Run、Artifact、Deployment 历史、任务、对话和控制面日志。
- 不自动把 `local` Environment 改写为 `ssh`，不推断 SSH 地址、用户或凭据，也不探测导入后 Environment 是否可部署。
- 不在导入过程中修改远端 workspace、容器、卷、`traefik` 网络、Gateway 或 Route provider 状态。
- 不增加复杂数据库约束、跨 Project 物理外键或新的多 Project 共享 Docker target 能力。
- 当前任务不删除 Python `database-transfer` CLI、其 dbtalk 运行时依赖、JSONL 表格式或脚本入口；新业务导入导出不得调用、依赖或扩展它们，后续是否清理另行决定。

## User scenarios

1. 当前用户从 Project 管理界面导出自己有权限访问的源 Project，浏览器下载一个版本化 JSON 包。包只包含该 Project 的 Project 配置、Environment、Environment credential、Gateway、全部 Application / Version / Component、Service 运行时覆盖、全部 Project Route 及 Route 证书字段和 Gateway Version bindings。
2. 当前用户在界面选择模式 A 并上传接管包。服务在同一个事务内生成目标 Project ID、以包中的 Project 非 ID 字段创建 Project、创建当前用户的 `project_member`，并将所有直属记录绑定到该 ID。导入成功后该用户可立即切换和访问新 Project。
3. 当前用户在界面选择模式 B、自己已有成员资格的目标 Project 并上传接管包。服务在一个事务中替换该 Project 现有 CD 配置聚合、Environment credential 和 Environment，并以包的 Project 非 ID 字段更新目标 Project；既有 `project_member` 保持不变。
4. 包内 Environment 为 `local` 时，导入结果仍是 `local`；包内为 `ssh` 时，SSH 连接属性和凭据的业务值保持不变。SSH 私钥在导出包中以源实例 Fernet key 加密为 `encrypted_private_key`；导入默认用目标实例的 system key 解密，也可由操作者提交解密 Key 覆盖。解密后的私钥仅在导入内存中存在，目标写入前以目标实例 Fernet key 加密。
5. 导入完成后，Service 的持久化状态与包中一致，Route 仍为禁用。已有容器和 Traefik 的实际运行状态不在此操作中改变，后续运行或路由操作才可能改变它们。

## Acceptance

- [ ] Project 接管 API / UI 不调用、依赖或扩展 Python `database-transfer`、dbtalk 或 JSONL；现有脚本入口及其测试、依赖和文档暂时保持不变。
- [ ] Web UI 支持从可访问 Project 导出，并支持上传包后选择模式 A（新 Project）或模式 B（选择当前用户可访问的已有 Project）。
- [ ] 传输格式是 Orbit 定义的单一 JSON 文档，含稳定的 format 标识和 format version；不包含 dbtalk 表块、SQL、DSN、事务模式或源 Project ID。
- [ ] 包中的 Project 只含可迁移业务配置，不序列化 `id`；所有直接 `project_id` 归属字段也不作为包字段。导入时由模式 A 新生成或模式 B 已选中的目标 Project ID 统一恢复归属。
- [ ] 包以源记录 ID 表达内部引用、Gateway binding 和 Project 内 code；这些 ID 只在包内用于关系解析，导入由领域命令生成目标记录 ID 并建立映射，不把源记录 ID 写作目标数据库身份。
- [ ] Project CD 聚合包含 Project 配置、Environment、Environment credential、GatewayConfig、Gateway Version bindings、Gateway Service、dashboard Route、普通 Application 的完整 Version lineage / Component 子表、Service 及运行时覆盖、Project 全部 Route 与 Route 证书字段。
- [ ] Project、Environment、Application / Version、Gateway、Service、Route 与 Pipeline 领域各自完成其数据读取、完整性校验和写入；Application / Handler 不直接查询 CD 表或逐行写入，也不新增一次性导入整个 Project 的领域仓储或领域命令。
- [ ] 模式 A 在同一个可回滚事务内创建 Project、当前用户成员关系和完整 CD 聚合；成功结果返回目标 Project 给 UI 使用。
- [ ] 模式 B 要求目标 Project 存在且当前用户是成员；它在同一个可回滚事务内更新 Project 配置并替换完整 CD 聚合、Environment credential 和 Environment，保留 `project_member`。
- [ ] 任意解码、完整性校验、目标校验、删除或写入错误均回滚整次导入，不留下新 Project、成员关系或部分 CD 数据。
- [ ] 源 / 目标 Project ID 相同不会导致 Project 行按源 ID upsert 或跨 Project 范围选择。
- [ ] SSH 公钥、Route PEM / key 和 Gateway token 作为普通业务数据传输；SSH 私钥在包中仅通过 `encrypted_private_key` 传输，导出和导入分别以源、目标实例的 Fernet key 完成必要的加密 / 解密。
- [ ] 导入路径不调用 Docker Compose、Docker lifecycle、SFTP workspace 写入、Traefik REST publish、证书签发或 Environment Probe。
- [ ] 导入时保留源包的 Service status；Route 仍按导入规则写为禁用。

## Open questions

- 模式 B 不迁移 CI 数据，但目标 Project 的 Pipeline / Pipeline snapshot 可能引用将被替换的 Application 或 Version ID。为避免留下失效引用，计划默认在发现这类引用时拒绝模式 B，并要求使用模式 A 或先处理目标 CI 数据；该具体拒绝范围需要在 Plan review 中确认。

## Decisions

- 接管只支持当前版本 Orbit 到当前版本 Orbit；源控制面退出是接管前提，不建立双控制面协调、兼容或冲突解决路径。
- 传输语义是“将源 Project 的 CD 配置闭包装配到目标 Project”，不是复制源 Project 身份；源 Project ID 永不迁移。
- 模式 A 创建新 Project，模式 B 由用户选择已有 Project；两种模式都由当前认证用户发起，模式 A 自动建立该用户的成员关系。
- Project 行随包传输 `name`、`code`、`is_active` 等业务字段但不传输 ID。模式 A 生成目标 ID 并插入 Project；模式 B 保留目标 ID，并以包中的 Project 业务字段更新该行。
- 用户与角色是跨 Project 的全局数据，`project_member` 是 Project 访问关系。接管包不包含任何成员；模式 A 只创建当前操作者的成员关系，模式 B 保留目标的成员关系。
- Environment 是 Project CD 聚合的一部分。除归属重绑定到目标 Project 外，Environment、Environment credential 及其关联值原样导入；不基于导入位置转换 `local` / `ssh`。
- 传输包是 Orbit 业务 DTO，而非数据库行转储：它按聚合嵌套表达实体与关联，省略 Project identity / direct ownership 字段；源记录 ID 只作为包内引用，目标记录 ID 由领域命令创建时生成，并使用明确 format version 做兼容边界。
- Project 的部署配置是多个既有领域对象的组合，不新增名为 Project Deployment Configuration 的大聚合或大仓储；Handover 只表示导出 / 导入 Application 用例，不作为领域仓储或模型名称。
- Application 层负责 Handover 的授权、模式选择、包编码/解码、SSH 私钥的包级加密 / 解密及持久化表示转换，并协调既有领域操作；各领域仓储负责自身对象的查询、引用校验、删除和写入。
- 不为接管新增“替换 Project 全部 CD 配置”之类的大领域操作。缺少的能力以最小的现有领域行为补齐，接管用例逐个调用这些行为，而不是把领域表集合当作一个可批量重放的模型。
- Project CD 聚合是唯一的导入事务粒度。覆盖模式的清理与恢复必须进入同一请求事务，不引入分段导入、导入后补救或业务层复杂预检。
- Route 手工 PEM、私钥、Gateway DNS token 与其他导入字段一样是业务数据；SSH 私钥是例外，必须以包级密文传输。
- 导入保留源库的 Service 状态；Route 按现有导入规则重置为禁用。
- 本次实现同时纠正既有跨领域删除语义：聚合内子对象可以随拥有者删除，领域之间的引用必须阻止父对象删除；拥有完整闭包删除流程的领域显式协调自己的子领域操作，不能把 Service、Gateway 或其他领域的删除隐藏在 Application 仓储或数据库级联中。
- 这一删除与引用边界不是接管特例：Project 判断废弃条件时向 Repository / Application 领域读取各自的资源事实；User 删除时调用 Project 领域移除其成员关系；Role 删除时向 User 领域确认不存在角色分配。跨领域关系不能再依靠拥有者仓储直查或数据库级联静默清理。

## Risk

- Project 闭包覆盖 Gateway、Version、Service、Route、Environment 与 credential 多张关联表；漏表会在首次 Gateway 部署或 Route sync 时造成配置漂移，因此领域聚合必须完整读取、验证和原子写入。
- 直接导入源 `local` Environment 到另一台控制面机器后，它指向目标控制面本机 Docker target，不会自动指向原服务器。需要远程管理时，操作者必须在导入完成后显式改为 SSH Environment；这不属于导入阻断条件。
- 覆盖模式只改数据库管理数据。目标服务器上现有容器、Compose 文件和 Traefik provider 保持不变，直到后续显式操作；首次后续运行操作可能按导入配置重建它们。
- 接管包包含加密的 SSH 私钥，以及 Route 私钥和 Gateway token。包不增加签名、脱敏、密钥协商或额外传输协议；使用者须按现有业务数据下载权限管理该文件及其解密 Key。
- 目标 Project 中不迁移的 CI 数据可能引用被替换的 CD Application / Version；导入必须在写入前阻止会留下失效引用的模式 B 操作，或在 Plan review 中确认替代的包范围。

## User review notes

- 用户要求使用 standard SpecFlow 开始此任务，并明确进入 Plan 阶段。
- 用户确认所有源 Orbit 在接管前已升级到最新版本；不应在历史版本兼容上扩大范围。
- 用户确认接管以 Project 为粒度，删除现有 Service 与 Environment 分拆导入导出能力。
- 用户确认源、目标 Project ID 可能相同，导出不得携带源 Project identity，导入必须由目标 Project 绑定。
- 用户确认导入以完整 Project CD 聚合为事务粒度，任何错误回滚。
- 用户确认公钥、Route 证书和 Gateway token 都是普通业务数据；SSH 私钥在包中使用实例 Fernet key 加密，并允许导入时提供覆盖 system key 的解密 Key。
- 用户确认导入后保留 Service 原始状态，Route 继续导入为禁用。
- 用户确认 Environment 应随包导出并原样导入；因为接管前不预先掌握项目 SSH 地址，导入不得转换为其他连接类型。
- 用户确认两种导入模式：模式 A 创建新 Project，模式 B 指定已有 Project 并覆盖其 CD 数据，包含 Environment。
- 用户要求将能力集成到 Orbit 业务 API 和 Web UI；模式 A 通过当前用户自动建立成员关系。
- 用户明确要求导入导出属于 Application 层，查询和写入属于领域层；不得实现为事务脚本。
- 用户确认不需要沿用原有 dbtalk JSONL 格式。
- 用户要求当前任务先保留 `database_transfer.py`；新能力不能依赖该脚本，脚本清理不属于本轮范围。
- 用户要求将这一领域删除边界原则同样应用于其他领域，作为纠正历史实现的一部分。
