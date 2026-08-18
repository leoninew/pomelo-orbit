# 可配置 CI/CD 工作目录与 DooD 宿主路径解析计划
最后修改时间: 2026-08-18 22:53:53

Review status: Accepted

Mode: strict

## Basis

- Requirement: [可配置 CI/CD 工作目录与 DooD 宿主路径解析](../requirement/20260818-configurable-ci-cd-workspaces.md)
- Spec：用户明确要求从已接受 Requirement 直接进入 Plan；本计划承担必要的实现设计细化。

## Implementation steps

1. **建立唯一的 workspace 配置契约**
   - 在 `internal/config/config.go` 增加 `WorkspaceConfig`，由 `Config.Workspace` 映射 `workspace.pipeline`、`workspace.deployment`；在 `configs/config.yaml` 显式给出当前布局的基准值 `data/pipeline`、`data/deployment`。
   - workspace 遵循既有 typed config 的统一加载与覆盖规则：默认 YAML、环境 YAML、`.env` / `.env.<env>` 和 OS 环境变量均可配置；在环境模板和 release package 中提供与其他配置相同的变量说明。Settings 不作为通用配置加载入口，不增加 workspace Settings 项。
   - 在 `Load()` 中对运行配置和 `Base` 配置同样执行路径 trim、相对 `orbit.root` 转绝对路径、非空校验和相同/祖先/子孙重叠校验。删除只为旧 workspace 服务的 `DataRoot()` 路径推导，不向业务层提供新旧回退。
   - 移除 `TraefikConfig.CertDir`、`traefik.cert_dir` 环境绑定、默认 YAML、Settings 展示项和相关测试；Gateway 证书目录由配置后的 `workspace.deployment` 单一派生。

2. **将两个 workspace 实现改为根目录语义**
   - 在 `internal/infrastructure/storage/local/pipelineworkspace/` 将构造参数和内部字段收敛为 Pipeline root，删除固定 `pipelineDataDir` 拼接；保留 `<pipeline-root>/<repository-code>/workspace`、`<pipeline-root>/runs/<run-id>/artifacts` 和 stage log 的内部隔离规则。
   - 在 `internal/infrastructure/storage/local/deploymentworkspace/` 将构造参数和内部字段收敛为 Deployment root，删除固定 `deploymentDataDir` 拼接；保留 `<deployment-root>/<service-code>/`、其 `docker-compose.yml`、`deployments/*.log` 与 Component logical mount 的现有层级。
   - 统一接口、fake 和定向测试的命名为 workspace root / physical workspace root；不改变 Service code、Run ID 或 Repository code 的目录身份与清理范围。

3. **在启动路径集中处理 DooD**
   - 复用 `runtimepath.ResolveDockerDaemonPath` 作为唯一的 Docker source 解析器。原生运行直接使用 Orbit 可见路径；仅容器运行时反查 Docker daemon source。
   - 在 `bootstrap` 提取可注入的 workspace-host-path 校验 helper；`serve`、`worker` 和 `mcp` 在创建 HTTP/worker dependencies 前校验 `workspace.pipeline` 与 `workspace.deployment`。裸机返回绝对路径；容器内必须由当前 Orbit 容器的 Docker inspect mounts 反查到宿主 source，否则用包含配置 key 的错误终止启动。
   - 将 pipeline/deployment workspace 都以各自 logical root + 同一 resolver 组装，确保创建、物化写入使用 Orbit 可见路径，而 `docker run --mount` 与 Compose 使用 daemon 可见绝对 host path。
   - 为 Docker inspect mount 的最长 destination 匹配、未映射错误、裸机绝对化与启动校验增加可注入单元测试，避免真实 Docker 依赖。

4. **修复 Gateway 与 Route 的证书目录边界**
   - Gateway usecase 通过 application port 接收 physical-path resolver；创建初始 Traefik Component 时以 `workspace.deployment/traefik/data/certs` 的逻辑路径写入 Route/证书文件，并将解析后的 host path 用于 `source_is_host_path=true` 的 cert 与 ACME mounts。
   - `RouteManager` 仅对同一逻辑派生目录执行 PEM 写入，删除 `data/deployment` fallback 和独立 `traefik.cert_dir` 路径 SoT；更新 Gateway factory/compile 和 Route Manager 的定向测试。
   - 保留 Version 已保存后的普通规格语义：部署/Route 同步不得重写已有 Version 的 mount 定义。

5. **收敛辅助工具、界面与活资料的目录表达**
   - `skills/_ragflow/prepare_ragflow_tei.py` 的 `backup-model` 将 `--archive` 改为必填，删除 `data/backup` 默认及对应测试；模型目录在 docs/skill contracts 中改为 `<workspace.deployment>/<service-code>/...`。
   - `scripts/cert.py` 的 `new`、`check` 接收一致的必填 `--cert-dir`，并同步 `scripts/cert.md` 与证书指南；不在 Python 工具中复制 Go 配置加载逻辑。
   - `scripts/manage.py` backup 接受调用者传入的 workspace exclude paths，删除固定 `./data/{pipeline,ci,deployment}` glob；不推断远程服务器的目录布局。
   - 更新 Repository/Service 删除提示、RAGFlow deployment contracts、配置注释和活文档，统一说明 workspace 配置或显式路径参数；保留历史 archive 文档不改。

6. **验证与文档闭环**
   - Go 定向覆盖：`internal/config`、`internal/bootstrap`、`internal/infrastructure/storage/local`、`internal/infrastructure/external/traefik`、`internal/application/{gateway,deployment,pipeline_run}`；确认 YAML 配置覆盖、根目录冲突、裸机/DooD 物理路径、CI mounts、Deployment Compose/物化和 Gateway cert mount。
   - Python 覆盖 RAGFlow archive 参数、证书 CLI 与 `manage.py` backup 的参数校验；再运行 Python 编译检查。
   - 前端更新后运行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`；后端运行 `task check` 与 `go test ./cmd/... ./internal/...`。
   - 更新 `docs/product/cd-model.md`、`docs/architecture/cd-runtime.md`、`docs/guides/{deployment,ci-pipeline-design,server-deployment,docker-deployment,certificate-management,ragflow-tei-operations}.md`，并执行全仓活动内容扫描，确认没有将 `data/pipeline`、`data/deployment` 或 `data/backup` 作为不可配置路径的残留。

## Files to change

| Area | Primary paths |
| --- | --- |
| Config | `configs/config.yaml`, `.env.example`, `configs/package.env`, `internal/config/{config.go,config_test.go}` |
| Bootstrap and DooD | `internal/bootstrap/{app,http,worker,*_test.go}`, `internal/infrastructure/storage/local/{physical_data_root.go,physical_data_root_test.go}` |
| Workspace storage | `internal/infrastructure/storage/local/{pipelineworkspace,deploymentworkspace}/**` and affected port/fake tests |
| Deployment/Gateway/Route | `internal/application/gateway/{port,usecase}/**`, `internal/infrastructure/external/traefik/{route.go,route_test.go}`, deployment and pipeline-run tests |
| Tools and UI | `skills/_ragflow/**`, `scripts/{cert.py,cert.md,manage.py}`, `web/src/{views/repository,i18n/locales}/**` |
| Documentation | current `docs/product/`, `docs/architecture/`, `docs/guides/`, active skill contracts and this SpecFlow set |

## Risks and rollback

- 更换根目录不会迁移现有数据。发布前须停止相关服务、复制所需 Deployment/Pipeline 数据、调整 Orbit 容器 mounts 并重新部署；不得让代码自动猜测或合并旧目录。
- DooD 校验会令未挂载 workspace 的容器在启动时失败。这是预期 fail-fast 行为；修复方式是提供 Docker socket 和两个对应的 host bind mount，不是改回容器内路径。
- `traefik.cert_dir`、模型归档默认路径和脚本固定参数均为硬切换。升级与回退必须使用匹配的配置/脚本版本，不能混用旧路径契约。
