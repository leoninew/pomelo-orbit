# Service Code Deployment Directory
最后修改时间: 2026-08-09 22:51:54

Review status: Accepted

## Background

当前部署工作目录以 `application_code/instance_key` 组织，组件目录挂载又使用 `./data/<component>`。当服务实例键为 `default` 时，持久化路径同时包含不稳定的实例键和多余的 `data` 层级，且普通 directory 挂载没有在部署前显式物化。

## Goal

以全局唯一、不可变的 Service code 作为部署工作目录根。服务 `sc` 的组件相对目录挂载使用 `./redis`、`./mysql` 等，分别落到 `data/deployment/sc/redis`、`data/deployment/sc/mysql`。

## Non-goal

- 不保留旧 `application_code/instance_key` 或 `data` 路径的兼容、回退或自动迁移。
- 不自动迁移既有目录；本地数据处置仅限用户明确授权的目标。
- 不执行部署生命周期操作。

## User Review Notes

2026-08-09，用户明确授权处置本地 `data/deployment`：将已停止的
`cli-proxy-api/default` 和 `ragflow-integrated/default` 移至各自的 Service
code 目录，扁平化 RAGFlow 的组件数据目录，并删除孤立的
`ragflow-integrated/default.rar` 备份。运行中的服务和没有 Service 映射的
遗留目录保持不动。

## Acceptance

- 服务工作目录、Compose 工作目录和部署日志目录均以 Service code 为根，不含 application code 或 instance key。
- 逻辑 directory 挂载相对服务根目录解析；`./mysql` 解析为 `<service-code>/mysql`。
- 部署前创建所有非 host-path directory 挂载目录。
- RAGFlow 集成式和拆分式组件均使用以组件 code 开头的 bind source，不使用 Docker named volume 或 `./data/...`。
- 相关测试覆盖新目录规则和普通目录挂载物化。

## Risk

这是持久化路径的破坏性变更。已有目录不会被读取；运行服务必须在用户完成数据处置后重新部署。未停止或无法映射到 Service 的目录不能由本变更自动处理。
