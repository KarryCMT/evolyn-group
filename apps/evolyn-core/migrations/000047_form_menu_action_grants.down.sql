-- 回滚 000047：从按签名补授的角色移除 form-actions，从 authenticated 系统
-- 分组关联角色移除 menu-favorites（幂等，范围与 up 对称）。

UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(rules) AS rule
    WHERE rule->>'resource' <> 'form-actions'
)
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*');

UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(rules) AS rule
    WHERE rule->>'resource' <> 'menu-favorites'
)
WHERE id IN (
      SELECT gr.role_id
      FROM group_roles gr
      INNER JOIN groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';
