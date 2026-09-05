-- 000066：表单记录列表属于 form-records 数据面，而非 forms 设计管理面。
-- 通过 authenticated 系统分组定位实际继承角色（角色展示名允许变化），补授
-- view；路由层再将 /forms/:code/records 受控映射为 form-records:get。
-- 这只开放进入数据面，具体行范围与字段矩阵仍由 FormPermissionEvaluator
-- 在 Service/Repository 查询链路裁决。
WITH authenticated_roles AS (
    SELECT DISTINCT r.id
    FROM tn_roles r
    INNER JOIN tn_group_roles gr ON gr.role_id = r.id
    INNER JOIN tn_groups g ON g.id = gr.group_id
    WHERE g.name = 'system:authenticated'
      AND g.kind = 'system'
      AND r.deleted_at IS NULL
)
UPDATE tn_roles r
SET rules = (
    r.rules::jsonb || jsonb_build_array(
        jsonb_build_object('resource', 'form-records', 'operation', 'view')
    )
)::json
WHERE r.id IN (SELECT id FROM authenticated_roles)
  AND json_typeof(COALESCE(r.rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1
      FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
      WHERE rule->>'resource' = 'form-records'
        AND rule->>'operation' = 'view'
  );
