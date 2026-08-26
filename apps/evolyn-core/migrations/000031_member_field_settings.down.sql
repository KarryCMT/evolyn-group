-- 000031 down：先收回租户管理员角色上的权限规则，再删除成员信息管理两张表。
-- member_profiles / tenant_member_field_settings 均为本迁移新增，直接 DROP。

UPDATE roles
SET rules = (
    SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS element
    WHERE element->>'resource' <> 'member-field-settings'
)::json
WHERE EXISTS (
    SELECT 1
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
    WHERE rule->>'resource' = 'member-field-settings'
);

DROP TABLE IF EXISTS member_profiles;
DROP TABLE IF EXISTS tenant_member_field_settings;
