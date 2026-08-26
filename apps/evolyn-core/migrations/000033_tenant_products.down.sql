-- 仅回滚本版本为租户管理员新增的 tenant-products 资源规则，保留其他授权配置
-- （口径同 000024 down；当前角色名为中文，见 000029）。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(COALESCE(roles.rules, '[]'::json)) AS rule
    WHERE rule->>'resource' <> 'tenant-products'
)
WHERE name = '租户管理员';

DROP TABLE IF EXISTS tenant_product_members;
DROP TABLE IF EXISTS tenant_product_departments;
DROP TABLE IF EXISTS tenant_product_configs;
DROP TABLE IF EXISTS product_catalogs;
