# Gitea 仓库支持
最后修改时间: 2026-09-06 09:44:06

Review status: Accepted

流程模式: 轻量 / light

## Background

流水线 `git clone` 阶段本身不识别 Git 托管平台，只对 HTTPS 仓库 URL 做 shallow fetch。平台差异集中在 Git 凭据类型和执行时注入。当前可执行的 Git 凭据只有 `github_token`（token 作为 URL username）和 `gitee_token`（`username:token`）。`git_ssh` 可保存但执行时被拒绝。代码、界面和活文档均没有 Gitea。

Gitea 私有库 HTTPS clone 的官方用法是 Personal Access Token：用户设置 → Applications → Manage Access Tokens → Generate New Token。Git 认证为 `https://<username>:<token>@<host>/<owner>/<repo>.git`，与现有 `gitee_token` 注入格式一致，而不是 `github_token`。

## Goal

- 新增一等 Git 凭据类型 `gitea_token`，数据格式为 `username:token`；`username` 是 Gitea 登录用户名，不是邮箱或组织名。
- 远程仓库可以绑定 `gitea_token`；流水线 git 阶段执行时把仓库 HTTPS URL 改写为带用户名和 token 的认证 URL。
- 凭据创建、编辑、导入和仓库凭据选择均展示并接受 `gitea_token`。
- 活文档说明 Gitea 使用该凭据类型，以及 token 来源是用户 Access Token，不是 OAuth Application。

## Non-goal

- 不改 `git clone` 阶段脚本、镜像或制品收集。
- 不实现 `git_ssh` / Deploy Key，也不为 Gitea 单独开通 SSH。
- 不支持非 HTTPS 仓库 URL（`http://`、SSH URL）。
- 不接入 Gitea Webhook、OAuth2 Application（Client ID / Secret）或 Gitea API 列举分支。
- 不把已有 `gitee_token` / `github_token` 迁移或别名为 Gitea；不为错误类型提供兼容层。
- 不校验 Gitea host、token 权限范围或仓库是否真实存在。

## User scenarios

- 用户在 Gitea「Manage Access Tokens」生成 token，在 Orbit 创建 `gitea_token` 凭据，内容为 `登录用户名:token`。
- 用户创建或编辑远程仓库，填写 `https://<gitea-host>/<owner>/<repo>.git`，并选择该 `gitea_token`。
- 用户运行绑定该仓库的 Application Pipeline；git 阶段用注入后的认证 URL 拉取代码。
- 公开 Gitea 仓库仍可不绑定凭据，行为与现有公开 HTTPS 仓库一致。

## Acceptance

- 凭据类型允许 `gitea_token`；非法类型仍拒绝。
- `gitea_token` 明文必须是非空 `username:token`，否则执行前失败。
- 流水线执行将 `https://host/owner/repo.git` 改写为 `https://username:token@host/owner/repo.git`，并沿用现有 `GIT_CONFIG ... insteadOf` 注入，不把 token 写入 stage 脚本。
- 凭据页类型列表、占位符和仓库远程凭据下拉包含 `gitea_token`，不包含 OAuth Application 字段。
- 后端相关测试覆盖类型校验与 URL 改写；`task check`、`go test ./cmd/... ./internal/...`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck` 通过。
- `docs/guides/ci-pipeline-design.md` 写明 Gitea 凭据类型与 token 来源。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 新增独立类型 `gitea_token`，注入路径对齐 `gitee_token`，不对齐 `github_token`。
- token 来源是用户 Access Token，不是同页 OAuth2 Application，也不是仓库 Deploy Key。
- git clone 阶段保持通用 HTTPS Git，不增加平台探测。
- `gitea_token` 的 username 是 Gitea 登录用户名，不是邮箱或组织名。
- 本期只开放 HTTPS 仓库 URL；非标准端口或路径前缀只要仍是 `https://`，沿用现有 URL 解析，不加额外规则。

## User review notes

- 2026-09-06：用户确认 username 为 Gitea 登录用户名；目前只开放 HTTPS。

## Risk

- 用户若把 token 填成 OAuth Client Secret，或只填 token 不填 `username:`，clone 会认证失败；产品只做格式切分，不探测 Gitea。
- token 权限不足（未授予 `read:repository`）时，失败发生在 git 阶段容器内，而不是凭据保存时。
- 误用 `github_token` 绑定 Gitea 仓库仍可能失败；本期不阻止这种错误绑定。
