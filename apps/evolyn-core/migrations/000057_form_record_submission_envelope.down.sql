-- 000057 down：移除提交协议幂等键与菜单入口快照。
DROP INDEX IF EXISTS uk_form_records_tenant_data_op;
ALTER TABLE form_records DROP COLUMN IF EXISTS entry_code;
ALTER TABLE form_records DROP COLUMN IF EXISTS data_op_id;
