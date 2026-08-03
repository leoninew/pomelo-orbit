# RAGFlow 部署契约渲染器

最后修改时间: 2026-08-03 12:29:10

`render_ragflow_deployment_contract.py` 从 `scripts/ragflow-split/deployment-contract.json` 和 `scripts/ragflow-integrated/deployment-contract.json` 生成两种拓扑的全部 Compose 参考文件与 deployment primitive。每个契约是其拓扑镜像、网络别名、组件环境、端点、CPU/GPU 差异、模型缓存路径和手工 Provider 初始化项的唯一手工维护来源。

## 用法

先检查两种拓扑的提交文件是否和契约一致：

```powershell
python scripts/render_ragflow_deployment_contract.py --check
```

在任一契约变更后重写全部生成物：

```powershell
python scripts/render_ragflow_deployment_contract.py --write
```

兼容入口 `render_ragflow_split_contract.py` 仅处理 split 契约。

## 安全边界

- 脚本只读写仓库中的 Compose 参考文件和 deployment primitive；不执行 Docker Compose、MCP 或 Orbit 生命周期操作。
- 运行时变量只保留键名和适用 Service，脚本不读取或输出其值。
- 不要直接编辑生成的 Compose 或 primitive 文件；修改对应契约后重新渲染，并以 `--check` 检测漂移。
