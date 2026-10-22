# Plan / 计划：服务目录不以历史环境修订号阻断操作
最后修改时间: 2026-10-16 16:36:50

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

- [意图](../intent/20261016-service-directory-revision.md) A01—A06。
- 用户已确认物理目录原则并要求“记录问题并实施”，授权补齐前置记录、实现和必要开发检查。
- 实施基线：HEAD `e8511cf8d786df17e1c9fcfa065f011b2add3dd6`，工作区已有 Route 相关实现/生成文件修改，不属于本次改动。
- [后端架构](../archive/architecture/backend.md)、[CD 产品模型](../archive/cd-model.md)、[CD 运行时](../archive/architecture/cd-runtime.md)、项目后端治理及前端单测 skill。

## Technical decisions / 技术设计与决策

- `runtimeServiceLocation` 仍要求运行目录存在于记录中，但移除历史运行修订号相等的条件。`lifecycleServiceLocation` 的确认目录选择同样不比较历史修订。
- 不在命令短事务中增加 SSH/Docker I/O。Worker、运行时查询和 Route 工作区复用现有当前目标路径解析、文件范围及 Compose 归属；不新增 identity 持久化、端口或基础设施。
- `checkDirectoryOwnership` 遍历其他 Service 的非空确认/运行目录，统一在当前目标解析并比较逻辑路径与 daemon 可见路径；保留实际 bind mount 和 shared 规则。
- Gateway 发布工作区只对缺失运行目录保留原错误，历史修订不阻断；发布来源 revision、session 和 preview 变化检查不变。
- Web 有确认目录即回填，缺失才根据当前 Environment 生成候选；提交继续携带当前 target revision，路径有效性照常校验。
- 不删除 schema/DTO 中的目录修订字段，不改写存量数据。部署仍保存当前确认/运行来源，历史记录、状态及接替语义不变。

## Implementation steps / 实现步骤

1. 修正目录选择、目录占用、Gateway 工作区和 Web 回填的历史修订条件。
2. 在既有用例测试中覆盖历史修订启停受理/当前快照、状态和工具查询、旧修订目录碰撞及不重叠、Gateway 历史目录来源；复用排队后和 session 内目标变化断言。
3. 调整 Web 目录回填单测，确认当前修订仍进入提交载荷；不扩展为页面集成或 UI 竞态测试。
4. 更新产品模型、运行时、部署指南及路由指南，区分历史目录来源和执行期间目标变更保护。
5. 按下述检查记录实际结果；主会话审查最终 diff 和范围，将实现结果写入本计划。

## Files to change / 需修改文件

- `internal/application/deployment/usecase/service_location.go`、`directory_ownership.go` 及相邻目录/运行时测试。
- `internal/infrastructure/external/traefik/publication.go`、`publication_cost_test.go`。
- `web/src/composables/useDeploymentDirectory.ts`、对应测试。
- `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/deployment.md`、`docs/guides/routing-and-certificates.md`。
- 同名 Intent / Plan / Verification 过程记录。

## Tasks and coordination / 子任务与协调

不适用。主会话顺序实施；不修改或还原既有无关变更。

## Verification plan / 验证计划

- 有针对性的 Go 目录/运行时/Traefik 回归与 Web composable 单测。
- 修改前端后：`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`；按前端 skill 运行显示日志的普通单测。
- 修改 Go 后端后：`task check`、`go test ./cmd/... ./internal/...`。
- 检查实现没有删改 `verifyDeploymentTargetSnapshot`、Route session/preview 变更校验、权限与文件范围规则。
- 不自动启动、停止或重启开发服务器，不对真实服务发起停止/部署/路由同步。
- 必要开发检查在 Implementation 完成；用户随后反馈实测可行并要求完成验证文档，已据此完成整体 Verification，结果见同名验证记录。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- 实际文件和 Compose 不存在等问题仍由目标 runtime 反馈，不因删除数字比较而补造目录或伪造成功。
- 非空旧路径会按当前目标解析并参与占用保护；不能以旧修订为由忽略另一个 Service 已记录的保留目录。
- `task check` 覆盖整个工作区，若既有无关变更失败，区分来源并保留真实结果。

## Rollback / recovery / 回滚 / 恢复

按本任务最终 diff 恢复规则与文档；不包含数据库变更和迁移，无需数据回滚。不覆盖其他工作区改动。

## User review notes / 用户审查记录

- 2026-10-16：基于用户明确实施授权接受方案；暂无需确认事项。
- 2026-10-16：用户反馈“实测可行，完成验证文档”，已确认实施结果并授权完成 Verification；验证记录更新为 Accepted。

## Implementation / 实施记录

- 已完成，实现范围与计划一致。交付为当前工作区相对上述 HEAD 的任务 diff；没有创建提交。
- 目录选择和 Gateway 工作区移除历史目录修订相等限制；目录占用纳入其他 Service 的所有非空确认/运行目录；Web 跨修订回填已保存目录。继续由当前选定目标解析路径，不改写数据库目录来源。
- 修改 4 个生产代码文件、5 个测试文件、4 个活文档，并维护同名 Intent / Plan / Verification。路由指南中其他会话的 Route 展示段落及其他无关工作区改动完整保留，不计入本任务交付。
- 新增/调整回归覆盖历史目录来源的 stop/restart 受理及执行、当前任务目标快照、运行状态/Compose 查询、逻辑/daemon 目录占用、Gateway 显式发布和 Web 提交载荷。既有任务排队后目标变化、发布 session 内目标/目录变化和共享挂载根目录保护测试仍通过；对应生产校验未删除。
- `task check` 通过（Web typecheck/lint/format、Go lint/format，0 issues）；`go test ./cmd/... ./internal/...` 通过。Go 定向回归、Web 定向 3 项及普通单测 40 文件/167 项、`yarn --cwd web lint:fix` 均通过。
- 开发证据、验收映射及用户实测确认见[验证记录](../verification/20261016-service-directory-revision.md)。暂无实施偏差或未完成项；整体 Verification 已完成，验证记录为 Accepted。
- 未改写业务数据库，未对真实服务启停、部署或同步路由，未启动或重启开发服务器。实际目录及 Compose 的可用性仍由当前目标 runtime 判断；本次不提供跨主机物理目录身份比对。
