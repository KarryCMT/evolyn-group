-- 仅恢复列默认值；已经保存的 v3 草稿不做有损降级，仍由 protocol_version 明确标识。
ALTER TABLE forms ALTER COLUMN protocol_version SET DEFAULT 2;
ALTER TABLE form_versions ALTER COLUMN protocol_version SET DEFAULT 2;

COMMENT ON COLUMN forms.protocol_version IS '表单保存协议版本（文档内不携带版本字段）；v2 增加 multitab 布局定义与引用序列';
COMMENT ON COLUMN form_versions.protocol_version IS '不可变发布快照协议版本；历史 v1 快照读取时迁移为当前结构';
COMMENT ON COLUMN forms.draft_content IS '目标保存协议草稿全文 JSONB：v2 根结构含 items/layout_fields/field_layout，原样字节存取保证未编辑属性不丢失';
