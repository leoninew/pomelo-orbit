# 可配置 CI/CD 工作目录与 DooD 宿主路径解析验证
最后修改时间: 2026-08-18 22:57:33

Review status: Accepted

Mode: strict

## Verification Basis

- Requirement: [可配置 CI/CD 工作目录与 DooD 宿主路径解析](../requirement/20260818-configurable-ci-cd-workspaces.md)
- Spec: 不适用。用户从已接受的 Requirement 直接进入 Plan，Plan 承担实现设计细化。
- Plan: [可配置 CI/CD 工作目录与 DooD 宿主路径解析计划](../plan/20260818-configurable-ci-cd-workspaces.md)
- Verification scope: `workspace.pipeline`、`workspace.deployment`、DooD Docker daemon path 解析、Gateway 证书目录、运维工具与对应活文档。

当前工作区同时存在 MCP、lint、Docker 制品绑定、MCP token 存储和独立前端展示变更；这些不属于本验证结论。`docs/product/cd-model.md` 与 `docs/guides/ci-pipeline-design.md` 中的 Docker 制品绑定段落也不纳入本任务。

## Requirement Alignment

- `WorkspaceConfig` 为 Pipeline 与 Deployment 提供独立根目录；默认 YAML 保持 `data/pipeline`、`data/deployment` 的既有物理位置，并移除 `traefik.cert_dir` 这个第二路径来源。
- 配置加载集中完成路径 trim、相对 `orbit.root` 转绝对路径、非空和根目录重叠校验；workspace 与其他 typed config 一样支持 YAML、`.env` 与 OS 环境变量覆盖，且不提供 Settings 项。
- Pipeline 与 Deployment workspace 不再追加固定目录名。原生运行使用 Orbit 可见根目录；DooD 运行时仅将 Docker bind source 转换为 daemon 可见宿主路径。
- Gateway cert/ACME 与 Route PEM 目录统一派生为 `<workspace.deployment>/traefik/data/certs`。Route REST snapshot 同时登记已保存证书的容器内路径。
- 模型归档、证书与备份工具改由调用者提供归档、证书目录或 workspace excludes，不新增无运行时所有者的配置项。

## Plan Alignment

1. 配置、bootstrap 和两个 workspace 实现已改为独立根目录语义，并由定向测试覆盖。
2. Docker daemon path resolver 在原生模式保持路径不变，在容器模式使用 Docker inspect mount 的最长 destination 匹配；启动时验证两个 workspace root 均有宿主映射。
3. Deployment 在 Orbit 可见逻辑路径物化 directory / controlled file；原生 Compose 保留相对 source，DooD Compose 使用解析后的宿主路径。
4. Gateway 初始 Component 和 RouteManager 使用 Deployment workspace 派生的证书目录，保留已保存 Version 可编辑且不被部署回写的边界。
5. RAGFlow 合同、辅助脚本、删除提示和活文档不再将 CI/CD 目录或模型归档描述为固定 `data/` 路径。
6. Requirement 与 Plan 指定的 Go、前端、Python 和 diff 健康检查已执行；见 Test Results。

## Actual Diff Summary

- 增加 typed `workspace.pipeline` / `workspace.deployment`，删除旧 `DataRoot()` 推导及 `traefik.cert_dir` 配置入口。
- Pipeline checkout、artifacts、stage log，以及 Deployment Service Compose、部署日志和 logical mount 均直接从各自 workspace root 派生。
- Docker bind source 解析统一为 Docker daemon path；Orbit 容器启动会对两个 workspace 进行 fail-fast mount 验证。
- Compose renderer 分离 logical materialization source 与 Compose source，保持原生 Compose 的相对路径语义。
- Gateway 证书/ACME mount 和 Route PEM 文件迁至 Deployment workspace；REST snapshot 增加 TLS certificate 引用。
- `scripts/cert.py`、`scripts/manage.py`、RAGFlow archive 工具与合同改为显式路径参数；Repository / Application 删除提示和操作文档同步更新。

## Expected Vs Actual Changed Files

| 预期区域 | 实际实现与验证 |
| --- | --- |
| Config | `configs/`、`.env.example`、`internal/config/` 已包含 workspace 配置、规范化与校验测试。 |
| Bootstrap / DooD | `internal/bootstrap/` 与 `internal/infrastructure/storage/local/physical_data_root.go` 已集中 resolver 注入和容器 mount 校验。 |
| Workspace storage | `pipelineworkspace/`、`deploymentworkspace/` 与 deployment use case 已改为根目录、logical path / daemon path 双路径语义。 |
| Gateway / Route | Gateway Component、Route PEM 目录和 Traefik REST snapshot 已收敛到 Deployment workspace，并有 use case 与 RouteManager 测试。 |
| Tools / UI | `scripts/`、`skills/`、RAGFlow contracts、Repository 删除提示与 i18n 已改用 workspace 或显式参数。 |
| Documentation | Requirement、Plan、活文档和本 Verification 已更新；包含另一项 Docker 制品规则的两份文档按提交边界暂留未暂存。 |

## Acceptance Checklist

- [x] 默认 YAML 声明两个 workspace root；workspace 与其他 typed config 一样支持环境 YAML、`.env` 和 OS 环境变量覆盖，且不提供 Settings 项。
- [x] 配置加载完成路径规范化、必填和重叠校验。
- [x] Pipeline workspace 不再固定追加 `pipeline`，并保持 checkout、artifact、stage log 的隔离结构。
- [x] Deployment workspace 不再固定追加 `deployment`，Service code 仍是唯一的服务根层级。
- [x] Pipeline Stage、artifact 与本地 Repository 的 Docker mount 使用 Docker daemon 可见路径。
- [x] Deployment preview 和实际 Compose 对原生相对 source、DooD 宿主 source 与 logical materialization 使用一致边界。
- [x] 容器运行时校验两个 workspace root 的宿主 bind mount；原生运行不调用 Docker inspect。
- [x] Gateway cert/ACME mount 与 Route PEM 均从配置化 Deployment workspace 派生。
- [x] RAGFlow 模型归档要求显式 `--archive`；不再默认写入 `data/backup`。
- [x] 证书与备份脚本要求显式路径参数，不再内嵌旧 CI/CD workspace pattern。
- [x] Repository / Application 删除提示、RAGFlow 合同和活资料以 workspace 或显式路径表述目录。
- [x] SQLite、日志、环境文件、MCP 凭据、exports 和用户本地 Repository 未被纳入 workspace 配置。
- [x] 覆盖 Config、DooD 解析、workspace mounts、Compose logical materialization 和 Gateway certificate path 的定向测试。
- [x] 活文档已描述新配置契约，不再将 `data/pipeline`、`data/deployment` 或 `data/backup` 作为不可变运行路径。

## Test Results

| Command | Result |
| --- | --- |
| `task check` | Passed |
| `go test ./cmd/... ./internal/...` | Passed |
| `go test ./internal/config ./internal/bootstrap ./internal/application/deployment/usecase ./internal/application/gateway/usecase ./internal/application/route/usecase ./internal/infrastructure/external/traefik ./internal/infrastructure/storage/local ./internal/infrastructure/storage/local/deploymentworkspace ./internal/infrastructure/storage/local/pipelineworkspace ./internal/infrastructure/storage/local/repositorysource` | Passed |
| `yarn --cwd web lint` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `python -m pytest skills/_ragflow/test_prepare_ragflow_tei.py` | Passed (5 tests) |
| `python -m pytest skills/deploy-ragflow-integrated-orbit/test_render_contract.py` | Passed (5 tests) |
| `python -m pytest skills/deploy-ragflow-split-orbit/test_render_contract.py` | Passed (5 tests) |
| `python scripts/cert.py --help` | Passed; `--cert-dir` is shown by subcommand help and examples. |
| `python scripts/manage.py backup --help` | Passed; repeatable `--exclude-workspace` is shown. |
| `git diff --check HEAD` | Passed |

## Scope And Residual Risk

- 本次不迁移、复制或删除既有 Pipeline、Deployment 或证书数据。升级前必须由运维方停止相关服务、迁移所需数据并调整 Orbit 容器的两个 workspace bind mount。
- 未启动开发服务器，也未执行真实 Docker Compose、Docker inspect、Traefik REST 或浏览器端到端场景。DooD host path 与证书发布仍需在目标部署环境按新 mount 契约确认。
- `traefik.cert_dir`、模型归档默认路径和脚本固定参数为硬切换；回滚时必须恢复相匹配的配置与脚本版本，不能混用新旧目录契约。
- 当前工作区含无关未暂存改动。提交前应保持本任务的 staged scope，不将它们合入本提交。

## Conclusion

实现与已接受的 Requirement 和 Plan 对齐；自动化与 CLI 验证通过，目录迁移和真实 DooD/Traefik 环境验证保留为部署前的运维检查。本验证记录已接受。
