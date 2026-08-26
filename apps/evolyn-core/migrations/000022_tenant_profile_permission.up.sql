-- 租户管理员可自助维护组织根节点（租户名称）；套餐、配额等运营字段仍不下放。
-- 仅在规则中尚未存在 tenant 资源时追加，迁移重放不会重复授权。
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'tenant', 'operation', 'edit'))
)::json
WHERE name = 'tenant-admin'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'tenant'
  );
