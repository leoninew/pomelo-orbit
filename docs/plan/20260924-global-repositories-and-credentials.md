# 代码仓库与仓库凭据跨项目复用开发计划
最后修改时间: 2026-09-26 00:29:21

Review status: Accepted

Mode: standard

## Intent basis

依据 `docs/intent/20260924-global-repositories-and-credentials.md`。Repository 与 Repository Credential 改为所有 Project 共用的全局资源；当前 Project 继续作为登录用户的成员校验和 Application Pipeline、Run、Environment 的作用域。用户确认本轮不新增凭据明文读取、导出或管理的细粒度权限，并明确要求就地修改已有迁移 SQL，不新增迁移版本。

## Implementation steps

1. **数据审计与既有迁移修订**：先审计现有仓库 `code`、仓库凭据名称在全局范围内的冲突，以及仓库到凭据、Pipeline 到仓库的引用；另按 `type` 审计旧 `credential`/`repository_credential` 中的非仓库凭据，尤其是 `deployment_ssh_private_key`。只把受支持的仓库凭据类型转换为 `project_id = NULL`；部署凭据不可清空归属后暴露为全局仓库凭据，须确认其引用并保持隔离或单独处置。保留有效资源 ID、密文、配置和历史关联，不自动合并或删除；冲突须显式改名后再应用全局唯一约束。就地修改三种数据库的既有建表、种子和相关重建迁移 SQL：仓库及受支持的仓库凭据类型以 `project_id = NULL` 表示全局记录，仓库 `code` 和凭据名称按全局唯一约束建模；同步当前 schema 定义。重点核对 `000019_credential`、`000021_repository`、`000033_seed_pipeline` 与 SQLite `000042_environment_credential` 的 up/down 链路，不新增迁移版本。已执行数据库不会重跑这些文件，需另行审计后按同一终态一次性修正 schema/数据，或在允许重建的环境中重建，记录实际操作与结果。
2. **持久化与领域边界**：将 Repository、Repository Credential 的 SQLC 查询及 Store port 改为按全局 ID、`code`、名称读取和写入，列表不按 Project 过滤；用 `project_id IS NULL` 和受支持的仓库凭据 `type` 共同限定仓库凭据读写，避免历史部署凭据经 ID 路径进入仓库领域。仓库引用凭据的检查覆盖所有仓库；仓库删除检查覆盖所有 Project 中现存的 `Pipeline(kind=application)` 绑定，不查询 Pipeline Run、Snapshot 或 Artifact。为 Application Pipeline 删除增加按当前 Project 与 Pipeline ID 查询未结束 Run 的能力，状态定义沿用 `WorkStatusIsComplete`；移除 Repository 按 Project 计数的旧接口及 Project 废弃时的仓库阻断；保留 Application 对 Project 的检查。重新生成 SQLC 代码。
3. **应用用例与运行时**：仓库和凭据的 HTTP 用例仍以请求中的 `project_id` 校验当前 Project 成员，创建的全局资源不记录 Project 归属。仓库创建/更新校验全局凭据，`code` 和凭据名称按全局查重。Application Pipeline 实例化、详情和 Run 创建/执行按全局 ID 读取仓库与凭据，但其 Pipeline、Run、目标和制品仍按当前 Project 校验和保存。删除仓库时只检查所有 Project 当前的 Application Pipeline 绑定；删除凭据时检查所有仓库的引用。删除 Application Pipeline 时，若其 Run 仍为 `waiting_to_run` 或 `running`，返回可理解的冲突提示并保留 Pipeline；Run 已结束则允许删除并保留历史记录。Template Pipeline 删除不受此规则影响。
4. **HTTP、Proto 与 Web**：保留仓库和凭据 HTTP API 的 `?project_id=` 作为成员资格上下文，不新增路由别名；从 Repository 响应中移除不再代表归属的 `project_id` 字段，并重新生成 Go/Web Proto。前端 API 继续显式传递当前 Project，仓库页、凭据页和详情展示同一套全局资源。仓库与凭据列表 API 继续提供 `search`、`page`、`per_page`；共享 `ComboboxSelect` 仅接收选项和通用加载状态，并发出搜索、翻页及选中事件。`RepositorySelect` 与 `RepositoryCredentialSelect` 分别负责业务查询、映射和回显，使 Pipeline 实例化的仓库选择、仓库表单/详情的凭据选择、Pipeline Run 与 Artifact 的仓库筛选按输入查询完整全局集合，并可翻页加载超过 100 条的结果。远端搜索结果不再被控件按显示标签二次过滤，以免按仓库 `code`、URL 或凭据类型命中后被隐藏。仍存在的已选仓库通过 ID 详情查询保留标签和筛选值；凭据在仓库编辑场景使用已有名称快照回显，避免选择器为显示标签读取明文凭据详情。已删除仓库不再列入选择器。Project 切换时丢弃过期响应并刷新项目内 Pipeline/Run 数据。凭据普通列表继续不含明文，详情和导出保持现有成员可访问行为。
5. **文档与回归**：更新产品概览、CI Pipeline 指南、迁移规则的任务例外记录及有关 Project 归属、仓库凭据的活文档。补充仓库/凭据 Store 与用例测试、Pipeline 实例化及 Run 执行测试、Project 废弃测试和前端选择流程测试；使用两个不同 Project 的成员验证同一资源可见、可绑定与可运行，并验证无成员资格时拒绝访问。

## Expected files

- 就地修改 `sql/migration/{postgres,mysql,sqlite}/000019_credential.*.sql`、`000021_repository.*.sql`、`000033_seed_pipeline.*.sql` 中受影响的定义/种子，并核对及修订 SQLite `000042_environment_credential.*.sql` 的仓库重建与索引；同步 `sql/schema/{mysql,sqlite}.sql`、`sql/query/{repository,repository_credential}/*.sql` 和对应 `internal/gen/sqlc/` 生成代码。不新增迁移文件。
- `internal/model/{repository,credential}.go`、`internal/repository/{vcs_repository,credential}.go`、`internal/repository/impl/sqlc/{repository,credential}/`。
- `internal/application/{repository,credential,pipeline,pipeline_run,project}/usecase/`、`internal/repository/{pipeline,pipeline_run}.go`、`sql/query/pipeline_run/pipeline_run.sql`，重点包括 Repository/凭据 CRUD、Pipeline 绑定、Application Pipeline 删除时的未结束 Run 检查、Run 凭据读取、跨项目删除检查及 Project 废弃逻辑。
- `internal/api/http/handler/{repository,credential}/`、`proto/orbit/v1/repository/repository.proto`、`internal/gen/proto/`、`web/src/gen/proto/`。
- `web/src/components/{ComboboxSelect,RepositorySelect,RepositoryCredentialSelect}.vue`、`web/src/api/{repository,credential}/`、`web/src/views/{repository,credential,pipeline,pipeline_run}/` 中实际受全局搜索、翻页、已选回显和项目切换影响的文件。
- `docs/product/overview.md`、`docs/guides/ci-pipeline-design.md`、`docs/decisions/ledger.md`、`docs/architecture/backend.md` 中与本次迁移例外冲突的记录及其他受影响活文档；相关定向测试。

## Verification plan

1. 在隔离的 PostgreSQL、MySQL、SQLite 空库从头运行修改后的完整迁移链及回退路径，检查种子、仓库凭据外键、全局唯一约束与最终 schema。对已有数据库先记录重复 `code`/名称、引用计数和旧凭据类型；仅转换 `git_ssh`、`github_token`、`gitee_token`、`gitea_token`、`registry_token`，确认其他类型及其引用已隔离后，再备份并执行经核对的一次性修正或明确选择重建。验证迁移版本未新增、全局记录终态、ID/引用不变，并记录旧库采用的具体修正方式。冲突未处理前不执行修正。
2. 运行 `task sqlc`、`task proto`，检查生成物与 SQL/Proto 定义一致。定向 Go 测试覆盖两 Project 共享读取/绑定、运行时凭据、跨项目 Application Pipeline 绑定阻止仓库删除、同一 Pipeline 的等待/运行中 Run 阻止其删除、终态 Run 允许删除 Pipeline 且历史保留、解除全部 Pipeline 绑定后可删除仓库、跨仓库凭据删除保护、非成员拒绝及 Project 废弃。
3. 运行仓库要求的 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`；补充受影响的 Web 选择器/项目切换测试并运行 `yarn --cwd web test`。用超过 100 条全局仓库与凭据验证服务端搜索、翻页、已选值回显，以及 Pipeline Run 和 Artifact 的可搜索筛选；覆盖搜索词只命中仓库 `code`/URL、未命中显示名称的情况。最后执行 `git diff --check`，核对实际 diff 只包含本需求和必要生成物。
4. 核对仓库、凭据 API 在两个 Project 中返回相同资源；Pipeline 和 Run 列表仍只返回当前 Project；凭据列表无明文，详情/导出按已确认的当前成员权限工作。检查 Pipeline Run 与 Artifact 筛选控件支持输入搜索、能选中第一页以外的现存仓库，且已删除仓库不再成为选项；相关历史记录仍可查看。

## Blockers and assumptions

- 历史跨 Project 同名 `code` 或凭据名称会阻止全局唯一约束建立；本计划采用迁移前审计和显式改名，保留 ID 与关联，不自动合并。具体冲突记录需在实施时根据目标库实际数据列出。
- 旧 `credential` 在迁移 000042 中整体重命名为 `repository_credential`，历史非仓库类型不会自动迁入 `environment_credential`。旧库转换前须按类型及引用审计，不能把部署私钥的 `project_id` 清为 `NULL`；仓库凭据 SQL 查询同时按类型限制，防止误转换后经 ID 读取或绑定。
- 本任务由用户明确授权修改已执行迁移 SQL，与仓库默认迁移规则不同；修改后的文件只定义从头建库的终态。现有数据库需单独修正或在确认可重建后重建，不能误以为编辑文件会自动升级。
- 当前仓库/凭据列表每页上限为 100，现有选择控件只搜索已加载选项；全局资源规模超过 100 时必须由列表 API 的搜索和分页支撑控件。仍存在的已选资源通过 ID 详情回显，不依赖当前结果页；已删除仓库不提供选择器选项。
- `local_directory` 仓库按同一全局模型处理；原有本地路径校验和 SSH 来源限制继续生效，跨 Project 共享不意味着任意执行目标都能访问该目录。
- Repository 自定义变量的展示与使用沿用现有行为；它们随仓库一起全局共享。若其中保存敏感值，本轮不引入额外脱敏或权限模型。
- 各 Project 继续使用各自 Environment；人为将不同 Project 配置到相同物理目标不属于本任务。共享仓库配置更新沿用现有行为，不增加更新限制或新的 Run 快照机制。
- Project Code 校验已由独立提交 `40bd9a678` 完成；本任务修改 Project 废弃逻辑时保留该提交的校验规则及测试。

## Risks and rollback

- 全局凭据可由任一通过当前 Project 成员校验的用户查看和导出明文，这是用户确认的暂行权限边界；未来出现其他角色时需重新评估。
- 全局仓库配置变化会影响多个 Project 的后续 Run；已有 Run 的历史快照、日志和制品不能因共享改造而被重写。
- Application Pipeline 删除与 Run 创建可能并发；两条路径须在请求事务内锁定同一 Pipeline，并检查锁更新的命中行数。MySQL 须按匹配行而非实际变更行报告 `RowsAffected`，且使用 `READ COMMITTED`，使等待锁后的未结束 Run 查询看见先行事务提交的 Run；其默认 `REPEATABLE READ` 会复用等待锁前建立的快照。仓库删除仍只检查现存 Application Pipeline 绑定，Run/Snapshot/Artifact 的逻辑引用与名称快照不构成数据库删除阻断。
- 将旧 `project_id` 清为 `NULL` 后，原归属无法仅从当前数据恢复。处理已有数据库前必须保留快照及冲突改名记录；若需回退，先停止写入并检查新增跨 Project 引用，再由快照恢复。修改既有 down SQL 只保证从头运行的迁移链自洽，不代表已运行数据库可自动恢复原归属。

## User review notes

- 用户确认所有 Project 共用一份全局资源，且当前只有 admin 角色，本轮先不加细粒度限制。
- 用户要求切换标准模式并完成开发计划；Plan 已接受，后续按补充的删除约束进入实施阶段。
- 用户明确要求不新增迁移、就地修改已有迁移 SQL，并考虑仓库/凭据作为选择控件的数据源；流水线运行记录和制品筛选须使用支持搜索的控件。
- 用户明确仓库删除只检查现存 Application Pipeline 的跨 Project 绑定，Run、Snapshot 和 Artifact 都不直接阻止仓库删除；未结束 Run 的限制放在 Application Pipeline 删除处。已删除仓库不再从选择器选取。各 Project 沿用自身 Environment，不新增共用物理目标或仓库配置变更的运行保护。
