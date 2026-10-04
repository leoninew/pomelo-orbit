# 服务容器日志与运行目录记录解耦
最后修改时间: 2026-10-04 18:15:15

Flow mode: light
Review status: Accepted

## Background / 背景

用户发现服务详情的组件日志抽屉一直显示“等待日志输出”。服务在 Orbit 中为 `stopped`，但可能已在目标环境手动启动；后续提供了 `k12-delivery-os-default` 的实际场景和截图。

使用用户授权的 Pomelo PW 登录当前开发环境后，确认日志请求返回 HTTP 200，但事件只有 `ready` 和 `waiting`，没有携带日志内容的 `chunk`。查询已能发现目标容器，读取阶段却依赖 `runtime_directory` 和该目录中的 `docker-compose.yml`。

该服务是新建服务，没有 Orbit Deployment 记录，`deployment_directory`、`runtime_directory` 均为空，目录修订均为 0。用户提供的真实目录为 `D:\SourceCodes\pomelo-orbit\data\deployment\k12-delivery-os-default`；环境根目录与服务编码计算出的默认路径与其一致。问题是库存运行目录记录未绑定，不是主机目录不存在或默认路径计算错误。目标上的手动启动不会自动写入这些字段。

## Goal / 目标

1. 服务组件日志以目标 Docker 上匹配 Service code 与 Component 的实际容器为来源，允许持久化状态仍为 `stopped` 的服务读取日志。
2. 读取不要求 Service 已保存运行目录，不要求目标上存在 Orbit 生成的 Compose 文件。
3. 保留现有授权、目标与凭据修订复核、容器重建跟随、游标续读和订阅取消行为。
4. 通过实际浏览器流程验证日志上屏与关闭重开，并同步当前产品和运行时文档。

## Non-goal / 非目标

- 不自动探测或写入 Service 的确认目录、运行目录和持久化状态，不迁移目标数据。
- 不改变部署、重启、停止、状态查询及其他运行时工具对运行目录的要求。
- 不修改前端日志组件、SSE 契约、API 路由、数据库结构或迁移文件。
- 不新增轮询兜底、跨用户共享日志订阅或依赖；不主动管理开发服务器。
- 不开展性能压测或承诺固定耗时、CPU 降幅。

## User scenarios / 用户场景

- 新服务尚未通过 Orbit 部署，用户在目标环境启动同一 Compose project，随后打开服务详情的组件日志。
- 已部署服务的库存状态为 `stopped`，目标上的同一服务被手动启动，用户仍可查看实际输出。
- 目标容器尚未出现时打开日志抽屉，订阅保持等待；容器出现或重建后继续读取。
- 用户关闭并重新打开抽屉，已有内容保留并使用游标继续读取。

## Acceptance / 验收标准

1. 对于 `stopped` 且运行目录为空的服务，只要目标有匹配容器，就能收到非空 `chunk` 和续读游标。
2. 流式命令使用显式 Compose project 与可选 Component，在 Environment 根目录执行，不加载 Compose 配置或检查服务目录中的文件。
3. local 与 SSH adapter 均支持无 Compose 文件的读取；SSH 读取不建立用于配置检查的 SFTP 会话。
4. 容器尚未出现仍等待；部署完成不结束容器读取；容器重建后恢复来源；成员撤销或目标修订变化停止订阅，取消后释放读取命令。
5. Pomelo PW 在实际服务上验证首次打开有日志上屏，关闭重开后仍有内容并携带续读游标。
6. 固定质量检查与相关 Go 回归通过，用户人工测试确认无问题。

## Open questions / 待定问题

暂无需要用户确认的未决事项。用户已确认测试没有问题，并要求以 light 模式记录问题及完成验证。

## Decisions / 决策

- 保留按 `com.docker.compose.project=<service-code>` 和可选 `com.docker.compose.service=<component>` 查询容器的现有规则。
- 使用 `docker compose -p <service-code> logs --follow --timestamps --no-color`，省去 `-f docker-compose.yml`；显式 project name 使 Compose 从现有容器读取日志，无须加载配置。
- 将日志专用 Runtime 流式端口收敛为 `StreamAtEnvironmentRoot`，local/SSH 按明确 target type 执行。
- 保持两秒复核、首次 `--tail 200`、游标时间重叠补读、SSE 与前端缓存边界。
- 本次在修复和浏览器验证完成后补记过程文档；依据用户明确要求完成验证，将 Intent 与 Verification 均标记为 `Accepted`。

## Risks / assumptions / 风险与假设

- 手动启动的容器须属于相同 Service code 的 Compose project，并具有匹配的 Component 标签；本次不识别任意无标签容器。
- Environment 仍须通过当前修订的 Probe，执行用户须具备目标 Docker 的日志读取权限，Docker 须保留对应输出。
- 持续读取的主要成本仍随订阅数与日志量增长；本次不提高复核频率。省去文件检查的收益为代码级判断，未做性能压测。

## User review notes / 用户审查记录

- 用户指定问题入口为服务详情的组件日志，表现为持续等待，无读取错误提示。
- 用户要求使用 `pomelo-pw:pomelo-pw` 自行登录检查，已执行真实浏览器复现与修复后验证；凭证不写入仓库文档。
- 用户提供目标主机真实目录，核对结果为默认路径一致、Service 目录记录为空。
- 用户最终确认“已经测试没有问题”，要求“light 记录问题和完成验证”。
