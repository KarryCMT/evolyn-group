DROP INDEX IF EXISTS idx_roles_role_group_sort;
ALTER TABLE roles DROP COLUMN IF EXISTS sort;
