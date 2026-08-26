-- 内部组织页角色展示分组：只承担角色归类，不参与 group_roles 的权限继承。
CREATE TABLE IF NOT EXISTS role_groups (
    id BIGSERIAL PRIMARY KEY,
    name varchar(100) NOT NULL,
    creator_id BIGINT NOT NULL DEFAULT 0,
    tenant_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

ALTER TABLE roles ADD COLUMN IF NOT EXISTS role_group_id BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS uk_role_groups_tenant_name
    ON role_groups (tenant_id, name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_roles_role_group_id ON roles (role_group_id);

ALTER TABLE roles DROP CONSTRAINT IF EXISTS fk_roles_role_group;
ALTER TABLE roles ADD CONSTRAINT fk_roles_role_group
    FOREIGN KEY (role_group_id) REFERENCES role_groups(id) ON DELETE SET NULL;

COMMENT ON TABLE role_groups IS '角色展示分组：仅供内部组织页归类角色，不参与权限继承';
COMMENT ON COLUMN role_groups.name IS '角色组名称，租户内未删除记录唯一';
COMMENT ON COLUMN role_groups.creator_id IS '创建该角色组的成员 ID';
COMMENT ON COLUMN roles.role_group_id IS '角色所属展示分组 ID；为空表示未归类';
