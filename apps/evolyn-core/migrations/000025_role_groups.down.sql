ALTER TABLE roles DROP CONSTRAINT IF EXISTS fk_roles_role_group;
DROP INDEX IF EXISTS idx_roles_role_group_id;
DROP INDEX IF EXISTS uk_role_groups_tenant_name;
ALTER TABLE roles DROP COLUMN IF EXISTS role_group_id;
DROP TABLE IF EXISTS role_groups;
