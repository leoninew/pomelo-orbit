-- v0.4.6: 删除凭据表
-- 凭据功能已移除，webhook 签名验证改为基于系统配置（webhook.secret）

DROP TABLE IF EXISTS credential;
