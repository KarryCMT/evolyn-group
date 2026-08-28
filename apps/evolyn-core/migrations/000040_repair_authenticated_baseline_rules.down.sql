-- 回滚 000040：只撤销系统已认证分组继承角色的本次补授，不影响其他角色。
WITH authenticated_roles AS (
    SELECT DISTINCT r.id
    FROM roles r
    INNER JOIN group_roles gr ON gr.role_id = r.id
    INNER JOIN groups g ON g.id = gr.group_id
    WHERE g.name = 'system:authenticated'
      AND g.kind = 'system'
      AND r.deleted_at IS NULL
)
UPDATE roles r
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(COALESCE(r.rules::jsonb, '[]'::jsonb)) AS rule
    WHERE NOT (
        (rule->>'resource' = 'form-records' AND rule->>'operation' = 'create')
        OR (rule->>'resource' = 'notifications' AND rule->>'operation' IN ('view', 'update'))
    )
)::json
WHERE r.id IN (SELECT id FROM authenticated_roles);
