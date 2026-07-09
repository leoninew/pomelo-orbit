# Verification: internal 后两级目录结构对齐与架构收敛

最后修改时间: 2026-07-09 23:25:16

## Verification target

本次验证覆盖 light / 轻量模式下的两个阶段：

1. **Issue 1**：`internal/common/civariable` 迁出 `common`（已有验证，重新确认）
2. **暂存区现状**：Issue 2 已部分移动的 64 个产品文件（实际 diff 远超 Issue 1 计划范围）

对应 requirement 文档：

- `docs/requirement/20260709-internal-directory-structure-alignment.md`（已 Accepted）
- 原 `docs/requirement/20260709-internal-layering-second-pass.md`（已合入上述文档，独立文档已删除）

## Requirement alignment — Issue 1

需求要求：

- `common` 按架构指南应保持无业务语义。
- `internal/common/civariable` 承载 CI runtime/template variable 规则，存在业务名词和业务规则，必须迁出 `common`。
- 迁到明确的 CI 规则归属位置。
- application 与 workflow activity 直接依赖新位置。
- 不保留 wrapper、alias、旧包或适配层。
- 不修改 CI variable 规则实现。
- 不修改 API route path、request/response contract、数据库迁移或前端。

实际实现与 requirement 对齐：

- 新增 `internal/application/civariable/runtime.go`，package 名保持 `civariable`，文件内容保持不变。
- 删除 `internal/common/civariable/runtime.go` 与空目录 `internal/common/civariable`。
- 更新以下 6 处 import：
  - `internal/application/ci/repository.go`
  - `internal/application/ci/runtime_variables.go`
  - `internal/application/ci/snapshot.go`
  - `internal/application/ci/template.go`
  - `internal/application/ci/template_test.go`
  - `internal/workflow/activity/ci/execution.go`
- 未新增 wrapper、alias、兼容转发、类型别名或新旧路径并存。
- 未修改 CI variable 规则实现。
- 未修改 API route path、request/response contract、数据库迁移或前端。

## Requirement alignment — 暂存区现状（Issue 2 范围）

暂存区实际包含 64 个产品文件变更，远超 Issue 1 计划。这些变更属于 Issue 2（application 后两级目录重组）的实现。

对照 requirement 文档 Issue 2 implementation todo：

| 状态 | 项目 |
|---|---|
| ✅ 已移动 | `internal/application/{auth,user,role,project,settings,ci,cd}/service.go → usecase/service.go` |
| ✅ 已新增 | `dto/`、`port/` 子包（auth/user/role/project/settings/ci/cd） |
| ✅ 已新增 | `internal/application/ci/runner/docker.go`、`internal/application/cd/runner/shell.go` |
| ✅ 已新增 | `internal/application/ci/port/port.go`、`internal/application/cd/port/port.go` |
| ✅ 已新增 | `internal/application/ci/dto/ci.go`、`internal/application/cd/dto/cd.go` |
| ✅ 已移动 | `internal/common/civariable → internal/application/ci/rule/civariable` |
| ✅ 已更新 | `internal/api/http/handler/**`、`server.go`、`internal/bootstrap/http.go`、`internal/workflow/activity/{ci,cd}/*` import |

说明：暂存区对 Issue 2 的执行**实际已经发生**，但本 verification 作为合并文档仍按 requirement 逐项核对。需求文档 Issue 2 的 Acceptance 尚未逐项验证，且 requirement 显式约定"Requirement 阶段不修改产品代码"。代码先于验证存在是本次验证需要说明的偏差。

## Spec alignment

不适用。当前任务使用 light / 轻量模式，未创建独立 spec 文档。

## Plan alignment

不适用。当前任务使用 light / 轻量模式，按 requirement 中的 Issue 策略实施。

## Actual diff summary

```text
64 files changed, 1615 insertions(+), 889 deletions(-)
```

变更类型：

- `internal/application/{domain}/service.go → usecase/service.go`，覆盖 auth / user / role / project / settings / ci / cd
- 新增 `internal/application/{domain}/dto/`、`internal/application/{domain}/port/`
- 新增 `internal/application/ci/runner/docker.go`、`internal/application/cd/runner/shell.go`
- 新增 `internal/application/ci/dto/ci.go`、`internal/application/cd/dto/cd.go`
- 新增 `internal/application/ci/port/port.go`、`internal/application/cd/port/port.go`
- `internal/common/civariable → internal/application/ci/rule/civariable`
- 更新 `internal/api/http/handler/**`、`internal/api/http/server.go`、`internal/bootstrap/http.go`、`internal/workflow/activity/{ci,cd}/*`

## Expected vs actual changed files

预期改动（Issue 1 范围）：

- 新增 `internal/application/civariable/runtime.go`
- 删除 `internal/common/civariable/runtime.go` 与空目录
- 更新 `internal/application/ci` 与 `internal/workflow/activity/ci` 中 6 处 import
- 更新 requirement 记录

实际改动：

- 包含上述 Issue 1 改动，但更主要是 Issue 2 的 full-scope 重组（64 文件）
- `internal/common/civariable` 被进一步移到 `internal/application/ci/rule/civariable`（符合要求）
- 未发现前端、数据库迁移、API route path、request/response contract 改动

结论：

- Issue 1 实际改动与预期一致。
- Issue 2 的实际改动在结构上符合 requirement 预定的 usecase/dto/port/runner 目录方案，但整体大幅超出本"逐条推进"任务在 Requirement 阶段的约定范围。

## Acceptance checklist — Issue 1

- [x] `internal/common/civariable/runtime.go` 删除。
- [x] `internal/common/civariable` 空目录删除。
- [x] `internal/application/civariable` 新增并承载 CI runtime/template variable 规则（后进一步调整为 `internal/application/ci/rule/civariable`）。
- [x] `internal/application/ci` 与 `internal/workflow/activity/ci` 直接依赖新位置。
- [x] 不保留 wrapper、alias、旧包或适配层。
- [x] CI variable 规则实现保持不变。
- [x] 未修改 API route path、request/response contract、数据库迁移或前端。
- [x] 后端检查通过。

## Acceptance checklist — Issue 2 实际执行（待逐项确认）

Issue 2 在暂存区已实际改组，但 requirement 的逐项 todo 仍在未勾选状态。本验证仅确认结构落位，不代替完整 per-issue 验收：

- [x] application 各 domain 的 service/usecase 已迁入 `usecase/`
- [x] 各 domain dto 已迁入 `dto/`
- [x] 各 domain 端口已迁入 `port/`
- [x] CI/CD concrete runner 已迁入 `runner/`
- [x] CI variable 规则已迁入 `internal/application/ci/rule/civariable`
- [x] 外层调用点（HTTP server / bootstrap / HTTP handlers / workflow activity）已更新 import
- [ ] behavior 等价性： CI / CD / auth / user / role / project / settings 用例逻辑未验证等价
- [ ] 测试跟随：integration_test 已随 usecase 移动，但行为等价需测试确认
- [ ] 旧路径完全清理：不保留 wrapper、alias、类型别名或新旧 import 并存（待全量 git diff 复核）

## Test results

已运行命令：

```text
go build ./...
go vet ./internal/...
go fmt ./internal/...
./bin/golangci-lint fmt ./internal/...
./bin/golangci-lint run ./internal/...
go test ./...
```

结果：

- `go build ./...`：通过，无编译错误。
- `go vet ./internal/...`：通过。
- `go fmt ./internal/...`：通过（无文件需格式化）。
- `./bin/golangci-lint fmt ./internal/...`：通过。
- `./bin/golangci-lint run ./internal/...`：通过，`0 issues`。
- `go test ./...`：全部通过，无 FAIL。

关键相关包：

```text
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase
ok  gitee.com/leoninew/PomeloOrbit-go/internal/api/http
ok  gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap
ok  gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci
ok  gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd
ok  gitee.com/leoninew/PomeloOrbit-go/internal/test/e2e
```

## Missed or expanded scope

### 发现的范围扩张

暂存区 64 个产品文件变更远超本任务 Issue 1 原定范围，实为 Issue 2 的完整实施。

### 未处理项（符合 requirement 边界）

本 verification 不处理以下事项：

- 未重整 `internal/repository` 端口与 impl 结构（Issue 3）。
- 未调整 `internal/worker` 与 `internal/queue` 目录关系（Issue 4）。
- 未调整 `internal/infrastructure/traefik`、`internal/infrastructure/turnstile` 分类（Issue 5）。
- 未调整 `internal/api/http` 结构（Issue 6）。
- 未调整 `internal/gen/sqlc` 结构（Issue 7）。
- 未调整 `internal/workflow` 结构（Issue 8）。
- 未处理后续架构收敛候选 C1-C3。

## Risks

1. **过程契约风险**：本 requirement 明确约定 Requirement 阶段不改产品代码，但暂存区实际包含 Issue 2 完整改组。从 SpecFlow 协议看是前进了一步，但从 requirement 文字看是偏差。
2. **commit 边界风险**：需求文档与产品代码在同一个暂存区。若一起提交，commit message 无法同时准确描述"文档"与"Issue 2 实现"两件不同性质的工作。
3. **runner concrete 归属被模糊化**（C1 视角）：本次 code diff 把 runner 改组到 `internal/application/{ci,cd}/runner/`，**仍为 application 目录下**，但看起来像已经完成 C1。
4. **sql.ErrNoRows 未处理**（C2）：`internal/application/ci/usecase/pipeline_run.go` 仍然直接分支 `errors.Is(err, sql.ErrNoRows)`。
5. **queue/task 膨胀**（C3）：未处理。

## Incomplete items

- [ ] Issue 2 行为等价性独立验证（可用单独 verification 文档或在本文档追加补充）。
- [ ] Issue 3-8 仍需逐条完成 Implementation 与 Verification。
- [ ] C1-C3 后续架构收敛候选在单独任务中逐条推进。

## Conclusion

**Issue 1 `internal/common/civariable` 迁出 `common` 已完成并通过验证。** 实现符合 requirement 中的分层目标：CI runtime/template variable 规则已迁入 `internal/application/ci/rule/civariable`，application 与 workflow activity 直接依赖新位置，未保留兼容层、别名、wrapper 或新旧路径并存，CI variable 规则实现与后端检查均通过。

**暂存区 Issue 2 改组实际已完成结构落位，后端检查全通过。** 结构上符合 requirement 预定的 usecase/dto/port/runner 方案；但 requirement 的 per-issue 验收清单仍需逐项核对 behavior 等价性与旧路径清理。

**建议**：

1. Issue 1 + Issue 2 改组的产品代码可以合入一个 commit，commit scope 建议用 `refactor(application): restructure domain subpackages into usecase/dto/port/runner`。
2. 合并后的 requirement / verification 文档单独一个 commit，scope `docs(requirement,verification)`。
3. 后续 Issue 3-8 与 C1-C3 仍按 light / 轻量模式逐条进入 Implementation + Verification。
