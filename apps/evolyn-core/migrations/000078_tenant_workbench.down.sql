-- 000078 回滚：恢复 000077 的成员个人配置模型（表结构还原；个人数据在
-- up 中已物理删除且无生产价值，不做数据反向迁移），权限补授还原为
-- authenticated 全体成员（view+update）。

UPDATE tn_roles
SET rules = (
    SELECT COALESCE(jsonb_agg(e), '[]'::jsonb)
    FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
    WHERE e::json->>'resource' <> 'workbench'
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
      WHERE e::json->>'resource' = 'workbench'
  );

UPDATE tn_roles
SET rules = (
    COALESCE(rules::jsonb, '[]'::jsonb)
    || '[{"resource": "workbench", "operation": "view"}, {"resource": "workbench", "operation": "update"}]'::jsonb
)::json
WHERE id IN (
      SELECT gr.role_id
      FROM tn_group_roles gr
      INNER JOIN tn_groups g ON g.id = gr.group_id
      WHERE g.name = 'system:authenticated' AND g.kind = 'system'
)
  AND deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND NOT EXISTS (
      SELECT 1 FROM json_array_elements(COALESCE(rules, '[]'::json)) AS e
      WHERE e::json->>'resource' = 'workbench'
  );

DROP TABLE IF EXISTS tn_workbenches;

CREATE TABLE tn_member_workbenches (
    id         BIGSERIAL   PRIMARY KEY,
    tenant_id  BIGINT      NOT NULL DEFAULT 1,
    member_id  BIGINT      NOT NULL,
    content    JSONB       NOT NULL,
    revision   BIGINT      NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_tn_member_workbenches_tenant_member UNIQUE (tenant_id, member_id)
);

COMMENT ON TABLE tn_member_workbenches IS '成员自定义工作台（000077，000078 回滚还原）：每租户每成员一份 DashboardSchema 单文档 JSON';
