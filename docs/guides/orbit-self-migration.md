# Orbit 作为普通应用的停机迁移
最后修改时间: 2026-10-06 16:27:36

本指南依据 2026-10-06 Tencent 只读核查，操作入口为 `ssh tencent`。以下是后续执行步骤，未表示远端迁移已经完成。Orbit 使用普通 Application / Version / Service，通过目录和 Docker socket 挂载使用现有 DooD。

## 1. 前置准备

暂停提交新的 CI/CD 任务并等待已有任务终态。重查 Environment target revision、Orbit/Gateway 的确认及运行目录、实际 Compose project、镜像、挂载、Docker socket GID 与文件权限。当前核查身份为 `1000:1000`、附加组 `988`；这些不是产品默认值。

备份 PostgreSQL、现有 `.env` 和密钥配置、旧 Compose/启动入口，以及 Gateway 的 Compose、静态配置、dynamic、certs、acme 和发布恢复材料。回退 Compose 应固定实际已成功运行的 `pomelo-orbit:20261002-081749`（执行前重新核对并记录 image ID），原 Compose 文件指定的镜像与运行容器不同。确认回退镜像仍存在于目标 daemon。新镜像必须包含本次身份和共享挂载能力。

外部执行端连接同一 PostgreSQL，使用同一凭据加密配置并能访问受管 SSH target；必须包含消费部署队列的 worker，单独 MCP 不能执行任务。避免旧、新、临时 worker 在不同程序版本下同时消费迁移任务。不要在资料中记录 DSN、密钥或 token。

## 2. 准备普通应用规格

Fork 可编辑的 Orbit Version，配置镜像、`restart_policy=unless-stopped`、实际运行 `user` 和 Docker socket 附加组。声明以下挂载，保留已有数据来源：

| Source | Target | 类型 | 属性 |
| --- | --- | --- | --- |
| `/opt/pomelo-orbit/data` | `/app/data` | directory | absolute host-path，rw，`shared=true` |
| `/opt/pomelo-orbit/logs` | `/app/logs` | directory | absolute host-path，rw |
| `/opt/pomelo-orbit/.env` | `/app/.env` | file | absolute host-path，沿用实际访问模式 |
| `/var/run/docker.sock` | `/var/run/docker.sock` | file | absolute host-path，沿用实际访问模式 |

不把共享 data 改为新部署目录下的相对 `./data`，也不将整棵 `/opt/pomelo-orbit` 搬进自己的子目录。`.env` 继续由 Orbit 的现有启动加载读取；保存数据库和凭据加密/JWT 密钥。声明 `HOME=/app/data`，核对 `orbit.root`、`workspace.root` 和部署日志配置使用有挂载映射的容器内路径，并检查 UID 对 data/logs 可写。

声明 HTTP 80 endpoint，保留接入现有 `traefik` 网络，发布 Version。缺失的 mount/env/endpoint 必须在 Version 补齐，Service 不能新增未声明项。

Tencent SSH 新目录为 `/opt/pomelo-orbit/data/deployment/pomelo-orbit-default`；确保它不存在或为空，不复制旧 Compose。local DooD 的同类目录使用容器内路径 `/app/data/deployment/pomelo-orbit-default`，复用已有 daemon 路径映射。不要更改 target_type 或 target revision 来规避目录冲突。

## 3. 停机与首次部署

在宿主机停止旧 project，保留 volumes：

```sh
docker compose -p pomelo-orbit -f /opt/pomelo-orbit/docker-compose.yml stop
```

停用旧 `boot.sh` 或其他自动启动入口，确认旧容器已停止，再由外部 worker 执行普通部署：选择新 Version、Service `pomelo-orbit-default` 和新目录。正常部署保存确认目录、创建 Deployment，准备 Compose 成功后绑定运行目录，然后执行 Compose。

先迁移 Orbit，再处理 Gateway。旧 Orbit 确认目录仍是 `/opt/pomelo-orbit` 时，即使已停机也会阻止子目录内其他服务部署。新 project 为 `pomelo-orbit-default`，预期容器名为 `pomelo-orbit-pomelo-orbit`，不要沿用手工容器名推断状态。

等待 Deployment 终态，核对新容器的 `id`、附加组、socket 操作、实际四项挂载、共享标签、数据库和凭据解密。DooD 启动仍须能 inspect 自身及映射工作区/日志路径。容器运行不等于自定义域名已发布。

## 4. Gateway 与域名

Gateway 保持原目录 `/opt/pomelo-orbit/data/deployment/traefik-default`、project `traefik-default`。先核对真实 Environment、Service code 和原 Compose 的用途，备份 Compose 后从 Gateway 入口确认原目录并正常重部署，生成受管配置和运行绑定。当前部署不要求 owner 标记；通过其他服务根目录及实际挂载占用检查后，会替换已有 Compose。保留 dynamic/certs/acme，不伪造数据库部署成功或运行目录。参见 [确认部署目录](./deployment.md#确认部署目录)。

外部或内部 worker 都可执行，因为共享规则依据已部署的容器标签与实际 bind。Gateway 就绪后，配置 `orbit.preflite.cn`、`orbit-api.preflite.cn` 两条 Route，目标为新 Service 的 `pomelo-orbit` component / HTTP 80 endpoint，使用实际支持的 entrypoint 与 resolver。先预览 Route sync，再显式确认冻结修订；普通部署不会自动发布 Route。检查 TLS/SNI、登录和 API。

## 5. 持续部署验收与回退

明确退出临时 worker 后，由新 Orbit worker 在自身在线、共享 data 挂载仍存在时，对隔离验收服务执行新部署、受控文件更新后重部署和 restart。核对 CI 工作区、日志、Route 文件路径。普通未共享 bind 冲突和 Service 根目录冲突应继续拒绝。一次迁移成功或外部 worker 成功不能替代这项持续部署验收。

Orbit 更新自身仍须外部 worker：停止执行任务的自身容器后，该 worker 无法继续完成操作。此约束与共享目录判定无关。

失败时终止迁移任务与临时 worker，停止新 project，保留目录、共享数据和日志；撤销本次两条 Route 发布或处理其 pending 恢复材料，不删除其他 Route/ACME。按故障范围恢复 Gateway 文件和原 project，再使用固定已知成功镜像的旧 Compose 启动 `pomelo-orbit`，恢复旧启动入口，复核两域名、数据库、密钥和 Docker 权限。数据库仅在实际不可恢复不兼容并决定恢复时使用停机备份；不清空 data、不重置密钥，也不将手工回退标记为受管部署成功。
