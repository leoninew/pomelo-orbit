# Plugin 与 Skill 发布

Pomelo Orbit 将仓库根目录作为 `pomelo-orbit` native plugin package，根目录
`skills/` 是该 plugin 的唯一技能发现来源。Claude、Codex 和 Grok 的安装状态
由各自 CLI 管理；仓库不镜像客户端的 `skills/`、cache、credentials 或主配置。
仓库没有独立于该 plugin 的 standalone skill，因此不执行第二条 skill 发布路径。
旧入口不提供迁移或兼容逻辑。

## 预检与发布

在仓库根目录执行：

```bash
python scripts/release.py plugin check
python scripts/release.py plugin list
python scripts/release.py plugin apply --dry-run
python scripts/release.py plugin apply
```

自动模式只处理已安装的客户端 CLI；缺失客户端会报告 `skipped`。显式指定
`--claude`、`--codex` 或 `--grok`，或使用 `--strict` 时，缺少必需 CLI 会在
任何 marketplace、安装或目录写入前失败。`check`、`list` 和 `--dry-run` 不写入
本机状态。

删除只作用于本 plugin 的精确 selector：

```bash
python scripts/release.py plugin remove --dry-run
python scripts/release.py plugin remove --claude
```

删除 marketplace 或其他 plugin 不属于本入口职责。

## Vendored 发布引擎

[`scripts/release.py`](../../scripts/release.py) 和
[`scripts/test_release.py`](../../scripts/test_release.py) 是规范发布引擎的项目副本。
`release.py` 顶部的项目配置区只声明本仓库的 package、marketplace 和资产；
其余同步逻辑必须与规范源整体一致，不新增外部 Python 依赖、配置模块或客户端
适配器。

升级规范源时，整体替换这两个副本，重新填写顶部项目配置并运行测试。

## Task 入口

本仓库的 plugin 发布不需要业务 Python 包安装步骤，`task skill` 固定先预检、
再应用：

```text
task skill
```

该任务只负责 plugin 的预检和原生 CLI 发布；应用三平台压缩包仍由 `task package`
负责，两者不会互相触发。

成功发布后重新开始对应 agent session，使新的 skill inventory 生效。
