# Backend Go 配置加载职责收敛验证
最后修改时间: 2026-06-24 11:04:50

## Review status

Accepted

## Requirement alignment

依据 `docs/requirement/20260624-backend-go-config-loading.md` 核对：

- 已将配置加载策略收敛到 `backend-go/internal/config`：`config.Load()` 不再接收外部配置路径参数。
- `config.defaults.yaml` 仍为必读 baseline，读取失败会返回 `read default config` 错误。
- `config.local.yaml` 由配置模块内部尝试合并；缺失时忽略，存在但读取或解析失败时报 `read local config`。
- 环境变量绑定逻辑未移除，仍通过 `bindEnv` 覆盖配置。
- Air API 和 worker 启动参数不再传入 `--config config.local.yaml`。
- 数据库 driver 和 sqlite path 的 Go 代码级静默补默认值已移除，缺失或空值改由 `Validate()` 暴露。

## Spec alignment

不适用。轻量模式 / light 未创建独立 spec 文档，按 requirement / 需求核对。

## Plan alignment

不适用。轻量模式 / light 未创建独立 plan 文档，按 requirement / 需求核对。

## Actual diff summary

实际变更文件：

- `backend-go/internal/config/config.go`
  - `Load(path string)` 改为 `Load()`。
  - 固定读取 `DefaultConfigFile`，再尝试合并 `LocalConfigFile`。
  - 新增 `isLocalConfigMissing`，仅忽略本地配置文件不存在。
  - 移除 `database.driver` 和 `database.sqlite.path` 的代码级补默认值。
- `backend-go/cmd/backend-go/main.go`
  - 删除 `--config` flag。
  - 调用 `config.Load()`。
- `backend-go/.air.api.toml`
  - API 启动参数改为 `args_bin = ["serve"]`。
- `backend-go/.air.worker.toml`
  - Worker 启动参数改为 `args_bin = ["worker"]`。
- `backend-go/internal/config/config_test.go`
  - 测试改为通过固定 `config.local.yaml` 覆盖默认配置。
  - 新增非法本地配置失败测试。
  - 将原先 sqlite path 派生测试改为 sqlite path 必填校验。
- `backend-go/internal/app/mysql_e2e_test.go`
  - MySQL e2e 测试改为在临时目录组合默认配置和外部本地覆盖，避免依赖 `Load(path)`，也避免污染真实 `config.local.yaml`。
- `docs/requirement/20260624-backend-go-config-loading.md`
  - 记录本次轻量需求。

## Expected vs actual changed files

| 文件 | 预期 | 实际 | 结论 |
| --- | --- | --- | --- |
| `backend-go/internal/config/config.go` | 需要 | 已修改 | 符合 |
| `backend-go/cmd/backend-go/main.go` | 需要 | 已修改 | 符合 |
| `backend-go/.air.api.toml` | 需要 | 已修改 | 符合 |
| `backend-go/.air.worker.toml` | 需要 | 已修改 | 符合 |
| `backend-go/internal/config/config_test.go` | 需要 | 已修改 | 符合 |
| `backend-go/internal/app/mysql_e2e_test.go` | 需要同步 `Load` 调用 | 已修改 | 符合 |
| 前端文件 | 不应修改 | 未修改 | 符合 |
| Git 写操作 | 不应执行 | 未执行 | 符合 |

## Acceptance criteria checklist

- [x] Air API 启动参数不再包含 `--config config.local.yaml`。
- [x] Air worker 启动参数不再包含 `--config config.local.yaml`。
- [x] `config.Load` 不依赖调用方传入本地配置路径即可执行默认配置、本地配置、环境变量合并。
- [x] `config.local.yaml` 缺失时不会报 `open config.local.yaml`；默认配置加载测试覆盖无本地配置文件场景。
- [x] `config.defaults.yaml` 缺失或读取失败仍然报错；默认配置仍通过 `ReadInConfig()` 必读。
- [x] `config.local.yaml` 存在但不可解析或不可读取时仍然报错；新增非法本地配置测试覆盖解析失败。
- [x] 数据库 driver、sqlite path 等普通配置不再由 Go 代码静默补默认值；空 sqlite path 测试改为期待错误。
- [x] 相关后端检查通过。

## Test results

已运行：

```powershell
go -C "backend-go" test ./internal/config ./cmd/backend-go ./internal/app
```

结果：

```text
ok  	backend/internal/config	(cached)
?   	backend/cmd/backend-go	[no test files]
ok  	backend/internal/app	(cached)
```

补充说明：实现阶段曾先在仓库根目录运行 `go test ./internal/config ./cmd/backend-go`，因 Go module 位于 `backend-go/` 下失败；随后改用 `go -C "backend-go" ...` 通过。该失败属于命令工作目录错误，不是代码失败。

前端检查未运行，因为本次实际 diff 未修改前端文件。

## Missed or expanded scope

- 未发现超出本次需求的前端、数据库迁移或业务逻辑修改。
- `--config` flag 被直接删除，属于需求中“不依赖外部传递本地配置路径”的直接落地；若外部未纳入仓库的脚本仍使用该 flag，需要调用方同步调整。

## Risks

- `config.Load()` 现在固定基于当前工作目录读取 `config.defaults.yaml` 和可选 `config.local.yaml`；启动时仍需保证工作目录与配置文件位置一致。
- 由于不再静默补数据库默认值，未来默认配置或本地覆盖如果把必填项清空，会更早失败。这符合需求，但可能暴露已有外部配置问题。

## Incomplete items

暂无。

## Conclusion

验证通过。本次实现符合轻量需求：配置加载职责已收敛到 `config` 包内部，默认配置必读、本地配置可选覆盖、环境变量保留覆盖能力，Air 不再强制传入 `config.local.yaml`，普通业务配置不再由 Go 代码静默补默认值。