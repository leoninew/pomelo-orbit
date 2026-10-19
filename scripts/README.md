# 开发打包脚本

`scripts/release.py` 使用 Python 标准库接续现有发布构建，不需要 uv、虚拟环境或第三方 Python 包。

```powershell
python scripts/release.py --help
python scripts/release.py
```

需要 Python ≥3.12、Go 和 Yarn，以及已安装的前端依赖。脚本通过自身位置定位仓库，读取 `VERSION`，构建前端和 Windows/Linux/macOS amd64 二进制。全部构建成功后，将 ZIP 写入 `dist/`，展开目录写入 `dist/package/`。脚本只生成本地产物，不上传或发布。

`package/` 保存启动脚本和 release 环境模板。打包保留 `configs/config.yaml`、`.env.example`、`.env.release`、`VERSION` 和前端静态资源；不会收集用户 dotenv、密钥或数据库。启动脚本以包目录为 cwd，选择 release profile；`stop/status/restart` 只匹配服务进程，端口转发由 CLI 自行管理。

环境管理使用同一 Go 二进制的 `pomelo-orbit cli env` 命令组，配置和业务与 WebAPI 共用。命令、开发启动和端口转发用法见[项目环境 CLI 指南](../docs/guides/project-environment-cli.md)。

`migrate_workspace.sh` 从仓库根目录启动迁移工具，默认 `POMELO_ORBIT_APP__ENV=development`，复用应用配置及已有数据库、SSH 信息：

```sh
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit'                 # 只评估
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit' --no-dry-run    # 执行或自动继续
```

自动选择当前工作区对应的 Environment，目标目录必须通过 `--target` 显式传入，没有默认迁移目录。支持本机与 SSH、Cygwin 和 Git Bash；仅对 Windows 目标转换 shell 路径。

计划、进度和备份保存在仓库 `data/workspace-migration/<批次>/`，中断后重复执行即可继续，无需指定计划文件、token 或启停命令。

控制机需要 Python ≥3.12、Go 和 dbtalk；目标宿主机需要 Python ≥3.12 和 Docker Compose。执行前自行停止 Orbit API 和 worker；脚本冷复制并校验数据、事务更新业务路径并重建已有运行组件，完成后提示自行启动 Orbit。迁移保留原有 Probe 记录，不发起 Probe 或等待应用健康检查。日志使用简单文本。范围、文件职责与恢复说明见[工作区迁移指南](../docs/guides/workspace-migration.md)。
