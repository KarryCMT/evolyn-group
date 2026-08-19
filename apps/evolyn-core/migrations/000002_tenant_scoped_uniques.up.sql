-- FIX-002/003：Role/Group 名称由全局唯一改为租户内唯一（软删除友好）。
-- 全局 UNIQUE 会阻止两个租户各自创建同名角色/分组，破坏多租户模型；
-- 部分唯一索引（WHERE deleted_at IS NULL）允许软删后重建同名。

-- 删除表约束形态的全局唯一（db.sql 路径产生的 *_name_key）
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_name_key;
-- 删除 AutoMigrate 路径产生的索引形态全局唯一
DROP INDEX IF EXISTS idx_roles_name;
DROP INDEX IF EXISTS idx_groups_name;

CREATE UNIQUE INDEX IF NOT EXISTS uk_roles_tenant_name
    ON roles (tenant_id, name)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uk_groups_tenant_name
    ON groups (tenant_id, name)
    WHERE deleted_at IS NULL;
