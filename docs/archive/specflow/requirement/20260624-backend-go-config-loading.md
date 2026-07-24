# Backend Go 配置加载职责收敛
最后修改时间: 2026-06-24 10:51:12

## Review status

Accepted

## Background

当前 Air 启动配置显式传入 `--config config.local.yaml`，导致 `config.local.yaml` 不存在时启动失败：

```text
ERROR load config failed error="read config: open config.local.yaml: The system cannot find the file specified."
```

用户确认两条原则：

1. Viper 应该自行读取文件并合并配置，不依赖外部启动命令传递本地配置路径。
2. 默认配置即使值不适合真实环境，也应该覆盖完整配置结构，等待用户本地配置或环境变量覆盖。

## Goal

- 将配置加载策略收敛到 `backend-go/internal/config` 内部。
- 默认配置 `config.defaults.yaml` 必须读取，作为完整 baseline。
- 本地配置 `config.local.yaml` 可选存在，存在时覆盖默认配置，不存在时不报错。
- 环境变量继续作为覆盖来源。
- 启动配置不再通过 Air 强制传入 `--config config.local.yaml`。
- 移除普通业务配置的代码级补默认值，缺失或非法配置由校验暴露。

## Non-goal

- 不引入兼容层或多套新旧配置路径。
- 不修改配置项语义或新增功能。
- 不调整前端代码。
- 不提交 git commit 或执行任何 git 写操作。

## User scenarios

- 开发者没有创建 `backend-go/config.local.yaml` 时，使用默认配置仍可启动或执行测试，不再因为缺少本地文件直接失败。
- 开发者创建 `backend-go/config.local.yaml` 时，其中字段覆盖默认配置。
- 本地配置文件存在但格式错误或值非法时，启动明确失败。

## Acceptance

- Air API 启动参数不再包含 `--config config.local.yaml`。
- Air worker 启动参数不再包含 `--config config.local.yaml`。
- `config.Load` 不依赖调用方传入本地配置路径即可执行默认配置、本地配置、环境变量合并。
- `config.local.yaml` 缺失时不会报 `open config.local.yaml`。
- `config.defaults.yaml` 缺失或读取失败仍然报错。
- `config.local.yaml` 存在但不可解析或不可读取时仍然报错。
- 数据库 driver、sqlite path 等普通配置不再由 Go 代码静默补默认值。
- 相关后端检查通过，若因环境限制无法运行需说明。

## Open questions

暂无。

## Decisions

- 本次按轻量模式直接进入实现，Requirement 记录为 `Accepted`。
- 本地配置文件名固定为 `config.local.yaml`，由配置模块内部处理。

## Risk

- 如果外部脚本仍传入 `--config`，需要确认是否保留该 flag 或同步调整调用方；本次优先让 Air 不再传本地配置路径，并让默认 `Load` 路径自行处理。
- 如果测试或其他代码依赖 `Load(path)` 注入临时配置，需要同步调整测试入口或保留明确用途的可测试 API。