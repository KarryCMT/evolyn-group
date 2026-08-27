-- 回滚 000036：移除基线管理员 enterprise-logs 补授规则（幂等，与 up 对称），
-- 再删除导出任务表与两张日志表的展示投影字段/索引。
UPDATE roles
SET rules = (
    SELECT COALESCE(json_agg(rule), '[]'::json)::json
    FROM json_array_elements(rules) AS rule
    WHERE rule->>'resource' <> 'enterprise-logs'
)
WHERE deleted_at IS NULL
  AND json_typeof(COALESCE(rules, '[]'::json)) = 'array'
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'members' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'roles' AND rule->>'operation' = '*'
  )
  AND EXISTS (
      SELECT 1 FROM json_array_elements(rules) AS rule
      WHERE rule->>'resource' = 'departments' AND rule->>'operation' = '*'
  );

DROP TABLE IF EXISTS enterprise_log_exports;

DROP INDEX IF EXISTS idx_audit_logs_tenant_member_created;
DROP INDEX IF EXISTS idx_audit_logs_tenant_category_created;
DROP INDEX IF EXISTS idx_audit_logs_tenant_created;
ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS event_code,
    DROP COLUMN IF EXISTS category_code,
    DROP COLUMN IF EXISTS actor_name_snapshot,
    DROP COLUMN IF EXISTS target_name_snapshot,
    DROP COLUMN IF EXISTS summary;

DROP INDEX IF EXISTS idx_login_logs_tenant_member_created;
DROP INDEX IF EXISTS idx_login_logs_tenant_created;
ALTER TABLE login_logs DROP COLUMN IF EXISTS actor_name_snapshot;
