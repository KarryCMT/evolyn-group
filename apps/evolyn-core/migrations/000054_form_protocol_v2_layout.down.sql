-- 仅恢复列默认值；已经保存的 v2 草稿不做有损降级，仍由 protocol_version 明确标识。
ALTER TABLE forms ALTER COLUMN protocol_version SET DEFAULT 1;
ALTER TABLE form_versions ALTER COLUMN protocol_version SET DEFAULT 1;

COMMENT ON COLUMN forms.protocol_version IS '协议版本外部承载（协议文档内不携带版本字段，字典 1.3）';
COMMENT ON COLUMN form_versions.protocol_version IS '快照协议版本（与发布时 forms 行一致）';
