# 本地目录仓库构建需求
最后修改时间: 2026-08-04 17:17:23

Review status: Accepted

流程模式: 标准 / standard

## Background

持续集成当前只接受远程 Git URL，并在流水线 Stage 中通过 shallow `git fetch` 检出代码。本机构建场景需要直接使用宿主机上的源码目录，但构建不得把 Git 操作写回真实目录。

## Goals

- Repository 支持 `remote_git` 与 `local_directory` 两种源码类型。
- 本地目录必须是 Worker 可访问的普通 Git 工作目录或 bare repository，并以指定 ref 解析出的 commit 作为本次构建输入。
- Stage 容器只读挂载本地源目录，通过 `file:///source` shallow clone 到既有的项目 `/workspace`；真实目录不被修改。
- 保持远程 Git 构建可用，远程凭据只用于远程 Git。

## Non-goals

- 不支持非 Git 目录、linked worktree 或运行时写回本地源目录。
- 不实现本地目录的 Webhook、自动发布、自动部署或 Registry 推送。
- 不改写现有 Pipeline Stage 的业务模型；checkout 仍是可见、可编辑的业务 Stage。

## User scenarios

1. 用户填写一个 Worker 可访问的 Git 工作目录，创建本地目录仓库并设置默认分支。
2. 用户触发流水线时，系统将该 ref 固定为 commit SHA；checkout Stage 在既有项目 workspace 从只读 `/source` shallow 检出该 commit。
3. 远程仓库仍按现有 URL 和 Git credential 拉取；本地目录仓库不显示或保存 Git credential。

## Acceptance criteria

- Repository API、模型和界面通过 `repository_type` 区分源码类型，并以唯一的 `repository_url` 保存远程 URL 或 Worker 可访问的本地 Git 路径。
- 本地路径须解析为可访问 Git 目录；非 Git 目录和 linked worktree 必须拒绝。
- 触发 Run 时将经 Git 解析的 commit SHA 写入 `trigger_ref`，并在标准 checkout Stage 对本地源检出该 SHA。
- 本地源码以只读 Docker bind mount 暴露为 `/source`；既有的项目 `/workspace` 策略保持不变，`/artifacts` 继续按 run 隔离。
- 远程 Git 使用既有凭据注入机制，本地目录不接受凭据。

## Open questions

暂无需要用户确认的未决事项。本期不新增 Pipeline 配置或本地源码根目录。

## Decisions

- 用户明确要求继续以 Git clone/checkout 的方式构建本地目录，避免真实目录被污染。
- 本期在 Run 创建时将 requested ref 解析为 commit SHA 并写入既有 `trigger_ref`；执行 Stage 读取当前 Repository 源配置。
- `repository_type` 位于 `repository_url` 前；不保留 `source_type` 或 `local_path` 字段。
- 开发数据库与 migration 源均就地演进：修改现有 repository、pipeline run 与 pipeline seed 的原始 DDL/seed，不新增编号 migration。
- 用户明确要求不进行浏览器测试；代码验证在实现完成后执行。
- 2026-08-04：用户确认现有 workspace 依赖 shallow clone 重置，按项目共享并非本需求要修复的问题；不得将其改为 run 专属目录。
- 2026-08-04：用户要求本地源为任意 Worker 可访问的 Git 目录，`data/` 只管理 pipeline workspace、运行日志和制品。
- 2026-08-04：保留原 `git clone` Stage；本地目录通过运行变量 `repository_url=file:///source` 使用同一张卡片。

## Risks

- 本地目录只能在执行 worker 所在宿主机可见；多 worker 部署需保证相同路径和源码内容可用。
- 只读 bind mount 依赖 Docker Desktop/宿主机对物理路径的共享配置；无法映射时运行会以明确错误失败。
- 提交 SHA 的解析依赖本地 Git 可执行文件；不存在的 ref 或损坏 Git 元数据将阻止 Run 创建。

## User review notes

- 2026-08-04：用户要求以标准 / standard 流程并行推进持续集成 MCP 评估和本地模式仓库构建支持；本地目录构建由当前 agent 实施。
