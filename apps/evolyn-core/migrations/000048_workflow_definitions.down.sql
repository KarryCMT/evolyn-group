-- 000048 down：与 up 严格对称（先撤规则再撤表/索引，全部 IF EXISTS）。
UPDATE roles
SET rules = (
    (
        SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
        FROM json_array_elements(rules) AS element
        WHERE element::jsonb->>'resource' <> 'workflows'
    )
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array';

DROP INDEX IF EXISTS idx_wf_definition_version_tenant;
DROP INDEX IF EXISTS uk_wf_definition_version_no;
DROP TABLE IF EXISTS wf_definition_version;
DROP INDEX IF EXISTS uk_wf_definition_tenant_code;
DROP INDEX IF EXISTS idx_wf_definition_tenant;
DROP TABLE IF EXISTS wf_definition;
