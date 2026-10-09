# Pomelo Orbit 工作目录与挂载
最后修改时间: 2026-10-09 17:19:54

Doc role: living guide。与代码冲突时以代码为准。

## 目录边界

| 配置 / 字段 | 内容 |
| --- | --- |
| `workspace.root` | 工作区根目录的启动配置基准；按 Project Environment 派生 `<root>/pipeline`，仅用于无项目上下文的控制面初始化与测试 |
| `Environment.workspace_root` | 当前 Project 的工作区根目录，原样保存绝对路径或 `~` / `~/...`。CI 使用 `<root>/pipeline`；CD 首次候选为 `<root>/deployment/<service-code>`，完整目录可由用户覆盖。local 在使用时把 `~` 展开为控制面进程用户主目录，SSH 展开为远端登录用户主目录。Service Compose、组件 logical mount、Gateway 证书和独立 Route 文件都在确认的 Service 目录下 |
| `logging.deployment_root` | 控制面部署执行日志根目录（`<root>/logs/<service-code>/<deployment-id>.log`），不属于 Environment workspace |

只有 `workspace.root` 属于 Orbit 进程配置，并相对 `orbit.root` 解析。Environment 的 `workspace_root` 由 Wizard / 环境页面或 MCP 原样保存，不在加载配置或返回 API 时翻译成绝对路径。`~` 与 `~/...` 只在 Probe、部署、CI runtime 使用时展开：local 用控制面 `UserHomeDir`，SSH 用远端登录用户主目录。初始化来源配置 `project_initialization.environment.local_workspace_root` 默认 `~/.pomelo-orbit`，同样原样展示和写入。

Service 目录默认候选为 `<Environment.workspace_root>/deployment/<service-code>/`，实际使用确认的完整目录，包含 `docker-compose.yml`、组件受控文件/目录，以及 Gateway 的 `gateway/dynamic/route-<code>.yaml`、`gateway/certs/route-<id>/<sha256-revision>/` 和 `.orbit/route-publication/<id>/`。动态文件及 Traefik 资源使用路由编码，证书和内部归属使用稳定 ID。发布记录和恢复材料不挂载给 Traefik。CI checkout、artifacts 和 stage logs 位于同一根下的 `pipeline/`。local 由控制面 Docker daemon 执行，SSH 由 SSH/SFTP 执行。两类目标不互相 fallback。

## 挂载语义

除 `named_volume` 外，Version mount source 必须是绝对路径或显式 `./` 相对路径。部署渲染将 `./` source 解析到 target Service 目录，并由 local filesystem 或 SFTP materialize 受控文件或目录；absolute source 保持目标宿主机路径语义。`ignore_if_exists` 只影响受控文件首次写入。

Orbit 运行在容器内时，`workspace.root`（及其 `pipeline/` 子目录）与 `logging.deployment_root` 需要满足对应的 host path 映射。local Environment 保存的 `workspace_root` 必须同时对控制面可写、对 Docker daemon 可见；Probe 验证工作区及 Docker/Compose 可用性，具体服务的目录可写和挂载映射在部署时检查。SSH Environment 不使用本地 path resolver。

## Orbit 配置覆盖文件

原生 Windows/Linux 以启动工作目录定位 `configs/`、启动 dotenv、`overwrite.env` 和 `data/config`，不是以可执行文件位置定位。Windows 服务、Linux systemd 或启动脚本必须明确工作目录；Windows 运行账号需要覆盖文件的读写权限及 `data/config` 的创建、修改权限，Linux 文件与目录归运行账号所有，覆盖文件建议 `0600`、恢复目录建议 `0700`。覆盖文件可先不创建，设置页首次保存时创建；启动读取仍需要创建锁文件的目录权限。修改启动 dotenv 或进程 ENV 后重启，设置页修改只写覆盖文件，保存与重置同样在重启后生效。

Orbit 固定读取启动工作目录下的 `overwrite.env`，当前镜像位置为 `/app/overwrite.env`。复用通用 `controlled_file`：source `./data/config/overwrite.env`，target `/app/overwrite.env`，content 空，mode `0600`，read_only=false，ignore_if_exists=true。local 与 SSH 实际 writer 在文件已存在时直接跳过，后续部署保留应用写入内容。

同时将同一持久化 data 目录普通挂载到 `/app/data`，运行账号对文件和 `/app/data/config` 可写。文件锁和写入恢复记录存在该目录；只有一个覆盖文件挂载而没有持久化恢复目录不能提供本方案的中断恢复保证。应用保存保持目标文件 inode，避免单文件 bind mount 的 rename 冲突。设置保存后重启 Orbit 读取，不需要后端尚未实现的 Compose env_file。

## Traefik 证书

Gateway 有效部署计划统一覆盖 dynamic/certs 只读 directory mount 和 acme 读写 directory mount。Route 同步通过对应 Project Environment target runtime 将 YAML、PEM 与私钥写入该 Gateway Service 目录，File provider YAML 引用容器内 `/etc/traefik/certs/route-<id>/<sha256-revision>/`。目录整体挂载供 watcher 观察替换后的文件；证书和记录使用现有文件 Mode，不扩展目录 owner/group 或 Windows ACL 管理。

Gateway deployment 按目标路径替换或补齐受管目录挂载，静态受控文件强制覆盖并重建 Gateway 容器；其他 Version mount、镜像、端口和 resolver 声明保留，不写回库存。部署不会从业务 Route 重写上述持久化文件，不清空路由、证书或 ACME 目录。

## 自定义目录与 DooD

输入目录使用执行目标可见路径：local 原生为控制面路径，local DooD 为 Orbit 容器内路径，SSH 为远端宿主机路径。文件准备使用此路径；DooD 的相对挂载源逐项经过 Docker daemon 路径映射（最长挂载前缀），包括独立嵌套挂载。显式绝对 host-path mount 与 named volume 保留原有语义。容器私有且无法映射的目录拒绝部署。

部署目录不能是文件系统根目录，包括 Windows 盘符根目录。运行挂载占用比较会统一 Windows 分隔符、盘符大小写和 Docker Desktop 报告的宿主机盘符路径；Orbit 运行在 Docker Desktop 容器中时同样处理该物理路径形式。

文件准备前，Orbit 查询同一目标上其他服务运行容器的实际 bind mount。如果本次部署目录或相对挂载的物理源已被运行服务使用，或与其源存在父子目录重叠，拒绝部署并指出占用服务；包括两个独立嵌套挂载指向同一物理目录的情况。显式共享目录按下节放行。依据实际 Docker 状态和挂载，不按服务库存状态或待部署版本推测。已停止容器不构成运行数据占用，本服务自己的挂载不阻止再次部署；服务根目录的既有归属保护继续有效。

确认目录与实际运行目录分别持久化并绑定目标修订。运行目录在准备成功、发出 Compose 命令前切换，后续停止、重启、状态查询及 MCP 的 RuntimeComposeLogs 快照读取沿用它。文件读取与发布限制在明确的 Service 范围，即使目录位于 Environment 根之外也不能越界。

Web 容器日志流在 Environment 工作区根目录按 Service code 的 Compose project 与可选 Component 读取，不要求 Service 已保存运行目录或存在 Compose 文件。容器日志可见不代表部署准备已通过；部署仍独立检查目录归属、写入权限和实际运行挂载占用。

## 显式共享目录

共享工作区在 Version 的 directory mount 上启用 `shared`，源必须是绝对目录，可按现有方式选择 `source_is_host_path`。相对源、file、controlled_file 和 named_volume 不能声明共享。Service 源覆盖也必须保持绝对目录；不能改变共享属性。

部署后容器带有 `io.pomelo-orbit.shared-mounts` 标签，记录共享 target。其他服务在实际共享 bind source 内部署或重启时可准备文件；准备范围包含该共享源的父目录时仍拒绝。嵌套未共享数据挂载仍拒绝冲突。Service 根目录独占，所以 Orbit 要部署到与其他服务平级的普通目录，不能保留占据整棵工作区的部署根目录。

修改共享声明后必须重新部署才能改变运行占用判断。不要把所有绝对挂载标记共享，只为允许其他服务写入的工作区启用；数据、配置和 socket 挂载按各自用途保留保护。Orbit 的迁移顺序见 [普通应用迁移指南](./orbit-self-migration.md)。
