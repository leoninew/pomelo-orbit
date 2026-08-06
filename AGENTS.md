# Repository Agent Instructions

## 项目约束

- 不主动启动、停止或重启开发服务器；由用户管理。
- 项目处于活跃开发期，不做兼容层、别名、默认值兜底或新旧逻辑并存。
- API 路由统一使用单数形式，例如 `/api/ci/repository`。
- 不修改已执行的迁移文件。
- 前端目录为 `web`；修改前端后必须运行 `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck`。
- 前端时间展示使用 `web/src/utils/time.ts`，不要在业务代码中散落 `dayjs()`。
- 前端页面优先使用项目共享组件，不直接散落底层 Reka primitives。
- Go 后端检查使用 `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...` 与 `go test ./cmd/... ./internal/...`。

## 文档阅读顺序

- 默认依据：`docs/README.md` -> `docs/INDEX.md` -> `docs/product/`、`docs/architecture/`、`docs/decisions/`，以及已校准的 `docs/guides/`、`docs/frontend/`。
- `docs/archive/**` 为历史材料：无须采信，不得作为实现或验收依据。
- SpecFlow 过程目录（`docs/requirement|spec|plan|verification`）仅用于进行中任务；已归档过程文档在 `docs/archive/specflow/`。
- 文档与代码冲突时以代码为准，并回写活文档。

## RAGFlow Deployment

- Use the `pomelo_delivery` Codex MCP server. Its local stdio command is `go run ./cmd/server mcp`; the exposed control-plane tools retain their `orbit_*` names. On first use, it opens Orbit's browser login/authorization page and saves the resulting bearer credential only in the local user configuration directory.
- Treat a request to deploy RAGFlow with Pomelo Orbit as the integrated deployment by default. Use the generated `skills/deploy-ragflow-integrated-orbit/references/docker-compose.yml` reference and the `deploy-ragflow-integrated-orbit` skill.
- Before any `pomelo_delivery` write, read the skill and inspect the current MCP tool definitions. Do not infer parameters from historical documents or implementation details.
- The integrated topology owns exactly one `ragflow-integrated` Application, two six-component Versions (`ragflow-integrated-cpu` and `ragflow-integrated-gpu`), and one `default` Service. Each Version contains `mysql`, `redis`, `minio`, `es01`, `tei`, and `ragflow-cpu`; only the TEI image and GPU device request differ. The integrated skill initializes only these resources.
- Use the generated references in `skills/deploy-ragflow-split-orbit/references/` and `deploy-ragflow-split-orbit` only when the user explicitly requests split deployment. It owns a separate `ragflow-split` Application with two Components in each Version (`ragflow-split-cpu` and `ragflow-split-gpu`), plus independent `ragflow-mysql`, `ragflow-redis`, `ragflow-minio`, and `ragflow-elasticsearch` Applications. Never share or mount data directories between the two RAGFlow Applications.
- The split skill initializes only `ragflow-split` and its four split backing Applications with their primitive-defined Versions and stopped Services. The integrated skill never initializes split resources, and the split skill never initializes `ragflow-integrated`. The invoked skill deploys only its own topology: integrated starts `ragflow-integrated/default`; split starts the backing Services and `ragflow-split/default`.
- Both RAGFlow Services expose `ragflow-cpu:80` through `gateway_http`; only one may run. Before starting the selected topology, stop the other RAGFlow Service through Orbit without removing volumes. The managed Traefik API is local at `127.0.0.1:8080` for the Dashboard.
- Before an Orbit lifecycle write, detect NVIDIA hardware and run `skills/_ragflow/prepare_ragflow_tei.py check --profile cpu` or `--profile gpu` for the selected topology and profile. Select GPU only when NVIDIA is detected and the GPU preflight passes; otherwise select CPU. A detected GPU whose preflight fails blocks deployment rather than silently falling back to CPU.
- `docs/archive/` is historical context, never operational authority. Do not use its workflows, `orbit_bootstrap_application`, or the removed `scripts/ragflow_initialize.py`.
- Perform deployment lifecycle writes only through `pomelo_delivery` MCP. Do not use Docker Compose lifecycle commands, and do not print runtime configuration or credentials.
