# Orbit 部署能力补充规格
最后修改时间: 2026-07-26 17:07:02

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-orbit-deployment-capability-gaps.md` (`Accepted`)
- Flow mode: 严格模式 / `strict`
- Downstream consumer: `docs/requirement/20260726-ragflow-split-deployment.md`

## Overview

本规格在现有 Application Version / Component 模型中增加受管、结构化的运行时能力：重启策略、tmpfs、ulimit 和基于 Credential 的环境变量引用。字段从 Version CRUD 经 Preview、Compose 渲染、Deployment、受管 Docker 诊断和 MCP 工具保持一致。

原有 `runtime_config` 继续兼容既有非秘密占位符，但不得作为本规格的秘密注入通道。新秘密值仅在部署进程内短暂解密，写入 Service workspace 内权限为 `0600` 的组件 env 文件；Version、Deployment options、Preview、MCP 响应、日志与验证证据均不得包含原始值。

## Parallel development contract

任务一与任务二可并行进入 Implementation。任务一拥有通用运行时字段、`runtime_env` Credential、Compose 渲染、秘密物化、脱敏及 MCP 工具契约的实现；任务二拥有 RAGFlow 的非秘密目标状态、镜像与健康检查调研、受管部署 Runbook 和脱敏 payload fixture。

任务二在任务一完成前不得创建实际 Credential、Application、Version 或 Deployment，也不得向共享 MCP 模块写入临时兼容层。双方以 Environment-free 的 `application_id + instance_key` 契约为共同基线。任务一提供稳定的组件字段和 MCP 工具契约后，任务二用其脱敏 fixture 进行集成核对；真实运行时写入和部署仍须等待任务一 Verification 完成。

## Design decisions

### D1. Component runtime schema

`VersionComponent` 新增以下可空字段；缺失或空值表示维持当前行为：

| 字段 | JSON / 枚举 | Compose 结果 | 限制 |
|---|---|---|---|
| `restart_policy` | `"no"` 或 `"unless-stopped"` | `restart` | 默认 `no`，不支持任意 Compose policy 字符串 |
| `tmpfs_json` | `[{"target":"/tmp","size_bytes":536870912,"mode":"1777"}]` | `tmpfs` | 最多 8 项；绝对容器路径、无 `..`、禁止 `/`、`/proc`、`/sys`、`/dev`；大小 1 MiB 至 8 GiB；mode 为 3 至 4 位八进制字符串 |
| `ulimits_json` | `[{"name":"memlock","soft":-1,"hard":-1}]` | `ulimits` | 最多 8 项，name 仅 `memlock`、`nofile`；仅 `memlock` 可使用 `-1`；否则 `0 <= soft <= hard` |
| `secret_env_refs_json` | `[{"env_key":"MYSQL_PASSWORD","credential_id":"...","data_key":"MYSQL_PASSWORD"}]` | 组件 `env_file` | 最多 64 项；一个 `env_key` 只能出现一次 |

`restart_policy` 用受限双值覆盖任务二的自动恢复需要；不支持 `always`、`on-failure` 和重试计数，避免引入未定义的故障重试语义。

### D2. Credential contract

Credential 领域新增 `runtime_env` 类型。其解密后的 `data` 必须为 JSON object，键名匹配环境变量标识符，值必须为非空 string。它沿用现有 `encrypted_data` 加密存储、Project 成员资格和 CRUD 权限模型。

`secret_env_refs_json` 在 Version 创建、更新和发布时必须验证：

1. `credential_id` 存在、类型为 `runtime_env`，且与 Application 同属同一 Project。
2. `data_key` 在解密后的 Credential data 中存在；`env_key` 和 `data_key` 均为合法环境变量名。
3. 同一 Component 内不存在重复 `env_key`，也不得覆盖该 Component 或 Version 的非秘密 `env_json` 键。

Version 只持久化引用 ID 和键名，不持久化秘密值。删除或修改被 Version 引用的 Credential 必须被拒绝，或在同一事务中明确处理引用关系；实现选择拒绝删除，以保持已发布 Version 的可复现性。

### D3. API and MCP contract

Application Version 的 HTTP create/update/read/preview 组件对象增加 D1 的四个字段。响应只返回结构和 Credential ID，不返回 Credential data。

MCP 的 `orbit_bootstrap_application`、`orbit_create_version`、`orbit_update_version`、`orbit_get_version`、`orbit_list_versions` 和 `orbit_preview_version` 透传同名组件字段，且必须对 `secret_env_refs_json` 保持结构化数组。

MCP 新增以下受限 Credential 工具：

1. `orbit_list_runtime_env_credentials(project_id)`：仅返回 ID、名称、类型和创建时间。
2. `orbit_create_runtime_env_credential(project_id, name, values)`：创建 `runtime_env` Credential；响应不得回显 `values`。
3. `orbit_update_runtime_env_credential(credential_id, name?, values?)`：更新元数据或秘密；响应不得回显 `values`。

不新增通用的 Credential 导出或读取秘密工具。HTTP 已有 Credential 访问权限不在本任务改变范围；本任务新增的 Application、Deployment 和 MCP 响应都不得附带解密数据。

### D4. Rendering and deployment

`restart_policy`、`tmpfs_json` 和 `ulimits_json` 经集中解析、验证后直接渲染为对应 Compose 字段。Preview 与实际部署必须调用同一解析和渲染代码。

对于每个含 `secret_env_refs_json` 的组件：

1. Deployment worker 在 Project 权限和 Version 结构已验证后解密所引用的 `runtime_env` Credential。
2. worker 在该 Service workspace 内原子写入 `.runtime/<component>.env`，文件和父目录权限为 `0600` / `0700`。
3. 渲染 Compose 时仅添加相对 `env_file: .runtime/<component>.env`，非秘密环境变量仍放在 `environment` 中。
4. Docker Compose 读取 env file 并把值注入容器；Deployment options、`command_text`、worker 日志和错误摘要不得写入文件内容。
5. 重新部署或 Credential 轮换后原子替换该 env file；停止、重启和预览不解密或重写秘密。

Preview 不读取 Credential data。它以稳定的脱敏占位值表示秘密环境变量，因此可校验 Compose 结构但不能恢复值。

### D5. Redaction boundary

新增统一的 `SecretRedactor`，输入为 Version 中的秘密环境键和仅在内存中保留的实际值。它必须：

1. 将 YAML / JSON / map 中的秘密环境键值替换为 `[REDACTED]`。
2. 将 `KEY=value` 形式的 inspect 环境变量替换为 `KEY=[REDACTED]`。
3. 在返回的文本日志、Compose CLI 输出、错误摘要和验证 evidence 中替换已知实际秘密值。
4. 在 HTTP preview、Deployment detail / logs、MCP `runtime_compose_config`、`runtime_container_inspect`、`runtime_compose_logs`、`orbit_deployment_logs` 与 `verify_deployment` 的所有成功和错误路径使用同一规则。

原始 Compose 文件、env file 和容器进程环境仍可被 Docker daemon 所在机器的高权限用户读取；这是本地 Docker 的既有信任边界，不能由 MCP 逻辑隔离。

### D6. Persistence and compatibility

开发阶段重建数据库，直接修改 SQLite 与 MySQL 现有 `000023_application` 建表 / 清理脚本：为 `version_component` 增加 `restart_policy`、`tmpfs_json`、`ulimits_json` 三列，并建立 VersionComponent 到 Credential 的显式引用表：`version_component_secret_env_ref`。引用表保存 `component_id`、`env_key`、`credential_id`、`data_key`，以唯一键约束 `(component_id, env_key)`。不新增增量 migration，也不维护历史库升级路径。

持久化实现以引用表为事实来源；`secret_env_refs_json` 仅是 HTTP/MCP 传输形态，不在 `version_component` 重复存储。`tmpfs_json` 和 `ulimits_json` 保留在 component 表，因其是组件自身不可拆分的规范。现有 Version 的新列均为 `NULL`，渲染结果保持不变。

Credential 删除引用检查扩展为 Repository 和 VersionComponent secret-ref 两类引用。更新 Credential data 后现有 Service 不自动重启；下次 deploy/restart 前必须重新解析引用，任务二 Runbook 负责安排轮换后的滚动部署。未设置新增字段的 Component 继续采用既有 Compose 输出。

### D7. Explicit exclusions

本规格不支持 named volume 顶层声明、`extra_hosts`、Compose `include`、`profiles`、`env_file` 用户直传、自由格式 runtime 片段、跨 Application `depends_on` 或健康编排。任务二继续使用逻辑目录挂载与 MCP 受管部署顺序。

## Affected components

| 层次 | 预期变化 |
|---|---|
| model / dto | VersionComponent 运行时字段、秘密引用 DTO |
| application domain | Version 校验、Credential 引用校验、Credential 删除保护 |
| repository / SQL | component 字段、秘密引用表、SQLC 查询和引用计数 |
| HTTP | Version binding、response、Preview / Deployment 脱敏、Credential 类型校验 |
| deployment | Compose renderer、workspace env-file 写入、worker 秘密解析、日志脱敏 |
| MCP | Version 字段映射、runtime Credential 工具、所有运行时读工具脱敏 |
| tests | 向后兼容、校验、渲染、文件权限、权限边界和泄漏回归测试 |

## Technical questions

1. `restart_policy=unless-stopped` 是任务一必须交付的上线行为。
2. 向量数据库仍由任务二选择；任务一交付通用、受限的 `tmpfs` / `ulimits` 能力，不依赖该选择。
3. 现有 Credential detail/export API 对有权限的 Project 成员仍可读解密数据；收紧该既有行为属于独立的 Credential 安全任务。
4. `runtime_env` Credential 的轮换由任务二 Runbook 触发重新部署；任务一不产品化轮换编排。

## Alternatives considered

1. 将秘密继续作为 `runtime_config`：拒绝，因为 Deployment options 会持久化明文。
2. 将秘密直接写入生成的 `docker-compose.yml`：拒绝，因为 Preview、Compose config 和工作目录更容易泄漏。
3. 接受任意 Compose JSON/YAML 片段：拒绝，因为会绕过模型校验和 MCP 的受管边界。
4. 将所有运行时字段放入单一 `runtime_json`：拒绝，因为校验、迁移和兼容语义不透明。

## Risks

1. Component secrets 的引用校验会让 Application 领域依赖 Credential 查询能力，需维持 Project 成员和错误转换边界。
2. Docker Compose 在不同版本和 Docker Desktop / Linux context 上对 `tmpfs`、`ulimits` 的实际行为可能不同，必须在 Verification 阶段显式验证。
3. 任何遗漏的响应、日志或 runtime 诊断路径都可能泄漏秘密，需以注入唯一测试值的端到端回归覆盖。
4. env file 位于本地 workspace，文件权限只降低同机低权限读取风险，不等同远程密钥管理系统。

## User review notes

用户于 2026-07-26 要求进入 Plan / 计划阶段，视为接受本规格及 Technical questions 中记录的决策。

用户于 2026-07-26 要求任务一与任务二并行开发；并行边界和集成门槛按本规格的 Parallel development contract 执行。
