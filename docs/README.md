# Pomelo Orbit 文档中心
最后修改时间: 2026-07-24 10:47:37

## 阅读顺序（实现 / Agent 默认）

1. **[INDEX.md](./INDEX.md)** — 主题 → 文件导航  
2. **活 SoT**  
   - [产品概览](./product/overview.md)  
   - [CD 领域模型](./product/cd-model.md)  
   - [后端架构](./architecture/backend.md)  
   - [CD 运行时](./architecture/cd-runtime.md)  
   - [决策账本](./decisions/ledger.md)  
3. **How-to**：[`guides/`](./guides/)、[`frontend/`](./frontend/)  
4. **进行中过程文档**：`requirement/` / `spec/` / `plan/` / `verification/`（仅未归档任务）  
5. **归档** [`archive/`](./archive/) — **无须采信，不得作为实现或验收依据**

与代码冲突时 **以代码为准**，并回写活文档。

## 目录职责

| 路径 | 职责 |
|------|------|
| `product/` | 产品真相：领域、术语、能力边界 |
| `architecture/` | 技术/设计真相：栈、分层、运行时 |
| `decisions/` | 有效 / 废止 / backlog 账本 |
| `guides/` | 运维与操作 How-to（须对齐活 SoT） |
| `frontend/` | 前端规范与架构 |
| `requirement` 等 | SpecFlow **进行中** 过程文档 |
| `archive/` | 历史过程库与过时文；见 [archive/README](./archive/README.md) |

## 项目工程约束

见仓库根目录 [`CLAUDE.md`](../CLAUDE.md)（API 单数、迁移不可改已执行文件、检查命令等）。

## 维护原则

1. 活文档保持短、可导航；细节以代码为准。  
2. SpecFlow 闭环后的过程文档应迁入 `archive/specflow/`。  
3. 不在归档正文上「改写历史」；废止关系写入 `decisions/ledger.md`。  
4. 新增文档放对目录；不要把过程草稿当 SoT。  
