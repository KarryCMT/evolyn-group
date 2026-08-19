-- 回滚 000008：清空全部表注释与字段注释（COMMENT ON ... IS NULL）。
-- 与 up 一一对应，只清元数据不动结构。

COMMENT ON TABLE accounts IS NULL;
COMMENT ON COLUMN accounts.id IS NULL;
COMMENT ON COLUMN accounts.name IS NULL;
COMMENT ON COLUMN accounts.nickname IS NULL;
COMMENT ON COLUMN accounts.phone IS NULL;
COMMENT ON COLUMN accounts.email IS NULL;
COMMENT ON COLUMN accounts.password IS NULL;
COMMENT ON COLUMN accounts.avatar IS NULL;
COMMENT ON COLUMN accounts.created_at IS NULL;
COMMENT ON COLUMN accounts.updated_at IS NULL;
COMMENT ON COLUMN accounts.deleted_at IS NULL;

COMMENT ON TABLE tenants IS NULL;
COMMENT ON COLUMN tenants.id IS NULL;
COMMENT ON COLUMN tenants.code IS NULL;
COMMENT ON COLUMN tenants.name IS NULL;
COMMENT ON COLUMN tenants.plan IS NULL;
COMMENT ON COLUMN tenants.status IS NULL;
COMMENT ON COLUMN tenants.owner_account_id IS NULL;
COMMENT ON COLUMN tenants.config IS NULL;
COMMENT ON COLUMN tenants.quotas IS NULL;
COMMENT ON COLUMN tenants.delete_requested_at IS NULL;
COMMENT ON COLUMN tenants.retention_until IS NULL;
COMMENT ON COLUMN tenants.purged_at IS NULL;
COMMENT ON COLUMN tenants.created_at IS NULL;
COMMENT ON COLUMN tenants.updated_at IS NULL;
COMMENT ON COLUMN tenants.deleted_at IS NULL;

COMMENT ON TABLE auth_infos IS NULL;
COMMENT ON COLUMN auth_infos.id IS NULL;
COMMENT ON COLUMN auth_infos.account_id IS NULL;
COMMENT ON COLUMN auth_infos.url IS NULL;
COMMENT ON COLUMN auth_infos.auth_type IS NULL;
COMMENT ON COLUMN auth_infos.auth_id IS NULL;
COMMENT ON COLUMN auth_infos.access_token IS NULL;
COMMENT ON COLUMN auth_infos.refresh_token IS NULL;
COMMENT ON COLUMN auth_infos.expiry IS NULL;
COMMENT ON COLUMN auth_infos.created_at IS NULL;
COMMENT ON COLUMN auth_infos.updated_at IS NULL;
COMMENT ON COLUMN auth_infos.deleted_at IS NULL;

COMMENT ON TABLE users IS NULL;
COMMENT ON COLUMN users.id IS NULL;
COMMENT ON COLUMN users.account_id IS NULL;
COMMENT ON COLUMN users.nickname IS NULL;
COMMENT ON COLUMN users.tenant_id IS NULL;
COMMENT ON COLUMN users.created_at IS NULL;
COMMENT ON COLUMN users.updated_at IS NULL;
COMMENT ON COLUMN users.deleted_at IS NULL;

COMMENT ON TABLE departments IS NULL;
COMMENT ON COLUMN departments.id IS NULL;
COMMENT ON COLUMN departments.parent_id IS NULL;
COMMENT ON COLUMN departments.name IS NULL;
COMMENT ON COLUMN departments."order" IS NULL;
COMMENT ON COLUMN departments.status IS NULL;
COMMENT ON COLUMN departments.tenant_id IS NULL;
COMMENT ON COLUMN departments.created_at IS NULL;
COMMENT ON COLUMN departments.updated_at IS NULL;
COMMENT ON COLUMN departments.deleted_at IS NULL;

COMMENT ON TABLE department_users IS NULL;
COMMENT ON COLUMN department_users.department_id IS NULL;
COMMENT ON COLUMN department_users.user_id IS NULL;

COMMENT ON TABLE groups IS NULL;
COMMENT ON COLUMN groups.id IS NULL;
COMMENT ON COLUMN groups.name IS NULL;
COMMENT ON COLUMN groups.kind IS NULL;
COMMENT ON COLUMN groups.describe IS NULL;
COMMENT ON COLUMN groups.creator_id IS NULL;
COMMENT ON COLUMN groups.updater_id IS NULL;
COMMENT ON COLUMN groups.tenant_id IS NULL;
COMMENT ON COLUMN groups.created_at IS NULL;
COMMENT ON COLUMN groups.updated_at IS NULL;
COMMENT ON COLUMN groups.deleted_at IS NULL;

COMMENT ON TABLE user_groups IS NULL;
COMMENT ON COLUMN user_groups.group_id IS NULL;
COMMENT ON COLUMN user_groups.user_id IS NULL;

COMMENT ON TABLE resources IS NULL;
COMMENT ON COLUMN resources.id IS NULL;
COMMENT ON COLUMN resources.name IS NULL;
COMMENT ON COLUMN resources.scope IS NULL;
COMMENT ON COLUMN resources.kind IS NULL;

COMMENT ON TABLE roles IS NULL;
COMMENT ON COLUMN roles.id IS NULL;
COMMENT ON COLUMN roles.name IS NULL;
COMMENT ON COLUMN roles.scope IS NULL;
COMMENT ON COLUMN roles.namespace IS NULL;
COMMENT ON COLUMN roles.rules IS NULL;
COMMENT ON COLUMN roles.tenant_id IS NULL;
COMMENT ON COLUMN roles.created_at IS NULL;
COMMENT ON COLUMN roles.updated_at IS NULL;
COMMENT ON COLUMN roles.deleted_at IS NULL;

COMMENT ON TABLE user_roles IS NULL;
COMMENT ON COLUMN user_roles.user_id IS NULL;
COMMENT ON COLUMN user_roles.role_id IS NULL;

COMMENT ON TABLE group_roles IS NULL;
COMMENT ON COLUMN group_roles.group_id IS NULL;
COMMENT ON COLUMN group_roles.role_id IS NULL;

COMMENT ON TABLE audit_logs IS NULL;
COMMENT ON COLUMN audit_logs.id IS NULL;
COMMENT ON COLUMN audit_logs.tenant_id IS NULL;
COMMENT ON COLUMN audit_logs.account_id IS NULL;
COMMENT ON COLUMN audit_logs.member_id IS NULL;
COMMENT ON COLUMN audit_logs.module IS NULL;
COMMENT ON COLUMN audit_logs.action IS NULL;
COMMENT ON COLUMN audit_logs.resource_type IS NULL;
COMMENT ON COLUMN audit_logs.resource_id IS NULL;
COMMENT ON COLUMN audit_logs.request_id IS NULL;
COMMENT ON COLUMN audit_logs.ip IS NULL;
COMMENT ON COLUMN audit_logs.user_agent IS NULL;
COMMENT ON COLUMN audit_logs.before_data IS NULL;
COMMENT ON COLUMN audit_logs.after_data IS NULL;
COMMENT ON COLUMN audit_logs.created_at IS NULL;
