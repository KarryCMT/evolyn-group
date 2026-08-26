-- 000032 down：回滚管理组（权限中心-管理员模块）。
-- 先删成员绑定再删主表；基线角色中的 admin-groups 规则一并摘除。

DELETE FROM admin_group_members;
DROP TABLE IF EXISTS admin_group_members;
DROP TABLE IF EXISTS admin_groups;

UPDATE roles
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
    WHERE rule->>'resource' <> 'admin-groups'
)::json
WHERE EXISTS (
    SELECT 1
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
    WHERE rule->>'resource' = 'admin-groups'
);
