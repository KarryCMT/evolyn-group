-- 000082 down: 回收本迁移补授的流程基线规则。仅删除完全匹配的资源与
-- 操作组合，不影响其他自定义流程权限。

UPDATE tn_roles AS r
SET rules = (
    SELECT COALESCE(jsonb_agg(existing.rule), '[]'::jsonb)
    FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
    WHERE NOT (
        existing.rule->>'resource' IN ('workflows', 'workflow-instances', 'workflow-tasks')
        AND existing.rule->>'operation' = '*'
    )
)::json
WHERE r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*');

UPDATE tn_roles AS r
SET rules = (
    SELECT COALESCE(jsonb_agg(existing.rule), '[]'::jsonb)
    FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
    WHERE NOT (
        existing.rule->>'resource' IN ('workflow-instances', 'workflow-tasks')
        AND existing.rule->>'operation' IN ('create', 'view')
    )
)::json
WHERE r.id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';
