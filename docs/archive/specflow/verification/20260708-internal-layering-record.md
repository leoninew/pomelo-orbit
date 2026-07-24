# internal 分层迁移验证
最后修改时间: 2026-07-08 14:04:39

Review status: Accepted

## Current stage

当前：轻量模式 / light，验证 / Verification。

## Requirement alignment

按 `docs/requirement/20260708-internal-layering-record.md` 核对：

- 已创建并维护迁移记录文档，路径为 `docs/requirement/20260708-internal-layering-record.md`。
- 文档包含三列表格：`原路径`、`新路径`、`备注`。
- 已处理迁移项已记录在表格中。
- 迁移后继续完成了包路径、package 名称、分层后引用关系和 lint 问题修复，使后端重新通过格式化、lint、vet 和测试。
- 用户后续要求消除 `internal/api/http/auth_support.go` 这类仅为编译通过保留、意义有限的代码；本次验证包含该清理结果。

## Spec alignment

不适用。当前任务使用轻量模式 / light，没有独立 `docs/spec/20260708-internal-layering-record.md`。

## Plan alignment

不适用。当前任务使用轻量模式 / light，没有独立 `docs/plan/20260708-internal-layering-record.md`。验证按 requirement / 需求、用户后续指令和实际 diff 核对。

## Actual diff summary

本次 diff 的主体是 `internal/` 分层迁移及迁移后的 Go 包路径修复：

- HTTP 层从 `internal/transport/http` 迁移到 `internal/api/http`，并调整 handler、middleware、codec、response、server 和相关测试引用。
- 应用服务从 `internal/service/*` 迁移到 `internal/application/*`、`internal/auth/*`、`internal/queue/*`、`internal/workflow/activity/*` 等分层位置。
- repository、model、common、infrastructure、gen、bootstrap、worker 等目录按新结构重组。
- 补齐 workflow activity 层独立执行所需的 service、workspace、runner、render、runtime variable helper，避免 activity 反向依赖 application 私有实现。
- 修复迁移后的 import path、package name、import alias、同目录混包和私有 helper 断链问题。
- 清理迁移后无业务意义的空壳代码：
  - 删除 `internal/transport/http/auth_support.go`。
  - 删除 `internal/transport/http/permission.go`。
  - 删除 `internal/transport/http/response/response_test.go`。
- 更新 `.golangci.yml` 为 golangci-lint v2 可校验结构，并保留主流基础 linters：`errcheck`、`govet`、`ineffassign`、`revive`、`staticcheck`、`unused`。
- 新增过程记录文档 `docs/requirement/20260708-internal-layering-record.md`，并在本阶段新增本文档。

`git diff --cached --stat` 当前显示：169 files changed, 1561 insertions(+), 448 deletions(-)。

## Expected vs actual changed files

### Expected

- `internal/**` 下迁移后的 Go 源码、测试、生成代码位置和 import/package 修复。
- `cmd/**` 中入口包引用修复。
- 与后端构建、运行入口相关的配置文件在迁移入口变化后同步调整。
- `docs/requirement/20260708-internal-layering-record.md` 记录迁移过程。
- `docs/verification/20260708-internal-layering-record.md` 记录验证结果。

### Actual

实际 diff 包含上述范围，同时包含以下需要用户审视的相邻改动：

- `.air.api.toml` rename 为 `.air.server.toml`。
- `.env.example`、`Dockerfile`、`Dockerfile.cn`、`README.md`、`configs/config.yaml`、`scripts/run.py` 有修改。
- 新增 `cmd/migrate/main.go`，`cmd/backend-go/main.go` rename 为 `cmd/server/main.go`。
- `sql/migration/mysql/000001_background_task.up.sql` 与 `sql/migration/sqlite/000001_background_task.up.sql` 当前在 diff 中显示修改。

注意：项目约束写明“不修改已执行的迁移文件”。本验证未对迁移 SQL 改动做业务确认；它们出现在当前 diff 中，应在提交前由用户单独确认来源，必要时拆分或回退。

## Acceptance checklist

- [x] `internal` 分层后的 Go import/package 不再触发 Go module 远程解析旧路径。
- [x] 后端格式化命令通过。
- [x] golangci-lint 配置校验通过。
- [x] golangci-lint 运行结果为 `0 issues`。
- [x] `go vet` 通过。
- [x] `go test ./cmd/... ./internal/...` 通过。
- [x] 删除 `internal/api/http/auth_support.go` 这类只为编译存在的低价值空壳代码。
- [x] 已扫描并确认 `internal/**/*.go` 下没有只剩 package 声明的 Go 文件。
- [x] 未执行 `git add`、`git commit`、`git push` 等 Git 写操作。
- [x] 未主动启动、停止或重启开发服务器。

## Command results

验证命令：

```bash
go fmt ./cmd/... ./internal/... && \
./bin/golangci-lint fmt ./cmd/... ./internal/... && \
./bin/golangci-lint config verify && \
./bin/golangci-lint run ./cmd/... ./internal/... && \
go vet ./cmd/... ./internal/... && \
go test ./cmd/... ./internal/...
```

结果：通过。

关键输出：

```text
0 issues.
```

`go test` 覆盖 `cmd/...` 与 `internal/...`，所有有测试的包均通过；无测试文件的包按 Go 输出标记为 `[no test files]`。

## Missed or expanded scope

- Requirement 最初只记录“移动文件，不修改源码内容”；后续用户明确要求继续修复包路径、lint、配置与空壳代码，因此实际工作范围已扩展为“迁移后可编译、可 lint、可测试”。
- 未运行前端检查，因为本轮没有验证前端改动；若当前 diff 中存在前端文件变化，应另行触发前端约束命令。
- 未验证开发服务器运行态，符合“不主动启动、停止或重启开发服务器”的项目约束。
- 当前 diff 中包含迁移 SQL 文件变更，本文档仅提示风险，不确认其业务正确性。

## Risks

- 当前 diff 很大，提交前建议用户重点审视是否需要按“目录迁移 / 包路径修复 / lint 清理 / 运行入口调整 / 迁移 SQL”拆分提交。
- `internal/application/*` 与 `internal/workflow/activity/*` 为避免迁移后私有 helper 断链，存在部分复制后的 helper；当前 lint/test 通过，但后续仍可继续做更精细的抽象边界梳理。
- 迁移 SQL 文件出现在 diff 中，与项目约束存在潜在冲突；提交前必须确认这些改动是否来自用户授权的独立任务。

## Incomplete items

- 没有未通过的后端验证项。
- 未对 SQL migration diff 做业务验收。
- 未对前端执行 `yarn --cwd web lint:fix` 或 `yarn --cwd web typecheck`，因为本次验证范围是后端 `cmd/internal` 分层迁移。

## Conclusion

本次 internal 分层迁移及迁移后的后端包路径、lint、vet、test 验证通过。后端当前达到可格式化、可 lint、可 vet、可测试状态。

提交前仍建议用户单独审视当前 diff 中的相邻配置、脚本和 SQL migration 改动，尤其是已执行迁移文件。