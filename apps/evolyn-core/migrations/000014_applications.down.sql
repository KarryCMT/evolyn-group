-- 000014 down: 回滚应用管理域——先摘除基线角色 applications 规则，再按依赖序删表
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(e), '[]'::json)
    FROM json_array_elements_text(rules) AS e
    WHERE e::json->>'resource' <> 'applications'
)
WHERE name IN ('tenant-admin', 'authenticated');

DROP TABLE IF EXISTS application_installations;
DROP TABLE IF EXISTS applications;
