# RAGFlow Split 部署契约兼容入口

最后修改时间: 2026-08-03 12:15:00

`render_ragflow_split_contract.py` 保留为只检查或生成 split 契约的兼容入口。新的开发和部署流程应使用 `render_ragflow_deployment_contract.py`，它会同时处理 split 与 integrated 契约。

## 用法

先检查提交文件是否和契约一致：

```powershell
python scripts/render_ragflow_split_contract.py --check
```

在契约变更后重写全部生成物：

```powershell
python scripts/render_ragflow_split_contract.py --write
```

## 安全边界

- 该脚本只读写仓库中的 Compose 参考文件和 split primitive；不执行 Docker Compose、MCP 或 Orbit 生命周期操作。
- 运行时变量只保留键名和适用 Service，脚本不读取或输出其值。
- 不要直接编辑生成的 Compose 或 primitive 文件；修改契约后重新渲染，并以 `--check` 检测漂移。
