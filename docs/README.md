# 文档

Pomelo Orbit 管理 CI 流水线、Docker Compose 应用部署与 Traefik 网关，支持本机和 SSH 执行环境。

[使用手册](handbook/index.md)面向平台使用者和实例维护者，提供项目配置、构建部署、网络入口及日常维护说明；在线入口为 [GitHub Pages](https://leoninew.github.io/pomelo-orbit/)。

首次使用从[自托管安装](handbook/getting-started/installation.md)和[首个应用部署](handbook/getting-started/first-deployment.md)开始；实例维护参阅[数据保留与升级](handbook/operations/data-upgrades.md)及[故障定位](handbook/operations/troubleshooting.md)。安装和运行示例应先在隔离环境按正文检查，不直接作为生产配置。

本目录其余内容为工程资料。参与源码开发见[开发入门](development.md)，不需要先完成源码开发环境才能阅读使用手册。

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

Cloudflare DNS 管理复用当前环境 Gateway 的 DNS API Token，需为目标 Zone 授予 Zone Read 和 DNS Edit 权限。网络接入中的 DNS 页面支持搜索、新增和删除 A 记录，以及修改已有记录的 IPv4、TTL 和代理状态，操作直接写入 Cloudflare，无需网关容器运行。可见 Zone 以 Token 权限范围为准，可能由多个环境共享。

前端布局、文字、表格及共享控件遵循[前端界面规范](frontend-design.md)。

访问令牌使用 `POMELO_ORBIT_JWT__SECRET_KEY` 进行 Fernet 加密存储，列表默认遮罩，可显隐及复制；认证仍使用令牌哈希。迁移前仅保存哈希的记录，其可查看令牌值为空，需要重新创建才能查看或复制。更换该密钥会影响已有令牌密文及其他加密凭据的解密。

本次全系统前端统一的[意图](intent/20261028-open-canvas-visual-language.md)、[计划](plan/20261028-open-canvas-visual-language.md)和[验证结果](verification/20261028-open-canvas-visual-language.md)使用同一份 SpecFlow 记录。

## 变更记录

进行中的 SpecFlow 文档位于 `intent/`、`spec/`、`plan/`、`verification/`。

[archive/](archive/README.md) 保存历史材料，不作为当前实现或验收依据。
