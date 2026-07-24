# Debian Trixie apt 安装 Docker Compose 改造验证
最后修改时间: 2026-06-24 18:55:16

## Review status

Accepted

## Requirement alignment

按 `docs/requirement/20260624-trixie-compose-apt.md` 核对，本次变更符合需求：

- `Dockerfile.cn` 最终阶段已改为 `debian:trixie-slim`。
- `Dockerfile` 最终阶段已改为 `debian:trixie-slim`。
- 两个 Dockerfile 都通过 apt 安装 Docker Compose v2 所需包。
- 两个 Dockerfile 都移除了直接从 GitHub 或 ghproxy 下载 Docker Compose release binary 的步骤。
- 两个 Dockerfile 都在构建期执行 `docker compose version` 作为可用性断言。
- 已分别执行本地 `docker build` 并验证成功。

## Spec alignment

不适用。当前为轻量模式 / light，本变更未创建独立 spec 文档。

## Plan alignment

不适用。当前为轻量模式 / light，本变更未创建独立 plan 文档；按 requirement / 需求直接实现和验证。

## Actual diff summary

- `Dockerfile`：最终阶段基础镜像从 `debian:bookworm-slim` 更新为 `debian:trixie-slim`；apt 安装列表新增 `docker-cli` 和 `docker-compose`；删除 GitHub release binary 下载步骤；新增构建期 `docker compose version` 断言。
- `Dockerfile.cn`：最终阶段基础镜像从 `debian:bookworm-slim` 更新为 `debian:trixie-slim`；保留清华 Debian 镜像源替换；apt 安装列表新增 `docker-cli` 和 `docker-compose`；删除 ghproxy/GitHub release binary 下载步骤；新增构建期 `docker compose version` 断言。
- `docs/requirement/20260624-trixie-compose-apt.md`：新增轻量模式需求文档，记录背景、目标、非目标、验收标准、决策和风险。
- `docs/verification/20260624-trixie-compose-apt.md`：新增当前验证文档。

## Expected vs actual changed files

### Expected

- `Dockerfile`
- `Dockerfile.cn`
- `docs/requirement/20260624-trixie-compose-apt.md`
- `docs/verification/20260624-trixie-compose-apt.md`

### Actual for this scope

与预期一致。

### Working tree note

验证时 `git status --short` 还显示以下不属于本次 scope 的变更：

- `backend-go/internal/service/ci/repository.go`
- `backend-go/internal/service/ci/repository_test.go`
- `backend-go/internal/transport/http/repository_routes_test.go`
- `docs/requirement/20260624-backend-go-repository-variables.md`

本次未修改这些文件，验证结论只覆盖 Docker Compose apt 改造相关文件。

## Acceptance checklist

- [x] `Dockerfile.cn` 最终阶段使用 `debian:trixie-slim`。
- [x] `Dockerfile` 最终阶段使用 `debian:trixie-slim`。
- [x] 两个 Dockerfile 都通过 apt 安装 `docker-compose`。
- [x] 两个 Dockerfile 都显式安装 `docker-cli`，避免 `--no-install-recommends` 下 `docker` 主命令缺失。
- [x] 两个 Dockerfile 都删除直接下载 Docker Compose release binary 的 `curl ... github.com/docker/compose ...` 步骤。
- [x] 两个 Dockerfile 都在构建期执行 `docker compose version`。
- [x] 本地 `docker build -f Dockerfile.cn ... .` 成功。
- [x] 本地 `docker build -f Dockerfile ... .` 成功。
- [x] 最终镜像运行时 `docker compose version` 可执行。

## Command results

### `Dockerfile.cn` build

Command:

```powershell
docker build --progress=plain -t pomelo-orbit:specflow-trixie-cn -f Dockerfile.cn .
```

Result: 成功。

关键输出：

```text
Docker Compose version 2.26.1-4
naming to docker.io/library/pomelo-orbit:specflow-trixie-cn
DONE
```

说明：首次验证发现 `docker.io + docker-compose` 在 `--no-install-recommends` 下缺少 `docker` 主命令，错误为 `/bin/sh: 1: docker: not found`。已通过显式安装 `docker-cli` 修复，并重新构建通过。

### `Dockerfile` build

Command:

```powershell
docker build --progress=plain -t pomelo-orbit:specflow-trixie -f Dockerfile .
```

Result: 成功。

关键输出：

```text
Docker Compose version 2.26.1-4
naming to docker.io/library/pomelo-orbit:specflow-trixie
DONE
```

完整输出因超过工具显示限制，保存于：

```text
C:\Users\wangm25\.claude\projects\D--SourceCodes-mywork-pomelo-orbit\d86aed50-9a3a-4bee-bb18-4ea5682df4a1\tool-results\bb90uofcg.txt
```

### Runtime command check: `Dockerfile.cn` image

Command:

```powershell
docker run --rm pomelo-orbit:specflow-trixie-cn docker compose version
```

Result: 成功。

Output:

```text
Docker Compose version 2.26.1-4
```

### Runtime command check: `Dockerfile` image

Command:

```powershell
docker run --rm pomelo-orbit:specflow-trixie docker compose version
```

Result: 成功。

Output:

```text
Docker Compose version 2.26.1-4
```

### Direct file/content checks

Command:

```powershell
git diff -- Dockerfile Dockerfile.cn docs/requirement/20260624-trixie-compose-apt.md
```

Result: 确认两个 Dockerfile 只在最终阶段基础镜像和 Docker/Compose 安装逻辑上发生预期变更。

Search:

```text
github.com/docker/compose|ghproxy.net/.*/docker/compose|docker-compose-linux-x86_64
```

Result: `Dockerfile*` 中无匹配，确认已移除 release binary 下载残留。

## Missed or expanded scope

- 本次没有修改前端源码，因此未额外运行 `yarn lint --fix && yarn typecheck`。
- 本次没有启动应用服务，也没有执行端到端功能部署验证；验证聚焦于本地 Docker build 和最终镜像内 `docker compose` 可用性。
- 实现过程中根据验证结果扩展安装 `docker-cli`，这是满足 `docker compose version` 断言和运行时命令可用性的必要修正，不属于加戏。

## Risks

- 最终阶段基础系统从 Debian bookworm 升级到 Debian trixie，运行时系统包版本随之变化。
- Compose 版本由 Debian trixie apt 仓库决定，当前验证到的是 `Docker Compose version 2.26.1-4`，不再是原先手动下载的 `v5.1.0`。
- `Dockerfile.cn` 仍依赖清华 Debian 镜像源可用性；本次只移除了 ghproxy/GitHub release binary 下载依赖。

## Incomplete items

无本次 scope 内未完成项。

## Conclusion

验证通过。`Dockerfile` 和 `Dockerfile.cn` 均可本地构建成功，最终镜像均能执行 `docker compose version`，并且不再依赖 GitHub 或 ghproxy 下载 Docker Compose release binary。
