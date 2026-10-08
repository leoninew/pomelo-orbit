# 决策账本（现行）
最后修改时间: 2026-10-08 11:39:56

Doc role: living SoT
说明：只记录**仍然有效**或**明确废止**的产品/技术结论。完整推导过程在 `docs/archive/specflow/`，**归档无须采信**。与代码冲突时以代码为准。

## 有效（Current）

| ID | 决策 | 备注 |
|----|------|------|
| C-01 | Version = 结构化业务规格，经 Render 生成 compose | 非 compose 全文当 Version 存储 |
| C-02 | 运行态 SoT = Service（app+env+instance） | 非 Application undeployed/deployed 状态机 |
| C-03 | Deployment = 操作流水 + 任务状态 | worker 执行 Docker |
| C-04 | Application.kind 明确区分 `standard` 与 `gateway` | Gateway 创建写入 `gateway`，已有类型按 `gateway_config` 真实关联纠正；服务入口依据应用类型禁用 Gateway 部署。GatewayConfig 保存配置，Environment binding 校验归属，Gateway 从专属入口部署 |
| C-05 | 暴露 SoT = VersionExpose（protocol + access + ports） | 无域名列 |
| C-06 | 域名 / REST 控制面在 GatewayConfig | `internal_domain` 用于 Docker labels 和 Gateway 派生地址；`external_domain` 仅辅助自定义 Route 表单拼接完整域名；另有 `rest_api_url` |
| C-07 | Environment = Project 下元数据 | 无域名 / ingress 用户 SoT |
| C-08 | 平台 Route 与应用 Expose 分流 | Route → 独立 File provider 文件；Expose → Docker labels |
| C-09 | 单节点 API+worker；挂 Docker socket | 无独立 worker 部署角色主路径 |
| C-10 | 单 active gateway Service | 同时仅一个 deploying/running gateway |
| C-11 | instance_key 默认 default；产品约束以实现为准 | TCP 等路径曾钉死 default |
| C-12 | 后端 Go + Gin + proto + migrate；前端 web/ Vue3 | 见 architecture |
| C-13 | API 路径单数资源名 | `/api/ci/repository` |
| C-14 | 默认不修改已执行迁移文件 | 活跃开发期可重建库；Repository/Repository Credential 全局共享任务按用户明确要求就地修订既有迁移，已运行数据库须另行转换或重建 |
| C-15 | 无兼容层 / 别名 / 新旧并存 | CLAUDE.md |
| C-16 | 文档：活 SoT vs archive；过程库仅进行中任务 | 本整理任务 |
| C-17 | Route 显式按范围发布持久化文件，与 Service 部署独立 | 普通编辑/证书先保存业务数据，启停为前端草稿；预览发布/撤销/跳过清单并冻结 ID、业务与受管文件依赖修订，不比较运行时差异。确认逐条保存和发布，失败继续且报告具体原因及恢复失败状态。Service deploy/restart 不获取 Route 同步锁，不检查或恢复自定义 Route，也不受发布失败或 pending 阻止；Gateway 覆盖受管 provider/mount 并重建容器，仅检查自身 API 就绪，不携带业务草稿。Traefik 重启直接加载文件。现存 REST Gateway 线下告知重新部署后显式同步，无专用 UI 或旧快照转换；未知文件保留 |
| C-18 | MCP 仅采用本地 stdio 与内部 actor 绑定 | Web Dialogue 为每 turn 建立固定 actor 的内存 Core；Grok/Codex stdio 用显式 MCP PAT `POMELO_ORBIT_MCP__ACCESS_TOKEN`，每次 tools/call 认证；无远程 /mcp |
| C-19 | Project 环境/Gateway 初始化只走 Web Wizard；MCP 使用 connection-local 已就绪 Project scope | Project 创建和 identity seed 不预建 Environment/Gateway；`ProjectInitializationConfig` 只给 Wizard 初值；运行时只读库存；Grok/Codex 先 list/select，后续工具不再传 `project_id` |
| C-20 | Repository 与 Repository Credential 全局共享，Project 只作成员校验上下文 | 仓库 `code`、凭据名称全局唯一；Application Pipeline、Run、Artifact 和 Environment 仍按 Project 归属；仓库删除检查所有现存应用流水线绑定，应用流水线删除检查其未结束 Run；历史 Run/Snapshot/Artifact 不直接阻止仓库删除 |
| C-21 | Web CI/CD 日志统一组件、功能与样式，采用 Fetch + SSE | 全部来源运行中持续读取；文件按字节续读，容器跟随并按时间重叠恢复；关闭停止订阅并保留页面缓存；不提供复制/导出，部署终态不结束容器日志，stop 仍不展示容器日志 |

## 废止（Superseded）

| ID | 旧结论 | 替代 |
|----|--------|------|
| X-01 | compose 全文当 Version 存储 | C-01 |
| X-02 | Application 状态机 undeployed/deployed 作运行 SoT | C-02 |
| X-03 | application_config_file 包模型为唯一规格 | C-01 + Component |
| X-04 | application_route / Application 级 domain+port 长期 SoT | C-05/C-06/C-08 |
| X-05 | attach_ingress / is_ingress | C-05/C-08 |
| X-06 | EnvironmentBinding 用户 SoT | C-05/C-06 |
| X-07 | 全局 env `traefik.api_url` / `domain_suffix` 产品路径 | C-06 |
| X-08 | Python/FastAPI 后端架构文档 | C-12；归档 designs |
| X-09 | 独立 pomelo-orbit-worker 多角色部署 | C-09 |
| X-10 | 以名称、code 或镜像推断 Gateway 身份 | C-04：明确的 Application.kind + GatewayConfig 配置及 Environment binding |
| X-11 | Route 普通写操作立即发布，或同步时覆盖整个 REST provider | C-17；保存与发布分离，只处理确认范围内的独立文件，未知资源保留 |
| X-12 | MCP 浏览器 grant、loopback callback、token file、HTTP /mcp Bearer transport | C-18；显式进程凭据与内存 Core |
| X-13 | Project 创建时自动预建 local Environment，空库预置 Gateway/Service | C-19；Project 与 membership 创建后由 Web Wizard 配置 Environment、Probe 与 Gateway |
| X-14 | local Environment 工作目录来自 `workspace.deployment` 或运行时 YAML 回退 | `Environment.workspace_root` 为已保存的唯一工作区根；当前 CI 只在 local 执行，CD 按 local/ssh 目标执行 |
| X-15 | Python 数据库重置、SQLite 复制、原生备份及 Service / Environment JSONL 传输命令 | 2026-10-08 移除 `database-ops`、`database-transfer` 与对应 skill、测试和 dbtalk 依赖；保留 `pomelo-orbit-cli` 框架供后续整合脚本。相关过程文档归档至 `docs/archive/specflow/`，[旧操作指南](../archive/guides/service-data-transfer.md)归档至 `docs/archive/guides/`；Project CD 接管 API / Web 继续使用现有业务实现 |
| X-16 | `manage.py` 的手工升级、Docker 操作、旧目录备份和日志清理 | 2026-10-08 移除旧脚本，仅保留 CLI `remote ssh/exec/scp/tunnel`；连接配置只需 `SSH_HOST`、`SSH_USER`。已完成的[工作区改造过程文档](../archive/specflow/verification/20260818-configurable-ci-cd-workspaces.md)归档，当前受管部署见 [迁移指南](../guides/orbit-self-migration.md) |

## Backlog（非本账本承诺交付）

| 项 | 说明 |
|----|------|
| 用户自管 external Traefik | 非主线必做 |
| 自定义/自签 PEM 产品化 F1 | 可选 |
| 多 gateway / K8s | 可选 |
| sqlx 全域迁 sqlc / DTO 拆分等 | 以进行中 SpecFlow 任务为准，完成后回写本账本与 architecture |
| 删除路径防御规则整理 | 审视通用 Application 的 Version 引用预检、Credential/Repository 的引用冲突分支、Pipeline 阶段依赖校验和 Gateway 网络预检；明确哪些是核心领域约束，哪些应移除重复或异常兜底。当前网关删除任务不处理。 |

## 文档治理

| 规则 | 说明 |
|------|------|
| 实现默认阅读 | `docs/README.md` → `docs/INDEX.md` → product/architecture/decisions |
| 归档 | `docs/archive/**` **无须采信**，不得作实现依据 |
| SpecFlow | 新任务写主树 `requirement/spec/plan/verification`；闭环后归档进 `archive/specflow` |
| 冲突 | **以代码为准** 回写活文档 |
