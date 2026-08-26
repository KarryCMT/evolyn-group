-- 租户成员接口采用 /members 资源；补齐存量租户管理员权限。
-- 创建者已绑定 tenant-admin，此处只追加缺失规则，确保迁移幂等且不覆盖自定义授权。
UPDATE roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || jsonb_build_array(jsonb_build_object('resource', 'members', 'operation', '*'))
)::json
WHERE name = 'tenant-admin'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'members'
  );
