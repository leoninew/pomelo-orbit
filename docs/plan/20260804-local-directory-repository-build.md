# 本地目录仓库构建计划
最后修改时间: 2026-08-04 17:17:23

Review status: Accepted

流程模式: 标准 / standard

## 需求依据

- [Requirement](../requirement/20260804-local-directory-repository-build.md) 已接受。
- 本地源码必须使用 Git commit shallow 检出至既有项目 workspace，不可写回真实目录。

## 实施步骤

1. 不增加 Pipeline 配置。就地修改 SQLite/MySQL repository 的原始 DDL，为 repository 保存 `repository_type` 与 `repository_url`；Pipeline Run 沿用既有 `trigger_ref`，不增加来源快照列或编号 migration。
2. 更新 Repository 模型、SQL 查询、SQLC 仓储、DTO、Proto、HTTP mapper 和 Web API。`repository_url` 是唯一来源地址：远程仓库保存 URL，本地仓库校验并保存任意 Worker 可访问的 Git 目录；远程仓库保持凭据规则。
3. 触发本地 Pipeline Run 时将 requested ref 解析为 commit SHA，并写入既有 `trigger_ref`。执行时继续读取当前 Repository：本地源额外挂载 `repository_url` 指向的目录为只读 `/source`，运行变量 `repository_url` 为 `file:///source`。
4. 保持按 repository code 共享的 workspace 策略。构建 Stage 继续挂载既有项目 workspace 和当前 run artifacts；本地源额外挂载为只读 `/source`。
5. 保留既有 `git clone` Stage 的脚本和版本；它通过运行变量同时支持远程 URL 与本地 `file:///source`。凭据注入只用于远程 Git。
6. 更新仓库列表和创建表单：选择远程 Git 或本地目录；本地目录输入 Worker 可访问的 Git 路径并隐藏 Git credential，远程模式保留 URL/credential。

## 预计涉及文件

| 区域 | 主要路径 |
| --- | --- |
| 模型、迁移 | `internal/model/**`、`sql/migration/{sqlite,mysql}/000021_repository.*.sql` |
| 仓库数据链路 | `sql/query/repository/**`、`internal/{repository,application,api}/repository/**`、`proto/orbit/v1/repository/**` |
| 流水线运行 | `internal/application/pipeline_run/**`、`internal/infrastructure/storage/local/{pipelineworkspace,repositorysource}/**` |
| 前端与生成代码 | `web/src/views/repository/**`、`web/src/api/repository/**`、`internal/gen/**`、`web/src/gen/**` |

## 验证计划

1. 覆盖本地路径校验：Worker 可访问路径、符号链接、非 Git 目录、bare repo、linked worktree。
2. 覆盖 ref 到 commit SHA 的解析并写入 `trigger_ref`；确认标准 `git clone` Stage 使用该 SHA。
3. 覆盖本地源只读挂载、既有项目 workspace 路径不变，以及远程源不产生 `/source` 挂载。
4. 执行 SQLC/Proto 生成，并运行相关 Go 测试与前端类型检查。浏览器测试按用户要求不纳入本轮。

## 假设与风险

- 本地目录必须对执行 Worker 可见；生产多 worker 需提供一致的本机路径和源码内容。
- 执行时读取当前 Repository 的源码配置；待执行 Run 在仓库目录被编辑后仍使用已固定的 commit SHA，但会从更新后的目录挂载。
- 浏览器测试按用户要求不纳入本轮；代码验证在实现完成后执行。

## Rollback

- 部署前可回滚应用二进制；本地目录功能不改变已有远程 Run 的读取路径。

## User review notes

- 2026-08-04：用户明确要求开始开发；Requirement 与 Plan 视为已接受。
- 2026-08-04：用户确认现有 workspace 由 shallow clone 重置；撤回将其改为 run 专属目录的范围外调整。
- 2026-08-04：用户要求本地源为任意 Worker 可访问的 Git 目录，`data/` 仅管理 workspace、日志与制品。
- 2026-08-04：用户确认标准 `git clone` Stage 已兼容本地 `file:///source`，不保留专用本地 clone Stage。
