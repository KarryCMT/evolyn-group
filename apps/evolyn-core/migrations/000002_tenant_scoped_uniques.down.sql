-- 回滚：恢复全局唯一（会因跨租户同名数据失败，属预期防护）
DROP INDEX IF EXISTS uk_roles_tenant_name;
DROP INDEX IF EXISTS uk_groups_tenant_name;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups (name);
