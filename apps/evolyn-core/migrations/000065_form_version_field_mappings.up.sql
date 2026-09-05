-- 发布版本冻结逻辑字段到两种存储模式的映射；历史快照保留空映射，读取侧可从
-- 不可变 content 回填推导，禁止重写既有记录 values。
ALTER TABLE tn_form_versions
    ADD COLUMN IF NOT EXISTS field_mappings JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN tn_form_versions.field_mappings IS
    '发布时冻结的 widgetName→JSONB 键/未来物理列映射，供受控查询编译器白名单解析';
