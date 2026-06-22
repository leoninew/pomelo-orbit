# backend-go CD Docker Compose 物理路径修复
最后修改时间: 2026-06-22 11:38:01

Review status: Accepted

## Background

用户在部署 Traefik 应用时看到部署日志报错：

```text
Working directory: data\cd\traefik
$ bash -x init.sh
...
$ docker compose -f docker-compose.yml up -d --remove-orphans --pull missing
service "traefik" refers to undefined volume data\cd\traefik/data/traefik.yml: invalid compose project
```

实际生成的 `backend-go/data/cd/traefik/docker-compose.yml` 中包含：

```yaml
volumes:
  - data\cd\traefik/data/traefik.yml:/etc/traefik/traefik.yml:ro
  - data\cd\traefik/data/dynamic:/etc/traefik/dynamic:ro
  - data\cd\traefik/data/certs:/etc/traefik/certs:ro
  - data\cd\traefik/data/acme.json:/etc/traefik/acme.json
```

Docker Compose 将 `data\cd\traefik/data/traefik.yml` 解释为 named volume，而不是宿主机 bind mount source，导致 volume 名称非法。

已有提交 `0bbe6393418003705d47ee7aa48e8bea6701cbf3` 修复了 backend-go CI Docker bind mount 的同类问题：它把 logical data path 与 physical host path 分层，并在容器模式下通过 Docker mount metadata 解析宿主机 data mount source。但该提交明确没有修改 CD 执行业务逻辑，因此 CD compose 模板仍把 logical path 当作 physical path 使用。

## Current investigation

### backend-go CD 当前问题

`backend-go/internal/service/cd/application_extra.go` 中 `renderApplicationTemplate` 当前构造模板上下文：

```go
"physical_dir":     cfg.DataRoot(),
"physical_app_dir": filepath.Join(cfg.DataRoot(), "cd", appCode),
```

当 `cfg.DataRoot()` 是默认相对路径 `data` 时，Windows 下 `app.physical_app_dir` 渲染为：

```text
data\cd\traefik
```

Traefik 初始化模板在 `backend-go/internal/migrations/{sqlite,mysql}/v0.1.4__business_data.sql` 中使用：

```yaml
- {{ app.physical_app_dir }}/data/traefik.yml:/etc/traefik/traefik.yml:ro
```

因此最终生成裸相对 Windows 路径，传给 Docker Compose 后报错。

### 旧 backend 对等实现

旧 Python backend 的 `backend/src/pomelo_orbit/infrastructure/cd/docker/manager.py` 在模板渲染时区分运行环境：

- 容器内运行：通过 `detect_container_id()` 和 `get_container_mount_source(container_id, "/app/data")` 找宿主机 data mount source，再生成 `physical_dir` / `physical_app_dir`。
- 本地运行：`physical_dir` 使用项目根目录，`physical_app_dir` 对普通应用可使用当前 compose 工作目录相对路径。

这说明 `physical_app_dir` 的语义本来就是 Docker Compose 可用的宿主机路径/工作目录路径，而不是 backend-go 当前的 logical data path。

### CI 修复可复用原则

`0bbe639...` 的 CI 修复已经确立原则：

- logical data path：backend-go 自身读写日志、workspace、artifacts、应用配置文件。
- physical data path：传给 Docker 子容器或 Docker Compose 的宿主机 bind mount source。
- 容器模式下不能默默使用容器内路径；必须解析宿主机 mount source，失败时显式报错。
- 不接受只在 runner 或模板里临时 `filepath.Abs` 的局部修补。

## Goal

- 修复 backend-go CD 部署中 `app.physical_dir` / `app.physical_app_dir` 使用 logical path 的问题。
- 让 Docker Compose 模板中的 host bind mount source 使用 Docker 可接受的 physical path。
- 复用或抽取 `0bbe639...` 中 CI 已有的 physical data root 解析能力，避免 CI/CD 两套实现漂移。
- 保持 CD 自身读写文件、部署日志、应用目录仍使用 logical data path。
- Traefik 重新部署时生成的 `docker-compose.yml` 不再包含 `data\cd\...` 这类裸相对 Windows 路径。
- 在本地 Windows 和容器化 backend-go 两类运行方式下都保持可诊断：本地解析为绝对宿主路径；容器模式无法解析 mount source 时显式失败。

## Non-goal

- 不手工修改 `backend-go/data/cd/traefik/docker-compose.yml` 作为最终修复；它是运行时产物，应由重新部署重新渲染。
- 不修改旧 Python backend。
- 不修改前端。
- 不改 Docker Compose 模板变量名，不新增兼容变量或别名。
- 不保留 logical path 和 physical path 两套模板分支。
- 不把 CD 的路径修复写成只针对 Traefik 的特例。
- 不在 Docker runner 或 ShellRunner 中做业务路径猜测。
- 不修改已执行迁移文件来修复现有业务数据；除非后续证明模板内容本身有错，本需求只修正模板上下文。

## User scenarios

1. 开发者在 Windows 本地运行 backend-go 并部署 Traefik：
   - CD 写文件目录仍为 `data/cd/traefik`。
   - 渲染后的 compose volume source 使用绝对宿主路径，并使用 slash 格式。
   - Docker Compose 不再把 source 误判为 named volume。

2. backend-go 在容器中运行并通过 Docker socket 部署应用：
   - CD 使用 backend-go 容器的 data mount metadata 解析宿主机 data 目录。
   - 模板中的 `app.physical_app_dir` 指向宿主机 data mount 下的 `cd/<app_code>`。
   - 如果 data 目录没有作为可解析 bind mount 挂载，部署失败并给出明确错误。

3. 预览 docker-compose：
   - compose preview 与实际部署使用同一模板上下文和 physical path 解析逻辑。
   - preview 不展示与部署不同的一套路径。

4. 其他使用 `app.physical_dir` / `app.physical_app_dir` 的内置应用（例如 filebrowser）：
   - 也获得一致的 physical path 语义，不需要应用级特殊处理。

## Acceptance

- backend-go 不再在 CD Docker Compose 模板中把 `cfg.DataRoot()` 或 `filepath.Join(cfg.DataRoot(), "cd", appCode)` 直接作为 `physical_dir` / `physical_app_dir`。
- CI 和 CD 复用同一个 physical data root resolver，或 resolver 被抽到中性包后两边调用同一实现。
- CD 领域有内聚的 path/workspace 组件，区分：
  - logical app dir / deployment log path：backend-go 自己读写。
  - physical data root / physical app dir：渲染给 Docker Compose。
- 本地相对 `dataRoot` 会解析为绝对 physical data root。
- 渲染给 Docker Compose 的 `physical_dir` / `physical_app_dir` 使用 `filepath.ToSlash` 或等价处理，避免 Windows 反斜杠和裸相对路径。
- 容器模式下无法解析当前容器 data mount source 时，CD 部署返回明确错误，不 fallback 到容器内 logical path。
- `ApplicationComposePreview` 和实际部署使用同一模板上下文逻辑。
- 相关 Go 测试覆盖：
  - CD liquid 模板渲染不再输出裸 `data\cd\...`。
  - `POMELO_ORBIT` 默认相对 data root 情况下，compose 中 volume source 是 physical path。
  - resolver 失败时部署失败且 deployment 标记 faulted。
  - CI 现有 physical mount 测试继续通过。
- `cd backend-go && go test ./...` 通过。

## Open questions

暂无需要用户决策的阻塞问题。默认按“对等 CI 修复 + 旧 backend 语义”推进。

需要注意：当前工作区已有上一项 `backend-go` runtime config 收敛的未提交改动。实施本需求时应避免混淆 diff 边界；最终汇报需要明确哪些文件属于本需求，哪些属于已有未提交改动。

## Decisions

- 采纳“对等修复”方案：不做 Traefik 特例，不做单点绝对化，不在 runner 中猜路径。
- 抽取或复用 CI 的 physical data root resolver，使 CI/CD 对 Docker 宿主路径的判断一致。
- CD 模板变量名保持不变：`app.physical_dir` / `app.physical_app_dir`，但修正其值为符合名称的 physical path。
- physical path 解析失败时 fail fast，避免生成看似可用但 Docker 会误判的 compose 文件。
- 运行时产物 `backend-go/data/cd/...` 不作为源码修复对象；修复后通过重新部署刷新。

## User review notes

- 用户指出 `0bbe6393418003705d47ee7aa48e8bea6701cbf3` 已经修过一波路径问题，要求结合之前修复评估。
- 用户要求参考旧 `backend` 实现，并注意全局原则：不要无声改代码、不要无限容忍错误、业务逻辑和架构要严谨一致、不要引入技术债。
