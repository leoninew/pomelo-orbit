# Verification / 验证：服务目录不以历史环境修订号阻断操作
最后修改时间: 2026-10-16 16:36:50

## Review status / 审查状态

Accepted

## Scope / 范围

对应[意图](../intent/20261016-service-directory-revision.md)和[计划](../plan/20261016-service-directory-revision.md)。2026-10-16，用户反馈“实测可行，完成验证文档”，据此进入并完成 Verification。验证依据为已完成的开发检查、当前任务 diff 复核及用户实测确认。

## Intent alignment / 意图对齐

保存目录的历史 Environment 修订仅记录来源，操作按当前选定目标解析路径。运行目录优先、确认目录次之；目录缺失、归属、文件范围及执行期间目标变化规则继续生效。没有为消除 409 而重写存量修订或服务状态。

## Plan alignment / 计划对齐

复用现有用例、runtime 和 Traefik session 边界，不新增持久化身份、端口、SSH/Docker 命令受理 I/O 或确认 UI。实施与计划一致，交付基线为 HEAD `e8511cf8d786df17e1c9fcfa065f011b2add3dd6` 加当前工作区本任务 diff，没有创建提交。

## Task deliveries and integration / 子任务交付与集成

不适用。主会话单任务顺序实施并审查实际 diff。

## Actual diff summary / 实际差异摘要

- `service_location.go`：停止、重启及运行查询不再比较保存目录的历史修订，缺失目录的结构化错误与诊断上下文保留。
- `directory_ownership.go`：其他 Service 的非空确认/运行目录均参与当前目标的占用解析，继续比较逻辑目录、daemon 可见路径及实际 bind mount。
- Traefik `publication.go`：Gateway 的历史运行目录来源不再阻断工作区访问，仍要求记录运行目录并使用当前目标和文件范围。
- `useDeploymentDirectory.ts`：保存过的确认目录跨修订回填，无目录时才生成默认值，提交仍使用当前环境修订。
- 产品模型、运行时、部署指南与路由指南同步上述语义。

## Expected vs actual changed files / 预期与实际修改文件对比

实际为计划中的 4 个生产代码文件、5 个相邻测试文件、4 个活文档和 3 个过程文档。测试文件为 `deployment_directory_test.go`、`directory_ownership_test.go`、`runtime_tools_test.go`、`publication_cost_test.go`、`useDeploymentDirectory.test.ts`。没有迁移、schema、DTO 或数据库数据变更。

工作区同时存在其他会话的 Route 变更，包括路由指南中的目标展示说明；已保留，不属于本任务差异。固定检查反映执行当时的整个工作区结果。

## Acceptance criteria checklist / 验收标准检查清单

A01—A06 的自动化证据如下；用户实测确认作为原故障场景的补充验收依据，详细记录见后文。

| 标准 | 实际证据 | 开发检查结果 |
| --- | --- | --- |
| A01 | `TestLifecycleOperationsUseHistoricalDirectoriesWithCurrentTarget` 覆盖 stop/restart × runtime/confirmed：来源 1、当前目标 3，验证受理快照、来源保留及执行时目录/目标。 | 通过 |
| A02 | `TestRuntimeStatusAddsVersionIdentityToObservedContainers` 使用历史运行修订验证目标解析、Compose PS 与状态；`TestHistoricalTargetRevisionAllowsExplicitRepublish` 保留 Gateway 历史修订完成显式发布；缺失目录回归继续拒绝。 | 通过 |
| A03 | `TestDirectoryOwnershipProtectsHistoricalDirectories` 覆盖确认/运行旧目录的重叠拒绝及独立路径放行；`TestDirectoryOwnershipChecksNestedDockerMountSources` 覆盖旧来源下的 daemon 嵌套路径；既有实际挂载和 `TestSharedMountDoesNotReleaseServiceRoot` 通过。 | 通过 |
| A04 | `TestDeploymentTargetSnapshotRejectsChangedEnvironment` 保留排队后目标变化拒绝；Traefik 历史修订发布测试及 `TestGatewayPublicationTracksCustomRuntimeDirectory` 保留 session 内目标/目录变化导致预览过期。相关生产校验未修改。 | 通过 |
| A05 | composable 3 项轻量测试验证默认值、跨修订回填、路径规则及使用当前修订的提交载荷。 | 通过 |
| A06 | Go/Web 定向回归、固定开发检查及普通前端单测通过；4 个活文档已同步。 | 通过 |

## Test / command results / 测试 / 命令结果

沿用项目 Taskfile、Go 和 Web package scripts；以下命令均已于实施阶段实际执行并成功结束。本次复核确认任务代码 diff 与已检查实现一致，仅补充过程文档，未重复运行 Go/Web 全量检查。

| 命令 | 实际结果 |
| --- | --- |
| `go test -count=1 ./internal/application/deployment/usecase ./internal/infrastructure/external/traefik` | 两个受影响包通过。 |
| `yarn --cwd web test src/composables/useDeploymentDirectory.test.ts --reporter=default --silent=false --no-color` | 1 文件、3 测试通过。 |
| `yarn --cwd web lint:fix` | 通过。 |
| `yarn --cwd web test --reporter=default --silent=false --no-color` | 40 文件、167 测试通过，开启日志输出，无警告。 |
| `task check` | Web typecheck/lint/format、golangci-lint 配置/格式/lint 检查均通过，0 issues。含固定要求的 `yarn --cwd web typecheck`。 |
| `go test ./cmd/... ./internal/...` | 全部通过。 |
| `git diff --check` | 通过。 |

## User validation / 用户实测与确认

- 2026-10-16，用户针对本问题反馈：“实测可行，完成验证文档”。原故障场景据此判定通过，交付已获用户确认。
- 实测证据来源为用户反馈；其余验收项依照上述自动化检查及实际 diff 核对，不将此次反馈扩写为逐项现场操作记录。
- 本次复核确认目录选择、占用保护、Gateway 工作区及 Web 回填实现与计划一致，没有新增代码变更或范围偏差；更新文档后执行差异格式和文档链接检查。

## Missed or expanded scope / 遗漏或扩大的范围

无实施偏差。Gateway、目录占用和 Web 回填均为计划内的同类规则修正，没有扩展到路由展示或数据库同步。

## Risks / 风险

- 当前选定 Environment 决定执行主机，保存路径由该目标解释；未建立跨主机物理目录身份比对，相同路径字符串不证明跨主机目录等价。
- 当前目标中的目录、Compose 文件或受管资源不可用时，仍由既有 runtime 和归属检查返回失败。本次不自动发现或迁移旧目标。

## Incomplete items / 未完成事项

无未完成项。用户已完成实测并确认可行；agent 未额外操作真实服务、改写业务数据库或重启开发服务器。

## Conclusion / 结论

验证完成，结论为通过，文档状态 Accepted。A01—A06 有对应开发证据，用户已确认原故障场景实测可行。历史目录修订不再单独阻断操作，目录占用及执行期间目标变化保护保留；已知边界如上，无范围偏差或待处理事项。
