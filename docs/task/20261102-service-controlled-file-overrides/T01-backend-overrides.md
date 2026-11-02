# Task / 子任务 T01：后端服务受控文件覆盖契约
最后修改时间: 2026-11-02 21:39:17

## Review status / 审查状态

Accepted

## Parent and authority / 父任务与权威记录

- 总意图：[服务受控文件覆盖](../../intent/20261102-service-controlled-file-overrides.md)。
- 权威总计划：[计划](../../plan/20261102-service-controlled-file-overrides.md)，位于 `D:/SourceCodes/mywork/pomelo-orbit`；任务 ID：T01；原分配方案修订1，最终方案修订3由主会话完成并复核。
- 依赖、执行者、编号和仲裁结论以总计划为准；本文状态只表示方案审查。

## Handoff basis / 交接依据

- 已分配给会话“查看华为云机器上部署的容器应用”（01a19ae7-d2b0-7241-a111-42e61e551fbf），编号 T01-20261102-170337-01a19ae7；用户已授权 Implementation 及必要开发检查，依据总计划分配历史。
- 协议入口：`C:/Users/wangm25/.codex/plugins/cache/specflow-local/specflow/0.2.1/skills/specflow/SKILL.md` 及 `references/task-coordination.md`。
- 代码工作区为 `D:/SourceCodes/mywork/pomelo-orbit`，HEAD 为 `09eda33e8d19ab6a255e8088db053829dd1c0d0d`，尚无其他产品改动；本任务没有前置任务。开工核对总计划登记，不以 task 路径自行扩大范围。
- 执行者可写本文接收、进展及交付章节；主会话读取权威根下本文件。默认输出可复制摘要，由用户转交，不自动发送外部消息。

## Goal and acceptance / 目标与验收

主要交付：服务受控文件覆盖从协议、持久化到有效部署计划完整贯通，且文件执行保留字段语义。

覆盖 A01–A04、A06–A07 和 A09 后端部分：字段独立可空、空内容/false/显式相等值保留、恢复继承、有效合并校验、HTTP/MCP 可回读、写入策略及权限准确。A08 的执行参数证据在本任务提供，真实容器结果由整体验证记录。

## Inputs and eligibility / 输入与可开始条件

直接前置见总计划。开工前核对总计划和本文已接受、用户实施授权、唯一分配编号及实际基线。采用现有 service/deployment model、Repository 和 runner 端口；不依赖真实 Gateway 或生产数据库。

## Scope and files / 范围与文件

- Proto、model/service 和挂载规则、service normalize/remap、deployment MergeServiceComponent/hash 相关行为。
- 三种数据库新迁移、两份 schema 镜像、service 查询、SQLC Repository 与生成代码。
- HTTP service mapper、MCP service tools/types/mapper 和必要注册测试；新增服务组件读工具。
- local/SSH writer mode 合同及必要显式 mode 调用方；不重构 Route 发布业务。
- 本任务独占生成 Go/TS Proto 与 SQLC，不修改前端编辑交互。
- 用户本轮要求统一两项写入策略，主会话取消 Gateway enrichment 对 IgnoreIfExists 的强制覆盖，并验证策略进入实际文件准备；不修改其选版、内容生成、配置同步和强制重建策略，不宣称完成 Gateway 整体治理。

## Technical decisions / 局部技术设计

1. 新增 `Content *string`、`Mode *string`、`IgnoreIfExists *bool`、`ReadOnly *bool`，协议 optional、数据库 NULL。source/source_is_host_path 同样支持独立继承，统一合并后校验。
2. 将单条声明与稀疏覆盖的合并及有效校验放入 model 挂载规则，由保存校验和部署构造复用。保持 target 身份、不允许服务改 source_type；文件专属字段仅适用于 controlled_file。
3. override 至少一个字段存在，deleted 无字段值；空字符串内容、false、与基线相等值不能丢失。组合关系只在 model 校验，不重复编写数据库跨字段 CHECK；数据库保留 state 枚举等简单约束。不清理无关字段的现有归一化。
4. SQLite/PostgreSQL 内容 TEXT，MySQL 迁移 LONGTEXT；更新 query 使用项目现有命名参数约定。新字段无默认值兜底，已有行自然继承。
5. HTTP 详情输出版本声明及服务覆盖。MCP 新增 `orbit_get_service_component`，复用 GetServiceComponent；输出完整 overlay 集合，写工具明确全量替换契约。
6. 全部新挂载值进入现有计划 hash；worker 使用同一合并路径，不单独缓存文件内容。
7. WorkspaceFile.Mode 为实际 mode，移除 0→0644 转换；核对生产调用点并显式表达原默认权限。测试 Linux `0000` 与 Windows 平台区别。
8. 产品策略继续使用现有 `ignore_if_exists`，false 为覆盖、true 为保留；不存在时均初始化。平台内容或权限变化、版本升级和容器重建均不改变保留行为。无新字段、兼容别名或适配分支，现有迁移和开发库不需为产品文案变化执行 DDL。
9. 用户最新确认受控文件不支持只读；版本和部署校验拒绝受控文件只读，服务覆盖拒绝受控文件 read_only，目录和普通文件继续支持。`000056` 清除旧受控文件的只读值；不为迁移编写测试。

## Implementation outline / 实施安排

1. 扩展源协议、模型、迁移和查询，使用 `task proto`、`task sqlc` 生成产物。
2. 完成 Repository 读写、稀疏合并/校验及版本切换的存在性语义。
3. 完成 HTTP/MCP 映射、读工具和说明，避免只扩展写入不可回读。
4. 核对 WorkspaceFile 传递和 mode 合同，保留原写入顺序及错误传播。
5. 增补必要规则、协议和本地持久化/文件测试；报告交付给前端的确切 DTO 和基线。

## Checks and delivery / 检查与交付

- 规则：独立字段、自定义空/false/相等值、恢复继承、无效 mode/大小/类型、版本切换兼容性和完整计划 hash。
- 协议：HTTP mapper 与 MCP 内存协议验证；读工具返回足以保留其他 overlay 的数据。
- 持久化：当前迁移创建临时 SQLite，验证 NULL/空/false 往返及失败原子性；不得通过手写局部 DDL 绕过迁移。
- 文件：实际临时文件验证覆盖、保留和初始化，mode 零值只在支持 POSIX 权限环境断言；SSH 复用既有协议/renamer 边界。
- 定向检查后运行 `task check`、`go test ./cmd/... ./internal/... ./sql`；生成的前端 DTO 纳入现有 typecheck。
- 交付提供实际 diff/可访问基线、验收证据、命令结果、未运行原因及对 T02 的接口说明。没有外部 Docker/SSH 证据时不宣称真实部署已通过。

## Risks / 风险

同值归一化、JSON omitempty、SQL NULL 和 TS 默认值均可能破坏存在性；新增 MCP 完整输出须保持 Project scope。共享 writer 权限变动必须核对所有生产调用。遇到契约变化先回报主会话，再调整计划，不独立扩大为 Gateway 改造。

## Reception and progress / 接收与进展

- 2026-11-02 17:06:56：会话 01a19ae7-d2b0-7241-a111-42e61e551fbf 接收 T01；有效编号 T01-20261102-170337-01a19ae7，方案修订 1。已核对总计划 Accepted 与唯一登记、主会话转达的用户 Implementation/必要开发检查授权；无前置任务。
- 实际基线为 HEAD `09eda33e8d19ab6a255e8088db053829dd1c0d0d`，工作树仅本功能过程文档未跟踪；权威协调根与产品工作区均为 `D:/SourceCodes/mywork/pomelo-orbit`。不操作生产、服务器、Git 提交或 Gateway 业务策略；仅修改 T01 范围及本记录。
- 已读取 SpecFlow 协议、总意图、方案、T01 与后端治理技能，开始核对模型、协议、Repository 和 writer 调用链。暂无阻塞。
- 2026-11-02 17:10:09：用户中断实施，要求优先处理远程 K12 数据库替换及 Casdoor 绑定问题；T01 暂停在只读代码核对阶段，尚无产品代码修改或生成/测试操作。此前实施授权与分配未自行撤销，待优先事项处理后继续；生产操作属于用户追加事项，不属于 T01。
- 2026-11-02 17:18:30：优先事项完成，用户确认登录恢复，现有迁移执行完成且无待执行项；临时凭据及本次转发已清理。恢复原编号 T01 实施，工作树仍无产品修改。
- 2026-11-02 17:26:15：完成模型稀疏合并、mode 零值修正、Proto/SQLC 生成、三种数据库增量迁移、HTTP/MCP 映射和 MCP 组件读工具。定向运行 model、service/deployment usecase、service repository、HTTP/MCP 和 local/SSH runner 测试均通过；正在补充存在性、真实 SQLite 持久化及协议证据。按用户追加要求修改的 `data/forward.sh` 不属于 T01 产品改动。
- 2026-11-02 17:39:28：核对总计划修订 2 保留本编号与后端范围，仅改变 T02 并行安排；前端手写代码由主会话拥有，未修改。补充迁移升级、回滚、作用域及版本切换测试，完成待审查交付记录。

## Delivery and review / 交付与审查

### 2026-11-02 17:39:28 交付待审查

- 执行者会话 `01a19ae7-d2b0-7241-a111-42e61e551fbf`，编号 `T01-20261102-170337-01a19ae7`，初始修订 1，当前总计划修订 2；本任务范围未变化。无前置任务，实际起点 HEAD `09eda33e8d19ab6a255e8088db053829dd1c0d0d`；交付位置为 `D:/SourceCodes/mywork/pomelo-orbit` 当前工作树的 T01 后端、迁移及生成物 diff，尚未提交。该工作树同时包含主会话 T02 前端手写变更，不能把整个 diff 当作 T01 独占交付。
- 实际变更：`ServiceComponentMount` 扩展四个可空字段，model 合并和校验复用于保存、选版重映射与有效部署计划；保留同值、空内容与 false。新增 SQLite/MySQL/PostgreSQL `000056_service_mount_overrides` 升降迁移与 SQLC schema/query，Repository 读写这些字段。HTTP 声明和 overlay 映射、MCP `orbit_get_service_component`、完整 overlay 输出及全量替换工具说明已更新。local/SSH writer 不再将 mode 0 转为 0644；Compose 和 Route 调用已有显式权限。`task proto` 和 `task sqlc` 更新 Go、TS 生成物，SQLC 的全 schema 模型生成影响多个生成包。
- A01/A02/A03/A07/A09：`mount_overlay_test.go` 覆盖独立继承、空内容、false、无效 mode/大小/类型；service remap 测试覆盖显式相等值和版本类型不兼容；HTTP mapper 与 MCP 内存协议测试覆盖声明、稀疏字段与回读。SQLite 当前迁移 Repository 测试覆盖 NULL/空/false、恢复继承、Project 隔离、注入失败事务回滚；`service_mount_migration_test.go` 从 55 升级验证旧记录保留与新字段 NULL。
- A03/A06：有效计划测试验证合并值和 hash 变化；构造/执行共用 `MergeServiceComponentMount`。A04：local 文件测试覆盖缺失初始化、存在保留及替换、零 mode 的 POSIX 断言；当前测试机为 Windows，POSIX 权限断言不适用。SSH writer 移除同样的兜底，现有 SSH 测试通过；未连接真实远端验证 POSIX mode。强制重建仍由既有用户选项决定，未改变 Gateway 特殊行为。
- 已运行：`task proto`、`task sqlc` 成功；定向 `go test -count=1` 覆盖 model、service/deployment usecase、SQLite database/service repository、HTTP/MCP、local/SSH runner，全通过；`go test -count=1 ./cmd/... ./internal/... ./sql` 全通过（随后仅补充并单独通过迁移、回滚测试及 MCP 输出的小改动）；`./bin/golangci-lint config verify`、`fmt --diff`、`run` 均通过；`git diff --check` 通过。`task check` 的 web typecheck 通过，随后在主会话并行开发的 T02 前端 lint 错误处停止，未修改其文件；整合后由主会话重跑统一检查。
- 未运行真实 MySQL/PostgreSQL 新迁移、真实 SSH/POSIX 权限及隔离 Docker bind mount/重建验证：本任务未获生产操作授权，外部验证有独立环境要求。迁移 down 会丢失新增字段，且存在仅新字段的 sparse override 时旧约束可能拒绝回退；不能当成无损降级。Gateway 仍有自身 enrichment 和强制重建行为，通用能力不代表该业务策略已改造。上述风险需要整体集成和环境验证，不属于 T01 已验证证据。
- 下游 DTO：`ServiceComponentDeclaredMount` 新增 `content`、`mode`、`ignore_if_exists`；`ServiceComponentMountOverlay` 新增 optional `content`、`mode`、`ignore_if_exists`、`read_only`，保留已有 optional source/source_is_host_path。`undefined` 表示继承，`""` 与 `false` 为显式覆盖；更新为整体替换。MCP 新读工具返回 `service_component` 完整稀疏 overlay 与 `version_component` 声明。前端生成 DTO 位于 `web/src/gen/proto/orbit/v1/service/service.ts`。无额外契约变更或新增阻塞。

### 2026-11-02 17:42:39 最终交付补充

- 主会话已审查 T01 diff、迁移、SQLC、mapper 及新增测试，反馈暂无返工项；由主会话维护最终审查登记，执行者不自行标记 accepted。当前方案修订 2，分配编号和实施范围不变。
- 2026-11-02 21:37:09：主会话按用户修订3完成受控文件只读清理及相关业务测试，移除迁移专用测试，复核最终 diff 和 T02 集成。完整 Go 测试、相关定向测试和 Linux 文件写入测试通过；不宣称真实 SSH 或生产数据库迁移已验证。
- 生成命令为 `task proto`、`task sqlc`，均成功。`go test -count=1 ./cmd/... ./internal/... ./sql` 全量通过；其后新增或调整的 T01 文件所涉包均已以 `go test -count=1` 定向复验通过。未执行真实 MySQL/PostgreSQL 新迁移、真实 SSH/POSIX 权限及隔离 Docker 挂载/重建验证，原因及风险见上一节。
- 主会话报告最终 `task check` 已通过，覆盖前端 typecheck/lint/format 和 Go fmt/lint（0 issues）；完整前端轻量单测 44 文件、197 项通过。以上为主会话的实际检查结果，本会话未重复运行。
- 按主会话要求，停止本任务的产品代码修改，等待主会话完成最终登记与交付。

### 2026-11-02 19:23:51 主会话追加调整

- 用户要求数据库保留简单约束，字段组合由业务校验，并授权本地库就地修改。主会话简化本次 `000056` 的 SQLite/MySQL/PostgreSQL up，调整 PostgreSQL down；既有 schema 镜像已是简单约束，无须改变生成代码。历史版本的 down 仍恢复其原结构。
- `MergeServiceComponentMount` 已拒绝空 override、带值 deleted、无效 mode、超限内容和不匹配文件类型；保存和部署复用该规则，本次无需新增重复校验或测试。
- 本地 PostgreSQL `pomelo_orbit` 在短事务内删除 `service_component_mount_values_check`，保留主键、非空、外键和 state 枚举。修改前后 3 条挂载记录完整一致，迁移版本仍为 56 且非 dirty；没有重建或清空本地库。
- 定向 `go test -count=1` 覆盖 database、model、service/usecase 和 service repository 均通过；`task check` 通过，Go lint 为 0 issues；完整 `go test -count=1 ./cmd/... ./internal/... ./sql` 和 `git diff --check` 通过。

### 2026-11-02 20:43:58 两项写入策略交付

- 主会话按用户追加授权取消 Gateway 对 `IgnoreIfExists` 的强制清零，保留其内容生成、选版及重建流程。Gateway 和普通服务均把所选保留策略传至 WorkspaceFile；重建不改变文件策略。未加入任何兼容路径或新字段。
- 调整 Gateway enrichment/运行测试中的旧强制覆盖断言。文件适配器测试模拟应用修改初始文件，再以不同内容和权限部署，确认保留原内容、权限、文件标识和修改时间；覆盖模式继续验证替换及显式 `0000`。
- 定向 deployment/usecase、local 和 SSH runner 测试及完整 `go test -count=1 ./cmd/... ./internal/... ./sql` 通过；最终 `task check` 为 0 issues，差异检查通过。
- 本地开发 PostgreSQL 仍为迁移 56、非 dirty；版本/服务的现有策略字段满足本次语义，未执行无必要的 DDL。挂载表只有 state 枚举 CHECK，3 条服务覆盖保留，Casdoor 回填内容仍为 54 字节。本轮未触发实际部署、远程文件写入或开发服务器启停，未执行真实 Docker/SSH 文件生效验证。
