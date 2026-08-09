# 决策账本（现行）
最后修改时间: 2026-08-09 13:28:43

Doc role: living SoT  
说明：只记录**仍然有效**或**明确废止**的产品/技术结论。完整推导过程在 `docs/archive/specflow/`，**归档无须采信**。与代码冲突时以代码为准。

## 有效（Current）

| ID | 决策 | 备注 |
|----|------|------|
| C-01 | Version = 结构化业务规格，经 Render 生成 compose | 非 compose 全文当 Version 存储 |
| C-02 | 运行态 SoT = Service（app+env+instance） | 非 Application undeployed/deployed 状态机 |
| C-03 | Deployment = 操作流水 + 任务状态 | worker 执行 Docker |
| C-04 | Application.kind ∈ {standard, gateway}；创建后不可改 | 渲染分支 |
| C-05 | 暴露 SoT = VersionExpose（protocol + access + ports） | 无域名列 |
| C-06 | 域名 / rest 控制面在 GatewayConfig | base_domain、rest_api_url |
| C-07 | Environment = Project 下元数据 | 无 base_domain / ingress 用户 SoT |
| C-08 | 平台 Route 与应用 Expose 分流 | Route → rest；Expose → labels |
| C-09 | 单节点 API+worker；挂 Docker socket | 无独立 worker 部署角色主路径 |
| C-10 | 单 active gateway Service | 同时仅一个 deploying/running gateway |
| C-11 | instance_key 默认 default；产品约束以实现为准 | TCP 等路径曾钉死 default |
| C-12 | 后端 Go + Gin + proto + migrate；前端 web/ Vue3 | 见 architecture |
| C-13 | API 路径单数资源名 | `/api/ci/repository` |
| C-14 | 不修改已执行迁移文件 | 活跃开发期可重建库 |
| C-15 | 无兼容层 / 别名 / 新旧并存 | CLAUDE.md |
| C-16 | 文档：活 SoT vs archive；过程库仅进行中任务 | 本整理任务 |

## 废止（Superseded）

| ID | 旧结论 | 替代 |
|----|--------|------|
| X-01 | compose 全文当 Version 存储 | C-01 |
| X-02 | Application 状态机 undeployed/deployed 作运行 SoT | C-02 |
| X-03 | application_config_file 包模型为唯一规格 | C-01 + Component |
| X-04 | application_route / Application 级 domain+port 长期 SoT | C-05/C-06/C-08 |
| X-05 | attach_ingress / is_ingress | C-04/C-05 |
| X-06 | EnvironmentBinding 用户 SoT | C-05/C-06 |
| X-07 | 全局 env `traefik.api_url` / `domain_suffix` 产品路径 | C-06 |
| X-08 | Python/FastAPI 后端架构文档 | C-12；归档 designs |
| X-09 | 独立 pomelo-orbit-worker 多角色部署 | C-09 |
| X-10 | 以 code==traefik 魔法代替 kind | C-04 |

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
