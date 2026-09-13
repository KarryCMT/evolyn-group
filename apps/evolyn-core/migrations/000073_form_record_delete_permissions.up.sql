-- 000073：数据管理批量删除使用 form-records:delete，和表单设计资源 forms
-- 分离。按管理员角色的稳定规则签名定位，避免依赖可改名的 tenant-admin 文案。
WITH administrator_roles AS (
    SELECT r.id
    FROM tn_roles r
    WHERE r.deleted_at IS NULL
      AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
      AND EXISTS (
          SELECT 1
          FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
          WHERE rule->>'resource' = 'members'
            AND rule->>'operation' = '*'
      )
      AND EXISTS (
          SELECT 1
          FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
          WHERE rule->>'resource' = 'forms'
            AND rule->>'operation' = '*'
      )
)
UPDATE tn_roles r
SET rules = (r.rules::jsonb || jsonb_build_array(
    jsonb_build_object('resource', 'form-records', 'operation', 'delete')
))::json
WHERE r.id IN (SELECT id FROM administrator_roles)
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'form-records'
        AND rule->>'operation' IN ('delete', '*')
  );
