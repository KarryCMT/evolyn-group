-- 仅回滚本版本为租户管理员新增的成员资源规则，保留其他授权配置。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(COALESCE(roles.rules, '[]'::json)) AS rule
    WHERE rule->>'resource' <> 'members'
)
WHERE name = 'tenant-admin';
