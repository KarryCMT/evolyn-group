DROP TABLE IF EXISTS files;

-- 仅移除本版本追加的 files 规则；其他资源规则保持不变。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(COALESCE(roles.rules, '[]'::json)) AS rule
    WHERE rule::json->>'resource' <> 'files'
)
WHERE name IN ('tenant-admin', 'authenticated');
