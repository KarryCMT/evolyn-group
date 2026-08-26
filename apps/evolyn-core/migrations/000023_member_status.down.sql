-- 000023 down: 回滚成员租户内生命周期字段及查询索引。
DROP INDEX IF EXISTS idx_users_tenant_status_id;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS chk_users_status,
    DROP COLUMN IF EXISTS resigned_at,
    DROP COLUMN IF EXISTS status;
