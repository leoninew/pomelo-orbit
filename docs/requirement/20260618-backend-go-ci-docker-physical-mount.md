# backend-go CI Docker 宿主机物理路径挂载
最后修改时间: 2026-06-18 16:48:06

Review status: Accepted

## Background

从仓库详情触发流水线后，git clone 阶段启动 Docker 容器失败，错误信息为：

```text
docker: Error response from daemon: create data\ci\awesome-compose\workspace: "data\\ci\\awesome-compose\\workspace" includes invalid characters for a local volume name, only "[a-zA-Z0-9][a-zA-Z0-9_.-]" are allowed. If you intended to pass a host directory, use absolute path
```

当前 backend-go 在执行 CI stage 时直接使用 `filepath.Join(e.dataRoot, "ci", ...)` 生成 Docker volume 的 host path。当 `dataRoot` 为默认相对路径 `data` 时，Docker CLI 收到的是 `data\ci\...`，会将其解释为 named volume，而不是宿主机 bind mount 路径，导致 Windows 下路径非法。

Python backend 的实现不是简单做 `abs(data)`，而是把应用内部数据路径和 Docker 挂载所需的宿主机物理路径分开：

- 本地模式：使用项目根目录下的 `data` 作为物理路径。
- 容器模式：检测当前 backend 容器，并通过 Docker metadata 解析 `/app/data` 对应的宿主机挂载源。
- Docker 执行器只消费已经解析好的物理路径，不在执行命令时临时猜测路径。

本次 backend-go 需要做对等实现，不能只用局部字符串修补绕过当前 Windows 报错。

## Goal

1. backend-go CI 执行器在启动 Docker stage 容器时，挂载宿主机可见的物理路径，而不是配置中的相对 `dataRoot` 或容器内逻辑路径。
2. 对齐 Python backend 的分层思路：
   - CI workspace / artifacts 路径解析集中在内聚的 workspace/path 组件中。
   - executor 负责组织执行上下文，不散落路径修正逻辑。
   - Docker runner 只负责把已解析的 mount 描述转换为 Docker CLI 参数，不负责推断业务路径。
3. 支持本地运行模式：默认 `data` 应解析为项目根目录下的绝对路径，Docker 不再收到 `data\ci\...` 这类相对路径。
4. 支持容器化运行模式：当 backend-go 自身运行在容器内并通过 Docker socket 启动子容器时，应能解析 `/app/data` 或配置的数据目录在宿主机上的实际 bind source；若无法解析，应明确失败并提示数据目录必须挂载。
5. Docker CLI 参数应避免 Windows 盘符冒号和 `-v source:target:mode` 解析歧义，优先使用更稳健的 bind mount 表达。

## Non-goal

1. 不重写 CI stage 执行模型，不引入 Docker SDK 替代当前 CLI runner。
2. 不改变 pipeline snapshot、runtime variables、stage orchestration 或 artifact 业务语义。
3. 不修改前端触发流程。
4. 不为了兼容旧错误行为保留相对路径 fallback。
5. 不在多个调用点散落 `filepath.Abs`、字符串替换或 Windows 特判作为长期方案。

## User scenarios

1. **本地 Windows 开发运行 backend-go**
   - 配置仍使用默认 `data`。
   - 触发流水线后，workspace 挂载路径解析为项目根目录下的绝对路径。
   - Docker stage 容器正常启动并进入 git clone 阶段。

2. **本地 Linux/macOS 开发运行 backend-go**
   - 相对 `data` 同样解析为项目根目录下的绝对路径。
   - Docker bind mount 使用宿主机路径，不依赖当前工作目录隐式行为。

3. **backend-go 容器化运行并挂载 Docker socket**
   - backend-go 在容器内看到的是容器内 data path。
   - 启动子 Docker 容器时，使用宿主机上的 data mount source。
   - 如果无法从当前容器 metadata 解析 host source，返回明确错误，而不是传入容器内路径造成隐蔽失败。

4. **流水线 stage 日志和 artifacts**
   - 应用内部读写日志、artifacts 时仍使用 backend-go 自身可访问的数据路径。
   - Docker stage mount 使用物理路径。
   - 两类路径职责清晰，不互相污染。

## Acceptance

1. backend-go 不再把 `data\ci\...` 或 `data/ci/...` 这类相对路径传给 Docker bind mount。
2. workspace 和 artifacts 的 Docker host path 由统一组件解析，代码位置内聚，不能在 runner/executor 各处临时拼补。
3. 本地模式下，默认 `dataRoot=data` 解析为项目根目录下的绝对物理路径。
4. 容器模式下，backend-go 尝试解析当前容器中数据目录对应的宿主机 mount source，并据此生成 workspace/artifacts physical path。
5. 容器模式无法解析物理数据目录时，错误信息必须明确说明无法解析宿主机数据目录及需要正确挂载 data 目录。
6. Docker runner 使用不易受 Windows 盘符冒号影响的 bind mount 参数形式；如果继续使用 `-v`，必须有测试证明 Windows 绝对路径不会被错误切分。
7. 增加覆盖路径解析和 Docker mount 参数生成的 Go 测试；至少覆盖：
   - 相对 data root 的本地绝对化。
   - 已经是绝对 data root 的保持语义。
   - runner 生成 bind mount 参数不包含裸相对 host path。
8. 现有 CI trigger、snapshot、runtime variable 行为不因本次变更回退。

## Open questions

暂无需要用户立即决策的未决事项。本次按“对等 Python backend 能力”推进：不仅修本地相对路径，也补容器模式宿主机物理路径解析。

实现时若发现 backend-go 当前配置缺少足以定位项目根目录或容器 data mount 的信息，应优先按现有配置和运行环境推导；只有在无法可靠推导时，再把新增配置项作为风险反馈，而不是先引入散落的兼容分支。

## Decisions

1. 采用对等实现，不接受仅在 Docker runner 里 `filepath.Abs(volume.HostPath)` 的局部修补作为最终方案。
2. 路径职责分层：
   - logical data path：backend-go 自身读写日志、workspace、artifacts。
   - physical data path：传给 Docker 子容器的宿主机 bind mount source。
3. physical path 解析应靠近 CI workspace/path 领域，保持内聚；runner 不理解 repository code、run id、dataRoot 业务结构。
4. Docker CLI bind mount 参数优先使用 `--mount type=bind,source=...,target=...`，减少 Windows 路径冒号歧义。
5. 对容器模式解析失败坚持显式报错，不默默 fallback 到容器内路径。

## Risk

1. 容器模式需要通过 Docker metadata 解析当前容器 mount source；如果运行环境没有 Docker socket 权限或容器 ID 检测失败，需要明确错误并保持可诊断。
2. Windows 路径传给 Docker Desktop 时可能需要兼容 CLI 可接受的路径格式；实现必须通过测试保护参数生成，不靠人工假设。
3. 如果当前 backend-go 没有统一项目根目录工具，新增路径解析组件时需要避免扩大范围或制造全局依赖。
4. 该变更触及 CI 执行基础设施，错误实现会影响所有流水线 stage，因此需要单元测试覆盖路径解析和 runner 参数生成。

## User review notes

- 用户明确要求：“需要对等的实现，不然没法儿玩”。
- 用户补充要求：“注意不要把代码写得七零八落，注意合理分层和内聚”。本 requirement 已将分层和内聚列为目标、非目标和验收标准。
