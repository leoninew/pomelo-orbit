# Verification / 验证：GitHub Pages 文档首期
最后修改时间: 2026-10-30 16:32:41

## Review status / 审查状态

Draft

## Intent alignment / 意图对齐

对照[首期意图](../intent/20261030-github-pages-docs-bootstrap.md)，已提供独立中文文档站、开发入门、本地构建与 Pages workflow；用户确认工作有效。发布后的公开访问仍未满足 A05，不能把本地预览或 workflow 配置视为线上验收通过。

## Plan alignment / 计划对齐

对照[计划](../plan/20261030-github-pages-docs-bootstrap.md)，站点使用 VitePress 1.6.4、Yarn 4 和独立的 `docs/handbook/` 内容源；配置项目站点子路径、最小主题样式和 PR/部署权限边界。实际 GitHub Actions、Pages 设置及线上浏览器检查尚未完成。

用户随后要求在 `docs/package.json` 增加 `preview` 入口，已增加 `vitepress preview .`；这是计划外但用户明确要求的小范围补充，不改变产品服务。

## Task deliveries and integration / 子任务交付与集成

不适用；按计划保持单任务，无子任务文档。

## Actual diff summary / 实际差异摘要

- 增加站点依赖与锁文件、配置、最小样式、首页和开发入门；增加文档 PR 检查与 Pages 部署 workflow。
- README 修正开发启动任务并增加文档入口；更新文档依赖、缓存与 Docker 构建上下文的忽略规则。
- 本地预览入口由用户追加；本次新增验证记录，不修改产品业务代码、现有产品发布脚本或历史文档草案。

## Expected vs actual changed files / 预期与实际修改文件对比

站点文件、`docs/package.json`、`docs/yarn.lock`、`.github/workflows/docs.yml`、`.dockerignore`、`README.md` 及意图、计划均在计划范围。`docs/package.json` 增加的 `preview` 脚本为用户明确追加；本验证文档按阶段授权新增。无其他有意扩展。

## Acceptance criteria checklist / 验收标准检查清单

- A01：通过。构建后仅有首页、开发入门和 404 三个 HTML 页面；过程与归档页面不在产物中。公开仓库的原文件仍可访问。
- A02：通过静态核对。中文首页、开发入门及 README 入口已提供；任务名、配置来源和健康检查与当前代码核对。用户确认工作有效，但未提供逐项浏览或运行记录，开发指南中的服务启动步骤未在本轮执行。
- A03：通过。更新后的 Yarn 4 锁文件以 `corepack yarn install --immutable` 安装成功，`corepack yarn build` 成功；临时坏链接使构建以退出码 1 失败，移除后重新构建成功。安装有间接依赖 peer 警告。
- A04：部分验证。workflow 的事件、发布分支、artifact 和权限已静态核查；尚无实际 GitHub PR 构建或 `develop` 部署运行证据。
- A05：未满足。产物中 39 个本地资源、页面引用及锚点按 `/pomelo-orbit/` 路径解析成功；公开首页和开发文章在本轮访问均返回 HTTP 404。线上深链接刷新、样式、导航和桌面/窄屏检查未完成。
- A06：通过静态核对。首页说明文档对应 `develop` 开发状态；未将未经验证的公开镜像、Release 或安装步骤写成已验证事实。

## Test / command results / 测试 / 命令结果

- `docs/`：`corepack yarn install --immutable` 和 `corepack yarn build` 通过。Yarn 报告 VitePress 间接依赖的 Algolia peer requirements 警告；首期未启用搜索。
- `docs/`：临时添加坏链接后构建按预期以退出码 1 失败；删除本次临时文件后，干净构建通过。
- 仓库根目录：一次性静态产物检查通过，确认三个 HTML 页面与 39 个本地引用及锚点；PyYAML 静态检查 workflow 的事件、分支、artifact 和 deploy job 权限通过。未安装或运行 `actionlint`。
- 仓库根目录：Node 检查 `build` 与 `preview` 脚本配置通过。用户确认工作有效；本轮没有启动开发或预览服务器。
- 仓库根目录：`git diff --cached --check` 未通过。用户更新并暂存的 Yarn 4 格式 `docs/yarn.lock` 有 1983 行 CRLF，Git 将回车报告为行尾空白；排除该文件后的暂存区检查通过。本轮未改写该锁文件。
- 本轮只读访问预期 Pages 首页与开发文章，均返回 HTTP 404；未运行 GitHub Actions，未变更 GitHub Pages 设置，未推送。未运行 Go 或 `web/` 业务测试，本期没有修改相应代码或依赖。

## Missed or expanded scope / 遗漏或扩大的范围

`preview` 入口为用户明确追加。线上 PR 检查、部署与站点访问仍待完成，不能视为已经实现并验证的发布闭环。

## Risks / 风险

- 当前暂存的锁文件使用 CRLF，与仓库 `.editorconfig` 的 LF 约定不一致，并使暂存区空白检查失败；保留用户更新，待明确处理。
- GitHub Pages 尚未获得成功发布证据；仓库设置、环境保护和 GitHub `develop` 的推送路径均可能影响部署。
- 本地构建及用户的有效性反馈不能证明线上站点、桌面/窄屏布局、完整开发操作或业务部署均已验证。

## Incomplete items / 未完成事项

- 处理锁文件行尾并重新检查暂存区；不擅自覆盖用户已暂存的锁文件。
- 在具备权限并获得相应授权后配置 GitHub Pages、将变更送入 GitHub `develop`，验证 PR 检查、部署结果及首页和文章深链接。
- 线上发布后检查页面刷新、样式、导航、资源和桌面/窄屏展示，再补充实际证据；本报告当前不证明 A04、A05 已通过。

## Conclusion / 结论

本地文档站可构建，用户确认工作有效；但暂存区空白检查存在锁文件行尾问题，且公开站点返回 404，首期线上发布验收尚未完成。验证记录已形成，整体验收不能判为全部通过。

## Next steps / 下一步建议

1. 优先收尾首期，不立即增加文档栏目：保留用户确认的 Yarn 配置和锁文件，另行明确行尾处理方式；获得对应授权后，启用 GitHub Actions 作为 Pages Source，将变更送入 GitHub `develop`。本地分支跟踪 Gitee，不能以 Gitee 推送代替 GitHub 发布。
2. 补齐 A04、A05 的实际证据：记录 PR 构建和 `develop` 部署的运行链接，验证首页、文章深链接直接访问及刷新、资源与导航，并完成桌面和窄屏检查。通过后更新本报告并交付审查；未完成前不将首期整体标为通过或归档。
3. 首期闭环后，单独发起第二期意图：先核实镜像或发布包的实际可用性，再选择一条可复现的安装主路径，覆盖前置条件、数据与密钥持久化、首次访问、健康检查和常见故障；以干净环境中的实际操作记录为验收依据，不一次扩展全部平台、双语或 CI/CD 手册。本轮只策划，不开始第二期实现。
