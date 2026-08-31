-- 000058 回滚：删除资产权限组两表并回收基线管理员补授规则。

DROP TABLE IF EXISTS asset_permission_group_subjects;
DROP TABLE IF EXISTS asset_permission_groups;

UPDATE roles
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
    WHERE COALESCE(rule->>'resource', '') NOT IN ('form-permissions', 'form-data')
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' IN ('form-permissions', 'form-data')
  );
