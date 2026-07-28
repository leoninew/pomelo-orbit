# MCP 部署测试可用性与清理能力
最后修改时间: 2026-07-27 15:49:19

Review status: Accepted

## Background

一次 RAGFlow 聚合 Compose 的真实部署测试表明，项目的 `mcp/` 包可以作为本地 stdio MCP Server 运行，且其 Application、Version、Service、Deployment、runtime 和 verification 工具可连接到本机 Orbit API。

但当前 Codex 会话没有将该 Server 注册为直接可调用的 MCP 工具。操作方只能从终端启动 Server，并以临时 Python `ClientSession` 包装调用；随后又调用了 `scripts/ragflow_initialize.py`。该脚本是针对拆分部署的编排器，不适合验证单一聚合 Compose 的直接 MCP 生命周期，额外创建了拆分基线和清理负担。

测试期间还观察到：

1. `orbit_stop` 要求调用方提供 `service_id`，但 MCP 没有公开列出一个 Application 所有 Service 的工具。调用方只能通过与 Service 查询无关的 runtime 工具间接取得该 ID，接口不自洽。
2. MCP 工具的验证、Orbit API、运行时和内部异常没有统一的外部响应模型与调用方处理入口。调用方可能将业务失败误记为成功，或依赖 SDK 的偶然行为处理错误。
3. 直接 MCP 操作必须依据用户描述选择 deploy、stop、delete 等动作；不能由测试流程替用户推导、补充或绑定生命周期步骤。

## Goal

1. 让项目 MCP 能被 Codex 等兼容客户端方便地按项目范围注册，并在会话中作为直接工具使用，而不是要求操作方编写 stdio client 包装代码。
2. 提供适合真实集成测试的直接 MCP 工作流：只执行用户明确描述的 Application、Version、Service、Deployment 和 runtime 动作。
3. 让 Application 到 Service 的查询、异常响应与处理、独立生命周期动作在 MCP 工具层自洽、可组合并可自动验证。

## Non-goal

1. 不修改 RAGFlow 镜像、Compose 业务参数或生产部署拓扑。
2. 不将 MCP stdio Server 暴露为 HTTP 网络服务。
3. 不在本需求中重做 Orbit Application、Service、Deployment 的领域模型或权限模型。
4. 不改变用户未明确要求的 Application、Service、Deployment、目录或卷。

## User scenarios

1. 开发人员打开项目后，可通过受版本控制的项目级配置或单一受支持命令注册 `pomelo-orbit-mcp`；客户端会话中直接出现具名 MCP 工具和输入 schema。
2. 测试人员创建一个临时测试 Application/Version/Service，直接调用 MCP 完成用户要求的 preview、publish、deploy、wait、verify 或 HTTP probe。
3. 测试失败或结束时，测试人员可先枚举目标 Application 的 Service，再按本次目标选择停止或删除；工具返回每一步的真实终态。绝大多数情况只需要停止服务。
4. 任意 MCP 调用失败时，客户端和编排层通过同一种结构化错误模型得到安全的错误摘要和 request ID，绝不把该调用记作成功。

## Acceptance

1. 项目提供可复现的 MCP 注册方式，适用于当前支持的 Codex 客户端；注册后可直接发现并调用 `pomelo-orbit-mcp` 的工具，无需临时 Python stdio client。
2. 注册配置使用项目相对工作目录或可移植的启动入口，在 Windows 和其他受支持环境不会因 `cwd`、`uv` 可执行文件解析或用户绝对路径而失效。
3. MCP 提供 `orbit_list_application_services(application_id)`，仅返回 Service 的非敏感身份、实例和状态摘要；调用方可据此获得 `orbit_stop` 所需的 `service_id`。
4. 所有 MCP 工具将验证、Orbit API、运行时和内部错误转换为同一种结构化错误响应；直接 MCP 调用将其视为工具失败，后续生命周期动作不得在失败后继续执行。
5. 直接测试路径明确使用现有、彼此独立的 `orbit_stop`、`orbit_wait_deployment` 和 `orbit_delete_application`。测试结束时通常仅停止 Service；Application 删除、目录删除或卷删除均为调用方显式选择，不新增以“清理”命名或固化其生命周期语义的复合工具。
6. 单一聚合 Compose 的测试可通过直接 MCP 调用完成，不必先创建拆分 RAGFlow Application；执行的动作严格与用户描述一致。
7. 注册、Service 列表、统一错误响应与处理、以及直接 MCP 动作路径均有自动化测试；真实 Docker 集成测试必须在显式启用且隔离的测试目标中运行。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. `mcp/` 继续使用本地 stdio transport，不作为远程 MCP endpoint。
2. 项目 MCP 使用受版本控制的项目 manifest 声明和注册，作为 Codex 等兼容客户端发现工具的标准入口。
3. `stop`、`wait` 和 `delete` 是独立的生命周期动作；调用方只执行用户明确要求的动作，不从“清理”等泛化词推导固定组合。
4. 测试 runtime config 使用现有普通 Service K/V 方式，不在本需求中引入额外的测试秘密模型。
5. MCP 的所有失败都使用统一的结构化错误契约；调用方不依赖个别 SDK 类型或单个工具的特殊返回。

## Risks

1. 自动注册依赖 Codex 客户端的实际配置模型；若没有稳定的项目级发现机制，需要提供受支持的安装和诊断命令，并明确会话重启要求。
2. 统一错误契约若未覆盖验证、Orbit API、运行时和内部异常，调用方仍可能出现分支化的错误处理或错误地继续后续步骤。
3. 直接测试仍会消耗 Docker 镜像、内存、端口和工作区；隔离命名、维护窗口和超时策略必须进入后续计划。

## User review notes

用户要求以 `standard` / 标准模式记录本次真实 MCP 部署测试的观察结果和改进项，特别包括便利的 MCP 注册能力。

用户确认：项目 MCP 以项目 manifest 声明；现有停止和删除接口/MCP 工具足以满足不同测试流程；测试配置采用常规方式满足需求。

用户澄清：“清理”不是 `stop + delete` 的语义定义。大多数测试场景的后续操作只到停止服务为止；删除是独立、显式的资源销毁选择。用户进一步确认：Application 到 Service 的查询是必须补足的接口自洽性；所有异常响应和处理方式必须一致；本需求不涉及脚本、journal 或其存储位置。
