-- 000067 回滚：移除系统字段列（回填的展示名快照一并丢弃，属预期损失）。
ALTER TABLE tn_form_records DROP COLUMN IF EXISTS updated_at;
ALTER TABLE tn_form_records DROP COLUMN IF EXISTS submitted_by_name;
