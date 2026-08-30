-- 表单协议 v4：子表单补齐行操作权限、快速填报、冻结列与移动端展示配置。
-- 历史 v1-v3 草稿与不可变发布快照保留原协议版本，由前端读取侧迁移器补齐默认配置。
ALTER TABLE forms ALTER COLUMN protocol_version SET DEFAULT 4;
ALTER TABLE form_versions ALTER COLUMN protocol_version SET DEFAULT 4;

COMMENT ON COLUMN forms.protocol_version IS '表单保存协议版本（文档内不携带版本字段）；v4 增加子表单权限与端侧展示配置';
COMMENT ON COLUMN form_versions.protocol_version IS '不可变发布快照协议版本；历史 v1-v3 快照读取时迁移为当前结构';
COMMENT ON COLUMN forms.draft_content IS '目标保存协议草稿全文 JSONB：v4 子表单含嵌套字段、行权限、快速填报、冻结列及移动端展示配置';
