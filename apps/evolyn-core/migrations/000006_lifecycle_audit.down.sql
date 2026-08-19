-- 回滚：移除审计表与租户生命周期时间线列
DROP TABLE IF EXISTS audit_logs;
ALTER TABLE tenants DROP COLUMN IF EXISTS purged_at;
ALTER TABLE tenants DROP COLUMN IF EXISTS retention_until;
ALTER TABLE tenants DROP COLUMN IF EXISTS delete_requested_at;
