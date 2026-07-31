# `version-calc.py` — 从 Git 历史计算版本

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

从最早提交到最新提交遍历 Git 历史，计算 `0.y.z` 版本：提交主题（忽略前导空白、忽略大小写）以 `feat` 开头时递增 minor 并将 patch 归零；其他提交递增 patch。

## 用法

```bash
# 计算并输出逐提交历史和最终版本（默认只读）
python scripts/version-calc.py

# 只输出最终版本
python scripts/version-calc.py --quiet

# 计算并写入版本文件
python scripts/version-calc.py --apply
```

`--quiet` 可与 `--apply` 组合。

## 安全边界

默认模式仅读取 Git 历史。`--apply` 会修改：

- `internal/shared/common/utils/version/version.go` 的 `Version`；
- `web/package.json` 的 `version`。

执行 `--apply` 需要可用的 Git 历史；完成后必须检查 diff。