# Verification / 验证：服务受控文件内容与写入策略覆盖
最后修改时间: 2026-11-02 21:39:17

## Review status / 审查状态

Accepted

用户已要求整体审查并完成 SpecFlow 验收。本报告核对最终用户修订后的产品行为，不以早期子任务完成声明代替整体结果。

用户随后明确受控文件不支持只读，目录和普通文件保留只读，并要求不为迁移编写测试；此前验证基线已变化，须以修订后的结果为准。

## Intent alignment / 意图对齐

依据[意图](../intent/20261102-service-controlled-file-overrides.md)：服务可编辑受控文件内容、权限和部署策略，产品仅提供覆盖/保留两项；直接编辑，不增加继承/自定义交互。不实现远程在线编辑，也不自动部署。

## Plan alignment / 计划对齐

依据[计划](../plan/20261102-service-controlled-file-overrides.md)。审查基线为 HEAD `09eda33e8d19ab6a255e8088db053829dd1c0d0d` 加本轮暂存、未暂存及未跟踪文件；不修改 Git 暂存状态。字段沿用 optional/NULL，业务校验字段组合，数据库保留简单约束。Gateway 只取消强制覆盖，其他业务改造仍有独立边界。

## Task deliveries and integration / 子任务交付与集成

| ID | 当前有效交付版本 / 位置 | 必要检查与审查证据 | 集成与基线核对 | 验收归属与结果 |
| --- | --- | --- | --- | --- |
| T01 | 当前工作区后端、协议、迁移、SQLC diff | 复核模型、Repository、协议、文件适配器和 Gateway，Go 测试通过 | Proto 与生成 DTO 对齐；修订3边界由主会话复核 | A01–A04、A06–A07、A09 后端部分通过 |
| T02 | 当前工作区服务编辑器、共享策略控件、版本入口与语言包 diff | 复核直接编辑、草稿隔离及稀疏提交，前端测试通过 | 使用 T01 当前生成 DTO，修订3界面与后端校验一致 | A02–A03、A05、A09 前端部分通过 |

## Actual diff summary / 实际差异摘要

- 服务挂载增加独立可空的内容、权限和保留策略覆盖；只读覆盖仅适用于非受控文件，保存与部署共用模型合并校验。
- HTTP/MCP 可读完整声明及覆盖；MCP 新增组件详情读取，避免整体替换时丢失无关覆盖。
- SQLite/MySQL/PostgreSQL `000056` 与 SQLC 查询贯通新字段，取消跨字段复杂 CHECK。
- 文件 writer 使用实际权限，保留模式不写现有文件；Gateway 不再清零保留策略。
- 服务直接编辑、整条重置；版本与服务共用覆盖/保留选择。按用户要求移除覆盖模式解释文案，宿主路径标签改为“使用 Docker 宿主机路径”。

## Expected vs actual changed files / 预期与实际修改文件对比

- 协议、模型、用例、Repository、迁移、HTTP/MCP、runner、前端入口和手册符合计划。
- 所有 SQLC 包的共享模型由生成器同步增加四字段，属于必要生成差异。
- 新增局部 Dialog 和共享策略控件，避免整页复杂化；版本入口同步采用相同产品语义。
- 用户要求不为迁移编写测试，此前新增的迁移专用测试已移除。

## Acceptance criteria checklist / 验收标准检查清单

- A01–A03：声明、稀疏覆盖与有效计划贯通；空值、false、同值和整条重置由模型、Repository、前端转换测试覆盖。
- A04、A06：文件测试验证初始化、覆盖、保留、内容及权限；Gateway 部署测试核对强制重建仍传递保留策略。
- A05：版本与服务共用两项策略；受控文件不设置只读，目录和普通文件保留只读；代码与轻量测试核对。
- A07：HTTP/MCP 字段存在性及完整回读、结构校验和业务边界通过定向测试。
- A08：Linux 容器执行实际 local writer 测试；隔离 Docker 单文件挂载验证原子替换、重建读取新文件和应用写入后重建保留。二者是互补边界验证，不是完整服务部署端到端测试。
- A09：前端轻量测试与后端项目测试通过，无新增前端端到端或迁移专用测试。
- A10：计划记录 Gateway 当前覆盖点、ACME、Route、凭据、版本、重建的目标与后续调整顺序；本轮仅取消强制覆盖策略。

## Test / command results / 测试 / 命令结果

- `task check`：通过；首次发现两个 Vue 格式问题，格式化后复跑通过。
- `go test -count=1 ./cmd/... ./internal/... ./sql`：通过；最后调整 HTTP/MCP 和 Repository 测试后，相关三个包定向复跑通过。
- `corepack yarn lint:fix`、`corepack yarn typecheck`、`corepack yarn test --reporter=default --silent=false --no-color`：通过，44 个文件、198 项测试，无警告。
- Linux 容器执行 `TestWorkspaceFilePreserveAndExplicitZeroMode`：通过，验证保留内容、权限和显式零权限；首次执行的 PowerShell 参数错误修正后通过。
- 隔离 Docker 挂载检查：通过，原子替换后旧容器仍见旧文件，重建见新文件，应用写入后再次重建仍保留。无真实服务操作。
- `git diff --check`：通过。按用户要求未编写迁移专用测试。

## Missed or expanded scope / 遗漏或扩大的范围

用户追加的数据库简单约束、直接编辑、统一两项策略、Gateway 不强制覆盖以及文案修改均在本次整体审查范围。远程 Casdoor 单次回填和 MCP 进程清理属于会话内操作，不是产品远程回读功能。

## Risks / 风险

- 保留模式下平台内容更新不改变已存在文件；Gateway 的新配置也可能因此不生效，符合用户所选策略。
- Gateway 仍修改生成内容、选版和部分挂载并强制重建，本轮不宣称完成整体业务治理。
- Docker 挂载和 Linux writer 为分开执行的边界检查，不代表真实 SSH 或完整生产部署已验证。
- 迁移 down 会丢失新增覆盖数据，不属于无损回退。生产环境不在本轮验收操作范围。
- `000056` 将既有受控文件版本声明的只读值清零，重部署后这些挂载不再由 Docker 强制只读；该调整无法通过 down 恢复。本地开发库现有 59 条受控文件均为非只读，未修改数据。

## Incomplete items / 未完成事项

未运行真实 SSH、MySQL/PostgreSQL 迁移、浏览器人工检查或生产部署；这些外部环境结果不在本次验收中声称通过。

## Conclusion / 结论

整体 Verification 通过，受控文件通用能力及 Gateway 策略分析符合本轮确认的范围。Gateway 其余业务改造不属于已交付结果。
