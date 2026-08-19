-- 回滚 000011：从 authenticated 角色移除「账号自助」资源规则
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(e), '[]'::json)
    FROM json_array_elements_text(rules) AS e
    WHERE e::json->>'resource' <> 'accounts'
)
WHERE name = 'authenticated'
  AND EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'accounts'
  );
