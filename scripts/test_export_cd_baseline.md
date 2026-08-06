# `test_export_cd_baseline.py`

`python scripts/test_export_cd_baseline.py` 运行导出器的隔离 round-trip 测试。测试在临时 SQLite 数据库中验证 Gateway、Route、Service，以及 Version Component、Service 和 Service Component 三层环境变量被原样导出，`deployment` 不会出现在输出中。
