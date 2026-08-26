-- 000030 down：回滚版本信息一期。
-- 先收回「租户管理员」角色的 editions 规则，再按依赖逆序删除四表。
-- tenants.plan/tenants.quotas 兼容投影保留原值，回滚后配额执行回到旧语义。
UPDATE roles
SET rules = (
    SELECT COALESCE(jsonb_agg(e), '[]'::jsonb)
    FROM jsonb_array_elements(COALESCE(rules::jsonb, '[]'::jsonb)) AS e
    WHERE e->>'resource' <> 'editions'
)::json
WHERE name = '租户管理员'
  AND EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'editions'
  );

DROP TABLE IF EXISTS tenant_entitlement_overrides;
DROP TABLE IF EXISTS tenant_subscriptions;
DROP TABLE IF EXISTS edition_plan_versions;
DROP TABLE IF EXISTS edition_plans;
