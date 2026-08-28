-- 000040：修复 000029 本地化系统角色名称后，000038/000039 仍以旧英文名称
-- “authenticated”补授权限而失效的问题。系统已认证分组是权限继承的事实来源，
-- 通过它关联角色，避免角色展示名称变更再次导致存量租户漏授。
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
    r.rules::jsonb
    || COALESCE((
        SELECT jsonb_agg(candidate)
        FROM jsonb_array_elements('[
            {"resource": "form-records", "operation": "create"},
            {"resource": "notifications", "operation": "view"},
            {"resource": "notifications", "operation": "update"}
        ]'::jsonb) AS candidate
        WHERE NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(COALESCE(r.rules::jsonb, '[]'::jsonb)) AS existing_rule
            WHERE existing_rule->>'resource' = candidate->>'resource'
              AND existing_rule->>'operation' = candidate->>'operation'
        )
    ), '[]'::jsonb)
)::json
WHERE r.id IN (SELECT id FROM authenticated_roles)
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';
