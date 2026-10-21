# Intent / 意图：环境凭据以环境为归属主表
最后修改时间: 2026-10-21 13:17:59

## Review status / 审查状态

Accepted

## Background / 背景

当前每个 Project 只有一个 Environment；SSH Environment 使用受管密钥完成 Probe、CI/CD、Web 终端与 CLI 连接。凭据是环境的附属数据，但当前持久化关系由 `environment` 保存 `ssh_credential_id`、`ssh_credential_revision`，`environment_credential` 则保存 `project_id`，没有直接保存其所属环境。

这种设计把凭据归属和当前绑定分散在两张表中。应用需要同时比较凭据 ID、Project ID 和修订；生成初始化命令时，还可能先创建凭据，再创建环境。用户指出应先有环境，再有环境凭据，要求记录并处理这一主从关系问题。

当前实现依据：

- [环境模型](../../internal/model/environment.go)与[环境凭据模型](../../internal/model/environment_credential.go)。
- [初始化用例](../../internal/application/environment/usecase/ssh_initialization.go)、[密钥创建及匹配](../../internal/application/environment/usecase/ssh_key.go)。
- [目标解析](../../internal/application/environment/usecase/target.go)、[环境配置与交接恢复](../../internal/application/environment/usecase/configuration.go)。
- [部署目标快照](../../internal/application/deployment/usecase/remote_target.go)、[项目交接](../../internal/application/project_handover/usecase/service.go)。
- [环境查询](../../sql/query/environment/environment.sql)与[环境凭据查询](../../sql/query/environment_credential/environment_credential.sql)。

## Goals / 目标

1. 让 `environment_credential` 通过 `environment_id` 直接归属于 Environment，明确先保存环境、再创建或读取其凭据的生命周期。
2. 收敛环境与当前受管密钥的关系，避免环境指向凭据、凭据又通过 Project 间接表达归属。讨论中的方案为 Environment 对当前凭据 `1 → 0..1`：local 和未初始化 SSH 环境没有当前凭据，已初始化 SSH 环境拥有一份当前受管密钥。
3. 计划阶段落实删除冗余持久化字段的范围：候选为凭据表的 `project_id`、环境表的 `ssh_credential_id` 与 `ssh_credential_revision`；凭据自身 ID、密钥内容及修订继续表达凭据身份。
4. 同步所有凭据消费入口、事务写入顺序和项目交接流程，保留任务快照及会话对目标、凭据变化的检测能力。
5. 提供既有数据的迁移安排，保留有效环境使用的密钥及必要历史关联，更新活文档和相关验证。

## Non-goals / 非目标

- 不新增凭据管理页面、独立 CRUD 接口、密钥轮换产品能力或多凭据选择能力。
- 不改变 Project 与 Environment 的一对一关系，不扩展多环境模型。
- 不调整仓库凭据 `repository_credential` 的归属或业务。
- 不更换 SSH 密钥算法、Fernet 加密方案或当前全局加密配置。
- 不改变 Probe 的诊断定位、主机指纹校验、项目成员权限或 local/SSH 执行方式。
- 不修改已执行的迁移文件，不引入新旧关系双读、双写或运行时兼容层。

## User stories / scenarios / 用户故事 / 场景

1. 用户保存 local 或完整 SSH 环境配置时，仅形成环境记录；保存行为不会生成 SSH 密钥。
2. 用户首次生成 SSH 初始化命令时，系统先保存环境，再创建归属于该环境的凭据；后续生成命令复用同一环境的有效凭据。
3. 用户发起 Probe、CI/CD、Web 终端或 CLI 连接时，系统从环境取得其凭据，并按现有规则校验身份与修订。
4. 用户迁移已有实例或通过项目交接恢复环境后，环境继续使用预期密钥，凭据关联到目标 Environment。
5. 已排队任务或活动连接遇到环境、凭据变化时，原有拒绝执行或断开规则继续生效。

## Acceptance criteria / 验收标准

- A01：环境凭据拥有明确的 `environment_id` 归属；当前凭据的存在性与唯一性符合已接受计划，不再依赖凭据的 Project ID 推断归属。
- A02：首次 SSH 初始化在同一事务中先保存环境、再建立凭据；失败整体回滚，重复生成命令复用有效凭据。
- A03：保存 local 或未初始化 SSH 配置不生成凭据；目标类型切换时，凭据处置有明确规则，不从历史未绑定凭据中隐式选择密钥。
- A04：Probe、目标解析、CI/CD、终端和 CLI 均使用新关系；任务和会话仍能识别环境或凭据修订变化，历史任务快照不会被当作当前归属依据。
- A05：项目交接导出与恢复适配新关系；环境私钥继续加密持久化，普通环境 API/MCP 不输出私钥，交接包沿用加密私钥传输。
- A06：SQLite、MySQL、PostgreSQL 的新增迁移及 SQLC 查询一致；有效既有绑定迁移后保留原密钥和身份，未绑定或异常历史记录的处置在执行前明确。
- A07：冗余持久化关系按计划移除；模型、仓储、装配、测试及活文档同步收敛，无新旧关系兼容分支。
- A08：完成项目固定 Go 检查 `task check` 与 `go test ./cmd/... ./internal/...`，并补充与关系约束、事务回滚、迁移和运行时消费对应的最小有效验证；若实际修改前端，运行规定的 lint 与 typecheck。

## Workload assessment / 工作量评估

- **主要交付**：环境凭据归属关系、写入生命周期及既有数据迁移的一次完整调整。
- **工作量**：中等。涉及三种数据库迁移、SQL/SQLC、模型和仓储、环境应用用例，以及 CI/CD、终端、CLI 和项目交接的消费链路。前端是否需要修改取决于计划阶段对公开契约的核对。
- **复杂度**：中等。领域关系明确，主要复杂点是移除持久化反向绑定后如何组装运行时目标，以及保持任务快照、会话复核和导入事务一致。
- **风险**：较高，集中在已有密钥归属迁移、未绑定历史记录、失败回滚和修订校验遗漏；不能仅凭编译通过证明正确。
- **关键未知项**：历史凭据是否被当前环境或历史任务引用，类型切换后旧记录如何处理，跨库迁移如何保留数据，以及逻辑关联与数据库约束的具体选择。
- **检查范围与审查上下文**：结合当前模型、全部凭据读取/写入调用点、三库迁移约定和运行时快照规则，验证真实归属、事务及任务行为。实施时遵循项目后端架构治理 skill。

结论：暂保持单任务。存储关系与各消费入口必须一致落地，共享模型、仓储和修订契约耦合紧密，拆开独立交付会增加中间状态。Plan 阶段细化实施顺序和迁移验证；若调查发现可独立交付的额外结果，再调整拆分。

## Candidate tasks and dependencies / 候选子任务与依赖

不适用；当前不建立 task，也不登记分配。

## Open questions / 待定问题

暂无需要用户确认的未决事项。

进入 Plan 时需调查并明确：

- 当前绑定、未绑定历史凭据和历史任务引用的数据迁移策略，不能仅凭相同 `project_id` 自动挑选一份密钥。
- `1 → 0..1` 当前凭据关系的唯一约束、是否使用物理外键，以及 SSH/local 切换后的记录生命周期；保持与项目既有逻辑关联约定一致。
- 移除环境持久化凭据修订后，运行时目标、任务快照及会话复核获取当前凭据身份的具体方式。

## Decisions / 决策

- 用户已明确要处理主从关系问题：`environment_credential` 持有 `environment_id`，环境先于其凭据存在。
- 用户已于 2026-10-21 要求进入 Plan；本意图据此接受，当前授权覆盖计划编写。
- 一环境一份当前受管密钥、移除冗余字段及事务顺序是本次讨论形成的拟议方向，具体约束与迁移设计在 Plan 中落实。
- [20260915 部署密钥归 Environment](20260915-environment-owned-deployment-key.md)记录的是此前拆分仓库凭据与环境凭据的工作。本任务单独记录后续关系调整，不改写其历史审查记录。

## Risks / assumptions / 风险 / 假设

- 当前代码允许存在未被环境绑定的历史凭据；尚未查询实际业务数据库，不能假定所有凭据都能唯一映射到现存环境。
- `1 → 0..1` 指当前业务关系；历史 Deployment/PipelineRun 的凭据快照仍需保留其执行输入语义，不据此扩展密钥历史管理能力。
- 关系迁移应尽量原样保留有效密钥密文，避免因结构调整要求目标机重新授权。
- 初始化、交接恢复和类型切换涉及多表写入，需核对既有 UoW 边界，避免关系调整留下部分写入。

## User review notes / 用户审查记录

- 2026-10-21：用户指出：“主从表设计有问题吧，肯定是先有环境，再有环境凭据，所以 environment_credential 应该持有 environment_id 才对吧”。
- 2026-10-21：讨论确认凭据生命周期归属于环境，并提出当前凭据 `1 → 0..1`、移除冗余反向绑定、先保存环境再创建凭据，以及保留任务凭据快照的方向。
- 2026-10-21：用户要求：“$specflow:specflow 记录 我们要处理这个问题”。据此新建本意图，供后续计划与实施使用。
- 2026-10-21：用户要求“开始计划”，据此将意图状态更新为 Accepted，进入同名计划编写。
