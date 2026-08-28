-- 000044: 表单类型成为持久化事实源；存量表单按标准表单回填。
ALTER TABLE forms
  ADD COLUMN form_type varchar(16) NOT NULL DEFAULT 'standard',
  ADD CONSTRAINT chk_forms_form_type CHECK (form_type IN ('standard', 'workflow'));

COMMENT ON COLUMN forms.form_type IS '表单类型：standard 标准表单 / workflow 流程表单；创建后不可变，设计器能力以此字段为准';
