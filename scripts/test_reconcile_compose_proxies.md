# `test_reconcile_compose_proxies.py` — Compose 代理协调单元测试

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

这是 `reconcile_compose_proxies.py` 的同目录 `unittest` 测试模块。它通过文件路径动态加载被测模块，验证 Compose 文档在内存中的协调逻辑。

覆盖重点包括：

- 映射和列表形式的 `environment` 处理；
- 保留无关环境变量与标签；
- 代理变量和 `extra_hosts` 去重；
- 重复协调的幂等性；
- 网络筛选；
- Mihomo listener 解析；
- 非法环境配置的拒绝行为。

## 用法

从仓库根目录执行：

```bash
python -m unittest scripts/test_reconcile_compose_proxies.py
```

## 安全边界

测试使用内存中的 Compose 文档，不会扫描 `/opt/pomelo-orbit/data/cd`、连接远程主机或调用 Docker。

## 相关文档

被测脚本的运行方式与写入边界请参阅 [`reconcile_compose_proxies.md`](./reconcile_compose_proxies.md)。