-- 000060 回滚：移除流程定义的表单绑定列与配套索引。
DROP INDEX IF EXISTS idx_wf_definition_form_code;
DROP INDEX IF EXISTS uk_wf_definition_form_code;
ALTER TABLE wf_definition DROP COLUMN IF EXISTS form_code;
