# RAGFlow 拆分部署 Runbook
最后修改时间: 2026-07-26 20:06:12

本文档是任务二的实施工件，不是对 `D:\SourceCodes\opensource\ragflow\docker\docker-compose.yml` 的直接导入。权威期望状态位于同目录的 `ragflow-split-deployment.fixture.yaml`；`scripts/ragflow_initialize.py` 是将其转化为受管 MCP 操作的唯一业务初始化入口。两者均不包含实际 Credential value、Project ID、资源 ID 或宿主机路径。

## Scope

目标为五个独立的 `standard` Application：`ragflow-mysql`、`ragflow-redis`、`ragflow-minio`、`ragflow-elasticsearch` 和 `ragflow`。它们以同一 Project 和 instance key 运行，通过平台注入的外部 `traefik` 网络以及 `<application-code>-<component>` 别名相互访问。

不导入上游 Compose 的 `include`、`profiles`、`env_file`、`depends_on`、宿主机端口、挂载的 entrypoint 或 `service_conf.yaml.template`。依赖服务不创建 Expose；RAGFlow 仅选择 local 或 public 中的一种 HTTP Expose 变体。

## Ownership

任务一拥有 `20260726-orbit-deployment-capability-gaps` 的 Version 与 Credential 契约。本拆分任务消费该契约，并拥有 Gateway MCP、受管 `traefik` 预检和固定 HTTP Probe；这些能力仅服务于本 Runbook 的 Gateway 前置条件与 RAGFlow 就绪验证。

并行期间，任务二可以完成 fixture 审阅、上游镜像/健康检查研究、初始化器和非秘密 Preview 集成用例。不得创建实际 `runtime_env` Credential、Application、Version、Service 或 Deployment，也不得用临时 `runtime_config` 注入密码。

## Initializer Usage

初始化器经 MCP 的 Python 环境运行：

```text
uv --directory mcp run python ../scripts/ragflow_initialize.py plan
uv --directory mcp run python ../scripts/ragflow_initialize.py check --project-id <PROJECT_ID> --instance-key <INSTANCE_KEY> --access <local|public> --capability-contract <TASK1_CONTRACT> --gateway-application-id <GATEWAY_APPLICATION_ID> [--gateway-instance-key <GATEWAY_INSTANCE_KEY>]
uv --directory mcp run python ../scripts/ragflow_initialize.py apply --project-id <PROJECT_ID> --instance-key <INSTANCE_KEY> --access <local|public> --capability-contract <TASK1_CONTRACT> --gateway-application-id <GATEWAY_APPLICATION_ID> [--gateway-instance-key <GATEWAY_INSTANCE_KEY>] --secrets-file <OUTSIDE_REPOSITORY_PATH> --confirm
uv --directory mcp run python ../scripts/ragflow_initialize.py apply --project-id <PROJECT_ID> --instance-key <INSTANCE_KEY> --access <local|public> --capability-contract <TASK1_CONTRACT> --gateway-application-id <GATEWAY_APPLICATION_ID> [--gateway-instance-key <GATEWAY_INSTANCE_KEY>] --secrets-file <OUTSIDE_REPOSITORY_PATH> --confirm --resume <RUN_DIRECTORY>
uv --directory mcp run python ../scripts/ragflow_initialize.py verify --resume <RUN_DIRECTORY> --capability-contract <TASK1_CONTRACT>
```

`plan` 不连接 MCP；`check` 只读检测能力和冲突；只有 `apply --confirm` 才允许写入。每次执行创建一个仅含脱敏参数、fixture 摘要、步骤状态和 MCP resource ID 的运行目录。`--resume` 在任何 Version 发布或 Deployment 创建前可自动重基 fixture，并通过 MCP 更新同一 journal 已记录的草稿 Version；发布或部署后仍拒绝跨 fixture 恢复。同名 Application 或 Credential 仅在 ID 已记录于该运行目录时允许继续，避免误接管外部资源。缺失任务一能力时，`check` / `apply` 输出 `blocked` 并列出缺少的工具。

`<TASK1_CONTRACT>` 是非秘密 JSON 文件，至少包含下列字段名；它仅证明任务一的最终 Version Component 契约已经完成集成核对：

```json
{
  "version_component_fields": [
    "restart_policy",
    "tmpfs_json",
    "ulimits_json",
    "secret_env_refs"
  ]
}
```

`<GATEWAY_APPLICATION_ID>` 必须指向已创建、发布并部署成功的 Orbit Gateway Application。先通过 `orbit_create_gateway` 创建 Gateway，再使用既有 `orbit_publish_version`、`orbit_deploy` 和 `orbit_wait_deployment` 完成其部署；不要用 Docker CLI 创建 `traefik` 网络。`--gateway-instance-key` 默认 `default`。初始化器随后以该受管 target 调用 `runtime_doctor`，只接受其派生出的 bridge `traefik` 网络。

RAGFlow Version 使用 fixture 中的候选 Docker healthcheck。`verify` 在 RAGFlow Deployment 成功并通过 `verify_deployment` 后，调用受管 `runtime_http_probe` 对 `ragflow-cpu` 的固定 `http://127.0.0.1:80/` 进行无正文 Probe；journal 仅记录状态、组件、端口和路径。

## Live Integration Gate

真实 MCP 写入前，必须全部满足：

1. 环境移除最终版本已合入，MCP 与 Deployment 只使用 `application_id + instance_key`，不含 `environment_id`。
2. 任务一的 VersionComponent 字段和受限 Credential 工具已实现；fixture 的 `restart_policy`、`tmpfs`、`ulimits`、`secret_env_refs` 能被其最终契约无歧义地编码。
3. 任务一 Verification 已证明 Credential value 不会出现在 Version、Preview、Deployment options、MCP 响应、日志、inspect 或验证证据中。
4. 已部署的 Gateway Application 作为 `runtime_doctor` 的受管 target，返回 `healthy=true`，并明确报告 bridge `traefik` 网络可用。缺失 Gateway、目标不是 `kind=gateway`、网络未由该 target 派生或网络不是 bridge 时，初始化器以 `runtime_doctor:external_network:traefik` 输出 `blocked`，不得用普通 Docker 命令绕过。
5. 已确定 Project、instance key、RAGFlow local/public 入口、端口或 Gateway 策略、备份位置和维护窗口。
6. 已接受 fixture 的候选 `infiniflow/ragflow:v0.26.4` Docker healthcheck；RAGFlow 部署后必须通过固定的受管 HTTP Probe，不能用容器 running 或手写 JSON 证明替代。
7. 宿主机已验证 Elasticsearch 的 8 GiB 内存限制、tmpfs、memlock、受管逻辑数据目录和备份空间。

## Fixture Encoding

fixture 是语义 YAML，MCP 写入前必须执行以下无损转换：

| Fixture field | Version Component field | Encoding |
| --- | --- | --- |
| `command` | `command_json` | JSON string array |
| `environment` | `env_json` | JSON array of `{key,value}` |
| `mounts` | `mounts_json` | JSON array of structured logical mounts |
| `healthcheck` | `healthcheck_json` | JSON object |
| `resources` | `resources_json` | JSON object |
| `restart_policy` | task 1 field | restricted enum |
| `tmpfs` | task 1 field | JSON array |
| `ulimits` | task 1 field | JSON array |
| `secret_env_refs` | task 1 field | structured array, after credential names resolve to IDs |

`$${NAME}` 出现在 Redis、MySQL 和 Elasticsearch 的 command / healthcheck 中。它必须原样进入 Compose，使 Docker Compose 将单个 `$` 交给容器 shell 展开；不得替换成真实密码，也不能使用 `${NAME}` 让宿主 Compose 解析。

## Apply Sequence

1. 使用 MCP 只读查询确认目标 Project、Gateway、Application 和 Service 不冲突；创建、发布并部署明确配置的 Gateway，等待其成功后以 `runtime_doctor(gateway_application_id, gateway_instance_key)` 确认其派生的 bridge `traefik` 网络。冲突或预检失败时停止，不收编、删除或手动创建网络。
2. 创建四个 `runtime_env` Credential。值仅从受控秘密来源传入 MCP 工具；响应仅记录 ID、名称和类型，Runbook 和 fixture 不记录 value。
3. 为五个 Application 创建初始 Version。每个 Component 按 fixture 编码，所有 `credential_name` 替换为同 Project Credential ID，且绝不将解密值放入 `env_json` 或 `runtime_config`。
4. 对每个 Version 调用 Environment-free Preview，核对 Compose 项目名、workspace、别名、逻辑挂载、healthcheck、restart、tmpfs、ulimits 和脱敏占位。Preview 不得含实际秘密。
5. 发布 Version 后依次部署 MySQL、Redis、MinIO、Elasticsearch。每个部署到达成功终态后，使用 `runtime_compose_config`、`runtime_compose_ps`、`runtime_container_inspect`、`runtime_compose_logs` 和 `verify_deployment` 检查一致性。
6. 仅当四个依赖均稳定、healthy 且 `verify_deployment` 为 `consistent` 时部署 RAGFlow。只创建一个已选 HTTP Expose；完成 RAGFlow Deployment 后执行 `verify`，其 `verify_deployment` 与固定 `runtime_http_probe` 都必须成功。
7. 记录脱敏后的 Application、Version、Service、Deployment ID、Gateway Application ID、Probe 结论、备份位置和恢复责任。不要将这些运行时值回写到 fixture。

## Failure And Rollback

任一依赖出现 `faulted`、`canceled`、超时、unhealthy、重启计数增加、Compose 漂移或秘密泄漏时，停止后续步骤并保留脱敏诊断证据。不得通过修改受管 Compose、手动 Docker 生命周期命令或改变 fixture 中的秘密占位绕过失败。

回滚顺序为 RAGFlow、Elasticsearch、MinIO、Redis、MySQL。所有 lifecycle 操作通过 MCP 执行且使用 `remove_volumes=false`；逻辑数据目录保留至备份、恢复或重新部署决定完成。

## Source Mapping

fixture基于上游 `docker/docker-compose-base.yml` 的 MySQL、Redis、MinIO、Elasticsearch 和 `docker/docker-compose.yml` 的 CPU RAGFlow 服务。上游 `env_file` 的默认秘密值被完全排除，原 Compose 对外端口和跨服务 `depends_on` 被受管网络别名、MCP 部署顺序和运行时验证替代。
