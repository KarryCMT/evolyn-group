-- 000037 down：与 up 严格对称（先撤规则再撤表/索引，全部 IF EXISTS）。
UPDATE roles
SET rules = (
    (
        SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
        FROM json_array_elements(rules) AS element
        WHERE element::jsonb->>'resource' <> 'forms'
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';

DROP INDEX IF EXISTS idx_form_versions_tenant;
DROP INDEX IF EXISTS uk_form_versions_form_no;
DROP TABLE IF EXISTS form_versions;
DROP INDEX IF EXISTS idx_forms_tenant_app;
DROP TABLE IF EXISTS forms;
