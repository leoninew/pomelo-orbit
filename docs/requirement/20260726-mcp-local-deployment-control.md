# 本地 Docker MCP 部署控制与产品验证需求
最后修改时间: 2026-07-26 14:48:37

## Review status

Accepted

## Background

Pomelo Orbit 当前将 Application、Version、Component、Expose、Service 与 Deployment 作为 CD 领域模型，并由后台 worker 渲染 Compose 后执行本地 Docker Compose。现有 Web/API 能表达业务意图，但缺少一个能供 LLM 使用、同时能读取 Docker 运行时实态的集成面。

本需求不是将 Docker 操作隔离在产品外。目标是让 LLM 在本地 Docker 环境中同时：

1. 通过 Pomelo Orbit 创建应用与版本、发起部署；
2. 通过 Docker / Docker Compose 获取容器、日志、端口、网络和 Labels 等运行时事实；
3. 对比领域规格、渲染产物、Deployment / Service 状态与 Docker 实态，从而验证产品设计和实现是否合理、正确且可观测。

用户已明确选择在仓库新增独立 `mcp/` 目录，使用 Python + uv 实现。

## Goal

- 新建独立的本地 MCP Server，作为 uv 管理的 Python 包运行，不嵌入现有 Go 服务进程。
- 暴露创建 Application、创建/维护 Version、维护 Component / Expose、预览、部署、查询部署状态与日志的能力。
- 暴露受管 Application + Environment + instance 维度的 Docker Compose 运行时查询能力，至少包括 Compose 配置、容器状态、容器/Compose 日志、容器 inspect 与网络 inspect。
- 提供面向产品验收的验证能力：检查 Version 渲染的预期、Compose 配置、Docker 实态以及 Pomelo Orbit 的 Service / Deployment 记录是否一致，并诊断漂移或失败原因。
- 使用 Dynaconf 和环境文件管理 MCP 配置，并复用或自动刷新 Orbit JWT。

## Non-goal

- 不把 MCP 作为绕过 Pomelo Orbit 领域模型、直接写 SQLite/MySQL 数据库的旁路。
- 不在首版实现远程 Docker 主机、Kubernetes、SSH 发布或通用 Shell 执行器。
- 不把任意 `docker` 命令列表直接暴露成无范围限制的工具；运行时查询应能关联到受管的应用、环境和实例。
- 首版不提供绕过 Orbit 的 `docker compose up`、`down`、`restart` 等生命周期写操作；后续是否收拢或开放此类运行时覆盖，另立需求评估。
- 不在本需求中重构既有 Go CD worker、Compose renderer 或前端部署 UI；为 MCP 所需补充的 HTTP API / 鉴权能力另由后续 Spec 明确。
- 不承诺首版支持 Gateway、证书、公开 TCP、逻辑挂载、任意 host path 挂载等所有高级部署模型。
- 不实现 MCP Token、PAT 或任何新的 Token 数据模型、鉴权接口和撤销链路。

## User scenarios

1. 用户要求创建一个标准应用，Agent 通过 MCP 创建 Application 和未发布 Version，添加镜像组件与 HTTP Expose，先预览生成的 Compose，再通过 Orbit 发起部署。
2. 部署后，Agent 获取 Deployment 状态和部署日志，并通过 Docker Compose 查询实际容器、镜像、端口、网络与 Labels，确认它们符合 Version 的预期。
3. 用户要求排障时，Agent 读取 `docker compose logs`、容器 inspect 和 Orbit 部署日志，说明“领域状态显示运行但容器已退出”等差异。
4. 用户要求验证产品行为时，Agent 通过 Orbit 发起创建、发布、部署、停止和重启；Docker / Docker Compose 只用于读取日志和运行时实态，并回查产品状态。
5. 多用户或长期使用时，MCP 以配置用户的 JWT 调用 Orbit API；Docker Socket/CLI 的本机权限不自动扩大 Orbit 用户对其他应用或环境的操作范围。
6. MCP 执行 Orbit 操作时，优先使用本地缓存且未过期的 JWT；JWT 缺失或已过期时，使用环境配置中的用户名和密码登录并更新缓存。

## Acceptance

1. 仓库存在独立 `mcp/` Python + uv 包，可通过约定命令以本地 stdio MCP Server 方式启动；Python 依赖和锁文件可复现。
2. MCP 的 Orbit 业务工具可完成应用、版本、组件、暴露、发布以及部署生命周期的核心流程，并使用 HTTP API / 后续明确的服务接口，不直接访问数据库。
3. MCP 的运行时工具能在指定的受管 Application + Environment + instance 范围内读取 Compose 配置、容器状态、日志、inspect 和网络信息。
4. MCP 至少提供一个部署验证工具，报告以下各层的匹配或差异：Version / Compose 预期、Docker 容器实态、Service 状态、Deployment 状态与日志。
5. 运行时查询结果可用于定位镜像拉取失败、容器退出、重启循环、端口/网络/Label 不一致等常见问题；本地部署数据不作脱敏处理，但认证凭据本身不得输出。
6. 任何会创建、发布、部署、停止、重启或删除的工具都有明确操作语义、目标范围和审计关联；无需增加确认步骤，MCP 必须即时展示已发起的关键业务步骤及命令/请求摘要。只读诊断工具不触发写操作。
7. MCP 配置由 Dynaconf 从本地环境文件和进程环境加载；用户名、密码和 JWT 不提交到仓库、不写入日志或 MCP 工具响应。
8. 每次 Orbit 操作前检查已存 JWT 的有效期；有效时直接作为 Bearer Token 使用，缺失或过期时使用配置用户名密码登录并存储新的 JWT。
9. 单元测试覆盖 Orbit HTTP 客户端、JWT 有效期判断、登录与缓存刷新、Docker/Compose 命令构造、目标范围解析和预期/实际对账逻辑；Docker 集成测试在显式标记的本机环境运行。

## Authentication baseline

当前认证基线为用户名密码登录后签发 Bearer JWT，JWT 固定 24 小时有效期，`logout` 不会撤销已发 JWT。MCP 不新增 Token/PAT，而是通过 Dynaconf 从本地环境文件和进程环境读取 Orbit 地址、用户名、密码与缓存 JWT。

每次需要调用 Orbit 时，MCP 先检查缓存 JWT 是否存在且 `exp` 未过期。满足条件时直接使用该 JWT；否则使用配置中的用户名密码调用登录接口，并将新 JWT 回写到本地配置缓存。现有 Project 成员资格检查继续决定该用户可以操作的 Application、Environment 和 Deployment。

## Open questions

暂无需要用户确认的未决事项。MCP 的工具粒度、对既有 API 的具体映射、等待轮询参数和命令展示来源在 Spec / Plan 中确定，但不得改变以下已确认能力边界。

## Decisions

1. 流程模式为严格模式 / strict，依次经过 Requirement、Spec、Plan、Implementation、Verification。
2. 新功能使用独立顶层 `mcp/` 目录，以 Python + uv 实现；不将 Python 运行时嵌入 Go API 服务。
3. MCP 同时覆盖 Orbit 业务控制面与本地 Docker / Docker Compose 运行时；Docker 日志和实态是产品设计验证的核心输入，不只是诊断附属能力。
4. 正常业务部署应优先经过 Orbit 的部署模型与异步任务；直接运行时操作必须能被识别为覆盖或漂移，不能静默伪造领域状态。
5. 首版的创建、发布、部署、停止和重启全部通过 Orbit；Docker / Docker Compose 仅执行日志和运行时实态查询。直接 Compose 生命周期写操作延后评估。
6. 首版仅支持已有 Environment 中的 `kind=standard` 应用；Gateway、public TCP、证书和高级挂载延后。
7. MCP 与 Docker CLI、Pomelo Orbit 数据目录使用同一台本机/WSL Docker context；远程 Docker 另立需求。
8. 首版不增加确认步骤；所有工具返回必须展示关键业务步骤与实际命令/请求摘要，认证凭据除外。
9. MCP 不实现 Token/PAT 管理，认证配置使用 Dynaconf + 本地环境文件；操作前复用未过期 JWT，否则使用配置用户名密码登录并更新 JWT 缓存。
10. MCP 优先复用既有 Orbit HTTP API；现有接口无法表达已确认能力时，修正或补充产品实现，不以数据库直连或 Docker 生命周期写操作绕过产品模型。
11. Version 的状态仅是产品标记，MCP 不以 Version 是否 published 限制创建、编辑、发布、预览、部署、停止、重启或删除等操作。
12. 部署工具即时返回已发起操作及可获得的命令/请求摘要，并提供基于既有 Deployment 状态 API 的观察/等待能力；不等待 worker 日志才返回。
13. 部署完成后执行短时稳定性验证：在默认 60 秒观察窗口内，目标容器不得退出、进入失败状态或出现新的重启；不要求 HTTP/TCP 探活。观察窗口应可通过 MCP 本地配置调整。
14. 本地部署数据默认不脱敏；密码、JWT、Cookie 和其他 MCP 认证配置不得在工具结果、日志或异常中返回。
15. 首版暴露删除 Application、删除 Version 和停止时 `remove_volumes` 的能力，并优先调用既有接口；接口能力不足时按决策 10 修正产品实现。
16. Deployment 关联的领域 ID 必须在命令创建时确定并持久化；首次部署由命令路径创建带生成 ID 的 Service，再创建引用该 `service_id` 的 Deployment，worker 不得补写或创建这条关联。

## Risk

- MCP 所在进程可访问 Docker Socket 时具有高宿主机权限；Orbit JWT 不能替代本机 Docker 权限隔离。
- MCP、Orbit 服务和 Docker CLI 若使用不同 Docker context、WSL 发行版或数据根，会得到错误的工作目录或运行时事实。
- 未来若开放直接 Compose 写操作，可能使 Service / Deployment 数据与容器现实漂移；这是后续功能必须检测的产品风险，不能被隐藏。
- 仅以 `docker compose up -d` 进程退出码判断成功不足以证明容器健康；验证工具需显式呈现 readiness / health 的缺口。
- 现有 HTTP API 可能未覆盖 Version 编辑、预览、运行时路径解析或登录会话管理的全部契约，后续 Spec 需要列出缺口并决定后端改动。
- 本地环境文件包含用户名密码和 JWT，必须纳入 `.gitignore`，限制文件权限，且不得在日志、异常或工具响应中输出。
- JWT 缓存只按到期时间判断有效；服务端若提前拒绝 JWT，MCP 需要清除缓存并重新登录后再报告失败。
- 本地部署数据不脱敏会使 Component 环境变量、Compose 配置和容器日志可见；这是用户为本地验证场景明确接受的边界，不得扩展到远程或多租户场景。

## User review notes

- 2026-07-26：用户要求以严格模式启动本功能。
- 2026-07-26：用户指定新建 `mcp/` 目录，并使用 uv + Python 实现。
- 2026-07-26：用户明确本地 Docker / Docker Compose 操作（特别是日志）属于允许且有价值的能力；MCP 的主要价值是评估 Pomelo Orbit 产品和功能设计的合理性、正确性。
- 2026-07-26：用户确认首版生命周期操作全部通过 Orbit，Docker / Docker Compose 用于日志和排查；首版仅 standard + 既有 Environment，运行在同一本机/WSL Docker context；不增加确认步骤，但要展示关键操作和命令。
- 2026-07-26：用户指定 MCP 使用 Dynaconf + 环境文件管理配置；不投入 MCP Token。每次操作优先复用未过期 JWT，缺失或过期时使用配置用户名密码登录并缓存新 JWT。
- 2026-07-26：用户确认开发环境 Turnstile 关闭；Requirement 进入 Accepted，开始 Spec。
- 2026-07-26：用户要求回到 Requirement 明确仍会影响产品范围和验收的事项；Requirement 恢复为 Draft，Spec 暂不推进。
- 2026-07-26：将创建工具粒度、Version 生命周期、异步等待、验证深度、秘密输出、异步命令展示和破坏性能力列为待确认需求决策。
- 2026-07-26：用户确认优先复用现有接口，能力不足时修正产品实现；Version 状态不阻止操作；部署后进行短时无失败/无重启验证；本地数据不脱敏；即时展示操作；暴露删除和 volume 删除能力。Requirement 进入 Accepted。
- 2026-07-26：用户指出所有领域 ID 均在创建时生成；首次部署必须在命令阶段创建 Service 并写入 Deployment 的 `service_id`，禁止 worker 事后补写。
- 2026-07-26：用户指定 `POMELO_ORBIT_DATA_ROOT` 未配置时使用仓库相对 `data/`；配置加载必须在目录缺失时创建它。
