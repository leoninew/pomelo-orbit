# RAGFlow 内嵌 TEI CPU/GPU Version 规格
最后修改时间: 2026-07-31 13:01:30

Review status: Accepted

## Requirement Basis

本规格依据已接受的 [Requirement](../requirement/20260731-ragflow-tei-variants.md) 及 [运维手册](../guides/ragflow-tei-operations.md)。流程已切换为标准模式 / standard；标准模式通常从需求直接进入计划，但用户明确要求保留本 Spec / 规格作为跨越数据模型、部署渲染和交付数据的设计审查节点。

本次只设计 RAGFlow 内嵌 `BAAI/bge-m3` 的 CPU/GPU Version、准备工具、技能与控制面 SQL 基线；不执行 Orbit 生命周期写入、不启动 Docker Compose、也不修改现有开发数据库。

## Overview

最终拓扑是一个 `ragflow` 标准 Application、两个完整 Version 和一个 `default` Service：

| Version | 组件 | `tei` 差异 | 使用场景 |
| --- | --- | --- | --- |
| `ragflow-tei-cpu` | `mysql`、`redis`、`minio`、`es01`、`tei`、`ragflow-cpu` | CPU TEI 镜像，无 device request | 当前开发机及无 NVIDIA GPU Host。 |
| `ragflow-tei-gpu` | 与 CPU Version 完全相同 | CUDA TEI 镜像，NVIDIA GPU device request | 通过 GPU 前置检查的 NVIDIA Host。 |

两个 Version 的 `tei` 都挂载 `tei/cache` 到 `/data`，以 `--model-id /data/bge-m3 --json-output` 加载预下载模型。`ragflow-cpu` 增加对 `tei` 的 `service_healthy` 依赖；它仍是 RAGFlow 镜像中的 CPU 应用组件名，不因 TEI 使用 GPU 改名。唯一 Service expose 是 local HTTP `127.0.0.1:9380 -> ragflow-cpu:80`，TEI 没有宿主机端口、公开路由或独立 Application。

Service 只能在当前部署已经停止且没有非终态 Deployment 时切换所选 Version。切换不复制数据库数据，也不迁移模型缓存；它只改变同一个 Service 将被渲染和部署的静态组件规格。

## Image And Model Baseline

基线固定已验证的上游不可变引用，不把不稳定的镜像站域名写进 Version 或 SQL：

| Profile | Image |
| --- | --- |
| CPU | `ghcr.io/huggingface/text-embeddings-inference:cpu-1.9.3@sha256:ad950d30878eceb72aaf32024d26fa2b1d04a75304fa0b4776b49aa1941fea07` |
| GPU initial | `ghcr.io/huggingface/text-embeddings-inference:cuda-1.9.3@sha256:249a0bc87522bfe2f1012b4d194f0225878f47079115ada3aeb0b1ef257b402a` |
| Model | `BAAI/bge-m3@5617a9f61b028005a4858fdac845db406aefb181` |

CPU 引用可作为本地已验证基线。GPU 引用用于数据模型、Compose 渲染和未来 GPU Host 前置校验；本轮明确跳过 NVIDIA 容器启动、`/health` 与 `/embed` 运行验收。最终 SQL 可以包含该固定 GPU Version，但 Version note 和手册必须标明“GPU runtime 未验证”，不得将它报告为可运行结论。目标 Host 若需要更高性能，应另行 Fork 出适配计算能力的 Version，并以独立任务实测 `86-1.9`、`89-1.9` 或 `hopper-1.9` 的 digest；不能用浮动标签替换本基线。

RAGFlow 的 HuggingFace Embedding Provider 由 RAGFlow 自身的首次初始化 UI 表单配置：名称 `tei-bge-m3`、模型 `BAAI/bge-m3`、基础地址 `http://tei:80`、Max tokens `8192`。不附加 `/embed`，不写旧独立 TEI 地址。该配置在 RAGFlow 的 MySQL 中，不属于 Orbit 基线 SQL。RAGFlow API 的注册与 API key 获取不是本轮前置条件，也不实现 API 或数据库自动化；运维手册只提供可重复的表单初始化及连接校验步骤。

## Device Request Model

现有 `VersionComponentResources` 只表达 CPU 和内存，不能正确映射 Compose 的 `deploy.resources.reservations.devices`。新增独立、可扩展的 `VersionComponentDeviceRequest`，不重载现有资源字段：

| Field | Type | Constraint |
| --- | --- | --- |
| `driver` | string | 非空；本次 GPU Version 为 `nvidia`。 |
| `count` | string | `all` 或十进制正整数；保留为字符串以忠实表达 Compose 的 `all`。 |
| `capabilities` | ordered string list | 至少一项、无空项或重复项；NVIDIA GPU Version 必须含 `gpu`。 |

`model.VersionComponent` 增加 `Devices []VersionComponentDeviceRequest`。设备表定义合并到 SQLite/MySQL 的 `000023_application` schema：表以 `(component_id, position)` 标识有序 device request，包含 `driver`、`device_count` 和序列化的 capability list；`component_id` 级联引用 `version_component`。capability JSON 只在该单一结构化字段中存储，由 Go/Python 的 JSON 编解码处理，数据库排序由 `position` 保证，不使用拼接字符串解析。已有开发库原地保留该表，并将 migration state 对齐到合并后的 schema。

Repository 读取组件时加载该集合；替换组件、创建组件和 Fork Version 均复制它。新增单独的 `UpdateVersionComponentDevices` 事务，只替换 device request 并触碰组件时间戳，不复用 `advanced` 更新，避免资源、tmpfs 或 ulimit 被部分客户端的旧请求意外清空。

领域校验在 Version 未发布时执行：拒绝空 driver、非法 count、空/重复 capability 和包含空白字符的值；`nvidia` driver 必须有 `gpu` capability。GPU Host 能力不在持久化模型中伪造，由准备脚本在实际部署 Host 上检查。只要 GPU Version 有 device request，合并技能在写入 Orbit 前必须运行 GPU profile 的只读预检；失败时不得继续部署。

## API, MCP And UI Contracts

`proto/orbit/v1/application/version.proto` 新增 `ComponentDeviceRequest`，并将 `repeated devices` 放入 `VersionComponentReq` 与 `VersionComponentResp`。新增 `VersionComponentDevicesUpdateReq` 和 HTTP 端点：

```text
PUT /api/version/:version_id/component/:component_id/devices
{ "devices": [{ "driver": "nvidia", "count": "all", "capabilities": ["gpu"] }] }
```

对应 Go DTO、mapper、应用服务、repository interface/sqlc 查询、HTTP handler 和 route 以独立 devices 更新路径实现。重新生成 Go 与 TypeScript Proto 产物；`componentForm.ts` 及 Version Component 详情新增单独的“设备请求”编辑区域，显示 driver、count 和 capability 列表。CPU 组件没有该区块的持久化值，GPU `tei` 显示一个只含 `nvidia/all/gpu` 的请求。

`mcp/src/pomelo_orbit_mcp/version_specs.py` 新增严格 Pydantic `DeviceRequestSpec` 和 `VersionComponentDevicesUpdate`，全量组件创建 payload 包含 `devices`。`orbit_update_version_component_devices` 调用新 HTTP 路径；不能通过读取后回填 `advanced` 来更新 devices。MCP 测试断言 schema、序列化及旧字段不丢失。

## Compose Rendering And Version Rules

渲染器在 resources 或 devices 任一非空时生成 `deploy.resources`。devices 存在时合并到 reservations，目标 YAML 为：

```yaml
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: all
          capabilities: [gpu]
```

CPU/内存 reservations 和 `devices` 必须可并存，任何一个为空时不产生空 Compose 映射。组件校验、Compose preview 和实际部署共享同一渲染和验证路径；GPU 不能因未生成 `devices` 而无声退化为 CPU 容器。

在 `scripts/ragflow-bundled/docker-compose.yml` 保持它作为原始五组件参考输入，但升级其注释以说明内嵌 TEI 的受管基线由新技能和导出 SQL 定义。实现时新增或派生可审阅的六组件参考定义，禁止用 Docker Compose 生命周期命令启动它。Orbit Version 数据中 CPU/GPU 的 MySQL、Redis、MinIO、Elasticsearch、RAGFlow command、挂载、healthcheck、环境变量占位符和 local expose 完全一致；仅 `tei.image`、`tei.devices` 及以后经实测确认的 TEI 资源参数可不同。

`scripts/ragflow-split/` 仍只用于用户明确要求五个独立 Application 的场景。该替代拓扑的 `docker-compose.ragflow.yml` 也必须内嵌 `tei`，并使用与 bundled CPU Version 相同的固定 TEI digest、`/data/bge-m3` 命令、缓存挂载和健康检查；TEI 不再有单独的 Compose、Application 或宿主机端口。

## Preparation Script

新增 `scripts/prepare_ragflow_tei.py`，只用 Python 标准库与 `argparse`，从仓库根目录推导默认缓存目录 `data/deployment/ragflow/default/tei/cache/bge-m3`。它必须有清晰的 `--help`、人类可读输出、`--json` 机器输出和稳定的非零退出码；JSON 中只报告路径、状态、固定 revision/digest、所选 profile 和阻塞原因，不报告凭据或运行时配置。

| Command | 默认副作用 | 行为 |
| --- | --- | --- |
| `check --profile cpu|gpu` | 无 | 检查 Git LFS/HF CLI 能力、实际模型文件、LFS 完整性、所选 image manifest 与本机 Docker/GPU 条件。 |
| `prepare-cli` | 仅显式安装选项 | 检查 `git lfs` 与 `hf download --dry-run`；HF CLI 可经显式 `--install-hf-cli` 使用当前 Python 的 pip 安装，Git LFS 只检测并按 Host 包管理器给出安装指引。 |
| `prepare-model --source git-lfs|hf-cli` | 有 | 使用固定 revision 下载到空目标；只允许 Git checkout 的显式 `--resume` 做 `git lfs pull`。非空但无效目录失败，不删除、不覆盖。 |
| `prepare-image --profile cpu|gpu` | 有 | 从上游 digest 拉取镜像并通过 image inspect 复核；镜像站必须先请求 OCI manifest 且与上游 digest 一致，失败不自动切换。 |
| `backup-model` | 有 | 在 Git LFS 缓存 `fsck` 和文件预检通过后，创建 `tar.gz` 压缩归档和逐文件 SHA-256 manifest；以临时文件写入，未经显式 `--replace` 不覆盖已有归档。 |
| `restore-model --archive <path>` | 有 | 验证归档和 manifest 后恢复到空目标目录；不覆盖非空缓存，恢复完成后重新执行文件与 hash 校验。 |
| `prepare --profile cpu|gpu` | 有 | 依次执行已明确选择的 CLI、模型和镜像准备，并在最后重新运行完整 `check`；不隐式创建大型备份，备份必须单独显式调用。 |

模型就绪不能仅以目录存在判断。Git LFS 源缓存还必须通过 `git lfs fsck`；两种下载路径都要求 `config.json`、tokenizer 文件、实际主权重和包含数据文件的 `onnx/`。HF CLI 的 dry-run 可能留下目录骨架，必须判定为未就绪。压缩归档以 Python 标准库 `tarfile` 生成 `tar.gz`，不包含 Git 元数据；恢复后的缓存以归档 manifest 的逐文件 SHA-256 和同一文件清单确认完整性。归档默认保存于项目根 `data/backup/bge-m3-<revision>.tar.gz`，仅归档模型工作树而不重复 `.git`/`.cache` 内容；报告真实归档大小和耗时，但不承诺对已经压缩的权重文件有显著空间收益。GPU profile 额外要求可用 `nvidia-smi`、Docker NVIDIA runtime/Container Toolkit 和兼容 Driver；本轮只验证该失败路径和 Compose 静态输出，不在无 GPU Host 上伪造成功。

脚本永不执行 Docker Compose、不会创建/更新/部署 Orbit 资源、不会自动移动或删除既有独立 `bge-m3` 缓存。实施时复制现有 Git LFS 缓存到 RAGFlow 管理目录；验证新目录后创建归档，原独立缓存继续保留。

## Skill Consolidation

新增 `skills/deploy-ragflow-tei-orbit` 作为唯一 RAGFlow + TEI 部署权威，包含引用 `scripts/prepare_ragflow_tei.py` 的前置步骤、CPU/GPU Version 选择、Oracle 的实际 MCP schema 检查、Gateway provisioning、Service 停止后切换和 Orbit 验收。所有生命周期写仍只通过 `pomelo_orbit` MCP。

旧 `deploy-ragflow-orbit` 与 `deploy-bge-m3-orbit` 不再各自维护拓扑或下载命令：前者改为清楚指向合并技能的兼容入口，后者改为弃用说明并阻止创建独立 TEI Application。仓库 `AGENTS.md` 的默认 RAGFlow 技能引用更新为合并技能及六组件/双 Version 规则。这样不会保留三个可能漂移的操作事实来源。

新技能的第一步始终是只读 `prepare_ragflow_tei.py check --profile <profile>`；若模型或镜像未准备好，只返回精确的脚本命令，不执行 MCP 生命周期写入。预检通过后，技能才读取实时 MCP tool schema、列出现有 Project/Application/Version/Service 并执行受管写入。

## SQLite Baseline Export

`scripts/ragflow-split/ragflow-bundled-sqlite.sql` 是当前结构和记录范围的参考，而不是可直接覆盖的运行态快照。新增确定性导出器 `scripts/export_ragflow_tei_baseline.py`，从指定 SQLite 数据库的只读连接生成候选基线 SQL；默认不覆盖任何文件，只有显式 `--output` 才写出结果。完成导入验收后，候选输出将替换受版本控制的 `scripts/ragflow-split/ragflow-bundled-sqlite.sql`。

导出前必须验证：数据库完整性和外键检查通过；目标 Project 唯一；含 `ragflow` 与受管 `traefik` Gateway Application；RAGFlow 有且仅有 CPU/GPU 两个目标 Version；每个 Version 恰有六个同名组件；RAGFlow 仅有一个 `default` Service，初始选择 CPU Version，GPU Version 通过该 Service 在停止后切换；两 Version 的 Deployment 记录均不导出。CPU Version 和 Gateway 必须完成运行验收；GPU Version 只要求 devices 通过领域/Compose 静态验证并带有“runtime 未验证”的 note。这里的“停机基线”定义为 SQL 不包含 `deployment`、运行日志和运行时状态，而不是将可部署 Version 改为不可用的 unpublished 状态。

输出按稳定表顺序、主键顺序和列顺序生成，并放在单个事务中。它只包含重建所需的 Project、两个 Application、`gateway_config`、Version、VersionComponent、全部组件子规格（包括新增 device 表）、Service 和 `service_expose`。它明确排除 `deployment`、`service_runtime_config`、任何凭据值、命令输出及无关 Application 数据。Gateway 的持久化路由配置可以被导出，运行时配置和值不可以被导出。导出后在空的、已迁移 SQLite 数据库中导入，并核对 `PRAGMA integrity_check`、`PRAGMA foreign_key_check`、表行数、两 Version 的六组件清单、TEI device 差异和唯一 local RAGFlow expose。

## Affected Components

- `internal/model/application.go`、application DTO/use case/validation、HTTP handler/mapper/routes、`internal/repository/application.go` 与 `internal/repository/impl/sqlc/application/repository.go`。
- `sql/migration/sqlite/000023_application.*.sql`、`sql/migration/mysql/000023_application.*.sql`、`sql/query/application/version.sql`、`internal/gen/sqlc/application/`。
- `proto/orbit/v1/application/version.proto`、`internal/gen/proto/`、`web/src/gen/proto/`、`web/src/views/application/VersionComponentDetail.vue`、`web/src/views/application/componentForm.ts`。
- `internal/application/deployment/usecase/compose_renderer.go` 及 renderer/validation tests。
- `mcp/src/pomelo_orbit_mcp/version_specs.py`、`mcp/src/pomelo_orbit_mcp/tools/orbit.py`、MCP tests。
- `scripts/prepare_ragflow_tei.py`、`scripts/export_ragflow_tei_baseline.py`、`scripts/ragflow-bundled/docker-compose.yml`、最终重新生成的 `scripts/ragflow-split/ragflow-bundled-sqlite.sql`。
- `skills/deploy-ragflow-tei-orbit/`、既有两项技能的兼容/弃用文档、`AGENTS.md` 和 `docs/guides/ragflow-tei-operations.md`。

## Verification Design

1. 新迁移在 SQLite 和 MySQL schema 上创建/回滚，SQLite `integrity_check` 与 `foreign_key_check` 通过；sqlc、Go/TypeScript Proto 通过项目既有 `task sqlc`、`task proto`。
2. 应用服务、repository、HTTP 和 MCP 测试覆盖 devices 的创建、读取、Fork、独立替换、非法 count/capability 拒绝，以及更新 devices 不改写 resources/tmpfs/ulimits。
3. renderer tests 对 CPU 与 GPU TEI 断言相同非 TEI 组件规格，GPU YAML 精确含 NVIDIA reservation，CPU YAML 不含 `devices`；缺失或无效 device request 必须失败。
4. 前置脚本单元测试以临时目录、模拟 CLI 和本地 OCI manifest 响应覆盖空缓存、不完整目录、Git LFS 失败、HF dry-run 骨架、镜像站 HTML/错误 digest、GPU Host 缺失、`tar.gz` manifest 损坏、非空恢复目标和成功 JSON 结果；不在测试中下载 4.6 GiB 模型或启动容器。
5. 导出器集成测试以迁移后的临时 SQLite fixture 生成 SQL，再导入空数据库，核对指定结构数据存在、runtime/deployment 表没有导出、结果字节稳定且外键完整。
6. CPU Host 上进行真实 `prepare ... --profile cpu`、模型压缩备份、恢复到空目录后的完整性检查、Orbit 预览/部署、TEI `/health`、`/embed` 1024 维和 RAGFlow Provider 表单连接验收。本轮 GPU 只运行领域、Proto/MCP、Compose renderer 和 GPU profile 缺失条件的静态检查；不启动 GPU 容器，不产生 GPU 运行验收结论。

## Risks And Open Questions

- GPU image digest 和 CUDA/Driver 组合仍需真实 NVIDIA Host 验收，但它是后续任务而非本轮发布阻塞；本轮导出的 GPU Version 必须保留未验证标记。
- Orbit 当前没有独立的远程 Host capability 资源；本设计假定准备脚本在实际承载 Orbit Docker runtime 的 Host 上运行。若未来支持远程调度，需要将 Host 能力建模为控制面能力而不是信任本地探针。
- RAGFlow provider 采用 UI 表单初始化；注册用户与取得 API key 的 API 流程明确排除。若需要无人值守初始化，应以当前 RAGFlow 公开 API 的实测契约另立需求。
- 大型模型归档的 gzip 压缩率可能很低，且 SHA-256 扫描与归档耗时较长；脚本必须报告实测值，恢复测试以完整性优先。
- 原独立 Git LFS 缓存保留，实施计划必须记录复制后的 `fsck`、归档 manifest 与新目录恢复验证。
- 备用 registry 的可用性、认证和内容一致性随网络变化；预检只能验证当次结果，不能把某次镜像站成功等同于长期 SLA。

## Alternatives Rejected

- **TEI 运行时使用远程 model ID 下载**：把大文件下载、鉴权和网络不确定性放到服务启动路径，无法可靠区分模型加载与下载失败。
- **继续保留独立 `bge-m3` Application**：Service DNS、生命周期与缓存归属分裂，无法用 RAGFlow Version 原子切换 CPU/GPU 变体。
- **将 GPU 写进 CPU/内存 resources 或 environment**：Compose 不会得到合法的 `reservations.devices`，GPU 容器可能无声回退。
- **将 devices 加入 broad advanced update**：旧客户端或局部更新容易覆写无关资源配置，违背组件编辑区的独立所有权。
- **直接复用或导入旧 SQL**：旧记录拓扑、版本与环境配置不匹配，并可能携带不应复用的运行态数据。
- **自动选择任意 registry mirror**：已观察到 401、403 或 HTML 响应；未验证 manifest/digest 的镜像源不能成为部署输入。

## User Review Notes

- 2026-07-31：用户要求提供 CPU 与 NVIDIA GPU Version，即使当前 Host 没有 GPU；应用的 Version 机制是变体选择边界。
- 2026-07-31：用户要求评估 Git LFS、Hugging Face CLI、镜像与镜像站路径，必要时以 Python 前置脚本统一模型、CLI 和 image 准备。
- 2026-07-31：用户要求将 TEI 合并到 RAGFlow，随后导出可复用的控制面 SQL，并把已验证数据写入交付手册。
- 2026-07-31：用户要求跳过 GPU 运行测试；模型缓存压缩备份和恢复用于反复 CPU 测试；Provider 只通过 RAGFlow 表单初始化，API 注册/API key 不纳入本轮。
- 2026-07-31：用户将模型备份格式确定为 `tar.gz`，并要求开始实施。
