# 历史 Service `deploying` 状态清理

`reconcile_legacy_service_deploying.py` 是离线脚本，不由 Pomelo Orbit 服务调用。

准备人工审核的 JSON 映射，键是 `service.id`，值只能是 `running`、`stopped` 或 `faulted`：

```json
{
  "01LEGACYSERVICE000000000001": "stopped"
}
```

先执行 dry-run：

```powershell
python scripts/reconcile_legacy_service_deploying.py --database data/db/pomelo-orbit.db --mapping reviewed-service-status.json
```

审核输出后，以不同路径创建备份并应用。脚本拒绝漏映射、未知 Service、源状态不是 `deploying` 的映射和已有备份文件：

```powershell
python scripts/reconcile_legacy_service_deploying.py --database data/db/pomelo-orbit.db --mapping reviewed-service-status.json --apply --backup data/db/pomelo-orbit-before-service-status.db
```
