-- 表单协议 v2：草稿 JSONB 增加 layout_fields / field_layout，首期仅开放 multitab。
-- 历史草稿与不可变发布快照继续保留 protocol_version=1，由读取侧迁移器无损补齐平铺引用。
ALTER TABLE forms ALTER COLUMN protocol_version SET DEFAULT 2;
ALTER TABLE form_versions ALTER COLUMN protocol_version SET DEFAULT 2;

COMMENT ON COLUMN forms.protocol_version IS '表单保存协议版本（文档内不携带版本字段）；v2 增加 multitab 布局定义与引用序列';
COMMENT ON COLUMN form_versions.protocol_version IS '不可变发布快照协议版本；历史 v1 快照读取时迁移为当前结构';
