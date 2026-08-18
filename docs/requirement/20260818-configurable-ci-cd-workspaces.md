# 可配置 CI/CD 工作目录与 DooD 宿主路径解析
最后修改时间: 2026-08-18 22:53:53

Review status: Accepted

Mode: strict

## Background

当前 Pipeline 和 Deployment workspace 均从 `Config.DataRoot()` 派生，并分别在实现中追加固定的 `pipeline`、`deployment` 子目录。部署出的 Compose bind source 与 CI Stage 的 `docker run --mount` source 必须是 Docker daemon 可见的宿主机绝对路径；Orbit 运行在容器中时，直接使用 Orbit 容器内路径会触发 Docker-outside-of-Docker（DooD）路径错误。

现有 Docker daemon path resolver 可在 Orbit 容器运行时，通过当前容器的 Docker inspect mount 表将进程可见路径反查为宿主机路径；原生运行时路径无需转换。初始 Gateway Component 的证书目录仍直接使用 `orbit.root + traefik.cert_dir` 作为 host-path mount source，在 Orbit 容器运行时也可能把容器内路径错误地交给 Docker daemon。

## Directory inventory

| 目录或路径 | 当前责任 | 本需求处置 |
| --- | --- | --- |
| `data/pipeline` | 后端创建项目 checkout workspace、Run artifacts、Stage logs，并作为 CI 容器 bind source | 收敛为 `workspace.pipeline` 的内部层级；不再固定追加 `data/pipeline`。 |
| `data/deployment` | 后端写 Compose、部署日志、logical mount 物化目录；Docker Compose 与应用容器使用其宿主路径 | 收敛为 `workspace.deployment` 的内部层级；包含 Service、Gateway 静态文件、ACME 和 Route PEM。 |
| `data/deployment/<service-code>/tei/...` | 受管 Service 的模型缓存和其他 Component 数据 | 是 Deployment workspace 的子目录，不新增 `workspace.model`。 |
| `data/backup` | `prepare_ragflow_tei.py` 的运维归档默认输出；不由后端或 Docker worker 读取/清理 | 不纳入后端 `workspace`；移除固定默认输出，要求运维脚本调用显式提供 `--archive`。 |
| `data/exports` | 当前没有后端运行时的创建或消费者；现有内容为人工/辅助脚本产物 | 不纳入配置，也不作为应用管理目录。 |
| `scripts/cert.py` 的证书输出 | 本地运维脚本直接写固定 Gateway 证书目录 | 不新增 workspace；`new` / `check` 均要求显式 `--cert-dir`，通常传入 `<workspace.deployment>/traefik/data/certs`。 |
| `scripts/manage.py` 的远程备份 excludes | 辅助脚本以固定 `data/pipeline`、`data/deployment` pattern 排除源码 workspace | 不读取后端配置；改为由备份调用显式传入需排除的 workspace 路径，不能保留过时固定 pattern。 |
| `data/db` | SQLite database，已由 `database.sqlite.path` 独立配置 | 保持独立数据库配置，不纳入 `workspace`。 |
| `logs/` | 后端进程日志，已由 `logging.file` 独立配置 | 保持独立日志配置，不纳入 `workspace`。 |
| `.env`、用户 MCP token 目录、本地 Repository | 配置/凭据或用户外部源码，生命周期不归 CI/CD workspace 所有 | 不纳入 `workspace`；本地 Repository 继续只读解析为 Docker host path。 |

## Goal

1. 提供 `workspace.pipeline` 与 `workspace.deployment` typed config，分别作为 CI Pipeline 与 CD Deployment 的工作目录根，替代对 `data/{pipeline,deployment}` 的固定拼接。
2. 保持默认配置的现有物理位置：`data/pipeline` 与 `data/deployment`；运行时代码不保留缺省回退或旧路径并存。
3. 统一 workspace 路径语义：相对路径相对 `orbit.root` 解析；绝对路径直接使用。Pipeline 与 Deployment 根目录必须非空、规范化且互不重叠。
4. 对所有 Docker bind source 使用同一 DooD 解析：裸机运行时使用绝对本地路径；Orbit 容器运行时从其 Docker mount 映射得出 Docker daemon 可见的宿主路径。配置路径未被显式挂入 Orbit 容器时，启动应以包含配置键名的错误失败。
5. Gateway 初始 Version 的证书/ACME host-path mount 与 Route 证书写入路径跟随 `workspace.deployment`，不再残留 `data/deployment` 的硬编码或容器路径误用。
6. 清除后端运行时、运维脚本、前端展示、active skill contracts 和活文档中将 Pipeline、Deployment、Gateway cert 或模型归档默认绑定到固定 `data/` 子目录的行为；非 runtime-owned 输出改为显式调用参数，而不是新增无责任边界的 workspace 配置。

## Non-goal

- 不迁移、复制、删除或自动发现既有 Pipeline workspace、Deployment workspace 或证书文件；目录数据处置由运维方显式完成。
- 不改变 Version/Service Component 挂载模型、Service code 目录层级、Pipeline Stage 的 `/workspace`、`/artifacts`、`/source` 容器目标，或本地目录 Repository 的只读源码挂载语义。
- 不改变 Docker socket 暴露、Pipeline Stage 执行模型、Compose project 命名、部署生命周期或数据库 Schema。
- 不提供旧配置字段、旧目录或未挂载路径的运行时兼容/回退。
- 不把 SQLite 数据库、后端日志、环境文件、MCP 本地凭据、用户自选的导出文件或外部 Repository 归入 CI/CD workspace。

## User scenarios

1. 裸机部署：运维将 `workspace.pipeline` 配为 `/srv/orbit/ci`、`workspace.deployment` 配为 `/srv/orbit/cd`。CI 与 Compose 均以这些绝对目录作为 Docker bind source。
2. DooD 容器部署：Docker Compose 将宿主 `/srv/orbit/ci`、`/srv/orbit/cd` 分别挂入 Orbit 容器的 `/app/data/pipeline`、`/app/data/deployment`，与默认 YAML 配置一致。Orbit 解析后向 Docker daemon 传递 `/srv/orbit/...`，而非容器内路径。
3. 配置错误：Orbit 容器将 workspace root 指向没有宿主挂载的 `/tmp/orbit-ci`。服务启动失败，诊断明确指出对应的 workspace 配置必须映射到宿主机。
4. Gateway：部署根更换后，Gateway 初始证书/ACME 数据和 Route PEM 写入 `<workspace.deployment>/traefik/data/certs`，并以解析后的宿主路径挂载进 Traefik。
5. 运维归档：管理员使用 RAGFlow 工具归档模型时显式传入 `--archive /srv/orbit-backups/bge-m3.tar.gz`；工具不再隐式写入仓库 `data/backup`。
6. 本地证书：管理员执行 `python scripts/cert.py new -n app.localhost --cert-dir /srv/orbit/cd/traefik/data/certs`；脚本不会假定仓库 `data/deployment`。
7. 远程备份：管理员执行辅助备份时传入已配置的 CI/CD workspace 排除路径；脚本不会将旧 `data/pipeline` 或 `data/deployment` 当作唯一布局。

## Acceptance

- [ ] 默认 `configs/config.yaml` 明确声明 `workspace.pipeline: data/pipeline` 与 `workspace.deployment: data/deployment`；与其他 typed config 一致，`configs/config.<env>.yaml`、`.env` / `.env.<env>` 和 OS 环境变量可按既有优先级覆盖这两个值，Settings 不提供 workspace 入口。
- [ ] Config load 集中完成路径 trim、相对路径解析、非空与 Pipeline/Deployment 根目录重叠校验；HTTP 与 worker 均只消费已解析的 typed config。
- [ ] Pipeline workspace 不再在实现中固定追加 `pipeline`；其 workspace、run artifacts、stage logs 仍保持各自现有的隔离结构。
- [ ] Deployment workspace 不再在实现中固定追加 `deployment`；Service code 仍是 Deployment workspace 下的唯一 Service 根目录。
- [ ] CI Stage 的 `/workspace`、`/artifacts`，以及本地目录 Repository 的只读 `/source` 挂载均传入 Docker daemon 可见的绝对宿主路径。
- [ ] Deployment preview 与实际 Compose 渲染对相对 logical mount source 产生同一 Docker daemon 可见的绝对宿主路径；directory 与 controlled_file 的物化仍发生在 Orbit 进程可见的 logical workspace。
- [ ] Docker 环境下启动会验证两个 workspace root 可反查到宿主 mount；裸机运行不要求 Docker inspect。
- [ ] Gateway 初始证书/ACME mount 和 Route PEM 写入使用配置化 Deployment workspace，且 Docker Compose 的 source 为解析后的宿主路径。
- [ ] RAGFlow 模型归档工具不再把 `data/backup` 作为隐式默认目录；归档输出路径必须由调用者显式提供，相关指南同步调整。
- [ ] `scripts/cert.py` 不再固定 Gateway cert directory，`new` 和 `check` 使用同一个必填 `--cert-dir`；`scripts/manage.py` 不再内嵌旧 CI/CD workspace exclude pattern。
- [ ] 前端删除目录提示、RAGFlow skills/contracts、辅助脚本和活文档使用 workspace 概念或显式路径参数，不再向用户声明 `data/pipeline`、`data/deployment` 是固定位置。
- [ ] `data/exports` 没有运行时代码归属，不被误加入 workspace；SQLite、日志、环境文件、MCP 凭据和用户本地 Repository 仍由其现有独立边界管理。
- [ ] 覆盖 Config 加载/校验、裸机与 DooD 路径解析、Pipeline mounts、Deployment logical mount 物化/Compose、Gateway cert mount 与 Route cert directory 的定向测试。
- [ ] 活文档更新为新的配置契约；不再将 `data/{pipeline,deployment}` 描述为不可变路径。

## Decisions

- 工作根归入 `workspace` 配置组，`workspace.pipeline` 与 `workspace.deployment` 分别表达 Orbit 管理的 CI/CD 持久工作区，而非任意 Docker volume 或用户 Component 的 `source`。
- `workspace` 遵循项目既有 typed config 的 YAML、`.env` 与 OS 环境变量覆盖语义；不为该配置组另设加载规则或运行时入口。Settings 不是通用配置加载来源，因此不增加 workspace Settings 项。
- 相对配置值相对 `orbit.root` 解析一次；绝对值可位于项目外。配置值表达 **Orbit 进程可见路径**，不是要求用户填写宿主机路径。仅 DooD 下才由 Docker daemon path resolver 转换。
- Pipeline 与 Deployment 根目录拒绝相同或祖先/子孙重叠，避免 CI 清理、Deployment 配置和证书写入互相覆盖。
- Gateway 证书目录固定派生为 `<workspace.deployment>/traefik/data/certs`，消除 `traefik.cert_dir` 与 Deployment 根目录的双重路径 SoT；保存到 Version 后仍遵循既有 Version 规格不被部署流程覆盖的规则。
- `workspace` 只表达后端拥有、会写入且会作为 Docker source 使用的 runtime workspace。因此不增加 `workspace.model`、`workspace.database`、`workspace.log` 或 `workspace.export`；模型缓存是 Deployment 子目录，数据库/日志已有独立配置，exports 没有 runtime owner。
- RAGFlow 模型归档是显式运维输出，不是 backend workspace。为消除固定目录，改为调用者必填 `--archive`，而不是引入后端不消费的 `workspace.backup`。
- 本地证书生成与远程备份辅助脚本不是后端配置消费者：它们使用必填命令参数与已配置 workspace 对接，避免各自实现第二套 YAML/环境变量解析逻辑。

## Open questions

暂无需要用户确认的功能性未决事项。上述 `workspace.pipeline` / `workspace.deployment` 命名、目录责任划分、相对路径基准和 Gateway 证书目录派生是基于当前实现与 DooD 约束提出的具体方案，等待本需求评审确认。

## Risk

- 这是持久工作目录的配置边界变更。将现有实例改到新根目录不会移动历史数据，重启或重新部署前必须由运维方完成数据迁移与宿主目录挂载。
- 在 DooD 部署中，Orbit 容器必须同时获得 Docker socket 和两个 workspace root 的 bind mount；少任一项均应 fail fast，不能把容器内路径传给 Docker daemon。
- 移除 `traefik.cert_dir` 会是配置契约的硬切换；与仓库的无兼容层约束一致，但发布说明必须明确要求改用 `workspace.deployment`。
