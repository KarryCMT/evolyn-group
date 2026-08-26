-- 000032: 权限中心-管理员模块（管理组）。
-- admin_groups：租户管理组（系统管理组/普通管理组），scope_config 为带范围的
-- 委托权限配置（部门/角色/互联组织/应用，先例 roles.rules 的 JSONB 口径）；
-- admin_group_members：自定义管理组的成员绑定（内置系统管理员组不落此表，
-- 成员由 tenant-admin 角色绑定实时推导，避免双事实源）。
-- 附：存量租户回填内置「系统管理员」组 + 租户管理员基线角色补授 admin-groups。

-- 管理组主表：name 合法性（非空/≤30）由服务层校验，数据库不做枚举约束
CREATE TABLE IF NOT EXISTS admin_groups (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name varchar(30) NOT NULL,
    scope varchar(16) NOT NULL,
    built_in boolean NOT NULL DEFAULT false,
    scope_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

-- 有效记录内 (tenant_id, name) 唯一：服务层预检 + 索引兜底（口径同 groups/roles）
CREATE UNIQUE INDEX IF NOT EXISTS uk_admin_groups_tenant_name
    ON admin_groups (tenant_id, name) WHERE deleted_at IS NULL;
-- 列表查询索引（按 scope 过滤）
CREATE INDEX IF NOT EXISTS idx_admin_groups_tenant_scope
    ON admin_groups (tenant_id, scope) WHERE deleted_at IS NULL;

-- 管理组成员绑定：移除即删行（无软删），变更流水走审计域
CREATE TABLE IF NOT EXISTS admin_group_members (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    admin_group_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    created_at timestamp with time zone,
    CONSTRAINT uk_admin_group_member UNIQUE (admin_group_id, member_id)
);
-- 成员反查（鉴权门按成员定位所属管理组）
CREATE INDEX IF NOT EXISTS idx_admin_group_members_member
    ON admin_group_members (tenant_id, member_id);

-- ---- 存量租户回填内置系统管理员组（幂等）：每租户一行 built_in 组，
-- ---- 成员不落表（读侧由 tenant-admin 角色绑定实时推导） ----
INSERT INTO admin_groups (tenant_id, name, scope, built_in, scope_config, created_at, updated_at)
SELECT t.id, '系统管理员', 'system', true, '{}'::jsonb, now(), now()
FROM tenants t
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
ON CONFLICT DO NOTHING;

-- ---- 权限：租户管理员基线角色补授 admin-groups（幂等，口径同 000030/000031）----
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'admin-groups', 'operation', '*'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'admin-groups'
  );

-- ---- 数据字典注释 ----
COMMENT ON TABLE admin_groups IS '租户管理组（权限中心-管理员模块）：一组成员 + 对部门/角色/应用/互联组织的带范围委托管理权；内置组（built_in）成员由 tenant-admin 角色绑定推导';
COMMENT ON COLUMN admin_groups.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN admin_groups.name IS '管理组名称，租户内有效记录唯一，最长 30 字符（服务层校验）';
COMMENT ON COLUMN admin_groups.scope IS '管理组类型：system=系统管理员页（通讯录管理组），application=灵衍云管理员页（普通管理组）';
COMMENT ON COLUMN admin_groups.built_in IS '是否内置组：true 为系统管理员组，不可改名/删除/改配置，成员读写代理到 tenant-admin 角色绑定';
COMMENT ON COLUMN admin_groups.scope_config IS '范围配置 JSONB：department/role/externalOrg/application/addressBook 区块，按 scope 适用性由服务层校验；ID 清单悬挂引用由读取侧丢弃';
COMMENT ON COLUMN admin_groups.created_at IS '创建时间';
COMMENT ON COLUMN admin_groups.updated_at IS '最后更新时间';
COMMENT ON COLUMN admin_groups.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE admin_group_members IS '管理组成员绑定（自定义组）：移除即删行，变更流水走审计域；内置系统管理员组不使用本表';
COMMENT ON COLUMN admin_group_members.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN admin_group_members.admin_group_id IS '管理组 admin_groups.id（同租户）';
COMMENT ON COLUMN admin_group_members.member_id IS '租户成员 users.id（同租户，服务层校验）';
COMMENT ON COLUMN admin_group_members.created_at IS '加入时间';
