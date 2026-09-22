ALTER TABLE tn_form_versions
    ADD COLUMN IF NOT EXISTS compiled_field_formulas JSONB NOT NULL DEFAULT '{}'::jsonb;
