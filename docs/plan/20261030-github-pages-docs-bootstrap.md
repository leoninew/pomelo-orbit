# Plan / 计划：GitHub Pages 文档首期
最后修改时间: 2026-10-30 16:27:15

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

依据已接受的[首期意图](../intent/20261030-github-pages-docs-bootstrap.md)，交付中文最小内容、独立文档构建和 GitHub Pages 发布闭环，按 A01–A06 验收。首期不建设产品内 `/doc`、完整手册、全文搜索或全量双语。

用户已授权实施、整体验证及暂存本次变更，尚未授权创建提交、修改 GitHub 仓库设置或发布。历史[文档中心草案](../intent/20260728-documentation-handbook.md)只供追溯，不作为本期技术或验收依据。

## Technical decisions / 技术设计与决策

### 站点与内容边界

- 在 `docs/` 建立独立 Yarn 项目，固定 `yarn@4.18.1` 和 VitePress `1.6.4`，使用 Node.js 22、`node-modules` 和提交的 `docs/yarn.lock`；与 `web/` 依赖和产品构建分开。首次生成锁文件后，以 `corepack yarn install --immutable` 验证可复现安装。
- 以 `docs/.vitepress/config.mts` 配置 `srcDir: './handbook'`、`base: '/pomelo-orbit/'`、中文语言、最小导航和侧栏；输出至 `docs/.vitepress/dist`。仅扫描 `docs/handbook/`，不依赖侧栏隐藏来排除过程文档。
- 使用默认主题及少量必要样式，不自建主题或前端文档模块。代码和命令展示使用 12px、普通代码块行高 20px；其他文字不使用 12px，阅读正文保持可读层级。桌面及小屏保持导航可操作、代码块可横向滚动。
- 首页说明产品定位、能力边界、文档适用开发版本及导航；`getting-started/development.md` 提供一篇真实开发入门，涵盖前置条件、工作目录、依赖、配置、启动、成功检查和常见问题。文档维护命令放在仓库 README，不增加仅占位的章节。
- 开发入门依据 `go.mod`、`Taskfile.yml`、`.env.example`、`internal/config/runtime.go`、`web/vite.config.ts` 和 HTTP 健康检查核对。`task webapi` 使用 development profile，应说明 `.env.development`、必需的 Fernet 密钥及 `overwrite.env` 的优先级；不得覆盖已有配置或展示真实密钥。前端端口以实际启动输出为准，健康检查只证明 HTTP 服务响应，不代表部署链路正常。
- README 中将 `task web:dev`、`task webapi:dev` 修正为实际的 `task web`、`task webapi`，加入文档站地址与 `docs/` 下的安装、构建命令。引用站点源目录外的仓库文件时使用明确的 GitHub 文件链接，不将其他 Markdown 复制进站点或绕过死链检查。
- VitePress 保持死链检查开启；静态构建负责校验其能识别的站内页面链接，标题锚点、外部链接、图片、站点行为另行核对，不宣称构建覆盖全部链接。首期不添加全文搜索、翻译占位页或多版本机制。

### GitHub Actions 与权限

- 新建 `.github/workflows/docs.yml`：所有 PR 运行构建检查且不部署；`develop` 上站点内容、站点配置、锁文件或该 workflow 的变更触发构建和部署；支持手动触发。PR 不做 paths 限制，避免将来设为必需检查时因工作流跳过而等待；无关业务代码的 push 不触发发布。
- 构建 job 仅有 `contents: read`，使用现有项目的 checkout、Node 22 和 Corepack 模式，在 `docs/` 执行锁定安装及构建。仅在允许发布的 `push` / `workflow_dispatch` 上上传 `docs/.vitepress/dist`；PR 不上传 Pages 部署 artifact。
- 独立 deploy job 依赖成功构建，条件限定为 GitHub `develop` 的 `push` 或 `workflow_dispatch`，不允许 PR 部署；仅该 job 授予 `pages: write`、`id-token: write`，使用 `github-pages` environment 和官方 Pages artifact/deploy actions，并设置 Pages 并发组。
- 目标为现有 GitHub 仓库的 Project Pages，不需要 `gh-pages` 分支或长期 PAT。Pages Source 须由有权限者在仓库设置中选择 GitHub Actions；本地 `develop` 跟踪 Gitee，须将包含变更的分支合入 GitHub `develop` 后才会触发对应工作流。
- `docs/.gitignore` 忽略文档依赖和 VitePress 缓存/产物；`.dockerignore` 排除文档依赖及缓存，避免增大 Docker 构建上下文。不改 `web/` 构建、现有产品发布 workflow 或发布包。

## Implementation steps / 实现步骤

1. 新建 `docs/` 独立依赖、锁文件、VitePress 配置和最小样式；确认公开源仅包含 `handbook/`，且生成路径与项目站点 base 一致。
2. 编写首页和开发入门，逐项对照当前代码、配置及 Taskfile；修正 README 的失效任务名，补充站点和文档维护入口。配置示例不包含真实凭据、不覆盖已有文件。
3. 新增文档 workflow 和忽略规则；按事件、分支、artifact、job 权限和 Pages environment 检查发布边界。
4. 执行本地锁定安装、构建、静态产物与链接检查；记录内容核对及未运行项。不启动或重启项目开发服务器。
5. 仅在获得对应授权且具备权限后，由授权方启用 GitHub Pages、将变更送入 GitHub `develop`，观察 workflow 并检查公开首页和深链接；否则报告外部条件及线上验收未完成，不以本地构建替代。

## Files to change / 需修改文件

- `docs/package.json`、`docs/yarn.lock`、`docs/.yarnrc.yml`、`docs/.gitignore`。
- `docs/.vitepress/config.mts`，以及必要的 `docs/.vitepress/theme/index.ts`、样式文件。
- `docs/handbook/index.md`、`docs/handbook/getting-started/development.md`。
- `.github/workflows/docs.yml`、`.dockerignore`、`README.md`。
- 本意图及计划；用户授权进入整体 Verification 时，再建立同名验证记录。不修改旧文档中心草案、业务代码和既有发布包脚本。

## Tasks and coordination / 子任务与协调

不适用。按已接受意图保持单任务，顺序完成站点、内容、构建和发布配置，不创建 task。

## Verification plan / 验证计划

- A01：检查构建产物页面清单，仅包含公开源生成的页面和 VitePress 自带页面；检查产物不包含过程与归档页面。记录站点隔离不改变 GitHub 仓库可见性。
- A02、A06：人工对照上述代码与配置核对开发指南、README 命令和版本状态；确认未把未经验证的镜像、Release、登录或安装路径写成可用事实。记录未实际执行的运行步骤。
- A03：在 `docs/` 运行 `corepack yarn install --immutable` 和 `corepack yarn build`，确认构建失败及可识别死链会返回非零；检查首页、文章 HTML、资源路径和站内导航目标。实际采用的外部链接和锚点另行核对。
- A04：静态核查 workflow 的 PR、push、手动触发条件与最小权限；有 GitHub 执行条件时分别记录 PR 构建和 `develop` 部署结果，否则标为未验证，不声称 Actions 已通过。
- A05：实际部署后访问 `https://leoninew.github.io/pomelo-orbit/` 和生成的开发文章深链接，检查刷新、导航、样式及资源；在桌面和窄屏核对阅读、导航和代码溢出，确认字号约束。若未发布，记录为未完成。
- 运行 `git diff --check` 并检查新增文件；本期不改 Go、`web/` 或其依赖，不以 `task check` 或 Go/前端业务测试替代文档构建检查。若实施范围发生变化，按项目 `AGENTS.md` 补充对应检查。
- 上述为实现期间必要检查；整体 Verification 须在用户授权后对照 A01–A06 形成同名验证记录。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

- GitHub Pages 尚需启用，仓库权限、环境保护或网络条件可能阻止真实发布；PR 构建与线上访问须有实际证据，不能从配置推断成功。
- 子路径、VitePress 1.6.4 的链接行为和 GitHub Pages 深链接须按最终产物及实际站点验证；默认主题不能假设自动满足项目 12px 约定。
- `develop` 文档可能先于正式版本，必须标注开发状态；真实下载及默认账号不能根据历史文档或测试数据推测。
- 验证开发文章不要求启动、停止或重启开发服务器；不能实际运行的步骤只记录静态核对结果，不称为已运行通过。

## Rollback / recovery / 回滚 / 恢复

- 无数据库或业务运行时变更。尚未发布时，可按后续授权调整或移除本期新增站点、workflow 与入口，保留既有工程文档和用户改动。
- 已发布后，修复并通过正常审核合入 GitHub `develop`，触发新构建替换站点；如需停用 Actions 或 Pages，须由有权限者按明确授权操作。GitHub Pages 设置与已发布内容不会因本地撤销自动恢复。

## Implementation notes / 实施记录

- 已增加独立 VitePress 项目与锁文件、中文首页和开发入门、最小主题样式、GitHub Actions 工作流及忽略规则；README 已修正开发任务名并说明文档构建与发布前提。产品业务、既有发布包和旧文档草案未改动。
- 在 `docs/` 首次生成锁文件后，`corepack yarn install --immutable` 和 `corepack yarn build` 均成功。安装提示 VitePress 间接依赖的 Algolia peer requirements 警告；首期未启用搜索，未为消除警告引入无关依赖。
- 使用临时坏链接确认构建以退出码 1 失败；移除临时文件后重新构建成功。静态产物仅有首页、开发入门和 404 页面；一次性检查确认 39 个本地资源、页面引用及锚点在 `/pomelo-orbit/` 下可解析。
- 使用现有 PyYAML 解析 workflow 并核查事件、分支、artifact 与 job 权限；本机未安装 `actionlint`。`git diff --check` 通过，新增文本文件另行检查无行尾空白。
- 未执行 GitHub PR、Pages 部署、公开站点及桌面/窄屏浏览器检查；仓库 Pages 设置和 GitHub 发布仍需权限与单独授权。未启动开发服务器，也未执行 Go 或 `web/` 业务测试。开发指南中的运行步骤仅与代码、配置和任务命令静态核对。
- 实施阶段只交付本地实现与检查结果，不宣称 A04 的实际 Actions 执行或 A05 的线上验收完成；后续整体验证见[验证记录](../verification/20261030-github-pages-docs-bootstrap.md)。

## User review notes / 用户审查记录

- 2026-10-30：用户要求开始计划；当前提交计划供审查，尚未实施或执行线上发布。
- 2026-10-30：用户要求开始兑现，接受计划并授权实施；未授权 Git 写入、修改 GitHub 仓库设置或执行线上发布。
- 2026-10-30：用户确认工作有效，要求形成验证文档并暂存变更；未授权创建提交、修改 GitHub 仓库设置或发布。
