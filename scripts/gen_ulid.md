# `gen_ulid.py` — 生成 ULID

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

生成一个或多个 ULID，并逐行输出到标准输出。脚本不写入仓库文件。

## 前置条件

Python 环境需要安装 `ulid` 包。

## 用法

```bash
# 生成一个 ULID
python scripts/gen_ulid.py

# 生成 10 个 ULID
python scripts/gen_ulid.py -n 10
```

`-n` 指定生成数量，默认值为 `1`。

## 安全边界

该脚本仅向标准输出写入生成结果；请自行决定是否以及如何保存这些标识符。
