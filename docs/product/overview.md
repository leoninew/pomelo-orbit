# 产品概览
最后修改时间: 2026-07-24 10:47:37

Doc role: living SoT  
权威：与代码冲突时以代码为准。

## 产品是什么

Pomelo Orbit 是自托管的 **CI + CD + 平台管理** 控制面：

| 域 | 能力（现行） |
|----|----------------|
| **CD** | 应用版本化规格、环境、服务运行绑定、部署流水、Gateway、平台 Route、本机 Docker Compose 部署 |
| **CI** | 仓库、流水线模板/快照、构建阶段、运行、产物、凭证与变量 |
| **平台** | 项目、用户、角色权限、系统设置、后台任务队列 |

前端目录：`web/`。后端：`cmd/server` + `internal/`（Go）。

## 阅读顺序

1. 本页（边界）
2. [CD 领域模型](./cd-model.md)
3. [后端架构](../architecture/backend.md) / [CD 运行时](../architecture/cd-runtime.md)
4. [决策账本](../decisions/ledger.md)
5. 需要操作步骤时再看 `docs/guides/`

## 非目标（产品层）

- 不做多集群 / 完整 K8s 编排产品化（代码中无对应主路径）。
- 不做「用户自管 external Traefik」为主线必做能力。
- 不做已执行迁移的历史兼容；开发期允许重建库。
- 活跃开发期：**无兼容层、别名、默认值兜底、新旧逻辑并存**（见根目录 `CLAUDE.md`）。

## 文档分层

| 类型 | 路径 | 用途 |
|------|------|------|
| 活 SoT | `docs/product`、`architecture`、`decisions`、校准后的 `guides`/`frontend` | 实现与评审默认依据 |
| 过程文档 | `docs/requirement\|spec\|plan\|verification` | **仅进行中** SpecFlow 任务 |
| 归档 | `docs/archive/**` | 历史；**无须采信** |
