# MCP

本地 stdio MCP 让代理操作项目、应用、服务、部署和网络资源。客户端启动 `pomelo-orbit mcp`，进程直接使用实例配置、数据和执行资源。

## 配置客户端

维护者准备与实例匹配的程序、数据库访问、配置和原有主密钥，然后按客户端格式填写：

| 字段 | 值 |
| --- | --- |
| 命令 | `pomelo-orbit` 程序绝对路径，Windows 使用 `.exe` |
| 参数 | `["mcp"]` |
| 工作目录 | 包含 `configs/config.yaml` 的实例配置根目录 |
| Profile | `POMELO_ORBIT_APP__ENV`，与实例一致 |
| 认证 | 通过可信环境注入 `POMELO_ORBIT_MCP__ACCESS_TOKEN`，值为个人 PAT |

配置优先级见[系统配置](../operations/configuration.md)，身份范围见[访问令牌](../guides/access-tokens.md)。更新配置后建立新 MCP 进程。

此入口使用 stdio，不提供 HTTP MCP URL。标准输出只传协议，诊断写标准错误或日志文件。省略 `mcp` 子命令会启动默认服务入口。

## 先确认项目

按以下顺序调用，读取资源前先回读确认项目：

| 工具 | 参数与检查 |
| --- | --- |
| `orbit_list_projects` | `{}`，核对项目 ID、编码和名称 |
| `orbit_select_project` | `{"project_id":"<真实项目 ID>"}` |
| `orbit_get_current_project` | `{}`，回读项目、环境和网关摘要 |
| `orbit_list_applications` | `{}`，读取该项目业务应用 |

选择要求当前用户是成员，且项目已有环境、网关及网关服务记录；缺失会返回 `project_not_ready`。这些记录齐全不代表网关容器运行。先在 Web 完成初始化，再接入。

选择属于当前连接，新进程重新选择。切换项目后回读确认，再执行写入；浏览器的当前项目与该连接独立。

## 部署结果

先按[服务](../guides/services.md)准备版本、变量、预览和部署目录，再调用写入工具。

| 工具 | 检查结果 |
| --- | --- |
| `orbit_deploy` | 保存返回的 `deployment_id` |
| `orbit_deployment_status` | 查询该次操作状态 |
| `orbit_deployment_logs` | 按 ID 和 `offset` 读取增量日志，保留新 offset |
| `orbit_wait_deployment` | 同时检查 `timed_out` 和实际终态 |

MCP 不执行数据库迁移，也不启动 HTTP 服务或 Worker。异步操作由实例已有执行进程处理，停止客户端或等待超时后仍需查询原记录。

工具可进行真实写入，日志与 Compose 结果也可能包含秘密。发送给代理前明确授权范围和可共享内容。
