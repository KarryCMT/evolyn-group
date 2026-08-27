-- 000035: 基线管理员权限补授兜底（ editions / member-field-settings / admin-groups）。
-- 与 000034 同因：000030/000031/000032 按 name = '租户管理员' 补授，改名后的
-- 基线角色（规则仍是管理员签名）三批授权全部漏授，导致 /editions/current 等
-- 页面 403。本迁移沿用 000034 的「管理员规则签名」口径（members:* + roles:* +
-- departments:*，与角色名无关）逐资源补齐缺失规则；rules 为非数组（如用户
-- 自建空角色）的行经 json_typeof 守卫跳过。

-- 版本信息只读概览（口径同 000030：editions:get）
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'editions', 'operation', 'get'))
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
      WHERE rule->>'resource' = 'editions'
  );

-- 成员信息管理（口径同 000031：member-field-settings:*）
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'member-field-settings', 'operation', '*'))
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
      WHERE rule->>'resource' = 'member-field-settings'
  );

-- 权限中心-管理员模块（口径同 000032：admin-groups:*）
UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'admin-groups', 'operation', '*'))
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
      WHERE rule->>'resource' = 'admin-groups'
  );
