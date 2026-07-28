# Orbit 部署能力补充计划
最后修改时间: 2026-07-26 17:07:02

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-orbit-deployment-capability-gaps.md` (`Accepted`)
- Spec: `docs/spec/20260726-orbit-deployment-capability-gaps.md` (`Accepted`)
- Flow mode: 严格模式 / `strict`
- Downstream consumer: `docs/requirement/20260726-ragflow-split-deployment.md`
- Environment baseline: `docs/requirement/20260726-remove-environment.md` 由独立 agent 实施；其 Environment-free MCP / Deployment 契约已完成复核，Task 1 不负责该迁移。
- Parallel peer: `docs/plan/20260726-ragflow-split-deployment.md` 与本任务并行开发，只产出不含秘密的 RAGFlow 目标状态、payload fixture 和 Runbook，不能修改本任务拥有的 MCP / Deployment 实现。

## Implementation steps

### Step 0: 基线与生成工具链

1. 记录当前 `git status`，保留用户已有的 deployment、web 等开发改动，不回退或格式化无关文件。
2. 核对 `proto/orbit/v1/application/version.proto`、根 `buf.gen.yaml`、`web/buf.gen.yaml`、`Taskfile.yml` 和 `sqlc.yaml`；源 Proto 是唯一修改点，生成代码通过 `task proto` 和 `task sqlc` 更新。
3. 运行目标化 Go 与 MCP 基线测试；若基线失败，记录后继续区分本任务回归，不修改无关实现。
4. 以已复核的 Environment-free MCP / Deployment 契约为基线；Task 1 仅在该基线之上添加运行时能力，不能恢复 `environment_id` 参数或旧 workspace 布局。若该改动尚未合入当前分支，按实际 diff 合并并运行全量受影响测试，不覆盖独立任务的文件。
5. 与 Task 2 共享仅包含字段名、合法枚举、脱敏占位和 MCP 工具签名的契约；不共享 Credential value、实际 Project ID、Docker 路径或运行资源 ID。Task 2 的 RAGFlow fixture 只能在 Step 3、Step 4 和 Step 6 的集成测试中作为非秘密输入使用。

### Step 1: Schema、模型和 SQLC 持久化

1. 开发阶段重建数据库，直接修改 SQLite / MySQL 现有 `000023_application` 建表 / 清理脚本：
   - 在 `version_component` 增加 nullable `restart_policy`、`tmpfs_json`、`ulimits_json`。
   - 创建 `version_component_secret_env_ref`，包含 component ID、环境变量键、Credential ID、Credential data 键、唯一约束和必要索引。
   - 不新增 `000031` 或任何增量 migration，不编写历史数据库兼容或升级逻辑。
2. 更新 `internal/model/application.go`，新增 Component 运行时字段及 `VersionComponentSecretEnvRef` 模型。
3. 扩充 `sql/query/application/version.sql`：Component 读写包含新列；Version Component 替换操作原子删除并重建秘密引用；提供按 component / version 读取引用、按 Credential 检查引用的查询。
4. 执行 `task sqlc`，更新 `internal/gen/sqlc/**`；调整 application 和 credential repository 接口与 SQLC 实现。
5. 未设置新增字段的 Component 保持既有 Compose 输出；开发数据库以修改后的 schema 重建，不处理历史数据。

### Step 2: Application 与 Credential 领域契约

1. 在 Application DTO 中为 Component 新增 `restart_policy`、`tmpfs_json`、`ulimits_json` 和结构化 `secret_env_refs` 输入/输出；不把秘密值映射进 model 或 Version response。
2. 建立集中校验器：
   - restart 只接受未设置、`no`、`unless-stopped`；
   - tmpfs 校验条数、目标路径、大小和 mode；
   - ulimit 校验名称白名单及 soft/hard 关系；
   - secret ref 校验字段格式、重复 env key 和非秘密环境变量冲突。
3. 扩展 Credential type 校验支持 `runtime_env`，对其数据解密后验证 JSON string map；保留其他既有 Credential 类型的行为。
4. 为 Version create/update/publish 注入受限 Credential 读取能力，校验 Credential 类型和 Project 一致性，并持久化引用表；将引用列入 Credential 删除保护。
5. 保留现有 Credential detail/export 授权语义；新增或既有 Application/Deployment 读模型绝不返回解密值。

### Step 3: HTTP、Proto 与生成客户端

1. 修改 `proto/orbit/v1/application/version.proto` 中的 Component request/response 字段；按既有 JSON-string 约定承载 tmpfs / ulimits，秘密引用使用明确结构而非自由 YAML。
2. 更新 Application bundle mapper、Version handler/binding/response，使 create、update、import、get、list、export 与 preview 在不含秘密值的前提下完整往返新字段。
3. 使用 `task proto` 更新 `internal/gen/proto/**` 和 `web/src/gen/proto/**`；不手改生成文件。
4. 根据当前 HTTP Credential 路由复用受限 `runtime_env` Create/Update/List 语义；如需单独响应模型，仅新增不回显 `data` 的 endpoint，不改变通用 export 行为。

### Step 4: Compose 渲染和秘密物化

1. 在 deployment 领域集中解析 Component runtime 字段，复用到 Preview 和实际 render：只为显式 `unless-stopped` 输出 `restart`；输出结构化 tmpfs、ulimits。
2. 扩展 Render input/result，携带每个组件的秘密引用元数据、预览占位符和部署期 env-file 材料；禁止让明文进入 `RuntimeConfig`、`OptionsJSON` 或 `command_text`。
3. 在部署 worker 中按 Version 引用读取并解密 `runtime_env` Credential，原子写入 Service workspace `.runtime/<component>.env`，确保目录 `0700`、文件 `0600`。
4. Compose 对相应组件只渲染相对 `env_file`；Preview 只输出稳定占位符且不读取 Credential data。
5. 重启 / 重新部署前重新解析引用；停止和 Preview 不修改 env file。所有失败路径删除临时文件且不把内容写入错误、日志或 Deployment 记录。

### Step 5: 统一脱敏与受管诊断

1. 引入可递归处理 map、YAML、JSON、`KEY=value` 和文本的 `SecretRedactor`，由秘密环境键及部署期内存值驱动。
2. 在 HTTP Preview、Deployment detail/logs 和任何 Compose 错误输出应用脱敏；当读取路径需要实际值时只在内存短暂解密。
3. 在 MCP `OrbitClient`、runtime tools、verification evidence 中调用同一脱敏规则，覆盖 `runtime_compose_config`、`runtime_container_inspect`、`runtime_compose_logs`、`orbit_deployment_logs` 与 `verify_deployment`。
4. 保持 Docker Runtime 命令白名单和受管目标解析不变；不新增 Docker 生命周期写命令或任意路径读取。

### Step 6: MCP 受限 Credential 与 Version 支持

1. 扩展 `OrbitClient` 的 Version Component 请求映射和响应模型；写操作的 `request_summary` 只保留秘密引用 ID / key，不保留值。
2. 在 `tools/orbit.py` 增加 `orbit_list_runtime_env_credentials`、`orbit_create_runtime_env_credential`、`orbit_update_runtime_env_credential`；响应只含元数据和 operation/resource IDs。
3. 更新 MCP server 工具清单、README 和 `.env.example`（仅在确有新增配置时）；不把 Credential data、JWT、CSRF 或 Compose env 文件内容写入文档。
4. 为新能力补齐单元测试，确保成功、认证失败、HTTP 错误和重试路径均不泄漏秘密。

### Step 7: 测试与实现阶段收尾

1. Go 单元 / 集成测试覆盖新字段校验、迁移、repository 原子替换、Project 边界、Credential 删除保护、旧 Version 兼容和 Proto/HTTP 往返。
2. Compose renderer / worker 测试覆盖 `restart`、tmpfs、ulimits、Preview 占位、env-file 原子写入、权限、轮换后重新部署及失败清理。
3. 注入唯一秘密测试值，断言 HTTP 响应、MCP 响应、Compose/inspect/logs、Deployment error/log、verification evidence 均不出现该值。
4. MCP pytest 覆盖 Credential 工具不回显 values、Version 字段映射与运行时读工具脱敏。
5. 运行项目约定的 `task proto`、`task sqlc`、目标化 `go test`、`go vet`、MCP `make -C mcp check`；Docker 标记测试仅在相同 Docker context 已准备并经明确启用时运行。
6. 实现完成后检查实际 diff 仅覆盖本 Plan，汇报已运行检查、未运行的 Docker 验证和残余本机 Docker 信任边界，停在 Implementation 等待进入 Verification。

## Files to change

### 必改

- `sql/migration/sqlite/000023_application.{up,down}.sql`
- `sql/migration/mysql/000023_application.{up,down}.sql`
- `sql/query/application/version.sql`
- `internal/gen/sqlc/**`（`task sqlc` 生成）
- `internal/model/application.go`
- `internal/repository/application.go`
- `internal/repository/credential.go`
- `internal/repository/impl/sqlc/application/repository.go`
- `internal/repository/impl/sqlc/credential/repository.go`
- `internal/application/application/dto/application.go`
- `internal/application/application/usecase/version.go` 及其测试
- `internal/application/credential/usecase/service.go` 及其测试
- `internal/application/deployment/{port,usecase}/**` 中的渲染、执行、workspace、日志路径和测试
- `proto/orbit/v1/application/version.proto`
- `internal/gen/proto/**`、`web/src/gen/proto/**`（`task proto` 生成）
- `internal/api/http/handler/application/**`
- `internal/api/http/handler/deployment/**`
- `mcp/src/pomelo_orbit_mcp/{orbit_client.py,server.py,verification.py}`
- `mcp/src/pomelo_orbit_mcp/tools/{common.py,orbit.py,runtime.py,verification.py}`
- `mcp/tests/**`

### 可能触及

- `proto/orbit/v1/credential/credential.proto`
- `internal/api/http/handler/credential/**`
- `internal/api/http/routes/credential.go`
- `cmd/**` 与 service wiring：仅当 Application/Deployment 服务新增的 Credential port 需显式装配时修改
- `mcp/README.md`、`mcp/.env.example`
- `docs/guides/**`：仅补充使用新秘密引用能力所必需的安全边界

### 预期不改

- `web/**` 的手写业务组件、页面和样式
- RAGFlow 源码或 `D:\SourceCodes\opensource\ragflow` 下文件
- named volume、`extra_hosts`、Compose include/profile、任意 Docker CLI 写命令
- 既有 Credential detail/export API 的授权语义

## Verification plan

1. 未设置新字段的 Version 在 Preview 和 deploy 渲染中保持既有结果。
2. HTTP 与 MCP 能完整创建、读取、更新和发布 `restart_policy`、tmpfs、ulimits、秘密引用；无效路径、越界限制、未知 ulimit、跨 Project Credential 和环境变量冲突均被拒绝。
3. `runtime_env` Credential data 始终加密存储；Version、Deployment options、Preview、MCP request summary / response 均不含其值。
4. Worker 生成的 env file 权限正确，重启或重部署重新解析轮换后的值，Preview / stop 不修改该文件。
5. Compose Preview/实际 deploy 分别正确输出 `restart: unless-stopped`、ES 所需 tmpfs、memlock ulimit；Docker 可用时在目标 context 验证实际容器配置。
6. 在全路径注入唯一测试秘密，断言 HTTP、MCP、Compose config、inspect、日志与 verification evidence 均已脱敏。
7. `task proto`、`task sqlc`、`go vet ./cmd/... ./internal/... ./sql`、相关 `go test` 和 `make -C mcp check` 通过；Docker 集成测试的运行状态单独记录。

## Blockers

无代码层阻塞。Docker `tmpfs` / `ulimits` 的真实行为依赖目标 Docker context；在该 context 不可用时，只能完成单元和渲染验证，不能声称完成运行时验证。

## Assumptions

1. `restart_policy=unless-stopped` 为任务一必须交付的受限自动恢复行为。
2. 任务一实现通用 tmpfs / ulimit 能力；向量引擎及具体参数由任务二确定。
3. `runtime_env` Credential 轮换后的重新部署由任务二 Runbook 负责。
4. Credential detail/export 的既有 Project 成员访问模型在本任务中不变。
5. 环境移除的 MCP 调整由独立任务完成；Task 1 触及相同 MCP 模块时必须基于其合入后的实现合并。
6. 任务二可与本任务并行准备 RAGFlow desired-state 工件，但实际 MCP 写入和部署必须等待本任务 Verification；该门槛不阻塞任务一实现。

## Risks

1. SQLC、Proto 生成物和运行时领域改动跨越多层，任何一层漏同步会造成 HTTP、MCP 或 worker 行为不一致。
2. 写入 Compose env file 后，同机高权限用户仍可读取；此任务只能防止经 Orbit/MCP 常规响应和日志泄漏。
3. 脱敏依赖引用键和实际值的完整收集，必须以唯一秘密值的回归测试覆盖所有错误路径。
4. 现有用户工作区包含并行开发的 deployment / web 改动，实施时必须逐文件合并并避免覆盖。
5. 环境移除与任务一都会修改 Deployment/MCP 边界；未先合入前置任务会造成参数、workspace 和测试冲突。
6. 并行开发时，任务二的 fixture 与任务一的 MCP / Version 契约可能漂移；两者合并前必须以任务一的组件字段映射、Preview 和脱敏测试共同核对，不能靠人工修改实际部署资源修复。

## Rollback

1. 开发环境回滚时还原既有建表脚本并重建数据库；不维护增量 migration 或历史数据回滚路径。
2. 回滚前先停止使用新 Component runtime fields 和 `runtime_env` Credential 引用的 Version；保留现有 Service 的最后成功 Version。
3. 回滚不会删除 Service workspace 的 `.runtime` 文件，需以受管停止或受限的人工运维流程清理，避免误删仍在运行的容器所需 env file。

## User review notes

用户于 2026-07-26 要求任务一与任务二并行开发；本 Plan 将任务二限定为非秘密工件和集成准备，真实运行资源仍由任务一 Verification 保护。

用户于 2026-07-26 要求开始 Implementation / 实现阶段，视为接受本 Plan。用户已明确开发阶段重建数据库；计划改为就地更新既有 schema 脚本，不新增增量 migration 或历史兼容路径。
