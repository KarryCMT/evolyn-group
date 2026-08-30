-- 表单协议 v3：content 增加表单默认列布局 layout，切换布局时设计器同步更新字段 lineWidth。
-- 历史 v1/v2 草稿与不可变发布快照保留原协议版本，由读取侧迁移器补齐 layout=normal。
ALTER TABLE forms ALTER COLUMN protocol_version SET DEFAULT 3;
ALTER TABLE form_versions ALTER COLUMN protocol_version SET DEFAULT 3;

COMMENT ON COLUMN forms.protocol_version IS '表单保存协议版本（文档内不携带版本字段）；v3 增加表单默认列布局';
COMMENT ON COLUMN form_versions.protocol_version IS '不可变发布快照协议版本；历史 v1/v2 快照读取时迁移为当前结构';
COMMENT ON COLUMN forms.draft_content IS '目标保存协议草稿全文 JSONB：v3 根结构含 layout/items/layout_fields/field_layout，原样字节存取保证未编辑属性不丢失';
