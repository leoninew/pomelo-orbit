# Pomelo Orbit 使用手册

管理源码构建、Docker Compose 部署和 Traefik 网络入口。执行目标支持本机和 SSH 主机。

## 从这里开始

| 目标 | 入口 |
| --- | --- |
| 了解平台能做什么 | [项目概览](./getting-started/overview.md) |
| 准备一个新实例 | [自托管运行](./getting-started/installation.md) |
| 通过对话部署第一个服务 | [部署首个应用](./getting-started/first-deployment.md) |
| 区分应用、版本、组件与服务 | [资源关系](./concepts/core-model.md) |
| 修改平台源码或自行构建镜像 | [本地开发](./getting-started/development.md) |

## 按任务查阅

- **准备项目**：[项目](./guides/projects.md)、[执行环境](./guides/environments.md)、[用户](./guides/users.md)、[权限](./guides/roles.md)。
- **构建源码**：[仓库](./guides/repositories.md) → [阶段](./guides/stages.md) → [流水线](./guides/pipelines.md) → [制品](./guides/artifacts.md)。
- **部署应用**：[版本](./guides/versions.md) → [组件](./guides/components.md) → [服务](./guides/services.md) → [部署记录](./guides/deployments.md)。
- **接入网络**：[网关](./guides/gateway.md)、[DNS](./guides/dns.md)、[路由](./guides/routes.md)、[证书](./guides/certificates.md)。
- **维护实例**：[配置](./operations/configuration.md)、[备份](./operations/backup.md)、[升级](./operations/upgrade.md)、[排障](./operations/troubleshooting.md)。

左侧导航按这些任务分组，可展开查找具体主题。

## 阅读约定

本手册跟随 GitHub `develop` 分支，操作以实际实例版本为准。终端命令及手写脚本使用 Bash；平台生成的初始化命令按页面指定的解释器执行。首次练习使用独立环境，有状态业务先安排备份。

[GitHub 仓库](https://github.com/leoninew/pomelo-orbit) · [工程文档](https://github.com/leoninew/pomelo-orbit/blob/develop/docs/README.md)
