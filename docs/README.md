# 文档

Pomelo Orbit 管理 CI 流水线、Docker Compose 应用部署与 Traefik 网关，支持本机和 SSH 执行环境。

## 核心模型

| 对象 | 职责与关系 |
| --- | --- |
| Project / Environment | Project 隔离成员和业务资源；初始化后拥有一个 CI/CD 共用环境，保存执行目标与工作区。 |
| Environment Credential | SSH 环境的当前受管凭据，通过唯一的 `environment_id` 归属环境；先保存环境，再显式初始化凭据。 |
| Repository / Credential | 所有 Project 共用的仓库与仓库凭据，与环境 SSH 凭据分开管理。 |
| Pipeline / Stage | 全局模板与阶段用于复用；Project 内的应用流水线绑定仓库，通过快照固定执行定义。 |
| Run / Artifact | 流水线执行及其产物；已映射组件的镜像可汇入一个新的应用版本。 |
| Application / Version / Component | Project 内的应用及其版本化 Compose 拓扑声明。 |
| Service / Deployment | Service 绑定版本、运行覆盖和部署目录；Deployment 冻结一次异步操作的输入。 |
| Gateway / Route | 环境绑定一个 Traefik Gateway；自定义 Route 管理 HTTP/TCP 入口及目标。 |

环境 Probe 仅用于诊断；执行操作校验当前目标、工作区、凭据与主机指纹。SSH 失败不回退本机。Service 以最新受理的操作为准，旧任务结果不能覆盖新状态。

开发与常用命令见 [项目 README](../README.md)，工程约定见 [AGENTS.md](../AGENTS.md)。具体业务规则以对应实现、测试及进行中的变更规格为准。

## 变更记录

进行中的 SpecFlow 文档位于 `intent/`、`spec/`、`plan/`、`verification/`。

[archive/](archive/README.md) 保存历史材料，不作为当前实现或验收依据。
