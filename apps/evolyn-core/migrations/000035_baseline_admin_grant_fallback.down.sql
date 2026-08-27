-- 回滚 000035：从按签名补授的角色中移除三个基线资源规则（幂等，范围与 up 对称）。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(rules) AS rule
    WHERE rule->>'resource' NOT IN ('editions', 'member-field-settings', 'admin-groups')
)
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  );
