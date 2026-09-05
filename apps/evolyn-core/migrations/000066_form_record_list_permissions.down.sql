-- 回滚 000066：仅回收 authenticated 系统分组继承角色的 form-records:view。
WITH authenticated_roles AS (
    SELECT DISTINCT r.id
    FROM tn_roles r
    INNER JOIN tn_group_roles gr ON gr.role_id = r.id
    INNER JOIN tn_groups g ON g.id = gr.group_id
    WHERE g.name = 'system:authenticated'
      AND g.kind = 'system'
      AND r.deleted_at IS NULL
)
UPDATE tn_roles r
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
    WHERE NOT (
        rule->>'resource' = 'form-records'
        AND rule->>'operation' = 'view'
    )
)::json
WHERE r.id IN (SELECT id FROM authenticated_roles)
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';
