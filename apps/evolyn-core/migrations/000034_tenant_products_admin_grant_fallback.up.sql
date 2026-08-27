-- 000034: 产品中心权限补授兜底（docs/低代码平台/产品中心/）。
-- 000033 按 name = '租户管理员' 补授 tenant-products，但基线管理员角色名是
-- 可在角色管理页修改的展示字段：改名后的角色既拿不到本次补授，也不被
-- IsTenantAdmin 识别。本迁移改按「管理员规则签名」补授——同时持有
-- members:* / roles:* / departments:* 的角色即视为租户管理员基线（改名
-- 不改规则），与其名字无关；不修改任何角色名称。
-- 注：rules 列可能被清成 JSON null（用户自建空角色），json_array_elements
-- 对非数组报错，一律先做 json_typeof 守卫。

UPDATE roles
SET rules = (
    rules::jsonb
    || jsonb_build_array(
        jsonb_build_object('resource', 'tenant-products', 'operation', 'view'),
        jsonb_build_object('resource', 'tenant-products', 'operation', 'update')
    )
)::json
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
  )
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'tenant-products'
  );
