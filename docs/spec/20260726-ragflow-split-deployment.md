# RAGFlow 拆分应用部署规格
最后修改时间: 2026-07-26 20:06:12

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-ragflow-split-deployment.md` (`Accepted`)
- Prerequisite capability task: `docs/plan/20260726-orbit-deployment-capability-gaps.md` (`Accepted`)
- Gateway MCP、受管 `traefik` 预检与固定 HTTP Probe 已合并入本规格，不再维护独立 feature 文档。
- Flow mode: 严格模式 / `strict`

## Overview

本规格将 RAGFlow 当前默认 CPU + Elasticsearch Compose 形态拆为五个同 Project、Docker context 和 instance key 的 `standard` Application：RAGFlow、MySQL、Redis、MinIO、Elasticsearch。每个 Application 只含一个 Component 和一个初始 Version，依赖服务不暴露宿主机端口，RAGFlow 通过 Orbit 共享 `traefik` 网络上的稳定别名访问它们。

该规格以非秘密 desired-state fixture、受管 MCP Runbook 和可重复调用的 Python 初始化编排器描述目标数据，不导入原 Compose 的 `include`、`profiles` 或 `env_file`。所有密码来自任务一的 `runtime_env` Credential 引用，不能使用 RAGFlow 源码 `.env` 中的默认值。

## Initializer contract

任务二提供 `scripts/ragflow_initialize.py`。该脚本是 fixture 到受管 MCP 操作的唯一编排入口，不调用 Docker lifecycle CLI，也不直连 Orbit REST API。它必须支持：

1. `plan`：离线校验 fixture、生成确定性的操作计划和 JSON 运行记录，不连接 MCP、不读取秘密也不写运行资源。
2. `check`：通过 stdio MCP 列出工具并检查 Environment-free 契约、任务一新增能力、参数、目标资源冲突和运行上下文；能力尚未实现时输出 `blocked`，而不是跳过或伪造成功。
3. `apply`：只有显式确认、Live integration gate 通过、受控秘密文件提供且所有工具存在时，创建 Credential、Application、Version 和 Deployment；每个成功步骤原子记录，可用 `--resume` 恢复同一运行。
4. `verify`：对已记录的 Deployment 调用 MCP 等待和 `verify_deployment`；RAGFlow 还必须执行固定 `runtime_http_probe`，写入脱敏状态与结论；不在运行记录中持久化任何 Credential value。

初始化器的输入为 fixture、显式参数和位于仓库外的秘密文件。`check` 与 `apply` 还必须接收已部署 Gateway 的 Application ID，instance key 默认为 `default`。秘密文件只允许在 `apply` 进程内读取，不写入日志、状态文件、异常消息或命令行参数。任务一尚未交付的工具和 Component 字段由 capability gate 占位；它们不会阻塞 `plan`，但阻塞 `check`、`apply` 和 `verify` 的相关阶段。初始化器通过该 Gateway target 的 `runtime_doctor` 证明固定 bridge `traefik` 网络；RAGFlow 候选 healthcheck 随 Version 发布，部署后的实际就绪由固定 `runtime_http_probe` 判断并写入无秘密 journal。`check` 同时拒绝未知的同名 Application / Credential；`--resume` 仅可继续同一运行目录已记录 ID 的资源。

## Gateway preflight and HTTP Probe contract

`OrbitClient` 与 MCP 公开 `orbit_list_gateways`、`orbit_create_gateway`、`orbit_get_gateway`，仅映射既有 `/api/gateway` HTTP API。Gateway 的 publish、deploy 与 wait 继续使用既有 Orbit lifecycle tools；初始化器不得自动创建、更新或删除 Gateway。

`runtime_doctor` 接受成对的 `gateway_application_id`、`gateway_instance_key`。该路径先要求 Application `kind=gateway`，再从其唯一受管 runtime target 的 Compose `ps` 与 container inspect 派生网络；只有派生集合包含固定名称 `traefik` 时才执行 network inspect，并且只有 `driver=bridge` 才健康。返回仅含 `external_networks` 和 name/driver/scope detail；不接受任意 network name，也不创建网络。

`runtime_http_probe(application_id, component_name, port, path="/", instance_key="default")` 只选择目标 Compose `ps` 中唯一、running 且 service 名匹配的 container。port 限制为 `1..65535`，path 必须以 `/` 开始、无空白且最长 512；固定执行非 shell argv `docker exec <derived-container> curl -fsS --max-time 10 --output /dev/null http://127.0.0.1:<port><path>`。成功只返回 reachability 与受控参数，任何 Docker/curl 失败均不回显 stdout/stderr。

## Parallel development and integration

任务二与任务一并行开发。任务二可在任务一实现期间完成 RAGFlow 源 Compose 到五个 Application 的逐字段映射、钉死镜像和健康检查验证、非秘密 desired-state payload fixture，以及脱敏 Runbook；这些工件使用符号化 Credential ID / value 占位，不包含实际秘密、Project ID、主机路径或资源 ID。

任务二不得在任务一完成其 MCP / Version 契约和秘密防护前创建实际 `runtime_env` Credential、Application、Version 或 Deployment。该限制只约束真实运行资源写入，不阻塞任务二的文档、fixture、镜像调研和集成测试开发。任务一 Verification 与任务二 fixture 的集成核对通过后，才可执行本规格的 Deployment gate 和实际部署。

## Deployment gate

任务二实施前必须同时满足：

1. 环境移除任务已合入并验证；MCP、Deployment、Service、Preview、runtime tools 和 workspace 均不再接受、返回或推导 Environment。
2. 任务一的 Plan 已接受，Implementation 完成，且其 Version / MCP 契约已与任务二的非秘密 fixture 集成核对；真实部署前，Verification 还必须证明 `restart_policy`、tmpfs、ulimits、`runtime_env` Credential 引用和全链路脱敏可用。
3. 已创建、发布并部署 Gateway Application；MCP `runtime_doctor` 使用其 Application / instance target 确认 Orbit data root、Docker context、Compose CLI 可用，并明确返回派生的 bridge `traefik` 网络。缺失、非 Gateway 或未部署 target 必须以 capability gate 阻断，而不是调用未受管 Docker CLI。
4. 已确定 Project ID、instance key、RAGFlow 入口方式、实际监听端口和备份位置。
5. 已创建强随机、互不复用的运行时 Credential；实际值不会进入 Version、MCP 记录或过程文档。
6. 目标主机满足 Elasticsearch 资源要求，特别是至少 8 GiB 可用容器内存及可用的 tmpfs / memlock 行为。
7. 初始化器已对 fixture 成功运行 `plan` 和 `check`，并生成不含秘密的运行记录；`apply` 使用新的或经 `--resume` 校验的运行目录。

## Topology

| Application code | Component | Image | Persistent logical mount | Shared-network alias |
|---|---|---|---|---|
| `ragflow-mysql` | `mysql` | `mysql:8.0.39` | `data` -> `/var/lib/mysql` | `ragflow-mysql-mysql` |
| `ragflow-redis` | `redis` | `valkey/valkey:8` | `data` -> `/data` | `ragflow-redis-redis` |
| `ragflow-minio` | `minio` | `pgsty/minio:RELEASE.2026-03-25T00-00-00Z` | `data` -> `/data` | `ragflow-minio-minio` |
| `ragflow-elasticsearch` | `es01` | `elasticsearch:8.11.3` | `data` -> `/usr/share/elasticsearch/data` | `ragflow-elasticsearch-es01` |
| `ragflow` | `ragflow-cpu` | `infiniflow/ragflow:v0.26.4` | `logs` -> `/ragflow/logs` | `ragflow-ragflow-cpu` |

所有 Application 使用 `kind=standard`、`image_pull_policy=missing`、同一 Project 和 `instance_key=default`。平台为标准组件自动加入共享网络并写入上述 alias；任何 Component 均不得自行声明另一跨 Application 网络。

## Version specification

### MySQL

- Version label: `8.0.39-initial`。
- Command：`--max_connections=1000`、`--character-set-server=utf8mb4`、`--collation-server=utf8mb4_unicode_ci`、`--default-authentication-plugin=mysql_native_password`、`--tls_version=TLSv1.2,TLSv1.3`、`--binlog_expire_logs_seconds=604800`。
- MySQL 官方初始化环境变量 `MYSQL_DATABASE=rag_flow` 创建数据库；仅保留 `data` -> `/var/lib/mysql` 的 logical data mount，不使用受限文件后缀策略不支持的 `init.sql` 内容挂载。
- `runtime_env` 引用：Credential `ragflow-mysql` 的 `MYSQL_PASSWORD` 映射为容器 `MYSQL_ROOT_PASSWORD`。
- Healthcheck：以容器 shell 使用 `$$MYSQL_ROOT_PASSWORD` 执行 `mysqladmin ping`；由 Docker Compose 将双美元符保留给容器 shell。
- `restart_policy=unless-stopped`。不创建 Expose，不向宿主机发布 3306。

### Redis

- Version label: `8-initial`。
- Command 使用容器 shell 执行 `redis-server --requirepass "$$REDIS_PASSWORD" --maxmemory 128mb --maxmemory-policy allkeys-lru`；不能把 `${REDIS_PASSWORD}` 直接放入数组 command，因为 Orbit 仅替换环境变量占位符。
- `runtime_env` 引用：Credential `ragflow-redis` 的 `REDIS_PASSWORD` 映射为同名容器变量。
- Healthcheck 使用 `redis-cli -a "$$REDIS_PASSWORD" ping`。
- `restart_policy=unless-stopped`，逻辑 data mount，不创建 Expose。

### MinIO

- Version label: `RELEASE.2026-03-25-initial`。
- Command：`server --console-address :9001 /data`。
- `runtime_env` 引用：Credential `ragflow-minio` 的 `MINIO_USER`、`MINIO_PASSWORD` 分别映射为 `MINIO_ROOT_USER`、`MINIO_ROOT_PASSWORD`。
- Healthcheck：`curl -fsS http://localhost:9000/minio/health/live`。
- `restart_policy=unless-stopped`，逻辑 data mount，不创建 Expose。

### Elasticsearch

- Version label: `8.11.3-initial`。
- 非秘密环境：`node.name=es01`、`bootstrap.memory_lock=false`、`discovery.type=single-node`、`xpack.security.enabled=true`、HTTP / transport SSL 均为 `false`、以及当前 Compose 的三个 disk watermark 值。
- `runtime_env` 引用：Credential `ragflow-elasticsearch` 的 `ELASTIC_PASSWORD` 映射为同名容器变量。
- tmpfs：`/tmp`，mode `1777`，size `536870912` bytes；ulimit：`memlock` soft / hard 均为 `-1`。
- Resources 采用与原 `MEM_LIMIT=8073741824` 等价的 Compose memory limit；先在任务一验证所生成的 `deploy.resources` 被 Docker Compose 实际接受，再写入 Version。
- Healthcheck 使用 `curl -fsS -u "elastic:$$ELASTIC_PASSWORD" http://localhost:9200`。
- `restart_policy=unless-stopped`，逻辑 data mount，不创建 Expose。

### RAGFlow CPU

- Version label: `v0.26.4-cpu-elasticsearch`。
- Command：`--enable-adminserver`、`--init-model-provider-tables`，保留镜像内置 `entrypoint.sh` 和 `service_conf.yaml.template`。
- 非秘密环境：`DOC_ENGINE=elasticsearch`、`DEVICE=cpu`、`API_PROXY_SCHEME=python`、`TZ=Asia/Shanghai`，以及下表所列的数据库、缓存、对象存储和向量库地址。`COMPOSE_PROFILES`、`RAGFLOW_IMAGE`、宿主机发布端口等原 Compose 变量不传入容器。
- `runtime_env` 引用使用同名四个 Credential，将密码只映射为 RAGFlow 的 `MYSQL_PASSWORD`、`REDIS_PASSWORD`、`MINIO_PASSWORD`、`ELASTIC_PASSWORD`；MinIO 用户名也通过 `ragflow-minio` 引用映射为 `MINIO_USER`。
- 逻辑 logs mount，`restart_policy=unless-stopped`。
- 不覆盖 image entrypoint；不挂载本地源码中的 `entrypoint.sh` 或 `service_conf.yaml.template`，以避免逻辑文件挂载权限和镜像内容漂移。

## RAGFlow connection matrix

| RAGFlow environment key | Value / source |
|---|---|
| `MYSQL_HOST` | `ragflow-mysql-mysql` |
| `MYSQL_PORT` | `3306` |
| `MYSQL_DBNAME` | `rag_flow` |
| `MYSQL_USER` | `root` |
| `REDIS_HOST` | `ragflow-redis-redis` |
| `MINIO_HOST` | `ragflow-minio-minio` |
| `ES_HOST` | `ragflow-elasticsearch-es01` |
| `MINIO_USER` / `MINIO_PASSWORD` | `ragflow-minio` Credential refs |
| `MYSQL_PASSWORD` | `ragflow-mysql` Credential ref |
| `REDIS_PASSWORD` | `ragflow-redis` Credential ref |
| `ELASTIC_PASSWORD` | `ragflow-elasticsearch` Credential ref |

RAGFlow 的镜像入口基于这些环境变量生成 `/ragflow/conf/service_conf.yaml`，并执行数据库表初始化。首次部署必须等待 MySQL、Redis、MinIO 与 Elasticsearch 已处于 Docker healthy / stable 状态后再启动 RAGFlow。

## Credential specification

创建四个同 Project 的 `runtime_env` Credential，名称与 Application code 对应：`ragflow-mysql`、`ragflow-redis`、`ragflow-minio`、`ragflow-elasticsearch`。每个值必须使用独立、强随机的部署秘密；不得复用 RAGFlow 源码 `.env` 的示例值。

Credential data 键仅为 `MYSQL_PASSWORD`、`REDIS_PASSWORD`、`MINIO_USER`、`MINIO_PASSWORD`、`ELASTIC_PASSWORD`。Version 只保存 `credential_id`、`data_key` 与目标 `env_key` 的引用。密码轮换后按依赖顺序重新部署相应依赖 Application，再重新部署 RAGFlow。

## Exposure and readiness

依赖服务维持仅集群内可达，不声明 `VersionExpose`。RAGFlow 只声明一个 HTTP Expose，入口策略在 Plan 前确定：

| 候选 | Compose 结果 | 适用条件 |
|---|---|---|
| local | `127.0.0.1:<RAGFLOW_LOCAL_HTTP_PORT>:80` | 本机开发、先验证 RAGFlow UI/API |
| public | 现有 Gateway 的 HTTP router | 已配置 Gateway、base domain、entrypoint 与 TLS 策略 |

RAGFlow 原 Compose 没有 Docker healthcheck。首版将 `curl -fsS http://localhost:80/` 写为候选 healthcheck，并在发布、部署成功后从受管 `ragflow-cpu` 容器执行固定的 `runtime_http_probe`。容器 running 不能替代该结果，且不得用手写 readiness proof 绕过失败。

## Deployment sequence

1. 执行所有 Deployment gate 检查，读取 Project / Gateway、验证 Credential 元数据、确认共享网络和宿主机资源。
2. 通过 MCP 创建四个 `runtime_env` Credential；创建五个 Application 和初始 Version，Preview 后发布 Version。所有 MCP 写响应只核对 resource ID 与脱敏摘要。
3. 依次部署 MySQL、Redis、MinIO、Elasticsearch；每个 Deployment 均等待终态成功，并运行 `runtime_compose_ps`、container inspect、日志读取与 `verify_deployment`。
4. 仅在所有依赖 stable 且 healthcheck 成功后部署 RAGFlow；等待终态、稳定性窗口和 RAGFlow Component healthcheck，再执行固定 HTTP Probe。
5. 通过受管 runtime 工具核对跨应用 alias、实际容器环境键（值必须脱敏）、数据目录、RAGFlow Compose config、deployment logs 与无正文 HTTP Probe 结论。
6. 入口为 local 时验证指定 loopback port；入口为 public 时验证 Gateway 路由。应用层验证命令和成功标准在 Plan 中固定，不能在实施时临时修改。

## Explicit exclusions

不部署 DeepDoc、TEI、NATS、Kibana、Sandbox、GPU、Go / hybrid API、其他向量引擎或原 Compose 的宿主机发布端口。NATS 在 `API_PROXY_SCHEME=python` 的首版不部署；若启用 Go / hybrid，必须先开立独立 Requirement。

## Technical questions

1. Elasticsearch 是否确认作为首版向量数据库？这是与当前 RAGFlow 默认 Compose 一致的草案选择；改为其他引擎需要重新审阅其镜像、配置、存储与健康模型。
2. 使用的 Orbit Project ID、`instance_key` 和 RAGFlow HTTP 入口模式 / 端口为何？
3. RAGFlow 镜像 `curl http://localhost:80/` 是否可作为稳定 healthcheck，还是应指定其他已验证端点？
4. 目标宿主机是否满足 Elasticsearch 8 GiB 内存、tmpfs、memlock、逻辑 data mount 的权限和备份要求？
5. MySQL、Redis、MinIO、Elasticsearch 数据目录的备份保留、恢复演练和可接受停机窗口为何？

## Risks

1. 五个 Application 共享 `traefik` 网络，任何网络缺失或 alias 冲突都会令 RAGFlow 连接失败。
2. 跨 Application 不支持声明式健康依赖；Runbook 顺序和验证遗漏会导致 RAGFlow 在依赖未就绪时失败。
3. Elasticsearch 的内存、tmpfs、memlock 与数据目录权限在 Docker Desktop 和 Linux Docker 上可能不同。
4. `runtime_env` 保护 Orbit / MCP 输出，但不能防御 Docker daemon 所在机器的高权限用户读取容器和 workspace 文件。
5. RAGFlow 启动时执行数据库初始化和模型表迁移；失败后需从 Deployment logs 判断，再经受管 restart / deploy 重试，不能手动改写 Compose。
6. 初始化器与任务一并行演进时，工具参数或 Component 字段可能漂移；脚本必须把未知/缺失能力记录为 `blocked`，不能降级为不含秘密引用或不含运行时约束的部署。

## User review notes

用户于 2026-07-26 要求完成任务二 Plan / 计划阶段，视为接受本规格。Technical questions 中的实际值将作为 Plan 的实施前参数；真实运行资源仍须等待环境移除和任务一 Verification。

用户于 2026-07-26 要求任务一与任务二并行开发；任务二的可并行范围和实际部署门槛按本规格的 Parallel development and integration 执行。

用户于 2026-07-26 要求将 RAGFlow 业务初始化写为可重复调用、可日志、可验证的 Python 脚本；缺失的任务一能力以显式 capability gate 占位。
