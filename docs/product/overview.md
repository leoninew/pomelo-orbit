# 产品概览
最后修改时间: 2026-08-07 16:35:00

Doc role: living SoT  
权威：与代码冲突时以代码为准。

## 产品是什么

Pomelo Orbit 是自托管的 **CI + CD + 平台管理** 控制面：

| 域 | 能力（现行） |
|----|----------------|
| **CD** | 应用版本化规格、环境、服务运行绑定、部署流水、Gateway、平台 Route、本机 Docker Compose 部署 |
| **CI** | 仓库、Pipeline 模板与应用流水线、Pipeline 自有阶段、运行、制品、凭据与变量 |
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

## CI 模型

CI 的可复用性与可运行性分为两个显式对象：

- `Pipeline(kind=template)` 是通用编排来源，不能运行、不能绑定 Application 或 Repository，也不创建 Snapshot。
- `Pipeline(kind=application)` 由一个 Template 物化，固定绑定 Repository；Application 仅在需要将制品写入 Component Version 时绑定。运行时只提交 ref 与变量。
- `PipelineStage` 由一个 Pipeline 独占，包含 DAG 依赖和制品声明；不再是跨流水线复用资源。
- `docker_image` 制品可以选择目标 Component。所有目标 Component 在同一 Pipeline 内唯一；一次成功 Run 将所有已映射镜像原子写入一个新 Version。
- 只有应用流水线创建不可变 `PipelineSnapshot`。Run、Artifact、Version Component 保存必要的显示快照，因而不依赖 Template、Application、Repository、Version 或 Pipeline 的物理存在。

CI 的详细模型、API 与变量语义见 [CI 流水线设计](../guides/ci-pipeline-design.md) 和 [CI 变量](../guides/ci-pipeline-vars-design.md)。

## 文档分层

| 类型 | 路径 | 用途 |
|------|------|------|
| 活 SoT | `docs/product`、`architecture`、`decisions`、校准后的 `guides`/`frontend` | 实现与评审默认依据 |
| 过程文档 | `docs/requirement\|spec\|plan\|verification` | **仅进行中** SpecFlow 任务 |
| 归档 | `docs/archive/**` | 历史；**无须采信** |
