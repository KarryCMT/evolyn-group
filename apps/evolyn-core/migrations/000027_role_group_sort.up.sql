-- 角色展示分组的排序仅服务于内部组织左侧树，数值越小展示越靠前。
ALTER TABLE role_groups
    ADD COLUMN IF NOT EXISTS sort INTEGER NOT NULL DEFAULT 0;

-- 为迁移前已有的角色组补齐稳定的初始顺序，避免全部落在同一个默认值上。
UPDATE role_groups
SET sort = id
WHERE sort = 0;

CREATE INDEX IF NOT EXISTS idx_role_groups_tenant_sort
    ON role_groups (tenant_id, sort, id)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN role_groups.sort IS '角色组在内部组织左侧角色树中的展示顺序，数值越小越靠前';
