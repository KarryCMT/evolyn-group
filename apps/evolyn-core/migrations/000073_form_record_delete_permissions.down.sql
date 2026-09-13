-- 回滚 000073：仅回收本迁移追加的 form-records:delete 规则。
UPDATE tn_roles r
SET rules = (
    SELECT COALESCE(jsonb_agg(rule), '[]'::jsonb)
    FROM json_array_elements(COALESCE(r.rules, '[]'::json)) AS rule
    WHERE NOT (
        rule->>'resource' = 'form-records'
        AND rule->>'operation' = 'delete'
    )
)::json
WHERE json_typeof(COALESCE(r.rules, '[]'::json)) = 'array';
