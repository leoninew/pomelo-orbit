# RAGFlow 内嵌 TEI CPU/GPU Version 实施计划
最后修改时间: 2026-07-31 12:40:00

Review status: Accepted

## Basis

依据已接受的 [Requirement](../requirement/20260731-ragflow-tei-variants.md) 与 [Spec](../spec/20260731-ragflow-tei-variants.md)。用户已明确要求开始实施，因此本计划直接进入 Implementation / 实施。GPU 容器运行测试、RAGFlow API 注册/API key 流程不在本轮。

## Steps

1. 在现有 `000023_application` SQLite/MySQL schema 中加入 device request 表，原地对齐开发库；扩展 model、sqlc query/repository、应用 DTO/validation/use case、HTTP/Proto/MCP/前端组件编辑界面，并重新生成 sqlc 与 Proto。
2. 将 device request 渲染到 Compose `deploy.resources.reservations.devices`，加入 CPU/GPU renderer 与独立 devices 更新的聚焦测试。
3. 新建 `scripts/prepare_ragflow_tei.py`：实现工具/缓存/manifest/镜像/GPU 条件预检、Git LFS/HF CLI 下载、image pull、`tar.gz` 备份和安全恢复。新增脚本测试，真实模型归档只在代码与安全检查完成后作为显式运行步骤。
4. 新建 `scripts/export_ragflow_tei_baseline.py`，导出控制面记录时为 RAGFlow `default` Service 随机生成首次初始化所需的运行时变量，Gateway 仍为空对象，Deployment 一律排除；用临时 SQLite fixture 覆盖导出后导入与密钥键集校验。
5. 把 bundled Compose 参考扩展为 TEI 组件，创建合并后的 `deploy-ragflow-tei-orbit` 技能，旧技能改为受控迁移入口，更新 `AGENTS.md` 和运维手册。所有实际生命周期操作保持经 `pomelo_orbit` MCP。
6. 运行生成、单元/集成和静态检查；完成 CPU 预检、缓存 `tar.gz` 备份/恢复和 CPU 部署验证时，记录真实结论。GPU 仅运行静态检查和缺失条件检查。

## Files And Ownership

- 组件领域和渲染：`internal/model/`、`internal/application/application/`、`internal/application/deployment/`、`internal/repository/`、`internal/api/http/`、`proto/`、`sql/`、`web/`、`mcp/`。
- 准备与导出：`scripts/prepare_ragflow_tei.py`、`scripts/export_ragflow_tei_baseline.py` 及各自测试。
- 交付契约：`scripts/ragflow-bundled/`、`scripts/ragflow-split/ragflow-bundled-sqlite.sql`、`skills/`、`AGENTS.md`、`docs/guides/`。

## Verification And Rollback

先验证 schema/renderer/脚本的离线单元测试，再执行不改变 Orbit 状态的预检和 SQL 导入测试。脚本不覆盖非空缓存且备份使用临时输出后原子替换；缓存恢复失败仅清理本次创建的临时目录。控制面基线 SQL 先写候选路径、导入空数据库验证后才替换跟踪文件。GPU profile 在本机只应明确失败，不可被测试伪造为可部署。
