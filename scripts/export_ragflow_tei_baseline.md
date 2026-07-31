# `export_ragflow_tei_baseline.py` — 导出 RAGFlow TEI SQLite 控制面基线

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

从 SQLite 数据库的只读连接导出 RAGFlow + TEI 停止态控制面 SQL 基线。导出范围包含指定 Project、RAGFlow 与 Gateway Application、CPU/GPU Version、组件、Service、Expose 及关联配置；Deployment 不在范围内。

为使 SQL 能直接初始化一个新的 RAGFlow 实例，导出时会为 RAGFlow 的 `default` Service 随机生成 `MYSQL_PASSWORD`、`REDIS_PASSWORD`、`MINIO_USER`、`MINIO_PASSWORD` 和 `ELASTIC_PASSWORD`。这些值不从现有 Service 复制，不写入终端日志，并由组件环境变量中的占位符引用。Gateway Service 不需要这些变量，仍导出为空对象。每次导出都会产生不同的 RAGFlow 运行时配置。

基线保留 Project 记录，但以 `INSERT OR IGNORE` 写入，以兼容标准 SQLite 迁移已经创建的同 ID `default` Project。其余记录保持普通 `INSERT`，因此在错误的预置数据或重复导入时会明确失败。

## 用法

```bash
# 使用默认数据库，经过完整校验后显式替换受版本控制的基线
python scripts/export_ragflow_tei_baseline.py \
  --output scripts/ragflow-split/ragflow-bundled-sqlite.sql \
  --replace

# 指定数据库和 Project
python scripts/export_ragflow_tei_baseline.py \
  --database data/db/pomelo-orbit.db \
  --project-id <project-id> \
  --output /tmp/ragflow-tei-baseline.sql
```

## 安全边界

数据库通过 SQLite `mode=ro` 打开；脚本先执行完整性和外键检查，并要求 RAGFlow/Gateway、CPU/GPU Version、组件与暴露配置完全符合预期。CPU/GPU 只能在 TEI 固定镜像及 GPU device request 上不同。`--output` 必填；已有文件只有明确给出 `--replace` 才会在全部校验成功后原子替换。脚本不会修改数据库。

## 相关文档

RAGFlow + TEI 运行模型和基线来源请参阅 [`../docs/guides/ragflow-tei-operations.md`](../docs/guides/ragflow-tei-operations.md)。
