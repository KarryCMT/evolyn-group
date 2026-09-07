-- 发布版本冻结提交校验程序；历史 v1-v6 以空对象表示无规则。
ALTER TABLE tn_form_versions
    ADD COLUMN IF NOT EXISTS compiled_submit_rules JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN tn_form_versions.compiled_submit_rules IS
    'v7 发布期受控公式程序、字段依赖和模板 token；提交只消费对应版本产物';
