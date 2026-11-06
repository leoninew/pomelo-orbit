# Verification / 验证：跨项目复制服务
最后修改时间: 2026-11-06 22:23:46

## Review status / 审查状态

Accepted

用户调用 `$specflow:specflow` 要求“完成验证”，授权当前主会话核对整体交付并修复与已接受方案的偏差。本轮未发现需要修复的产品偏差，仅更新过程文档并按项目约定归档；未执行 Git 暂存、提交、推送或部署。

## Intent alignment / 意图对齐

依据[意图](../intent/20261106-cross-project-service-copy.md) A01–A08，已交付服务详情的跨项目复制入口，版本范围默认最新、可选全部，目标部署目录根据目标环境计算并允许编辑。应用、所选版本及组件、Service 环境值和稀疏覆盖复制为目标项目内的独立配置；冲突和权限失败拒绝整次操作，写入失败由既有事务回滚。

复制成功切换到目标项目并导航至返回的新服务 ID，详情监听路由 ID 重新加载，后续服务 API 使用当前 ID。源资源不变，目标未部署；不迁移数据库、卷或宿主机文件，不操作路由，不生成或轮换凭据。

## Plan alignment / 计划对齐

依据[计划](../plan/20261106-cross-project-service-copy.md)修订 3 及 T01-002 / T02-003：

- 应用领域创建应用、版本与组件并返回 ID 映射，服务领域编排环境读取、目标目录校验、服务与覆盖写入；HTTP 仅处理认证、输入输出和错误映射，bootstrap 注入依赖。
- `POST /api/service/:service_id/copy` 使用现有请求 UoW，成功提交后响应 `201 {id}`；请求不包含原始配置，响应不包含环境值或凭据。
- 最新模式绑定最新版本副本并清除未复制的父版本引用；全量模式保留版本顺序及映射后的来源关系，并绑定源服务当前版本对应的副本。最新组件集合不一致时拒绝并提示选择全量。
- 复制不调用 `normalizeOverlay` 或定义导入工厂；保留显式空覆盖、与声明相同的显式覆盖及 `${KEY}` 引用。目录使用目标平台校验及环境修订，运行目录、部署历史及运行状态不带入目标。
- 前端复用 AppDialog、ComboboxSelect、SelectControl、DeploymentDirectoryField 和 useDeploymentDirectory；切换目标及关闭重开由既有 composable 重置目录，项目切换使用已有 store。

没有新增迁移、SQL 查询、MCP/CLI 复制入口或前端状态封装。目标目录和成功跳转的修订均来自用户已确认要求。

## Task deliveries and integration / 子任务交付与集成

核对基线：`557763af3bea5f4570c64a95978b0c5f259f9e00`。交付为当前工作树相对基线的产品 diff，共 21 个文件；以 `git diff HEAD -- internal proto web` 的原始输出计算 SHA-256：`fb55cf7a51cb7a1f751d760bdbf90b00318bbf8f1a9b1ef1ce5c3db193e5b4e4`。本轮检查前后产品 diff 未变化，过程文档不纳入该指纹。

| ID | 当前有效交付版本 / 位置 | 必要检查与审查证据 | 集成与基线核对 | 验收归属与结果 |
| --- | --- | --- | --- | --- |
| T01 | [后端任务](../task/20261106-cross-project-service-copy/T01-backend-copy.md)，T01-002；上述工作树的后端、Proto 及生成物 | 真实用例与端口替身、临时 SQLite 回读/回滚、HTTP 协议及事务装配测试；完整 Go 测试及静态检查通过 | 现有 HTTP UoW 覆盖复制路由，两端生成类型一致；复核后维持 accepted | A02–A07、A08 后端通过 |
| T02 | [前端任务](../task/20261106-cross-project-service-copy/T02-frontend-copy.md)，T02-003；上述工作树的前端接入 | 页面与共享 composable/Router/Store 静态核对；43 文件/193 单测、lint、类型与格式检查通过 | 使用 T01-002 的目录/修订输入和 ID 响应；复核后维持 accepted | A01、A08 前端及 A04/A07 交互通过，运行态限制见下文 |

## Corrections and scope / 修复与范围

| 对齐依据 | 发现的偏差 | 实际修复 | 复验结果与证据 |
| --- | --- | --- | --- |
| 意图 A01–A08、计划修订 3、当前有效 task | 本轮未发现实现偏差 | 无产品修复；补齐最终验证、task 交付复核及归档记录 | 下述检查通过，产品 diff 指纹未变 |

最终产品变更与计划一致：应用/服务复制 DTO 和用例及测试，专用 HTTP handler 与路由装配，Proto 和 Go/TS 生成物，前端 API、服务详情和双语文案。目录修订另在已有 composable 测试补充 Linux 默认路径断言，无新增前端测试文件。过程文档为同名意图、计划、两个 task 与本验证记录，完成后按阶段归档。

## Acceptance results / 验收结果

| 验收 ID | 结果 | 证据或未满足原因 |
| --- | --- | --- |
| A01 | 通过（代码核对及轻量检查） | ServiceDetail 右上复制入口；activeProjects 排除源项目；SelectControl 默认 latest、可选 all；useDeploymentDirectory 读取目标环境工作区并允许编辑，现有单测含 Linux 默认路径。成功切换目标项目并使用返回 ID 导航，详情 watch 当前 ID 重新加载；未据此宣称浏览器实测。 |
| A02 | 通过 | `TestCopyApplicationUsesSelectedVersionScopeAndRemapsLineage`、`TestCopyServicePreservesRawConfigurationAndSelectsVersion` 和 `TestCopyServicePersistsIndependentConfiguration` 验证两种范围、独立 ID、版本绑定、组件归属与 lineage。 |
| A03 | 通过 | 用例及 SQLite 回读断言配置、文件内容、`${PORT}` 和原始 Service 环境值、显式空值与覆盖保留；Windows 源到 Linux 目标使用编辑目录并保存目标修订。平台不符、缺失目录和目标修订变化由目录边界测试拒绝。 |
| A04 | 通过 | `TestCopyServiceRollsBackAllResourcesAndRejectsUnauthorizedCopies` 分别验证应用名称、应用编码及服务编码冲突；已有目标资源不变。前端提交错误留在弹窗，不改名或合并。 |
| A05 | 通过 | 临时 SQLite 验证源/目标成员权限拒绝，以及应用已创建后环境写入失败时全部新资源回滚；现有 bootstrap 事务装配测试包含复制路由，沿用提交后响应机制。 |
| A06 | 通过 | 配置回读对比确认源未变，目标 stopped、无运行目录；代码核对清除 CurrentDeploymentId、RuntimeTargetRevision，未调用部署、文件迁移、SSH/Docker 或路由操作。 |
| A07 | 通过 | 原样保留环境值及引用，无凭据生成/轮换；`TestCopyServiceHTTPContract` 确认响应仅包含新 ID。代码核对请求仅含目标、范围、目录和修订，复制用例未打印原始配置；错误沿用安全 writer。 |
| A08 | 通过 | 下列已有检查入口全部通过；测试覆盖主要流程和真实失败边界。SQLite 测试保留 `!unit` 分类，前端未新增整页、并发、性能、E2E 或覆盖率测试。 |

## Check results / 检查结果

以下命令均在本轮验证实际执行；没有因检查失败进行复跑。`task test` 的核心 Go 单测使用已有缓存，不作计时结论；完整 Go 验证另用 `-count=1`。

| 命令 / 检查 | 实际结果 | 未运行原因 |
| --- | --- | --- |
| `corepack yarn lint:fix`（web） | 通过，未产生产品修改 | 不适用 |
| `task check` | 通过；包含 web typecheck/lint/Prettier 与 Go config/fmt/lint，0 issues | 不适用 |
| `task test` | 通过；web 43 文件/193 用例及核心 Go 单测 | 不适用 |
| `go test -count=1 ./cmd/... ./internal/... ./sql` | 通过，包含复制的本地持久化/协议/事务验证 | 不适用 |
| `corepack yarn test --reporter=default --silent=false --no-color`（web） | 43 文件/193 用例通过，无警告 | 不适用 |
| `git diff HEAD --check`、归档文档相对链接与最终产品指纹核对 | 通过 | 不适用 |
| 浏览器交互、真实复制跳转及窄屏布局 | 未运行，结论限于代码核对及已有轻量测试 | 本轮未使用现成运行环境；遵循不启动开发服务器、不以现有项目做资源写入验证的边界 |
| 外部数据库、Docker、远端 SSH 与部署联调 | 未运行 | 不属于配置复制交付范围，沿用测试原有显式启用条件 |

## Risks and incomplete items / 风险与未完成项

- 无阻塞交付的实现问题。浏览器选择/提交/失败反馈、实际导航和窄屏布局未实测，不能从静态检查推定运行态均已验证；本轮未添加复杂 UI 自动化。
- 除部署目录外的挂载来源、端口和业务连接值按要求原样保留，是否适用于目标环境由用户复制后编辑确认。配置复制不代表数据库、外部文件或镜像数据已经迁移。
- 实际部署及 Git 提交/推送不属于本轮范围；保留既有暂存区，归档文档在工作区供后续统一审阅。

## Conclusion / 结论

实现与已接受的意图、计划修订 3 及当前有效 T01-002 / T02-003 对齐，A01–A08 的代码和检查证据成立，交付接受。本次没有产品修复；完成整体 Verification 并将同名意图、计划、task 和验证记录归档到 `docs/archive/specflow/`，保留相互引用。浏览器运行态验证限制如上。
