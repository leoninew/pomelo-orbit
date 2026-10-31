# Pomelo Orbit 使用手册

Pomelo Orbit 是自托管 CI/CD 平台，用于管理流水线、应用部署和 Traefik 网关。它支持在本机和 SSH 环境中执行 Docker Compose 部署。

本手册帮助你使用平台完成源码构建、应用部署、网络接入和日常维护。Pomelo Orbit 面向 Compose 工作流，不以 Kubernetes 或多集群编排为主要使用路径；阅读手册不要求先搭建平台的源码开发环境。

本手册的终端命令和手写脚本统一使用 Bash。执行前确认所在主机、工作目录和目标平台；从本地 Bash 发起命令不代表远端系统也使用 Bash，平台自动生成的初始化命令仍须遵守页面标明的解释器和权限要求。

## 选择阅读路线

| 你想做什么 | 推荐入口 |
| --- | --- |
| 先了解产品适用范围、工作区和交付流程 | [项目概览](./getting-started/overview.md) |
| 在隔离主机准备实例、完成首登和改密 | [自托管安装](./getting-started/installation.md) |
| 用单组件 HTTP 服务走通首次部署与访问检查 | [首个应用部署](./getting-started/first-deployment.md) |
| 理解项目、环境、流水线、应用和服务的关系 | [核心概念](./concepts/core-model.md) |
| 在已有实例中配置本机或 SSH 执行目标并检查环境 | [项目与执行环境](./guides/projects-environments.md) |
| 接入源码并选择受支持的认证方式 | [仓库与仓库凭据](./guides/repositories-credentials.md) |
| 复用构建模板，检查运行、制品及生成版本 | [流水线与制品](./guides/pipelines-artifacts.md) |
| 管理组件、应用版本、服务配置和部署结果 | [应用、版本与服务](./guides/applications-services.md) |
| 准备网络入口，区分 DNS、组件入口与路由同步 | [网关、DNS 与路由](./guides/gateway-dns-routes.md) |
| 管理账号、系统角色、项目成员与个人令牌 | [用户、权限与访问令牌](./guides/users-access.md) |
| 理解应用内部署对话的前提、共享历史和执行风险 | [部署对话](./guides/deployment-dialogue.md) |
| 查明实例配置来源、系统设置生效及密钥变更风险 | [配置与密钥](./operations/configuration.md) |
| 识别平台与业务数据、安排备份恢复和升级 | [数据保留与升级](./operations/data-upgrades.md) |
| 从登录、环境、构建、部署或访问症状定位问题 | [故障定位](./operations/troubleshooting.md) |
| 从本地代理或命令行接入已有能力，理解认证与执行结果 | [MCP 与 CLI](./reference/mcp-cli.md) |
| 查明变量取值、服务覆盖及生成 Compose 的限制 | [变量与 Compose 约束](./reference/variables-compose.md) |

## 项目资料

- [GitHub 仓库](https://github.com/leoninew/pomelo-orbit)：提交使用问题；分享日志或配置前先脱敏。
- [工程文档（仓库）](https://github.com/leoninew/pomelo-orbit/blob/develop/docs/README.md)：仅供参与开发的读者，不属于本手册的使用步骤。

## 文档状态

本手册适用于 GitHub `develop` 分支，不保证与某个已发布版本完全一致。安装篇提供隔离试用的准备、操作和检查路径，日常指南以已有可用实例及相应身份 / 权限为前提。文中示例不应直接作为生产配置；按你的目标、镜像、权限和数据条件确认后再操作。
