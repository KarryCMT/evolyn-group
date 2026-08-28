-- 000044 down：移除表单类型约束与字段。
ALTER TABLE forms
  DROP CONSTRAINT IF EXISTS chk_forms_form_type,
  DROP COLUMN IF EXISTS form_type;
