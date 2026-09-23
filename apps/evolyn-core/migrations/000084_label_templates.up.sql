-- 000084: 二维码标签模板（LabelSchema V1 草稿 + 不可变发布快照）。
CREATE TABLE IF NOT EXISTS tn_label_templates (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    app_id BIGINT NOT NULL,
    form_id BIGINT NOT NULL REFERENCES tn_forms(id),
    form_code VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    draft_schema JSONB NOT NULL,
    draft_revision BIGINT NOT NULL DEFAULT 1,
    previewed_draft_revision BIGINT NOT NULL DEFAULT 0,
    latest_version_id BIGINT,
    published_version INTEGER NOT NULL DEFAULT 0,
    published_draft_revision BIGINT NOT NULL DEFAULT 0,
    width NUMERIC(12,4) NOT NULL,
    height NUMERIC(12,4) NOT NULL,
    unit VARCHAR(10) NOT NULL DEFAULT 'mm',
    dpi INTEGER NOT NULL DEFAULT 300,
    creator_member_id BIGINT NOT NULL,
    creator_id BIGINT,
    updater_id BIGINT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_tn_label_templates_status CHECK (status IN ('draft', 'published', 'disabled')),
    CONSTRAINT ck_tn_label_templates_unit CHECK (unit IN ('mm', 'px')),
    CONSTRAINT ck_tn_label_templates_schema CHECK (jsonb_typeof(draft_schema) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_label_templates_tenant_code
    ON tn_label_templates (tenant_id, code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uk_tn_label_templates_form
    ON tn_label_templates (tenant_id, form_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tn_label_templates_tenant
    ON tn_label_templates (tenant_id, id DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS tn_label_template_versions (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    template_id BIGINT NOT NULL REFERENCES tn_label_templates(id),
    version_no INTEGER NOT NULL,
    schema_version VARCHAR(20) NOT NULL DEFAULT '1.0',
    schema_snapshot JSONB NOT NULL,
    published_by_member_id BIGINT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ,
    CONSTRAINT ck_tn_label_template_versions_schema CHECK (jsonb_typeof(schema_snapshot) = 'object'),
    CONSTRAINT uk_tn_label_template_versions_no UNIQUE (template_id, version_no)
);

CREATE INDEX IF NOT EXISTS idx_tn_label_template_versions_tenant
    ON tn_label_template_versions (tenant_id, template_id, version_no DESC);

-- 标签设计和正式渲染先授予基线管理员；记录级查看仍由表单权限组二次裁决。
UPDATE tn_roles AS r
SET rules = (
    r.rules::jsonb || COALESCE((
        SELECT jsonb_agg(candidate.rule)
        FROM jsonb_array_elements('[
          {"resource": "label-templates", "operation": "*"},
          {"resource": "labels", "operation": "*"}
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
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*')
  AND EXISTS (SELECT 1 FROM json_array_elements(r.rules) rule WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*')
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements('[
        {"resource": "label-templates", "operation": "*"},
        {"resource": "labels", "operation": "*"}
      ]'::jsonb) AS candidate(rule)
      WHERE NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(r.rules::jsonb) AS existing(rule)
          WHERE existing.rule->>'resource' = candidate.rule->>'resource'
            AND existing.rule->>'operation' = '*'
      )
  );

COMMENT ON TABLE tn_label_templates IS '二维码标签模板：draft_schema 为 LabelSchema V1 唯一设计事实源，draft_revision 为乐观锁口令；每个有效表单至多绑定一个模板';
COMMENT ON TABLE tn_label_template_versions IS '标签模板不可变发布快照：正式渲染固定读取 published_version 指向的 schema_snapshot';
