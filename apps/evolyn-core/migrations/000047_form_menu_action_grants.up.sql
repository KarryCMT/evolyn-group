-- 000047: 表单菜单按钮动作授权补授（ADR-011）。
-- 1) 基线管理员（规则签名 members:* + roles:* + departments:*，与可改名的
--    角色名无关，口径同 000035/000037/000039）补授 form-actions:*——切换
--    表单类型/复制（当前应用/跨应用）/对成员隐藏的动作授权键。该资源不对应
--    任何 URL 首段，动作键仅由各域 Service 复核与菜单读取投影消费。
-- 2) 全体成员（authenticated 系统分组关联角色，口径同 000040）补授
--    menu-favorites create/delete——个人收藏（凡节点可见即可收藏）。
-- rules 为非数组（如用户自建空角色）的行经 json_typeof 守卫跳过。

UPDATE roles
SET rules = (
    rules::jsonb || jsonb_build_array(jsonb_build_object('resource', 'form-actions', 'operation', '*'))
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*')
  AND NOT EXISTS (SELECT 1 FROM json_array_elements(rules) AS rule WHERE rule->>'resource' = 'form-actions');

UPDATE roles
SET rules = (
    rules::jsonb
    || '[{"resource": "menu-favorites", "operation": "create"}, {"resource": "menu-favorites", "operation": "delete"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM group_roles gr
      INNER JOIN groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements_text(rules) AS e
      WHERE e::json->>'resource' = 'menu-favorites'
  );
