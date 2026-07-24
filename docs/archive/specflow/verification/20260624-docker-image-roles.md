# Docker 镜像角色启动方式改进验证
最后修改时间: 2026-06-24 22:22:52

Review status: Accepted

## Requirement alignment

按 `docs/requirement/20260624-docker-image-roles.md` 核对：

- 已将 `Dockerfile` 和 `Dockerfile.cn` 改为 `ENTRYPOINT ["backend-go"]` 与 `CMD ["serve"]`。
- 已移除镜像层 HTTP API 专用 `HEALTHCHECK`，避免 worker 角色继承错误健康检查。
- README 已展示默认启动 HTTP API、覆盖 command 启动 worker 的 `docker run` 用法。
- README 已展示 Compose 中 `api` 和 `worker` 使用同一 image、不同 command，并将 HTTP healthcheck 放在 `api` 服务上的示例。

## Spec alignment

不适用。轻量模式未创建独立 spec 文档，按 requirement / 需求核对。

## Plan alignment

不适用。轻量模式未创建独立 plan 文档，按 requirement / 需求核对。

## Actual diff summary

- `Dockerfile`
  - 移除 `HEALTHCHECK --interval=30s ... curl -f http://localhost:80/api/health`。
  - 将 `CMD ["backend-go", "serve"]` 改为 `ENTRYPOINT ["backend-go"]` + `CMD ["serve"]`。
- `Dockerfile.cn`
  - 同步移除镜像层 HTTP healthcheck。
  - 同步改为 `ENTRYPOINT ["backend-go"]` + `CMD ["serve"]`。
- `README.md`
  - 新增“Docker 镜像角色”章节，说明同一镜像启动 HTTP API 与 worker 的方式。
  - 新增 Compose 示例，将 HTTP healthcheck 放到 `api` 服务，将 `worker` 服务 healthcheck 禁用。
  - 补充 worker 执行 Docker / Compose 任务时应只给 worker 授权 Docker socket 或远程 Docker host。
- `docs/requirement/20260624-docker-image-roles.md`
  - 新增轻量需求文档。

## Expected vs actual changed files

| File | Expected | Actual | Result |
|------|----------|--------|--------|
| `Dockerfile` | 修改入口和默认命令，移除镜像层 healthcheck | 已修改 | Pass |
| `Dockerfile.cn` | 与 `Dockerfile` 保持一致 | 已修改 | Pass |
| `README.md` | 补充 Docker / Compose 使用方式 | 已修改 | Pass |
| `docs/requirement/20260624-docker-image-roles.md` | 记录轻量需求 | 已新增 | Pass |
| `docs/verification/20260624-docker-image-roles.md` | 记录验证结果 | 已新增 | Pass |

当前工作区还存在 `justfile` 修改，但该文件不是本次变更产生的内容，未纳入本次验证范围。

## Acceptance checklist

- [x] `Dockerfile` 使用 `ENTRYPOINT ["backend-go"]` 与 `CMD ["serve"]`。
- [x] `Dockerfile.cn` 使用 `ENTRYPOINT ["backend-go"]` 与 `CMD ["serve"]`。
- [x] `Dockerfile` 和 `Dockerfile.cn` 不再包含 `HEALTHCHECK`。
- [x] `pomelo-orbit:local` 镜像元数据显示 `Entrypoint=["backend-go"]`、`Cmd=["serve"]`、`Healthcheck=null`。
- [x] 默认运行 `pomelo-orbit:local` 会进入 HTTP API 角色。
- [x] 覆盖 command 为 `worker` 会进入后台 worker 角色。
- [x] README 展示 `docker run` API / worker 用法。
- [x] README 展示 Compose 中同一镜像不同 command 的 API / worker 服务示例。

## Command results

### Inspect image metadata

Command:

```bash
docker image inspect pomelo-orbit:local --format 'Entrypoint={{json .Config.Entrypoint}} Cmd={{json .Config.Cmd}} Healthcheck={{json .Config.Healthcheck}}'
```

Result:

```text
Entrypoint=["backend-go"] Cmd=["serve"] Healthcheck=null
```

### Run default API role briefly

Command:

```bash
timeout 8s docker run --rm --name pomelo-orbit-verify-api pomelo-orbit:local
```

Result:

```text
Exit code 124
time=2026-06-24T14:22:05.368Z level=INFO msg="http server started" addr=0.0.0.0:80
```

`124` 是 `timeout 8s` 主动终止长驻 API 进程的预期结果；日志已证明默认命令进入 HTTP API 角色。

### Run worker role briefly

Command:

```bash
timeout 8s docker run --rm --name pomelo-orbit-verify-worker pomelo-orbit:local worker
```

Result:

```text
Exit code 124
time=2026-06-24T14:22:17.711Z level=INFO msg="background worker started" worker_id=default concurrency=2
```

`124` 是 `timeout 8s` 主动终止长驻 worker 进程的预期结果；日志已证明覆盖 command 为 `worker` 会进入后台 worker 角色。

### Static checks

`Grep` 检查结果：

```text
Dockerfile:71:ENTRYPOINT ["backend-go"]
Dockerfile:72:CMD ["serve"]
Dockerfile.cn:78:ENTRYPOINT ["backend-go"]
Dockerfile.cn:79:CMD ["serve"]
```

未发现 `Dockerfile` / `Dockerfile.cn` 中残留 `HEALTHCHECK`。

README 关键内容检查结果包含：

```text
## Docker 镜像角色
docker run --rm -p 80:80 pomelo-orbit:latest
docker run --rm pomelo-orbit:latest worker
command: ["serve"]
healthcheck:
command: ["worker"]
healthcheck:
```

## Missed or expanded scope

- 未运行完整 Docker build；`pomelo-orbit:local` 已由用户自行构建，本次基于该镜像做基本验证。
- 未验证 `Dockerfile.cn` 构建出的镜像元数据；本次对 `Dockerfile.cn` 做静态一致性检查。
- 未做完整 API 健康检查请求或数据库/任务业务联调；本次验证范围是镜像角色入口、默认命令、worker command 覆盖和 healthcheck 移除。

## Risks

- 直接 `docker run` 不再自带 Docker health status；生产或 Compose 部署需要在 `api` 服务上声明 HTTP healthcheck。
- worker 完整执行业务任务仍依赖数据库、数据目录和 Docker / Compose 执行权限，本次只验证 worker 进程能够通过镜像 command 正确启动。
- 工作区存在非本次修改的 `justfile` 变更，提交前应由用户确认是否拆分或一起处理。

## Incomplete items

无本次范围内未完成项。

## Conclusion

验证通过。`pomelo-orbit:local` 镜像已经具备单镜像多角色启动能力：默认启动 HTTP API，覆盖 command 为 `worker` 时启动后台 worker；镜像层 HTTP healthcheck 已移除，README 已展示推荐的 Docker / Compose 使用方式。
