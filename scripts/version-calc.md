# `version-calc.py` - 从 Git 历史派生版本
最后修改时间: 2026-08-17 09:35:00

## 用途

按提交时间从早到晚计算 `0.y.z`：提交主题以 `feat` 开头时递增 minor 并将 patch 归零，其他提交递增 patch。

默认只读；`--apply` 同步写入受版本控制的 `VERSION`、`configs/config.yaml` 的 `app.version`、`.env.example` 的版本示例和 `web/package.json` 的 `version`。发布任务仅读取 `VERSION`，不会在打包时修改工作树，也不使用 Go 编译期 ldflags 注入版本。

## 用法

```bash
python scripts/version-calc.py
python scripts/version-calc.py --quiet
python scripts/version-calc.py --quiet --apply
```

`task version` 计算并显示版本；在准备发布版本时执行 `task version:apply`，检查这三个文件的变更后提交。

## 边界

- 必须在完整 Git 历史的工作树中运行；浅克隆会产生不完整版本。
- 提交主题是版本计算输入，应在合并前确认其符合预期。
- 版本文件是发布制品命名依据，不代表数据库 schema migration 版本。
