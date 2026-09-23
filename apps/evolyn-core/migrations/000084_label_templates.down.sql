UPDATE tn_roles
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(rules) rule
    WHERE rule->>'resource' NOT IN ('label-templates', 'labels')
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';

DROP INDEX IF EXISTS idx_tn_label_template_versions_tenant;
DROP TABLE IF EXISTS tn_label_template_versions;
DROP INDEX IF EXISTS idx_tn_label_templates_tenant;
DROP INDEX IF EXISTS uk_tn_label_templates_form;
DROP INDEX IF EXISTS uk_tn_label_templates_tenant_code;
DROP TABLE IF EXISTS tn_label_templates;
