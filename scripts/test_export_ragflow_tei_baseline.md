# `test_export_ragflow_tei_baseline.py` — RAGFlow TEI 基线导出测试

验证 `export_ragflow_tei_baseline.py` 的 SQLite 往返行为，不读取开发数据库、不写入 Orbit。

```powershell
python -m unittest scripts/test_export_ragflow_tei_baseline.py
```

测试用临时 SQLite fixture 导出 SQL，再导入相同空 schema，确认组件子表和 ServiceComponent 映射能稳定导出，设备请求仍存在，Service 均为 `stopped`，且不导出运行时环境覆盖。
