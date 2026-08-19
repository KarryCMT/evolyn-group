-- 回滚：恢复 tenants.tenant_id 列（默认租户归属）与 Owner 0 哨兵语义
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS tenant_id BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS fk_tenants_owner;
ALTER TABLE tenants ALTER COLUMN owner_account_id SET DEFAULT 0;
ALTER TABLE tenants ALTER COLUMN owner_account_id SET NOT NULL;
UPDATE tenants SET owner_account_id = 0 WHERE owner_account_id IS NULL;
