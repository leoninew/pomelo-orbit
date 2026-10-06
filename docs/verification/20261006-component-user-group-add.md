# 组件运行身份、共享挂载与 Orbit 迁移验证
最后修改时间: 2026-10-06 21:49:12

Review status: Draft

Mode: standard

## Intent alignment

依据已接受的 [Intent](../intent/20261006-component-user-group-add.md)，实现 Version 运行身份、Service 继承/覆盖/显式清空、通用共享目录声明及 Orbit 停机迁移说明。Orbit 使用普通应用模型，沿用已有 DooD 容器识别与路径映射。

验证期间用户补充的独立运行身份区域、独立保存接口及两个单行文本框已落实。用户再次要求验证并修复发现的问题；本轮修复失败测试并补足高风险路径的回归覆盖。

用户随后授权真实 Tencent 停机迁移，要求最终通过本地 `orbit.preflite.cn` 项目部署 `pomelo-orbit-default`，并验证新 Orbit 能部署其他服务。初次迁移与部署验证已执行；发现配置摘要问题后，用户要求描述问题及复现步骤，后续真实测试由用户完成，不再提交新的远端部署或测试任务。

用户进一步授权 `ssh oracle` 按相同流程先检查再迁移，并在 Oracle 的既有数据库中补齐 Orbit 普通应用与服务记录；本轮执行该迁移及最小 DooD 验证。

用户随后要求将 Tencent、Oracle 的原 ENV 补进普通应用库存：组件保存占位符，服务保存真实值。本轮通过业务接口补齐并核对保存结果及部署预览，没有重新部署。

## Spec alignment

不适用。标准模式没有单独 Spec，按 Intent 与 Plan 核对。

## Plan alignment

与已接受的 [Plan](../plan/20261006-component-user-group-add.md) 一致：身份字段贯通模型、SQL、HTTP/MCP、Web、定义传递、有效计划、hash 与 Compose；共享属性限绝对目录源，并依据实际运行容器的标签和 bind 判断占用。

Service 根目录、未共享 bind、嵌套私有 bind 和父目录写入仍受保护。共享规则与 worker 或应用名称无关。此前 owner 标记移除已按当前代码同步到迁移说明。

## Actual diff summary

- Version `user` / `group_add`、Service 三态覆盖及增量迁移 `000052`；新增 schema、query 与相应生成代码。
- 独立 `PUT /api/version/:version_id/component/:component_id/identity` 和 MCP `orbit_update_version_component_identity`；Basic 更新的契约与 SQL 不包含身份字段。
- Version mount `shared` 声明、Compose 标签 `io.pomelo-orbit.shared-mounts`、实际挂载与标签匹配及通用目录占用判定。
- Version 身份编辑为 480px 响应式弹窗，两个纵向等高单行输入框，附加组按逗号回填与输入；Service 保留覆盖、清空和恢复继承。
- 活产品/运行时/挂载/MCP 文档与 Tencent 停机迁移、持续部署验收及回退说明。
- 本轮新增 local DooD/SSH 完整目录检查及 CI 派生配置保留测试，修复凭据更新时间测试的系统时钟精度依赖。
- 真实部署发现 Service 详情与部署执行对网关网络名的摘要计算不一致；统一使用产品固定的 `model.GatewayNetworkName()`，补充有/无 Gateway 投影摘要相等的回归断言。

## Expected vs actual changed files

| 预期范围 | 实际变更及核对 |
| --- | --- |
| 模型、身份校验、有效配置和渲染 | `internal/model/`、application/service/deployment usecase；符合预期 |
| 三种数据库增量迁移、schema/query、持久化 | `000052`、application/service SQLC Repository；未修改已执行迁移 |
| HTTP/MCP、Proto/DTO、生成代码 | 独立身份接口、Service 附加组存在性 wrapper；生成物与新增字段对应 |
| 定义传递、fork/CI 派生 | 定义创建/普通 fork/CI fork SQLite 集成及接管包 JSON 保留测试 |
| 前端配置和展示 | Version 身份区域/表单、Service 覆盖、shared 编辑、locale/API/helper；符合用户调整 |
| 文档与迁移步骤 | 活文档、迁移指南及同名 Intent/Plan/Verification |
| SQLC 多包 models.go | 公共 schema 变更产生的生成物同步，不是手工扩展其他领域 |
| 凭据更新时间测试 | 用户授权修复验证失败后追加；仅测试夹具，不改变凭据业务实现 |
| ServiceDetail runtime_directory | 此前独立工作，保留用户变更，不计入本任务功能验收 |

## Acceptance criteria

- [x] Version 运行身份保存/读取/清除；身份与 Basic 独立保存且相互保留字段。SQLite 集成、MCP 工具和页面 DOM/API mock 测试通过。
- [x] Service 继承、整体替换、显式清空和恢复继承；SQL NULL/空字符串及 NULL/空数组往返保留，HTTP Proto wrapper 和 Web helper 保留存在性。
- [x] 以 `1000:1000`、`["988"]` 验证合并、Compose YAML 和 hash；渲染代码核对确认空身份不输出相应字段，没有宿主机 UID/GID 默认值。
- [x] 定义创建、普通 fork、CI 派生保留身份和 shared；接管包 JSON 往返保留 Service 显式空覆盖。
- [x] 身份写入校验覆盖用户/主组、附加组与实际支持的格式；前端字段错误关联控件并独立消除。
- [x] SQLite 51 到当前版本迁移保留原有默认身份和非共享语义，回退到 51 成功；MySQL/PostgreSQL 新迁移做结构核对。
- [x] 标签只标记声明为共享的 target；实际 bind source 匹配，未共享/不同 target/嵌套私有 bind/父写入/Service 根目录继续拒绝。
- [x] local DooD 与 Linux SSH 完整目录检查使用各自路径：共享根允许准备子服务目录，移除共享标签后拒绝；原有 Windows/Desktop 路径与映射回归通过。运行时为测试替身，不是实际 Docker/SSH 部署。
- [x] 迁移指南包含绝对 data/logs/.env/socket 挂载、HOME/工作区配置、版本发布、外部 worker、停旧 project、空目录部署、Gateway 运行绑定、Route 同步、持续部署与回退顺序。
- [x] 工程固定检查、前端全量测试和 Go 全量测试通过；过程文档及迁移说明未加入真实凭据值。
- [x] 初次 Tencent 停机迁移：普通 Service 部署、实际进程身份/socket 权限、共享标签、挂载映射、PostgreSQL 49 -> 52、两域名 HTTPS/Web/API 返回 200。
- [x] 新 Orbit 自身 worker 完成隔离服务的新部署、force-recreate 重部署、restart 入口和 stop；本地项目经 SSH 在共享 data 下部署另一隔离服务成功。
- [ ] 配置摘要修正版 `20261006-migration-24a016038-r1` 的真实升级、详情待部署状态和持续部署验收。用户接手，尚未部署该镜像。
- [ ] 登录、页面视觉、真实用户保存流程，以及新 Orbit 上的 CI 工作流。由用户继续实测，不能由 API 探测或部署队列成功替代。
- [ ] 页面视觉及用户真实会话中的保存验收。用户禁止操作其浏览器，保留人工验收。

## Test results

命令从仓库根目录执行；前端命令通过 `--cwd web` 使用现有工具链。

| 命令或证据 | 结果 |
| --- | --- |
| `yarn --cwd web lint:fix` | 前轮 UI 修正后通过；本轮 `task check` 再次验证 lint |
| `yarn --cwd web typecheck` | 通过，包含本轮 `task check` |
| `yarn --cwd web test` | 50 个测试文件、224 项测试通过 |
| `task check` | Web 类型/lint/格式与 Go 格式/静态检查通过，0 issues |
| `go test ./cmd/... ./internal/...` | 修复后全量通过；新增回归后再次通过 |
| `go test ./internal/application/credential/usecase -run TestCredentialServiceTracksUpdatesWithoutChangingCreationTime -count=20` | 20 次通过 |
| `go test ./internal/application/application/usecase ./internal/application/deployment/usecase` | 新增 CI 派生及 local/SSH 共享目录回归通过 |
| `git diff HEAD --check` | 通过 |
| `task sqlc`、Go `buf generate` | 前轮已成功生成并检查；本轮未修改 SQL/Proto |
| TypeScript Proto 生成 | 前轮 Buf 远程插件不可用，使用本地同版本 ts-proto 2.11.8、相同配置生成；不将 `task proto` 全流程记录为通过 |
| 实际 PostgreSQL、Docker/SSH、域名 | 本轮初次迁移已验证，具体证据见下文；修正版镜像尚未真实升级 |
| 实际 MySQL、浏览器 | 未验证；用户禁止操作浏览器，保留人工验收 |
| 摘要修正后 `task check` | 通过，0 issues |
| 摘要修正后 `go test ./cmd/... ./internal/...` | 全量通过 |
| 摘要修正后 `go test ./internal/application/deployment/usecase ./internal/application/service/usecase` | 通过，覆盖详情与部署摘要一致、网络开关仍影响摘要 |

## Tencent execution and handoff

本地与远端使用独立库存和队列。首次迁移由本地现有 worker 通过 SSH 执行；没有启动额外远端 worker，也没有停止、重启用户开发服务器。远端隔离服务任务由新容器内 `orbit-tencent-managed` worker 消费，容器日志包含相应 task claimed/completed；本地 worker 无法消费这套远端队列。

原指定 `pomelo-orbit:20261006-091926` 对应旧提交 `f1a5bd9e6`，镜像缺少身份接口、共享标签及 000052 迁移。改用当前已提交源码 `24a016038` 构建并部署 `pomelo-orbit:20261006-migration-24a016038`，image ID 为 `sha256:702e118a09ee564d01a5433363223f95b9a101a582bdb42a022b0c3ab748a9be`。

| 操作 | 证据与结果 |
| --- | --- |
| 本地项目部署 Orbit | `deployment/01M4892SFEPEDYPEFN8S7N0N14`，`ran_to_completion`；服务状态 running，确认/运行目录均为 `/opt/pomelo-orbit/data/deployment/pomelo-orbit-default`，修订 1 |
| 实际运行配置 | 用户 `1000:1000`、附加组 `988`；data/logs/.env/socket 四个原绝对源保留；`io.pomelo-orbit.shared-mounts=["/app/data"]`；data/logs 可写，容器内 Docker 查询成功 |
| 数据库 | 启动日志记录 PostgreSQL 从 49 升至 52，dirty=false；旧 JWT 环境覆盖沿用，未重置数据库或密钥 |
| Gateway 普通部署 | `deployment/01M4894WTMSATNPYHD3ZDQ27WM` 成功，原目录与 dynamic/certs/acme 保留；补齐 HTTPS ingress 配置及运行绑定 |
| 两域名发布 | 仅同步 `orbit-web`、`orbit-api` 两条新增 Route，configuration_match=matched；两个域名的 HTTPS/Web/API 返回 200 |
| 运行校验 | `verify_deployment` 返回 consistent、stable，未检测漂移或重启 |
| 远端 DooD 新部署 | `deployment/01M489F44VZMCF5SB9V8BTMJ2A` 成功；容器内目录 `/app/data/deployment/orbit-migration-check-default` 映射到宿主机同名 deployment 子目录；受控文件落盘，HTTP 18085 返回 200 |
| 远端 DooD 重部署/restart | `01M489H6F509WNA21B1AVT6FV8`、`01M489J5J4NYE6C17KAM2N3AZZ` 成功；重部署实际重建。普通服务 restart 当前使用 Compose up 语义，配置未变时没有重建进程，不能据此声称实际进程重启 |
| 本地 SSH 隔离服务 | 旧 MCP 提交的 `01M489KYW5X6DW5772NV2QD556` 因旧版摘要被拒绝；通过当前接口重提 `01M489NX2YQ0EJRYTM2G49GSH5` 成功，受控文件挂载来自正确远端目录，HTTP 18086 返回 200 |
| 测试服务清理 | 两个测试服务通过业务 stop 入口完成，`01M489Q9BYXWBYS8ZW43XRNV8Z`、`01M489R2RB0F2ZSH3X7VP8YRDM` 成功；容器已退出，测试应用记录与小型目录保留供核查 |

回退材料保留在远端 `/home/ubuntu/orbit-migration-backup-20261006-24a016038`，包括 PostgreSQL dump、原配置/启动入口、旧容器 inspect、Gateway 文件和根据实际已运行镜像生成且校验过的 rollback Compose。旧容器已停止，restart=no，旧 `boot.sh` 改名为 `boot.sh.pre-migration-disabled`。数据仍位于原绝对目录，不把整个父目录搬入自己的子目录。

额外探测既有 Windows SSH target `10.126.126.1:22` 返回连接超时；Tencent 宿主机直接连接也不可达，路由经默认网关而非目标专网。未修改该 Environment target 或目标机器，不把此环境记为 SSH 部署验收通过。

## Configuration hash issue and reproduction

**问题**：普通服务部署成功、运行配置没有改变，但详情页仍显示 `pending_deploy=true`。

**原因**：详情页构建有效计划时不加载 Gateway 投影，摘要的 `GatewayNetworkName` 为 `""`；部署构建相同计划时带有 Gateway 投影，该字段为 `"traefik"`。默认加入 Traefik 网络的同一服务得到不同摘要，详情误判有未部署变更。与运行用户、附加组、数据目录或容器状态无关。

复现步骤（修正前代码）：

1. 在已准备 Gateway 的项目中配置一个普通服务，保留默认加入 Traefik 网络；使用 internal HTTP endpoint 即可。
2. 确认部署目录并部署，等待 Deployment 进入 `ran_to_completion`，Service 显示 running。
3. 不修改 Version、Service 或 Environment，重新打开服务详情；`pending_deploy` 仍为 true。重复部署后依旧出现。

修正统一从产品固定定义 `model.GatewayNetworkName()` 计算网络名，详情不依赖加载 Gateway 对象；不加入网络时继续使用空网络名，网络开关仍改变摘要。实际部署原本带有固定网络名，其摘要不因这项修正而改变，不改写已有成功记录。

回归测试在 `TestEffectiveServicePlanHashTracksTraefikNetworkOption` 中以带 Gateway 投影的计划作为部署基线，移除投影后断言摘要相同；该断言在原实现下不成立。随后断言关闭网络仍改变摘要，避免把真实网络配置变化也消除。相关测试、Go 全量测试与 `task check` 均通过；使用当前代码读取本地已成功部署的 Orbit，`pending_deploy=false`，没有补写成功记录。

交接时修正版镜像已构建为 `pomelo-orbit:20261006-migration-24a016038-r1`，image ID `sha256:73580ae7dfa5785890e348a0ecf0b18a1314526db88acabd21d468b0eabaf68d`，来源为 `24a016038` 加本次未提交的摘要修正。当时尚未部署；用户后续自行构建和部署的镜像及观察结果见下节。

用户后续真实验收：确保 HTTP/MCP/worker 使用当前代码；旧 MCP 是长期运行进程，需刷新会话。通过本地 `orbit.preflite.cn` 项目将既有 Orbit 组件镜像更新为 r1，再部署相同普通目录。部署成功后应为 running 且 pending_deploy=false；修改一项环境值或运行身份应变为 true，再成功部署后恢复 false。再从新 Orbit 部署隔离服务，检查共享目录、socket、HTTPS 与真实登录。此轮不继续执行这些操作。

## 用户部署后的只读观察

用户自行远程构建和部署后，于 2026-10-06 20:53（Asia/Shanghai）请求观察。此轮只查询记录、接口、容器与日志，没有提交部署、修改配置或操作浏览器。

| 项目 | 观察结果 |
| --- | --- |
| 最新部署 | `01M48KZZRA0WKN5PTZKS8TQEP6`，`ran_to_completion`；完成时间 `2026-10-06T12:46:13Z` |
| 实际镜像 | `pomelo-orbit:20261006-124314`；image ID `sha256:144bea916756bbd143f9cbfa9f630970610c1f713905027275e6baa294d982e8`；容器 `b015823b2a40` |
| 目录与详情 | 本地服务详情接口返回 running、`pending_deploy=false`、`effective_error=""`、`active_deployment=false`；确认目录与运行目录均为 `/opt/pomelo-orbit/data/deployment/pomelo-orbit-default`，目标与运行修订均为 1 |
| 运行一致性 | `verify_deployment` 返回 consistent、stable；无漂移，13 次采样均为 running、restart_count=0 |
| 启动日志 | PostgreSQL 52 -> 52，dirty=false；HTTP 服务及 `orbit-tencent-managed` worker 已启动；所查启动日志无异常 |
| DooD 条件 | 用户 `1000:1000`、附加组 `988`；原 data/logs/.env/socket 挂载及共享 data 标签保留；容器内 Docker Server 27.5.1 可访问，`/app/data` 可写 |
| HTTPS | Web 返回 200；远端 `/api/health` 返回 200、`{"status":"ok"}` |

这次实际成功部署后的“待部署”误判未再出现。修改配置后的标记切换、此次新镜像的真实 DooD 部署、登录及 CI 仍由用户验收；只读运行条件检查不能替代这些完整流程。Verification 保持 Draft。

## Oracle execution

2026-10-06 用户授权后，先检查 `ssh oracle` 的运行容器和通过本地 `5434` 转发的 Oracle PostgreSQL。旧 Orbit 由手工 Compose 运行，项目为 `pomelo-orbit`，目录 `/data/apps/pomelo-orbit/dist/20260908`，尚无共享 data 标签。数据库中没有 Orbit 自身的 Application/Service；既有默认 Project 的 local Environment 使用 `/app/data`，target revision=3，保留该目标与修订。

| 项目 | 实际结果 |
| --- | --- |
| 库存 | Oracle 数据库的默认 Project `01KRRKK0K3T519ZQZES3M4QA9Z`；Application `01M48N0AQXHP31MF2E2R9DF3WF`、Version `01M48N0AQXHP31MF2E2TW4XA8Y`、Component `01M48N0BFDQ73PDVB0D0WEXW6Z`、Service `01M48N2BJ1156HD21HQBDJ95HB`，服务编码 `pomelo-orbit-default`；全部由业务接口创建 |
| 普通部署 | `deployment/01M48NAMHZACR4BD8PZPFJ5SZX`，ran_to_completion；临时外部执行端日志确认领取和完成部署任务 |
| 目录 | Service 确认及运行目录均为 `/app/data/deployment/pomelo-orbit-default`，修订均为 3；映射宿主目录为 `/data/apps/pomelo-orbit/dist/20260908/data/deployment/pomelo-orbit-default` |
| 镜像与状态 | `pomelo-orbit:20261006-124804`，image ID `sha256:9df40d2d1484d5f23c9936dc457c0d13869c152ce59dd8a6cd58592c41cf19e4`；新容器 `a5d4e4caec7b`，running、restart_count=0、OOMKilled=false；详情 pending_deploy=false、effective_error 为空 |
| 身份与挂载 | 重新核查用户 `1000:1000`、socket 附加组 `989`；原 data/logs/.env/socket 来源保留，`.env` 仍为只读；data 标签为共享；data/logs 可写，容器内 Docker Server 29.8.0 可查询 |
| 数据库及 worker | PostgreSQL 保持 52，dirty=false；HTTP 和 `orbit-oracle-managed` worker 启动成功，原数据库、JWT/加密密钥及其他启动配置沿用 |
| 原入口 | 保留 `127.0.0.1:9020`；既有 `orbit` Route `01M2EPSFABD0CQVTQB640J11HT` 从自定义 `http://pomelo-orbit` 改为新 Service 的 HTTP 80 受管 target；只预览并发布此条 Route，configuration_match=matched，未重部署 Gateway |
| 外部访问 | `https://orbit.oracle.preflite.cn` 返回 200；其 `/api/health` 返回 200、`{"status":"ok"}`；既有 HTTPS 属性保留 |
| 新 Orbit 的 DooD | 临时执行端退出后，新 Orbit worker 完成 `deployment/01M48NMPNDJZXP2T4QC7Y9MRD7`；隔离服务的受控 `index.html` 实际 bind source 为宿主部署子目录，HTTP 18087 返回 200、`orbit-oracle-dood-ok` |
| 测试清理 | `deployment/01M48NQ7WWQPJDE8EYPGD625SQ` 通过业务 stop 完成；测试容器已移除，测试应用与停止状态的 Service 记录及小型目录保留 |

数据库 dump、原 `.env`/Compose 和 Gateway 文件归档保存于 `/home/opc/orbit-migration-backup-20261006-ordinary`，目录权限 0700；dump 的 archive list 校验成功。另保留实际旧镜像标签 `pomelo-orbit:pre-oracle-migration-20261006`。旧 `pomelo-orbit` 容器保留为 exited、restart=no；未发现对应 systemd unit 或用户 crontab 启动项。没有搬移整个父目录、重置数据、修改目标修订或以 SQL 补写成功状态。

迁移使用临时容器 `orbit-oracle-migration-worker` 提供外部业务入口和 worker，完成主部署后已停止并移除。临时启动首次因 Windows CRLF 导致 `serve` 参数带 `\r` 失败，旧服务当时未停；清理该临时容器并去除输入 CR 后重建成功。隔离测试首次 `01M48NHMRCDAHMMVME1EYDNDYX` 因 Oracle 已有 Alpine 镜像没有 `httpd` 失败；通过业务 fork 改用已有 Nginx 镜像重部署成功，失败记录保留。没有新增辅助脚本文件或操作浏览器；本轮临时 SSH 转发和本地操作会话均已关闭。

停机回退没有实际执行。真实登录、CI 和用户人工验收未完成；普通应用后续更新自身仍须外部 worker。Verification 保持 Draft。

## ENV configuration correction

迁移时保留了原 `.env` 文件挂载，但遗漏将全部环境值纳入普通应用库存。Tencent 原文件 `/opt/pomelo-orbit/.env` 包含 15 项，组件仅声明 3 项、Service 仅保存 JWT 一项；Oracle 原文件 `/data/apps/pomelo-orbit/dist/20260908/.env` 包含 19 项，组件仅声明 5 项、Service ENV 为空。原文件仍存在并挂载至 `/app/.env`；业务详情中的环境配置不完整，后续不能只依靠挂载文件维护配置。

按用户要求合并原文件的全部变量、容器中的业务环境及现有库存值，保留暂时不用的键；已有 Service 值和当前生效覆盖优先，数据库连接及 JWT/凭据加密密钥保持原值。两个服务均没有组件级 ENV 覆盖。原版本已发布，因此通过业务 fork 创建修正版，将所有组件 ENV 改为同名 `${KEY}`，在 Service ENV 保存真实值，发布修正版并切换服务选用版本。

| 环境 | 修正版 Version | 组件占位符 / Service 真实值 | 保留的原文件键 |
| --- | --- | --- | --- |
| Tencent，本地 `orbit.preflite.cn` 项目 | `01M48Q852DRF2TYXG685ESPPXQ`，`build-20261006-124314-env-20261006` | 19 / 19 | 15 / 15 |
| Oracle，远端默认项目 | `01M48Q8MNN8QTTPYH2S9Z4QCJA`，`pomelo-orbit-env-20261006` | 24 / 24 | 19 / 19 |

保存后重新读取 Service 与 Version，逐项核对真实值及全部占位符；原已发布版本未变。修正版除 ENV 外的组件配置完整保留，包括镜像、挂载/共享属性、运行用户、附加组、端点及制品引用；服务编码、确认/运行目录和目标修订未变。两份 Service 预览经 YAML 解析及 `docker compose -f - config --format json --no-path-resolution` 验证，最终 environment 与保存的真实值逐项一致，Compose 没有再次插值改变实际值。Oracle 一次接口读取超时，重试完成核对。

运行容器 ID、启动时间、重启次数、镜像、实际 ENV 与挂载均与修改前一致。Docker inspect 返回的挂载列表顺序不同，按目标路径排序后内容一致。两个服务仍为 running，`effective_error` 为空、无进行中的 Deployment；`pending_deploy=true` 是已保存 ENV 修正尚未部署的预期状态。本轮未创建 Deployment、重启容器或改动原 `.env` 文件及挂载，真实部署由用户执行。迁移指南已补充 ENV 入库、占位符与 Compose 核对步骤，没有新增辅助脚本文件。

## Issues and fixes

1. 前轮运行身份 UI 使用两列布局、单行与多行混用。已收敛为两个纵向单行控件与紧凑弹窗，按钮为“保存”。附加组响应数据改为逗号回填，避免单行输入剥离换行导致组值合并；页面回填与独立 API 保存测试通过。
2. 凭据更新时间测试要求极短间隔的连续 `time.Now()` 必然严格递增，Windows 上反复失败。测试以明确的过去时间作为持久化基线，分别验证名称和内容更新刷新更新时间、保留创建时间及列表返回值；保留严格更新断言，没有加入 sleep、重试或业务时间兜底。连续 20 次和全量测试通过。
3. 共享规则原有测试主要验证底层挂载判定，CI 身份保留主要靠普通 fork 覆盖。本轮补充完整目录检查的 local DooD/SSH 正向与未标记拒绝回归，以及经过 SQLite 保存再读取的 CI 派生版本配置保留测试。
4. 真实部署发现详情与部署摘要因 Gateway 网络投影差异而不一致，按上文修正并完成自动化验证；修正版真实升级由用户接手。
5. 两环境迁移保留了 `.env` 挂载但未补齐 ENV 库存，按用户要求通过业务接口补齐组件占位符和 Service 真实值，重新读取及 Compose 解析验证通过；本轮未部署。

## Missed or expanded scope

独立身份保存与简化 UI 为用户明确补充的需求，已同步 Intent/Plan。凭据测试修正为本次验证发现问题后的授权修复，可与本功能分开提交。保留原有 runtime_directory 展示变更。

没有新增控制面特例、DooD 路径映射、兼容接口、业务默认 UID/GID 或额外第三方依赖。真实迁移及普通业务库存写入为用户后续明确授权；没有以 SQL 伪造部署记录或运行绑定，没有管理开发服务器或执行 Git 写操作。用户明确禁止浏览器操作后没有继续相关验证；后续真实测试交接后没有继续提交远端任务。

Oracle 的新增真实迁移与隔离服务部署按用户后续明确授权执行，与 Tencent 修正版镜像的人工验收交接分别记录。

两环境 ENV 补齐按用户后续明确要求执行，仅保存业务配置和验证预览；没有通过 SQL 写入库存，也没有操作浏览器或管理开发服务器。

## Risks and incomplete items

- 初次 Tencent、Docker 权限、Gateway、域名和两种部署链路已有上文实际证据；未执行停机回退，不能将备份和 rollback Compose 校验称为实际回退成功。
- PostgreSQL 已实际升级至 52；MySQL 仍未用真实数据库验证。
- 已运行容器需要重新部署才带有共享标签；声明共享不释放 Service 根目录或嵌套私有挂载。
- `988` 只是 Tencent 核查值；迁移时重查 socket GID、进程身份及 data/logs 写权限。
- Orbit 更新自身仍需外部 worker。Tencent 初次迁移未启动额外 worker；初次远端 DooD 验收任务只由新 Orbit 的远端队列 worker 执行。Oracle 迁移的临时执行端已移除，隔离部署由其新 Orbit worker 完成。Tencent 后续构建部署的镜像已完成上文只读观察，真实 CI、登录及此次 Tencent 镜像的 DooD 完整流程仍待用户验收。
- 当前会话的旧 MCP 进程未刷新，旧工具表和摘要可能与当前 worker 不一致；后续测试前刷新客户端 MCP 会话，不引入新旧逻辑兼容层。
- 已构建的 r1 镜像未直接用于运行；用户自行部署了 `pomelo-orbit:20261006-124314`。本地详情在此次实际成功部署后为 `pending_deploy=false`；修改配置后标记变为 true、再次部署后恢复 false 的流程仍由用户完成。
- 页面视觉验收由用户完成。Verification 保持 Draft，待用户人工接受。
- 两环境已选用 ENV 修正版，尚未实际部署；此时 `pending_deploy=true` 为预期结果。原 `.env` 挂载保留，后续 ENV 变更应维护 Service 真实值及组件同名占位符。

## Conclusion

Tencent 初次普通应用迁移及 DooD/SSH 隔离服务部署成功。配置摘要问题已修正、自动化检查通过；用户后续自行构建部署的 Tencent 镜像运行一致且稳定，本地详情未再出现“待部署”误判。Oracle 也已通过普通业务记录与部署完成迁移，既有入口可用，新 Orbit worker 完成隔离 DooD 部署及停止。两环境 ENV 库存现已补齐，部署预览与 Compose 解析逐项一致；ENV 修正尚未实际部署，此时“待部署”为预期状态。其余真实验收由用户完成，Verification 保持 Draft。
