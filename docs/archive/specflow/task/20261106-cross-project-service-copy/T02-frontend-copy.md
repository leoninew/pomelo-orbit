# Task / 子任务 T02：前端复制入口与项目选择弹窗
最后修改时间: 2026-11-06 22:23:46

## Review status / 审查状态

Accepted

用户已确认入口在服务详情页；方案及 T01/T02 产品交付均已 Accepted。

## Parent and authority / 父任务与权威记录

- 总意图：[跨项目复制服务](../../intent/20261106-cross-project-service-copy.md)。
- 权威总计划：[计划](../../plan/20261106-cross-project-service-copy.md)；任务 ID：T02；方案修订：3。
- 权威协调根：`D:/SourceCodes/mywork/pomelo-orbit`；主会话 `/root` 唯一登记分配与审查。

## Goal and acceptance / 目标与验收

- 在服务详情页提供复制按钮，弹窗选择目标项目及复制版本范围，显示并允许编辑目标目录；默认最新版本，用户可改选全部版本。
- 对应 A01、A08 前端部分及 A04/A07 的交互部分；除目标部署目录外不改配置，成功切换目标项目并跳转新服务，不展示凭据。

## Scope and files / 范围与文件

- `web/src/api/service/service.ts`：新增使用生成类型的 copy 方法。
- `web/src/views/service/ServiceDetail.vue`：右上操作区的复制按钮、页面内弹窗与局部状态；用户已确认该入口。
- `web/src/i18n/locales/zh-CN.ts`、`en-US.ts`：复制操作、目标项目、必填、空选项、成功及失败文案。
- 复用现有 `AppDialog`、`AppDialogActions`、`ComboboxSelect`、project store、错误样式与操作状态，不新建 store/composable/专用弹窗组件，不改后端、Proto 或生成物。

## Inputs / 输入

- 直接前置与放行条件以总计划 T02 行为准：T01 接口及生成类型已交付并经主会话接受，实施时记录具体版本。
- 固定契约：`POST /api/service/:service_id/copy?project_id=<源项目ID>`；请求 `ServiceCopyReq { target_project_id, copy_all_versions, deployment_directory, environment_target_revision }`，默认 false 复制最新版本，true 复制全部版本；响应 `ServiceCopyResp { id }`。
- 复制只写配置，要求目标环境配置用于目录计算，但不执行运行环境探测；权限和名称/编码冲突由后端裁定。前端不预查服务或应用列表断言唯一性。

## Technical decisions / 局部技术设计

- `serviceApi.copy(projectId, serviceId, payload): Promise<ServiceCopyResp>` 沿用现有 request wrapper；不手写请求类型、错误解析或兼容响应。
- 右上“复制”按钮在源服务已加载时可用，禁用状态沿用 `operating`；复制没有运行状态门槛，不因源服务正在运行而要求停止或部署。
- 弹窗包含必填目标项目选择及版本范围 SelectControl，默认“最新版本”，另可选择“全部版本”。最新模式只复制最新版本及组件且副本服务绑定该版本；全量模式复制全部版本且副本服务绑定源服务当前版本的对应版本。项目选项使用 `projectStore.activeProjects` 排除源项目，label 为项目名称，description 为编码；不重复存储选项或选择值。
- 项目列表尚未加载时使用现有 fetchProjects，加载失败明确反馈，不把失败显示成“没有项目”。无候选项目时提示没有可用目标并禁用确认，不默认选中某项目。
- 局部状态包含弹窗开关、目标项目 ID、版本范围、字段错误、提交错误，并使用独立的已有 useDeploymentDirectory 实例管理目标目录；版本范围每次打开默认最新。操作忙碌状态复用页面 `useStatusAsync`。不增加规范化、缓存或持久化逻辑。
- 选择目标并成功加载目录前禁用确认；目录采用共享字段的必填、平台校验及字段错误反馈。业务文字沿用项目字号，不采用技能示例中的 12px 字段文字。
- 异步冲突、权限或网络错误显示在弹窗提交区，不伪装成本地必填错误；失败不关闭弹窗、不清空目标选择，下一次提交清除旧提交错误。
- 使用 `AppDialogActions` 的当前弹窗确认事件，不以外部 form 属性连接背景表单。提交期间按钮禁用，open-change 处理阻止关闭；取消、关闭和重开均清理当前弹窗状态。
- 成功后使用复制返回的新 ID 跳转新服务详情，并通过已有 setActiveProject 切换目标项目；Toast 显示目标项目和服务编码。详情页监听路由 ID 重新加载，不新增 storage 写入或调用部署入口。

## Implementation outline / 实施安排

1. 接收并核对 T01 已接受的类型与接口，在已确认的服务详情页实现，不重新定义协议。
2. 增加 API 方法、双语文案、复制按钮和页面内弹窗。
3. 接入项目选项、必填反馈、操作状态、失败保留选择与成功反馈，检查关闭重开状态。
4. 完成现有前端静态检查及轻量测试，记录未运行的界面验证与限制，提交交付证据。

## Checks and risks / 检查与风险

- 静态审视：版本范围控件默认最新并可选全部，提交的 `copy_all_versions` 与选择对应；目标项目、版本范围和可编辑目标目录，无名称/编码/其他配置编辑；源项目不可选；项目加载失败与空列表不同；冲突留在弹窗；成功切换到目标项目并跳转新服务；请求过程中不可重复提交或关闭。
- 前端测试遵循项目技能：不挂载整页串联 Router、Store 和 API，不写并发、集成、E2E 或 Portal/focus 测试；本任务的简单局部赋值及薄 API 方法不强制新增测试文件。新增独立业务规则时再补必要轻量单测。
- 在 `web/` 运行 `corepack yarn lint:fix`、`corepack yarn typecheck` 与 `corepack yarn test --reporter=default --silent=false --no-color`；记录真实错误与警告，不为测试抽出生产 helper。
- 不启动、停止或重启开发服务器。没有获授权的运行环境时，空提交、选择、失败、成功及窄屏弹窗的运行态检查如实标为未验证，由整体 Verification 安排。
- 风险：项目切换沿用 setActiveProject；必须先使用复制返回的新 ID，避免以目标项目请求源服务 ID。

## Handoff basis / 交接依据

已分配 T02-001 给主会话 `/root`；T01 当前工作树交付已接受。协议入口为已读取的 SpecFlow `SKILL.md` 与 `references/task-coordination.md`；总计划与 task 方案已 Accepted，实施授权见总计划。派发时由主会话核对 T01 已接受版本，再登记编号与工作基线。

- 可写回报位置：本文“接收与进展”“交付与审查”；主会话同步总计划登记。
- 回报渠道：当前会话直接记录并输出事实摘要。

## Reception and progress / 接收与进展

接收：2026-11-06 21:17:29，T02-001 / `/root`。T01 已经主会话接受，使用同一工作树的接口及生成类型，开始前端实施。

| 时间 | 分配编号 / 执行者 | 接收、进展或阻塞 | 证据 / 回报位置 |
| --- | --- | --- | --- |
| 2026-11-06 20:25:27 | 无 / 无 | 局部方案建立；用户确认服务详情入口，等待总计划接受、T01 交付及实施授权 | 本文与总计划修订 1 |

## Delivery and review / 交付与审查

交付：2026-11-06 21:26:49，T02-001 / `/root`。使用 T01 已接受的当前工作树 HTTP 接口与 TypeScript 生成类型。

- 已在 ServiceDetail 右上加入复制按钮，页面内使用 AppDialog、ComboboxSelect、AppDialogActions 及已有 SelectControl。按用户纠正改为现有选择框；版本范围每次打开默认 latest，可选 all，提交时直接映射到 copy_all_versions。
- A01：目标列表来自已有 project store 的 activeProjects，排除当前项目；无候选时提示并禁用确认，不预查目标环境。项目未加载时使用现有 fetchProjects，加载失败显示错误。
- A04/A07 交互：项目必填错误归属选择框，异步失败保留目标和范围并在弹窗显示；操作期间控件及操作按钮禁用，open-change 阻止关闭。成功提示目标项目与服务编码，留在源详情，无项目切换或 storage 写入。
- API wrapper 和双语文案已完成。未新增 store、composable、弹窗组件或前端测试文件；此处是薄 API 与局部状态，不以整页测试串联 Store/Router/API。
- A08：web 的 corepack yarn lint:fix、typecheck、Prettier 均通过。task check 通过（Go lint 0 issues），task test 通过；显式日志命令 corepack yarn test --reporter=default --silent=false --no-color 为 43 个文件、193 个用例通过，无警告。
- 初次 lint 发现新增 if 缺少大括号与非空断言，已修复。曾在仓库根使用 yarn --cwd，错误选到 Yarn 1 且找不到 Prettier；改为 web 工作目录后使用项目锁定的 Yarn 完成全部检查。
- 静态核对了默认值、载荷、错误清理、忙碌锁定及不切换项目。没有启动开发服务器；空提交、复制成功/失败和窄屏布局的浏览器运行态交互尚未实测，不以静态检查代替。

主会话审查：Accepted。已核对改动范围、共享控件、与 T01 的请求/响应一致性及上述检查。T01/T02 实施完成；正式整体 Verification 尚未进行。


## Revision 2 / 目录调整

2026-11-06 21:37:42：用户授权按目标环境计算并编辑复制目录。局部方案 Accepted，具体契约与检查以总计划修订 2 为准；上轮交付的目录行为已被替代。等待 T01-002 新契约被接受后继续前端调整。

2026-11-06 21:42:50：接收 T02-002，使用 T01-002 已接受的目录输入契约与生成类型；复用现有目录 composable/字段，按目标项目加载并保存编辑值。

修订 2 交付：2026-11-06 22:03:56，T02-002 / `/root`。已复用 DeploymentDirectoryField 与 useDeploymentDirectory，切换目标项目重新计算目录；复制载荷包含编辑值及目标修订，加载失败保留弹窗错误。版本范围继续使用 SelectControl。前端未新增测试文件，仅在现有默认目录测试中补充 Linux 目标断言；未添加整页 UI、性能、并发或覆盖率测试。

主会话审查：Accepted。task check、task test 通过；同一前端代码的 lint:fix、typecheck、显式日志单测通过（43 个文件/193 个用例，无警告）。浏览器交互、窄屏布局及正式整体 Verification 未实测。用户要求继续后保持暂存区不变，本轮记录对应当前工作区。

## Revision 3 / 复制成功跳转

2026-11-06 22:11:01：方案 Accepted，主会话接收 T02-003；依赖仍为 T01-002。成功后切换项目并跳转新服务，详情随 route.params.id 变化加载。

修订 3 交付与审查：2026-11-06 22:15:11，T02-003 Accepted。先设置目标项目，再跳转 `/service/${copied.id}`；详情监听路由服务 ID，所有 API 操作使用当前 ID。lint:fix、typecheck、Prettier 与显式日志单测通过（43 文件/193 用例）。未新增复杂 UI 或其他测试，浏览器导航尚未实测。

## Overall verification / 整体复核

2026-11-06 22:23:46：主会话按用户验证授权复核 T02-003，维持 Accepted；所依赖 T01-002 契约保持有效。本轮无产品修复或新增测试，最终前端交付与产品 diff 指纹见[验证记录](../../verification/20261106-cross-project-service-copy.md)。

静态核对共享控件、默认最新与可选全量、目标目录计算与重置、提交锁定和失败保留选择；复制成功切换目标项目并使用返回 ID 导航，详情随路由 ID 加载，全部服务 API 使用当前 ID。`corepack yarn lint:fix`、`task check` 内的 typecheck/lint/格式和显式日志单测通过（43 文件/193 用例，无警告）。浏览器交互、实际导航及窄屏布局未实测，未运行整页或端到端自动化测试。
