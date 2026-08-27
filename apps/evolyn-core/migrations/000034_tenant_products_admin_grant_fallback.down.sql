-- 回滚 000034：从按签名补授的角色中移除 tenant-products 规则（幂等）。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(
        CASE WHEN json_typeof(COALESCE(rules, '[]'::json)) = 'array' THEN rules ELSE '[]'::json END
    ) AS rule
    WHERE rule->>'resource' <> 'tenant-products'
)
WHERE deleted_at IS NULL;
