# 文档主题索引
最后修改时间: 2026-08-22 21:01:18

独立索引文件（与 [README.md](./README.md) 分工：README 讲规则与阅读顺序，本页讲主题 → 路径）。

## 产品

| 主题 | 活文档 |
|------|--------|
| 产品边界与阅读入口 | [product/overview.md](./product/overview.md) |
| CD 领域模型 / 术语 / 生命周期 | [product/cd-model.md](./product/cd-model.md) |
| 有效与废止决策 | [decisions/ledger.md](./decisions/ledger.md) |

## 架构与技术

| 主题 | 活文档 |
|------|--------|
| 后端栈与分层 | [architecture/backend.md](./architecture/backend.md) |
| CD Render / Gateway / 部署执行 | [architecture/cd-runtime.md](./architecture/cd-runtime.md) |
| 前端架构（目录 `web/`） | [frontend/architecture.md](./frontend/architecture.md) |
| 前端 Reka / 样式 | [frontend/reka-ui-guide.md](./frontend/reka-ui-guide.md)、[frontend/style-guide.md](./frontend/style-guide.md) |

## 运维与 How-to

| 主题 | 活文档 |
|------|--------|
| CD 部署概念 | [guides/deployment.md](./guides/deployment.md) |
| Docker 部署 | [guides/docker-deployment.md](./guides/docker-deployment.md) |
| 服务器首次部署 | [guides/server-deployment.md](./guides/server-deployment.md) |
| 路由与证书 | [guides/routing-and-certificates.md](./guides/routing-and-certificates.md) |
| 证书管理 | [guides/certificate-management.md](./guides/certificate-management.md) |
| Docker labels 路由 | [guides/docker-label-routing.md](./guides/docker-label-routing.md) |
| 卷挂载 | [guides/volume-mounting.md](./guides/volume-mounting.md) |
| CI Pipeline（模板、应用流水线、制品与运行） | [guides/ci-pipeline-design.md](./guides/ci-pipeline-design.md) |
| CI Pipeline 变量 | [guides/ci-pipeline-vars-design.md](./guides/ci-pipeline-vars-design.md) |
| 服务导出导入（dbtalk JSONL） | [guides/service-data-transfer.md](./guides/service-data-transfer.md) |
| Agent Plugin 与 Skill 发布 | [guides/plugin-release.md](./guides/plugin-release.md) |
| 权限 RBAC | [guides/permissions.md](./guides/permissions.md) |
| Google OAuth（国内网络） | [guides/google-oauth-china-network.md](./guides/google-oauth-china-network.md) |
| AntDV icons 笔记 | [guides/antdv-icons.md](./guides/antdv-icons.md) |

## 工程约束

| 主题 | 位置 |
|------|------|
| 项目硬约束 | [AGENTS.md](../AGENTS.md) |

## SpecFlow 过程文档

| 状态 | 路径 |
|------|------|
| 进行中 | `docs/requirement/`、`docs/spec/`、`docs/plan/`、`docs/verification/` |
| 已归档（无须采信） | [archive/specflow/](./archive/specflow/) |

## 归档索引（无须采信）

归档内容仅供历史追溯，不作为实现或验收依据。目录说明和使用约束见
[archive/README.md](./archive/README.md)；废止主题及其替代方案见
[decisions/ledger.md](./decisions/ledger.md)。
