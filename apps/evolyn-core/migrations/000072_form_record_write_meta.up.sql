-- 000072: 表单记录信封补「最后写人人」双列（000067 系统字段族扩展）
-- 创建人=提交人（提交即创建）；审批编辑/发起人修改写回时刷新为操作人。
ALTER TABLE tn_form_records
    ADD COLUMN IF NOT EXISTS updated_by_member_id BIGINT,
    ADD COLUMN IF NOT EXISTS updated_by_name TEXT;

COMMENT ON COLUMN tn_form_records.updated_by_member_id IS '最后写人人成员 ID（提交时=提交人；审批编辑/发起人修改写回时刷新为操作人；系统自动路径保持原值）';
COMMENT ON COLUMN tn_form_records.updated_by_name IS '最后写人人展示名快照（与提交人快照同口径：成员改名/退出后历史展示不失真）';

-- 存量回填：历史记录无写回流水可考，「最后写人」回落提交人
UPDATE tn_form_records
SET updated_by_member_id = submitted_by_member_id,
    updated_by_name      = submitted_by_name
WHERE updated_by_member_id IS NULL;
