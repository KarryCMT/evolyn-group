-- 000077 回滚：先撤 workbench 权限补授，再删除成员工作台表。
-- 个人配置无保留语义，直接物理删除。

UPDATE tn_roles
SET rules = (
    SELECT COALESCE(jsonb_agg(e), '[]'::jsonb)
    FROM json_array_elements(rules) AS e
    WHERE e::json->>'resource' <> 'workbench'
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS e
      WHERE e::json->>'resource' = 'workbench'
  );

DROP TABLE IF EXISTS tn_member_workbenches;
