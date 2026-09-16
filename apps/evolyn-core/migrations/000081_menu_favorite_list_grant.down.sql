-- 000081 down: 回收菜单收藏列表权限（仅移除 list 操作项，保留 000047 的
-- create/delete；幂等，范围与 up 对称）。

UPDATE tn_roles
SET rules = (
    SELECT COALESCE(json_agg(e), '[]'::json)
    FROM json_array_elements(rules::jsonb) AS e
    WHERE NOT (e::json->>'resource' = 'menu-favorites' AND e::json->>'operation' = 'list')
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS e
      WHERE e::json->>'resource' = 'menu-favorites' AND e::json->>'operation' = 'list'
  );
