# backend-go 环境配置统一规格
最后修改时间: 2026-06-24 16:18:16

Review status: Accepted

## Requirement basis

基于已接受的需求文档：

- `docs/requirement/20260624-backend-go-env-config.md`

关键要求：

- 使用严格模式 / strict 推进。
- 统一 backend-go 配置优先级：`config.defaults.yaml < config.local.yaml < .env < OS env`。
- 开发阶段 `.env` 使用 `backend-go/.env`；镜像阶段 `.env` 使用 `<WORKDIR>/.env`，当前为 `/app/.env`。
- `.env` 不得覆盖已存在的真实 OS env。
- Settings 服务读写的 `.env` 与启动时加载的 `.env` 保持同一入口。
- Settings 服务写 `.env` 后能简单可靠热更新就热更新，否则保持现有写文件、重启后生效语义。
- 部署脚本或部署文档不再遗漏新增 `POMELO_ORBIT_...` 配置。
- Turnstile 生产 key 通过 `.env` 或平台环境变量覆盖默认测试 key。

## Overview

目标方案将 backend-go 的运行配置明确为四层：

```text
config.defaults.yaml
< config.local.yaml
< .env
< OS env
```

实现方式：

1. backend-go 启动时先确定当前运行目录下的 `.env` 路径。
2. 使用不覆盖已有 OS env 的 dotenv 加载语义，将 `.env` 中的变量注入当前进程环境。
3. 保持当前 Viper `BindEnv` 机制，由 `POMELO_ORBIT_...` 环境变量覆盖 YAML 配置。
4. 在 `config.Config` 中记录本次启动使用的 `.env` 路径，Settings 服务使用该路径读写 `.env`，不再基于 `orbit.root` 另行推导。
5. Settings 服务维持文件级配置编辑能力；本次不做全局运行时热更新，避免对 HTTP server、数据库、JWT、worker、Turnstile verifier 等已初始化组件引入不一致状态。
6. 部署脚本传输远端 `.env` 时，从 `scripts/.env` 中筛选应用配置环境变量，而不是只写少量白名单。

## Design decisions

### 1. `.env` 路径规则

`.env` 路径与进程工作目录绑定：

- 开发阶段：从 `backend-go` 目录运行，读取 `backend-go/.env`。
- 镜像阶段：Dockerfile `WORKDIR /app`，读取 `/app/.env`。

不额外查找仓库根目录 `.env`，避免重新引入多位置配置源。

### 2. 配置优先级

最终优先级为：

```text
config.defaults.yaml < config.local.yaml < .env < OS env
```

技术上通过以下顺序达成：

1. 在 `config.Load()` 早期加载 `.env`，加载语义为“仅设置当前 OS env 中不存在的变量”。
2. Viper 读取 `config.defaults.yaml`。
3. Viper 合并 `config.local.yaml`。
4. Viper 通过已绑定的 `POMELO_ORBIT_...` 环境变量覆盖 YAML 值。

因为 `.env` 被加载到进程环境，且加载时不覆盖已有 OS env，所以 OS env 天然高于 `.env`。

### 3. dotenv 实现选择

采用主流 Go dotenv 加载库 `github.com/joho/godotenv` 的 `Load` 语义，避免自行维护解析器并支持常见 `.env` 格式。

约束：

- 不使用 `Overload` 或其他会覆盖已有 OS env 的机制。
- `.env` 缺失不视为错误。
- `.env` 存在但格式错误时应返回配置加载错误，避免静默忽略错误配置。

需要新增依赖：

- `backend-go/go.mod`
- `backend-go/go.sum`

### 4. `Config` 记录 env file path

在 `config.Config` 增加运行时字段，例如：

```go
type Config struct {
    ...
    EnvFilePath string `mapstructure:"-" yaml:"-"`
}
```

语义：

- `EnvFilePath` 是启动时解析出的 `.env` 绝对路径。
- 不参与 YAML / mapstructure 解析。
- Settings 服务使用该路径读写 `.env`。

这样 Settings 不再通过 `cfg.OrbitRoot()` 推导 `.env`，避免 `orbit.root` 改变后 Settings 写入路径与启动加载路径不一致。

### 5. Settings 服务热更新策略

本次不实现全局热更新。

原因：

- 当前 `Server`、auth handler、Turnstile verifier、settings service、database connection、worker router 等组件都在启动时基于 `config.Config` 初始化。
- Settings 写 `.env` 后若只局部更新某些字段，容易造成同一进程内不同组件配置不一致。
- 数据库 DSN、JWT secret、server host/port、worker concurrency 等配置不是简单安全的运行时可变项。

规格要求：

- Settings 服务写入或重置 `.env` 后，返回的配置列表应反映 `.env` 文件中的覆盖值。
- 当前进程已初始化组件不承诺立即采用新值。
- 文档或 Settings 配置描述中明确“配置修改后需要重启 backend-go 生效”。

### 6. `.env.example`

新增 `backend-go/.env.example`，作为 backend-go 的正式 `.env` 模板。

内容覆盖：

- JWT secret
- server host / port 示例
- database driver / sqlite path / mysql dsn 示例
- Traefik API / domain suffix 示例
- Turnstile enabled / site key / secret key / verify URL 示例
- Let's Encrypt 示例
- worker 示例

Turnstile 部分必须明确：

- 默认 `config.defaults.yaml` 使用 Cloudflare test keys。
- 生产必须覆盖为真实 site key / secret key。
- secret key 不得提交到仓库。

### 7. 部署脚本 `.env` 传输

`scripts/manage.py` 当前 `_deploy_env()` 只传：

- `POMELO_ORBIT_JWT__SECRET_KEY`
- 可选 `POMELO_ORBIT_TRAEFIK__API_URL`

目标改为：

- 读取 `scripts/.env` 中所有应用配置变量。
- 写入远端 `${REMOTE_DEPLOY_DIR}/.env`。
- 不写入 SSH、GHCR、REMOTE_DEPLOY_DIR、SOURCE_REPO 等脚本自身配置。

应用配置变量筛选规则：

- key 以 `POMELO_ORBIT_` 开头；
- 排除脚本本地元配置 `POMELO_ORBIT_IMAGE`，因为它不是 backend-go 配置项，只用于选择部署镜像。

必填校验：

- 继续要求 `POMELO_ORBIT_JWT__SECRET_KEY` 存在。
- 不强制要求 Turnstile key，因为可通过平台 OS env 注入；但 `scripts/.env.example` 应给出配置项示例。

输出顺序：

- 保持稳定排序，减少远端 `.env` diff 噪声。

### 8. `scripts/.env.example`

更新 `scripts/.env.example`：

- 保留 SSH / remote deploy / GHCR 等部署脚本配置。
- 增加应用 `.env` 相关示例，至少包括：
  - `POMELO_ORBIT_TURNSTILE__ENABLED`
  - `POMELO_ORBIT_TURNSTILE__SITE_KEY`
  - `POMELO_ORBIT_TURNSTILE__SECRET_KEY`
  - `POMELO_ORBIT_TURNSTILE__VERIFY_URL`
  - 数据库和证书相关可选配置
- 标注 `POMELO_ORBIT_IMAGE` 是脚本本地变量，不会写入远端应用 `.env`。

## Affected components

### Backend config

- `backend-go/internal/config/config.go`
  - 加载 `.env`。
  - 记录 `EnvFilePath`。
  - 保持现有 `BindEnv`。
- `backend-go/internal/config/config_test.go`
  - 增加 `.env` 缺失、`.env` 覆盖、OS env 优先、`config.local.yaml < .env`、Turnstile 从 `.env` 覆盖等测试。
- `backend-go/go.mod`
- `backend-go/go.sum`
  - 新增 dotenv 依赖。

### Settings

- `backend-go/internal/service/settings/service.go`
  - `envPath()` 改用 `cfg.EnvFilePath`。
  - 配置项描述中明确重启后生效语义。
- `backend-go/internal/transport/http/settings_routes_test.go`
  - 调整测试使用 `cfg.EnvFilePath`，确认读写路径与启动 `.env` 一致。

### Env templates and deployment

- `backend-go/.env.example`
  - 新增 backend-go 正式 `.env` 模板。
- `scripts/manage.py`
  - `_deploy_env()` 改为传输所有应用配置 `POMELO_ORBIT_...`，排除脚本本地变量。
- `scripts/.env.example`
  - 补齐 Turnstile 等应用配置示例。

### Optional documentation

- 如 README 或部署文档已有配置章节，则补充 backend-go 配置优先级和 `.env` 位置规则。

## Interfaces

### `.env` key 形式

继续沿用现有环境变量命名：

```env
POMELO_ORBIT_TURNSTILE__ENABLED=true
POMELO_ORBIT_TURNSTILE__SITE_KEY=your-site-key
POMELO_ORBIT_TURNSTILE__SECRET_KEY=your-secret-key
POMELO_ORBIT_TURNSTILE__VERIFY_URL=https://challenges.cloudflare.com/turnstile/v0/siteverify
```

不新增兼容别名。

### Settings API

现有 API 保持不变：

- `GET /api/settings/config`
- `PUT /api/settings/config`
- `DELETE /api/settings/config`

响应结构不强制变更。若实现阶段选择增加“重启后生效”提示，应优先通过 description 文案表达，避免改变 API contract。

## Technical questions

暂无阻塞实现的问题。

实现阶段需要注意：

- 测试中修改工作目录或进程环境时必须隔离，避免 `.env` 加载污染其他测试。
- 如果使用 `godotenv.Load` 会修改进程环境，测试应使用唯一变量值并用 `t.Setenv` / 清理逻辑控制优先级。
- Windows 路径下 `EnvFilePath` 应使用 `filepath.Abs` / `filepath.Clean`，不要硬编码 POSIX 路径。

## Risks

- `.env` 加载会修改当前进程环境，测试需要严格隔离。
- Settings 服务不热更新运行时配置，用户需要理解“写入后重启生效”；否则可能误判配置已立即生效。
- `scripts/manage.py` 传输所有应用 `POMELO_ORBIT_...` 后，若 `scripts/.env` 中存在拼错的应用变量，也会进入远端 `.env`；这比静默丢配置更可见，但可能在 Settings 页面显示额外项。
- `POMELO_ORBIT_IMAGE` 虽然使用相同前缀，但不是应用配置，必须明确排除，避免 Settings 页面出现无关配置。

## Alternatives

### 1. 不引入依赖，复用自研 `.env` parser

不采用为默认方案。当前 Settings parser 只支持非常简单的 `KEY=value`，不支持主流 `.env` 常见格式。为了符合主流实践，优先采用 `godotenv.Load`。

### 2. 同时查找仓库根目录 `.env` 和 `backend-go/.env`

不采用。多路径查找会重新引入 `.env` 定位分裂。需求已明确开发阶段使用 `backend-go/.env`，镜像阶段使用 `<WORKDIR>/.env`。

### 3. Settings 服务立即热更新所有配置

不采用。当前架构中多个组件在启动时捕获配置，强行热更新会引入不一致和长期技术债。本次保持写文件、重启后生效语义。

### 4. `.env` 覆盖 OS env

不采用。生产平台注入的环境变量必须优先，避免本地文件覆盖 secret manager、CI/CD 或容器平台配置。

## User review notes

- 用户确认进入 Spec / 规格阶段。
