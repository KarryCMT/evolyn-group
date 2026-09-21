-- 000082: 修复流程权限基线漂移。000048/000049 只补授了当时已存在的
-- 角色，但租户开通的 Go 侧基线种子未同步流程权限，导致之后创建的
-- 租户即使创建人已绑定租户管理员，仍无法访问 /workflows。

-- 管理员角色仍按既有规则签名识别（不依赖可展示的角色名），并逐项
-- 追加尚未拥有通配权限的流程资源，保留用户自定义规则。
UPDATE tn_roles AS r
SET rules = (
    r.rules::jsonb || COALESCE((
        SELECT jsonb_agg(candidate.rule)
        FROM jsonb_array_elements('[
          {"resource": "workflows", "operation": "*"},
          {"resource": "workflow-instances", "operation": "*"},
          {"resource": "workflow-tasks", "operation": "*"}
        ]'::jsonb) AS candidate(rule)
        WHERE NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
            WHERE existing.rule->>'resource' = candidate.rule->>'resource'
              AND existing.rule->>'operation' = '*'
        )
    ), '[]'::jsonb)
)::json
WHERE r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) AS rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*')
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements('[
        {"resource": "workflows", "operation": "*"},
        {"resource": "workflow-instances", "operation": "*"},
        {"resource": "workflow-tasks", "operation": "*"}
      ]'::jsonb) AS candidate(rule)
      WHERE NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
          WHERE existing.rule->>'resource' = candidate.rule->>'resource'
            AND existing.rule->>'operation' = '*'
      )
  );

-- 已认证用户基线角色补齐流程实例和待办的参与权限；具体数据范围
-- 仍由发起人与 TaskActor 校验收窄，不会因 URL 资源门放大跨成员访问。
UPDATE tn_roles AS r
SET rules = (
    r.rules::jsonb || COALESCE((
        SELECT jsonb_agg(candidate.rule)
        FROM jsonb_array_elements('[
          {"resource": "workflow-instances", "operation": "create"},
          {"resource": "workflow-instances", "operation": "view"},
          {"resource": "workflow-tasks", "operation": "create"},
          {"resource": "workflow-tasks", "operation": "view"}
        ]'::jsonb) AS candidate(rule)
        WHERE NOT EXISTS (
            SELECT 1
            FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
            WHERE existing.rule->>'resource' = candidate.rule->>'resource'
              AND existing.rule->>'operation' IN (candidate.rule->>'operation', '*')
        )
    ), '[]'::jsonb)
)::json
WHERE r.id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND r.deleted_at IS NULL
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements('[
        {"resource": "workflow-instances", "operation": "create"},
        {"resource": "workflow-instances", "operation": "view"},
        {"resource": "workflow-tasks", "operation": "create"},
        {"resource": "workflow-tasks", "operation": "view"}
      ]'::jsonb) AS candidate(rule)
      WHERE NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
          WHERE existing.rule->>'resource' = candidate.rule->>'resource'
            AND existing.rule->>'operation' IN (candidate.rule->>'operation', '*')
      )
  );
