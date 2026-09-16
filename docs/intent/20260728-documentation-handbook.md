# Pomelo Orbit 双语 Markdown 文档与前端文档中心需求
最后修改时间: 2026-07-28 08:43:10

Review status: Draft

## Background

Pomelo Orbit 是面向本地 Docker 环境的自托管 CI/CD 与平台管理控制面。仓库根目录 `README.md` 已介绍产品能力并给出极简开发启动方式；`docs/` 下已有产品、架构、运维和前端文档，但它们的目标是活文档 SoT、工程实施和零散 How-to。

目前读者无法从一个面向使用场景的入口，顺序完成以下路径：理解产品、完成安装、配置系统、建立 CI、部署首个应用、配置网关与路由，以及参与项目开发。已有 `docs/README.md` 与 `docs/INDEX.md` 的职责是工程文档入口和主题索引，应继续保持该定位。

本需求拟增加一套读者导向的文档专题（handbook），以章节导航串起当前代码所支持的能力，并对引用的操作步骤重新核验，避免将历史模型或过时配置写成现行行为。

专题同时交付两层内容：仓库内可审阅和维护的双语 Markdown 原文，以及产品前端内大型 `/doc` 文档中心。`/doc` 不是 Markdown 文件列表的替代品，而是这些内容的可浏览、可分享入口。现有 Vue Router 没有该路由，且默认将非公开路由跳转至登录页；前端还没有可复用的文档布局、章节导航、文章目录或 Markdown 内容模型。

## Goal

1. 以成对的中文、英文 Markdown 文件作为专题章节的可审阅原文，并在 `/doc` 中渲染相同内容。
2. 在产品前端提供一个可直接访问的 `/doc` 文档中心，并为每个章节提供稳定、可分享的子路由。
3. 建立适合持续扩展的三栏文档体验：章节侧栏、文章正文与当前文章目录；小屏幕应转换为不遮挡正文的可操作导航。
4. 在文档中心提供中文/English 切换；同一章节切换语言后仍停留在相应内容，缺少译文时有明确的回退提示。
5. 覆盖三条完整主线：开发贡献、管理员安装运维、平台使用者操作。
6. 每篇操作文档给出前置条件、可执行步骤、验证方式、常见失败处理和到相关概念/参考文档的链接。
7. 清楚区分“当前支持的能力”“尚未作为主路径支持的能力”和历史归档内容；与代码冲突时以代码为准。
8. 文档内容与现有 `docs/product/`、`docs/architecture/`、`docs/guides/` 的 SoT 分工明确，避免复制后发生漂移。

## Non-goal

1. 本轮不改变 CI/CD、部署、认证或网关的产品行为。
2. 不把 `docs/archive/` 的历史材料重新作为现行操作依据。
3. 不引入 Docusaurus、VitePress、MkDocs 或另一套独立构建/托管系统；文档中心随现有 Vue 应用构建和发布。
4. 首版不实现服务端全文检索、在线编辑、版本切换、评论或文档 CMS。
5. 文档区域不使用 `vue-i18n` 或现有应用 i18n 词条作为内容翻译机制；语言与章节映射独立于应用业务界面。
6. 不为尚未由代码和测试证实的功能编造 API 参考、集群/Kubernetes 教程或外部 Traefik 主路径。
7. 不以“大而全”为目标重复所有内部实现细节；深入设计应链接到既有活 SoT。

## User scenarios

### 新用户

作为评估者，我可以在未登录的情况下访问 `/doc`，从“产品与能力”开始，了解 Pomelo Orbit 是单节点、本机 Docker Compose 导向的 CI/CD 控制面，以及它不提供多集群/Kubernetes 编排主路径。

### 平台管理员

作为管理员，我可以按“安装与运维”完成 Docker 前置检查、持久化目录与 Docker socket 挂载、必需配置、启动验证、日志查看、升级和备份边界，并知道何时需要配置 Gateway。

### 平台使用者

作为使用者，我可以按“使用平台”依次完成登录、项目与权限准备、CI 仓库/阶段/模板配置和一次流水线运行，随后创建 Application、Version、Environment、Service 并部署，最后理解 Route、Expose、Gateway 和证书的适用边界。

### 贡献者

作为开发者，我可以按“开发与贡献”安装 Go、Node/Yarn、Task、代码生成依赖，使用项目定义的命令启动本地环境，配置 `.env`，运行格式化、静态检查、测试、生成代码和镜像构建。

### 文档维护者

作为维护者，我可以在仓库中审阅每一篇中英文 Markdown 原文；改动产品行为时同步更新相应译文，并通过导航检查发现断链或语言缺失。

### 双语读者

作为中文或英文读者，我可以在文档顶栏切换语言，继续阅读当前章节；语言选择应对后续文档页面保持一致，而不受应用工作台语言设置影响。

## Proposed information architecture

`/doc` 是专题首页，呈现产品定位、建议阅读路径和所有章节分组。文章通过稳定子路由访问，例如 `/doc/getting-started/overview`；每个文章 slug 对应一对中文、英文 Markdown 原文。具体目录、元数据格式和前端加载方式在方案阶段确定。

桌面端页面结构：

```text
全宽文档顶栏：品牌 | 文档中心 | 中文 / English | 返回产品 / 登录入口
------------------------------------------------------
章节侧栏             文章正文                       本页目录
开始使用             标题、摘要、更新时间           标题锚点
安装与运维           前置条件、步骤、代码示例       当前章节高亮
使用平台             提示、风险与关联文章
开发与贡献
参考
```

小屏幕：章节侧栏变为可开关抽屉或菜单，本页目录变为文章内的紧凑控件；正文保持可读行宽，代码块可横向滚动且不破坏页面布局。

专题首页应按“从认识到落地”的阅读顺序展示以下导航。中文、英文分别维护同结构 Markdown；下列各项是首版必须交付的章节，而不是可选占位项：

```text
Documentation / 文档中心

├─ Getting Started / 开始使用
│  ├─ Overview / 产品简介与边界
│  ├─ Core Concepts / 核心概念
│  └─ Quick Start / 快速体验与下一步
│
├─ Installation & Operations / 安装与运维
│  ├─ Prerequisites / 安装前检查
│  ├─ Docker Deployment / Docker 部署
│  ├─ Configuration / 配置参考
│  ├─ Verification & First Login / 启动验证与初始登录
│  └─ Upgrade, Backup & Troubleshooting / 升级、备份与排障
│
├─ Using Pomelo Orbit / 使用平台
│  ├─ Projects, Users & Roles / 项目、用户与角色
│  ├─ CI Overview / CI 工作流
│  ├─ Repositories, Stages & Templates / 仓库、阶段与模板
│  ├─ Pipeline Variables & Runs / 变量与运行记录
│  ├─ CD Overview / CD 工作流
│  ├─ Applications, Versions & Environments / 应用、版本与环境
│  ├─ Services & Deployments / 服务与部署
│  └─ Gateway, Expose, Routes & TLS / 网关、暴露、路由与证书
│
├─ Development & Contribution / 开发与贡献
│  ├─ Local Development Setup / 本地开发环境
│  ├─ Configuration & Runtime / 配置与运行
│  ├─ Architecture Tour / 架构导览
│  ├─ Testing, Checks & Code Generation / 测试、检查与代码生成
│  ├─ Building Images / 镜像构建
│  └─ Documentation Maintenance / 文档维护
│
└─ Reference / 参考
   ├─ Configuration Reference / 配置项
   ├─ Command Reference / 命令速查
   ├─ Architecture & Domain Index / 架构与领域索引
   └─ FAQ / 常见问题
```

## Acceptance

1. `/doc` 是公开可访问的前端路由；未登录访问不会被重定向到 `/login`，也不加载需要认证信息的工作台布局或接口。
2. 文档中心有独立的、全宽的文档外壳，包含品牌标识、返回产品或登录的明确入口，并与现有应用的明暗主题一致。
3. 桌面端有固定可扫描的章节侧栏、居中可读的正文和当前文章目录；当前文章与当前标题在导航中有清晰状态。侧栏中的每个文章均有稳定 URL，可刷新、前进和后退。
4. 小屏幕下，章节导航可显式打开和关闭，正文、长链接和代码块不会横向挤压或遮挡其他内容。
5. 顶栏有独立于应用 i18n 的中文/English 切换控件，语言选择在刷新和文档路由之间保持；切换同一章节时保留文章 slug 和阅读位置可行时应保留。
6. 每个首版文章均有中文、英文两份 Markdown 原文；页面显示的标题、导航、正文、本页目录与代码说明与所选原文一致。
7. 首页可在不依赖搜索的情况下导航到开始使用、安装运维、使用平台、开发贡献和参考五类内容；每个分组显示所选语言的章节名称和简短用途。
8. 首版至少交付上述信息架构中所有一级分组及其列出的核心章节，中文和英文正文均不是占位文字。
9. “开发”主线包含当前实际的依赖安装、`task` 常用命令、前后端检查、测试、生成与构建入口，并说明命令运行位置和前置条件。
10. “安装与运维”主线只使用当前代码/镜像支持的配置和路径，明确 Docker socket 的高权限风险、数据持久化位置、健康检查和升级前备份建议。
11. “使用”主线准确反映现行领域模型：CI 的 Stage/Template/Snapshot/Run；CD 的 Application/Version/Environment/Service/Deployment，以及 Gateway、Expose 与 Route 的差异。
12. 每个首次操作类章节至少包含前置条件、步骤、成功验证、失败或风险提示；命令示例不包含真实密钥或不可逆删除操作。
13. 专题不覆盖或篡改 `docs/README.md`、`docs/INDEX.md` 的工程 SoT 导航职责；对既有活文档采用链接或明确引用，而非无来源复制。

## Current evidence and constraints

| 事实 | 当前依据 | 对专题的影响 |
| --- | --- | --- |
| 后端为 Go 1.25，前端为 Vue 3/Vite，统一开发任务由 Taskfile 提供 | `go.mod`、`web/package.json`、`Taskfile.yml` | 开发章节必须按项目实际命令组织，不能假定 npm 或通用 Makefile 流程 |
| 生产镜像将 Web 静态文件与 API/worker 打包为同一进程镜像，默认执行 `serve` | `Dockerfile`、`cmd/server` | 安装章节应说明单镜像和 Docker socket 的运行边界 |
| 配置从 `configs/config.yaml`、`.env` 与系统环境变量合并，JWT 密钥为启动必需项 | `.env.example`、`internal/config` | 配置参考必须给出优先级、必需项和安全要求 |
| 当前活 SoT 已明确 CD 的 Version/Component/Expose、Service/Deployment 与 Gateway 模型 | `docs/product/`、`docs/architecture/` | 使用章节应以它们为事实来源，并保留链接 |
| 现有运维指南中存在需重新核验的陈述，例如 `backend/.env.example` 路径和旧部署叙述 | `docs/guides/docker-deployment.md` | 不能直接复制现有指南；实施前必须与代码和实际容器入口校验 |
| 前端使用 Vue Router，所有非 `meta.public` 路由默认要求登录 | `web/src/router/index.ts` | `/doc` 若服务安装前读者，需显式声明为公开路由 |
| 应用布局只对登录页例外，其余路由套用顶栏、工作台侧栏和认证状态 | `web/src/App.vue` | 文档中心需要独立的 public layout 分支，不能复用数据工作台外壳 |
| 仓库没有现成 Markdown 渲染或文档站点框架 | 根目录、`web/package.json` | 需为双语 Markdown 原文建立可由 Vue 构建期加载和安全渲染的内容模型；不额外引入第二个站点 |

## Open questions

1. `/doc` 是否确认对未登录用户公开？推荐公开，因为“安装”章节正是在用户获得系统账号之前需要阅读的内容；公开页不应显示任何项目、用户或运行数据。
2. 文档的首要读者是团队内部、开源社区，还是两者兼顾？这会影响账号、镜像地址、支持渠道和示例的脱敏程度。
3. 安装主路径是否以 GitHub Container Registry 的公开镜像为准，还是需要同时支持私有镜像仓库/离线交付？

## Decisions

| 决策 | 说明 |
| --- | --- |
| 使用严格模式 / strict | 专题横跨读者导航、安装运维、产品操作和开发流程，且现有资料包含需校准内容，需要经过规格和计划审查 |
| 交付形态为 Markdown + 前端文档中心 | 用户明确要求在提供 Markdown 文档的同时，交付现有 Vue SPA 内的 `/doc` 页面与章节 |
| 双语原文 | 每篇首版文章有中文、英文 Markdown 原文，`/doc` 显示用户选择的版本 |
| 文档语言机制独立 | 不使用 `vue-i18n`；文档语言状态、导航标签与内容映射由文档模块自行维护 |
| 保留既有活 SoT 的职责 | 专题作为读者体验层，产品/架构/决策文档仍是工程事实来源 |
| 文档事实以代码为准 | 与 `CLAUDE.md` 和现有文档治理规则一致 |

## Risk

1. Docker socket 等于对 Docker daemon 的高权限访问；安装步骤必须显著说明其安全边界。
2. 默认账号、镜像发布地址、初始化数据和启动命令若未通过代码/镜像验证，可能误导首次部署；实施阶段需逐项核验。
3. 现有部署指南混有旧模型或错误路径；若不重写或标记替代关系，新专题可能产生相互矛盾的操作路径。
4. 文档路由被错误设为受保护，或沿用工作台布局，会使安装前读者无法访问或造成未认证请求失败；实现必须覆盖匿名访问路径。
5. 章节大量增加后，如果内容与路由/侧栏声明分散维护，容易产生死链和导航漂移；方案需要定义单一内容与导航来源。
6. 中英文 Markdown 内容失配会导致切换后章节缺失或事实不一致；内容索引和验证必须检查每个 slug 的双语配对。

## User review notes

- 2026-07-28：用户希望为项目建立类似开源项目大导航的文档专题，重点描述开发、安装和使用。
- 2026-07-28：用户明确期望交付为前端内的大型 `/doc` 页面及章节，而非仅仓库 Markdown 导航。
- 2026-07-28：用户要求同时交付 Markdown 原文与 `/doc`；正文支持中文、英文切换，但不使用应用既有 i18n 机制。
