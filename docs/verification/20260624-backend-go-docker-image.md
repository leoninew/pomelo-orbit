# backend-go Docker 镜像迁移验证
最后修改时间: 2026-06-24 13:07:00

Review status: Draft

## Requirement alignment

按 `docs/requirement/20260624-backend-go-docker-image.md` 核对，当前 diff 与轻量模式 / light 的需求范围基本一致：根目录 Docker 镜像后端目标迁移到 `backend-go`，保留前端单镜像部署，补齐静态文件与 SPA fallback，并将 docker publish workflow 的后端检查切换为 Go tests。

## Spec alignment

不适用。轻量模式 / light 未创建 spec 文档。

## Plan alignment

不适用。轻量模式 / light 未创建 plan 文档；按 requirement / 需求核对。

## Actual diff summary

- `.dockerignore`：新增 `backend-go` 构建、临时、数据和日志目录忽略项。
- `.github/workflows/docker-publish.yml`：从 Python/uv/pytest 检查切换为 `actions/setup-go` 与 `go -C backend-go test ./...`。
- `Dockerfile`：后端构建阶段从 Python backend 改为 `golang:1.25-bookworm` 构建 `backend-go` binary；final image 改为 Debian，复制 `backend-go` binary、`backend-go/config.defaults.yaml` 与前端 `dist`，设置 `POMELO_ORBIT_SERVER__HOST=0.0.0.0` 和 `POMELO_ORBIT_SERVER__PORT=80`，启动命令改为 `backend-go serve`。
- `Dockerfile.cn`：与 `Dockerfile` 保持同等 backend-go 迁移逻辑，并保留 npm、Go proxy、apt、Compose 下载镜像加速意图。
- `backend-go/internal/transport/http/server.go`：对非 `/api/` 的 GET/HEAD 未命中请求增加 `static` 目录静态文件服务与 `index.html` fallback；`/api/` 未命中仍返回 JSON 404。
- `backend-go/internal/transport/http/server_test.go`：新增静态资源、SPA fallback、API 404 和非 GET fallback 行为测试。
- `docs/requirement/20260624-backend-go-docker-image.md`：记录并接受本次轻量需求。

## Expected vs actual changed files

| 文件 | 需求预期 | 实际状态 |
| --- | --- | --- |
| `Dockerfile` | 迁移到 backend-go 单镜像 | 已修改，符合预期 |
| `Dockerfile.cn` | 与默认 Dockerfile 同步迁移并保留国内加速 | 已修改，符合预期 |
| `.github/workflows/docker-publish.yml` | 发布前运行 backend-go tests | 已修改，符合预期 |
| `backend-go/internal/transport/http/server.go` | 补齐静态文件与 SPA fallback，保持 `/api/` JSON 404 | 已修改，符合预期 |
| `backend-go/internal/transport/http/server_test.go` | 覆盖静态文件与 fallback 行为 | 已修改，符合预期 |
| `.dockerignore` | 可忽略 backend-go 运行/构建产物 | 已修改，属于支持性改动，未超出目标 |
| `docs/requirement/20260624-backend-go-docker-image.md` | 过程文档 | 已新增，符合 SpecFlow 要求 |

## Acceptance criteria checklist

- [x] `Dockerfile` 使用 Go build stage 构建 `backend-go/cmd/backend-go`，final image 不再依赖 Python backend runtime。
- [x] `Dockerfile.cn` 与 `Dockerfile` 保持同等 backend-go 迁移逻辑，并保留国内镜像加速意图。
- [x] final image 复制 `backend-go` binary、`backend-go/config.defaults.yaml` 和前端 `dist` 到运行目录。
- [x] final image 设置 `POMELO_ORBIT_SERVER__HOST=0.0.0.0` 与 `POMELO_ORBIT_SERVER__PORT=80`。
- [x] final image 创建运行所需目录，但不复制或迁移旧数据目录。
- [x] `backend-go` 对非 `/api/` 路径提供静态文件或 `index.html` fallback；对 `/api/` 未命中路径仍返回 JSON 404。
- [x] `.github/workflows/docker-publish.yml` 不再运行 Python backend tests，改为运行 backend-go tests。

## Command results

按当前验证过程已发生的检查记录如下：

- `gofmt -l backend-go/internal/transport/http/server.go backend-go/internal/transport/http/server_test.go`：通过，无输出。
- `go -C backend-go vet ./...`：通过，无输出。
- `go -C backend-go test ./...`：通过。
- `go -C backend-go test -count=1 ./...`：通过。

说明：曾误执行一次 `docker build -f Dockerfile -t pomelo-orbit:verify .`，结果通过，但本验证结论不依赖该镜像构建结果；按用户要求，不再继续做镜像级验证。

## Missed or expanded scope

- 未发现产品代码范围外扩展。
- `.dockerignore` 的 backend-go 忽略项属于镜像构建上下文支持性改动，未改变业务行为。
- 未运行 `Dockerfile.cn` 镜像构建和容器运行时 smoke test；按用户要求，本次不继续做镜像验证。

## Risks

- `Dockerfile` 与 `Dockerfile.cn` 使用 `golang:1.25-bookworm`，依赖该 tag 可用性；当前仅从代码与已发生默认 Dockerfile build 结果看未发现问题。
- 静态文件服务基于工作目录下 `static/index.html`，容器 `WORKDIR /app` 与 `COPY ... ./static` 保持一致；如果未来运行目录变化，需要同步调整。
- 发布 workflow 切到 backend-go tests 后，Python backend regression 不再阻塞镜像发布，符合迁移方向但改变 CI 覆盖边界。

## Incomplete items

暂无必须修复项。若要做更完整的运行时验收，需要用户明确允许后再执行镜像/容器级验证。

## Conclusion

基于 RSP 文档和当前 diff 静态核对，当前实现满足 `backend-go Docker 镜像迁移需求` 的验收标准。已发生的 Go 格式、vet 和测试检查均通过。验证文档状态保持 `Draft`，等待用户审阅确认。
