-- 000072 down: 回滚「最后写人人」双列（连带丢弃回填/写回数据，仅开发回滚用）
ALTER TABLE tn_form_records
    DROP COLUMN IF EXISTS updated_by_member_id,
    DROP COLUMN IF EXISTS updated_by_name;
