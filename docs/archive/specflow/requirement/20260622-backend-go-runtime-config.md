# backend-go 运行时配置入口收敛
最后修改时间: 2026-06-22 11:05:15

Review status: Accepted

## Background

当前 Go 后端已经通过 `backend-go/internal/config/config.go` 使用 Viper 统一加载 `backend-go/config.defaults.yaml`、可选配置文件和 `POMELO_ORBIT_*` 环境变量覆盖。

但开发启动入口仍存在重复配置：

- `justfile` 的 `dev-backend` 在命令前缀中显式设置 `POMELO_ORBIT_SERVER__HOST` 和 `POMELO_ORBIT_SERVER__PORT`。
- `scripts/dev.py` 在聚合启动后端 API 和 worker 时硬编码注入同一组 server host/port 环境变量。
- `backend-go/config.defaults.yaml` 已定义 `server.host` 和 `server.port`，HTTP server 也已经从 `config.Config.Server` 派生监听地址。

这会让默认启动行为分散在多个入口里。后续如果继续以这种方式添加环境变量，容易形成绕过 Viper 配置系统的第二套配置来源。

## Requirement 阶段检查结果

### Go 后端运行时环境变量入口

已检查 `backend-go/**/*.go` 中的 `POMELO_ORBIT_*`、`BACKEND_GO_*`、`os.Getenv`、`os.LookupEnv` 和测试注入：

- 应用运行时配置的 `POMELO_ORBIT_*` 入口集中在 `backend-go/internal/config/config.go` 的 `bindEnv`。
- `backend-go/internal/config/config_test.go` 使用 `t.Setenv` 覆盖配置，属于配置加载测试。
- `backend-go/internal/service/settings/service.go` 使用 `POMELO_ORBIT_` 前缀读写 `${orbit.root}/.env`，属于配置管理能力，不是 HTTP server / worker 启动旁路。
- `backend-go/internal/app/mysql_e2e_test.go` 读取 `BACKEND_GO_E2E_CONFIG`，只用于选择 MySQL e2e 测试配置文件，不属于应用运行时配置。
- 未发现 `cmd/backend-go`、`internal/app`、`internal/transport/http` 或 worker 启动路径通过 `os.Getenv` / `os.LookupEnv` 读取 server host/port 或其他应用运行时配置。

### 已发现的配置 key 不一致

- `backend-go/internal/config/config.go` 的 `bindEnv` 当前绑定了 `repository.root`，但实际配置结构和默认配置使用的是 `orbit.root`：
  - `Config.Orbit.Root` 的 `mapstructure` key 是 `orbit.root`。
  - `backend-go/config.defaults.yaml` 使用 `orbit.root`。
  - `Config.OrbitRoot()` / `Config.DataRoot()` 都从 `cfg.Orbit.Root` 派生。
- 因此当前应支持的环境变量应是 `POMELO_ORBIT_ORBIT__ROOT`，而不是没有对应配置结构的 `POMELO_ORBIT_REPOSITORY__ROOT`。
- 实现时应直接将 `repository.root` 修正为 `orbit.root`，不添加兼容别名。

### `.env` 与 Viper 的边界

- `settings.Service` 会读写 `${orbit.root}/.env` 中的 `POMELO_ORBIT_*` key。
- 当前 `config.Load` 只读取 `config.defaults.yaml`、可选 `-config` 文件和进程环境变量；它没有直接解析 `.env` 文件。
- 这意味着 `.env` 是否生效取决于外层启动方式是否把它加载为进程环境变量，例如 Docker Compose `--env-file`。本需求不静默引入新的 dotenv 加载优先级；如要把 `.env` 纳入 Go 进程自身的 Viper 加载链，应作为单独设计点确认。

### settings definitions 一致性

- `settingsDefinitions` 当前暴露的是可通过系统配置页面管理的一组选定 key，不是 `Config` 全量字段清单。
- 但它已经暴露 `server__host`、`server__port`、logging、database、jwt、部分 traefik、turnstile、cert 和 worker 配置。
- 本次实现只要求它不要使用不存在或错误命名的配置 key；是否扩展为全量配置管理不纳入本需求。

## Goal

- 让 Go 后端本地默认启动行为以 `backend-go/config.defaults.yaml` 为准。
- 移除 `justfile` 和 `scripts/dev.py` 中重复注入的 `POMELO_ORBIT_SERVER__HOST` / `POMELO_ORBIT_SERVER__PORT`。
- 保留并复用 Viper 的 `POMELO_ORBIT_*` 环境变量覆盖能力。
- 检查 Go 后端相关环境变量入口，确认运行时配置不在启动层、HTTP server 层或 worker 层通过 `os.Getenv` / `os.LookupEnv` 旁路读取。
- 修正 `bindEnv` 中与实际配置结构不一致的环境变量 key，尤其是 `repository.root` 与 `orbit.root` 的不一致。

## Non-goal

- 不修改旧 Python backend 的 Dynaconf 配置体系。
- 不重构 `scripts/manage.py` 的部署管理变量读取；它读取 `scripts/.env` 并用于部署/生成远程 `.env`，不属于 Go 后端本地启动绕过 Viper 的问题。
- 不新增 `--host` / `--port` 等命令行参数，避免形成第二套配置入口。
- 不主动启动、停止或重启开发服务器。
- 不引入兼容层、别名、默认值兜底分支或新旧配置并存逻辑。
- 不在本次变更中新增 Go 进程对 `.env` 文件的直接加载优先级。
- 不把系统配置页面扩展为所有 `Config` 字段的全量管理界面。

## User scenarios

- 开发者执行 `just dev-backend` 时，后端监听地址来自 Go 配置默认值，而不是 just 命令行硬编码环境变量。
- 开发者执行 `just dev` 时，后端 API 和 worker 不依赖 `scripts/dev.py` 注入 server host/port；默认配置仍能支持前端 Vite proxy 访问 `http://127.0.0.1:9001`。
- 需要临时覆盖配置时，开发者仍可通过 `POMELO_ORBIT_*` 环境变量覆盖，但该覆盖必须经由 Viper `config.Load` 进入 `config.Config`。
- 需要覆盖数据根目录时，开发者应使用与配置结构一致的 `POMELO_ORBIT_ORBIT__ROOT`。
- 系统配置页面读写 `${orbit.root}/.env` 时，展示和写入的 key 与 Go 配置结构保持一致。

## Acceptance

- `backend-go/config.defaults.yaml` 明确提供本地开发默认监听地址，`server.host` 与前端代理目标一致。
- `justfile` 不再设置 `POMELO_ORBIT_SERVER__HOST` / `POMELO_ORBIT_SERVER__PORT`。
- `scripts/dev.py` 不再为后端 API / worker 硬编码注入 server host/port 环境变量。
- Go 后端启动监听地址仍只通过 `config.Load` -> `config.Config.Server` -> HTTP server 地址派生。
- `backend-go/internal/config/config.go` 绑定 `orbit.root`，不再绑定不存在的 `repository.root`。
- Viper env override 测试覆盖 `POMELO_ORBIT_SERVER__HOST`、`POMELO_ORBIT_SERVER__PORT` 和 `POMELO_ORBIT_ORBIT__ROOT`，证明环境变量仍通过配置系统生效。
- 静态搜索可区分运行时配置、测试、部署脚本和历史文档；本次不误改旧后端或部署管理脚本。
- `cd backend-go && go test ./...` 通过。

## Open questions

- 暂无阻塞本次实现的问题。
- 如果后续希望 Go 进程自身读取 `${orbit.root}/.env`，需要单独定义配置加载优先级，再纳入 Viper 加载链。
- 如果后续希望系统配置页面展示并管理所有 `Config` 字段，也应作为单独需求处理。

## Decisions

- 默认开发监听地址采用 `127.0.0.1:9001`，保持与当前 `justfile` / `scripts/dev.py` 实际行为以及 `frontend/vite.config.ts` 代理目标一致。
- 环境变量覆盖继续使用现有 `POMELO_ORBIT_` + 双下划线嵌套 key 规则，并统一进入 Viper。
- `BACKEND_GO_E2E_CONFIG` 仅作为测试选择配置文件的控制变量，不视为应用运行时配置旁路。
- `repository.root` 是错误 key，直接修正为 `orbit.root`，不保留兼容别名。

## Risk

- `server.host` 从 `localhost` 改为 `127.0.0.1` 后，依赖 IPv6 localhost 或主机名解析差异的本地环境可能表现不同；本项目开发代理当前明确指向 `127.0.0.1:9001`，风险可接受。
- 移除启动脚本注入后，如果某个开发者依赖 shell 环境外的隐式覆盖，需要改为配置文件或显式 `POMELO_ORBIT_*` 环境变量覆盖。
- `orbit.root` env key 修正后，如果外部曾误用 `POMELO_ORBIT_REPOSITORY__ROOT`，将不再有效；该 key 与配置结构不匹配，本项目不保留兼容别名。
- 静态搜索会命中旧 Python backend、历史文档和部署脚本，实施时需要按范围筛选，避免扩大改动。