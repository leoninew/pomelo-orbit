# backend-go 环境配置统一需求
最后修改时间: 2026-06-24 15:39:35

Review status: Accepted

## Background

当前旧 Python backend 使用 `.env.example -> .env` 的配置体验；`backend-go` 当前配置来源为 `config.defaults.yaml`、可选 `config.local.yaml`、以及进程环境变量 `POMELO_ORBIT_...`。

代码上，`backend-go/internal/config/config.go` 已通过 Viper 绑定 `POMELO_ORBIT_...` 环境变量，但不会主动读取 `.env` 文件；另一方面，`backend-go/internal/service/settings/service.go` 已读写 `OrbitRoot()/.env`，导致项目里 `.env` 的定位出现割裂：

- `.env` 可被设置服务读写；
- `.env` 不会被 backend-go 启动配置自动加载；
- 部署脚本 `scripts/manage.py` 也只传递部分变量，新增 Turnstile 等配置不会自动进入远端 `.env`。

本任务目标是按主流实践统一 backend-go 配置来源和优先级，恢复并明确 `.env.example -> .env` 的使用心智，同时保留 `config.local.yaml` 和平台环境变量能力。

## Goals

- 明确 backend-go 配置优先级为：`config.defaults.yaml < config.local.yaml < .env < OS env`。
- backend-go 启动时支持读取 `.env`，但不得让 `.env` 覆盖已经存在的真实系统环境变量。
- 明确 `.env` 位置规则：开发阶段使用 `backend-go/.env`；镜像运行阶段使用进程工作目录 `<WORKDIR>/.env`，当前单镜像为 `/app/.env`。
- 保留 `config.local.yaml` 作为结构化本地覆盖文件。
- 保留 Viper 对 `POMELO_ORBIT_...` 环境变量的绑定方式。
- 新增或更新 backend-go 的 `.env.example`，覆盖生产需要配置的关键项，至少包括 JWT、数据库、Traefik、Turnstile、证书相关配置。
- 统一 Settings 服务对 `.env` 的语义：它读写的 `.env` 应与 backend-go 启动时读取的 `.env` 是同一配置入口。
- Settings 服务写入 `.env` 后，如能以简单、可靠、低风险方式热更新则热更新；否则沿用现有“写文件、后续重启生效”的行为，并在文档或界面语义中明确。
- 更新部署脚本或部署文档，避免 `scripts/.env` 中新增 `POMELO_ORBIT_...` 变量无法进入远端 `.env`。
- 对 Turnstile 生产配置给出清晰路径：`.env` / 平台环境变量提供真实 `site_key` 与 `secret_key`，默认测试 key 不作为生产防护。

## Non-goals

- 不引入多套配置命名体系，不新增与 `POMELO_ORBIT_...` 并行的旧式变量名。
- 不让 `.env` 覆盖 Docker、systemd、Kubernetes、CI/CD 等平台已经注入的真实环境变量。
- 不把 secret 推荐写入 `config.defaults.yaml` 或提交到仓库。
- 不恢复旧 Python backend 的配置加载实现。
- 不改变业务 API 行为。
- 不强制支持从仓库根目录直接运行 backend-go；开发阶段以 `backend-go` 工作目录为配置基准。
- 不在本阶段实现代码变更；本阶段只确认需求。

## User scenarios

- 开发者复制 `backend-go/.env.example` 为 `backend-go/.env` 后，从 `backend-go` 工作目录启动 `backend-go serve` 即可读取其中的 `POMELO_ORBIT_...` 配置。
- 生产部署人员在镜像工作目录挂载或写入 `<WORKDIR>/.env` 后，容器内 `backend-go serve` 能读取其中的 Turnstile 真实 key、JWT secret、数据库 DSN 等敏感配置。
- 容器平台或 CI/CD 已注入的环境变量应优先于 `.env` 文件内容，避免部署平台 secret 被文件覆盖。
- 用户在系统设置页修改 `.env` 后，配置文件语义与启动配置入口一致；若无法可靠热更新，应明确重启后生效。
- 使用 `scripts/manage.py install/upgrade` 部署时，`scripts/.env` 中明确声明的 `POMELO_ORBIT_...` 应能进入远端部署 `.env`，不再只传递少量白名单变量导致新配置丢失。

## Acceptance criteria

- backend-go 启动配置加载顺序明确为 `config.defaults.yaml < config.local.yaml < .env < OS env`。
- `.env` 加载使用不覆盖已有环境变量的语义，例如 `godotenv.Load`，不得使用覆盖 OS env 的机制作为默认行为。
- `.env` 缺失时 backend-go 可正常启动并继续使用 `config.defaults.yaml` / `config.local.yaml` / OS env。
- 开发阶段 `.env` 位置为 `backend-go/.env`，与 `backend-go` 工作目录下的 `config.defaults.yaml`、`config.local.yaml` 保持一致。
- 镜像运行阶段 `.env` 位置为 `<WORKDIR>/.env`；当前 Dockerfile 的 `WORKDIR /app` 对应 `/app/.env`。
- `.env` 中的 `POMELO_ORBIT_TURNSTILE__SITE_KEY`、`POMELO_ORBIT_TURNSTILE__SECRET_KEY` 等变量能通过现有 Viper env binding 覆盖配置。
- OS env 与 `.env` 同时存在同名变量时，OS env 优先。
- `config.local.yaml` 与 `.env` 同时配置同一项时，`.env` 优先。
- 新增或更新 `backend-go/.env.example`，包含 Turnstile 生产配置示例，并明确默认 test keys 不适用于生产防护。
- Settings 服务读写的 `.env` 路径与启动时加载的 `.env` 路径保持一致：开发为 `backend-go/.env`，镜像为 `<WORKDIR>/.env`。
- Settings 服务写入 `.env` 后若未实现热更新，应明确当前进程配置需重启后生效；若实现热更新，必须保证配置变更不会破坏现有请求处理与后台任务。
- 部署脚本或部署文档不再遗漏新增 `POMELO_ORBIT_...` 配置，Turnstile 相关变量能够进入生产部署环境。
- 增加配置加载测试，覆盖 `.env`、OS env 优先级、`.env` 缺失、Turnstile 从 `.env` 覆盖、`config.local.yaml < .env` 优先级等场景。

## Open questions

暂无需要用户确认的未决事项。用户已确认：开发阶段使用 `backend-go/.env`，镜像阶段使用 `<WORKDIR>/.env`；Settings 服务写 `.env` 后能热更新就热更新，不能则按现有实现并明确语义。

## Decisions

- 使用严格模式 / strict。
- 采用主流配置优先级：`config.defaults.yaml < config.local.yaml < .env < OS env`。
- `.env` 只作为便利配置源，不应覆盖平台注入的真实环境变量。
- 继续使用 `POMELO_ORBIT_...` 双下划线环境变量命名，不新增兼容别名。
- 开发阶段 `.env` 固定为 `backend-go/.env`。
- 镜像阶段 `.env` 固定为 `<WORKDIR>/.env`，当前为 `/app/.env`。
- Settings 服务优先尝试简单可靠的热更新；如果会引入复杂度或风险，则保持现有写文件行为并明确需要重启生效。

## Risks

- 如果部署脚本继续只白名单少量变量，Turnstile 等新增配置仍可能无法进入生产环境。
- 如果运行中 Settings 服务写入 `.env` 但当前进程不热更新，用户可能误以为配置立即生效；需要在产品语义或文档中说明。
- 引入 `.env` 加载库会增加 Go 依赖，需要确认依赖选择和维护成本。
- 生产环境使用 `.env` 文件存放 secret 时，需要依赖部署目录权限控制。
- 如果未来需要支持从仓库根目录直接启动 backend-go，需要另行设计配置查找路径，避免重新引入多位置 `.env`。

## User review notes

- 用户要求使用严格模式推进。
- 用户要求采用主流实践。
- 用户要求消除“项目里 `.env` 的定位有点分裂”等问题。
- 用户确认开发阶段使用 `backend-go/.env`，镜像使用 `<WORKDIR>/.env`。
- 用户确认 Settings 服务写 `.env` 后能热更新就热更新，不能就按现有实现。
