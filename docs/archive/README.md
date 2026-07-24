# 归档文档

最后修改时间: 2026-07-24 10:47:37

## 重要声明

本目录及子目录内文档为 **历史材料**。

- **无须采信**  
- **不得作为实现、设计或验收的依据**  
- 仅供追溯「当时为何那样做」  

现行真相见：

- [docs/README.md](../README.md)  
- [docs/INDEX.md](../INDEX.md)  
- `docs/product/`、`docs/architecture/`、`docs/decisions/`、校准后的 `docs/guides/`  

与代码或活文档冲突时，**以代码与活文档为准**。

## 目录结构

| 路径 | 内容 |
|------|------|
| `specflow/requirement/` | 已闭环 SpecFlow 需求（原 `docs/requirement` 已提交历史） |
| `specflow/spec/` | 已闭环规格 |
| `specflow/plan/` | 已闭环计划 |
| `specflow/verification/` | 已闭环验证记录 |
| `designs/` | 早期设计稿与过时架构（含 Python 栈描述） |
| `discussions/` | 设计讨论过程 |
| `specs/` | 更早期阶段性 specs |
| `guides/` | 已退场 guides（如旧 Application 状态机） |
| `analyze/` | 已消化的分析文原文 |
| 根下散落 md | 如 `ci-quickstart.md`、`frontend-comparison.md` 等 |

## 使用方式

1. 开发与 Agent：**不要**打开本目录寻找「当前该怎么做」。  
2. 考古：从 [INDEX 归档节](../INDEX.md) 或主题关键词进入。  
3. 废止关系摘要：见 [decisions/ledger.md](../decisions/ledger.md)。  
