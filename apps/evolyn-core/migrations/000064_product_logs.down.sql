-- 回滚 000064：移除 product-logs 补授规则、tn_product_log_exports 表与
-- tn_audit_logs 应用维度投影（幂等，与 up 对称）

-- ---------- 基线管理员权限回收 ----------
UPDATE tn_roles
SET rules = (
    SELECT json_agg(rule) AS rules
    FROM json_array_elements(rules) AS rule
    WHERE rule->>'resource' <> 'product-logs'
)::json
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'product-logs'
  );

-- ---------- 导出任务表 ----------
DROP INDEX IF EXISTS idx_tn_product_log_exports_tenant;
DROP TABLE IF EXISTS tn_product_log_exports;

-- ---------- 应用维度投影 ----------
DROP INDEX IF EXISTS idx_tn_audit_logs_tenant_application_created;
ALTER TABLE tn_audit_logs
    DROP COLUMN IF EXISTS application_id,
    DROP COLUMN IF EXISTS application_code,
    DROP COLUMN IF EXISTS application_name_snapshot;
