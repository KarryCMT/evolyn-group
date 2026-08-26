-- 回滚本版本追加的 tenant 资源规则；不影响其他角色权限。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(COALESCE(roles.rules, '[]'::json)) AS rule
    WHERE rule->>'resource' <> 'tenant'
)
WHERE name = 'tenant-admin';
