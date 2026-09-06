# Gitea 仓库支持验证
最后修改时间: 2026-09-06 09:53:02

Review status: Accepted

## Requirement alignment

- 新增独立凭据类型 `gitea_token`，创建与导入走同一校验；明文约定为 Gitea 登录用户名:`token`。
- 流水线执行把 HTTPS 仓库 URL 改写为 `https://username:token@host/...`，注入路径与 `gitee_token` 相同，不对齐 `github_token`。
- 凭据页类型列表、占位符和仓库远程凭据下拉包含 `gitea_token`。
- 活文档 `docs/guides/ci-pipeline-design.md` 写明类型、token 来源（Access Token，非 OAuth Application）和 HTTPS 限制。
- 未改 git clone 阶段脚本、镜像或制品；未实现 SSH、Webhook 或类型别名。

## Spec alignment

不适用。该任务采用轻量模式 / light，未创建独立 Spec / 规格文档。

## Plan alignment

不适用。按 requirement / 需求核对。

## Actual diff summary

- 后端：`validCredentialType` 允许 `gitea_token`；执行器将 `gitea_token` 与 `gitee_token` 共用 `username:token` 解析。
- 测试：凭据创建/导出接受 `gitea_token` 并拒绝未知类型；URL 改写覆盖带端口的 Gitea HTTPS 与缺少 `username:` 的失败。
- 前端：凭据创建/导入类型、详情占位符、仓库列表与详情的 Git 凭据过滤加入 `gitea_token`。
- 文档：CI Pipeline 活指南增加 Git 凭据说明；本 feature 的 requirement 过程文档。

## Expected vs actual changed files

预期范围与暂存区一致：

- `internal/application/credential/usecase/service.go`
- `internal/application/credential/usecase/service_integration_test.go`
- `internal/application/pipeline_run/usecase/executor.go`
- `internal/application/pipeline_run/usecase/executor_test.go`
- `web/src/views/credential/CredentialPage.vue`
- `web/src/views/credential/CredentialDetail.vue`
- `web/src/views/repository/RepositoryPage.vue`
- `web/src/views/repository/RepositoryDetail.vue`
- `docs/guides/ci-pipeline-design.md`
- `docs/requirement/20260906-gitea-repository-support.md`

工作区另有未暂存的 `docs/requirement/20260906-mcp-standard-authentication.md` 与 `docs/requirement/20260906-pipeline-stage-variable-defaults.md`，不属于本 feature。

## Acceptance checklist

- [x] 凭据类型允许 `gitea_token`；非法类型仍拒绝。
- [x] `gitea_token` 缺少 `username:token` 时执行前失败。
- [x] HTTPS URL（含非标准端口）改写为 `https://username:token@host/...`；注入仍走既有 `GIT_CONFIG ... insteadOf`。
- [x] 凭据页类型列表、占位符和仓库远程凭据下拉包含 `gitea_token`，无 OAuth Application 字段。
- [x] 相关测试、`task check`、`go test ./cmd/... ./internal/...`、前端 lint/typecheck 通过。
- [x] `docs/guides/ci-pipeline-design.md` 写明 Gitea 凭据类型与 token 来源。

## Test results

| Command or check | Result |
| --- | --- |
| `go test ./internal/application/credential/usecase ./internal/application/pipeline_run/usecase` | Passed |
| `go test ./cmd/... ./internal/...` | Passed |
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `task check` | Passed（frontend typecheck/lint/format，golangci-lint 0 issues） |
| `git diff --check --cached` | Passed |

## Missed or expanded scope

无超出需求的产品行为。`splitGiteeCredential` 重命名为 `splitUsernameTokenCredential` 仅为 `gitee_token` / `gitea_token` 共用解析，不改变 Gitee 语义。

## Risks

- 保存凭据时不校验 Gitea host、token 权限或仓库是否存在；权限不足或填成 OAuth Client Secret 会在 git 阶段失败。
- 误把 `github_token` 绑到 Gitea 仓库本期不阻止。
- 未对真实 Gitea 实例跑 clone 阶段。

## Incomplete items

- 未在浏览器中点击凭据创建和仓库绑定表单。
- 未用真实 Gitea HTTPS 仓库跑一次 Application Pipeline。

## Conclusion

本 feature 的暂存实现与已接受需求一致，代码验收通过。浏览器与真实 Gitea clone 未覆盖，不阻断本阶段结论。
