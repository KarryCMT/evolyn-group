-- 000038 down：与 up 严格对称。
UPDATE roles
SET rules = (
    (
        SELECT COALESCE(jsonb_agg(element), '[]'::jsonb)
        FROM json_array_elements(rules) AS element
        WHERE element::jsonb->>'resource' <> 'form-records'
    )
)::json
WHERE name = 'authenticated'
  AND deleted_at IS NULL;

DROP INDEX IF EXISTS idx_form_records_tenant_form;
DROP TABLE IF EXISTS form_records;
