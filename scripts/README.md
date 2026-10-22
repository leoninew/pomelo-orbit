# 脚本

## 发布包

需要 Python ≥3.12、Go、Node.js 22、Corepack，以及已安装的前端依赖。

```sh
python scripts/release.py
```

读取 `VERSION`，构建 Windows、Linux、macOS amd64 包到 `dist/`。`package/` 保存运行入口与环境模板。

## 工作区迁移

```sh
sh scripts/migrate_workspace.sh apply --target 'D:/Documents/orbit'
```

默认只评估；添加 `--no-dry-run` 执行。需要 Go、dbtalk、Python ≥3.12，以及目标环境的 Docker Compose。实际迁移前停止 Orbit API 和 worker，完成后自行启动；进度保存在 `data/workspace-migration/`，可重复执行继续。

远程环境操作使用 `task cli -- env --help`。
