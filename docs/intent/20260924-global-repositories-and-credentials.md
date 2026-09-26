# 代码仓库与仓库凭据跨项目复用
最后修改时间: 2026-09-26 00:26:53

Review status: Accepted

Mode: standard

## Background

改造前，Repository 和 Repository Credential 均以 `project_id` 归属 Project。仓库列表、详情、更新和删除按 Project 过滤；仓库 `code` 在同一 Project 内唯一；凭据名称、查询和删除引用检查也按 Project 处理。Application Pipeline 必须绑定本 Project 的 Repository，Pipeline Run 按 Project 读取仓库及其凭据。切换 Project 后，即使要使用同一个 Git 仓库和同一份凭据，也需要重新创建资源。

全局 Template Pipeline 和阶段模板已能跨 Project 复用，但它们不绑定实际源码。仓库及仓库凭据跨项目使用将改变资源归属、权限、唯一性、引用完整性和历史数据处理，不能只放开列表查询。

仓库凭据的详情和导出接口当前会返回解密后的数据；Repository 自定义变量也可能包含敏感配置。扩大可见范围时需要同时审视明文读取、编辑、删除和运行时使用的授权边界。

## Goal

1. 使所有 Project 共用同一份全局代码仓库，并可分别在自己的 Application Pipeline 中选择和使用它，不要求每个 Project 复制一份仓库记录。
2. 使所有 Project 共用同一份全局仓库凭据，并保留密文存储与运行时取用能力。
3. 让跨项目引用、读取、更新和删除采用一致的权限与引用规则；一个 Project 的使用关系不能被另一个 Project 的操作意外破坏。
4. 对已有项目内仓库与凭据给出明确的数据收敛方式，保留仍有效的 Pipeline 绑定、Run 历史和凭据引用，不默默丢弃或混并数据。
5. 在 Web 的仓库、凭据管理页及 Pipeline 仓库选择器中一致呈现可使用的资源；当前 Project 仍决定 Application Pipeline、Run、Environment 和制品的归属。

## Non-goal

- 不将 Application Pipeline、Pipeline Run、Artifact、Application、Environment 或部署凭据改为跨项目资源。
- 不将 Environment 的 SSH 部署私钥并入仓库凭据；两类凭据继续保持独立。
- 不引入每个 Project 一份的仓库或凭据自动复制与同步机制。
- 不增加逐 Project 授权名单或新的仓库、凭据细粒度 RBAC；本轮沿用现有登录及当前 Project 成员校验。
- 不借此次变更放开未登录访问，或绕过当前 Project 的运行、Application 和 Environment 权限检查。
- 不以 Pipeline Run、Snapshot 或 Artifact 的历史引用阻止仓库删除，也不为共享仓库增加跨 Project 工作区隔离或配置更新限制。
- 不新增迁移版本或新旧作用域双轨逻辑；按用户明确要求就地修改已有迁移 SQL。

## User scenarios

1. 用户在 Project A 建立仓库和仓库凭据后，在 Project B 创建 Application Pipeline 时可以选择同一仓库；两个 Project 的 Pipeline 和 Run 仍分别归属各自 Project。
2. 用户更新共享仓库的 URL、默认分支、凭据或变量后，后续使用该仓库的 Project 读取到同一份当前配置；已有 Run 继续展示其创建时冻结的信息。
3. 用户尝试删除被任一 Project 的现存 Application Pipeline 绑定的仓库时，系统要求先删除相关 Application Pipeline。若某条 Application Pipeline 还有未结束的 Run，须先等待其结束或取消，才能删除该 Pipeline；已结束的 Run、Snapshot 和 Artifact 保留历史，不阻止随后删除仓库。删除被任一仓库引用的凭据时，须先解除仓库中的凭据引用。
4. 当前 Project 成员通过仓库或凭据管理页检索并维护全局资源；凭据详情与导出沿用当前可读取明文的行为，不新增 admin 专属限制。

## Acceptance

- [ ] 所有 Project 读取同一份全局仓库与仓库凭据；同一资源能从两个不同 Project 的正常工作流中被查到、选择和使用，不靠复制实现。
- [ ] 两个 Project 可分别将同一仓库绑定到自己的 Application Pipeline，触发后的 Run、目标 Environment、日志和制品继续按各自 Project 隔离。
- [ ] 仓库与凭据的 API、SQL 查询、用例、前端列表和选择器采用同一套已确定的共享与授权语义；跨项目调用不能通过资源 ID 绕过鉴权。
- [ ] 凭据继续密文存储；当前 Project 成员可按现有行为读取详情明文、导出和管理全局凭据；普通列表及日志不泄露明文，Pipeline 执行可取得源码访问凭据。
- [ ] 仓库与凭据的创建、命名冲突、更新和删除在共享范围内行为一致；仓库删除检查覆盖所有 Project 中现存的 Application Pipeline 绑定，凭据删除检查覆盖所有仓库的引用。Pipeline Run、Snapshot 和 Artifact 均不作为仓库删除条件。
- [ ] 删除 Application Pipeline 时，`waiting_to_run` 或 `running` 的 Run 阻止删除；Run 进入 `ran_to_completion`、`faulted` 或 `canceled` 后可删除 Pipeline，历史 Run、Snapshot 和 Artifact 继续保留。
- [ ] 既有仓库、凭据和有效绑定在数据转换后仍可使用；跨 Project 的重复 `code`、凭据名称及可能不同的配置有确定的处理规则，不静默覆盖。
- [ ] Web 的仓库、凭据与 Pipeline 页面清楚显示共享资源；Project 切换后列表、详情和选择状态不会残留不适用的数据。
- [ ] 仓库和凭据作为选择控件数据源时可搜索完整全局集合；流水线运行记录与制品筛选保留可搜索控件，超过单页 100 条时仍能找到并显示现存目标资源。已删除仓库不再出现在选择器中，相关历史 Run 和制品仍保留其名称快照。
- [ ] 覆盖跨项目复用、运行时凭据读取、权限边界、跨项目删除保护和既有数据转换的主要成功路径与高风险失败边界。

## Open questions

1. 既有不同 Project 的同名仓库 `code`、同名凭据和可能重复的实际仓库应如何处理？进入计划时暂定保留各自 ID 和配置，发现冲突先审计并处理，不自动合并。
2. 当前 `local_directory` 仓库是否也纳入跨项目共享？进入计划时暂定纳入，但继续遵守原有本地路径和 SSH 来源约束。
3. Repository 自定义变量若含敏感值，跨项目列表、详情与运行时使用应采用何种查看权限和脱敏规则？进入计划时暂定沿用现有行为，不增加新的变量权限或展示规则。

## Decisions

- 本 Intent 是独立的新需求，不沿用或修改此前 Project 租户作用域、全局模板任务的过程文档。
- Repository 与 Repository Credential 都是所有 Project 共用的全局资源，不设置逐 Project 授权名单。
- 本轮不增加 admin 专属或其他细粒度权限；API 保留当前 Project 作为成员校验上下文，现有成员可管理、查看和导出全局凭据明文。
- Application Pipeline、Run 及运行目标继续由当前 Project 管辖；共享仓库与凭据不改变这些资源的归属。
- 仓库删除只受所有 Project 中现存 Application Pipeline 的绑定阻止；Pipeline Run、Snapshot 和 Artifact 对仓库是逻辑历史引用，不直接参与仓库删除校验。
- 删除 Application Pipeline 前须检查同一 Project、同一 Pipeline 的 Run；`waiting_to_run` 和 `running` 均视为未结束并阻止删除。`ran_to_completion`、`faulted` 和 `canceled` 是终态，不阻止删除 Pipeline，其历史引用与冗余展示字段保留。
- 各 Project 继续使用自己的 Environment；本任务不处理多个 Project 主动配置相同物理目标的情况。共享仓库配置可按现有权限更新，不增加对已创建 Run 的更新限制。
- 仓库凭据与 Environment 部署凭据保持不同资源和用途。
- 用户明确要求就地修改已存在的迁移 SQL，不新增迁移版本；这是一项对仓库默认迁移规则的本任务例外。
- 仓库与凭据选择器须覆盖全局集合；流水线运行记录和制品筛选须使用支持搜索的控件。
- 用户要求改用标准模式并进入 Plan；其余开放问题按本 Intent 所列假设推进，作为计划风险保留。

## User review notes

- 用户提出：“代码仓库和仓库凭据，我们要改成跨项目可用的”。
- 用户确认“所有 Project 共用一份全局资源”。
- 用户说明“目前只有 admin 角色，先不限制”，因此本轮不新增凭据管理或明文读取的细粒度限制。
- 用户要求切换标准模式并完成开发计划，视为接受本 Intent。
- 用户随后要求不新增迁移、就地修改已有迁移 SQL，并补充仓库与凭据选择控件场景，特别要求流水线和制品筛选支持搜索。
- 用户明确仓库删除仅检查现存 Application Pipeline 绑定，Run、Snapshot 和 Artifact 的逻辑历史引用不阻止删除；未结束 Run 的删除限制应放在 Application Pipeline 删除处。已删除仓库不再从下拉框选择。
- 用户确认各 Project 的 Environment 自行隔离，本任务不处理人为共用相同物理环境，也不增加共享仓库配置更新后的运行保护。

## Risk

- 全局共享凭据沿用现有“当前 Project 成员可读明文/导出”的权限，会扩大密钥暴露范围；本轮按用户决定保留该行为。
- 仓库当前可保存自定义变量，修改一份共享仓库会影响多个 Project 的未来运行；需明确其敏感值和变更责任。
- 仓库 `code` 在改造前按 Project 唯一，旧数据中可能出现跨项目同名不同配置；凭据名称也可能重复，不能假定自动合并安全。
- 改造前删除检查和 Pipeline Run 执行都按 Project 读取仓库或凭据，若仅放开管理页会造成跨项目绑定或执行失败。改造前 Application Pipeline 删除没有检查未结束 Run；实施时须在该删除路径补上限制，才能保持仓库删除只看现存 Pipeline 绑定的规则。
- 修改已执行迁移 SQL 不会自动改变现有数据库；实施时需单独核对并处理已运行数据库的 schema 与数据。
