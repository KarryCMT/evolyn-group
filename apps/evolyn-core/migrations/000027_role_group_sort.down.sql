DROP INDEX IF EXISTS idx_role_groups_tenant_sort;
ALTER TABLE role_groups DROP COLUMN IF EXISTS sort;
