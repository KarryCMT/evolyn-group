-- 角色在同一展示分组中的拖拽顺序，数值越小越靠前。
ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS sort INTEGER NOT NULL DEFAULT 0;

-- 为已存在角色补齐稳定排序，确保首次加载不会因相同默认值而乱序。
UPDATE roles
SET sort = id
WHERE sort = 0;

CREATE INDEX IF NOT EXISTS idx_roles_role_group_sort
    ON roles (tenant_id, role_group_id, sort, id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN roles.sort IS '角色在所属展示分组中的展示顺序，数值越小越靠前';
