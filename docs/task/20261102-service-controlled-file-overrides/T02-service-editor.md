# Task / 子任务 T02：服务挂载与受控文件编辑
最后修改时间: 2026-11-02 21:39:17

## Review status / 审查状态

Accepted

## Parent and authority / 父任务与权威记录

- 总意图：[服务受控文件覆盖](../../intent/20261102-service-controlled-file-overrides.md)。
- 权威总计划：[计划](../../plan/20261102-service-controlled-file-overrides.md)，位于 `D:/SourceCodes/mywork/pomelo-orbit`；任务 ID：T02；原分配方案修订2，最终方案修订3由主会话完成并复核。
- 依赖、执行者、编号和仲裁结论以总计划为准，本文只记录局部方案与事实。

## Handoff basis / 交接依据

- 已分配给主会话 01a19b5a-d828-7fc1-abfb-aed164b1fb46，编号 T02-20261102-172330-main，用户已授权 Implementation 及必要检查。
- 协议入口：`C:/Users/wangm25/.codex/plugins/cache/specflow-local/specflow/0.2.1/skills/specflow/SKILL.md` 及 `references/task-coordination.md`。
- 基线为 HEAD 09eda33e8d19ab6a255e8088db053829dd1c0d0d + 本次主会话核对接受的源 Proto 字段 diff；生成 DTO 由 T01 独占产出。
- 执行者可写权威根下本文接收、进展及交付章节；主会话读取记录，默认由用户转交可复制摘要，不自动通知其他会话。

## Goal and acceptance / 目标与验收

主要交付：沿用原挂载弹窗直接编辑来源；目录和普通文件提供只读，受控文件展开内容、权限和写入策略，不提供只读或继承/自定义操作。

覆盖 A05、A02–A03 交互部分及 A09 前端部分：刷新回读保持选择、显式同值不变继承、空/false 不丢失、保留模式说明初始化语义、请求保留其他覆盖。

## Inputs and eligibility / 输入与可开始条件

直接前置以总计划修订2为准：无前置实现任务，使用已核对源 Proto 和计划固定的字段存在性契约；T01 生成完成后再运行 typecheck，最终集成不得遗漏后端验收。不开辟兼容接口或修改生成物。

## Scope and files / 范围与文件

- `web/src/views/service/ServiceComponentDetail.vue`；本轮统一版本入口 `VersionComponentDetail.vue` 与共享 `ControlledFileWritePolicy.vue`。
- 新增同目录 `serviceComponentMount.ts` 和轻量单测，必要时提取局部编辑组件。
- 项目现有 Dialog/Drawer、MonacoEditor 的复用，`web/src/i18n/locales/zh-CN.ts`、`en-US.ts`。
- `docs/handbook/reference/compose.md`：服务覆盖、写入策略、声明内容与运行文件、重建生效边界。
- 不修改协议源或生成 DTO，不改后端/Gateway 策略；发现缺口回报主会话协调 T01。

## Technical decisions / 局部技术设计

1. 挂载编辑状态同时保留版本值、当前草稿及每字段是否自定义，不用值相等判断继承。
2. source/source_is_host_path 使用同一继承模型；非受控文件支持 read_only，controlled_file 展示 content/mode/ignore_if_exists。target/source_type/shared 沿用版本声明。
3. 字段直接编辑，不展示继承/自定义和字段恢复控件；用户编辑时记录对应字段，未操作字段保持原状态。挂载列表的整条重置删除 overlay。
4. 版本内容编辑和服务挂载编辑共用写入策略控件，只有“覆盖已有文件”“保留已有文件”两项，不显示来源，不新增容器权限联动。保留模式中内容和权限仍可编辑，但明确提示只在文件不存在时初始化，已有文件在重部署或强制重建时保持不变。
5. 使用已有 MonacoEditor，代码字号 12px，其余文字遵循项目样式；不散落底层 Reka primitives。
6. 只序列化显式字段，全继承不发送 mount 条目；显式 content 空串、布尔 false 保留，deleted 不带任何值。保留完整组件中不相关覆盖。
7. 内容不 trim、不替换变量；用 UTF-8 字节数校验 256 KiB，mode 校验 `0000`–`0777`。字段错误就地显示，保存失败不关闭编辑丢失草稿。
8. 保持当前页面保存边界，不额外引入自动保存。编辑器展示声明内容，不展示为服务器当前文件；不按 Traefik/ACME 特判可选策略。

## Implementation outline / 实施安排

1. 建立版本声明+覆盖到表单状态的转换和稀疏请求生成，先验证存在性。
2. 接入现有挂载编辑入口，受控文件展开内容、权限和写入策略，字段直接编辑。
3. 更新挂载变更标记、提交/回读和中英文提示，确保取消/失败不误保存。
4. 更新操作文档并完成定向轻量单测和前端质量检查。

## Checks and delivery / 检查与交付

- 遵循 `.agents/skills/frontend-unit-testing/SKILL.md`：以转换规则及小范围交互为主，不整页串联 Router/Store/API，不增加并发、性能、集成或端到端测试。
- 覆盖直接编辑、未操作字段保留、整条重置、显式同值、空内容/false、保留初始化说明、非法 mode/超限及完整 overlay 请求保留无关字段。
- 在 `web/` 运行 `corepack yarn lint:fix`、`corepack yarn typecheck`、`corepack yarn test --reporter=default --silent=false --no-color`，核对测试警告；不新增格式或测试工具。
- 提供实际 diff/版本位置、使用的 T01 基线、界面和请求行为证据、检查结果与未运行原因。只做代码/轻量测试时不声称已完成真实部署验证。

## Risks / 风险

旧页面提交所有挂载为 override，会把继承值固化；必须改为字段存在性驱动。只靠 mountHasChanges 的值差异会漏掉显式同值配置。Gateway 现有 enrichment 不在本任务修改，交付不能暗示业务运行策略已同步调整。

## Reception and progress / 接收与进展

- 2026-11-02 17:23:30：主会话接收，编号 T02-20261102-172330-main，修订2。依据用户“开始实现/继续”实施，已核对总计划、源协议和文件隔离。工作区为权威根，基线 HEAD 09eda33e8d19ab6a255e8088db053829dd1c0d0d + T01 源协议 diff；无前置实现任务，生成结果等待 T01，不同时写后端或生成文件。
- 2026-11-02 17:32:16：完成表单状态转换、独立挂载 Dialog、稀疏提交及中英文说明，5 个定向单测通过；生成 DTO 已取得，typecheck 通过。
- 2026-11-02 17:36:14：完整前端 44 文件 / 197 测试通过，无日志警告；首次 lint 发现新增代码括号和非空断言问题，已修正，重新执行 lint:fix 和 typecheck 均通过。
- 2026-11-02 17:39:07：主会话执行最终 task check 通过，含前端类型/lint/格式及 Go 格式/lint（0 issues）；后端交付记录待 T01 完成，未进入整体 Verification。

## Delivery and review / 交付与审查

- 交付时间：2026-11-02 17:39:07；执行者为主会话；编号 T02-20261102-172330-main，修订2。
- 交付位置：权威工作区 HEAD 09eda33e8d19ab6a255e8088db053829dd1c0d0d 上的未提交 diff；前置合同为总计划修订2核对的源 Proto，实际使用 T01 已生成的 web/src/gen/proto/orbit/v1/service/service.ts。
- 实际变更：新增 serviceComponentMount.ts 与5个行为单测、局部 ServiceComponentMountDialog.vue；ServiceComponentDetail.vue 改用独立编辑器和稀疏 payload，移除旧的来源值比较及全量固化挂载；增加中英文文案及 Compose 手册说明。
- A02–A03：已覆盖显式同值、空内容、false、独立mode/source、恢复继承、删除恢复及编辑草稿隔离；测试跨版本回读保留自定义内容。
- A05：内容、权限、只读及策略可编辑，控件显示继承/自定义；保留模式说明初始化用途；Dialog确认只更新页面草稿，保存组件后仍等待显式部署。内容不是服务器文件镜像。
- A09：定向5测试通过；完整前端44文件197测试通过且无警告；lint:fix、typecheck和最终task check通过。首次lint问题已修复，未隐藏规则。
- 范围偏差：为避免扩展主页面，使用任务方案允许的局部Dialog组件；无新增协议、后端或业务策略改造。没有额外UI测试框架。
- 未运行：浏览器人工操作、真实Docker/SSH及生产部署；当前为实现开发检查，不宣称运行态整体验收。
- 未完成实现项：无。已知边界：Gateway仍有独立enrichment；其分析在总计划，未宣称业务策略已改造。下游影响：新增生成DTO随T01发布，必须同时交付两端；无待仲裁问题。
- 2026-11-02 20:10:03：按用户截图精简挂载弹窗，移除类型和目标路径展示，受控文件不再显示源路径编辑及其继承提示；保留权限、只读、写入策略与内容编辑。其他挂载类型继续支持源路径编辑，现有源路径覆盖仍随草稿保留。`corepack yarn lint:fix`、`corepack yarn typecheck` 和 `git diff --check` 通过；本次仅移除控件，未新增测试。
- 2026-11-02 20:22:02：用户进一步明确沿用原挂载弹窗，直接操作字段，受控文件展开内容编辑，不需要继承/自定义交互。恢复类型、来源和目标的原布局，去掉全部继承/自定义控件与文案；只读及宿主路径直接勾选，权限和内容直接编辑，写入策略只有覆盖/保留两项。改动字段自动保存覆盖，未操作字段保留原状态；整条重置保留。删除旧字段切换 helper，将测试调整到直接编辑，并补充空内容和取消只读/保留的请求检查。前端 lint:fix、typecheck、44 文件 198 项单测和 git diff 检查通过，无测试警告；未启动或重启服务器。
- 2026-11-02 20:43:58：先更新父级意图、计划和手册，再统一实现两项产品策略。新增共享 ControlledFileWritePolicy，版本内容编辑与服务挂载弹窗共用，移除版本的“已存在时忽略”勾选及重复文案；保留模式说明编辑内容和权限仅初始化缺失文件，重部署与强制重建不覆盖已有文件。继续使用既有布尔字段，不新增策略类型、字段适配或兼容层。lint:fix、typecheck、44 文件 198 项单测通过且无警告；统一检查首次发现两处格式问题，已用项目 Prettier 修正，最终 task check 与差异检查通过。未新增重复的控件映射测试，未运行浏览器人工检查。
- 2026-11-02 21:37:09：主会话按修订3移除版本与服务受控文件的只读选项及列表值，普通文件和目录保留；更新轻量单测。前端 lint:fix、typecheck、44 文件 198 项单测通过，无警告。与当前后端校验及 DTO 集成复核通过；未进行浏览器人工检查。
