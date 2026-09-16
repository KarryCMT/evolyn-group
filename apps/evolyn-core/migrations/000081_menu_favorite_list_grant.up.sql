-- 000081: 菜单收藏列表权限补授（我的收藏 P2，GET /api/v1/menu-favorites）。
-- 全体成员（authenticated 系统分组关联角色，口径同 000047）补授
-- menu-favorites list——跨应用「我的收藏」读取。数据范围恒为
-- 「当前租户 × 当前成员」，由 Repository 双条件兜底，不做对象级放大；
-- 读侧可见性裁剪（应用归档/节点隐藏/资产权限失效）由服务层统一执行。
-- rules 为非数组（如用户自建空角色）的行经 json_typeof 守卫跳过。

UPDATE tn_roles
SET rules = (
    rules::jsonb || '[{"resource": "menu-favorites", "operation": "list"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
  )
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (SELECT 1 FROM json_array_elements(rules) AS e WHERE e::json->>'resource' = 'menu-favorites')
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS e
      WHERE e::json->>'resource' = 'menu-favorites' AND e::json->>'operation' IN ('list', '*')
  );
