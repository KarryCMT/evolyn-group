-- 000031: 成员信息管理（docs/低代码平台/成员信息管理/管理后台-成员信息管理开发文档.md）。
-- tenant_member_field_settings：租户级字段显示策略（字段设置/卡片展示两页签共用，
-- 每租户每字段一行，revision 为租户配置快照版本，同租户所有行同步递增）；
-- member_profiles：正式成员扩展档案（工号/职务等，邀请接受时从邀请草稿迁入；
-- 不重复存储手机/邮箱/部门/角色）。
-- 附：存量租户默认配置回填 + 租户管理员角色补授 member-field-settings 权限。

-- 租户字段显示策略：field_key 合法性由服务端字段注册表校验，不做数据库枚举
CREATE TABLE IF NOT EXISTS tenant_member_field_settings (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    field_key varchar(64) NOT NULL,
    personal_visible boolean NOT NULL DEFAULT true,
    personal_editable boolean NOT NULL DEFAULT false,
    card_visible boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

-- 有效记录内 (tenant_id, field_key) 唯一：seed 与读取兜底按此幂等
CREATE UNIQUE INDEX IF NOT EXISTS uk_member_field_settings_tenant_field
    ON tenant_member_field_settings (tenant_id, field_key) WHERE deleted_at IS NULL;
-- 租户配置快照读取索引
CREATE INDEX IF NOT EXISTS idx_member_field_settings_tenant_updated
    ON tenant_member_field_settings (tenant_id, updated_at) WHERE deleted_at IS NULL;

-- 正式成员扩展档案：identifier 为企业内编号（租户内有效记录唯一）；
-- attributes 只允许字段注册表定义的扩展 key（服务层校验，日期 YYYY-MM-DD、
-- 文本最长 50 字符）
CREATE TABLE IF NOT EXISTS member_profiles (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    identifier varchar(50),
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone
);

-- 每成员一份有效档案
CREATE UNIQUE INDEX IF NOT EXISTS uk_member_profiles_tenant_member
    ON member_profiles (tenant_id, member_id) WHERE deleted_at IS NULL;
-- 编号租户内唯一（空编号不参与唯一）
CREATE UNIQUE INDEX IF NOT EXISTS uk_member_profiles_tenant_identifier
    ON member_profiles (tenant_id, identifier) WHERE identifier <> '' AND deleted_at IS NULL;

-- ---- 存量租户默认配置回填（幂等）：15 个预置字段与服务端注册表逐项一致 ----
INSERT INTO tenant_member_field_settings (tenant_id, field_key, personal_visible, personal_editable, card_visible, revision)
SELECT t.id, f.field_key, f.personal_visible, f.personal_editable, f.card_visible, 1
FROM tenants t
CROSS JOIN (VALUES
    ('name',        true,  false, true),
    ('code',        false, false, true),
    ('mobile',      true,  true,  true),
    ('email',       true,  true,  true),
    ('department',  false, false, true),
    ('role',        false, false, false),
    ('alias',       false, false, false),
    ('employeeId',  false, false, false),
    ('gender',      false, false, false),
    ('position',    false, false, false),
    ('employment',  false, false, false),
    ('hireDate',    false, false, false),
    ('workplace',   false, false, false),
    ('birthday',    false, false, false),
    ('education',   false, false, false)
) AS f(field_key, personal_visible, personal_editable, card_visible)
WHERE t.deleted_at IS NULL
  AND t.status <> 'deleted'
ON CONFLICT DO NOTHING;

-- ---- 权限：租户管理员基线角色补授 member-field-settings（幂等，口径同 000024/000030）----
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'member-field-settings', 'operation', '*'))
)::json
WHERE name = '租户管理员'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'member-field-settings'
  );

-- ---- 数据字典注释 ----
COMMENT ON TABLE tenant_member_field_settings IS '租户成员字段显示策略：字段设置与卡片展示页签的租户级配置，每租户每字段一行';
COMMENT ON COLUMN tenant_member_field_settings.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN tenant_member_field_settings.field_key IS '预置字段 key（服务端字段注册表校验合法性，如 mobile/hireDate）';
COMMENT ON COLUMN tenant_member_field_settings.personal_visible IS '成员在个人设置页可见该字段';
COMMENT ON COLUMN tenant_member_field_settings.personal_editable IS '成员在个人设置页可编辑该字段（仅对扩展字段生效，手机/邮箱走绑定流程）';
COMMENT ON COLUMN tenant_member_field_settings.card_visible IS '成员资料卡片展示该字段（服务端按此裁剪卡片数据）';
COMMENT ON COLUMN tenant_member_field_settings.revision IS '租户配置快照版本号：同租户所有行同步递增，PATCH 以整页 revision 做乐观锁';
COMMENT ON COLUMN tenant_member_field_settings.created_at IS '创建时间';
COMMENT ON COLUMN tenant_member_field_settings.updated_at IS '最后更新时间';
COMMENT ON COLUMN tenant_member_field_settings.deleted_at IS '软删除时间，NULL 表示有效';

COMMENT ON TABLE member_profiles IS '正式成员扩展档案：别名/工号/性别/职务/日期等租户内资料，邀请接受时从邀请草稿迁入；不重复存储手机号、邮箱、部门和角色';
COMMENT ON COLUMN member_profiles.tenant_id IS '所属租户 ID';
COMMENT ON COLUMN member_profiles.member_id IS '对应租户成员 users.id，每成员一份有效档案';
COMMENT ON COLUMN member_profiles.identifier IS '企业内编号（编号字段 code 的数据来源），租户内有效记录唯一，空值不参与唯一';
COMMENT ON COLUMN member_profiles.attributes IS '扩展字段 JSONB：alias/employeeId/gender/position/employment/hireDate/workplace/birthday/education，日期统一 YYYY-MM-DD、文本最长 50 字符（服务层校验）';
COMMENT ON COLUMN member_profiles.created_at IS '创建时间';
COMMENT ON COLUMN member_profiles.updated_at IS '最后更新时间';
COMMENT ON COLUMN member_profiles.deleted_at IS '软删除时间，NULL 表示有效';
