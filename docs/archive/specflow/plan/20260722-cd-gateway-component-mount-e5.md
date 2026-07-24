# CD E5：Component 全规格 / 部署运行时 / rest 路由 — 实现计划
最后修改时间: 2026-07-22 17:05:40

Review status: Accepted

Flow mode: strict

## Requirement / Spec basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260722-cd-gateway-component-mount-e5.md` | **Accepted** |
| `docs/spec/20260722-cd-gateway-component-mount-e5.md` | **Accepted** |
| `docs/requirement/20260721-cd-application-version-cadence.md` | Accepted；本 Plan = **E5** |
| `docs/verification/20260722-cd-application-kind-gateway.md` | R3 已交付；kind 分支不得回退 |

**硬约束（不得偏离 Spec）**：

1. Version 规格 = 结构化字段 + 表单；禁止巨型自由 JSON 作主 SoT。  
2. RuntimeConfig 属部署面，挂 Deployment；不回写 Version；废止 env_file 主路径。  
3. 挂载解析与表单 **全 kind 共用**。  
4. top-level network 仅 `kind=gateway` × Environment（默认名 `traefik`）。  
5. 平台动态路由 = Traefik **`providers.rest`** 全量 PUT；废除 File/dynamic 目录写路由。  
6. **不为** gateway 做特殊部署目录树（不为接平台路由强制 dynamic 子树）。  
7. 路由快照由平台维护并 PUT；不进 Version 表单。  
8. File 与 rest **不得**双轨并行写平台路由。  
9. **全局至多一个**已部署 gateway（D9）。  
10. 自定义 PEM / rest 鉴权 / 多 gateway / dashboard 等 → **F1–F7**，不塞进本期。  
11. **网关领域（E6）**不塞进本期：`docs/requirement/20260722-cd-gateway-domain-config-e6.md`（managed 容器 / external 宿主机双形态；配置对人、部署借 Version）。E5 不实现该产品面。

## Legacy：`init.sh` 现状与定位

### 旧行为（事实）

| 来源 | 内容 |
|------|------|
| `data/cd/traefik/init.sh` / bak | `mkdir -p data/dynamic`、`mkdir -p data/certs`；不存在则 `touch data/acme.json` |
| `data/cd/filebrowser/init.sh` | `mkdir -p data` + `chown`/`chmod`（应用侧权限准备） |
| 文档（`docs/guides/deployment.md` 等） | 「若存在 init.sh 则部署前执行」——**历史叙述** |
| `cdworkspace.WriteConfig` | 写 `init.sh` 时规范化 LF + `chmod 755`（仅物化能力） |
| **当前 Go 部署路径** | `renderAndDeploy` **只**写 `docker-compose.yml` + `docker compose up`；**不执行** `init.sh` |

结论：**`init.sh` 是临时/手工举措**，不是当前平台正式部署闭环的一部分；E5 不得把它升级为 gateway 专用平台契约，也不得继续依赖「用户记得跑脚本」才能部署成功。

### 旧行为语义拆解

| 旧 `init.sh` 动作 | 真实意图 | 与 E5 关系 |
|-------------------|----------|------------|
| `mkdir data/dynamic` | 为 **File provider** 写路由目录准备空树 | **废止**：平台路由改 rest；**不**再为路由强制 dynamic 子树（Spec D5/D6.3） |
| `mkdir data/certs` | 为自定义 PEM / File TLS 准备目录 | **本期不做写入**（F1）；若 Version 声明 logical 挂载再走通用准备 |
| `touch data/acme.json` | 保证宿主机上是**文件**而非目录；Docker 在源路径不存在时会把 bind 挂成目录 | **必须保留语义**：归入**全 kind 共用**的「挂载源物化」 |
| `chown`/`chmod`（filebrowser） | 容器 UID 与数据目录权限对齐 | **不**用 shell 脚本平台化；见下文 P-init；复杂权限可走镜像 entrypoint 或后续任务 |

### 关键失败模式（必须在 Plan 内显式处理）

Docker bind mount：若宿主机源路径**不存在**，Docker 常会**创建目录**。  
对 `acme.json` 这类应挂为**文件**的路径，缺 `touch` 会导致 Traefik 启动失败（目录当文件）。  
因此「创建文件型 logical 挂载源」是**部署预检/物化**职责，不是应用特判脚本。

---

## Plan decisions

| # | 决策 | 说明 |
|---|------|------|
| **P-init** | **废除平台对 `init.sh` 的依赖** | 不在 E5 部署闭环中执行/要求 `init.sh`；不把 gateway 目录准备写回 shell。文档中「部署前跑 init.sh」改为挂载物化说明。workspace 对 `init.sh` 的 WriteConfig 特判可保留为无关遗留或后续清理，**不作为正式路径**。 |
| **P-init-1** | **挂载源物化（Mount source materialize）** | Deploy（及需要一致的 Preview 校验）在 `docker compose up` **之前**，对解析后的 **logical** 挂载：目录型 → `MkdirAll`；文件型 → 父目录 `MkdirAll` + 源文件不存在则创建空文件（**不覆盖**已有内容）。**全 kind 共用**，与 Spec 挂载模型一致。 |
| **P-init-2** | **文件 vs 目录判定** | 初版规则（写死一处，可单测）：`target` 或 `source` 带常见文件后缀（如 `.json`、`.yml`、`.yaml`、`.toml`、`.pem`、`.key`、`.crt`、`.conf`）→ **文件型**；否则 **目录型**。禁止依赖 gateway code 特判。若后续要更准，再扩展 `source_kind` 字段（**本期可不加列**，用上述启发式 + 测试钉死 acme.json）。 |
| **P-init-3** | **`acme.json` 权限** | 新建空 `acme.json` 时使用 **0600**（与 Traefik ACME 常见要求一致）。已有文件不改权限、不截断。容器内 `chmod 600` 若仍写在 Version command/entrypoint 中，属于规格内容，平台不强制注入。 |
| **P-init-4** | **dynamic / certs 目录** | 平台 **不**再默认创建 `data/dynamic` 或 `data/certs`。需要持久化目录时，由 VersionComponent **mounts 表单**声明 `logical` 源；物化只跟声明走。平台路由 **不**写入任何应用 data/dynamic。 |
| **P-init-5** | **special / volume** | `special`（如 `docker.sock`）、`volume`：不做宿主机 mkdir/touch。 |
| **P-init-6** | **权限 chown** | E5 **不**实现通用 chown 平台步骤（避免 Windows/Linux 分叉与 root 假设）。filebrowser 类需求：镜像 entrypoint、或 Version 内文档化运维步骤、或后续任务；**不**复活 init.sh 作为平台 SoT。 |
| **P-init-7** | **静态配置文件（如 traefik.yml）** | 内容属于应用规格：**logical 文件 mount + 可选 content**（或导入包）；平台 **不**在部署时根据 kind 生成 gateway 专用静态树。须启用 `providers.rest` 的约束写在运维/规格层（文档 + 示例 Version content），非代码魔法注入。 |
| **P-init-8** | **文件 mount content** | Schema：`content?`、`content_mode?: seed\|sync`（默认 seed）；仅 logical 文件型；上限 256KiB。物化：seed=仅不存在时写；sync=每次覆盖；无 content 则 touch 空文件且不覆盖已有。UI：文件型 logical 可编辑正文与 mode。 |
| **P1** | 挂载 schema | Spec D1：`source_type` ∈ `logical`\|`volume`\|`special`；`source`/`target`/`read_only`；`content`/`content_mode`；`special` 初版仅 `docker.sock`。 |
| **P2** | env + RuntimeConfig | Spec D2：env 数组 `[{key,value}]`；占位 `${NAME}` / `${NAME:-default}`；Deploy 带 `runtime_config`；缺必填 4xx + 缺失键列表；落 Deployment options/JSON。 |
| **P3** | Render | Preview 与 Deploy **同一**挂载/env 解析函数；失败明确报错。 |
| **P4** | rest 路由 | `RouteManager`：Sync/Revoke/对账 → 重建全集 → **一次** `PUT {api_url}/api/providers/rest`；删除写 dynamic yaml + HUP 促热更路径；`ListRouters` 仍走只读 API。 |
| **P5** | 配置清理 | 去掉「`dynamic_route_dir` 作为平台写路由 SoT」的用法；配置键可删或降级为 unused 并在实现中删引用（开发期直接删，不做兼容别名）。 |
| **P6** | 单 gateway | 已有 **Running** 的 `kind=gateway` Service 时，拒绝再部署另一 gateway；明确错误文案。 |
| **P7** | network | 仅 gateway Render 注入 top-level networks；默认 name `traefik`；Environment 可覆盖（字段名实现时钉死一处）。 |
| **P8** | UI | Version 未发布：mounts（含文件 content/mode）/env/ports 表单；Deploy：RuntimeConfig 表单；Preview：解析后 volumes/environment。 |
| **P9** | 实现顺序 | 与 Spec 一致：**C → D9 → A → B → D4**，并在 **A 的 Render/Deploy 路径接入 P-init-1/P-init-8 物化**（与挂载解析同批）。 |
| **P10** | 测试 | rest PUT 契约（mock HTTP）；File 写路径无调用；单 gateway 拒绝；mounts schema 校验；logical 文件 seed/sync content；无 content 不覆盖已有；directory 型 MkdirAll；RuntimeConfig 缺键失败；gateway network + dashboard labels；无 dynamic 强制路径。 |

---

## init.sh → 新部署模式对照表（实施验收用）

| 旧（临时） | 新（E5） | 负责层 | 验收点 |
|------------|----------|--------|--------|
| 手工/文档要求跑 `init.sh` | 部署流水线自动 **Mount materialize**（含文件 content seed/sync） | Deploy 预检 | 无 init.sh 也能首次部署含 acme / traefik.yml content 的 gateway |
| `mkdir data/dynamic` | **不做**（除非 Version mounts 声明） | — | 平台代码无「为路由 mkdir dynamic」 |
| `mkdir data/certs` | **不做默认**；F1 再谈证书写入 | mounts / F1 | E5 无强制 certs 树 |
| `touch data/acme.json` | logical 文件挂载物化 + 0600 | Mount materialize | 源为文件；二次部署不覆盖已有 ACME 数据 |
| gateway 特殊 data 树 | 与 standard **同一** ServiceDir 策略 | Workspace | 无 kind 分支建子树 |
| File 写 `dynamic/*.yml` | rest 全量 PUT | RouteManager | 无写路由文件 |
| `env_file: .env` 主路径 | RuntimeConfig → environment | Deploy/Render | Preview 无强制 env_file |
| filebrowser `chown` | **不**平台化 shell | 镜像/后续 | Plan 明确 non-goal |

**部署顺序（E5 目标态）**：

```text
① 校验 Version / RuntimeConfig / 单 gateway
② Render(Version × Environment × RuntimeConfig)  → compose YAML
③ 解析 mounts → 物化 logical 源（mkdir / touch，幂等、不覆盖文件内容）
④ WriteConfig docker-compose.yml
⑤ docker compose up
⑥（路由变更时）平台 Route 快照 → PUT rest   # 与 compose 生命周期解耦，但同属 E5 交付面
```

不再存在：`⑤ 若存在 init.sh 则 bash init.sh`。

---

## Implementation steps

### Step 1 — C：平台路由改为 rest（废除 File 写路由）

**目标**：Route 同步唯一写路径 = rest 全量 PUT。

**改动要点**：

- `internal/infrastructure/external/traefik/route.go`（及测试）：  
  - 实现快照组装（routers + services；本期支持的 tls/certResolver 字段按 Spec）。  
  - `Sync` / `Revoke` / 全量对账：重建全集 → **一次 PUT**。  
  - **删除**：`deployRouteFile`、dynamic 目录写 yaml、reload/HUP 促路由热更。  
  - `ListRouters` 保留 GET 只读。  
  - 自定义 PEM 写入（`WriteCertificate`）本期可保留接口但 **不作为路由热更前置**；完整对齐 F1。  
- 配置：`dynamic_route_dir` 引用删除；`api_url` 为 rest 基址。  
- port 接口若暴露「写文件路由」语义，改为「ApplySnapshot / SyncAll」。  
- 串行化：进程内 mutex 或现有锁，保证 rest 唯一写者。

**验证**：单元测试 mock `httptest`；无对 `dynamic_route_dir` 的写文件断言；清空 body 形态符合 Spec。

### Step 2 — D9：单 gateway 部署约束

- Deploy / `upsertServiceDeploying` 前：若目标 `app.kind=gateway`，查询是否已有 **其他** application 的 gateway Service 处于 Running（或 Deploying，策略：Running+Deploying 均拦，避免双开窗口）。  
- 错误信息明确：已有 gateway 运行，本期不支持多实例。  
- 测试：已有 running gateway → 第二次 deploy 失败。

### Step 3 — A：挂载 / 环境变量模型 + Render + UI + **挂载物化（P-init）**

**3.1 模型与校验**

- VersionComponent `mounts_json` / `env_json`：写入时校验 schema（source_type、target 绝对路径、special 白名单、env key/占位语法）。  
- UI：表单编辑，非整段 JSON 唯一入口。  
- ports/command 表单可同期收紧（已有 JSON 则结构化编辑）。

**3.2 Render**

- 统一 `resolveMounts(component, app, env, service, workspace)`：  
  - logical → 物理路径（应用数据根 / ServiceDir 规则与现有 physical path 能力对齐）。  
  - volume / special → 规范 compose volume 项。  
- env：字面量 + 占位（无 RuntimeConfig 时 Preview 可显示占位未解析或标注 required）。  
- **禁止**平台注入 dynamic 路由目录挂载。

**3.3 挂载源物化（取代 init.sh 目录/文件准备）**

- 新函数（建议）：`MaterializeLogicalMountSources(ctx, resolved []ResolvedMount) error`  
  - 目录型：`os.MkdirAll(hostSource, 0o755)`  
  - 文件型：`MkdirAll(parent)`；若 `!exists` 则 `OpenFile` 创建，`acme.json` 或通用文件 **0600**；**exists → no-op**  
  - 失败返回明确错误，部署 faulted  
- 调用点：`renderAndDeploy` 在 `WriteConfig(compose)` 之后、`docker compose up` **之前**（或 WriteConfig 前亦可，但须在 up 前完成；推荐 **Render 成功后、up 前**）。  
- Preview：**不**写盘物化（只读解析展示）；可选 API 提示「首次部署将创建缺失文件源」。

**3.4 Frontend**

- Version 编辑：mounts 行编辑（source_type / source / target / read_only）、env 行编辑。  
- i18n；lint:fix + typecheck。

### Step 4 — B：RuntimeConfig + Deploy

- Deploy 请求体增加 `runtime_config` map。  
- 扫描 Version + Components env 占位 → 必填/默认。  
- 缺必填：4xx + 缺失键列表。  
- 解析结果写入 compose `environment`；options 落本次 Deployment。  
- UI Deploy 表单展示占位。  
- 废止部署主路径对工作区 `.env` / `env_file` 的依赖（渲染不再默认写 env_file）。

### Step 5 — D4：gateway top-level network

- `renderGatewayCompose`：注入  
  ```yaml
  networks:
    <key>:
      name: <resolved>  # 默认 traefik
      driver: bridge
  ```  
  gateway 各 service join 该 network。  
- standard **不**注入。  
- Environment 覆盖字段（实现选一名，如 `gateway_network_name` 或复用已有 ingress 相关配置——**若无列则本期固定默认名 + 配置/Env 扩展点在实现时与 schema 对齐；无则仅默认 `traefik`**）。  
- 与域名后缀禁止混用（测试钉死）。

### Step 6 — 清理与文档

- 删除/改写依赖 File 路由、init.sh 必跑、gateway 强制 dynamic 的测试与注释。  
- 更新 `docs/guides/deployment.md`、volume 相关叙述：部署顺序含 **挂载物化**；标明 init.sh 为历史临时手段、非平台路径。  
- cadence：E5 Plan/实现状态；F1–F7 仍挂后续。  
- 本地 `data/cd/**/init.sh` **不**作为仓库 SoT；用户数据目录可保留但不被代码调用。

### Step 7 — 测试与检查命令

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

---

## Files to change（预期）

| 区域 | 路径（示意） |
|------|----------------|
| 路由 | `internal/infrastructure/external/traefik/*` |
| 配置 | `internal/config/config.go`、`configs/config.yaml`、相关 test |
| Render / Deploy | `internal/application/cd/usecase/compose_renderer.go`、`deployment_execution.go`、新 `mount_*.go` |
| DTO / proto | Deploy runtime_config；VersionComponent mounts/env 结构化 API |
| Repo / model | Deployment options；若 env 序列化格式变更则读写对齐 |
| Port | Traefik client 接口语义 |
| Frontend | Version 表单、Deploy RuntimeConfig、Preview |
| Docs | guides；本 feature requirement/spec 交叉引用；cadence |
| Tests | compose、deploy、traefik rest、单 gateway、materialize |

---

## Verification plan

| # | 验收项 | 方式 |
|---|--------|------|
| V1 | rest PUT 全量；无 dynamic 写路由文件 | 单测 + 代码检索无 File 写路径 |
| V2 | 单 gateway 二次部署拒绝 | 单测 / 集成 |
| V3 | mounts 表单 schema；非法 source_type 拒绝 | 单测 + UI |
| V4 | logical 文件型首次部署创建文件（如 `…/acme.json`），二次不覆盖内容 | 单测 temp dir |
| V5 | logical 目录型 MkdirAll | 单测 |
| V6 | 平台不强制创建 dynamic/certs | 单测 / 无代码路径 |
| V7 | RuntimeConfig 缺键失败；有默认填充 | 单测 |
| V8 | gateway compose 含 top-level network 默认 traefik；standard 无 | compose 单测 |
| V9 | Preview/Deploy 同解析函数 | 单测或共享 helper 断言 |
| V10 | 部署日志体现物化步骤（可选） | 日志断言或手工 |
| V11 | 前端 lint/typecheck | yarn |
| V12 | 无 init.sh 执行步骤 | 部署路径代码审查 + 测试无 bash init |

---

## Assumptions

1. logical 挂载物理根 = 现有 CD workspace / physical data root 规则（与 P2 物理路径一致），物化写在解析后的**真实宿主机路径**。  
2. 文件/目录启发式后缀表足够覆盖 acme.json 与常见配置文件；特殊需求后续再加显式字段。  
3. Environment 覆盖 gateway 网络名：若库表暂无字段，本期默认 `traefik` 硬编码常量 + 预留读取点，不阻塞 D4。  
4. rest 管理面本机/管理网可达；鉴权 F2。  
5. Traefik 镜像 ≥ 3.6 且静态配置已启用 `providers.rest`（规格/文档责任，非平台运行时改写静态文件）。

## Risks

| 风险 | 缓解 |
|------|------|
| rest 全量 PUT 丢路由 | 唯一写者 + 完整快照重建；测试 |
| 文件/目录误判导致 acme 被建成目录 | 后缀启发式 + acme 专用单测；文档说明 |
| 物化写盘权限不足 | 明确错误；部署 faulted |
| 用户仍依赖旧 init.sh 习惯 | 文档更新；部署不再需要 |
| 双轨 File+rest | Step1 删 File 写路径；CI 检索 |
| 单 gateway 校验竞态 | Deploying 状态一并拦截 |

## Rollback

- 开发期分支回退；无生产迁移承诺。  
- 若 rest 不可用：不自动回退 File 写路由（产品已否决双轨）；修复 rest 或暂停路由同步。  
- 挂载物化失败不影响已运行容器，仅使本次 deploy faulted。

## Out of scope（再次钉死）

- **E6** 网关领域配置 + 专用渲染（独立扩展；本 Plan 不交付）  
- F1 自定义 PEM 推送与 certs 目录对齐  
- F2 rest 鉴权  
- F3 多 gateway  
- F5 dashboard  
- 通用 chown / 复活 init.sh 平台契约  
- Version 巨型 Traefik JSON  


## User review notes

- 用户要求：进入 Plan，并**把原 init.sh 行为在新部署模式下如何实施策划清楚**；原实现为临时举措。  
- 本 Plan 以 **P-init / 挂载源物化** 承接 acme/目录准备；dynamic 路由目录准备**显式废弃**；不执行 init.sh。  

---

**当前：严格模式 / strict，计划 / Plan — Review status: Draft**

请审阅本 Plan，重点确认：

1. **挂载源物化**是否可完全替代 init.sh（含 acme.json 0600、不覆盖）。  
2. 文件 vs 目录的**后缀启发式**是否可接受（或要求本期就加显式字段）。  
3. **不**平台化 chown 是否接受。  
4. 实现顺序 C → D9 → A(+物化) → B → D4 是否同意。

确认后将 Plan 标为 Accepted，再进入 Implementation / 实现。
