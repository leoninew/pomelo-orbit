# CD：Gateway TCP 对外访问（本机 / 公共 · 部署期执行）
最后修改时间: 2026-07-23 13:18:30

Review status: Accepted

Flow mode: strict

Parent / 关联：
- `docs/requirement/20260723-cd-gateway-entrypoint-ownership.md`（Gateway 持有入口/TLS/base_domain SoT）
- `docs/requirement/20260722-cd-gateway-domain-config-e6.md`
- `docs/requirement/20260721-cd-environment-expose-service.md`（Version Expose `http|tcp`，无 host port）
- `docs/spec/20260722-cd-environment-ingress-policy.md`

## Background

平台已有 **Version Expose**（是否对外暴露、协议 `http|tcp`、容器端口等）。部署具体版本时，网关侧应 **读取暴露声明与暴露方式**，决定动作——而不是另建一套与部署脱节的「TCP 路由 CRUD 控制台」。

可达性应分成 **三层**，避免混谈：

| 层级 | 谁连 | 典型路径 | 是否依赖 Expose / 网关 |
|------|------|----------|------------------------|
| **集群内** | 其它应用容器（如 app2 Spring → app1 Redis） | Docker 网络 DNS + **容器端口** | **否**（默认同网可达；无 expose 也可） |
| **本机 local** | 宿主机上的人/工具 | 约等于平台生成的 `127.0.0.1:listen:container`（安全版 `ports`） | **是**（`access=local`） |
| **公共 public** | 局域网/公网客户端 | 域名 + 网关 HTTP(80/443) 或 TCP 端口 | **是**（`access=public`） |

现状问题：

1. TCP labels 仍挂在 HTTP 入口 `web`/`websecure`（80/443）上，**不是**业务端口（6379/3306）语义。  
2. 曾讨论「Gateway 全局 TCP 路由表 CRUD」——与裁定 **不符**：网关应 **展示 + 部署时执行**，不是独立 CRUD 面板。  
3. 用户预期：版本声明 **协议（http|tcp）** 与 **暴露范围（本机|公共）**；无暴露则 **不产生出口**，但 **集群内仍可互访**；HTTP 公共走现有形式；TCP 出口为实现重点。  
4. local 与手写 `ports: - "xx:xx"` **对使用者语义相近**；差异是 **平台按 expose 生成**（且应绑 loopback），不是用户在版本里手写 ports。

协议事实（产品约束）：

- 同机两个 `localhost:同端口` 冲突。  
- 明文 TCP 不能靠双域名共用同端口；TLS+SNI 可以。  
- Traefik 3 + LE 支持 TLS/SNI；ACME 不绑业务 TCP 端口。  
- **集群内互访不依赖 TLS/SNI/域名**；应用配置使用 **服务名 + 容器端口**。  
- **public 域名**沿用既有规则：`{app_code}.{gateway.base_domain}`（component 级是否细分 Spec 定）；**域名 ≠ 强制 HTTPS**。

## Goal

1. **SoT：出口意图在「版本的 Expose 声明」**  
   - 声明至少包含：  
     - **是否做本机/公共出口**（无对应 expose / 无 access = 不产生 local/public 出口）  
     - **`protocol`**：`http` | `tcp`  
     - **`access`**：`local` | `public`  
     - **`container_port`**（及 HTTP 既有 path 等）  
   - **不**要求用户在版本里手写 Docker `ports`。  
   - **不**做 Gateway TCP 路由 **CRUD 控制面板** 作为主路径。

2. **三层可达（必须写清边界）**  
   - **集群内（默认能力，非 Expose 子类）**  
     - 同平台互通网络内，组件可被其它应用容器用 **稳定 DNS 名 + 容器端口** 访问。  
     - **不要求** 为互访配置 local/public expose。  
     - **禁止** 把「app2 连 app1 Redis」设计成必须先 public 再绕网关（可作为文档反模式）。  
     - 命名公式 Spec 钉死（倾向可跨应用解析的 `{app_code}-{component}` 或等价）；部署后可在 UI/文档展示「内网地址」。  
     - 无 expose 的组件 **仍应** 能加入互通网络（不得「只有 expose 才进网」导致无法互访）。  
   - **本机 local**：见下。  
   - **公共 public**：见下。

3. **网关职责：部署期执行出口 + 只读展示**  
   - **部署（及预览/校验）时**：读取 exposes + Gateway 配置，生成 local/public 出口；**集群内 DNS/网络** 由平台网络与命名规则保证（可与网关页展示并列，但不等于 Traefik 路由 CRUD）。  
     - 无 local/public expose → **不**生成宿主/公网入口、**不**占对外端口；**不影响** 集群内名访问。  
     - HTTP public → **现有** Host / 80/443 路径。  
     - TCP public → **新实现**（网关监听业务端口等）。  
     - local → 平台生成 loopback publish 或等价（见下）。  
   - **网关 UI**：展示部署形成的 **出口**（端口/域名占用）；可附带内网名提示；**不是**手工 CRUD 路由表。

4. **本机暴露（local）≈ 受控的 host port 映射**  
   - 对使用者：与 `ports: - "xx:xx"` 同类——`localhost:<port>` 可达。  
   - 实现：平台生成 **`127.0.0.1:<listen_port>:<container_port>`**（或等价），**不是**用户手写；`listen_port` 默认 = `container_port`。  
   - HTTP 与 TCP **均**做 **宿主机端口冲突检查**。  
   - TCP local **不强制** 经 Traefik（直连容器即可）；与 public TCP（经网关）区分。  
   - HTTP local 的流量形态 Spec 细化（独立 loopback 端口 vs 其它）。

5. **公共暴露（public）与域名**  
   - **沿用**现有分配：`{app_code}.{gateway.base_domain}`（如 `myapp.lvh.me`）；public 端口/入口 **复用该域名** 作为 Host / 展示名 /（若 TLS）SNI，不另搞一套无关命名（component 级扩展 Spec 定，但原则仍是「网关后缀 + 应用侧标识」）。  
   - HTTP：现有公共路径；**默认可走明文 HTTP（:80 / entrypoint web）**，**不**要求开发环境必须 HTTPS。  
   - TCP：网关对外端口 + **同一套域名作标识**；**明文 TCP 不需要 HTTPS**（域名只是 DNS 指到网关 IP，客户端 `host=域名 port=6379` 即可）。  
   - TLS/HTTPS/LE 为 **可选增强**（Gateway `tls_mode` / expose 策略）：生产或需同端口多 hostname SNI 时再开；**开发阶段 `tls_mode=none` + `default_entrypoint=web` 必须可完整测 public HTTP 与 public 明文 TCP**。  
   - LE 与内网后缀（`lvh.me` 等）本来就签不下来——开发路径 **禁止** 做成「没有有效证书就不能测 public」。

6. **废弃** 产品面「TCP 入口 = 选 web/websecure」。

7. **失败明确**：local/public 端口冲突、public 无法推导域名等 → 部署/校验失败。

8. **质量门禁**：`task check` / `task test`；strict 流程。

## Non-goal

1. **Gateway 集成式 TCP 路由 CRUD 控制面板**。  
2. 把 **集群内互访** 做成必须经 Traefik / 必须 public。  
3. 一期完整 **服务依赖注入**（app2 自动注入 app1 redis 地址）——可展示内网地址；自动 wiring 可二期。  
4. 多 Gateway / 多机编排、UDP、协议感知代理、TLS 透传。  
5. 用户编辑 Traefik static 全文；强制公网 LE E2E。  
6. 修改已执行 migration；历史双轨。  
7. 无 expose 却强制进 Traefik 路由。

## User scenarios

1. **public TCP Redis（人/外网工具，开发可无 HTTPS）**  
   - expose `tcp` + `6379` + `access=public` → 域名沿用 `{app}.{base_domain}`，网关监听，业务无 host ports。  
   - 开发：`tls_mode=none` 时 `redis-cli -h myapp.lvh.me -p 6379`（明文）即可，**不需要**证书。

2. **local TCP/HTTP（本机工具）**  
   - `access=local` → 平台生成 `127.0.0.1:…`；冲突则失败。  
   - 与手写 `ports` 体验同类，配置在 Expose 不在用户 YAML。

3. **不暴露出口，仅集群内使用**  
   - app1 版本含 redis **无** local/public expose。  
   - app2 Spring：`host=<内网DNS如 app1-redis>`，`port=6379`（容器端口）。  
   - **不需要** 为 app2 去连 `localhost` 或 `*.lvh.me`。

4. **跨应用：app1 Redis → app2 Spring（核心补充场景）**  
   - 两应用部署后加入 **同一互通网络**。  
   - 平台提供稳定可引用的服务名（Spec 公式）。  
   - UI/部署结果可展示内网连接串提示。  
   - 可选：仅当运维要用笔记本连 redis 时，再给 redis 加 local 或 public。

5. **HTTP 公共（开发无 HTTPS）**  
   - `http://{app}.{base_domain}` 走 :80；`tls_mode=none` 时 **不**强制跳转 HTTPS / 不强制 websecure。

6. **两套 public 明文 TCP 同端口**  
   - 第二套失败。

7. **网关页**  
   - 展示出口占用；可只读展示内网名；无 CRUD 建路由。

## Acceptance

1. Expose 含 **protocol** 与 **access（local|public）**；无 expose ≠ 不能集群内访问。  
2. 文档与实现区分三层：**集群内 / local / public**。  
3. **集群内**：无 local/public 时，跨应用容器仍可按 **服务名+容器端口** 互访（网络与命名一期交付或明确平台约定 + 可验证）；**不**要求经网关。  
4. **local**：语义为平台生成的 loopback publish；HTTP/TCP 端口冲突检查；用户版本不手写 ports。  
5. **public HTTP**：现有路径；**public TCP**：网关出口新实现。  
6. public 域名 **沿用** `{app_code}.{base_domain}`（或 Spec 确定的同源扩展）；**不**强制 HTTPS。  
7. **开发无证书可测**：`tls_mode=none` 下 public HTTP（:80）与 public 明文 TCP 为验收路径。  
8. 无 Gateway TCP 路由 CRUD 主路径；网关展示出口状态（及内网名提示可选）。  
9. 去掉 TCP=web/websecure 产品语义；TLS 终结 + 可挂 certResolver（L1）为可选/生产向。  
10. `task check` / `task test` 通过。

## Open questions

**已闭合**——见 `docs/spec/20260723-cd-gateway-tcp-access.md`「Requirement Open questions 闭合」。


## Decisions

1. **出口意图在 Expose**；网关部署执行 + 展示；**非** CRUD 路由表。  
2. Expose：`http|tcp` + `local|public`。  
3. **三层可达**：集群内（默认、服务名+容器端口）/ local（≈ loopback ports）/ public（域名+网关或 80/443）。  
4. **local ≈ 平台生成的 `127.0.0.1:listen:container`**，与手写 ports 同类，非用户手写。  
5. **集群内互访不依赖 Expose**；禁止强制「先 public 再给 Spring 用」。  
6. 无 local/public → 无宿主/公网出口；集群内仍可达（网络允许时）。  
7. public TCP：监听在 Gateway 侧；业务不写 `0.0.0.0` 业务 ports。  
8. **public 域名沿用 app_code + base_domain**；与是否 HTTPS 解耦。  
9. **开发阶段无 HTTPS 可测** public HTTP 与明文 TCP；TLS/LE 可选，非开发门槛。  
10. TLS 终结；LE=L1（有公网域名时）；删除 TCP→web/websecure。  
11. 撤销 S3 Gateway 全局路由表 CRUD。  
12. 一期自动依赖注入 env 为非目标；**展示内网地址** 为目标的一部分。  
13. Flow：**strict**。

## Risk

1. 今日「仅 expose 才加入 traefik 网」会阻断无 expose 的跨应用互访 → 须改网络加入策略。  
2. 多 env/instance 时内网名歧义。  
3. HTTP local 与 80/443 纠缠。  
4. Gateway 动态 TCP entryPoint 需滚动。  
5. 客户端 TLS 能力不一。  
6. 旧 TCP 挂 80/443 行为移除。

## Assumptions

| 假设 | 说明 |
|------|------|
| A1 | 单 active Gateway |
| A2 | 平台可提供跨应用互通的 user-defined 网络 |
| A3 | 集群内使用 **容器端口**，不用 host publish 端口 |
| A4 | 部署能读 version exposes + gateway config |
| A5 | 无历史双轨 |

## User review notes

- 2026-07-23：部署声明暴露 + 网关执行/展示；非 CRUD。  
- 2026-07-23：http|tcp + local|public；local 冲突检查；public 分配应用+容器域名。  
- 2026-07-23：确认 local 与 `ports` 对使用者同类（平台生成 loopback）。  
- 2026-07-23：**补充集群内场景**——app1 Redis 给 app2 Spring：服务名+容器端口，不依赖 Expose/网关绕行。  
- 2026-07-23：public 域名沿用 app+后缀；**开发无 HTTPS 仍可测**（明文 HTTP/TCP），TLS 非门槛。  
- 2026-07-23：用户「没有不明事项就进入 spec」→ Requirement **Accepted**，进入 Spec。  
- 2026-07-23：运行模型简化——**禁止同 app 并行多套**；**不做多环境并发**；内网/容器名 **`{app}-{component}`**（Spec 已同步）。
