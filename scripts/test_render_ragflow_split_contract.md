# RAGFlow 部署契约渲染测试

最后修改时间: 2026-08-03 12:29:10

`test_render_ragflow_split_contract.py` 覆盖 split 与 integrated 部署契约的生成物一致性、运行时变量分配约束和通用渲染器检查入口。

## 用法

```powershell
python -m unittest scripts/test_render_ragflow_split_contract.py
```

测试只在临时目录中制造生成物漂移，不会执行 Docker Compose、MCP 或 Orbit 生命周期操作。
