-- 000011: 存量租户 authenticated 角色补授「账号自助」资源
-- /accounts/me 全部为本人范围（资料/密码/注册画像），已认证成员即应可访问；
-- 新租户种子已在 seedTenantBaseline 同步补齐，本迁移只订正存量行（幂等）
UPDATE roles
SET rules = (rules::jsonb || '[{"resource": "accounts", "operation": "*"}]'::jsonb)::json
WHERE name = 'authenticated'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'accounts'
  );
