# `test_prepare_ragflow_tei.py` — TEI 准备脚本测试

验证 `prepare_ragflow_tei.py` 的本地归档安全行为，不下载模型、不拉取镜像，也不访问 Orbit。

```powershell
python -m unittest scripts/test_prepare_ragflow_tei.py
```

测试使用临时小型模型目录，覆盖 `tar.gz` 备份/恢复、损坏 manifest 拒绝、非空恢复目标拒绝，以及从已验证缓存 staging 后的 manifest 校验。
