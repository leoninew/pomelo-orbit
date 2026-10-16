# 开发打包脚本

`scripts/release.py` 使用 Python 标准库接续现有发布构建，不需要 uv、虚拟环境或第三方 Python 包。

```powershell
python scripts/release.py --help
python scripts/release.py
```

需要 Python ≥3.12、Go 和 Yarn，以及已安装的前端依赖。脚本通过自身位置定位仓库，读取 `VERSION`，构建前端和 Windows/Linux/macOS amd64 二进制。全部构建成功后，将 ZIP 写入 `dist/`，展开目录写入 `dist/package/`。脚本只生成本地产物，不上传或发布。

`package/` 保存启动脚本和 release 环境模板。打包保留 `configs/config.yaml`、`.env.example`、`.env.release`、`VERSION` 和前端静态资源；不会收集用户 dotenv、密钥或数据库。启动脚本以包目录为 cwd，选择 release profile；`stop/status/restart` 只匹配服务进程，端口转发由 CLI 自行管理。

环境管理使用同一 Go 二进制的 `pomelo-orbit cli env` 命令组，配置和业务与 WebAPI 共用。命令、开发启动和端口转发用法见[项目环境 CLI 指南](../docs/guides/project-environment-cli.md)。
