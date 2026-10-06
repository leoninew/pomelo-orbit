# 组件运行身份、共享挂载与 Orbit 停机迁移实施计划
最后修改时间: 2026-10-06 17:53:32

Review status: Accepted

Mode: standard

## Intent basis

用户已要求“开始实现”。依据已接受的 [Intent](../intent/20261006-component-user-group-add.md)，实现运行身份与共享挂载，并提供迁移说明。Orbit 按普通 Application / Version / Service 部署，DooD 通过目录和 Docker socket 挂载实现，复用已有路径映射。

## Design decisions

### 运行身份

| 层 | user | group_add |
| --- | --- | --- |
| Version Component | 可选字符串，空表示镜像默认用户 | 字符串列表，空表示不声明附加组 |
| Service Component | nil 继承，非空替换，显式空恢复镜像默认 | nil 继承，非空整体替换，显式空列表移除附加组 |
| Effective / Compose | 有有效值才输出 | 有有效值才输出字符串列表 |

组件表新增 nullable container_user、group_add_json。Service 保留 SQL NULL / 空字符串及 NULL / [] 的区别。HTTP/Proto 的 Service 列表使用 ServiceComponentGroupAdd { values } 保留存在性。非空值等同 Version 时可归一化为继承，显式空仍保留。校验用户名称或 UID、可选主组名称或 GID、附加组名称或 GID，拒绝空分段、空组、空白及控制字符，不探测宿主机账号或推断 GID。

Version 组件详情的运行身份为独立区域，`PUT /api/version/:version_id/component/:component_id/identity` 仅保存 user/group_add；Basic 更新契约与 SQL 不含身份字段。MCP 使用独立的 orbit_update_version_component_identity。创建和完整定义仍可携带初始身份。编辑使用紧凑弹窗内的两个纵向单行输入框，附加组按逗号分隔展示与输入，提交文案为“保存”。

### 通用共享挂载

- Version directory mount 新增 shared 布尔声明，默认 false，限绝对目录源。与类型、target、read_only 一样属于声明；Service 可以覆盖 source 或删除挂载，不能改变共享属性。
- 共享意味着该目录允许其他服务在内部准备部署文件。Service 的确认/运行根目录仍独占；未声明共享的实际 bind 保持现有保护。
- Compose 在组件容器上生成 io.pomelo-orbit.shared-mounts 标签，JSON 数组记录共享 mount 的容器 target。检查使用实际运行容器的标签、Mounts.Source 和 Mounts.Destination；编辑未部署的 Version 不改变运行占用判断。
- 仅共享 bind source 包含本次写入源时放行。写入源包含共享 bind source 时仍拒绝。同一容器的嵌套非共享 bind 独立检查，不能由父挂载共享属性覆盖。
- local DooD、SSH 和外部 worker 使用同一判定，不识别 Orbit 名称、当前 worker 容器 ID 或控制面角色。保留 Docker 查询失败时的拒绝行为。
- 不新增 ControllerWorkspaceInspector、bootstrap 注入或另一套 DooD 映射。

## Implementation steps

1. 模型、校验、三种数据库增量迁移、schema、SQLC 与 Repository：身份字段和 Version mount.shared；保留 Service 空覆盖。
2. DTO、Proto、HTTP/MCP、定义传递、fork/CI 派生：贯通输入输出与复制。
3. Effective 合并、hash、渲染：身份和 shared 参与配置，运行标签承载已部署的共享声明。
4. Web：Version 新建支持初始身份，已存组件详情通过独立运行身份区域及接口保存；mount 编辑支持 shared；Service runtime 支持身份覆盖、显式清空和恢复继承。
5. 占用检查：按运行标签匹配实际 bind target；保留根目录、普通数据与嵌套非共享源保护。
6. 活文档及迁移指南：说明共享语义与普通应用迁移步骤，保留现有文档编辑。
7. 执行固定工程检查与最小有效测试，Implementation 完成后等待用户授权验证；进入 Verification 后记录验证结果并修复发现的问题。真实迁移按用户后续执行指令推进，不执行 Git 写操作。

## Files to change

- internal/model、internal/application/{application,service,deployment} 与相关测试。
- sql/migration/{sqlite,mysql,postgres}/000052_component_identity_shared_mount.{up,down}.sql、schema/query、Repository 与 SQLC 生成物。
- application/service Proto、HTTP routes/handler/MCP、Repository 端口与生成 DTO。
- web/src/views/application/{componentForm,VersionComponentDetail}、web/src/views/service/ServiceComponentDetail.vue、身份草稿转换 helper、locale。
- docs/product/cd-model.md、docs/architecture/cd-runtime.md、docs/guides/volume-mounting.md、新增 docs/guides/orbit-self-migration.md、INDEX。

已有 runtime_directory 服务详情展示及部署指南修改需保留，不归本次身份/挂载功能所有。

## Tencent deployment procedure

后续执行，当前只编写指南：

1. 重查 revision、运行镜像、socket GID、挂载与 Gateway 状态；备份 PostgreSQL、密钥配置、旧 Compose、Gateway 文件与证书/ACME。回退镜像以实际已成功运行的镜像为准。
2. Fork 可编辑 Orbit Version，选择包含本次功能的镜像；设置 user=1000:1000、实际 socket 附加 GID；补齐 data/logs/.env/socket 绝对 host-path 挂载，其中共享工作区 data 标记 shared=true。补齐 HOME、工作区/日志配置、HTTP 80 endpoint，发布 Version。
3. 外部执行端使用同一 PostgreSQL 与凭据加密配置，具备实际消费部署任务的 worker；排空任务进入停机窗口，停止旧 pomelo-orbit project、停用旧启动入口。
4. 通过普通部署入口将 pomelo-orbit-default 部署到空目录 /opt/pomelo-orbit/data/deployment/pomelo-orbit-default；由业务生成 Compose 并绑定运行目录，不搬整棵 data 到自己的子目录。
5. 核查新容器身份、挂载、数据库与启动。Gateway 原目录先核对实际 project/service、备份 Compose，再通过 Gateway 入口重部署，保留证书/ACME；当前代码不要求 owner 标记。不能伪造数据库部署成功。此时外部或内部 worker 都能执行。
6. 明确退出临时 worker 后，由新 Orbit worker 在线部署/重启隔离验收服务；预览并显式确认两条 Route，检查 HTTPS、登录、数据、凭据解密及 CI/CD/Route 路径。
7. 失败时终止迁移任务及临时 worker，停止新 project，撤销本次 Route 发布；按故障范围恢复 Gateway，再以已知成功镜像启动旧 project。保留共享数据与真实业务记录，数据库恢复仅在需要时按停机备份执行。

local DooD 的确认目录使用容器内路径，如 /app/data/deployment/pomelo-orbit-default；Tencent SSH 使用远端路径。既有最长挂载前缀映射继续复用。Orbit 更新自身仍须外部 worker，停掉自身后无法继续消费该任务。

## Verification plan

- 身份校验、Version 保存/清空、Service 继承/替换/显式空/恢复继承的数据库与 API 往返。
- Version Basic 与 Identity 独立保存：双方不替换对方字段，身份校验不阻断基本信息表单，MCP 与 HTTP 分组契约一致。
- 合并与 Compose YAML 验证 1000:1000、字符串 ["988"]、省略空身份；hash 包含有效值且等价空值一致。
- 定义导入导出与 fork/CI 保留新值；Web 表单和覆盖请求保留三态。
- SQLite 升级验证；MySQL/PostgreSQL 迁移结构核对，未实测明确说明。
- 渲染标签和实际挂载判定：共享源下部署允许，未标记/嵌套普通 bind/Service 根目录仍拒绝，运行标签不依赖待部署规格。覆盖 Linux、Windows/Desktop 路径及现有 DooD 映射回归。
- 运行 task proto、task sqlc、yarn --cwd web lint:fix、yarn --cwd web typecheck、task check、go test ./cmd/... ./internal/... 和聚焦 Web 测试、git diff --check。
- 腾讯停机迁移、真实 Docker DooD 与域名验收按用户后续授权执行，代码测试不能替代真实验收；用户接手后续实测后停止提交远端任务。

## Risks and rollback

Service 显式空在 JSON/SQL 中丢失会误变成继承，优先测试。shared 是主动放宽数据占用保护，只对实际需要共享的工作区声明；部署根目录仍保护。已有运行容器须重新部署才有新标签。UID/GID 格式正确不保证目录/socket 权限，实际迁移重查并验证。数据库回退优先保留增量 schema，不自动执行 down。

## User review notes

用户接受停机、要求覆盖 DooD 并明确 Orbit 应当作为普通应用；本计划采用通用共享挂载方案。用户“开始实现”视为接受 Plan。

实现期间仓库并行提交 2dbffa194 移除了部署 owner 标记机制，迁移步骤已按当前代码同步。

用户要求“开始验证”；验证中补充运行身份从组件基本信息拆出并单独保存的要求，已作为当前任务调整。当前：标准模式 / standard，验证 / Verification。

用户要求编辑界面简化为两个单行文本框，已落实。再次进入验证时明确要求修复发现的问题，凭据更新时间测试修正纳入验证修复，未扩展凭据产品行为；补充 local DooD/SSH 完整目录检查和 CI 派生版本配置保留的正向回归。

用户禁止操作其浏览器；本轮不继续浏览器验证，页面视觉验收由用户执行。

用户随后授权实际迁移，要求最终通过本地 `orbit.preflite.cn` 项目部署 `pomelo-orbit-default`，正常运行且能部署其他服务。初次迁移、PostgreSQL 升级、域名及 DooD/SSH 部署已执行，实际证据见 Verification。发现详情配置摘要问题后，用户要求描述问题及复现步骤并接手真实测试；摘要修正和自动化检查完成，r1 镜像已构建但不继续远端升级或测试。
