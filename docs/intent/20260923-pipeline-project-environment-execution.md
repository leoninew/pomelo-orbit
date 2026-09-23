# CI/CD 共用 Project Environment 与流水线远端运行
最后修改时间: 2026-09-23 11:43:58

Review status: Accepted

Mode: standard

## Background

Project 创建时只保存项目与成员关系；Web 初始化向导保存唯一 Environment、完成最新 Probe，再创建 Gateway 和默认 Service。Environment 有显式 `local | ssh` target、工作区根目录与修订号，SSH 还绑定受管密钥和 host-key fingerprint。Gateway 待部署不等于 Environment 未配置。

CD 已按 Project Environment 分发到控制面本地或 SSH 目标，排队的 Deployment 保存目标和凭据修订并在执行时复核。CI Run 当前也查询同一个 Project Environment，但只接受 `local`：触发时解析控制面工作区，Worker 用本地 Docker CLI 执行，并在本地读写日志和制品。因而初始化了 SSH Environment 的 Project 虽可配置 CD，却无法运行自己的流水线。Web 的环境管理入口当前仍在“持续部署”导航中，初始化向导又把 Environment Probe 和 Gateway 创建串为一个 Project `ready` 状态；这两处呈现方式掩盖了环境实际由 CI/CD 共用的归属。当前限制见 `docs/guides/ci-pipeline-design.md`，初始化与环境模型见 `docs/product/cd-model.md`。

较早的 `20260902-remote-ssh-deployment-environments` 等过程记录包含“Project 创建时预建 local Environment”的历史目标；它已由 `20260910-project-environment-configuration` 与现行产品模型替代，不作为本需求的前提。历史过程正文不在本任务中改写。

## Goal

1. Environment 是 CI/CD 共用的 Project 执行目标；每个 Project 仅有一个 Environment。环境管理从持续部署导航移至系统管理中的独立“环境”路由与入口，继续操作当前 Project 的配置与 Probe，不另建 CI Environment。
2. CI 运行和 CD 部署前均检查所属 Project 的环境就绪情况：Environment 已保存、工作目录有效且最新目标修订的 Probe 成功，SSH 另需有效受管凭据与 host-key fingerprint。Web 在不就绪时重定向到该 Project 的初始化页面；后端/API/MCP 仍独立拒绝不就绪操作，不能仅依赖前端守卫。
3. 环境就绪与 CD Gateway/default Service 就绪分别判断：CI 不以 Gateway 为前置条件，CD 继续检查自己的 Gateway/Service。现有初始化页面继续引导 Environment → Probe → Gateway；CI 不因缺少 Gateway 而被错误重定向。
4. 应用流水线按所属 Project Environment 运行：`local` 在控制面本地执行；`ssh` 在远端执行，不把远端路径当成本地 Docker 挂载源，也不隐式回退本地。先完成并验证 Windows native OpenSSH + WSL2 Docker Desktop，再完成并验证 Linux OpenSSH + Docker。
5. SSH Run 的 Stage 日志和文件制品留在目标主机的受管工作区，不回传控制面；日志增量读取、制品元数据、清理与失败反馈均按原 Run 的目标处理。排队后目标/凭据变更时不得将旧 Run 投递到新目标。
6. 保持本地目录型 Repository 在现有 local 环境下的路径与只读 `/source` 挂载行为；不增加远端源码传输，也不改变变量、DAG、Snapshot 或制品到 Version 的关联语义。

## Non-goal

- 不建设全局 Environment 池、跨 Project 共享绑定、第二个 CI Environment 或全局 Runner 池；不在本需求内处理多个 Project 配置相同 Docker target 的协同写入。现有 target 冲突校验保持不变。
- 不把“放进系统设置”解释为嵌入现有 `/settings` 页；环境保留独立路由，也不因菜单迁移自动改成全局设置权限。
- 不要求 Gateway 已部署才能运行 CI；不重做 CD 的 Compose/Route/Gateway，不增加两个独立的初始化向导。
- 不迁移已有 Run 的本地日志/制品，不实现远端文件制品回传或下载，不增加本地目录 Repository 的远端源码传输与镜像仓库凭据管理。
- 不把“保持本地目录 Repository 当前行为”解释为将控制面本地绝对路径作为 SSH 目标的 `/source` 挂载，也不在 SSH 不可用时退回本地执行。

## User scenarios

1. 项目成员从系统管理的独立“环境”入口打开当前 Project 的 local/SSH 环境配置，完成保存和 Probe；持续部署菜单不再有重复入口，切换 Project 后看到对应环境。
2. CI 触发/重试或 CD 部署前，若 Environment 缺失、工作目录无效或 Probe 非当前成功结果，Web 重定向到该 Project 的初始化页面；绕过 Web 的调用收到服务端拒绝。CD 的 Gateway/Service 条件仍单独校验。
3. Environment 已就绪但 Gateway 尚未创建时，成员可以运行 CI；CD Gateway 仍按向导的下一步配置。
4. 本地 Environment 上的流水线仍使用当前控制面工作区与 Docker；本地目录型 Repository 按现有只读 `/source` 挂载执行。
5. Windows SSH Environment 的流水线先能在远端完成 Stage、查看远端日志和文件制品元数据；其验收完成后，再让 Linux SSH Environment 达到同等结果。
6. Run 排队后目标或凭据变更，Worker 拒绝旧任务并保留明确的失败状态；重试绑定新目标。删除 SSH Run 时只处理其远端受管文件，不做推测性的控制面路径清理。

## Acceptance

- “环境”为系统管理中的独立路由和菜单入口，不在持续部署菜单中重复；每个 Project 保持一个共用 Environment，CI/CD 读取同一配置，不引入全局环境池或改变原有权限。
- CI 运行/CD 部署前均按当前环境修订做就绪检查；未就绪的 Web 动作定向至 `/project/:id/initialization`，API/MCP 保留后端校验。环境已就绪但无 Gateway 时 CI 可运行；CD 依其自身 Gateway 规则拒绝。
- Windows SSH 实现与测试先于 Linux SSH；两类目标最终均按显式环境远端运行，无控制面路径的远端 bind mount，也无 SSH 失败时的本地 fallback。
- SSH 日志及文件制品仅在远端持久化，日志支持增量查看；文件制品保留位置与存在性元数据及清理语义，命令/Docker 镜像制品在目标端采集，不新增回传或下载功能。
- 目标/凭据修订在 Run 创建时冻结并在执行前核对；过期任务不执行，重试创建新 Run；密钥不进入可读快照或日志。
- local 环境中本地目录 Repository 行为不变；SSH 与本地目录 Repository 的组合不伪造远端 `/source`，在触发前明确拒绝并引导使用可在远端获取的 Repository，以维持“当前不支持远端本地路径”的事实。
- 历史 local Run 在原环境未变时仍可读取；环境已变更时不臆测目标，失败可解释。活文档在功能交付后更新为最终实现语义。

## Open questions

- 无阻止 Plan review 的产品决策。远端本地目录 Repository 的具体错误文案、前端重定向后的返回路径和 Windows/Linux SSH 的测试环境前置条件在实施时确定；不借此改变上述范围。

## Decisions

- Environment 为每 Project 唯一的 CI/CD 共用配置；系统管理中提供独立路由与菜单，不嵌入 `/settings`，不设计跨 Project 相同目标的并发写入或新共享模型；保留现有 target 冲突校验。
- CI 运行与 CD 部署前都做环境最新 Probe 就绪检查，包括 local；Web 不满足时进入现有 Project 初始化页面，后端依旧拒绝未就绪调用。缺 Gateway 不阻断已就绪环境的 CI，CD 保留 Gateway 前提。
- SSH 的交付/验证顺序为 Windows native OpenSSH + WSL2 Docker Desktop → Linux OpenSSH + Docker。
- 远端日志和文件制品驻留远程目标，不回传控制面；不新增文件下载。
- 本地目录型 Repository 保留现有 local 行为，不添加远端传输；SSH 与该 Repository 组合无可用远端源，明确拒绝而非回退 local。
- 仍使用 standard 流程；Plan 按可验收的工程边界顺序分批，本轮不进入产品代码实施或验证阶段。

## User review notes

- 用户要求先统一文档口径，再分析任务项；任务拆解用 SpecFlow 标准模式记录，按粒度决定是否分步。
- 用户明确：系统管理中使用独立路由/入口；每 Project 一个 Environment，不规划多个 Project 共享相同目标的写入协同；环境不就绪时 CI/CD 的 Web 动作进入初始化页。
- 用户明确：Windows 先实现并验证，再做 Linux；SSH 日志与文件制品保存在远端，不回传；本地目录 Repository 在既有 local 环境下行为不变。
